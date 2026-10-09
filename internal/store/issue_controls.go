package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// MaxIntakeDeliveries bounds the durable webhook-delivery dedup log. Intake
// records each assessed delivery ID; redelivered native webhooks reuse
// their task UUID, so the log suppresses duplicates until pruned.
const MaxIntakeDeliveries = 1024

// MaxIssueControls bounds one repository's control listing for queue views.
const MaxIssueControls = 2048

// RecordIssueAssessment stores one readiness assessment when its inputs
// changed: equal fingerprints replay the current record without advancing
// the revision, so unchanged blockers never trigger repeated work. The
// first assessment for an issue always records.
func (s *Store) RecordIssueAssessment(ctx context.Context, candidate factory.IssueControl, now time.Time) (factory.IssueControl, bool, error) {
	if err := candidate.Validate(); err != nil {
		return factory.IssueControl{}, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.IssueControl{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM issue_controls WHERE repository=$1 AND issue=$2 FOR UPDATE`, candidate.Repository, candidate.Issue).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return factory.IssueControl{}, false, err
	}
	stored := candidate
	stored.AssessedUnix = now.Unix()
	if err == nil {
		var current factory.IssueControl
		if err = json.Unmarshal(raw, &current); err != nil {
			return factory.IssueControl{}, false, err
		}
		if current.Fingerprint == candidate.Fingerprint {
			if candidate.NativeRev > current.NativeRev {
				current.NativeRev = candidate.NativeRev
				current.AssessedUnix = now.Unix()
				data, marshalErr := json.Marshal(current)
				if marshalErr != nil {
					return factory.IssueControl{}, false, marshalErr
				}
				if _, err = tx.ExecContext(ctx, `UPDATE issue_controls SET data=$3 WHERE repository=$1 AND issue=$2`, candidate.Repository, candidate.Issue, string(data)); err != nil {
					return factory.IssueControl{}, false, fmt.Errorf("readiness observation refresh failed: %w", err)
				}
			}
			return current, false, tx.Commit()
		}
		stored.Revision = current.Revision + 1
		stored.FirstSeenUnix = current.FirstSeenUnix
	} else {
		stored.Revision = 1
		stored.FirstSeenUnix = now.Unix()
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return factory.IssueControl{}, false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO issue_controls(repository,issue,revision,data) VALUES($1,$2,$3,$4)
		ON CONFLICT(repository,issue) DO UPDATE SET revision=excluded.revision, data=excluded.data`,
		stored.Repository, stored.Issue, stored.Revision, string(data)); err != nil {
		return factory.IssueControl{}, false, fmt.Errorf("readiness assessment failed: %w", err)
	}
	return stored, true, tx.Commit()
}

// IssueControl returns the current readiness record for one native issue.
func (s *Store) IssueControl(ctx context.Context, repository, issue int64) (factory.IssueControl, error) {
	var control factory.IssueControl
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM issue_controls WHERE repository=$1 AND issue=$2`, repository, issue).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &control)
	}
	return control, err
}

// IssueControls lists one repository's current readiness records oldest
// first: native issue indexes increase with creation, so index order is
// oldest-ready-first queue order. At most limit records are returned.
func (s *Store) IssueControls(ctx context.Context, repository int64, limit int) ([]factory.IssueControl, error) {
	if limit <= 0 || limit > MaxIssueControls {
		return nil, errors.New("invalid issue control listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM issue_controls WHERE repository=$1 ORDER BY issue ASC LIMIT $2`, repository, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var controls []factory.IssueControl
	for rows.Next() {
		var data []byte
		var control factory.IssueControl
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &control); err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, rows.Err()
}

// IntakeDeliverySeen reports whether a native webhook delivery ID was
// already assessed. Delivery IDs are native task UUIDs; redeliveries reuse
// them, so a seen ID suppresses duplicate intake work.
func (s *Store) IntakeDeliverySeen(ctx context.Context, delivery string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM intake_deliveries WHERE delivery=$1`, delivery).Scan(&count); err != nil {
		return false, err
	}
	return count != 0, nil
}

// RecordIntakeDelivery logs one assessed delivery and prunes the dedup log
// to its bound. A duplicate ID replays without logging twice.
func (s *Store) RecordIntakeDelivery(ctx context.Context, delivery string, repository, issue int64, kind string, now time.Time) (bool, error) {
	if delivery == "" || len(delivery) > 128 || repository <= 0 || issue <= 0 || kind == "" || len(kind) > 64 {
		return false, errors.New("invalid intake delivery")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO intake_deliveries(delivery,repository,issue,kind,received_at) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(delivery) DO NOTHING`, delivery, repository, issue, kind, now.Unix())
	if err != nil {
		return false, fmt.Errorf("intake delivery log failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM intake_deliveries WHERE delivery NOT IN
		(SELECT delivery FROM intake_deliveries ORDER BY received_at DESC, delivery DESC LIMIT $1)`, MaxIntakeDeliveries); err != nil {
		return false, fmt.Errorf("intake delivery prune failed: %w", err)
	}
	return affected == 0, nil
}

const acceptanceDependantsPageSize = 64

// AcceptanceDependantsPage returns one bounded, stable page of acceptance
// heads that name the endpoint as a prerequisite. Cursor advances over every
// scanned head, including heads that do not match, so sparse pages can resume
// without rescanning. HasMore describes candidate heads after Cursor, not
// whether this page contained any matches.
func (s *Store) AcceptanceDependantsPage(ctx context.Context, repository, issue int64, after factory.DependenceRef) ([]factory.DependenceRef, factory.DependenceRef, bool, error) {
	if repository <= 0 || issue <= 0 || after.Repository < 0 || after.Issue < 0 ||
		(after.Repository == 0) != (after.Issue == 0) {
		return nil, factory.DependenceRef{}, false, errors.New("invalid acceptance dependant page")
	}
	return acceptanceDependantsPage(ctx, s.db, repository, issue, after)
}

type readinessQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func acceptanceDependantsPage(ctx context.Context, queryer readinessQueryer, repository, issue int64, after factory.DependenceRef) ([]factory.DependenceRef, factory.DependenceRef, bool, error) {
	rows, err := queryer.QueryContext(ctx, `WITH candidate_heads AS MATERIALIZED (
			SELECT h.repository, h.issue, h.decision
			FROM issue_acceptance_heads h
			WHERE (h.repository, h.issue) > ($1, $2)
			ORDER BY h.repository, h.issue
			LIMIT $5
		), page AS (
			SELECT h.repository, h.issue,
				EXISTS (
					SELECT 1 FROM issue_acceptance_decisions d
					WHERE d.id = h.decision AND EXISTS (
						SELECT 1
						FROM jsonb_array_elements(d.data->'prerequisites') AS prerequisite(value)
						WHERE jsonb_typeof(value->'endpoint_repository') = 'string'
						  AND jsonb_typeof(value->'endpoint_issue') = 'string'
						  AND value->>'endpoint_repository' = $3::TEXT
						  AND value->>'endpoint_issue' = $4::TEXT
					)
				) AS matches
			FROM candidate_heads h
		), more AS (
			SELECT EXISTS (
				SELECT 1
				FROM issue_acceptance_heads h
				CROSS JOIN LATERAL (
					SELECT p.repository, p.issue FROM page p
					ORDER BY p.repository DESC, p.issue DESC LIMIT 1
				) last
				WHERE (h.repository, h.issue) > (last.repository, last.issue)
			) AS has_more
		)
		SELECT p.repository, p.issue, p.matches, more.has_more
		FROM page p CROSS JOIN more
		ORDER BY p.repository, p.issue`,
		after.Repository, after.Issue, strconv.FormatInt(repository, 10), strconv.FormatInt(issue, 10), acceptanceDependantsPageSize)
	if err != nil {
		return nil, factory.DependenceRef{}, false, err
	}
	defer rows.Close()

	dependants := make([]factory.DependenceRef, 0, acceptanceDependantsPageSize)
	cursor := after
	hasMore := false
	for rows.Next() {
		var candidate factory.DependenceRef
		var matches bool
		if err := rows.Scan(&candidate.Repository, &candidate.Issue, &matches, &hasMore); err != nil {
			return nil, factory.DependenceRef{}, false, err
		}
		cursor = candidate
		if matches {
			dependants = append(dependants, candidate)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, factory.DependenceRef{}, false, err
	}
	if err := rows.Close(); err != nil {
		return nil, factory.DependenceRef{}, false, err
	}
	return dependants, cursor, hasMore, nil
}

// VisitAcceptanceDependants visits recorded heads whose typed acceptance
// declares the endpoint as a prerequisite. Heads are scanned in stable keyset
// pages; each page's rows are closed before invoking the visitor, which
// receives matches in repository/issue order. The native graph stays
// canonical: unaccepted relations never appear here.
func (s *Store) VisitAcceptanceDependants(ctx context.Context, repository, issue int64, visit func(factory.DependenceRef) error) error {
	if repository <= 0 || issue <= 0 || visit == nil {
		return errors.New("invalid acceptance dependant scan")
	}
	after := factory.DependenceRef{}
	for {
		dependants, cursor, hasMore, err := s.AcceptanceDependantsPage(ctx, repository, issue, after)
		if err != nil {
			return err
		}
		for _, dependant := range dependants {
			if err := visit(dependant); err != nil {
				return err
			}
		}
		if !hasMore {
			return nil
		}
		after = cursor
	}
}

// ReadinessSweepRevision returns the native revision of one repository's
// last clean readiness sweep. Absent state reports ErrNotFound: unknown
// repositories sweep fully the first time.
func (s *Store) ReadinessSweepRevision(ctx context.Context, repository int64) (int64, error) {
	var revision int64
	err := s.db.QueryRowContext(ctx, `SELECT revision FROM factory_readiness_sweeps WHERE repository=$1`, repository).Scan(&revision)
	return revision, err
}

// SaveReadinessSweep records one repository's last clean sweep revision.
func (s *Store) SaveReadinessSweep(ctx context.Context, repository, revision int64, now time.Time) error {
	if repository <= 0 || revision < 1 {
		return errors.New("invalid readiness sweep state")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO factory_readiness_sweeps(repository,revision,swept_at) VALUES($1,$2,$3)
		ON CONFLICT(repository) DO UPDATE SET revision=excluded.revision, swept_at=excluded.swept_at`,
		repository, revision, now.Unix())
	if err != nil {
		return fmt.Errorf("readiness sweep state failed: %w", err)
	}
	return nil
}

// FactoryPolicies returns every recorded repository factory policy for
// readiness sweeping. Callers filter enablement from the records.
func (s *Store) FactoryPolicies(ctx context.Context) ([]factory.RepositoryPolicy, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_policies ORDER BY repository ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var policies []factory.RepositoryPolicy
	for rows.Next() {
		var data []byte
		var policy factory.RepositoryPolicy
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &policy); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}
