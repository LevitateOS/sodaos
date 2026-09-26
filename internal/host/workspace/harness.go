package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (c Config) validateHarnessDigest() error {
	file, err := os.Open(filepath.Join(c.HarnessDirectory, "bin/codex"))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != c.HarnessSHA256 {
		return errors.New("codex executable digest differs from configured pin")
	}
	return nil
}

// CheckHarness verifies the bytes and version that this runtime will execute.
func (w *Runtime) CheckHarness(ctx context.Context) error {
	if err := w.Config.validateHarnessDigest(); err != nil {
		return err
	}
	out, err := w.Exec.Run(ctx, nil, filepath.Join(w.Config.HarnessDirectory, "bin/codex"), "--version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) != "codex-cli "+w.Config.HarnessVersion {
		return errors.New("codex executable version differs from configured pin")
	}
	return nil
}
