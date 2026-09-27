package control

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Handler requires Unix peer admission at the listener. Runtime-only operations
// are excluded from the dashboard's admin listener.
func (c *Controller) Handler(runtimeAllowed bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("Origin") != "" {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		var in identity.Request
		if err := strictjson.Decode(http.MaxBytesReader(w, r.Body, 512<<10), &in); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		out, err := c.dispatch(r, in, runtimeAllowed)
		if err != nil {
			status, code := responseError(err)
			http.Error(w, code, status)
			return
		}
		if out == nil {
			out = struct{}{}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func responseError(err error) (int, string) {
	switch {
	case errors.Is(err, identity.ErrDenied):
		return http.StatusForbidden, "denied"
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "missing"
	case errors.Is(err, identity.ErrBusy):
		return http.StatusConflict, "busy"
	case errors.Is(err, identity.ErrStale):
		return http.StatusConflict, "stale"
	case errors.Is(err, identity.ErrUncertain):
		return http.StatusConflict, "reauth"
	default:
		return http.StatusInternalServerError, "unavailable"
	}
}

func (c *Controller) dispatch(r *http.Request, in identity.Request, runtimeAllowed bool) (any, error) {
	switch r.URL.Path {
	case "/connections", "/available", "/revoke":
		return c.connectionRequest(r, in)
	case "/enrollment/start", "/enrollment/read", "/enrollment/complete", "/enrollment/cancel":
		return c.enrollmentRequest(r, in)
	case "/grants", "/grant/create", "/grant/revoke":
		return c.grantRequest(r, in)
	case "/leases":
		return c.Leases(r.Context(), in.OwnerID, in.ID)
	case "/lease/end":
		return nil, c.EndLease(r.Context(), in.OwnerID, in.ID)
	default:
		if !runtimeAllowed {
			return nil, identity.ErrDenied
		}
		return c.runtimeRequest(r, in)
	}
}

func (c *Controller) connectionRequest(r *http.Request, in identity.Request) (any, error) {
	switch r.URL.Path {
	case "/connections":
		return c.Connections(r.Context(), in.OwnerID)
	case "/available":
		return c.Available(r.Context(), in.OwnerID, in.ProjectID)
	default:
		return nil, c.Revoke(r.Context(), in.OwnerID, in.ID)
	}
}

func (c *Controller) enrollmentRequest(r *http.Request, in identity.Request) (any, error) {
	switch r.URL.Path {
	case "/enrollment/start":
		return c.StartEnrollment(r.Context(), in.OwnerID, in.ProviderID, in.Label)
	case "/enrollment/read":
		return c.Enrollment(r.Context(), in.OwnerID, in.ID)
	case "/enrollment/complete":
		return c.CompleteEnrollment(r.Context(), in.OwnerID, in.ID, in.State, in.Code)
	default:
		return nil, c.CancelEnrollment(r.Context(), in.OwnerID, in.ID)
	}
}

func (c *Controller) grantRequest(r *http.Request, in identity.Request) (any, error) {
	switch r.URL.Path {
	case "/grants":
		return c.Grants(r.Context(), in.OwnerID, in.ID)
	case "/grant/create":
		if in.Grant == nil {
			return nil, identity.ErrDenied
		}
		return c.CreateGrant(r.Context(), in.OwnerID, *in.Grant)
	default:
		return nil, c.RevokeGrant(r.Context(), in.OwnerID, in.ID)
	}
}

func (c *Controller) runtimeRequest(r *http.Request, in identity.Request) (any, error) {
	ctx := r.Context()
	switch r.URL.Path {
	case "/acquire":
		if in.Acquire == nil {
			return nil, identity.ErrDenied
		}
		return c.Acquire(ctx, *in.Acquire)
	case "/register", "/reject", "/return":
		return c.boundLeaseRequest(r, in)
	case "/reconcile-lease":
		return nil, c.ReconcileLease(ctx, in.ID)
	default:
		return nil, identity.ErrDenied
	}
}

func (c *Controller) boundLeaseRequest(r *http.Request, in identity.Request) (any, error) {
	if in.Binding == nil {
		return nil, identity.ErrDenied
	}
	switch r.URL.Path {
	case "/register":
		delivery, err := c.Register(r.Context(), in.ID, *in.Binding)
		if err != nil {
			return nil, err
		}
		return identity.DeliveryWire(delivery), nil
	case "/reject":
		return nil, c.Reject(r.Context(), in.ID, *in.Binding)
	case "/return":
		return nil, c.Return(r.Context(), in.ID, *in.Binding, in.Credential)
	default:
		return nil, identity.ErrDenied
	}
}
