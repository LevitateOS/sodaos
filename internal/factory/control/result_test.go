package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestCredentialSnapshotsRetainOriginalAndRenewedValues(t *testing.T) {
	before, err := credentialStrings([]byte(`{"tokens":{"access_token":"synthetic-original-credential"}}`))
	if err != nil {
		t.Fatal(err)
	}
	after, err := credentialStrings([]byte(`{"tokens":{"access_token":"synthetic-renewed-credential"}}`))
	if err != nil {
		t.Fatal(err)
	}
	secrets := append(before, after...)
	for _, secret := range []string{"synthetic-original-credential", "synthetic-renewed-credential"} {
		if _, err := resultBytes(factory.Result{Summary: secret}, secrets); err == nil {
			t.Fatal("original or renewed credential omitted from returned denylist")
		}
	}
}

func TestKnownCredentialMaterialCannotEnterRetainedOrPublishedResult(t *testing.T) {
	result := factory.Result{Status: "blocked", Summary: "needs clarification", Findings: []string{}}
	if _, err := resultBytes(result, []string{"synthetic-secret"}); err != nil {
		t.Fatal(err)
	}
	result.Findings = []string{"accidentally included synthetic-secret"}
	if _, err := resultBytes(result, []string{"synthetic-secret"}); err == nil {
		t.Fatal("credential entered result artifact")
	}
	result.Findings = []string{"synthetic-\"escaped-credential"}
	if _, err := resultBytes(result, []string{"synthetic-\"escaped-credential"}); err == nil {
		t.Fatal("JSON-escaped credential entered result artifact")
	}
	result.Findings = nil
	result.Summary = strings.Repeat("x", 301<<10)
	if _, err := resultBytes(result, nil); err == nil {
		t.Fatal("unbounded retained output")
	}
}
