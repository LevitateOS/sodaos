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

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: soda-artifacts inspect-oci|seal|verify|verify-installed|bundle|fetch-coreos|convert-butane [flags]")
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
		return errors.New("invalid artifact command flags")
	}
	switch action {
	case "inspect-oci":
		image, err := nativebuild.InspectOCI(*source, *arch, *revision)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(image)
	case "seal":
		return nativebuild.Seal(*source, *arch, *revision)
	case "verify":
		_, err := nativebuild.Verify(*source, *arch, *revision)
		return err
	case "verify-installed":
		return nativebuild.VerifyInstalled(ctx, *source, *arch, *revision)
	case "bundle":
		if err := nativebuild.RequireNative(*arch); err != nil {
			return err
		}
		return nativebuild.Bundle(*source, *out, *arch, *revision)
	case "fetch-coreos":
		_, err := nativebuild.FetchCoreOS(ctx, *lock, *arch, *keyring, *signer, *out)
		return err
	case "convert-butane":
		if err := nativebuild.RequireNative(*arch); err != nil {
			return err
		}
		if err := nativebuild.PrivateDestination(*out); err != nil {
			return err
		}
		if _, err := exec.LookPath("butane"); err != nil {
			return err
		}
		info, err := os.Lstat(*source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return errors.New("private regular Butane input required")
		}
		in, err := os.Open(*source)
		if err != nil {
			return err
		}
		defer in.Close()
		dest, err := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		phase, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		cmd := exec.CommandContext(phase, "butane", "--strict")
		cmd.Stdin = in
		cmd.Stdout = dest
		cmd.WaitDelay = time.Second
		// Butane diagnostics may echo private inline values; do not retain stderr.
		if err = errors.Join(cmd.Run(), dest.Close()); err != nil {
			return errors.New("strict Butane conversion failed; restricted output retained, not bootable evidence")
		}
		return nil
	default:
		return errors.New("unknown artifact action; ISO/QCOW2 media delivery is not selected")
	}
}
