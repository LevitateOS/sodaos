package store

import (
	"context"
	"database/sql"
	"errors"
)

// Old experimental exclusive-connection rows are not a current broker format.
func verifyIdentityLeaseFormat(ctx context.Context, tx *sql.Tx) error {
	var obsolete int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pragma_index_list('identity_leases') i JOIN pragma_index_info(i.name) c WHERE i."unique"=1 AND c.name='connection_id'`).Scan(&obsolete)
	if err != nil {
		return err
	}
	if obsolete != 0 {
		return errors.New("obsolete identity lease format; provision a current broker database")
	}
	return nil
}
