package image

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/installer"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

type mediaTools struct{ Butane, Architecture string }

// butaneImage floats on the upstream tag; the observed version is recorded
// per build. No pinned digest or version string precedes the pull.
const butaneImage = "quay.io/coreos/butane:latest"

func prepareBuildMedia(p build.Production, r Request) (tools mediaTools, lock mediaLock, err error) {
	if !r.WantsMedia() {
		return tools, lock, err
	}
	if err = p.Next("P2 / Verify native media tooling"); err != nil {
		return tools, lock, err
	}
	tools, err = admitMediaTools(p.Source, filepath.Join(r.Out, "evidence"), p)
	if err == nil {
		lock, err = prepareAssembler(p, filepath.Join(r.Out, "work/media"))
	}
	return tools, lock, err
}

func finishBuildMedia(ctx context.Context, p build.Production, r Request, tools mediaTools, lock mediaLock, next func(string) error) error {
	if !r.WantsMedia() {
		return nil
	}
	if err := p.Next("P6 / Prepare candidate live Ignition"); err != nil {
		return err
	}
	if err := prepareMediaInputs(p.Source, p.Out, tools, p); err != nil {
		return err
	}
	_, err := assembleMedia(ctx, p, r, lock, next)
	return err
}

func verifyButaneImage(source, evidence string, p build.Production) (string, error) {
	if err := p.Execute(source, "podman", "--remote=false", "pull", butaneImage); err != nil {
		return "", err
	}
	platform, _ := build.OCIArchitecture(p.Arch)
	observed, err := p.Capture(source, "podman", "--remote=false", "image", "inspect", "--format", "{{.Os}}/{{.Architecture}}", butaneImage)
	if err != nil {
		return "", err
	}
	if observed != "linux/"+platform {
		return "", errors.New("native Butane platform mismatch")
	}
	version, err := p.Capture(source, "podman", "--remote=false", "run", "--pull=never", "--cidfile", filepath.Join(evidence, "butane-version.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", butaneImage, "--version")
	if err != nil {
		return "", err
	}
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "Butane v") || strings.ContainsAny(version, "\r\n") {
		return "", errors.New("unrecognized Butane version output")
	}
	return version, nil
}

func admitMediaTools(source, evidence string, p build.Production) (mediaTools, error) {
	tools := mediaTools{Butane: butaneImage, Architecture: p.Arch}
	version, err := verifyButaneImage(source, evidence, p)
	if err != nil {
		return tools, err
	}
	if err := build.WriteNew(filepath.Join(evidence, "butane-version.txt"), []byte(version+"\n"), 0o600); err != nil {
		return tools, err
	}
	return tools, nil
}

// Generate the live handoff in the same source-to-candidate run, using the exact
// candidate and once-compiled console. This is public build output, not protected
// media admission, ISO assembly or installation authority.
func prepareMediaInputs(source, out string, tools mediaTools, p build.Production) error {
	destination, err := p.Capture(source, "podman", "--remote=false", "run", "--pull=never", "--cidfile", filepath.Join(out, "destination-convert.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=label=disable", "--volume="+filepath.Join(source, "appliance/provisioning/candidate.json")+":/input.json:ro", tools.Butane, "--strict", "/input.json")
	if err != nil {
		return err
	}
	var candidate deliver.Candidate
	if err = build.ReadJSON(filepath.Join(out, "candidate.json"), &candidate); err != nil {
		return err
	}
	payload, err := os.ReadFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return err
	}
	console, err := build.HashFile(filepath.Join(out, "tools/soda-installer"))
	if err != nil {
		return err
	}
	live, err := installer.CandidateLiveConfig(payload, []byte(destination), candidate.Host.Manifest, console)
	if err != nil {
		return err
	}
	if err = build.WriteNew(filepath.Join(out, "destination.ign"), []byte(destination+"\n"), 0o644); err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(out, "live.ign"), append(live, '\n'), 0o644)
}
