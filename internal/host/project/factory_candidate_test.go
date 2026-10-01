package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func TestPrepareCandidateUsesOnlyApprovedSnapshot(t *testing.T) {
	const sourceID = "f111111111111111111111111"
	for _, refusal := range []string{"", "role", "not-ready", "digest", "credential", "writable", "symlink"} {
		t.Run(refusal, func(t *testing.T) {
			executor := &scriptedExec{}
			prepareScript(executor, "", "", domain.PrepareRunning)
			coderScript := executor.handle
			original := func(args []string, stdin []byte) ([]byte, error) {
				adapted := append([]string(nil), args...)
				for i := range adapted {
					adapted[i] = strings.ReplaceAll(adapted[i], domain.RoleReviewer, domain.RoleCoder)
				}
				out, err := coderScript(adapted, []byte(strings.ReplaceAll(string(stdin), domain.RoleReviewer, domain.RoleCoder)))
				return []byte(strings.ReplaceAll(string(out), domain.RoleCoder, domain.RoleReviewer)), err
			}
			approved := false
			executor.handle = func(args []string, stdin []byte) ([]byte, error) {
				joined := strings.Join(args, " ")
				switch {
				case strings.HasSuffix(joined, domain.FactoryHelper):
					var payload map[string]any
					if err := json.Unmarshal(stdin, &payload); err != nil {
						return nil, err
					}
					if payload["op"] == "inspect" && payload["id"] == sourceID {
						raw := []byte(strings.ReplaceAll(string(execHelperState(domain.PrepareReady, "")), domain.RoleCoder, domain.RoleReviewer))
						if refusal == "role" {
							raw = execHelperState(domain.PrepareReady, "")
						} else if refusal == "not-ready" {
							raw = []byte(strings.ReplaceAll(string(execHelperState(domain.PrepareRunning, "")), domain.RoleCoder, domain.RoleReviewer))
						}
						return raw, nil
					}
					if payload["op"] == "approve" {
						approved = true
					}
				case strings.Contains(joined, "/usr/bin/stat -c %u:%g:%a --"):
					return []byte(strings.Repeat("0:0:755\n", 4)), nil
				case strings.Contains(joined, "/usr/bin/stat -c %u:%g:%a:%h:%F"):
					if refusal == "writable" {
						return []byte("1001:1001:644:1:regular file\n"), nil
					}
					if refusal == "symlink" {
						return []byte("0:0:777:1:symbolic link\n"), nil
					}
					return []byte("0:0:644:1:regular file\n"), nil
				case strings.Contains(joined, "/usr/bin/head -c"):
					if strings.HasSuffix(joined, "/request.json") {
						credential := ""
						if refusal == "credential" {
							credential = "other-service"
						}
						return json.Marshal(map[string]string{"id": sourceID, "role": domain.RoleReviewer, "setup_digest": domain.SetupDigestOf(execTestSetup().Files), "source_commit": execTestCommit, "credential": credential})
					}
					if refusal == "digest" {
						return []byte("unapproved\n"), nil
					}
					return []byte("true\n"), nil
				case strings.Contains(joined, "/usr/bin/ls -1A --"):
					return []byte("check.sh\nsetup.sh\nsource.bundle\n"), nil
				}
				return original(args, stdin)
			}
			runtime := &Runtime{Exec: executor}
			prep := execTestPreparation()
			prep.Role = domain.RoleReviewer
			state, err := runtime.PrepareCandidate(context.Background(), domain.FactoryCandidate{Preparation: prep, SourcePreparation: sourceID, Bundle: []byte("candidate-bundle")})
			if refusal == "" {
				if err != nil || state.Phase != domain.PrepareRunning || !approved {
					t.Fatalf("candidate did not enter existing preparation: state=%+v err=%v", state, err)
				}
			} else if err == nil || approved {
				t.Fatalf("unsafe source reached preparation: approved=%v err=%v", approved, err)
			}
		})
	}
}

func TestFactoryCandidateInspectionRequiresSettledRecordedRun(t *testing.T) {
	f := openTestFactory(t, "")
	run := factoryTestRun()
	in := domain.FactoryCandidateInspect{Project: run.Project, ID: run.ID}
	if _, err := f.InspectCandidate(context.Background(), in); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unrecorded run: %v", err)
	}
	if err := f.storeReceipt(factoryReceipt{Run: run, Phase: domain.FactoryRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.InspectCandidate(context.Background(), in); err == nil {
		t.Fatal("unsettled run inspection accepted")
	}
	recorded := strings.Repeat("c", 64)
	if err := f.storeReceipt(factoryReceipt{Run: run, Phase: domain.FactoryCompleted, Binding: &identity.Binding{Project: recorded, Kind: identity.Factory, ID: run.ID}}); err != nil {
		t.Fatal(err)
	}
	fake := f.terminal.Exec.(*factoryFakeExec)
	fake.run = func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		return []byte(strings.ReplaceAll(string(execInspection(true)), execTestProject, run.Project)), nil
	}
	if _, err := f.InspectCandidate(context.Background(), in); !errors.Is(err, identity.ErrStale) {
		t.Fatalf("replacement container accepted: %v", err)
	}
}

func TestCandidateInspectGitDetectsReviewerChangesWithoutAgentConfig(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("/usr/bin/git", append([]string{"-C", source}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	git("init", "--template=")
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("code.txt", "initial\n")
	write(".gitignore", "node_modules/\n.artifacts/\n")
	git("add", "code.txt", ".gitignore")
	git("-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "initial")
	head := git("rev-parse", "HEAD")
	inspect := func(wantHead, wantState string) {
		t.Helper()
		cmd := exec.Command("/usr/bin/sh", "-c", candidateInspectScript, "soda-candidate", source)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "TMPDIR=" + dir, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_REPLACE_OBJECTS=1", "GIT_OPTIONAL_LOCKS=0"}
		out, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != wantHead+" "+wantState {
			t.Fatalf("inspection got %q, %v; want %s %s", out, err, wantHead, wantState)
		}
	}
	inspect(head, "clean")
	if err := os.Mkdir(filepath.Join(source, ".soda-home"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(".soda-home/output", "run output")
	inspect(head, "clean")
	for _, name := range []string{"node_modules", ".artifacts"} {
		if err := os.Mkdir(filepath.Join(source, name), 0o700); err != nil {
			t.Fatal(err)
		}
		write(name+"/output", "disposable check output")
	}
	write("node_modules/.gitignore", "*\n")
	inspect(head, "clean")
	if err := os.Mkdir(filepath.Join(source, "reviewer-output"), 0o700); err != nil {
		t.Fatal(err)
	}
	write("reviewer-output/.gitignore", "*\n")
	write("reviewer-output/hidden", "unapproved output")
	inspect(head, "dirty")
	if err := os.RemoveAll(filepath.Join(source, "reviewer-output")); err != nil {
		t.Fatal(err)
	}
	write(".gitignore", "*\n")
	inspect(head, "dirty")
	write(".gitignore", "node_modules/\n.artifacts/\n")
	write(".git/info-exclude-test", "local-output\n")
	git("config", "core.excludesFile", filepath.Join(source, ".git/info-exclude-test"))
	write("local-output", "must remain visible")
	inspect(head, "dirty")
	if err := os.Remove(filepath.Join(source, "local-output")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, ".git/info"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(".git/info/exclude", "info-output\n")
	write("info-output", "must also remain visible")
	inspect(head, "dirty")
	if err := os.Remove(filepath.Join(source, "info-output")); err != nil {
		t.Fatal(err)
	}
	git("config", "core.fsmonitor", "/does-not-exist")
	git("config", "diff.external", "/does-not-exist")
	write("code.txt", "reviewer edit\n")
	inspect(head, "dirty")
	git("-c", "core.fsmonitor=false", "add", "code.txt")
	write("code.txt", "initial\n")
	inspect(head, "dirty")
	git("-c", "core.fsmonitor=false", "reset", "--hard", "HEAD")
	write("untracked.txt", "reviewer addition\n")
	inspect(head, "dirty")
	git("-c", "core.fsmonitor=false", "add", "untracked.txt")
	git("-c", "core.fsmonitor=false", "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "reviewer commit")
	inspect(git("rev-parse", "HEAD"), "clean")
}
