package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// seedUnreadyProject retains one reservation the way an interrupted Create
// leaves it: stored, unready, carrying its creation profile.
func seedUnreadyProject(t *testing.T, s *Server, id string, repo int64) project.Profile {
	t.Helper()
	profile := testCreationProfile()
	p := store.Project{ID: id, Name: "repo", RepositoryID: repo, OwnerID: 1, Repository: "alice/repo", Profile: &profile}
	if err := s.Store.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return profile
}

func jsonNativeResponse(t *testing.T, status int, body string) *http.Response {
	t.Helper()
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

// TestCreateCancellationStillRecordsResult is the N-RUN1 transport
// regression: Fountain's 30-second ordinary bound and browser disconnect
// cancel the request, but the admitted provisioning must still record its
// result instead of stranding Ready=false.
func TestCreateCancellationStillRecordsResult(t *testing.T) {
	s := apiTestServer(t)
	profile := testCreationProfile()
	rawProfile, _ := json.Marshal(profile)
	r := apiTestRequest("POST", "/api/environments", `{"repository_id":"7"}`, "alice")
	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)
	created := ""
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/profile":
			return jsonNativeResponse(t, 200, string(rawProfile)), nil
		case "/create":
			var in project.Create
			if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
				t.Error(err)
				return jsonNativeResponse(t, 500, ""), nil
			}
			created = in.ID
			// The Fountain ordinary bound / browser disconnect lands while
			// native provisioning is still running.
			cancel()
			time.Sleep(200 * time.Millisecond)
			out, _ := json.Marshal(project.Environment{ID: in.ID, IP: "10.89.0.2", Running: true, Profile: in.Profile})
			return jsonNativeResponse(t, 200, string(out)), nil
		default:
			t.Errorf("unexpected native operation %s", req.URL.Path)
			return jsonNativeResponse(t, 500, ""), nil
		}
	})}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, r)
	if created == "" {
		t.Fatal("native creation never dispatched")
	}
	// The response cannot reach a disconnected caller, but the server must
	// have recorded the admitted result.
	p, err := s.Store.ProjectByRepository(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Ready || p.ID != created {
		t.Fatalf("cancellation stranded the reservation: ready=%v code=%d body=%s", p.Ready, w.Code, w.Body.String())
	}
	if w.Code != 201 {
		t.Fatalf("admitted creation misreported: code=%d body=%s", w.Code, w.Body.String())
	}
}

// inspectStub serves one recorded native observation for reconcile tests.
type inspectStub struct {
	env project.Environment
	err bool
}

func (s inspectStub) roundTrip(t *testing.T, profile project.Profile, lifecycle func(body map[string]any) (int, string)) roundTrip {
	return func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/inspect":
			if s.err {
				return nil, errNativeGone
			}
			out, _ := json.Marshal(s.env)
			return jsonNativeResponse(t, 200, string(out)), nil
		case "/lifecycle":
			var body map[string]any
			_ = json.NewDecoder(req.Body).Decode(&body)
			code, out := lifecycle(body)
			return jsonNativeResponse(t, code, out), nil
		default:
			t.Errorf("unexpected native operation %s", req.URL.Path)
			return jsonNativeResponse(t, 500, ""), nil
		}
	}
}

var errNativeGone = errTestNativeGone{}

type errTestNativeGone struct{}

func (errTestNativeGone) Error() string { return "missing native container" }

func lifecycleInspectBody(env project.Environment, boot bool) (int, string) {
	out, _ := json.Marshal(project.LifecycleState{Environment: env, BootEnabled: boot})
	return 200, string(out)
}

func decodeDetail(t *testing.T, w *httptest.ResponseRecorder) (provisioned, unavailable bool, observed *project.Environment) {
	t.Helper()
	var result struct {
		Environment struct {
			Provisioned bool `json:"provisioned"`
		} `json:"environment"`
		NativeUnavailable bool                 `json:"native_unavailable"`
		Observed          *project.Environment `json:"observed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.Environment.Provisioned, result.NativeUnavailable, result.Observed
}

// TestInterruptedCreationReconcilesFromHostEvidence is the N-RUN2
// durable-state regression: one retained reservation finishes from host
// evidence through ordinary reads, and stays truthfully unresolved when
// evidence does not confirm it. Nothing here deletes or recreates roots.
func TestInterruptedCreationReconcilesFromHostEvidence(t *testing.T) {
	t.Run("detailReadFinishesRunningReservation", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p111111111111111111111111"
		profile := seedUnreadyProject(t, s, id, 11)
		running := project.Environment{ID: id, IP: "10.89.0.11", Running: true, Profile: &profile}
		s.Host.HTTP = &http.Client{Transport: inspectStub{env: running}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			return lifecycleInspectBody(running, true)
		})}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/environments/"+id, "", "alice"))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if provisioned, unavailable, _ := decodeDetail(t, w); !provisioned || unavailable {
			t.Fatalf("running reservation not finished: provisioned=%v unavailable=%v body=%s", provisioned, unavailable, w.Body.String())
		}
		p, err := s.Store.Project(context.Background(), id)
		if err != nil || !p.Ready || p.IP != "10.89.0.11" {
			t.Fatalf("ready result not recorded: %+v %v", p, err)
		}
	})

	t.Run("LifecycleInspectSeesUnreadyReservation", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p222222222222222222222222"
		profile := seedUnreadyProject(t, s, id, 12)
		running := project.Environment{ID: id, IP: "10.89.0.12", Running: true, Profile: &profile}
		s.Host.HTTP = &http.Client{Transport: inspectStub{env: running}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			if body["action"] != "inspect" {
				return 500, ""
			}
			return lifecycleInspectBody(running, true)
		})}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/environments/"+id+"/lifecycle", "", "alice"))
		if w.Code != 200 {
			t.Fatalf("unready reservation not inspectable: %d %s", w.Code, w.Body.String())
		}
		var state project.LifecycleState
		if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil || !state.Environment.Running {
			t.Fatal("native evidence missing", w.Body.String())
		}
		if p, err := s.Store.Project(context.Background(), id); err != nil || !p.Ready {
			t.Fatalf("inspect did not finish the confirmed reservation: %+v %v", p, err)
		}
	})

	t.Run("RepeatCreateFinishesInsteadOfConflicting", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p333333333333333333333333"
		profile := seedUnreadyProject(t, s, id, 13)
		running := project.Environment{ID: id, IP: "10.89.0.13", Running: true, Profile: &profile}
		s.Host.HTTP = &http.Client{Transport: inspectStub{env: running}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			return lifecycleInspectBody(running, true)
		})}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/environments", `{"repository_id":"13"}`, "alice"))
		if w.Code != 200 {
			t.Fatalf("repeat create did not reconcile: %d %s", w.Code, w.Body.String())
		}
		var view struct {
			ID          string `json:"id"`
			Provisioned bool   `json:"provisioned"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil || view.ID != id || !view.Provisioned {
			t.Fatal("reconciled environment misreported", w.Body.String())
		}
		p, err := s.Store.ProjectByRepository(context.Background(), 13)
		if err != nil || !p.Ready || p.ID != id {
			t.Fatalf("one reservation not finished: %+v %v", p, err)
		}
	})

	t.Run("UnconfirmedReservationStaysTruthfullyUnresolved", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p444444444444444444444444"
		profile := seedUnreadyProject(t, s, id, 14)
		s.Host.HTTP = &http.Client{Transport: inspectStub{err: true}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			return 500, ""
		})}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/environments/"+id, "", "alice"))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if provisioned, unavailable, observed := decodeDetail(t, w); provisioned || !unavailable || observed != nil {
			t.Fatal("unconfirmed reservation misreported", w.Body.String())
		}
		retry := httptest.NewRecorder()
		nativeAPIServe(t, s, retry, apiTestRequest("POST", "/api/environments", `{"repository_id":"14"}`, "alice"))
		if retry.Code != 409 {
			t.Fatalf("unconfirmed retry misreported: %d %s", retry.Code, retry.Body.String())
		}
		var conflict struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
			Environment struct {
				ID          string `json:"id"`
				Provisioned bool   `json:"provisioned"`
			} `json:"environment"`
		}
		if err := json.Unmarshal(retry.Body.Bytes(), &conflict); err != nil {
			t.Fatal(err)
		}
		if conflict.Error.Code != "provisioning_incomplete" || conflict.Environment.ID != id || conflict.Environment.Provisioned {
			t.Fatal("unresolved state not truthful", retry.Body.String())
		}
		// No blind delete/recreate: the same single reservation is retained.
		p, err := s.Store.ProjectByRepository(context.Background(), 14)
		if err != nil || p.ID != id || p.Ready {
			t.Fatalf("reservation not retained: %+v %v", p, err)
		}
		inspect := httptest.NewRecorder()
		nativeAPIServe(t, s, inspect, apiTestRequest("GET", "/api/environments/"+id+"/lifecycle", "", "alice"))
		if inspect.Code != 502 {
			t.Fatalf("unconfirmed native lifecycle misreported: %d %s", inspect.Code, inspect.Body.String())
		}
	})

	t.Run("StartFinishesExpectedButStoppedRoot", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p555555555555555555555555"
		profile := seedUnreadyProject(t, s, id, 15)
		started := project.Environment{ID: id, IP: "10.89.0.15", Running: true, Profile: &profile}
		s.Host.HTTP = &http.Client{Transport: inspectStub{env: project.Environment{ID: id, Running: false, Profile: &profile}}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			if body["action"] != "start" {
				return 500, ""
			}
			return lifecycleInspectBody(started, true)
		})}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/environments/"+id+"/lifecycle", `{"action":"start"}`, "alice"))
		if w.Code != 200 {
			t.Fatalf("start did not finish the expected root: %d %s", w.Code, w.Body.String())
		}
		if p, err := s.Store.Project(context.Background(), id); err != nil || !p.Ready {
			t.Fatalf("started root not recorded ready: %+v %v", p, err)
		}
	})

	t.Run("StopAndJoinStayGatedWhileUnconfirmed", func(t *testing.T) {
		s := apiTestServer(t)
		id := "p666666666666666666666666"
		profile := seedUnreadyProject(t, s, id, 16)
		s.Host.HTTP = &http.Client{Transport: inspectStub{err: true}.roundTrip(t, profile, func(body map[string]any) (int, string) {
			return 500, ""
		})}
		stop := httptest.NewRecorder()
		nativeAPIServe(t, s, stop, apiTestRequest("POST", "/api/environments/"+id+"/lifecycle", `{"action":"stop","confirm_stop":true}`, "alice"))
		if stop.Code != 409 || !strings.Contains(stop.Body.String(), "not_provisioned") {
			t.Fatalf("stop on unconfirmed provisioning misreported: %d %s", stop.Code, stop.Body.String())
		}
		join := httptest.NewRecorder()
		nativeAPIServe(t, s, join, apiTestRequest("POST", "/api/environments/"+id+"/join", `{}`, "alice"))
		if join.Code != 409 {
			t.Fatalf("join on unconfirmed provisioning misreported: %d %s", join.Code, join.Body.String())
		}
	})
}
