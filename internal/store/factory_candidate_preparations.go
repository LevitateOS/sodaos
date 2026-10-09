package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// CandidatePreparationRegistration is the Store admission request for one
// reviewer checkout. The authorization, owner, control and deadline are
// recorded beside the ordinary preparation record before the host is called.
type CandidatePreparationRegistration struct {
	Repository        int64
	Issue             int64
	ActorID           int64
	Project           string
	OwnerAssignment   string
	ChildAssignment   string
	AttemptRoot       string
	PublicationID     string
	Connection        string
	Authority         factory.AuthorityRef
	GateRevision      int64
	Control           factory.IssueControl
	RequestedNotAfter int64
}

// CandidatePreparationRecord joins the public preparation view with its
// Store-owned admission and native-stop receipt.
type CandidatePreparationRecord struct {
	Preparation project.StoredPreparation
	Admission   CandidatePreparationRegistration
	NotAfter    int64
	Stopped     bool
	Retirement  string
}

type candidatePreparationEnvelope struct {
	project.StoredPreparation
	Admission candidatePreparationRegistration `json:"factory_admission,omitempty"`
}

type candidatePreparationRegistration struct {
	Repository      int64                `json:"repository"`
	Issue           int64                `json:"issue"`
	ActorID         int64                `json:"actor_id,string"`
	Project         string               `json:"project"`
	OwnerAssignment string               `json:"owner_assignment"`
	ChildAssignment string               `json:"child_assignment"`
	AttemptRoot     string               `json:"attempt_root"`
	PublicationID   string               `json:"publication_id"`
	Connection      string               `json:"connection"`
	Authority       factory.AuthorityRef `json:"authority"`
	GateRevision    int64                `json:"gate_revision"`
	Control         factory.IssueControl `json:"control"`
	NotAfter        int64                `json:"not_after_unix"`
	Stopped         bool                 `json:"stopped,omitempty"`
	Retirement      string               `json:"retirement,omitempty"`
}

var ErrCandidatePreparationPending = errors.New("candidate preparation retirement is not confirmed")

// AdmitCandidatePreparation is the atomic pre-host gate for a candidate
// preparation. Replays retain the first absolute native deadline and exact
// owner/authority tuple.
func (s *Store) AdmitCandidatePreparation(ctx context.Context, preparation project.StoredPreparation, registration CandidatePreparationRegistration) (CandidatePreparationRecord, bool, error) {
	var empty CandidatePreparationRecord
	if err := preparation.Validate(); err != nil {
		return empty, false, err
	}
	if preparation.State.ID != "" || preparation.State.Phase != "" {
		return empty, false, errors.New("candidate preparation admission must start without observed state")
	}
	if err := validateCandidatePreparationRegistration(preparation, registration); err != nil {
		return empty, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return empty, false, err
	}
	defer func() { _ = tx.Rollback() }()
	grants, err := authorizeCandidatePreparationTx(ctx, tx, preparation, registration)
	if err != nil {
		return empty, false, err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, registration.Repository, registration.Issue, registration.AttemptRoot)
	if err != nil {
		return empty, false, err
	}
	if err = checkCandidatePreparationOwnerTx(ctx, tx, preparation, registration, grants.sponsorship.ActorID); err != nil {
		return empty, false, err
	}
	if allowance.Closed {
		return empty, false, factory.ErrAttemptClosed
	}
	if !allowance.Active {
		return empty, false, factory.ErrAttemptTimeExhausted
	}
	now := time.Now()
	deadline, err := attemptDeadlineUnix(allowance, now)
	if err != nil {
		return empty, false, err
	}
	if registration.RequestedNotAfter < deadline {
		deadline = registration.RequestedNotAfter
	}
	if deadline <= now.Unix() {
		return empty, false, factory.ErrAttemptTimeExhausted
	}
	meta := candidatePreparationRegistration{
		Repository: registration.Repository, Issue: registration.Issue, ActorID: registration.ActorID, Project: registration.Project,
		OwnerAssignment: registration.OwnerAssignment, ChildAssignment: registration.ChildAssignment,
		AttemptRoot: registration.AttemptRoot, PublicationID: registration.PublicationID,
		Connection: registration.Connection, Authority: registration.Authority,
		GateRevision: registration.GateRevision, Control: registration.Control, NotAfter: deadline,
	}
	data, err := marshalCandidatePreparation(preparation, meta)
	if err != nil {
		return empty, false, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data)
		VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, preparation.Preparation.ID,
		preparation.Preparation.Project, preparation.Preparation.Role, preparation.Preparation.Revision,
		preparation.Preparation.Requirements.ID, preparation.Preparation.Approval.ID, string(data))
	if err != nil {
		return empty, false, fmt.Errorf("candidate preparation admission failed: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return empty, false, err
	}
	created := inserted == 1
	if created {
		if err = checkAdmissionLimitsTx(ctx, tx, registration.Repository, registration.Connection, registration.Project, grants); err != nil {
			return empty, false, err
		}
	}
	var storedData []byte
	var revision int64
	if err = tx.QueryRowContext(ctx, `SELECT revision,data FROM project_preparations WHERE id=$1 FOR UPDATE`, preparation.Preparation.ID).Scan(&revision, &storedData); err != nil {
		return empty, false, err
	}
	var stored project.StoredPreparation
	var storedMeta *candidatePreparationRegistration
	if stored, storedMeta, err = unmarshalCandidatePreparation(storedData); err != nil {
		return empty, false, err
	}
	if !samePreparationIdentity(stored, preparation) || storedMeta == nil {
		return empty, false, errors.New("preparation identity already carries different admission")
	}
	if !sameCandidatePreparationAdmission(*storedMeta, meta, !created) {
		return empty, false, errors.New("candidate preparation admission changed")
	}
	if revision != stored.Preparation.Revision {
		return empty, false, errors.New("candidate preparation revision differs")
	}
	if err = tx.Commit(); err != nil {
		return empty, false, err
	}
	return CandidatePreparationRecord{
		Preparation: stored, Admission: registrationFromMeta(*storedMeta), NotAfter: storedMeta.NotAfter,
		Stopped: storedMeta.Stopped, Retirement: storedMeta.Retirement,
	}, created, nil
}

// AuthorizeCandidatePreparation rechecks the recorded exact gate and owner
// immediately before the host call and returns its original deadline.
func (s *Store) AuthorizeCandidatePreparation(ctx context.Context, id string) (int64, error) {
	if !project.ValidPreparationID(id) {
		return 0, errors.New("invalid candidate preparation identity")
	}
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM project_preparations WHERE id=$1`, id).Scan(&raw); err != nil {
		return 0, err
	}
	preparation, meta, err := unmarshalCandidatePreparation(raw)
	if err != nil || meta == nil {
		return 0, errors.New("candidate preparation admission is missing")
	}
	registration := registrationFromMeta(*meta)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	grants, err := authorizeCandidatePreparationTx(ctx, tx, preparation, registration)
	if err != nil {
		return 0, err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, registration.Repository, registration.Issue, registration.AttemptRoot)
	if err != nil {
		return 0, err
	}
	if err = checkCandidatePreparationOwnerTx(ctx, tx, preparation, registration, grants.sponsorship.ActorID); err != nil {
		return 0, err
	}
	var currentData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM project_preparations WHERE id=$1 FOR UPDATE`, id).Scan(&currentData); err != nil {
		return 0, err
	}
	currentPreparation, currentMeta, err := unmarshalCandidatePreparation(currentData)
	if err != nil || currentMeta == nil || !samePreparationIdentity(currentPreparation, preparation) ||
		!sameCandidatePreparationAdmission(*currentMeta, *meta, false) {
		return 0, errors.New("candidate preparation admission changed")
	}
	meta = currentMeta
	preparation = currentPreparation
	if allowance.Closed {
		return 0, factory.ErrAttemptClosed
	}
	if !allowance.Active {
		return 0, factory.ErrAttemptTimeExhausted
	}
	deadline, err := attemptDeadlineUnix(allowance, time.Now())
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	if meta.Stopped || meta.Retirement == "confirmed" {
		return 0, factory.ErrAttemptClosed
	}
	if now >= meta.NotAfter || meta.NotAfter > deadline {
		return 0, factory.ErrAttemptTimeExhausted
	}
	if preparation.State.Ready || preparation.State.Phase == project.PrepareFailed {
		return 0, errors.New("candidate preparation is already terminal")
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return meta.NotAfter, nil
}

// CandidatePreparations pages exact pending admissions for one repository.
// A zero issue/root scope includes every pending preparation in the project.
func (s *Store) CandidatePreparations(ctx context.Context, repository int64, projectID string, issue int64, root, after string, limit int, expiredBefore int64) ([]CandidatePreparationRecord, error) {
	if repository <= 0 || !project.ValidID(projectID) || limit <= 0 || limit > 256 || issue < 0 || root != "" && !factory.ValidID(root) || after != "" && !project.ValidPreparationID(after) {
		return nil, errors.New("invalid candidate preparation listing scope")
	}
	query := `SELECT data FROM project_preparations
		WHERE project_id=$1 AND id>$2 AND data ? 'factory_admission'
		AND (data->'factory_admission'->>'repository')::bigint=$3
		AND (COALESCE(data->'factory_admission'->>'issue','0')::bigint=$4 OR $4=0)
		AND (COALESCE(data->'factory_admission'->>'attempt_root','')=$5 OR $5='')
		AND (COALESCE(data->'factory_admission'->>'not_after_unix','0')::bigint<=$6 OR $6=0)
		AND NOT (COALESCE((data#>>'{state,ready}')::boolean,FALSE)
			OR (COALESCE((data#>>'{factory_admission,stopped}')::boolean,FALSE)
				AND COALESCE(data#>>'{factory_admission,retirement}'='confirmed',FALSE)))
		ORDER BY id LIMIT $7`
	rows, err := s.db.QueryContext(ctx, query, projectID, after, repository, issue, root, expiredBefore, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]CandidatePreparationRecord, 0)
	for rows.Next() {
		var data []byte
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		preparation, meta, decodeErr := unmarshalCandidatePreparation(data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		if meta == nil {
			return nil, errors.New("candidate preparation admission disappeared")
		}
		out = append(out, CandidatePreparationRecord{
			Preparation: preparation, Admission: registrationFromMeta(*meta), NotAfter: meta.NotAfter,
			Stopped: meta.Stopped, Retirement: meta.Retirement,
		})
	}
	return out, rows.Err()
}

// RecordCandidatePreparationStop saves the native stop tombstone without
// fabricating role/setup observations for a stop-before-approve response.
func (s *Store) RecordCandidatePreparationStop(ctx context.Context, id string, state project.PrepareState) error {
	if !project.ValidPreparationID(id) || state.ID != id || !state.Stopped ||
		(state.Retirement != "confirmed" && state.Retirement != "uncertain") {
		return errors.New("invalid candidate preparation stop receipt")
	}
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM project_preparations WHERE id=$1`, id).Scan(&raw); err != nil {
		return err
	}
	preparation, meta, err := unmarshalCandidatePreparation(raw)
	if err != nil || meta == nil {
		return errors.New("candidate preparation admission is missing")
	}
	if state.Project != preparation.Preparation.Project || state.Role != "" && state.Role != preparation.Preparation.Role {
		return errors.New("candidate preparation stop receipt identity differs")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	allowance, lockErr := lockAttemptAllowanceRootTx(ctx, tx, meta.Repository, meta.Issue, meta.AttemptRoot)
	if lockErr != nil && !errors.Is(lockErr, sql.ErrNoRows) {
		return lockErr
	}
	var current []byte
	var revision int64
	if err = tx.QueryRowContext(ctx, `SELECT revision,data FROM project_preparations WHERE id=$1 FOR UPDATE`, id).Scan(&revision, &current); err != nil {
		return err
	}
	stored, currentMeta, err := unmarshalCandidatePreparation(current)
	if err != nil || currentMeta == nil || !samePreparationIdentity(stored, preparation) ||
		!sameCandidatePreparationAdmission(*currentMeta, *meta, false) {
		return errors.New("candidate preparation admission changed during stop")
	}
	if currentMeta.Stopped && currentMeta.Retirement == "confirmed" && state.Retirement != "confirmed" {
		return errors.New("confirmed candidate preparation retirement cannot be downgraded")
	}
	if currentMeta.Stopped && currentMeta.Retirement == state.Retirement {
		return tx.Commit()
	}
	currentMeta.Stopped, currentMeta.Retirement = state.Stopped, state.Retirement
	stored.Preparation.Revision++
	data, err := marshalCandidatePreparation(stored, *currentMeta)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE project_preparations SET revision=$1,data=$2 WHERE id=$3 AND revision=$4`, stored.Preparation.Revision, string(data), id, revision)
	if err != nil {
		return err
	}
	if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
		if rowsErr != nil {
			return rowsErr
		}
		return errors.New("candidate preparation stop receipt became stale")
	}
	if lockErr == nil {
		if err = transitionLockedAttemptRootTx(ctx, tx, allowance, allowance.Closed, time.Now()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validateCandidatePreparationRegistration(preparation project.StoredPreparation, r CandidatePreparationRegistration) error {
	if r.Repository <= 0 || r.Issue <= 0 || r.ActorID <= 0 || !project.ValidID(r.Project) || r.Project != preparation.Preparation.Project ||
		!factory.ValidID(r.OwnerAssignment) || !factory.ValidID(r.ChildAssignment) || !factory.ValidID(r.AttemptRoot) ||
		!factory.ValidID(r.PublicationID) || r.Connection == "" || r.GateRevision < 0 || r.RequestedNotAfter <= 0 ||
		r.Control.Validate() != nil || r.Control.Repository != r.Repository || r.Control.Issue != r.Issue ||
		r.Control.Readiness != factory.ReadinessQueued || r.Authority.RequirementsID != preparation.Preparation.Requirements.ID ||
		r.Authority.ApprovalID != preparation.Preparation.Approval.ID || r.Authority.SponsorshipConnection != r.Connection ||
		preparation.Preparation.Role != project.RoleReviewer {
		return errors.New("invalid candidate preparation registration")
	}
	return nil
}

func authorizeCandidatePreparationTx(ctx context.Context, tx *sql.Tx, preparation project.StoredPreparation, r CandidatePreparationRegistration) (admissionGrants, error) {
	var empty admissionGrants
	open, revision, _, err := ensureDispatchGateTx(ctx, tx, r.Repository)
	if err != nil {
		return empty, err
	}
	if !open {
		return empty, ErrDispatchClosed
	}
	if revision != r.GateRevision {
		return empty, ErrStaleRevision
	}
	if err = checkCandidatePreparationControlTx(ctx, tx, r.Control, r.Repository, r.Issue, r.Control.Acceptance); err != nil {
		return empty, err
	}
	assignment := factory.Assignment{
		Authority: r.Authority, ID: r.ChildAssignment, AttemptRoot: r.AttemptRoot,
		PublicationAssignment: r.OwnerAssignment, ProjectID: r.Project,
		ActorID: r.ActorID, Role: project.RoleReviewer, Repository: r.Repository, Issue: r.Issue,
		NativeRev: r.Control.NativeRev, Acceptance: r.Control.Acceptance,
		Preparation: preparation.Preparation.ID, Connection: r.Connection,
		SourceCommit: preparation.Preparation.SourceCommit,
	}
	if err = validateAttemptPacketTx(ctx, tx, assignment, false); err != nil {
		return empty, err
	}
	var publicationData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, r.OwnerAssignment).Scan(&publicationData); err != nil {
		return empty, err
	}
	var publication factory.Publication
	if err = json.Unmarshal(publicationData, &publication); err != nil {
		return empty, err
	}
	if publication.ID != r.PublicationID || publication.Candidate != preparation.Preparation.SourceCommit ||
		publication.ProjectID != r.Project || publication.Stage != factory.PublicationPublished || publication.WithdrawRequested {
		return empty, ErrAttemptRootMismatch
	}
	grants, err := loadAdmissionGrantsTx(ctx, tx, r.Repository, r.Connection)
	if err != nil {
		return empty, err
	}
	if grants.sponsorship.ProjectID != r.Project || grants.sponsorship.ActorID != r.ActorID {
		return empty, ErrAdmissionChanged
	}
	if err = checkAssignmentAuthorityTx(ctx, tx, r.Authority, assignment, grants); err != nil {
		return empty, err
	}
	return grants, nil
}

func checkCandidatePreparationOwnerTx(ctx context.Context, tx *sql.Tx, preparation project.StoredPreparation, r CandidatePreparationRegistration, actorID int64) error {
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, r.Repository, r.Issue, r.AttemptRoot)
	if err != nil {
		return err
	}
	if allowance.Repository != r.Repository || allowance.Issue != r.Issue || allowance.RootAssignment != r.AttemptRoot ||
		allowance.Closed || !allowance.Active {
		return factory.ErrAttemptClosed
	}
	owner, err := currentAttemptOwnerTx(ctx, tx, r.Repository, r.Issue, r.AttemptRoot)
	if err != nil || owner != r.OwnerAssignment {
		return ErrAttemptRootMismatch
	}
	var data []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, owner).Scan(&data); err != nil {
		return ErrAttemptRootMismatch
	}
	var assignment factory.Assignment
	if err = json.Unmarshal(data, &assignment); err != nil || assignment.ActorID != actorID || assignment.ProjectID != r.Project {
		return ErrAttemptRootMismatch
	}
	return nil
}

func marshalCandidatePreparation(p project.StoredPreparation, meta candidatePreparationRegistration) ([]byte, error) {
	return json.Marshal(candidatePreparationEnvelope{StoredPreparation: p, Admission: meta})
}

func unmarshalCandidatePreparation(data []byte) (project.StoredPreparation, *candidatePreparationRegistration, error) {
	var preparation project.StoredPreparation
	if err := json.Unmarshal(data, &preparation); err != nil {
		return preparation, nil, err
	}
	var envelope struct {
		Admission *candidatePreparationRegistration `json:"factory_admission"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return preparation, nil, err
	}
	return preparation, envelope.Admission, nil
}

func registrationFromMeta(m candidatePreparationRegistration) CandidatePreparationRegistration {
	return CandidatePreparationRegistration{
		Repository: m.Repository, Issue: m.Issue, ActorID: m.ActorID, Project: m.Project, OwnerAssignment: m.OwnerAssignment,
		ChildAssignment: m.ChildAssignment, AttemptRoot: m.AttemptRoot, PublicationID: m.PublicationID,
		Connection: m.Connection, Authority: m.Authority, GateRevision: m.GateRevision,
		Control: m.Control, RequestedNotAfter: m.NotAfter,
	}
}

func samePreparationIdentity(left, right project.StoredPreparation) bool {
	left.Preparation.Revision, right.Preparation.Revision = 0, 0
	return reflect.DeepEqual(left.Preparation, right.Preparation)
}

func sameCandidatePreparationAdmission(left, right candidatePreparationRegistration, preserveDeadline bool) bool {
	if preserveDeadline {
		left.NotAfter = 0
		right.NotAfter = 0
	}
	left.Stopped, right.Stopped = false, false
	left.Retirement, right.Retirement = "", ""
	left.Control.NativeRev, right.Control.NativeRev = 0, 0
	left.Control.AssessedUnix, right.Control.AssessedUnix = 0, 0
	return reflect.DeepEqual(left, right)
}

// Candidate preparation replay binds semantic readiness, not the native
// revision used to order fresh reads. NativeRev may advance while the same
// queued authorization remains valid.
func checkCandidatePreparationControlTx(ctx context.Context, tx *sql.Tx, expected factory.IssueControl, repository, issue int64, acceptance string) error {
	if err := expected.Validate(); err != nil || expected.Readiness != factory.ReadinessQueued ||
		expected.Repository != repository || expected.Issue != issue || expected.Acceptance != acceptance {
		return ErrDispatchControlStale
	}
	var data []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM issue_controls WHERE repository=$1 AND issue=$2 FOR UPDATE`, repository, issue).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDispatchControlStale
		}
		return err
	}
	var current factory.IssueControl
	if err := json.Unmarshal(data, &current); err != nil {
		return err
	}
	if err := current.Validate(); err != nil || current.Readiness != factory.ReadinessQueued ||
		current.Repository != repository || current.Issue != issue || current.Acceptance != acceptance ||
		current.NativeRev < expected.NativeRev || current.Revision != expected.Revision ||
		current.Fingerprint != expected.Fingerprint || current.Authority != expected.Authority ||
		!sameControlOutcome(current, expected) {
		return ErrDispatchControlStale
	}
	return nil
}
