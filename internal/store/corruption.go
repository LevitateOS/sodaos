package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Test-only corruption and failure-injection helpers. Production code must
// never call these: they create invalid rows, remove schema, or inject a
// transaction-boundary condition that the public store API cannot stage.
// They live here because raw SQL is allowed only in internal/store.

// InjectCorruptFactoryRun inserts one factory_runs row with caller-supplied
// bytes, bypassing every validation the store API applies.
func InjectCorruptFactoryRun(ctx context.Context, dsn, id string, active, settled bool, data string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("postgres connection string is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES($1,$2,$3,$4)`,
		id, active, settled, data)
	return err
}

// DropFactoryRunViews removes the run-view table so view-read failure paths
// can be exercised against a live database.
func DropFactoryRunViews(ctx context.Context, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("postgres connection string is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `DROP TABLE factory_run_views`)
	return err
}

// CloseDispatchAfterReviewRegistration installs a one-test trigger that
// closes the repository gate after a review intent is recorded. It lets
// control tests verify that a post-registration close prevents native submit.
func CloseDispatchAfterReviewRegistration(ctx context.Context, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("postgres connection string is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `CREATE FUNCTION close_dispatch_after_review_registration() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE previous_count integer; next_count integer;
BEGIN
  previous_count := jsonb_array_length(COALESCE(OLD.data->'review_operations','[]'::jsonb));
  next_count := jsonb_array_length(COALESCE(NEW.data->'review_operations','[]'::jsonb));
  IF next_count > previous_count THEN
    UPDATE factory_dispatch SET revision=revision+1, open=FALSE,
      data=jsonb_build_object(
        'active_causes',jsonb_build_array('test_review_close'),
        'publications',jsonb_build_object('publications',jsonb_build_array(),'operations',jsonb_build_array(),'pending',FALSE),
        'merges',jsonb_build_object('merges',jsonb_build_array(),'operations',jsonb_build_array(),'pending',FALSE),
        'captured',jsonb_build_array(),'repository',NEW.repository::text,'revision',factory_dispatch.revision+1,
        'cause','test_review_close','closed_by','test:review-boundary')
    WHERE repository=NEW.repository;
  END IF;
  RETURN NEW;
END $$;`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `CREATE TRIGGER close_dispatch_after_review_registration AFTER UPDATE ON factory_publications FOR EACH ROW EXECUTE FUNCTION close_dispatch_after_review_registration()`)
	return err
}
