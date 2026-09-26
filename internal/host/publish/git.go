package publish

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

type repository struct{ dir, token, user string }

func (g *repository) initialize(ctx context.Context, bundle []byte) error {
	if len(bundle) == 0 || len(bundle) > 4<<20 {
		return errors.New("candidate bundle exceeds publication limit")
	}
	if err := os.WriteFile(filepath.Join(g.dir, "candidate.bundle"), bundle, 0o600); err != nil {
		return err
	}
	script := `#!/bin/sh
case "$1" in
 *Username*) printf '%s\n' "$SODA_PUBLISH_USERNAME" ;;
 *Password*) cat "$SODA_PUBLISH_TOKEN_FILE" ;;
 *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(g.dir, "askpass"), []byte(script), 0o700); err != nil {
		return err
	}
	_, err := g.run(ctx, "init", "--bare", filepath.Join(g.dir, "repo.git"))
	return err
}

func (g *repository) run(ctx context.Context, args ...string) ([]byte, error) {
	prefix := []string{"-c", "core.hooksPath=/dev/null", "-c", "http.followRedirects=false", "-c", "protocol.file.allow=always", "-c", "protocol.ext.allow=never", "--git-dir=" + filepath.Join(g.dir, "repo.git")}
	command := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	command.Dir = g.dir
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + g.dir, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=" + filepath.Join(g.dir, "askpass"), "SODA_PUBLISH_TOKEN_FILE=" + g.token, "SODA_PUBLISH_USERNAME=" + g.user, "LC_ALL=C"}
	var output limitedOutput
	command.Stdout = &output
	// Native diagnostics can contain credentials or remote repository content.
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("publication Git operation failed")
	}
	return output.Bytes(), nil
}

type limitedOutput struct{ bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4<<20 {
		return 0, errors.New("publication output limit exceeded")
	}
	return b.Buffer.Write(p)
}
