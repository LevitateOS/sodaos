package appliancerelease

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// ImportRetained loads release-owned images into ordinary Podman storage. The
// historical v1 reader loads its two retained roles; v2/v3 load all five images.
// No container/project/service is started, replaced or removed.
// run is the existing concrete native command boundary, injectable for tests.
func ImportRetained(ctx context.Context, p Payload, images string, run func(context.Context, string, ...string) error) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Format == 3 {
		// Validate the whole layout before importing anything, including images
		// already present locally. No corrupt later image can authorize a partial import.
		if _, _, err := VerifyContent(p, images); err != nil {
			return err
		}
	}
	names := []string{"project-os", "tailnet"}
	if p.Format >= 2 {
		names = Names
	}
	for _, name := range names {
		im := p.Images[name]
		archive := filepath.Join(images, name+".oci")
		if p.Format != 3 {
			hash, err := nativebuild.HashFile(archive)
			if err != nil || hash != im.ArchiveSHA256 {
				return fmt.Errorf("%s archive integrity unavailable", name)
			}
			revision := p.Revision
			if name == "proxy" {
				revision = "" // Unmodified upstream image, never relabel its provenance.
			}
			observed, err := nativebuild.InspectOCI(archive, p.Architecture, revision)
			if err != nil || observed.Config != im.Config || observed.Manifest != im.Manifest {
				return fmt.Errorf("%s archive identity mismatch", name)
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		err := run(ctx, "/usr/bin/podman", "--remote=false", "image", "exists", im.Config)
		if err == nil {
			continue
		}
		var exit interface{ ExitCode() int }
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return fmt.Errorf("%s image observation failed", name)
		}
		if p.Format == 3 {
			// Explicit local OCI transport: no registry lookup, mutable tag or
			// source credentials. The reference is the same immutable config ID
			// used by our retained OCI archives and generated Quadlets.
			err = run(ctx, "/usr/bin/podman", "--remote=false", "pull", "--retry=0", "oci:"+images+":"+im.Config)
		} else {
			err = run(ctx, "/usr/bin/podman", "--remote=false", "load", "--input", archive)
		}
		if err != nil {
			return fmt.Errorf("%s image import unconfirmed", name)
		}
		if err = run(ctx, "/usr/bin/podman", "--remote=false", "image", "exists", im.Config); err != nil {
			return fmt.Errorf("%s imported image unavailable", name)
		}
	}
	return nil
}

func NativeImport(ctx context.Context, p Payload) error {
	return ImportRetained(ctx, p, ImagesPath, func(ctx context.Context, cmd string, args ...string) error {
		// Only exact local archive/ID arguments. Do not capture full engine inspection
		// or emit raw native diagnostics from the privileged startup helper.
		return exec.CommandContext(ctx, cmd, args...).Run()
	})
}
