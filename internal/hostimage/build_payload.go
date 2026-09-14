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

// completeCandidate is image-layout assembly, not a second component producer.
// The legacy installer uses the same Production methods with its explicit layout.
func completeCandidate(source, context, out, arch, revision, prefix string, base Base, packageHash string, producer nativebuild.Production, phase func(string) error) (appliancerelease.Payload, error) {
	p := appliancerelease.Payload{Format: 3, ID: base.Release + ".soda-" + revision[:12], Revision: revision, Architecture: arch, CoreOS: base.Release, Base: base.Images[arch], RepositoryPrefix: prefix, Schema: store.SchemaVersion(), Images: map[string]appliancerelease.Image{}, UpgradeFrom: []string{}}
	var err error
	if err = producer.Next("Verify admitted host package transaction"); err != nil {
		return p, err
	}
	p.HostPackagesSHA256, err = nativebuild.HashFile(filepath.Join(context, "packages.expected"))
	if err != nil || p.HostPackagesSHA256 != packageHash {
		return p, fmt.Errorf("admitted package inventory changed: %v", err)
	}
	native := producer.Native
	if err = os.MkdirAll(filepath.Join(native, "bin"), 0755); err != nil {
		return p, err
	}
	commands, err := nativebuild.SodaCommands(source)
	if err != nil {
		return p, err
	}
	// Reuse the already compiled vendor binaries; never compile again for assets.
	for _, name := range commands {
		if err = os.Link(filepath.Join(context, "rootfs/usr/libexec/soda", name), filepath.Join(native, "bin", name)); err != nil {
			return p, err
		}
	}
	forgejoContext := filepath.Join(out, "forgejo-context")
	if err = os.Mkdir(forgejoContext, 0700); err != nil {
		return p, err
	}
	if err = producer.Assets(context, forgejoContext); err != nil {
		return p, err
	}
	if err = producer.Next("Verify immutable Forgejo presentation"); err != nil {
		return p, err
	}
	p.PresentationSHA256, err = StagePresentation(forgejoContext, context)
	if err != nil {
		return p, err
	}
	recipe, err := os.ReadFile(filepath.Join(source, "appliance/forgejo.Containerfile"))
	if err != nil {
		return p, err
	}
	if err = nativebuild.WriteNew(filepath.Join(forgejoContext, "Containerfile"), recipe, 0644); err != nil {
		return p, err
	}
	if err = preparedChecks(producer); err != nil {
		return p, err
	}
	if err = phase("P4 / Build application images"); err != nil {
		return p, err
	}
	images, err := producer.Images(forgejoContext)
	if err != nil {
		return p, err
	}
	for name, im := range images {
		p.Images[name] = appliancerelease.Image{Reference: prefix + "-" + name + "@" + im.Manifest, Manifest: im.Manifest, Config: im.Config, ArchiveSHA256: im.ArchiveSHA256, Storage: "podman"}
	}
	if err = producer.Next("Inspect immutable Forgejo presentation"); err != nil {
		return p, err
	}
	// Read-only upstream binary, not its database/bootstrap entrypoint.
	result, err := producer.Capture(source, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "forgejo-inspect.cid"), "--network=none", "--read-only", "--entrypoint=/bin/sh", images["forgejo"].Config, "-ec", `test "$GITEA_CUSTOM" = /usr/share/soda/forgejo
 test "$FORGEJO_CUSTOM" = "$GITEA_CUSTOM"
 test "$(readlink "$GITEA_CUSTOM/conf")" = /data/gitea/conf
 test "$(stat -c '%u:%g:%a' "$GITEA_CUSTOM/templates/custom/header.tmpl")" = 0:0:444
 test -s "$GITEA_CUSTOM/public/assets/soda/forgejo/soda-native-page.js"
 /usr/local/bin/gitea --version`)
	if err != nil {
		return p, fmt.Errorf("Forgejo payload image inspection failed: %w", err)
	}
	if err = nativebuild.WriteNew(filepath.Join(out, "forgejo-inspection.txt"), []byte(result+"\n"), 0600); err != nil {
		return p, err
	}
	if err = producer.Next("Assemble host payload and ordinary Podman image references"); err != nil {
		return p, err
	}
	if err = p.Validate(); err != nil {
		return p, err
	}
	if err = Complete(source, context, filepath.Join(out, "images"), p, producer.Execute); err != nil {
		return p, err
	}
	record, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	err = nativebuild.WriteNew(filepath.Join(out, "payload.json"), append(record, '\n'), 0600)
	return p, err
}
