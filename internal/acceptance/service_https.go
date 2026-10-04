// Configured-origin TLS observation from the intended client. Go port of the
// retired tests/installed/service-https.py: no insecure mode, redirect
// journey, login, cookie jar or response-body capture. It is not appliance
// runtime code.
package acceptance

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ServiceHTTPSUsage documents the probe's CLI surface.
const ServiceHTTPSUsage = "soda-installed-probes service-https ORIGIN CA_FILE"

// UsageError reports a command-line rejection the entrypoint must exit 2 on,
// matching argparse's exit status for invalid arguments.
type UsageError struct {
	Probe   string
	Message string
}

func (e *UsageError) Error() string {
	return "usage: " + e.Probe + "\n" + e.Probe + ": error: " + e.Message
}

// HTTPSOrigin validates a plain configured HTTPS origin and returns its
// canonical trailing-slash form. It mirrors the retired Python origin()
// cases, including the distinct invalid-port message.
func HTTPSOrigin(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.New("invalid HTTPS origin")
	}
	if err := checkHTTPSPort(parsed.Port()); err != nil {
		return "", err
	}
	if !plainHTTPSOrigin(parsed, value) {
		return "", errors.New("plain configured HTTPS origin required")
	}
	return strings.TrimRight(value, "/") + "/", nil
}

func checkHTTPSPort(port string) error {
	if port == "" {
		return nil
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return errors.New("invalid HTTPS origin")
	}
	if number == 0 {
		return errors.New("plain configured HTTPS origin required")
	}
	return nil
}

func plainHTTPSOrigin(parsed *url.URL, value string) bool {
	if !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.EscapedPath() != "" && parsed.EscapedPath() != "/" {
		return false
	}
	if strings.Contains(value, "?") || strings.Contains(value, "#") {
		return false
	}
	return true
}

// trustedCAFile validates the absolute trusted regular CA file. It must not
// be writable by group/others; readability is unrestricted, as before.
func trustedCAFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("absolute trusted regular CA file required")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, errors.New("absolute trusted regular CA file required")
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return nil, errors.New("absolute trusted regular CA file required")
	}
	if st.Mode().Perm()&0o022 != 0 {
		return nil, errors.New("absolute trusted regular CA file required")
	}
	return os.ReadFile(path)
}

// httpsClient builds the direct client: no proxy, no redirect following, and
// trust rooted only in the supplied CA file.
func httpsClient(pool *x509.CertPool) *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
			TLSClientConfig: &tls.Config{
				RootCAs:    pool,
				MinVersion: tls.VersionTLS12,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// CheckServiceHTTPS verifies configured-origin TLS and reports the HTTP
// status. URLs, redirect queries and response bodies never surface.
func CheckServiceHTTPS(origin, ca string, stdout io.Writer) error {
	data, err := trustedCAFile(ca)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return errors.New("absolute trusted regular CA file required")
	}
	response, err := httpsClient(pool).Get(origin)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return errors.New("unexpected service response")
	}
	_, err = fmt.Fprintf(stdout, "Configured-origin TLS verified; HTTP status %d (not an authentication/product observation).\n", response.StatusCode)
	return err
}

// RunServiceHTTPS is the service-https entrypoint: ORIGIN CA_FILE.
func RunServiceHTTPS(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return &UsageError{Probe: ServiceHTTPSUsage, Message: "origin and CA file required"}
	}
	origin, err := HTTPSOrigin(args[0])
	if err != nil {
		return &UsageError{Probe: ServiceHTTPSUsage, Message: err.Error()}
	}
	if err := CheckServiceHTTPS(origin, args[1], stdout); err != nil {
		return failParen("HTTPS substrate check failed (", ").", err)
	}
	return nil
}
