package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Static contract for the appliance PostgreSQL runtime: image pin, storage,
// startup dependencies and secret-safe wiring. The live backup/restore proof
// is TestPostgresBackupRoundTrip; this test guards the shipped unit text.
func readRuntimeFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func requireContains(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestPostgresUnitTopology(t *testing.T) {
	body := readRuntimeFile(t, "appliance/services/soda-postgres.container")
	requireContains(t, body,
		"Image=docker.io/library/postgres:17@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675",
		"ContainerName=soda-postgres",
		"Pull=missing",
		"Network=soda.network",
		"Volume=/var/lib/soda/postgres:/var/lib/postgresql/data:Z",
		"Volume=/run/soda/postgres:/var/run/postgresql:z",
		"Volume=/etc/soda/postgres/super.passwd:/run/secrets/soda-pg-super:ro,Z",
		"Environment=POSTGRES_PASSWORD_FILE=/run/secrets/soda-pg-super",
		"WantedBy=multi-user.target",
	)
	for line := range strings.Lines(body) {
		if strings.HasPrefix(strings.TrimSpace(line), "PublishPort=") {
			t.Fatal("database must not publish a host port")
		}
	}
	// App-role passwords stay in root-only files consumed by the init unit;
	// the DB container never sees them (its init runs as uid 999).
	for _, leak := range []string{"soda-pg-forgejo", "soda-pg-soda", "initdb.d"} {
		if strings.Contains(body, leak) {
			t.Fatalf("database unit must not reference %s", leak)
		}
	}
	if strings.Contains(body, "POSTGRES_PASSWORD=") {
		t.Fatal("superuser password must arrive by file, never environment value")
	}
}

func TestForgejoUnitPostgresWiring(t *testing.T) {
	body := readRuntimeFile(t, "appliance/services/forgejo.container")
	requireContains(t, body,
		"Requires=soda-postgres-init.service",
		"After=soda-postgres-init.service",
		"Network=soda.network",
		"Volume=/etc/soda/postgres/forgejo.passwd:/etc/forgejo/db_passwd:ro,Z",
		"Environment=FORGEJO__database__DB_TYPE=postgres",
		"Environment=FORGEJO__database__HOST=soda-postgres:5432",
		"Environment=FORGEJO__database__NAME=forgejo",
		"Environment=FORGEJO__database__USER=forgejo",
		"Environment=FORGEJO__database__PASSWD_URI=file:/etc/forgejo/db_passwd",
	)
	if strings.Contains(body, "sqlite3") {
		t.Fatal("forgejo unit still references sqlite3")
	}
}

func TestPostgresProvisionUnitWiring(t *testing.T) {
	body := readRuntimeFile(t, "appliance/services/soda-pg-provision.service")
	requireContains(t, body,
		"ConditionPathExists=/etc/soda/installed",
		"ConditionPathExists=!/etc/soda/postgres/super.passwd",
		"ExecStart=/usr/bin/soda-setup --provision-db-only",
		"WantedBy=multi-user.target",
	)
	database := readRuntimeFile(t, "appliance/services/soda-postgres.container")
	requireContains(t, database,
		"Requires=soda-pg-provision.service",
		"After=soda-pg-provision.service",
	)
	prepare := readRuntimeFile(t, "internal/release/image/prepare.go")
	requireContains(t, prepare, `"soda-pg-provision.service"`)
}

func TestDashboardUnitDatabaseWiring(t *testing.T) {
	body := readRuntimeFile(t, "appliance/services/soda-dashboard.container")
	requireContains(t, body,
		"Volume=/etc/soda/postgres/soda.dsn:/etc/soda/postgres/soda.dsn:ro,Z",
		"Volume=/run/soda/postgres:/run/soda/postgres:ro,z",
	)
}

func TestPostgresBackupScheduleAndStaging(t *testing.T) {
	init := readRuntimeFile(t, "appliance/services/soda-postgres-init.service")
	requireContains(t, init,
		"Requires=soda-postgres.service",
		"After=soda-postgres.service",
		"Before=forgejo.service",
		"Type=oneshot",
		"ExecStart=/usr/bin/soda-pg-init-roles",
		"RemainAfterExit=yes",
		"UMask=0077",
	)
	roles := readRuntimeFile(t, "appliance/bin/soda-pg-init-roles")
	requireContains(t, roles,
		"pg_isready",
		"CREATE ROLE",
		"CREATE DATABASE",
		"ON_ERROR_STOP",
	)
	service := readRuntimeFile(t, "appliance/services/soda-postgres-backup.service")
	requireContains(t, service,
		"After=soda-postgres-init.service",
		"Type=oneshot",
		"ExecStart=/usr/bin/soda-pg-backup",
		"UMask=0077",
	)
	timer := readRuntimeFile(t, "appliance/services/soda-postgres-backup.timer")
	requireContains(t, timer,
		"OnCalendar=",
		"Persistent=true",
		"Unit=soda-postgres-backup.service",
		"WantedBy=timers.target",
	)
	tmpfiles := readRuntimeFile(t, "appliance/config/soda.tmpfiles")
	requireContains(t, tmpfiles,
		"d /var/lib/soda/postgres 0700 999 999 -",
		"d /var/lib/soda/backups/postgres 0700 root root -",
		"d /run/soda/postgres 0755 999 999 -",
	)
	prepare := readRuntimeFile(t, "internal/release/image/prepare.go")
	requireContains(t, prepare,
		`"soda-postgres.container"`,
		`"soda.network"`,
		`"soda-postgres-backup.service"`,
		`"soda-postgres-backup.timer"`,
		`"soda-postgres-init.service"`,
		`"appliance/bin/soda-pg-backup"`,
		`"appliance/bin/soda-pg-restore"`,
		`"appliance/bin/soda-pg-init-roles"`,
	)
	if strings.Contains(prepare, "postgres-init/") {
		t.Fatal("prepare must not reference the removed initdb.d staging")
	}
}

func TestFixtureMatchesProductImage(t *testing.T) {
	unit := readRuntimeFile(t, "appliance/services/soda-postgres.container")
	var pinned string
	for line := range strings.Lines(unit) {
		if ref, ok := strings.CutPrefix(line, "Image="); ok {
			pinned = strings.TrimSpace(ref)
		}
	}
	if pinned == "" {
		t.Fatal("no Image= in soda-postgres.container")
	}
	fixture := readRuntimeFile(t, "rust/soda-pg-fixture/src/main.rs")
	if !strings.Contains(fixture, `"`+pinned+`"`) {
		t.Fatalf("fixture image differs from product unit: %s", pinned)
	}
}
