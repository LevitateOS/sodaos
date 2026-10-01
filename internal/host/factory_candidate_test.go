package host

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryInspectCandidateClientBindsObservation(t *testing.T) {
	in := project.FactoryCandidateInspect{Project: "p0123456789abcdef01234567", ID: strings.Repeat("a", 32)}
	for _, wrong := range []string{"", "run", "project", "container", "candidate", "missing", "stale"} {
		t.Run(wrong, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/factory-candidate-inspect" {
					t.Fatal("wrong candidate route")
				}
				out := project.FactoryCandidateState{ID: in.ID, Project: in.Project, Container: strings.Repeat("c", 64), Candidate: strings.Repeat("d", 40)}
				switch wrong {
				case "run":
					out.ID = strings.Repeat("b", 32)
				case "project":
					out.Project = "p111111111111111111111111"
				case "container":
					out.Container = "unknown"
				case "candidate":
					out.Candidate = "unknown"
				}
				response := jsonResponse(out)
				if wrong == "missing" {
					response.StatusCode = http.StatusNotFound
				} else if wrong == "stale" {
					response.StatusCode = http.StatusConflict
				}
				return response, nil
			})}
			_, err := c.FactoryInspectCandidate(context.Background(), in)
			if (err != nil) != (wrong != "") {
				t.Fatalf("observation error: %v", err)
			}
			if wrong == "missing" && !errors.Is(err, ErrRunNotFound) {
				t.Fatalf("missing run: %v", err)
			}
			if wrong == "stale" && !errors.Is(err, ErrRunStale) {
				t.Fatalf("stale run: %v", err)
			}
		})
	}
}

func TestPrepareCandidateClientBindsExactSource(t *testing.T) {
	prep := project.Preparation{
		ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Role: project.RoleReviewer,
		Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Approver: 7, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)},
		Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Approver: 9, EffectsDigest: strings.Repeat("b", 64)},
		SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64),
	}
	for _, wrong := range []bool{false, true} {
		c := NewClient("unused")
		c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/prepare-candidate" {
				t.Fatal("wrong preparation route")
			}
			out := project.PrepareState{ID: prep.ID, Project: prep.Project, Role: prep.Role, SourceCommit: prep.SourceCommit, SetupDigest: prep.SetupDigest, Container: strings.Repeat("e", 64), Phase: project.PrepareRunning}
			if wrong {
				out.SourceCommit = strings.Repeat("f", 40)
			}
			return jsonResponse(out), nil
		})}
		_, err := c.PrepareCandidate(context.Background(), project.FactoryCandidate{Preparation: prep, SourcePreparation: "f111111111111111111111111", Bundle: []byte("candidate")})
		if (err != nil) != wrong {
			t.Fatalf("source confirmation: %v", err)
		}
	}
}
