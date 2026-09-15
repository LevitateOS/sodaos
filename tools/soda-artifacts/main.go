// Native build/input support, deliberately not an appliance cmd/* binary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type artifactFlags struct {
	arch, revision, source, out, lock, keyring, signer string
}

func parseArtifactFlags(args []string) (string, artifactFlags, error) {
	if len(args) == 0 {
		return "", artifactFlags{}, errors.New("usage: soda-artifacts inspect-oci|seal|verify|verify-installed|bundle|fetch-coreos|fetch-coreos-iso|convert-butane [flags]")
	}
	action := args[0]
	f := flag.NewFlagSet(action, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	arch := f.String("arch", "", "matching native architecture")
	revision := f.String("revision", "", "full source revision")
	source := f.String("source", "", "stage/bundle or private Butane file")
	out := f.String("out", "", "new absolute output directory/file")
	lock := f.String("lock", "", "selected CoreOS lock")
	keyring := f.String("keyring", "", "already trusted Fedora keyring")
	signer := f.String("signer", "", "full independently trusted signer fingerprint")
	if err := f.Parse(args[1:]); err != nil || len(f.Args()) != 0 {
		return "", artifactFlags{}, errors.New("invalid artifact command flags")
	}
	return action, artifactFlags{*arch, *revision, *source, *out, *lock, *keyring, *signer}, nil
}

func inspectArtifactOCI(source, arch, revision string) error {
	image, err := build.InspectOCI(source, arch, revision)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(image)
}

func openPrivateButane(source string) (*os.File, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("private regular Butane input required")
	}
	return os.Open(source)
}

func convertButane(ctx context.Context, source, out, arch string) error {
	if err := build.RequireNative(arch); err != nil {
		return err
	}
	if err := build.PrivateDestination(out); err != nil {
		return err
	}
	if _, err := exec.LookPath("butane"); err != nil {
		return err
	}
	in, err := openPrivateButane(source)
	if err != nil {
		return err
	}
	defer in.Close()
	dest, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	phase, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(phase, "butane", "--strict")
	cmd.Stdin = in
	cmd.Stdout = dest
	cmd.WaitDelay = time.Second
	if err = errors.Join(cmd.Run(), dest.Close()); err != nil {
		return errors.New("strict Butane conversion failed; restricted output retained, not bootable evidence")
	}
	return nil
}

func runCoreOSArtifact(ctx context.Context, action string, f artifactFlags) error {
	switch action {
	case "fetch-coreos":
		_, err := build.FetchCoreOS(ctx, f.lock, f.arch, f.keyring, f.signer, f.out)
		return err
	case "fetch-coreos-iso":
		_, err := build.FetchCoreOSISO(ctx, f.lock, f.arch, f.keyring, f.signer, f.out)
		return err
	default:
		return errors.New("unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected")
	}
}

func runArtifactAction(ctx context.Context, action string, f artifactFlags) error {
	switch action {
	case "inspect-oci":
		return inspectArtifactOCI(f.source, f.arch, f.revision)
	case "seal":
		return build.Seal(f.source, f.arch, f.revision)
	case "verify":
		_, err := build.Verify(f.source, f.arch, f.revision)
		return err
	case "verify-installed":
		return build.VerifyInstalled(ctx, f.source, f.arch, f.revision)
	case "bundle":
		if err := build.RequireNative(f.arch); err != nil {
			return err
		}
		return build.Bundle(f.source, f.out, f.arch, f.revision)
	case "convert-butane":
		return convertButane(ctx, f.source, f.out, f.arch)
	default:
		return runCoreOSArtifact(ctx, action, f)
	}
}

func run(ctx context.Context, args []string) error {
	action, flags, err := parseArtifactFlags(args)
	if err != nil {
		return err
	}
	return runArtifactAction(ctx, action, flags)
}
