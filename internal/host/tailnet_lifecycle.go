package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func (c *Client) lifecycleRequest(ctx context.Context, path string, body []byte) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://soda-host"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

// decodeLifecycleResponse maps the Rust lifecycle wire onto the existing
// tailnet sentinels: 400 is a pin/identity mismatch the service already
// validated past, 409 is a busy or changed identity, 503 is unavailable,
// and anything else (timeout, truncation, echo drift) is unconfirmed.
// A non-200 never yields an observation.
func decodeLifecycleResponse(response *http.Response, observation *extensions.LifecycleObservation) error {
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		return tailnet.ErrInvalid
	case http.StatusConflict:
		return tailnet.ErrConflict
	case http.StatusServiceUnavailable:
		return tailnet.ErrUnavailable
	default:
		return tailnet.ErrUnconfirmed
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(extensions.MaximumLifecycleBodyBytes)+1))
	if err != nil {
		return tailnet.ErrUnconfirmed
	}
	if len(body) > extensions.MaximumLifecycleBodyBytes {
		return tailnet.ErrUnconfirmed
	}
	decoded, err := extensions.DecodeLifecycleObservation(body)
	if err != nil {
		return tailnet.ErrUnconfirmed
	}
	*observation = decoded
	return nil
}

// Tailnet lifecycle RPCs (RCX-2 T01). The request is the manager-verified SDK
// call: no token and no browser-selected image cross this hop. The Rust
// owner matches the tuple against its trusted release pin; the service
// validates authority before calling and revalidates the echoed observation.
func (c *Client) tailnetLifecycleCall(ctx context.Context, path string, call extensions.LifecycleCall) (extensions.LifecycleObservation, error) {
	var empty extensions.LifecycleObservation
	body, err := json.Marshal(call)
	if err != nil {
		return empty, tailnet.ErrInvalid
	}
	observation, err := c.postLifecycle(ctx, path, body)
	if err != nil {
		return empty, err
	}
	if !observation.Echoes(call) {
		return empty, tailnet.ErrUnconfirmed
	}
	return observation, nil
}

func (c *Client) postLifecycle(ctx context.Context, path string, body []byte) (extensions.LifecycleObservation, error) {
	var observation extensions.LifecycleObservation
	request, err := c.lifecycleRequest(ctx, path, body)
	if err != nil {
		return observation, tailnet.ErrInvalid
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return observation, tailnet.ErrUnconfirmed
	}
	defer response.Body.Close()
	return observation, decodeLifecycleResponse(response, &observation)
}

// TailnetPrepare validates and pulls the selected Tailnet dependency through
// the privileged host helper. It never opens admission or starts a daemon;
// a revoked controller after download leaves a harmless verified cache.
func (c *Client) TailnetPrepare(ctx context.Context, call extensions.LifecycleCall) (extensions.LifecycleObservation, error) {
	return c.tailnetLifecycleCall(ctx, "/tailnet/prepare", call)
}

// TailnetLifecycle applies one verified observe, activate or deactivate tuple
// and returns the host's strict observation of the resulting facts.
func (c *Client) TailnetLifecycle(ctx context.Context, call extensions.LifecycleCall) (extensions.LifecycleObservation, error) {
	return c.tailnetLifecycleCall(ctx, "/tailnet/lifecycle", call)
}
