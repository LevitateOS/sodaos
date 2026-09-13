package host

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const admissionBody = `{"id":"p0123456789abcdef01234567"}`
const admissionMutation = `{"project":"p0123456789abcdef01234567","action":"start"}`

type delayedAdmissionExec struct {
	entered, release chan struct{}
	calls            atomic.Int32
}

func (e *delayedAdmissionExec) Run(context.Context, []byte, string, ...string) ([]byte, error) {
	if e.calls.Add(1) == 1 {
		close(e.entered)
		<-e.release
	}
	return nil, errors.New("synthetic command failure")
}

type admissionContext struct {
	context.Context
	once    sync.Once
	entered chan struct{}
}

func (c *admissionContext) Deadline() (time.Time, bool) {
	c.once.Do(func() { close(c.entered) })
	return c.Context.Deadline()
}

type admissionBodyReader struct {
	io.Reader
	reads  atomic.Int32
	cancel context.CancelFunc
}

func (r *admissionBodyReader) Read(p []byte) (int, error) {
	r.reads.Add(1)
	if r.cancel != nil {
		r.cancel()
	}
	return r.Reader.Read(p)
}

func TestCancelledAdmissionLeavesWithoutDecodeOrExecution(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			executor := &noExec{}
			d := &Daemon{Exec: executor}
			if err := d.acquireAdmission(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer func() { <-d.admission }()
			ctx, cancel := context.WithCancel(t.Context())
			if mode == "deadline" {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
			}
			defer cancel()
			entered := &admissionContext{Context: ctx, entered: make(chan struct{})}
			body := &admissionBodyReader{Reader: strings.NewReader(admissionMutation)}
			response, done := httptest.NewRecorder(), make(chan struct{})
			go func() {
				defer close(done)
				d.ServeHTTP(response, httptest.NewRequest("POST", "/lifecycle", body).WithContext(entered))
			}()
			<-entered.entered
			if mode == "cancel" {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("cancelled waiter retained admission")
			}
			if response.Code != 408 || body.reads.Load() != 0 || executor.called {
				t.Fatal("cancelled mutation decoded/dispatched", response.Code)
			}
		})
	}
}

func TestMutationAdmissionRemainsSerialized(t *testing.T) {
	executor := &delayedAdmissionExec{entered: make(chan struct{}), release: make(chan struct{})}
	d := &Daemon{Exec: executor}
	var release sync.Once
	defer release.Do(func() { close(executor.release) })
	var workers sync.WaitGroup
	for range 12 {
		workers.Go(func() {
			d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/lifecycle", strings.NewReader(admissionMutation)))
		})
	}
	<-executor.entered
	time.Sleep(50 * time.Millisecond)
	if executor.calls.Load() != 1 {
		t.Error("mutations overlapped")
	}
	release.Do(func() { close(executor.release) })
	workers.Wait()
	if executor.calls.Load() != 12 {
		t.Fatal("queued mutation lost")
	}
}

func TestReadOnlyObservationsDoNotWaitForMutationAdmission(t *testing.T) {
	for _, path := range []string{"/inspect", "/profile", "/os", "/connection"} {
		t.Run(path, func(t *testing.T) {
			d := &Daemon{Exec: &noExec{}}
			if err := d.acquireAdmission(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer func() { <-d.admission }()
			body := admissionBody
			if path == "/profile" {
				body = `{}`
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", path, strings.NewReader(body)))
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("observation blocked on writer")
			}
		})
	}
}

func TestCancelledContextCannotWinAvailableAdmission(t *testing.T) {
	for _, path := range []string{"/inspect", "/lifecycle"} {
		executor := &noExec{}
		d := &Daemon{Exec: executor}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for range 100 {
			w := httptest.NewRecorder()
			d.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(admissionBody)).WithContext(ctx))
			if w.Code != 408 || executor.called {
				t.Fatal("cancelled request dispatched", w.Code)
			}
		}
	}
}

func TestCancellationDuringDecodeCannotDispatch(t *testing.T) {
	for _, path := range []string{"/inspect", "/lifecycle"} {
		executor := &noExec{}
		d := &Daemon{Exec: executor}
		ctx, cancel := context.WithCancel(t.Context())
		input := admissionBody
		if path == "/lifecycle" {
			input = admissionMutation
		}
		body := &admissionBodyReader{Reader: strings.NewReader(input), cancel: cancel}
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", path, body).WithContext(ctx))
		if w.Code != 500 || body.reads.Load() == 0 || executor.called {
			t.Fatal("cancelled decode dispatched", w.Code)
		}
		if err := d.acquireAdmission(t.Context()); err != nil {
			t.Fatal(err)
		}
		<-d.admission
	}
}

func TestInvalidRequestReleasesAdmission(t *testing.T) {
	for _, request := range []struct{ path, body string }{{"/inspect", `{`}, {"/inspect", `{"id":"bad"}`}, {"/lifecycle", `{`}, {"/unknown", `{}`}} {
		executor := &noExec{}
		d := &Daemon{Exec: executor}
		d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", request.path, strings.NewReader(request.body)))
		if executor.called {
			t.Fatal("invalid request executed")
		}
		if err := d.acquireAdmission(t.Context()); err != nil {
			t.Fatal(err)
		}
		<-d.admission
	}
}
