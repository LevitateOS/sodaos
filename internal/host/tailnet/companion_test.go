package tailnet

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/levitateos/sodaos/internal/tailnet"
)

func TestCompanionResolverUsesActualInodeRatherThanGeneratedMetadata(t *testing.T) {
	run := fileRun()
	run.Resolver = filepath.Join(t.TempDir(), "resolver")
	if e := os.WriteFile(run.Resolver, []byte("nameserver 10.89.0.1\n"), 0o600); e != nil {
		t.Fatal(e)
	}
	other := filepath.Join(t.TempDir(), "resolver")
	if e := os.WriteFile(other, []byte("nameserver 10.89.0.1\n"), 0o600); e != nil {
		t.Fatal(e)
	}
	for _, same := range []bool{true, false} {
		stat := func(path string) (os.FileInfo, error) {
			if path == "/proc/99/root/etc/resolv.conf" {
				path = run.Resolver
				if !same {
					path = other
				}
			}
			return os.Stat(path)
		}
		e := companionResolver(run, 99, stat)
		if (e == nil) != same {
			t.Fatal("resolver identity", same, e)
		}
	}
}

func TestCompanionRecordRequiresImmutableCIDRecipeAndRunningIncarnation(t *testing.T) {
	run := fileRun()
	run.UID = 300000
	run.GID = 300000
	run.Resolver = "/var/lib/containers/storage/overlay-containers/" + run.Target.Container + "/userdata/resolv.conf"
	image := "sha256:" + strings.Repeat("d", 64)
	id := strings.Repeat("e", 64)
	args, e := companionCreateArgs(run, image)
	if e != nil {
		t.Fatal(e)
	}
	original := companion{ID: id, Image: image, Command: append([]string{"/usr/bin/podman"}, args...), Running: true, PID: 99, Started: "2026-09-12T12:00:00Z"}
	if e = validateCompanionRecord(original, run, image, id); e != nil {
		t.Fatal(e)
	}
	// A new release default is not authority to adopt/replace an existing run's
	// companion. Native update qualification must cover coordinated run turnover.
	if e = validateCompanionRecord(original, run, "sha256:"+strings.Repeat("f", 64), id); e == nil {
		t.Fatal("changed release default adopted an existing companion")
	}
	// Before first native start there is no daemon PID or generated resolver metadata.
	created := original
	created.Running = false
	created.PID = 0
	created.Started = ""
	if e = validateCompanionRecord(created, run, image, id); e != nil {
		t.Fatal("new stopped container refused", e)
	}
	for _, kind := range []string{"cid", "image", "namespace", "secret-env", "missing-command", "foreign-exec", "many-execs", "pid", "started"} {
		t.Run(kind, func(t *testing.T) {
			changed := original
			changed.Command = append([]string{}, original.Command...)
			switch kind {
			case "cid":
				changed.ID = strings.Repeat("f", 64)
			case "image":
				changed.Image = "sha256:" + strings.Repeat("f", 64)
			case "namespace":
				for i, arg := range changed.Command {
					if strings.HasPrefix(arg, "--network=") {
						changed.Command[i] = "--network=host"
					}
				}
			case "secret-env":
				changed.Command = append(changed.Command, "--env=TS_AUTHKEY=synthetic-secret")
			case "missing-command":
				changed.Command = nil
			case "foreign-exec":
				changed.Execs = []string{"unknown"}
			case "many-execs":
				for range 17 {
					changed.Execs = append(changed.Execs, id)
				}
			case "pid":
				changed.PID = 1
			case "started":
				changed.Started = "0001-01-01T00:00:00Z"
			}
			if e := validateCompanionRecord(changed, run, image, id); e == nil {
				t.Fatal("changed companion admitted")
			}
		})
	}
}

func TestPreparationStagesPreserveTypedCauses(t *testing.T) {
	for _, tc := range []struct {
		stage string
		cause error
	}{
		{"project runtime not ready", domain.ErrUnavailable},
		{"Tailnet policy unconfirmed", domain.ErrConflict},
		{"companion startup unconfirmed", domain.ErrUnconfirmed},
		{"companion status unavailable", domain.ErrUnavailable},
		{"enrollment unconfirmed", domain.ErrUnconfirmed},
	} {
		if e := preparationError(tc.stage, tc.cause); !errors.Is(e, tc.cause) || !strings.Contains(e.Error(), tc.stage) {
			t.Fatal("stage lost its cause", tc.stage, e)
		}
	}
}
