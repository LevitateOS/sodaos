package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runtimePolicy(t *testing.T) (*Control, EnrollmentView, string) {
	t.Helper()
	p, parent := policyFixture(t)
	p.runtime = true
	saved, e := p.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	return &Control{policy: *p}, saved.Enrollment, parent
}
func TestProjectRuntimeSelectionAndIndependentOriginalBindings(t *testing.T) {
	m, policy, _ := runtimePolicy(t)
	enabled := true
	result, e := m.policy.update(t.Context(), EnrollmentRequest{Action: "default", Revision: policy.Revision, Default: &enabled}, acceptedCredential)
	if e != nil || result.Validate() != nil || !result.Enrollment.Default {
		t.Fatal(result, e)
	}
	options, e := m.Options(t.Context())
	if e != nil || options.Validate() != nil || !options.Available || !options.Default {
		t.Fatal(options, e)
	}
	for index := range 2 {
		project := "p" + strings.Repeat(string(rune('a'+index)), 24)
		cid := strings.Repeat(string(rune('c'+index)), 64)
		// A provisioned project uses the ordinary exact-project operation, not a
		// second reservation or a policy-lock-held native creation callback.
		if _, e = m.Project(t.Context(), ProjectRequest{Project: project, Action: "enable", Revision: "0", Binding: options.Binding, ConfirmID: project}, cid); e != nil {
			t.Fatal(e)
		}
		view, e := m.Project(t.Context(), ProjectRequest{Project: project, Action: "inspect"}, cid)
		if e != nil || view.Validate() != nil || !view.Enabled || view.Binding != options.Binding || view.AvailableBinding != options.Binding {
			t.Fatal(view, e)
		}
		target := RunTarget{Project: project, Container: cid, Run: strings.Repeat("e", 64)}
		binding, e := m.RunBinding(t.Context(), target)
		if e != nil || !binding.Enabled || !binding.Admission || binding.Tailnet != options.Tailnet {
			t.Fatal(binding, e)
		}
		if index == 0 {
			off, e := m.Project(t.Context(), ProjectRequest{Project: project, Action: "disable", Revision: view.Revision, ConfirmID: project}, cid)
			if e != nil || off.Validate() != nil || off.Enabled || !off.Saved {
				t.Fatal(off, e)
			}
		}
	}
	// A configured default does not apply to old projects or a legacy omission.
	old, e := m.Project(t.Context(), ProjectRequest{Project: "p" + strings.Repeat("f", 24), Action: "inspect"}, strings.Repeat("a", 64))
	if e != nil || old.Enabled || old.Revision != "0" {
		t.Fatal("default enrolled old project", old, e)
	}
}
func TestProjectRuntimeClosedAdmissionHasNoReservation(t *testing.T) {
	m, p, parent := runtimePolicy(t)
	project := "p" + strings.Repeat("a", 24)
	result, e := m.policy.update(t.Context(), EnrollmentRequest{Action: "disable", Revision: p.Revision}, acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	yes := true
	if _, e = m.policy.update(t.Context(), EnrollmentRequest{Action: "default", Revision: result.Enrollment.Revision, Default: &yes}, acceptedCredential); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if _, e = m.Project(t.Context(), ProjectRequest{Project: project, Action: "enable", Revision: "0", Binding: p.Binding, ConfirmID: project}, strings.Repeat("b", 64)); !errors.Is(e, ErrConflict) {
		t.Fatal("closed admission enabled project", e)
	}
	entries, e := os.ReadDir(filepath.Join(parent, "soda-tailnet"))
	if e != nil || len(entries) != 1 || entries[0].Name() != "policy.json" {
		t.Fatal("failed selection wrote a reservation", e)
	}
}
func TestProjectRuntimeEnableCASAndNoImplicitRetarget(t *testing.T) {
	m, p, _ := runtimePolicy(t)
	project, cid := "p"+strings.Repeat("a", 24), strings.Repeat("b", 64)
	in := ProjectRequest{Project: project, Action: "enable", Revision: "0", Binding: p.Binding, ConfirmID: project}
	first, e := m.Project(t.Context(), in, cid)
	if e != nil || first.Validate() != nil || !first.Enabled {
		t.Fatal(first, e)
	}
	if _, e = m.Project(t.Context(), in, cid); !errors.Is(e, ErrConflict) {
		t.Fatal("stale enable", e)
	}
	in.Revision = first.Revision
	in.Binding = strings.Repeat("f", 32)
	if _, e = m.Project(t.Context(), in, cid); !errors.Is(e, ErrConflict) {
		t.Fatal("binding retarget", e)
	}
	in.Binding = p.Binding
	in.Action = "retry"
	retried, e := m.Project(t.Context(), in, cid)
	if e != nil || retried.Validate() != nil || retried.Revision == first.Revision {
		t.Fatal(retried, e)
	}
	if _, e = m.Project(t.Context(), ProjectRequest{Project: project, Action: "inspect"}, strings.Repeat("c", 64)); !errors.Is(e, ErrConflict) {
		t.Fatal("original CID not bound", e)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, e = m.Project(cancelled, ProjectRequest{Project: project, Action: "disable", Revision: retried.Revision, ConfirmID: project}, cid); e == nil {
		t.Fatal("cancelled policy admitted")
	}
}
func TestProjectPolicyMissingIsOffWhileMalformedFailsSafely(t *testing.T) {
	m, policy, parent := runtimePolicy(t)
	project, cid := "p"+strings.Repeat("a", 24), strings.Repeat("b", 64)
	// Missing project policy is Off, never an error or an enrollment.
	view, e := m.Project(t.Context(), ProjectRequest{Project: project, Action: "inspect"}, cid)
	if e != nil || view.Enabled || view.Validate() != nil {
		t.Fatal("missing project policy is not Off", view, e)
	}
	target := RunTarget{Project: project, Container: cid, Run: strings.Repeat("e", 64)}
	if binding, e := m.RunBinding(t.Context(), target); e != nil || binding.Enabled {
		t.Fatal("missing binding is not Off", binding, e)
	}
	// An enabled project binds its original container; a replacement is conflict.
	in := ProjectRequest{Project: project, Action: "enable", Revision: "0", Binding: policy.Binding, ConfirmID: project}
	first, e := m.Project(t.Context(), in, cid)
	if e != nil || !first.Enabled {
		t.Fatal(first, e)
	}
	if _, e = m.Project(t.Context(), ProjectRequest{Project: project, Action: "inspect"}, strings.Repeat("c", 64)); !errors.Is(e, ErrConflict) {
		t.Fatal("replacement container adopted", e)
	}
	// A malformed configured file fails safely and distinctly from missing.
	path := filepath.Join(parent, "soda-tailnet", "project-"+project+".json")
	if e = os.WriteFile(path, []byte(`{}`), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e = m.Project(t.Context(), ProjectRequest{Project: project, Action: "inspect"}, cid); !errors.Is(e, ErrUnavailable) {
		t.Fatal("malformed project policy accepted", e)
	}
	if _, e = m.RunBinding(t.Context(), target); !errors.Is(e, ErrUnavailable) {
		t.Fatal("malformed binding accepted", e)
	}
	if after, e := os.ReadFile(path); e != nil || string(after) != `{}` {
		t.Fatal("malformed policy altered", e)
	}
}

func TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey(t *testing.T) {
	for _, tc := range []struct {
		body          string
		node, invalid bool
	}{
		{`{"BackendState":"NeedsLogin","HaveNodeKey":false}`, false, false},
		{`{"Version":"other","BackendState":"Running","HaveNodeKey":true}`, true, false},
		{`{"BackendState":"NeedsMachineAuth","HaveNodeKey":true}`, true, false},
		{`{"BackendState":"NeedsLogin","HaveNodeKey":true}`, true, false},
		{`{"BackendState":"Running","HaveNodeKey":false}`, false, true},
		{`{"BackendState":"Unknown","HaveNodeKey":false}`, false, true},
		{`{"BackendState":"NeedsLogin"}`, false, false},
		{`{"BackendState":"Running"}`, false, true},
		{`{"HaveNodeKey":false}`, false, true},
		{`{"BackendState":"NeedsLogin","HaveNodeKey":null}`, false, true},
		{`{"BackendState":"NeedsLogin","HaveNodeKey":"false"}`, false, true},
		{`{"BackendState":"NeedsLogin","HaveNodeKey":0}`, false, true},
		{`{"BackendState":"NeedsLogin","haveNodeKey":false}`, false, true},
		{`{"BackendState":"NeedsLogin","HaveNodeKey":false,"HaveNodeKey":true}`, false, true},
		{`null`, false, true},
	} {
		node, e := ProjectHasNode([]byte(tc.body))
		if node != tc.node || (e != nil) != tc.invalid {
			t.Fatal(tc.body, node, e)
		}
	}
}

func TestProjectStatusFreshDaemonOmitsFalseNodeKey(t *testing.T) {
	data := []byte(`{"Version":"1.102.4","BackendState":"NeedsLogin"}`)
	state, ips, dns, err := ProjectStatus(data, nil, RunBinding{})
	if err != nil || state != "needs-login" || len(ips) != 0 || dns != "" {
		t.Fatal("fresh project status", state, err)
	}
	for _, value := range []string{"null", `"false"`, "0"} {
		data := []byte(`{"Version":"1.102.4","BackendState":"NeedsLogin","HaveNodeKey":` + value + `}`)
		if _, _, _, err := ProjectStatus(data, nil, RunBinding{}); !errors.Is(err, ErrUnavailable) {
			t.Fatal("malformed optional node key accepted", value, err)
		}
	}
}

func TestProjectStatusRequiresExactNetworkTagsAndNativePreferences(t *testing.T) {
	binding := RunBinding{Enabled: true, Tailnet: "soda.example.test", Tags: []string{"tag:soda-project"}}
	status := map[string]any{"Version": "1.102.4", "BackendState": "Running", "HaveNodeKey": true, "CurrentTailnet": map[string]string{"Name": binding.Tailnet}, "Self": map[string]any{"ID": "node-project-a", "Online": true, "DNSName": "project.soda.ts.net.", "TailscaleIPs": []string{"100.64.0.2"}, "Tags": binding.Tags}, "AuthURL": "private", "Health": []string{"private"}}
	prefs := map[string]any{"WantRunning": true, "CorpDNS": true, "RouteAll": false, "RunSSH": false, "ExitNodeID": "", "ExitNodeIP": "", "AdvertiseRoutes": nil, "Persist": map[string]string{"PrivateNodeKey": "private"}}
	encode := func(v any) []byte {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	state, ips, dns, e := ProjectStatus(encode(status), encode(prefs), binding)
	if e != nil || state != "connected" || len(ips) != 1 || dns != "project.soda.ts.net" {
		t.Fatal(state, ips, dns, e)
	}
	for _, field := range []string{"CorpDNS", "RouteAll", "RunSSH", "ExitNodeID", "AdvertiseRoutes"} {
		original := prefs[field]
		switch field {
		case "CorpDNS":
			prefs[field] = false
		case "RouteAll", "RunSSH":
			prefs[field] = true
		case "ExitNodeID":
			prefs[field] = "foreign"
		case "AdvertiseRoutes":
			prefs[field] = []string{"0.0.0.0/0"}
		}
		if _, _, _, e = ProjectStatus(encode(status), encode(prefs), binding); e == nil {
			t.Fatal("unsafe preferences", field)
		}
		prefs[field] = original
	}
	binding.Tags = []string{"tag:other"}
	if state, ips, _, e = ProjectStatus(encode(status), encode(prefs), binding); e != nil || state == "connected" || len(ips) != 0 {
		t.Fatal("wrong tags published", state, e)
	}
	for _, native := range []string{"NeedsMachineAuth", "NeedsLogin", "Starting"} {
		status["BackendState"] = native
		state, ips, dns, e = ProjectStatus(encode(status), nil, binding)
		if e != nil || state == "connected" || len(ips) != 0 || dns != "" {
			t.Fatal(state, e)
		}
	}
	status["Version"] = "a different release"
	status["BackendState"] = "NeedsLogin"
	if _, _, _, e = ProjectStatus(encode(status), nil, binding); e != nil {
		t.Fatal("number-only runtime veto", e)
	}
}
