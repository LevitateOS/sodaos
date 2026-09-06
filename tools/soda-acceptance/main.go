// Outside support only: invokes existing owned checks; no product scenarios.
package main

import (
	"context"
	"encoding/json"
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
	"github.com/levitateos/sodaos/internal/nativebuild"
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
func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: soda-acceptance exec|native|transfer|vm|probe-ssh|report [flags]")
	}
	action := args[0]
	if action == "report" {
		return report(args[1:])
	}
	switch action {
	case "exec", "native", "transfer", "vm", "probe-ssh":
	default:
		return errors.New("no product/media/release workflow is implemented by this support tool")
	}
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
	if err := f.Parse(args[1:]); err != nil {
		return errors.New("invalid support command flags")
	}
	if !nativebuild.Revision(*revision) || !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`).MatchString(*target) || *timeout <= 0 || *timeout > 24*time.Hour {
		return errors.New("revision, non-secret target and bounded timeout required")
	}
	if _, err := nativebuild.OCIArchitecture(*arch); err != nil {
		return err
	}
	if !acceptance.ValidOwner(*owner) {
		return errors.New("explicit owner required; P07/P08 are not independent tasks")
	}
	if action == "exec" && len(f.Args()) == 0 {
		return errors.New("an existing owned check command is required")
	}
	if action != "exec" && len(f.Args()) != 0 {
		return errors.New("unexpected check command")
	}
	if action == "native" && (*owner != "P02" || *request == "" || *remoteFile == "") {
		return errors.New("native requires P02, remote and request")
	}
	if action == "transfer" && (*owner != "P05" || *source == "" || *destination == "" || *remoteFile == "") {
		return errors.New("transfer requires P05, pinned remote, bundle and destination")
	}
	if action == "probe-ssh" && (*owner != "P11" || *remoteFile == "") {
		return errors.New("probe-ssh requires P11 and a pinned Git endpoint")
	}
	if action == "vm" && (*owner != "P03" || *config == "" || *remoteFile != "") {
		return errors.New("vm requires P03 and config")
	}
	if action != "vm" && (*hold || *restart) {
		return errors.New("hold/restart are explicit fresh-VM actions only")
	}
	var secrets [][]byte
	for _, file := range secretFiles {
		b, err := acceptance.PrivateFile(file)
		if err != nil {
			return err
		}
		if len(b) == 0 {
			return errors.New("empty secret input")
		}
		secrets = append(secrets, b, []byte(strings.TrimRight(string(b), "\r\n")))
	}
	var vmConfig acceptance.VMConfig
	if action == "vm" {
		if err := nativebuild.ReadJSON(*config, &vmConfig); err != nil {
			return err
		}
		if vmConfig.Architecture != *arch || vmConfig.Name != *target {
			return errors.New("VM target/platform mismatch")
		}
		private, err := acceptance.ProvisioningSecrets(vmConfig.Ignition)
		if err != nil {
			return err
		}
		secrets = append(secrets, private...)
	}
	e, err := acceptance.CreateEvidence(*evidence, secrets)
	if err != nil {
		return err
	}
	defer e.Close()
	phase, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	o := acceptance.Observation{Owner: *owner, RequestedRevision: *revision, RequestedArchitecture: *arch, Target: *target, ClientPlatform: runtime.GOOS + "/" + runtime.GOARCH, Action: action, Started: time.Now().UTC(), Execution: "not-started", Cleanup: "not-applicable", Topology: "local command", Artifacts: map[string]string{}}
	var operationErr, evidenceErr error
	for _, file := range artifactFiles {
		sum, err := nativebuild.HashFile(file)
		if err != nil {
			operationErr = err
			break
		}
		o.Artifacts[file] = sum
	}
	var remote acceptance.Remote
	if *remoteFile != "" && operationErr == nil {
		operationErr = nativebuild.ReadJSON(*remoteFile, &remote)
		remote.Timeout = *timeout
		o.Topology = fmt.Sprintf("pinned SSH %s@%s:%d; management transport, not a project client route", remote.User, remote.Host, remote.Port)
		o.Cleanup = "remote phase bounded by timeout + 10s; interrupted transport is not remote cleanup proof"
	}
	if operationErr == nil {
		switch action {
		case "exec", "native":
			var c acceptance.Command
			if action == "exec" {
				o.Invocation = f.Args()
				c = acceptance.Command{Name: f.Args()[0], Args: f.Args()[1:], Stdin: os.Stdin}
				if *remoteFile != "" {
					c, operationErr = remote.Command(f.Args(), os.Stdin)
				}
			} else {
				c, operationErr = remote.NativePhase(*request, *revision, *arch, *target)
				var req acceptance.RemoteRequest
				if operationErr == nil {
					operationErr = nativebuild.ReadJSON(*request, &req)
					o.Invocation = []string{"embedded native executor", req.Phase, req.Revision, req.Architecture, req.Target, req.Work}
					o.Action = "native-" + req.Phase
				}
			}
			if operationErr == nil {
				r, werr := acceptance.Execute(phase, e, "check", c)
				o.ExitCode = r.ExitCode
				operationErr = r.Err
				evidenceErr = werr
				if r.Started {
					o.Execution = "completed"
					if r.Err != nil {
						o.Execution = "failed"
					}
				}
			}
		case "probe-ssh":
			o.Topology = fmt.Sprintf("direct client TCP to %s:%d; no proxy, forwarding inference or Git authentication", remote.Host, remote.Port)
			o.Invocation = []string{"SSH key exchange only", remote.Host, fmt.Sprint(remote.Port), remote.KnownHosts}
			fingerprint, err := remote.ProbeSSHKey(phase)
			operationErr = err
			o.Execution = "failed"
			o.Cleanup = "connection closed"
			if err == nil {
				o.Execution = "completed"
				evidenceErr = e.Write("ssh-key.txt", []byte(fingerprint+"\nPinned SSH endpoint observed, not Git authentication or project routing proof.\n"))
			}
		case "transfer":
			o.Invocation = []string{"transfer sealed bundle", *source, *destination}
			r, werr := remote.TransferBundle(phase, e, *source, *destination, *arch, *revision, *target)
			o.ExitCode = r.ExitCode
			operationErr = r.Err
			evidenceErr = werr
			if r.Started {
				o.Execution = "completed"
				if r.Err != nil {
					o.Execution = "failed"
				}
			}
		case "vm":
			o.Topology = "matching-native KVM; loopback management SSH only, no routed-client claim"
			o.Invocation = []string{"fresh VM", vmConfig.Name, vmConfig.Work, vmConfig.BaseReceipt, "private single fw_cfg Ignition (not retained)"}
			if *restart {
				o.Invocation = append(o.Invocation, "--restart")
			}
			if *hold {
				o.Invocation = append(o.Invocation, "--hold")
			}
			o.Cleanup = "not-reached or failed launch; work retained"
			var vm *acceptance.VM
			vm, operationErr = acceptance.LaunchVM(phase, vmConfig, e)
			if operationErr == nil {
				o.Execution = "completed"
				evidenceErr = e.Write("boot-ready.txt", []byte("Fresh guest reached pinned SSH readiness. This is fixture evidence, not a product check.\n"))
				if *restart {
					operationErr = vm.Restart(phase)
					if operationErr == nil {
						evidenceErr = errors.Join(evidenceErr, e.Write("restart-ready.txt", []byte("Same owned disk/NVRAM reached pinned SSH after restart. No project persistence assertion was performed.\n")))
					}
				}
				if operationErr == nil && *hold {
					fmt.Fprintln(os.Stdout, "Fresh guest ready. Invoke owned checks separately. Interrupt shuts down and retains disks; cancellation is not a pass.")
					operationErr = vm.Wait(phase)
				}
				cleanupErr := vm.Close()
				o.Cleanup = "completed; private disk/NVRAM retained at " + vmConfig.Work
				if cleanupErr != nil {
					o.Cleanup = "failed; private work retained"
				}
				operationErr = errors.Join(operationErr, cleanupErr)
				if operationErr != nil {
					o.Execution = "failed"
				}
			}
		}
	}
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
	evidenceErr = errors.Join(evidenceErr, e.CheckSecrets())
	allErr := errors.Join(operationErr, evidenceErr)
	if allErr != nil {
		evidenceErr = errors.Join(evidenceErr, e.Write("failure.txt", []byte(e.RedactError(allErr).Error()+"\n")))
	}
	o.Files, err = e.Hashes()
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
	data, merr := json.MarshalIndent(o, "", "  ")
	if merr == nil {
		merr = e.Write("observation.json", append(data, '\n'))
	}
	return e.RedactError(errors.Join(operationErr, evidenceErr, merr, e.CheckSecrets()))
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
