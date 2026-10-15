package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/tailnet"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Dedicated authenticated lifecycle route (RCX-2 T01). Fountain's manager
// admits exactly one tuple per token; the required Soda controller forwards
// that exact call and token here over the private Soda service socket. This
// route runs ahead of the ordinary browser gate: peer UID alone is not
// authority (other extension processes share Forge UID 1000), no Actor is
// consulted or fabricated, and no browser admission is accepted.
//
// The service independently invokes the manager authorize callback before
// handing a bounded command to the Rust host socket, and revalidates after
// a potentially long acquisition before effects or reply. Rejected,
// replayed, wrong-variant or unbound authority causes zero host effects:
// verification precedes every subprocess and RPC.
const extensionLifecycleRoute = "/api/extensions/lifecycle"

// Lifecycle controller and target scope owned by this service. Only the
// required soda controller's tailnet tuples are admitted; any other
// controller or package is a 403 without effects.
const (
	extensionLifecycleController = "soda"
	extensionLifecyclePackage    = "tailnet"
)

// Per-operation ceilings from the frozen contract. Every call also honors
// the lesser remaining whole-operation budget via its deadline.
const (
	extensionLifecycleObserveBudget    = 10 * time.Second
	extensionLifecycleActivateBudget   = 300 * time.Second
	extensionLifecycleDeactivateBudget = 120 * time.Second
)

func extensionLifecycleBudget(operation string) (time.Duration, bool) {
	switch operation {
	case extensions.LifecycleObserve:
		return extensionLifecycleObserveBudget, true
	case extensions.LifecycleActivate:
		return extensionLifecycleActivateBudget, true
	case extensions.LifecycleDeactivate:
		return extensionLifecycleDeactivateBudget, true
	}
	return 0, false
}

func (s *API) apiExtensionLifecycle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		auth.JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Lifecycle calls require POST.")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "Lifecycle calls accept no query parameters.")
		return
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "Lifecycle calls require a JSON body.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, extensions.MaximumLifecycleBodyBytes)
	call, ok := decodeExtensionLifecycleCall(w, r)
	if !ok {
		return
	}
	now := time.Now().Unix()
	if call.ControllerID != extensionLifecycleController || call.PackageID != extensionLifecyclePackage {
		auth.JSONError(w, http.StatusForbidden, "lifecycle_not_admitted", "This service admits only its required controller tuple.")
		return
	}
	if call.DeadlineUnix <= now {
		auth.JSONError(w, http.StatusForbidden, "lifecycle_expired", "The lifecycle authority has expired; observe before retrying.")
		return
	}
	if err := extensions.ValidateLifecycleCall(call, now); err != nil {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "The lifecycle call is malformed.")
		return
	}
	budget, ok := extensionLifecycleBudget(call.Operation)
	if !ok {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "The lifecycle operation is unknown.")
		return
	}
	authority, err := extensions.LifecycleContext(r)
	if err != nil {
		auth.JSONError(w, http.StatusForbidden, "lifecycle_not_admitted", "Lifecycle authority is required.")
		return
	}
	timeout := time.Until(time.Unix(call.DeadlineUnix, 0))
	if timeout > budget {
		timeout = budget
	}
	if timeout <= 0 {
		auth.JSONError(w, http.StatusForbidden, "lifecycle_expired", "The lifecycle authority has expired; observe before retrying.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	verification, ok := verifyExtensionLifecycle(w, ctx, authority, call)
	if !ok {
		return
	}
	if call.Operation == extensions.LifecycleActivate && !verification.DesiredEnabled {
		auth.JSONError(w, http.StatusForbidden, "lifecycle_not_admitted", "Activation requires current desired enablement.")
		return
	}
	observation, err := applyExtensionLifecycle(ctx, s.Host, authority, call)
	if err != nil {
		extensionLifecycleHostError(w, err)
		return
	}
	if err := extensions.ValidateLifecycleObservation(observation, call, time.Now().Unix()); err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "lifecycle_unavailable", "The lifecycle observation is unconfirmed; observe before retrying.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(observation)
}

func readExtensionLifecycleBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("lifecycle call requires a body")
	}
	return io.ReadAll(r.Body)
}

func decodeExtensionLifecycleCall(w http.ResponseWriter, r *http.Request) (extensions.LifecycleCall, bool) {
	var empty extensions.LifecycleCall
	body, err := readExtensionLifecycleBody(r)
	if err != nil || len(body) > extensions.MaximumLifecycleBodyBytes {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "The lifecycle call is malformed.")
		return empty, false
	}
	call, err := extensions.DecodeLifecycleCall(body)
	if err != nil {
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "The lifecycle call is malformed.")
		return empty, false
	}
	return call, true
}

func verifyExtensionLifecycle(w http.ResponseWriter, ctx context.Context, authority extensions.LifecycleAuthority, call extensions.LifecycleCall) (extensions.LifecycleVerification, bool) {
	verification, err := authority.Verify(ctx, call)
	if err == nil {
		return verification, true
	}
	switch {
	case strings.Contains(err.Error(), "malformed"):
		auth.JSONError(w, http.StatusBadRequest, "invalid_lifecycle_call", "The lifecycle call is malformed.")
	case strings.Contains(err.Error(), "unavailable"):
		auth.JSONError(w, http.StatusServiceUnavailable, "lifecycle_unavailable", "Lifecycle authorization is unavailable.")
	default:
		auth.JSONError(w, http.StatusForbidden, "lifecycle_not_admitted", "Lifecycle authority is unknown, expired, revoked or mismatched.")
	}
	return extensions.LifecycleVerification{}, false
}

// applyExtensionLifecycle runs the verified operation against the Rust host
// within the single bounded activation budget. Activation acquires the
// selected dependency first, then revalidates the still-live authority
// before any effect; a lost authority after download reports unavailable
// and never opens admission. Any host failure stops the flow with no
// further effects.
func applyExtensionLifecycle(ctx context.Context, hostClient *host.Client, authority extensions.LifecycleAuthority, call extensions.LifecycleCall) (extensions.LifecycleObservation, error) {
	if hostClient == nil {
		return extensions.LifecycleObservation{}, tailnet.ErrUnavailable
	}
	if call.Operation == extensions.LifecycleActivate {
		if _, err := hostClient.TailnetPrepare(ctx, call); err != nil {
			return extensions.LifecycleObservation{}, err
		}
		if _, err := authority.Verify(ctx, call); err != nil {
			return extensions.LifecycleObservation{}, tailnet.ErrUnavailable
		}
	}
	return hostClient.TailnetLifecycle(ctx, call)
}

func extensionLifecycleHostError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tailnet.ErrConflict):
		auth.JSONError(w, http.StatusConflict, "lifecycle_identity_changed", "The lifecycle identity changed; observe before retrying.")
	default:
		auth.JSONError(w, http.StatusServiceUnavailable, "lifecycle_unavailable", "The lifecycle operation is unavailable or unconfirmed; observe before retrying.")
	}
}
