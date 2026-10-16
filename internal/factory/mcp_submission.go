package factory

// D14 capability/submission DTOs (FW38).
//
// A capability binds one run to one assignment, profile, subject and
// input snapshot. A submission is one immutable tool call under that
// capability. Store persists both; control admits both; transport
// binds every request to both. Secrets never appear here: the
// per-capability secret derives from the immutable identity plus a
// service-private key held outside these DTOs, and rows retain only
// the ID and scope.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// Capability states. Only active capabilities admit calls; expiry,
// stop intent, withdrawal and terminal retirement revoke access.
const (
	MCPCapabilityStaged  = "staged"
	MCPCapabilityActive  = "active"
	MCPCapabilityRevoked = "revoked"
	MCPCapabilityRetired = "retired"
)

// MCPCapability is the recorded run authority. Generation orders
// re-deliveries of the same run: exact restart delivery reproduces
// the same capability, never a new deadline or allowance.
type MCPCapability struct {
	ID                string     `json:"id"`
	AssignmentID      string     `json:"assignment_id"`
	RunID             string     `json:"run_id"`
	Generation        int64      `json:"generation"`
	Profile           MCPProfile `json:"profile"`
	Subject           string     `json:"subject"`
	Repository        int64      `json:"repository"`
	Project           string     `json:"project"`
	PreparationDigest string     `json:"preparation_digest"`
	SourceDigest      string     `json:"source_digest"`
	AcceptedRevision  string     `json:"accepted_revision"`
	Authority         string     `json:"authority"`
	DeadlineUnix      int64      `json:"deadline_unix"`
	State             string     `json:"state"`
}

// Validate checks the immutable identity and scope of a capability.
// Deadline freshness and stop state are admission checks, not shape.
func (c MCPCapability) Validate() error {
	if !ValidID(c.ID) {
		return errors.New("invalid capability identity")
	}
	if !ValidID(c.AssignmentID) {
		return errors.New("invalid capability assignment")
	}
	if c.RunID == "" || len(c.RunID) > MCPMaxRefChars {
		return errors.New("invalid capability run")
	}
	if c.Generation < 1 {
		return errors.New("invalid capability generation")
	}
	if MCPPhaseOf(c.Profile) == "" {
		return errors.New("unknown capability profile")
	}
	if c.Subject == "" || len(c.Subject) > MCPMaxRefChars {
		return errors.New("invalid capability subject")
	}
	if c.Repository < 1 {
		return errors.New("invalid capability repository")
	}
	if !ValidProjectID(c.Project) {
		return errors.New("invalid capability project")
	}
	for name, digest := range map[string]string{
		"preparation": c.PreparationDigest,
		"authority":   c.Authority,
	} {
		if !ValidDigest(digest) {
			return errors.New("invalid capability " + name + " digest")
		}
	}
	if c.SourceDigest == "" || c.AcceptedRevision == "" {
		return errors.New("capability source snapshot incomplete")
	}
	switch c.State {
	case MCPCapabilityStaged, MCPCapabilityActive, MCPCapabilityRevoked, MCPCapabilityRetired:
	default:
		return errors.New("unknown capability state")
	}
	return nil
}

// Expired reports whether the capability is past its deadline at now.
func (c MCPCapability) Expired(now time.Time) bool {
	return !now.Before(time.Unix(c.DeadlineUnix, 0))
}

// ScopeDigest binds the immutable identity: any change to assignment,
// run, generation, profile, subject, repository, project or input
// snapshot changes the scope, so a replayed capability with edited
// bytes cannot borrow the original authority.
func (c MCPCapability) ScopeDigest() string {
	sum := sha256.Sum256([]byte(
		c.ID + "\n" + c.AssignmentID + "\n" + c.RunID + "\n" +
			itoa(c.Generation) + "\n" + string(c.Profile) + "\n" + c.Subject + "\n" +
			itoa(c.Repository) + "\n" + c.Project + "\n" + c.PreparationDigest + "\n" +
			c.SourceDigest + "\n" + c.AcceptedRevision + "\n" + c.Authority + "\n" +
			itoa(c.DeadlineUnix) + "\n"))
	return hex.EncodeToString(sum[:])
}

// DeriveMCPCapabilitySecret derives the per-capability secret with
// HMAC-SHA256 over the immutable identity and scope digest under the
// service-private persistent key. Host-owned delivery stages the
// secret separately from provider credentials; callers prove it
// without the server ever storing secret bytes beside the row.
func DeriveMCPCapabilitySecret(key []byte, capability MCPCapability) (string, error) {
	if len(key) < 32 {
		return "", errors.New("capability key below 256 bits")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(capability.ID + "\n" + capability.ScopeDigest()))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Receipt states. Transport/schema/authentication failure creates no
// receipt; only admitted submissions record one.
const (
	MCPReceiptRecorded  = "recorded"
	MCPReceiptVerifying = "verifying"
	MCPReceiptAccepted  = "accepted"
	MCPReceiptRefused   = "refused"
)

// MCPSubmission is one immutable tool call. Unique
// (assignment_id, submission_id) is the idempotency key: exact replay
// returns the existing receipt, changed bytes under that ID conflict.
type MCPSubmission struct {
	AssignmentID        string  `json:"assignment_id"`
	SubmissionID        string  `json:"submission_id"`
	Tool                MCPTool `json:"tool"`
	ExpectedInputDigest string  `json:"expected_input_digest"`
	PayloadHash         string  `json:"payload_hash"`
	Payload             []byte  `json:"payload,omitempty"`
}

// MCPInputDigest digests canonical tool payload bytes before
// submission so admission can compare the caller's expectation
// against the recorded hash without trusting caller prose.
func MCPInputDigest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Validate checks submission shape and byte bounds. Tool/profile
// binding and terminal freezing are admission checks, not shape.
func (s MCPSubmission) Validate() error {
	if !ValidID(s.AssignmentID) {
		return errors.New("invalid submission assignment")
	}
	if !ValidID(s.SubmissionID) {
		return errors.New("invalid submission identity")
	}
	if !MCPToolKnown(s.Tool) {
		return errors.New("unknown submission tool")
	}
	if !ValidDigest(s.ExpectedInputDigest) {
		return errors.New("invalid expected input digest")
	}
	if !ValidDigest(s.PayloadHash) {
		return errors.New("invalid submission payload hash")
	}
	if len(s.Payload) == 0 {
		return errors.New("empty submission payload")
	}
	if len(s.Payload) > MCPMaxSubmissionPayload {
		return errors.New("submission payload exceeds 64 KiB")
	}
	if MCPInputDigest(s.Payload) != s.PayloadHash {
		return errors.New("submission payload hash mismatch")
	}
	if s.ExpectedInputDigest != s.PayloadHash {
		return errors.New("submission input digest mismatch")
	}
	return nil
}

// MCPReceipt is the durable terminal record for one submission.
type MCPReceipt struct {
	SubmissionID string  `json:"submission_id"`
	Tool         MCPTool `json:"tool"`
	State        string  `json:"state"`
	InputDigest  string  `json:"input_digest"`
	Reason       string  `json:"reason,omitempty"`
}

// ValidateReceipt bounds refusal diagnostics: bounded reason text,
// never unbounded new receipt rows.
func (r MCPReceipt) Validate() error {
	if !ValidID(r.SubmissionID) {
		return errors.New("invalid receipt submission")
	}
	if !MCPToolKnown(r.Tool) {
		return errors.New("unknown receipt tool")
	}
	switch r.State {
	case MCPReceiptRecorded, MCPReceiptVerifying, MCPReceiptAccepted, MCPReceiptRefused:
	default:
		return errors.New("unknown receipt state")
	}
	if !ValidDigest(r.InputDigest) {
		return errors.New("invalid receipt input digest")
	}
	if len(r.Reason) > MCPMaxReasonChars {
		return errors.New("receipt reason exceeds bound")
	}
	return nil
}
