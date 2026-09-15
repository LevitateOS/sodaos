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

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

type releaseFlags struct {
	operation     string
	trustFile     string
	out           string
	input         string
	media         string
	qualification string
	permitFile    string
	signerFile    string
	auth          string
	ledger        string
	state         string
	repo          string
	transport     string
	name          string
	arch          string
	base          string
	observe       bool
}

var allowedOptions = map[string]string{
	"prepare":     "input media qualification",
	"channel":     "input",
	"policy":      "base-policy",
	"init-state":  "",
	"init-ledger": "repository",
	"sign":        "permit signer input transport",
	"publish":     "permit input auth-file ledger observe",
	"fetch":       "channel arch state",
}

func validateOperationFlags(flags *flag.FlagSet, operation, out string) error {
	if flags.NArg() != 0 || !filepath.IsAbs(out) {
		return errors.New("explicit absolute output and no positional arguments required")
	}
	options, ok := allowedOptions[operation]
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
	return nil
}

func loadReleaseTrust(path string) (deliver.Trust, error) {
	var trust deliver.Trust
	if deliver.ReadJSON(path, &trust) != nil || trust.Validate() != nil {
		return trust, errors.New("public release trust configuration refused")
	}
	return trust, nil
}

func parseReleaseArgs(args []string) (releaseFlags, deliver.Trust, error) {
	if len(args) < 2 {
		return releaseFlags{}, deliver.Trust{}, errors.New("operation required: prepare, channel, policy, init-state, init-ledger, sign, publish, fetch")
	}
	op := args[1]
	flags := flag.NewFlagSet(op, flag.ContinueOnError)
	trustFile := flags.String("trust", "", "explicit public trust configuration")
	out := flags.String("out", "", "fresh absolute output directory (init operations: new private state file)")
	input := flags.String("input", "", "candidate directory, channel JSON, or local signing input")
	media := flags.String("media", "", "exact media.json binding ISO/rootfs hash/size/location")
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
	if err := flags.Parse(args[2:]); err != nil {
		return releaseFlags{}, deliver.Trust{}, err
	}
	if err := validateOperationFlags(flags, op, *out); err != nil {
		return releaseFlags{}, deliver.Trust{}, err
	}
	trust, err := loadReleaseTrust(*trustFile)
	if err != nil {
		return releaseFlags{}, trust, err
	}
	return releaseFlags{
		operation:     op,
		trustFile:     *trustFile,
		out:           *out,
		input:         *input,
		media:         *media,
		qualification: *qualification,
		permitFile:    *permitFile,
		signerFile:    *signerFile,
		auth:          *auth,
		ledger:        *ledger,
		state:         *state,
		repo:          *repo,
		transport:     *transport,
		name:          *name,
		arch:          *arch,
		base:          *base,
		observe:       *observe,
	}, trust, nil
}

func runPrepare(trust deliver.Trust, qualificationFile, input, media, out string) error {
	var q deliver.Qualification
	if err := deliver.ReadJSON(qualificationFile, &q); err != nil {
		return err
	}
	digest, err := deliver.Prepare(trust, input, media, q, out)
	if err != nil {
		return err
	}
	ref, err := deliver.ReferenceForDocument(trust, "release", digest)
	if err != nil {
		return err
	}
	fmt.Println(ref)
	return nil
}

func runChannel(trust deliver.Trust, inputFile, out string) error {
	var c deliver.Channel
	if err := deliver.ReadJSON(inputFile, &c); err != nil {
		return err
	}
	// Validate shape/freshness before emitting; publication additionally verifies
	// every release/artifact and compares the previous approved channel.
	if _, err := deliver.AdmitChannel(trust, deliver.EmptyState(), c, "sha256:0000000000000000000000000000000000000000000000000000000000000000", c.Name, time.Now()); err != nil {
		return err
	}
	digest, err := deliver.WriteDocument(out, c)
	if err != nil {
		return err
	}
	ref, err := deliver.ReferenceForDocument(trust, c.Name, digest)
	if err != nil {
		return err
	}
	fmt.Println(ref)
	return nil
}

func runPolicy(trust deliver.Trust, baseFile, out string) error {
	data := []byte(`{"default":[{"type":"reject"}]}`)
	if baseFile != "" {
		var err error
		data, err = deliver.ReadFile(baseFile, 1<<20)
		if err != nil {
			return err
		}
	}
	proposed, err := deliver.MergePolicy(trust, data)
	if err != nil {
		return err
	}
	if err = build.FreshDirectory(out); err != nil {
		return err
	}
	if err = build.WriteNew(filepath.Join(out, "policy.json"), proposed, 0o600); err != nil {
		return err
	}
	if err = deliver.WriteRegistryConfig(out, trust); err != nil {
		return err
	}
	fmt.Println("Proposed public policy only; no host trust changed:", out)
	return nil
}

func loadPermit(permitFile string) (deliver.Permit, error) {
	if deliver.PrivateFile(permitFile) != nil {
		return deliver.Permit{}, errors.New("restricted protected-worker permit required")
	}
	var permit deliver.Permit
	if deliver.ReadJSON(permitFile, &permit) != nil {
		return deliver.Permit{}, errors.New("protected-worker permit refused")
	}
	return permit, nil
}

func runSign(ctx context.Context, native deliver.Native, trust deliver.Trust, permit deliver.Permit, signerFile, transport, input, out string) error {
	if deliver.PrivateFile(signerFile) != nil {
		return errors.New("restricted signer configuration required")
	}
	var keys deliver.SecretFiles
	if deliver.ReadJSON(signerFile, &keys) != nil {
		return errors.New("restricted signer configuration refused")
	}
	if err := deliver.Sign(ctx, native, trust, permit, transport, input, out, keys); err != nil {
		return err
	}
	fmt.Println("Receipt:", filepath.Join(out, "receipt.json"))
	return nil
}

func runPublish(ctx context.Context, native deliver.Native, trust deliver.Trust, permit deliver.Permit, input, auth, ledger, out string, observe bool) error {
	if err := deliver.Publish(ctx, native, trust, permit, input, auth, ledger, out, observe); err != nil {
		return err
	}
	fmt.Println("Receipt:", filepath.Join(out, "receipt.json"))
	return nil
}

func runFetch(ctx context.Context, native deliver.Native, trust deliver.Trust, name, arch, state, out string) error {
	if err := deliver.CheckNative(ctx, native); err != nil {
		return err
	}
	if err := deliver.Fetch(ctx, native, trust, name, arch, state, out, time.Now()); err != nil {
		return err
	}
	fmt.Println("Verified download only; no installation:", filepath.Join(out, "verified.json"))
	return nil
}

func executeSignOrPublish(ctx context.Context, native deliver.Native, trust deliver.Trust, rf releaseFlags) error {
	permit, err := loadPermit(rf.permitFile)
	if err != nil {
		return err
	}
	if err = deliver.CheckNative(ctx, native); err != nil {
		return err
	}
	if rf.operation == "sign" {
		return runSign(ctx, native, trust, permit, rf.signerFile, rf.transport, rf.input, rf.out)
	}
	return runPublish(ctx, native, trust, permit, rf.input, rf.auth, rf.ledger, rf.out, rf.observe)
}

func executeReleaseOperation(ctx context.Context, rf releaseFlags, trust deliver.Trust) error {
	native := deliver.Native{Home: rf.out}
	switch rf.operation {
	case "prepare":
		return runPrepare(trust, rf.qualification, rf.input, rf.media, rf.out)
	case "channel":
		return runChannel(trust, rf.input, rf.out)
	case "policy":
		return runPolicy(trust, rf.base, rf.out)
	case "init-state":
		return deliver.InitState(rf.out, trust)
	case "init-ledger":
		return deliver.InitLedger(rf.out, trust, rf.repo)
	case "sign", "publish":
		return executeSignOrPublish(ctx, native, trust, rf)
	case "fetch":
		return runFetch(ctx, native, trust, rf.name, rf.arch, rf.state, rf.out)
	default:
		return errors.New("unsupported release operation")
	}
}

func run() error {
	rf, trust, err := parseReleaseArgs(os.Args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	return executeReleaseOperation(ctx, rf, trust)
}

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
