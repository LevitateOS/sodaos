package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Prepare creates fresh, private task inputs after the run intent is durable.
// The input object contains role/revisions/limits; no host credential paths.
func (w *Runtime) Prepare(r factory.Run, task any, bundle []byte) error {
	dir := filepath.Join(w.Config.Root, r.ID)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return err
	}
	input := filepath.Join(dir, "input")
	if err := os.Mkdir(input, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	files := map[string][]byte{"task.json": data, "source.bundle": bundle, "result-schema.json": []byte(factory.ResultSchema)}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(input, name), content, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// Launch uses the external OCI boundary. The CLI's successful exit alone is
// insufficient: it must also produce a bounded, valid structured result.
func (w *Runtime) Launch(ctx context.Context, r factory.Run) (factory.Result, error) {
	id := w.resource(r, "workspace").ID
	out, err := w.Exec.Run(ctx, nil, "podman", "exec", id, "/opt/codex/bin/codex", "login", "status")
	if err != nil || !strings.Contains(string(out), "Logged in using ChatGPT") {
		return factory.Result{}, errors.New("enrolled Codex subscription needs authentication")
	}
	args := []string{"exec", "--workdir", "/workspace/repo", id, "/opt/codex/bin/codex", "--ask-for-approval", "never", "exec", "--ignore-user-config", "--ephemeral", "--json", "--sandbox", "danger-full-access", "--config", `sqlite_home="/workspace/.codex-state"`, "--config", `log_dir="/workspace/.codex-log"`, "--color", "never", "--model", w.Config.Model, "--output-schema", "/input/result-schema.json", "--output-last-message", "/workspace/result.json", "-"}
	prompt := []byte("Read the immutable task in /input/task.json. Perform only its assigned role against the checkout and source revisions. Use tools to edit and test implementation/repair work and commit the changes locally. Review work must inspect and test the assigned candidate without creating implementation changes. Do not publish, approve, merge, change policy, inspect credentials or access host services. Your structured result must report the actual HEAD commit, a concise summary, findings, and review_passed only for the review role. Report blocked if the task is ambiguous or cannot be completed. A statement of success is insufficient: verify the actual files and tests with tools.")
	if _, err := w.Exec.Run(ctx, prompt, "podman", args...); err != nil {
		return factory.Result{}, err
	}
	out, err = w.Exec.Run(ctx, nil, "podman", "exec", id, "cat", "/workspace/result.json")
	if err != nil {
		return factory.Result{}, errors.New("harness produced no result")
	}
	var result factory.Result
	if err := strictjson.Decode(bytes.NewReader(out), &result); err != nil {
		return result, errors.New("invalid structured harness result")
	}
	return result, result.Validate()
}
