package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestProjectCreationAdmission(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	owner := uint32(os.Geteuid())
	check := func(want string) {
		t.Helper()
		release, err := acquireProjectCreation(directory, owner)
		if release != nil {
			release()
		}
		var refusal *ProjectAdmissionError
		if !errors.As(err, &refusal) || refusal.Code() != want {
			t.Fatalf("admission = %v, want %s", err, want)
		}
	}
	release, err := acquireProjectCreation(directory, owner)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	if err := unix.Flock(int(writer.Fd()), unix.LOCK_EX|unix.LOCK_NB); !errors.Is(err, unix.EWOULDBLOCK) {
		t.Fatalf("maintenance crossed admitted creation: %v", err)
	}
	release()
	if err := unix.Flock(int(writer.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	check("project_maintenance")
	if err := unix.Flock(int(writer.Fd()), unix.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(directory, "p0123456789abcdef01234567.json")
	if err := os.WriteFile(marker, []byte("private interrupted state"), 0); err != nil {
		t.Fatal(err)
	}
	check("project_maintenance") // No access to private file content is required.
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o777); err != nil {
		t.Fatal(err)
	}
	check("project_admission_unavailable")
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if release, err := acquireProjectCreation(directory, owner+1); err == nil {
		release()
		t.Fatal("foreign-owned gate admitted")
	}
	link := filepath.Join(t.TempDir(), "gate")
	if err := os.Symlink(directory, link); err != nil {
		t.Fatal(err)
	}
	if release, err := acquireProjectCreation(link, owner); err == nil {
		release()
		t.Fatal("symlink gate admitted")
	}
	missing := filepath.Join(directory, "missing")
	if release, err := acquireProjectCreation(missing, owner); err == nil {
		release()
		t.Fatal("missing gate silently created")
	}
}
