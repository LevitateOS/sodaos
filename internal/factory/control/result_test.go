package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestKnownCredentialMaterialCannotEnterRetainedOrPublishedResult(t *testing.T) {
	result := factory.Result{Status: "blocked", Summary: "needs clarification", Findings: []string{}}
	if _, err := resultBytes(result, []string{"synthetic-secret"}); err != nil {
		t.Fatal(err)
	}
	result.Findings = []string{"accidentally included synthetic-secret"}
	if _, err := resultBytes(result, []string{"synthetic-secret"}); err == nil {
		t.Fatal("credential entered result artifact")
	}
	result.Findings = nil
	result.Summary = strings.Repeat("x", 301<<10)
	if _, err := resultBytes(result, nil); err == nil {
		t.Fatal("unbounded retained output")
	}
}
