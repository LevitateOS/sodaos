package image

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

// Skopeo owns OCI export and shared-blob reuse. Detached per-image archives stay
// untouched for delivery/provenance; only the shared layout enters the host image.
func validStageLayout(archives, destination string, p deliver.Payload, run build.BuildExec) bool {
	return p.Format == 3 && run != nil && filepath.IsAbs(archives) && filepath.IsAbs(destination) && !strings.ContainsAny(archives+destination, ":\r\n")
}

func verifyArchiveDigests(archives string, p deliver.Payload) error {
	for _, name := range deliver.Names {
		hash, err := build.HashFile(filepath.Join(archives, name+".oci"))
		if err != nil || hash != p.Images[name].ArchiveSHA256 {
			return fmt.Errorf("%s archive changed before staging", name)
		}
	}
	return nil
}

func copyStagedArchives(archives, destination string, p deliver.Payload, run build.BuildExec) error {
	for _, name := range deliver.Names {
		if err := run(archives, "skopeo", "copy", "--preserve-digests", "--dest-oci-accept-uncompressed-layers", "oci-archive:"+filepath.Join(archives, name+".oci"), "oci:"+destination+":"+p.Images[name].Config); err != nil {
			return err
		}
	}
	return nil
}

func chmodStagedFiles(destination string, files map[string]string) error {
	for name := range files {
		if err := os.Chmod(filepath.Join(destination, name), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func stageImages(archives, destination string, p deliver.Payload, run build.BuildExec) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if !validStageLayout(archives, destination, p, run) {
		return errors.New("explicit local v3 layout staging required")
	}
	if err := verifyArchiveDigests(archives, p); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	if err := copyStagedArchives(archives, destination, p, run); err != nil {
		return err
	}
	files, _, err := deliver.VerifyContent(p, destination)
	if err != nil {
		return err
	}
	return chmodStagedFiles(destination, files)
}
