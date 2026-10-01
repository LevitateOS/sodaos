package host

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"

	"github.com/levitateos/sodaos/internal/identity"
)

func (c *Client) IdentityLaunch(ctx context.Context, in identity.TerminalStart) (identity.Lease, error) {
	var out identity.Lease
	err := c.call(ctx, "/identity/launch", in, &out)
	return out, err
}

func (d *Daemon) identityLaunch(ctx context.Context, in identity.TerminalStart) (identity.Lease, error) {
	if d.Identity == nil || d.Config.CodexHarness == "" {
		return identity.Lease{}, identity.ErrDenied
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return identity.Lease{}, err
	}
	lease, err := d.Identity.Acquire(ctx, identity.AcquireRequest{ProviderID: identity.Codex, ActorID: in.ActorID, ConnectionID: in.ConnectionID, ProjectID: in.ProjectID, ExecutionID: hex.EncodeToString(raw[:]), Kind: identity.Terminal, Deadline: time.Now().Add(12 * time.Hour)})
	if err != nil {
		return lease, err
	}
	binding, err := d.Terminal.PrepareIdentity(ctx, lease, in.Login, in.Scope, in.Cols, in.Rows)
	if err != nil {
		_ = d.Identity.ReconcileLease(context.Background(), lease.ID)
		return lease, err
	}
	delivery, err := d.Identity.Register(ctx, lease.ID, binding)
	if err != nil {
		_ = d.Identity.ReconcileLease(context.Background(), lease.ID)
		return lease, err
	}
	defer func() {
		for i := range delivery.Credential {
			delivery.Credential[i] = 0
		}
	}()
	_, err = d.Terminal.Identity(ctx, "start", delivery)
	if err != nil {
		_ = d.Identity.ReconcileLease(context.Background(), lease.ID)
		return lease, err
	}
	return delivery.Lease, nil
}

// IdentityValidate attests the managed terminal before broker credential delivery.
func (c *Client) IdentityValidate(ctx context.Context, l identity.Lease) error {
	return c.call(ctx, "/identity/validate", identity.DeliveryWire{Lease: l}, nil)
}

func (c *Client) IdentityStop(ctx context.Context, l identity.Lease) error {
	return c.call(ctx, "/identity/stop", identity.DeliveryWire{Lease: l}, nil)
}

func (c *Client) IdentityStart(ctx context.Context, d identity.Delivery) error {
	return c.call(ctx, "/identity/start", identity.DeliveryWire(d), nil)
}

// IdentityFinish returns a private credential stream after freezing and ending
// the entire managed terminal cgroup. It is never a browser response.
func (c *Client) IdentityFinish(ctx context.Context, l identity.Lease) (identity.Delivery, error) {
	var out identity.DeliveryWire
	err := c.callLimit(ctx, "/identity/finish", identity.DeliveryWire{Lease: l}, &out, 384<<10)
	return identity.Delivery(out), err
}

func validIdentityRequest(r *http.Request) bool {
	return r.Method == http.MethodPost && r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" && len(r.Header.Values("Origin")) == 0
}

func (d *Daemon) identityOperation(ctx context.Context, path string, body io.Reader) (any, error) {
	if path == "/identity/launch" {
		var in identity.TerminalStart
		if err := strictjson.Decode(body, &in); err != nil {
			return nil, err
		}
		return d.identityLaunch(ctx, in)
	}
	var in identity.DeliveryWire
	if err := strictjson.Decode(body, &in); err != nil {
		return nil, err
	}
	if in.Lease.Kind == identity.Factory {
		return d.factoryIdentityOperation(ctx, strings.TrimPrefix(path, "/identity/"), identity.Delivery(in))
	}
	var out identity.Delivery
	var err error
	if in.Lease.ProviderID == identity.Muse && in.Lease.Binding != nil && in.Lease.Binding.Scope == "muse-project" && d.Muse != nil {
		out, err = d.Muse.Muse(ctx, strings.TrimPrefix(path, "/identity/"), identity.Delivery(in))
	} else {
		out, err = d.Terminal.Identity(ctx, strings.TrimPrefix(path, "/identity/"), identity.Delivery(in))
	}
	return identity.DeliveryWire(out), err
}

// factoryIdentityOperation serves broker validate/stop/finish callbacks for
// supervised factory runs through the exact run binding. Unproved providers
// and harnesses deny at binding validation; they deliver nothing.
func (d *Daemon) factoryIdentityOperation(ctx context.Context, action string, delivery identity.Delivery) (any, error) {
	if d.Terminal == nil {
		return nil, identity.ErrDenied
	}
	out := delivery
	out.Credential = nil
	var err error
	switch action {
	case "validate":
		err = d.Terminal.FactoryCodexValidate(ctx, delivery.Lease)
	case "stop":
		err = d.Terminal.FactoryCodexStop(ctx, delivery.Lease)
	case "finish":
		out.Credential, err = d.Terminal.FactoryCodexFinish(ctx, delivery.Lease)
	default:
		err = identity.ErrDenied
	}
	if err != nil {
		return nil, err
	}
	return identity.DeliveryWire(out), nil
}

func (d *Daemon) identityHandler(w http.ResponseWriter, r *http.Request) {
	if !validIdentityRequest(r) || d.Terminal == nil {
		http.Error(w, "invalid identity operation", 400)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 384<<10)
	out, err := d.identityOperation(r.Context(), r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "identity operation unconfirmed", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
