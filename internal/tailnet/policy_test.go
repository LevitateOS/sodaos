package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func policyFixture(t *testing.T) (*policyStore, string) {
	t.Helper()
	parent := t.TempDir()
	return &policyStore{uid: uint32(os.Geteuid()), parent: func() (*os.Root, error) { return os.OpenRoot(parent) }}, parent
}
func enrollmentInput() EnrollmentRequest {
	pre := false
	return EnrollmentRequest{Action: "save", Revision: "0", Tailnet: "soda.example.test", Tags: []string{"tag:soda-project"}, Preauthorized: &pre, ClientID: "synthetic-client", ClientSecret: "tskey-client-soda-synthetic-secret"}
}
func acceptedCredential(context.Context, EnrollmentRequest) error { return nil }
func TestTailnetPolicyReadsAndChecksAreNonMutating(t *testing.T) {
	p, parent := policyFixture(t)
	v, e := p.enrollment(t.Context())
	if e != nil || v.Validate() != nil || v.Configured || v.Revision != "0" {
		t.Fatal(v, e)
	}
	in := enrollmentInput()
	in.Action = "check"
	r, e := p.update(t.Context(), in, acceptedCredential)
	if e != nil || r.Saved || !r.CredentialChecked || r.Enrollment.Configured {
		t.Fatal(r, e)
	}
	entries, e := os.ReadDir(parent)
	if e != nil || len(entries) != 0 {
		t.Fatal("observation/check wrote state", e)
	}
}
func TestTailnetUnsupportedDefaultHasNoFilesOrProviderEffects(t *testing.T) {
	p, parent := policyFixture(t)
	b := true
	_, e := p.update(t.Context(), EnrollmentRequest{Action: "default", Revision: "0", Default: &b}, func(context.Context, EnrollmentRequest) error { t.Fatal("provider called"); return nil })
	if !errors.Is(e, ErrUnsupported) {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(parent)
	if e != nil || len(entries) != 0 {
		t.Fatal("unsupported default wrote state", e)
	}
}
func TestTailnetPolicyRotationCASAndSecretProjection(t *testing.T) {
	p, parent := policyFixture(t)
	in := enrollmentInput()
	first, e := p.update(t.Context(), in, acceptedCredential)
	if e != nil || !first.Saved || first.Validate() != nil {
		t.Fatal(first, e)
	}
	if _, e = p.update(t.Context(), in, acceptedCredential); !errors.Is(e, ErrConflict) {
		t.Fatal("stale save", e)
	}
	rotate := in
	rotate.Action = "rotate"
	rotate.Revision = first.Enrollment.Revision
	rotate.ClientSecret = "tskey-client-another-synthetic-secret"
	bad := rotate
	bad.Tailnet = "other.example.test"
	if _, e = p.update(t.Context(), bad, acceptedCredential); !errors.Is(e, ErrConflict) {
		t.Fatal("cross-network rotation", e)
	}
	second, e := p.update(t.Context(), rotate, acceptedCredential)
	if e != nil || second.Enrollment.Binding != first.Enrollment.Binding || second.Enrollment.Revision == first.Enrollment.Revision {
		t.Fatal(second, e)
	}
	root := filepath.Join(parent, "soda-tailnet")
	entries, e := os.ReadDir(root)
	if e != nil {
		t.Fatal(e)
	}
	credentials := 0
	for _, entry := range entries {
		info, _ := entry.Info()
		if info.Mode().Perm() != 0600 {
			t.Fatal("public state", entry.Name(), info.Mode())
		}
		if strings.HasPrefix(entry.Name(), "credential-") {
			credentials++
		}
	}
	if credentials != 2 {
		t.Fatal("previous credential was not retained", credentials)
	}
	encoded, _ := json.Marshal(second)
	if strings.Contains(string(encoded), "synthetic") || strings.Contains(string(encoded), "credential-") {
		t.Fatal("secret/reference in projection")
	}
	disabled, e := p.update(t.Context(), EnrollmentRequest{Action: "disable", Revision: second.Enrollment.Revision}, acceptedCredential)
	if e != nil || disabled.Enrollment.Admission || disabled.Enrollment.Default {
		t.Fatal(disabled, e)
	}
	v, e := p.enrollment(t.Context())
	if e != nil || v.Revision != disabled.Enrollment.Revision {
		t.Fatal(v, e)
	}
}
func TestTailnetPolicyConcurrencyAndCancelledWaiter(t *testing.T) {
	p, _ := policyFixture(t)
	seed, e := p.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	in := enrollmentInput()
	in.Action = "rotate"
	in.Revision = seed.Enrollment.Revision
	var calls atomic.Int32
	check := func(context.Context, EnrollmentRequest) error { calls.Add(1); return nil }
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := p.update(context.Background(), in, check); results <- e }()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 || calls.Load() != 1 {
		t.Fatal(success, conflict, calls.Load())
	}
	root, lock, e := p.lock(t.Context(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	defer lock.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	if _, e = p.enrollment(ctx); e == nil {
		t.Fatal("cancelled waiter acquired lock")
	}
}
func TestTailnetPolicyRefusesUnsafeAndAmbiguousState(t *testing.T) {
	for _, kind := range []string{"root-link", "policy-link", "credential-hardlink", "permissions", "corrupt", "missing-credential"} {
		t.Run(kind, func(t *testing.T) {
			p, parent := policyFixture(t)
			if kind == "root-link" {
				if e := os.Symlink(t.TempDir(), filepath.Join(parent, "soda-tailnet")); e != nil {
					t.Fatal(e)
				}
			} else {
				if _, e := p.update(t.Context(), enrollmentInput(), acceptedCredential); e != nil {
					t.Fatal(e)
				}
				dir := filepath.Join(parent, "soda-tailnet")
				path := filepath.Join(dir, "policy.json")
				entries, _ := os.ReadDir(dir)
				credentialPath := ""
				for _, f := range entries {
					if strings.HasPrefix(f.Name(), "credential-") {
						credentialPath = filepath.Join(dir, f.Name())
					}
				}
				switch kind {
				case "policy-link":
					if e := os.Rename(path, path+".kept"); e != nil {
						t.Fatal(e)
					}
					if e := os.Symlink(path+".kept", path); e != nil {
						t.Fatal(e)
					}
				case "credential-hardlink":
					if e := os.Link(credentialPath, credentialPath+".link"); e != nil {
						t.Fatal(e)
					}
				case "permissions":
					if e := os.Chmod(path, 0644); e != nil {
						t.Fatal(e)
					}
				case "corrupt":
					if e := os.WriteFile(path, []byte(`{}`), 0600); e != nil {
						t.Fatal(e)
					}
				case "missing-credential":
					if e := os.Rename(credentialPath, credentialPath+".kept"); e != nil {
						t.Fatal(e)
					}
				}
			}
			if _, e := p.enrollment(t.Context()); e == nil {
				t.Fatal("unsafe state accepted")
			}
		})
	}
}
func TestTailnetPublicationFailureDoesNotRollbackOrReplay(t *testing.T) {
	p, _ := policyFixture(t)
	first, e := p.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	p.syncDir = func(*os.File) error { return errors.New("synthetic fsync failure") }
	in := EnrollmentRequest{Action: "disable", Revision: first.Enrollment.Revision}
	if _, e = p.update(t.Context(), in, acceptedCredential); !errors.Is(e, ErrUnconfirmed) {
		t.Fatal(e)
	}
	p.syncDir = nil
	after, e := p.enrollment(t.Context())
	if e != nil || after.Admission || after.Revision == first.Enrollment.Revision {
		t.Fatal("publication was rolled back", after, e)
	}
	if _, e = p.update(t.Context(), in, acceptedCredential); !errors.Is(e, ErrConflict) {
		t.Fatal("uncertain write replayed", e)
	}
}
func TestTailnetProjectPolicyDisabledUntilRuntimeExists(t *testing.T) {
	p, parent := policyFixture(t)
	const id = "p0123456789abcdef01234567"
	cid := strings.Repeat("a", 64)
	v, e := p.project(t.Context(), ProjectRequest{Project: id, Action: "inspect"}, cid)
	if e != nil || v.Enabled || v.Validate() != nil {
		t.Fatal(v, e)
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Fatal("read initialized policy")
	}
	in := ProjectRequest{Project: id, Action: "enable", Revision: "0", Binding: strings.Repeat("b", 32), ConfirmID: id}
	if _, e = p.project(t.Context(), in, cid); !errors.Is(e, ErrUnsupported) {
		t.Fatal(e)
	}
	in.Action = "disable"
	in.Binding = ""
	v, e = p.project(t.Context(), in, cid)
	if e != nil || v.Enabled || v.Outcome != "disconnect-unconfirmed" {
		t.Fatal(v, e)
	}
	if _, e = p.project(t.Context(), ProjectRequest{Project: id, Action: "inspect"}, strings.Repeat("c", 64)); !errors.Is(e, ErrConflict) {
		t.Fatal("replacement adopted", e)
	}
	if _, e = p.project(t.Context(), in, cid); !errors.Is(e, ErrConflict) {
		t.Fatal("duplicate disable accepted", e)
	}
}
func TestTailnetInputRefusesCredentialEndpointAndPolicyOverrides(t *testing.T) {
	for _, mutate := range []func(*EnrollmentRequest){
		func(r *EnrollmentRequest) { r.ClientSecret += "?baseURL=https://attacker.invalid" },
		func(r *EnrollmentRequest) { r.ClientSecret += "?ephemeral=false" },
		func(r *EnrollmentRequest) { r.Tailnet = "-" },
		func(r *EnrollmentRequest) { r.Tailnet = "../other" },
		func(r *EnrollmentRequest) { r.Tags = []string{"tag:z", "tag:a"} },
		func(r *EnrollmentRequest) { r.Tags = []string{"tag:a", "tag:a"} },
		func(r *EnrollmentRequest) { r.Tags = []string{"--flag"} },
		func(r *EnrollmentRequest) { r.Action = "disable" },
	} {
		r := enrollmentInput()
		mutate(&r)
		if r.Validate() == nil {
			t.Fatal("invalid credential/policy accepted")
		}
	}
}
