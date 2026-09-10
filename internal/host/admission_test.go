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

type delayedAdmissionExec struct {
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
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
			executor := &delayedAdmissionExec{entered: make(chan struct{}), release: make(chan struct{})}
			d := &Daemon{Exec: executor}
			firstDone, waiterDone := make(chan struct{}), make(chan struct{})
			var release sync.Once
			t.Cleanup(func() {
				release.Do(func() { close(executor.release) })
				for _, done := range []chan struct{}{firstDone, waiterDone} {
					select {
					case <-done:
					case <-time.After(2 * time.Second):
						t.Error("handler leaked")
					}
				}
			})
			go func() {
				defer close(firstDone)
				d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)))
			}()
			<-executor.entered
			ctx, cancel := context.WithCancel(t.Context())
			if mode == "deadline" {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
			}
			defer cancel()
			entered := &admissionContext{Context: ctx, entered: make(chan struct{})}
			body := &admissionBodyReader{Reader: strings.NewReader(admissionBody)}
			response := httptest.NewRecorder()
			go func() {
				defer close(waiterDone)
				d.ServeHTTP(response, httptest.NewRequest("POST", "/inspect", body).WithContext(entered))
			}()
			<-entered.entered
			if mode == "cancel" {
				cancel()
			}
			select {
			case <-waiterDone:
			case <-time.After(2 * time.Second):
				t.Fatal("cancelled waiter remained behind the active operation")
			}
			if response.Code != 408 || body.reads.Load() != 0 || executor.calls.Load() != 1 {
				t.Fatal("cancelled operation decoded or dispatched", response.Code, body.reads.Load(), executor.calls.Load())
			}
			release.Do(func() { close(executor.release) })
			<-firstDone
			// The cancelled waiter did not consume/leak admission for later work.
			d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)))
			if executor.calls.Load() != 2 {
				t.Fatal("admission not reusable")
			}
		})
	}
}

func TestAdmissionRemainsGloballySerialized(t *testing.T) {
	executor := &delayedAdmissionExec{entered: make(chan struct{}), release: make(chan struct{})}
	d := &Daemon{Exec: executor}
	var release sync.Once
	defer release.Do(func() { close(executor.release) })
	var workers sync.WaitGroup
	for range 12 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)))
		}()
	}
	<-executor.entered
	time.Sleep(50 * time.Millisecond)
	if executor.calls.Load() != 1 {
		t.Error("buffered native operations overlapped")
	}
	release.Do(func() { close(executor.release) })
	workers.Wait()
	if executor.calls.Load() != 12 {
		t.Fatal("queued work lost")
	}
}

func TestCancelledContextCannotWinAvailableAdmission(t *testing.T) {
	executor := &noExec{}
	d := &Daemon{Exec: executor}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// Both the gate and Done can be ready. Every selection must still refuse.
	for range 100 {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)).WithContext(ctx))
		if w.Code != 408 || executor.called {
			t.Fatal("cancelled request won admission", w.Code)
		}
	}
}

func TestCancellationDuringDecodeCannotDispatch(t *testing.T) {
	executor := &noExec{}
	d := &Daemon{Exec: executor}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	body := &admissionBodyReader{Reader: strings.NewReader(admissionBody), cancel: cancel}
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/inspect", body).WithContext(ctx))
	if w.Code != 500 || body.reads.Load() == 0 || executor.called {
		t.Fatal("cancelled decode reached executor", w.Code)
	}
	// Failure after admission also releases the gate.
	d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)))
	if !executor.called {
		t.Fatal("admission leaked after decode")
	}
}

func TestInvalidRequestReleasesAdmission(t *testing.T) {
	for _, request := range []struct{ path, body string }{
		{"/inspect", `{`}, {"/inspect", `{"id":"bad"}`}, {"/unknown", `{}`},
	} {
		executor := &noExec{}
		d := &Daemon{Exec: executor}
		d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", request.path, strings.NewReader(request.body)))
		if executor.called {
			t.Fatal("invalid request reached executor")
		}
		d.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/inspect", strings.NewReader(admissionBody)))
		if !executor.called {
			t.Fatal("invalid request leaked admission")
		}
	}
}
