package installer

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPrivateSetupAddresses(t *testing.T) {
	for address, expected := range map[string]string{
		"192.168.2.100": "https://192.168.2.100",
		"100.90.1.2":    "https://100.90.1.2",
		"fd00::123":     "https://[fd00::123]",
	} {
		if got, err := privateSetupOrigin(address); err != nil || got != expected {
			t.Fatalf("private origin: %q, %v", got, err)
		}
	}
	for _, address := range []string{"127.0.0.1", "::1", "0.0.0.0", "8.8.8.8", "169.254.1.2", "fe80::1%eth0", "192.168.1.2;reboot", "soda.example.test", "::ffff:192.168.1.2"} {
		if _, err := privateSetupOrigin(address); err == nil {
			t.Errorf("unusable/private-setup address accepted: %q", address)
		}
	}
	data := []byte(`[
		{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"},{"local":"8.8.8.8","scope":"global"}]},
		{"ifname":"eth1","flags":[],"addr_info":[{"local":"10.0.0.2","scope":"global"}]},
		{"ifname":"soda0","flags":["UP"],"addr_info":[{"local":"10.89.0.1","scope":"global"}]},
		{"ifname":"lo","flags":["UP"],"addr_info":[{"local":"127.0.0.1","scope":"host"}]}
	]`)
	addresses, err := setupAddresses(data)
	if err != nil || len(addresses) != 1 || addresses[0] != (setupAddress{"eth0", "192.168.1.5"}) {
		t.Fatalf("wrong eligible interfaces: %v, %v", addresses, err)
	}
	for _, data := range []string{"not json", "[]"} {
		if _, err := setupAddresses([]byte(data)); err == nil {
			t.Fatal("missing network accepted")
		}
	}
}

func TestLocalCertificateExportRequiresPublicCA(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, ca := range []bool{false, true} {
		template := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: ca, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		encoded := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		fingerprint, err := localCAFingerprint(encoded)
		if ca && (err != nil || len(fingerprint) != 64) || !ca && err == nil {
			t.Fatalf("unexpected CA fingerprint result: %v", err)
		}
		if _, err := localCAFingerprint(append(encoded, []byte("private trailing data")...)); err == nil {
			t.Fatal("unexpected extra certificate data accepted")
		}
	}
	if _, err := localCAFingerprint(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("synthetic")})); err == nil {
		t.Fatal("private key accepted as a public certificate")
	}
}

func TestPrivateSetupKeepsCredentialOutOfCommandsAndTranscript(t *testing.T) {
	for _, cancelSetup := range []bool{false, true} {
		t.Run(map[bool]string{false: "configure", true: "cancel"}[cancelSetup], func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "installed"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			master, slave := openTestPTY(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c := console{tty: slave, ctx: ctx}
			const token = "synthetic-operator-token"
			mutations := 0
			run := func(_ context.Context, name string, args []string, input io.Reader) ([]byte, error) {
				if input != nil || strings.Contains(strings.Join(args, " "), token) {
					t.Error("credential reached command arguments or unexpected stdin")
				}
				if name == "ip" {
					return []byte(`[{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"}]}]`), nil
				}
				if name == "systemctl" {
					if len(args) != 3 || args[0] != "is-active" || args[1] != "--quiet" {
						t.Fatal("setup guidance changed service state")
					}
					return nil, nil
				}
				mutations++
				if name == "/usr/local/sbin/soda-setup" {
					if len(args) != 6 || args[0] != "--forgejo-url" || args[1] != "https://192.168.1.5" || args[2] != "--token-file" || args[4] != "--out" {
						t.Errorf("unexpected setup arguments: %v", args)
						return nil, io.ErrUnexpectedEOF
					}
					data, err := os.ReadFile(args[3])
					st, statErr := os.Stat(args[3])
					if err != nil || statErr != nil || string(data) != token+"\n" || st.Mode().Perm() != 0600 {
						t.Error("token was not passed in a restricted file")
					}
					config, _ := json.Marshal(map[string]string{"forgejo_url": args[1]})
					return nil, os.WriteFile(args[5], config, 0600)
				}
				if name != "/usr/local/sbin/soda-activate" || strings.Join(args, " ") != "--bind-ip 192.168.1.5 --local-tls" {
					t.Errorf("unexpected activation: %s %v", name, args)
				}
				return nil, os.WriteFile(filepath.Join(root, "proxy.env"), []byte("SODA_TLS=internal\n"), 0600)
			}
			done := make(chan error, 1)
			go func() {
				done <- configurePrivateInstall(ctx, c, run, root, root, filepath.Join(root, "not-yet-created-ca.crt"))
			}()
			var transcript strings.Builder
			readUntil := func(needle string) {
				t.Helper()
				for !strings.Contains(transcript.String(), needle) && ctx.Err() == nil {
					var data [4096]byte
					n, _ := unix.Read(master, data[:])
					if n > 0 {
						transcript.Write(data[:n])
					}
					time.Sleep(time.Millisecond)
				}
				if ctx.Err() != nil {
					t.Fatalf("missing setup prompt: %s", needle)
				}
			}
			readUntil("Address number, or cancel")
			unix.Write(master, []byte("1\n"))
			readUntil("When Forgejo setup is complete")
			if cancelSetup {
				unix.Write(master, []byte("cancel\n"))
			} else {
				unix.Write(master, []byte("CONFIGURE SODA\n"))
				readUntil("Operator Forgejo token: ")
				unix.Write(master, []byte(token+"\n"))
			}
			err := <-done
			if cancelSetup {
				if err == nil || mutations != 0 {
					t.Fatal("cancelled setup made native changes")
				}
				if _, err := os.Stat(filepath.Join(root, "setup-started")); !os.IsNotExist(err) {
					t.Fatal("cancelled setup reserved an attempt")
				}
			} else {
				if err != nil || mutations != 2 {
					t.Fatalf("setup failed: %v, mutations %d", err, mutations)
				}
				readUntil("certificate is not available yet")
			}
			if strings.Contains(transcript.String(), token) {
				t.Fatal("operator token echoed to console")
			}
		})
	}
}

func TestPrivateSetupDoesNotReplayExistingState(t *testing.T) {
	for _, state := range []string{"dashboard.json", "setup-started", "activated"} {
		t.Run(state, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"installed", state} {
				if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, tty := openTestPTY(t)
			run := func(context.Context, string, []string, io.Reader) ([]byte, error) {
				t.Fatal("existing state caused a native command or another setup attempt")
				return nil, errors.New("unexpected command")
			}
			// Empty existing config fails read-only, without enrolling another OAuth client.
			if err := configurePrivateInstall(context.Background(), console{tty: tty}, run, root, root, "missing-ca"); err == nil {
				t.Fatal("incomplete existing state accepted")
			}
		})
	}
}

func TestLocalTrustGuidanceRejectsUntrustedDestination(t *testing.T) {
	for _, origin := range []string{"https://8.8.8.8", "https://soda.example.test", "https://192.168.1.2:444", "https://192.168.1.2/path", "https://root@192.168.1.2", "https://192.168.1.2?argument"} {
		t.Run(origin, func(t *testing.T) {
			root := t.TempDir()
			config, _ := json.Marshal(map[string]string{"forgejo_url": origin})
			if err := os.WriteFile(filepath.Join(root, "dashboard.json"), config, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "proxy.env"), []byte("SODA_TLS=internal\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, tty := openTestPTY(t)
			if err := configuredAccess(context.Background(), console{tty: tty}, root, "missing-ca", func(context.Context, string, []string, io.Reader) ([]byte, error) {
				t.Fatal("invalid origin reached native service inspection")
				return nil, errors.New("unexpected command")
			}); err == nil {
				t.Fatal("invalid local certificate copy destination accepted")
			}
		})
	}
}

func TestConfigureRequiresLaptopTerminalBeforeInput(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	if err := configureInstall(context.Background(), console{}, func(context.Context, string, []string, io.Reader) ([]byte, error) {
		t.Fatal("local configure reached native operations")
		return nil, errors.New("unexpected command")
	}); err == nil || !strings.Contains(err.Error(), "laptop") {
		t.Fatalf("local console was not redirected to SSH: %v", err)
	}
}

func TestConfiguredGuidanceReportsInactiveServiceWithoutReplay(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "dashboard.json"), []byte(`{"forgejo_url":"https://192.168.1.5"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "proxy.env"), []byte("SODA_TLS=internal\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, tty := openTestPTY(t)
	var observed []string
	err := configuredAccess(context.Background(), console{tty: tty}, root, "missing-ca", func(_ context.Context, command string, args []string, input io.Reader) ([]byte, error) {
		if command != "systemctl" || len(args) != 3 || args[0] != "is-active" || args[1] != "--quiet" || input != nil {
			t.Fatal("configured guidance attempted a mutation")
		}
		observed = append(observed, args[2])
		if args[2] == "soda-dashboard.service" {
			return nil, errors.New("synthetic inactive unit")
		}
		return nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "soda-dashboard.service") || len(observed) != 2 {
		t.Fatalf("incomplete activation was not reported: %v, %v", err, observed)
	}
}
