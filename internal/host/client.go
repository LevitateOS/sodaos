package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/levitateos/sodaos/internal/project"
)

type (
	Client          struct{ HTTP *http.Client }
	nativeHTTPError struct{ status int }
	// FJ-874: typed setup-route refusal. The code is one of
	// "committed", "conflict", or "unconfirmed" and comes only from
	// the fixed (status, body) wire codes the native setup route
	// pins — never from host stderr or free text.
	setupClaimError struct {
		status int
		code   string
	}
)

func (e nativeHTTPError) Error() string {
	return fmt.Sprintf("native project operation failed (HTTP %d); operator should inspect soda-host journal", e.status)
}

func (e setupClaimError) Error() string {
	// Fixed literals only, prefix-compatible with the API layer's
	// existing setup-claim mapping; no host text is surfaced here.
	switch e.code {
	case "committed":
		return "setup claim is committed; renewal refused with the retained credential kept"
	case "conflict":
		return "setup claim conflicts; renewal refused with the retained credential kept"
	default:
		return "setup claim is unconfirmed; renewal refused with the retained credential kept"
	}
}

func NewClient(socket string) *Client {
	return &Client{HTTP: &http.Client{Timeout: 4 * time.Minute, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func decodeNativeResponseLimit(body []byte, out any, limit int) error {
	if len(body) > limit || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return fmt.Errorf("invalid native response")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("invalid native response")
	}
	return nil
}

// FJ-874: setup-route response reader. On 200 it behaves exactly like
// readNativeResponseLimit; on any other status it retains the body's
// fixed (status, body) claim code as a typed setupClaimError instead of
// discarding the body. Only the /application/setup route uses this —
// every other caller keeps using readNativeResponseLimit unchanged.
func readNativeSetupResponseLimit(res *http.Response, out any, limit int) error {
	if res.StatusCode == 200 {
		return readNativeResponseLimit(res, out, limit)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 256))
	if err != nil {
		return setupClaimError{status: res.StatusCode, code: "unconfirmed"}
	}
	switch {
	case res.StatusCode == 409 && string(body) == "factory run incarnation changed\n":
		return setupClaimError{status: res.StatusCode, code: "committed"}
	case res.StatusCode == 409 && string(body) == "factory output incarnation changed\n":
		return setupClaimError{status: res.StatusCode, code: "conflict"}
	default:
		return setupClaimError{status: res.StatusCode, code: "unconfirmed"}
	}
}

func readNativeResponseLimit(res *http.Response, out any, limit int) error {
	if res.StatusCode != 200 {
		return nativeHTTPError{res.StatusCode}
	}
	if out == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, int64(limit+1)))
	if err != nil {
		return fmt.Errorf("invalid native response")
	}
	return decodeNativeResponseLimit(body, out, limit)
}

func (c *Client) call(ctx context.Context, path string, in, out any) error {
	return c.callLimit(ctx, path, in, out, 65536)
}

func (c *Client) callLimit(ctx context.Context, path string, in, out any, limit int) error {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://soda-host"+path, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("host service unavailable")
	}
	defer res.Body.Close()
	return readNativeResponseLimit(res, out, limit)
}

// FJ-874: setup-route call path. Identical transport to callLimit, but
// retains the typed setup-claim code on refusal. Only ApplicationSetup
// uses this; every other route keeps using call/callLimit unchanged.
func (c *Client) callSetupLimit(ctx context.Context, path string, in, out any, limit int) error {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://soda-host"+path, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("host service unavailable")
	}
	defer res.Body.Close()
	return readNativeSetupResponseLimit(res, out, limit)
}

func (c *Client) Create(ctx context.Context, in project.Create) (project.Environment, error) {
	var out project.Environment
	err := c.call(ctx, "/create", in, &out)
	if err == nil && (out.ID != in.ID || !out.Running || !validAddress(out.IP) || in.Profile == nil || out.Profile == nil || *out.Profile != *in.Profile) {
		err = fmt.Errorf("native creation did not return the expected running endpoint")
	}
	return out, err
}

func (c *Client) Inspect(ctx context.Context, id string) (project.Environment, error) {
	var out project.Environment
	err := c.call(ctx, "/inspect", project.Create{ID: id}, &out)
	if err == nil && (out.ID != id || (out.IP != "" && !validAddress(out.IP))) {
		err = fmt.Errorf("invalid native environment observation")
	}
	return out, err
}

func (c *Client) Join(ctx context.Context, in project.Account) error {
	var result struct {
		OK bool `json:"ok"`
	}
	err := c.call(ctx, "/account", in, &result)
	if err == nil && !result.OK {
		err = fmt.Errorf("native account result was not confirmed")
	}
	return err
}

// ProjectAccess observes native Project administrator status for the current
// project account. False is a confirmed nonadministrator result.
func (c *Client) ProjectAccess(ctx context.Context, in project.ProjectAccessRequest) (project.ProjectAccessStatus, error) {
	var out project.ProjectAccessStatus
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/project-access", in, &out, 4096)
	if err == nil && (out.Administrator == nil || out.Project != in.Project || out.Login != in.Login || out.Identity != in.Identity) {
		err = fmt.Errorf("native project privilege observation did not match the request")
	}
	return out, err
}

func (c *Client) Connection(ctx context.Context, id string) (project.Connection, error) {
	var result project.Connection
	err := c.call(ctx, "/connection", project.Create{ID: id}, &result)
	if err == nil && (result.Environment.ID != id || (result.Environment.Running && (!validAddress(result.Environment.IP) || result.HostKey == "" || result.Fingerprint == ""))) {
		err = fmt.Errorf("native connection result incomplete")
	}
	return result, err
}

func validAddress(value string) bool {
	ip, err := netip.ParseAddr(value)
	return err == nil && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLoopback()
}
