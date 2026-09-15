package image

import (
	"encoding/json"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

// Called only after the fixed candidate checks and requested media checks succeed.
// The candidate hash binds all five app identities through its payload hash; this
// producer receipt is never protected qualification evidence.
func recordBuildResult(p build.Production, r Request) (result Result, err error) {
	result = Result{Revision: p.Revision, Architecture: p.Arch,
		Candidate: filepath.Join(p.Out, "candidate.json"), Purpose: r.Purpose(),
		RequestedTarget: r.RequestedTarget(), CompletedTarget: "candidate",
		Scope:  "P1-P6 verified candidate; not a qualified release",
		Checks: []string{"Go source tests", "TypeScript and Lit checks", "Prepared frontend tests", "Prepared Forgejo tests", "Prepared layout tests", "Build/source fixtures", "ELF architecture", "Application OCI identities", "Host identity, RPM inventory, shared layout and Quadlets", "Host OCI export"}}
	var candidate deliver.Candidate
	if err = build.ReadJSON(result.Candidate, &candidate); err != nil {
		return
	}
	result.HostManifest, result.PayloadSHA256 = candidate.Host.Manifest, candidate.PayloadSHA256
	if result.CandidateSHA256, err = build.HashFile(result.Candidate); err != nil {
		return
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
		err = build.WriteNew(filepath.Join(r.Out, "evidence/build.json"), append(data, '\n'), 0600)
	}
	return
}

// The final host manifest cannot be embedded in itself. This detached local
// receipt binds that manifest to the exact embedded payload bytes. It is NOT a
// signed release/channel authority or a qualified upgrade declaration.
func recordCandidate(out, prefix string, host build.Image, archiveHash string) error {
	payloadHash, err := build.HashFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return err
	}
	record := struct {
		Format                                                            int
		Host                                                              build.Image
		HostReference, HostArchiveSHA256, PayloadSHA256, Migration, Notes string
	}{
		1, host, prefix + "-host@" + host.Manifest, archiveHash, payloadHash,
		"No upgrade or writable-install migration is qualified. Conflicting saved image selections are refused; machine settings/secrets require explicit first-install setup.",
		"Complete unsigned local appliance payload only. All app content is embedded in one shared OCI layout for ordinary Podman import; repository names are intended, not provisioned. No production, native boot, signature or recovery acceptance.",
	}
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(out, "candidate.json"), append(b, '\n'), 0600)
}
