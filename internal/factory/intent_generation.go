package factory

import "encoding/json"

// IntentGeneration records the immutable native mutation before submission.
// Native receipts and exact task acceptances retain distinct provenance.
type IntentGeneration struct {
	RevisionID   string          `json:"revision_id"`
	IntentID     string          `json:"intent_id"`
	OperationID  string          `json:"operation_id"`
	ActorID      int64           `json:"actor_id,string"`
	NativeRev    int64           `json:"native_revision"`
	AuthRevision string          `json:"authorization_revision"`
	NotAfter     int64           `json:"not_after"`
	Payload      json.RawMessage `json:"payload"`
	IntentDigest string          `json:"intent_digest"`
	Receipt      json.RawMessage `json:"receipt,omitempty"`
	State        string          `json:"state"`
}
type IntentTaskBinding struct {
	RevisionID     string `json:"revision_id"`
	TaskKey        string `json:"task_key"`
	IssueID        int64  `json:"issue_id,string"`
	IssueIndex     int64  `json:"issue_index,string"`
	ContentVersion int    `json:"content_version"`
	AcceptanceID   string `json:"acceptance_id"`
}
