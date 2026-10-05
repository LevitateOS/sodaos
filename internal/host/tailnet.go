package host

import (
	"context"
	"errors"
	"slices"

	"github.com/levitateos/sodaos/internal/tailnet"
)

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

func admitEnrollmentSave(out tailnet.EnrollmentResult, in tailnet.EnrollmentRequest) error {
	if out.Saved != (in.Action != "check") {
		return tailnet.ErrUnavailable
	}
	if out.Saved && (out.Enrollment.Revision == in.Revision || !out.Enrollment.Configured) {
		return tailnet.ErrUnavailable
	}
	if !out.Saved && (!out.CredentialChecked || out.Enrollment.Revision != in.Revision) {
		return tailnet.ErrUnavailable
	}
	return nil
}

func admitEnrollmentSaveRotate(out tailnet.EnrollmentResult, in tailnet.EnrollmentRequest) error {
	if out.Enrollment.Tailnet != in.Tailnet || !slices.Equal(out.Enrollment.Tags, in.Tags) || out.Enrollment.Preauthorized != *in.Preauthorized {
		return tailnet.ErrUnavailable
	}
	return nil
}

func admitEnrollmentAction(out tailnet.EnrollmentResult, in tailnet.EnrollmentRequest) error {
	switch in.Action {
	case "save", "rotate":
		return admitEnrollmentSaveRotate(out, in)
	case "disable":
		if out.Enrollment.Admission || out.Enrollment.Default {
			return tailnet.ErrUnavailable
		}
	case "default":
		if out.Enrollment.Default != *in.Default {
			return tailnet.ErrUnavailable
		}
	}
	return nil
}

func (c *Client) TailnetEnrollment(ctx context.Context, in tailnet.EnrollmentRequest) (tailnet.EnrollmentResult, error) {
	if err := in.Validate(); err != nil {
		return tailnet.EnrollmentResult{}, err
	}
	var out tailnet.EnrollmentResult
	err := c.tailnetCall(ctx, "enrollment", in, &out)
	if err != nil {
		return out, err
	}
	err = out.Validate()
	if e := admitEnrollmentSave(out, in); e != nil {
		err = e
	}
	if e := admitEnrollmentAction(out, in); e != nil {
		err = e
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

func tailnetProjectConfirmed(in tailnet.ProjectRequest, out tailnet.ProjectView) bool {
	if out.Project != in.Project || out.Saved != (in.Action != "inspect") {
		return false
	}
	if !out.Saved {
		return true
	}
	if out.Revision == in.Revision || out.Enabled != (in.Action != "disable") {
		return false
	}
	return in.Action == "disable" || out.Binding == in.Binding
}

func (c *Client) TailnetProject(ctx context.Context, in tailnet.ProjectRequest) (tailnet.ProjectView, error) {
	if err := in.Validate(); err != nil {
		return tailnet.ProjectView{}, err
	}
	var out tailnet.ProjectView
	err := c.tailnetCall(ctx, "project", in, &out)
	if err == nil {
		err = out.Validate()
		if !tailnetProjectConfirmed(in, out) {
			err = tailnet.ErrUnavailable
		}
	}
	return out, err
}
