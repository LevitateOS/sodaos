package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/store"
)

// completeCandidate is image-layout assembly, not a second component producer.
// The legacy installer uses the same Production methods with its explicit layout.
func completeCandidate(source, context, out, arch, revision, prefix string, base hostimage.Base, producer nativebuild.Production) (appliancerelease.Payload, error) {
	p := appliancerelease.Payload{Format: 1, ID: base.Release + ".soda-" + revision[:12], Revision: revision, Architecture: arch, CoreOS: base.Release, Base: base.Images[arch], RepositoryPrefix: prefix, Schema: store.SchemaVersion(), Images: map[string]appliancerelease.Image{}, UpgradeFrom: []string{}}
	var err error
	if err = producer.Next("Lock host package transaction"); err != nil {
		return p, err
	}
	p.HostPackagesSHA256, err = hostimage.LockHostPackages(source, context, arch, base)
	if err != nil {
		return p, err
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
		b, err := os.ReadFile(filepath.Join(context, "rootfs/usr/libexec/soda", name))
		if err != nil {
			return p, err
		}
		if err = nativebuild.WriteNew(filepath.Join(native, "bin", name), b, 0755); err != nil {
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
	p.PresentationSHA256, err = hostimage.StagePresentation(forgejoContext, context)
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
	images, err := producer.Images(forgejoContext)
	if err != nil {
		return p, err
	}
	for name, im := range images {
		storage := "bound"
		if name == "project-os" || name == "tailnet" {
			storage = "retained"
		}
		p.Images[name] = appliancerelease.Image{Reference: prefix + "-" + name + "@" + im.Manifest, Manifest: im.Manifest, Config: im.Config, ArchiveSHA256: im.ArchiveSHA256, Storage: storage}
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
	if err = producer.Next("Assemble host payload and bound image references"); err != nil {
		return p, err
	}
	if err = p.Validate(); err != nil {
		return p, err
	}
	if err = hostimage.Complete(source, context, filepath.Join(out, "images"), p); err != nil {
		return p, err
	}
	record, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	err = nativebuild.WriteNew(filepath.Join(out, "payload.json"), append(record, '\n'), 0600)
	return p, err
}
