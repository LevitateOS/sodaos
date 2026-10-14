package application

import (
	"encoding/json"
	"strings"
	"testing"
)

func fixture() Spec {
	return Spec{ID: "a" + strings.Repeat("1", 24), Revision: 1, Name: "Example", ArtifactSHA256: strings.Repeat("2", 64), BundleSHA256: strings.Repeat("3", 64), ToolsSHA256: strings.Repeat("4", 64), RuntimeImage: "sha256:" + strings.Repeat("5", 64), Image: "sha256:" + strings.Repeat("6", 64), Data: "a" + strings.Repeat("7", 24), OwnerTeam: "12345678-1234-1234-1234-123456789abc", Origin: "https://app.example.test", Port: 20001}
}

func TestAdmittedReferences(t *testing.T) {
	base := fixture()
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Spec){
		func(s *Spec) { s.ID = "../application" }, func(s *Spec) { s.Revision = 0 }, func(s *Spec) { s.ArtifactSHA256 = strings.Repeat("A", 64) }, func(s *Spec) { s.Image = "latest" }, func(s *Spec) { s.OwnerTeam = "" }, func(s *Spec) { s.Port = 30000 }, func(s *Spec) { s.Name = "other app" },
	} {
		s := base
		mutate(&s)
		if s.Validate() == nil {
			t.Fatalf("accepted invalid reference %+v", s)
		}
	}
	for _, origin := range []string{"http://app.example.test", "https://app.example.test/", "https://app.example.test:443", "https://APP.example.test", "https://app.example.test?", "https://app.example.test#", "https://user@app.example.test", "https://127.0.0.1", "https://localhost", "https://app.example.test:02001", "https://app.example.test/path"} {
		s := base
		s.Origin = origin
		if s.Validate() == nil {
			t.Errorf("accepted origin %q", origin)
		}
	}
	base.Origin = "https://app.example.test:8443"
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdministrationAndNativeWire(t *testing.T) {
	base := fixture()
	r := Record{Application: base, AdministratorID: 41}
	if err := r.Authorize(41); err != nil {
		t.Fatal(err)
	}
	if r.Authorize(42) == nil || r.Authorize(0) == nil {
		t.Fatal("administrator grant bypass")
	}
	for _, action := range []string{"inspect", "install"} {
		if err := (Request{Application: base, Action: action}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, action := range []string{"start", "stop", "replace"} {
		if (Request{Application: base, Action: action}).Validate() == nil {
			t.Fatalf("unfenced %s", action)
		}
		if err := (Request{Application: base, Action: action, ExpectedContainer: strings.Repeat("a", 64)}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := json.Marshal(Request{Application: base, Action: "start", ExpectedContainer: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"application"`, `"artifact_sha256"`, `"runtime_image"`, `"owner_team"`, `"expected_container"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("native key absent %s", key)
		}
	}
}

func TestReplacementAndObservation(t *testing.T) {
	base := fixture()
	next := base
	next.Revision++
	next.Image = "sha256:" + strings.Repeat("a", 64)
	if err := base.Replacement(next); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Spec){func(s *Spec) { s.Revision++ }, func(s *Spec) { s.Data = "a" + strings.Repeat("b", 24) }, func(s *Spec) { s.OwnerTeam = "87654321-1234-1234-1234-123456789abc" }, func(s *Spec) { s.Origin = "https://other.example.test" }, func(s *Spec) { s.Name = "Other" }, func(s *Spec) { s.Port++ }} {
		s := next
		mutate(&s)
		if base.Replacement(s) == nil {
			t.Fatal("replacement changed stable identity")
		}
	}
	o := Observation{ID: base.ID, Revision: 1, Container: strings.Repeat("a", 64), Image: base.Image, Origin: base.Origin, Running: true, Ready: true}
	if err := o.Validate(base); err != nil {
		t.Fatal(err)
	}
	if o.Validate(next) == nil {
		t.Fatal("old runtime observation accepted for new desired revision")
	}
	bad := o
	bad.Running = false
	if bad.Validate(base) == nil {
		t.Fatal("ready without running")
	}
	if SameObservation(nil, &o) || !SameObservation(nil, nil) || !SameObservation(&o, &o) {
		t.Fatal("unknown observation misrepresented")
	}
	bad = o
	bad.Ready = false
	if SameObservation(&o, &bad) {
		t.Fatal("observation CAS did not compare state")
	}
}
