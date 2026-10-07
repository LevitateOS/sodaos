package control

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

func operatorRequest(t *testing.T, c *Coordinator, principal, body string) *httptest.ResponseRecorder {
	t.Helper()
	return operatorRequestReader(t, c, principal, io.NopCloser(strings.NewReader(body)))
}

func operatorRequestReader(t *testing.T, c *Coordinator, principal string, body io.ReadCloser) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, OperatorPath, nil)
	r.Body = body
	if principal != "" {
		r = r.WithContext(WithOperatorPrincipal(r.Context(), principal))
	}
	w := httptest.NewRecorder()
	c.OperatorHandler().ServeHTTP(w, r)
	return w
}

type errorAfterReader struct {
	data []byte
	err  error
}

func (r *errorAfterReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, r.err
}

func (*errorAfterReader) Close() error { return nil }

func TestOperatorBodyLimitReadsThroughEOF(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	base := `{"type":"status"}`
	exact := base + strings.Repeat(" ", operatorBodyLimit-len(base))
	w := operatorRequest(t, c, "os-uid:0", exact)
	if w.Code != http.StatusOK {
		t.Fatalf("exact %d-byte body rejected: status=%d body=%q", operatorBodyLimit, w.Code, w.Body.String())
	}

	over := exact + " "
	w = operatorRequest(t, c, "os-uid:0", over)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("%d-byte body admitted: status=%d", len(over), w.Code)
	}

	w = operatorRequestReader(t, c, "os-uid:0", &errorAfterReader{data: []byte(base), err: errors.New("late body read failed")})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("body read failure admitted: status=%d", w.Code)
	}
}

func TestOperatorRequiresPrincipalAndEnvelope(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	w := operatorRequest(t, c, "", `{"type":"status"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatal("principal-less command admitted")
	}
	r := httptest.NewRequest(http.MethodGet, OperatorPath, nil)
	r = r.WithContext(WithOperatorPrincipal(r.Context(), "os-uid:0"))
	w = httptest.NewRecorder()
	c.OperatorHandler().ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("non-POST command admitted")
	}
	for _, body := range []string{
		`{"type":"launch"}`,
		`{"type":"status","command_id":"` + factory.NewID() + `"}`,
		`{"type":"stop","command_id":"` + factory.NewID() + `"}`,
		`{"type":"reconcile","command_id":"` + factory.NewID() + `","target":"` + factory.NewID() + `"}`,
		`{"type":"stop","command_id":"short","target":"` + factory.NewID() + `"}`,
	} {
		if w = operatorRequest(t, c, "os-uid:0", body); w.Code != http.StatusBadRequest {
			t.Fatalf("invalid envelope accepted: %s", body)
		}
	}
}

func TestOperatorUsesEncodingJSONFieldCaseAndDocumentOrder(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	for _, body := range []string{
		`{"Type":"status"}`,
		`{"type":"launch","Type":"status"}`,
	} {
		if w := operatorRequest(t, c, "os-uid:0", body); w.Code != http.StatusOK {
			t.Fatalf("typed field binding rejected %s: status=%d body=%q", body, w.Code, w.Body.String())
		}
	}
	if w := operatorRequest(t, c, "os-uid:0", `{"type":"status","commandId":"not-a-tag-alias"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("non-tag alias accepted: status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestOperatorStatusAndStopRoundTrip(t *testing.T) {
	host := &stubHost{}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	r := recordRun(t, c, nil)
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: r.ID, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stopped"}, nil
	}
	w := operatorRequest(t, c, "os-uid:0", `{"type":"status"}`)
	if w.Code != http.StatusOK {
		t.Fatal("status failed", w.Code)
	}
	var status struct {
		Runs []factory.Run `json:"runs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || len(status.Runs) != 1 {
		t.Fatal("status omitted its recorded run", status, err)
	}
	cmd := factory.NewID()
	w = operatorRequest(t, c, "os-uid:0", `{"type":"stop","command_id":"`+cmd+`","target":"`+r.ID+`"}`)
	if w.Code != http.StatusOK {
		t.Fatal("stop failed", w.Code)
	}
	var receipt StopReceipt
	if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil || !receipt.Confirmed {
		t.Fatal("stop unconfirmed", receipt, err)
	}
	stored, err := c.Store.FactoryCommand(context.Background(), cmd)
	if err != nil || stored.Principal != "os-uid:0" || stored.Finished == "" {
		t.Fatal("stop command unattributed", stored, err)
	}
	w = operatorRequest(t, c, "os-uid:0", `{"type":"status","target":"`+factory.NewID()+`"}`)
	if w.Code != http.StatusNotFound {
		t.Fatal("unknown run reported status")
	}
}

func TestOperatorReplaysAndConflicts(t *testing.T) {
	host := &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stopped"}, nil
	}}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	r := recordRun(t, c, nil)
	other := recordRun(t, c, nil)
	cmd := factory.NewID()
	first := operatorRequest(t, c, "os-uid:0", `{"type":"stop","command_id":"`+cmd+`","target":"`+r.ID+`"}`)
	again := operatorRequest(t, c, "os-uid:0", `{"type":"stop","command_id":"`+cmd+`","target":"`+r.ID+`"}`)
	if again.Code != http.StatusOK || again.Body.String() != first.Body.String() {
		t.Fatal("duplicate command did not replay its outcome")
	}
	conflict := operatorRequest(t, c, "os-uid:0", `{"type":"stop","command_id":"`+cmd+`","target":"`+other.ID+`"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatal("reused command identity accepted changed content")
	}
	running := factory.NewID()
	if _, _, err := c.Store.RecordFactoryCommand(context.Background(), factory.Command{ID: running, Type: factory.CommandStop, Target: other.ID, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandStop, other.ID)}, time.Now()); err != nil {
		t.Fatal(err)
	}
	pending := operatorRequest(t, c, "os-uid:0", `{"type":"stop","command_id":"`+running+`","target":"`+other.ID+`"}`)
	if pending.Code != http.StatusAccepted {
		t.Fatal("unfinished duplicate did not return 202")
	}
	if host.calls != 1 {
		t.Fatal("replayed commands reached the host")
	}
}
