package appliancerelease

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// ImportImages verifies release content and imports missing exact images into
// ordinary Podman. It never starts, replaces or deletes workloads.
func ImportImages(ctx context.Context, p Payload, images string, run func(context.Context, string, ...string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, _, err := VerifyContent(p, images); err != nil {
		return err
	}
	for _, name := range Names {
		im := p.Images[name]
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
		if err = run(ctx, "/usr/bin/podman", "--remote=false", "pull", "--retry=0", "oci:"+images+":"+im.Config); err != nil {
			return fmt.Errorf("%s image import unconfirmed", name)
		}
		if err = run(ctx, "/usr/bin/podman", "--remote=false", "image", "exists", im.Config); err != nil {
			return fmt.Errorf("%s imported image unavailable", name)
		}
	}
	return nil
}

func NativeImport(ctx context.Context, p Payload) error {
	return ImportImages(ctx, p, ImagesPath, func(ctx context.Context, cmd string, args ...string) error {
		// No engine inspection or raw diagnostics enter the startup helper's output.
		return exec.CommandContext(ctx, cmd, args...).Run()
	})
}
