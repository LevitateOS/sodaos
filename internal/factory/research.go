package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// ResearchReportGuidance is the durable report contract for the Researcher
// profile, independent of its shared reviewer execution account.
const ResearchReportGuidance = "Begin with factory.context and factory.research_scope. Read the bound inquiry, prepared source and granted evidence. Use factory.record_research for question-indexed findings and limitations. Finish through factory.finish_research with questions, findings, evidence, conclusion and uncertainty; poll its immutable receipt before ending. Controller acceptance of that typed report completes the assignment. Do not substitute stdout or a code-review verdict, name a candidate, create a PR, edit source or change issue/product policy. If required evidence is unavailable, use factory.finish_incomplete with the concrete gap.\n"

// Research bounds (R10.2). A research report answers declared questions
// with findings and evidence, not code. Counts stay small so reports
// remain reviewable; prose stays within finding bounds.
const (
	MaxResearchQuestions = 16
	MaxResearchFindings  = 64
	MaxResearchEvidence  = 32
)

// ResearchReport is the typed research-phase result (R10.2). Questions
// names the declared inquiry; Findings and Evidence carry the answerable
// results with their source refs; Conclusion states the useful outcome
// and Uncertainty the open remainder. It never names a candidate.
type ResearchReport struct {
	Questions   []string `json:"questions,omitempty"`
	Findings    []string `json:"findings,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Conclusion  string   `json:"conclusion,omitempty"`
	Uncertainty string   `json:"uncertainty,omitempty"`
}

// DecodeResearchReport decodes one finish_research payload into its
// completed report. Unknown fields refuse; the worker never smuggles a
// candidate through an unrecognized key. Blocked research ends through
// finish_incomplete, so only the completed shape settles here.
func DecodeResearchReport(tool MCPTool, raw []byte) (ResearchReport, error) {
	var r ResearchReport
	if tool != MCPToolFinishResearch {
		return r, errors.New("unknown research operation")
	}
	if len(raw) > MCPMaxSubmissionPayload {
		return r, errors.New("research report exceeds 64 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	if err := r.validate("completed"); err != nil {
		return r, err
	}
	return r, nil
}

func (r ResearchReport) validate(status string) error {
	for _, question := range r.Questions {
		if strings.TrimSpace(question) == "" || len(question) > MaxPlanTextBytes {
			return errors.New("research question outside its bound")
		}
	}
	for _, finding := range r.Findings {
		if strings.TrimSpace(finding) == "" || len(finding) > MaxFindingProseBytes {
			return errors.New("research finding outside its bound")
		}
	}
	for _, ref := range r.Evidence {
		if ref == "" || len(ref) > MaxPlanTextBytes || strings.ContainsRune(ref, 0) {
			return errors.New("research evidence outside its bound")
		}
	}
	if strings.ContainsRune(r.Conclusion, 0) || strings.ContainsRune(r.Uncertainty, 0) {
		return errors.New("research prose carries a NUL byte")
	}
	if status == "completed" {
		if len(r.Questions) == 0 || len(r.Questions) > MaxResearchQuestions {
			return errors.New("completed research answers no declared question")
		}
		if strings.TrimSpace(r.Conclusion) == "" || len(r.Conclusion) > MaxFindingProseBytes {
			return errors.New("completed research lacks its conclusion")
		}
		if len(r.Findings) > MaxResearchFindings || len(r.Evidence) > MaxResearchEvidence {
			return errors.New("completed research exceeds its report bound")
		}
		return nil
	}
	if len(r.Questions) > MaxResearchQuestions || len(r.Findings) > MaxResearchFindings ||
		len(r.Evidence) > MaxResearchEvidence {
		return errors.New("research report exceeds its bound")
	}
	return nil
}
