package nativequalification

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

// Reviewed against the unchanged native console contract, not learned from the
// candidate under test. Screenshots and secret input remain private.
//
//go:embed console-regions.json
var consoleRegions []byte

func installB(ctx context.Context, c Config, b Artifact, media hostimage.Media, f *fixture) (commit string, err error) {
	var random [24]byte
	if _, err = rand.Read(random[:]); err != nil {
		return "", err
	}
	password := hex.EncodeToString(random[:])
	if err = nativebuild.WriteNew(filepath.Join(c.Work, "b-password"), []byte(password+"\n"), 0600); err != nil {
		return "", err
	}
	e, err := acceptance.CreateEvidence(filepath.Join(c.Work, "install-evidence"), [][]byte{[]byte(password)})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, e.Close()) }()
	key := filepath.Join(c.Work, "b-key")
	known := filepath.Join(c.Work, "b-known-hosts")
	if _, err = f.run(ctx, "ssh-key", "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "soda-p9-install", "-f", key); err != nil {
		return "", err
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		return "", err
	}
	disk := filepath.Join(c.Work, "b.qcow2")
	variables := filepath.Join(c.Work, "b-vars.fd")
	if _, err = f.run(ctx, "blank-install-disk", "qemu-img", "create", "-f", "qcow2", disk, "64G"); err != nil {
		return "", err
	}
	if err = exec.CommandContext(ctx, "cp", "--", c.Variables, variables).Run(); err != nil {
		return "", err
	}
	m, err := newMachine(ctx, c, filepath.Join(c.Work, "b-vm"), disk, variables, filepath.Join("/run/soda-p9-input/candidate/media", media.ISO.Path), 32295, key, known, e)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, m.vm.Close()) }()
	var regions map[string]acceptance.ConsoleRegion
	if err = json.Unmarshal(consoleRegions, &regions); err != nil {
		return "", err
	}
	prompt := func(name, text string) error {
		region, ok := regions[name]
		if !ok {
			return errors.New("unreviewed console prompt")
		}
		region.Y = -1
		wait, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		if err := m.vm.Console().WaitConsole(wait, filepath.Join(c.Work, "b-"+name+".png"), region); err != nil {
			return err
		}
		if name == "erase" {
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
			if err := e.WriteJSON("before-erasure.json", blocks); err != nil {
				return err
			}
		}
		return m.vm.Console().TypeConsole(ctx, text)
	}
	for _, step := range [][2]string{{"welcome", "\n"}, {"network", "keep\n"}, {"network-confirm", "yes\n"}, {"disk", "1\n"}, {"hostname", "soda-tester\n"}, {"password", password + "\n"}, {"password-confirm", password + "\n"}, {"subnet", "\n"}, {"erase", "ERASE /dev/vda\n"}, {"installed", ""}} {
		if err = prompt(step[0], step[1]); err != nil {
			return "", err
		}
	}
	if err = m.vm.EjectInstallationMedia(ctx); err != nil {
		return "", err
	}
	if err = f.rootfs.Close(); err != nil {
		return "", err
	}
	f.rootfs = nil
	if err = m.vm.Console().Execute(ctx, "system_reset", "media-free-first-boot", nil, nil); err != nil {
		return "", err
	}
	for _, step := range [][2]string{{"login", "root\n"}, {"root-password", password + "\n"}, {"root", "umask 077; mkdir -p /root/.ssh; echo '" + strings.TrimSpace(string(pub)) + "' > /root/.ssh/authorized_keys; chmod 600 /root/.ssh/authorized_keys; restorecon -RF /root/.ssh; clear\n"}} {
		if err = prompt(step[0], step[1]); err != nil {
			return "", err
		}
	}
	scan, err := exec.CommandContext(ctx, "ssh-keyscan", "-T", "15", "-p", "32295", "-t", "ed25519", "127.0.0.1").Output()
	if err != nil {
		return "", err
	}
	if err = nativebuild.WriteNew(known, scan, 0600); err != nil {
		return "", err
	}
	ready, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err = m.remote.WaitReady(ready); err != nil {
		return "", err
	}
	if err = m.driver(ctx, c.Executable); err != nil {
		return "", err
	}
	return m.identity(ctx, b, false)
}
