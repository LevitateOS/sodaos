package hostimage

import (
	"encoding/json"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// The final host manifest cannot be embedded in itself. This detached local
// receipt binds that manifest to the exact embedded payload bytes. It is NOT a
// signed release/channel authority or a qualified upgrade declaration.
func recordCandidate(out, prefix string, host nativebuild.Image, archiveHash string) error {
	payloadHash, err := nativebuild.HashFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return err
	}
	record := struct {
		Format                                                            int
		Host                                                              nativebuild.Image
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
	return nativebuild.WriteNew(filepath.Join(out, "candidate.json"), append(b, '\n'), 0600)
}
