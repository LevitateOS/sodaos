package hostimage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

// Skopeo owns OCI export and shared-blob reuse. Detached per-image archives stay
// untouched for delivery/provenance; only the shared layout enters the host image.
func stageImages(archives, destination string, p appliancerelease.Payload, run nativebuild.BuildExec) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Format != 3 || run == nil || !filepath.IsAbs(archives) || !filepath.IsAbs(destination) || strings.ContainsAny(archives+destination, ":\r\n") {
		return errors.New("explicit local v3 layout staging required")
	}
	for _, name := range appliancerelease.Names {
		hash, err := nativebuild.HashFile(filepath.Join(archives, name+".oci"))
		if err != nil || hash != p.Images[name].ArchiveSHA256 {
			return fmt.Errorf("%s archive changed before staging", name)
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0755); err != nil {
		return err
	}
	for _, name := range appliancerelease.Names {
		if err := run(archives, "skopeo", "copy", "--preserve-digests", "--dest-oci-accept-uncompressed-layers", "oci-archive:"+filepath.Join(archives, name+".oci"), "oci:"+destination+":"+p.Images[name].Config); err != nil {
			return err
		}
	}
	files, _, err := appliancerelease.VerifyContent(p, destination)
	if err != nil {
		return err
	}
	for name := range files {
		if err := os.Chmod(filepath.Join(destination, name), 0644); err != nil {
			return err
		}
	}
	return nil
}
