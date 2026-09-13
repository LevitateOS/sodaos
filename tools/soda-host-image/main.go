// soda-host-image is a local, noninteractive host-content builder. It neither
// installs a system nor signs, publishes, schedules or activates updates.
package main

import (
	"context"
	"debug/elf"
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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	arch := flag.String("arch", "", "matching native x86_64 or aarch64")
	out := flag.String("out", "", "new absolute attempt directory below this checkout's .artifacts (parent must exist)")
	build := flag.Bool("build", false, "also pull the locked base, build/export and inspect the local OCI image")
	complete := flag.Bool("complete", false, "build the full local appliance payload (requires --build; no publication)")
	repository := flag.String("repository-prefix", "", "intended GHCR repository prefix for complete candidates; does not register or publish it")
	flag.Parse()
	if *complete && (!*build || !appliancerelease.ValidRepositoryPrefix(*repository)) {
		return errors.New("--complete requires --build and an explicit ghcr.io/OWNER/PREFIX")
	}
	if !*complete && *repository != "" {
		return errors.New("repository prefix applies only to complete candidates")
	}
	if flag.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if err := nativebuild.RequireNative(*arch); err != nil {
		return err
	}
	if runtime.Version() != "go1.26.7" {
		return errors.New("run with pinned Go 1.26.7")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()
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
	if _, err = hostimage.LoadBase(source, *arch); err != nil {
		return err
	}
	if !filepath.IsAbs(*out) || !strings.HasPrefix(filepath.Clean(*out), filepath.Join(source, ".artifacts")+string(os.PathSeparator)) {
		return errors.New("fresh output must be below this checkout's .artifacts")
	}
	if err = nativebuild.FreshDirectory(*out); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(*out, "build.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	execute := func(dir, name string, args ...string) error {
		fmt.Fprintf(log, "\n$ %s %s\n", name, strings.Join(args, " "))
		fmt.Printf("%s %s\n", name, strings.Join(args, " "))
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Stdout = log
		cmd.Stderr = log
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=-mod=readonly", "CGO_ENABLED=0", "PATH="+filepath.Join(runtime.GOROOT(), "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed; retain attempt and inspect build.log: %w", name, err)
		}
		return nil
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
	contextDir := filepath.Join(*out, "context")
	base, err := hostimage.Prepare(snapshot, contextDir, *arch, revision)
	if err != nil {
		return err
	}
	commands, err := hostimage.Commands(snapshot)
	if err != nil {
		return err
	}
	goTool := filepath.Join(runtime.GOROOT(), "bin/go")
	for _, name := range commands {
		binary := filepath.Join(contextDir, "rootfs/usr/libexec/soda", name)
		if err = execute(snapshot, goTool, "build", "-mod=readonly", "-buildvcs=false", "-trimpath", "-tags=soda_host_image", "-o", binary, "./cmd/"+name); err != nil {
			return err
		}
		if err = os.Chmod(binary, 0755); err != nil {
			return err
		}
		if err = checkELF(binary, *arch); err != nil {
			return err
		}
	}
	scope := "host-content-only"
	if *complete {
		if _, err = completeCandidate(snapshot, contextDir, *out, *arch, revision, *repository, base, execute, capture); err != nil {
			return err
		}
		scope = "complete-local-payload"
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
	if err = execute(contextDir, "podman", "--remote=false", "pull", "--platform=linux/"+platform, pinned); err != nil {
		return err
	}
	iid := filepath.Join(*out, "host.iid")
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
	if err = execute(contextDir, "podman", "--remote=false", "save", "--format=oci-archive", "--output", filepath.Join(*out, "host.oci"), id); err != nil {
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

func checkELF(path, arch string) error {
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	machine := elf.EM_X86_64
	if arch == "aarch64" {
		machine = elf.EM_AARCH64
	}
	if f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB || f.Machine != machine {
		return errors.New("built ELF architecture mismatch")
	}
	return nil
}
