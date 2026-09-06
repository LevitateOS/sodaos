package host

import (
	"context"
	"errors"
	"testing"
)

type noExec struct{ called bool }

func (n *noExec) Run(context.Context, []byte, string, ...string) ([]byte, error) {
	n.called = true
	return nil, errors.New("unexpected command")
}
func TestInvalidProjectNeverExecutes(t *testing.T) {
	n := &noExec{}
	d := Daemon{Exec: n}
	if _, _, err := d.inspect(context.Background(), "../../other"); err == nil || n.called {
		t.Fatal("untrusted project reached executor")
	}
}
func TestInvalidAccountNeverExecutes(t *testing.T) {
	n := &noExec{}
	d := Daemon{Exec: n}
	if err := d.account(context.Background(), Account{Login: "root;id", Identity: 1, Keys: []string{"x"}}); err == nil || n.called {
		t.Fatal("untrusted account reached executor")
	}
}
