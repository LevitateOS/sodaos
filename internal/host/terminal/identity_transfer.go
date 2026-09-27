package terminal

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
)

// The guest tmpfs owns its user-namespace IDs. Tar preserves executable modes
// while extraction deliberately keeps the guest's ownership of staged files.
func (s *Service) streamIdentityHarness(ctx context.Context, container, path string) error {
	producerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	producer := exec.CommandContext(producerContext, "/usr/bin/tar", "--create", "--file=-", "--directory", filepath.Clean(s.CodexHarness), ".")
	producer.Stderr = io.Discard
	stream, err := producer.StdoutPipe()
	if err != nil {
		return errors.New("codex harness stream unavailable")
	}
	defer func() { _ = stream.Close() }()
	if err = producer.Start(); err != nil {
		return errors.New("codex harness stream unavailable")
	}
	_, consumerErr := s.Exec.RunReader(producerContext, stream, "/usr/bin/podman", "--remote=false", "exec", "--interactive", container, "/usr/bin/tar", "--extract", "--file=-", "--directory", path, "--no-same-owner", "--same-permissions")
	if consumerErr != nil {
		cancel()
	}
	_ = stream.Close()
	producerErr := producer.Wait()
	if consumerErr != nil || producerErr != nil {
		return errors.New("codex harness staging failed")
	}
	return nil
}
