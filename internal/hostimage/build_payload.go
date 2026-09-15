package hostimage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/store"
)

func admitHostPackages(p *appliancerelease.Payload, context, packageHash string, producer nativebuild.Production) error {
	if err := producer.Next("Verify admitted host package transaction"); err != nil {
		return err
	}
	var err error
	p.HostPackagesSHA256, err = nativebuild.HashFile(filepath.Join(context, "packages.expected"))
	if err != nil || p.HostPackagesSHA256 != packageHash {
		return fmt.Errorf("admitted package inventory changed: %v", err)
	}
	return nil
}

func linkCandidateCommands(source, context, native string) error {
	if err := os.MkdirAll(filepath.Join(native, "bin"), 0o755); err != nil {
		return err
	}
	commands, err := nativebuild.SodaCommands(source)
	if err != nil {
		return err
	}
	// Reuse the already compiled vendor binaries; never compile again for assets.
	for _, name := range commands {
		if err = os.Link(filepath.Join(context, "rootfs/usr/libexec/soda", name), filepath.Join(native, "bin", name)); err != nil {
			return err
		}
	}
	return nil
}

func stageCandidateForgejo(p *appliancerelease.Payload, source, context, out string, producer nativebuild.Production) (string, error) {
	forgejoContext := filepath.Join(out, "forgejo-context")
	if err := os.Mkdir(forgejoContext, 0o700); err != nil {
		return "", err
	}
	if err := producer.Assets(context, forgejoContext); err != nil {
		return "", err
	}
	if err := producer.Next("Verify immutable Forgejo presentation"); err != nil {
		return "", err
	}
	var err error
	p.PresentationSHA256, err = StagePresentation(forgejoContext, context)
	if err != nil {
		return "", err
	}
	recipe, err := os.ReadFile(filepath.Join(source, "appliance/forgejo.Containerfile"))
	if err != nil {
		return "", err
	}
	if err = nativebuild.WriteNew(filepath.Join(forgejoContext, "Containerfile"), recipe, 0o644); err != nil {
		return "", err
	}
	return forgejoContext, preparedChecks(producer)
}

func recordCandidateImages(p *appliancerelease.Payload, prefix string, images map[string]nativebuild.ProducedImage) {
	for name, im := range images {
		p.Images[name] = appliancerelease.Image{Reference: prefix + "-" + name + "@" + im.Manifest, Manifest: im.Manifest, Config: im.Config, ArchiveSHA256: im.ArchiveSHA256}
	}
}

func inspectCandidateForgejo(source, out string, images map[string]nativebuild.ProducedImage, producer nativebuild.Production) error {
	if err := producer.Next("Inspect immutable Forgejo presentation"); err != nil {
		return err
	}
	// Read-only upstream binary, not its database/bootstrap entrypoint.
	result, err := producer.Capture(source, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "forgejo-inspect.cid"), "--network=none", "--read-only", "--entrypoint=/bin/sh", images["forgejo"].Config, "-ec", `test "$GITEA_CUSTOM" = /usr/share/soda/forgejo
 test "$FORGEJO_CUSTOM" = "$GITEA_CUSTOM"
 test "$(readlink "$GITEA_CUSTOM/conf")" = /data/gitea/conf
 test "$(stat -c '%u:%g:%a' "$GITEA_CUSTOM/templates/custom/header.tmpl")" = 0:0:444
 test -s "$GITEA_CUSTOM/public/assets/soda/forgejo/soda-native-page.js"
 /usr/local/bin/gitea --version`)
	if err != nil {
		return fmt.Errorf("forgejo payload image inspection failed: %w", err)
	}
	return nativebuild.WriteNew(filepath.Join(out, "forgejo-inspection.txt"), []byte(result+"\n"), 0o600)
}

func sealCandidatePayload(p appliancerelease.Payload, source, context, out string, producer nativebuild.Production) error {
	if err := producer.Next("Assemble host payload and ordinary Podman image references"); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err := Complete(source, context, filepath.Join(out, "images"), p, producer.Execute); err != nil {
		return err
	}
	record, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return nativebuild.WriteNew(filepath.Join(out, "payload.json"), append(record, '\n'), 0o600)
}

// completeCandidate is image-layout assembly, not a second component producer.
// The legacy installer uses the same Production methods with its explicit layout.
func completeCandidate(source, context, out, arch, revision, prefix string, base Base, packageHash string, producer nativebuild.Production, phase func(string) error) (appliancerelease.Payload, error) {
	p := appliancerelease.Payload{Format: 3, ID: base.Release + ".soda-" + revision[:12], Revision: revision, Architecture: arch, CoreOS: base.Release, Base: base.Images[arch], RepositoryPrefix: prefix, Schema: store.SchemaVersion(), Images: map[string]appliancerelease.Image{}, UpgradeFrom: []string{}}
	if err := admitHostPackages(&p, context, packageHash, producer); err != nil {
		return p, err
	}
	if err := linkCandidateCommands(source, context, producer.Native); err != nil {
		return p, err
	}
	forgejoContext, err := stageCandidateForgejo(&p, source, context, out, producer)
	if err != nil {
		return p, err
	}
	if err = phase("P4 / Build application images"); err != nil {
		return p, err
	}
	images, err := producer.Images(forgejoContext)
	if err != nil {
		return p, err
	}
	recordCandidateImages(&p, prefix, images)
	if err = inspectCandidateForgejo(source, out, images, producer); err != nil {
		return p, err
	}
	return p, sealCandidatePayload(p, source, context, out, producer)
}
