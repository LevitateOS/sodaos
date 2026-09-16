package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestFailureHintMatchesKnownSignatures(t *testing.T) {
	for _, tc := range []struct {
		reason string
		want   string
	}{
		{"open /run/go/src/a.go: permission denied", "setup-soda-candidate.sh"},
		{"internal/strictjson/decode.go:5:2: could not import bytes (permission denied)", "map its build cache"},
		{"go failed; retain attempt and inspect build.log: exit status 1", "setup-soda-candidate.sh"},
		{"GOPROXY list is not the empty string", "setup-soda-candidate.sh"},
		{"go: module lookup disabled by GOPROXY=off", "setup-soda-candidate.sh"},
		{"worker unit is already present or could not be checked", "wait for it"},
		{"sd-bus call: Interactive authentication required", "setup-soda-candidate.sh"},
		{"some brand-new failure mode", ""},
		{"", ""},
	} {
		if got := failureHint(tc.reason); (tc.want == "") != (got == "") || (tc.want != "" && !strings.Contains(got, tc.want)) {
			t.Fatalf("reason %q hint %q", tc.reason, got)
		}
	}
}

func TestFailedPanelShowsHintLine(t *testing.T) {
	var b bytes.Buffer
	r := newRenderer(&b, false, 80)
	r.SetOutDir("/out/run-03")
	if err := r.feed("FAILED Compile soda-dashboard | section 00:00:00 | total 00:00:00 | reason x.go: permission denied"); err != nil {
		t.Fatal(err)
	}
	if err := r.finish(1); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); !strings.Contains(got, "  hint: ") || !strings.Contains(got, "setup-soda-candidate.sh") {
		t.Fatalf("hint line missing:\n%s", got)
	}
}
