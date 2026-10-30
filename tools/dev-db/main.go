// Command dev-db emits the known schema upgrade for the retained development
// database copy. It never connects to a database or changes the native store.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/levitateos/sodaos/internal/store"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: dev-db > dev-schema.sql")
		os.Exit(2)
	}
	if store.SchemaVersion() != 70 {
		fmt.Fprintln(os.Stderr, "dev-db: review the development-copy upgrade for the current store schema")
		os.Exit(1)
	}
	quote := func(statement string) string {
		return "'" + strings.ReplaceAll(statement, "'", "''") + "'"
	}
	var posts strings.Builder
	for _, statement := range store.FeedPostsUpgradeDDL {
		fmt.Fprintf(&posts, "            EXECUTE %s;\n", quote(statement))
	}
	sql := fmt.Sprintf(`-- Apply only to the retained development copy, never the native Soda database.
BEGIN;
DO $soda_dev_copy$
DECLARE
    versions BIGINT;
    minimum INTEGER;
    maximum INTEGER;
BEGIN
    IF current_database() <> 'soda_dev' THEN
        RAISE EXCEPTION 'development-copy upgrade requires database soda_dev';
    END IF;
    LOCK TABLE schema_version IN EXCLUSIVE MODE;
    SELECT count(*), min(version), max(version)
        INTO versions, minimum, maximum FROM schema_version;
    IF versions <> 1 OR minimum IS NULL OR minimum <> maximum THEN
        RAISE EXCEPTION 'invalid development database schema version record';
    END IF;
    CASE minimum
        WHEN 67 THEN
            EXECUTE %s;
            EXECUTE %s;
        WHEN 68 THEN
            EXECUTE %s;
        WHEN 69 THEN
            NULL;
        WHEN 70 THEN
            NULL;
        ELSE
            RAISE EXCEPTION 'unsupported development database schema version: %%', minimum;
    END CASE;
    IF minimum < 70 THEN
%s
    END IF;
END;
$soda_dev_copy$;
COMMIT;
`, quote(store.ApplicationBuildsDDL), quote(store.ApplicationSetupRenewalsDDL), quote(store.ApplicationSetupRenewalsDDL), posts.String())
	if _, err := os.Stdout.WriteString(sql); err != nil {
		fmt.Fprintln(os.Stderr, "dev-db: write development-copy SQL:", err)
		os.Exit(1)
	}
}
