// Port of test_candidate_gate.py: check-native.sh uses the Go validator.
package build

import (
	"strings"
	"testing"
)

// Distinctive messages of the retired independent Python candidate rules.
var retiredRules = []string{
	"candidate architecture mismatch",
	"candidate provenance architecture mismatch",
	"independent extension image identity required",
	"pinned Forgejo compiler provenance required",
	"resolved Forgejo APK provenance required",
	"host and candidate content inventories differ",
	"extension installer CLI differs from patched Forgejo",
	"unsafe extension asset inventory",
	"incomplete candidate content inventory",
	"candidate content hash mismatch: ",
	"host package inventory hash mismatch",
	"missing application archives: ",
}

func splitNative(t *testing.T) (before, after string) {
	t.Helper()
	parts := strings.Split(ReadFile(t, "scripts/check-native.sh"), "bun run check:source\n")
	Require(t, len(parts) == 2, "check-native.sh has %d source-check markers", len(parts))
	return parts[0], parts[1]
}

func TestCandidateCheckUsesTheRustValidatorOwner(t *testing.T) {
	before, after := splitNative(t)
	for _, want := range []string{
		"--bin soda-candidate-check",
		`--candidate "$artifacts"`,
		`--arch "$arch"`,
		`--soda-revision "$revision"`,
		`--forgejo-revision "$forgejo_revision"`,
		"git -C ../forgejo-ext rev-parse HEAD",
		"Exact Fountain source revision required",
	} {
		Check(t, strings.Contains(before, want), "missing before marker %q", want)
	}
	Check(t, !strings.Contains(after, "soda-candidate-check"), "candidate check after source checks")
}

func TestCandidateNoIndependentPythonCandidateRules(t *testing.T) {
	native := ReadFile(t, "scripts/check-native.sh")
	Check(t, !strings.Contains(native, "python3 - <<PY"), "inline python present")
	for _, rule := range retiredRules {
		t.Run(rule, func(t *testing.T) {
			Check(t, !strings.Contains(native, rule), "retired rule present")
		})
	}
}

func TestCandidateNativeGuardsStillSurroundTheOwnerCheck(t *testing.T) {
	before, after := splitNative(t)
	for _, want := range []string{
		"Pinned native source-check tools required",
		"Check requires a clean exact-revision checkout",
		"soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)",
		`"$verifier" verify --source "$artifacts" --arch "$arch" --revision "$revision"`,
	} {
		Check(t, strings.Contains(before, want), "missing before marker %q", want)
	}
	Check(t, strings.Contains(after, `$(git rev-parse HEAD) == "$revision"`), "missing revision check")
}
