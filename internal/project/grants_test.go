package project

import (
	"strings"
	"testing"
)

func grantTestProfile() *Profile {
	return &Profile{
		ID: RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
		Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}
}

func TestEnvironmentGrantValidation(t *testing.T) {
	good := EnvironmentGrant{Repository: 42, Owner: 7, Profile: grantTestProfile(), Active: true}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	good.Project = "p123456789012345678901234"
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]EnvironmentGrant{
		"no repository": {Owner: 7, Profile: grantTestProfile()},
		"no owner":      {Repository: 42, Profile: grantTestProfile()},
		"no profile":    {Repository: 42, Owner: 7},
		"bad project":   {Repository: 42, Owner: 7, Profile: grantTestProfile(), Project: "nope"},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("%s: grant accepted", name)
		}
	}
}

func TestRequirementDecisionValidation(t *testing.T) {
	digest := strings.Repeat("d", 64)
	good := RequirementDecision{
		ID: "d123456789012345678901234", Project: "p123456789012345678901234",
		Approver: 7, SourceCommit: strings.Repeat("e", 40), SetupDigest: digest, InputsDigest: digest,
	}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	ref := good.Ref(1)
	if err := ref.Validate(); err != nil || ref.ID != good.ID || ref.Digest != digest || ref.Revision != 1 {
		t.Fatalf("requirement reference: %+v %v", ref, err)
	}
	good.Predecessor = good.ID
	if err := good.Validate(); err == nil {
		t.Fatal("self-predecessor accepted")
	}
}

func TestApprovalDecisionValidation(t *testing.T) {
	digest := strings.Repeat("d", 64)
	good := ApprovalDecision{
		ID: "d223456789012345678901234", Project: "p123456789012345678901234",
		Requirement: "d123456789012345678901234", Approver: 7,
		EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	ref := good.Ref(1)
	if err := ref.Validate(); err != nil || ref.ID != good.ID || ref.EffectsDigest != digest {
		t.Fatalf("approval reference: %+v %v", ref, err)
	}
	good.Requirement = "not-a-decision"
	if err := good.Validate(); err == nil {
		t.Fatal("unbound approval accepted")
	}
}
