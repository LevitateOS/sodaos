package runners

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestEveryReadAndMutationUsesTheCrossProcessLock(t *testing.T) {
	root := t.TempDir()
	native := &Native{RootPath: filepath.Join(root, "state"), LockPath: filepath.Join(root, "lock")}
	lock, err := native.lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	other := &Native{RootPath: native.RootPath, LockPath: native.LockPath}
	for _, action := range []string{"list", "create", "start", "stop", "restart", "remove"} {
		t.Run(action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			var err error
			switch action {
			case "list":
				_, err = other.List(ctx)
			case "create":
				err = other.Create(ctx, githubRequest())
			case "start":
				err = other.Start(ctx, "one")
			case "stop":
				err = other.Stop(ctx, "one")
			case "restart":
				err = other.Restart(ctx, "one")
			case "remove":
				err = other.Remove(ctx, "one")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("%s did not wait on existing native lock: %v", action, err)
			}
		})
	}
}
