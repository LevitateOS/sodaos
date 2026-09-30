package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

// These are the actual bytes staged into the host and component images. The archive
// identities in the payload bind the images; this inventory names the fork,
// independent package, and host service inputs within those images.
func candidateContent(out string) (map[string]string, error) {
	root := filepath.Dir(out)
	paths := map[string]string{
		"dashboard:/usr/local/bin/soda-dashboard":                     filepath.Join(root, "work/host-context/rootfs/usr/libexec/soda/soda-dashboard"),
		"forgejo:/usr/local/bin/gitea":                                filepath.Join(out, "forgejo-context/forgejo-bin"),
		"extension:/usr/local/bin/gitea":                              filepath.Join(out, "forgejo-context/forgejo-bin"),
		"extension:/usr/share/soda/extension/extension.json":          filepath.Join(out, "extension-context/extension/extension.json"),
		"extension:/usr/share/soda/extension/backend":                 filepath.Join(out, "extension-context/extension/backend"),
		"extension:/usr/share/soda/extension/run":                     filepath.Join(out, "extension-context/extension/run"),
		"host:/usr/share/containers/systemd/forgejo.container":        filepath.Join(root, "work/host-context/rootfs/usr/share/containers/systemd/forgejo.container"),
		"host:/usr/share/containers/systemd/soda-dashboard.container": filepath.Join(root, "work/host-context/rootfs/usr/share/containers/systemd/soda-dashboard.container"),
		"host:/usr/lib/systemd/system/soda-extension-install.service": filepath.Join(root, "work/host-context/rootfs/usr/lib/systemd/system/soda-extension-install.service"),
	}
	assets := filepath.Join(out, "extension-context/extension/assets")
	if err := filepath.WalkDir(assets, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return errors.New("extension asset must be regular")
		}
		rel, err := filepath.Rel(assets, path)
		if err == nil {
			paths["extension:/usr/share/soda/extension/assets/"+filepath.ToSlash(rel)] = path
		}
		return err
	}); err != nil {
		return nil, err
	}
	files := make(map[string]string, len(paths))
	for name, path := range paths {
		hash, err := build.HashFile(path)
		if err != nil {
			return nil, err
		}
		files[name] = hash
	}
	if !deliver.ValidCandidateContent(files) {
		return nil, errors.New("incomplete candidate content inventory")
	}
	return files, nil
}

func contentInventoryBytes(files map[string]string) ([]byte, error) {
	data, err := json.MarshalIndent(files, "", "  ")
	return append(data, '\n'), err
}

func writeContentInventory(context, out string) error {
	files, err := candidateContent(out)
	if err != nil {
		return err
	}
	data, err := contentInventoryBytes(files)
	if err != nil {
		return err
	}
	return ownedWrite(filepath.Join(context, "rootfs/usr/share/soda/host-image/content.json"), data, 0o644)
}

// Called only after the fixed candidate artifact checks and requested media
// checks succeed. The candidate hash binds all app identities through
// its payload hash; this producer receipt is never protected qualification
// evidence. Language and presentation suites are not build gates, so they
// are not listed here; run them directly when they matter.
func recordBuildResult(p build.Production, r Request) (result Result, err error) {
	result = Result{
		Revision: p.Revision, Architecture: p.Arch,
		Candidate: filepath.Join(p.Out, "candidate.json"), Purpose: r.Purpose(),
		RequestedTarget: r.RequestedTarget(), CompletedTarget: "candidate",
		Scope:  "P1-P6 verified candidate; not a qualified release",
		Checks: []string{"ELF architecture", "Application OCI identities", "Host identity, RPM inventory, shared layout and Quadlets", "Host OCI export"},
	}
	var candidate deliver.Candidate
	if err = build.ReadJSON(result.Candidate, &candidate); err != nil {
		return result, err
	}
	result.HostManifest, result.PayloadSHA256 = candidate.Host.Manifest, candidate.PayloadSHA256
	if result.CandidateSHA256, err = build.HashFile(result.Candidate); err != nil {
		return result, err
	}
	if r.WantsMedia() {
		result.MediaCompression = r.MediaCompression
		result.Media = filepath.Join(p.Out, "media/media.json")
		result.CompletedTarget = "media"
		result.Scope = "P1-P8 candidate-derived media; not a qualified release"
		result.Checks = append(result.Checks, "Authenticated packaging inputs", "Native media identity, Ignition, kernel arguments and rootfs chunks")
	}
	if r.Development {
		result.Scope = "development-only; not release-qualified"
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		err = build.WriteNew(filepath.Join(r.Out, "evidence/build.json"), append(data, '\n'), 0o600)
	}
	return result, err
}

// The final host manifest cannot be embedded in itself. This detached local
// receipt binds that manifest to the exact embedded payload bytes. It is NOT a
// signed release/channel authority or a qualified upgrade declaration.
func recordCandidate(out, prefix string, host build.Image, archiveHash, forgejoRevision, arch string) error {
	record := deliver.Candidate{
		Format:            1,
		Host:              host,
		HostReference:     prefix + "-host@" + host.Manifest,
		HostArchiveSHA256: archiveHash,
		ForgejoRevision:   forgejoRevision,
		Architecture:      arch,
		Migration:         "No upgrade or writable-install migration is qualified. Conflicting saved image selections are refused; machine settings/secrets require explicit first-install setup.",
		Notes:             "Complete unsigned local appliance payload only. All app content is embedded in one shared OCI layout for ordinary Podman import; repository names are intended, not provisioned. No production, native boot, signature or recovery acceptance.",
	}
	if err := recordCandidateInputs(out, &record); err != nil {
		return err
	}
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(out, "candidate.json"), append(b, '\n'), 0o600)
}

func recordCandidateInputs(out string, record *deliver.Candidate) error {
	if !build.Revision(record.ForgejoRevision) {
		return errors.New("exact Forgejo source revision required")
	}
	platform, err := build.OCIArchitecture(record.Architecture)
	if err != nil || record.Host.Architecture != platform {
		return errors.New("native candidate architecture differs from host image")
	}
	record.ForgejoSourceSHA256, err = build.HashFile(filepath.Join(filepath.Dir(out), "inputs/forgejo-source.tar"))
	if err != nil {
		return err
	}
	if err := build.ReadJSON(filepath.Join(out, "forgejo-toolchain.json"), &record.ForgejoToolchain); err != nil {
		return err
	}
	if err := record.ForgejoToolchain.Validate(); err != nil {
		return err
	}
	record.ContentSHA256, err = candidateContent(out)
	if err != nil {
		return err
	}
	if err := verifyEmbeddedContentInventory(out, record.ContentSHA256); err != nil {
		return err
	}
	record.PayloadSHA256, err = build.HashFile(filepath.Join(out, "payload.json"))
	return err
}

func verifyEmbeddedContentInventory(out string, content map[string]string) error {
	data, err := contentInventoryBytes(content)
	if err != nil {
		return err
	}
	embedded, err := os.ReadFile(filepath.Join(filepath.Dir(out), "work/host-context/rootfs/usr/share/soda/host-image/content.json"))
	if err != nil || !bytes.Equal(data, embedded) {
		return errors.New("candidate content inventory differs from host image input")
	}
	return nil
}
