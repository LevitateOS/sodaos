package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func TestSharedRepositoryActivityAndSettings(t *testing.T) {
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
	const repository collaboration.RepositoryID = 9223372036854775807
	publication := factory.Publication{Repository: int64(repository), Issue: 3, Stage: factory.PublicationPublished, Outcome: factory.Succeeded, Reason: factory.PublishReasonLinked, PRID: 104, PRNumber: 4, FinishedUnix: 10, Publish: factory.PublicationOperation{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete}, PRCreate: factory.PublicationOperation{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete, PRID: 104, PRNumber: 4}}
	raw, _ := json.Marshal(publication)
	if _, err := inspect.ExecContext(ctx, `INSERT INTO factory_publications(assignment,repository,issue,run,stage,revision,data) VALUES('declared-publication',$1,3,'fixture-run','published',1,$2)`, repository, string(raw)); err != nil {
		t.Fatal(err)
	}
	publication.PRCreate.Completion = factory.OpCompletionPending
	raw, _ = json.Marshal(publication)
	if _, err := inspect.ExecContext(ctx, `INSERT INTO factory_publications(assignment,repository,issue,run,stage,revision,data) VALUES('unconfirmed-publication',$1,3,'fixture-run','published',1,$2)`, repository, string(raw)); err != nil {
		t.Fatal(err)
	}
	merge := factory.Merge{ID: "declared-merge", Repository: int64(repository), Issue: 3, IssueID: 103, PRID: 104, PRNumber: 4, Stage: factory.MergeMerged, Outcome: factory.Succeeded, Reason: factory.MergeReasonMerged, HeadOID: "exact-head", MergedCommit: "exact-head", MergedUnix: 20, ClosedUnix: 21, FinishedUnix: 22, OriginalClosingOccurrence: "34", OriginalClosedUnix: 21, Operation: factory.MergeOperation{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete, MergedCommit: "exact-head", PRID: 104, PRNumber: 4, IssueID: 103}}
	raw, _ = json.Marshal(merge)
	if _, err := inspect.ExecContext(ctx, `INSERT INTO factory_merges(publication,repository,issue,pr,stage,revision,data) VALUES('declared-merge',$1,3,4,'merged',1,$2)`, repository, string(raw)); err != nil {
		t.Fatal(err)
	}
	allowRead, allowWrite, revokeAfterRead, badNavigation := true, true, false, false
	reads, writes := 0, 0
	prefix := "https://review.example.test/old-name"
	denied := errors.New("declared current activity visibility denied")
	adminDenied := errors.New("declared current activity administration withdrawn")
	application := factoryservice.Activity{Store: db,
		Visibility: func(ctx context.Context, owner collaboration.ActorID, repo collaboration.RepositoryID) (factoryservice.ActivityNavigation, error) {
			reads++
			if owner != actor || repo != repository {
				t.Fatal("activity authority subject changed", owner, repo)
			}
			if !allowRead {
				return factoryservice.ActivityNavigation{}, denied
			}
			if badNavigation {
				return factoryservice.ActivityNavigation{}, nil
			}
			if reads == 2 {
				prefix = "https://review.example.test/renamed"
			}
			if revokeAfterRead {
				allowRead = false
			}
			return factoryservice.ActivityNavigation{RepositoryPath: prefix, ItemURL: func(item factory.ActivityItem) string {
				if item.Kind == "pr_created" {
					return prefix + "/change/" + item.PR
				}
				return prefix + "/completed/" + item.Issue
			}}, nil
		},
		WriteAccess: func(ctx context.Context, owner collaboration.ActorID, repo int64) error {
			writes++
			if owner != actor || repo != int64(repository) {
				t.Fatal("activity mutation subject changed", owner, repo)
			}
			if !allowWrite {
				return adminDenied
			}
			return nil
		},
	}
	facts, err := db.RepositoryActivity(ctx, int64(repository))
	if err != nil || len(facts.Items) != 2 || facts.RepositoryPath != "" || facts.Items[0].URL != "" || facts.Items[1].URL != "" {
		t.Fatal("facts reader fabricated host navigation", facts, err)
	}
	view, err := application.Read(ctx, actor, repository)
	if err != nil || view.Repository != strconv.FormatInt(int64(repository), 10) || view.RepositoryPath != prefix || len(view.Items) != 2 || view.Items[0].URL != prefix+"/completed/3" || view.Items[1].URL != prefix+"/change/4" || view.ObservedUnix <= 0 || writes != 0 || reads != 2 {
		t.Fatalf("shared activity facts/navigation: %+v %v reads=%d writes=%d", view, err, reads, writes)
	}
	allowRead = false
	if view, err := application.Read(ctx, actor, repository); !errors.Is(err, denied) || view.Repository != "" || view.Items != nil {
		t.Fatal("denied activity leaked facts", view, err)
	}
	allowRead, revokeAfterRead = true, true
	if view, err := application.Read(ctx, actor, repository); !errors.Is(err, denied) || view.Items != nil {
		t.Fatal("late visibility denial leaked activity", view, err)
	}
	allowRead, revokeAfterRead = true, false
	allowWrite = false
	settings := factory.ActivitySettings{PRCreated: true}
	if _, err := application.Update(ctx, actor, repository, 0, settings); !errors.Is(err, adminDenied) {
		t.Fatal("unauthorized settings saved", err)
	}
	current, revision, err := db.ActivitySettings(ctx, int64(repository))
	if err != nil || revision != 0 || current != factory.DefaultActivitySettings() {
		t.Fatal("denied write changed settings", current, revision, err)
	}
	allowWrite, badNavigation = true, true
	if _, err := application.Update(ctx, actor, repository, 0, settings); !errors.Is(err, factoryservice.ErrActivityNavigation) {
		t.Fatal("unavailable navigation allowed a write", err)
	}
	badNavigation = false
	view, err = application.Update(ctx, actor, repository, 0, settings)
	if err != nil || view.Revision != 1 || view.Settings != settings || len(view.Items) != 1 || view.Items[0].Kind != "pr_created" {
		t.Fatal("shared settings update/readback", view, err)
	}
	var recordedActor int64
	if err := inspect.QueryRowContext(ctx, `SELECT actor_id FROM factory_activity_settings WHERE repository=$1`, repository).Scan(&recordedActor); err != nil || recordedActor != int64(actor) {
		t.Fatal("settings lost canonical attribution", recordedActor, err)
	}
	if _, err := application.Update(ctx, actor, repository, 0, settings); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("stale settings CAS succeeded", err)
	}
	allowWrite = false
	if _, err := application.Update(ctx, actor, repository, 0, settings); !errors.Is(err, adminDenied) {
		t.Fatal("known revision bypassed current authority", err)
	}
	allowWrite = true
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if view, err := application.Read(cancelled, actor, repository); !errors.Is(err, context.Canceled) || view.Items != nil {
		t.Fatal("cancelled read leaked activity", view, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	application.Store = reopened
	view, err = application.Read(ctx, actor, repository)
	if err != nil || view.Revision != 1 || view.Settings != settings || len(view.Items) != 1 || view.Items[0].URL != prefix+"/change/4" {
		t.Fatal("reopened activity/settings changed", view, err)
	}
	if runs, err := reopened.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
		t.Fatal("activity launched work", runs, err)
	}
	var projects int
	if err := inspect.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&projects); err != nil || projects != 0 {
		t.Fatal("activity required native Project inventory", projects, err)
	}
}

func TestFactoryActivityNativeVisibilityOwnerAndCAS(t *testing.T) {
	s := factorySettingsServer(t)
	body := `{"expected_revision":0,"settings":{"issue_created":false,"pr_created":true,"verified_merge":false}}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/repositories/7/factory/activity", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out factory.RepositoryActivity
	decodeBody(t, w, &out)
	if out.Repository != "7" || out.RepositoryPath != "/alice/repo" || out.Revision != 0 || out.Settings != factory.DefaultActivitySettings() || out.Items == nil {
		t.Fatal(out)
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "bob"))
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	decodeBody(t, w, &out)
	if out.Revision != 1 || out.Settings.IssueCreated || !out.Settings.PRCreated {
		t.Fatal(out)
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"))
	if w.Code != 412 {
		t.Fatal(w.Code, w.Body.String())
	}
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation == extensions.OperationRepository {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "bob", Name: "renamed", Permission: "read"}}
		}
		owner := false
		return extensions.CallbackResponse{Owner: &owner}
	}
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"), callback)
	if w.Code != 403 {
		t.Fatal("operator bypassed native ownership", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories/7/factory/activity", "", "alice"), func(extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{ErrorCode: "not_found"}
	})
	if w.Code != http.StatusNotFound {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", `{"expected_revision":1,"settings":{"issue_created":false}}`, "alice"))
	if w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestFactoryActivityRejectsChangedNativeSessionBeforeMutation(t *testing.T) {
	s := factorySettingsServer(t)
	var changed atomic.Bool
	proxy := nativeProductProxyForActorWithCallbacks(t, s, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, "alice", func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation == extensions.OperationRepository {
			changed.Store(true)
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}}
		}
		owner := false
		return extensions.CallbackResponse{Owner: &owner}
	}, func(extensions.CallbackRequest) extensions.CallbackResponse {
		if changed.Load() {
			return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "2", Username: "bob"}}
		}
		return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}}
	})
	response := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "PUT", "/api/repositories/7/factory/activity", []byte(`{"expected_revision":0,"settings":{"issue_created":false,"pr_created":false,"verified_merge":false}}`))
	if response.StatusCode != 401 {
		t.Fatal(response.StatusCode)
	}
	settings, revision, err := s.Store.ActivitySettings(t.Context(), 7)
	if err != nil || revision != 0 || settings != factory.DefaultActivitySettings() {
		t.Fatal(settings, revision, err)
	}
}
