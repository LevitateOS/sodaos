package build

import (
	"os"
	"slices"
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
	Require(t, workflow.Jobs["packaging"].If == "", "packaging must run for every source CI event")
	env := os.Environ()
	for job, key := range map[string]string{
		"go": "GO_RESULT", "browser": "BROWSER_RESULT", "rust": "RUST_RESULT", "arm64": "ARM64_RESULT", "packaging": "PACKAGING_RESULT",
	} {
		Require(t, slices.Contains(gate.Needs, job), "CI does not require %s", job)
		_, exists := workflow.Jobs[job]
		Require(t, exists, "required job %s is missing", job)
		Require(t, gate.Steps[0].Env[key] == "${{ needs."+job+".result }}", "CI must bind %s to the actual %s result", key, job)
		env = SetEnv(env, key, "success")
	}
	run := func(env []string) ProcResult {
		return Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 10 * time.Second}, "/bin/bash", "-eu", "-c", gate.Steps[0].Run)
	}
	Require(t, run(env).Code == 0, "successful source jobs should pass CI")
	for _, key := range []string{"GO_RESULT", "BROWSER_RESULT", "RUST_RESULT", "ARM64_RESULT", "PACKAGING_RESULT"} {
		for _, result := range []string{"failure", "cancelled", "skipped", ""} {
			t.Run(key+"/"+result, func(t *testing.T) {
				Check(t, run(SetEnv(env, key, result)).Code != 0, "CI admitted %s=%q", key, result)
			})
		}
	}
}
