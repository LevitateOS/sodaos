package acceptance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAccessTarget(t *testing.T) {
	dir := t.TempDir()
	target, err := loadAccessTarget(dir)
	if err != nil {
		t.Fatal(err)
	}
	if target.ip != "10.89.0.2" || target.mode != "project namespace (nested host mode)" || !target.modeSet {
		t.Errorf("default target = %+v", target)
	}
	path := filepath.Join(dir, "target.json")
	if err := os.WriteFile(path, []byte(`{"ip": "10.89.0.9", "network_mode": "bridge"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err = loadAccessTarget(dir)
	if err != nil || target.ip != "10.89.0.9" || target.mode != "bridge" || !target.modeSet {
		t.Errorf("file target = %+v %v", target, err)
	}
	for _, body := range []string{
		`{"ip": "10.90.0.2", "network_mode": "bridge"}`,
		`{"ip": "not-an-ip"}`,
		`{"ip": 1234}`,
		`{"host": "10.89.0.2"}`,
		`not json`,
		`["10.89.0.2"]`,
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadAccessTarget(dir); err == nil {
			t.Errorf("bad target accepted: %s", body)
		}
	}
	if err := os.WriteFile(path, []byte(`{"ip": "10.89.0.2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err = loadAccessTarget(dir)
	if err != nil || target.modeSet {
		t.Errorf("missing mode = %+v %v", target, err)
	}
}

func TestAccessSQL(t *testing.T) {
	run := "0123456789abcdef0123456789abcdef"
	if createProbeTableSQL != "CREATE TABLE IF NOT EXISTS soda_u08_probe (run_id text PRIMARY KEY, value text NOT NULL)" {
		t.Errorf("create SQL = %q", createProbeTableSQL)
	}
	if got := insertProbeSQL(run); got != "INSERT INTO soda_u08_probe VALUES ('"+run+"', 'client-committed')" {
		t.Errorf("insert SQL = %q", got)
	}
	if got := selectProbeSQL(run); got != "SELECT value FROM soda_u08_probe WHERE run_id='"+run+"'" {
		t.Errorf("select SQL = %q", got)
	}
	if got := updateProbeSQL(run); got != "UPDATE soda_u08_probe SET value='bob-committed' WHERE run_id='"+run+"';\n" {
		t.Errorf("update SQL = %q", got)
	}
}

func stubAccessSSH(t *testing.T, body string, exit int) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "argv.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + log + "\ncat >/dev/null\nprintf '%s' '" + body + "'\nexit " + map[bool]string{true: "0", false: "1"}[exit == 0] + "\n"
	path := filepath.Join(dir, "ssh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestAccessRemote(t *testing.T) {
	log := stubAccessSSH(t, "remote-bytes", 0)
	probe := &accessProbe{config: "/fixture/developer-client.conf", ip: "10.89.0.2"}
	out, err := probe.accessRemote("alice", "head -c 1 x", []byte("stdin-bytes"))
	if err != nil || string(out) != "remote-bytes" {
		t.Fatalf("remote = %q %v", out, err)
	}
	argv, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-F", "/fixture/developer-client.conf", "u08-alice-8417@10.89.0.2", "head -c 1 x"} {
		if !strings.Contains(string(argv), want) {
			t.Errorf("argv log misses %q:\n%s", want, argv)
		}
	}
	stubAccessSSH(t, "", 1)
	if _, err := probe.accessRemote("bob", "true", nil); err == nil {
		t.Error("failing remote accepted")
	}
}

func TestClientPSQL(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "psql.log")
	script := "#!/bin/sh\nprintf 'PGPASSFILE=%s\\n' \"$PGPASSFILE\" >> " + log + "\n" +
		"printf 'PGCONNECT_TIMEOUT=%s\\n' \"$PGCONNECT_TIMEOUT\" >> " + log + "\n" +
		"printf '%s\\n' \"$@\" >> " + log + "\nprintf 'client-committed\\n'\n"
	path := filepath.Join(dir, "psql")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	probe := &accessProbe{ip: "10.89.0.2", passfile: "/fixture/pgpass"}
	out, err := probe.clientPSQL("SELECT 1")
	if err != nil || strings.TrimSpace(string(out)) != "client-committed" {
		t.Fatalf("psql = %q %v", out, err)
	}
	recorded, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PGPASSFILE=/fixture/pgpass", "PGCONNECT_TIMEOUT=10", "-X", "-w", "-h", "10.89.0.2", "developer", "soda_example", "-At", "ON_ERROR_STOP=1", "SELECT 1"} {
		if !strings.Contains(string(recorded), want) {
			t.Errorf("psql invocation misses %q:\n%s", want, recorded)
		}
	}
}

func TestWritePassfile(t *testing.T) {
	dir := t.TempDir()
	path, err := writePassfile(dir, "10.89.0.2", strings.Repeat("a", 43))
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("passfile mode = %o", st.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "10.89.0.2:5432:soda_example:developer:"+strings.Repeat("a", 43)+"\n" {
		t.Errorf("passfile record = %q", data)
	}
	if _, err := writePassfile(dir, "10.89.0.2", strings.Repeat("b", 43)); err == nil {
		t.Error("second passfile write accepted")
	}
}

func TestAccessResultsOrder(t *testing.T) {
	results := workloadAccessResults{
		RunID:               "run",
		NetworkMode:         "bridge",
		HTTPBindEdit:        true,
		Postgres:            true,
		CommittedValue:      "bob-committed",
		RestartPersistence:  false,
		DefaultNestedBridge: true,
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n" +
		"  \"run_id\": \"run\",\n" +
		"  \"network_mode\": \"bridge\",\n" +
		"  \"http_bind_edit_client_and_members\": true,\n" +
		"  \"postgres_client_and_members\": true,\n" +
		"  \"committed_value\": \"bob-committed\",\n" +
		"  \"restart_persistence_verified\": false,\n" +
		"  \"default_nested_bridge_verified\": true\n" +
		"}"
	if string(encoded) != want {
		t.Errorf("results JSON order differs:\n%s", encoded)
	}
}

func TestRunWorkloadAccessValidation(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "target.json"), []byte(`{"SYNTHETIC_PRIVATE_MARKER":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RunWorkloadAccess([]string{dir}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("bad target accepted")
	}
	if !strings.HasPrefix(err.Error(), "Workload access incomplete; retained state; failure type: ") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_MARKER") {
		t.Errorf("failure leaks request: %q", err.Error())
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "wrong")
	if err := RunWorkloadAccess([]string{dir}, &bytes.Buffer{}); err == nil {
		t.Error("wrong env accepted")
	}
}
