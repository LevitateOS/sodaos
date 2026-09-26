package factory

import "errors"

// Result is the process contract. A completed harness reports an outcome; the
// platform still verifies its candidate, independent review and CI evidence.
type Result struct {
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	Candidate    string   `json:"candidate"`
	ReviewPassed bool     `json:"review_passed"`
	Findings     []string `json:"findings"`
}

const ResultSchema = `{"type":"object","additionalProperties":false,"properties":{"status":{"type":"string","enum":["completed","blocked","failed","cancelled"]},"summary":{"type":"string"},"candidate":{"type":"string","pattern":"^([a-f0-9]{40})?$"},"review_passed":{"type":"boolean"},"findings":{"type":"array","items":{"type":"string"}}},"required":["status","summary","candidate","review_passed","findings"]}`

func (r Result) Validate() error {
	switch r.Status {
	case "completed", "blocked", "failed", "cancelled":
	default:
		return errors.New("invalid harness result status")
	}
	if len(r.Summary) > 16<<10 || len(r.Findings) > 64 {
		return errors.New("harness result is too large")
	}
	if r.Status == "completed" && !ValidCommit(r.Candidate) {
		return errors.New("completed result must identify its commit")
	}
	for _, finding := range r.Findings {
		if len(finding) > 4096 {
			return errors.New("review finding is too large")
		}
	}
	return nil
}
