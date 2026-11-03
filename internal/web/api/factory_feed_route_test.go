package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryaccess "github.com/levitateos/factory/pkg/factory/access"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryruntime "github.com/levitateos/factory/pkg/factory/runtime"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

// The registered Factory catalogue works without any native Project inventory.
// The HTTP leg uses the real native authority callback and private PostgreSQL;
// the direct leg supplies a second admitted provider with colliding external IDs.
func TestFactoryRegisteredRepositoryCatalogue(t *testing.T) {
	product, db, dsn, call, _ := q05MuxFull(t, "")
	ctx := t.Context()
	inspect, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = inspect.Close() })
	var projects int
	if err := inspect.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&projects); err != nil || projects != 0 {
		t.Fatalf("fixture must have no native Project inventory: %d %v", projects, err)
	}
	// Stored labels are observations, not current authority or display metadata.
	if _, err := db.ObserveCollaborationRepository(ctx, product.FountainIdentities.HostID(), "7", "stale/private-label"); err != nil {
		t.Fatal(err)
	}
	read := func(path string) factoryservice.RepositoryCatalogue {
		t.Helper()
		code, body := call("GET", path, "")
		if code != 200 {
			t.Fatalf("catalogue status=%d body=%s", code, body)
		}
		var page struct {
			factoryservice.RepositoryCatalogue
			Host  collaboration.HostID `json:"host"`
			Actor string               `json:"actor"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}
		if page.Host != product.FountainIdentities.HostID() || page.Actor != "1" || strings.Contains(string(body), "stale/private-label") {
			t.Fatalf("wrong scope or leaked observation: %s", body)
		}
		return page.RepositoryCatalogue
	}
	for _, path := range []string{
		"/api/factory/repositories?limit=0", "/api/factory/repositories?limit=101",
		"/api/factory/repositories?after=0", "/api/factory/repositories?after=01",
		"/api/factory/repositories?after=9223372036854775808", "/api/factory/repositories?limit=2&limit=3",
		"/api/factory/repositories?actor=2", "/api/factory/repositories?after=%xx",
	} {
		if code, body := call("GET", path, ""); code != 400 {
			t.Fatalf("invalid catalogue admitted %s: %d %s", path, code, body)
		}
	}
	first := read("/api/factory/repositories?limit=3")
	if len(first.Items) != 0 || first.NextAfter != "3" || first.Complete {
		t.Fatalf("denied page lost its continuation: %+v", first)
	}
	second := read("/api/factory/repositories?limit=3&after=" + first.NextAfter)
	if len(second.Items) != 0 || second.NextAfter != "6" || second.Complete {
		t.Fatalf("second denied page: %+v", second)
	}
	last := read("/api/factory/repositories?limit=3&after=" + second.NextAfter)
	if len(last.Items) != 1 || last.Items[0].Repository != 7 || last.Items[0].Name != "alice/Alpha" || last.NextAfter != "" || !last.Complete {
		t.Fatalf("current native visibility/label: %+v", last)
	}

	remote, err := db.PutCollaborationHost(ctx, collaboration.Host{ID: collaboration.HostID(strings.Repeat("e", 32)), Provider: collaboration.GitHub, APIBase: "https://api.github.com"})
	if err != nil {
		t.Fatal(err)
	}
	observe := func(sequence, external string) collaboration.RepositoryRef {
		t.Helper()
		if _, err := inspect.ExecContext(ctx, `SELECT setval('collaboration_repositories_id_seq',$1,false)`, sequence); err != nil {
			t.Fatal(err)
		}
		ref, err := db.ObserveCollaborationRepository(ctx, remote.ID, external, "stored/remote")
		if err != nil {
			t.Fatal(err)
		}
		return ref
	}
	other := observe("9007199254740993", "7")
	maximum := observe("9223372036854775807", "opaque-remote")
	const actor collaboration.ActorID = 9007199254740995
	if _, err := inspect.ExecContext(ctx, `SELECT setval('collaboration_actors_id_seq',$1,false)`, int64(actor)); err != nil {
		t.Fatal(err)
	}
	account, err := db.ObserveCollaborationActor(ctx, remote.ID, "opaque-account", "remote-owner")
	if err != nil || account.ID != actor {
		t.Fatal("native account canonical identity", account, err)
	}
	profile := collaboration.Profile{ID: collaboration.ProfileID(collaboration.NewID()), Name: "declared native account profile"}
	if err := db.PutCollaborationProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	connection := collaboration.ConnectionRef{ID: collaboration.ConnectionID(collaboration.NewID()), Profile: profile.ID, Host: remote.ID, Actor: actor}
	if err := db.PutCollaborationConnection(ctx, connection); err != nil {
		t.Fatal(err)
	}
	visible := true
	outage := errors.New("declared provider access outage")
	var accessErr error
	var seen []collaboration.RepositoryID
	var alterEvidence func(factoryaccess.ActorEvidence) factoryaccess.ActorEvidence
	var cancelDuringRead context.CancelFunc
	authority := factoryservice.RepositoryAuthority{Host: remote.ID, Actor: actor, Source: factoryaccess.NativeAccount, Connection: &connection.ID}
	authority.Current = func(ctx context.Context, id collaboration.RepositoryID, _ factoryaccess.Permission) (factoryservice.RepositoryAuthorization, error) {
		seen = append(seen, id)
		if accessErr != nil {
			return factoryservice.RepositoryAuthorization{}, accessErr
		}
		ref, err := db.CollaborationRepository(ctx, id)
		if err != nil {
			return factoryservice.RepositoryAuthorization{}, err
		}
		if !visible || ref.Host != remote.ID {
			return factoryservice.RepositoryAuthorization{}, factoryservice.ErrRepositoryHidden
		}
		bound, err := db.CollaborationConnection(ctx, connection.ID)
		if err != nil || bound != connection {
			t.Fatal("native account connection changed", bound, err)
		}
		now := time.Now()
		evidence := factoryaccess.ActorEvidence{Source: factoryaccess.NativeAccount, Host: ref.Host, Repository: id, Actor: bound.Actor, Connection: &bound.ID, Authority: "declared-current-native-account", Revision: "current-permission-1", Permissions: []factoryaccess.Permission{factoryaccess.Read}, ObservedAt: now, ExpiresAt: now.Add(time.Minute)}
		if alterEvidence != nil {
			evidence = alterEvidence(evidence)
		}
		admitted, err := factoryaccess.Admit(evidence, now)
		if err != nil {
			return factoryservice.RepositoryAuthorization{}, err
		}
		if cancelDuringRead != nil {
			cancelDuringRead()
		}
		return factoryservice.RepositoryAuthorization{Actor: admitted, Name: "current/remote"}, nil
	}
	service := factoryservice.Repositories{Store: db, Visibility: authority.Visibility}
	page, err := service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: 6, Limit: 2})
	if err != nil || len(page.Items) != 1 || page.Items[0].Repository != other.ID || page.NextAfter != "9007199254740993" || page.Complete || len(seen) != 2 {
		t.Fatalf("cross-provider canonical pagination: %+v %v checked=%v", page, err, seen)
	}
	lastRemote, err := service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: other.ID, Limit: 1})
	if err != nil || len(lastRemote.Items) != 1 || lastRemote.Items[0].Repository != maximum.ID || !lastRemote.Complete || lastRemote.NextAfter != "" {
		t.Fatalf("maximum int64 continuation: %+v %v", lastRemote, err)
	}
	raw, err := json.Marshal(lastRemote)
	if err != nil || !strings.Contains(string(raw), `"repository":"9223372036854775807"`) {
		t.Fatalf("numeric identity on wire: %s %v", raw, err)
	}
	visible = false
	page, err = service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: 7, Limit: 1})
	if err != nil || len(page.Items) != 0 || page.Complete || page.NextAfter != "9007199254740993" {
		t.Fatalf("revoked authority leaked a row/lost continuation: %+v %v", page, err)
	}
	accessErr = outage
	page, err = service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: 6, Limit: 2})
	if !errors.Is(err, outage) || page.Items != nil || page.Complete || page.NextAfter != "" {
		t.Fatalf("unknown authority reported a partial/complete catalogue: %+v %v", page, err)
	}
	accessErr, visible = nil, true
	// A real registered account/connection cannot borrow another principal's
	// evidence. No partial catalogue or stale display label may escape.
	for _, change := range []func(factoryaccess.ActorEvidence) factoryaccess.ActorEvidence{
		func(e factoryaccess.ActorEvidence) factoryaccess.ActorEvidence { e.Actor++; return e },
		func(e factoryaccess.ActorEvidence) factoryaccess.ActorEvidence {
			other := collaboration.ConnectionID(collaboration.NewID())
			e.Connection = &other
			return e
		},
	} {
		alterEvidence = change
		page, err = service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: other.ID, Limit: 1})
		if !errors.Is(err, factoryaccess.ErrDenied) || page.Items != nil || page.NextAfter != "" {
			t.Fatal("changed native principal disclosed catalogue", page, err)
		}
	}
	alterEvidence = nil
	if err := authority.Configure(ctx, actor, int64(other.ID)); !errors.Is(err, factoryaccess.ErrDenied) {
		t.Fatal("read admission granted configuration authority", err)
	}
	expired := authority
	expired.Now = func() time.Time { return time.Now().Add(time.Hour) }
	service.Visibility = expired.Visibility
	if page, err := service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: other.ID, Limit: 1}); !errors.Is(err, factoryaccess.ErrDenied) || page.Items != nil {
		t.Fatal("expired authority disclosed catalogue", page, err)
	}
	service.Visibility = authority.Visibility
	interrupted, interrupt := context.WithCancel(ctx)
	cancelDuringRead = interrupt
	if page, err := service.Catalogue(interrupted, actor, factoryservice.RepositoryQuery{After: other.ID, Limit: 1}); !errors.Is(err, context.Canceled) || page.Items != nil {
		t.Fatal("cancellation during current authority disclosed catalogue", page, err)
	}
	cancelDuringRead = nil
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.Catalogue(cancelled, actor, factoryservice.RepositoryQuery{Limit: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read: %v", err)
	}
	// Fountain never serves the other host's colliding native locator.
	page = read("/api/factory/repositories?after=7&limit=1")
	if len(page.Items) != 0 || page.NextAfter != "9007199254740993" || page.Complete {
		t.Fatalf("foreign provider leaked through Fountain: %+v", page)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	db, service.Store = reopened, reopened
	page, err = service.Catalogue(ctx, actor, factoryservice.RepositoryQuery{After: other.ID, Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0] != lastRemote.Items[0] || !page.Complete {
		t.Fatalf("reopened catalogue: %+v %v", page, err)
	}
	if err := inspect.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&projects); err != nil || projects != 0 {
		t.Fatalf("catalogue created native inventory: %d %v", projects, err)
	}
	var work int
	if err := inspect.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_runs)+(SELECT count(*) FROM factory_feed_posts)+(SELECT count(*) FROM factory_intent_heads)+(SELECT count(*) FROM factory_priority_judgments)+(SELECT count(*) FROM factory_usage)`).Scan(&work); err != nil || work != 0 {
		t.Fatalf("catalogue reads created workflow/model/usage records: %d %v", work, err)
	}
	// Compose the actual native application owner over this registered account
	// and reopened ledger. Account/provider truth remains a declared current peer;
	// there is no native browser authority, model call or executor in this journey.
	var admitted, bindingDown, mixedStore, withdrawDuringVisibility atomic.Bool
	var holdBinding atomic.Bool
	var bindingCalls atomic.Int32
	bindingEntered, bindingCancelled, releaseBinding := make(chan struct{}), make(chan struct{}), make(chan struct{})
	lateRead := make(chan error, 1)
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseBinding) })
	admitted.Store(true)
	var coordinatorOwner *control.Coordinator
	var boundStore *store.Store
	bindings := func(ctx context.Context, owned factoryservice.Application) (factoryhttpapi.Product, error) {
		bindingCalls.Add(1)
		if bindingDown.Load() {
			return factoryhttpapi.Product{}, errors.New("private declared account lookup failure")
		}
		if coordinatorOwner == nil || boundStore != owned.Store {
			coordinatorOwner, boundStore = owned.Coordinator, owned.Store
		} else if coordinatorOwner != owned.Coordinator {
			t.Error("native binding constructed a second application/coordinator")
			return factoryhttpapi.Product{}, errors.New("application owner changed")
		}
		registered, err := owned.Store.CollaborationConnection(ctx, connection.ID)
		if err != nil || registered != connection {
			return factoryhttpapi.Product{}, errors.New("current connection does not match profile/account")
		}
		if holdBinding.Swap(false) {
			close(bindingEntered)
			<-ctx.Done()
			close(bindingCancelled)
			<-releaseBinding
			// A declared slow account peer settles after its HTTP connection
			// disappears. The application must retain Store until it returns.
			settle, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			retained, err := owned.Store.CollaborationConnection(settle, connection.ID)
			cancel()
			if err == nil && retained != connection {
				err = errors.New("late account read changed retained connection")
			}
			lateRead <- err
		}
		visibility := func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) (string, error) {
			name, err := authority.Visibility(ctx, selected, repo)
			if withdrawDuringVisibility.Load() {
				admitted.Store(false)
			}
			return name, err
		}
		workVisibility := func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
			_, err := visibility(ctx, selected, repo)
			return err
		}
		posts := factoryservice.Posts{Store: owned.Store, Visibility: func(ctx context.Context, selected collaboration.ActorID, repo int64) error {
			return workVisibility(ctx, selected, collaboration.RepositoryID(repo))
		}}
		product := factoryhttpapi.Product{Store: owned.Store, Host: remote.ID, Actor: actor,
			Principal: "profile:declared-owner", AuthorName: "Current native owner",
			Current: func(w http.ResponseWriter, r *http.Request) bool {
				if !admitted.Load() || r.Header.Get("Authorization") != "Bearer declared-product-client" {
					w.WriteHeader(http.StatusUnauthorized)
					return false
				}
				return true
			},
			Feed: factoryhttpapi.Feed{Service: factoryservice.Feed{Store: owned.Store}, Visibility: visibility,
				Posts: posts, Commands: factoryservice.PostCommands{Posts: posts},
				Host: collaboration.HostID(factory.NewID()), Actor: 1, AuthorName: "Unbound group author"},
			Requests: factoryhttpapi.Requests{Service: factoryservice.Requests{Store: owned.Store, WriteAccess: authority.Configure}, Posts: posts, Actor: 1},
			Work:     factoryhttpapi.Work{Reads: factoryservice.WorkReads{Store: owned.Store, Visibility: workVisibility}},
			Repositories: factoryhttpapi.Repositories{Store: owned.Store, Visibility: workVisibility,
				Activity: factoryservice.Activity{Store: owned.Store, Visibility: func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) (factoryservice.ActivityNavigation, error) {
					if err := workVisibility(ctx, selected, repo); err != nil {
						return factoryservice.ActivityNavigation{}, err
					}
					return factoryservice.ActivityNavigation{RepositoryPath: "/declared/remote", ItemURL: func(factory.ActivityItem) string { return "/declared/remote/activity" }}, nil
				}}},
			Records:     factoryhttpapi.Records{Library: factoryservice.Library{Source: owned.Store, Visibility: workVisibility}, Artifacts: factoryservice.Artifacts{Source: owned.Store, Visibility: workVisibility}},
			Acceptances: factoryhttpapi.Acceptances{Visibility: workVisibility, Service: factoryservice.Acceptances{Writer: owned.Coordinator, WriteAccess: workVisibility}},
		}
		if mixedStore.Load() {
			product.Requests.Service.Store = &store.Store{}
		}
		return product, nil
	}
	var active atomic.Pointer[factoryruntime.NativeApplication]
	active.Store(factoryruntime.NewNativeApplication(factoryservice.ApplicationPorts{Store: db}, bindings))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { active.Load().ServeHTTP(w, r) }))
	defer func() {
		server.CloseClientConnections()
		releaseOnce.Do(func() { close(releaseBinding) })
		_ = active.Load().Close()
	}()
	t.Cleanup(server.Close)
	request := func(method, path, body string, expected int) []byte {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-product-client")
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != expected || expected < 300 && response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("native product HTTP", method, path, response.StatusCode, string(data), err)
		}
		return data
	}
	identity := request(http.MethodGet, "/api/factory/identity", "", 200)
	if !strings.Contains(string(identity), `"actor":"9007199254740995"`) || !strings.Contains(string(identity), string(remote.ID)) {
		t.Fatal("product identity was overridden by a route group", string(identity))
	}
	request(http.MethodGet, "/api/factory/repositories?after=9007199254740993&limit=1", "", 200)
	request(http.MethodPut, "/api/factory/feed/subscription", `{"repositories":["9007199254740993"],"expected_revision":0}`, 200)
	if saved, revision, err := db.FeedSubscription(ctx, strconv.FormatInt(int64(actor), 10)); err != nil || revision != 1 || len(saved.Repositories) != 1 || saved.Repositories[0] != int64(other.ID) {
		t.Fatal("product subscription lost bound actor/canonical scope", saved, revision, err)
	}
	department := `{"id":"d` + strings.Repeat("5", 24) + `","owner":"1","name":"Native direction","description":"Bound to current account"}`
	body := request(http.MethodPost, "/api/factory/departments", department, 201)
	if !strings.Contains(string(body), `"owner":"9007199254740995"`) {
		t.Fatal("product department accepted a body/group owner", string(body))
	}
	request(http.MethodGet, "/api/factory/departments", "", 200)
	submission := `{"repository":"9007199254740993","body":"Retained native direction","idempotency_key":"native-product-owned"}`
	body = request(http.MethodPost, "/api/factory/feed/posts", submission, 200)
	var submitted factory.PostResponse
	if json.Unmarshal(body, &submitted) != nil || submitted.SourcePost == "" {
		t.Fatal("product post response", string(body))
	}
	if saved, err := db.FeedPost(ctx, submitted.SourcePost); err != nil || saved.Record.Author != strconv.FormatInt(int64(actor), 10) || saved.AuthorName != "Current native owner" {
		t.Fatal("product post changed bound actor/display author", err)
	}
	request(http.MethodGet, "/api/factory/feed/posts/"+submitted.SourcePost, "", 200)
	request(http.MethodGet, "/api/factory/library?repository=9007199254740993", "", 200)
	request(http.MethodGet, "/api/factory/artifacts/"+factory.NewID()+"?repository=9007199254740993", "", 404)
	request(http.MethodGet, "/api/repositories/9007199254740993/factory/activity", "", 200)
	request(http.MethodGet, "/api/repositories/9007199254740993/factory/issues/3/assignment", "", 404)
	request(http.MethodGet, "/api/repositories/9007199254740993/factory/issues/3/acceptance-preview", "", 503)
	state := func() string {
		t.Helper()
		var value string
		if err := inspect.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM factory_departments d),
		 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM factory_feed_posts p),
		 (SELECT jsonb_agg(to_jsonb(i) || jsonb_build_object('data',i.data-'last_checked_at') ORDER BY post) FROM factory_post_intakes i),
		 (SELECT jsonb_agg(to_jsonb(s) ORDER BY subscriber) FROM factory_feed_subscriptions s),
		 (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_runs),
		 (SELECT count(*) FROM factory_priority_judgments),(SELECT count(*) FROM factory_usage))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	retained := state()
	admitted.Store(false)
	request(http.MethodGet, "/api/factory/departments", "", 401)
	request(http.MethodPost, "/api/factory/feed/posts", submission, 401)
	admitted.Store(true)
	withdrawDuringVisibility.Store(true)
	if body := request(http.MethodGet, "/api/factory/repositories?after=9007199254740993&limit=1", "", 401); len(body) != 0 {
		t.Fatal("late account withdrawal disclosed product catalogue", string(body))
	}
	withdrawDuringVisibility.Store(false)
	admitted.Store(true)
	bindingDown.Store(true)
	if body := request(http.MethodGet, "/api/factory/departments", "", 503); strings.Contains(string(body), "private declared") {
		t.Fatal("product binding exposed provider/account failure", string(body))
	}
	bindingDown.Store(false)
	mixedStore.Store(true)
	request(http.MethodGet, "/api/factory/departments", "", 503)
	mixedStore.Store(false)
	if state() != retained {
		t.Fatal("product refusal changed private/work journals")
	}
	// Closing the native application must join admitted account/product work,
	// including a handler still settling after forced HTTP disconnection.
	holdBinding.Store(true)
	requestDone := make(chan error, 1)
	// Use a fresh connection so net/http cannot transparently retry this GET
	// after the forced close and replace its disconnect with a new 503 response.
	heldClient := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 10 * time.Second}
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/factory/identity", nil)
		if err != nil {
			requestDone <- err
			return
		}
		req.Header.Set("Authorization", "Bearer declared-product-client")
		response, err := heldClient.Do(req)
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	select {
	case <-bindingEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("native request did not reach held account binding")
	}
	closing := active.Load()
	closing.StopAdmission()
	callsBeforeStop := bindingCalls.Load()
	request(http.MethodGet, "/api/factory/departments", "", 503)
	if bindingCalls.Load() != callsBeforeStop {
		t.Fatal("stopped native application invoked account binding")
	}
	closed := make(chan error, 1)
	go func() { closed <- closing.Close() }()
	server.CloseClientConnections()
	select {
	case <-bindingCancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("forced HTTP close did not cancel admitted account request")
	}
	select {
	case err := <-closed:
		t.Fatal("native application closed before admitted account work settled", err)
	case <-time.After(50 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(releaseBinding) })
	select {
	case err := <-lateRead:
		if err != nil {
			t.Fatal("admitted account work lost its live store during shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("admitted account work did not settle")
	}
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("native application did not drain released account work")
	}
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("forced HTTP close unexpectedly delivered a response")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("disconnected native client did not finish")
	}
	request(http.MethodGet, "/api/factory/departments", "", 503)
	if bindingCalls.Load() != callsBeforeStop || state() != retained {
		t.Fatal("closed native application entered binding or changed journals")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	active.Store(factoryruntime.NewNativeApplication(factoryservice.ApplicationPorts{Store: db}, bindings))
	request(http.MethodGet, "/api/factory/departments", "", 200)
	request(http.MethodGet, "/api/factory/feed/posts/"+submitted.SourcePost, "", 200)
	request(http.MethodPost, "/api/factory/feed/posts", submission, 200)
	if state() != retained {
		t.Fatal("native application reopen/replay changed retained product/work journals")
	}
}

// Complete pages use actual PostgreSQL posts/records/intakes through shared
// services and shared HTTP, with declared current authority and no native
// session or work driver.
func TestSharedCompleteFeedPage(t *testing.T) {
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
	const repository int64 = 9223372036854775807
	const actor collaboration.ActorID = 9007199254740993
	author := strconv.FormatInt(int64(actor), 10)
	save := func(key, body string) string {
		id, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: repository, Body: body, IdempotencyKey: key}, author, "Canonical owner", 1700000000)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RecordPostIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
		return id
	}
	older := save("older", "Old continuation")
	first := save("first", "Private first body")
	second := save("second", "Private second body")
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	system := factory.NewID()
	systemData, _ := json.Marshal(map[string]any{"actor": "system", "author_name": "System", "at": 1700000001, "parent_id": first, "communication": map[string]any{"kind": "system", "title": "Recorded progress", "summary": "Private system detail", "parent_id": first}})
	if _, err := raw.ExecContext(ctx, `INSERT INTO factory_feed_posts(id,repository,revision,author,body,idempotency,data) VALUES($1,$2,0,'worker:fixture','Private system detail',$1,$3::jsonb)`, system, repository, string(systemData)); err != nil {
		t.Fatal(err)
	}
	record := factory.NewID()
	assignment := factory.Assignment{ID: record, AttemptRoot: record, PublicationAssignment: record, ProjectID: "p" + strings.Repeat("1", 24), Role: "coder", Phase: "implement", Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24), Run: "run-1", RunHistory: []string{"run-1"}, Stage: factory.AssignmentFinished, Outcome: factory.Succeeded, Repository: repository, Issue: 3, CreatedUnix: 1700000000, FinishedUnix: 1700000002, Result: &factory.AssignmentResult{AssignmentID: record, RunID: "run-1", Status: "completed", Summary: "Private recorded result", RecordedUnix: 1700000002}}
	recordData, _ := json.Marshal(assignment)
	if _, err := raw.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,3,'run-1','finished',1,$3::jsonb)`, record, repository, string(recordData)); err != nil {
		t.Fatal(err)
	}
	feed := factoryservice.Feed{Store: db}
	if err := feed.SaveSubscription(ctx, actor, 0, factory.FeedSubscription{Repositories: []int64{repository}}); err != nil {
		t.Fatal(err)
	}
	judge := &fakePostJudge{}
	drafter := &fakePostDrafter{}
	creator := &fakePostCreator{}
	posts := factoryservice.Posts{Store: db, Collaborators: &factoryservice.PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}}
	query := factory.FeedQuery{Repositories: []int64{repository}, Limit: 4}
	checks := 0
	cutAt := 0
	var accessError error
	visible := func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) (string, error) {
		if admitted != actor || int64(repo) != repository {
			t.Fatal("page widened canonical authority", admitted, repo)
		}
		checks++
		if cutAt > 0 && checks >= cutAt {
			return "", accessError
		}
		return "current/repository", ctx.Err()
	}
	snapshot := func() string {
		var value string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_object('posts',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM factory_feed_posts p),'intakes',(SELECT jsonb_agg(to_jsonb(i) ORDER BY post) FROM factory_post_intakes i),'runs',(SELECT count(*) FROM factory_runs),'commands',(SELECT count(*) FROM factory_commands))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	page, err := feed.Page(ctx, actor, query, false, visible, posts)
	if err != nil || len(page.Entries) != 4 || page.Next == "" || !page.Truncated {
		t.Fatal("complete page lost its window", page, err)
	}
	seen := map[string]bool{}
	for _, entry := range page.Entries {
		seen[entry.ID] = true
		if entry.Project != "current/repository" || entry.Repository != repository {
			t.Fatal("page lost current canonical scope", entry)
		}
		if entry.ID == first || entry.ID == second {
			detail, ok := entry.Post.(map[string]any)
			if !ok || detail["post_id"] != entry.ID || detail["repository"] != strconv.FormatInt(repository, 10) || detail["author"] != author || detail["body"] != entry.Summary {
				t.Fatal("human post detail missing or out of scope", entry.Post)
			}
		} else if entry.ID == system && (entry.Payload == nil || entry.Payload.Communication == nil || entry.Post != nil) {
			t.Fatal("system producer replaced by human detail", entry)
		}
	}
	if !seen[first] || !seen[second] || !seen[system] || !seen[record] || seen[older] {
		t.Fatal("unexpected page members", seen)
	}
	// Both early entries and worker/system rows need a final check after detail
	// assembly: revoke at the second human detail read, after the first joined.
	checks = 0
	cutAt = len(page.Entries) + 2
	accessError = factoryservice.ErrRepositoryHidden
	hidden, err := feed.Page(ctx, actor, query, false, visible, posts)
	if err != nil || len(hidden.Entries) != 0 || hidden.Next != page.Next || hidden.Truncated != page.Truncated {
		t.Fatal("late revocation leaked a row or lost continuation", hidden, err)
	}
	checks = 0
	cutAt = 0
	resume := query
	resume.Cursor = hidden.Next
	continued, err := feed.Page(ctx, actor, resume, false, visible, posts)
	if err != nil || len(continued.Entries) != 1 || continued.Entries[0].ID != older {
		t.Fatal("filtered page hid authorized continuation", continued, err)
	}
	outage := errors.New("declared unknown current access")
	for _, at := range []int{1, len(page.Entries) + 2, len(page.Entries) + 3} {
		checks = 0
		cutAt = at
		accessError = outage
		if result, err := feed.Page(ctx, actor, query, false, visible, posts); !errors.Is(err, outage) || len(result.Entries) != 0 || result.Next != "" {
			t.Fatal("unknown authority returned a partial page", at, result, err)
		}
	}
	checks = 0
	cutAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	if result, err := feed.Page(cancelled, actor, query, false, func(ctx context.Context, a collaboration.ActorID, r collaboration.RepositoryID) (string, error) {
		cancel()
		return "old/name", nil
	}, posts); !errors.Is(err, context.Canceled) || len(result.Entries) != 0 {
		t.Fatal("cancelled page returned data", result, err)
	}
	if _, err := feed.Page(ctx, actor, query, false, nil, posts); !errors.Is(err, factoryservice.ErrFeedUnavailable) {
		t.Fatal("missing authority admitted", err)
	}
	if _, err := feed.Page(ctx, actor, query, false, visible, factoryservice.Posts{}); !errors.Is(err, factoryservice.ErrFeedUnavailable) {
		t.Fatal("missing detail ledger admitted", err)
	}
	other, _, otherCleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(otherCleanup)
	if _, err := feed.Page(ctx, actor, query, false, visible, factoryservice.Posts{Store: other}); !errors.Is(err, factoryservice.ErrFeedUnavailable) {
		t.Fatal("different ledger joined by numeric identity", err)
	}
	if snapshot() != before || judge.calls != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatal("page read drove or mutated work")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	feed.Store = db
	posts.Store = db
	checks = 0
	reopened, err := feed.Page(ctx, actor, query, false, visible, posts)
	encoded, _ := json.Marshal(page)
	again, _ := json.Marshal(reopened)
	if err != nil || string(encoded) != string(again) {
		t.Fatal("reopen changed complete page", err)
	}
	// The standalone-facing HTTP transport consumes these same canonical
	// services, with host admission and no API or native-session object.
	var admissionChecks, revokeAt atomic.Int32
	shared := factoryhttpapi.Feed{Service: feed, Posts: posts, Actor: actor,
		Host: collaboration.HostID(strings.Repeat("c", 32)), Visibility: visible,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			count := admissionChecks.Add(1)
			if r.Header.Get("Authorization") != "Bearer declared-feed-client" || revokeAt.Load() > 0 && count >= revokeAt.Load() {
				http.Error(w, "declared account admission unavailable", http.StatusUnauthorized)
				return false
			}
			return true
		}}
	server := httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-feed-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status {
			t.Fatalf("shared HTTP %s %s: %d %s %v", method, path, response.StatusCode, data, err)
		}
		if status == http.StatusOK && response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("shared feed response was cacheable")
		}
		return data
	}
	path := "/api/factory/feed?repositories=" + strconv.FormatInt(repository, 10) + "&limit=4"
	body := request(http.MethodGet, path, "", 200)
	var transported factoryservice.FeedPage
	if err := json.Unmarshal(body, &transported); err != nil {
		t.Fatal(err)
	}
	transportedJSON, _ := json.Marshal(transported)
	if string(transportedJSON) != string(encoded) {
		t.Fatal("shared HTTP changed canonical feed projection", string(body))
	}
	var subscription struct {
		Subscriber   string   `json:"subscriber"`
		Repositories []string `json:"repositories"`
		Revision     int64    `json:"revision"`
	}
	if err := json.Unmarshal(request(http.MethodGet, "/api/factory/feed/subscription", "", 200), &subscription); err != nil || subscription.Subscriber != author || subscription.Revision != 1 {
		t.Fatal("shared HTTP lost exact subscription identity", subscription, err)
	}
	change := `{"repositories":["9223372036854775807"],"expected_revision":1,"subscriber":"foreign-account"}`
	request(http.MethodPut, "/api/factory/feed/subscription", change, 200)
	request(http.MethodPut, "/api/factory/feed/subscription", change, 409)
	request(http.MethodPut, "/api/factory/feed/subscription", change+` {}`, 400)
	request(http.MethodPut, "/api/factory/feed/subscription", change+strings.Repeat(" ", 65536), 400)
	request(http.MethodDelete, "/api/factory/feed", "", 405)
	request(http.MethodGet, "/api/environments", "", 404)
	if _, revision, err := feed.Subscription(ctx, actor+2); err != nil || revision != 0 {
		t.Fatal("shared HTTP wrote another account's preferences", revision, err)
	}
	admissionChecks.Store(0)
	revokeAt.Store(2) // Admission changes after the actual feed read.
	if denied := request(http.MethodGet, path, "", 401); strings.Contains(string(denied), "Private") || strings.Contains(string(denied), "entries") {
		t.Fatal("revoked host admission disclosed recorded feed", string(denied))
	}
	revokeAt.Store(0)
	server.Close()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	shared.Service.Store, shared.Posts.Store = db, db
	server = httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	if err := json.Unmarshal(request(http.MethodGet, "/api/factory/feed/subscription", "", 200), &subscription); err != nil || subscription.Subscriber != author || subscription.Revision != 2 || len(subscription.Repositories) != 1 || subscription.Repositories[0] != "9223372036854775807" {
		t.Fatal("shared HTTP lost retained preferences", subscription, err)
	}
	if after := snapshot(); after != before || judge.calls != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatal("shared HTTP read/preferences drove or changed work")
	}
	// The same transport records human requests and revisions in the existing
	// journals while a declared disconnected provider keeps capture unavailable.
	server.Close()
	shared.AuthorName = "Declared canonical owner"
	var postVisible atomic.Bool
	postVisible.Store(true)
	shared.Posts.Visibility = func(ctx context.Context, current collaboration.ActorID, repo int64) error {
		if !postVisible.Load() || repo != repository || current != actor && current != actor+2 {
			return factoryservice.ErrPostVisibility
		}
		return ctx.Err()
	}
	var captures atomic.Int32
	shared.Commands = factoryservice.PostCommands{Posts: shared.Posts, Capture: factoryservice.PostCapture{
		Context: func(ctx context.Context, current collaboration.ActorID, id string, revision int64) error {
			detail, err := db.FeedPost(ctx, id)
			if err != nil || current != actor || detail.Record.Repository != repository || detail.Record.Author != author || detail.Record.Revision != revision {
				t.Error("HTTP capture lost canonical source scope", detail.Record, err)
			}
			captures.Add(1)
			return errors.New("declared provider context unavailable")
		},
	}}
	var loseRetryReply atomic.Bool
	postHandler := shared.Handler()
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/retry") && loseRetryReply.Swap(false) {
			// Run the actual handler/PG commit, then close the actual HTTP socket
			// before its receipt reaches the client. No synthetic success reply.
			response := httptest.NewRecorder()
			postHandler.ServeHTTP(response, r)
			if response.Code != 200 {
				t.Error("lost-reply command did not commit", response.Code, response.Body.String())
			}
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = connection.Close()
			return
		}
		postHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	postBody := `{"repository":"9223372036854775807","body":"Please preserve the original native request","idempotency_key":"standalone-http-post","author":"foreign-account"}`
	var saved factory.PostResponse
	if err := json.Unmarshal(request(http.MethodPost, "/api/factory/feed/posts", postBody, 200), &saved); err != nil || saved.SourcePost == "" || saved.Revision != 0 {
		t.Fatal("shared HTTP lost original durable request", saved, err)
	}
	var replay factory.PostResponse
	if err := json.Unmarshal(request(http.MethodPost, "/api/factory/feed/posts", postBody, 200), &replay); err != nil || replay.SourcePost != saved.SourcePost || replay.Revision != saved.Revision {
		t.Fatal("shared HTTP duplicated an exact submission", replay, err)
	}
	detail, err := db.FeedPost(ctx, saved.SourcePost)
	if err != nil || detail.Record.Author != author || detail.AuthorName != shared.AuthorName || detail.Body != "Please preserve the original native request" {
		t.Fatal("HTTP body replaced the admitted author or original content", detail.Record, err)
	}
	postPath := "/api/factory/feed/posts/" + saved.SourcePost
	if body := request(http.MethodGet, postPath, "", 200); !strings.Contains(string(body), detail.Body) {
		t.Fatal("shared HTTP lost stored post detail", string(body))
	}
	replyBody := `{"repository":"9223372036854775807","body":"Clarify this same request","idempotency_key":"standalone-http-reply","parent_id":"` + saved.SourcePost + `"}`
	var reply factory.PostResponse
	if err := json.Unmarshal(request(http.MethodPost, "/api/factory/feed/posts", replyBody, 200), &reply); err != nil || reply.Source == nil || reply.Source.ID != saved.SourcePost {
		t.Fatal("shared HTTP lost canonical reply parent", reply, err)
	}
	if body := request(http.MethodGet, postPath+"/replies?repository=9223372036854775807", "", 200); !strings.Contains(string(body), reply.SourcePost) {
		t.Fatal("shared HTTP lost recorded reply", string(body))
	}
	const retryID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	retryBody := `{"command_id":"` + retryID + `","repository":"9223372036854775807","expected_revision":0,"body":"The revised native request"}`
	loseRetryReply.Store(true)
	lostRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+postPath+"/retry", strings.NewReader(retryBody))
	if err != nil {
		t.Fatal(err)
	}
	lostRequest.Header.Set("Authorization", "Bearer declared-feed-client")
	if response, err := server.Client().Do(lostRequest); err == nil {
		response.Body.Close()
		t.Fatal("lost-reply client unexpectedly received the receipt")
	}
	beforeCaptures := captures.Load()
	receiptPath := postPath + "/retry/" + retryID + "?repository=9223372036854775807"
	var receipt factory.FeedPostRetryReceipt
	if err := json.Unmarshal(request(http.MethodGet, receiptPath, "", 200), &receipt); err != nil || receipt.CommandID != retryID || receipt.Revision != 1 || receipt.Author != author || receipt.Repository != "9223372036854775807" {
		t.Fatal("HTTP lost-reply receipt not recoverable", receipt, err)
	}
	if replay := request(http.MethodPost, postPath+"/retry", retryBody, 200); strings.TrimSpace(string(replay)) != strings.TrimSpace(string(request(http.MethodGet, receiptPath, "", 200))) {
		t.Fatal("retry replay did not return the same committed receipt")
	}
	if captures.Load() != beforeCaptures {
		t.Fatal("receipt read/replay recaptured context")
	}
	request(http.MethodPost, postPath+"/retry", strings.Replace(retryBody, "The revised native request", "Changed command reuse", 1), 409)
	request(http.MethodPost, postPath+"/retry", `{"expected_revision":0}`, 400)
	request(http.MethodPost, postPath+"/retry", strings.Replace(retryBody, `"9223372036854775807"`, `9223372036854775807`, 1), 400)
	request(http.MethodGet, postPath+"/retry/"+factory.NewID()+"?repository=9223372036854775807", "", 404)
	request(http.MethodGet, postPath+"/retry/"+retryID+"?repository=7", "", 404)
	request(http.MethodGet, "/api/factory/feed/posts/"+reply.SourcePost+"/retry/"+retryID+"?repository=9223372036854775807", "", 404)
	postVisible.Store(false)
	request(http.MethodGet, receiptPath, "", 404)
	request(http.MethodPost, postPath+"/retry", retryBody, 404)
	postVisible.Store(true)
	request(http.MethodPost, postPath+"/retry", retryBody+` {}`, 400)
	request(http.MethodPost, "/api/factory/feed/posts", postBody+strings.Repeat(" ", 65536), 400)
	request(http.MethodGet, postPath+"/extra", "", 404)
	server.Close()
	shared.Actor = actor + 2
	server = httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	request(http.MethodPost, postPath+"/retry", retryBody, 403)
	request(http.MethodGet, receiptPath, "", 403)
	server.Close()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	shared.Actor = actor
	shared.Service.Store, shared.Posts.Store, shared.Commands.Posts.Store = db, db, db
	server = httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	if body := request(http.MethodGet, postPath, "", 200); !strings.Contains(string(body), "The revised native request") {
		t.Fatal("fresh HTTP handler lost retained revision", string(body))
	}
	if restored := request(http.MethodGet, receiptPath, "", 200); strings.TrimSpace(string(restored)) != strings.TrimSpace(string(request(http.MethodPost, postPath+"/retry", retryBody, 200))) || captures.Load() != beforeCaptures {
		t.Fatal("reopened handler/store changed receipt or recaptured context")
	}
	// Advance the head independently. The retained receipt still names its own
	// transition, and neither read recovery nor replay may change newer writing.
	if _, err := db.EditFeedPost(ctx, saved.SourcePost, author, "Newer HTTP writing", 1, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if err := db.NotePostAttempt(ctx, saved.SourcePost); err != nil {
		t.Fatal(err)
	}
	request(http.MethodPost, postPath+"/retry", retryBody, 200)
	request(http.MethodGet, receiptPath, "", 200)
	head, err := db.FeedPost(ctx, saved.SourcePost)
	intake, intakeErr := db.PostIntake(ctx, saved.SourcePost)
	if err != nil || intakeErr != nil || head.Body != "Newer HTTP writing" || head.Record.Revision != 2 || intake.Attempts != 1 || captures.Load() != beforeCaptures {
		t.Fatal("historical HTTP recovery changed newer state", head.Record.Revision, intake.Attempts, err, intakeErr)
	}
	if err := json.Unmarshal(request(http.MethodPost, "/api/factory/feed/posts", postBody, 200), &replay); err != nil || replay.SourcePost != saved.SourcePost || replay.Revision != 2 {
		t.Fatal("reopen lost canonical submission receipt", replay, err)
	}
	var count int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM factory_feed_posts WHERE idempotency='standalone-http-post'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("HTTP replay forked original source", count, err)
	}
	admissionChecks.Store(0)
	revokeAt.Store(1)
	refused := strings.Replace(postBody, "standalone-http-post", "refused-http-post", 1)
	request(http.MethodPost, "/api/factory/feed/posts", refused, 401)
	if _, _, err := db.FeedPostSubmission(ctx, repository, author, "refused-http-post"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("refused host admission recorded a source", err)
	}
	if err := raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_runs)+(SELECT count(*) FROM factory_usage)+(SELECT count(*) FROM factory_commands WHERE type<>'feed-post-retry')`).Scan(&count); err != nil || count != 0 || judge.calls != 0 || drafter.calls != 0 || creator.creates != 0 || captures.Load() == 0 {
		t.Fatal("HTTP recorded work without capture or drove execution/model/effects", count, captures.Load(), err)
	}
}

// Every source call reads real PG; the optional hook declares authority loss or
// caller cancellation after I/O, rather than simulating persisted library data.
type libraryReadThrough struct {
	*store.Store
	after func()
	fail  error
	reads int
}

func (s *libraryReadThrough) LibrarySearch(ctx context.Context, repository int64, query, kind, cursor string, history, current bool) (factory.LibraryPage, error) {
	s.reads++
	page, err := s.Store.LibrarySearch(ctx, repository, query, kind, cursor, history, current)
	if s.after != nil {
		s.after()
	}
	if s.fail != nil {
		return page, s.fail
	}
	return page, err
}
func (s *libraryReadThrough) LibraryRead(ctx context.Context, repository int64, id string, history bool) (factory.LibraryRead, error) {
	s.reads++
	item, err := s.Store.LibraryRead(ctx, repository, id, history)
	if s.after != nil {
		s.after()
	}
	if s.fail != nil {
		return item, s.fail
	}
	return item, err
}

func seedLibraryObservation(t *testing.T, ctx context.Context, db *sql.DB, item factory.LibraryItem) {
	t.Helper()
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO factory_library_items(id,repository,revision,kind,removed,data) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, item.ID, item.Repository, item.Revision, item.Kind, item.Removed, string(raw)); err != nil {
		t.Fatal(err)
	}
	for revision := item.Revision; revision > 0; revision-- {
		prior := item
		prior.Revision = revision
		if revision < item.Revision {
			prior.Body = "Original recorded knowledge"
		}
		data, _ := json.Marshal(factory.LibraryRevision{Item: prior, Reason: "recorded evidence", Actor: "worker:declared-assignment"})
		if _, err := db.ExecContext(ctx, `INSERT INTO factory_library_revisions(item,repository,revision,data) VALUES($1,$2,$3,$4::jsonb)`, item.ID, item.Repository, revision, string(data)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSharedLibraryAdmissionAndNativeHTTP(t *testing.T) {
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
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	const repository collaboration.RepositoryID = 9223372036854775807
	const actor collaboration.ActorID = 9007199254740993
	idFor := func(index int) string {
		hex := strconv.FormatInt(int64(index), 16)
		return strings.Repeat("0", 32-len(hex)) + hex
	}
	for index := 1; index <= 56; index++ {
		item := factory.LibraryItem{ID: idFor(index), Repository: int64(repository), Revision: 1, Kind: "knowledge", Title: "Recorded knowledge", Body: "Recorded detail", Scope: "repository", State: "observed", OwnerRole: "builder", Assignment: "declared-assignment", SourcePost: factory.NewID(), UpdatedAt: 1700000000}
		if index == 1 {
			item.Revision = 2
			item.Body = "Private exact boundary evidence"
		}
		if index == 52 {
			item.State = "contested"
		}
		if index == 53 {
			item.State = "superseded"
		}
		if index == 54 {
			item.State = "retracted"
		}
		if index == 55 {
			item.Removed = true
		}
		if index == 56 {
			item.Kind = "decision"
			item.Decision = "Explicit recorded decision"
		}
		seedLibraryObservation(t, ctx, raw, item)
	}
	snapshot := func() string {
		var value string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_object('items',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM factory_library_items i),'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY item,revision) FROM factory_library_revisions r),'commands',(SELECT count(*) FROM factory_commands),'runs',(SELECT count(*) FROM factory_runs))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	allowed := true
	denied := errors.New("declared current visibility denial")
	source := &libraryReadThrough{Store: db}
	app := factoryservice.Library{Source: source, Visibility: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
		if admitted != actor || repo != repository {
			t.Fatal("library widened canonical scope", admitted, repo)
		}
		if !allowed {
			return denied
		}
		return ctx.Err()
	}}
	first, err := app.Search(ctx, actor, repository, factory.LibraryQuery{CurrentOnly: true})
	if err != nil || len(first.Items) != 50 || first.Complete || first.NextCursor != idFor(50) {
		t.Fatal("library lost bounded current page", first, err)
	}
	second, err := app.Search(ctx, actor, repository, factory.LibraryQuery{CurrentOnly: true, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 3 || !second.Complete || second.Items[1].State != "contested" {
		t.Fatal("current continuation omitted contested or admitted old evidence", second, err)
	}
	all, err := app.Search(ctx, actor, repository, factory.LibraryQuery{History: true, Cursor: first.NextCursor})
	if err != nil || len(all.Items) != 6 || !all.Items[4].Removed {
		t.Fatal("history lost removed observation", all, err)
	}
	decision, err := app.Search(ctx, actor, repository, factory.LibraryQuery{Kind: "decision"})
	if err != nil || len(decision.Items) != 1 || decision.Items[0].Kind != "decision" {
		t.Fatal("kind filter changed", decision, err)
	}
	search, err := app.Search(ctx, actor, repository, factory.LibraryQuery{Query: "boundary"})
	if err != nil || len(search.Items) != 1 || search.Items[0].ID != idFor(1) {
		t.Fatal("full text search changed", search, err)
	}
	read, err := app.Read(ctx, actor, repository, idFor(1), false)
	if err != nil || len(read.History) != 2 || read.History[0].Item.Revision != 2 || read.History[1].Item.Body != "Original recorded knowledge" {
		t.Fatal("revision history changed", read, err)
	}
	encoded, _ := json.Marshal(read)
	var wire struct {
		Item struct {
			Repository string `json:"repository"`
		}
		History []struct {
			Item struct {
				Repository string `json:"repository"`
			}
		}
	}
	if err := json.Unmarshal(encoded, &wire); err != nil || wire.Item.Repository != strconv.FormatInt(int64(repository), 10) || wire.History[1].Item.Repository != wire.Item.Repository {
		t.Fatal("library scope rounded on browser wire", err)
	}
	if _, err := app.Read(ctx, actor, repository, idFor(55), false); !errors.Is(err, factoryservice.ErrLibraryNotFound) {
		t.Fatal("removed item leaked without history", err)
	}
	if removed, err := app.Read(ctx, actor, repository, idFor(55), true); err != nil || !removed.Item.Removed {
		t.Fatal("removed history unavailable", removed, err)
	}
	for _, query := range []factory.LibraryQuery{{Kind: "unknown"}, {Query: strings.Repeat("x", 513)}, {Cursor: "bad-cursor"}} {
		previous := source.reads
		if _, err := app.Search(ctx, actor, repository, query); !errors.Is(err, factory.ErrLibraryQuery) || source.reads != previous {
			t.Fatal("invalid query reached source", err)
		}
	}
	allowed = false
	previous := source.reads
	if _, err := app.Search(ctx, actor, repository, factory.LibraryQuery{}); !errors.Is(err, denied) || source.reads != previous {
		t.Fatal("denied search reached source", err)
	}
	allowed = true
	source.after = func() { allowed = false }
	if page, err := app.Search(ctx, actor, repository, factory.LibraryQuery{}); !errors.Is(err, denied) || len(page.Items) != 0 {
		t.Fatal("late denial returned partial search", page, err)
	}
	allowed = true
	if value, err := app.Read(ctx, actor, repository, idFor(1), false); !errors.Is(err, denied) || value.Item.ID != "" {
		t.Fatal("late denial returned private item", value, err)
	}
	source.after = nil
	allowed = true
	cancelled, cancel := context.WithCancel(ctx)
	source.after = cancel
	if value, err := app.Read(cancelled, actor, repository, idFor(1), false); !errors.Is(err, context.Canceled) || value.Item.ID != "" {
		t.Fatal("cancelled read returned item", value, err)
	}
	source.after = nil
	source.fail = errors.New("declared unknown library read")
	if page, err := app.Search(ctx, actor, repository, factory.LibraryQuery{}); !errors.Is(err, source.fail) || len(page.Items) != 0 {
		t.Fatal("source failure returned partial page", page, err)
	}
	source.fail = nil
	if _, err := (factoryservice.Library{Source: source}).Search(ctx, actor, repository, factory.LibraryQuery{}); !errors.Is(err, factoryservice.ErrLibraryUnavailable) {
		t.Fatal("missing admission returned library", err)
	}
	if _, err := (factoryservice.Library{Visibility: app.Visibility}).Read(ctx, actor, repository, idFor(1), false); !errors.Is(err, factoryservice.ErrLibraryUnavailable) {
		t.Fatal("missing reader returned item", err)
	}
	// Retained JSON cannot override the canonical scope chosen by the SQL row.
	if _, err := raw.ExecContext(ctx, `UPDATE factory_library_items SET data=jsonb_set(data,'{repository}','7'::jsonb) WHERE id=$1`, idFor(1)); err != nil {
		t.Fatal(err)
	}
	if page, err := app.Search(ctx, actor, repository, factory.LibraryQuery{Query: "boundary"}); !errors.Is(err, factoryservice.ErrLibraryScope) || len(page.Items) != 0 {
		t.Fatal("foreign payload escaped search", page, err)
	}
	if _, err := raw.ExecContext(ctx, `UPDATE factory_library_items SET data=jsonb_set(data,'{repository}',to_jsonb($2::bigint)) WHERE id=$1`, idFor(1), int64(repository)); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `UPDATE factory_library_revisions SET data=jsonb_set(data,'{item,repository}','7'::jsonb) WHERE item=$1 AND revision=1`, idFor(1)); err != nil {
		t.Fatal(err)
	}
	if value, err := app.Read(ctx, actor, repository, idFor(1), false); !errors.Is(err, factoryservice.ErrLibraryScope) || value.Item.ID != "" {
		t.Fatal("foreign revision escaped read", value, err)
	}
	if _, err := raw.ExecContext(ctx, `UPDATE factory_library_revisions SET data=jsonb_set(data,'{item,repository}',to_jsonb($2::bigint)) WHERE item=$1 AND revision=1`, idFor(1), int64(repository)); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before {
		t.Fatal("library reads changed retained records or work")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	source.Store = db
	reopened, err := app.Read(ctx, actor, repository, idFor(1), false)
	again, _ := json.Marshal(reopened)
	if err != nil || string(encoded) != string(again) {
		t.Fatal("reopen changed library observation", err)
	}
	// A second host serves the reopened canonical records over real TCP HTTP.
	// Declared admission is independent of the native callback fixture below.
	var admitted atomic.Bool
	admitted.Store(true)
	shared := app
	shared.Visibility = func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
		if repo != repository {
			return factoryservice.ErrRepositoryHidden
		}
		return app.Visibility(ctx, selected, repo)
	}
	transport := factoryhttpapi.Records{Library: shared,
		Host: collaboration.HostID(factory.NewID()), Actor: actor,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if !admitted.Load() || r.Header.Get("Authorization") != "Bearer declared-record-client" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
	}
	request := recordHTTPReader(t, transport.Handler())
	base := "/api/factory/library?repository=9223372036854775807"
	itemPath := "/api/factory/library/item?repository=9223372036854775807&id="
	for _, check := range []struct {
		path string
		want any
	}{
		{base + "&current_only=true", first},
		{base + "&current_only=true&cursor=" + first.NextCursor, second},
		{base + "&history=true&cursor=" + first.NextCursor, all},
		{base + "&kind=decision", decision},
		{base + "&query=boundary", search},
		{itemPath + idFor(1), read},
	} {
		_, body := request(http.MethodGet, check.path, "", 200)
		expected, _ := json.Marshal(check.want)
		if strings.TrimSpace(string(body)) != string(expected) {
			t.Fatal("HTTP changed retained library projection", check.path, string(body))
		}
	}
	request(http.MethodGet, itemPath+idFor(55), "", 404)
	request(http.MethodGet, itemPath+idFor(55)+"&history=true", "", 200)
	request(http.MethodGet, itemPath+factory.NewID(), "", 404)
	request(http.MethodGet, "/api/factory/library?repository=7", "", 404)
	for _, path := range []string{
		"/api/factory/library?repository=07", base + "&kind=unknown",
		base + "&query=" + strings.Repeat("x", 513), base + "&cursor=bad-cursor",
	} {
		request(http.MethodGet, path, "", 400)
	}
	request(http.MethodHead, base, "", 405)
	request(http.MethodPost, base, "", 405)
	source.fail = errors.New("private declared library outage")
	if _, body := request(http.MethodGet, base, "", 503); strings.Contains(string(body), "private declared") {
		t.Fatal("HTTP exposed library source failure", string(body))
	}
	source.fail = nil
	allowed = false
	previous = source.reads
	request(http.MethodGet, base, "", 503)
	if source.reads != previous {
		t.Fatal("denied HTTP library reached source")
	}
	allowed = true
	source.after = func() { allowed = false }
	request(http.MethodGet, itemPath+idFor(1), "", 503)
	source.after = nil
	allowed = true
	admitted.Store(false)
	previous = source.reads
	request(http.MethodGet, base, "", 401)
	if source.reads != previous {
		t.Fatal("withdrawn host reached library source")
	}
	admitted.Store(true)
	source.after = func() { admitted.Store(false) }
	if _, body := request(http.MethodGet, itemPath+idFor(1), "", 401); len(body) != 0 {
		t.Fatal("late host withdrawal disclosed library facts", string(body))
	}
	source.after = nil
	admitted.Store(true)
	missing := transport
	missing.Current = nil
	requestMissing := recordHTTPReader(t, missing.Handler())
	requestMissing(http.MethodGet, base, "", 503)
	missing = transport
	missing.Library.Source = nil
	requestMissing = recordHTTPReader(t, missing.Handler())
	requestMissing(http.MethodGet, itemPath+idFor(1), "", 503)
	if snapshot() != before {
		t.Fatal("shared library HTTP changed retained observations or work")
	}
	// The native human route uses the same shared projection and current
	// canonical visibility port; a registered worker keeps its existing MCP wire.
	_, _, httpDSN, call, _ := q05MuxFull(t, "")
	httpDB, err := sql.Open("pgx", httpDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer httpDB.Close()
	item := factory.LibraryItem{ID: factory.NewID(), Repository: 7, Revision: 1, Kind: "knowledge", Title: "Native library", Body: "Exact evidence", Scope: "project", State: "observed", UpdatedAt: 1700000000}
	seedLibraryObservation(t, ctx, httpDB, item)
	for _, path := range []string{"/api/factory/library?repository=7&current_only=true", "/api/factory/library/item?repository=7&id=" + item.ID} {
		code, body := call("GET", path, "")
		if code != 200 || !strings.Contains(string(body), `"repository":"7"`) || strings.Contains(string(body), `"repository":7`) {
			t.Fatal("native library lost shared scope", code, string(body))
		}
	}
	if code, _ := call("GET", "/api/factory/library?repository=9", ""); code != 404 {
		t.Fatal("hidden native library disclosed", code)
	}
	if code, _ := call("GET", "/api/factory/library?repository=7&kind=unknown", ""); code != 400 {
		t.Fatal("invalid native query admitted", code)
	}
	if code, _ := call("GET", "/api/factory/library/item?repository=7&id="+factory.NewID(), ""); code != 404 {
		t.Fatal("missing native item misreported", code)
	}
}

type artifactReadThrough struct {
	*store.Store
	after func()
	alter func(*factory.FeedArtifact, *[]byte)
	last  []byte
	reads int
}

func (s *artifactReadThrough) FeedArtifact(ctx context.Context, repository int64, id string) (factory.FeedArtifact, []byte, error) {
	s.reads++
	a, data, err := s.Store.FeedArtifact(ctx, repository, id)
	if s.alter != nil {
		s.alter(&a, &data)
	}
	s.last = data
	if s.after != nil {
		s.after()
	}
	return a, data, err
}

func TestSharedArtifactReadsAndNativeDownload(t *testing.T) {
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
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	const repository collaboration.RepositoryID = 9223372036854775807
	const actor collaboration.ActorID = 9007199254740993
	id, partial := factory.NewID(), factory.NewID()
	content := []byte("hello world")
	digest := factory.MCPInputDigest(content)
	seed := func(target *sql.DB, repo int64, id string, complete bool) {
		t.Helper()
		if _, err := target.ExecContext(ctx, `INSERT INTO factory_feed_artifacts(id,repository,assignment,name,media_type,content,complete,digest) VALUES($1,$2,'declared-assignment','sample.txt','text/plain',$3,$4,$5)`, id, repo, content, complete, digest); err != nil {
			t.Fatal(err)
		}
	}
	seed(raw, int64(repository), id, true)
	seed(raw, int64(repository), partial, false)
	snapshot := func() string {
		var value string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_object('artifacts',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM factory_feed_artifacts a),'posts',(SELECT count(*) FROM factory_feed_posts),'commands',(SELECT count(*) FROM factory_commands),'runs',(SELECT count(*) FROM factory_runs))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	allowed := true
	denied := errors.New("declared current artifact denial")
	source := &artifactReadThrough{Store: db}
	reader := factoryservice.Artifacts{Source: source, Visibility: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
		if admitted != actor || repo != repository {
			t.Fatal("artifact widened canonical scope", admitted, repo)
		}
		if !allowed {
			return denied
		}
		return ctx.Err()
	}}
	read, err := reader.Read(ctx, actor, repository, id)
	if err != nil || string(read.Content) != string(content) || read.Artifact.Digest != digest || read.Artifact.Href != factory.ArtifactHref(int64(repository), id) {
		t.Fatal("artifact lost recorded scope/content", read.Artifact, err)
	}
	source.last[0] = 'X'
	if string(read.Content) != string(content) {
		t.Fatal("adapter buffer reuse changed verified content")
	}
	if _, err := reader.Read(ctx, actor, repository, partial); !errors.Is(err, factoryservice.ErrArtifactNotFound) {
		t.Fatal("incomplete artifact became readable", err)
	}
	if _, err := reader.Read(ctx, actor, repository, factory.NewID()); !errors.Is(err, factoryservice.ErrArtifactNotFound) {
		t.Fatal("unknown artifact misclassified", err)
	}
	allowed = false
	previous := source.reads
	if _, err := reader.Read(ctx, actor, repository, id); !errors.Is(err, denied) || source.reads != previous {
		t.Fatal("denied source read dispatched", err)
	}
	allowed = true
	source.after = func() { allowed = false }
	if result, err := reader.Read(ctx, actor, repository, id); !errors.Is(err, denied) || len(result.Content) != 0 || result.Artifact.ID != "" {
		t.Fatal("late revocation exposed bytes", result.Artifact, err)
	}
	allowed = true
	source.after = nil
	cancelled, cancel := context.WithCancel(ctx)
	source.after = cancel
	if result, err := reader.Read(cancelled, actor, repository, id); !errors.Is(err, context.Canceled) || len(result.Content) != 0 {
		t.Fatal("cancelled artifact exposed bytes", err)
	}
	source.after = nil
	for _, alter := range []func(*factory.FeedArtifact, *[]byte){
		func(a *factory.FeedArtifact, _ *[]byte) { a.ID = factory.NewID() },
		func(a *factory.FeedArtifact, _ *[]byte) { a.Href = factory.ArtifactHref(7, id) },
		func(a *factory.FeedArtifact, _ *[]byte) { a.Size++ },
		func(a *factory.FeedArtifact, _ *[]byte) { a.Digest = strings.Repeat("a", 64) },
		func(a *factory.FeedArtifact, _ *[]byte) { a.Digest = "" },
		func(a *factory.FeedArtifact, _ *[]byte) { a.Name = "../private.txt" },
		func(a *factory.FeedArtifact, _ *[]byte) { a.MediaType = "text/html" },
		func(_ *factory.FeedArtifact, data *[]byte) { (*data)[0] = 'X' },
	} {
		source.alter = alter
		if value, err := reader.Read(ctx, actor, repository, id); !errors.Is(err, factoryservice.ErrArtifactInvalid) || len(value.Content) != 0 {
			t.Fatal("invalid retained artifact exposed bytes", err)
		}
	}
	source.alter = nil
	for _, unavailable := range []factoryservice.Artifacts{{Source: source}, {Visibility: reader.Visibility}} {
		if _, err := unavailable.Read(ctx, actor, repository, id); !errors.Is(err, factoryservice.ErrArtifactUnavailable) {
			t.Fatal("missing artifact port admitted", err)
		}
	}
	if snapshot() != before {
		t.Fatal("artifact reads mutated retained uploads or work")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	source.Store = db
	if result, err := reader.Read(ctx, actor, repository, id); err != nil || string(result.Content) != string(content) || result.Artifact != read.Artifact {
		t.Fatal("reopen changed retained artifact", result.Artifact, err)
	}
	var admitted atomic.Bool
	admitted.Store(true)
	shared := reader
	shared.Visibility = func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
		if repo != repository {
			return factoryservice.ErrRepositoryHidden
		}
		return reader.Visibility(ctx, selected, repo)
	}
	transport := factoryhttpapi.Records{Artifacts: shared,
		Host: collaboration.HostID(factory.NewID()), Actor: actor,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if !admitted.Load() || r.Header.Get("Authorization") != "Bearer declared-record-client" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
	}
	requestShared := recordHTTPReader(t, transport.Handler())
	path := "/api/factory/artifacts/" + id + "?repository=9223372036854775807"
	headers, body := requestShared(http.MethodGet, path, "", 200)
	if string(body) != string(content) || headers.Get("Content-Type") != "text/plain" || headers.Get("X-Content-Type-Options") != "nosniff" || headers.Get("Content-Security-Policy") != "sandbox; default-src 'none'" || headers.Get("Cache-Control") != "private, no-store" || !strings.Contains(headers.Get("Content-Disposition"), "attachment") {
		t.Fatal("shared download changed verified bytes or policy", headers, string(body))
	}
	headers, body = requestShared(http.MethodGet, path, "bytes=0-4", 206)
	if string(body) != "hello" || headers.Get("Content-Range") != "bytes 0-4/11" {
		t.Fatal("shared artifact range changed", headers, string(body))
	}
	requestShared(http.MethodGet, path, "bytes=30-40", 416)
	requestShared(http.MethodHead, path, "", 405)
	requestShared(http.MethodPost, path, "", 405)
	for _, artifact := range []string{partial, factory.NewID()} {
		requestShared(http.MethodGet, "/api/factory/artifacts/"+artifact+"?repository=9223372036854775807", "", 404)
	}
	requestShared(http.MethodGet, "/api/factory/artifacts/"+id+"?repository=7", "", 404)
	requestShared(http.MethodGet, "/api/factory/artifacts/"+id+"?repository=07", "", 400)
	requestShared(http.MethodGet, "/api/factory/artifacts/short?repository=9223372036854775807", "", 400)
	source.alter = func(a *factory.FeedArtifact, _ *[]byte) { a.Digest = strings.Repeat("a", 64) }
	requestShared(http.MethodGet, path, "", 503)
	source.alter = nil
	allowed = false
	previous = source.reads
	requestShared(http.MethodGet, path, "", 503)
	if source.reads != previous {
		t.Fatal("denied HTTP download reached artifact source")
	}
	allowed = true
	source.after = func() { allowed = false }
	requestShared(http.MethodGet, path, "", 503)
	source.after = nil
	allowed = true
	admitted.Store(false)
	previous = source.reads
	requestShared(http.MethodGet, path, "", 401)
	if source.reads != previous {
		t.Fatal("withdrawn host reached artifact source")
	}
	admitted.Store(true)
	source.after = func() { admitted.Store(false) }
	if headers, body := requestShared(http.MethodGet, path, "bytes=0-4", 401); len(body) != 0 || headers.Get("Content-Disposition") != "" {
		t.Fatal("late host withdrawal disclosed download bytes or metadata", headers, string(body))
	}
	source.after = nil
	admitted.Store(true)
	missing := transport
	missing.Current = nil
	requestMissing := recordHTTPReader(t, missing.Handler())
	requestMissing(http.MethodGet, path, "", 503)
	missing = transport
	missing.Artifacts.Source = nil
	requestMissing = recordHTTPReader(t, missing.Handler())
	requestMissing(http.MethodGet, path, "", 503)
	if snapshot() != before {
		t.Fatal("shared artifact HTTP changed uploads, posts, commands or runs")
	}
	// Actual native callback admission and HTTP transport retain download policy
	// and ranges around the same shared read. This is a declared native peer.
	product, _, httpDSN, _, _ := q05MuxFull(t, "")
	httpDB, err := sql.Open("pgx", httpDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer httpDB.Close()
	seed(httpDB, 7, id, true)
	seed(httpDB, 7, partial, false)
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)
	authority, _ := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "artifact-test", SessionGeneration: "generation-1", Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"}, Actor: extensions.Actor{ID: "1", Username: "alice"}})
	request := func(repo, artifact, rangeHeader string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/factory/artifacts/"+artifact+"?repository="+repo, nil)
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		if rangeHeader != "" {
			r.Header.Set("Range", rangeHeader)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	response := request("7", id, "")
	if response.Code != 200 || response.Body.String() != string(content) || response.Header().Get("Content-Type") != "text/plain" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || response.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(response.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("native download policy/content changed", response.Code, response.Header())
	}
	response = request("7", id, "bytes=0-4")
	if response.Code != 206 || response.Body.String() != "hello" || response.Header().Get("Content-Range") != "bytes 0-4/11" {
		t.Fatal("native range download changed", response.Code, response.Body.String())
	}
	for _, entry := range []struct{ repo, id string }{{"9", id}, {"7", partial}, {"7", factory.NewID()}} {
		if response := request(entry.repo, entry.id, ""); response.Code != 404 || strings.Contains(response.Body.String(), "hello world") {
			t.Fatal("unavailable native artifact disclosed", response.Code)
		}
	}
	if response := request("07", id, ""); response.Code != 400 {
		t.Fatal("noncanonical artifact scope admitted", response.Code)
	}
	if _, err := httpDB.ExecContext(ctx, `UPDATE factory_feed_artifacts SET digest=$2 WHERE id=$1`, id, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if response := request("7", id, ""); response.Code != 503 || strings.Contains(response.Body.String(), "hello world") {
		t.Fatal("native download exposed unverified bytes", response.Code)
	}
}

// The existing PostgreSQL record journeys use a real listener for the shared
// host transport. A declared channel admission is not native login evidence.
func recordHTTPReader(t *testing.T, handler http.Handler) func(string, string, string, int) (http.Header, []byte) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return func(method, path, rangeHeader string, expected int) (http.Header, []byte) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-record-client")
		if rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != expected || (expected == 200 || expected == 206) && !strings.Contains(res.Header.Get("Cache-Control"), "no-store") {
			t.Fatal("shared record HTTP", method, path, res.StatusCode, string(body), err)
		}
		if expected >= 400 && (strings.Contains(string(body), "Private exact boundary evidence") || strings.Contains(string(body), "hello world")) {
			t.Fatal("refused record HTTP disclosed private content", string(body))
		}
		return res.Header, body
	}
}

func TestParseFeedQueryParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/factory/feed?repositories=7,9,7&kinds=record&limit=10&cursor=v1:42", nil)
	query, err := factoryhttpapi.ParseFeedQuery(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(query.Repositories) != 2 || query.Repositories[0] != 7 || query.Repositories[1] != 9 {
		t.Fatalf("scope: %+v", query.Repositories)
	}
	if len(query.Kinds) != 1 || query.Kinds[0] != "record" || query.Limit != 10 || query.Cursor != "v1:42" {
		t.Fatalf("query: %+v", query)
	}
	r = httptest.NewRequest("GET", "/api/factory/feed?repositories=7", nil)
	query, err = factoryhttpapi.ParseFeedQuery(r)
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != 50 {
		t.Fatalf("default limit = %d, want 50", query.Limit)
	}
	for _, target := range []string{
		"/api/factory/feed",
		"/api/factory/feed?repositories=",
		"/api/factory/feed?repositories=0",
		"/api/factory/feed?repositories=abc",
		"/api/factory/feed?repositories=7&limit=0",
		"/api/factory/feed?repositories=7&limit=9999",
		"/api/factory/feed?repositories=7&limit=abc",
		"/api/factory/feed?repositories=7&cursor=v9:1",
		"/api/factory/feed?repositories=7&kinds=digest",
	} {
		r = httptest.NewRequest("GET", target, nil)
		if _, err := factoryhttpapi.ParseFeedQuery(r); err == nil {
			t.Fatalf("params admitted: %s", target)
		}
	}
}

func TestEnrichFeedRecordsFillsRecordContent(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	id := strings.Repeat("1", 32)
	assignment := factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: id,
		ProjectID: "p" + strings.Repeat("1", 24), Role: "coder", Phase: "correct",
		Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24),
		Run: "run-1", RunHistory: []string{"run-1"}, Stage: factory.AssignmentFinished,
		Outcome: factory.Succeeded, Repository: 7, Issue: 3,
		CreatedUnix: 1700000000, FinishedUnix: 1700000600,
		Result: &factory.AssignmentResult{
			AssignmentID: id, RunID: strings.Repeat("a", 32), Phase: "correct",
			Status: "completed", Summary: "addressed both findings", RecordedUnix: 1700000600,
		},
	}
	rawJSON, err := json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data)
		VALUES($1,7,3,'run-1','finished',1,$2)`, id, string(rawJSON)); err != nil {
		t.Fatal(err)
	}
	entries := []factory.FeedEntry{
		{Kind: factory.FeedKindRecord, ID: id, Repository: 7, Seq: 2},
		{Kind: factory.FeedKindPost, ID: strings.Repeat("2", 32), Repository: 7, Seq: 1},
	}
	factoryservice.EnrichRecords(ctx, st, map[int64]string{7: "owner/repo"}, entries)
	record := entries[0]
	if record.Actor != "fixer" || record.Author != "Fixer" || record.Project != "owner/repo" ||
		record.At != 1700000600 || record.Status != "correct:finished:succeeded" ||
		record.Issue != 3 || record.Summary != "addressed both findings" {
		t.Fatalf("record entry: %+v", record)
	}
	// Comments pass through untouched: their enrichment stays with the
	// comment owner.
	if entries[1].Actor != "" || entries[1].Project != "" || entries[1].Summary != "" {
		t.Fatalf("post entry touched: %+v", entries[1])
	}
	// A nil store still fills the project from the visibility recheck
	// and leaves every other field empty instead of guessing.
	bare := []factory.FeedEntry{{Kind: factory.FeedKindRecord, ID: id, Repository: 7, Seq: 2}}
	factoryservice.EnrichRecords(ctx, nil, map[int64]string{7: "owner/repo"}, bare)
	if bare[0].Project != "owner/repo" || bare[0].Actor != "" || bare[0].Status != "" {
		t.Fatalf("bare entry: %+v", bare[0])
	}
	t.Run("headless_application_service", func(t *testing.T) { checkFeedApplication(t, st, id) })
}

// Direct application calls use the real PostgreSQL ledger and a supplied
// authority port. No HTTP request, native header, session or API object exists.
func checkFeedApplication(t *testing.T, st *store.Store, recordID string) {
	ctx := t.Context()
	actor := collaboration.ActorID(9007199254740993)
	subscriber := strconv.FormatInt(int64(actor), 10)
	feed := factoryservice.Feed{Store: st}
	query := factory.FeedQuery{Repositories: []int64{7}, Limit: 1}
	checks := 0
	visible := func(ctx context.Context, repository int64) (string, error) {
		checks++
		if repository != 7 {
			t.Error("application widened its canonical repository scope", repository)
		}
		return "owner/repo", ctx.Err()
	}
	initial, err := feed.Read(ctx, actor, query, false, visible)
	if err != nil || len(initial.Entries) != 0 || checks != 0 {
		t.Fatal("missing subscription became activity or authority", initial, checks, err)
	}
	postID, _, err := st.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Newer saved post", IdempotencyKey: "application-feed-post"}, subscriber, "Canonical reader", 1700000700)
	if err != nil {
		t.Fatal(err)
	}
	preferences := factory.FeedSubscription{Subscriber: "1", Revision: 99, Repositories: []int64{7}, Kinds: []string{factory.FeedKindRecord}}
	if err := feed.SaveSubscription(ctx, actor, 0, preferences); err != nil {
		t.Fatal(err)
	}
	stored, revision, err := feed.Subscription(ctx, actor)
	if err != nil || revision != 1 || stored.Revision != 1 || stored.Subscriber != subscriber {
		t.Fatal("body subscriber/revision replaced canonical admission", stored, revision, err)
	}
	if native, rev, err := feed.Subscription(ctx, 1); err != nil || rev != 0 || len(native.Repositories) != 0 {
		t.Fatal("canonical preferences acquired a native account", native, rev, err)
	}
	if err := feed.SaveSubscription(ctx, actor, 0, preferences); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("stale subscription overwrote preferences", err)
	}
	records, err := feed.Read(ctx, actor, query, false, visible)
	if err != nil || len(records.Entries) != 1 || records.Entries[0].ID != recordID || records.Entries[0].Summary != "addressed both findings" || records.Entries[0].Project != "owner/repo" {
		t.Fatal("application lost subscription kinds or record projection", records, err)
	}
	conflicting := query
	conflicting.Kinds = []string{factory.FeedKindPost}
	before := checks
	if page, err := feed.Read(ctx, actor, conflicting, false, visible); err != nil || len(page.Entries) != 0 || checks != before {
		t.Fatal("conflicting kind filter widened the saved subscription", page, err)
	}
	preferences.Kinds = nil
	if err := feed.SaveSubscription(ctx, actor, 1, preferences); err != nil {
		t.Fatal(err)
	}
	denied := errors.New("declared revoked repository authority")
	hidden, err := feed.Read(ctx, actor, query, false, func(context.Context, int64) (string, error) { return "", denied })
	if err != nil || len(hidden.Entries) != 0 || hidden.Next == "" || !hidden.Truncated {
		t.Fatal("revoked row lost continuation or escaped visibility", hidden, err)
	}
	resume := query
	resume.Cursor = hidden.Next
	continued, err := feed.Read(ctx, actor, resume, false, visible)
	if err != nil || len(continued.Entries) != 1 || continued.Entries[0].ID != recordID || continued.Next == hidden.Next {
		t.Fatal("filtered row hid the next authorized page", continued, err)
	}
	perEntry := query
	perEntry.Limit = 2
	checks = 0
	page, err := feed.Read(ctx, actor, perEntry, false, func(context.Context, int64) (string, error) {
		checks++
		if checks == 1 {
			return "owner/repo", nil
		}
		return "", denied
	})
	if err != nil || checks != 2 || len(page.Entries) != 1 || page.Entries[0].ID != postID || page.Entries[0].Actor != subscriber || page.Entries[0].Summary != "Newer saved post" {
		t.Fatal("application cached authority across rows or lost post projection", page, checks, err)
	}
	preferences.Repositories = []int64{9}
	if err := feed.SaveSubscription(ctx, actor, 2, preferences); err != nil {
		t.Fatal(err)
	}
	if personal, err := feed.Read(ctx, actor, query, false, visible); err != nil || len(personal.Entries) != 0 {
		t.Fatal("personal subscription was widened", personal, err)
	}
	if project, err := feed.Read(ctx, actor, query, true, visible); err != nil || len(project.Entries) != 1 || project.Entries[0].ID != postID {
		t.Fatal("visible project scope borrowed personal preferences", project, err)
	}
	if _, err := feed.Read(ctx, actor, query, true, func(context.Context, int64) (string, error) { return "", denied }); !errors.Is(err, factoryservice.ErrFeedRepositoryUnavailable) {
		t.Fatal("project selection granted revoked access", err)
	}
	invalidProject := query
	invalidProject.Repositories = []int64{7, 9}
	if _, err := feed.Read(ctx, actor, invalidProject, true, visible); !errors.Is(err, factoryservice.ErrFeedProjectScope) {
		t.Fatal("project scope admitted multiple repositories", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := feed.Read(cancelled, actor, query, false, visible); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled application read became successful activity", err)
	}
}
