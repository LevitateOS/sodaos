package factory

import (
	"reflect"
	"strings"
	"testing"
)

func TestDescribeAlternativesCoversRepresentativeObjectives(t *testing.T) {
	for objective, pattern := range map[string]string{
		"I need a time-off request my manager can approve or decline with a reason": "decision",
		"Let new hires submit onboarding details so staff can complete their intake": "intake",
	} {
		first, second := DescribeAlternatives(objective), DescribeAlternatives(objective)
		if !reflect.DeepEqual(first, second) {
			t.Fatal("authored alternatives are not deterministic")
		}
		if err := first.Validate(); err != nil {
			t.Fatalf("objective %q: %v", objective, err)
		}
		if !first.Supported || first.Pattern != pattern || len(first.Alternatives) != 2 || len(first.Unknowns) == 0 {
			t.Fatalf("objective %q: unexpected set %+v", objective, first)
		}
		for _, alternative := range first.Alternatives {
			if len(alternative.Outcomes) != 1 || alternative.Outcomes[0] != objective {
				t.Fatal("alternative did not preserve the full objective")
			}
			for _, step := range alternative.Steps {
				if step.Actor == "" || step.Action == "" || step.Result == "" {
					t.Fatal("walkthrough step misses actor, action or result")
				}
			}
			rendered := alternative.Audience + alternative.Summary + alternative.Visibility + alternative.Consequences
			for _, step := range alternative.Steps {
				rendered += step.Actor + step.Action + step.Result
			}
			for _, invented := range []string{"email", "calendar", "payment"} {
				if strings.Contains(strings.ToLower(rendered), invented) {
					t.Fatalf("authored content silently adds %s", invented)
				}
			}
		}
	}
}

func TestDescribeAlternativesReportsUnsupportedObjectivesHonestly(t *testing.T) {
	for _, objective := range []string{"", "Build a database storage engine", strings.Repeat("x", 16001)} {
		set := DescribeAlternatives(objective)
		if err := set.Validate(); err != nil {
			t.Fatalf("objective %q: %v", objective, err)
		}
		if set.Supported || len(set.Alternatives) != 0 || len(set.Unknowns) == 0 {
			t.Fatalf("objective %q: unsupported set %+v", objective, set)
		}
	}
}

func TestAlternativeSetRejectsTamperedContent(t *testing.T) {
	set := DescribeAlternatives("Submit a vacation request for approval")
	set.Alternatives[0].Key = "invented"
	if set.Validate() == nil {
		t.Fatal("unknown alternative key accepted")
	}
	set = DescribeAlternatives("Submit a vacation request for approval")
	set.Alternatives[0].Steps[0].Action = ""
	if set.Validate() == nil {
		t.Fatal("empty walkthrough action accepted")
	}
	set = DescribeAlternatives("Submit a vacation request for approval")
	set.Producer = "jev"
	if set.Validate() == nil {
		t.Fatal("misattributed producer accepted")
	}
}
