package terminal

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type harnessStreamExecutor struct {
	*identityExecutor
	consume func(context.Context, io.Reader, string, ...string) ([]byte, error)
}

func (e harnessStreamExecutor) RunReader(ctx context.Context, in io.Reader, command string, args ...string) ([]byte, error) {
	return e.consume(ctx, in, command, args...)
}

func TestIdentityHarnessStreamsModesWithoutHostOwnership(t *testing.T) {
	harness := t.TempDir()
	if err := os.WriteFile(filepath.Join(harness, "codex"), []byte("synthetic release"), 0o755); err != nil {
		t.Fatal(err)
	}
	container := strings.Repeat("c", 64)
	target := "/run/soda-terminals/" + strings.Repeat("b", 32) + "/model/harness"
	consumer := func(_ context.Context, in io.Reader, command string, args ...string) ([]byte, error) {
		expected := []string{"--remote=false", "exec", "--interactive", container, "/usr/bin/tar", "--extract", "--file=-", "--directory", target, "--no-same-owner", "--same-permissions"}
		if command != "/usr/bin/podman" || !reflect.DeepEqual(args, expected) {
			t.Fatalf("unexpected extraction: %s %v", command, args)
		}
		archive := tar.NewReader(in)
		found := false
		for {
			h, err := archive.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Name == "./codex" {
				found = true
				if h.Mode&0o777 != 0o755 {
					t.Fatal("executable mode lost")
				}
				body, err := io.ReadAll(archive)
				if err != nil {
					return nil, err
				}
				if string(body) != "synthetic release" {
					t.Fatal("release content changed")
				}
			}
		}
		if !found {
			t.Fatal("release missing")
		}
		return nil, nil
	}
	service := Service{CodexHarness: harness, Exec: harnessStreamExecutor{consume: consumer}}
	if err := service.streamIdentityHarness(context.Background(), container, target); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityHarnessConsumerFailureRetiresProducer(t *testing.T) {
	harness := t.TempDir()
	if err := os.WriteFile(filepath.Join(harness, "release"), make([]byte, 4<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	consumer := func(context.Context, io.Reader, string, ...string) ([]byte, error) {
		return nil, errors.New("synthetic private diagnostic")
	}
	service := Service{CodexHarness: harness, Exec: harnessStreamExecutor{consume: consumer}}
	started := time.Now()
	err := service.streamIdentityHarness(context.Background(), strings.Repeat("c", 64), "/run/task/harness")
	if err == nil || strings.Contains(err.Error(), "private diagnostic") {
		t.Fatal("transfer failure missing or leaked diagnostic")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("failed consumer left producer blocked")
	}
}

func TestIdentityHarnessProducerFailureAndCancellation(t *testing.T) {
	t.Run("producer failure", func(t *testing.T) {
		consumer := func(_ context.Context, in io.Reader, _ string, _ ...string) ([]byte, error) {
			_, err := io.Copy(io.Discard, in)
			return nil, err
		}
		service := Service{CodexHarness: filepath.Join(t.TempDir(), "missing"), Exec: harnessStreamExecutor{consume: consumer}}
		if err := service.streamIdentityHarness(context.Background(), strings.Repeat("c", 64), "/run/task/harness"); err == nil {
			t.Fatal("producer failure accepted")
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		harness := t.TempDir()
		if err := os.WriteFile(filepath.Join(harness, "release"), make([]byte, 4<<20), 0o600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		consumer := func(ctx context.Context, in io.Reader, _ string, _ ...string) ([]byte, error) {
			cancel()
			_, _ = io.Copy(io.Discard, in)
			return nil, ctx.Err()
		}
		service := Service{CodexHarness: harness, Exec: harnessStreamExecutor{consume: consumer}}
		started := time.Now()
		if err := service.streamIdentityHarness(ctx, strings.Repeat("c", 64), "/run/task/harness"); err == nil {
			t.Fatal("canceled transfer accepted")
		}
		if time.Since(started) > 2*time.Second {
			t.Fatal("canceled producer did not retire")
		}
	})
}
