package control_test

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// seedStagedDependencyEdge records one blocked-by-blocker edge in a staged
// Forgejo database. The staged external database serves dependency reads but
// exposes no working REST writer, so native fixture setup inserts the edge
// directly and Soda observes it through production snapshots. This helper is
// test-only; product Store remains PostgreSQL-only.
func seedStagedDependencyEdge(fountainDB string, creatorID, blockedID, blockerID int64) error {
	db, err := sql.Open("sqlite", "file:"+fountainDB+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`PRAGMA table_info(issue_dependency)`)
	if err != nil {
		return err
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt *string
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			_ = rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !columns["issue_id"] || !columns["dependency_id"] {
		return fmt.Errorf("native dependency table lacks its edge columns: %v", columns)
	}
	names := []string{"issue_id", "dependency_id"}
	values := []any{blockedID, blockerID}
	now := time.Now().Unix()
	for _, optional := range []struct {
		name  string
		value any
	}{{"user_id", creatorID}, {"created_unix", now}, {"updated_unix", now}} {
		if columns[optional.name] {
			names = append(names, optional.name)
			values = append(values, optional.value)
		}
	}
	placeholders := strings.Repeat("?,", len(names))
	_, err = db.Exec(`INSERT INTO issue_dependency(`+strings.Join(names, ",")+
		`) VALUES(`+placeholders[:len(placeholders)-1]+`)`, values...)
	return err
}
