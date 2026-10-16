package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// deliveryChooser is the deterministic FW21 adapter: scripted answers, no
// live Jev consultation.
type deliveryChooser struct {
	calls  int
	answer func(fingerprint string) (factory.DeliveryAnswer, error)
}

func (f *deliveryChooser) SelectDelivery(_ context.Context, state any, _ string, _ map[string]any) (factory.DeliveryAnswer, error) {
	f.calls++
	in, ok := state.(factory.DeliveryFingerprintInput)
	if !ok {
		return factory.DeliveryAnswer{}, errors.New("delivery state is not its evidence input")
	}
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		return factory.DeliveryAnswer{}, err
	}
	return f.answer(fingerprint)
}

func readyDeliveryAnswer(fingerprint string) (factory.DeliveryAnswer, error) {
	return factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.1, factory.DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.85,
	}, nil
}

func reviewDeliveryAnswer(fingerprint string) (factory.DeliveryAnswer, error) {
	return factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.9, factory.DeliveryReadyToMerge: 0.1},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReviewAgain,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.9,
	}, nil
}

// deliveryFunding mirrors the store funding contract in memory:
// attempt-root subledger plus a global ceiling, reserved in one step,
// with fingerprint-keyed receipts.
type deliveryFunding struct {
	subledgerLimit int
	globalBudget   int
	subledger      []string
	requestOwner   map[string]string
	spent          int
	reserved       map[string]bool
	receipts       map[string]factory.DeliveryReceipt
	reserveCalls   int
}

func newDeliveryFunding(subledgerLimit, globalBudget int) *deliveryFunding {
	return &deliveryFunding{
		subledgerLimit: subledgerLimit,
		globalBudget:   globalBudget,
		requestOwner:   map[string]string{},
		reserved:       map[string]bool{},
		receipts:       map[string]factory.DeliveryReceipt{},
	}
}

func (f *deliveryFunding) DeliveryJudgment(_ context.Context, fingerprint string) (factory.DeliveryReceipt, error) {
	r, ok := f.receipts[fingerprint]
	if !ok {
		return factory.DeliveryReceipt{}, store.ErrNotFound
	}
	return r, nil
}

func (f *deliveryFunding) ReserveDeliveryJudgment(_ context.Context, _, _ int64, _, request, fingerprint string) (bool, error) {
	f.reserveCalls++
	if !factory.ValidID(request) || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid delivery reservation")
	}
	if f.reserved[fingerprint] {
		return false, nil
	}
	if owner, seen := f.requestOwner[request]; seen && owner != fingerprint {
		return false, store.ErrCommandConflict
	}
	for _, seen := range f.subledger {
		if seen == request {
			return false, store.ErrCommandConflict
		}
	}
	if len(f.subledger) >= f.subledgerLimit || f.spent >= f.globalBudget {
		return false, nil
	}
	f.subledger = append(append([]string(nil), f.subledger...), request)
	f.requestOwner[request] = fingerprint
	f.spent++
	f.reserved[fingerprint] = true
	return true, nil
}

func (f *deliveryFunding) RecordDeliveryJudgment(_ context.Context, r factory.DeliveryReceipt) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if !f.reserved[r.Fingerprint] {
		return store.ErrCommandConflict
	}
	if _, done := f.receipts[r.Fingerprint]; done {
		return store.ErrCommandConflict
	}
	f.receipts[r.Fingerprint] = r
	return nil
}

func deliveryTestInput() factory.DeliveryFingerprintInput {
	return factory.DeliveryFingerprintInput{
		Reviews:        []string{strings.Repeat("1", 32)},
		Conversations:  []string{strings.Repeat("2", 64)},
		Constraints:    []string{"protected-files:docs/public"},
		AttemptRoot:    strings.Repeat("3", 32),
		PublicationID:  strings.Repeat("4", 32),
		ScopeDigest:    strings.Repeat("5", 64),
		FixDigest:      strings.Repeat("6", 64),
		DiffDigest:     strings.Repeat("7", 64),
		CheckDigest:    strings.Repeat("8", 64),
		TreeDigest:     strings.Repeat("9", 64),
		Head:           strings.Repeat("a", 40),
		Base:           strings.Repeat("b", 40),
		Model:          "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: 3,
		Round:          1,
	}
}

func deliveryTestAdvice(funding *deliveryFunding, chooser *deliveryChooser, in factory.DeliveryFingerprintInput, reviewsFit bool) DeliveryAdvice {
	return DeliveryAdvice{
		Input:                  in,
		Repository:             7,
		Issue:                  3,
		AttemptRoot:            strings.Repeat("3", 32),
		RequestID:              factory.NewID(),
		ReviewsFit:             reviewsFit,
		ObservedNativeRevision: 41,
		Chooser:                chooser,
		Funding:                funding,
	}
}

func TestAdviseDeliveryReadyReusesCompletedResult(t *testing.T) {
	c := &Coordinator{}
	funding := newDeliveryFunding(8, 100)
	chooser := &deliveryChooser{answer: readyDeliveryAnswer}
	in := deliveryTestInput()
	first, cached, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true))
	if err != nil || cached || first.Effective != factory.DeliveryEffectiveReady {
		t.Fatalf("first advice=%+v cached=%v err=%v", first, cached, err)
	}
	if funding.reserveCalls != 1 || len(funding.subledger) != 1 || funding.spent != 1 {
		t.Fatalf("funding not consumed once: %+v", funding)
	}
	second, cached, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true))
	if err != nil || !cached || second.Effective != factory.DeliveryEffectiveReady {
		t.Fatalf("reuse advice=%+v cached=%v err=%v", second, cached, err)
	}
	if chooser.calls != 1 || funding.reserveCalls != 1 || funding.spent != 1 {
		t.Fatal("completed decision bought another paid call")
	}
}

func TestAdviseDeliveryExercisesReviewAgainAndCleanReviewConsults(t *testing.T) {
	c := &Coordinator{}
	funding := newDeliveryFunding(8, 100)
	chooser := &deliveryChooser{answer: reviewDeliveryAnswer}
	in := deliveryTestInput()
	decision, cached, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true))
	if err != nil || cached || decision.Effective != factory.DeliveryEffectiveReview {
		t.Fatalf("review advice=%+v cached=%v err=%v", decision, cached, err)
	}
	if decision.Answer.Selected != factory.DeliveryReviewAgain {
		t.Fatalf("raw choice=%q, want review_again retained", decision.Answer.Selected)
	}
	// A clean first review still consults with its empty finding set.
	clean := deliveryTestInput()
	clean.Reviews, clean.Conversations = nil, nil
	clean.ScopeDigest = strings.Repeat("5", 64)
	before := chooser.calls
	if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, clean, true)); err != nil {
		t.Fatalf("clean review refused: %v", err)
	}
	if chooser.calls != before+1 {
		t.Fatal("clean review bypassed its funded judgment")
	}
	// in is already decided; exhaustion needs fresh evidence.
	fresh := deliveryTestInput()
	fresh.Round = 2
	advice := deliveryTestAdvice(funding, &deliveryChooser{answer: reviewDeliveryAnswer}, fresh, false)
	drained, _, err := c.AdviseDelivery(context.Background(), advice)
	if err != nil || drained.Effective != factory.DeliveryEffectiveExhausted {
		t.Fatalf("exhausted advice=%+v err=%v", drained, err)
	}
}

func TestAdviseDeliveryStaleEvidenceInvalidates(t *testing.T) {
	c := &Coordinator{}
	funding := newDeliveryFunding(8, 100)
	chooser := &deliveryChooser{answer: readyDeliveryAnswer}
	in := deliveryTestInput()
	if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true)); err != nil {
		t.Fatal(err)
	}
	stale := deliveryTestInput()
	stale.CheckDigest = strings.Repeat("0", 64)
	second, cached, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, stale, true))
	if err != nil || cached || second.Effective != factory.DeliveryEffectiveReady {
		t.Fatalf("stale advice=%+v cached=%v err=%v", second, cached, err)
	}
	if chooser.calls != 2 || funding.reserveCalls != 2 || funding.spent != 2 {
		t.Fatal("changed evidence did not buy its own funded call")
	}
}

func TestAdviseDeliveryFailureStaysChargedWithoutRepeat(t *testing.T) {
	c := &Coordinator{}
	funding := newDeliveryFunding(8, 100)
	failing := &deliveryChooser{answer: func(string) (factory.DeliveryAnswer, error) {
		return factory.DeliveryAnswer{}, errors.New("consultation down")
	}}
	in := deliveryTestInput()
	if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, failing, in, true)); !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("failure error=%v", err)
	}
	healthy := &deliveryChooser{answer: readyDeliveryAnswer}
	if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, healthy, in, true)); !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("charged retry error=%v", err)
	}
	if healthy.calls != 0 {
		t.Fatal("charged fingerprint silently repeated the consultation")
	}
	if len(funding.subledger) != 1 || funding.spent != 1 {
		t.Fatal("failure did not retain its funding charge")
	}
}

func TestAdviseDeliveryRejectsForeignAnswers(t *testing.T) {
	c := &Coordinator{}
	for name, answer := range map[string]func(string) (factory.DeliveryAnswer, error){
		"foreign fingerprint": func(fingerprint string) (factory.DeliveryAnswer, error) {
			a, _ := readyDeliveryAnswer(strings.Repeat("0", 64))
			return a, nil
		},
		"foreign model": func(fingerprint string) (factory.DeliveryAnswer, error) {
			a, _ := readyDeliveryAnswer(fingerprint)
			a.Model = "other-model"
			return a, nil
		},
		"below threshold": func(fingerprint string) (factory.DeliveryAnswer, error) {
			a, _ := readyDeliveryAnswer(fingerprint)
			a.Probability = 0.79
			return a, nil
		},
	} {
		funding := newDeliveryFunding(8, 100)
		chooser := &deliveryChooser{answer: answer}
		in := deliveryTestInput()
		decision, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true))
		if name == "below threshold" {
			// D08 (FW36): a threshold failure waits with
			// judgment_attention, never an automatic review.
			if err != nil || decision.Effective != factory.DeliveryEffectiveAttention {
				t.Fatalf("%s routed %+v err=%v", name, decision, err)
			}
			continue
		}
		if !errors.Is(err, ErrDeliveryUnavailable) {
			t.Fatalf("%s error=%v", name, err)
		}
	}
}

func TestAdviseDeliveryReserveRefusalStaysUnavailable(t *testing.T) {
	c := &Coordinator{}
	t.Run("no subledger funds", func(t *testing.T) {
		funding := newDeliveryFunding(0, 100)
		chooser := &deliveryChooser{answer: readyDeliveryAnswer}
		if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, deliveryTestInput(), true)); !errors.Is(err, ErrDeliveryUnavailable) {
			t.Fatalf("unfunded error=%v", err)
		}
		if chooser.calls != 0 {
			t.Fatal("unreserved consultation was attempted")
		}
	})
	t.Run("no global ceiling", func(t *testing.T) {
		funding := newDeliveryFunding(8, 0)
		chooser := &deliveryChooser{answer: readyDeliveryAnswer}
		if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, deliveryTestInput(), true)); !errors.Is(err, ErrDeliveryUnavailable) {
			t.Fatalf("ceiling error=%v", err)
		}
		if chooser.calls != 0 {
			t.Fatal("unreserved consultation was attempted")
		}
	})
	t.Run("missing dependencies", func(t *testing.T) {
		funding := newDeliveryFunding(8, 100)
		chooser := &deliveryChooser{answer: readyDeliveryAnswer}
		advice := deliveryTestAdvice(funding, chooser, deliveryTestInput(), true)
		advice.Chooser = nil
		if _, _, err := c.AdviseDelivery(context.Background(), advice); !errors.Is(err, ErrDeliveryUnavailable) {
			t.Fatalf("nil chooser error=%v", err)
		}
	})
}

func TestAdviseDeliveryReadbackAddsNoPaidCall(t *testing.T) {
	c := &Coordinator{}
	funding := newDeliveryFunding(8, 100)
	chooser := &deliveryChooser{answer: readyDeliveryAnswer}
	in := deliveryTestInput()
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.AdviseDelivery(context.Background(), deliveryTestAdvice(funding, chooser, in, true)); err != nil {
		t.Fatal(err)
	}
	receipt, err := funding.DeliveryJudgment(context.Background(), fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	reserveCalls, spent, calls := funding.reserveCalls, funding.spent, len(funding.subledger)
	advanced, err := receipt.ConfirmExpectedResolution(42)
	if err != nil || advanced.ObservedNativeRevision != 42 {
		t.Fatalf("readback=%+v err=%v", advanced, err)
	}
	if funding.reserveCalls != reserveCalls || funding.spent != spent || len(funding.subledger) != calls {
		t.Fatal("expected resolution readback consumed a funded call")
	}
	if !advanced.CurrentFor(fingerprint) {
		t.Fatal("exact expected readback invalidated its acceptance")
	}
}
