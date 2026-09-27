//go:build linux

// soda-muse-compose explicitly registers one nested Compose service for Muse.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type options struct{ login, service, file string }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "soda-muse-compose:", err)
		os.Exit(1)
	}
}

func run() error {
	o, err := loadOptions()
	if err != nil {
		return err
	}
	actor, err := account(o.login)
	if err != nil {
		return err
	}
	root, registration, err := registrationRoot()
	if err != nil {
		return err
	}
	child, err := launchCompose(o, root)
	if err != nil {
		return err
	}
	return register(identity.NestedRegistration{ChildID: child, ActorID: actor, RegistrationID: registration})
}

func loadOptions() (options, error) {
	flags := flag.NewFlagSet("soda-muse-compose", flag.ContinueOnError)
	var o options
	flags.StringVar(&o.login, "login", "", "authorizing provisioned Soda account")
	flags.StringVar(&o.service, "service", "", "Compose service to opt in")
	flags.StringVar(&o.file, "file", "compose.yml", "Compose file")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return o, err
	}
	if !validOptions(o, flags.NArg()) {
		return o, errors.New("project root, provisioned login and one service required")
	}
	return o, nil
}

func validOptions(o options, remaining int) bool {
	return remaining == 0 && os.Geteuid() == 0 && o.login != "" && filepath.Base(o.login) == o.login && o.service != ""
}

func registrationRoot() (string, string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", "", err
	}
	registration := hex.EncodeToString(id[:])
	root := "/run/soda-muse/nested/" + registration
	if err := os.MkdirAll(filepath.Dir(root), 0o711); err != nil {
		return "", "", err
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(filepath.Dir(root), &fs); err != nil || fs.Type != 0x01021994 {
		return "", "", errors.New("nested registration requires runtime tmpfs")
	}
	if err := os.Mkdir(root, 0o711); err != nil {
		return "", "", err
	}
	return root, registration, nil
}

func launchCompose(o options, root string) (string, error) {
	override := filepath.Join(root, "compose.json")
	if err := writeOverride(override, o.service, root); err != nil {
		return "", err
	}
	args := []string{"-f", o.file, "-f", override}
	command := exec.Command("/usr/local/bin/podman-compose", append(args, "up", "-d", o.service)...)
	command.Stdout, command.Stderr, command.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := command.Run(); err != nil {
		return "", errors.New("compose did not confirm opted-in service creation")
	}
	out, err := exec.Command("/usr/local/bin/podman-compose", append(args, "ps", "-q")...).Output()
	if err != nil {
		return "", errors.New("compose container identification failed")
	}
	return selectComposeChild(string(out), o.service, func(id string) (string, error) {
		body, err := exec.Command("/usr/bin/podman", "--remote=false", "inspect", "--format", `{{.ID}} {{index .Config.Labels "io.podman.compose.service"}}`, id).Output()
		return string(body), err
	})
}

func account(login string) (int64, error) {
	path := "/var/lib/soda/accounts/" + login
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0o600 {
		return 0, errors.New("provisioned account marker required")
	}
	native, ok := st.Sys().(*syscall.Stat_t)
	if !ok || native.Uid != 0 {
		return 0, errors.New("root-owned account marker required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	actor, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || actor <= 0 {
		return 0, errors.New("invalid provisioned account")
	}
	return actor, nil
}

func writeOverride(path, service, root string) error {
	mounts := []string{
		"/usr/local/bin/muse:/usr/local/bin/muse:ro",
		"/usr/local/libexec/soda/muse:/usr/local/libexec/soda/muse:ro",
		filepath.Dir(identity.MuseLaunchSocket) + ":" + filepath.Dir(identity.MuseLaunchSocket) + ":ro",
		root + ":/run/soda-muse/credentials:ro",
	}
	wire := map[string]any{"services": map[string]any{service: map[string]any{"volumes": mounts}}}
	data, err := json.Marshal(wire)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func register(request identity.NestedRegistration) error {
	conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: identity.MuseLaunchSocket, Net: "unixpacket"})
	if err != nil {
		return errors.New("muse launch service unavailable")
	}
	defer func() { _ = conn.Close() }()
	if err = conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	data, err := json.Marshal(identity.LaunchRequest{Register: &request})
	if err != nil {
		return err
	}
	if _, err = conn.Write(data); err != nil {
		return err
	}
	body := make([]byte, 4096)
	n, err := conn.Read(body)
	if err != nil {
		return errors.New("muse registration unconfirmed")
	}
	var response identity.LaunchExit
	if err = json.Unmarshal(body[:n], &response); err != nil || response.Code != 0 || response.Error != "" {
		return errors.New("muse registration rejected")
	}
	return nil
}
