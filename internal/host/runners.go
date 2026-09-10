package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/runners"
)

var errRunnerUnconfirmed = errors.New("native runner result unconfirmed")

// The systemd root:soda socket ACL is this native service's caller boundary.
// Human operator authorization remains in web; the root CLI keeps its own gate.
func (d *Daemon) runnerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	action := strings.TrimPrefix(r.URL.Path, "/runners/")
	if r.Method != http.MethodPost || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "invalid runner operation", http.StatusBadRequest)
		return
	}
	switch action {
	case "list", "create", "start", "stop", "restart", "remove":
	default:
		http.NotFound(w, r)
		return
	}
	if d.Runners == nil {
		http.Error(w, "runner integration unavailable", 503)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	// Do not take the project mutex. Native's cross-process lock serializes this
	// protocol with Cockpit/CLI, including reads and the whole restart transaction.
	result, err := d.Runners.Execute(ctx, action, http.MaxBytesReader(w, r.Body, 65536))
	if err != nil {
		// Neither provider diagnostics nor registration input enters the journal/body.
		http.Error(w, "runner operation unconfirmed; inspect local and provider state before retrying", 502)
		return
	}
	body, err := json.Marshal(result)
	if err != nil || len(body) > 65536 {
		http.Error(w, "runner response unavailable", 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (c *Client) RunnersList(ctx context.Context) ([]runners.RunnerView, error) {
	var views []runners.RunnerView
	err := c.call(ctx, "/runners/list", runners.EmptyRequest{}, &views)
	return views, err
}
func (c *Client) RunnerCreate(ctx context.Context, in runners.CreateRequest) error {
	var result runners.MutationResponse
	err := c.call(ctx, "/runners/create", in, &result)
	if err == nil && !result.OK {
		return errRunnerUnconfirmed
	}
	return err
}
func (c *Client) RunnerAction(ctx context.Context, action string, in runners.RunnerRequest) error {
	switch action {
	case "start", "stop", "restart", "remove":
	default:
		return errRunnerUnconfirmed
	}
	var result runners.MutationResponse
	err := c.call(ctx, "/runners/"+action, in, &result)
	if err == nil && !result.OK {
		return errRunnerUnconfirmed
	}
	return err
}
