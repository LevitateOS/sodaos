package control

// Dispatch-side MCP capability producer (SUPERVISOR-031, lane-mcp-producer).
//
// Production serving (web/api FactoryMCPHandler), admission
// (AdmitMCPCapability, AdmitMCPSubmission) and durability (store
// Stage/Activate/Record) are wired but starved: no production caller
// ever stages or activates a capability, so every admission refuses
// inactive. This producer is the smallest P-gate-free joint that
// lights that path: one staged capability per dispatched run, built
// from already-recorded state only — the assignment, the
// packet-settled run, the recorded preparation digest and recorded
// stop-intent — then activated under the store's observed-binding
// guard. Alone it lights the full served path: all fourteen
// admissions, durable record, receipt and status readback.
//
// Deliberately OUT of scope (FW39 held intact): provider calls,
// relay transport, proof-of-possession secrets, permission-model or
// native-auth reads, host run observation, schema/DDL changes,
// revocation/retirement, catalog checks and every FW40+
// graph/checkpoint/review/adoption/verification effect. Production
// failures are best-effort: dispatch never fails a launch because
// capability production refused.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// mcpCapabilityGeneration orders re-deliveries of the same run.
// Dispatch produces exactly one capability per run, so the first
// generation is the only generation; exact restart delivery replays
// it identically, never a new deadline or allowance.
const mcpCapabilityGeneration = 1

// mcpProfileForPhase maps one dispatch execution phase to its
// admitted MCP profile. Phases and profiles share names by
// construction (see factory.MCPPhaseOf); the pre-phase legacy record
// and anything unmapped default by role: coder runs build, reviewer
// runs inspect. The verify profile has no dispatch phase and is
// never produced here.
func mcpProfileForPhase(phase, role string) factory.MCPProfile {
	switch phase {
	case project.ExecutionDecompose:
		return factory.MCPProfileProductLead
	case project.ExecutionInvestigate:
		return factory.MCPProfileArchitect
	case project.ExecutionImplement:
		return factory.MCPProfileBuilder
	case project.ExecutionReview:
		return factory.MCPProfileInspector
	case project.ExecutionCorrect:
		return factory.MCPProfileFixer
	case project.ExecutionReassess:
		return factory.MCPProfileStrategist
	}
	if role == project.RoleReviewer {
		return factory.MCPProfileInspector
	}
	return factory.MCPProfileBuilder
}

// mcpAuthorityDigest binds the recorded grant revisions behind one
// dispatch: the assignment authority fields plus the connection and
// actor the packet recorded. The digest carries revisions, never
// secret bytes, and changes whenever the recorded authority does,
// so a replayed capability cannot borrow edited authority.
func mcpAuthorityDigest(a factory.Assignment) string {
	authority := a.Authority
	fields := []string{
		authority.RequirementsID, authority.ApprovalID, authority.SponsorshipConnection,
		strconv.FormatInt(authority.Policy, 10),
		strconv.FormatInt(authority.Operator, 10),
		strconv.FormatInt(authority.Capacity, 10),
		strconv.FormatInt(authority.Sponsorship, 10),
		strconv.FormatInt(authority.ConnectionUsageBudget, 10),
		strconv.FormatInt(authority.Environment, 10),
		a.Connection,
		strconv.FormatInt(a.ActorID, 10),
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\n")))
	return hex.EncodeToString(sum[:])
}

// mcpSubjectForAssignment names the recorded scope the capability
// serves: the canonical subject key when the assignment carries one,
// otherwise the bare issue scope. Either way the value is short,
// non-empty and bound to the recorded repository and issue.
func mcpSubjectForAssignment(a factory.Assignment) string {
	if key := a.Subject.SubjectKey(a.Repository); key != "" {
		return key
	}
	return "issue:" + strconv.FormatInt(a.Repository, 10) + ":" + strconv.FormatInt(a.Issue, 10)
}

// BuildMCPCapability derives one staged capability from recorded
// state only: the assignment, its packet-settled run and the
// recorded preparation. The capability ID is the run identity, so
// one run owns exactly one capability and exact restart delivery
// replays the same staged bytes; a changed scope under that ID
// conflicts at staging instead of replacing authority. Accepted
// revision is the recorded native revision the assignment was
// admitted against; the deadline is the settled run deadline.
func BuildMCPCapability(a factory.Assignment, run factory.Run, preparation project.StoredPreparation) (factory.MCPCapability, error) {
	if !factory.ValidID(run.ID) {
		return factory.MCPCapability{}, errors.New("mcp producer refuses a run without recorded identity")
	}
	if !factory.ValidID(a.ID) || a.Repository < 1 {
		return factory.MCPCapability{}, errors.New("mcp producer refuses an assignment without recorded scope")
	}
	if !factory.ValidProjectID(a.ProjectID) {
		return factory.MCPCapability{}, errors.New("mcp producer refuses an assignment without recorded project")
	}
	setup := preparation.Preparation
	if !factory.ValidDigest(setup.SetupDigest) {
		return factory.MCPCapability{}, errors.New("mcp producer refuses a preparation without recorded digest")
	}
	source := a.SourceCommit
	if source == "" {
		source = run.InputSHA
	}
	if source == "" {
		return factory.MCPCapability{}, errors.New("mcp producer refuses a run without recorded source")
	}
	if run.Deadline.IsZero() {
		return factory.MCPCapability{}, errors.New("mcp producer refuses a run without recorded deadline")
	}
	subject := mcpSubjectForAssignment(a)
	if subject == "" || len(subject) > factory.MCPMaxRefChars {
		return factory.MCPCapability{}, errors.New("mcp producer refuses an unnameable recorded subject")
	}
	capability := factory.MCPCapability{
		ID:                run.ID,
		AssignmentID:      a.ID,
		RunID:             run.ID,
		Generation:        mcpCapabilityGeneration,
		Profile:           mcpProfileForPhase(a.Phase, a.Role),
		Subject:           subject,
		Repository:        a.Repository,
		Project:           a.ProjectID,
		PreparationDigest: setup.SetupDigest,
		SourceDigest:      source,
		AcceptedRevision:  strconv.FormatInt(a.NativeRev, 10),
		Authority:         mcpAuthorityDigest(a),
		DeadlineUnix:      run.Deadline.Unix(),
		State:             factory.MCPCapabilityStaged,
	}
	if err := capability.Validate(); err != nil {
		return factory.MCPCapability{}, err
	}
	return capability, nil
}

// StageAndActivateMCPCapability records one staged capability and
// moves it to active under the store's observed-binding guard. Stop
// intent stages but never activates: the staged row is retained as
// history and the activation refuses, so stop ordered before start
// permanently prevents admission. Exact restart replay succeeds:
// staging replays identically and an already-active identical scope
// reports success instead of a state refusal. A changed scope under
// the same ID conflicts, and that refusal is preserved.
func StageAndActivateMCPCapability(ctx context.Context, st *store.Store, capability factory.MCPCapability, stopped bool, now time.Time) error {
	if st == nil {
		return errors.New("mcp producer has no store")
	}
	if err := capability.Validate(); err != nil {
		return err
	}
	if capability.State != factory.MCPCapabilityStaged {
		return errors.New("mcp producer stages only staged capabilities")
	}
	if err := st.StageMCPCapability(ctx, capability); err != nil {
		return err
	}
	if stopped {
		return ErrMCPCapabilityStopped
	}
	if now.IsZero() {
		now = time.Now()
	}
	if err := st.ActivateMCPCapability(ctx, capability.ID, capability.Generation,
		capability.RunID, capability.AssignmentID, capability.Authority, now.Unix(), false); err != nil {
		if errors.Is(err, store.ErrMCPCapabilityState) {
			if recorded, rerr := st.GetMCPCapability(ctx, capability.ID); rerr == nil &&
				recorded.State == factory.MCPCapabilityActive &&
				recorded.ScopeDigest() == capability.ScopeDigest() {
				return nil
			}
		}
		return err
	}
	return nil
}

// recordedMCPPreparation loads the recorded preparation digest and
// stop state behind one assignment: the standing preparation first,
// then the retry-custody candidate record that shares the same table.
// Both are durable recorded state; no host observation is consulted.
func recordedMCPPreparation(ctx context.Context, st *store.Store, id string) (project.StoredPreparation, bool, error) {
	if prep, err := st.Preparation(ctx, id); err == nil {
		return prep, prep.State.Stopped, nil
	} else if record, cerr := st.CandidatePreparation(ctx, id); cerr == nil {
		return record.Preparation, record.Stopped || record.Preparation.State.Stopped, nil
	} else {
		return project.StoredPreparation{}, false, err
	}
}

// ProduceMCPCapabilityForLaunch stages and activates the capability
// for one packet-settled launch from recorded state only. Stop
// intent is the recorded preparation stop plus the recorded
// dispatch-gate closure: either stages without activating. An
// unreadable stop signal refuses production rather than guessing.
// Dispatch calls this best-effort and ignores refusal: a refused
// capability never fails its launch.
func ProduceMCPCapabilityForLaunch(ctx context.Context, st *store.Store, a factory.Assignment, run factory.Run) error {
	if st == nil {
		return errors.New("mcp producer has no store")
	}
	preparation, stopped, err := recordedMCPPreparation(ctx, st, a.Preparation)
	if err != nil {
		return err
	}
	open, _, _, derr := st.DispatchState(ctx, a.Repository)
	if derr != nil {
		return derr
	}
	if !open {
		stopped = true
	}
	capability, err := BuildMCPCapability(a, run, preparation)
	if err != nil {
		return err
	}
	return StageAndActivateMCPCapability(ctx, st, capability, stopped, time.Now())
}
