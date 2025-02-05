// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by @aleksmaus schema-generate quick&dirty mod for ES mappings. DO NOT EDIT.

package es

const (

	// Action An Elastic Agent action
	MappingAction = `{
	"properties": {
		"action_id": {
			"type": "keyword"
		},
		"agents": {
			"type": "keyword"
		},
		"data": {
			"enabled" : false,
			"type": "object"
		},
		"expiration": {
			"type": "date"
		},
		"input_type": {
			"type": "keyword"
		},
		"timeout": {
			"type": "integer"
		},
		"@timestamp": {
			"type": "date"
		},
		"type": {
			"type": "keyword"
		},
		"user_id": {
			"type": "keyword"
		}		
	}
}`

	// ActionData The opaque payload.
	MappingActionData = `{
	"properties": {
		
	}
}`

	// ActionResponse The custom action response payload.
	MappingActionResponse = `{
	"properties": {
		
	}
}`

	// ActionResult An Elastic Agent action results
	MappingActionResult = `{
	"properties": {
		"action_data": {
			"enabled" : false,
			"type": "object"
		},
		"action_id": {
			"type": "keyword"
		},
		"action_response": {
			"enabled" : false,
			"type": "object"
		},
		"agent_id": {
			"type": "keyword"
		},
		"completed_at": {
			"type": "date"
		},
		"data": {
			"enabled" : false,
			"type": "object"
		},
		"error": {
			"type": "keyword"
		},
		"started_at": {
			"type": "date"
		},
		"@timestamp": {
			"type": "date"
		}		
	}
}`

	// Agent An Elastic Agent that has enrolled into Fleet
	MappingAgent = `{
	"properties": {
		"access_api_key_id": {
			"type": "keyword"
		},
		"action_seq_no": {
			"type": "integer"
		},
		"active": {
			"type": "boolean"
		},
		"agent": {
			"properties": {
				"id": {
					"type": "keyword"
				},
				"version": {
					"type": "keyword"
				}				
			}
		},
		"default_api_key": {
			"type": "keyword"
		},
		"default_api_key_history": {
			"properties": {
				"id": {
					"type": "keyword"
				},
				"retired_at": {
					"type": "date"
				}				
			}
		},
		"default_api_key_id": {
			"type": "keyword"
		},
		"enrolled_at": {
			"type": "date"
		},
		"last_checkin": {
			"type": "date"
		},
		"last_checkin_status": {
			"type": "keyword"
		},
		"last_updated": {
			"type": "date"
		},
		"local_metadata": {
			"enabled" : false,
			"type": "object"
		},
		"packages": {
			"type": "keyword"
		},
		"policy_coordinator_idx": {
			"type": "integer"
		},
		"policy_id": {
			"type": "keyword"
		},
		"policy_output_permissions_hash": {
			"type": "keyword"
		},
		"policy_revision_idx": {
			"type": "integer"
		},
		"shared_id": {
			"type": "keyword"
		},
		"type": {
			"type": "keyword"
		},
		"unenrolled_at": {
			"type": "date"
		},
		"unenrolled_reason": {
			"type": "keyword"
		},
		"unenrollment_started_at": {
			"type": "date"
		},
		"updated_at": {
			"type": "date"
		},
		"upgrade_started_at": {
			"type": "date"
		},
		"upgraded_at": {
			"type": "date"
		},
		"user_provided_metadata": {
			"enabled" : false,
			"type": "object"
		}		
	}
}`

	// AgentMetadata An Elastic Agent metadata
	MappingAgentMetadata = `{
	"properties": {
		"id": {
			"type": "keyword"
		},
		"version": {
			"type": "keyword"
		}		
	}
}`

	// Artifact An artifact served by Fleet
	MappingArtifact = `{
	"properties": {
		"body": {
			"enabled" : false,
			"type": "object"
		},
		"compression_algorithm": {
			"type": "keyword"
		},
		"created": {
			"type": "date"
		},
		"decoded_sha256": {
			"type": "keyword"
		},
		"decoded_size": {
			"type": "integer"
		},
		"encoded_sha256": {
			"type": "keyword"
		},
		"encoded_size": {
			"type": "integer"
		},
		"encryption_algorithm": {
			"type": "keyword"
		},
		"identifier": {
			"type": "keyword"
		},
		"package_name": {
			"type": "keyword"
		}		
	}
}`

	// Body Encoded artifact data
	MappingBody = `{
	"properties": {
		
	}
}`

	// Data The opaque payload.
	MappingData = `{
	"properties": {
		
	}
}`

	// DefaultApiKeyHistoryItems
	MappingDefaultApiKeyHistoryItems = `{
	"properties": {
		"id": {
			"type": "keyword"
		},
		"retired_at": {
			"type": "date"
		}		
	}
}`

	// EnrollmentApiKey An Elastic Agent enrollment API key
	MappingEnrollmentApiKey = `{
	"properties": {
		"active": {
			"type": "boolean"
		},
		"api_key": {
			"type": "keyword"
		},
		"api_key_id": {
			"type": "keyword"
		},
		"created_at": {
			"type": "date"
		},
		"expire_at": {
			"type": "date"
		},
		"name": {
			"type": "keyword"
		},
		"policy_id": {
			"type": "keyword"
		},
		"updated_at": {
			"type": "date"
		}		
	}
}`

	// HostMetadata The host metadata for the Elastic Agent
	MappingHostMetadata = `{
	"properties": {
		"architecture": {
			"type": "keyword"
		},
		"id": {
			"type": "keyword"
		},
		"ip": {
			"type": "keyword"
		},
		"name": {
			"type": "keyword"
		}		
	}
}`

	// LocalMetadata Local metadata information for the Elastic Agent
	MappingLocalMetadata = `{
	"properties": {
		
	}
}`

	// Policy A policy that an Elastic Agent is attached to
	MappingPolicy = `{
	"properties": {
		"coordinator_idx": {
			"type": "integer"
		},
		"data": {
			"enabled" : false,
			"type": "object"
		},
		"default_fleet_server": {
			"type": "boolean"
		},
		"policy_id": {
			"type": "keyword"
		},
		"revision_idx": {
			"type": "integer"
		},
		"@timestamp": {
			"type": "date"
		},
		"unenroll_timeout": {
			"type": "integer"
		}		
	}
}`

	// PolicyLeader The current leader Fleet Server for a policy
	MappingPolicyLeader = `{
	"properties": {
		"server": {
			"properties": {
				"id": {
					"type": "keyword"
				},
				"version": {
					"type": "keyword"
				}				
			}
		},
		"@timestamp": {
			"type": "date"
		}		
	}
}`

	// Server A Fleet Server
	MappingServer = `{
	"properties": {
		"agent": {
			"properties": {
				"id": {
					"type": "keyword"
				},
				"version": {
					"type": "keyword"
				}				
			}
		},
		"host": {
			"properties": {
				"architecture": {
					"type": "keyword"
				},
				"id": {
					"type": "keyword"
				},
				"ip": {
					"type": "keyword"
				},
				"name": {
					"type": "keyword"
				}				
			}
		},
		"server": {
			"properties": {
				"id": {
					"type": "keyword"
				},
				"version": {
					"type": "keyword"
				}				
			}
		},
		"@timestamp": {
			"type": "date"
		}		
	}
}`

	// ServerMetadata A Fleet Server metadata
	MappingServerMetadata = `{
	"properties": {
		"id": {
			"type": "keyword"
		},
		"version": {
			"type": "keyword"
		}		
	}
}`

	// UserProvidedMetadata User provided metadata information for the Elastic Agent
	MappingUserProvidedMetadata = `{
	"properties": {
		
	}
}`
)
