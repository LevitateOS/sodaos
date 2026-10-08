package scripts

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	pgMaintBinOnce sync.Once
	pgMaintBinRoot string
	pgMaintBinDiag string
)

// pgMaintenanceBinary builds the Rust maintenance tools once (offline,
// std-only) and returns the path of the named binary. The ports preserve
// the shell CLI, exit codes and file behavior exactly.
func pgMaintenanceBinary(t *testing.T, name string) string {
	t.Helper()
	pgMaintBinOnce.Do(func() {
		root, err := filepath.Abs("..")
		if err != nil {
			pgMaintBinDiag = err.Error()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		build := exec.CommandContext(ctx, "cargo", "build", "-p", "soda-pg-maintenance")
		build.Dir = root
		if out, err := build.CombinedOutput(); err != nil {
			pgMaintBinDiag = string(out)
			return
		}
		pgMaintBinRoot = root
	})
	if pgMaintBinRoot == "" {
		t.Fatalf("build soda-pg-maintenance: %s", pgMaintBinDiag)
	}
	return filepath.Join(pgMaintBinRoot, "target", "debug", name)
}

// Live proof that the shipped backup/restore tools round-trip both appliance
// databases on disposable PostgreSQL. Skips when the container engine or the
// pinned image is unavailable, like other engine-dependent checks.
func pgFixtureStart(t *testing.T, env ...string) map[string]string {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman is not installed on this test host")
	}
	// The Rust port preserves the shell CLI exactly: `start` prints
	// KEY=VALUE assignments, `stop <container>` removes the container.
	bin := buildRustPortBinary(t, "soda-pg-fixture")
	cmd := exec.Command(bin, "start")
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 3 {
		t.Skipf("postgres fixture unavailable: %s", strings.TrimSpace(string(out)))
	}
	if err != nil {
		t.Fatalf("fixture start: %v: %s", err, out)
	}
	vars := map[string]string{}
	for line := range strings.Lines(string(out)) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key == "" {
			t.Fatalf("bad fixture line: %q", line)
		}
		vars[key] = strings.Trim(value, `"`)
	}
	for _, key := range []string{"SODA_PG_CONTAINER", "SODA_PG_PORT", "SODA_PG_DIR"} {
		if vars[key] == "" {
			t.Fatalf("fixture omitted %s: %s", key, out)
		}
	}
	t.Cleanup(func() {
		stop := exec.Command(bin, "stop", vars["SODA_PG_CONTAINER"])
		if out, err := stop.CombinedOutput(); err != nil {
			t.Errorf("fixture stop: %v: %s", err, out)
		}
	})
	return vars
}

func pgExec(t *testing.T, container, db, stdin string) string {
	t.Helper()
	cmd := exec.Command("podman", "exec", "-i", "-u", "postgres", container, "psql", "-v", "ON_ERROR_STOP=1", "-d", db, "-f", "-")
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql %s: %v: %s", db, err, out)
	}
	return string(out)
}

func pgQuery(t *testing.T, container, db, sql string) string {
	t.Helper()
	cmd := exec.Command("podman", "exec", "-i", "-u", "postgres", container, "psql", "-v", "ON_ERROR_STOP=1", "-tA", "-c", sql, "-d", db)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql %s: %v: %s", db, err, out)
	}
	return string(out)
}

func TestPostgresBackupRoundTrip(t *testing.T) {
	fix := pgFixtureStart(t)
	container := fix["SODA_PG_CONTAINER"]

	seed := `CREATE TABLE soda_probe(id serial primary key, note text);
INSERT INTO soda_probe(note) VALUES ('first'), ('it''s quoted'), ('third');`
	for db := range strings.FieldsSeq(fix["SODA_PG_DATABASES"]) {
		pgExec(t, container, db, seed)
	}

	backupRoot := t.TempDir()
	runBackup := func(keep string) string {
		t.Helper()
		cmd := exec.Command(pgMaintenanceBinary(t, "soda-pg-backup"))
		cmd.Env = append(os.Environ(),
			"SODA_PG_CONTAINER="+container,
			"SODA_PG_BACKUP_DIR="+backupRoot,
			"SODA_PG_DATABASES="+fix["SODA_PG_DATABASES"],
			"SODA_PG_KEEP="+keep,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("soda-pg-backup: %v: %s", err, out)
		}
		return string(out)
	}

	output := runBackup("2")
	if !strings.Contains(output, "backup complete: ") {
		t.Fatalf("no completion line: %s", output)
	}
	// Secret safety: random per-run passwords must not leak into tool output.
	passwords, err := filepath.Glob(filepath.Join(fix["SODA_PG_DIR"], "*.passwd"))
	if err != nil || len(passwords) == 0 {
		t.Fatalf("fixture wrote no password files: %v", passwords)
	}
	for _, path := range passwords {
		secret, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output, strings.TrimSpace(string(secret))) {
			t.Fatalf("backup output leaks %s", filepath.Base(path))
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode %o, want 600", path, info.Mode().Perm())
		}
	}

	run := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(output[strings.LastIndex(output, "backup complete: "):]), "backup complete: "))
	for _, db := range strings.Fields(fix["SODA_PG_DATABASES"]) {
		dump := filepath.Join(run, db+".dump")
		info, err := os.Stat(dump)
		if err != nil || info.Size() == 0 {
			t.Fatalf("missing dump %s: %v", dump, err)
		}
	}
	if info, err := os.Stat(filepath.Join(run, "globals.sql")); err != nil || info.Size() == 0 {
		t.Fatalf("missing globals.sql: %v", err)
	}

	// Destroy and restore through the shipped tool; the application-visible
	// rows must come back exactly.
	for db := range strings.FieldsSeq(fix["SODA_PG_DATABASES"]) {
		pgExec(t, container, db, "DROP TABLE soda_probe;")
	}
	restore := exec.Command(pgMaintenanceBinary(t, "soda-pg-restore"), "--yes", run)
	restore.Env = append(os.Environ(), "SODA_PG_CONTAINER="+container)
	if out, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("soda-pg-restore: %v: %s", err, out)
	}
	for db := range strings.FieldsSeq(fix["SODA_PG_DATABASES"]) {
		got := pgQuery(t, container, db, "SELECT note FROM soda_probe ORDER BY id;")
		if got != "first\nit's quoted\nthird\n" {
			t.Fatalf("%s rows after restore: %q", db, got)
		}
	}

	// Rotation keeps the newest runs only and never the in-progress dir.
	time.Sleep(1100 * time.Millisecond)
	runBackup("2")
	time.Sleep(1100 * time.Millisecond)
	runBackup("2")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("rotation kept %d runs, want 2: %v", len(entries), names)
	}

	// First-boot path: the shipped init tool provisions roles/databases on a
	// bare cluster, idempotently, with hostile password content.
	bare := pgFixtureStart(t, "SODA_PG_DATABASES=")
	pwdir := t.TempDir()
	for _, role := range []string{"forgejo", "soda"} {
		if err := os.WriteFile(filepath.Join(pwdir, role+".passwd"), []byte("q'uote$d\\"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	initRoles := func() string {
		t.Helper()
		cmd := exec.Command(pgMaintenanceBinary(t, "soda-pg-init-roles"))
		cmd.Env = append(os.Environ(),
			"SODA_PG_CONTAINER="+bare["SODA_PG_CONTAINER"],
			"SODA_PG_PASSWORD_DIR="+pwdir,
			"SODA_PG_DATABASES=forgejo soda",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("soda-pg-init-roles: %v: %s", err, out)
		}
		return string(out)
	}
	initRoles()
	initRoles()
	got := pgQuery(t, bare["SODA_PG_CONTAINER"], "postgres", "SELECT rolname FROM pg_roles WHERE rolname IN ('forgejo','soda') ORDER BY 1;")
	if got != "forgejo\nsoda\n" {
		t.Fatalf("roles after init: %q", got)
	}
	got = pgQuery(t, bare["SODA_PG_CONTAINER"], "postgres", "SELECT datname FROM pg_database WHERE datname IN ('forgejo','soda') ORDER BY 1;")
	if got != "forgejo\nsoda\n" {
		t.Fatalf("databases after init: %q", got)
	}
	// The quoted password actually authenticates over TCP.
	tcp := exec.Command("podman", "exec", "-e", `PGPASSWORD=q'uote$d\`, bare["SODA_PG_CONTAINER"],
		"psql", "-h", "127.0.0.1", "-U", "forgejo", "-d", "forgejo", "-tA", "-c", "SELECT 1;")
	if out, err := tcp.CombinedOutput(); err != nil {
		t.Fatalf("role password login: %v: %s", err, out)
	}
	if out, err := func() ([]byte, error) {
		cmd := exec.Command(pgMaintenanceBinary(t, "soda-pg-init-roles"))
		cmd.Env = append(os.Environ(),
			"SODA_PG_CONTAINER="+bare["SODA_PG_CONTAINER"],
			"SODA_PG_PASSWORD_DIR="+t.TempDir(),
		)
		return cmd.CombinedOutput()
	}(); err == nil {
		t.Fatalf("init with missing password files succeeded: %s", out)
	}

	// Refusals: restore without explicit confirmation, backup with no retention.
	refuse := exec.Command(pgMaintenanceBinary(t, "soda-pg-restore"), run)
	refuse.Env = append(os.Environ(), "SODA_PG_CONTAINER="+container)
	if out, err := refuse.CombinedOutput(); err == nil {
		t.Fatalf("restore without --yes succeeded: %s", out)
	}
	keep := exec.Command(pgMaintenanceBinary(t, "soda-pg-backup"))
	keep.Env = append(os.Environ(),
		"SODA_PG_CONTAINER="+container,
		"SODA_PG_BACKUP_DIR="+t.TempDir(),
		"SODA_PG_KEEP=0",
	)
	if out, err := keep.CombinedOutput(); err == nil {
		t.Fatalf("backup with KEEP=0 succeeded: %s", out)
	}
}

func TestPostgresChildSQLDelivery(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	scratchRoot := filepath.Join(root, "target", "pg-child-delivery")
	if err := os.MkdirAll(scratchRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(scratchRoot, "case-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove test scratch: %v", err)
		}
	})

	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	callsPath := filepath.Join(dir, "calls")
	fixture := `#!/bin/sh
printf '%s\n' "$*" >> "$SODA_FAKE_CALLS"
case "$*" in
  "exec -u postgres fixture-container pg_isready -q") exit 0 ;;
  "exec -i -u postgres fixture-container psql -v ON_ERROR_STOP=1 -f -")
    case "$SODA_FAKE_MODE" in
      drain) exec /bin/cat > "$SODA_FAKE_CAPTURE" ;;
      close) exec 0<&-; exit "$SODA_FAKE_EXIT" ;;
      *) exit 91 ;;
    esac ;;
  *) exit 92 ;;
esac
`
	podman := filepath.Join(binDir, "podman")
	if err := os.WriteFile(podman, []byte(fixture), 0o700); err != nil {
		t.Fatal(err)
	}

	baseEnv := []string{
		"PATH=" + binDir,
		"SODA_PG_CONTAINER=fixture-container",
		"SODA_FAKE_CALLS=" + callsPath,
	}
	run := func(binary string, args []string, extra ...string) ([]byte, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env = append(append([]string{}, baseEnv...), extra...)
		return cmd.CombinedOutput()
	}
	clearCalls := func(t *testing.T) {
		t.Helper()
		if err := os.WriteFile(callsPath, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	assertCalls := func(t *testing.T, want string) {
		t.Helper()
		got, err := os.ReadFile(callsPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("podman calls = %q, want %q", got, want)
		}
	}

	passwordDir := filepath.Join(dir, "passwords")
	if err := os.Mkdir(passwordDir, 0o700); err != nil {
		t.Fatal(err)
	}
	password := []byte("bounded-synthetic-password\n")
	if err := os.WriteFile(filepath.Join(passwordDir, "soda.passwd"), password, 0o600); err != nil {
		t.Fatal(err)
	}
	databases := strings.Repeat("soda ", 12_000)
	if len(databases) >= 64<<10 || len(password) >= 64<<10 {
		t.Fatal("synthetic SQL inputs exceed the bounded fixture profile")
	}
	initEnv := []string{
		"SODA_PG_PASSWORD_DIR=" + passwordDir,
		"SODA_PG_DATABASES=" + databases,
		"SODA_FAKE_MODE=drain",
		"SODA_FAKE_CAPTURE=" + filepath.Join(dir, "init.sql"),
	}
	clearCalls(t)
	out, err := run(pgMaintenanceBinary(t, "soda-pg-init-roles"), nil, initEnv...)
	if err != nil {
		t.Fatalf("soda-pg-init-roles success delivery: %v: %s", err, out)
	}
	assertCalls(t, "exec -u postgres fixture-container pg_isready -q\nexec -i -u postgres fixture-container psql -v ON_ERROR_STOP=1 -f -\n")
	initSQL, err := os.ReadFile(filepath.Join(dir, "init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if len(initSQL) <= 1<<20 || !bytes.Contains(initSQL, []byte(`ALTER ROLE "soda" WITH LOGIN PASSWORD 'bounded-synthetic-password';`)) {
		t.Fatalf("captured init SQL size/content = %d bytes", len(initSQL))
	}

	globalsSize := 2 << 20
	pattern := []byte("-- synthetic globals\nSELECT 1;\n")
	globals := bytes.Repeat(pattern, globalsSize/len(pattern)+1)
	globalsPath := filepath.Join(dir, "globals.sql")
	if err := os.WriteFile(globalsPath, globals, 0o600); err != nil {
		t.Fatal(err)
	}
	clearCalls(t)
	restoreCapture := filepath.Join(dir, "restored-globals.sql")
	out, err = run(pgMaintenanceBinary(t, "soda-pg-restore"), []string{"--yes", "--globals", dir},
		"SODA_FAKE_MODE=drain", "SODA_FAKE_CAPTURE="+restoreCapture)
	if err != nil {
		t.Fatalf("soda-pg-restore success delivery: %v: %s", err, out)
	}
	assertCalls(t, "exec -i -u postgres fixture-container psql -v ON_ERROR_STOP=1 -f -\n")
	restored, err := os.ReadFile(restoreCapture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, globals) {
		t.Fatalf("restored globals differ: got %d bytes, want %d", len(restored), len(globals))
	}

	failureCases := []struct {
		name  string
		bin   string
		args  []string
		extra []string
		calls string
	}{
		{
			name:  "init-roles",
			bin:   "soda-pg-init-roles",
			extra: initEnv[:2],
			calls: "exec -u postgres fixture-container pg_isready -q\nexec -i -u postgres fixture-container psql -v ON_ERROR_STOP=1 -f -\n",
		},
		{
			name:  "restore-globals",
			bin:   "soda-pg-restore",
			args:  []string{"--yes", "--globals", dir},
			calls: "exec -i -u postgres fixture-container psql -v ON_ERROR_STOP=1 -f -\n",
		},
	}
	for _, test := range failureCases {
		for _, code := range []string{"0", "23"} {
			t.Run(test.name+"-exit"+code, func(t *testing.T) {
				clearCalls(t)
				failureEnv := append(append([]string{}, test.extra...),
					"SODA_FAKE_MODE=close", "SODA_FAKE_EXIT="+code,
					"SODA_FAKE_CAPTURE="+filepath.Join(dir, "unused.sql"))
				out, err := run(pgMaintenanceBinary(t, test.bin), test.args, failureEnv...)
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("close mode: err = %v, output %s", err, out)
				}
				want := 1
				if code == "23" {
					want = 23
				}
				if exit.ExitCode() != want {
					t.Fatalf("close mode exited %d, want %d", exit.ExitCode(), want)
				}
				assertCalls(t, test.calls)
			})
		}
	}
}
