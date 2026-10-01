package project

import (
	"strings"
	"testing"
)

const (
	testPreparationID = "f0123456789abcdef01234567"
	testProjectID     = "p0123456789abcdef01234567"
	testDecisionA     = "d0123456789abcdef01234567"
	testDecisionB     = "d123456789abcdef012345678"
	testDigest        = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testCommit        = "0123456789abcdef0123456789abcdef01234567"
)

func testAcceptance() RequirementAcceptance {
	return RequirementAcceptance{ID: testDecisionA, Revision: 1, Approver: 7, SourceCommit: testCommit, Digest: testDigest}
}

func testApproval() AdminApproval {
	return AdminApproval{ID: testDecisionB, Revision: 1, Approver: 9, EffectsDigest: testDigest}
}

func testPreparation() Preparation {
	return Preparation{
		ID: testPreparationID, Project: testProjectID, Role: RoleCoder,
		Requirements: testAcceptance(), Approval: testApproval(),
		SourceCommit: testCommit, SetupDigest: testDigest, Tools: []string{"python3"},
	}
}

func TestFactoryRolesAreFixed(t *testing.T) {
	if !ValidFactoryRole(RoleCoder) || !ValidFactoryRole(RoleReviewer) {
		t.Fatal("fixed roles rejected")
	}
	for _, role := range []string{"", "root", "soda-worker", "coder", "SODA-CODER", "soda-coder "} {
		if ValidFactoryRole(role) {
			t.Fatalf("unexpected role accepted: %q", role)
		}
	}
}

func TestPreparationRequiresBothApprovals(t *testing.T) {
	base := testPreparation()
	if err := base.Validate(); err != nil {
		t.Fatalf("valid preparation rejected: %v", err)
	}
	cases := map[string]Preparation{}
	withoutMaintainer := base
	withoutMaintainer.Requirements = RequirementAcceptance{}
	cases["maintainer"] = withoutMaintainer
	withoutAdmin := base
	withoutAdmin.Approval = AdminApproval{}
	cases["admin"] = withoutAdmin
	for name, p := range cases {
		if err := p.Validate(); err == nil {
			t.Fatalf("preparation without %s approval accepted", name)
		}
	}
}

func TestPreparationRejectsUntrustedIdentities(t *testing.T) {
	base := testPreparation()
	cases := []func(*Preparation){
		func(p *Preparation) { p.ID = "../../x" },
		func(p *Preparation) { p.Project = "other" },
		func(p *Preparation) { p.Role = "root" },
		func(p *Preparation) { p.SourceCommit = "short" },
		func(p *Preparation) { p.SetupDigest = "zz" },
		func(p *Preparation) { p.Tools = []string{"good", "evil;id"} },
		func(p *Preparation) { p.Credential = "../secret" },
		func(p *Preparation) { p.Requirements.ID = "d-too-short" },
		func(p *Preparation) { p.Approval.EffectsDigest = "" },
	}
	for i, mutate := range cases {
		p := base
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}

func TestApprovedSetupRequiresFixedEntrypoints(t *testing.T) {
	base := ApprovedSetup{
		Files:  map[string][]byte{"setup.sh": []byte("true"), "check.sh": []byte("true")},
		Bundle: []byte("bundle"),
	}
	prep := testPreparation()
	prep.SetupDigest = SetupDigestOf(base.Files)
	if err := (Prepare{Preparation: prep, Setup: base}).Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	skewed := prep
	skewed.SetupDigest = testDigest
	if err := (Prepare{Preparation: skewed, Setup: base}).Validate(); err == nil {
		t.Fatal("approved inputs skewing from their digest accepted")
	}
	missing := base
	missing.Files = map[string][]byte{"setup.sh": []byte("true")}
	if err := (Prepare{Preparation: prep, Setup: missing}).Validate(); err == nil {
		t.Fatal("missing check entrypoint accepted")
	}
	traversal := base
	traversal.Files = map[string][]byte{"setup.sh": []byte("true"), "check.sh": []byte("true"), "../evil": []byte("x")}
	if err := (Prepare{Preparation: prep, Setup: traversal}).Validate(); err == nil {
		t.Fatal("traversal file name accepted")
	}
	oversized := base
	oversized.Bundle = make([]byte, MaxSourceBundle+1)
	if err := (Prepare{Preparation: prep, Setup: oversized}).Validate(); err == nil {
		t.Fatal("oversized bundle accepted")
	}
}

func TestPreparePhasesAreKnown(t *testing.T) {
	for _, phase := range []string{PrepareApproved, PrepareWaiting, PrepareRunning, PrepareReady, PrepareFailed, PrepareStopped, PrepareInterrupted} {
		if !ValidPreparePhase(phase) {
			t.Fatalf("known phase rejected: %q", phase)
		}
	}
	if ValidPreparePhase("resumed") || ValidPreparePhase("") {
		t.Fatal("unknown phase accepted")
	}
}

func TestLifecycleGrantAndHoldValidate(t *testing.T) {
	profile := Profile{ID: RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
	grant := LifecycleGrant{Project: testProjectID, Owner: 3, Profile: &profile, Active: true}
	if err := grant.Validate(); err != nil {
		t.Fatalf("valid grant rejected: %v", err)
	}
	grant.Owner = 0
	if err := grant.Validate(); err == nil {
		t.Fatal("grant without owner accepted")
	}
	hold := MaintenanceHold{Project: testProjectID, Revision: 2, Hold: true}
	if err := hold.Validate(); err != nil {
		t.Fatalf("valid hold rejected: %v", err)
	}
	hold.Project = "nope"
	if err := hold.Validate(); err == nil {
		t.Fatal("hold without project accepted")
	}
}

func TestStoredPreparationBindsStateIdentity(t *testing.T) {
	stored := StoredPreparation{Preparation: testPreparation()}
	if err := stored.Validate(); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	stored.State = PrepareState{ID: "f99999999999999999999999", Project: testProjectID, Role: RoleCoder, Phase: PrepareReady}
	if err := stored.Validate(); err == nil {
		t.Fatal("mismatched state identity accepted")
	}
}
