package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host/workspace"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/store"
)

type brokerTransport struct{ url string }

func (b brokerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = strings.TrimPrefix(b.url, "http://")
	return http.DefaultTransport.RoundTrip(clone)
}

func testBroker(t *testing.T, handler http.HandlerFunc) *identityclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &identityclient.Client{HTTP: &http.Client{Transport: brokerTransport{server.URL}}}
}

func credentialRun(t *testing.T) (*store.Store, factory.Run) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "execution.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	a := testAttempt(t, s)
	copy := a
	r, err := copy.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Image = "sha256:" + strings.Repeat("c", 64)
	r.Harness, r.Model = "test", "test"
	r.Resources = []factory.Resource{{Kind: "workspace", Name: factory.ResourceName(r.ID, "workspace"), ID: strings.Repeat("d", 64)}}
	if err = s.StartFactoryRun(t.Context(), &a, r); err != nil {
		t.Fatal(err)
	}
	return s, r
}

func TestDelegationPersistsNativeBindingBeforeBrokerDelivery(t *testing.T) {
	s, r := credentialRun(t)
	r.IdentityLeaseID, r.IdentityGeneration = "lease", 2
	original := []byte(`{"tokens":{"access_token":"synthetic-injected-credential"}}`)
	seeded := false
	runtime := &workspace.Runtime{Exec: executorFunc(func(_ context.Context, data []byte, _ string, _ ...string) ([]byte, error) {
		if string(data) != string(original) {
			t.Fatal("delivery bytes changed")
		}
		seeded = true
		return nil, nil
	})}
	broker := testBroker(t, func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path != "/register" {
			t.Fatal("unexpected broker operation")
		}
		var in identity.Request
		if err := json.NewDecoder(q.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		runs, err := s.FactoryRuns(t.Context(), r.AttemptID)
		if err != nil {
			t.Fatal(err)
		}
		if len(runs) != 1 || runs[0].IdentityBinding == nil || *runs[0].IdentityBinding != *in.Binding || !runs[0].CredentialDelegated {
			t.Fatal("delivery preceded durable execution binding")
		}
		if in.ID != "lease" || in.Binding.ID != r.Resources[0].ID || in.Binding.Generation != 2 {
			t.Fatal("delivery authority changed")
		}
		if err := json.NewEncoder(w).Encode(identity.DeliveryWire{Lease: identity.Lease{ID: "lease"}, Credential: original}); err != nil {
			t.Error(err)
		}
	})
	c := Controller{Store: s, Config: Config{ProjectID: "project"}, Identity: broker, Workspace: runtime}
	secrets, err := c.delegateCredential(t.Context(), &r)
	if err != nil {
		t.Fatal(err)
	}
	if !seeded {
		t.Fatal("credential was not delivered")
	}
	if _, err = resultBytes(factory.Result{Summary: "synthetic-injected-credential"}, secrets); err == nil {
		t.Fatal("injected secret escaped artifact check")
	}
}

func TestRejectedRegistrationCannotSeedWorkspace(t *testing.T) {
	s, r := credentialRun(t)
	r.IdentityLeaseID, r.IdentityGeneration = "lease", 1
	broker := testBroker(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("denied"))
	})
	c := Controller{Store: s, Identity: broker, Workspace: &workspace.Runtime{Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		t.Fatal("rejected lease seeded workspace")
		return nil, nil
	})}}
	if _, err := c.delegateCredential(t.Context(), &r); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("registration rejection lost")
	}
}

func TestFailedCaptureReconcilesOnlyItsLeaseWithoutReturningStaleBytes(t *testing.T) {
	s, r := credentialRun(t)
	r.IdentityLeaseID, r.IdentityGeneration = "lease", 1
	r.CredentialDelegated = true
	r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: r.Resources[0].ID, Generation: 1}
	reconciled := false
	broker := testBroker(t, func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path != "/reconcile-lease" {
			t.Fatal("failed capture attempted credential restore")
		}
		var in identity.Request
		if err := json.NewDecoder(q.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		if in.ID != "lease" || len(in.Credential) != 0 {
			t.Fatal("wrong reconciliation authority")
		}
		reconciled = true
		_, _ = w.Write([]byte("{}"))
	})
	c := Controller{Store: s, Identity: broker, Workspace: &workspace.Runtime{Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		return nil, errors.New("lost native boundary")
	})}}
	if err := c.returnDelegatedCredential(t.Context(), &r); err != nil {
		t.Fatal(err)
	}
	if !reconciled || r.CredentialReturned {
		t.Fatal("uncertain state reported as returned")
	}
}

func TestBusyReservationHonorsRunCancellationAndHumanIdentity(t *testing.T) {
	s, r := credentialRun(t)
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	bytes := []byte("synthetic executable")
	if err := os.WriteFile(filepath.Join(directory, "bin/codex"), bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	broker := testBroker(t, func(w http.ResponseWriter, q *http.Request) {
		var in identity.Request
		if err := json.NewDecoder(q.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		if in.Acquire.ActorID != 7 || in.Acquire.ExecutionID != r.ID || in.Acquire.Role != string(r.Role) || in.Acquire.ProjectID != "project" {
			t.Fatal("reservation identity changed")
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("busy"))
	})
	c := Controller{Store: s, Config: Config{ConnectionID: "connection", ProjectID: "project"}, Identity: broker, Workspace: &workspace.Runtime{Config: workspace.Config{HarnessDirectory: directory, HarnessVersion: "0.153.4", HarnessSHA256: hex.EncodeToString(digest[:])}, Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		return []byte("codex-cli 0.153.4\n"), nil
	})}}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if err := c.acquireCredential(ctx, 7, &r); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("busy lease escaped cancellation")
	}
	if r.IdentityLeaseID != "" {
		t.Fatal("busy reservation created authority")
	}
}

func TestTaskDoesNotExposeBrokerLeaseAuthority(t *testing.T) {
	_, r := credentialRun(t)
	r.IdentityLeaseID, r.IdentityGeneration = "private-lease", 1
	r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: r.Resources[0].ID, Generation: 1}
	r.CredentialDelegated = true
	task, err := (&Controller{}).task(t.Context(), factory.Attempt{}, r)
	if err != nil {
		t.Fatal(err)
	}
	if task.Run.IdentityLeaseID != "" || task.Run.IdentityGeneration != 0 || task.Run.IdentityBinding != nil || task.Run.CredentialDelegated || task.Run.CredentialReturned {
		t.Fatal("immutable workload task exposes lease authority")
	}
}

func TestBrokerCleanupFailureRemainsRecoverableAfterWorkspaceRemoval(t *testing.T) {
	s, r := credentialRun(t)
	r.IdentityLeaseID, r.IdentityGeneration = "lease", 1
	available := false
	removed := false
	broker := testBroker(t, func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path != "/reconcile-lease" {
			t.Fatal("unexpected reconciliation operation")
		}
		if !available {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("{}"))
	})
	runtime := &workspace.Runtime{Exec: executorFunc(func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		if args[0] == "inspect" {
			return []byte("false"), nil
		}
		if args[0] == "rm" {
			removed = true
			return nil, nil
		}
		if args[0] == "stop" {
			return nil, nil
		}
		if args[1] == "exists" {
			if removed {
				return nil, &workspace.ExitError{Code: 1}
			}
			return nil, nil
		}
		return json.Marshal(map[string]string{"id": r.Resources[0].ID, "name": r.Resources[0].Name, "owner": r.ID})
	})}
	c := Controller{Store: s, Identity: broker, Workspace: runtime}
	a, err := s.FactoryAttempt(t.Context(), r.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.finishRun(&a, &r, factory.Result{}, errors.New("execution failed")); err == nil {
		t.Fatal("unresolved broker cleanup reported success")
	}
	runs, err := s.FactoryRuns(t.Context(), r.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if !removed || runs[0].CleanupComplete {
		t.Fatal("physical cleanup hid unresolved credential authority")
	}
	available = true
	if err = c.clean(t.Context(), r.AttemptID); err != nil {
		t.Fatal(err)
	}
	runs, err = s.FactoryRuns(t.Context(), r.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if !runs[0].CleanupComplete || runs[0].CredentialReturned {
		t.Fatal("recovery did not resolve uncertain lease correctly")
	}
}
