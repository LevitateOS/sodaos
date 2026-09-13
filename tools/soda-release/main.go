// soda-release is a noninteractive release-worker tool, not an appliance update
// helper. No build code is executed by its signing/publication operations.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/releasedelivery"
)

func run() error {
	if len(os.Args) < 2 {
		return errors.New("operation required: prepare, channel, policy, init-state, init-ledger, sign, publish, fetch")
	}
	operation := os.Args[1]
	flags := flag.NewFlagSet(operation, flag.ContinueOnError)
	trustFile := flags.String("trust", "", "explicit public trust configuration")
	out := flags.String("out", "", "fresh absolute output directory (init operations: new private state file)")
	input := flags.String("input", "", "candidate directory, channel JSON, or local signing input")
	qualification := flags.String("qualification", "", "public qualification metadata; not signing permission")
	permitFile := flags.String("permit", "", "restricted protected-worker exact-digest permit")
	signerFile := flags.String("signer", "", "restricted JSON naming Key and Passphrase files")
	auth := flags.String("auth-file", "", "restricted destination registry auth file; never credentials in argv")
	ledger := flags.String("ledger", "", "existing persistent publication ledger")
	state := flags.String("state", "", "existing persistent verification high-water state")
	repo := flags.String("repository", "", "exact repository for init-ledger")
	transport := flags.String("transport", "oci", "local signing transport: oci, oci-archive, dir")
	name := flags.String("channel", "", "candidate, preview or stable")
	arch := flags.String("arch", "", "requested architecture; no sibling fallback")
	base := flags.String("base-policy", "", "public existing policy to preserve; otherwise emit standalone reject-default policy")
	observe := flags.Bool("observe", false, "observe an uncertain/completed publication, never replay registry writes")
	if e := flags.Parse(os.Args[2:]); e != nil {
		return e
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*out) {
		return errors.New("explicit absolute output and no positional arguments required")
	}
	allowed := map[string]string{
		"prepare": "input qualification", "channel": "input", "policy": "base-policy",
		"init-state": "", "init-ledger": "repository", "sign": "permit signer input transport",
		"publish": "permit input auth-file ledger observe", "fetch": "channel arch state",
	}
	options, ok := allowed[operation]
	if !ok {
		return errors.New("unsupported release operation")
	}
	var badFlag bool
	flags.Visit(func(f *flag.Flag) {
		if !strings.Contains(" trust out "+options+" ", " "+f.Name+" ") {
			badFlag = true
		}
	})
	if badFlag {
		return errors.New("flag does not apply to this operation")
	}
	var trust releasedelivery.Trust
	if releasedelivery.ReadJSON(*trustFile, &trust) != nil || trust.Validate() != nil {
		return errors.New("public release trust configuration refused")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	native := releasedelivery.Native{Home: *out}
	nativeCheck := func() error { return releasedelivery.CheckNative(ctx, native) }
	switch operation {
	case "prepare":
		var q releasedelivery.Qualification
		if e := releasedelivery.ReadJSON(*qualification, &q); e != nil {
			return e
		}
		digest, e := releasedelivery.Prepare(trust, *input, q, *out)
		if e != nil {
			return e
		}
		ref, e := releasedelivery.ReferenceForDocument(trust, "release", digest)
		if e != nil {
			return e
		}
		fmt.Println(ref)
	case "channel":
		var c releasedelivery.Channel
		if e := releasedelivery.ReadJSON(*input, &c); e != nil {
			return e
		}
		// Validate shape/freshness before emitting; publication additionally verifies
		// every release/artifact and compares the previous approved channel.
		if _, e := releasedelivery.AdmitChannel(trust, releasedelivery.EmptyState(), c, "sha256:0000000000000000000000000000000000000000000000000000000000000000", c.Name, time.Now()); e != nil {
			return e
		}
		digest, e := releasedelivery.WriteDocument(*out, c)
		if e != nil {
			return e
		}
		ref, e := releasedelivery.ReferenceForDocument(trust, c.Name, digest)
		if e != nil {
			return e
		}
		fmt.Println(ref)
	case "policy":
		data := []byte(`{"default":[{"type":"reject"}]}`)
		if *base != "" {
			var e error
			data, e = releasedelivery.ReadFile(*base, 1<<20)
			if e != nil {
				return e
			}
		}
		proposed, e := releasedelivery.MergePolicy(trust, data)
		if e != nil {
			return e
		}
		if e = nativebuild.FreshDirectory(*out); e != nil {
			return e
		}
		if e = nativebuild.WriteNew(filepath.Join(*out, "policy.json"), proposed, 0600); e != nil {
			return e
		}
		if e = releasedelivery.WriteRegistryConfig(*out, trust); e != nil {
			return e
		}
		fmt.Println("Proposed public policy only; no host trust changed:", *out)
	case "init-state":
		return releasedelivery.InitState(*out, trust)
	case "init-ledger":
		return releasedelivery.InitLedger(*out, trust, *repo)
	case "sign", "publish":
		if releasedelivery.PrivateFile(*permitFile) != nil {
			return errors.New("restricted protected-worker permit required")
		}
		var permit releasedelivery.Permit
		if releasedelivery.ReadJSON(*permitFile, &permit) != nil {
			return errors.New("protected-worker permit refused")
		}
		if e := nativeCheck(); e != nil {
			return e
		}
		if operation == "sign" {
			if releasedelivery.PrivateFile(*signerFile) != nil {
				return errors.New("restricted signer configuration required")
			}
			var keys releasedelivery.SecretFiles
			if releasedelivery.ReadJSON(*signerFile, &keys) != nil {
				return errors.New("restricted signer configuration refused")
			}
			if e := releasedelivery.Sign(ctx, native, trust, permit, *transport, *input, *out, keys); e != nil {
				return e
			}
		} else {
			if e := releasedelivery.Publish(ctx, native, trust, permit, *input, *auth, *ledger, *out, *observe); e != nil {
				return e
			}
		}
		fmt.Println("Receipt:", filepath.Join(*out, "receipt.json"))
	case "fetch":
		if e := nativeCheck(); e != nil {
			return e
		}
		if e := releasedelivery.Fetch(ctx, native, trust, *name, *arch, *state, *out, time.Now()); e != nil {
			return e
		}
		fmt.Println("Verified download only; no installation:", filepath.Join(*out, "verified.json"))
	default:
		return errors.New("unsupported release operation")
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
