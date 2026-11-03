// Command dev-db emits the known schema upgrade for the retained development
// database copy. It never connects to a database or changes the native store.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/levitateos/factory/pkg/store"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: dev-db > dev-schema.sql")
		os.Exit(2)
	}
	if store.SchemaVersion() != 80 {
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
	var comments strings.Builder
	for _, statement := range store.HumanCommentJournalUpgradeDDL {
		fmt.Fprintf(&comments, "            EXECUTE %s;\n", quote(statement))
	}
	var edits strings.Builder
	for _, statement := range store.HumanEditJournalUpgradeDDL {
		fmt.Fprintf(&edits, "            EXECUTE %s;\n", quote(statement))
	}
	var issues strings.Builder
	for _, statement := range store.HumanIssueCreateJournalUpgradeDDL {
		fmt.Fprintf(&issues, "            EXECUTE %s;\n", quote(statement))
	}
	var locks strings.Builder
	for _, statement := range store.HumanLockJournalUpgradeDDL {
		fmt.Fprintf(&locks, "            EXECUTE %s;\n", quote(statement))
	}
	var reviews strings.Builder
	for _, statement := range store.HumanReviewJournalUpgradeDDL {
		fmt.Fprintf(&reviews, "            EXECUTE %s;\n", quote(statement))
	}
	var inlines strings.Builder
	for _, statement := range store.HumanInlineJournalUpgradeDDL {
		fmt.Fprintf(&inlines, "            EXECUTE %s;\n", quote(statement))
	}
	var threads strings.Builder
	for _, statement := range store.HumanThreadJournalUpgradeDDL {
		fmt.Fprintf(&threads, "            EXECUTE %s;\n", quote(statement))
	}
	var reviewers strings.Builder
	for _, statement := range store.HumanReviewRequestJournalUpgradeDDL {
		fmt.Fprintf(&reviewers, "            EXECUTE %s;\n", quote(statement))
	}
	var merges strings.Builder
	for _, statement := range store.HumanMergeJournalUpgradeDDL {
		fmt.Fprintf(&merges, "            EXECUTE %s;\n", quote(statement))
	}
	var pulls strings.Builder
	for _, statement := range store.HumanPullCreateJournalUpgradeDDL {
		fmt.Fprintf(&pulls, "            EXECUTE %s;\n", quote(statement))
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
        WHEN 71 THEN
            NULL;
        WHEN 72 THEN
            NULL;
        WHEN 73 THEN
            NULL;
        WHEN 74 THEN
            NULL;
        WHEN 75 THEN
            NULL;
		WHEN 76 THEN
			NULL;
		WHEN 77 THEN
			NULL;
		WHEN 78 THEN
			NULL;
		WHEN 79 THEN
			NULL;
		WHEN 80 THEN
			NULL;
        ELSE
            RAISE EXCEPTION 'unsupported development database schema version: %%', minimum;
    END CASE;
    IF minimum < 70 THEN
%s
    END IF;
    IF minimum < 71 THEN
%s
    END IF;
    IF minimum < 72 THEN
%s
    END IF;
    IF minimum < 73 THEN
%s
    END IF;
    IF minimum < 74 THEN
%s
    END IF;
    IF minimum < 75 THEN
%s
    END IF;
    IF minimum < 76 THEN
%s
    END IF;
    IF minimum < 77 THEN
%s
    END IF;
    IF minimum < 78 THEN
%s
    END IF;
    IF minimum < 79 THEN
%s
    END IF;
    IF minimum < 80 THEN
%s
    END IF;
END;
$soda_dev_copy$;
COMMIT;
`, quote(store.ApplicationBuildsDDL), quote(store.ApplicationSetupRenewalsDDL), quote(store.ApplicationSetupRenewalsDDL), posts.String(), comments.String(), edits.String(), issues.String(), locks.String(), reviews.String(), inlines.String(), threads.String(), reviewers.String(), merges.String(), pulls.String())
	if _, err := os.Stdout.WriteString(sql); err != nil {
		fmt.Fprintln(os.Stderr, "dev-db: write development-copy SQL:", err)
		os.Exit(1)
	}
}
