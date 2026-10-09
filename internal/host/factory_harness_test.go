package host

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryHarnessRequestsSelectedFamily(t *testing.T) {
	c := NewClient("unused")
	c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/factory-harness" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var selection map[string]string
		if err := json.Unmarshal(body, &selection); err != nil {
			t.Fatal(err)
		}
		if len(selection) != 1 || selection["harness"] != project.FactoryHarnessMuse {
			t.Fatalf("request selection = %v", selection)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"harness":"muse","version":"1.4.0","sha256":"` + strings.Repeat("a", 64) + `","image":"sha256:` + strings.Repeat("b", 64) + `"}`)),
			Header:     make(http.Header),
		}, nil
	})

	pin, err := c.FactoryHarness(context.Background(), project.FactoryHarnessMuse)
	if err != nil {
		t.Fatal(err)
	}
	if pin.Harness != project.FactoryHarnessMuse || pin.Version != "1.4.0" {
		t.Fatalf("pin = %+v", pin)
	}
}

func TestFactoryHarnessRejectsUnsupportedFamilyBeforeIO(t *testing.T) {
	c := NewClient("unused")
	c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unsupported family reached host")
		return nil, nil
	})
	if _, err := c.FactoryHarness(context.Background(), "unknown"); err == nil {
		t.Fatal("unsupported family accepted")
	}
}

func TestFactoryHarnessRejectsWrongFamilyResponse(t *testing.T) {
	c := NewClient("unused")
	c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"harness":"codex","version":"0.157.1","sha256":"` + strings.Repeat("a", 64) + `","image":"sha256:` + strings.Repeat("b", 64) + `"}`)),
			Header:     make(http.Header),
		}, nil
	})
	if _, err := c.FactoryHarness(context.Background(), project.FactoryHarnessMuse); err == nil {
		t.Fatal("wrong-family host response accepted")
	}
}
