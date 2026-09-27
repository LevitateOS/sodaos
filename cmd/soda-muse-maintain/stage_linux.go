//go:build linux

package main

import (
	"archive/tar"
	"context"
	"io"
)

// Destinations are fixed by the installed project interface, never repository input.
var destinations = []string{"/usr/local/bin/muse", "/usr/local/bin/soda-muse-compose", "/usr/local/libexec/soda/muse"}

const installScript = `
set -eu
safe_parent() {
 path=$(dirname "$1")
 while [ "$path" != / ]; do
  test ! -L "$path"
  if test -e "$path"; then test -d "$path"; fi
  path=$(dirname "$path")
 done
}
for target in "$@"; do
 safe_parent "$target"
 test ! -L "$target"
 if test -e "$target"; then test -f "$target"; fi
 done
for target in "$@"; do mkdir -p "$(dirname "$target")"; done
stage=$(mktemp -d "$(dirname "$3")/.soda-muse-maintain.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/muse" "$stage/soda-muse-compose" "$stage/muse-native"
mv -T -- "$stage/muse" "$1"
mv -T -- "$stage/soda-muse-compose" "$2"
mv -T -- "$stage/muse-native" "$3"
`

func stageTools(ctx context.Context, target observation, sources []tool) error {
	return stagePublicTools(ctx, target, sources, installScript, destinations)
}

func stagePublicTools(ctx context.Context, target observation, sources []tool, script string, targets []string) error {
	if err := confirmProject(ctx, target); err != nil {
		return err
	}
	read, write := io.Pipe()
	defer func() { _ = read.Close() }()
	complete := make(chan error, 1)
	go func() { err := archiveTools(write, sources); _ = write.CloseWithError(err); complete <- err }()
	args := []string{"exec", "--user", "0:0", "-i", target.ID, "/bin/sh", "-ceu", script, "soda-muse-maintain"}
	args = append(args, targets...)
	_, err := podman(ctx, read, args...)
	_ = read.Close()
	archiveErr := <-complete
	if err != nil {
		return err
	}
	return archiveErr
}

func archiveTools(out io.Writer, sources []tool) error {
	archive := tar.NewWriter(out)
	for _, source := range sources {
		if err := archive.WriteHeader(&tar.Header{Name: source.name, Mode: 0o755, Size: source.size, Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		if _, err := io.CopyN(archive, source.file, source.size); err != nil {
			return err
		}
	}
	return archive.Close()
}
