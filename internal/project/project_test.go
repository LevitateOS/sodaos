package project

import (
	"encoding/json"
	"strings"
	"testing"
)

func validProfile() Profile {
	return Profile{ID: RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
}

func TestProfileValidation(t *testing.T) {
	if err := validProfile().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := validProfile()
	bad.Architecture = "riscv"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected unsupported architecture to fail")
	}
	if _, err := Decode(`{"id":"other"}`); err == nil {
		t.Fatal("expected incomplete profile to fail decode")
	}
	profile, err := Decode(`{"id":"rocky-headless","distribution":"rocky","version":"10.2","interface":"headless","architecture":"amd64","image":"sha256:` + strings.Repeat("a", 64) + `","revision":"` + strings.Repeat("b", 40) + `"}`)
	if err != nil || *profile != validProfile() {
		t.Fatal(profile, err)
	}
}

func TestCreateValidation(t *testing.T) {
	profile := validProfile()
	if err := (Create{Profile: &profile, ID: "p" + strings.Repeat("0", 24), Owner: 1}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Create{Profile: &profile, ID: "bad", Owner: 1}).Validate(); err == nil {
		t.Fatal("expected malformed project id to fail")
	}
	if err := (Create{ID: "p" + strings.Repeat("0", 24), Owner: 1}).Validate(); err == nil {
		t.Fatal("expected missing profile to fail")
	}
}

// TestWireShapeFrozen guards the Unix-socket contract with the privileged
// host daemon: these field names cross the privilege boundary and must not
// drift when domain code is refactored.
func TestWireShapeFrozen(t *testing.T) {
	profile := validProfile()
	raw, err := json.Marshal(Create{Profile: &profile, ID: "p" + strings.Repeat("0", 24), Owner: 7})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"profile", "id", "owner"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("wire field %q missing: %s", key, raw)
		}
	}
	var back Create
	if err := json.Unmarshal(raw, &back); err != nil || *back.Profile != profile {
		t.Fatal(back, err)
	}
	env, err := json.Marshal(Environment{ID: "p" + strings.Repeat("0", 24), Running: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(env), `"running":true`) {
		t.Fatalf("unexpected environment wire shape: %s", env)
	}
}

func TestIdentityHelpers(t *testing.T) {
	if !ValidID("p"+strings.Repeat("0", 24)) || ValidID("q"+strings.Repeat("0", 24)) {
		t.Fatal("ValidID mismatch")
	}
	// "root" is shape-valid; rejecting it is execution policy, not identity shape.
	if !ValidLogin("alice-1") || !ValidLogin("root") || ValidLogin("Root") {
		t.Fatal("ValidLogin mismatch")
	}
	if !ValidImageRef("sha256:"+strings.Repeat("a", 64)) || ValidImageRef("latest") {
		t.Fatal("ValidImageRef mismatch")
	}
	if !ValidContainerID(strings.Repeat("c", 64)) || ValidContainerID("short") {
		t.Fatal("ValidContainerID mismatch")
	}
}
