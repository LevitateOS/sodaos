package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/store"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// scriptedBackgroundServer speaks the background dispatcher paths with a
// rotating admission: each bootstrap revokes the previous one like the
// native host, and unknown admissions get 401.
type scriptedBackgroundServer struct {
	mu               sync.Mutex
	admissions       []string
	revision         int64
	ops              map[string]extensions.OperationRecord
	lookupStates     map[string]string
	conflict         map[string]bool
	submits          int
	intents          []extensions.OperationIntent
	bootstrapStarted chan struct{}
	bootstrapRelease chan struct{}
	beforeRequest    func(context.Context, string, string)
	intake           http.HandlerFunc
	sources          http.HandlerFunc
}

type backgroundRequest struct {
	operationID string
	admission   string
	attempt     int
}

func (f *scriptedBackgroundServer) current() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.admissions[len(f.admissions)-1]
}

func (f *scriptedBackgroundServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case extensions.BackgroundBootstrapPath:
			if f.bootstrapStarted != nil {
				select {
				case f.bootstrapStarted <- struct{}{}:
				default:
				}
				<-f.bootstrapRelease
			}
			f.mu.Lock()
			token := strings.Repeat(string(rune('a'+len(f.admissions))), 43)
			f.admissions = append(f.admissions, token)
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{
				"admission": token, "installation_id": "install-1",
			})
		case extensions.BackgroundRevisionPath,
			extensions.BackgroundSubmitPath, extensions.BackgroundGetPath, extensions.BackgroundCancelPath, extensions.BackgroundIssueIntakePath,
			extensions.BackgroundSnapshotPath, extensions.BackgroundCreationsPath:
			var operationID string
			if r.URL.Path == extensions.BackgroundGetPath && f.beforeRequest != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "invalid background request", http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
				var request struct {
					OperationID string `json:"operation_id"`
				}
				if json.Unmarshal(body, &request) == nil {
					operationID = request.OperationID
				}
			}
			admission := r.Header.Get(extensions.AdmissionHeader)
			if operationID != "" {
				f.beforeRequest(r.Context(), operationID, admission)
			}
			f.mu.Lock()
			current := f.admissions[len(f.admissions)-1]
			f.mu.Unlock()
			if admission != current {
				http.Error(w, "background admission expired", http.StatusUnauthorized)
				return
			}
			if r.URL.Path == extensions.BackgroundIssueIntakePath && f.intake != nil {
				f.intake(w, r)
			} else if (r.URL.Path == extensions.BackgroundSnapshotPath || r.URL.Path == extensions.BackgroundCreationsPath) && f.sources != nil {
				f.sources(w, r)
			} else {
				f.serveOp(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	})
}

// Exercise source translation through the real private Unix transport and
// credential/revision boundaries. The endpoint supplies native wire records;
// actual Fountain ledger semantics are qualified separately by T02-ENGINE.
func TestServiceBackgroundNeutralGraphSources(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	host, err := db.BindConfiguredFountainLedger(t.Context(), "https://creation-source-fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	foreign := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(t.Context(), foreign); err != nil {
		t.Fatal(err)
	}
	foreignRepository, err := db.ObserveCollaborationRepository(t.Context(), foreign.ID, "7", "same/repository")
	if err != nil {
		t.Fatal(err)
	}
	foreignActor, err := db.ObserveCollaborationActor(t.Context(), foreign.ID, "91", "same-actor")
	if err != nil {
		t.Fatal(err)
	}
	identities, err := NewFountainIdentities(t.Context(), db, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	localRepository, err := identities.LocalRepository(t.Context(), 7)
	if err != nil || localRepository == 7 || localRepository == foreignRepository.ID {
		t.Fatalf("fixture repository namespaces did not differ: %d %v", localRepository, err)
	}
	localActor, err := identities.LocalActor(t.Context(), "91")
	if err != nil || localActor == 91 || localActor == foreignActor.ID {
		t.Fatal("fixture actor namespaces did not differ", localActor, err)
	}
	var mode atomic.Int64
	var calls atomic.Int64
	fake := &scriptedBackgroundServer{revision: 12}
	fake.sources = func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case extensions.BackgroundSnapshotPath:
			var in struct {
				extensions.SnapshotRequest
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Token != "test-pat" || in.RepositoryID != "7" || in.ActorID != "91" || in.IssueIndex != "1" {
				t.Error("graph source credential or scope changed", err)
				http.Error(w, "bad", 400)
				return
			}
			snapshot := extensions.NativeSnapshot{RepositoryID: "7", Issue: &extensions.SnapshotIssue{
				ID: "101", Index: "1", Title: "Current title", Content: "Exact body\n", ContentVersion: 0,
				TitleDigest: ContentDigest("Current title"), ContentDigest: ContentDigest("Exact body\n"), Visible: true, Complete: true,
				Provenance: extensions.SnapshotCreationProvenance{PosterID: "91", CreatedUnix: 1700000000, FirstCreated: true, Verified: true},
				Lifecycle:  []extensions.SnapshotLifecycleEvent{{Kind: "retitled", OccurrenceID: "50"}},
			}, Dependencies: &extensions.SnapshotDependencyPage{IssueID: "101", Total: 2, Complete: true, Items: []extensions.SnapshotDependency{
				{OccurrenceID: "60", IssueID: "101", DependencyID: "102", Visible: true, Complete: true},
				{OccurrenceID: "61", IssueID: "101", Visible: false, Complete: true},
			}}}
			if mode.Load() == 1 {
				snapshot.Issue = &extensions.SnapshotIssue{ID: "101", Index: "1", Complete: true}
			} else if mode.Load() == 2 {
				snapshot.Issue.Content = "Changed after digest capture"
			}
			_ = json.NewEncoder(w).Encode(snapshot)
		case extensions.BackgroundCreationsPath:
			var in struct {
				extensions.CreationRecoveryRequest
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Token != "test-pat" || in.RepositoryID != "7" || in.ActorID != "91" || in.AfterID != "20" || in.Limit != 16 {
				t.Error("creation source credential or cursor changed", err)
				http.Error(w, "bad", 400)
				return
			}
			page := extensions.CreationRecoveryPage{
				RepositoryID: "7", NextCursor: "22", OldestRetainedTaskID: "10", Complete: true, ProvenanceComplete: false, RetentionLimited: true,
				Creations: []extensions.NativeCreationObservation{{TaskID: "22", DeliveryID: "original-creation", RepositoryID: "7", IssueID: "101", IssueIndex: "1", CreatorID: "91", CreatorWrite: true, CreatedUnix: 1700000000, TitleDigest: ContentDigest("Original title"), ContentDigest: ContentDigest("Original body"), Verified: true}},
			}
			if mode.Load() == 3 {
				page.Creations[0].RepositoryID = "8"
			} else if mode.Load() == 4 {
				page.Creations[0].TaskID = "20"
			} else if mode.Load() == 5 {
				page.Creations[0].CreatorID = "091"
			} else if mode.Load() == 6 {
				page.RepositoryID = "8"
			}
			_ = json.NewEncoder(w).Encode(page)
		}
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	credential := observationCredential(t, "test-pat")
	rest := observationREST(t, 91)
	writer := NewIssueGraphWriter(background, rest, credential, identities)
	ctx := context.Background()
	observation, revision, err := writer.ReadGeneratedIssue(ctx, int64(localRepository), 1, int64(localActor))
	if err != nil {
		t.Fatal(err)
	}
	issue := observation.Issue
	if revision != 12 || issue == nil || issue.ID != "101" || issue.Index != "1" || issue.Content != "Exact body\n" || issue.ContentVersion != 0 || issue.Creator == nil || *issue.Creator != localActor || issue.CreatedUnix != 1700000000 || !issue.CreationVerified || !issue.FirstCreated || issue.TitleOccurrence != "50" {
		t.Fatal("generated issue evidence changed", observation, revision)
	}
	if observation.Dependencies == nil || !observation.Dependencies.Complete || len(observation.Dependencies.Items) != 2 || observation.Dependencies.Items[0].DependencyID != "102" || observation.Dependencies.Items[1].Visible || observation.Dependencies.Items[1].DependencyID != "" {
		t.Fatal("edge visibility or occurrence evidence changed", observation.Dependencies)
	}
	beforeGraphReads := calls.Load()
	for _, repository := range []int64{int64(foreignRepository.ID), 7} {
		if _, _, err := writer.ReadGeneratedIssue(ctx, repository, 1, int64(localActor)); err == nil || calls.Load() != beforeGraphReads {
			t.Fatal("foreign/unmapped graph repository reached native evidence", repository, err)
		}
	}
	if _, _, err := writer.ReadGeneratedIssue(ctx, int64(localRepository), 1, int64(foreignActor.ID)); err == nil || calls.Load() != beforeGraphReads {
		t.Fatal("foreign graph actor reached native evidence", err)
	}
	mode.Store(1)
	observation, _, err = writer.ReadGeneratedIssue(ctx, int64(localRepository), 1, int64(localActor))
	if err != nil || observation.Issue == nil || observation.Issue.Visible || observation.Issue.Creator != nil || observation.Issue.Content != "" || observation.Issue.TitleOccurrence != "" {
		t.Fatal("redacted issue gained adoption facts", observation, err)
	}
	mode.Store(2)
	if _, _, err := writer.ReadGeneratedIssue(ctx, int64(localRepository), 1, int64(localActor)); err == nil {
		t.Fatal("source content digest drift accepted")
	}
	mode.Store(0)
	observer := NewServiceObserver(socket, uint32(os.Getuid()), credential, rest)
	observer.Identities = identities
	observer.ShareBackground(background)
	if creator, err := db.CollaborationActorByExternalID(ctx, host.ID, "91"); err != nil || creator.ID != localActor {
		t.Fatal("graph read and recovery disagree on creator identity", creator.ID, err)
	}
	page, err := observer.ReadCreations(ctx, int64(localRepository), 20, 16)
	if err != nil || page.Repository != int64(localRepository) || page.NextPosition != 22 || page.OldestRetainedPosition != 10 || !page.ReachedRetainedEnd || page.ProvenanceComplete || !page.RetentionLimited || len(page.Events) != 1 {
		t.Fatal("creation coverage evidence changed", page, err)
	}
	creator, err := db.CollaborationActorByExternalID(ctx, host.ID, "91")
	if err != nil || creator.ID == 91 || creator.ID == foreignActor.ID {
		t.Fatal("recovery creator did not gain its own canonical mapping", creator.ID, err)
	}
	if event := page.Events[0]; event.Repository != int64(localRepository) || event.Position != 22 || event.Issue != 1 || event.OriginalIssueID != 101 || event.Creator != int64(creator.ID) || !event.CreatorWrite || !event.Verified || event.TitleDigest != ContentDigest("Original title") || event.ContentDigest != ContentDigest("Original body") || event.CreatedUnix != 1700000000 || event.Delivery != "original-creation" {
		t.Fatal("original creation evidence changed", event)
	}
	for _, invalid := range []int64{3, 4, 5, 6} {
		mode.Store(invalid)
		if _, err := observer.ReadCreations(ctx, int64(localRepository), 20, 16); err == nil {
			t.Fatal("foreign, malformed or non-advancing creation accepted", invalid)
		}
	}
	before := calls.Load()
	if _, err := observer.ReadCreations(ctx, int64(localRepository), -1, 16); err == nil || calls.Load() != before {
		t.Fatal("invalid cursor reached credential-bearing source", err)
	}
	for _, repository := range []int64{int64(foreignRepository.ID), 7} {
		if _, err := observer.ReadCreations(ctx, repository, 20, 16); err == nil || calls.Load() != before {
			t.Fatal("foreign or unmapped local repository reached native source", repository, err)
		}
	}
	observer.Identities = nil
	if _, err := observer.ReadCreations(ctx, int64(localRepository), 20, 16); err == nil || calls.Load() != before {
		t.Fatal("unbound creation source used a credential", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	observer.Identities, err = NewFountainIdentities(ctx, reopened, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	mode.Store(0)
	recovered, err := observer.ReadCreations(ctx, int64(localRepository), 20, 16)
	if err != nil || len(recovered.Events) != 1 || recovered.Events[0] != page.Events[0] || recovered.Repository != page.Repository || recovered.NextPosition != page.NextPosition {
		t.Fatal("reopen changed creation identity or evidence", recovered, err)
	}
}

func TestServiceBackgroundNeutralGraphOutcomes(t *testing.T) {
	identities := graphIdentityFixture(t, 7, 91, 7)
	repository, _ := identities.LocalRepository(t.Context(), 7)
	actor, _ := identities.LocalActor(t.Context(), "91")
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}, lookupStates: map[string]string{}}
	var lookupCalls atomic.Int64
	fake.beforeRequest = func(context.Context, string, string) { lookupCalls.Add(1) }
	background := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	writer := NewIssueGraphWriter(background, observationREST(t, 91), observationCredential(t, "test-pat"), identities)
	ctx := context.Background()
	generation := factory.IntentGeneration{
		OperationID: "graph-outcome", ActorID: int64(actor), NativeRev: 12, AuthRevision: "accepted-plan", NotAfter: 2000000000,
		Payload: json.RawMessage(`{"creates":[{"key":"task","title":"Exact title","body":"Exact body"}]}`),
	}
	lookup, err := writer.LookupGraph(ctx, generation, int64(repository))
	if err != nil || lookup.State != collaboration.LookupNotObserved || lookup.Record != nil {
		t.Fatal("missing operation observation changed", lookup, err)
	}
	for _, state := range []string{extensions.BackgroundOutcomeOperationsUnavailable, "future-status"} {
		fake.mu.Lock()
		fake.lookupStates[generation.OperationID] = state
		fake.mu.Unlock()
		lookup, err = writer.LookupGraph(ctx, generation, int64(repository))
		// Fountain requires a record for every status except not_observed;
		// malformed future/unavailable replies must retain that refusal.
		if err == nil || lookup.State == collaboration.LookupNotObserved || lookup.Record != nil {
			t.Fatal("unavailable/unknown lookup became retryable absence", lookup, err)
		}
	}
	outcome, err := writer.SubmitGraph(ctx, generation, int64(repository))
	if err != nil || outcome.OperationID != generation.OperationID || outcome.Effect != collaboration.EffectPending {
		t.Fatal("pending graph submission changed", outcome, err)
	}
	fake.mu.Lock()
	submitted := fake.ops[generation.OperationID]
	fake.mu.Unlock()
	if submitted.ActorID != "91" || submitted.RepositoryID != "7" {
		t.Fatal("canonical scope reached native graph submission")
	}
	receipt := json.RawMessage(`{"actor_id":91,"repository_id":7,"issues":[{"key":"task","issue_id":101,"issue_index":1,"content_version":0}],"dependencies":[]}`)
	for _, effect := range []collaboration.EffectState{collaboration.EffectPending, collaboration.EffectIndeterminate, collaboration.EffectPartial, collaboration.EffectNotCommitted, collaboration.EffectCommitted, "future-effect"} {
		record := extensions.OperationRecord{
			InstallationID: "install-1", OperationID: generation.OperationID, ActorID: "91", RepositoryID: "7", Kind: extensions.OperationKindIssueGraph,
			EffectState: string(effect), Outcome: string(effect), CancellationStatus: extensions.CancellationIndeterminate, CompletionState: extensions.CompletionNeedsIntervention, ReasonCode: "original-reason", Receipt: receipt,
		}
		fake.mu.Lock()
		fake.ops[generation.OperationID] = record
		fake.mu.Unlock()
		lookup, err = writer.LookupGraph(ctx, generation, int64(repository))
		if err != nil || lookup.State != collaboration.LookupFound || lookup.Record == nil {
			t.Fatal("recorded operation not found", lookup, err)
		}
		got := lookup.Record
		if got.OperationID != generation.OperationID || got.Effect != effect || got.Cancellation != collaboration.CancellationIndeterminate || got.Completion != collaboration.CompletionNeedsIntervention || got.Reason != "original-reason" || !bytes.Equal(got.Receipt, receipt) {
			t.Fatal("effect or original receipt evidence changed", got)
		}
		facts, err := writer.DecodeGraphReceipt(ctx, got.Receipt)
		if err != nil || facts.Repository != int64(repository) || facts.Actor != int64(actor) || !bytes.Equal(facts.Original, receipt) || facts.Creates["task"].IssueID != 101 || facts.Creates["task"].IssueIndex != 1 {
			t.Fatal("receipt did not retain evidence with canonical scope", facts, err)
		}
	}
	// Unknown receipt encoding remains original provider evidence. It cannot
	// become successful adoption or retryable absence through this projection.
	futureReceipt := json.RawMessage(`{"future_receipt":"retained"}`)
	fake.mu.Lock()
	future := fake.ops[generation.OperationID]
	future.Receipt = futureReceipt
	fake.ops[generation.OperationID] = future
	fake.mu.Unlock()
	lookup, err = writer.LookupGraph(ctx, generation, int64(repository))
	if err != nil || lookup.Record == nil || lookup.State != collaboration.LookupFound || lookup.Record.Effect != "future-effect" || !bytes.Equal(lookup.Record.Receipt, futureReceipt) {
		t.Fatal("unknown receipt lost its scoped outcome", lookup, err)
	}
	if _, err := writer.DecodeGraphReceipt(ctx, futureReceipt); err == nil {
		t.Fatal("unknown receipt became adoptable")
	}
	unknown := json.RawMessage(`{"actor_id":777,"repository_id":7,"issues":[],"dependencies":[]}`)
	if _, err := writer.DecodeGraphReceipt(ctx, unknown); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("unobserved receipt actor acquired canonical scope", err)
	}
	if _, err := identities.registry.CollaborationActorByExternalID(ctx, identities.host, "777"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("receipt decoding allocated a missing actor", err)
	}
	beforeLookup, beforeSubmit := lookupCalls.Load(), fake.submits
	for _, local := range []int64{1, 7} {
		if _, err := writer.LookupGraph(ctx, generation, local); err == nil || lookupCalls.Load() != beforeLookup {
			t.Fatal("foreign/unmapped operation repository reached lookup", local, err)
		}
		if _, err := writer.SubmitGraph(ctx, generation, local); err == nil || fake.submits != beforeSubmit {
			t.Fatal("foreign/unmapped operation repository reached submission", local, err)
		}
	}
	foreignGeneration := generation
	foreignGeneration.ActorID = 1
	if _, err := writer.LookupGraph(ctx, foreignGeneration, int64(repository)); err == nil || lookupCalls.Load() != beforeLookup {
		t.Fatal("foreign graph actor reached lookup", err)
	}
	if _, err := writer.SubmitGraph(ctx, foreignGeneration, int64(repository)); err == nil || fake.submits != beforeSubmit {
		t.Fatal("foreign graph actor reached submission", err)
	}
	for _, foreign := range []string{"actor", "repository", "installation", "operation"} {
		fake.mu.Lock()
		record := fake.ops[generation.OperationID]
		switch foreign {
		case "actor":
			record.ActorID = "92"
		case "repository":
			record.RepositoryID = "8"
		case "installation":
			record.InstallationID = "other-installation"
		case "operation":
			record.OperationID = "other-operation"
		}
		fake.ops[generation.OperationID] = record
		fake.mu.Unlock()
		if _, err := writer.LookupGraph(ctx, generation, int64(repository)); err == nil {
			t.Fatal("foreign operation evidence adopted", foreign)
		}
		fake.mu.Lock()
		record.ActorID, record.RepositoryID, record.InstallationID, record.OperationID = "91", "7", "install-1", generation.OperationID
		fake.ops[generation.OperationID] = record
		fake.mu.Unlock()
	}
	cancelled, err := writer.CancelGraph(ctx, "graph-cancel-before-submit")
	if err != nil || cancelled.OperationID != "graph-cancel-before-submit" || cancelled.Effect != collaboration.EffectNotCommitted || cancelled.Cancellation != collaboration.CancellationCancelled || len(cancelled.Receipt) != 0 {
		t.Fatal("pre-submit cancellation gained a receipt", cancelled, err)
	}
}

func (f *scriptedBackgroundServer) serveOp(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case extensions.BackgroundRevisionPath:
		_ = json.NewEncoder(w).Encode(map[string]any{"revision": f.revision, "idle": true})
	case extensions.BackgroundSubmitPath:
		var in struct {
			extensions.OperationIntent
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background intent", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.submits++
		f.intents = append(f.intents, in.OperationIntent)
		if in.Token != "test-pat" {
			http.Error(w, "invalid_credential", http.StatusUnauthorized)
			return
		}
		if existing, ok := f.ops[in.OperationID]; ok {
			_ = json.NewEncoder(w).Encode(existing)
			return
		}
		if f.conflict[in.OperationID] {
			http.Error(w, "intent_conflict", http.StatusConflict)
			return
		}
		record := extensions.OperationRecord{
			InstallationID: "install-1", OperationID: in.OperationID, Outcome: "pending",
			ActorID: in.ActorID, RepositoryID: in.RepositoryID, Kind: in.Kind,
			EffectState: "pending", CancellationStatus: "none", CompletionState: "pending",
		}
		f.ops[in.OperationID] = record
		_ = json.NewEncoder(w).Encode(record)
	case extensions.BackgroundGetPath:
		var in struct {
			OperationID string `json:"operation_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if record, ok := f.ops[in.OperationID]; ok {
			_ = json.NewEncoder(w).Encode(extensions.OperationLookup{
				InstallationID: "install-1", OperationID: in.OperationID,
				Status: record.EffectState, Record: &record,
			})
			return
		}
		state := extensions.BackgroundOutcomeNotObserved
		if status := f.lookupStates[in.OperationID]; status != "" {
			state = status
		}
		_ = json.NewEncoder(w).Encode(extensions.OperationLookup{
			InstallationID: "install-1", OperationID: in.OperationID,
			Status: state,
		})
	case extensions.BackgroundCancelPath:
		var in struct {
			OperationID string `json:"operation_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		record, ok := f.ops[in.OperationID]
		if !ok {
			record = extensions.OperationRecord{
				InstallationID: "install-1", OperationID: in.OperationID,
				Outcome: "not_committed", EffectState: "not_committed",
				CancellationStatus: "cancelled",
			}
		} else if record.EffectState == extensions.EffectCommitted {
			record.CancellationStatus = extensions.CancellationTooLate
		} else {
			record.CancellationStatus = "cancelled"
			record.EffectState, record.Outcome = "not_committed", "not_committed"
		}
		f.ops[in.OperationID] = record
		_ = json.NewEncoder(w).Encode(record)
	}
}

// The real private Unix HTTP transport and peer-UID/admission checks exercise
// translation. The scripted native endpoint is wire evidence, not a claim that
// Fountain's actual grant issuer or conditional issue ledger ran in this test.
func TestServiceBackgroundNeutralIssueIntakeBoundary(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, _, cleanup, err := store.OpenEphemeral(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	host, err := db.BindConfiguredFountainLedger(t.Context(), "https://intake-fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	foreign := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(t.Context(), foreign); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ObserveCollaborationActor(t.Context(), foreign.ID, "9007199254740993", "other actor"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ObserveCollaborationRepository(t.Context(), foreign.ID, "9007199254740995", "other/repository"); err != nil {
		t.Fatal(err)
	}
	identities, err := NewFountainIdentities(t.Context(), db, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := identities.LocalActor(t.Context(), "9007199254740993")
	if err != nil {
		t.Fatal(err)
	}
	repository, err := identities.LocalRepository(t.Context(), 9007199254740995)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int64
	var corrupt atomic.Bool
	mutation := factory.IssueMakerMutation{Action: "refine", Slot: 3, Step: 7, Title: "Exact title", Body: "Exact body\nwith trailing newline\n", ExpectedTitle: "Old title", ExpectedBodyDigest: ContentDigest("Old body")}
	fake := &scriptedBackgroundServer{ops: map[string]extensions.OperationRecord{}}
	fake.intake = func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request extensions.IssueIntakeMutation
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || !extensions.ValidIssueIntakeMutation(request) ||
			request.Grant != "fixture-provider-grant" || request.Action != mutation.Action || request.Slot != mutation.Slot ||
			request.Step != mutation.Step || request.Title != mutation.Title || request.Body != mutation.Body ||
			request.ExpectedTitle != mutation.ExpectedTitle || request.ExpectedBodyDigest != mutation.ExpectedBodyDigest {
			t.Error("neutral mutation did not preserve the native request")
			http.Error(w, "invalid", 400)
			return
		}
		slot := request.Slot
		if corrupt.Load() {
			slot++
		}
		_ = json.NewEncoder(w).Encode(extensions.IssueIntakeReceipt{Index: 9007199254740997, IssueID: 9007199254740999, Slot: slot, Title: request.Title, BodyDigest: ContentDigest(request.Body)})
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	intake := &FountainIssueIntake{Identities: identities, Background: background}
	grant := extensions.IssueIntakeGrant{Token: "fixture-provider-grant", Claims: extensions.IssueIntakeClaims{Installation: "install-1", Actor: "9007199254740993", Repository: "9007199254740995", Source: strings.Repeat("a", 32), Digest: strings.Repeat("b", 64), Expires: 2000000000}}
	scope, proof, err := identities.EncodeIssueIntakeGrant(t.Context(), grant)
	if err != nil || scope.Actor != actor || scope.Repository != repository || int64(scope.Actor) == 9007199254740993 || int64(scope.Repository) == 9007199254740995 {
		t.Fatal("scope identity changed", err)
	}
	var retained extensions.IssueIntakeGrant
	if json.Unmarshal(proof, &retained) != nil || retained != grant {
		t.Fatal("canonical scope rewrote the provider proof")
	}
	receipt, err := intake.ApplyIssueIntake(t.Context(), proof, mutation)
	if err != nil || receipt.Issue.Repository != scope.Repository || receipt.Issue.Kind != "issue" ||
		receipt.Issue.ExternalID != "9007199254740999" || receipt.Issue.Number != 9007199254740997 || receipt.Slot != 3 ||
		receipt.Title != mutation.Title || receipt.BodyDigest != ContentDigest(mutation.Body) {
		t.Fatal("receipt translation changed identity or content", receipt, err)
	}
	corrupt.Store(true)
	if _, err = intake.ApplyIssueIntake(t.Context(), proof, mutation); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("mismatched native receipt accepted", err)
	}
	if _, err = intake.ApplyIssueIntake(t.Context(), json.RawMessage(`{"token":"unbound"}`), mutation); err == nil {
		t.Fatal("unbound proof dispatched")
	}
	unmapped := grant
	unmapped.Claims.Actor = "321"
	unknown, _ := json.Marshal(unmapped)
	if _, err = intake.ApplyIssueIntake(t.Context(), unknown, mutation); err == nil {
		t.Fatal("unmapped proof dispatched")
	}
	if _, err := db.CollaborationActorByExternalID(t.Context(), host.ID, "321"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("proof decoding allocated a missing actor", err)
	}
	intake.Identities = nil
	if _, err = intake.ApplyIssueIntake(t.Context(), proof, mutation); err == nil {
		t.Fatal("unbound identity adapter dispatched")
	}
	intake.Identities = identities
	background.hostUID++
	if _, err = intake.ApplyIssueIntake(t.Context(), proof, mutation); err == nil {
		t.Fatal("proof sent to an unverified peer")
	}
	if calls.Load() != 2 {
		t.Fatal("refused proof reached private endpoint", calls.Load())
	}
}

func TestServiceBackgroundVerifiesPeerBeforeSendingCredential(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	if _, err := background.ReadNativeRevision(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Change the configured UID after successful bootstrap so this exercises
	// the next request's fresh connection, not the bootstrap check.
	background.hostUID++
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	if _, err := background.SubmitOperation(context.Background(), credential, extensions.OperationIntent{OperationID: "test"}); err == nil {
		t.Fatal("request sent to an unverified peer")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.submits != 0 {
		t.Fatal("credential-bearing request reached the wrong peer")
	}
}

func TestServiceBackgroundRejectsForeignNestedRecord(t *testing.T) {
	background := &ServiceBackground{pinned: "installation-a"}
	lookup := extensions.OperationLookup{
		InstallationID: "installation-a", OperationID: "op-1", Status: "committed",
		Record: &extensions.OperationRecord{InstallationID: "installation-b", OperationID: "op-1", Outcome: "committed"},
	}
	if err := checkBackgroundLookup(background, "op-1", lookup); err == nil {
		t.Fatal("foreign installation's nested record adopted")
	}
}

// shortSocketPath returns a unix socket path under a short fixed-parent
// directory. t.TempDir inherits TMPDIR, and a long temp parent plus the
// test-name suffix exceeds the 107-byte unix socket path limit, so fixtures
// that bind unix sockets must not build their path from it.
func shortSocketPath(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "soda-forgejo-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, name)
}

func serveScriptedBackground(t *testing.T, fake *scriptedBackgroundServer) string {
	t.Helper()
	socket := shortSocketPath(t, "background.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: fake.handler()}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestServiceBackgroundRefusesRedirects(t *testing.T) {
	for _, redirectedPath := range []string{"bootstrap", "revision"} {
		t.Run(redirectedPath, func(t *testing.T) {
			const admission = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			var mu sync.Mutex
			counts := map[string]int{}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				counts[r.URL.Path]++
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case extensions.BackgroundBootstrapPath:
					if redirectedPath == "bootstrap" {
						http.Redirect(w, r, "/redirect-target", http.StatusFound)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
				case extensions.BackgroundRevisionPath:
					if redirectedPath == "revision" {
						http.Redirect(w, r, "/redirect-target", http.StatusFound)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"revision": 12, "idle": true})
				case "/redirect-target":
					if redirectedPath == "bootstrap" {
						_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
					} else {
						_ = json.NewEncoder(w).Encode(map[string]any{"revision": 12, "idle": true})
					}
				default:
					http.NotFound(w, r)
				}
			})
			socket := shortSocketPath(t, "redirect.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			server := &http.Server{Handler: handler}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })

			background := NewServiceBackground(socket, uint32(os.Getuid()), "")
			if _, err = background.ReadNativeRevision(context.Background()); err == nil {
				t.Fatal("background client accepted a redirected response")
			}
			mu.Lock()
			defer mu.Unlock()
			if counts["/redirect-target"] != 0 {
				t.Fatalf("redirect target received %d requests", counts["/redirect-target"])
			}
			if redirectedPath == "bootstrap" && counts[extensions.BackgroundRevisionPath] != 0 {
				t.Fatalf("revision path received a request after bootstrap redirect: %d", counts[extensions.BackgroundRevisionPath])
			}
		})
	}
}

func TestServiceBackgroundSharesOneAdmission(t *testing.T) {
	fake := &scriptedBackgroundServer{admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	observation, err := background.ReadNativeRevision(ctx)
	if err != nil || observation.Revision != 12 || !observation.Idle {
		t.Fatalf("revision: %+v %v", observation, err)
	}
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	intent := extensions.OperationIntent{
		OperationID: "soda-test-publish-1", ActorID: "5", RepositoryID: "7",
		Kind: "git.ref.publish", AuthorizationRevision: "rev",
		ExpectedNativeRevision: 12, NotAfter: 9999999999,
	}
	record, err := background.SubmitOperation(ctx, credential, intent)
	if err != nil || record.EffectState != "pending" {
		t.Fatalf("submit: %+v %v", record, err)
	}
	lookup, err := background.GetOperation(ctx, "soda-test-publish-1")
	if err != nil || lookup.Status != "pending" {
		t.Fatalf("lookup: %+v %v", lookup, err)
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 1 {
		t.Fatalf("bootstraps: %d", bootstraps)
	}
	env, err := background.PublishPushEnv(ctx, "soda-test-publish-1", false)
	if err != nil {
		t.Fatalf("push env: %v", err)
	}
	var operation, admission string
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_0="); ok {
			operation, _ = strings.CutPrefix(value, extensions.PublishOperationHeader+": ")
		}
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_1="); ok {
			admission, _ = strings.CutPrefix(value, extensions.AdmissionHeader+": ")
		}
	}
	if operation != "soda-test-publish-1" || admission != fake.current() {
		t.Fatalf("env: %q", env)
	}
}

func TestServiceBackgroundBootstrapWaiterContextDoesNotCancelSharedBootstrap(t *testing.T) {
	fake := &scriptedBackgroundServer{
		admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{},
		bootstrapStarted: make(chan struct{}, 1), bootstrapRelease: make(chan struct{}),
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	first := make(chan error, 1)
	go func() {
		_, err := background.ReadNativeRevision(context.Background())
		first <- err
	}()
	<-fake.bootstrapStarted

	waiterCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	waiter := make(chan error, 1)
	go func() {
		_, err := background.ReadNativeRevision(waiterCtx)
		waiter <- err
	}()
	var waiterErr error
	select {
	case waiterErr = <-waiter:
	case <-time.After(time.Second):
		close(fake.bootstrapRelease)
		t.Fatal("waiting caller ignored its deadline")
	}
	if !errors.Is(waiterErr, context.DeadlineExceeded) {
		t.Fatalf("waiting caller did not honor its deadline: %v", waiterErr)
	}
	close(fake.bootstrapRelease)
	if err := <-first; err != nil {
		t.Fatalf("shared bootstrap was cancelled by waiter: %v", err)
	}
	if _, err := background.ReadNativeRevision(context.Background()); err != nil {
		t.Fatalf("shared admission was not published: %v", err)
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 1 {
		t.Fatalf("bootstrap count: %d", bootstraps)
	}
}

func TestServiceBackgroundStaleRejectionWaitsForForcedBootstrap(t *testing.T) {
	rejected, cached := strings.Repeat("a", 43), strings.Repeat("b", 43)
	fake := &scriptedBackgroundServer{
		admissions: []string{rejected, cached}, revision: 12,
		ops:              map[string]extensions.OperationRecord{},
		bootstrapStarted: make(chan struct{}, 1), bootstrapRelease: make(chan struct{}),
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	background.mu.Lock()
	background.admission = cached
	background.pinned = "install-1"
	background.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	forceResult := make(chan struct {
		env []string
		err error
	}, 1)
	joined := make(chan struct{})
	waiterDone := make(chan struct{})
	var cancelWaiter context.CancelFunc
	waiterStarted := false
	var releaseOnce sync.Once
	releaseBootstrap := func() { releaseOnce.Do(func() { close(fake.bootstrapRelease) }) }
	t.Cleanup(func() {
		releaseBootstrap()
		cancel()
		if cancelWaiter != nil {
			cancelWaiter()
		}
		select {
		case <-joined:
		case <-time.After(time.Second):
			t.Error("forced bootstrap caller did not stop")
		}
		if waiterStarted {
			select {
			case <-waiterDone:
			case <-time.After(time.Second):
				t.Error("stale waiter did not stop")
			}
		}
	})
	go func() {
		defer close(joined)
		env, err := background.PublishPushEnv(ctx, "soda-test-stale-force", true)
		forceResult <- struct {
			env []string
			err error
		}{env, err}
	}()
	select {
	case <-fake.bootstrapStarted:
	case <-ctx.Done():
		t.Fatalf("forced bootstrap did not start: %v", ctx.Err())
	}

	waiterCtx, cancelStaleWaiter := context.WithTimeout(context.Background(), 50*time.Millisecond)
	cancelWaiter = cancelStaleWaiter
	waiterResult := make(chan struct {
		admission string
		err       error
	}, 1)
	waiterStarted = true
	go func() {
		defer close(waiterDone)
		admission, err := background.bootstrap(waiterCtx, true, rejected)
		waiterResult <- struct {
			admission string
			err       error
		}{admission, err}
	}()
	var staleResult struct {
		admission string
		err       error
	}
	select {
	case staleResult = <-waiterResult:
	case <-time.After(time.Second):
		t.Fatal("stale rejection ignored its deadline while waiting for forced bootstrap")
	}
	select {
	case <-forceResult:
		t.Fatal("forced bootstrap completed before its gate was released")
	default:
	}
	releaseBootstrap()
	var forced struct {
		env []string
		err error
	}
	select {
	case forced = <-forceResult:
	case <-ctx.Done():
		t.Fatalf("forced bootstrap did not finish: %v", ctx.Err())
	}
	if forced.err != nil {
		t.Fatalf("explicit forced rebind: %v", forced.err)
	}
	var envAdmission string
	for _, entry := range forced.env {
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_1="); ok {
			envAdmission, _ = strings.CutPrefix(value, extensions.AdmissionHeader+": ")
		}
	}
	if envAdmission == "" || envAdmission != fake.current() {
		t.Fatalf("forced rebind environment admission %q, current %q", envAdmission, fake.current())
	}
	if !errors.Is(staleResult.err, context.DeadlineExceeded) || staleResult.admission != "" {
		t.Fatalf("stale waiter returned admission %q and error %v; want its deadline", staleResult.admission, staleResult.err)
	}
}

func TestServiceBackgroundRebindsAfterRevocation(t *testing.T) {
	fake := &scriptedBackgroundServer{admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	if _, err := background.ReadNativeRevision(ctx); err != nil {
		t.Fatal(err)
	}
	first := fake.current()
	fake.mu.Lock()
	fake.admissions = append(fake.admissions, strings.Repeat("z", 43))
	fake.mu.Unlock()
	observation, err := background.ReadNativeRevision(ctx)
	if err != nil || observation.Revision != 12 {
		t.Fatalf("rebind: %+v %v", observation, err)
	}
	if fake.current() == first {
		t.Fatal("revoked admission still current")
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 3 {
		t.Fatalf("bootstraps: %d", bootstraps)
	}
}

func TestServiceBackgroundDelayedUnauthorizedReusesReplacementAdmission(t *testing.T) {
	firstID, secondID := "soda-test-stale-401-first", "soda-test-stale-401-second"
	firstInitialRelease, secondInitialRelease := make(chan struct{}), make(chan struct{})
	firstRetryRelease := make(chan struct{})
	firstGate, secondGate, retryGate := new(sync.Once), new(sync.Once), new(sync.Once)
	requestSeen := make(chan backgroundRequest, 8)
	retryStarted := make(chan struct{}, 1)
	var countMu sync.Mutex
	requestCounts := map[string]int{}
	fake := &scriptedBackgroundServer{
		admissions: []string{},
		revision:   12,
		ops:        map[string]extensions.OperationRecord{},
		beforeRequest: func(ctx context.Context, operationID, admission string) {
			countMu.Lock()
			requestCounts[operationID]++
			attempt := requestCounts[operationID]
			countMu.Unlock()
			select {
			case requestSeen <- backgroundRequest{operationID: operationID, admission: admission, attempt: attempt}:
			case <-ctx.Done():
				return
			}
			switch {
			case operationID == firstID && attempt == 1:
				select {
				case <-firstInitialRelease:
				case <-ctx.Done():
				}
			case operationID == secondID && attempt == 1:
				select {
				case <-secondInitialRelease:
				case <-ctx.Done():
				}
			case operationID == firstID && attempt == 2:
				select {
				case retryStarted <- struct{}{}:
				case <-ctx.Done():
					return
				}
				select {
				case <-firstRetryRelease:
				case <-ctx.Done():
				}
			}
		},
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if _, err := background.ReadNativeRevision(ctx); err != nil {
		t.Fatalf("initial admission: %v", err)
	}
	results := make(chan struct {
		id  string
		err error
	}, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		_, err := background.GetOperation(ctx, firstID)
		results <- struct {
			id  string
			err error
		}{firstID, err}
	}()
	go func() {
		defer workers.Done()
		_, err := background.GetOperation(ctx, secondID)
		results <- struct {
			id  string
			err error
		}{secondID, err}
	}()
	joined := make(chan struct{})
	go func() {
		workers.Wait()
		close(joined)
	}()
	releaseFirstInitial := func() { firstGate.Do(func() { close(firstInitialRelease) }) }
	releaseSecondInitial := func() { secondGate.Do(func() { close(secondInitialRelease) }) }
	releaseFirstRetry := func() { retryGate.Do(func() { close(firstRetryRelease) }) }
	t.Cleanup(func() {
		releaseFirstInitial()
		releaseSecondInitial()
		releaseFirstRetry()
		cancel()
		select {
		case <-joined:
		case <-time.After(time.Second):
			t.Error("background request workers did not stop")
		}
	})

	waitRequest := func(wantID string, wantAttempt int) backgroundRequest {
		t.Helper()
		for {
			select {
			case request := <-requestSeen:
				if request.operationID == wantID && request.attempt == wantAttempt {
					return request
				}
			case <-ctx.Done():
				t.Fatalf("waiting for %s attempt %d: %v", wantID, wantAttempt, ctx.Err())
				return backgroundRequest{}
			}
		}
	}
	waitResult := func(wantID string) error {
		t.Helper()
		for {
			select {
			case result := <-results:
				if result.id == wantID {
					return result.err
				}
				t.Fatalf("unexpected request completed before %s: %s (%v)", wantID, result.id, result.err)
				return errors.New("unexpected request completion")
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", wantID, ctx.Err())
				return ctx.Err()
			}
		}
	}
	initial := make(map[string]backgroundRequest, 2)
	for len(initial) < 2 {
		select {
		case request := <-requestSeen:
			if request.attempt == 1 {
				initial[request.operationID] = request
			}
		case <-ctx.Done():
			t.Fatalf("waiting for both initial requests: %v", ctx.Err())
		}
	}
	first, second := initial[firstID], initial[secondID]
	if first.admission != second.admission {
		t.Fatalf("initial admissions differ: %q and %q", first.admission, second.admission)
	}
	// Model the host revoking A outside these requests, as in
	// TestServiceBackgroundRebindsAfterRevocation.
	fake.mu.Lock()
	fake.admissions = append(fake.admissions, strings.Repeat("z", 43))
	fake.mu.Unlock()
	releaseFirstInitial()
	select {
	case <-retryStarted:
	case <-ctx.Done():
		t.Fatalf("first replacement retry did not reach the server: %v", ctx.Err())
	}
	firstRetry := waitRequest(firstID, 2)
	releaseSecondInitial()
	secondRetry := waitRequest(secondID, 2)
	secondErr := waitResult(secondID)
	releaseFirstRetry()
	firstErr := waitResult(firstID)
	if firstErr != nil || secondErr != nil {
		t.Fatalf("calls failed: first=%v second=%v", firstErr, secondErr)
	}
	if secondRetry.admission != firstRetry.admission {
		t.Fatalf("delayed stale response replaced the shared admission: %q then %q", firstRetry.admission, secondRetry.admission)
	}

	fake.mu.Lock()
	// One publication is the explicit host-side revocation marker above.
	bootstraps := len(fake.admissions) - 1
	fake.mu.Unlock()
	countMu.Lock()
	counts := map[string]int{firstID: requestCounts[firstID], secondID: requestCounts[secondID]}
	countMu.Unlock()
	if bootstraps != 2 {
		t.Fatalf("bootstraps: got %d, want initial admission plus one replacement", bootstraps)
	}
	if counts[firstID] != 2 || counts[secondID] != 2 {
		t.Fatalf("request counts: first=%d second=%d, want two each", counts[firstID], counts[secondID])
	}
}

func TestServiceBackgroundMapsDispatchStatuses(t *testing.T) {
	fake := &scriptedBackgroundServer{
		admissions: []string{}, revision: 12,
		ops:      map[string]extensions.OperationRecord{},
		conflict: map[string]bool{"soda-test-clash-1": true},
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	intent := extensions.OperationIntent{
		OperationID: "soda-test-clash-1", ActorID: "5", RepositoryID: "7",
		Kind: "git.ref.publish", AuthorizationRevision: "rev",
		ExpectedNativeRevision: 12, NotAfter: 9999999999,
	}
	_, err := background.SubmitOperation(ctx, credential, intent)
	var status *forgejopublish.StatusError
	if !errors.As(err, &status) || status.Status != 409 || status.Body != "intent_conflict" {
		t.Fatalf("conflict: %v", err)
	}
	wrong := extensions.CredentialFile(observationCredential(t, "wrong-pat"))
	intent.OperationID = "soda-test-pat-1"
	_, err = background.SubmitOperation(ctx, wrong, intent)
	if !errors.As(err, &status) || status.Status != 401 {
		t.Fatalf("credential: %v", err)
	}
	if _, err := background.GetOperation(ctx, "bad id"); err == nil {
		t.Fatal("malformed lookup accepted")
	}
	missing := NewServiceBackground(filepath.Join(t.TempDir(), "missing.sock"), uint32(os.Getuid()), "")
	if _, err := missing.ReadNativeRevision(ctx); err == nil {
		t.Fatal("missing socket observed")
	}
	if _, err := missing.PublishPushEnv(ctx, "soda-test-publish-1", false); err == nil {
		t.Fatal("missing socket bound push headers")
	}
	pinned := NewServiceBackground(socket, uint32(os.Getuid()), "install-2")
	if _, err := pinned.ReadNativeRevision(ctx); err == nil {
		t.Fatal("installation mismatch accepted")
	}
}
