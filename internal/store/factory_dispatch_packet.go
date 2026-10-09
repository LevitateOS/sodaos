package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// ErrAssignmentActive reports a dispatch packet for an issue that already
// carries an unfinished assignment. One issue runs at most one assignment
// at a time; a concurrent dispatcher won this issue.
var ErrAssignmentActive = errors.New("issue already carries an unfinished assignment")

// Admission refusals from the atomic dispatch gate. The packet records
// first and the limits are rechecked inside the same transaction, so a
// refusal means a concurrent admission consumed the room this attempt
// planned against; the dispatcher waits instead of launching.
var (
	ErrDispatchControlStale  = errors.New("queued issue control changed during dispatch")
	ErrCapacityFull          = errors.New("appliance runs at its limit")
	ErrRepositoryFull        = errors.New("repository runs at its limit")
	ErrSponsorshipFull       = errors.New("sponsorship runs at its limit")
	ErrAllowanceExhausted    = errors.New("sponsorship allowance is exhausted")
	ErrConnectionUsageBudget = errors.New("connection rolling usage budget is exhausted")
	ErrAdmissionChanged      = errors.New("admission grants changed during dispatch")
)

// RecordDispatchPacket stores one dispatch atomically: the gate
// registration, the exact current queued control, the assigned assignment, its held reservation, the
// recorded run and the display binding. Either the whole packet lands or
// nothing does, so a crash never leaves a reservation without its
// assignment or a run without its dispatch. A second unfinished
// assignment for the issue refuses with ErrAssignmentActive instead of
// dispatching twice.
// The control row is locked after the dispatch gate and before packet
// writes; readiness assessment takes the same row lock, so changed or
// missing outcome evidence refuses the whole transaction.
//
// The packet is also the atomic admission gate: the limit rows are
// locked, the packet records, and then appliance, repository,
// sponsorship, repository allowance and connection rolling usage limits are rechecked inside the same
// transaction. Concurrent passes serialize here, so only room that
// actually exists is admitted; the losers wait instead of launching.
func (s *Store) RecordDispatchPacket(ctx context.Context, d factory.DispatchRegistration, expected factory.IssueControl, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = registerDispatchTx(ctx, tx, d); err != nil {
		return err
	}
	if err = recordDispatchPacketTx(ctx, tx, d, expected, a, r, run, view, false); err != nil {
		return err
	}
	return tx.Commit()
}

func sameControlOutcome(left, right factory.IssueControl) bool {
	return left.Reason == right.Reason && slices.Equal(left.Blockers, right.Blockers) &&
		maps.Equal(left.EndpointHeads, right.EndpointHeads) && maps.Equal(left.Satisfied, right.Satisfied)
}

func recordDispatchPacketTx(ctx context.Context, tx *sql.Tx, d factory.DispatchRegistration, expected factory.IssueControl, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView, freshAttempt bool) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := a.Validate(); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := run.Validate(); err != nil {
		return err
	}
	if err := view.Validate(); err != nil {
		return err
	}
	if expected.NativeRev != a.NativeRev {
		return ErrDispatchControlStale
	}
	if err := checkQueuedControlTx(ctx, tx, expected, a.Repository, a.Issue, a.Acceptance); err != nil {
		return err
	}
	if d.ID != a.ID || d.Repository != a.Repository {
		return errors.New("dispatch packet registration does not match its assignment")
	}
	if d.Authority != a.Authority {
		return ErrAdmissionChanged
	}
	if a.Stage != factory.AssignmentAssigned || a.Outcome != "" || a.Result != nil {
		return errors.New("dispatch packet carries a fresh assignment")
	}
	if r.AssignmentID != a.ID || r.Repository != a.Repository || r.Connection != a.Connection || r.State != factory.ReservationHeld {
		return errors.New("dispatch packet reservation does not match its assignment")
	}
	if run.ID != a.Run || run.ProjectID != a.ProjectID || run.Role != a.Role || run.InputSHA != a.SourceCommit || run.Outcome != "" || run.Reconciled {
		return errors.New("dispatch packet run does not match its assignment")
	}
	if view.RunID != run.ID || view.Repository != a.Repository || view.Issue != a.Issue || view.Attempt != a.ID {
		return errors.New("dispatch packet view does not match its assignment")
	}
	if err := validateAttemptPacketTx(ctx, tx, a, freshAttempt); err != nil {
		return err
	}
	grants, err := loadAdmissionGrantsTx(ctx, tx, a.Repository, a.Connection)
	if err != nil {
		return err
	}
	if err = checkAssignmentAuthorityTx(ctx, tx, a.Authority, a, grants); err != nil {
		return err
	}
	run.Admission = &factory.RunAdmission{Authority: a.Authority, Policy: grants.policy, Profile: *grants.environment.Profile}
	if err = run.Validate(); err != nil {
		return err
	}
	adata, err := json.Marshal(a)
	if err != nil {
		return err
	}
	rdata, err := json.Marshal(r)
	if err != nil {
		return err
	}
	rundata, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(adata)); err != nil {
		return dispatchPacketError(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_reservations(assignment,repository,connection,state,data) VALUES($1,$2,$3,$4,$5)`,
		r.AssignmentID, r.Repository, r.Connection, r.State, string(rdata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES($1,TRUE,FALSE,$2)`, run.ID, string(rundata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_run_views(run,repository,issue,attempt) VALUES($1,$2,$3,$4)`,
		view.RunID, view.Repository, view.Issue, view.Attempt); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if err = checkAdmissionLimitsTx(ctx, tx, a.Repository, a.Connection, a.ProjectID, grants); err != nil {
		return err
	}
	now := time.Now()
	if freshAttempt {
		return startFreshAttemptAllowanceTx(ctx, tx, a.Repository, a.Issue, a.AttemptRoot, grants.policy.AttemptLimits, run.Deadline, now)
	}
	if a.PublicationAssignment != a.ID && a.Role == project.RoleCoder {
		if err = consumeCorrectionAttemptTx(ctx, tx, a, now); err != nil {
			return err
		}
	}
	return admitAttemptAllowanceTx(ctx, tx, a.Repository, a.Issue, a.AttemptRoot, &grants.policy.AttemptLimits, run.Deadline, now)
}

// validateAttemptPacketTx checks whether this packet owns a publication or
// is a reviewer/correction child of the currently published candidate.
// Owner replans retain their prior allowance root while starting a new
// publication; children retain both parent publication and allowance roots.
func validateAttemptPacketTx(ctx context.Context, tx *sql.Tx, a factory.Assignment, freshAttempt bool) error {
	owner := a.PublicationAssignment == a.ID
	if freshAttempt {
		if !owner || a.AttemptRoot != a.ID || a.Role != project.RoleCoder {
			return errors.New("fresh allowance requires a self-owned coder assignment")
		}
	}
	if owner {
		if a.Role != project.RoleCoder {
			return errors.New("publication owner assignment must be a coder")
		}
		if a.AttemptRoot != a.ID {
			var exists bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_attempt_allowances WHERE repository=$1 AND issue=$2 AND root_assignment=$3)`, a.Repository, a.Issue, a.AttemptRoot).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrAttemptRootMismatch
			}
		}
		return nil
	}
	if freshAttempt || (a.Role != project.RoleCoder && a.Role != project.RoleReviewer) {
		return errors.New("child assignment has an invalid role or fresh allowance")
	}
	var parentData []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1 FOR UPDATE`, a.PublicationAssignment).Scan(&parentData); err != nil {
		return err
	}
	var parent factory.Assignment
	if err := json.Unmarshal(parentData, &parent); err != nil {
		return err
	}
	if err := parent.Validate(); err != nil {
		return err
	}
	if parent.ID != a.PublicationAssignment || parent.PublicationAssignment != parent.ID || parent.Role != project.RoleCoder ||
		parent.Stage != factory.AssignmentFinished || parent.Outcome != factory.Succeeded ||
		parent.Repository != a.Repository || parent.Issue != a.Issue || parent.ProjectID != a.ProjectID ||
		parent.Acceptance != a.Acceptance || parent.AttemptRoot != a.AttemptRoot {
		return errors.New("child assignment does not match its publication owner")
	}
	var publicationData []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1 FOR UPDATE`, parent.ID).Scan(&publicationData); err != nil {
		return err
	}
	var publication factory.Publication
	if err := json.Unmarshal(publicationData, &publication); err != nil {
		return err
	}
	if err := publication.Validate(); err != nil {
		return err
	}
	if publication.AssignmentID != parent.ID || publication.Repository != a.Repository || publication.Issue != a.Issue ||
		publication.Acceptance != a.Acceptance || publication.Stage != factory.PublicationPublished ||
		publication.WithdrawRequested || publication.PRNumber <= 0 || publication.PRID <= 0 || publication.Candidate != a.SourceCommit {
		return errors.New("child assignment does not match the current published candidate")
	}
	var accepted string
	if err := tx.QueryRowContext(ctx, `SELECT decision FROM issue_acceptance_heads WHERE repository=$1 AND issue=$2 FOR UPDATE`, a.Repository, a.Issue).Scan(&accepted); err != nil {
		return err
	}
	if accepted != a.Acceptance {
		return ErrAdmissionChanged
	}
	if a.Role == project.RoleCoder && a.Preparation != parent.Preparation {
		return errors.New("correction must retain the publication coder preparation")
	}
	if a.Role == project.RoleReviewer && a.Preparation == parent.Preparation {
		return errors.New("reviewer child requires a distinct preparation")
	}
	var rootExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_attempt_allowances WHERE repository=$1 AND issue=$2 AND root_assignment=$3)`, a.Repository, a.Issue, a.AttemptRoot).Scan(&rootExists); err != nil {
		return err
	}
	if !rootExists {
		return ErrAttemptRootMismatch
	}
	return nil
}

func checkQueuedControlTx(ctx context.Context, tx *sql.Tx, expected factory.IssueControl, repository, issue int64, acceptance string) error {
	if err := expected.Validate(); err != nil || expected.Readiness != factory.ReadinessQueued || expected.Repository != repository || expected.Issue != issue || expected.Acceptance != acceptance {
		return ErrDispatchControlStale
	}
	var controlData []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM issue_controls WHERE repository=$1 AND issue=$2 FOR UPDATE`, repository, issue).Scan(&controlData); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDispatchControlStale
		}
		return err
	}
	var current factory.IssueControl
	if err := json.Unmarshal(controlData, &current); err != nil {
		return err
	}
	if err := current.Validate(); err != nil || current.Readiness != factory.ReadinessQueued || current.Repository != repository || current.Issue != issue || current.Acceptance != acceptance || current.NativeRev != expected.NativeRev || current.Revision != expected.Revision || current.Fingerprint != expected.Fingerprint || current.Authority != expected.Authority || !sameControlOutcome(current, expected) {
		return ErrDispatchControlStale
	}
	return nil
}

type admissionGrants struct {
	capacity    factory.Capacity
	policy      factory.RepositoryPolicy
	operator    factory.OperatorGrant
	sponsorship factory.Sponsorship
	usageBudget factory.ConnectionUsageBudget
	environment project.EnvironmentGrant
}

func loadAdmissionGrantsTx(ctx context.Context, t *sql.Tx, repository int64, connection string) (admissionGrants, error) {
	var grants admissionGrants
	var cdata, pdata, gdata []byte
	if err := t.QueryRowContext(ctx, `SELECT data FROM factory_capacity WHERE id=1 FOR UPDATE`).Scan(&cdata); err != nil {
		return admissionGrants{}, admissionChanged(err)
	}
	if err := json.Unmarshal(cdata, &grants.capacity); err != nil {
		return admissionGrants{}, err
	}
	if err := t.QueryRowContext(ctx, `SELECT data FROM factory_policies WHERE repository=$1 FOR UPDATE`, repository).Scan(&pdata); err != nil {
		return admissionGrants{}, admissionChanged(err)
	}
	if err := json.Unmarshal(pdata, &grants.policy); err != nil {
		return admissionGrants{}, err
	}
	if err := t.QueryRowContext(ctx, `SELECT data FROM factory_operator_grants WHERE repository=$1 FOR UPDATE`, repository).Scan(&gdata); err != nil {
		return admissionGrants{}, admissionChanged(err)
	}
	if err := json.Unmarshal(gdata, &grants.operator); err != nil {
		return admissionGrants{}, err
	}
	rows, err := t.QueryContext(ctx, `SELECT connection,data FROM factory_sponsorships WHERE repository=$1 ORDER BY connection LIMIT 33 FOR UPDATE`, repository)
	if err != nil {
		return admissionGrants{}, err
	}
	rowCount := 0
	selected := false
	for rows.Next() {
		rowCount++
		var candidate factory.Sponsorship
		var candidateConnection string
		var data []byte
		if err = rows.Scan(&candidateConnection, &data); err != nil {
			_ = rows.Close()
			return admissionGrants{}, err
		}
		if err = json.Unmarshal(data, &candidate); err != nil {
			_ = rows.Close()
			return admissionGrants{}, err
		}
		if candidate.Connection != candidateConnection {
			_ = rows.Close()
			return admissionGrants{}, ErrAdmissionChanged
		}
		if candidate.Active {
			if !selected {
				grants.sponsorship = candidate
				selected = true
			}
		}
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return admissionGrants{}, err
	}
	if err = rows.Close(); err != nil {
		return admissionGrants{}, err
	}
	if rowCount >= 33 || !selected || grants.sponsorship.Connection != connection {
		return admissionGrants{}, ErrAdmissionChanged
	}
	var budgetData []byte
	if err = t.QueryRowContext(ctx, `SELECT data FROM factory_connection_usage_budgets WHERE connection=$1 FOR UPDATE`, connection).Scan(&budgetData); err != nil {
		return admissionGrants{}, admissionChanged(err)
	}
	if err = json.Unmarshal(budgetData, &grants.usageBudget); err != nil {
		return admissionGrants{}, err
	}
	if grants.usageBudget.Connection != connection || grants.usageBudget.Validate() != nil {
		return admissionGrants{}, ErrAdmissionChanged
	}
	var environmentData []byte
	if err = t.QueryRowContext(ctx, `SELECT data FROM project_environment_grants WHERE repository=$1 FOR UPDATE`, repository).Scan(&environmentData); err != nil {
		return admissionGrants{}, admissionChanged(err)
	}
	if err = json.Unmarshal(environmentData, &grants.environment); err != nil {
		return admissionGrants{}, err
	}
	if grants.environment.Validate() != nil {
		return admissionGrants{}, ErrAdmissionChanged
	}
	return grants, nil
}

// checkAssignmentAuthorityTx binds fresh authorization to the grant rows locked
// for this admission. A retry retains its original assignment history while
// checking the current role grant and the same approved preparation decisions.
func checkAssignmentAuthorityTx(ctx context.Context, t *sql.Tx, expected factory.AuthorityRef, a factory.Assignment, grants admissionGrants) error {
	var requirement, approval string
	if err := t.QueryRowContext(ctx, `SELECT decision FROM project_requirement_heads WHERE project_id=$1 FOR UPDATE`, a.ProjectID).Scan(&requirement); err != nil {
		return admissionChanged(err)
	}
	if err := t.QueryRowContext(ctx, `SELECT decision FROM project_approval_heads WHERE project_id=$1 FOR UPDATE`, a.ProjectID).Scan(&approval); err != nil {
		return admissionChanged(err)
	}

	permitted := false
	for _, role := range grants.sponsorship.Roles {
		permitted = permitted || role == a.Role
	}
	if !grants.policy.Enabled || grants.policy.Paused || !grants.operator.Active ||
		!grants.sponsorship.Active || !permitted || !grants.environment.Active || grants.environment.Profile == nil {
		return ErrAdmissionChanged
	}
	current := factory.AuthorityRef{
		Policy: grants.policy.Revision, Operator: grants.operator.Revision,
		Capacity: grants.capacity.Revision, Sponsorship: grants.sponsorship.Revision,
		SponsorshipConnection: grants.sponsorship.Connection, ConnectionUsageBudget: grants.usageBudget.Revision,
		Environment: grants.environment.Revision, RequirementsID: requirement, ApprovalID: approval,
	}
	if current != expected || requirement != a.Authority.RequirementsID || approval != a.Authority.ApprovalID {
		return ErrAdmissionChanged
	}
	return nil
}

// checkAdmissionLimitsTx enforces appliance, one-session-per-repository,
// sponsorship, repository allowance and connection rolling usage limits inside the admission transaction. The limit rows are
// locked first so concurrent admissions serialize here; the packet's own
// held reservation is already recorded, so every comparison accounts for
// it and refuses exactly when the pre-packet state plus this admission
// would exceed a limit. Count reads are capped just past each limit,
// which decides exact admission without scanning settled history.
func checkAdmissionLimitsTx(ctx context.Context, t *sql.Tx, repository int64, connection, projectID string, grants admissionGrants) error {
	charge, err := connectionUsageCharge(ctx, t, connection, time.Now())
	if err != nil {
		return err
	}
	if charge > grants.usageBudget.RollingMinutes*usageMicrosPerMinute {
		return ErrConnectionUsageBudget
	}
	heldTotal, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_reservations WHERE state='held'`, grants.capacity.MaxConcurrentRuns+1)
	if err != nil {
		return err
	}
	unattributed, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_runs WHERE active AND id NOT IN (SELECT run FROM factory_assignments WHERE run!='')`, grants.capacity.MaxConcurrentRuns+1)
	if err != nil {
		return err
	}
	if heldTotal+unattributed > grants.capacity.MaxConcurrentRuns {
		return ErrCapacityFull
	}
	// One executing factory session per repository is a product invariant,
	// even when the policy or operator has configured a larger concurrency
	// value. Count held reservations and every active attributed run: a run
	// whose reservation was consumed or released still owns its process slot.
	repoSessions, err := cappedCountTx(ctx, t, `
		SELECT 1 FROM factory_reservations
			WHERE state='held' AND repository=$1
		UNION ALL
		SELECT 1 FROM factory_runs r
			JOIN factory_assignments a ON a.run=r.id
			WHERE r.active AND a.repository=$1
			  AND NOT EXISTS (SELECT 1 FROM factory_reservations res
				WHERE res.assignment=a.id AND res.state='held')
		UNION ALL
		SELECT 1 FROM factory_runs r
			WHERE r.active
			  AND r.id NOT IN (SELECT run FROM factory_assignments WHERE run!='')
			  AND r.data->>'project_id'=$2`, 2, repository, projectID)
	if err != nil {
		return err
	}
	if repoSessions > 1 {
		return ErrRepositoryFull
	}
	slots, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_reservations WHERE state='held' AND repository=$1 AND connection=$2`, grants.sponsorship.MaxConcurrent+1, repository, connection)
	if err != nil {
		return err
	}
	if slots > grants.sponsorship.MaxConcurrent {
		return ErrSponsorshipFull
	}
	var planned int
	err = t.QueryRowContext(ctx, `SELECT coalesce(sum((data->>'planned_minutes')::bigint),0)::bigint FROM factory_reservations WHERE state='held' AND repository=$1 AND connection=$2`,
		repository, connection).Scan(&planned)
	if err != nil {
		return err
	}
	var used int
	err = t.QueryRowContext(ctx, `SELECT COALESCE(SUM(minutes),0) FROM factory_usage WHERE repository=$1 AND connection=$2`,
		repository, connection).Scan(&used)
	if err != nil {
		return err
	}
	if grants.sponsorship.AllowanceMinutes-used-planned < 0 {
		return ErrAllowanceExhausted
	}
	return nil
}

func admissionChanged(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrAdmissionChanged
	}
	return err
}

// cappedCountTx counts a query's rows up to and including limit: it
// returns the exact count below the limit and the limit itself above
// it. Admission compares against limits far below any table size, so
// the capped count decides exact admission with bounded work.
func cappedCountTx(ctx context.Context, t *sql.Tx, query string, limit int, args ...any) (int, error) {
	var n int
	params := append(append([]any{}, args...), limit)
	limitParameter := "$" + strconv.Itoa(len(params))
	err := t.QueryRowContext(ctx, `SELECT count(*) FROM (`+query+` LIMIT `+limitParameter+`) t`, params...).Scan(&n)
	return n, err
}

func dispatchPacketError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAssignmentActive
	}
	return fmt.Errorf("dispatch packet failed: %w", err)
}
