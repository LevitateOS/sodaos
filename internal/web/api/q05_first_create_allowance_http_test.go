package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// The shared service uses only the real store and an admitted authority port;
// no appliance Project, browser session or native callback is constructed here.
func TestSharedRepairFundingAdmission(t *testing.T) {
	ctx := t.Context()
	_, db, dsn, _, _ := q05MuxFull(t, "")
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	const sponsor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	const principal = "profile:repair-sponsor"
	viewer, checks, denyAt := sponsor, 0, 0
	denied := errors.New("declared current repair administration denied")
	var cancelAtFinal context.CancelFunc
	application := factoryservice.RepairFunding{Store: db, Administration: func(ctx context.Context, actor collaboration.ActorID, repo int64) error {
		if actor != viewer || repo != int64(repository) {
			t.Fatal("repair funding changed canonical authority subject", actor, repo)
		}
		checks++
		if checks == denyAt {
			return denied
		}
		if checks == 2 && cancelAtFinal != nil {
			cancelAtFinal()
		}
		return nil
	}}
	read := func() *factoryservice.RepairFundingView {
		t.Helper()
		checks = 0
		view, err := application.Read(ctx, viewer, repository)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	apply := func(in factoryservice.RepairFundingChange) (factory.GrantResult, error) {
		checks = 0
		return application.Apply(ctx, viewer, principal, repository, in)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),
		 (SELECT count(*) FROM factory_runs),(SELECT count(*) FROM factory_usage),
		 (SELECT count(*) FROM factory_stall_events),(SELECT count(*) FROM projects))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	view := read()
	if view.Revision != 0 || view.Grant != nil || view.Charges == nil || len(view.Charges) != 0 || view.Episodes == nil || len(view.Episodes) != 0 {
		t.Fatal("known unfunded state became funded or ambiguous", view)
	}
	first := factoryservice.RepairFundingChange{CommandID: factory.NewID(), InvestigatorRuns: 1, ProposalJudgments: 1, ActiveMinutes: 30, Active: true}
	result, err := apply(first)
	if err != nil || result.CommandID != first.CommandID || result.Revision != 1 {
		t.Fatal("initial shared funding", result, err)
	}
	view = read()
	if view.Grant == nil || view.Grant.Repository != int64(repository) || view.Grant.Sponsor != int64(sponsor) || view.Grant.Revision != 0 || view.Revision != 1 {
		t.Fatal("funding lost canonical sponsor/repository", view)
	}
	command, err := db.FactoryCommand(ctx, first.CommandID)
	if err != nil || command.Principal != principal || command.Outcome != "" || command.Finished == "" {
		t.Fatal("funding changed principal or saved a response", command, err)
	}
	firstGrant := *view.Grant
	before := snapshot()
	if replay, err := apply(first); err != nil || replay.CommandID != result.CommandID || replay.Revision != result.Revision || snapshot() != before {
		t.Fatal("exact funding replay changed captured command", replay, err)
	}
	charge := factory.RepairCharge{ID: factory.NewID(), GrantID: first.CommandID, Kind: factory.RepairChargeRun, EpisodeID: factory.NewID(), Cycle: 1,
		EvidenceDigest: strings.Repeat("a", 64), ActiveMinutes: 10, SpentUnix: time.Now().Unix()}
	if spent, err := db.AdmitStallInvestigation(ctx, int64(repository), charge, time.Now()); err != nil || !spent {
		t.Fatal("actual first repair charge", spent, err)
	}
	raised := first
	raised.CommandID, raised.ExpectedRevision, raised.InvestigatorRuns, raised.ProposalJudgments, raised.ActiveMinutes = factory.NewID(), 1, 2, 2, 60
	before = snapshot()
	if _, err := apply(raised); !errors.Is(err, factoryservice.ErrRepairFundingIncrease) || snapshot() != before {
		t.Fatal("implicit funding increase admitted", err)
	}
	raised.Effect = "Sponsor explicitly raises finite limits without refilling consumption."
	if result, err := apply(raised); err != nil || result.Revision != 2 {
		t.Fatal("explicit funding increase", result, err)
	}
	view = read()
	if view.Grant.Predecessor != first.CommandID || len(view.Charges) != 1 || view.Charges[0].ID != charge.ID || view.Charges[0].GrantID != first.CommandID {
		t.Fatal("increase lost predecessor or charged history", view)
	}
	secondCharge := charge
	secondCharge.ID, secondCharge.GrantID, secondCharge.EpisodeID, secondCharge.EvidenceDigest = factory.NewID(), raised.CommandID, factory.NewID(), strings.Repeat("b", 64)
	if spent, err := db.AdmitStallInvestigation(ctx, int64(repository), secondCharge, time.Now()); err != nil || !spent {
		t.Fatal("actual raised repair charge", spent, err)
	}
	excess := secondCharge
	excess.ID, excess.EpisodeID, excess.EvidenceDigest = factory.NewID(), factory.NewID(), strings.Repeat("c", 64)
	if spent, err := db.AdmitStallInvestigation(ctx, int64(repository), excess, time.Now()); !errors.Is(err, factory.ErrRepairRunsExhausted) || spent {
		t.Fatal("raise refilled prior charged investigation", spent, err)
	}
	before = snapshot()
	if replay, err := apply(first); err != nil || replay.Revision != 1 || snapshot() != before {
		t.Fatal("old exact replay moved the current head or timestamp", replay, err)
	}
	view = read()
	if view.Revision != 2 || len(view.Charges) != 2 || view.Charges[0].ActiveMinutes+view.Charges[1].ActiveMinutes != 20 {
		t.Fatal("old replay reset funding history", view)
	}
	storedFirst, err := db.FactoryCommand(ctx, first.CommandID)
	var retainedFirst factory.RepairGrant
	if err != nil || json.Unmarshal([]byte(storedFirst.Payload), &retainedFirst) != nil || retainedFirst != firstGrant {
		t.Fatal("exact replay recaptured initial grant", retainedFirst, err)
	}
	changed := first
	changed.ActiveMinutes = 29
	if _, err := apply(changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("changed captured command replayed", err)
	}
	if _, err := application.Apply(ctx, viewer, "profile:another-owner", repository, first); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("other principal replayed funding", err)
	}
	stale := first
	stale.CommandID = factory.NewID()
	if _, err := apply(stale); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("stale repair head admitted", err)
	}
	viewer = sponsor + 2
	if other := read(); other.Grant.Sponsor != int64(sponsor) {
		t.Fatal("current administrator changed recorded sponsor", other)
	}
	other := raised
	other.CommandID, other.ExpectedRevision = factory.NewID(), 2
	if _, err := apply(other); !errors.Is(err, factoryservice.ErrRepairFundingSponsor) {
		t.Fatal("another administrator inherited immutable funding", err)
	}
	viewer = sponsor
	checks, denyAt = 0, 2
	if view, err := application.Read(ctx, viewer, repository); view != nil || !errors.Is(err, denied) {
		t.Fatal("late admin denial returned funding history", view, err)
	}
	if _, err := apply(first); !errors.Is(err, denied) {
		t.Fatal("captured replay bypassed late admin denial", err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancelAtFinal = cancel
	checks = 0
	if _, err := application.Apply(cancelled, viewer, principal, repository, first); !errors.Is(err, context.Canceled) {
		t.Fatal("late cancellation reached funding write", err)
	}
	cancelAtFinal = nil
	invalid := other
	invalid.ActiveMinutes, invalid.Effect = factory.DefaultRepairActiveMinutes+1, "Explicit but out of bounds."
	if _, err := apply(invalid); !errors.Is(err, factoryservice.ErrRepairFundingInvalid) {
		t.Fatal("unbounded repair funding admitted", err)
	}
	missing := application
	missing.Administration = nil
	if _, err := missing.Apply(ctx, viewer, principal, repository, first); !errors.Is(err, factoryservice.ErrRepairFundingUnavailable) {
		t.Fatal("missing administration replayed funding", err)
	}
	if view, err := missing.Read(ctx, viewer, repository); view != nil || !errors.Is(err, factoryservice.ErrRepairFundingUnavailable) {
		t.Fatal("missing administration read private history", view, err)
	}
	if snapshot() != before {
		t.Fatal("refused funding operations changed durable records")
	}
	// The database itself refuses captured-command mutation. Keep that guard
	// intact while testing a real SQL read failure against the private fixture.
	if _, err := raw.ExecContext(ctx, `UPDATE factory_commands SET payload='not-json' WHERE id=$1`, raised.CommandID); err == nil {
		t.Fatal("captured repair command was mutable")
	}
	if _, err := raw.ExecContext(ctx, `ALTER TABLE factory_commands RENAME TO repair_funding_unavailable_commands`); err != nil {
		t.Fatal(err)
	}
	if view, err := application.Read(ctx, viewer, repository); view != nil || err == nil {
		t.Fatal("unknown grant returned an unfunded view", view, err)
	}
	if _, err := apply(other); err == nil {
		t.Fatal("unknown current grant admitted a replacement", err)
	}
	if _, err := raw.ExecContext(ctx, `ALTER TABLE repair_funding_unavailable_commands RENAME TO factory_commands`); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before {
		t.Fatal("restored fixture changed funding records")
	}
	withdraw := raised
	withdraw.CommandID, withdraw.ExpectedRevision, withdraw.Active, withdraw.Effect = factory.NewID(), 2, false, ""
	if result, err := apply(withdraw); err != nil || result.Revision != 3 {
		t.Fatal("repair funding withdrawal", result, err)
	}
	excess.GrantID = withdraw.CommandID
	if spent, err := db.AdmitStallInvestigation(ctx, int64(repository), excess, time.Now()); !errors.Is(err, factory.ErrRepairGrantInactive) || spent {
		t.Fatal("withdrawn repair permission still charged", spent, err)
	}
	before = snapshot()
	retained := read()
	beforeJSON, _ := json.Marshal(retained)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	application.Store = reopened
	if replay, err := apply(first); err != nil || replay.Revision != 1 {
		t.Fatal("store reopen lost captured exact replay", replay, err)
	}
	retained = read()
	afterJSON, _ := json.Marshal(retained)
	if string(beforeJSON) != string(afterJSON) || snapshot() != before || retained.Grant.Active || len(retained.Charges) != 2 {
		t.Fatal("store reopen changed head, withdrawal or charged history")
	}
	var runs, usage, projects int
	if err := raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_runs),(SELECT count(*) FROM factory_usage),(SELECT count(*) FROM projects)`).Scan(&runs, &usage, &projects); err != nil || runs != 0 || usage != 0 || projects != 0 {
		t.Fatal("funding started work or required native Project", runs, usage, projects, err)
	}
}

func TestRepairFundingRejectsDifferentSponsor(t *testing.T) {
	product, db, _, call, errorCode := q05MuxFull(t, "")
	path := "/api/repositories/7/factory/repair"
	first := `{"command_id":"` + factory.NewID() + `","expected_revision":0,"investigator_runs":1,"proposal_judgments":1,"active_minutes":30,"active":true}`
	if code, body := call(http.MethodPut, path, first); code != 200 {
		t.Fatalf("initial repair grant: %d %s", code, body)
	}
	other := actorCall(t, product, "2", "bob")
	change := `{"command_id":"` + factory.NewID() + `","expected_revision":1,"investigator_runs":1,"proposal_judgments":1,"active_minutes":30,"active":true}`
	if code, body := other(http.MethodPut, path, change); code != 403 || errorCode(body) != "repair_sponsor_required" {
		t.Fatalf("different current administrator did not receive a sponsor refusal: %d %s", code, body)
	}
	grant, err := db.RepairGrantCommand(t.Context(), 7)
	if err != nil || grant.Sponsor != 1 || grant.Revision != 0 {
		t.Fatal("refused repair command changed funding", grant, err)
	}
}

func TestSharedVerificationAllowanceAdmission(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("native PostgreSQL fixture is required")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	inspect, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = inspect.Close() })
	const actor collaboration.ActorID = 9007199254740993
	const firstRepository int64 = 9007199254740995
	const secondRepository int64 = 9223372036854775807
	idea := factory.ProductIdea{ID: "d818181818181818181818181", Description: "Keep a task", Consequence: "Tasks persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: int64(actor)}
	task := factory.IntentTask{Key: "d828282828282828282828282", Objective: "Store tasks", Outcomes: []string{idea.Consequence}, IdeaIDs: []string{idea.ID}}
	head := factory.IntentRevision{ID: "d838383838383838383838383", Intent: "d848484848484848484848484", Owner: int64(actor), Repository: firstRepository, Objective: "Tasks", State: "draft", MaxIssues: 1, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
	if err := db.SaveIntentRevision(ctx, head); err != nil {
		t.Fatal(err)
	}
	checks := 0
	requests := factoryservice.Requests{Store: db, WriteAccess: func(ctx context.Context, owner collaboration.ActorID, repository int64) error {
		checks++
		if owner != actor || repository != firstRepository {
			t.Fatalf("wrong authority subject: %d %d", owner, repository)
		}
		// A different admitted request command commits a new binding while
		// this command is awaiting the old repository's access response.
		head.Predecessor, head.ID, head.Repository, head.State = head.ID, "d858585858585858585858585", secondRepository, "accepted"
		return db.SaveIntentRevision(ctx, head)
	}}
	change := factory.AppVerificationAllowanceChange{CommandID: factory.NewID(), Limits: factory.AppVerificationLimits{Runs: 3, ActiveMinutes: 60}}
	if view, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", change); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("grant crossed the repository checked for authority: %+v %v", view, err)
	}
	if checks != 1 {
		t.Fatalf("scope race silently retried admission: %d", checks)
	}
	if _, err := db.FactoryCommand(ctx, change.CommandID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("scope race retained a grant: %v", err)
	}
	var accessErr error
	denied := errors.New("declared current repository authority revoked")
	requests.WriteAccess = func(ctx context.Context, owner collaboration.ActorID, repository int64) error {
		checks++
		if owner != actor || repository != secondRepository {
			t.Fatalf("canonical allowance authority subject changed: %d %d", owner, repository)
		}
		return accessErr
	}
	for _, read := range []bool{true, false} {
		before := checks
		var err error
		if read {
			_, err = requests.VerificationAllowance(ctx, actor+1, head.Intent)
		} else {
			_, err = requests.IncreaseVerificationAllowance(ctx, actor+1, head.Intent, "profile:foreign", change)
		}
		if !errors.Is(err, factory.ErrVerificationAllowanceOwner) || checks != before {
			t.Fatalf("foreign owner reached repository admission: %v checks=%d", err, checks-before)
		}
	}
	unwired := factoryservice.Requests{Store: db}
	if view, err := unwired.VerificationAllowance(ctx, actor, head.Intent); !errors.Is(err, factoryservice.ErrRequestWriteUnavailable) || view.Intent != "" {
		t.Fatalf("bound read without current authority: %+v %v", view, err)
	}
	if _, err := unwired.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", change); !errors.Is(err, factoryservice.ErrRequestWriteUnavailable) {
		t.Fatalf("unwired increase: %v", err)
	}
	initial, err := requests.VerificationAllowance(ctx, actor, head.Intent)
	if err != nil || initial.Revision != "" || initial.Repository != secondRepository || initial.Limits.Runs != 2 {
		t.Fatalf("shared allowance read: %+v %v", initial, err)
	}
	raised, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", change)
	if err != nil || raised.Repository != secondRepository || raised.Revision != change.CommandID || raised.Limits.Runs != 3 || raised.Consumption != initial.Consumption {
		t.Fatalf("admitted canonical increase: %+v %v", raised, err)
	}
	command, err := db.FactoryCommand(ctx, change.CommandID)
	if err != nil || command.Principal != "profile:local-owner" || command.Finished == "" || command.Outcome != "" {
		t.Fatalf("neutral principal command journal: %+v %v", command, err)
	}
	second := change
	second.CommandID, second.ExpectedRevision, second.Limits.Runs = factory.NewID(), change.CommandID, 4
	if _, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", second); err != nil {
		t.Fatal(err)
	}
	replay, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:reconnected-owner", change)
	if err != nil || replay.Revision != second.CommandID || replay.CommandID != change.CommandID || replay.Limits.Runs != 4 || replay.Consumption != initial.Consumption {
		t.Fatalf("saved command did not return current limits/usage: %+v %v", replay, err)
	}
	command, err = db.FactoryCommand(ctx, change.CommandID)
	if err != nil || command.Principal != "profile:local-owner" {
		t.Fatalf("replay rewrote captured principal: %+v %v", command, err)
	}
	accessErr = denied
	if view, err := requests.VerificationAllowance(ctx, actor, head.Intent); !errors.Is(err, denied) || view.Intent != "" {
		t.Fatalf("revoked read leaked private allowance: %+v %v", view, err)
	}
	if view, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", change); !errors.Is(err, denied) || view.Intent != "" {
		t.Fatalf("known command bypassed current authority: %+v %v", view, err)
	}
	accessErr = nil
	stale := change
	stale.CommandID, stale.ExpectedRevision, stale.Limits.Runs = factory.NewID(), change.CommandID, 5
	if _, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", stale); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("allowance predecessor CAS lost: %v", err)
	}
	changed := change
	changed.Limits.Runs = 5
	if _, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatalf("command content conflict lost: %v", err)
	}
	decrease := change
	decrease.CommandID, decrease.ExpectedRevision = factory.NewID(), second.CommandID
	if _, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:local-owner", decrease); !errors.Is(err, factory.ErrVerificationAllowanceIncrease) {
		t.Fatalf("limits decreased: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := requests.VerificationAllowance(cancelled, actor, head.Intent); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled allowance read: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	db, requests.Store = reopened, reopened
	current, err := requests.VerificationAllowance(ctx, actor, head.Intent)
	if err != nil || current.Revision != second.CommandID || current.CommandID != "" || current.Limits.Runs != 4 || current.Consumption != initial.Consumption {
		t.Fatalf("reopened service lost current allowance: %+v %v", current, err)
	}
	recovered, err := requests.IncreaseVerificationAllowance(ctx, actor, head.Intent, "profile:reconnected-owner", change)
	if err != nil || recovered.Revision != current.Revision || recovered.Consumption != current.Consumption {
		t.Fatalf("reopened replay: %+v %v", recovered, err)
	}
	var admitted atomic.Bool
	admitted.Store(true)
	var currentCalls, loseAt atomic.Int64
	transport := factoryhttpapi.Requests{Service: requests,
		Host: collaboration.HostID(factory.NewID()), Actor: actor, Principal: "profile:http-owner",
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			check := currentCalls.Add(1)
			if !admitted.Load() || check == loseAt.Load() || r.Header.Get("Authorization") != "Bearer declared-allowance-client" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
		AuthorityError: func(w http.ResponseWriter, err error) { w.WriteHeader(http.StatusForbidden) },
	}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	call := func(target *httptest.Server, method, data, suffix string, expected int) []byte {
		t.Helper()
		currentCalls.Store(0)
		path := target.URL + "/api/factory/intents/" + head.Intent + "/verification-allowance" + suffix
		req, err := http.NewRequestWithContext(ctx, method, path, strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-allowance-client")
		res, err := target.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != expected || expected == 200 && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("shared allowance HTTP", method, res.StatusCode, string(body), err)
		}
		return body
	}
	var projected factory.AppVerificationAllowance
	decode := func(body []byte) factory.AppVerificationAllowance {
		t.Helper()
		var view factory.AppVerificationAllowance
		if err := json.Unmarshal(body, &view); err != nil {
			t.Fatal(err)
		}
		return view
	}
	currentBody := call(server, http.MethodGet, "", "", 200)
	projected = decode(currentBody)
	expectedCurrent, _ := json.Marshal(current)
	if strings.TrimSpace(string(currentBody)) != string(expectedCurrent) || projected.Revision != current.Revision || projected.Consumption != initial.Consumption || projected.Limits != current.Limits {
		t.Fatal("HTTP lost reopened allowance projection", projected)
	}
	encodedChange, _ := json.Marshal(change)
	projected = decode(call(server, http.MethodPost, string(encodedChange), "", 200))
	if projected.CommandID != change.CommandID || projected.Revision != second.CommandID || projected.Limits.Runs != 4 {
		t.Fatal("HTTP replay did not restore current limits", projected)
	}
	third := second
	third.CommandID, third.ExpectedRevision, third.Limits.Runs = factory.NewID(), second.CommandID, 5
	encodedThird, _ := json.Marshal(third)
	projected = decode(call(server, http.MethodPost, string(encodedThird), "", 200))
	if projected.Revision != third.CommandID || projected.Limits.Runs != 5 || projected.Consumption != initial.Consumption {
		t.Fatal("HTTP did not apply exact owner increase", projected)
	}
	call(server, http.MethodPost, string(encodedThird), "", 200)
	if saved, err := reopened.FactoryCommand(ctx, third.CommandID); err != nil || saved.Principal != "profile:http-owner" {
		t.Fatal("HTTP allowance did not bind separate host principal", saved, err)
	}
	encodedStale, _ := json.Marshal(stale)
	call(server, http.MethodPost, string(encodedStale), "", 409)
	call(server, http.MethodGet, "", "?", 400)
	call(server, http.MethodPost, string(encodedThird), "?other=1", 400)
	call(server, http.MethodHead, "", "", 405)
	call(server, http.MethodPost, string(encodedThird)+string(encodedThird), "", 400)
	call(server, http.MethodPost, `{"principal":"other"}`, "", 400)
	call(server, http.MethodPost, `{"command_id":"`+strings.Repeat("x", 65536)+`"}`, "", 413)
	accessErr = denied
	call(server, http.MethodGet, "", "", 403)
	call(server, http.MethodPost, string(encodedThird), "", 403)
	accessErr = nil
	admitted.Store(false)
	call(server, http.MethodGet, "", "", 401)
	admitted.Store(true)
	loseAt.Store(2)
	if body := call(server, http.MethodGet, "", "", 401); len(body) != 0 {
		t.Fatal("late account withdrawal disclosed allowance", string(body))
	}
	fourth := third
	fourth.CommandID, fourth.ExpectedRevision, fourth.Limits.Runs = factory.NewID(), third.CommandID, 6
	encodedFourth, _ := json.Marshal(fourth)
	call(server, http.MethodPost, string(encodedFourth), "", 401)
	loseAt.Store(0)
	if _, err := reopened.FactoryCommand(ctx, fourth.CommandID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("withdrawal after decode retained an allowance command", err)
	}
	missingPrincipal := transport
	missingPrincipal.Principal = ""
	missing := httptest.NewServer(missingPrincipal.Handler())
	t.Cleanup(missing.Close)
	call(missing, http.MethodPost, string(encodedFourth), "", 503)
	var commands, work int
	if err := inspect.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM projects)+(SELECT count(*) FROM factory_runs)+(SELECT count(*) FROM factory_priority_judgments)+(SELECT count(*) FROM factory_usage)`).Scan(&commands, &work); err != nil || commands != 3 || work != 0 {
		t.Fatalf("allowance work/journal changed beyond the three grants: %d %d %v", commands, work, err)
	}
}

// TestFirstCreateAllowanceHTTPBoundary exercises the actual product API: a
// fresh draft explores without any allowance (503 without TypeSafe
// credentials, never 409) and leaves the intent head and generation state
// untouched.
func TestFirstCreateAllowanceHTTPBoundary(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("explore without TypeSafe credentials status=%d body=%s", code, body)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/"+intentID, "")
	if code != http.StatusOK {
		t.Fatalf("intent read status=%d body=%s", code, body)
	}
	var head factory.IntentRevision
	if err := json.Unmarshal(body, &head); err != nil {
		t.Fatal(err)
	}
	if head.ID != draftID {
		t.Fatalf("failed explore moved the intent head to %s", head.ID)
	}
	if _, err := s.IntentGeneration(context.Background(), draftID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("generation state err=%v", err)
	}
	t.Run("shared_request_commands_and_catalogue", func(t *testing.T) {
		ctx := t.Context()
		admin, ok := store.TestFixtureDSN()
		if !ok {
			t.Fatal("native PostgreSQL fixture is required")
		}
		db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cleanup)
		inspect, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = inspect.Close() })
		state := func() string {
			t.Helper()
			var digest string
			if err := inspect.QueryRowContext(ctx, `SELECT md5(jsonb_build_array((SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM factory_intent_heads h),(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_intent_revisions r),(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM factory_feed_posts p),(SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM factory_departments d))::text)`).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			return digest
		}
		const actor collaboration.ActorID = 9007199254740993
		const repository int64 = 9007199254740995
		allowed := true
		denied := errors.New("declared current repository access withdrawn")
		var accessCalls atomic.Int32
		requests := factoryservice.Requests{Store: db, WriteAccess: func(ctx context.Context, owner collaboration.ActorID, repo int64) error {
			accessCalls.Add(1)
			if owner != actor || repo != repository {
				return denied
			}
			if !allowed {
				return denied
			}
			return ctx.Err()
		}}
		department, err := requests.CreateDepartment(ctx, actor, factory.Department{ID: "d" + strings.Repeat("a", 24), Owner: 7, Name: "Private work", Description: "The owner's standalone requests"})
		if err != nil || department.Owner != int64(actor) {
			t.Fatal("department ownership was not rebound", department, err)
		}
		in := factory.IntentRevision{ID: "d" + strings.Repeat("1", 24), Intent: "d" + strings.Repeat("2", 24), Owner: 7, CreatedUnix: 1, Repository: repository, Department: department.ID, Objective: "A standalone request", State: "draft", MaxIssues: 1}
		saved, err := requests.SaveRevision(ctx, actor, in)
		if err != nil || saved.Owner != int64(actor) || saved.CreatedUnix == 1 || accessCalls.Load() != 1 {
			t.Fatal("request ownership/time/admission was not bound", saved, err)
		}
		before := state()
		in.CreatedUnix = saved.CreatedUnix + 100
		replayed, err := requests.SaveRevision(ctx, actor, in)
		if err != nil || replayed.CreatedUnix != saved.CreatedUnix || state() != before {
			t.Fatal("identity replay changed immutable bytes", replayed, err)
		}
		allowed = false
		if _, err := requests.SaveRevision(ctx, actor, in); !errors.Is(err, factoryservice.ErrRequestWriteAccess) || !errors.Is(err, denied) || state() != before || accessCalls.Load() != 3 {
			t.Fatal("replay bypassed current repository authority", err)
		}
		allowed = true
		withoutAccess := requests
		withoutAccess.WriteAccess = nil
		if _, err := withoutAccess.SaveRevision(ctx, actor, in); !errors.Is(err, factoryservice.ErrRequestWriteUnavailable) || state() != before {
			t.Fatal("missing authority port admitted a write", err)
		}
		changed := in
		changed.Objective = "Changed immutable content"
		if _, err := requests.SaveRevision(ctx, actor, changed); !errors.Is(err, store.ErrCommandConflict) || state() != before {
			t.Fatal("immutable identity was overwritten", err)
		}
		results := make(chan error, 2)
		for _, digit := range []string{"3", "4"} {
			next := in
			next.ID, next.Predecessor = "d"+strings.Repeat(digit, 24), saved.ID
			go func() { _, err := requests.SaveRevision(ctx, actor, next); results <- err }()
		}
		wins, stale := 0, 0
		for range 2 {
			err := <-results
			if err == nil {
				wins++
			} else if errors.Is(err, store.ErrStaleRevision) {
				stale++
			} else {
				t.Fatal("revision race failed outside CAS", err)
			}
		}
		if wins != 1 || stale != 1 {
			t.Fatal("concurrent successors did not serialize", wins, stale)
		}
		head, err := requests.Revision(ctx, actor, in.Intent)
		if err != nil || head.ID == saved.ID {
			t.Fatal("winning revision was not durable", head, err)
		}
		history, err := requests.History(ctx, actor, in.Intent, 10)
		if err != nil || len(history) != 2 || history[0].ID != head.ID || history[1].ID != saved.ID {
			t.Fatal("revision history changed the winning chain", history, err)
		}
		second := factory.IntentRevision{ID: "d" + strings.Repeat("5", 24), Intent: "d" + strings.Repeat("6", 24), Department: department.ID, Objective: "An unbound private request", State: "draft", MaxIssues: 1}
		if _, err := withoutAccess.SaveRevision(ctx, actor, second); err != nil {
			t.Fatal("unbound request needed repository authority", err)
		}
		page, cursor, err := requests.Catalogue(ctx, actor, factory.IntentListFilter{Department: department.ID, Limit: 1})
		if err != nil || len(page) != 1 || cursor == "" {
			t.Fatal("private catalogue lost its continuation", page, cursor, err)
		}
		after, next, err := requests.Catalogue(ctx, actor, factory.IntentListFilter{Department: department.ID, Cursor: cursor, Limit: 1})
		if err != nil || len(after) != 1 || next != "" || after[0].Intent == page[0].Intent {
			t.Fatal("private catalogue repeated or dropped a request", after, next, err)
		}
		if foreign, _, err := requests.Catalogue(ctx, 7, factory.IntentListFilter{Limit: 10}); err != nil || len(foreign) != 0 {
			t.Fatal("catalogue exposed another owner's requests", foreign, err)
		}
		if foreign, err := requests.Departments(ctx, 7); err != nil || len(foreign) != 0 {
			t.Fatal("departments exposed another owner", foreign, err)
		}
		if _, err := requests.History(ctx, 7, in.Intent, 10); !errors.Is(err, factoryservice.ErrRequestNotFound) {
			t.Fatal("history exposed another owner", err)
		}
		before = state()
		if _, err := requests.SaveRevision(ctx, 0, in); !errors.Is(err, factoryservice.ErrRequestNotFound) || state() != before {
			t.Fatal("anonymous mutation was admitted", err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = reopened.Close() })
		requests.Store = reopened
		// A lost successful response resumes with the captured head after
		// process/store recomposition, preserving its original timestamp.
		if replay, err := requests.SaveRevision(ctx, actor, head); err != nil || replay.CreatedUnix != head.CreatedUnix || state() != before {
			t.Fatal("reopen changed the saved request", replay, err)
		}
		if runs, err := reopened.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
			t.Fatal("request operations dispatched work", runs, err)
		}
	})
	t.Run("explicit repository repair funding through authenticated HTTP", func(t *testing.T) {
		path := "/api/repositories/7/factory/repair"
		if code, body := call(http.MethodGet, path, ""); code != 200 || !strings.Contains(string(body), `"revision":0`) {
			t.Fatalf("unfunded read: %d %s", code, body)
		}
		first := factoryservice.RepairFundingChange{CommandID: factory.NewID(), InvestigatorRuns: 1, ProposalJudgments: 1, ActiveMinutes: 30, Active: true}
		encoded, _ := json.Marshal(first)
		for i := 0; i < 2; i++ {
			if code, body := call(http.MethodPut, path, string(encoded)); code != 200 {
				t.Fatalf("grant/replay: %d %s", code, body)
			}
		}
		grant, err := s.RepairGrantCommand(context.Background(), 7)
		if err != nil || grant.Sponsor != 1 || grant.Revision != 0 || grant.ID != first.CommandID {
			t.Fatal("grant lost authenticated sponsor", err)
		}
		charge := factory.RepairCharge{ID: factory.NewID(), GrantID: grant.ID, Kind: factory.RepairChargeRun, EpisodeID: factory.NewID(), Cycle: 1, EvidenceDigest: strings.Repeat("a", 64), ActiveMinutes: 10, SpentUnix: time.Now().Unix()}
		if spent, err := s.AdmitStallInvestigation(context.Background(), 7, charge, time.Now()); err != nil || !spent {
			t.Fatal("canonical charge", err)
		}
		conflict := first
		conflict.ActiveMinutes = 29
		bad, _ := json.Marshal(conflict)
		if code, body := call(http.MethodPut, path, string(bad)); code != 409 || errorCode(body) != "command_conflict" {
			t.Fatalf("changed replay: %d %s", code, body)
		}
		stale := first
		stale.CommandID = factory.NewID()
		bad, _ = json.Marshal(stale)
		if code, body := call(http.MethodPut, path, string(bad)); code != 412 {
			t.Fatalf("stale head: %d %s", code, body)
		}
		raised := factoryservice.RepairFundingChange{CommandID: factory.NewID(), ExpectedRevision: 1, InvestigatorRuns: 2, ProposalJudgments: 2, ActiveMinutes: 60, Effect: "Owner explicitly funds one additional bounded investigation without refilling spent work.", Active: true}
		more, _ := json.Marshal(raised)
		if code, body := call(http.MethodPut, path, string(more)); code != 200 {
			t.Fatalf("explicit raise: %d %s", code, body)
		}
		if code, body := call(http.MethodPut, path, string(encoded)); code != 200 {
			t.Fatalf("old exact receipt replay: %d %s", code, body)
		}
		code, body := call(http.MethodGet, path, "")
		var view struct {
			Revision int64                  `json:"revision"`
			Grant    factory.RepairGrant    `json:"grant"`
			Charges  []factory.RepairCharge `json:"charges"`
		}
		if json.Unmarshal(body, &view) != nil || code != 200 || view.Revision != 2 || view.Grant.Sponsor != 1 || len(view.Charges) != 1 || view.Charges[0].ID != charge.ID {
			t.Fatalf("funding/readback refilled history: %d %s", code, body)
		}
		foreign := actorCall(t, product, "2", "bob")
		if code, _ := foreign(http.MethodPut, path, string(more)); code < 400 {
			t.Fatal("foreign actor funded repair")
		}
		charges, err := s.RepairCharges(context.Background(), 7)
		if err != nil || len(charges) != 1 {
			t.Fatal("read/refusal charged work", err)
		}
	})
	t.Run("owner verification allowance returns current limits without saved responses", func(t *testing.T) {
		ctx := context.Background()
		idea := factory.ProductIdea{ID: "d121212121212121212121212", Description: "Store a task", Consequence: "Tasks remain readable", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}
		task := factory.IntentTask{Key: "d232323232323232323232323", Objective: "Implement task storage", Outcomes: []string{idea.Consequence}, IdeaIDs: []string{idea.ID}}
		head := factory.IntentRevision{ID: "d343434343434343434343434", Intent: "d454545454545454545454545", Owner: 1, Repository: 7, Objective: "Task app", State: "accepted", MaxIssues: 1, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
		if err := s.SaveIntentRevision(ctx, head); err != nil {
			t.Fatal(err)
		}
		path := "/api/factory/intents/" + head.Intent + "/verification-allowance"
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			if admitted, handled := extensionFactoryIntentsRoute(strings.Split("factory/intents/"+head.Intent+"/verification-allowance", "/"), method); !admitted || !handled {
				t.Fatal("native bridge omitted verification allowance", method)
			}
		}
		if admitted, _ := extensionFactoryIntentsRoute(strings.Split("factory/intents/"+head.Intent+"/verification-allowance", "/"), http.MethodPut); admitted {
			t.Fatal("native bridge admitted another mutation method")
		}
		code, body := call(http.MethodGet, path, "")
		var initial factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(body, &initial) != nil || initial.Intent != head.Intent || initial.Revision != "" || initial.CommandID != "" || initial.Limits.Runs != 2 || initial.Limits.ActiveMinutes != 60 || initial.Consumption.Runs != 0 || initial.Remaining.Runs != 2 || initial.Remaining.ActiveSeconds != 3600 || !initial.CanIncrease {
			t.Fatalf("default verification read: %d %s", code, body)
		}
		first := factory.AppVerificationAllowanceChange{CommandID: factory.NewID(), Limits: factory.AppVerificationLimits{Runs: 3, ActiveMinutes: 60}}
		encoded, _ := json.Marshal(first)
		code, originalReceipt := call(http.MethodPost, path, string(encoded))
		var raised factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(originalReceipt, &raised) != nil || raised.Revision != first.CommandID || raised.CommandID != first.CommandID || raised.Intent != head.Intent || raised.Limits.Runs != 3 || raised.Limits.ActiveMinutes != 60 || raised.Consumption != initial.Consumption {
			t.Fatalf("owner increase: %d %s", code, originalReceipt)
		}
		grant, err := s.FactoryCommand(ctx, first.CommandID)
		if err != nil || grant.Finished == "" || grant.Outcome != "" {
			t.Fatalf("grant was not admitted directly or retained a response: %+v %v", grant, err)
		}
		second := first
		second.CommandID, second.ExpectedRevision, second.Limits.Runs = factory.NewID(), first.CommandID, 4
		more, _ := json.Marshal(second)
		if code, body := call(http.MethodPost, path, string(more)); code != 200 {
			t.Fatalf("second explicit increase: %d %s", code, body)
		}
		head.ID, head.Predecessor = "d565656565656565656565656", head.ID
		if err := s.SaveIntentRevision(ctx, head); err != nil {
			t.Fatal(err)
		}
		code, body = call(http.MethodPost, path, string(encoded))
		var repeated factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(body, &repeated) != nil || repeated.Revision != second.CommandID || repeated.CommandID != first.CommandID || repeated.Limits.Runs != 4 || repeated.Consumption != initial.Consumption {
			t.Fatalf("repeated grant did not return current limits and usage: %d %s", code, body)
		}
		stale := first
		stale.CommandID, stale.Limits.Runs = factory.NewID(), 5
		raw, _ := json.Marshal(stale)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "stale_revision" {
			t.Fatalf("stale verification CAS: %d %s", code, body)
		}
		if _, err := s.FactoryCommand(ctx, stale.CommandID); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("stale increase retained a command", err)
		}
		decrease := first
		decrease.CommandID, decrease.ExpectedRevision = factory.NewID(), second.CommandID
		raw, _ = json.Marshal(decrease)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "increase_required" {
			t.Fatalf("decrease admitted: %d %s", code, body)
		}
		conflict := first
		conflict.Limits.Runs = 9
		raw, _ = json.Marshal(conflict)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "command_conflict" {
			t.Fatalf("changed command replay: %d %s", code, body)
		}
		if code, body := call(http.MethodPost, path, `{"command_id":"`+factory.NewID()+`","expected_revision":"`+second.CommandID+`","limits":{"runs":5,"active_minutes":60,"invented":true}}`); code != 400 {
			t.Fatalf("unknown limit field: %d %s", code, body)
		}
		if code, body := call(http.MethodPost, path, `{"command_id":"`+factory.NewID()+`","expected_revision":"`+second.CommandID+`","limits":{"runs":0,"active_minutes":60}}`); code != 400 {
			t.Fatalf("nonfinite limit: %d %s", code, body)
		}
		foreign := actorCall(t, product, "2", "bob")
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			if code, body := foreign(method, path, string(more)); code != 403 {
				t.Fatalf("foreign verification owner: %d %s", code, body)
			}
		}
		if code, body := call(http.MethodGet, "/api/factory/commands/"+first.CommandID, ""); code != 404 {
			t.Fatalf("removed command readback endpoint remained available: %d %s", code, body)
		}
		code, body = call(http.MethodGet, path, "")
		var current factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(body, &current) != nil || current.Revision != second.CommandID || current.CommandID != "" || current.Limits.Runs != 4 || current.Limits.ActiveMinutes != 60 || current.Consumption != initial.Consumption {
			t.Fatalf("current status refilled consumption: %d %s", code, body)
		}
		if runs, err := s.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
			t.Fatal("allowance reads or increases launched work", runs, err)
		}
	})
}

// q05Mux builds the real product mux on an ephemeral PG store with a fake
// native callback authority. keyFile, when non-empty, enables live TypeSafe
// judgments; empty keeps the deterministic no-credentials boundary.
func q05Mux(t *testing.T, keyFile string) (*API, *store.Store, func(method, path, body string) (int, []byte), func(body []byte) string) {
	product, s, _, call, errorCode := q05MuxFull(t, keyFile)
	return product, s, call, errorCode
}

// q05MuxFull builds the same mux as q05Mux and additionally returns the
// ephemeral database URL for seeds that have no production Store writer
// (canonical completions, controller-owned F10 facts).
func q05MuxFull(t *testing.T, keyFile string) (*API, *store.Store, string, func(method, path, body string) (int, []byte), func(body []byte) string) {
	t.Helper()
	suffix := "q05-" + strconv.Itoa(os.Getpid())
	callbackPath := "@soda-callback-" + suffix
	callbackListener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}})
		case extensions.OperationRepository:
			if in.RepositoryID != "7" {
				http.Error(w, "not visible", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "Alpha", Permission: "admin"}})
		default:
			http.Error(w, "denied", http.StatusForbidden)
		}
	}))
	callback.Listener.Close()
	callback.Listener = callbackListener
	callback.Start()
	t.Cleanup(callback.Close)
	t.Setenv(extensions.CallbackEnv, callbackPath)

	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	product := &API{Config: &config.Config{ForgejoURL: "https://soda.test", TypeSafeAPIKeyFile: keyFile}, Store: s}
	// This peer declares the retained equal-number Fountain namespace. The
	// mapped-actor journey in web uses differing/colliding provider identities.
	host, err := s.BindConfiguredFountainLedger(t.Context(), product.Config.ForgejoURL)
	if err != nil {
		t.Fatal(err)
	}
	product.FountainIdentities, err = forgejo.NewFountainIdentities(t.Context(), s, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	for native := int64(1); native <= 7; native++ {
		if _, err := product.FountainIdentities.LocalRepository(t.Context(), native); err != nil {
			t.Fatal(err)
		}
	}
	for native := int64(1); native <= 16; native++ {
		if _, err := product.FountainIdentities.LocalActor(t.Context(), strconv.FormatInt(native, 10)); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)

	call := func(method, path, body string) (int, []byte) {
		t.Helper()
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1",
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: "1", Username: "alice"},
		})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		if method != http.MethodGet && method != http.MethodHead {
			r.Header.Set("Origin", "https://soda.test")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code, w.Body.Bytes()
	}
	errorCode := func(body []byte) string {
		t.Helper()
		var decoded struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded.Error.Code
	}
	return product, s, dsn, call, errorCode
}

// TestFirstExploreLiveSuccess proves funded exploration success through the
// actual mux for a reviewed-direction draft (one accepted idea): U-contract
// allowance 201 -> explore 201 with saved proposal head and live judgments
// charged. Declared external judgment: LIVE TypeSafe API (model jev-latest
// default), keyed by TYPESAFE_API_KEY; skips when absent. Model-selected
// content is intentionally unasserted; only structural success facts are
// pinned (status, head advance, exploration presence, consumption). A 201
// here is funded-judged-saved success, not proof of coherent ready work.
func TestFirstExploreLiveSuccess(t *testing.T) {
	_, s, call, _ := q05Mux(t, q05KeyFile(t))

	const draftID = "d777777777777777777777777"
	const intentID = "d888888888888888888888888"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	q05LiveExplore(t, s, call, draft, intentID, draftID, "reviewed-direction")
}

// TestFirstExploreFreshEntryLive is the byte-equivalent first-entry case:
// U nextRevision with no current revision emits repository "0", empty
// ideas/tasks, state draft and control defaults (max_issues 20,
// planning_calls 8). Same structural success assertions as the
// reviewed-direction case; no accept/repository/generate/native binding.
func TestFirstExploreFreshEntryLive(t *testing.T) {
	_, s, call, _ := q05Mux(t, q05KeyFile(t))

	const draftID = "d777777777777777777777777"
	const intentID = "d888888888888888888888888"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","repository":"0","objective":"Time-off requests my manager can approve",` +
		`"ideas":[],"tasks":[],"state":"draft","max_issues":20,"planning_calls":8}`
	q05LiveExplore(t, s, call, draft, intentID, draftID, "fresh-entry")
}

func q05KeyFile(t *testing.T) string {
	t.Helper()
	key := strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY"))
	if key == "" {
		t.Skip("TYPESAFE_API_KEY unavailable")
	}
	keyFile := filepath.Join(t.TempDir(), "typesafe.key")
	if err := os.WriteFile(keyFile, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return keyFile
}

func q05LiveExplore(t *testing.T, s *store.Store, call func(method, path, body string) (int, []byte), draft, intentID, draftID, label string) {
	t.Helper()
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("%s draft save status=%d body=%s", label, code, body)
	}
	start := time.Now()
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	t.Logf("%s live explore status=%d elapsed=%s", label, code, time.Since(start).Round(time.Second))
	if code != http.StatusCreated {
		t.Fatalf("%s live explore status=%d body=%s", label, code, body)
	}
	var explored struct {
		factory.IntentRevision
		Exploration struct {
			Content factory.ProposalContent `json:"content"`
		} `json:"exploration"`
	}
	if err := json.Unmarshal(body, &explored); err != nil {
		t.Fatal(err)
	}
	if explored.ID == draftID || explored.Predecessor != draftID {
		t.Fatalf("%s explore head id=%s predecessor=%s", label, explored.ID, explored.Predecessor)
	}
	if !explored.Exploration.Content.Supported && len(explored.Exploration.Content.Unknowns) == 0 {
		t.Fatalf("%s explore returned no content", label)
	}
	head, err := s.IntentRevision(context.Background(), intentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != explored.ID {
		t.Fatalf("%s stored head %s != explored %s", label, head.ID, explored.ID)
	}
}
