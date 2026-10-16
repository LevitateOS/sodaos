package factory

import (
	"errors"
	"reflect"
	"time"
)

// DeliveryRubric is the fixed Jev criterion for the delivery-review-v1
// purpose (D08). Ready_to_merge means the supplied evidence adequately
// covers the accepted outcomes and every actionable finding and another
// independent pass is unlikely to find a material defect; review_again
// means a further independent inspection is warranted. No missing
// evidence, unresolved dispute or requirement is ever treated as
// satisfied. Code supplies this rubric; Jev supplies only the choice.
const DeliveryRubric = "Decide whether the supplied evidence adequately covers the accepted outcomes and every actionable finding. Choose ready_to_merge only when the evidence covers the accepted outcomes and every actionable finding and another independent pass is unlikely to find a material defect, with no missing evidence, unresolved dispute or unsatisfied requirement. Otherwise a further independent inspection is warranted and you choose review_again. Judge only supplied facts; grant no authority, access or spending."

// DeliveryOptions returns the exactly-two delivery choices with their
// fixed criterion text.
func DeliveryOptions() map[string]any {
	return map[string]any{
		DeliveryReviewAgain:  "A further independent inspection is warranted to establish coverage or correction sufficiency from the supplied evidence.",
		DeliveryReadyToMerge: "The supplied evidence adequately covers the accepted outcomes and every actionable finding; another independent pass is unlikely to find a material defect.",
	}
}

// DeliveryCall is one funded delivery consultation over the exact
// semantic fingerprint, with its answer or its provider failure. A call
// carrying a failure never authorizes merge.
type DeliveryCall struct {
	Fingerprint string         `json:"fingerprint"`
	Answer      DeliveryAnswer `json:"answer"`
	Failure     string         `json:"failure,omitempty"`
}

// DeliveryReceiptKind identifies durable funded delivery judgments in the
// shared judgment journal.
const DeliveryReceiptKind = "delivery-v1"

// DeliveryOutcome names the two durable delivery states. Decided retains
// the raw answer with its effective action; unavailable retains a
// reserved fingerprint without a completed answer. An unavailable receipt
// stays charged and is never automatically retried.
const (
	DeliveryOutcomeDecided     = "decided"
	DeliveryOutcomeUnavailable = "unavailable"
)

// DeliveryReceipt is the durable funded delivery judgment for one exact
// semantic fingerprint. Input is the complete semantic evidence the
// fingerprint binds; Decision retains the raw answer with its derived
// effective action. ObservedNativeRevision is the raw native revision
// observed alongside the evidence, kept separately for native mutation
// guards; it never enters the fingerprint. FW22 owns wiring this receipt
// into real correction/cycle paths.
type DeliveryReceipt struct {
	Kind                   string                   `json:"kind"`
	Fingerprint            string                   `json:"fingerprint"`
	Input                  DeliveryFingerprintInput `json:"input"`
	Calls                  []DeliveryCall           `json:"calls"`
	Decision               DeliveryDecision         `json:"decision"`
	Outcome                string                   `json:"outcome"`
	ObservedNativeRevision int64                    `json:"observed_native_revision"`
	CreatedUnix            int64                    `json:"created_unix"`
}

// Validate checks the receipt shape and its evidence binding, not the
// model's factual accuracy. The stored fingerprint must be exactly the
// fingerprint of the stored input, so a receipt is reusable only while
// its evidence still matches.
func (r DeliveryReceipt) Validate() error {
	if r.Kind != DeliveryReceiptKind {
		return errors.New("invalid delivery receipt kind")
	}
	if !ValidDigest(r.Fingerprint) {
		return errors.New("delivery receipt lacks its evidence fingerprint")
	}
	if err := r.Input.Validate(); err != nil {
		return err
	}
	bound, err := DeliveryFingerprint(r.Input)
	if err != nil {
		return err
	}
	if bound != r.Fingerprint {
		return errors.New("delivery receipt evidence differs from its fingerprint")
	}
	if len(r.Calls) != 1 || r.Calls[0].Fingerprint != r.Fingerprint {
		return errors.New("delivery receipt lacks its single bound call")
	}
	switch r.Outcome {
	case DeliveryOutcomeDecided:
		if r.Calls[0].Failure != "" {
			return errors.New("decided delivery receipt carries its provider failure")
		}
		if err := r.Calls[0].Answer.Validate(); err != nil {
			return err
		}
		if r.Calls[0].Answer.Fingerprint != r.Fingerprint {
			return errors.New("delivery answer judged different evidence")
		}
		if !reflect.DeepEqual(r.Decision.Answer, r.Calls[0].Answer) {
			return errors.New("delivery decision differs from its raw answer")
		}
		if err := r.Decision.Validate(); err != nil {
			return err
		}
	case DeliveryOutcomeUnavailable:
		if r.Calls[0].Failure == "" {
			return errors.New("unavailable delivery receipt lacks its provider failure")
		}
		if !reflect.DeepEqual(r.Decision, DeliveryDecision{}) {
			return errors.New("unavailable delivery receipt carries a decision")
		}
	default:
		return errors.New("invalid delivery receipt outcome")
	}
	if r.ObservedNativeRevision < 0 || r.CreatedUnix <= 0 {
		return errors.New("invalid delivery receipt observation")
	}
	return nil
}

// CurrentFor reports whether this receipt authorizes reuse for the given
// current fingerprint. Any relevant semantic change (head, base, scope,
// policy, review, finding, fix, check, tree or constraint evidence)
// yields a different fingerprint, so stale evidence never reuses a prior
// decision; timestamp or native-revision-only changes never enter the
// fingerprint and therefore never invalidate it.
func (r DeliveryReceipt) CurrentFor(fingerprint string) bool {
	return r.Validate() == nil && r.Outcome == DeliveryOutcomeDecided && r.Fingerprint == fingerprint
}

// ConfirmExpectedResolution advances the observed native revision after
// an exact expected-resolution readback. Expected resolutions are named
// consequences of the acceptance: their exact confirmed effects may
// advance the native revision without buying another judgment, so this
// pure readback consumes no funded call. It refuses anything but a
// decided receipt and a monotone revision; any unexpected reply,
// content, state, head, base, policy, scope or check-outcome change
// invalidates the acceptance instead of advancing it.
func (r DeliveryReceipt) ConfirmExpectedResolution(revision int64) (DeliveryReceipt, error) {
	if err := r.Validate(); err != nil {
		return DeliveryReceipt{}, err
	}
	if r.Outcome != DeliveryOutcomeDecided {
		return DeliveryReceipt{}, errors.New("unavailable delivery judgment has no expected resolution")
	}
	if revision < r.ObservedNativeRevision {
		return DeliveryReceipt{}, errors.New("expected resolution readback moved backwards")
	}
	next := r
	next.ObservedNativeRevision = revision
	return next, nil
}

// NewDeliveryUnavailable builds the failure receipt for one reserved
// fingerprint without a completed answer. The reservation stays charged:
// the receipt records the failure and carries no decision, so it can
// never authorize merge however much allowance remains.
func NewDeliveryUnavailable(in DeliveryFingerprintInput, fingerprint string, failure string, observedRevision int64) (DeliveryReceipt, error) {
	if failure == "" {
		return DeliveryReceipt{}, errors.New("delivery failure needs its reason")
	}
	r := DeliveryReceipt{
		Kind:                   DeliveryReceiptKind,
		Fingerprint:            fingerprint,
		Input:                  in,
		Calls:                  []DeliveryCall{{Fingerprint: fingerprint, Failure: failure}},
		Outcome:                DeliveryOutcomeUnavailable,
		ObservedNativeRevision: observedRevision,
		CreatedUnix:            time.Now().Unix(),
	}
	if err := r.Validate(); err != nil {
		return DeliveryReceipt{}, err
	}
	return r, nil
}

// NewDeliveryReceipt builds the decided receipt for one completed funded
// answer. The answer must judge exactly the fingerprinted evidence; its
// effective action derives from the remaining review allowance.
func NewDeliveryReceipt(in DeliveryFingerprintInput, fingerprint string, answer DeliveryAnswer, reviewsFit bool, observedRevision int64) (DeliveryReceipt, error) {
	r := DeliveryReceipt{
		Kind:        DeliveryReceiptKind,
		Fingerprint: fingerprint,
		Input:       in,
		Calls:       []DeliveryCall{{Fingerprint: fingerprint, Answer: answer}},
		Decision: DeliveryDecision{
			Answer:     answer,
			Effective:  answer.EffectiveAction(reviewsFit),
			ReviewsFit: reviewsFit,
		},
		Outcome:                DeliveryOutcomeDecided,
		ObservedNativeRevision: observedRevision,
		CreatedUnix:            time.Now().Unix(),
	}
	if err := r.Validate(); err != nil {
		return DeliveryReceipt{}, err
	}
	return r, nil
}
