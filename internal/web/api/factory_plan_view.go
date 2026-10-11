package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// planCandidateView renders the plan record's candidate binding: identity
// and readiness only. Native issue text stays out; the plan carries IDs
// and codes, never the objective prose behind them.
type planCandidateView struct {
	ID            string `json:"id"`
	Repository    string `json:"repository"`
	Issue         string `json:"issue"`
	Readiness     string `json:"readiness"`
	Acceptance    string `json:"acceptance,omitempty"`
	FirstSeenUnix int64  `json:"first_seen_unix"`
}

func planCandidateDTO(c factory.PriorityCandidate) planCandidateView {
	return planCandidateView{
		ID:         c.ID,
		Repository: strconv.FormatInt(c.Control.Repository, 10),
		Issue:      strconv.FormatInt(c.Control.Issue, 10),
		Readiness:  c.Control.Readiness, Acceptance: c.Control.Acceptance,
		FirstSeenUnix: c.Control.FirstSeenUnix,
	}
}

// planJudgmentView renders the recorded planning judgment: the selected
// option, its uncertainty and cost. Probabilities carry the deciding
// distribution only when bounded; larger maps report their count.
type planJudgmentView struct {
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Selected      string             `json:"selected,omitempty"`
	Model         string             `json:"model,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	InputTokens   int64              `json:"input_tokens,omitempty"`
	OutputTokens  int64              `json:"output_tokens,omitempty"`
	ProbabilityN  int                `json:"probability_count"`
}

// maxRenderedProbabilities bounds one rendered answer distribution. Stored
// answers are not revalidated on read, so rendering caps the map and
// reports the recorded count.
const maxRenderedProbabilities = 256

func planJudgmentDTO(a factory.PriorityAnswer) *planJudgmentView {
	if a.Selected == "" && a.Model == "" && len(a.Probabilities) == 0 {
		return nil
	}
	view := &planJudgmentView{
		Selected: a.Selected, Model: a.Model, Confidence: a.Confidence,
		InputTokens: a.InputTokens, OutputTokens: a.OutputTokens,
		ProbabilityN: len(a.Probabilities),
	}
	if len(a.Probabilities) <= maxRenderedProbabilities {
		view.Probabilities = a.Probabilities
	}
	return view
}

// planView is the recorded implementation plan for one native issue: the
// approach, steps and provisional prerequisites behind it. Absence of a
// record means no plan was recorded yet, never an implicit approach.
type planView struct {
	Candidate   planCandidateView `json:"candidate"`
	Judgment    *planJudgmentView `json:"judgment,omitempty"`
	Missing     []factory.Blocker `json:"missing"`
	Steps       []string          `json:"steps"`
	Strategy    string            `json:"strategy"`
	Snapshot    string            `json:"snapshot"`
	Outcome     string            `json:"outcome"`
	CreatedUnix int64             `json:"created_unix"`
}

func planDTO(r factory.PlanningRecord) *planView {
	steps := r.Steps
	if steps == nil {
		steps = []string{}
	}
	missing := r.Input.Missing
	if missing == nil {
		missing = []factory.Blocker{}
	}
	return &planView{
		Candidate: planCandidateDTO(r.Input.Candidate), Judgment: planJudgmentDTO(r.Judgment),
		Missing: missing, Steps: steps, Strategy: r.Strategy,
		Snapshot: r.Snapshot, Outcome: r.Outcome, CreatedUnix: r.CreatedUnix,
	}
}

// planFreshnessStates name the plan's currency. Provisional plans carry
// missing prerequisites; stale plans no longer match current readiness.
const (
	planFreshMissing     = "missing"
	planFreshStale       = "stale"
	planFreshProvisional = "provisional"
	planFreshCurrent     = "current"
)

// planFreshnessReasons are the bounded codes behind a stale plan.
const (
	planReasonNoReadiness = "current_readiness_missing"
	planReasonAcceptance  = "acceptance_changed"
	planReasonInputs      = "inputs_changed"
)

type planFreshnessView struct {
	Reasons     []string `json:"reasons,omitempty"`
	State       string   `json:"state"`
	Provisional bool     `json:"provisional"`
}

type planFreshnessInput struct {
	PlanAcceptance     string
	PlanFingerprint    string
	ControlAcceptance  string
	ControlFingerprint string
	HasPlan            bool
	Provisional        bool
	HasControl         bool
}

func planFreshnessDTO(in planFreshnessInput) planFreshnessView {
	if !in.HasPlan {
		return planFreshnessView{State: planFreshMissing}
	}
	view := planFreshnessView{Provisional: in.Provisional}
	switch {
	case !in.HasControl:
		view.Reasons = []string{planReasonNoReadiness}
	case in.PlanAcceptance != in.ControlAcceptance:
		view.Reasons = []string{planReasonAcceptance}
	case in.PlanFingerprint != in.ControlFingerprint:
		view.Reasons = []string{planReasonInputs}
	}
	switch {
	case len(view.Reasons) > 0:
		view.State = planFreshStale
	case in.Provisional:
		view.State = planFreshProvisional
	default:
		view.State = planFreshCurrent
	}
	return view
}

// priorityCandidateView renders one comparison candidate: identity,
// readiness, queue age and which optional evidence was known. Native
// text and blocker details stay out; cross-repository candidates are
// counted, never enumerated.
type priorityCandidateView struct {
	BlockerCodes []string `json:"blocker_codes,omitempty"`
	ID           string   `json:"id"`
	Repository   string   `json:"repository"`
	Issue        string   `json:"issue"`
	Readiness    string   `json:"readiness"`
	Reason       string   `json:"reason,omitempty"`
	Acceptance   string   `json:"acceptance,omitempty"`
	FirstSeen    int64    `json:"first_seen_unix"`
	Assessed     int64    `json:"assessed_unix"`
	AgeSeconds   int64    `json:"age_seconds"`
	ImpactKnown  bool     `json:"impact_known"`
	RiskKnown    bool     `json:"risk_known"`
	EffortKnown  bool     `json:"effort_known"`
}

func priorityCandidateDTO(c factory.PriorityCandidate, observed int64) priorityCandidateView {
	view := priorityCandidateView{
		ID:         c.ID,
		Repository: strconv.FormatInt(c.Control.Repository, 10),
		Issue:      strconv.FormatInt(c.Control.Issue, 10),
		Readiness:  c.Control.Readiness, Reason: c.Control.Reason, Acceptance: c.Control.Acceptance,
		FirstSeen: c.Control.FirstSeenUnix, Assessed: c.Control.AssessedUnix,
		ImpactKnown: c.Impact != "", RiskKnown: c.Risk != "", EffortKnown: c.Effort != "",
	}
	if age := observed - c.Control.FirstSeenUnix; age > 0 {
		view.AgeSeconds = age
	}
	for _, blocker := range c.Control.Blockers {
		view.BlockerCodes = append(view.BlockerCodes, blocker.Code)
	}
	return view
}

// priorityCallView renders one recorded judgment call: the deciding
// model, its choice, uncertainty and cost. The call's request
// candidates stay out; the comparison carries them once.
type priorityCallView struct {
	Selected     string  `json:"selected,omitempty"`
	Model        string  `json:"model,omitempty"`
	Failure      string  `json:"failure,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	InputTokens  int64   `json:"input_tokens,omitempty"`
	OutputTokens int64   `json:"output_tokens,omitempty"`
}

func priorityCallDTO(c factory.PriorityCall) priorityCallView {
	return priorityCallView{
		Selected: c.Answer.Selected, Model: c.Answer.Model, Failure: c.Failure,
		Confidence:  c.Answer.Confidence,
		InputTokens: c.Answer.InputTokens, OutputTokens: c.Answer.OutputTokens,
	}
}

// priorityComparisonView renders one recorded next-work comparison over
// the requested repository: the eligible candidates the journal named,
// the recorded choice and its uncertainty. Candidates from other
// repositories count toward the total without enumerating their
// identities; a masked choice renders empty with SelectedVisible false,
// and the deciding distribution covers rendered candidates only with
// ProbabilitiesFull reporting whether anything was withheld.
type priorityComparisonView struct {
	Candidates        []priorityCandidateView `json:"candidates"`
	Calls             []priorityCallView      `json:"calls"`
	Probabilities     map[string]float64      `json:"probabilities,omitempty"`
	Snapshot          string                  `json:"snapshot"`
	Selected          string                  `json:"selected,omitempty"`
	Outcome           string                  `json:"outcome"`
	CreatedUnix       int64                   `json:"created_unix"`
	CandidatesTotal   int                     `json:"candidates_total"`
	OtherCount        int                     `json:"other_count"`
	ProbabilityN      int                     `json:"probability_count"`
	SelectedVisible   bool                    `json:"selected_visible"`
	ProbabilitiesFull bool                    `json:"probabilities_full"`
}

func priorityComparisonDTO(r factory.PriorityReceipt, repository int64, observed int64) priorityComparisonView {
	view := priorityComparisonView{
		Candidates: []priorityCandidateView{}, Calls: []priorityCallView{},
		Snapshot: r.Snapshot, Selected: r.Selected, Outcome: r.Outcome,
		CreatedUnix: r.CreatedUnix, CandidatesTotal: len(r.Candidates),
	}
	visible := map[string]bool{}
	for _, candidate := range r.Candidates {
		if candidate.Control.Repository != repository {
			view.OtherCount++
			continue
		}
		visible[candidate.ID] = true
		view.Candidates = append(view.Candidates, priorityCandidateDTO(candidate, observed))
	}
	if view.Selected = r.Selected; r.Selected != "" && !visible[r.Selected] {
		view.Selected = ""
	} else {
		view.SelectedVisible = true
	}
	for _, call := range r.Calls {
		view.Calls = append(view.Calls, priorityCallDTO(call))
	}
	if len(r.Calls) > 0 {
		final := r.Calls[len(r.Calls)-1].Answer
		view.ProbabilityN = len(final.Probabilities)
		if len(final.Probabilities) <= maxRenderedProbabilities {
			view.Probabilities = map[string]float64{}
			for id, probability := range final.Probabilities {
				if visible[id] {
					view.Probabilities[id] = probability
				}
			}
			view.ProbabilitiesFull = len(view.Probabilities) == len(final.Probabilities)
		}
	}
	return view
}

// factoryIssuePlanView is the plan and priority read for one native
// issue: the recorded plan with its freshness, and the recent recorded
// next-work comparisons naming the issue, newest first. Comparisons
// page by trailing cursor; absent plans and empty comparison pages are
// explicit, never guesses.
type factoryIssuePlanView struct {
	Plan               *planView                `json:"plan,omitempty"`
	Comparisons        []priorityComparisonView `json:"comparisons"`
	Freshness          planFreshnessView        `json:"freshness"`
	Repository         string                   `json:"repository"`
	Issue              string                   `json:"issue"`
	NextBeforeSnapshot string                   `json:"next_before_snapshot,omitempty"`
	ObservedUnix       int64                    `json:"observed_unix"`
	NextBeforeUnix     int64                    `json:"next_before_unix,omitempty"`
}

func parsePlanComparisonQuery(raw string) (store.PriorityComparisonCursor, int, bool) {
	var cursor store.PriorityComparisonCursor
	if len(raw) > 2048 {
		return cursor, 0, false
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return cursor, 0, false
	}
	for key := range values {
		if key != "limit" && key != "before_unix" && key != "before_snapshot" {
			return cursor, 0, false
		}
	}
	limit := store.DefaultPriorityComparisons
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > store.MaxPriorityComparisons {
			return cursor, 0, false
		}
		limit = parsed
	}
	beforeUnix := values.Get("before_unix")
	beforeSnapshot := values.Get("before_snapshot")
	if beforeUnix == "" && beforeSnapshot == "" {
		return cursor, limit, true
	}
	parsed, err := strconv.ParseInt(beforeUnix, 10, 64)
	if err != nil || parsed <= 0 || !factory.ValidDigest(beforeSnapshot) {
		return cursor, 0, false
	}
	return store.PriorityComparisonCursor{BeforeUnix: parsed, BeforeSnapshot: beforeSnapshot}, limit, true
}

// apiFactoryIssuePlan shows the recorded plan and recent priority
// comparisons for one native issue. It reads the planning journal and
// priority receipts only; it never makes a judgment, provisions or
// dispatches work.
func (s *API) apiFactoryIssuePlan(w http.ResponseWriter, r *http.Request, v store.Session) {
	cursor, limit, ok := parsePlanComparisonQuery(r.URL.RawQuery)
	if r.URL.ForceQuery || !ok {
		auth.JSONError(w, 400, "invalid_request", "Use limit 1..20 with an optional before_unix/before_snapshot cursor pair.")
		return
	}
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return
	}
	observed := time.Now().Unix()
	view := factoryIssuePlanView{
		Comparisons: []priorityComparisonView{},
		Repository:  strconv.FormatInt(repository, 10), Issue: issue, ObservedUnix: observed,
	}
	index, _ := strconv.ParseInt(issue, 10, 64)
	freshness := planFreshnessInput{}
	if record, err := s.Store.PlanningRecord(r.Context(), repository, index); err == nil {
		freshness.HasPlan, freshness.Provisional = true, record.Provisional
		freshness.PlanAcceptance, freshness.PlanFingerprint = record.Input.Candidate.Acceptance.ID, record.Input.Candidate.Control.Fingerprint
		view.Plan = planDTO(record)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue plan.")
		return
	}
	if control, err := s.Store.IssueControl(r.Context(), repository, index); err == nil {
		freshness.HasControl = true
		freshness.ControlAcceptance, freshness.ControlFingerprint = control.Acceptance, control.Fingerprint
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue readiness.")
		return
	}
	view.Freshness = planFreshnessDTO(freshness)
	receipts, hasMore, err := s.Store.PriorityReceiptsForIssue(r.Context(), repository, index, cursor, limit)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read priority comparisons.")
		return
	}
	for _, receipt := range receipts {
		view.Comparisons = append(view.Comparisons, priorityComparisonDTO(receipt, repository, observed))
	}
	if hasMore && len(receipts) > 0 {
		trailing := receipts[len(receipts)-1]
		view.NextBeforeUnix, view.NextBeforeSnapshot = trailing.CreatedUnix, trailing.Snapshot
	}
	auth.JSONResponse(w, 200, view)
}
