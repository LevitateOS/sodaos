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

func runtimePolicy(t *testing.T) (*Management, EnrollmentView, string) {
	t.Helper()
	p, parent := policyFixture(t)
	p.runtime = true
	saved, e := p.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	return &Management{policy: *p}, saved.Enrollment, parent
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
		calls := 0
		create := func() (string, error) { calls++; return cid, nil }
		selection := ProjectSelection{Enabled: true, Revision: options.Revision, Binding: options.Binding}
		stale := selection
		stale.Revision = policy.Revision
		if e = m.ProvisionProject(t.Context(), project, stale, create); !errors.Is(e, ErrConflict) || calls != 0 {
			t.Fatal("stale selection created", e, calls)
		}
		if e = m.ProvisionProject(t.Context(), project, selection, create); e != nil || calls != 1 {
			t.Fatal(e, calls)
		}
		if e = m.ProvisionProject(t.Context(), project, selection, create); !errors.Is(e, ErrConflict) || calls != 1 {
			t.Fatal("recreated reservation", e, calls)
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
func TestProjectRuntimeRetainsFailedReservationsAndClosesAdmission(t *testing.T) {
	m, p, parent := runtimePolicy(t)
	project := "p" + strings.Repeat("a", 24)
	selection := ProjectSelection{Enabled: true, Revision: p.Revision, Binding: p.Binding}
	calls := 0
	create := func() (string, error) { calls++; return "", errors.New("synthetic provisioning failure") }
	if e := m.ProvisionProject(t.Context(), project, selection, create); !errors.Is(e, ErrUnconfirmed) {
		t.Fatal(e)
	}
	if e := m.ProvisionProject(t.Context(), project, selection, create); !errors.Is(e, ErrConflict) || calls != 1 {
		t.Fatal("failed creation replay", e, calls)
	}
	b, e := os.ReadFile(filepath.Join(parent, "soda-tailnet", "reservation-"+project+".json"))
	if e != nil || strings.Contains(string(b), "tskey") || strings.Contains(string(b), "secret") {
		t.Fatal("unsafe/missing reservation", e)
	}
	result, e := m.policy.update(t.Context(), EnrollmentRequest{Action: "disable", Revision: p.Revision}, acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	yes := true
	if _, e = m.policy.update(t.Context(), EnrollmentRequest{Action: "default", Revision: result.Enrollment.Revision, Default: &yes}, acceptedCredential); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	selection.Revision = result.Enrollment.Revision
	if e = m.ProvisionProject(t.Context(), "p"+strings.Repeat("b", 24), selection, create); !errors.Is(e, ErrConflict) || calls != 1 {
		t.Fatal("closed admission created", e, calls)
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
func TestProjectStatusRequiresExactNetworkTagsAndNativePreferences(t *testing.T) {
	binding := RunBinding{Enabled: true, Tailnet: "soda.example.test", Tags: []string{"tag:soda-project"}}
	status := map[string]any{"Version": ManagementCLIRelease, "BackendState": "Running", "HaveNodeKey": true, "CurrentTailnet": map[string]string{"Name": binding.Tailnet}, "Self": map[string]any{"ID": "node-project-a", "Online": true, "DNSName": "project.soda.ts.net.", "TailscaleIPs": []string{"100.64.0.2"}, "Tags": binding.Tags}, "AuthURL": "private", "Health": []string{"private"}}
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
	if CheckDaemonRelease([]byte(`{"Version":"1.0"}`)) != ErrUnsupported || ProjectCLIRelease([]byte(`{"short":"1.0"}`)) != ErrUnsupported {
		t.Fatal("unreviewed runtime version accepted")
	}
}
