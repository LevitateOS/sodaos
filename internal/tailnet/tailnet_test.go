package tailnet

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func statusClient(t *testing.T, output string) *Client {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status.json"), []byte(output), 0600))
	// Exercise the real command boundary, including fixed arguments and no stdin.
	cli := filepath.Join(dir, "tailscale")
	require.NoError(t, os.WriteFile(cli, []byte(`#!/bin/sh
[ "$#" = 2 ] && [ "$1" = status ] && [ "$2" = --json ] || exit 2
if read -r input; then exit 3; fi
cat "${0%/*}/status.json"
`), 0700))
	return New(Options{CLI: cli})
}

func TestClientReadsCanonicalMagicDNSIdentity(t *testing.T) {
	client := statusClient(t, `{"BackendState":"Running","Self":{"DNSName":"Atlas.Example.ts.net."}}`)
	status, err := client.Status(t.Context())
	require.NoError(t, err)
	require.Equal(t, "atlas.example.ts.net", status.Identity)
}

func TestClientReadsTailnetEndpoint(t *testing.T) {
	client := statusClient(t, `{"BackendState":"Running","Self":{"DNSName":"Atlas.Example.ts.net.","TailscaleIPs":["fd7a:115c:a1e0::1","100.88.77.66"]},"CurrentTailnet":{"MagicDNSEnabled":true}}`)
	endpoint, err := client.Endpoint(t.Context())
	require.NoError(t, err)
	require.Equal(t, Endpoint{Identity: "atlas.example.ts.net", IPv4: "100.88.77.66"}, endpoint)
}

func TestEndpointDoesNotAdvertiseUnavailableAccess(t *testing.T) {
	for name, test := range map[string]struct {
		output string
		want   error
	}{
		"needs login": {`{"BackendState":"NeedsLogin","Self":{}}`, ErrNotEnrolled},
		"stopped":     {`{"BackendState":"Stopped","Self":{"TailscaleIPs":["100.88.77.66"]}}`, ErrNotEnrolled},
		"no address":  {`{"BackendState":"Running","Self":{}}`, ErrIPv4Unavailable},
		"IPv6 only":   {`{"BackendState":"Running","Self":{"DNSName":"atlas.example.ts.net","TailscaleIPs":["fd7a:115c:a1e0::1"]}}`, ErrIPv4Unavailable},
		"expired":     {`{"BackendState":"Running","Self":{"Expired":true,"DNSName":"atlas.example.ts.net.","TailscaleIPs":["100.88.77.66"]}}`, ErrNotEnrolled},
	} {
		t.Run(name, func(t *testing.T) {
			endpoint, err := statusClient(t, test.output).Endpoint(t.Context())
			require.ErrorIs(t, err, test.want)
			require.Empty(t, endpoint)
		})
	}
}

func TestStatusRejectsInvalidMagicDNSIdentity(t *testing.T) {
	_, err := statusClient(t, `{"BackendState":"Running","Self":{"DNSName":"atlas.local"}}`).Status(t.Context())
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorIs(t, err, ErrInvalidMagicDNSName)
}

func TestStatusReportsUnavailableCLI(t *testing.T) {
	client := New(Options{CLI: filepath.Join(t.TempDir(), "missing")})
	_, err := client.Status(t.Context())
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestEndpointUsesIPv4WhenMagicDNSIsDisabled(t *testing.T) {
	for _, dns := range []string{"", "atlas.example.ts.net."} {
		client := statusClient(t, `{"BackendState":"Running","Self":{"DNSName":"`+dns+`","TailscaleIPs":["100.88.77.66"]}}`)
		endpoint, err := client.Endpoint(t.Context())
		require.NoError(t, err)
		require.Equal(t, Endpoint{Identity: "100.88.77.66", IPv4: "100.88.77.66"}, endpoint)
	}
}

func TestStatusRejectsMalformedOutputAndPreservesAuthPending(t *testing.T) {
	for _, output := range []string{"", `{}`, `{"BackendState":"Running"`, `{} {}`} {
		_, err := statusClient(t, output).Status(t.Context())
		require.ErrorIs(t, err, ErrUnavailable)
	}
	status, err := statusClient(t, `{"BackendState":"NeedsLogin","AuthURL":"https://fixture.invalid/auth"}`).Status(t.Context())
	require.NoError(t, err)
	require.True(t, status.AuthPending)
}

func TestStatusCommandFailureAndCancellation(t *testing.T) {
	cli := filepath.Join(t.TempDir(), "tailscale")
	client := New(Options{CLI: cli})
	require.NoError(t, os.WriteFile(cli, []byte("#!/bin/sh\nprintf '{\"BackendState\":\"Running\"}'\nprintf 'daemon unavailable' >&2\nexit 7\n"), 0700))
	status, err := client.Status(t.Context())
	require.Empty(t, status, "nonzero exit never supplies status")
	require.ErrorIs(t, err, ErrUnavailable)
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError)
	require.Equal(t, 7, exitError.ExitCode())
	require.ErrorContains(t, err, "daemon unavailable")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = client.Status(ctx)
	require.ErrorIs(t, err, context.Canceled)

	// exec keeps only the test-owned child; cancellation must not wait for sleep.
	require.NoError(t, os.WriteFile(cli, []byte("#!/bin/sh\nexec sleep 60\n"), 0700))
	ctx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = client.Status(ctx)
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.Less(t, time.Since(start), 5*time.Second)
}
