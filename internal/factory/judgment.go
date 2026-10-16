package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
)

// PriorityCandidate contains observed evidence only. Empty impact, risk and
// effort mean unknown; the classifier must not manufacture estimates.
type PriorityCandidate struct {
	ID                  string         `json:"id"`
	Control             IssueControl   `json:"control"`
	Objective           string         `json:"objective"`
	Acceptance          Acceptance     `json:"acceptance"`
	AcceptedSections    []PromptSource `json:"accepted_sections,omitempty"`
	Authority           AuthorityRef   `json:"authority"`
	Preparation         string         `json:"preparation"`
	PreparationRevision int64          `json:"preparation_revision"`
	Source              string         `json:"source"`
	GateRevision        int64          `json:"gate_revision"`
	PlannedMinutes      int            `json:"planned_minutes"`
	PromptDigest        string         `json:"prompt_digest"`
	Impact              string         `json:"impact,omitempty"`
	Risk                string         `json:"risk,omitempty"`
	Effort              string         `json:"effort,omitempty"`
}

const PriorityRubric = "Choose the eligible issue whose next execution best advances its accepted objective. Prefer evidenced prerequisite leverage and material user impact, then reduce evidenced delivery risk and effort. Use only provided facts; missing impact, risk and effort are unknown, not zero. Queue age breaks equivalent preferences. Eligibility, acceptance, authority, spending and launch are decided by code, never by this preference."

type PriorityRequest struct {
	Snapshot   string              `json:"snapshot"`
	Candidates []PriorityCandidate `json:"candidates"`
	Rubric     string              `json:"rubric"`
	State      json.RawMessage     `json:"state,omitempty"`
	Criteria   map[string]any      `json:"criteria,omitempty"`
}

type PriorityAnswer struct {
	Selected      string             `json:"selected"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
	Model         string             `json:"model"`
	InputTokens   int64              `json:"input_tokens"`
	OutputTokens  int64              `json:"output_tokens"`
}

type PriorityCall struct {
	Request PriorityRequest `json:"request"`
	Answer  PriorityAnswer  `json:"answer"`
	Failure string          `json:"failure,omitempty"`
}

type PriorityReceipt struct {
	Snapshot    string              `json:"snapshot"`
	Purpose     JudgmentPurpose     `json:"purpose,omitempty"`
	Candidates  []PriorityCandidate `json:"candidates"`
	Calls       []PriorityCall      `json:"calls"`
	Selected    string              `json:"selected,omitempty"`
	Outcome     string              `json:"outcome"`
	CreatedUnix int64               `json:"created_unix"`
}

// PrioritySnapshot binds the complete candidate evidence and current rubric.
func PrioritySnapshot(candidates []PriorityCandidate) (string, error) {
	candidates = append([]PriorityCandidate(nil), candidates...)
	for i := range candidates {
		candidates[i].Control.NativeRev = 0
		candidates[i].Control.AssessedUnix = 0
		candidates[i].PromptDigest = ""
	}
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.ID == "" || seen[c.ID] {
			return "", errors.New("invalid priority candidate identities")
		}
		seen[c.ID] = true
	}
	data, err := json.Marshal(struct {
		Rubric     string
		Candidates []PriorityCandidate
	}{PriorityRubric, candidates})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Validate checks coverage and normalization, not the model's factual accuracy.
func (a PriorityAnswer) Validate(request PriorityRequest) error {
	if len(request.Candidates) < 2 || len(request.Candidates) > 255 || a.Model == "" ||
		math.IsNaN(a.Confidence) || a.Confidence < 0 || a.Confidence > 1 ||
		a.InputTokens < 0 || a.OutputTokens < 0 || len(a.Probabilities) != len(request.Candidates) {
		return errors.New("invalid priority answer")
	}
	sum, selected := 0.0, false
	for _, c := range request.Candidates {
		p, ok := a.Probabilities[c.ID]
		if !ok || math.IsNaN(p) || p < 0 || p > 1 {
			return errors.New("incomplete priority distribution")
		}
		sum += p
		selected = selected || a.Selected == c.ID
	}
	if !selected || math.Abs(sum-1) > 0.001 {
		return errors.New("invalid priority selection")
	}
	for _, p := range a.Probabilities {
		if p > a.Probabilities[a.Selected]+0.000001 {
			return errors.New("priority choice is not maximal")
		}
	}
	return nil
}

// JudgmentPurpose names one of the eight bounded System One purposes owned by
// the product planning-and-priority-judgments contract. Every purpose is a
// Choice judgment over caller-supplied evidence and explicit options.
type JudgmentPurpose string

const (
	JudgmentProductApproaches JudgmentPurpose = "product-approaches"
	JudgmentAssumptions       JudgmentPurpose = "assumptions"
	JudgmentConsolidation     JudgmentPurpose = "consolidation"
	JudgmentDecomposition     JudgmentPurpose = "decomposition"
	JudgmentPlanningFreshness JudgmentPurpose = "planning-freshness"
	JudgmentNextWork          JudgmentPurpose = "next-work"
	JudgmentConsistency       JudgmentPurpose = "consistency"
	// JudgmentDeliveryReview is the funded delivery choice (D08): Jev
	// judges the accepted correction set against fixed criteria and
	// returns exactly review_again or ready_to_merge.
	JudgmentDeliveryReview JudgmentPurpose = "delivery-review-v1"
)

// JudgmentPurposes lists the eight purposes in owning-doc order.
func JudgmentPurposes() []JudgmentPurpose {
	return []JudgmentPurpose{JudgmentProductApproaches, JudgmentAssumptions, JudgmentConsolidation, JudgmentDecomposition, JudgmentPlanningFreshness, JudgmentNextWork, JudgmentConsistency, JudgmentDeliveryReview}
}

func (p JudgmentPurpose) valid() bool {
	for _, known := range JudgmentPurposes() {
		if known == p {
			return true
		}
	}
	return false
}

// JudgmentFingerprint binds a purpose to its exact semantic inputs. Identical
// state under different purposes or questions yields different fingerprints,
// so a result is reusable only while its purpose and inputs still match.
func JudgmentFingerprint(purpose JudgmentPurpose, state any, question string, options map[string]any) (string, error) {
	if !purpose.valid() || strings.TrimSpace(question) == "" || len(question) > 16000 || strings.ContainsRune(question, 0) {
		return "", errors.New("invalid judgment purpose or question")
	}
	if len(options) < 2 || len(options) > 255 {
		return "", errors.New("invalid judgment option size")
	}
	for key := range options {
		if strings.TrimSpace(key) == "" || len(key) > 200 || strings.ContainsRune(key, 0) {
			return "", errors.New("invalid judgment option identity")
		}
	}
	data, err := json.Marshal(struct {
		Kind     string
		Purpose  JudgmentPurpose
		State    any
		Question string
		Options  map[string]any
	}{"judgment-v1", purpose, state, question, options})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Delivery options. Exactly these two: a further independent inspection
// is warranted, or the supplied evidence adequately covers the accepted
// outcomes and every actionable finding.
const (
	DeliveryReviewAgain  = "review_again"
	DeliveryReadyToMerge = "ready_to_merge"
)

// Effective delivery actions. The raw answer is always retained alongside
// the effective action; a malformed, missing or provider-failed answer is
// judgment_unavailable and can never authorize merge.
const (
	DeliveryEffectiveReady     = "ready_to_merge"
	DeliveryEffectiveReview    = "review_again"
	DeliveryEffectiveExhausted = "review_allowance_exhausted"
	DeliveryUnavailable        = "judgment_unavailable"
)

// Effective delivery thresholds. Ready requires the selected
// ready_to_merge probability and the answer confidence both at or above
// these conservative product cutoffs; they are not probabilities that
// the code is correct.
const (
	DeliveryReadyMinProbability = 0.8
	DeliveryReadyMinConfidence  = 0.8
)

// DeliveryAnswer is one funded delivery judgment answer: the selected
// option with its probability, the answer confidence, the full
// two-option distribution, the semantic fingerprint it judged and the
// configured model behind it. Failure carries a provider failure instead
// of a choice; a failed answer never validates.
type DeliveryAnswer struct {
	Probabilities map[string]float64 `json:"probabilities"`
	Fingerprint   string             `json:"fingerprint"`
	Selected      string             `json:"selected"`
	Model         string             `json:"model"`
	Failure       string             `json:"failure,omitempty"`
	Probability   float64            `json:"probability"`
	Confidence    float64            `json:"confidence"`
}

// Validate checks the answer shape, not the model's factual accuracy. The
// distribution covers exactly both delivery options; the selected option
// is maximal or tied. Ties route to review at effective-action time.
func (a DeliveryAnswer) Validate() error {
	if a.Failure != "" {
		return errors.New("delivery answer carries its provider failure")
	}
	switch a.Selected {
	case DeliveryReviewAgain, DeliveryReadyToMerge:
	default:
		return errors.New("invalid delivery selection")
	}
	if !ValidDigest(a.Fingerprint) {
		return errors.New("delivery answer lacks its evidence fingerprint")
	}
	if a.Model == "" || len(a.Model) > 128 || strings.IndexFunc(a.Model, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	}) >= 0 {
		return errors.New("invalid delivery answer model")
	}
	for _, value := range []float64{a.Probability, a.Confidence} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return errors.New("invalid delivery answer score")
		}
	}
	if len(a.Probabilities) != 2 {
		return errors.New("incomplete delivery distribution")
	}
	sum := 0.0
	for _, option := range []string{DeliveryReviewAgain, DeliveryReadyToMerge} {
		p, ok := a.Probabilities[option]
		if !ok || math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
			return errors.New("incomplete delivery distribution")
		}
		sum += p
		if p > a.Probabilities[a.Selected]+0.000001 {
			return errors.New("delivery choice is not maximal")
		}
	}
	if math.Abs(sum-1) > 0.001 {
		return errors.New("invalid delivery distribution")
	}
	return nil
}

// EffectiveAction derives the effective action from the raw answer and
// the remaining review allowance. Ready requires the selected
// ready_to_merge, both thresholds met and a strict probability lead;
// below-threshold or tied answers route to review. Malformed or failed
// answers are unavailable however much allowance remains. A clean first
// review still consults: no finding state bypasses this function.
func (a DeliveryAnswer) EffectiveAction(reviewsFit bool) string {
	if a.Validate() != nil {
		return DeliveryUnavailable
	}
	if a.Selected == DeliveryReadyToMerge && a.Probability >= DeliveryReadyMinProbability &&
		a.Confidence >= DeliveryReadyMinConfidence &&
		a.Probabilities[DeliveryReadyToMerge] > a.Probabilities[DeliveryReviewAgain] {
		return DeliveryEffectiveReady
	}
	if reviewsFit {
		return DeliveryEffectiveReview
	}
	return DeliveryEffectiveExhausted
}

// DeliveryDecision retains one raw delivery answer with its derived
// effective action. Both travel together into merge authorization.
type DeliveryDecision struct {
	Answer     DeliveryAnswer `json:"answer"`
	Effective  string         `json:"effective"`
	ReviewsFit bool           `json:"reviews_fit"`
}

// Validate checks the retained pair agrees with the policy function.
func (d DeliveryDecision) Validate() error {
	if err := d.Answer.Validate(); err != nil {
		return err
	}
	switch d.Effective {
	case DeliveryEffectiveReady, DeliveryEffectiveReview, DeliveryEffectiveExhausted, DeliveryUnavailable:
	default:
		return errors.New("invalid delivery effective action")
	}
	if d.Effective != d.Answer.EffectiveAction(d.ReviewsFit) {
		return errors.New("delivery effective action differs from its raw answer")
	}
	return nil
}

// DeliveryFingerprintSchema versions the delivery evidence fingerprint.
const DeliveryFingerprintSchema = "delivery-fingerprint-v1"

// DeliveryFingerprintInput is the complete semantic evidence behind one
// delivery judgment: schema inputs only. Timestamps, the global native
// revision and poll counters are excluded by construction: they are not
// fields here, and the caller keeps the raw observed revision separately
// for native mutation guards. The struct carries no floats, so Go's
// deterministic typed JSON plus SHA-256 cannot emit non-finite numbers.
type DeliveryFingerprintInput struct {
	Reviews        []string `json:"reviews"`
	Conversations  []string `json:"conversations"`
	Constraints    []string `json:"constraints"`
	AttemptRoot    string   `json:"attempt_root"`
	PublicationID  string   `json:"publication_id"`
	ScopeDigest    string   `json:"scope_digest"`
	FixDigest      string   `json:"fix_digest"`
	DiffDigest     string   `json:"diff_digest"`
	CheckDigest    string   `json:"check_digest"`
	TreeDigest     string   `json:"tree_digest"`
	Head           string   `json:"head"`
	Base           string   `json:"base"`
	Model          string   `json:"model"`
	Rubric         string   `json:"rubric"`
	PolicyRevision int64    `json:"policy_revision"`
	Round          int      `json:"round"`
}

// Validate checks the fingerprint inputs. Request state always carries
// the finding digest scope, proposed fixes, diff, checks and tree
// evidence: a clean first review consults with its empty finding set,
// never by skipping the call.
func (in DeliveryFingerprintInput) Validate() error {
	if !ValidID(in.AttemptRoot) || !ValidID(in.PublicationID) || in.Round < 1 {
		return errors.New("invalid delivery fingerprint scope")
	}
	if !ValidCommit(in.Head) || !ValidCommit(in.Base) || in.Head == in.Base {
		return errors.New("delivery fingerprint candidate is not a fresh exact commit")
	}
	for _, digest := range []string{in.ScopeDigest, in.FixDigest, in.DiffDigest, in.CheckDigest, in.TreeDigest} {
		if !ValidDigest(digest) {
			return errors.New("delivery fingerprint lacks its evidence digest")
		}
	}
	if in.PolicyRevision < 0 {
		return errors.New("invalid delivery fingerprint policy")
	}
	if in.Model == "" || len(in.Model) > 128 {
		return errors.New("invalid delivery fingerprint model")
	}
	if strings.TrimSpace(in.Rubric) == "" || len(in.Rubric) > 16384 {
		return errors.New("invalid delivery fingerprint rubric")
	}
	for _, id := range in.Reviews {
		if !ValidID(id) {
			return errors.New("invalid delivery fingerprint review")
		}
	}
	for _, digest := range in.Conversations {
		if !ValidDigest(digest) {
			return errors.New("invalid delivery fingerprint conversation")
		}
	}
	for _, constraint := range in.Constraints {
		if strings.TrimSpace(constraint) == "" || len(constraint) > 1024 {
			return errors.New("invalid delivery fingerprint constraint")
		}
	}
	return nil
}

// DeliveryFingerprint binds the delivery purpose to its exact semantic
// inputs. Set-like collections sort before encoding; ordered tasks and
// diffs keep their order in their digests. Identical evidence under a
// changed head, base, scope, review, fix, check or constraint yields a
// different fingerprint, so a successful unchanged decision is reusable
// only while its evidence still matches.
func DeliveryFingerprint(in DeliveryFingerprintInput) (string, error) {
	if err := in.Validate(); err != nil {
		return "", err
	}
	reviews := append([]string(nil), in.Reviews...)
	conversations := append([]string(nil), in.Conversations...)
	constraints := append([]string(nil), in.Constraints...)
	sort.Strings(reviews)
	sort.Strings(conversations)
	sort.Strings(constraints)
	data, err := json.Marshal(struct {
		Schema         string
		Purpose        JudgmentPurpose
		Options        []string
		AttemptRoot    string
		PublicationID  string
		Round          int
		Head           string
		Base           string
		ScopeDigest    string
		PolicyRevision int64
		Model          string
		Rubric         string
		Reviews        []string
		Conversations  []string
		FixDigest      string
		DiffDigest     string
		CheckDigest    string
		TreeDigest     string
		Constraints    []string
	}{
		DeliveryFingerprintSchema, JudgmentDeliveryReview,
		[]string{DeliveryReviewAgain, DeliveryReadyToMerge},
		in.AttemptRoot, in.PublicationID, in.Round, in.Head, in.Base,
		in.ScopeDigest, in.PolicyRevision, in.Model, in.Rubric,
		reviews, conversations, in.FixDigest, in.DiffDigest,
		in.CheckDigest, in.TreeDigest, constraints,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
