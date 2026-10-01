package factory

import (
	"strings"
	"testing"
)

func validControl() IssueControl {
	return IssueControl{
		Repository: 7, Issue: 3, Revision: 1, NativeRev: 9,
		FirstSeenUnix: 100, AssessedUnix: 200,
		Readiness: ReadinessQueued, Reason: ReasonEligible,
		Fingerprint: strings.Repeat("a", 64), Authority: strings.Repeat("b", 64),
	}
}

func TestIssueControlValidate(t *testing.T) {
	if err := validControl().Validate(); err != nil {
		t.Fatal("valid queued control refused:", err)
	}
	blocked := validControl()
	blocked.Readiness = ReadinessBlocked
	blocked.Reason = BlockerCodePending
	blocked.Blockers = []Blocker{{
		Code: BlockerCodePending, EndpointRepo: 7, EndpointIssue: 9,
		Resolution: BlockerResolution(BlockerCodePending),
	}}
	if err := blocked.Validate(); err != nil {
		t.Fatal("valid blocked control refused:", err)
	}
	cases := map[string]func(*IssueControl){
		"repository":  func(c *IssueControl) { c.Repository = 0 },
		"readiness":   func(c *IssueControl) { c.Readiness = "ready" },
		"reason":      func(c *IssueControl) { c.Reason = "nope" },
		"fingerprint": func(c *IssueControl) { c.Fingerprint = "zz" },
		"authority":   func(c *IssueControl) { c.Authority = "" },
		"acceptance":  func(c *IssueControl) { c.Acceptance = "nope" },
		"queued with blockers": func(c *IssueControl) {
			c.Blockers = []Blocker{{Code: BlockerCodePending, Resolution: BlockerResolution(BlockerCodePending)}}
		},
	}
	for name, mutate := range cases {
		c := validControl()
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Error("invalid control accepted:", name)
		}
	}
	unready := validControl()
	unready.Readiness = ReadinessBlocked
	unready.Reason = BlockerCodePending
	if err := unready.Validate(); err == nil {
		t.Error("blockerless blocked control accepted")
	}
	oneSided := validControl()
	oneSided.Readiness = ReadinessBlocked
	oneSided.Reason = BlockerCodePending
	oneSided.Blockers = []Blocker{{Code: BlockerCodePending, EndpointRepo: 7, Resolution: BlockerResolution(BlockerCodePending)}}
	if err := oneSided.Validate(); err == nil {
		t.Error("one-sided blocker endpoint accepted")
	}
	mismatched := validControl()
	mismatched.Readiness = ReadinessBlocked
	mismatched.Reason = BlockerCodePending
	mismatched.Blockers = []Blocker{{Code: BlockerCodePending, Resolution: "custom text"}}
	if err := mismatched.Validate(); err == nil {
		t.Error("custom blocker resolution accepted")
	}
}

func TestBlockerResolutionsCoverEveryCode(t *testing.T) {
	for _, code := range []string{
		BlockerAcceptanceMissing, BlockerAcceptanceInvalid,
		BlockerAuthorityMissing, BlockerIssueInaccessible,
		BlockerEvidenceIncomplete, BlockerCodePending,
		BlockerResultPending, BlockerEndpointHidden,
		BlockerCycle, BlockerDependencyDepth,
	} {
		if BlockerResolution(code) == "" {
			t.Error("blocker code without resolution:", code)
		}
	}
	if BlockerResolution("nope") != "" {
		t.Error("unknown blocker code resolved")
	}
}

func fingerprintFixture() FingerprintInput {
	return FingerprintInput{
		Acceptance: "d" + strings.Repeat("c", 24),
		Authority:  strings.Repeat("b", 64),
		Validity:   []string{},
		Prereqs: []FingerprintPrereq{{
			Occurrence: "11", Outcome: PrereqResult, EndpointHead: "d" + strings.Repeat("d", 24),
			Closed: true, Lifecycle: 5, ClosedUnix: 300, Satisfied: true,
		}},
	}
}

func TestFingerprintStableOverOrdering(t *testing.T) {
	base := fingerprintFixture()
	reordered := fingerprintFixture()
	reordered.Validity = []string{"b_reason", "a_reason"}
	shuffled := fingerprintFixture()
	shuffled.Validity = []string{"a_reason", "b_reason"}
	if reordered.Fingerprint() != shuffled.Fingerprint() {
		t.Error("fingerprint depends on validity order")
	}
	multi := fingerprintFixture()
	multi.Prereqs = append(multi.Prereqs, FingerprintPrereq{Occurrence: "9", Outcome: PrereqCode, Blocker: BlockerCodePending})
	swapped := fingerprintFixture()
	swapped.Prereqs = []FingerprintPrereq{
		{Occurrence: "9", Outcome: PrereqCode, Blocker: BlockerCodePending},
		swapped.Prereqs[0],
	}
	if multi.Fingerprint() != swapped.Fingerprint() {
		t.Error("fingerprint depends on prerequisite order")
	}
	if base.Fingerprint() == multi.Fingerprint() {
		t.Error("fingerprint ignores added prerequisite")
	}
}

func TestFingerprintChangesOnSemanticInputs(t *testing.T) {
	base := fingerprintFixture().Fingerprint()
	mutations := map[string]func(*FingerprintInput){
		"acceptance":   func(in *FingerprintInput) { in.Acceptance = "d" + strings.Repeat("e", 24) },
		"authority":    func(in *FingerprintInput) { in.Authority = strings.Repeat("f", 64) },
		"validity":     func(in *FingerprintInput) { in.Validity = []string{"objective_changed"} },
		"cycle":        func(in *FingerprintInput) { in.Cycle = "7/3 -> 7/3" },
		"satisfied":    func(in *FingerprintInput) { in.Prereqs[0].Satisfied = false },
		"blocker":      func(in *FingerprintInput) { in.Prereqs[0].Blocker = BlockerResultPending },
		"endpointHead": func(in *FingerprintInput) { in.Prereqs[0].EndpointHead = "" },
		"closed":       func(in *FingerprintInput) { in.Prereqs[0].Closed = false },
		"closedUnix":   func(in *FingerprintInput) { in.Prereqs[0].ClosedUnix++ },
		"lifecycle":    func(in *FingerprintInput) { in.Prereqs[0].Lifecycle++ },
	}
	for name, mutate := range mutations {
		in := fingerprintFixture()
		mutate(&in)
		if in.Fingerprint() == base {
			t.Error("fingerprint ignores semantic input:", name)
		}
	}
}

func TestAuthorityFingerprintBindsGrants(t *testing.T) {
	verdict := EffectiveAuthority{Missing: []string{}, Authority: AuthorityRef{Policy: 1}, Effective: true, DispatchOpen: true}
	base := AuthorityFingerprint(verdict)
	changed := verdict
	changed.Authority.Policy = 2
	if AuthorityFingerprint(changed) == base {
		t.Error("authority fingerprint ignores grant revision")
	}
	missing := verdict
	missing.Missing = []string{MissingSponsorship}
	missing.Effective = false
	if AuthorityFingerprint(missing) == base {
		t.Error("authority fingerprint ignores missing grant")
	}
	reordered := EffectiveAuthority{Missing: []string{"b", "a"}, Effective: false}
	shuffled := EffectiveAuthority{Missing: []string{"a", "b"}, Effective: false}
	if AuthorityFingerprint(reordered) != AuthorityFingerprint(shuffled) {
		t.Error("authority fingerprint depends on missing order")
	}
}
