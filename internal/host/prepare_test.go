package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestPrepareClientConfirmsIdentity(t *testing.T) {
	c := NewClient("unused")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		state := project.PrepareState{ID: "f99999999999999999999999", Project: "p0123456789abcdef01234567", Role: project.RoleCoder, Phase: project.PrepareRunning, Container: strings.Repeat("c", 64)}
		return jsonResponse(state), nil
	})}
	prep := project.Preparation{ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Role: project.RoleCoder}
	if _, err := c.Prepare(context.Background(), project.Prepare{Preparation: prep}); err == nil {
		t.Fatal("mismatched preparation identity accepted")
	}
}

func TestStopClientRequiresConfirmedStop(t *testing.T) {
	c := NewClient("unused")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		state := project.PrepareState{ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Phase: project.PrepareRunning}
		return jsonResponse(state), nil
	})}
	if _, err := c.StopPreparation(context.Background(), project.PrepareStop{Project: "p0123456789abcdef01234567", ID: "f0123456789abcdef01234567"}); err == nil {
		t.Fatal("unconfirmed stop accepted")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}
}
