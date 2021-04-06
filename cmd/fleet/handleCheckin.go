// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/elastic/fleet-server/v7/internal/pkg/action"
	"github.com/elastic/fleet-server/v7/internal/pkg/bulk"
	"github.com/elastic/fleet-server/v7/internal/pkg/cache"
	"github.com/elastic/fleet-server/v7/internal/pkg/config"
	"github.com/elastic/fleet-server/v7/internal/pkg/dl"
	"github.com/elastic/fleet-server/v7/internal/pkg/limit"
	"github.com/elastic/fleet-server/v7/internal/pkg/model"
	"github.com/elastic/fleet-server/v7/internal/pkg/monitor"
	"github.com/elastic/fleet-server/v7/internal/pkg/policy"
	"github.com/elastic/fleet-server/v7/internal/pkg/smap"
	"github.com/elastic/fleet-server/v7/internal/pkg/sqn"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrAgentNotFound = errors.New("agent not found")
)

const kEncodingGzip = "gzip"

func (rt Router) handleCheckin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	id := ps.ByName("id")
	err := rt.ct._handleCheckin(w, r, id, rt.bulker)

	if err != nil {
		lvl := zerolog.DebugLevel

		var code int
		switch err {
		case ErrAgentNotFound:
			code = http.StatusNotFound
			lvl = zerolog.WarnLevel
		case limit.ErrRateLimit:
			code = http.StatusTooManyRequests
		case limit.ErrMaxLimit:
			// Log this as warn for visibility that limit has been reached.
			// This allows customers to tune the configuration on detection of threshold.
			code = http.StatusTooManyRequests
			lvl = zerolog.WarnLevel
		case context.Canceled:
			code = http.StatusServiceUnavailable
		default:
			lvl = zerolog.InfoLevel
			code = http.StatusBadRequest
		}

		log.WithLevel(lvl).
			Err(err).
			Str("id", id).
			Int("code", code).
			Msg("fail checkin")

		http.Error(w, "", code)
	}
}

type CheckinT struct {
	cfg    *config.Server
	cache  cache.Cache
	bc     *BulkCheckin
	pm     policy.Monitor
	gcp    monitor.GlobalCheckpointProvider
	ad     *action.Dispatcher
	tr     *action.TokenResolver
	bulker bulk.Bulk
	limit  *limit.Limiter
}

func NewCheckinT(
	cfg *config.Server,
	c cache.Cache,
	bc *BulkCheckin,
	pm policy.Monitor,
	gcp monitor.GlobalCheckpointProvider,
	ad *action.Dispatcher,
	tr *action.TokenResolver,
	bulker bulk.Bulk,
) *CheckinT {

	log.Info().
		Interface("limits", cfg.Limits.CheckinLimit).
		Dur("long_poll_timeout", cfg.Timeouts.CheckinLongPoll).
		Dur("long_poll_timestamp", cfg.Timeouts.CheckinTimestamp).
		Msg("Checkin install limits")

	ct := &CheckinT{
		cfg:    cfg,
		cache:  c,
		bc:     bc,
		pm:     pm,
		gcp:    gcp,
		ad:     ad,
		tr:     tr,
		limit:  limit.NewLimiter(&cfg.Limits.CheckinLimit),
		bulker: bulker,
	}

	return ct
}

func (ct *CheckinT) _handleCheckin(w http.ResponseWriter, r *http.Request, id string, bulker bulk.Bulk) error {

	limitF, err := ct.limit.Acquire()
	if err != nil {
		return err
	}
	defer limitF()

	agent, err := authAgent(r, id, ct.bulker, ct.cache)

	if err != nil {
		return err
	}

	ctx := r.Context()

	// Interpret request; TODO: defend overflow, slow roll
	var req CheckinRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		return err
	}

	// Compare local_metadata content and update if different
	fields, err := parseMeta(agent, &req)
	if err != nil {
		return err
	}

	// Resolve AckToken from request, fallback on the agent record
	seqno, err := ct.resolveSeqNo(ctx, req, agent)
	if err != nil {
		return err
	}

	// Subsribe to actions dispatcher
	aSub := ct.ad.Subscribe(agent.Id, seqno)
	defer ct.ad.Unsubscribe(aSub)
	actCh := aSub.Ch()

	// Subscribe to policy manager for changes on PolicyId > policyRev
	sub, err := ct.pm.Subscribe(agent.Id, agent.PolicyId, agent.PolicyRevisionIdx, agent.PolicyCoordinatorIdx)
	if err != nil {
		return err
	}
	defer ct.pm.Unsubscribe(sub)

	// Update check-in timestamp on timeout
	tick := time.NewTicker(ct.cfg.Timeouts.CheckinTimestamp)
	defer tick.Stop()

	// Chill out for for a bit. Long poll.
	longPoll := time.NewTicker(ct.cfg.Timeouts.CheckinLongPoll)
	defer longPoll.Stop()

	// Intial update on checkin, and any user fields that might have changed
	ct.bc.CheckIn(agent.Id, fields, seqno)

	// Initial fetch for pending actions
	var (
		actions  []ActionResp
		ackToken string
	)

	// Check agent pending actions first
	pendingActions, err := ct.fetchAgentPendingActions(ctx, seqno, agent.Id)
	if err != nil {
		return err
	}
	actions, ackToken = convertActions(agent.Id, pendingActions)

	if len(actions) == 0 {
	LOOP:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case acdocs := <-actCh:
				var acs []ActionResp
				acs, ackToken = convertActions(agent.Id, acdocs)
				actions = append(actions, acs...)
				break LOOP
			case policy := <-sub.Output():
				actionResp, err := parsePolicy(ctx, bulker, agent.Id, policy)
				if err != nil {
					return err
				}
				actions = append(actions, *actionResp)
				break LOOP
			case <-longPoll.C:
				log.Trace().Msg("Fire long poll")
				break LOOP
			case <-tick.C:
				ct.bc.CheckIn(agent.Id, nil, seqno)
			}
		}
	}

	// For now, empty response
	resp := CheckinResponse{
		AckToken: ackToken,
		Action:   "checkin",
		Actions:  actions,
	}

	return ct.writeResponse(w, r, resp)
}

func (ct *CheckinT) writeResponse(w http.ResponseWriter, r *http.Request, resp CheckinResponse) error {

	payload, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	compressionLevel := ct.cfg.CompressionLevel
	compressThreshold := ct.cfg.CompressionThresh

	if len(payload) > compressThreshold && compressionLevel != flate.NoCompression && acceptsEncoding(r, kEncodingGzip) {

		zipper, err := gzip.NewWriterLevel(w, compressionLevel)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Encoding", kEncodingGzip)

		if _, err = zipper.Write(payload); err != nil {
			return err
		}

		err = zipper.Close()

		log.Trace().
			Err(err).
			Int("dataSz", len(payload)).
			Int("lvl", compressionLevel).
			Msg("Compressing checkin response")
	} else {
		_, err = w.Write(payload)
	}

	return err
}

func acceptsEncoding(r *http.Request, encoding string) bool {
	for _, v := range r.Header.Values("Accept-Encoding") {
		if v == encoding {
			return true
		}
	}
	return false
}

// Resolve AckToken from request, fallback on the agent record
func (ct *CheckinT) resolveSeqNo(ctx context.Context, req CheckinRequest, agent *model.Agent) (seqno sqn.SeqNo, err error) {
	// Resolve AckToken from request, fallback on the agent record
	ackToken := req.AckToken
	seqno = agent.ActionSeqNo

	if ct.tr != nil && ackToken != "" {
		var sn int64
		sn, err = ct.tr.Resolve(ctx, ackToken)
		if err != nil {
			if errors.Is(err, dl.ErrNotFound) {
				log.Debug().Str("token", ackToken).Str("agent_id", agent.Id).Msg("revision token not found")
				err = nil
			} else {
				return
			}
		}
		seqno = []int64{sn}
	}
	return seqno, nil
}

func (ct *CheckinT) fetchAgentPendingActions(ctx context.Context, seqno sqn.SeqNo, agentId string) ([]model.Action, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	return dl.FindActions(ctx, ct.bulker, dl.QueryAgentActions, map[string]interface{}{
		dl.FieldSeqNo:      seqno.Get(0),
		dl.FieldMaxSeqNo:   ct.gcp.GetCheckpoint(),
		dl.FieldExpiration: now,
		dl.FieldAgents:     []string{agentId},
	})
}

func convertActions(agentId string, actions []model.Action) ([]ActionResp, string) {
	var ackToken string
	sz := len(actions)

	respList := make([]ActionResp, 0, sz)
	for _, action := range actions {
		respList = append(respList, ActionResp{
			AgentId:   agentId,
			CreatedAt: action.Timestamp,
			Data:      []byte(action.Data),
			Id:        action.ActionId,
			Type:      action.Type,
			InputType: action.InputType,
		})
	}

	if sz > 0 {
		ackToken = actions[sz-1].Id
	}

	return respList, ackToken
}

func parsePolicy(ctx context.Context, bulker bulk.Bulk, agentId string, p model.Policy) (*ActionResp, error) {
	// Need to inject the default api key into the object. So:
	// 1) Deserialize the action
	// 2) Lookup the DefaultApiKey in the save agent (we purposefully didn't decode it before)
	// 3) If not there, generate and persist DefaultAPIKey
	// 4) Inject default api key into structure
	// 5) Re-serialize and return AgentResp structure

	// using json.RawMessage to avoid the full json de-serialization
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(p.Data, &actionObj); err != nil {
		return nil, err
	}

	// Repull and decode the agent object
	var agent model.Agent
	agent, err := dl.FindAgent(ctx, bulker, dl.QueryAgentByID, dl.FieldId, agentId)
	if err != nil {
		return nil, err
	}

	// Check if need to generate a new output api key
	var (
		hash    string
		needKey bool
		roles   []byte
	)

	if agent.DefaultApiKey == "" {
		hash, roles, err = policy.GetRoleDescriptors(actionObj[policy.OutputPermissionsProperty])
		if err != nil {
			return nil, err
		}
		needKey = true
		log.Debug().Str("agentId", agentId).Msg("agent API key is not present")
	} else {
		hash, roles, needKey, err = policy.CheckOutputPermissionsChanged(agent.PolicyOutputPermissionsHash, actionObj[policy.OutputPermissionsProperty])
		if err != nil {
			return nil, err
		}
		if needKey {
			log.Debug().Str("agentId", agentId).Msg("policy output permissions changed")
		} else {
			log.Debug().Str("agentId", agentId).Msg("policy output permissions are the same")
		}
	}

	if needKey {
		log.Debug().Str("agentId", agentId).RawJSON("roles", roles).Str("hash", hash).Msg("generating a new API key")
		defaultOutputApiKey, err := generateOutputApiKey(ctx, bulker.Client(), agent.Id, policy.DefaultOutputName, roles)
		if err != nil {
			return nil, err
		}
		agent.DefaultApiKey = defaultOutputApiKey.Agent()
		agent.DefaultApiKeyId = defaultOutputApiKey.Id
		agent.PolicyOutputPermissionsHash = hash

		log.Info().Str("agentId", agentId).Msg("rewriting full agent record to pick up default output key.")
		if err = dl.IndexAgent(ctx, bulker, agent); err != nil {
			return nil, err
		}
	}

	// Parse the outputs maps in order to inject the api key
	const outputsProperty = "outputs"
	outputs, err := smap.Parse(actionObj[outputsProperty])
	if err != nil {
		return nil, err
	}

	if outputs != nil {
		if ok := setMapObj(outputs, agent.DefaultApiKey, "default", "api_key"); !ok {
			log.Debug().Msg("cannot inject api_key into policy")
		} else {
			outputRaw, err := json.Marshal(outputs)
			if err != nil {
				return nil, err
			}
			actionObj[outputsProperty] = json.RawMessage(outputRaw)
		}
	}

	dataJSON, err := json.Marshal(struct {
		Policy map[string]json.RawMessage `json:"policy"`
	}{actionObj})
	if err != nil {
		return nil, err
	}

	r := policy.RevisionFromPolicy(p)
	resp := ActionResp{
		AgentId:   agent.Id,
		CreatedAt: p.Timestamp,
		Data:      dataJSON,
		Id:        r.String(),
		Type:      TypePolicyChange,
	}

	return &resp, nil
}

func setMapObj(obj map[string]interface{}, val interface{}, keys ...string) bool {
	if len(keys) == 0 {
		return false
	}

	for _, k := range keys[:len(keys)-1] {
		v, ok := obj[k]
		if !ok {
			return false
		}

		obj, ok = v.(map[string]interface{})
		if !ok {
			return false
		}
	}

	k := keys[len(keys)-1]
	obj[k] = val

	return true
}

func findAgentByApiKeyId(ctx context.Context, bulker bulk.Bulk, id string) (*model.Agent, error) {
	agent, err := dl.FindAgent(ctx, bulker, dl.QueryAgentByAssessAPIKeyID, dl.FieldAccessAPIKeyID, id)
	if err != nil && errors.Is(err, dl.ErrNotFound) {
		err = ErrAgentNotFound
	}
	return &agent, err
}

// parseMeta compares the agent and the request local_metadata content
// and returns fields to update the agent record or nil
func parseMeta(agent *model.Agent, req *CheckinRequest) (fields Fields, err error) {
	// Quick comparison first
	if bytes.Equal(req.LocalMeta, agent.LocalMetadata) {
		log.Trace().Msg("quick comparing local metadata is equal")
		return nil, nil
	}

	// Compare local_metadata content and update if different
	var reqLocalMeta Fields
	var agentLocalMeta Fields
	err = json.Unmarshal(req.LocalMeta, &reqLocalMeta)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(agent.LocalMetadata, &agentLocalMeta)
	if err != nil {
		return nil, err
	}

	if reqLocalMeta != nil && !reflect.DeepEqual(reqLocalMeta, agentLocalMeta) {
		log.Trace().RawJSON("oldLocalMeta", agent.LocalMetadata).RawJSON("newLocalMeta", req.LocalMeta).Msg("Local metadata not equal")
		log.Info().RawJSON("req.LocalMeta", req.LocalMeta).Msg("applying new local metadata")
		fields = map[string]interface{}{
			FieldLocalMetadata: req.LocalMeta,
		}
	}
	return fields, nil
}
