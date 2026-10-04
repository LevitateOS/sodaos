package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The migrate script drops only the inline [database] PASSWD that conflicts
// with PASSWD_URI; any other section's credentials must survive byte-identical.
func TestForgejoMigrateScrubsDatabasePasswdOnly(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "app.ini")
	before := `[database]
DB_TYPE = postgres
HOST = soda-postgres:5432
PASSWD = installer-secret

[mailer]
ENABLED = true
PASSWD = smtp-secret
`
	if err := os.WriteFile(conf, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "../appliance/bin/soda-forgejo-migrate")
	cmd.Env = append(os.Environ(), "SODA_FORGEJO_APP_INI="+conf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soda-forgejo-migrate: %v: %s", err, out)
	}
	after, err := os.ReadFile(conf)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "installer-secret") {
		t.Fatal("inline [database] PASSWD survived")
	}
	for _, keep := range []string{"DB_TYPE = postgres", "HOST = soda-postgres:5432", "PASSWD = smtp-secret", "[mailer]"} {
		if !strings.Contains(string(after), keep) {
			t.Fatalf("lost %q", keep)
		}
	}
	st, err := os.Stat(conf)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatal("mode changed")
	}
}

func TestForgejoMigrateSkipsMissingConfig(t *testing.T) {
	cmd := exec.Command("bash", "../appliance/bin/soda-forgejo-migrate")
	cmd.Env = append(os.Environ(), "SODA_FORGEJO_APP_INI="+filepath.Join(t.TempDir(), "absent.ini"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("missing config must skip cleanly: %v: %s", err, out)
	}
}

func TestForgejoMigrateUnitWiring(t *testing.T) {
	unit := readRuntimeFile(t, "appliance/services/soda-forgejo-migrate.service")
	requireContains(t, unit,
		"Before=forgejo.service",
		"ConditionPathExists=/var/lib/soda/forgejo/gitea/conf/app.ini",
		"ExecStart=/usr/bin/soda-forgejo-migrate",
	)
	container := readRuntimeFile(t, "appliance/services/forgejo.container")
	requireContains(t, container,
		"Requires=soda-forgejo-migrate.service",
		"After=soda-forgejo-migrate.service",
	)
	prepare := readRuntimeFile(t, "internal/release/image/prepare.go")
	requireContains(t, prepare,
		`"soda-forgejo-migrate.service"`,
		`"appliance/bin/soda-forgejo-migrate"`,
	)
}
