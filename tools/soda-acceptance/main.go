// Outside support only: invokes existing owned checks; no product scenarios.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
)

type paths []string

func (p *paths) String() string     { return "input paths" }
func (p *paths) Set(s string) error { *p = append(*p, s); return nil }
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var targetPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type runOptions struct {
	action        string
	owner         string
	revision      string
	arch          string
	target        string
	evidence      string
	remoteFile    string
	request       string
	config        string
	source        string
	destination   string
	hold          bool
	restart       bool
	timeout       time.Duration
	secretFiles   paths
	artifactFiles paths
	cmdArgs       []string
}

func validateCommonOptions(opts runOptions) error {
	if !build.Revision(opts.revision) || !targetPattern.MatchString(opts.target) || opts.timeout <= 0 || opts.timeout > 24*time.Hour {
		return errors.New("revision, non-secret target and bounded timeout required")
	}
	if _, err := build.OCIArchitecture(opts.arch); err != nil {
		return err
	}
	if !acceptance.ValidOwner(opts.owner) {
		return errors.New("explicit owner required; P07/P08 are not independent tasks")
	}
	return nil
}

func validateVMFlags(action string, hold, restart bool) error {
	if action != "vm" && (hold || restart) {
		return errors.New("hold/restart are explicit fresh-VM actions only")
	}
	return nil
}

func validateExecCommand(action string, cmdArgs []string) error {
	if action == "exec" && len(cmdArgs) == 0 {
		return errors.New("an existing owned check command is required")
	}
	if action != "exec" && len(cmdArgs) != 0 {
		return errors.New("unexpected check command")
	}
	return nil
}

func validateNativeOpts(owner, request, remote string) error {
	if owner != "P02" || request == "" || remote == "" {
		return errors.New("native requires P02, remote and request")
	}
	return nil
}

func validateTransferOpts(owner, source, dest, remote string) error {
	if owner != "P05" || source == "" || dest == "" || remote == "" {
		return errors.New("transfer requires P05, pinned remote, bundle and destination")
	}
	return nil
}

func validateProbeSSHOpts(owner, remote string) error {
	if owner != "P11" || remote == "" {
		return errors.New("probe-ssh requires P11 and a pinned Git endpoint")
	}
	return nil
}

func validateVMOpts(owner, config, remote string) error {
	if owner != "P03" || config == "" || remote != "" {
		return errors.New("vm requires P03 and config")
	}
	return nil
}

func validateRemoteAction(opts runOptions) error {
	switch opts.action {
	case "native":
		return validateNativeOpts(opts.owner, opts.request, opts.remoteFile)
	case "transfer":
		return validateTransferOpts(opts.owner, opts.source, opts.destination, opts.remoteFile)
	case "probe-ssh":
		return validateProbeSSHOpts(opts.owner, opts.remoteFile)
	case "vm":
		return validateVMOpts(opts.owner, opts.config, opts.remoteFile)
	default:
		return nil
	}
}

func validateActionOptions(opts runOptions) error {
	if err := validateExecCommand(opts.action, opts.cmdArgs); err != nil {
		return err
	}
	if err := validateVMFlags(opts.action, opts.hold, opts.restart); err != nil {
		return err
	}
	return validateRemoteAction(opts)
}

func parseOptionFlags(action string, args []string) (runOptions, error) {
	f := flag.NewFlagSet(action, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	owner := f.String("owner", "", "P02/P03/P04/P05/P06/P11 or core U01-U20 owner")
	revision := f.String("revision", "", "requested full source revision")
	arch := f.String("arch", "", "requested appliance architecture")
	target := f.String("target", "", "explicit non-secret target name")
	evidence := f.String("evidence", "", "new absolute private evidence directory")
	remoteFile := f.String("remote", "", "pinned SSH connection JSON")
	request := f.String("request", "", "restricted exact-source native request JSON")
	config := f.String("config", "", "VM configuration JSON")
	source := f.String("bundle", "", "verified source bundle directory")
	destination := f.String("destination", "", "new absolute remote ARCH directory")
	hold := f.Bool("hold", false, "keep fresh VM running for separately invoked core checks")
	restart := f.Bool("restart", false, "explicitly restart this new VM once, retaining disk/NVRAM")
	timeout := f.Duration("timeout", 30*time.Minute, "bounded phase duration")
	var secretFiles, artifactFiles paths
	f.Var(&secretFiles, "secret-file", "restricted secret to redact (repeatable)")
	f.Var(&artifactFiles, "artifact-file", "public artifact/manifest whose bytes are relevant (never credentials)")
	if err := f.Parse(args); err != nil {
		return runOptions{}, errors.New("invalid support command flags")
	}
	return runOptions{
		action:        action,
		owner:         *owner,
		revision:      *revision,
		arch:          *arch,
		target:        *target,
		evidence:      *evidence,
		remoteFile:    *remoteFile,
		request:       *request,
		config:        *config,
		source:        *source,
		destination:   *destination,
		hold:          *hold,
		restart:       *restart,
		timeout:       *timeout,
		secretFiles:   secretFiles,
		artifactFiles: artifactFiles,
		cmdArgs:       f.Args(),
	}, nil
}

func parseRunOptions(args []string) (runOptions, error) {
	action := args[0]
	switch action {
	case "exec", "native", "transfer", "vm", "probe-ssh":
	default:
		return runOptions{}, errors.New("no product/media/release workflow is implemented by this support tool")
	}
	opts, err := parseOptionFlags(action, args[1:])
	if err != nil {
		return runOptions{}, err
	}
	if err := validateCommonOptions(opts); err != nil {
		return runOptions{}, err
	}
	if err := validateActionOptions(opts); err != nil {
		return runOptions{}, err
	}
	return opts, nil
}

func readSecretFiles(files []string) ([][]byte, error) {
	var secrets [][]byte
	for _, file := range files {
		b, err := acceptance.PrivateFile(file)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, errors.New("empty secret input")
		}
		secrets = append(secrets, b, []byte(strings.TrimRight(string(b), "\r\n")))
	}
	return secrets, nil
}

func loadVMSecrets(configPath, arch, target string) (acceptance.VMConfig, [][]byte, error) {
	var vmConfig acceptance.VMConfig
	if err := build.ReadJSON(configPath, &vmConfig); err != nil {
		return vmConfig, nil, err
	}
	if vmConfig.Architecture != arch || vmConfig.Name != target {
		return vmConfig, nil, errors.New("VM target/platform mismatch")
	}
	private, err := acceptance.ProvisioningSecrets(vmConfig.Ignition)
	if err != nil {
		return vmConfig, nil, err
	}
	return vmConfig, private, nil
}

func collectAllSecrets(opts runOptions) ([][]byte, acceptance.VMConfig, error) {
	secrets, err := readSecretFiles(opts.secretFiles)
	if err != nil {
		return nil, acceptance.VMConfig{}, err
	}
	var vmConfig acceptance.VMConfig
	if opts.action == "vm" {
		var vmSecrets [][]byte
		vmConfig, vmSecrets, err = loadVMSecrets(opts.config, opts.arch, opts.target)
		if err != nil {
			return nil, vmConfig, err
		}
		secrets = append(secrets, vmSecrets...)
	}
	return secrets, vmConfig, nil
}

func initObservation(opts runOptions) (acceptance.Observation, acceptance.Remote, error) {
	o := acceptance.Observation{
		Owner:                 opts.owner,
		RequestedRevision:     opts.revision,
		RequestedArchitecture: opts.arch,
		Target:                opts.target,
		ClientPlatform:        runtime.GOOS + "/" + runtime.GOARCH,
		Action:                opts.action,
		Started:               time.Now().UTC(),
		Execution:             "not-started",
		Cleanup:               "not-applicable",
		Topology:              "local command",
		Artifacts:             map[string]string{},
	}
	for _, file := range opts.artifactFiles {
		sum, err := build.HashFile(file)
		if err != nil {
			return o, acceptance.Remote{}, err
		}
		o.Artifacts[file] = sum
	}
	var remote acceptance.Remote
	if opts.remoteFile != "" {
		if err := build.ReadJSON(opts.remoteFile, &remote); err != nil {
			return o, acceptance.Remote{}, err
		}
		remote.Timeout = opts.timeout
		o.Topology = fmt.Sprintf("pinned SSH %s@%s:%d; management transport, not a project client route", remote.User, remote.Host, remote.Port)
		o.Cleanup = "remote phase bounded by timeout + 10s; interrupted transport is not remote cleanup proof"
	}
	return o, remote, nil
}

func executeExecOrNative(phase context.Context, e *acceptance.Evidence, action string, args []string, remoteFile, request, revision, arch, target string, remote *acceptance.Remote, o *acceptance.Observation) (error, error) {
	var c acceptance.Command
	var opErr error
	if action == "exec" {
		o.Invocation = args
		c = acceptance.Command{Name: args[0], Args: args[1:], Stdin: os.Stdin}
		if remoteFile != "" {
			c, opErr = remote.Command(args, os.Stdin)
		}
	} else {
		c, opErr = remote.NativePhase(request, revision, arch, target)
		if opErr == nil {
			var req acceptance.RemoteRequest
			opErr = build.ReadJSON(request, &req)
			o.Invocation = []string{"embedded native executor", req.Phase, req.Revision, req.Architecture, req.Target, req.Work}
			o.Action = "native-" + req.Phase
		}
	}
	if opErr != nil {
		return opErr, nil
	}
	r, werr := acceptance.Execute(phase, e, "check", c)
	o.ExitCode = r.ExitCode
	if r.Started {
		o.Execution = "completed"
		if r.Err != nil {
			o.Execution = "failed"
		}
	}
	return r.Err, werr
}

func executeProbeSSH(phase context.Context, e *acceptance.Evidence, remote acceptance.Remote, o *acceptance.Observation) (error, error) {
	o.Topology = fmt.Sprintf("direct client TCP to %s:%d; no proxy, forwarding inference or Git authentication", remote.Host, remote.Port)
	o.Invocation = []string{"SSH key exchange only", remote.Host, fmt.Sprint(remote.Port), remote.KnownHosts}
	fingerprint, err := remote.ProbeSSHKey(phase)
	o.Execution = "failed"
	o.Cleanup = "connection closed"
	if err != nil {
		return err, nil
	}
	o.Execution = "completed"
	werr := e.Write("ssh-key.txt", []byte(fingerprint+"\nPinned SSH endpoint observed, not Git authentication or project routing proof.\n"))
	return nil, werr
}

func executeTransfer(phase context.Context, e *acceptance.Evidence, remote acceptance.Remote, source, destination, arch, revision, target string, o *acceptance.Observation) (error, error) {
	o.Invocation = []string{"transfer sealed bundle", source, destination}
	r, werr := remote.TransferBundle(phase, e, source, destination, arch, revision, target)
	o.ExitCode = r.ExitCode
	for name, hash := range r.Artifacts {
		o.Artifacts[name] = hash
	}
	if r.Started {
		o.Execution = "completed"
		if r.Err != nil {
			o.Execution = "failed"
		}
	}
	return r.Err, werr
}

func runVMPhase(phase context.Context, e *acceptance.Evidence, vm *acceptance.VM, vmConfig acceptance.VMConfig, restart, hold bool, o *acceptance.Observation) (error, error) {
	o.Execution = "completed"
	evidenceErr := e.Write("boot-ready.txt", []byte("Fresh guest reached pinned SSH readiness. This is fixture evidence, not a product check.\n"))
	var operationErr error
	if restart {
		operationErr = vm.Restart(phase)
		if operationErr == nil {
			evidenceErr = errors.Join(evidenceErr, e.Write("restart-ready.txt", []byte("Same owned disk/NVRAM reached pinned SSH after restart. No project persistence assertion was performed.\n")))
		}
	}
	if operationErr == nil && hold {
		fmt.Fprintln(os.Stdout, "Fresh guest ready. Invoke owned checks separately. Interrupt shuts down and retains disks; cancellation is not a pass.")
		operationErr = vm.Wait(phase)
	}
	if operationErr != nil {
		o.Execution = "failed"
	}
	cleanupErr := vm.Close()
	o.Cleanup = "completed; private disk/NVRAM retained at " + vmConfig.Work
	if cleanupErr != nil {
		o.Cleanup = "failed; private work retained"
	}
	return errors.Join(operationErr, cleanupErr), evidenceErr
}

func executeVM(phase context.Context, e *acceptance.Evidence, vmConfig acceptance.VMConfig, restart, hold bool, o *acceptance.Observation) (error, error) {
	o.Topology = "matching-native KVM; loopback management SSH only, no routed-client claim"
	o.Invocation = []string{"fresh VM", vmConfig.Name, vmConfig.Work, vmConfig.BaseReceipt, "private single fw_cfg Ignition (not retained)"}
	if restart {
		o.Invocation = append(o.Invocation, "--restart")
	}
	if hold {
		o.Invocation = append(o.Invocation, "--hold")
	}
	o.Cleanup = "not-reached or failed launch; work retained"
	vm, opErr := acceptance.LaunchVM(phase, vmConfig, e)
	if opErr != nil {
		if vm != nil {
			o.Execution = "failed"
			o.Cleanup = "completed after failed launch; private work retained"
			if vm.Close() != nil {
				o.Cleanup = "failed after failed launch; private work retained"
			}
		}
		return opErr, nil
	}
	return runVMPhase(phase, e, vm, vmConfig, restart, hold, o)
}

func executeAction(phase context.Context, e *acceptance.Evidence, opts runOptions, vmConfig acceptance.VMConfig, remote acceptance.Remote, o *acceptance.Observation) (error, error) {
	switch opts.action {
	case "exec", "native":
		return executeExecOrNative(phase, e, opts.action, opts.cmdArgs, opts.remoteFile, opts.request, opts.revision, opts.arch, opts.target, &remote, o)
	case "probe-ssh":
		return executeProbeSSH(phase, e, remote, o)
	case "transfer":
		return executeTransfer(phase, e, remote, opts.source, opts.destination, opts.arch, opts.revision, opts.target, o)
	case "vm":
		return executeVM(phase, e, vmConfig, opts.restart, opts.hold, o)
	default:
		return nil, nil
	}
}

func readBuildSettings(o *acceptance.Observation) {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				o.ToolRevision = s.Value
			case "vcs.modified":
				o.ToolDirty = s.Value == "true"
			}
		}
	}
	if o.ToolRevision == "" {
		o.ToolRevision = "unknown"
	}
}

func redactObservationStrings(e *acceptance.Evidence, o *acceptance.Observation) {
	for i, arg := range o.Invocation {
		o.Invocation[i] = e.RedactString(arg)
	}
	o.Topology = e.RedactString(o.Topology)
	o.Cleanup = e.RedactString(o.Cleanup)
	safeArtifacts := map[string]string{}
	for name, sum := range o.Artifacts {
		safeArtifacts[e.RedactString(name)] = sum
	}
	o.Artifacts = safeArtifacts
}

func finalizeObservation(e *acceptance.Evidence, o acceptance.Observation, operationErr, evidenceErr error) error {
	readBuildSettings(&o)
	evidenceErr = errors.Join(evidenceErr, e.CheckSecrets())
	allErr := errors.Join(operationErr, evidenceErr)
	if allErr != nil {
		evidenceErr = errors.Join(evidenceErr, e.Write("failure.txt", []byte(e.RedactError(allErr).Error()+"\n")))
	}
	hashes, err := e.Hashes()
	o.Files = hashes
	evidenceErr = errors.Join(evidenceErr, err)
	o.Evidence = "completed"
	if evidenceErr != nil {
		o.Evidence = "failed"
	}
	o.Outcome = "completed"
	if operationErr != nil || evidenceErr != nil {
		o.Outcome = "failed"
	}
	if errors.Is(operationErr, context.Canceled) {
		o.Outcome = "cancelled"
	}
	o.Finished = time.Now().UTC()
	redactObservationStrings(e, &o)
	// No success-shaped final record exists until retention has finalized.
	merr := e.PublishObservation(o)
	return e.RedactError(errors.Join(operationErr, evidenceErr, merr))
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: soda-acceptance exec|native|transfer|vm|probe-ssh|report [flags]")
	}
	action := args[0]
	if action == "report" {
		return report(args[1:])
	}
	opts, err := parseRunOptions(args)
	if err != nil {
		return err
	}
	secrets, vmConfig, err := collectAllSecrets(opts)
	if err != nil {
		return err
	}
	e, err := acceptance.CreateEvidence(opts.evidence, secrets)
	if err != nil {
		return err
	}
	defer e.Close()
	phase, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	o, remote, opErr := initObservation(opts)
	var evErr error
	if opErr == nil {
		opErr, evErr = executeAction(phase, e, opts, vmConfig, remote, &o)
	}
	return finalizeObservation(e, o, opErr, evErr)
}

func report(args []string) error {
	f := flag.NewFlagSet("report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	arch := f.String("arch", "", "candidate architecture")
	revision := f.String("revision", "", "full revision")
	out := f.String("out", "", "new private Markdown output")
	var records paths
	f.Var(&records, "record", "existing observation.json (repeatable)")
	if err := f.Parse(args); err != nil || len(f.Args()) != 0 {
		return errors.New("invalid report flags")
	}
	return acceptance.Handoff(*out, *arch, *revision, records)
}
