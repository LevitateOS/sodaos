package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// Progressive research records (R32.13). A researcher declares its
// inquiry and saves attributable findings, experiments and open
// questions during the work instead of dumping everything at finish:
// research_scope reads the declared scope with per-question coverage,
// record_research appends one bounded batch, finish_research settles
// the typed report unchanged. Records never manufacture authority —
// only the researcher's own assignment log grows.
const (
	MaxResearchRecordExperiments = 32
	MaxResearchRecordUnanswered  = 16
	MaxResearchRecordSources     = 32
)

// ResearchRecordFinding ties one finding to its declared question by
// index into the assignment's cumulative question list.
type ResearchRecordFinding struct {
	Question int      `json:"question"`
	Finding  string   `json:"finding"`
	Evidence []string `json:"evidence,omitempty"`
}

// ResearchRecord is one bounded record_research batch: newly declared
// questions, findings against the cumulative question list, observed
// experiments, still-unanswered question indices and inspected
// sources. Empty batches refuse: a record must declare or save.
type ResearchRecord struct {
	Questions   []string                `json:"questions,omitempty"`
	Findings    []ResearchRecordFinding `json:"findings,omitempty"`
	Experiments []string                `json:"experiments,omitempty"`
	Unanswered  []int                   `json:"unanswered,omitempty"`
	Sources     []string                `json:"sources,omitempty"`
}

// ResearchLog is the accumulated per-assignment record state.
type ResearchLog struct {
	Questions   []string                `json:"questions"`
	Findings    []ResearchRecordFinding `json:"findings"`
	Experiments []string                `json:"experiments"`
	Unanswered  []int                   `json:"unanswered"`
	Sources     []string                `json:"sources"`
	Records     int                     `json:"records"`
}

// ResearchQuestionCoverage marks one declared question answered when
// at least one recorded finding cites it.
type ResearchQuestionCoverage struct {
	Question string `json:"question"`
	Answered bool   `json:"answered"`
	Findings int    `json:"findings"`
}

// ResearchScopeView is the research_scope read: the admitted subject
// plus the declared inquiry with coverage and the recorded batches.
type ResearchScopeView struct {
	Assignment  string                     `json:"assignment"`
	Issue       int64                      `json:"issue"`
	Acceptance  string                     `json:"acceptance"`
	Sources     []string                   `json:"sources,omitempty"`
	Questions   []ResearchQuestionCoverage `json:"questions"`
	Findings    []ResearchRecordFinding    `json:"findings"`
	Experiments []string                   `json:"experiments,omitempty"`
	Unanswered  []string                   `json:"unanswered,omitempty"`
	Records     int                        `json:"records"`
}

// DecodeResearchRecord decodes one record_research payload. Unknown
// fields refuse and empty batches refuse; question-index bounds apply
// against the caller's cumulative question count at append time.
func DecodeResearchRecord(tool MCPTool, raw []byte) (ResearchRecord, error) {
	var r ResearchRecord
	if tool != MCPToolRecordResearch {
		return r, errors.New("unknown research operation")
	}
	if len(raw) > MCPMaxSubmissionPayload {
		return r, errors.New("research record exceeds 64 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	if err := r.validate(); err != nil {
		return r, err
	}
	return r, nil
}

func validResearchProse(s string) bool {
	return strings.TrimSpace(s) != "" && len(s) <= MaxFindingProseBytes && !strings.ContainsRune(s, 0)
}

func (r ResearchRecord) validate() error {
	if len(r.Questions) == 0 && len(r.Findings) == 0 && len(r.Experiments) == 0 && len(r.Unanswered) == 0 && len(r.Sources) == 0 {
		return errors.New("research record declares and saves nothing")
	}
	if len(r.Questions) > MaxResearchQuestions {
		return errors.New("research record declares too many questions")
	}
	for _, q := range r.Questions {
		if !validResearchProse(q) {
			return errors.New("research question outside its bound")
		}
	}
	if len(r.Findings) > MaxResearchFindings {
		return errors.New("research record saves too many findings")
	}
	for _, f := range r.Findings {
		if f.Question < 0 {
			return errors.New("research finding cites no question")
		}
		if !validResearchProse(f.Finding) {
			return errors.New("research finding outside its bound")
		}
		if len(f.Evidence) > MaxResearchEvidence {
			return errors.New("research finding cites too much evidence")
		}
		for _, ref := range f.Evidence {
			if ref == "" || len(ref) > MaxPlanTextBytes || strings.ContainsRune(ref, 0) {
				return errors.New("research evidence outside its bound")
			}
		}
	}
	if len(r.Experiments) > MaxResearchRecordExperiments {
		return errors.New("research record saves too many experiments")
	}
	for _, e := range r.Experiments {
		if !validResearchProse(e) {
			return errors.New("research experiment outside its bound")
		}
	}
	if len(r.Unanswered) > MaxResearchRecordUnanswered {
		return errors.New("research record leaves too many questions unanswered")
	}
	for _, u := range r.Unanswered {
		if u < 0 {
			return errors.New("research record leaves no question unanswered")
		}
	}
	if len(r.Sources) > MaxResearchRecordSources {
		return errors.New("research record cites too many sources")
	}
	for _, s := range r.Sources {
		if !validResearchProse(s) {
			return errors.New("research source outside its bound")
		}
	}
	return nil
}

// AppendResearchRecord folds one decoded batch into the cumulative
// log. Questions append first so findings cite post-append indices;
// the whole batch refuses when any cumulative bound or index misses.
func AppendResearchRecord(log ResearchLog, r ResearchRecord) (ResearchLog, error) {
	out := log
	if out.Questions == nil {
		out.Questions = []string{}
	}
	if out.Findings == nil {
		out.Findings = []ResearchRecordFinding{}
	}
	for _, q := range r.Questions {
		if len(out.Questions) >= MaxResearchQuestions {
			return log, errors.New("research inquiry declares too many questions")
		}
		out.Questions = append(out.Questions, q)
	}
	for _, f := range r.Findings {
		if f.Question >= len(out.Questions) {
			return log, errors.New("research finding cites an undeclared question")
		}
		if len(out.Findings) >= MaxResearchFindings {
			return log, errors.New("research inquiry saves too many findings")
		}
		out.Findings = append(out.Findings, f)
	}
	for _, e := range r.Experiments {
		if len(out.Experiments) >= MaxResearchRecordExperiments {
			return log, errors.New("research inquiry saves too many experiments")
		}
		out.Experiments = append(out.Experiments, e)
	}
	for _, u := range r.Unanswered {
		if u >= len(out.Questions) {
			return log, errors.New("research record leaves an undeclared question unanswered")
		}
		out.Unanswered = append(out.Unanswered, u)
	}
	for _, s := range r.Sources {
		if len(out.Sources) >= MaxResearchRecordSources {
			return log, errors.New("research inquiry cites too many sources")
		}
		out.Sources = append(out.Sources, s)
	}
	out.Records++
	return out, nil
}

// CoverResearchLog renders the scope view over the cumulative log:
// per-question answered flags with finding counts and resolved
// unanswered question texts.
func CoverResearchLog(assignment string, issue int64, acceptance string, declared []string, log ResearchLog) ResearchScopeView {
	view := ResearchScopeView{
		Assignment: assignment, Issue: issue, Acceptance: acceptance,
		Sources: declared, Records: log.Records,
		Questions: []ResearchQuestionCoverage{}, Findings: log.Findings,
		Experiments: log.Experiments, Unanswered: []string{},
	}
	if view.Findings == nil {
		view.Findings = []ResearchRecordFinding{}
	}
	counts := make([]int, len(log.Questions))
	for _, f := range log.Findings {
		if f.Question >= 0 && f.Question < len(counts) {
			counts[f.Question]++
		}
	}
	for i, q := range log.Questions {
		view.Questions = append(view.Questions, ResearchQuestionCoverage{Question: q, Answered: counts[i] > 0, Findings: counts[i]})
	}
	seen := map[int]bool{}
	for _, u := range log.Unanswered {
		if u < 0 || u >= len(log.Questions) || seen[u] {
			continue
		}
		seen[u] = true
		view.Unanswered = append(view.Unanswered, log.Questions[u])
	}
	return view
}
