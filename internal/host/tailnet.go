package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
)

// The root:soda socket admits the service, not human operators. Web owns fresh
// human authority. Tailnet never takes the unrelated global project gate.
func (d *Daemon) tailnetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "POST" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
		http.Error(w, "invalid Tailnet operation", 400)
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/tailnet/")
	switch action {
	case "settings", "host", "enrollment", "options", "project":
	default:
		http.NotFound(w, r)
		return
	}
	if d.Tailnet == nil {
		http.Error(w, "Tailnet management disabled", 503)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	body := http.MaxBytesReader(w, r.Body, 65536)
	decode := func(out any) error {
		if strictjson.Decode(body, out) != nil {
			return tailnet.ErrInvalid
		}
		return nil
	}
	var out any
	var err error
	switch action {
	case "settings", "options":
		var in struct{}
		err = decode(&in)
		if err == nil && ctx.Err() == nil {
			if action == "settings" {
				out, err = d.Tailnet.Settings(ctx)
			} else {
				out, err = d.Tailnet.Options(ctx)
			}
		}
	case "host":
		var in tailnet.HostRequest
		if err = decode(&in); err == nil {
			err = in.Validate()
		}
		if err == nil && ctx.Err() == nil {
			out, err = d.Tailnet.HostAction(ctx, in)
		}
	case "enrollment":
		var in tailnet.EnrollmentRequest
		if err = decode(&in); err == nil {
			err = in.Validate()
		}
		if err == nil && ctx.Err() == nil {
			out, err = d.Tailnet.Enrollment(ctx, in)
		}
		in.ClientSecret = ""
	case "project":
		var in tailnet.ProjectRequest
		if err = decode(&in); err == nil {
			err = in.Validate()
		}
		if err == nil && ctx.Err() == nil {
			// Reuse the original project/user-mapping validator. No request CID, UID,
			// namespace, arbitrary socket or developer-controlled binary is admitted.
			var cid string
			cid, err = d.projectContainer(ctx, in.Project, false)
			if err == nil {
				out, err = d.Tailnet.Project(ctx, in, cid)
				if err == nil {
					after, e := d.projectContainer(ctx, in.Project, false)
					if e != nil || after != cid {
						err = tailnet.ErrUnconfirmed
					}
				}
			}
		}
	}
	if err != nil || ctx.Err() != nil || out == nil {
		status := 502
		switch {
		case errors.Is(err, tailnet.ErrInvalid):
			status = 400
		case errors.Is(err, tailnet.ErrConflict):
			status = 409
		case errors.Is(err, tailnet.ErrUnsupported):
			status = 422
		case errors.Is(err, tailnet.ErrUnavailable):
			status = 503
		}
		// Input and provider/native error text must never enter journal or response.
		http.Error(w, "Tailnet operation unavailable or unconfirmed; observe before retrying", status)
		return
	}
	data, err := json.Marshal(out)
	if err != nil || len(data) > 65536 {
		http.Error(w, "Tailnet response unavailable", 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
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
func (c *Client) TailnetProject(ctx context.Context, in tailnet.ProjectRequest) (tailnet.ProjectView, error) {
	if err := in.Validate(); err != nil {
		return tailnet.ProjectView{}, err
	}
	var out tailnet.ProjectView
	err := c.tailnetCall(ctx, "project", in, &out)
	if err == nil {
		err = out.Validate()
		if out.Project != in.Project || out.Saved != (in.Action == "disable") || (out.Saved && out.Revision == in.Revision) {
			err = tailnet.ErrUnavailable
		}
	}
	return out, err
}
