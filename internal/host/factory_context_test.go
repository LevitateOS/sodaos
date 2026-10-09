package host

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/project"
)

func TestReadPreparationContextBindsCheckoutAndRejectsUnboundedContent(t *testing.T) {
	in := project.FactoryPreparationContextRequest{
		Project: "p0123456789abcdef01234567", ID: "f0123456789abcdef01234567",
		SourceCommit: strings.Repeat("a", 40), ApprovedBase: strings.Repeat("b", 40),
		DiffBase: strings.Repeat("c", 40), Candidate: strings.Repeat("d", 40),
		Paths:    []string{"README.md", "internal/factory/assignment.go"},
		NotAfter: time.Now().Add(2 * time.Minute).Truncate(time.Second).UTC(),
	}
	response := project.FactoryPreparationContext{
		Project: in.Project, ID: in.ID, Role: project.RoleCoder,
		SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase,
		DiffBase: in.DiffBase, Candidate: in.Candidate,
		Setup: []byte("approved setup commands"), Check: []byte("required checks: ci"),
		Files: []project.FactoryContextFile{
			{Path: in.Paths[0], Content: []byte("Repository instructions")},
			{Path: in.Paths[1], Content: []byte("func BuildDispatchPrompt()")},
		},
		Diff: []byte("candidate diff"),
	}
	for _, tc := range []struct {
		name   string
		mutate func(*project.FactoryPreparationContext)
		wantOK bool
	}{
		{name: "exact request bindings", wantOK: true},
		{name: "wrong approved base", mutate: func(out *project.FactoryPreparationContext) { out.ApprovedBase = strings.Repeat("e", 40) }},
		{name: "wrong role", mutate: func(out *project.FactoryPreparationContext) { out.Role = project.RoleReviewer }},
		{name: "per-file bound", mutate: func(out *project.FactoryPreparationContext) {
			out.Files[0].Content = []byte(strings.Repeat("x", project.MaxFactoryContextFileBytes+1))
		}},
		{name: "aggregate bound", mutate: func(out *project.FactoryPreparationContext) {
			out.Setup = []byte(strings.Repeat("s", 11*1024))
			out.Check = []byte(strings.Repeat("c", 11*1024))
			out.Diff = []byte(strings.Repeat("d", 11*1024))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.Path != "/prepare-context" {
					t.Fatalf("unexpected context request: %s %s", r.Method, r.URL.Path)
				}
				var got project.FactoryPreparationContextRequest
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil || !reflect.DeepEqual(got, in) {
					t.Fatalf("request=%+v err=%v", got, err)
				}
				out := response
				out.Files = append([]project.FactoryContextFile(nil), response.Files...)
				if tc.mutate != nil {
					tc.mutate(&out)
				}
				return jsonResponse(out), nil
			})}
			got, err := c.ReadPreparationContext(context.Background(), in, project.RoleCoder)
			if tc.wantOK {
				if err != nil || !reflect.DeepEqual(got, response) {
					t.Fatalf("context=%+v err=%v", got, err)
				}
			} else if err == nil {
				t.Fatalf("accepted mismatched or unbounded context: %+v", got)
			}
		})
	}
}
