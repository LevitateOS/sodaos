package factory

import (
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestST10ReviewerCannotBecomePublication(t *testing.T) {
	p := testPublication()
	if err := p.Validate(); err != nil {
		t.Fatalf("coder fixture: %v", err)
	}
	p.Role = project.RoleReviewer
	if err := p.Validate(); err == nil {
		t.Fatal("reviewer candidate accepted for publication")
	}
}

func TestParseReviewReport(t *testing.T) {
	good := "notes\n```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```\ntail"
	report, ok := ParseReviewReport(good)
	if !ok || report.Verdict != "approve" || report.Body != "LGTM" {
		t.Fatalf("valid report rejected: %+v %v", report, ok)
	}
	changes := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"line 3 is wrong\",\"findings\":[\"line 3\"]}\n```"
	if report, ok := ParseReviewReport(changes); !ok || report.Verdict != "request-changes" || len(report.Findings) != 1 {
		t.Fatalf("changes report rejected: %+v %v", report, ok)
	}
	for name, output := range map[string]string{
		"missing fence":     "no report here",
		"unterminated":      "```review-json\n{\"verdict\":\"approve\"}",
		"unknown field":     "```review-json\n{\"verdict\":\"approve\",\"summary\":\"s\",\"body\":\"b\",\"findings\":[],\"extra\":1}\n```",
		"bad verdict":       "```review-json\n{\"verdict\":\"maybe\",\"summary\":\"s\",\"body\":\"b\",\"findings\":[]}\n```",
		"changes w/o body":  "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"s\",\"body\":\"  \",\"findings\":[]}\n```",
		"first block stale": "```review-json\n{\"verdict\":\"maybe\"}\n```\n" + changes,
	} {
		t.Run(name, func(t *testing.T) {
			report, ok := ParseReviewReport(output)
			if name == "first block stale" {
				if !ok || report.Verdict != "request-changes" {
					t.Fatalf("last block not preferred: %+v %v", report, ok)
				}
				return
			}
			if ok {
				t.Fatalf("invalid report accepted: %+v", report)
			}
		})
	}
	if got := ReviewAuthRevision("a1", "r1"); got == "" || len(got) > 512 {
		t.Fatalf("review auth revision malformed: %q", got)
	}
}
