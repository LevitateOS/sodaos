package host

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestStopPreparationPreservesConfirmedAndUncertainReceipts(t *testing.T) {
	in := project.PrepareStop{Project: "p0123456789abcdef01234567", ID: "f0123456789abcdef01234567"}
	for _, retirement := range []string{"confirmed", "uncertain"} {
		t.Run(retirement, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/prepare-stop" {
					t.Fatalf("unexpected route %s", r.URL.Path)
				}
				var request project.PrepareStop
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request != in {
					t.Fatalf("stop request identity = %+v, err=%v", request, err)
				}
				return jsonResponse(project.PrepareState{
					ID: in.ID, Project: in.Project, Phase: project.PrepareStopped,
					Stopped: true, Retirement: retirement,
				}), nil
			})}
			state, err := c.StopPreparation(context.Background(), in)
			if err != nil || state.ID != in.ID || state.Project != in.Project || !state.Stopped || state.Retirement != retirement {
				t.Fatalf("stop receipt was not preserved: state=%+v err=%v", state, err)
			}
		})
	}
}
