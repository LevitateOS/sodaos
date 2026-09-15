package qualify

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/image"
	"golang.org/x/crypto/ssh"
)

// Reviewed against the unchanged native console contract, not learned from the
// candidate under test. Screenshots and secret input remain private.
//
//go:embed console-regions.json
var consoleRegions []byte

func writeInstallPassword(work string) (string, error) {
	var random [24]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	password := hex.EncodeToString(random[:])
	return password, build.WriteNew(filepath.Join(work, "b-password"), []byte(password+"\n"), 0o600)
}

func prepareInstallDisk(ctx context.Context, c Config, f *fixture) (key, known, disk, variables string, pub []byte, err error) {
	key = filepath.Join(c.Work, "b-key")
	known = filepath.Join(c.Work, "b-known-hosts")
	if _, err = f.run(ctx, "ssh-key", "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "soda-p9-install", "-f", key); err != nil {
		return key, known, disk, variables, pub, err
	}
	pub, err = os.ReadFile(key + ".pub")
	if err != nil {
		return key, known, disk, variables, pub, err
	}
	disk = filepath.Join(c.Work, "b.qcow2")
	variables = filepath.Join(c.Work, "b-vars.fd")
	if _, err = f.run(ctx, "blank-install-disk", "qemu-img", "create", "-f", "qcow2", disk, "64G"); err != nil {
		return key, known, disk, variables, pub, err
	}
	err = exec.CommandContext(ctx, "cp", "--", c.Variables, variables).Run()
	return key, known, disk, variables, pub, err
}

func assertTargetUnwritten(ctx context.Context, m *machine, e *acceptance.Evidence) error {
	var blocks []struct {
		Device string
		Stats  struct {
			Writes uint64 `json:"wr_operations"`
			Bytes  uint64 `json:"wr_bytes"`
		}
	}
	if err := m.vm.Console().Execute(ctx, "query-blockstats", "before-erasure", nil, &blocks); err != nil {
		return err
	}
	found := false
	for _, block := range blocks {
		if block.Device == "target-disk" {
			found = true
			if block.Stats.Writes != 0 || block.Stats.Bytes != 0 {
				return errors.New("target written before ERASE confirmation")
			}
		}
	}
	if !found {
		return errors.New("target disk write counter unavailable")
	}
	return e.WriteJSON("before-erasure.json", blocks)
}

func typeInstallPrompt(ctx context.Context, m *machine, regions map[string]acceptance.ConsoleRegion, work, name, text string, e *acceptance.Evidence) error {
	region, ok := regions[name]
	if !ok {
		return errors.New("unreviewed console prompt")
	}
	region.Y = -1
	wait, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	if err := m.vm.Console().WaitConsole(wait, filepath.Join(work, "b-"+name+".png"), region); err != nil {
		return err
	}
	if name == "erase" {
		if err := assertTargetUnwritten(ctx, m, e); err != nil {
			return err
		}
	}
	return m.vm.Console().TypeConsole(ctx, text)
}

func typeConsoleSteps(ctx context.Context, m *machine, regions map[string]acceptance.ConsoleRegion, work string, steps [][2]string, e *acceptance.Evidence) error {
	for _, step := range steps {
		if err := typeInstallPrompt(ctx, m, regions, work, step[0], step[1], e); err != nil {
			return err
		}
	}
	return nil
}

func installPromptSteps(password string) [][2]string {
	return [][2]string{{"welcome", "\n"}, {"network", "keep\n"}, {"network-confirm", "yes\n"}, {"disk", "1\n"}, {"hostname", "soda-tester\n"}, {"password", password + "\n"}, {"password-confirm", password + "\n"}, {"subnet", "\n"}, {"erase", "ERASE /dev/vda\n"}, {"installed", ""}}
}

func firstBootPromptSteps(password string, pub []byte) [][2]string {
	return [][2]string{{"login", "root\n"}, {"root-password", password + "\n"}, {"root", "umask 077; mkdir -p /root/.ssh; echo '" + strings.TrimSpace(string(pub)) + "' > /root/.ssh/authorized_keys; chmod 600 /root/.ssh/authorized_keys; restorecon -RF /root/.ssh; cat /etc/ssh/ssh_host_ed25519_key.pub > /dev/ttyS0; yes P9_PUBLIC_PADDING | head -n 100 > /dev/ttyS0; clear\n"}}
}

func completeMediaFreeBoot(ctx context.Context, m *machine, f *fixture) error {
	if err := m.vm.EjectInstallationMedia(ctx); err != nil {
		return err
	}
	if err := f.rootfs.Close(); err != nil {
		return err
	}
	f.rootfs = nil
	return m.vm.Console().Execute(ctx, "system_reset", "media-free-first-boot", nil, nil)
}

func runInstallConsole(ctx context.Context, c Config, m *machine, f *fixture, password string, pub []byte, e *acceptance.Evidence) error {
	var regions map[string]acceptance.ConsoleRegion
	if err := json.Unmarshal(consoleRegions, &regions); err != nil {
		return err
	}
	if err := typeConsoleSteps(ctx, m, regions, c.Work, installPromptSteps(password), e); err != nil {
		return err
	}
	if err := completeMediaFreeBoot(ctx, m, f); err != nil {
		return err
	}
	return typeConsoleSteps(ctx, m, regions, c.Work, firstBootPromptSteps(password, pub), e)
}

func loadConsoleHostKey(evidencePath string, hostKey *string) (bool, error) {
	serial, err := os.ReadFile(filepath.Join(evidencePath, "boot-1.serial"))
	if err != nil {
		return false, err
	}
	*hostKey = consoleHostKey(serial)
	return *hostKey != "", nil
}

func waitConsoleHostKey(ctx context.Context, evidencePath string) (string, error) {
	var hostKey string
	keyWait, stop := context.WithTimeout(ctx, time.Minute)
	defer stop()
	err := poll(keyWait, func() (bool, error) {
		return loadConsoleHostKey(evidencePath, &hostKey)
	})
	return hostKey, err
}

func bindInstallSSH(ctx context.Context, m *machine, executable, known, hostKey string) error {
	// Bind SSH to the public key observed through this VM's private console,
	// not an unauthenticated network keyscan. Padding flushes the stream redactor.
	if err := build.WriteNew(known, []byte("[127.0.0.1]:32295 "+hostKey+"\n"), 0o600); err != nil {
		return err
	}
	ready, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := m.remote.WaitReady(ready); err != nil {
		return err
	}
	return m.driver(ctx, executable)
}

func installB(ctx context.Context, c Config, b Artifact, media image.Media, f *fixture) (commit string, err error) {
	password, err := writeInstallPassword(c.Work)
	if err != nil {
		return "", err
	}
	e, err := acceptance.CreateEvidence(filepath.Join(c.Work, "install-evidence"), [][]byte{[]byte(password)})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, e.Close()) }()
	key, known, disk, variables, pub, err := prepareInstallDisk(ctx, c, f)
	if err != nil {
		return "", err
	}
	m, err := newMachine(ctx, c, filepath.Join(c.Work, "b-vm"), disk, variables, filepath.Join("/run/soda-p9-input/candidate/media", media.ISO.Path), 32295, key, known, e)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, m.vm.Close()) }()
	if err = runInstallConsole(ctx, c, m, f, password, pub, e); err != nil {
		return "", err
	}
	hostKey, err := waitConsoleHostKey(ctx, e.Path())
	if err != nil {
		return "", err
	}
	if err = bindInstallSSH(ctx, m, c.Executable, known, hostKey); err != nil {
		return "", err
	}
	return m.identity(ctx, b, false)
}

func consoleHostKey(serial []byte) string {
	for _, line := range strings.Split(string(serial), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "ssh-ed25519" {
			continue
		}
		key := fields[0] + " " + fields[1]
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key)); err == nil {
			return key
		}
	}
	return ""
}

func (m *machine) recoveryConsole(ctx context.Context, work string, password []byte) error {
	var regions map[string]acceptance.ConsoleRegion
	if err := json.Unmarshal(consoleRegions, &regions); err != nil {
		return err
	}
	wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	for _, step := range [][2]string{{"login", "root\n"}, {"root-password", string(password) + "\n"}, {"root", ""}} {
		region := regions[step[0]]
		region.Y = -1
		if err := m.vm.Console().WaitConsole(wait, filepath.Join(work, "recovered-console.png"), region); err != nil {
			return err
		}
		if err := m.vm.Console().TypeConsole(wait, step[1]); err != nil {
			return err
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	marker := hex.EncodeToString(nonce[:])
	path := "/run/soda-console-" + marker
	if err := m.vm.Console().TypeConsole(wait, fmt.Sprintf("printf '%s' > %s\n", marker, path)); err != nil {
		return err
	}
	return poll(wait, func() (bool, error) {
		out, err := m.command(wait, "recovered-console-command", "cat "+path, nil)
		return err == nil && string(out) == marker, nil
	})
}
