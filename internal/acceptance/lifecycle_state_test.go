package acceptance

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func lifecycleFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunLifecycleCompare(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	root := lifecycleFixture(t)
	before := `{"stable": {"soda": {"users": []}, "projects": {}}, "boot_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`
	after := `{"stable": {"projects": {}, "soda": {"users": []}}, "boot_id": "ffffffff-ffff-ffff-ffff-ffffffffffff"}`
	if err := os.WriteFile(filepath.Join(root, "before.json"), []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "after.json"), []byte(after), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := RunLifecycleState([]string{root, "compare", "before", "after"}, &stdout); err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	want := "All declared project/account/Git/tool/workload/database state matches; boot identity compared separately.\n"
	if stdout.String() != want {
		t.Errorf("compare output = %q", stdout.String())
	}
	if err := os.WriteFile(filepath.Join(root, "after.json"), []byte(`{"stable": {"soda": 1}, "boot_id": "x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RunLifecycleState([]string{root, "compare", "before", "after"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("differing stable accepted")
	}
	if !strings.HasPrefix(err.Error(), "Lifecycle snapshot/comparison incomplete: ") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Stable state differs; inspect private snapshots, do not repair by reset") {
		t.Errorf("reportable detail missing: %q", err.Error())
	}
}

func TestSnapshotEntries(t *testing.T) {
	raw := []map[string]json.RawMessage{
		{"environmentID": json.RawMessage(`"p111111111111111111111111"`), "login": json.RawMessage(`"u08-alice-8417"`)},
		{"environmentID": json.RawMessage(`"p222222222222222222222222"`), "login": json.RawMessage(`"u08-bob-8417"`)},
	}
	entries, err := snapshotEntries(raw, "p333333333333333333333333")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || !entries[0].workloads || entries[1].workloads || !entries[2].workloads {
		t.Errorf("entries = %+v", entries)
	}
	if _, err := snapshotEntries(raw, "p111111111111111111111111"); err == nil {
		t.Error("duplicate identifier accepted")
	}
	if _, err := snapshotEntries(raw[:1], "p333333333333333333333333"); err == nil {
		t.Error("short bindings accepted")
	}
	bad := []map[string]json.RawMessage{{"login": json.RawMessage(`"u08-alice-8417"`)}, raw[1]}
	if _, err := snapshotEntries(bad, "p333333333333333333333333"); err == nil {
		t.Error("missing environmentID accepted")
	}
}

func TestProjectSnapshotIP(t *testing.T) {
	ip, err := projectSnapshotIP([]byte(`{"soda-projects": {"IPAddress": "10.89.0.7"}}`))
	if err != nil || ip != "10.89.0.7" {
		t.Errorf("ip = %q %v", ip, err)
	}
	for _, bad := range []string{
		`{"other": {"IPAddress": "10.89.0.7"}}`,
		`{"soda-projects": {"IPAddress": "10.90.0.7"}}`,
		`{"soda-projects": {"IPAddress": "not-an-ip"}}`,
		`not json`,
		`[]`,
	} {
		if _, err := projectSnapshotIP([]byte(bad)); err == nil {
			t.Errorf("network accepted: %s", bad)
		}
	}
}

func TestWriteSnapshot(t *testing.T) {
	root := lifecycleFixture(t)
	stable := map[string]any{"soda": map[string]any{"b": 1, "a": 1}, "projects": map[string]any{}}
	if err := writeSnapshot(root, "snap", stable, "boot"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "snap.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Compact separators are the documented Go difference from Python's
	// json.dump; key order stays sorted.
	want := `{"boot_id":"boot","stable":{"projects":{},"soda":{"a":1,"b":1}}}`
	if string(data) != want {
		t.Errorf("snapshot bytes = %s", data)
	}
	st, err := os.Stat(filepath.Join(root, "snap.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("snapshot mode = %o", st.Mode().Perm())
	}
	if err := writeSnapshot(root, "snap", stable, "boot"); err == nil {
		t.Error("second snapshot write accepted")
	}
}

func TestLifecycleRun(t *testing.T) {
	out, err := lifecycleRun("", []string{"sh", "-c", "printf out"}, nil)
	if err != nil || string(out) != "out" {
		t.Errorf("run = %q %v", out, err)
	}
	_, err = lifecycleRun("", []string{"sh", "-c", "echo 'Project snapshot failed: snap RuntimeError boom' >&2; exit 3"}, nil)
	if err == nil || !strings.Contains(err.Error(), "Snapshot command failed; no empty substitution: Project snapshot failed: snap RuntimeError boom") {
		t.Errorf("prefixed detail = %v", err)
	}
	_, err = lifecycleRun("", []string{"sh", "-c", "echo noise >&2; exit 3"}, nil)
	if err == nil || !strings.Contains(err.Error(), "operator observation failed") || strings.Contains(err.Error(), "noise") {
		t.Errorf("generic detail = %v", err)
	}
}

// stubDashboardDB builds a real dashboard database with seeded rows and
// returns its bytes for the VM stub to serve.
func stubDashboardDB(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dashboard.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, table := range []string{"users", "keys", "projects", "memberships"} {
		if _, err := db.Exec("create table " + table + " (name text, scope integer, note text)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`insert into users values ('bob', 2, null), ('alice', 1, 'root')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into keys values ('k2', 7, null)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// stubTestVM installs a fake target/debug/soda-test-vm that emulates the VM session.
func stubTestVM(t *testing.T, repo string, dbRaw []byte) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" != ssh ]; then exit 9; fi\n" +
		"case \"$2\" in\n" +
		"'cat /etc/soda/dashboard.json') printf '{\"database\": \"/var/lib/soda/dashboard.db\"}';;\n" +
		"'cat '\"'\"'/var/lib/soda/dashboard.db'\"'\"'') printf '" + base64.StdEncoding.EncodeToString(dbRaw) + "' | base64 -d;;\n" +
		"*'{{.Id}}'*) printf 'abc123 image456 soda-p000';;\n" +
		"*'Networks'*) printf '{\"soda-projects\": {\"IPAddress\": \"10.89.0.5\"}}';;\n" +
		"*'podman exec'*'project-state'*) body=$(cat); case \"$body\" in *'SNAPSHOT-PAYLOAD-MARKER'*) printf '{\"people\": {\"u\": 1}, \"files\": {\"f\": 1}}';; *) exit 7;; esac;;\n" +
		"*'boot_id'*) printf 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee';;\n" +
		"*) exit 6;;\n" +
		"esac\n"
	path := filepath.Join(repo, "target/debug/soda-test-vm")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRunSnapshotAt(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	repo := t.TempDir()
	dbRaw := stubDashboardDB(t)
	stubTestVM(t, repo, dbRaw)
	bindings := `[{"environmentID": "p111111111111111111111111", "login": "u08-alice-8417"}, {"environmentID": "p222222222222222222222222", "login": "u08-bob-8417"}]`
	bindPath := filepath.Join(repo, ".artifacts/test-vm/u08-8417a90/observed-bindings.json")
	if err := os.MkdirAll(filepath.Dir(bindPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bindPath, []byte(bindings), 0o644); err != nil {
		t.Fatal(err)
	}
	probePath := filepath.Join(repo, "target/debug/soda-acceptance-remote")
	if err := os.WriteFile(probePath, []byte("SNAPSHOT-PAYLOAD-MARKER"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := lifecycleFixture(t)
	if err := os.WriteFile(filepath.Join(root, "target.json"), []byte(`{"environment_id": "p333333333333333333333333"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := runSnapshotAt(root, "first", repo, &stdout); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Complete bounded snapshot captured for three projects") {
		t.Errorf("snapshot output = %q", stdout.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "first.json"))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		BootID string `json:"boot_id"`
		Stable struct {
			Soda     map[string]any `json:"soda"`
			Projects map[string]struct {
				Identity string         `json:"identity"`
				State    map[string]any `json:"state"`
			} `json:"projects"`
		} `json:"stable"`
	}
	if json.Unmarshal(data, &snapshot) != nil {
		t.Fatalf("snapshot invalid: %s", data)
	}
	if len(snapshot.Stable.Projects) != 3 || snapshot.BootID == "" {
		t.Errorf("snapshot = %s", data)
	}
	for id, project := range snapshot.Stable.Projects {
		if project.Identity == "" || len(project.State) == 0 {
			t.Errorf("project %s = %+v", id, project)
		}
	}
	users, _ := snapshot.Stable.Soda["users"].([]any)
	if len(users) != 2 {
		t.Fatalf("soda users = %v", snapshot.Stable.Soda["users"])
	}
	// Canonical JSON order: alice's row sorts before bob's.
	first, _ := users[0].([]any)
	if len(first) != 3 || first[0] != "alice" || first[1] != 1.0 || first[2] != "root" {
		t.Errorf("first user = %v", first)
	}
	keys, _ := snapshot.Stable.Soda["keys"].([]any)
	if len(keys) != 1 {
		t.Errorf("soda keys = %v", snapshot.Stable.Soda["keys"])
	}
	for _, table := range []string{"projects", "memberships"} {
		rows, _ := snapshot.Stable.Soda[table].([]any)
		if rows == nil || len(rows) != 0 {
			t.Errorf("soda %s = %v", table, snapshot.Stable.Soda[table])
		}
	}
}

func TestDumpHostSoda(t *testing.T) {
	data, err := dumpHostSoda(stubDashboardDB(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Fatalf("tables = %v", data)
	}
	for _, bad := range [][]byte{
		[]byte("not a database"),
		[]byte("SQLite format 3\x00short"),
	} {
		if _, err := dumpHostSoda(bad); err == nil {
			t.Errorf("corrupt database accepted: %q", bad)
		}
	}
}

func TestRemotePayloadMissing(t *testing.T) {
	repo := t.TempDir()
	if _, err := remotePayload(repo); err == nil || !strings.Contains(err.Error(), "cargo build -p soda-acceptance --bin soda-acceptance-remote") {
		t.Errorf("missing payload = %v", err)
	}
}

func TestShQuote(t *testing.T) {
	if got := shQuote("/var/lib/soda/dashboard.db"); got != "'/var/lib/soda/dashboard.db'" {
		t.Errorf("quote = %s", got)
	}
	if got := shQuote("/odd'path/x.db"); got != `'/odd'\''path/x.db'` {
		t.Errorf("quote = %s", got)
	}
}

func TestExecProjectSnapshotRejectsBadIP(t *testing.T) {
	for _, ip := range []string{"10.90.0.5", "not-an-ip", "10.89.0.5; id", ""} {
		if _, _, err := execProjectSnapshot("vm", "repo", nil, snapshotEntry{}, "id", ip); err == nil {
			t.Errorf("ip accepted: %q", ip)
		}
	}
}

func TestRunLifecycleValidation(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	root := lifecycleFixture(t)
	err := RunLifecycleState([]string{root, "compare", "only-one"}, &bytes.Buffer{})
	if err == nil {
		t.Error("short compare accepted")
	} else if !strings.HasPrefix(err.Error(), "Lifecycle snapshot/comparison incomplete: ") {
		t.Errorf("failure shape = %q", err.Error())
	}
	for _, args := range [][]string{
		{root, "bogus", "label"},
		{root, "snapshot"},
		{root, "snapshot", "one", "two"},
		{root, "compare", "Bad-Label", "other"},
		{root[:len(root)-1] + string(os.PathSeparator) + "missing", "compare", "a", "b"},
	} {
		if err := RunLifecycleState(args, &bytes.Buffer{}); err == nil {
			t.Errorf("args accepted: %v", args)
		}
	}
	if err := RunLifecycleState([]string{root}, &bytes.Buffer{}); err == nil {
		t.Error("short argv accepted")
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "wrong")
	if err := RunLifecycleState([]string{root, "compare", "a", "b"}, &bytes.Buffer{}); err == nil {
		t.Error("wrong env accepted")
	}
}

func TestRepoRoot(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("repo root %q holds no go.mod", root)
	}
}
