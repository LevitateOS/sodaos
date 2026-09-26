package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func workRepository() Repository {
	return Repository{ID: 1, Name: "target", Owner: User{ID: 2, Login: "soda-tester"}}
}

func TestWorkActionsBindsNativeRunToCandidate(t *testing.T) {
	for _, commit := range []string{"candidate", "stale"} {
		t.Run(commit, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/repos/soda-tester/target/actions/runs" || r.URL.Query().Get("head_sha") != "candidate" || r.URL.Query().Get("workflow_id") != "verify.yaml" {
					t.Error("incorrect native filter")
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"workflow_runs": []ActionRun{{ID: 7, Commit: commit, Workflow: "verify.yaml", Status: "success"}}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			runs, err := New(server.URL).WorkActions(context.Background(), "test", workRepository(), "candidate", "verify.yaml", 1)
			if commit == "stale" {
				if err == nil {
					t.Fatal("stale CI accepted")
				}
				return
			}
			if err != nil || len(runs) != 1 || runs[0].ID != 7 {
				t.Fatalf("native run: %v", err)
			}
		})
	}
}

func TestReviewSubmissionHasExactCommitAndNoMergeOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/repos/soda-tester/target/pulls/3/reviews" {
			t.Error("unexpected authority")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["commit_id"] != "candidate" || body["event"] != "REQUEST_CHANGES" {
			t.Error("incorrect review binding")
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(WorkReview{ID: 1, Commit: "candidate", State: "REQUEST_CHANGES", User: User{ID: 3}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := New(server.URL)
	if _, err := client.SubmitWorkReview(context.Background(), "test", workRepository(), 3, "candidate", "REQUEST_CHANGES", "finding"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SubmitWorkReview(context.Background(), "test", workRepository(), 3, "candidate", "MERGE", ""); err == nil {
		t.Fatal("merge accepted")
	}
}
