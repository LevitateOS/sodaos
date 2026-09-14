package host

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func validateTailnetRequest(r *http.Request) (string, int) {
	if r.Method != "POST" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
		return "", http.StatusBadRequest
	}
	action := strings.TrimPrefix(r.URL.Path, "/tailnet/")
	switch action {
	case "settings", "host", "enrollment", "options", "project", "policy":
		return action, 0
	default:
		return "", http.StatusNotFound
	}
}

func decodeTailnetBody(body io.Reader, out any) error {
	if strictjson.Decode(body, out) != nil {
		return tailnet.ErrInvalid
	}
	return nil
}

func executeTailnetQuery(ctx context.Context, m *tailnet.Management, action string, body io.Reader) (any, error) {
	var in struct{}
	if err := decodeTailnetBody(body, &in); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if action == "settings" {
		return m.Settings(ctx)
	}
	return m.Options(ctx)
}

func executeTailnetHost(ctx context.Context, m *tailnet.Management, body io.Reader) (any, error) {
	var in tailnet.HostRequest
	if err := decodeTailnetBody(body, &in); err != nil {
		return nil, err
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return m.HostAction(ctx, in)
}

func executeTailnetEnrollment(ctx context.Context, m *tailnet.Management, body io.Reader) (any, error) {
	var in tailnet.EnrollmentRequest
	if err := decodeTailnetBody(body, &in); err != nil {
		return nil, err
	}
	defer func() { in.ClientSecret = "" }()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return m.Enrollment(ctx, in)
}

func (d *Daemon) runProjectTailnetAction(ctx context.Context, in tailnet.ProjectRequest, cid, action string) (any, error) {
	if action == "policy" {
		return d.Tailnet.Project(ctx, in, cid)
	}
	return d.observeProjectTailnet(ctx, in, cid)
}

func (d *Daemon) verifyProjectContainerUnchanged(ctx context.Context, project, cid string) error {
	after, err := d.projectContainer(ctx, project, false)
	if err != nil || after != cid {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func (d *Daemon) executeTailnetProjectOrPolicy(ctx context.Context, action string, body io.Reader) (any, error) {
	var in tailnet.ProjectRequest
	if err := decodeTailnetBody(body, &in); err != nil {
		return nil, err
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if action == "policy" && in.Action != "inspect" {
		return nil, tailnet.ErrInvalid
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	cid, err := d.projectContainer(ctx, in.Project, false)
	if err != nil {
		return nil, err
	}
	out, err := d.runProjectTailnetAction(ctx, in, cid, action)
	if err != nil {
		return nil, err
	}
	if err := d.verifyProjectContainerUnchanged(ctx, in.Project, cid); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *Daemon) dispatchTailnetAction(ctx context.Context, action string, body io.Reader) (any, error) {
	switch action {
	case "settings", "options":
		return executeTailnetQuery(ctx, d.Tailnet, action, body)
	case "host":
		return executeTailnetHost(ctx, d.Tailnet, body)
	case "enrollment":
		return executeTailnetEnrollment(ctx, d.Tailnet, body)
	case "project", "policy":
		return d.executeTailnetProjectOrPolicy(ctx, action, body)
	default:
		return nil, tailnet.ErrInvalid
	}
}

func tailnetErrorStatus(err error) int {
	switch {
	case errors.Is(err, tailnet.ErrInvalid):
		return 400
	case errors.Is(err, tailnet.ErrConflict):
		return 409
	case errors.Is(err, tailnet.ErrUnsupported):
		return 422
	case errors.Is(err, tailnet.ErrUnavailable):
		return 503
	default:
		return 502
	}
}

func writeTailnetResponse(w http.ResponseWriter, out any) {
	data, err := json.Marshal(out)
	if err != nil || len(data) > 65536 {
		http.Error(w, "Tailnet response unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// The root:soda socket admits the service, not human operators. Web owns fresh
// human authority. Tailnet never takes the unrelated global project gate.
func (d *Daemon) tailnetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	action, code := validateTailnetRequest(r)
	if code != 0 {
		if code == http.StatusNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "invalid Tailnet operation", code)
		}
		return
	}
	if d.Tailnet == nil {
		http.Error(w, "Tailnet management disabled", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	body := http.MaxBytesReader(w, r.Body, 65536)
	out, err := d.dispatchTailnetAction(ctx, action, body)
	if err != nil || ctx.Err() != nil || out == nil {
		// Input and provider/native error text must never enter journal or response.
		http.Error(w, "Tailnet operation unavailable or unconfirmed; observe before retrying", tailnetErrorStatus(err))
		return
	}
	writeTailnetResponse(w, out)
}

func (c *Client) tailnetCall(ctx context.Context, path string, in, out any) error {
	err := c.call(ctx, "/tailnet/"+path, in, out)
	var native nativeHTTPError
	if errors.As(err, &native) {
		switch native.status {
		case 400:
			return tailnet.ErrInvalid
		case 409:
			return tailnet.ErrConflict
		case 422:
			return tailnet.ErrUnsupported
		case 503:
			return tailnet.ErrUnavailable
		}
	}
	if err != nil {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func (c *Client) TailnetSettings(ctx context.Context) (tailnet.SettingsView, error) {
	var out tailnet.SettingsView
	err := c.tailnetCall(ctx, "settings", struct{}{}, &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}

func (c *Client) TailnetHost(ctx context.Context, in tailnet.HostRequest) (tailnet.HostResult, error) {
	if err := in.Validate(); err != nil {
		return tailnet.HostResult{}, err
	}
	var out tailnet.HostResult
	err := c.tailnetCall(ctx, "host", in, &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}

func (c *Client) TailnetEnrollment(ctx context.Context, in tailnet.EnrollmentRequest) (tailnet.EnrollmentResult, error) {
	if err := in.Validate(); err != nil {
		return tailnet.EnrollmentResult{}, err
	}
	var out tailnet.EnrollmentResult
	err := c.tailnetCall(ctx, "enrollment", in, &out)
	if err == nil {
		err = out.Validate()
		if out.Saved != (in.Action != "check") || (out.Saved && (out.Enrollment.Revision == in.Revision || !out.Enrollment.Configured)) || (!out.Saved && (!out.CredentialChecked || out.Enrollment.Revision != in.Revision)) {
			err = tailnet.ErrUnavailable
		}
		switch in.Action {
		case "save", "rotate":
			if out.Enrollment.Tailnet != in.Tailnet || !slices.Equal(out.Enrollment.Tags, in.Tags) || out.Enrollment.Preauthorized != *in.Preauthorized {
				err = tailnet.ErrUnavailable
			}
		case "disable":
			if out.Enrollment.Admission || out.Enrollment.Default {
				err = tailnet.ErrUnavailable
			}
		case "default":
			if out.Enrollment.Default != *in.Default {
				err = tailnet.ErrUnavailable
			}
		}
	}
	return out, err
}

func (c *Client) TailnetOptions(ctx context.Context) (tailnet.ProjectOptions, error) {
	var out tailnet.ProjectOptions
	err := c.tailnetCall(ctx, "options", struct{}{}, &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}

// TailnetPolicy is the compact Spaces read: saved intent only, no daemon exec,
// connection addresses or background enrollment. Full Network uses TailnetProject.
func (c *Client) TailnetPolicy(ctx context.Context, project string) (tailnet.ProjectView, error) {
	var out tailnet.ProjectView
	in := tailnet.ProjectRequest{Project: project, Action: "inspect"}
	if in.Validate() != nil {
		return out, tailnet.ErrInvalid
	}
	e := c.tailnetCall(ctx, "policy", in, &out)
	if e == nil {
		e = out.Validate()
		if out.Project != project || out.Saved {
			e = tailnet.ErrUnavailable
		}
	}
	return out, e
}

func (c *Client) TailnetProject(ctx context.Context, in tailnet.ProjectRequest) (tailnet.ProjectView, error) {
	if err := in.Validate(); err != nil {
		return tailnet.ProjectView{}, err
	}
	var out tailnet.ProjectView
	err := c.tailnetCall(ctx, "project", in, &out)
	if err == nil {
		err = out.Validate()
		if out.Project != in.Project || out.Saved != (in.Action != "inspect") || (out.Saved && (out.Revision == in.Revision || out.Enabled != (in.Action != "disable") || (in.Action != "disable" && out.Binding != in.Binding))) {
			err = tailnet.ErrUnavailable
		}
	}
	return out, err
}
