// Port of test_u08_state.py: snapshot entry() contracts via a thin driver.
//
// project-state.py is still Python, so a minimal driver imports it and
// reports observations as JSON; every assertion below lives in Go.
package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// moduleDriver imports the module at argv[1] and calls one function with the
// remaining argv, printing {"ok": value} or {"error": type, "mro": [...]}.
const moduleDriver = `
import importlib.util, json, sys
spec = importlib.util.spec_from_file_location('subject', sys.argv[1])
subject = importlib.util.module_from_spec(spec)
spec.loader.exec_module(subject)
call, args = sys.argv[2], sys.argv[3:]
try:
    if call == 'entry':
        value = subject.entry(args[0], contents=(args[1] == '1'))
    else:
        value = subject.observe(args[0])
    print(json.dumps({'ok': value}))
except Exception as failure:
    print(json.dumps({'error': type(failure).__name__, 'mro': [c.__name__ for c in type(failure).__mro__]}))
`

// driveModule calls call in the Python module and returns the decoded value
// plus the raised exception's MRO names (nil when the call succeeded).
func driveModule(t *testing.T, module string, call string, args ...string) (any, []string) {
	t.Helper()
	argv := append([]string{"-c", moduleDriver, module, call}, args...)
	result := Run(t, RunOpt{}, Python3(t), argv...)
	Require(t, result.Code == 0, "driver failed: %s", result.Stderr)
	var decoded struct {
		Ok    any      `json:"ok"`
		Error string   `json:"error"`
		Mro   []string `json:"mro"`
	}
	Require(t, json.Unmarshal([]byte(result.Stdout), &decoded) == nil, "parse driver output %q", result.Stdout)
	if decoded.Error != "" {
		return nil, decoded.Mro
	}
	return decoded.Ok, nil
}

func raisedAs(mro []string, names ...string) bool {
	for _, entry := range mro {
		for _, name := range names {
			if entry == name {
				return true
			}
		}
	}
	return false
}

func TestU08PrivateMetadataDoesNotExportHashOrContent(t *testing.T) {
	dir := TempDir(t)
	p := filepath.Join(dir, "private-input")
	WriteFile(t, p, []byte("synthetic-sensitive-input"), 0o600)
	value, mro := driveModule(t, filepath.Join(RepoRoot, "tests/installed/project-state.py"), "entry", p, "0")
	Require(t, mro == nil, "entry raised %v", mro)
	entry := value.(map[string]any)
	_, present := entry["sha256"]
	Check(t, !present, "sha256 exported: %v", entry)
	Check(t, entry["mode"] == float64(0o600), "mode = %v", entry["mode"])
	info, err := os.Stat(p)
	Require(t, err == nil, "stat: %v", err)
	Check(t, entry["size"] == float64(info.Size()), "size = %v", entry["size"])
}

func TestU08RequiredMissingFileFails(t *testing.T) {
	dir := TempDir(t)
	_, mro := driveModule(t, filepath.Join(RepoRoot, "tests/installed/project-state.py"), "entry", filepath.Join(dir, "missing"), "1")
	Check(t, raisedAs(mro, "FileNotFoundError"), "mro = %v", mro)
}

func TestU08PublicHashChangesAndSymlinkIsNotFollowed(t *testing.T) {
	dir := TempDir(t)
	p := filepath.Join(dir, "public")
	module := filepath.Join(RepoRoot, "tests/installed/project-state.py")
	WriteFile(t, p, []byte("before"), 0o644)
	before, mro := driveModule(t, module, "entry", p, "1")
	Require(t, mro == nil, "entry raised %v", mro)
	WriteFile(t, p, []byte("after"), 0o644)
	after, mro := driveModule(t, module, "entry", p, "1")
	Require(t, mro == nil, "entry raised %v", mro)
	Check(t, before.(map[string]any)["sha256"] != after.(map[string]any)["sha256"], "hash unchanged")
	link := filepath.Join(dir, "link")
	Require(t, os.Symlink(p, link) == nil, "symlink")
	value, mro := driveModule(t, module, "entry", link, "1")
	Require(t, mro == nil, "entry raised %v", mro)
	entry := value.(map[string]any)
	Check(t, entry["link"] == p, "link = %v", entry["link"])
	_, present := entry["sha256"]
	Check(t, !present, "sha256 exported for symlink: %v", entry)
}
