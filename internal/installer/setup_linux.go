package installer

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/installlayout"
)

const localCAPath = "/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt"

type setupAddress struct{ Interface, Address string }

func privateSetupOrigin(value string) (string, error) {
	address, err := netip.ParseAddr(value)
	if err != nil || address.Zone() != "" || address.Is4In6() ||
		(!address.IsPrivate() && !netip.MustParsePrefix("100.64.0.0/10").Contains(address)) {
		return "", errors.New("select an assigned private LAN or Tailscale address")
	}
	host := address.String()
	if address.Is6() {
		host = "[" + host + "]"
	}
	return (&url.URL{Scheme: "https", Host: host}).String(), nil
}

func setupAddresses(data []byte) ([]setupAddress, error) {
	var interfaces []struct {
		Name      string   `json:"ifname"`
		Flags     []string `json:"flags"`
		Addresses []struct {
			Local string `json:"local"`
			Scope string `json:"scope"`
		} `json:"addr_info"`
	}
	if err := json.Unmarshal(data, &interfaces); err != nil {
		return nil, errors.New("cannot inspect private setup addresses")
	}
	var choices []setupAddress
	seen := map[string]bool{}
	for _, network := range interfaces {
		up := false
		for _, flag := range network.Flags {
			up = up || flag == "UP"
		}
		if !up || network.Name == "soda0" || strings.HasPrefix(network.Name, "podman") || strings.HasPrefix(network.Name, "veth") {
			continue
		}
		for _, address := range network.Addresses {
			if _, err := privateSetupOrigin(address.Local); err == nil && address.Scope == "global" && !seen[address.Local] {
				choices = append(choices, setupAddress{network.Name, address.Local})
				seen[address.Local] = true
			}
		}
	}
	if len(choices) == 0 {
		return nil, errors.New("no private setup address is available; configure networking first")
	}
	return choices, nil
}

// configureInstall uses the existing native Forgejo installer and setup command.
// Soda does not create a second account/password authority or expose the unfinished
// Forgejo installer. Initial browser access remains an operator SSH tunnel.
func configureInstall(ctx context.Context, c console, run commandRunner) error {
	if os.Getenv("SSH_CONNECTION") == "" || os.Getenv("SSH_TTY") == "" {
		return errors.New("run configure in your laptop's interactive SSH terminal so you can paste the Forgejo token; first use enroll-key at the local console if needed")
	}
	return configurePrivateInstall(ctx, c, run, "/etc/soda", "/run", localCAPath)
}

func checkPreexistingInstall(root string) (bool, error) {
	if _, err := readRegular(filepath.Join(root, "installed"), 256); err != nil {
		return false, errors.New("install the included Soda components before configuring browser access")
	}
	if _, err := os.Lstat(filepath.Join(root, "activated")); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, errors.New("cannot inspect existing activation")
	}
	for _, name := range []string{"dashboard.json", "setup-started"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			return false, errors.New("existing or partial operator setup requires inspection; do not create another OAuth application")
		}
	}
	return false, nil
}

func querySetupAddresses(ctx context.Context, run commandRunner) ([]setupAddress, error) {
	data, err := run(ctx, "ip", []string{"-json", "address", "show", "up"}, nil)
	if err != nil {
		return nil, errors.New("cannot inspect network addresses")
	}
	return setupAddresses(data)
}

func promptSetupAddress(c console, choices []setupAddress) (setupAddress, error) {
	c.page("Private browser setup")
	c.print("Use this SSH terminal to paste the Forgejo token when asked; input will be hidden.")
	c.print("Select the appliance address your laptop can reach. No domain is needed.")
	for i, choice := range choices {
		c.print("%d. %s on %q", i+1, choice.Address, choice.Interface)
	}
	for {
		answer, err := c.ask("Address number, or cancel")
		if err != nil {
			return setupAddress{}, err
		}
		if answer == "cancel" {
			return setupAddress{}, errors.New("browser setup cancelled")
		}
		i, err := strconv.Atoi(answer)
		if err == nil && i > 0 && i <= len(choices) {
			return choices[i-1], nil
		}
		c.print("Choose one of the listed address numbers.")
	}
}

func validOperatorToken(token string) bool {
	return token != "" && token == strings.TrimSpace(token) && !strings.ContainsAny(token, "\r\n\x00")
}

func promptOperatorToken(c console, address, origin string) (string, error) {
	c.print("The final Soda address will be %s", origin)
	c.print("Use a stable address or DHCP reservation. Changing it later needs explicit configuration maintenance.")
	c.print("If you have no SSH key access yet, cancel and run %s enroll-key at the local console.", installerBinary)
	c.print("From your laptop, connect with an SSH tunnel to the native Forgejo installer:")
	c.print("ssh -L 33000:127.0.0.1:3000 root@%s", address)
	c.print("Open http://localhost:33000 and complete Forgejo's own installation and administrator account setup.")
	c.print("Keep its localhost browser URL for this bootstrap; activation below sets the final private URL.")
	c.print("In Forgejo Settings > Applications, create the operator token described in the operator setup guide.")
	c.print("Required scope: write:user (includes read:user). No admin or repository scope is needed. Paste it here through the SSH terminal; input is hidden.")
	c.print("Caddy will issue local HTTPS certificates. You will explicitly trust its public root certificate on your laptop.")
	answer, err := c.ask("When Forgejo setup is complete, type CONFIGURE SODA; anything else cancels")
	if err != nil {
		return "", err
	}
	if answer != "CONFIGURE SODA" {
		return "", errors.New("browser setup cancelled; no Soda configuration written")
	}
	token, err := c.secret("Operator Forgejo token")
	if err != nil {
		return "", err
	}
	if !validOperatorToken(token) {
		return "", errors.New("operator token is empty or malformed")
	}
	return token, nil
}

func verifyAddressStillAssigned(ctx context.Context, run commandRunner, selected setupAddress) error {
	current, err := querySetupAddresses(ctx, run)
	if err != nil {
		return errors.New("cannot recheck the selected address")
	}
	for _, choice := range current {
		if choice == selected {
			return nil
		}
	}
	return errors.New("selected network address changed; restart browser setup")
}

func executeSetupAndActivation(ctx context.Context, run commandRunner, root, temporary, origin, token string, selected setupAddress) error {
	work, err := os.MkdirTemp(temporary, "soda-setup-")
	if err != nil {
		return err
	}
	tokenPath := filepath.Join(work, "operator-token")
	if err := writeSetupFile(tokenPath, []byte(token+"\n")); err != nil {
		return err
	}
	if err := writeSetupFile(filepath.Join(root, "setup-started"), []byte(origin+"\n")); err != nil {
		return errors.New("cannot reserve operator setup; no OAuth request made")
	}
	if _, err := run(ctx, installlayout.Sbin+"/soda-setup", []string{"--forgejo-url", origin, "--token-file", tokenPath, "--out", filepath.Join(root, "dashboard.json")}, nil); err != nil {
		return errors.New("operator setup failed; preserve private inputs and inspect native Forgejo applications before retrying. " + failureSummary(err))
	}
	if _, err := run(ctx, installlayout.Sbin+"/soda-activate", []string{"--bind-ip", selected.Address, "--local-tls"}, nil); err != nil {
		return errors.New("private activation failed; preserve the existing configuration for inspection. " + failureSummary(err))
	}
	return nil
}

func configurePrivateInstall(ctx context.Context, c console, run commandRunner, root, temporary, caPath string) error {
	activated, err := checkPreexistingInstall(root)
	if err != nil {
		return err
	}
	if activated {
		return configuredAccess(ctx, c, root, caPath, run)
	}
	choices, err := querySetupAddresses(ctx, run)
	if err != nil {
		return err
	}
	selected, err := promptSetupAddress(c, choices)
	if err != nil {
		return err
	}
	origin, _ := privateSetupOrigin(selected.Address)
	token, err := promptOperatorToken(c, selected.Address, origin)
	if err != nil {
		return err
	}
	if err := verifyAddressStillAssigned(ctx, run, selected); err != nil {
		return err
	}
	if err := executeSetupAndActivation(ctx, run, root, temporary, origin, token, selected); err != nil {
		return err
	}
	return configuredAccess(ctx, c, root, caPath, run)
}

func writeSetupFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func validInstalledHTTPSOrigin(origin *url.URL, raw string) bool {
	if origin.Scheme != "https" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	if origin.Path != "" && origin.Path != "/" {
		return false
	}
	return !strings.ContainsAny(raw, "\r\n\x00")
}

func installedForgejoOrigin(root string) (*url.URL, string, error) {
	data, err := readRegular(filepath.Join(root, "dashboard.json"), 65536)
	if err != nil {
		return nil, "", errors.New("cannot read installed browser address")
	}
	var config struct {
		URL string `json:"forgejo_url"`
	}
	if json.Unmarshal(data, &config) != nil {
		return nil, "", errors.New("invalid installed browser configuration")
	}
	origin, err := url.Parse(config.URL)
	if err != nil || !validInstalledHTTPSOrigin(origin, config.URL) {
		return nil, "", errors.New("invalid installed HTTPS address")
	}
	return origin, config.URL, nil
}

func usesInternalTLS(root string) (bool, error) {
	proxy, err := readRegular(filepath.Join(root, "proxy.env"), 65536)
	if err != nil {
		return false, errors.New("cannot inspect the installed TLS mode")
	}
	local := false
	for _, line := range strings.Split(string(proxy), "\n") {
		local = local || line == "SODA_TLS=internal"
	}
	return local, nil
}

func confirmActiveBrowserUnits(ctx context.Context, run commandRunner) error {
	for _, unit := range []string{"forgejo.service", "soda-dashboard.service", "soda-proxy.service"} {
		if _, err := run(ctx, "systemctl", []string{"is-active", "--quiet", unit}, nil); err != nil {
			return fmt.Errorf("private activation was requested, but %s is not confirmed active; inspect its native service state, then rerun configure for read-only guidance; setup was not replayed", unit)
		}
	}
	return nil
}

func printLocalCAGuidance(c console, origin *url.URL, caPath string) error {
	certificate, err := readRegular(caPath, 16384)
	if err != nil {
		c.print("The local certificate is not available yet. Inspect soda-proxy.service, then run configure again to show the trust instructions. Existing setup will not be replayed.")
		return nil
	}
	fingerprint, err := localCAFingerprint(certificate)
	if err != nil {
		return err
	}
	c.print("Local CA certificate SHA-256: %s", fingerprint)
	c.print("Copy only the public root.crt file over your verified SSH connection:")
	c.print("scp root@%s:%s ./soda-local-ca.crt", origin.Host, caPath)
	c.print("Compare its certificate fingerprint, then trust it in your laptop/browser certificate settings. Never copy the CA private key.")
	return nil
}

func configuredAccess(ctx context.Context, c console, root, caPath string, run commandRunner) error {
	origin, address, err := installedForgejoOrigin(root)
	if err != nil {
		return err
	}
	local, err := usesInternalTLS(root)
	if err != nil {
		return err
	}
	if local {
		expected, err := privateSetupOrigin(origin.Hostname())
		if err != nil || strings.TrimSuffix(address, "/") != expected {
			return errors.New("local TLS must use the selected private IP origin")
		}
	}
	if err := confirmActiveBrowserUnits(ctx, run); err != nil {
		return err
	}
	c.print("The browser services report active. Open %s after setting up client trust; browser login still needs verification.", address)
	if local {
		if err := printLocalCAGuidance(c, origin, caPath); err != nil {
			return err
		}
	}
	c.print("Sign in to native Forgejo, create or choose a repository, and open Sodaspaces to create and join its development environment.")
	c.print("Verify a browser terminal in that project. Opening this setup screen is not a completed project/access test.")
	return nil
}

func localCAFingerprint(data []byte) (string, error) {
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || len(strings.TrimSpace(string(rest))) != 0 {
		return "", errors.New("expected one public CA certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !certificate.IsCA || !certificate.BasicConstraintsValid || certificate.CheckSignatureFrom(certificate) != nil {
		return "", errors.New("invalid local CA certificate")
	}
	return fmt.Sprintf("%x", sha256.Sum256(certificate.Raw)), nil
}
