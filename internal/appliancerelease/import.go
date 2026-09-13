package appliancerelease

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// ImportRetained adds only the two release-owned creation/companion images to
// ordinary Podman storage. Never point persistent projects at bootc's GC-owned
// additional store. No container/project/service is started, replaced or removed.
// run is the existing concrete native command boundary, injectable for tests.
func ImportRetained(ctx context.Context, p Payload, images string, run func(context.Context, string, ...string) error) error {
	if err := p.Validate(); err != nil {
		return err
	}
	for _, name := range []string{"project-os", "tailnet"} {
		im := p.Images[name]
		archive := filepath.Join(images, name+".oci")
		hash, err := nativebuild.HashFile(archive)
		if err != nil || hash != im.ArchiveSHA256 {
			return fmt.Errorf("%s archive integrity unavailable", name)
		}
		observed, err := nativebuild.InspectOCI(archive, p.Architecture, p.Revision)
		if err != nil || observed.Config != im.Config || observed.Manifest != im.Manifest {
			return fmt.Errorf("%s archive identity mismatch", name)
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		err = run(ctx, "/usr/bin/podman", "--remote=false", "image", "exists", im.Config)
		if err == nil {
			continue
		}
		var exit interface{ ExitCode() int }
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return fmt.Errorf("%s image observation failed", name)
		}
		if err = run(ctx, "/usr/bin/podman", "--remote=false", "load", "--input", archive); err != nil {
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
