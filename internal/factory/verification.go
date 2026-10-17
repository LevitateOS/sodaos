// D16 whole-app verification subjects and typed result envelope. Test
// Pilot verifies a delivered accepted intent version; one in-scope
// Strategist repair proposal may answer each failed verification. This
// file owns the admission identity for both kinds and the parity result
// envelope R10 binds to the repair/verify phases. Extended D15/D16
// cause, coverage, adoption and receipt semantics belong to the R37-R40
// finish controls; the envelope here carries the identity, digest and
// outcome bindings every phase-role-subject-result matrix case needs.
package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// DeliverySubject binds one app verification to its accepted intent
// version: the delivery identity, the accepted intent revision and scope,
// the source and preparation it was admitted against, and the approved
// verification manifest it must execute. Key `delivery:<delivery_id>`,
// phase `verify`, reviewer account, no publication or children.
type DeliverySubject struct {
	DeliveryID                 string `json:"delivery_id"`
	IntentID                   string `json:"intent_id"`
	AcceptedRevision           string `json:"accepted_revision"`
	ScopeDigest                string `json:"scope_digest"`
	SourceCommit               string `json:"source_commit"`
	PreparationDigest          string `json:"preparation_digest"`
	VerificationManifestDigest string `json:"verification_manifest_digest"`
}

// DeliveryKey names the delivery subject key space.
func DeliveryKey(id string) string { return "delivery:" + id }

func (s DeliverySubject) Validate() error {
	if !ValidID(s.DeliveryID) {
		return errors.New("invalid delivery subject identity")
	}
	if !project.ValidDecisionID(s.IntentID) || !project.ValidDecisionID(s.AcceptedRevision) {
		return errors.New("invalid delivery subject intent revision")
	}
	if !ValidDigest(s.ScopeDigest) || !ValidDigest(s.PreparationDigest) ||
		!ValidDigest(s.VerificationManifestDigest) {
		return errors.New("delivery subject must bind its scope digests")
	}
	if !ValidCommit(s.SourceCommit) {
		return errors.New("invalid delivery subject source")
	}
	return nil
}

// DeliveryRepairSubject binds one verification repair to its verified
// delivery failure: the delivery, the exact failed verification run, the
// failure digest, the sealed inventory it may change, and its repair
// cycle. Key `delivery_repair:<delivery_id>:<failed_run>`, phase
// `repair`, Strategist profile/coder account, no publication or children.
type DeliveryRepairSubject struct {
	DeliveryID              string `json:"delivery_id"`
	FailedVerificationRunID string `json:"failed_verification_run_id"`
	FailureDigest           string `json:"failure_digest"`
	InventoryDigest         string `json:"inventory_digest"`
	CycleID                 string `json:"cycle_id"`
}

// DeliveryRepairKey names the delivery-repair subject key space.
func DeliveryRepairKey(delivery, failedRun string) string {
	return "delivery_repair:" + delivery + ":" + failedRun
}

func (s DeliveryRepairSubject) Validate() error {
	if !ValidID(s.DeliveryID) || !ValidID(s.CycleID) {
		return errors.New("invalid delivery repair subject identity")
	}
	if !ValidID(s.FailedVerificationRunID) {
		return errors.New("delivery repair subject lacks its failed verification run")
	}
	if !ValidDigest(s.FailureDigest) || !ValidDigest(s.InventoryDigest) {
		return errors.New("delivery repair subject must bind its failure evidence")
	}
	return nil
}

// App verification outcomes: the exact delivery is verified, failed with
// recorded evidence, or incomplete and never finished.
const (
	AppVerificationVerified   = "verified"
	AppVerificationFailed     = "failed"
	AppVerificationIncomplete = "incomplete"
)

// ValidAppVerificationOutcome reports whether outcome is an admitted D16
// app verification outcome.
func ValidAppVerificationOutcome(outcome string) bool {
	switch outcome {
	case AppVerificationVerified, AppVerificationFailed, AppVerificationIncomplete:
		return true
	default:
		return false
	}
}

// AppVerificationReport is the typed output of a completed verify phase:
// the delivery it checked, the exact admitted input digest, and its
// outcome. It carries no candidate: verification never authorizes
// publication. Check/artifact receipts behind the outcome belong to the
// R38 trusted-manifest consumers; the envelope binds the result.
type AppVerificationReport struct {
	DeliveryID  string `json:"delivery_id"`
	InputDigest string `json:"input_digest"`
	Outcome     string `json:"outcome"`
}

func (r AppVerificationReport) Validate() error {
	if !ValidID(r.DeliveryID) {
		return errors.New("invalid app verification delivery")
	}
	if !ValidDigest(r.InputDigest) {
		return errors.New("app verification lacks its admitted input digest")
	}
	if !ValidAppVerificationOutcome(r.Outcome) {
		return errors.New("invalid app verification outcome")
	}
	return nil
}

// DeliveryRepairReport is the typed output of a completed repair phase
// answering a delivery failure: the repair subject, the exact admitted
// input digest, and its outcome under the shared D15 repair rubric. It
// carries no candidate. Cause/coverage/adoption detail belongs to the
// R37/R40 repair consumers; the envelope binds the result.
type DeliveryRepairReport struct {
	DeliveryID              string `json:"delivery_id"`
	FailedVerificationRunID string `json:"failed_verification_run_id"`
	InputDigest             string `json:"input_digest"`
	InventoryDigest         string `json:"inventory_digest"`
	CycleID                 string `json:"cycle_id"`
	Outcome                 string `json:"outcome"`
}

func (r DeliveryRepairReport) Validate() error {
	if !ValidID(r.DeliveryID) || !ValidID(r.CycleID) {
		return errors.New("invalid delivery repair report identity")
	}
	if !ValidID(r.FailedVerificationRunID) {
		return errors.New("delivery repair report lacks its failed verification run")
	}
	if !ValidDigest(r.InputDigest) || !ValidDigest(r.InventoryDigest) {
		return errors.New("delivery repair report must bind its evidence digests")
	}
	if !ValidRepairOutcome(r.Outcome) {
		return errors.New("invalid delivery repair outcome")
	}
	return nil
}
