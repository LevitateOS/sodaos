package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

	sdk "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// Real PostgreSQL drives the intake, role post, native-effect journal and
// replies. This native peer supplies explicit conditional operation receipts.
type intakeNativePeer struct {
	forgejo.GraphCodec
	identities *forgejo.FountainIdentities
	issues     map[int64]*sdk.SnapshotIssue
	operations map[string]sdk.OperationRecord
	lostReply  bool
}

func seedIntakeGrant(t *testing.T, s *store.Store, ctx context.Context, id string) *forgejo.FountainIdentities {
	t.Helper()
	comment, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	host, err := s.FountainLedgerHost(ctx)
	if errors.Is(err, store.ErrNotFound) {
		host, err = s.BindConfiguredFountainLedger(ctx, "https://intake-fixture.test")
	}
	if err != nil {
		t.Fatal(err)
	}
	identities, err := forgejo.NewFountainIdentities(ctx, s, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	author, err := strconv.ParseInt(comment.Record.Author, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	nativeActor, err := identities.NativeActor(ctx, collaboration.ActorID(author))
	if err != nil {
		t.Fatal(err)
	}
	nativeRepository, err := identities.NativeRepository(ctx, collaboration.RepositoryID(comment.Record.Repository))
	if err != nil {
		t.Fatal(err)
	}
	grant := sdk.IssueIntakeGrant{Token: id, Claims: sdk.IssueIntakeClaims{Installation: "declared-native-peer", Source: id, Revision: comment.Record.Revision, Actor: nativeActor, Repository: nativeRepository, Digest: factory.MCPInputDigest([]byte(comment.Body)), Expires: time.Now().Add(time.Hour).Unix()}}
	scope, proof, err := identities.EncodeIssueIntakeGrant(ctx, grant)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SaveIssueIntakeGrant(ctx, id, comment.Record.Revision, scope, proof); err != nil {
		t.Fatal(err)
	}
	return identities
}

func (g *intakeNativePeer) ApplyIssueIntake(ctx context.Context, proof json.RawMessage, request factory.IssueMakerMutation) (factory.IssueMakerReceipt, error) {
	scope, err := g.identities.DecodeIssueIntakeGrant(ctx, proof)
	if err != nil {
		return factory.IssueMakerReceipt{}, err
	}
	var grant sdk.IssueIntakeGrant
	if err = json.Unmarshal(proof, &grant); err != nil {
		return factory.IssueMakerReceipt{}, err
	}
	receiptFor := func(issue sdk.IssueGraphIssueReceipt) factory.IssueMakerReceipt {
		return factory.IssueMakerReceipt{Issue: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: scope.Repository, ExternalID: strconv.FormatInt(issue.IssueID, 10), Number: issue.IssueIndex}, Slot: request.Slot, Title: request.Title, BodyDigest: factory.MCPInputDigest([]byte(request.Body))}
	}
	id := grant.Token + "/" + request.Action + "/" + strconv.Itoa(request.Slot)
	if request.Action == "refine" {
		id = grant.Token + "/refine/" + strconv.Itoa(request.Step)
	}
	if row, ok := g.operations[id]; ok {
		var receipt sdk.IssueGraphReceipt
		_ = json.Unmarshal(row.Receipt, &receipt)
		return receiptFor(receipt.Issues[0]), nil
	}
	payload := sdk.IssueGraphPayload{}
	if request.Action == "create" {
		payload.Creates = []sdk.IssueGraphCreate{{Key: strconv.Itoa(request.Slot), Title: request.Title, Body: request.Body}}
	} else {
		issue := g.issues[int64(request.Slot+1)]
		if issue == nil || issue.Title != request.ExpectedTitle || factory.MCPInputDigest([]byte(issue.Content)) != request.ExpectedBodyDigest {
			return factory.IssueMakerReceipt{}, errors.New("human edit is preserved")
		}
		payload.Edits = []sdk.IssueGraphEdit{{IssueID: int64(request.Slot + 1), ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: request.Title, Body: request.Body}}
	}
	raw, _ := json.Marshal(payload)
	record, err := g.SubmitGraph(ctx, factory.IntentGeneration{OperationID: id, Payload: raw}, 7)
	if err != nil {
		return factory.IssueMakerReceipt{}, err
	}
	var receipt sdk.IssueGraphReceipt
	_ = json.Unmarshal(record.Receipt, &receipt)
	return receiptFor(receipt.Issues[0]), nil
}

func (g *intakeNativePeer) CheckIntentOwner(context.Context, int64, int64, int64) (int64, error) {
	return 1, nil
}

func (g *intakeNativePeer) LookupGraph(_ context.Context, in factory.IntentGeneration, _ int64) (factory.GraphOperationLookup, error) {
	if row, ok := g.operations[in.OperationID]; ok {
		return forgejo.GraphLookupFromNative(sdk.OperationLookup{Record: &row}), nil
	}
	return factory.GraphOperationLookup{}, nil
}

func (g *intakeNativePeer) CancelGraph(context.Context, string) (factory.GraphOperationOutcome, error) {
	return factory.GraphOperationOutcome{}, nil
}

func (g *intakeNativePeer) ReadGeneratedIssue(_ context.Context, _ int64, index, _ int64) (factory.GeneratedIssueObservation, int64, error) {
	issue := g.issues[index]
	if issue == nil {
		return factory.GeneratedIssueObservation{}, 0, errors.New("missing issue")
	}
	copy := *issue
	observation, err := forgejo.GeneratedIssueObservationFromNative(sdk.NativeSnapshot{Issue: &copy})
	return observation, 1, err
}

func (g *intakeNativePeer) SubmitGraph(_ context.Context, in factory.IntentGeneration, _ int64) (factory.GraphOperationOutcome, error) {
	if row, ok := g.operations[in.OperationID]; ok {
		return forgejo.GraphOutcomeFromNative(row), nil
	}
	var payload sdk.IssueGraphPayload
	if err := json.Unmarshal(in.Payload, &payload); err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	if g.issues == nil {
		g.issues = map[int64]*sdk.SnapshotIssue{}
		g.operations = map[string]sdk.OperationRecord{}
	}
	var receipt sdk.IssueGraphReceipt
	for _, create := range payload.Creates {
		index := int64(len(g.issues) + 1)
		g.issues[index] = &sdk.SnapshotIssue{ID: strconv.FormatInt(index, 10), Index: strconv.FormatInt(index, 10), Title: create.Title, Content: create.Body, ContentVersion: 1, Visible: true, Complete: true}
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{Key: create.Key, IssueID: index, IssueIndex: index, ContentVersion: 1})
	}
	for _, edit := range payload.Edits {
		issue := g.issues[edit.IssueID]
		if issue == nil || issue.Title != edit.ExpectedTitle || issue.Content != edit.ExpectedBody || issue.ContentVersion != edit.ExpectedContentVersion {
			return factory.GraphOperationOutcome{}, errors.New("conditional native edit refused")
		}
		issue.Title, issue.Content = edit.Title, edit.Body
		issue.ContentVersion++
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{IssueID: edit.IssueID, IssueIndex: edit.IssueID, ContentVersion: issue.ContentVersion})
	}
	raw, _ := json.Marshal(receipt)
	row := sdk.OperationRecord{OperationID: in.OperationID, EffectState: sdk.EffectCommitted, Receipt: raw}
	g.operations[in.OperationID] = row
	if g.lostReply {
		g.lostReply = false
		return factory.GraphOperationOutcome{}, errors.New("declared lost native reply")
	}
	return forgejo.GraphOutcomeFromNative(row), nil
}

func TestIssueMakerCapturesBeforePlanningAndPublishesOnePostPerIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Add a login screen and notifications", "issue-maker-intake", true)
	legacy := saveDrivePost(t, s, ctx, "Saved request left for its author", "legacy-request")
	if captured, err := s.PostUsesIssueMaker(ctx, legacy, 0); err != nil || captured {
		t.Fatal("older intake was opted into new work", err)
	}
	if captured, err := s.PostUsesIssueMaker(ctx, id, 0); err != nil || !captured {
		t.Fatal("new intake did not atomically select Issue Maker", err)
	}
	if _, err := s.PostUsesIssueMaker(ctx, factory.NewID(), 0); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("missing intake path became a legacy request", err)
	}
	identities := seedIntakeGrant(t, s, ctx, id)
	graph := &intakeNativePeer{identities: identities}
	app := &API{Store: s, Config: &config.Config{}, FountainIdentities: identities, IssueMakerNative: graph, Coordinator: &control.Coordinator{Store: s, Graph: graph, GraphActorID: 9}}
	admitted, proof, err := app.capturedIssueEffects().Grant(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	var poisoned sdk.IssueIntakeGrant
	if err := json.Unmarshal(proof, &poisoned); err != nil {
		t.Fatal(err)
	}
	poisoned.Claims.Digest = strings.Repeat("b", 64)
	wrongContent, _ := json.Marshal(poisoned)
	// Deliberately persist an inconsistent trusted-adapter envelope. Recovery
	// must recheck it against the real source before reaching the native peer.
	if err := s.SaveIssueIntakeGrant(ctx, id, 0, admitted, wrongContent); err != nil {
		t.Fatal(err)
	}
	creator := app.capturedIssueEffects()
	if _, err := creator.CreatePostIssue(ctx, validTestDraft(id), 7); err == nil || len(graph.operations) != 0 {
		t.Fatal("a retained proof for different content reached the native writer")
	}
	if err := s.SaveIssueIntakeGrant(ctx, id, 0, admitted, proof); err != nil {
		t.Fatal(err)
	}
	expired := poisoned
	expired.Claims.Digest = admitted.Digest
	expired.Claims.Expires = time.Now().Unix() - 1
	expiredProof, _ := json.Marshal(expired)
	if err := s.SaveIssueIntakeGrant(ctx, id, 0, admitted, expiredProof); err != nil {
		t.Fatal(err)
	}
	if _, err := creator.CreatePostIssue(ctx, validTestDraft(id), 7); err == nil || len(graph.operations) != 0 {
		t.Fatal("expired captured authority reached the native writer", err, len(graph.operations))
	}
	if err := s.SaveIssueIntakeGrant(ctx, id, 0, admitted, proof); err != nil {
		t.Fatal(err)
	}
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}, IssueMaker: app.FactoryApplication().IssueMaker().Drive}
	withoutEffects := &factoryservice.IssueMaker{Store: s, Transport: app.Coordinator}
	unwired := *wired
	unwired.IssueMaker = withoutEffects.Drive
	if _, err := factoryservice.DrivePostIntake(ctx, s, &unwired, id); err != nil {
		t.Fatal(err)
	}
	waiting, err := s.PostIssueMaker(ctx, id, 0)
	if err != nil || waiting.State != "waiting" || len(graph.issues) != 0 {
		t.Fatal("shared capture dispatched without issue effects", waiting, err)
	}
	if _, err := s.IntentRevision(ctx, factory.PostDecisionID("intent", id, 0), 42); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("unwired capture began paid planning", err)
	}
	result, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil || result.Effect != factory.PostEffectVerified || result.Issue != 1 {
		t.Fatalf("initial issue waited for planning: %+v %v", result, err)
	}
	post, err := s.PostIssueMaker(ctx, id, 0)
	if err != nil || post.State != "queued" || post.Run != "" {
		t.Fatalf("initial role state: %+v %v", post, err)
	}
	rolePost, err := s.FeedPost(ctx, post.Scope.Post)
	if err != nil || rolePost.Record.ParentID != id {
		t.Fatalf("role post lost its source parent: %+v %v", rolePost, err)
	}
	parentContext := []factory.FeedEntry{{Kind: factory.FeedKindPost, ID: post.Scope.Post, Repository: 7, ParentID: id}}
	factoryservice.EnrichParents(ctx, s, parentContext)
	if parentContext[0].ParentAuthor == "" {
		t.Fatal("off-page parent author missing")
	}
	humanParent, err := s.FeedPost(ctx, id)
	if err != nil || parentContext[0].ParentActor != humanParent.Record.Author {
		t.Fatalf("off-page human parent actor mismatch: %+v %v", parentContext[0], err)
	}
	if _, err = s.IntentRevision(ctx, factory.PostDecisionID("intent", id, 0), 42); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("planning started before the first issue receipt: %v", err)
	}
	replies, _, err := s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(replies) != 1 {
		t.Fatal("initial receipt is missing from the role thread", replies, err)
	}
	// A later progress pass can wait on AI preparation without hiding the issue.
	if _, err = factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	post, err = s.PostIssueMaker(ctx, id, 0)
	if err != nil || post.State != "waiting" || post.Scope.Issue != 1 || post.Run != "" {
		t.Fatalf("refinement wait lost its first issue: %+v %v", post, err)
	}
	pending, err := s.PendingPostIntakes(ctx, 16)
	queued := false
	for _, request := range pending {
		queued = queued || request.SourcePost == id
	}
	if err != nil || !queued {
		t.Fatal("service continuation lost the captured issue", pending, err)
	}
	graph.lostReply = true
	if _, err = app.capturedIssueEffects().Mutate(ctx, post.Scope, "create", "notifications", "Add notifications", "Notify on important changes.", "create-notifications"); err == nil {
		t.Fatal("lost native response was not surfaced")
	}
	for range 2 {
		index, err := app.capturedIssueEffects().Mutate(ctx, post.Scope, "create", "notifications", "Add notifications", "Notify on important changes.", "create-notifications")
		if err != nil || index != 2 {
			t.Fatal("native replay did not reconcile", index, err)
		}
	}
	replies, _, err = s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(replies) != 2 || len(graph.issues) != 2 {
		t.Fatal("replay duplicated or lost issue comments", replies, len(graph.issues), err)
	}
	for _, id := range replies {
		detail, err := s.FeedPost(ctx, id)
		if err != nil || detail.Actor != "issue_maker" || detail.Record.ParentID != post.Scope.Post || detail.Record.Issue <= 0 {
			t.Fatal("reply lost native issue provenance", detail, err)
		}
	}
	entries, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	if !seen[post.Scope.Post] || !seen[replies[0]] || !seen[replies[1]] {
		t.Fatal("feed omitted the role post or its live receipts", seen)
	}
	graph.issues[1].Content = "A human changed this issue."
	if _, err = app.capturedIssueEffects().Mutate(ctx, post.Scope, "refine", "request", "Refined request", "Replacement body", "human-conflict"); err == nil {
		t.Fatal("refinement overwrote human content")
	}
	if graph.issues[1].Content != "A human changed this issue." {
		t.Fatal("human content was not retained")
	}
	if _, err = app.capturedIssueEffects().Mutate(ctx, post.Scope, "refine", "unrelated", "Unrelated", "Must refuse", "cross-scope"); err == nil {
		t.Fatal("role edited an issue outside its own created set")
	}
}

type waitingPostPlanner struct{ dispatches int }

// This declared conditional provider holds external outcomes across a local
// reopen. Source, slot/step allocations and published replies use real PG.
type capturedEffectsPeer struct {
	operations map[string]factory.IssueMakerReceipt
	issues     map[int]factory.IssueMakerReceipt
	calls      int
	lostReply  bool
	alter      func(*factory.IssueMakerReceipt)
}

func (p *capturedEffectsPeer) ApplyIssueIntake(_ context.Context, proof json.RawMessage, request factory.IssueMakerMutation) (factory.IssueMakerReceipt, error) {
	p.calls++
	var envelope struct {
		Scope factory.IssueIntakeScope `json:"scope"`
	}
	if err := json.Unmarshal(proof, &envelope); err != nil {
		return factory.IssueMakerReceipt{}, err
	}
	key := envelope.Scope.Source + "/" + request.Action + "/" + strconv.Itoa(request.Slot) + "/" + strconv.Itoa(request.Step)
	receipt, exists := p.operations[key]
	if !exists {
		if request.Action == "refine" {
			previous, ok := p.issues[request.Slot]
			if !ok || previous.Title != request.ExpectedTitle || previous.BodyDigest != request.ExpectedBodyDigest {
				return factory.IssueMakerReceipt{}, errors.New("declared conditional edit refused")
			}
		}
		receipt = factory.IssueMakerReceipt{Issue: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: envelope.Scope.Repository, ExternalID: "external-issue-" + strconv.Itoa(request.Slot), Number: int64(55 + request.Slot)}, Slot: request.Slot, Title: request.Title, BodyDigest: factory.MCPInputDigest([]byte(request.Body))}
		if p.operations == nil {
			p.operations = map[string]factory.IssueMakerReceipt{}
			p.issues = map[int]factory.IssueMakerReceipt{}
		}
		p.operations[key] = receipt
		p.issues[request.Slot] = receipt
	}
	if p.lostReply {
		p.lostReply = false
		return factory.IssueMakerReceipt{}, errors.New("declared lost captured effect reply")
	}
	if p.alter != nil {
		p.alter(&receipt)
	}
	return receipt, nil
}

func TestSharedCapturedIssueEffectsAdmission(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	const repository, owner int64 = 9223372036854775807, 9007199254740993
	id, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Body: "Capture the original request", IdempotencyKey: "shared-captured"}, strconv.FormatInt(owner, 10), "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordIssueMakerIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	grant := factory.IssueIntakeScope{Source: id, Actor: collaboration.ActorID(owner), Repository: collaboration.RepositoryID(repository), Digest: factory.MCPInputDigest([]byte("Capture the original request")), Expires: time.Now().Add(time.Hour).Unix()}
	const declaredProof = "opaque-captured-fixture"
	encodeGrant := func(scope factory.IssueIntakeScope) json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(struct {
			Scope factory.IssueIntakeScope `json:"scope"`
			Proof string                   `json:"proof"`
		}{scope, declaredProof})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	saveGrant := func(scope factory.IssueIntakeScope) {
		t.Helper()
		raw := encodeGrant(scope)
		if err := db.SaveIssueIntakeGrant(ctx, id, 0, grant, raw); err != nil {
			t.Fatal(err)
		}
	}
	var onDecode func(context.Context)
	var decodeFailure error
	decode := func(ctx context.Context, proof json.RawMessage) (factory.IssueIntakeScope, error) {
		if decodeFailure != nil {
			return factory.IssueIntakeScope{}, decodeFailure
		}
		var value struct {
			Scope factory.IssueIntakeScope `json:"scope"`
			Proof string                   `json:"proof"`
		}
		if err := json.Unmarshal(proof, &value); err != nil {
			return factory.IssueIntakeScope{}, err
		}
		if value.Proof != declaredProof {
			return factory.IssueIntakeScope{}, errors.New("unrecognized declared proof")
		}
		if onDecode != nil {
			onDecode(ctx)
		}
		return value.Scope, nil
	}
	peer := &capturedEffectsPeer{}
	effects := &factoryservice.CapturedIssueEffects{Store: db, Decode: decode, Writer: peer}
	// Capture is shared by provider adapters. Only this declared issuer knows
	// the proof format; actual PG owns source/revision/author binding and reuse.
	issued, visible := 0, true
	denied := errors.New("declared current account cannot read this repository")
	var issuerFailure error
	var alterIssued func(*factory.IssueIntakeScope, *json.RawMessage)
	var onIssue func(context.Context)
	capture := factoryservice.IssueIntakeCapture{Effects: effects, Posts: factoryservice.Posts{Store: db, Visibility: func(ctx context.Context, actor collaboration.ActorID, repo int64) error {
		if actor != collaboration.ActorID(owner) || repo != repository {
			t.Fatal("proof capture changed its admitted account/repository")
		}
		if !visible {
			return denied
		}
		return ctx.Err()
	}}, Authorize: func(ctx context.Context, request factoryservice.IssueIntakeRequest) (factory.IssueIntakeScope, json.RawMessage, error) {
		issued++
		if issuerFailure != nil {
			return factory.IssueIntakeScope{}, nil, issuerFailure
		}
		scope := factory.IssueIntakeScope{Source: request.Source, Revision: request.Revision, Actor: request.Actor, Repository: request.Repository, Digest: request.Digest, Expires: grant.Expires}
		proof := encodeGrant(scope)
		if alterIssued != nil {
			alterIssued(&scope, &proof)
		}
		if onIssue != nil {
			onIssue(ctx)
		}
		return scope, proof, nil
	}}
	noProof := func(source string) {
		t.Helper()
		if proof, err := db.IssueIntakeGrant(ctx, source, 0); !errors.Is(err, store.ErrNotFound) || len(proof) != 0 {
			t.Fatal("refused capture persisted provider authority", err)
		}
	}
	for _, subject := range []struct {
		actor    collaboration.ActorID
		revision int64
	}{{collaboration.ActorID(owner + 1), 0}, {collaboration.ActorID(owner), 1}} {
		if err := capture.Capture(ctx, subject.actor, id, subject.revision); err == nil || issued != 0 {
			t.Fatal("another author/revision requested a provider proof", err, issued)
		}
	}
	visible = false
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); !errors.Is(err, denied) || issued != 0 {
		t.Fatal("withdrawn account requested a provider proof", err, issued)
	}
	visible = true
	issuerFailure = errors.New("declared provider issuance unavailable")
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); !errors.Is(err, issuerFailure) {
		t.Fatal("issuer failure reported capture success", err)
	}
	issuerFailure = nil
	noProof(id)
	for _, alter := range []func(*factory.IssueIntakeScope, *json.RawMessage){
		func(scope *factory.IssueIntakeScope, _ *json.RawMessage) { scope.Actor++ },
		func(scope *factory.IssueIntakeScope, _ *json.RawMessage) { scope.Expires = 1 },
		func(scope *factory.IssueIntakeScope, proof *json.RawMessage) {
			other := *scope
			other.Repository = 7
			*proof = encodeGrant(other)
		},
	} {
		alterIssued = alter
		if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); !errors.Is(err, factoryservice.ErrCapturedIssueAuthority) {
			t.Fatal("issuer scope/proof mismatch admitted", err)
		}
		noProof(id)
	}
	alterIssued = nil
	stopped, stopCapture := context.WithCancel(ctx)
	onIssue = func(context.Context) { stopCapture() }
	if err := capture.Capture(stopped, collaboration.ActorID(owner), id, 0); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled issuance persisted a proof", err)
	}
	noProof(id)
	onIssue = func(context.Context) { visible = false }
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); !errors.Is(err, denied) {
		t.Fatal("late account withdrawal persisted a proof", err)
	}
	noProof(id)
	onIssue, visible = nil, true
	changed, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Body: "Withdraw this source during issuance", IdempotencyKey: "proof-withdrawn"}, strconv.FormatInt(owner, 10), "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordIssueMakerIntake(ctx, changed, 0); err != nil {
		t.Fatal(err)
	}
	onIssue = func(ctx context.Context) {
		if err := db.MarkFeedPostRevoked(ctx, changed); err != nil {
			t.Fatal(err)
		}
	}
	if err := capture.Capture(ctx, collaboration.ActorID(owner), changed, 0); err == nil {
		t.Fatal("source revoked during issuance retained a proof")
	}
	noProof(changed)
	onIssue = nil
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); err != nil {
		t.Fatal("shared authority capture failed", err)
	}
	issuedBeforeReuse := issued
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); err != nil || issued != issuedBeforeReuse || peer.calls != 0 {
		t.Fatal("proof reuse issued authority or mutated provider issues", err, issued, peer.calls)
	}
	decodeFailure = errors.New("declared retained proof decoder unavailable")
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); !errors.Is(err, decodeFailure) || issued != issuedBeforeReuse {
		t.Fatal("unknown retained proof was treated as absent", err, issued)
	}
	decodeFailure = nil
	expired := grant
	expired.Expires = 1
	saveGrant(expired)
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); err != nil || issued != issuedBeforeReuse+1 {
		t.Fatal("known expired proof was not renewed through admitted issuer", err, issued)
	}
	for _, bad := range []factory.IssueIntakeScope{
		func() factory.IssueIntakeScope { g := grant; g.Expires = time.Now().Unix() - 1; return g }(),
		func() factory.IssueIntakeScope { g := grant; g.Actor = 42; return g }(),
		func() factory.IssueIntakeScope { g := grant; g.Repository = 7; return g }(),
		func() factory.IssueIntakeScope { g := grant; g.Digest = strings.Repeat("b", 64); return g }(),
	} {
		saveGrant(bad)
		if got, proof, err := effects.Grant(ctx, id, 0); err == nil || len(proof) != 0 || got.Source != "" {
			t.Fatal("invalid scope returned proof", err)
		}
	}
	saveGrant(grant)
	for _, unavailable := range []factoryservice.CapturedIssueEffects{{Store: db, Writer: peer}, {Decode: decode, Writer: peer}, {Store: db, Decode: decode}} {
		if _, err := unavailable.CreatePostIssue(ctx, validTestDraft(id), repository); !errors.Is(err, factoryservice.ErrCapturedIssueUnavailable) {
			t.Fatal("missing captured port admitted", err)
		}
	}
	maker := &factoryservice.IssueMaker{Store: db, Effects: effects}
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}, IssueMaker: maker.Drive}
	response, err := factoryservice.DrivePostIntake(ctx, db, wired, id)
	if err != nil || response.Issue != 55 || response.Effect != factory.PostEffectVerified || peer.calls != 1 {
		t.Fatal("shared capture failed", response, err)
	}
	progress, err := db.PostIssueMaker(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	scope := progress.Scope
	if err := effects.AuthorizeIssueMaker(ctx, scope); err != nil {
		t.Fatal(err)
	}
	wrong := scope
	wrong.Issue++
	if _, err := effects.Mutate(ctx, wrong, "create", "foreign", "Foreign", "Must refuse", "foreign"); !errors.Is(err, factoryservice.ErrCapturedIssueSource) {
		t.Fatal("foreign role scope admitted", err)
	}
	if _, err := effects.Mutate(ctx, scope, "refine", "unrelated", "Other", "Must refuse", "unrelated"); err == nil {
		t.Fatal("refinement outside created set admitted")
	}
	// Change the role during the second proof read, after its durable slot/step
	// was allocated. That prepared command remains, but no provider call occurs.
	reads := 0
	onDecode = func(context.Context) {
		reads++
		if reads == 2 {
			changed := progress
			changed.State = "completed"
			if err := db.SaveIssueMakerProgress(ctx, changed); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := effects.Mutate(ctx, scope, "create", "late", "Late", "Must wait", "late"); !errors.Is(err, factoryservice.ErrCapturedIssueSource) || peer.calls != 1 {
		t.Fatal("late completed role dispatched", err, peer.calls)
	}
	onDecode = nil
	if err := db.SaveIssueMakerProgress(ctx, progress); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	onDecode = func(context.Context) { cancel() }
	if _, err := effects.Mutate(cancelled, scope, "create", "cancelled", "Cancelled", "Must wait", "cancelled"); !errors.Is(err, context.Canceled) || peer.calls != 1 {
		t.Fatal("cancelled command dispatched", err)
	}
	onDecode = nil
	// Reuse the prepared operation after a lost reply and reopen: the external
	// peer has committed, while the actual local progress has no issue receipt.
	peer.lostReply = true
	if _, err := effects.Mutate(ctx, scope, "create", "late", "Late", "Must wait", "late"); err == nil {
		t.Fatal("lost reply pretended success")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	effects = &factoryservice.CapturedIssueEffects{Store: db, Decode: decode, Writer: peer}
	capture.Posts.Store, capture.Effects = db, effects
	if err := capture.Capture(ctx, collaboration.ActorID(owner), id, 0); err != nil || issued != issuedBeforeReuse+1 {
		t.Fatal("reopen reissued or lost retained authority", err, issued)
	}
	for range 2 {
		if index, err := effects.Mutate(ctx, scope, "create", "late", "Late", "Must wait", "late"); err != nil || index != 56 {
			t.Fatal("reopen lost exact operation", index, err)
		}
	}
	if len(peer.operations) != 2 {
		t.Fatal("lost reply duplicated provider work", len(peer.operations))
	}
	if _, err := effects.Mutate(ctx, scope, "create", "late", "Changed", "Must wait", "late"); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("operation identity accepted changed meaning", err)
	}
	// The provider can commit while returning a malformed receipt. Publish no
	// local issue reply until the same operation yields exact matching evidence.
	for _, alter := range []func(*factory.IssueMakerReceipt){
		func(r *factory.IssueMakerReceipt) { r.Slot++ },
		func(r *factory.IssueMakerReceipt) { r.Title = "Other" },
		func(r *factory.IssueMakerReceipt) { r.BodyDigest = strings.Repeat("a", 64) },
		func(r *factory.IssueMakerReceipt) { r.Issue.Repository = 7 },
		func(r *factory.IssueMakerReceipt) { r.Issue.Kind = collaboration.SubjectPull },
		func(r *factory.IssueMakerReceipt) { r.Issue.ExternalID = "" },
	} {
		peer.alter = alter
		if _, err := effects.Mutate(ctx, scope, "create", "receipt", "Receipt", "Exact body", "receipt"); !errors.Is(err, factoryservice.ErrCapturedIssueReceipt) {
			t.Fatal("invalid receipt accepted", err)
		}
		current, err := db.PostIssueMaker(ctx, id, 0)
		if err != nil || len(current.Issues) != 2 {
			t.Fatal("invalid receipt was published", len(current.Issues), err)
		}
	}
	peer.alter = nil
	if index, err := effects.Mutate(ctx, scope, "create", "receipt", "Receipt", "Exact body", "receipt"); err != nil || index != 57 {
		t.Fatal("valid receipt replay failed", index, err)
	}
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var journal []byte
	if err := raw.QueryRowContext(ctx, `SELECT data->'issue_operations' FROM factory_feed_posts WHERE id=$1`, scope.Post).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(journal), declaredProof) || strings.Contains(string(journal), "proof") {
		t.Fatal("provider proof entered semantic operation journal")
	}
	before := peer.calls
	onDecode = func(context.Context) {
		if err := db.MarkFeedPostRevoked(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := effects.Mutate(ctx, scope, "create", "revoked", "Revoked", "Must refuse", "revoked"); err == nil || peer.calls != before {
		t.Fatal("revoked source dispatched", err)
	}
}

func (p *waitingPostPlanner) AssembleIntentPlanningInput(context.Context, factory.IntentRevision, string) (factory.SemanticPlannerInput, project.Preparation, *control.DispatchWait, error) {
	return factory.SemanticPlannerInput{}, project.Preparation{}, &control.DispatchWait{Reason: control.WaitPreparation, Detail: "project has no accepted requirements"}, nil
}

func (p *waitingPostPlanner) DispatchIntentPlan(context.Context, string, int64, string, factory.SemanticPlannerInput, string, string) control.DispatchReport {
	p.dispatches++
	return control.DispatchReport{}
}

func TestPostAuthoringWaitIsPersistedWithoutPretendingToRun(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Please add a login flow", "authoring-wait")
	planner := &waitingPostPlanner{}
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}, Drafter: &factoryservice.IntentPostDrafter{Store: s, Transport: planner}}
	if _, err := factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	status, err := s.PostAuthoring(ctx, id)
	if err != nil || status == nil {
		t.Fatal("authoring status missing", err)
	}
	if status.State != factory.PostAuthoringWaiting || status.Reason != control.WaitPreparation || status.Detail != "project has no accepted requirements" || status.Run != "" || status.Intent == "" || status.CheckedUnix == 0 {
		t.Fatal("waiting state lost its actual cause", status)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil || pipe.Attempts != 0 || planner.dispatches != 0 {
		t.Fatal("wait dispatched or spent an attempt", pipe, err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	reader := factoryservice.Posts{Store: s, Visibility: func(context.Context, collaboration.ActorID, int64) error { return nil }}
	view, err := reader.Read(ctx, 42, detail.Record.ID)
	if err != nil {
		t.Fatal(err)
	}
	projected, ok := view["authoring"].(factory.PostAuthoring)
	if !ok || projected.State != factory.PostAuthoringWaiting || projected.Reason != control.WaitPreparation {
		t.Fatal("read projection lost authoring state", view)
	}
	t.Run("shared_post_visibility_privacy_and_replies", func(t *testing.T) {
		checks, denyAfter := 0, 0
		denied := errors.New("declared repository visibility withdrawn")
		posts := factoryservice.Posts{Store: s, Visibility: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
			checks++
			if repository != 7 || actor <= 0 || (denyAfter > 0 && checks > denyAfter) {
				return denied
			}
			return ctx.Err()
		}}
		owner, err := posts.Read(ctx, 42, id)
		if err != nil || owner["body"] != detail.Body || owner["authoring"].(factory.PostAuthoring).Intent == "" {
			t.Fatal("shared owner view lost its stored request", err)
		}
		other, err := posts.Read(ctx, 9007199254740993, id)
		if err != nil || other["authoring"].(factory.PostAuthoring).Intent != "" || other["author"] != "42" || other["repository"] != "7" {
			t.Fatal("shared view leaked private intent or rounded canonical identity", other, err)
		}
		source, err := s.ResolvePostParent(ctx, 7, id)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"first-shared-reply", "second-shared-reply"} {
			reply, revision, err := s.SaveFeedPostWithSource(ctx, factory.PostRequest{Repository: 7, Body: "A visible follow-up", ParentID: id, SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: key}, "42", "human", time.Now().Unix(), source)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.RecordPostIntake(ctx, reply, revision); err != nil {
				t.Fatal(err)
			}
		}
		checks = 0
		replies, err := posts.Replies(ctx, 42, 7, id, 0)
		if err != nil || len(replies.Items) != 2 || checks != 3 {
			t.Fatal("shared replies did not recheck every visible entry", len(replies.Items), checks, err)
		}
		checks, denyAfter = 0, 2
		revoked, err := posts.Replies(ctx, 42, 7, id, 0)
		if !errors.Is(err, factoryservice.ErrPostVisibility) || !errors.Is(err, denied) || len(revoked.Items) != 0 || checks != 3 {
			t.Fatal("reply list bypassed revocation or returned partial private data", checks, err)
		}
		checks, denyAfter = 0, 0
		withoutAccess := posts
		withoutAccess.Visibility = nil
		if _, err := withoutAccess.Read(ctx, 42, id); !errors.Is(err, factoryservice.ErrPostViewUnavailable) || checks != 0 {
			t.Fatal("missing visibility port admitted a read", err)
		}
		if _, err := posts.Read(ctx, 0, id); !errors.Is(err, factoryservice.ErrPostNotFound) || checks != 0 {
			t.Fatal("anonymous read reached repository authority", err)
		}
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := posts.Read(cancelled, 42, id); !errors.Is(err, context.Canceled) || checks != 0 {
			t.Fatal("cancelled read reached repository authority", err)
		}
		pipeAfter, err := s.PostIntake(ctx, id)
		if err != nil || pipeAfter != pipe || planner.dispatches != 0 {
			t.Fatal("shared views advanced authoring or its intake", err)
		}
	})
	if err := s.NotePostAuthoring(ctx, id, pipe.Revision, *status, status.CheckedUnix+5); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.PostAuthoring(ctx, id)
	if err != nil || fresh.SinceUnix != status.SinceUnix || fresh.CheckedUnix <= status.CheckedUnix {
		t.Fatal("wait timestamps drifted", fresh, err)
	}
	if err := s.NotePostAuthoring(ctx, id, pipe.Revision, factory.PostAuthoring{State: factory.PostAuthoringRunning, Run: "recorded-run"}, status.CheckedUnix+10); err != nil {
		t.Fatal(err)
	}
	fresh, err = s.PostAuthoring(ctx, id)
	if err != nil || fresh.State != factory.PostAuthoringRunning || fresh.SinceUnix != status.CheckedUnix+10 {
		t.Fatal("run transition was not recorded", fresh, err)
	}

	t.Run("shared_authoring_resumes_funded_attempt_after_reopen", func(t *testing.T) {
		_, db, dsn, _, _ := q05MuxFull(t, "")
		if err := db.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
			t.Fatal(err)
		}
		seedPlanFunding(t, ctx, db)
		post, revision, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Record leave requests with explicit manager approval", IdempotencyKey: "shared-funded-authoring"}, "1", "alice", time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		comment, err := db.FeedPost(ctx, post)
		if err != nil {
			t.Fatal(err)
		}
		peer := &planStubHost{launchUnconfirmed: true}
		coordinator := &control.Coordinator{Store: db, Host: peer, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
		drafter := &factoryservice.IntentPostDrafter{Store: db, Transport: coordinator}
		if _, err := drafter.DraftPostIssue(ctx, comment); !errors.Is(err, factoryservice.ErrPostDraftPending) {
			t.Fatal("lost launch reply did not retain pending authoring", err)
		}
		intent, rev := factory.PostDecisionID("intent", post, revision), factory.PostDecisionID("revision", post, revision)
		attempt, err := db.IntentPlanAttempt(ctx, intent, rev)
		if err != nil || len(peer.launches) != 1 {
			t.Fatal("funded launch was not recorded exactly once", len(peer.launches), err)
		}
		allowance, err := db.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: intent, OwnerID: 1}, attempt.ID)
		if err != nil {
			t.Fatal("funded attempt has no allowance", err)
		}
		beforeAllowance, _ := json.Marshal(allowance)
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = reopened.Close() })
		coordinator = &control.Coordinator{Store: reopened, Host: peer, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
		drafter = &factoryservice.IntentPostDrafter{Store: reopened, Transport: coordinator}
		if _, err := drafter.DraftPostIssue(ctx, comment); !errors.Is(err, factoryservice.ErrPostDraftPending) || len(peer.launches) != 1 {
			t.Fatal("reopen relaunched the unresolved attempt", len(peer.launches), err)
		}
		retained, err := reopened.IntentPlanAttempt(ctx, intent, rev)
		if err != nil || retained.ID != attempt.ID || retained.Run != attempt.Run || string(retained.Prompt) != string(attempt.Prompt) {
			t.Fatal("reopen changed captured planning identity or input", err)
		}
		allowance, err = reopened.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: intent, OwnerID: 1}, attempt.ID)
		afterAllowance, _ := json.Marshal(allowance)
		if err != nil || string(beforeAllowance) != string(afterAllowance) {
			t.Fatal("pending recovery changed funded allowance", err)
		}
		head, err := reopened.IntentRevision(ctx, intent, 1)
		if err != nil {
			t.Fatal(err)
		}
		answer := factory.SemanticPlannerOutput{
			Intent: intent, Revision: rev, Operation: factory.IntentPlanningPropose,
			Candidates: []factory.SemanticCandidate{{Key: "one", Rationale: "Keep approval explicit.", Tasks: []factory.IntentTask{{Key: factory.PostDecisionID("task", post, revision), Objective: "Record leave requests for manager approval", Outcomes: []string{"Requests retain the manager decision"}, IdeaIDs: head.EffectiveOutcomes}}}},
		}
		walkthroughs := planWalkthroughContent(t, head)
		answer.Walkthroughs = &walkthroughs
		result, err := json.Marshal(factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "completed", Summary: "planning update", Output: &factory.PhaseOutput{Decomposition: &answer}})
		if err != nil {
			t.Fatal(err)
		}
		settlePlanRun(t, ctx, reopened, attempt.Run, string(result))
		var first factory.IssueDraft
		for i := range 2 {
			draft, err := drafter.DraftPostIssue(ctx, comment)
			if err != nil || draft.Validate() != nil || draft.SourcePost != post || draft.Revision != revision || draft.Objective != answer.Candidates[0].Tasks[0].Objective || len(peer.launches) != 1 {
				t.Fatal("settled authoring did not reuse the funded answer", draft, len(peer.launches), err)
			}
			if i == 0 {
				first = draft
			} else if draft != first {
				t.Fatal("recorded answer changed on replay")
			}
		}
	})
}

// Fake Lane B collaborators: deterministic routing, authoring and
// native effects without spending, dispatching or network calls.

type fakePostJudge struct {
	judgment factoryservice.PostJudgment
	err      error
	calls    int
	input    factoryservice.PostJudgeInput
}

func (f *fakePostJudge) JudgePost(ctx context.Context, input factoryservice.PostJudgeInput) (factoryservice.PostJudgment, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return factoryservice.PostJudgment{}, f.err
	}
	return f.judgment, nil
}

func TestSharedPostIssueContextCapture(t *testing.T) {
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
	const actor collaboration.ActorID = 9007199254740993
	const repository int64 = 9223372036854775807
	author := strconv.FormatInt(int64(actor), 10)
	parent, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Issue: 8, Body: "Original work discussion", IdempotencyKey: "context-parent"}, author, "Owner", 1)
	if err != nil {
		t.Fatal(err)
	}
	id, revision, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Issue: 9, ParentID: parent, Body: "Continue this work", IdempotencyKey: "context-child"}, author, "Owner", 2)
	if err != nil {
		t.Fatal(err)
	}
	mode, calls, allowed := "cycle", 0, true
	denied := errors.New("declared current human repository access withdrawn")
	stopCycle := errors.New("declared peer stops an unbounded cycle after three pages")
	capture := factoryservice.PostContext{Posts: factoryservice.Posts{Store: db, Visibility: func(ctx context.Context, owner collaboration.ActorID, repo int64) error {
		if owner != actor || repo != repository {
			t.Fatalf("context authority subject changed: %d %d", owner, repo)
		}
		if !allowed {
			return denied
		}
		return nil
	}}}
	capture.Issues = func(ctx context.Context, owner collaboration.ActorID, repo collaboration.RepositoryID, query factoryservice.PostIssueQuery) (collaboration.Page[collaboration.Issue], error) {
		calls++
		if owner != actor || int64(repo) != repository || query.Limit != 50 && query.Limit != 1 {
			t.Fatalf("provider read subject/bound changed: %d %d %+v", owner, repo, query)
		}
		observation := collaboration.Observation{State: collaboration.ReadAvailable, ObservedAt: time.Now()}
		issue := func(number int64, title, state string) collaboration.Issue {
			return collaboration.Issue{Ref: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: repo, Number: number}, Observation: observation, Title: title, Body: "Recorded issue meaning", State: state}
		}
		page := collaboration.Page[collaboration.Issue]{Observation: observation, Complete: true, Items: []collaboration.Issue{}}
		if mode == "cycle" {
			if calls > 3 {
				return page, stopCycle
			}
			page.Items, page.Complete, page.Next = []collaboration.Issue{issue(41, "Repeated work", "open")}, false, "1"
			if query.Cursor == "1" {
				page.Next = "2"
			}
			return page, nil
		}
		if mode == "outage" {
			return page, denied
		}
		if query.Number == 0 {
			page.Items = []collaboration.Issue{issue(41, "Login flow", "open")}
			if query.Cursor == "" {
				page.Complete, page.Next = false, "41"
			} else {
				page.Items[0].Title = "Current login flow"
				page.Items = append(page.Items, issue(42, "Other work", "open"))
			}
			if mode == "foreign" {
				page.Items[0].Ref.Repository = 7
			}
			if mode == "unknown" {
				page.Observation.State = collaboration.ReadUnknown
			}
			if mode == "partial" {
				page.Next, page.Complete = "", false
			}
			return page, nil
		}
		page.Items = []collaboration.Issue{issue(query.Number, "Closed selected work", "closed")}
		if mode == "selected" {
			page.Items[0].Ref.Number++
		}
		if query.Number == 8 {
			if mode == "revoked" {
				allowed = false
			}
			if mode == "edited" {
				if _, err := db.EditFeedPost(ctx, id, author, "A changed request", revision, 3); err != nil {
					t.Fatal(err)
				}
			}
		}
		return page, nil
	}
	if err := capture.Capture(ctx, actor, id, revision); !errors.Is(err, factoryservice.ErrPostContextPage) || calls != 3 {
		t.Fatalf("multi-page cycle was not bounded before another provider read: calls=%d err=%v", calls, err)
	}
	if _, err := db.PostIssueContext(ctx, id, revision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("failed capture saved partial context: %v", err)
	}
	mode, calls = "complete", 0
	if err := capture.Capture(ctx, actor, id, revision); err != nil {
		t.Fatal(err)
	}
	baseline, err := db.PostIssueContext(ctx, id, revision)
	if err != nil || len(baseline) != 4 || calls != 4 || baseline[0].Issue != 41 || baseline[0].Title != "Current login flow" || baseline[2].Issue != 9 || baseline[2].Status != "closed" || baseline[2].Origin != factory.PostCandidateBound || baseline[3].Issue != 8 || baseline[3].Origin != factory.PostCandidateSource {
		t.Fatalf("captured paging/closed source facts: %+v calls=%d err=%v", baseline, calls, err)
	}
	snapshot, _ := json.Marshal(baseline)
	for _, broken := range []string{"outage", "foreign", "unknown", "partial", "selected", "revoked"} {
		mode, calls = broken, 0
		if err := capture.Capture(ctx, actor, id, revision); err == nil {
			t.Fatalf("invalid capture succeeded: %s", broken)
		}
		allowed = true
		retained, err := db.PostIssueContext(ctx, id, revision)
		raw, _ := json.Marshal(retained)
		if err != nil || string(raw) != string(snapshot) {
			t.Fatalf("failed capture changed prior context %s: %+v %v", broken, retained, err)
		}
	}
	mode, calls = "complete", 0
	if err := capture.Capture(ctx, actor+1, id, revision); !errors.Is(err, factoryservice.ErrPostAuthorRequired) || calls != 0 {
		t.Fatalf("another actor reached provider capture: %d %v", calls, err)
	}
	if err := capture.Capture(ctx, actor, id, revision+1); !errors.Is(err, store.ErrStaleRevision) || calls != 0 {
		t.Fatalf("stale capture reached provider: %d %v", calls, err)
	}
	mode = "edited"
	if err := capture.Capture(ctx, actor, id, revision); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("capture crossed an edited revision: %v", err)
	}
	revision++
	if _, err := db.PostIssueContext(ctx, id, revision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("edited post retained old context authority: %v", err)
	}
	mode = "complete"
	if err := capture.Capture(ctx, actor, id, revision); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	capture.Posts.Store = reopened
	detail, err := reopened.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := capture.Prepare(ctx, factoryservice.PostJudgeInput{SourcePost: detail})
	raw, _ := json.Marshal(prepared.Candidates)
	if err != nil || string(raw) != string(snapshot) {
		t.Fatalf("reopened background preparation changed captured facts: %+v %v", prepared.Candidates, err)
	}
	if runs, err := reopened.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
		t.Fatalf("context capture launched work: %+v %v", runs, err)
	}
}

func TestPostRoutingUsesSavedHumanContextWithoutCoordinator(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Just discussing the existing login flow", "human-context-routing")
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteDiscussion}}
	contextCapture := factoryservice.PostContext{Posts: factoryservice.Posts{Store: s}}
	wired := &factoryservice.PostCollaborators{Judge: judge, Prepare: contextCapture.Prepare}
	// Older requests wait without spending until their author refreshes context.
	if _, err := factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil || pipe.Attempts != 0 || judge.calls != 0 {
		t.Fatal("missing human context spent a judgment", pipe, err)
	}
	if err := s.SavePostIssueContext(ctx, id, pipe.Revision, []factory.PostWorkCandidate{{Issue: 55, Title: "Login flow", Objective: "Existing login behavior", Origin: factory.PostCandidateRecent, Status: "open"}}); err != nil {
		t.Fatal(err)
	}
	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil || response.Route != factory.PostRouteDiscussion || judge.calls != 1 {
		t.Fatal("routing required Factory setup", response, err)
	}
	if len(judge.input.Candidates) != 1 || judge.input.Candidates[0].Title != "Login flow" {
		t.Fatal("routing lost native issue meaning", judge.input.Candidates)
	}
}

type fakePostDrafter struct {
	draft factory.IssueDraft
	err   error
	errs  []error
	calls int
}

func (f *fakePostDrafter) DraftPostIssue(ctx context.Context, comment store.FeedPostDetail) (factory.IssueDraft, error) {
	f.calls++
	if len(f.errs) != 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		if err != nil {
			return factory.IssueDraft{}, err
		}
		return f.draft, nil
	}
	if f.err != nil {
		return factory.IssueDraft{}, f.err
	}
	return f.draft, nil
}

type fakePostCreator struct {
	issues    map[string]int64
	next      int64
	creates   int
	createErr error
	lookupErr error
}

type feedbackPostCreator struct {
	fakePostCreator
	deliveries int
}

func TestClarificationQuestionAndAnswerResumeTheSameRequest(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Change that page", IdempotencyKey: "clarification"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	judgment, err := factoryservice.PostJudgmentFromSelection("clarify_target", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := factoryservice.DrivePostIntake(ctx, s, &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: judgment}}, id)
	if err != nil || response.Revision != pipe.Revision || response.Route != factory.PostRouteClarify {
		t.Fatal("clarification did not advance the recorded request", response, err)
	}
	question, err := s.PostProgress(ctx, id)
	if err != nil || question != judgment.Question {
		t.Fatalf("missing specific question: %q %v", question, err)
	}
	if err = s.RetryFeedPost(ctx, id, "42", 0, "Change that page\n\nClarification: the onboarding page", 1729000001); err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil || detail.Record.Revision != 1 || detail.Body != "Change that page\n\nClarification: the onboarding page" {
		t.Fatalf("answer lost original request: %+v %v", detail, err)
	}
	resumed, err := s.PostIntake(ctx, id)
	if err != nil || resumed.Judgment != factory.PostJudgmentPending {
		t.Fatalf("answer did not resume assessment: %+v %v", resumed, err)
	}
	t.Run("shared_post_commands", func(t *testing.T) {
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
		defer inspect.Close()
		const actor collaboration.ActorID = 9007199254740993
		author := strconv.FormatInt(int64(actor), 10)
		parent, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Original parent context", IdempotencyKey: "shared-parent"}, author, "canonical", time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RecordPostIntake(ctx, parent, 0); err != nil {
			t.Fatal(err)
		}
		visible, contextReady := true, false
		grants, contexts := 0, 0
		unavailable := errors.New("declared capture unavailable")
		commands := factoryservice.PostCommands{Posts: factoryservice.Posts{Store: db, Visibility: func(ctx context.Context, current collaboration.ActorID, repository int64) error {
			if current <= 0 || repository != 7 || !visible {
				return unavailable
			}
			return ctx.Err()
		}}}
		commands.Capture.Authorization = func(ctx context.Context, current collaboration.ActorID, id string, revision int64) error {
			grants++
			stored, err := db.FeedPost(ctx, id)
			if err != nil || current != actor || stored.Record.Author != author || stored.Record.Revision != revision {
				t.Fatal("capture lost admitted actor or durable source", current, err)
			}
			return unavailable
		}
		commands.Capture.Context = func(ctx context.Context, current collaboration.ActorID, id string, revision int64) error {
			contexts++
			if current != actor {
				t.Fatal("capture used another canonical actor", current)
			}
			if !contextReady {
				return unavailable
			}
			return db.SavePostIssueContext(ctx, id, revision, []factory.PostWorkCandidate{})
		}
		req := factory.PostRequest{Repository: 7, Body: "Keep my submitted request", ParentID: parent, IdempotencyKey: "shared-child"}
		saved, err := commands.Submit(ctx, actor, "canonical", req)
		if err != nil || saved.SourcePost == "" || saved.Source == nil || saved.Source.ID != parent || saved.Source.Summary != "Original parent context" || saved.Attempts != 0 || saved.Effect != factory.PostEffectPending {
			t.Fatal("shared submission lost its durable source or invented work", saved, err)
		}
		if stall, err := db.PostStall(ctx, saved.SourcePost); err != nil || stall != factoryservice.PostStalledContext || grants != 1 || contexts != 1 {
			t.Fatal("capture failure did not retain a recoverable wait", stall, grants, contexts, err)
		}
		if captured, err := db.PostUsesIssueMaker(ctx, saved.SourcePost, 0); err != nil || !captured {
			t.Fatal("captured request fell back to legacy", err)
		}
		if _, err := db.DeleteFeedPost(ctx, parent, author, 0); err != nil {
			t.Fatal(err)
		}
		replay := req
		replay.Body = "An idempotency replay must not overwrite the saved text"
		again, err := commands.Submit(ctx, actor, "renamed", replay)
		if err != nil || again.SourcePost != saved.SourcePost || again.Source == nil || *again.Source != *saved.Source {
			t.Fatal("saved submission could not recover after its parent changed", again.SourcePost, err)
		}
		stored, err := db.FeedPost(ctx, saved.SourcePost)
		if err != nil || stored.Body != req.Body || stored.Record.Author != author || stored.AuthorName != "canonical" {
			t.Fatal("replay rewrote captured submission bytes", stored, err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		db, commands.Posts.Store = reopened, reopened
		if restored, err := commands.Submit(ctx, actor, "canonical", req); err != nil || restored.SourcePost != saved.SourcePost || restored.Revision != 0 {
			t.Fatal("reopen did not resume the original submitted identity", restored, err)
		}
		beforeGrants, beforeContexts := grants, contexts
		retryInput := factory.FeedPostRetryRequest{CommandID: factory.NewID(), Repository: 7, Body: "Keep my submitted request with clarification"}
		if _, err := commands.Retry(ctx, 43, saved.SourcePost, retryInput); !errors.Is(err, factoryservice.ErrPostAuthorRequired) {
			t.Fatal("another author revised the request", err)
		}
		stale := retryInput
		stale.ExpectedRevision = 1
		if _, err := commands.Retry(ctx, actor, saved.SourcePost, stale); !errors.Is(err, factoryservice.ErrPostRetryConflict) {
			t.Fatal("stale retry was admitted", err)
		}
		if grants != beforeGrants || contexts != beforeContexts {
			t.Fatal("refused retries captured new authority or context")
		}
		contextReady = true
		updated, err := commands.Retry(ctx, actor, saved.SourcePost, retryInput)
		if err != nil || updated.Revision != 1 || updated.CommandID != retryInput.CommandID {
			t.Fatal("shared retry did not advance its existing CAS", updated, err)
		}
		stored, err = db.FeedPost(ctx, saved.SourcePost)
		if err != nil || stored.Body != "Keep my submitted request with clarification" || stored.Record.Author != author || stored.Source == nil || stored.Source.Summary != saved.Source.Summary {
			t.Fatal("retry lost its author/body/source", stored, err)
		}
		beforeGrants, beforeContexts = grants, contexts
		if again, err := commands.Retry(ctx, actor, saved.SourcePost, retryInput); err != nil || again != updated || grants != beforeGrants || contexts != beforeContexts {
			t.Fatal("lost retry reply repeated transition or capture", again, err)
		}
		if receipt, err := commands.RetryReceipt(ctx, actor, saved.SourcePost, retryInput.CommandID, 7); err != nil || receipt != updated || grants != beforeGrants || contexts != beforeContexts {
			t.Fatal("read-only receipt lookup changed capture", err)
		}
		if _, err := reopened.EditFeedPost(ctx, saved.SourcePost, author, "Newer captured writing", 1, time.Now().Unix()); err != nil {
			t.Fatal(err)
		}
		if err := reopened.NotePostContextWait(ctx, saved.SourcePost, 1, factoryservice.PostStalledContext, "Late failed capture"); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal("late capture attached a wait to newer writing", err)
		}
		if stall, err := reopened.PostStall(ctx, saved.SourcePost); err != nil || stall != "" {
			t.Fatal("late capture changed the newer intake", stall, err)
		}
		visible = false
		beforeGrants, beforeContexts = grants, contexts
		if _, err := commands.Retry(ctx, actor, saved.SourcePost, retryInput); !errors.Is(err, factoryservice.ErrPostVisibility) {
			t.Fatal("retry replay bypassed withdrawn visibility", err)
		}
		if _, err := commands.RetryReceipt(ctx, actor, saved.SourcePost, retryInput.CommandID, 7); !errors.Is(err, factoryservice.ErrPostVisibility) {
			t.Fatal("receipt read bypassed withdrawn visibility", err)
		}
		if _, err := commands.Submit(ctx, actor, "canonical", req); !errors.Is(err, factoryservice.ErrPostVisibility) || grants != beforeGrants || contexts != beforeContexts {
			t.Fatal("submission replay bypassed withdrawn visibility", err)
		}
		visible = true
		var original string
		if err := inspect.QueryRowContext(ctx, `SELECT data::text FROM factory_post_intakes WHERE post=$1`, saved.SourcePost).Scan(&original); err != nil {
			t.Fatal(err)
		}
		if _, err := inspect.ExecContext(ctx, `UPDATE factory_post_intakes SET data=jsonb_set(data,'{issue_maker_version}','"unsupported"'::jsonb) WHERE post=$1`, saved.SourcePost); err != nil {
			t.Fatal(err)
		}
		if _, err := commands.Submit(ctx, actor, "canonical", req); err == nil || grants != beforeGrants || contexts != beforeContexts {
			t.Fatal("unknown intake path selected another capture workflow", err)
		}
		if _, err := inspect.ExecContext(ctx, `UPDATE factory_post_intakes SET data=$2 WHERE post=$1`, saved.SourcePost, original); err != nil {
			t.Fatal(err)
		}
		if runs, err := db.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
			t.Fatal("post commands started execution", runs, err)
		}
	})
}

func (f *feedbackPostCreator) DeliverPostFeedback(_ context.Context, detail store.FeedPostDetail, issue int64) error {
	if issue != 55 || detail.Body != "Fix the filter" {
		return errors.New("feedback changed source or issue")
	}
	f.deliveries++
	return nil
}

func TestExistingWorkFeedbackActuallyReachesTheSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Fix the filter", Issue: 55, IdempotencyKey: "feedback"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	creator := &feedbackPostCreator{}
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteExistingWork, Issue: 55}}, Creator: creator}
	for range 2 {
		response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
		if err != nil || response.Effect != factory.PostEffectApplied {
			t.Fatalf("feedback not delivered: %+v %v", response, err)
		}
	}
	if creator.deliveries != 1 || creator.creates != 0 {
		t.Fatalf("feedback duplicated or created new work: %+v", creator)
	}
}

func (f *fakePostCreator) CreatePostIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.creates++
	f.next++
	if f.issues == nil {
		f.issues = make(map[string]int64)
	}
	f.issues[draft.SourcePost] = f.next
	return f.next, nil
}

func (f *fakePostCreator) LookupPostIssue(ctx context.Context, repository int64, postID string) (int64, error) {
	if f.lookupErr != nil {
		return 0, f.lookupErr
	}
	return f.issues[postID], nil
}

func driveTestStore(t *testing.T) (*store.Store, context.Context) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s, context.Background()
}

func saveDrivePost(t *testing.T, s *store.Store, ctx context.Context, body, key string, issueMaker ...bool) string {
	t.Helper()
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: body, IdempotencyKey: key}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	var intakeErr error
	if len(issueMaker) > 0 && issueMaker[0] {
		intakeErr = s.RecordIssueMakerIntake(ctx, id, 0)
	} else {
		intakeErr = s.RecordPostIntake(ctx, id, 0)
	}
	if intakeErr != nil {
		t.Fatal(intakeErr)
	}
	return id
}

func validTestDraft(comment string) factory.IssueDraft {
	return factory.IssueDraft{SourcePost: comment, Title: "Fix login redirect", Body: "Change the callback.", Objective: "Logins land back in the app.", Acceptance: "Redirect returns 302.", Metadata: `{"work":"soda"}`}
}

// The external graph peer declares conditional receipts; the source and intake
// state below are actual PostgreSQL records, not simulated persistence.
type postEffectsNativePeer struct {
	forgejo.GraphCodec
	repository, owner, actor int64
	lookup                   factory.GraphOperationLookup
	operations               map[string]factory.GraphOperationOutcome
	ownerHook                func(context.Context)
	lookupHook               func(context.Context)
	denied, lostReply        bool
	submits, feedback        int
	last                     factory.IntentGeneration
	marker, body             string
}

func (p *postEffectsNativePeer) CheckIntentOwner(ctx context.Context, repository, owner, actor int64) (int64, error) {
	if repository != p.repository || owner != p.owner || actor != p.actor {
		return 0, errors.New("foreign canonical graph admission")
	}
	if p.ownerHook != nil {
		p.ownerHook(ctx)
	}
	if p.denied {
		return 0, errors.New("declared current permission denial")
	}
	return 31, ctx.Err()
}

func (p *postEffectsNativePeer) LookupGraph(ctx context.Context, generation factory.IntentGeneration, repository int64) (factory.GraphOperationLookup, error) {
	if repository != p.repository || generation.ActorID != p.actor {
		return factory.GraphOperationLookup{}, errors.New("foreign graph lookup")
	}
	if p.lookupHook != nil {
		p.lookupHook(ctx)
	}
	if p.lookup.State == collaboration.LookupFound && p.lookup.Record == nil {
		if record, ok := p.operations[generation.OperationID]; ok {
			return factory.GraphOperationLookup{State: collaboration.LookupFound, Record: &record}, nil
		}
		return factory.GraphOperationLookup{State: collaboration.LookupNotObserved}, nil
	}
	return p.lookup, nil
}

func (p *postEffectsNativePeer) SubmitGraph(ctx context.Context, generation factory.IntentGeneration, repository int64) (factory.GraphOperationOutcome, error) {
	if repository != p.repository || generation.ActorID != p.actor || generation.NativeRev != 31 || ctx.Err() != nil {
		return factory.GraphOperationOutcome{}, errors.New("foreign/cancelled graph submit")
	}
	p.submits++
	p.last = generation
	if record, ok := p.operations[generation.OperationID]; ok {
		return record, nil
	}
	var payload sdk.IssueGraphPayload
	if err := json.Unmarshal(generation.Payload, &payload); err != nil || len(payload.Creates) != 1 || len(payload.Edits)+len(payload.Dependencies) != 0 {
		return factory.GraphOperationOutcome{}, errors.New("expected one exact declared issue create")
	}
	raw, err := json.Marshal(sdk.IssueGraphReceipt{RepositoryID: p.repository, ActorID: p.actor, Issues: []sdk.IssueGraphIssueReceipt{{Key: payload.Creates[0].Key, IssueID: 1001, IssueIndex: 55, ContentVersion: 0}}})
	if err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	record := factory.GraphOperationOutcome{OperationID: generation.OperationID, Effect: collaboration.EffectCommitted, Receipt: raw}
	if p.operations == nil {
		p.operations = map[string]factory.GraphOperationOutcome{}
	}
	p.operations[generation.OperationID] = record
	if p.lostReply {
		p.lostReply = false
		return factory.GraphOperationOutcome{}, errors.New("declared lost native reply after commit")
	}
	return record, nil
}

func (p *postEffectsNativePeer) DeliverIssueFeedback(ctx context.Context, repository, issue, actor int64, marker, body string) error {
	if repository != p.repository || issue != 55 || actor != p.actor || ctx.Err() != nil {
		return errors.New("foreign/cancelled feedback")
	}
	p.feedback++
	p.marker, p.body = marker, body
	return nil
}

func TestGraphPostEffectsRetainUnknownAndSourceScope(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	const repository, owner, actor int64 = 9223372036854775807, 9007199254740993, 9007199254740995
	author := strconv.FormatInt(owner, 10)
	save := func(key string) string {
		id, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Body: "Create a login fix", IdempotencyKey: key}, author, "ada", 1729000000)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RecordPostIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
		return id
	}
	id := save("shared-post-effects")
	peer := &postEffectsNativePeer{repository: repository, owner: owner, actor: actor}
	creator := &factoryservice.PostIssueEffects{Store: db, Graph: peer, Actor: actor}
	if issue, err := creator.LookupPostIssue(ctx, repository, id); err == nil || issue != 0 || peer.submits != 0 {
		t.Fatal("unknown lookup became confirmed absence", issue, err, peer.submits)
	}
	for _, state := range []collaboration.LookupState{collaboration.LookupUnknown, collaboration.LookupUnavailable} {
		peer.lookup.State = state
		if _, err := creator.LookupPostIssue(ctx, repository, id); !errors.Is(err, factoryservice.ErrPostEffectUnknown) {
			t.Fatal(state, err)
		}
	}
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}
	response, err := factoryservice.DrivePostIntake(ctx, db, wired, id)
	if err != nil || response.Effect == factory.PostEffectVerified || peer.submits != 0 {
		t.Fatal("unknown lookup dispatched", response, err)
	}
	draft, err := db.PostDraft(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := creator.CreatePostIssue(ctx, draft, 7); !errors.Is(err, factoryservice.ErrPostEffectSource) {
		t.Fatal("equal-number/foreign repository admitted", err)
	}
	changed := draft
	changed.Body += " altered"
	if _, err := creator.CreatePostIssue(ctx, changed, repository); !errors.Is(err, factoryservice.ErrPostEffectSource) {
		t.Fatal("unrecorded draft admitted", err)
	}
	for _, unavailable := range []factoryservice.PostIssueEffects{{Store: db, Actor: actor}, {Graph: peer, Actor: actor}, {Store: db, Graph: peer}} {
		if _, err := unavailable.CreatePostIssue(ctx, draft, repository); !errors.Is(err, factoryservice.ErrPostEffectUnavailable) {
			t.Fatal("missing port admitted", err)
		}
	}
	peer.denied = true
	if _, err := creator.CreatePostIssue(ctx, draft, repository); err == nil {
		t.Fatal("current denial admitted")
	}
	peer.denied = false
	peer.ownerHook = func(context.Context) {
		if err := db.RecordPostDraft(ctx, changed); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := creator.CreatePostIssue(ctx, draft, repository); !errors.Is(err, factoryservice.ErrPostEffectSource) {
		t.Fatal("late draft replacement admitted", err)
	}
	peer.ownerHook = nil
	if err := db.RecordPostDraft(ctx, draft); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	peer.ownerHook = func(context.Context) { cancel() }
	if _, err := creator.CreatePostIssue(cancelled, draft, repository); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled submit admitted", err)
	}
	peer.ownerHook = nil
	late := save("late-revocation")
	lateDraft := validTestDraft(late)
	if err := db.RecordPostDraft(ctx, lateDraft); err != nil {
		t.Fatal(err)
	}
	peer.ownerHook = func(context.Context) {
		if err := db.MarkFeedPostRevoked(ctx, late); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := creator.CreatePostIssue(ctx, lateDraft, repository); !errors.Is(err, factoryservice.ErrPostEffectSource) {
		t.Fatal("late source revocation admitted", err)
	}
	peer.ownerHook = nil
	if peer.submits != 0 {
		t.Fatal("refused inputs reached provider", peer.submits)
	}
	// The peer commits but loses its reply. Only the external operation retains
	// that receipt; the real intake remains pending until a new service recovers.
	peer.lookup = factory.GraphOperationLookup{State: collaboration.LookupFound}
	peer.lostReply = true
	response, err = factoryservice.DrivePostIntake(ctx, db, wired, id)
	if err != nil || response.Effect == factory.PostEffectVerified || peer.submits != 1 || len(peer.operations) != 1 {
		t.Fatal("lost reply pretended success", response, err)
	}
	expectedPayload, _ := json.Marshal(sdk.IssueGraphPayload{Creates: []sdk.IssueGraphCreate{{Key: id, Title: draft.Title, Body: factoryservice.PostIssueBody(draft)}}})
	if peer.last.OperationID != "commentissue-"+id || peer.last.AuthRevision != "comment/"+id+"/r0" || string(peer.last.Payload) != string(expectedPayload) || peer.last.NotAfter < time.Now().Unix()+280 {
		t.Fatal("historical conditional operation changed", peer.last.OperationID, peer.last.AuthRevision)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	creator = &factoryservice.PostIssueEffects{Store: db, Graph: peer, Actor: actor}
	wired.Creator = creator
	response, err = factoryservice.DrivePostIntake(ctx, db, wired, id)
	if err != nil || response.Effect != factory.PostEffectVerified || response.Issue != 55 || peer.submits != 1 {
		t.Fatal("reopen duplicated or lost effect", response, peer.submits, err)
	}
	for range 2 {
		if _, err := factoryservice.DrivePostIntake(ctx, db, wired, id); err != nil {
			t.Fatal(err)
		}
	}
	if peer.submits != 1 {
		t.Fatal("settled intake repeated provider write")
	}
	baseline := peer.operations["commentissue-"+id]
	for _, change := range []func(*sdk.IssueGraphReceipt){
		func(receipt *sdk.IssueGraphReceipt) { receipt.RepositoryID = 7 },
		func(receipt *sdk.IssueGraphReceipt) { receipt.ActorID = 9 },
		func(receipt *sdk.IssueGraphReceipt) { receipt.Issues[0].Key = "foreign" },
		func(receipt *sdk.IssueGraphReceipt) {
			receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{Key: "extra", IssueID: 1002, IssueIndex: 56})
		},
	} {
		var receipt sdk.IssueGraphReceipt
		if err := json.Unmarshal(baseline.Receipt, &receipt); err != nil {
			t.Fatal(err)
		}
		change(&receipt)
		bad := baseline
		bad.Receipt, err = json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		peer.lookup = factory.GraphOperationLookup{State: collaboration.LookupFound, Record: &bad}
		if issue, err := creator.LookupPostIssue(ctx, repository, id); !errors.Is(err, factoryservice.ErrPostEffectReceipt) || issue != 0 {
			t.Fatal("complete foreign/extra receipt returned issue", issue, err)
		}
	}
	for _, mutate := range []func(*factory.GraphOperationOutcome){
		func(out *factory.GraphOperationOutcome) { out.OperationID = "foreign" },
		func(out *factory.GraphOperationOutcome) { out.Effect = collaboration.EffectIndeterminate },
		func(out *factory.GraphOperationOutcome) { out.Receipt = nil },
		func(out *factory.GraphOperationOutcome) {
			out.Receipt = json.RawMessage(`{"repository_id":7,"actor_id":9}`)
		},
		func(out *factory.GraphOperationOutcome) { out.Receipt = json.RawMessage(`{"future":"unknown"}`) },
	} {
		bad := baseline
		mutate(&bad)
		peer.lookup = factory.GraphOperationLookup{State: collaboration.LookupFound, Record: &bad}
		if issue, err := creator.LookupPostIssue(ctx, repository, id); err == nil || issue != 0 {
			t.Fatal("foreign/unknown receipt returned issue", issue, err)
		}
	}
	peer.lookup = factory.GraphOperationLookup{State: collaboration.LookupNotObserved, Record: &baseline}
	if _, err := creator.LookupPostIssue(ctx, repository, id); !errors.Is(err, factoryservice.ErrPostEffectUnknown) {
		t.Fatal("contradictory absence admitted", err)
	}
	peer.lookup = factory.GraphOperationLookup{State: collaboration.LookupFound}
	peer.lookupHook = func(context.Context) { peer.denied = true }
	if issue, err := creator.LookupPostIssue(ctx, repository, id); err == nil || issue != 0 {
		t.Fatal("late permission loss exposed receipt", issue, err)
	}
	peer.lookupHook = nil
	peer.denied = false
	detail, err := db.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	forged := detail
	forged.Body = "unrecorded feedback"
	if err := creator.DeliverPostFeedback(ctx, forged, 55); !errors.Is(err, factoryservice.ErrPostEffectSource) {
		t.Fatal("unrecorded feedback admitted", err)
	}
	peer.denied = true
	if err := creator.DeliverPostFeedback(ctx, detail, 55); err == nil || peer.feedback != 0 {
		t.Fatal("denied feedback sent", err)
	}
	peer.denied = false
	if err := creator.DeliverPostFeedback(ctx, detail, 55); err != nil || peer.feedback != 1 || peer.marker != "<!-- soda-feed:"+id+":r0 -->" || peer.body != "Feedback from @ada in the project feed:\n\n"+detail.Body {
		t.Fatal("feedback changed source/marker", err)
	}
	if peer.submits != 1 {
		t.Fatal("feedback created another issue")
	}
}

func TestDrivePostNewIssueToVerified(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}
	drafter := &fakePostDrafter{draft: validTestDraft(id)}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &factoryservice.PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.SourcePost != id || response.Judgment != factory.PostJudgmentApplied || response.Route != factory.PostRouteNewIssue ||
		response.Draft != factory.PostDraftApplied || response.Effect != factory.PostEffectVerified || response.Issue != 101 {
		t.Fatalf("drove response: %+v", response)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("calls judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
	// A replay recovers the same identity and verified link without any
	// new call, draft or native create.
	replayed, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.SourcePost != id || replayed.Effect != factory.PostEffectVerified || replayed.Issue != 101 {
		t.Fatalf("replayed response: %+v", replayed)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("replay spent: judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
}

func TestDrivePostDiscussionCreatesNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	if err := s.RecordRepositoryOpening(ctx, 7, 1729000000); err != nil {
		t.Fatal(err)
	}
	entries, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 20})
	if err != nil || len(entries) != 1 {
		t.Fatalf("opening question missing: %+v %v", entries, err)
	}
	id, revision, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, ParentID: entries[0].ID, Body: "thanks, acknowledged", IdempotencyKey: "key-1"}, "42", "ada", 1729000001)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordIssueMakerIntake(ctx, id, revision); err != nil {
		t.Fatal(err)
	}
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteDiscussion}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &factoryservice.PostCollaborators{Judge: judge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.PostRouteDiscussion || response.Draft != factory.PostDraftPending || response.Effect != factory.PostEffectPending || response.Issue != 0 {
		t.Fatalf("discussion response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("discussion created an issue")
	}
	if judge.calls != 1 || judge.input.Parent == nil || judge.input.Parent.Body != store.RepositoryOpeningQuestion {
		t.Fatalf("human reply was not evaluated with its Factory question: %+v", judge.input)
	}
}

func TestDrivePostMalformedDraftBlocksEffect(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	broken := validTestDraft(id)
	broken.Acceptance = ""
	drafter := &fakePostDrafter{err: errors.New("source assembly unavailable")}
	wired := &factoryservice.PostCollaborators{
		Judge:   &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}},
		Drafter: drafter,
		Creator: &fakePostCreator{next: 100, issues: map[string]int64{}},
	}
	for range factory.PostAttemptLimit {
		if _, err := factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetryFeedPost(ctx, id, "42", 0, "", 1729000050); err != nil {
		t.Fatal(err)
	}
	drafter.err, drafter.draft = nil, broken
	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	// The retry re-routes the request as a new source revision.
	if response.Draft != factory.PostDraftInvalid || response.Effect != factory.PostEffectBlocked || response.Revision != 1 {
		t.Fatalf("malformed response: %+v", response)
	}
	if _, err := s.PostDraft(ctx, id, 1); err == nil {
		t.Fatal("refused draft stays servable")
	}
}

func TestDrivePostEffectReplyLossReconciles(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	// The native issue 101 already exists from a lost reply: the
	// creator knows it, the pipeline does not.
	creator := &fakePostCreator{next: 101, issues: map[string]int64{id: 101}}
	creator.lookupErr = errors.New("declared unknown native lookup")
	wired := &factoryservice.PostCollaborators{
		Judge:   &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}},
		Drafter: &fakePostDrafter{draft: validTestDraft(id)},
		Creator: creator,
	}
	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil || response.Effect == factory.PostEffectVerified || creator.creates != 0 {
		t.Fatal("unknown lookup authorized another issue creation", response, creator.creates, err)
	}
	creator.lookupErr = nil
	response, err = factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.PostEffectVerified || response.Issue != 101 {
		t.Fatalf("reconciled response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("lost reply duplicated the issue")
	}
}

func TestDrivePostDeletedSourceSpendsNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}
	wired := &factoryservice.PostCollaborators{
		Judge:   judge,
		Drafter: &fakePostDrafter{draft: validTestDraft(id)},
		Creator: &fakePostCreator{next: 100, issues: map[string]int64{}},
	}
	if _, err := s.DeleteFeedPost(ctx, id, "42", 0); err != nil {
		t.Fatal(err)
	}
	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.PostEffectBlocked {
		t.Fatalf("deleted response: %+v", response)
	}
	if judge.calls != 0 {
		t.Fatal("deleted source spent a judgment call")
	}
}

func TestDrivePostJudgmentFailureExhaustsHonestly(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{err: errors.New("jev down")}}
	var response factory.PostResponse
	var err error
	for range factory.PostAttemptLimit {
		response, err = factoryservice.DrivePostIntake(ctx, s, wired, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if response.Judgment != factory.PostJudgmentFailed {
		t.Fatalf("exhausted judgment = %q", response.Judgment)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != factory.PostAttemptLimit {
		t.Fatalf("attempts = %d", pipe.Attempts)
	}
}

func TestDrivePostUnwiredStagesWaitInPending(t *testing.T) {
	t.Run("captured_request_requires_its_issue_maker", func(t *testing.T) {
		s, ctx := driveTestStore(t)
		id := saveDrivePost(t, s, ctx, "Keep the original captured request", "captured-unwired", true)
		drafter := &fakePostDrafter{draft: validTestDraft(id)}
		creator := &fakePostCreator{next: 100}
		wired := &factoryservice.PostCollaborators{
			Judge:   &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}},
			Drafter: drafter, Creator: creator,
		}
		response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
		if err != nil || response.Judgment != factory.PostJudgmentApplied || response.Draft != factory.PostDraftPending || response.Effect != factory.PostEffectPending || response.Attempts != 0 || drafter.calls != 0 || creator.creates != 0 {
			t.Fatal("missing captured-path collaborator entered legacy authoring", response, drafter.calls, creator.creates, err)
		}
	})
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	response, err := factoryservice.DrivePostIntake(ctx, s, &factoryservice.PostCollaborators{}, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Intake != factory.PostIntakeRecorded || response.Judgment != factory.PostJudgmentPending {
		t.Fatalf("unwired response: %+v", response)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != 0 {
		t.Fatalf("unwired drive recorded %d attempts", pipe.Attempts)
	}
}

// TestDrivePendingPostsResumesAfterRequestEnd proves the durable
// continuation: the POST's two durable writes (comment + intake) are
// all that survive the request, and service passes alone carry the
// stored request to its verified issue. A still-settling planner
// keeps the draft pending with attempts intact; the next pass
// finishes it, and the issue appears exactly once in readback.
func TestDrivePendingPostsResumesAfterRequestEnd(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := t.Context()
	s, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	// Exactly what the POST persists before answering promptly.
	id := saveDrivePost(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}
	drafter := &fakePostDrafter{draft: validTestDraft(id), errs: []error{factoryservice.ErrPostDraftPending}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &factoryservice.PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	// An unreadable/unsupported stored path cannot silently become legacy
	// authoring. Restore the exact record before the normal recovery journey.
	inspection, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = inspection.Close() })
	var recordedPath string
	if err := inspection.QueryRowContext(ctx, `SELECT data::text FROM factory_post_intakes WHERE post=$1`, id).Scan(&recordedPath); err != nil {
		t.Fatal(err)
	}
	if _, err := inspection.ExecContext(ctx, `UPDATE factory_post_intakes SET data=jsonb_set(data,'{issue_maker_version}','"unsupported"'::jsonb) WHERE post=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PostUsesIssueMaker(ctx, id, 0); err == nil {
		t.Fatal("unknown stored intake version became a legacy path")
	}
	// The declared routing peer may choose new work, but an unknown stored
	// intake path still cannot authorize drafting or a native effect.
	if _, err := factoryservice.DrivePostIntake(ctx, s, wired, id); err == nil || drafter.calls != 0 || creator.creates != 0 {
		t.Fatal("unknown intake path reached authoring/effects", drafter.calls, creator.creates, err)
	}
	if _, err := inspection.ExecContext(ctx, `UPDATE factory_post_intakes SET data=$2::jsonb,judgment='pending',route='' WHERE post=$1`, id, recordedPath); err != nil {
		t.Fatal(err)
	}
	judge.calls = 0

	if driven := factoryservice.DrivePendingPosts(ctx, s, wired, factoryservice.PostDriveLimit); driven != 1 {
		t.Fatalf("first pass driven %d, want 1", driven)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Judgment != factory.PostJudgmentApplied || pipe.Route != factory.PostRouteNewIssue ||
		pipe.Draft != factory.PostDraftPending || pipe.Attempts != 0 {
		t.Fatalf("pending planner burned state: %+v", pipe)
	}
	if creator.creates != 0 {
		t.Fatal("pending planner created an issue")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	// A new service composition reads the retained stage; it must not route
	// the saved request again or allocate another native effect.
	restartedJudge := &fakePostJudge{err: errors.New("unexpected repeated routing after restart")}
	wired = &factoryservice.PostCollaborators{
		Judge: restartedJudge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator,
	}

	if driven := factoryservice.DrivePendingPosts(ctx, s, wired, factoryservice.PostDriveLimit); driven != 1 {
		t.Fatalf("second pass driven %d, want 1", driven)
	}
	pipe, err = s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Draft != factory.PostDraftApplied || pipe.Effect != factory.PostEffectVerified || pipe.Issue != 101 {
		t.Fatalf("resumed pipeline: %+v", pipe)
	}
	if judge.calls != 1 || restartedJudge.calls != 0 || creator.creates != 1 {
		t.Fatalf("resume spent twice: judge=%d restarted=%d creates=%d", judge.calls, restartedJudge.calls, creator.creates)
	}
	if status := store.PostStatus(&pipe); status != "issue:101" {
		t.Fatalf("readback status %q, want issue:101", status)
	}
	// Settled requests leave the service work list entirely.
	if driven := factoryservice.DrivePendingPosts(ctx, s, wired, factoryservice.PostDriveLimit); driven != 0 {
		t.Fatalf("settled request drove %d more times", driven)
	}
	if creator.creates != 1 {
		t.Fatal("settled request created a duplicate issue")
	}
}

// TestDrivePostExistingWorkPersistsSelectedIssue proves an
// existing_work judgment persists WHICH tracked work it selected,
// the judge saw the candidate it picked, and a receipt recorded
// before a crash replays without a second judgment call.
func TestFactoryBackgroundResumesWithoutHTTP(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := t.Context()
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	id := saveDrivePost(t, db, ctx, "Please fix the login redirect", "background-reopen")
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}
	drafter := &fakePostDrafter{errs: []error{factoryservice.ErrPostDraftPending}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	runner := factoryservice.Background{Store: db, Stages: factoryservice.PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}}
	_ = runner.Collaborators()
	if judge.calls != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatal("composition drove a stage")
	}
	runner.Drive(ctx)
	pipe, err := db.PostIntake(ctx, id)
	if err != nil || pipe.Judgment != factory.PostJudgmentApplied || pipe.Draft != factory.PostDraftPending || pipe.Attempts != 0 || judge.calls != 1 || creator.creates != 0 {
		t.Fatal("background lost pending authored stage", pipe, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	restartedJudge := &fakePostJudge{err: errors.New("unexpected repeated routing")}
	runner = factoryservice.Background{Store: db, Stages: factoryservice.PostCollaborators{Judge: restartedJudge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}}
	runner.Drive(ctx)
	runner.Drive(ctx)
	pipe, err = db.PostIntake(ctx, id)
	if err != nil || pipe.Effect != factory.PostEffectVerified || pipe.Issue != 101 || creator.creates != 1 || restartedJudge.calls != 0 {
		t.Fatal("reopen duplicated or lost saved stage", pipe, creator.creates, err)
	}
	// Exercise the production Choice transport through a declared HTTPS peer;
	// no configured account or external provider receives this request.
	var calls atomic.Int32
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Model     string `json:"model"`
			Questions map[string]struct {
				Criteria map[string]any `json:"criteria"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Model != "fixture-background" || r.Header.Get("Authorization") != "Bearer fixture-background-key" {
			t.Error("background Choice input changed")
			http.Error(w, "invalid fixture", 400)
			return
		}
		criteria := request.Questions["priority"].Criteria
		if _, ok := criteria[factory.PostRouteDiscussion]; !ok {
			t.Error("routing policy omitted discussion")
			http.Error(w, "missing route", 400)
			return
		}
		calls.Add(1)
		probabilities := map[string]float64{}
		for key := range criteria {
			probabilities[key] = 0
		}
		probabilities[factory.PostRouteDiscussion] = 1
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture-background", "answers": map[string]any{"priority": map[string]any{"type": "choice", "choice": factory.PostRouteDiscussion, "confidence": 1, "probabilities": probabilities}}})
	}))
	defer provider.Close()
	wire := provider.Client().Transport.(*http.Transport).Clone()
	wire.TLSClientConfig.ServerName = provider.Certificate().DNSNames[0]
	wire.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, provider.Listener.Addr().String())
	}
	defer wire.CloseIdleConnections()
	keyfile := filepath.Join(t.TempDir(), "background.key")
	application := factoryservice.NewApplication(factoryservice.ApplicationPorts{Store: db, CurrentChooser: func() control.ProductJudgment {
		if !control.TypeSafeKeyReadable(keyfile) {
			return nil
		}
		return &control.TypeSafePriority{APIKeyFile: keyfile, Model: "fixture-background", Client: &http.Client{Transport: wire}}
	}})
	if application.Coordinator.Store != db || application.Coordinator.CommentDrive == nil || calls.Load() != 0 {
		t.Fatal("application construction started work or lost its canonical store/continuation")
	}
	runner = application.Background()
	discussion := saveDrivePost(t, db, ctx, "Thanks for the explanation", "background-discussion")
	if err := db.SavePostIssueContext(ctx, discussion, 0, []factory.PostWorkCandidate{}); err != nil {
		t.Fatal(err)
	}
	application.Coordinator.CommentDrive(ctx)
	if pipe, err := db.PostIntake(ctx, discussion); err != nil || pipe.Judgment != factory.PostJudgmentPending || pipe.Attempts != 0 || calls.Load() != 0 {
		t.Fatal("disconnected judgment spent a stage", pipe, err)
	}
	if err := os.WriteFile(keyfile, []byte("fixture-background-key"), 0600); err != nil {
		t.Fatal(err)
	}
	if runner.Collaborators().Judge == nil || calls.Load() != 0 {
		t.Fatal("readback judged or ignored current provider")
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	runner.Drive(stopped)
	if calls.Load() != 0 {
		t.Fatal("cancelled runner called provider")
	}
	application.Coordinator.CommentDrive(ctx)
	runner.Drive(ctx)
	pipe, err = db.PostIntake(ctx, discussion)
	if err != nil || pipe.Route != factory.PostRouteDiscussion || pipe.Judgment != factory.PostJudgmentApplied || calls.Load() != 1 || creator.creates != 1 {
		t.Fatal("connected runner lost exact routing", pipe, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	application = factoryservice.NewApplication(factoryservice.ApplicationPorts{Store: db, CurrentChooser: application.Chooser})
	runner = application.Background()
	application.Coordinator.CommentDrive(ctx)
	if retained, err := db.PostIntake(ctx, discussion); err != nil || retained.Judgment != factory.PostJudgmentApplied || calls.Load() != 1 {
		t.Fatal("fresh application repeated or lost its retained judgment", retained, err)
	}
	if err := os.Remove(keyfile); err != nil {
		t.Fatal(err)
	}
	withdrawn := saveDrivePost(t, db, ctx, "Another discussion", "background-withdrawn")
	if err := db.SavePostIssueContext(ctx, withdrawn, 0, []factory.PostWorkCandidate{}); err != nil {
		t.Fatal(err)
	}
	runner.Drive(ctx)
	if pipe, err := db.PostIntake(ctx, withdrawn); err != nil || pipe.Judgment != factory.PostJudgmentPending || pipe.Attempts != 0 || calls.Load() != 1 {
		t.Fatal("withdrawn judgment provider was retained", pipe, err)
	}
}

func TestDrivePostExistingWorkPersistsSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{
		Repository: 7, Body: "this belongs with the filter work", Issue: 55,
		IdempotencyKey: "key-existing",
	}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	judge := &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteExistingWork, Issue: 55}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &factoryservice.PostCollaborators{Judge: judge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := factoryservice.DrivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.PostRouteExistingWork || response.Issue != 55 || response.ExistingIssue != 55 {
		t.Fatalf("existing work response lost its selection: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("existing work created an issue")
	}
	found := false
	for _, candidate := range judge.input.Candidates {
		if candidate.Issue == 55 && candidate.Origin == factory.PostCandidateBound {
			found = true
		}
	}
	if !found {
		t.Fatalf("judge never saw bound candidate 55: %+v", judge.input.Candidates)
	}
}

// TestPostJudgeChoiceHelpers pins the candidate-bound choice
// contract: existing-work keys exist only for listed candidates,
// selections outside the offered set refuse, and the judge state
// carries source, parent and candidate evidence.
func TestPostJudgeChoiceHelpers(t *testing.T) {
	candidates := []factory.PostWorkCandidate{
		{Issue: 55, Origin: factory.PostCandidateBound, Summary: "filter work"},
		{Issue: 9, Origin: factory.PostCandidateRecent},
	}
	keys := factoryservice.PostChoiceKeys(candidates)
	want := []string{"new_issue", "discussion", "clarify_target", "clarify_result", "clarify_conflict", "existing_55", "existing_9"}
	if len(keys) != len(want) {
		t.Fatalf("keys %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys %v, want %v", keys, want)
		}
	}
	if keys := factoryservice.PostChoiceKeys(nil); len(keys) != 5 {
		t.Fatalf("candidate-less keys %v must exclude existing work", keys)
	}
	judgment, err := factoryservice.PostJudgmentFromSelection("existing_9", candidates)
	if err != nil || judgment.Route != factory.PostRouteExistingWork || judgment.Issue != 9 {
		t.Fatalf("selection parse: %+v err=%v", judgment, err)
	}
	if _, err := factoryservice.PostJudgmentFromSelection("existing_404", candidates); err == nil {
		t.Fatal("unlisted existing work admitted")
	}
	if _, err := factoryservice.PostJudgmentFromSelection("delete_everything", candidates); err == nil {
		t.Fatal("unknown selection admitted")
	}
	input := factoryservice.PostJudgeInput{
		SourcePost: store.FeedPostDetail{
			Record: factory.PostRecord{Repository: 7, Issue: 55, ParentID: "parent"},
			Body:   "more filter",
			Source: &factory.PostSource{Kind: factory.PostSourceRecord, ID: "rec", Issue: 55, Summary: "built filter"},
		},
		Parent:     &factoryservice.PostParentEvidence{SourcePost: "parent", Body: "start filter", Issue: 55},
		Candidates: candidates,
	}
	state := factoryservice.PostJudgeState(input)
	source, ok := state["source"].(map[string]any)
	if !ok || source["summary"] != "built filter" || source["issue"] != int64(55) {
		t.Fatalf("state source missing evidence: %+v", state["source"])
	}
	parent, ok := state["parent"].(map[string]any)
	if !ok || parent["body"] != "start filter" {
		t.Fatalf("state parent missing evidence: %+v", state["parent"])
	}
	listed, ok := state["candidates"].([]map[string]any)
	if !ok || len(listed) != 2 || listed[0]["summary"] != "filter work" {
		t.Fatalf("state candidates missing evidence: %+v", state["candidates"])
	}
}

// TestRecordInvitationThreadReachesOriginalAssignment proves the FUX14
// API-level leg: an eligible role is invited on a real finished assignment
// record (never a separate authored post), and its worker reply lands in the
// original record thread behind the existing ResolvePostSource and
// FeedReplyIDs owners. JEV threshold, per-revision idempotency and the
// receipt/consultation/reply loop exclusion are preserved.
// afterContributionChoices changes a live fixture only after every independent
// HTTPS Choice reply has arrived, before the shared runner settles its decision.
type afterContributionChoices struct {
	delegate control.ProductJudgment
	finished atomic.Int32
	after    func() error
}

func (c *afterContributionChoices) Select(ctx context.Context, state any, question string, choices map[string]any) (factory.PriorityAnswer, error) {
	answer, err := c.delegate.Select(ctx, state, question, choices)
	if err == nil && c.finished.Add(1) == int32(len(factory.ContributionQuestions)) {
		err = c.after()
	}
	return answer, err
}

func TestRecordInvitationThreadReachesOriginalAssignment(t *testing.T) {
	const repository int64 = 9223372036854775807
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	s, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	now := time.Now().Unix()
	digest := strings.Repeat("a", 64)
	capability := factory.MCPCapability{
		ID: factory.NewID(), AssignmentID: factory.NewID(), RunID: "run-1", Generation: 1,
		Profile: factory.MCPProfileInspector, Subject: "review:42", Repository: repository,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityStaged,
	}
	eligible := factory.Assignment{
		ID: capability.AssignmentID, AttemptRoot: capability.AssignmentID,
		PublicationAssignment: capability.AssignmentID, ProjectID: capability.Project,
		Role: "reviewer", Acceptance: "d" + strings.Repeat("a", 24),
		Preparation: "f" + strings.Repeat("b", 24),
		Run:         capability.RunID, RunHistory: []string{capability.RunID},
		Repository: repository, Issue: 42, CreatedUnix: now,
		AcceptedContext: &factory.AcceptedContext{Title: "Equipment approval policy", Body: "Implement protected manager and Finance approvals without self-approval."},
	}
	eligibleRaw, err := json.Marshal(eligible)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$4,42,$2,'assigned',1,$3)`,
		eligible.ID, eligible.Run, string(eligibleRaw), repository); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,'{}')`, capability.RunID); err != nil {
		t.Fatal(err)
	}
	if err := s.StageMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	if err := s.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, time.Now().Unix(), false); err != nil {
		t.Fatal(err)
	}
	capability, err = s.GetMCPCapability(ctx, capability.ID)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := factory.NewID()
	source := factory.Assignment{
		ID: sourceID, AttemptRoot: sourceID, PublicationAssignment: sourceID, ProjectID: capability.Project,
		Role: "coder", Phase: "implement", Acceptance: "d" + strings.Repeat("a", 24),
		Preparation: "f" + strings.Repeat("b", 24),
		Run:         "run-" + sourceID[:8], RunHistory: []string{"run-" + sourceID[:8]},
		Repository: repository, Issue: 42, CreatedUnix: now, FinishedUnix: now,
	}
	result := factory.ResultSynthesized(sourceID, source.Run, "succeeded", "Implemented the delivery guard without self-approval; candidate abc123.", now)
	source.Result = &result
	sourceRaw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$4,42,$2,'finished',1,$3)`,
		sourceID, source.Run, string(sourceRaw), repository); err != nil {
		t.Fatal(err)
	}
	sources, err := s.ContributionSources(ctx)
	if err != nil || len(sources) != 1 || sources[0] != sourceID {
		t.Fatalf("record undiscovered via API owners: %v %v", sources, err)
	}
	review, err := s.ContributionContext(ctx, sourceID)
	if err != nil || review.SourceKind != factory.PostSourceRecord || len(review.Roles) != 1 {
		t.Fatalf("record context via API owners: %+v %v", review, err)
	}
	// Exercise the actual assessment path through a declared HTTPS JEV peer.
	// Only the assessment post retains its questions, answers and decision.
	var judged atomic.Int32
	var weakConfidence atomic.Bool
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			State struct {
				Role factory.ContributionRole `json:"eligible_running_role"`
			} `json:"state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.State.Role.Assignment != capability.AssignmentID {
			t.Error("assessment omitted the eligible role", err)
			http.Error(w, "missing role", http.StatusBadRequest)
			return
		}
		judged.Add(1)
		confidence := 0.9
		if weakConfidence.Load() {
			confidence = 0.2
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture-jev", "answers": map[string]any{"priority": map[string]any{"type": "choice", "choice": "contribute", "confidence": confidence, "probabilities": map[string]float64{"contribute": 0.95, "no_contribution": 0.05}}}})
	}))
	t.Cleanup(provider.Close)
	wire := provider.Client().Transport.(*http.Transport).Clone()
	wire.TLSClientConfig.ServerName = provider.Certificate().DNSNames[0]
	wire.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, provider.Listener.Addr().String())
	}
	previousTransport := http.DefaultTransport
	http.DefaultTransport = wire
	t.Cleanup(func() {
		http.DefaultTransport = previousTransport
		wire.CloseIdleConnections()
	})
	keyfile := filepath.Join(t.TempDir(), "jev.key")
	if err := os.WriteFile(keyfile, []byte("fixture-jev-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	chooser := &control.TypeSafePriority{APIKeyFile: keyfile, Model: "fixture-jev"}
	continuation := factoryservice.Contributions{Store: s, Chooser: chooser}
	if claimed, err := (factoryservice.Contributions{Store: s}).Drive(ctx); err != nil || claimed || judged.Load() != 0 {
		t.Fatal("unwired continuation claimed or judged a source", claimed, err)
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if claimed, err := continuation.Drive(stopped); !errors.Is(err, context.Canceled) || claimed || judged.Load() != 0 {
		t.Fatal("cancelled continuation claimed or judged a source", claimed, err)
	}
	background := factoryservice.Background{Store: s}
	background.Drive(ctx)
	if judged.Load() != 0 {
		t.Fatal("unwired background claimed a contribution")
	}
	background.Chooser = func() control.ProductJudgment { return chooser }
	background.Drive(ctx)
	if judged.Load() != int32(len(factory.ContributionQuestions)) {
		t.Fatalf("independent contribution calls = %d", judged.Load())
	}
	invitation := factory.ContributionReviewInvitationID(review)
	assessment, err := s.FeedPost(ctx, invitation)
	if err != nil || assessment.Contribution == nil || assessment.Contribution.State != "invited" || len(assessment.Contribution.Calls) != len(factory.ContributionQuestions) || assessment.Contribution.Assignment != capability.AssignmentID {
		t.Fatalf("assessment post lost its calls or decision: %+v %v", assessment, err)
	}
	// Render the real persisted invitation through the same response owner used
	// by post GET, replies and combined-feed enrichment. Neither scope may round,
	// and projecting the wire must not rewrite the retained contribution.
	wireJSON, err := json.Marshal(factoryservice.WorkerPostResponse(assessment))
	if err != nil {
		t.Fatal(err)
	}
	var postWire struct {
		Repository   string `json:"repository"`
		Contribution struct {
			Repository string `json:"repository"`
		} `json:"contribution"`
	}
	if err := json.Unmarshal(wireJSON, &postWire); err != nil || postWire.Repository != strconv.FormatInt(repository, 10) || postWire.Contribution.Repository != postWire.Repository || assessment.Contribution.Repository != repository {
		t.Fatalf("worker contribution lost exact wire/retained scope: %+v %v", postWire, err)
	}
	if claimed, err := continuation.Drive(ctx); err != nil || claimed || judged.Load() != int32(len(factory.ContributionQuestions)) {
		t.Fatal("settled contribution was assessed again", claimed, judged.Load(), err)
	}
	resolved, err := s.ResolvePostSource(ctx, repository, factory.PostSourceRecord, sourceID)
	if err != nil || resolved.ID != sourceID {
		t.Fatalf("record source unresolved: %+v %v", resolved, err)
	}
	submit := func(tool factory.MCPTool, payload []byte) store.FeedPostDetail {
		t.Helper()
		submission := factory.MCPSubmission{
			AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
			CapabilityID: capability.ID, Tool: tool,
			Payload: payload,
		}
		receipt, err := s.RecordWorkerPost(ctx, submission)
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	receipt := submit(factory.MCPToolReply, []byte(`{"title":"Approval boundary","summary":"I checked the delivery guard and the self-approval case still fails before changing code.","parent_id":"`+invitation+`"}`))
	thread, _, err := s.FeedReplyIDs(ctx, repository, "record", sourceID, 0)
	if err != nil || len(thread) != 2 || thread[0] != receipt.Record.ID || thread[1] != invitation {
		t.Fatalf("reply missed original record thread: %v %v", thread, err)
	}
	page, err := s.MCPDiscussion(ctx, capability, invitation, "")
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != receipt.Record.ID {
		t.Fatalf("invited thread unreadable: %+v %v", page, err)
	}
	duplicate := factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
		CapabilityID: capability.ID, Tool: factory.MCPToolReply,

		Payload: []byte(`{"title":"Again","summary":"Second reply.","parent_id":"` + invitation + `"}`),
	}
	if _, err := s.RecordWorkerPost(ctx, duplicate); err == nil {
		t.Fatal("record invitation admitted a second reply")
	}

	// A new source revision still requires confident JEV agreement, even when
	// the provider strongly favors contributing. Its real post records refusal.
	weakConfidence.Store(true)
	if _, err := raw.ExecContext(ctx, `UPDATE factory_assignments SET revision=2 WHERE id=$1`, sourceID); err != nil {
		t.Fatal(err)
	}
	uncertain, err := s.ContributionContext(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := continuation.Drive(ctx); err != nil || !claimed {
		t.Fatal("new source revision was not assessed", claimed, err)
	}
	assessment, err = s.FeedPost(ctx, factory.ContributionReviewInvitationID(uncertain))
	if err != nil || assessment.Contribution == nil || assessment.Contribution.State != "not_invited" || assessment.Contribution.Assignment != "" {
		t.Fatalf("uncertain judgment bypassed the confidence cutoff: %+v %v", assessment, err)
	}
	// Cancellation after all valid replies but before selection must settle an
	// unavailable decision, not publish an invitation during shutdown.
	weakConfidence.Store(false)
	if _, err := raw.ExecContext(ctx, `UPDATE factory_assignments SET revision=3 WHERE id=$1`, sourceID); err != nil {
		t.Fatal(err)
	}
	cancelledReview, err := s.ContributionContext(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	callctx, cancel := context.WithCancel(ctx)
	defer cancel()
	continuation.Chooser = &afterContributionChoices{delegate: chooser, after: func() error { cancel(); return nil }}
	claimed, cancelledErr := continuation.Drive(callctx)
	assessment, err = s.FeedPost(ctx, factory.ContributionReviewInvitationID(cancelledReview))
	if !claimed || !errors.Is(cancelledErr, context.Canceled) || err != nil || assessment.Contribution == nil || assessment.Contribution.State != "unavailable" || assessment.Contribution.Assignment != "" {
		t.Fatal("late cancellation published an invitation or lost settlement", claimed, cancelledErr, assessment.Contribution, err)
	}
	// Eligibility is checked again at settlement, after actual model replies.
	if _, err := raw.ExecContext(ctx, `UPDATE factory_assignments SET revision=4 WHERE id=$1`, sourceID); err != nil {
		t.Fatal(err)
	}
	withdrawnReview, err := s.ContributionContext(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	continuation.Chooser = &afterContributionChoices{delegate: chooser, after: func() error {
		_, err := raw.ExecContext(ctx, `UPDATE factory_mcp_capabilities SET state='revoked' WHERE id=$1`, capability.ID)
		return err
	}}
	if claimed, err := continuation.Drive(ctx); err != nil || !claimed {
		t.Fatal("role withdrawal journey did not settle", claimed, err)
	}
	assessment, err = s.FeedPost(ctx, factory.ContributionReviewInvitationID(withdrawnReview))
	if err != nil || assessment.Contribution == nil || assessment.Contribution.State != "role_unavailable" || assessment.Contribution.Assignment != "" {
		t.Fatal("revoked role retained an invitation", assessment.Contribution, err)
	}
	var journalRows int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments`).Scan(&journalRows); err != nil || journalRows != 0 {
		t.Fatalf("contributions wrote duplicate judgments: %d %v", journalRows, err)
	}
}
