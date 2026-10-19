package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestCIRequiresDockerPackagingAndRefusesIncompleteJobs(t *testing.T) {
	var workflow struct {
		Jobs map[string]struct {
			Needs []string
			If    string
			Steps []struct {
				Run string
				Env map[string]string
			}
		}
	}
	Require(t, yaml.Unmarshal([]byte(ReadFile(t, ".github/workflows/ci.yml")), &workflow) == nil, "decode CI workflow")
	gate := workflow.Jobs["ci"]
	Require(t, gate.If == "${{ always() }}", "CI must evaluate failed/cancelled/skipped dependencies")
	Require(t, len(gate.Steps) == 1 && gate.Steps[0].Run != "", "CI admission command required")
	Require(t, slices.Contains(gate.Needs, "reuse"), "CI must require the reuse decision")
	env := os.Environ()
	env = SetEnv(env, "REUSE_RESULT", "success")
	env = SetEnv(env, "REUSED_CHECKS", "false")
	for job, key := range map[string]string{
		"go": "GO_RESULT", "browser": "BROWSER_RESULT", "rust": "RUST_RESULT", "packaging": "PACKAGING_RESULT",
	} {
		Require(t, slices.Contains(gate.Needs, job), "CI does not require %s", job)
		_, exists := workflow.Jobs[job]
		Require(t, exists, "required job %s is missing", job)
		Require(t, slices.Contains(workflow.Jobs[job].Needs, "reuse"), "%s must wait for the reuse decision", job)
		Require(t, workflow.Jobs[job].If == "${{ needs.reuse.outputs.reused != 'true' }}", "%s may skip only verified reused checks", job)
		Require(t, gate.Steps[0].Env[key] == "${{ needs."+job+".result }}", "CI must bind %s to the actual %s result", key, job)
		env = SetEnv(env, key, "success")
	}
	run := func(env []string) ProcResult {
		return Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 10 * time.Second}, "/bin/bash", "-eu", "-c", gate.Steps[0].Run)
	}
	Require(t, run(env).Code == 0, "successful source jobs should pass CI")
	for _, reused := range []string{"", "invalid"} {
		Check(t, run(SetEnv(append([]string(nil), env...), "REUSED_CHECKS", reused)).Code != 0, "CI admitted invalid reuse decision")
	}
	reused := SetEnv(append([]string(nil), env...), "REUSED_CHECKS", "true")
	for _, key := range []string{"GO_RESULT", "BROWSER_RESULT", "RUST_RESULT", "PACKAGING_RESULT"} {
		reused = SetEnv(reused, key, "skipped")
	}
	Require(t, run(reused).Code == 0, "verified reuse should admit deliberately skipped source jobs")
	Require(t, run(SetEnv(append([]string(nil), reused...), "REUSE_RESULT", "failure")).Code != 0, "failed reuse cannot authorize CI")
	Require(t, run(SetEnv(append([]string(nil), reused...), "REUSED_CHECKS", "false")).Code != 0, "unverified skipped jobs cannot authorize CI")
	for _, key := range []string{"GO_RESULT", "BROWSER_RESULT", "RUST_RESULT", "PACKAGING_RESULT"} {
		for _, result := range []string{"failure", "cancelled", "skipped", ""} {
			t.Run(key+"/"+result, func(t *testing.T) {
				Check(t, run(SetEnv(env, key, result)).Code != 0, "CI admitted %s=%q", key, result)
			})
		}
	}
}

func TestCIReuseRequiresTheLatestSuccessfulExactTagRun(t *testing.T) {
	sha := strings.Repeat("a", 40)
	valid := map[string]any{"id": 23, "head_sha": sha, "head_branch": "ci/soda-main-" + sha, "event": "push", "path": ".github/workflows/ci.yml", "status": "completed", "conclusion": "success"}
	cases := []struct {
		name, field string
		value       any
	}{
		{"passed", "", nil}, {"wrong-sha", "head_sha", strings.Repeat("b", 40)},
		{"wrong-tag", "head_branch", "topic"}, {"pr", "event", "pull_request"},
		{"wrong-workflow", "path", ".github/workflows/other.yml"},
		{"pending", "status", "in_progress"}, {"failed", "conclusion", "failure"},
		{"cancelled", "conclusion", "cancelled"}, {"skipped", "conclusion", "skipped"},
		{"missing-id", "id", nil}, {"invalid-id", "id", "23"},
		{"missing-run", "", nil}, {"bad-json", "", nil}, {"api-error", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			run := map[string]any{}
			for key, value := range valid {
				run[key] = value
			}
			if tc.field != "" {
				run[tc.field] = tc.value
			}
			body, err := json.Marshal(map[string]any{"workflow_runs": []any{run}})
			Require(t, err == nil, "marshal CI fixture")
			if tc.name == "missing-run" {
				body = []byte(`{"workflow_runs":[]}`)
			} else if tc.name == "bad-json" {
				body = []byte("{")
			}
			reply := filepath.Join(dir, "reply.json")
			Require(t, os.WriteFile(reply, body, 0o600) == nil, "write CI fixture")
			Require(t, os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\ncase \"$2\" in */attempts/*) date -u +%Y-%m-%dT%H:%M:%SZ; exit 0 ;; esac\nif test \"${CI_API_FAILED:-}\" = true; then exit 1; fi\ncat \"$CI_REPLY\"\n"), 0o700) == nil, "write gh fixture")
			output := filepath.Join(dir, "output")
			env := os.Environ()
			for key, value := range map[string]string{"PATH": dir + ":" + os.Getenv("PATH"), "TMPDIR": dir, "GITHUB_RUN_ID": "42", "GITHUB_EVENT_NAME": "push", "GITHUB_REF": "refs/heads/main", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_SHA": sha, "GITHUB_REPOSITORY": "example/sodaos", "GITHUB_OUTPUT": output, "CI_REPLY": reply} {
				env = SetEnv(env, key, value)
			}
			if tc.name == "api-error" {
				env = SetEnv(env, "CI_API_FAILED", "true")
			}
			check := func(expected string) {
				t.Helper()
				Require(t, os.WriteFile(output, nil, 0o600) == nil, "reset output")
				result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 10 * time.Second}, "/bin/bash", "scripts/ci-reuse.sh")
				Require(t, result.Code == 0, "reuse decision failed: %s", result.Stderr)
				got, err := os.ReadFile(output)
				Require(t, err == nil && string(got) == "reused="+expected+"\n", "unexpected reuse result: %s", got)
			}
			expected := "false"
			if tc.name == "passed" {
				expected = "true"
			}
			check(expected)
			if tc.name == "passed" {
				for key, value := range map[string]string{"GITHUB_EVENT_NAME": "pull_request", "GITHUB_REF": "refs/tags/ci/soda-main-" + sha, "GITHUB_RUN_ATTEMPT": "2"} {
					original := append([]string(nil), env...)
					env = SetEnv(env, key, value)
					check("false")
					env = original
				}
			}
		})
	}
}

func TestPushCommandEnforcesFiveMinuteMaximum(t *testing.T) {
	dir := t.TempDir()
	WriteFile(t, filepath.Join(dir, "git"), []byte("#!/bin/sh\nsleep 60\n"), 0o700)
	env := SetEnv(os.Environ(), "PATH", dir+":"+os.Getenv("PATH"))
	env = SetEnv(env, "SODA_PUSH_BOUNDED", "")
	env = SetEnv(env, "SODA_PUSH_TIMEOUT_SECONDS", "2")
	started := time.Now()
	result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 5 * time.Second}, "/bin/bash", "scripts/push.sh")
	Check(t, result.Code != 0 && time.Since(started) < 3*time.Second && strings.Contains(result.Stderr, "Push budget expired"), "publisher failed to bound network wait: %+v", result)
	for _, budget := range []string{"0", "301", "invalid"} {
		result := Run(t, RunOpt{Cwd: RepoRoot, Env: SetEnv(env, "SODA_PUSH_TIMEOUT_SECONDS", budget), Timeout: 5 * time.Second}, "/bin/bash", "scripts/push.sh")
		Check(t, result.Code != 0, "publisher admitted budget %q", budget)
	}
}
