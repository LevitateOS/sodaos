package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrDeliveryUnavailable reports a delivery judgment that cannot consult:
// funding refused or exhausted, a reserved fingerprint without a completed
// answer, or a malformed, missing or provider-failed answer. It never
// authorizes merge.
var ErrDeliveryUnavailable = errors.New("delivery judgment unavailable or exhausted")

// DeliveryChooser is one funded delivery consultation. Code supplies the
// complete semantic evidence, the fixed delivery rubric and exactly the
// two delivery options; the chooser supplies only the choice. The caller
// must reserve attempt-root and global funding before selecting.
type DeliveryChooser interface {
	SelectDelivery(context.Context, any, string, map[string]any) (factory.DeliveryAnswer, error)
}

// DeliveryFunding is the durable funded-delivery ledger: attempt-root
// call subledger plus the existing global judgment ceiling in one
// transaction, with fingerprint-keyed decision persistence. It is
// satisfied by *store.Store; the assertion below keeps the joint
// compile-checked for the FW22 real correction/cycle wiring.
type DeliveryFunding interface {
	DeliveryJudgment(context.Context, string) (factory.DeliveryReceipt, error)
	ReserveDeliveryJudgment(context.Context, int64, int64, string, string, string) (bool, error)
	RecordDeliveryJudgment(context.Context, factory.DeliveryReceipt) error
}

var _ DeliveryFunding = (*store.Store)(nil)

// DeliveryAdvice is one funded delivery-review-v1 request. Input is the
// complete semantic evidence; RequestID is the immutable funded-call
// identity charged to the attempt-root subledger (FW22 supplies
// cycle-scoped IDs); ReviewsFit reports whether another review fits the
// remaining allowance; ObservedNativeRevision is the raw native revision
// kept beside the receipt for mutation guards. Real correction/cycle
// paths integrate in FW22: callers must pass deterministic
// evidence/authority/check prerequisites first, because no paid request
// occurs when those already block, and a clean first review still
// consults through this same path with its empty finding set.
type DeliveryAdvice struct {
	Input                  factory.DeliveryFingerprintInput
	State                  any
	Repository             int64
	Issue                  int64
	AttemptRoot            string
	RequestID              string
	ReviewsFit             bool
	ObservedNativeRevision int64
	Chooser                DeliveryChooser
	Funding                DeliveryFunding
}

// AdviseDelivery returns the funded delivery choice for the exact
// semantic evidence. A matching completed receipt reuses without a new
// reservation or consultation. Otherwise funding reserves the
// attempt-root subledger and the global ceiling before exactly one
// chooser call, and the raw answer with its effective action persists.
// Failures retain consumption: a reserved fingerprint without a
// completed answer stays charged and unavailable, with no refund,
// upstream lookup or silent repeat of unchanged input. Stale semantic
// evidence never reuses a prior decision: any evidence change yields a
// different fingerprint and consults anew.
func (c *Coordinator) AdviseDelivery(ctx context.Context, advice DeliveryAdvice) (factory.DeliveryDecision, bool, error) {
	if advice.Chooser == nil || advice.Funding == nil {
		return factory.DeliveryDecision{}, false, ErrDeliveryUnavailable
	}
	state := advice.State
	if state == nil {
		state = advice.Input
	}
	if validator, ok := advice.Chooser.(interface{ ValidateDeliveryState(any) error }); ok {
		if err := validator.ValidateDeliveryState(state); err != nil {
			return factory.DeliveryDecision{}, false, err
		}
	}
	fingerprint, err := factory.DeliveryFingerprint(advice.Input)
	if err != nil {
		return factory.DeliveryDecision{}, false, err
	}
	if advice.ObservedNativeRevision < 0 {
		return factory.DeliveryDecision{}, false, errors.New("invalid delivery observation")
	}
	prior, err := advice.Funding.DeliveryJudgment(ctx, fingerprint)
	if err == nil {
		if prior.CurrentFor(fingerprint) {
			return prior.Decision, true, nil
		}
		return factory.DeliveryDecision{}, false, ErrDeliveryUnavailable
	}
	if !errors.Is(err, store.ErrNotFound) {
		return factory.DeliveryDecision{}, false, err
	}
	reserved, err := advice.Funding.ReserveDeliveryJudgment(ctx, advice.Repository, advice.Issue, advice.AttemptRoot, advice.RequestID, fingerprint)
	if err != nil {
		return factory.DeliveryDecision{}, false, err
	}
	if !reserved {
		return factory.DeliveryDecision{}, false, ErrDeliveryUnavailable
	}
	options := factory.DeliveryOptions()
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	answer, callErr := advice.Chooser.SelectDelivery(bounded, state, factory.DeliveryRubric, options)
	cancel()
	if callErr == nil {
		callErr = answer.Validate()
	}
	if callErr == nil && (answer.Fingerprint != fingerprint || answer.Model != advice.Input.Model) {
		callErr = errors.New("delivery answer judged different evidence")
	}
	if callErr != nil {
		receipt, recordErr := factory.NewDeliveryUnavailable(advice.Input, fingerprint, "consultation_failed", advice.ObservedNativeRevision)
		if recordErr != nil {
			return factory.DeliveryDecision{}, false, recordErr
		}
		if err := advice.Funding.RecordDeliveryJudgment(ctx, receipt); err != nil {
			return factory.DeliveryDecision{}, false, err
		}
		return factory.DeliveryDecision{}, false, ErrDeliveryUnavailable
	}
	receipt, err := factory.NewDeliveryReceipt(advice.Input, fingerprint, answer, advice.ReviewsFit, advice.ObservedNativeRevision)
	if err != nil {
		return factory.DeliveryDecision{}, false, err
	}
	if err := advice.Funding.RecordDeliveryJudgment(ctx, receipt); err != nil {
		return factory.DeliveryDecision{}, false, err
	}
	return receipt.Decision, false, nil
}
