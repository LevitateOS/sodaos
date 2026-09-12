package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func keyFixture(t *testing.T) (*Management, *atomic.Int32) {
	t.Helper()
	m := managementFixture(t)
	calls := new(atomic.Int32)
	m.provider = &http.Client{Transport: boundedProviderTransport{managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("token has no operation deadline")
		}
		if r.ParseForm() != nil || r.Form.Get("scope") != "auth_keys" || r.Form.Get("tags") != "tag:soda-project" {
			t.Error("wrong OAuth scope/tags")
		}
		if r.Form.Get("client_secret") != "" {
			t.Error("unexpected auth-style fallback")
		}
		return managementResponse(200, `{"access_token":"synthetic-bearer","token_type":"Bearer","expires_in":3600}`), nil
	})}}
	m.keyHTTP = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.String() != "https://api.tailscale.com/api/v2/tailnet/soda.example.test/keys" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic-bearer" {
			t.Error("wrong key target/auth")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("key has no deadline")
		}
		var body struct {
			Capabilities struct {
				Devices struct {
					Create struct {
						Reusable, Ephemeral, Preauthorized bool
						Tags                               []string
					}
				}
			}
			ExpirySeconds int64
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Capabilities.Devices.Create.Reusable || !body.Capabilities.Devices.Create.Ephemeral || body.Capabilities.Devices.Create.Preauthorized || strings.Join(body.Capabilities.Devices.Create.Tags, ",") != "tag:soda-project" || body.ExpirySeconds != 300 {
			t.Error("wrong key capabilities")
		}
		return managementResponse(200, keyResponse()), nil
	})
	return m, calls
}
func keyResponse() string {
	b, _ := json.Marshal(map[string]any{"id": "synthetic-id", "key": "tskey-auth-synthetic-only", "created": time.Now().UTC(), "expires": time.Now().Add(5 * time.Minute).UTC(), "capabilities": map[string]any{"devices": map[string]any{"create": map[string]any{"reusable": false, "ephemeral": true, "preauthorized": false, "tags": []string{"tag:soda-project"}}}}})
	return string(b)
}
func TestProjectKeyUsesSelectedSDKAndExactPolicy(t *testing.T) {
	m, calls := keyFixture(t)
	v, e := m.policy.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	root, lock, e := m.policy.lock(t.Context(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	defer lock.Close()
	p, e := m.policy.load(root)
	if e != nil {
		t.Fatal(e)
	}
	var c credential
	if m.policy.read(root, "credential-"+p.Credential+".json", &c) != nil {
		t.Fatal("credential read")
	}
	key, e := m.projectKey(t.Context(), p, c)
	if e != nil || key != "tskey-auth-synthetic-only" || calls.Load() != 1 {
		t.Fatal("key creation failed", e)
	}
	b, _ := json.Marshal(v)
	if strings.Contains(string(b), "synthetic-only") || strings.Contains(string(b), "bearer") {
		t.Fatal("credential projected")
	}
}
func TestProjectKeyRefusesUnsafeOrAmbiguousProviderResult(t *testing.T) {
	for _, mode := range []string{"redirect", "error", "transport", "oversize", "null", "missing reusable", "duplicate", "reusable", "persistent", "wrong tags", "wrong preauthorization", "expired", "long expiry", "wrong key", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			m, _ := keyFixture(t)
			calls := 0
			m.keyHTTP = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				body := keyResponse()
				switch mode {
				case "redirect":
					res := managementResponse(307, "private provider text")
					res.Header.Set("Location", "https://other.example.test/")
					return res, nil
				case "error":
					return managementResponse(403, `{"message":"private provider text"}`), nil
				case "transport":
					return nil, errors.New("private provider text")
				case "oversize":
					body += strings.Repeat(" ", 65536)
				case "null":
					body = "null"
				case "missing reusable":
					body = strings.Replace(body, `"reusable":false,`, "", 1)
				case "duplicate":
					body = strings.Replace(body, `"reusable":false`, `"reusable":false,"reusable":false`, 1)
				case "reusable":
					body = strings.Replace(body, `"reusable":false`, `"reusable":true`, 1)
				case "persistent":
					body = strings.Replace(body, `"ephemeral":true`, `"ephemeral":false`, 1)
				case "wrong tags":
					body = strings.Replace(body, "tag:soda-project", "tag:other", 1)
				case "wrong preauthorization":
					body = strings.Replace(body, `"preauthorized":false`, `"preauthorized":true`, 1)
				case "wrong key":
					body = strings.Replace(body, "tskey-auth-", "tskey-client-", 1)
				case "expired", "long expiry":
					var v map[string]any
					json.Unmarshal([]byte(body), &v)
					v["expires"] = time.Now().Add(-time.Minute)
					if mode == "long expiry" {
						v["expires"] = time.Now().Add(time.Hour)
					}
					b, _ := json.Marshal(v)
					body = string(b)
				}
				return managementResponse(200, body), nil
			})
			in := enrollmentInput()
			p := enrollmentPolicy{Revision: strings.Repeat("a", 32), Tailnet: in.Tailnet, Tags: in.Tags}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			key, e := m.projectKey(ctx, p, credential{in.ClientID, in.ClientSecret})
			if key != "" || e == nil || strings.Contains(e.Error(), "private") || calls > 1 {
				t.Fatal("unsafe key or diagnostic accepted")
			}
			if mode == "cancelled" && calls != 0 {
				t.Fatal("cancelled operation created key")
			}
		})
	}
}
func runFixture(t *testing.T) (*Management, RunTarget, *atomic.Int32, string) {
	t.Helper()
	m, calls := keyFixture(t)
	p, dir := policyFixture(t)
	m.policy = *p
	v, e := p.update(t.Context(), enrollmentInput(), acceptedCredential)
	if e != nil {
		t.Fatal(e)
	}
	target := RunTarget{Project: "p" + strings.Repeat("a", 24), Container: strings.Repeat("b", 64), Run: strings.Repeat("c", 64)}
	root, lock, e := p.lock(t.Context(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	defer lock.Close()
	if e = p.publish(root, lock, "project-"+target.Project+".json", projectPolicy{Version: 1, Revision: strings.Repeat("d", 32), Project: target.Project, Container: target.Container, Binding: v.Enrollment.Binding, Enabled: true}); e != nil {
		t.Fatal(e)
	}
	return m, target, calls, dir
}
func TestRunEnrollmentHasOneAttemptAndNoSavedSecrets(t *testing.T) {
	m, target, calls, dir := runFixture(t)
	if phase, e := m.RunAttempt(t.Context(), target); e != nil || phase != "none" || calls.Load() != 0 {
		t.Fatal(phase, e)
	}
	var consumed atomic.Int32
	consume := func(ctx context.Context, key string) error {
		if key != "tskey-auth-synthetic-only" {
			t.Error("wrong single-use input")
		}
		consumed.Add(1)
		return nil
	}
	validate := func(context.Context) error { return nil }
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			e := m.EnrollRun(t.Context(), target, validate, consume)
			if e != nil && !errors.Is(e, ErrConflict) {
				t.Error(e)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 || consumed.Load() != 1 {
		t.Fatal("duplicate key or consumption")
	}
	if phase, e := m.RunAttempt(t.Context(), target); e != nil || phase != "submitted" || calls.Load() != 1 {
		t.Fatal(phase, e)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "soda-tailnet", "attempt-*"))
	if len(files) != 1 {
		t.Fatal("missing attempt")
	}
	b, _ := os.ReadFile(files[0])
	if strings.Contains(string(b), "tskey") || strings.Contains(string(b), "bearer") || !strings.Contains(string(b), "submitted") {
		t.Fatal("unsafe attempt projection")
	}
	// Loss of the same-run journal is not new-runtime admission.
	if os.Remove(files[0]) != nil {
		t.Fatal("fixture removal")
	}
	if phase, e := m.RunAttempt(t.Context(), target); e != nil || phase != "unconfirmed" || calls.Load() != 1 {
		t.Fatal(phase, e)
	}
	if e := m.EnrollRun(t.Context(), target, validate, consume); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	// A genuinely new incarnation is a separate attempt/device.
	target.Run = strings.Repeat("e", 64)
	if e := m.EnrollRun(t.Context(), target, validate, consume); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 2 || consumed.Load() != 2 {
		t.Fatal("new incarnation not admitted")
	}
}
func TestRunEnrollmentSerializesPolicyAndCancelsWaitingOff(t *testing.T) {
	m, target, calls, _ := runFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	original := m.keyHTTP
	m.keyHTTP = managementRoundTrip(func(r *http.Request) (*http.Response, error) { close(entered); <-release; return original.RoundTrip(r) })
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	done := make(chan error, 1)
	go func() {
		done <- m.EnrollRun(t.Context(), target, func(context.Context) error { return nil }, func(context.Context, string) error { return nil })
	}()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, e := m.policy.update(ctx, EnrollmentRequest{Action: "disable", Revision: "0"}, acceptedCredential); e == nil {
		t.Fatal("cancelled waiter acquired policy")
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	v, e := m.policy.enrollment(t.Context())
	if e != nil || !v.Admission {
		t.Fatal("cancelled Off changed policy")
	}
	if _, e = m.policy.update(t.Context(), EnrollmentRequest{Action: "disable", Revision: v.Revision}, acceptedCredential); e != nil {
		t.Fatal(e)
	}
	target.Run = strings.Repeat("f", 64)
	if e = m.EnrollRun(t.Context(), target, func(context.Context) error { return nil }, func(context.Context, string) error { t.Fatal("closed admission consumed key"); return nil }); !errors.Is(e, ErrConflict) || calls.Load() != 1 {
		t.Fatal("closed admission minted again", e)
	}
}

func TestRunEnrollmentFencesBindingIdentityAndUncertainty(t *testing.T) {
	for _, mode := range []string{"replaced cid", "invalid incarnation", "binding replaced", "admission closed", "provider failure", "changed during key", "consume failure", "fsync"} {
		t.Run(mode, func(t *testing.T) {
			m, target, calls, _ := runFixture(t)
			consumed := 0
			checks := 0
			switch mode {
			case "replaced cid":
				target.Container = strings.Repeat("f", 64)
			case "invalid incarnation":
				target.Run = "../other"
			case "binding replaced":
				in := enrollmentInput()
				v, _ := m.policy.enrollment(t.Context())
				in.Revision = v.Revision
				if _, e := m.policy.update(t.Context(), in, acceptedCredential); e != nil {
					t.Fatal(e)
				}
			case "admission closed":
				v, _ := m.policy.enrollment(t.Context())
				if _, e := m.policy.update(t.Context(), EnrollmentRequest{Action: "disable", Revision: v.Revision}, acceptedCredential); e != nil {
					t.Fatal(e)
				}
			case "provider failure":
				m.keyHTTP = managementRoundTrip(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, io.ErrUnexpectedEOF })
			case "fsync":
				m.policy.syncDir = func(*os.File) error { return io.ErrUnexpectedEOF }
			}
			validate := func(context.Context) error {
				checks++
				if mode == "changed during key" && checks > 1 {
					return ErrConflict
				}
				return nil
			}
			consume := func(context.Context, string) error {
				consumed++
				if mode == "consume failure" {
					return io.ErrUnexpectedEOF
				}
				return nil
			}
			e := m.EnrollRun(t.Context(), target, validate, consume)
			if e == nil {
				t.Fatal("unsafe enrollment admitted")
			}
			if mode != "consume failure" && consumed != 0 {
				t.Fatal("unsafe consumption")
			}
			if mode == "provider failure" || mode == "changed during key" || mode == "consume failure" || mode == "fsync" {
				before := calls.Load()
				m.policy.syncDir = nil
				if e = m.EnrollRun(t.Context(), target, validate, consume); e == nil || calls.Load() != before {
					t.Fatal("uncertain attempt replayed")
				}
			} else if calls.Load() != 0 {
				t.Fatal("provider work before admission")
			}
		})
	}
}
