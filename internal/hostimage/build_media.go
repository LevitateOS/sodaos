package hostimage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/installer"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/releasedelivery"
)

type mediaTools struct{ Butane, Version, Architecture string }

func prepareBuildMedia(p nativebuild.Production, r Request) (tools mediaTools, lock mediaLock, err error) {
	if !r.WantsMedia() {
		return
	}
	if err = p.Next("P2 / Verify native media tooling"); err != nil {
		return
	}
	tools, err = admitMediaTools(p.Source, filepath.Join(r.Out, "evidence"), r.Arch, p)
	if err == nil {
		lock, err = prepareAssembler(p, filepath.Join(r.Out, "work/media"))
	}
	return
}

func finishBuildMedia(ctx context.Context, p nativebuild.Production, r Request, tools mediaTools, lock mediaLock, next func(string) error) error {
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

func admitMediaTools(source, evidence, arch string, p nativebuild.Production) (mediaTools, error) {
	var tools mediaTools
	if err := nativebuild.ReadJSON(filepath.Join(source, "appliance/locks/installer-tools.json"), &tools); err != nil {
		return tools, err
	}
	const prefix = "quay.io/coreos/butane@sha256:"
	if tools.Architecture != arch || !strings.HasPrefix(tools.Butane, prefix) || !nativebuild.Digest(strings.TrimPrefix(tools.Butane, prefix)) || !strings.HasPrefix(tools.Version, "Butane v") || strings.ContainsAny(tools.Version, "\r\n") {
		return tools, errors.New("unreviewed native media tool")
	}
	if err := p.Execute(source, "podman", "--remote=false", "pull", tools.Butane); err != nil {
		return tools, err
	}
	platform, _ := nativebuild.OCIArchitecture(arch)
	observed, err := p.Capture(source, "podman", "--remote=false", "image", "inspect", "--format", "{{.Os}}/{{.Architecture}}", tools.Butane)
	if err != nil {
		return tools, err
	}
	if observed != "linux/"+platform {
		return tools, errors.New("native Butane platform mismatch")
	}
	version, err := p.Capture(source, "podman", "--remote=false", "run", "--pull=never", "--cidfile", filepath.Join(evidence, "butane-version.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", tools.Butane, "--version")
	if err != nil {
		return tools, err
	}
	if version != tools.Version {
		return tools, errors.New("native Butane version mismatch")
	}
	return tools, nil
}

// Generate the live handoff in the same source-to-candidate run, using the exact
// candidate and once-compiled console. This is public build output, not protected
// media admission, ISO assembly or installation authority.
func prepareMediaInputs(source, out string, tools mediaTools, p nativebuild.Production) error {
	destination, err := p.Capture(source, "podman", "--remote=false", "run", "--pull=never", "--cidfile", filepath.Join(out, "destination-convert.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=label=disable", "--volume="+filepath.Join(source, "appliance/provisioning/candidate.json")+":/input.json:ro", tools.Butane, "--strict", "/input.json")
	if err != nil {
		return err
	}
	var candidate releasedelivery.Candidate
	if err = nativebuild.ReadJSON(filepath.Join(out, "candidate.json"), &candidate); err != nil {
		return err
	}
	payload, err := os.ReadFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return err
	}
	console, err := nativebuild.HashFile(filepath.Join(out, "tools/soda-installer"))
	if err != nil {
		return err
	}
	live, err := installer.CandidateLiveConfig(payload, []byte(destination), candidate.Host.Manifest, console)
	if err != nil {
		return err
	}
	if err = nativebuild.WriteNew(filepath.Join(out, "destination.ign"), []byte(destination+"\n"), 0644); err != nil {
		return err
	}
	return nativebuild.WriteNew(filepath.Join(out, "live.ign"), append(live, '\n'), 0644)
}
