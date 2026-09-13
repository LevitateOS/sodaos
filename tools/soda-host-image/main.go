// soda-host-image is a local, noninteractive host-content builder. It neither
// installs a system nor signs, publishes, schedules or activates updates.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

func main() {
	// Reuse the timer commit's process-group supervisor, including descendant
	// cancellation. The legacy shell caller already runs under this supervisor.
	if os.Getenv("SODA_BUILD_SUPERVISED") != "1" {
		source, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "repository source required")
			os.Exit(1)
		}
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		python, err := exec.LookPath("python3")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		args := append([]string{python, filepath.Join(strings.TrimSpace(string(source)), "scripts/build_progress.py"), "supervise", self}, os.Args[1:]...)
		// Replace this bootstrap process so signals to its PID reach the existing
		// supervisor, rather than killing a Go wrapper and orphaning the build.
		if err = syscall.Exec(python, args, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(nativebuild.BuildExitCode(err))
	}
}

type buildInterrupted struct{ code int }

func (e buildInterrupted) Error() string { return "build interrupted" }
func (e buildInterrupted) ExitCode() int { return e.code }

func run() (resultErr error) {
	arch := flag.String("arch", "", "matching native x86_64 or aarch64")
	out := flag.String("out", "", "new absolute attempt directory below this checkout's .artifacts (parent must exist)")
	legacy := flag.Bool("legacy-native", false, "compatibility producer for build-native.sh; no host image or ISO")
	build := flag.Bool("build", false, "also pull the locked base, build/export and inspect the local OCI image")
	complete := flag.Bool("complete", false, "build the full local appliance payload (requires --build; no publication)")
	repository := flag.String("repository-prefix", "", "intended GHCR repository prefix for complete candidates; does not register or publish it")
	flag.Parse()
	ctx, stop := context.WithCancelCause(context.Background())
	defer stop(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func(wait context.Context) {
		select {
		case sig := <-signals:
			stop(buildInterrupted{128 + int(sig.(syscall.Signal))})
		case <-wait.Done():
		}
	}(ctx)
	if !*legacy {
		// Preserve the image builder's existing deadline, without imposing a new
		// timeout on the previously unbounded legacy native build.
		timed, cancel := context.WithTimeout(ctx, 45*time.Minute)
		ctx = timed
		defer cancel()
	}
	capture := func(dir, name string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		b, err := cmd.Output()
		return strings.TrimSpace(string(b)), err
	}
	source, err := capture("", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	title := "Soda release artifacts"
	if *legacy {
		title = "Legacy native components"
	}
	progress, err := nativebuild.NewBuildProgress(source, title)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, progress.Finish(resultErr)) }()
	defer func() { resultErr = errors.Join(context.Cause(ctx), resultErr) }()
	if err = progress.Next(title + " / Check candidate inputs"); err != nil {
		return err
	}
	if *legacy && (*build || *complete || *repository != "") {
		return errors.New("legacy-native cannot select release image options")
	}
	if *complete && (!*build || !appliancerelease.ValidRepositoryPrefix(*repository)) {
		return errors.New("--complete requires --build and an explicit ghcr.io/OWNER/PREFIX")
	}
	if !*complete && *repository != "" {
		return errors.New("repository prefix applies only to complete candidates")
	}
	if flag.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if err = nativebuild.RequireNative(*arch); err != nil {
		return err
	}
	if runtime.Version() != "go1.26.7" {
		return errors.New("run with pinned Go 1.26.7")
	}
	status, err := capture(source, "git", "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return err
	}
	if status != "" {
		return errors.New("clean committed source required; uncommitted work is never staged automatically")
	}
	revision, err := capture(source, "git", "rev-parse", "HEAD")
	if err != nil || !nativebuild.Revision(revision) {
		return errors.New("exact source revision required")
	}
	if wanted := os.Getenv("SODA_BUILD_REVISION"); wanted != "" && revision != wanted {
		return errors.New("source revision differs from requested build")
	}
	if !*legacy {
		if _, err = hostimage.LoadBase(source, *arch); err != nil {
			return err
		}
	}
	if !filepath.IsAbs(*out) || !strings.HasPrefix(filepath.Clean(*out), filepath.Join(source, ".artifacts")+string(os.PathSeparator)) {
		return errors.New("fresh output must be below this checkout's .artifacts")
	}
	if *legacy && *out != filepath.Join(source, ".artifacts/native", *arch) {
		return errors.New("legacy-native requires the owning native stage path")
	}
	if err = nativebuild.FreshDirectory(*out); err != nil {
		return err
	}
	if err = progress.CreateLog(filepath.Join(*out, "timing.log")); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(*out, "build.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	execution := nativebuild.BuildExecution{Context: ctx, Log: log, Output: os.Stdout}
	execute := execution.Execute
	capture = execution.Capture
	next := func(label string) error { return progress.Next(title + " / " + label) }
	if *legacy {
		return legacyNative(source, *out, *arch, revision, execution, next)
	}
	if err = next("Freeze committed source"); err != nil {
		return err
	}
	// git archive creates an exact, secret-free committed snapshot, not a worktree.
	// Later edits to the canonical checkout cannot change these candidate inputs.
	snapshot := filepath.Join(*out, "source")
	if err = os.Mkdir(snapshot, 0700); err != nil {
		return err
	}
	archive := filepath.Join(*out, "source.tar")
	if err = execute(source, "git", "archive", "--format=tar", "--output", archive, revision); err != nil {
		return err
	}
	if err = execute(snapshot, "tar", "--extract", "--file", archive, "--no-same-owner"); err != nil {
		return err
	}
	if err = next("Prepare host context"); err != nil {
		return err
	}
	contextDir := filepath.Join(*out, "context")
	base, err := hostimage.Prepare(snapshot, contextDir, *arch, revision)
	if err != nil {
		return err
	}
	producer := nativebuild.Production{Source: snapshot, Native: filepath.Join(snapshot, ".artifacts/native", *arch), Out: *out, Arch: *arch, Revision: revision, Vendor: true, Execute: execute, Capture: capture, Next: next}
	commands, err := nativebuild.SodaCommands(snapshot)
	if err != nil {
		return err
	}
	for _, name := range commands {
		binary := filepath.Join(contextDir, "rootfs/usr/libexec/soda", name)
		if err = producer.Compile(name, "./cmd/"+name, binary); err != nil {
			return err
		}
	}
	scope := "host-content-only"
	if *complete {
		if _, err = completeCandidate(snapshot, contextDir, *out, *arch, revision, *repository, base, producer); err != nil {
			return err
		}
		scope = "complete-local-payload"
	}
	if err = next("Inventory host context"); err != nil {
		return err
	}
	if err = hostimage.Inventory(contextDir); err != nil {
		return err
	}
	if !*build {
		fmt.Println("Prepared host-content context at", contextDir, "(no OCI build or native boot proof)")
		return nil
	}
	platform, _ := nativebuild.OCIArchitecture(*arch)
	pinned := base.Images[*arch]
	if err = next("Pull locked CoreOS base"); err != nil {
		return err
	}
	if err = execute(contextDir, "podman", "--remote=false", "pull", "--platform=linux/"+platform, pinned); err != nil {
		return err
	}
	iid := filepath.Join(*out, "host.iid")
	if err = next("Build host image"); err != nil {
		return err
	}
	if err = execute(contextDir, "podman", "--remote=false", "build", "--pull=never", "--rm=false", "--platform=linux/"+platform, "--build-arg=BASE_IMAGE="+pinned, "--build-arg=PAYLOAD_SCOPE="+scope, "--label=org.opencontainers.image.revision="+revision, "--label=org.opencontainers.image.base.name="+pinned, "--label=org.opencontainers.image.base.digest="+strings.SplitN(pinned, "@", 2)[1], "--label=org.opencontainers.image.version="+base.Release+".soda-"+revision[:12], "--iidfile", iid, "--file", "Containerfile", "."); err != nil {
		return err
	}
	data, err := os.ReadFile(iid)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(string(data))
	if !strings.HasPrefix(id, "sha256:") || !nativebuild.Digest(strings.TrimPrefix(id, "sha256:")) {
		return errors.New("invalid built image ID")
	}
	if err = next("Inspect host identity and contents"); err != nil {
		return err
	}
	// Selected public fields only, never full container/environment inspection.
	observed, err := capture(contextDir, "podman", "--remote=false", "image", "inspect", "--format", `{{.Os}}/{{.Architecture}} {{index .Labels "org.opencontainers.image.revision"}} {{index .Labels "io.soda.host-image.scope"}}`, id)
	if err != nil {
		return err
	}
	if observed != "linux/"+platform+" "+revision+" "+scope {
		return errors.New("built image identity/platform mismatch")
	}
	// Networkless, read-only build inspection, not an appliance boot. Keep the
	// exact stopped inspection container/CID as evidence; no --rm or cleanup.
	packages, err := capture(contextDir, "podman", "--remote=false", "run",
		"--cidfile", filepath.Join(*out, "inspect.cid"), "--network=none", "--read-only",
		"--cap-drop=all", "--security-opt=no-new-privileges", "--entrypoint=/bin/sh", id,
		"-ec", `test "$(stat -c %a /usr/libexec/soda/soda-host)" = 755
 test -f /usr/lib/systemd/system/soda-project@.service
 test -f /usr/share/containers/systemd/forgejo.container
 test ! -e /usr/local/libexec/soda/soda-host
 test ! -e /etc/soda/host.json
 test ! -e /etc/soda/dashboard.json
 rpm -q bootc rpm-ostree cockpit-ostree tailscale >/dev/null
 cat /usr/share/soda/host-image/packages.txt`)
	if err != nil {
		return fmt.Errorf("read-only image inspection failed; preserve inspect.cid: %w", err)
	}
	if packages == "" {
		return errors.New("empty built package inventory")
	}
	if err = nativebuild.WriteNew(filepath.Join(*out, "packages.txt"), []byte(packages+"\n"), 0600); err != nil {
		return err
	}
	if *complete {
		if err = inspectComplete(contextDir, *out, id, capture); err != nil {
			return err
		}
	}
	if err = next("Export host archive"); err != nil {
		return err
	}
	if err = execute(contextDir, "podman", "--remote=false", "save", "--format=oci-archive", "--output", filepath.Join(*out, "host.oci"), id); err != nil {
		return err
	}
	if err = next("Verify exported host and record candidate"); err != nil {
		return err
	}
	archiveImage, err := nativebuild.InspectOCI(filepath.Join(*out, "host.oci"), *arch, revision)
	if err != nil {
		return err
	}
	if archiveImage.Config != id || archiveImage.BaseName != pinned || archiveImage.BaseDigest != strings.SplitN(pinned, "@", 2)[1] {
		return errors.New("exported OCI differs from built image/base")
	}
	hash, err := nativebuild.HashFile(filepath.Join(*out, "host.oci"))
	if err != nil {
		return err
	}
	if *complete {
		if err = recordCandidate(*out, *repository, archiveImage, hash); err != nil {
			return err
		}
	}
	result := struct{ Revision, Architecture, Base, Image, Manifest, ArchiveSHA256, Scope string }{revision, *arch, pinned, id, archiveImage.Manifest, hash, scope + "; unsigned/unpublished; no boot/install/upgrade acceptance"}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = nativebuild.WriteNew(filepath.Join(*out, "result.json"), append(encoded, '\n'), 0600); err != nil {
		return err
	}
	fmt.Println("Built local unsigned", scope, "at", *out, "; no publication or boot/upgrade acceptance")
	return nil
}
