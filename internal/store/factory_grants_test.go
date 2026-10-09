package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func grantTestTime() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }

func grantStoreFixture(t *testing.T) *Store {
	t.Helper()
	db, _ := postgresFixture(t, nil)
	return db
}

func grantTestPolicy() factory.RepositoryPolicy {
	return factory.RepositoryPolicy{
		Repository: 42, GrantedBy: 7, Enabled: true,
		TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
			project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
		},
		Checks:        []string{"native-ci/build"},
		MergeMethod:   factory.MergeFastForward,
		Publish:       factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:        factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:        factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:         factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		AttemptLimits: factory.DefaultAttemptLimits(),
		MaxConcurrent: 2,
	}
}

func TestPolicySaveIsCAS(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	if err := db.SaveRepositoryPolicy(ctx, grantTestPolicy()); err != nil {
		t.Fatal(err)
	}
	stored, err := db.RepositoryPolicy(ctx, 42)
	if err != nil || stored.Revision != 1 || !stored.Enabled {
		t.Fatalf("policy: %+v %v", stored, err)
	}
	stale := grantTestPolicy()
	stale.Paused = true
	if err = db.SaveRepositoryPolicy(ctx, stale); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale policy revision saved", err)
	}
	stored.Paused = true
	if err = db.SaveRepositoryPolicy(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if _, err = db.RepositoryPolicy(ctx, 43); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing policy reported", err)
	}
}

func TestCapacityAndOperatorGrant(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	if _, err := db.Capacity(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing capacity reported", err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2}); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale capacity saved", err)
	}
	grant := factory.OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}
	if err := db.SaveOperatorGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	grant.Active = false
	if err := db.SaveOperatorGrant(ctx, grant); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale operator grant saved", err)
	}
}

func TestSponsorships(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	sp := factory.Sponsorship{
		Repository: 42, GrantedBy: 9, Connection: "conn-1", GrantID: "grant-1",
		Generation: 3, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
	}
	if err := db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	stored, err := db.Sponsorship(ctx, 42, "conn-1")
	if err != nil || stored.Revision != 1 {
		t.Fatalf("sponsorship: %+v %v", stored, err)
	}
	list, err := db.Sponsorships(ctx, 42)
	if err != nil || len(list) != 1 {
		t.Fatalf("sponsorships: %+v %v", list, err)
	}
	sp.Active = false
	if err = db.SaveSponsorship(ctx, sp); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale sponsorship saved", err)
	}
}

func TestDispatchWithdrawalOrdering(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	open, revision, _, err := db.DispatchState(ctx, 42)
	if err != nil || !open || revision != 0 {
		t.Fatal("fresh dispatch is not open", open, revision, err)
	}
	first := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1}}
	second := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1}}
	if err = db.RegisterDispatch(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err = db.RegisterDispatch(ctx, first); err != nil {
		t.Fatal("identical registration refused", err)
	}
	changed := first
	changed.Revision = 9
	if err = db.RegisterDispatch(ctx, changed); !errors.Is(err, ErrDispatchConflict) {
		t.Fatal("changed registration accepted", err)
	}
	if err = db.RegisterDispatch(ctx, second); err != nil {
		t.Fatal(err)
	}
	withdrawal, err := db.WithdrawDispatch(ctx, 42, "policy_paused", "native:7")
	if err != nil || len(withdrawal.Captured) != 2 || withdrawal.Captured[0] != first.ID || withdrawal.Revision != 1 {
		t.Fatalf("withdrawal: %+v %v", withdrawal, err)
	}
	late := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42}
	if err = db.RegisterDispatch(ctx, late); !errors.Is(err, ErrDispatchClosed) {
		t.Fatal("late registration escaped withdrawal", err)
	}
	replay, err := db.WithdrawDispatch(ctx, 42, "policy_paused", "native:7")
	if err != nil || len(replay.Captured) != 2 || replay.Revision != 1 {
		t.Fatalf("withdrawal replay: %+v %v", replay, err)
	}
	if _, _, err = db.ReopenDispatch(ctx, 42, 0); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale reopen accepted", err)
	}
	if opened, revision, reopenErr := db.ReopenDispatch(ctx, 42, 1); reopenErr != nil || !opened || revision != 2 {
		t.Fatal(opened, revision, reopenErr)
	}
	if err = db.RegisterDispatch(ctx, late); err != nil {
		t.Fatal("registration refused after reopen", err)
	}
}

func TestDispatchCausesComposeAndClearIndependently(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	first, err := db.WithdrawDispatch(ctx, 42, factory.CauseProjectStop, "native:7")
	if err != nil || first.Revision != 1 || len(first.ActiveCauses) != 1 {
		t.Fatalf("first cause: %+v %v", first, err)
	}
	second, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "native:8")
	if err != nil || second.Revision != first.Revision || second.Cause != first.Cause || second.ClosedBy != first.ClosedBy || len(second.ActiveCauses) != 2 {
		t.Fatalf("second cause changed first-closure receipt: %+v %v", second, err)
	}
	open, revision, state, err := db.DispatchState(ctx, 42)
	if err != nil || open || revision != 2 || len(state.Captured) != 0 || len(state.ActiveCauses) != 2 {
		t.Fatalf("composed state: %v %d %+v %v", open, revision, state, err)
	}
	duplicate, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "different-principal")
	if err != nil || len(duplicate.ActiveCauses) != 2 {
		t.Fatalf("duplicate cause changed state: %+v %v", duplicate, err)
	}
	open, revision, err = db.ClearProjectStop(ctx, 42, revision)
	if err != nil || open || revision != 3 {
		t.Fatalf("clearing Project stop cleared independent pause: %v %d %v", open, revision, err)
	}
	open, revision, err = db.ReopenDispatch(ctx, 42, revision)
	if err != nil || !open || revision != 4 {
		t.Fatalf("resume did not clear final cause: %v %d %v", open, revision, err)
	}
	open, revision, state, err = db.DispatchState(ctx, 42)
	if err != nil || !open || revision != 4 || len(state.ActiveCauses) != 0 || state.Cause != factory.CauseProjectStop {
		t.Fatalf("open state lost first-closure receipt or retained causes: %v %d %+v %v", open, revision, state, err)
	}
}

func TestProjectStopCauseComposesAfterExistingPause(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	first, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "native:7")
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.WithdrawDispatch(ctx, 42, factory.CauseProjectStop, "native:8")
	if err != nil || second.Cause != first.Cause || second.ClosedBy != first.ClosedBy || len(second.ActiveCauses) != 2 {
		t.Fatalf("Project stop overwrote the first pause receipt: %+v %v", second, err)
	}
	open, revision, err := db.ClearProjectStop(ctx, 42, 2)
	if err != nil || open || revision != 3 {
		t.Fatalf("Start cleared the independent pause: %v %d %v", open, revision, err)
	}
	open, revision, _, err = db.DispatchState(ctx, 42)
	if err != nil || open || revision != 3 {
		t.Fatalf("pause did not remain active: %v %d %v", open, revision, err)
	}
	open, revision, err = db.ReopenDispatch(ctx, 42, revision)
	if err != nil || !open || revision != 4 {
		t.Fatalf("authorized Resume did not clear the final pause: %v %d %v", open, revision, err)
	}
}

func TestSettingsCommandReplay(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	payload := `{"enabled":true}`
	cmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandPolicy, Target: "repository/42/policy",
		Principal: "native:7", Payload: payload, Digest: factory.SettingsDigest(factory.CommandPolicy, "repository/42/policy", payload),
	}
	stored, created, err := db.RecordFactoryCommand(ctx, cmd, grantTestTime())
	if err != nil || !created || stored.Payload != payload {
		t.Fatalf("command: %+v %v %v", stored, created, err)
	}
	again, created, err := db.RecordFactoryCommand(ctx, cmd, grantTestTime())
	if err != nil || created || again.ID != cmd.ID {
		t.Fatalf("replay: %+v %v %v", again, created, err)
	}
	changed := cmd
	changed.Payload = `{"enabled":false}`
	changed.Digest = factory.SettingsDigest(changed.Type, changed.Target, changed.Payload)
	if _, _, err = db.RecordFactoryCommand(ctx, changed, grantTestTime()); !errors.Is(err, ErrCommandConflict) {
		t.Fatal("changed payload reused the command identity", err)
	}
}

func TestPolicyCommandCommitsReceiptAndCancellationIntentsTogether(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	registration := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1}}
	if err := db.RegisterDispatch(ctx, registration); err != nil {
		t.Fatal("register dispatch", err)
	}
	publication := publicationTestRecord()
	publication.Repository = 42
	if err := db.RecordPublication(ctx, publication); err != nil {
		t.Fatal("record publication", err)
	}
	merge := mergeTestRecord()
	merge.Repository = 42
	if err := db.RecordMerge(ctx, merge); err != nil {
		t.Fatal("record merge", err)
	}

	policy := grantTestPolicy()
	policy.Paused = true
	payload, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	target := "repository/42/policy"
	cmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandPolicy, Target: target, Principal: "native:7",
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandPolicy, target, string(payload)),
	}
	stored, created, err := db.ApplyRepositoryPolicyCommand(ctx, cmd, policy, grantTestTime())
	if err != nil || !created || stored.Finished == "" {
		t.Fatalf("apply policy command: %+v created=%t err=%v", stored, created, err)
	}
	var receipt factory.GrantReceipt
	if err = json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		t.Fatal("decode durable receipt", err)
	}
	if !receipt.Withdrawn || receipt.Revision != 1 || len(receipt.Captured) != 1 || receipt.Captured[0] != registration.ID {
		t.Fatalf("receipt omitted policy withdrawal: %+v", receipt)
	}
	if !receipt.PublicationsPending {
		t.Fatalf("receipt omitted publication intent: %+v", receipt)
	}
	if !receipt.MergesPending {
		t.Fatalf("receipt omitted merge intent: %+v", receipt)
	}
	if len(stored.Outcome) > 64<<10 {
		t.Fatalf("bounded grant receipt grew to %d bytes", len(stored.Outcome))
	}
	withdrawnPublication, err := db.PublicationByAssignment(ctx, publication.AssignmentID)
	if err != nil || !withdrawnPublication.WithdrawRequested {
		t.Fatalf("publication intent not committed: %+v %v", withdrawnPublication, err)
	}
	withdrawnMerge, err := db.MergeByPublication(ctx, merge.PublicationID)
	if err != nil || !withdrawnMerge.WithdrawRequested {
		t.Fatalf("merge intent not committed: %+v %v", withdrawnMerge, err)
	}
	replay, created, err := db.ApplyRepositoryPolicyCommand(ctx, cmd, policy, grantTestTime().Add(time.Minute))
	if err != nil || created || replay.Outcome != stored.Outcome || replay.Finished != stored.Finished {
		t.Fatalf("command replay changed immutable receipt: %+v created=%t err=%v", replay, created, err)
	}
}

func TestPolicyReceiptFailureDoesNotCommitPartialWithdrawal(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	if err := db.SaveRepositoryPolicy(ctx, grantTestPolicy()); err != nil {
		t.Fatal("save initial policy", err)
	}
	registration := factory.DispatchRegistration{
		ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1},
	}
	if err := db.RegisterDispatch(ctx, registration); err != nil {
		t.Fatal("register dispatch", err)
	}
	_, err := db.db.ExecContext(ctx, `CREATE FUNCTION refuse_factory_command_receipt() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.finished <> '' THEN
    RAISE EXCEPTION 'injected command receipt write failure';
  END IF;
  RETURN NEW;
END $$;`)
	if err != nil {
		t.Fatal("install isolated command receipt failure trigger", err)
	}
	_, err = db.db.ExecContext(ctx, `CREATE TRIGGER refuse_factory_command_receipt BEFORE INSERT OR UPDATE ON factory_commands FOR EACH ROW EXECUTE FUNCTION refuse_factory_command_receipt()`)
	if err != nil {
		t.Fatal("install isolated command receipt failure trigger", err)
	}

	commandID := factory.NewID()
	paused := grantTestPolicy()
	paused.Revision, paused.Paused = 1, true
	payload, err := json.Marshal(paused)
	if err != nil {
		t.Fatal("marshal paused policy", err)
	}
	target := "repository/42/policy"
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandPolicy, Target: target, Principal: "native:7",
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandPolicy, target, string(payload)),
	}
	_, _, err = db.ApplyRepositoryPolicyCommand(ctx, cmd, paused, grantTestTime())
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Message != "injected command receipt write failure" {
		t.Fatalf("policy withdrawal did not reach the injected receipt failure: %v", err)
	}

	storedPolicy, err := db.RepositoryPolicy(ctx, 42)
	if err != nil {
		t.Fatal("read policy after failed withdrawal", err)
	}
	if storedPolicy.Revision != 1 || storedPolicy.Paused || !storedPolicy.Enabled {
		t.Errorf("policy changed without a durable command receipt: %+v", storedPolicy)
	}
	open, _, _, err := db.DispatchState(ctx, 42)
	if err != nil {
		t.Fatal("read dispatch after failed withdrawal", err)
	}
	if !open {
		t.Error("dispatch closed without a durable command receipt")
	}
	if _, err := db.FactoryCommand(ctx, commandID); !errors.Is(err, ErrNotFound) {
		t.Errorf("failed atomic command remained recorded: err=%v", err)
	}
}

func TestPolicyCommandReceiptStaysBoundedAcrossOutstandingWork(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	for i := 0; i < factory.MaxCapturedDispatch; i++ {
		registration := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42}
		if err := db.RegisterDispatch(ctx, registration); err != nil {
			t.Fatalf("register dispatch %d: %v", i, err)
		}
	}
	for i := 0; i < storePublicationLimit; i++ {
		publication := publicationTestRecord()
		publication.Repository = 42
		if err := db.RecordPublication(ctx, publication); err != nil {
			t.Fatalf("record publication %d: %v", i, err)
		}
	}
	for i := 0; i < storeMergeLimit; i++ {
		merge := mergeTestRecord()
		merge.Repository = 42
		if err := db.RecordMerge(ctx, merge); err != nil {
			t.Fatalf("record merge %d: %v", i, err)
		}
	}

	policy := grantTestPolicy()
	policy.Paused = true
	payload, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	target := "repository/42/policy"
	cmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandPolicy, Target: target, Principal: "native:7",
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandPolicy, target, string(payload)),
	}
	stored, created, err := db.ApplyRepositoryPolicyCommand(ctx, cmd, policy, grantTestTime())
	if err != nil || !created || stored.Finished == "" {
		t.Fatalf("apply bounded policy command: %+v created=%t err=%v", stored, created, err)
	}
	var receipt factory.GrantReceipt
	if err = json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		t.Fatal("decode durable receipt", err)
	}
	if len(receipt.Captured) != factory.MaxCapturedDispatch || !receipt.PublicationsPending || !receipt.MergesPending {
		t.Fatalf("receipt omitted bounded outstanding work: captured=%d publications=%t merges=%t", len(receipt.Captured), receipt.PublicationsPending, receipt.MergesPending)
	}
	if len(stored.Outcome) > 64<<10 {
		t.Fatalf("maximum-work grant receipt grew to %d bytes", len(stored.Outcome))
	}
	policyAfter, err := db.RepositoryPolicy(ctx, 42)
	if err != nil || !policyAfter.Paused || policyAfter.Revision != 1 {
		t.Fatalf("policy decision did not commit with bounded receipt: %+v %v", policyAfter, err)
	}
	open, _, _, err := db.DispatchState(ctx, 42)
	if err != nil || open {
		t.Fatalf("dispatch gate did not close with bounded receipt: open=%t err=%v", open, err)
	}
	var publicationCount, mergeCount int
	if err = db.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_publications WHERE repository=42 AND COALESCE((data->>'withdraw_requested')::boolean,FALSE)`).Scan(&publicationCount); err != nil {
		t.Fatal("count latched publication intents", err)
	}
	if err = db.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_merges WHERE repository=42 AND COALESCE((data->>'withdraw_requested')::boolean,FALSE)`).Scan(&mergeCount); err != nil {
		t.Fatal("count latched merge intents", err)
	}
	if publicationCount != storePublicationLimit || mergeCount != storeMergeLimit {
		t.Fatalf("atomic cancellation intent counts: publications=%d merges=%d", publicationCount, mergeCount)
	}
}
