package image

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/release/build"
)

// The fork is a second exact source input. It is archived beside the Soda
// snapshot before any generated bindata or native compilation changes it.
func extractForgejoSnapshot(r Request, execute build.BuildExec) error {
	if !build.Revision(r.ForgejoRevision) {
		return errors.New("exact Forgejo source revision required")
	}
	snapshot := filepath.Join(r.Out, "work/forgejo-source")
	if err := os.Mkdir(snapshot, 0o700); err != nil {
		return err
	}
	archive := filepath.Join(r.Out, "inputs/forgejo-source.tar")
	if err := execute(r.ForgejoSource, "git", "-c", "safe.directory="+r.ForgejoSource, "archive", "--format=tar", "--output", archive, r.ForgejoRevision); err != nil {
		return err
	}
	return execute(snapshot, "tar", "--extract", "--file", archive, "--no-same-owner")
}
