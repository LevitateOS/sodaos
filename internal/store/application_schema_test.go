package store

import (
	"context"
	"strings"
	"testing"
)

func TestOpenRefusesIncompleteApplicationSchema(t *testing.T) {
	for _, test := range []struct{ name, damage, probe string }{
		{"applications_table", `ALTER TABLE applications RENAME TO applications_missing`, `SELECT count(*) FROM information_schema.tables WHERE table_name='applications' AND table_schema='public'`},
		{"revisions_table", `ALTER TABLE application_revisions RENAME TO application_revisions_missing`, `SELECT count(*) FROM information_schema.tables WHERE table_name='application_revisions' AND table_schema='public'`},
		{"observation_column", `ALTER TABLE applications DROP COLUMN observation`, `SELECT count(*) FROM information_schema.columns WHERE table_name='applications' AND column_name='observation' AND table_schema='public'`},
		{"identity_guard", `DROP TRIGGER application_identity_immutable ON applications`, `SELECT count(*) FROM information_schema.triggers WHERE trigger_name='application_identity_immutable' AND event_object_table='applications'`},
		{"revision_update_guard", `DROP TRIGGER application_revision_immutable_update ON application_revisions`, `SELECT count(*) FROM information_schema.triggers WHERE trigger_name='application_revision_immutable_update' AND event_object_table='application_revisions'`},
		{"revision_delete_guard", `DROP TRIGGER application_revision_immutable_delete ON application_revisions`, `SELECT count(*) FROM information_schema.triggers WHERE trigger_name='application_revision_immutable_delete' AND event_object_table='application_revisions'`},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, dsn := postgresFixture(t, nil)
			ctx := context.Background()
			if _, err := s.db.ExecContext(ctx, test.damage); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(dsn)
			if err == nil {
				_ = opened.Close()
				t.Fatal("incomplete application schema accepted")
			}
			if !strings.Contains(err.Error(), "database schema is incomplete") {
				t.Fatalf("unexpected refusal: %v", err)
			}
			var missing int
			if err := s.db.QueryRowContext(ctx, test.probe).Scan(&missing); err != nil || missing != 0 {
				t.Fatalf("startup repaired incomplete schema: count=%d err=%v", missing, err)
			}
			var version int
			if err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&version); err != nil || version != SchemaVersion() {
				t.Fatalf("startup changed schema marker %d %v", version, err)
			}
		})
	}
}
