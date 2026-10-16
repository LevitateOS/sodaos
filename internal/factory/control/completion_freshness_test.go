package control

import (
	"strings"
	"testing"
)

func freshBindings() CompletionFreshness {
	return CompletionFreshness{
		SourceCommit:         strings.Repeat("a", 40),
		PreparationDigest:    strings.Repeat("b", 64),
		ScopeDigest:          strings.Repeat("c", 64),
		AuthorityFingerprint: "authority-fingerprint",
		EvidenceDigest:       strings.Repeat("d", 64),
	}
}

func TestCheckCompletionFreshnessReleasesUnchanged(t *testing.T) {
	sealed, current := freshBindings(), freshBindings()
	if err := CheckCompletionFreshness(sealed, current); err != nil {
		t.Fatalf("fresh release refused: %v", err)
	}
}

func TestCheckCompletionFreshnessInvalidatesEachDimension(t *testing.T) {
	sealed := freshBindings()
	cases := []struct {
		name   string
		mutate func(*CompletionFreshness)
		want   string
	}{
		{"source", func(f *CompletionFreshness) { f.SourceCommit = strings.Repeat("e", 40) }, "source"},
		{"preparation", func(f *CompletionFreshness) { f.PreparationDigest = strings.Repeat("e", 64) }, "preparation"},
		{"scope", func(f *CompletionFreshness) { f.ScopeDigest = strings.Repeat("e", 64) }, "scope"},
		{"authority", func(f *CompletionFreshness) { f.AuthorityFingerprint = "another-authority" }, "authority"},
		{"evidence", func(f *CompletionFreshness) { f.EvidenceDigest = strings.Repeat("e", 64) }, "evidence"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current := freshBindings()
			tc.mutate(&current)
			err := CheckCompletionFreshness(sealed, current)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want a %s invalidation", err, tc.want)
			}
		})
	}
}

func TestCheckCompletionFreshnessRefusesMissingBindings(t *testing.T) {
	sealed := freshBindings()
	current := freshBindings()
	current.SourceCommit = ""
	if err := CheckCompletionFreshness(sealed, current); err == nil {
		t.Fatal("missing source binding released")
	}
	empty := CompletionFreshness{}
	if err := CheckCompletionFreshness(empty, empty); err == nil {
		t.Fatal("empty bindings released")
	}
}
