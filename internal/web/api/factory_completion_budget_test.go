package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// scriptedBudgetReader serves a deterministic synthetic issue set
// through the REAL production snapshot adapter and bracket reads
// while counting every native call. It proves budgeted multi-pass
// resumption, not native/provider/semantic capture.
type scriptedBudgetReader struct {
	total int
	calls int
}

func (r *scriptedBudgetReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	r.calls++
	return extensions.NativeRevisionObservation{Revision: 41, Idle: true}, nil
}

func (r *scriptedBudgetReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	r.calls++
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRepositoryIssues {
		start := 0
		if req.Cursor != "" {
			v, err := strconv.Atoi(req.Cursor)
			if err != nil {
				return forgejo.NativeSnapshot{}, err
			}
			start = v
		}
		limit := req.Limit
		if limit < 1 || limit > 50 {
			return forgejo.NativeSnapshot{}, errors.New("invalid requested limit")
		}
		end := start + limit
		if end > r.total {
			end = r.total
		}
		items := []forgejo.RepositoryIssueItem{}
		for n := start + 1; n <= end; n++ {
			v := strconv.Itoa(n)
			items = append(items, forgejo.RepositoryIssueItem{IssueID: v + "0", IssueIndex: v, Visible: true})
		}
		page := &forgejo.RepositoryIssuesPage{Items: items, Complete: end == r.total}
		if !page.Complete {
			page.NextCursor = strconv.Itoa(end)
		}
		return forgejo.NativeSnapshot{Revision: 41, RepositoryID: req.RepositoryID, RepositoryIssues: page}, nil
	}
	snapshot := completionBody(req.IssueIndex, "title "+req.IssueIndex, "body "+req.IssueIndex)
	snapshot.Revision = 41
	snapshot.RepositoryID = req.RepositoryID
	return snapshot, nil
}

// A stable inventory larger than one pass worth of native calls
// persists its prefix cursor across DriveCompletions passes and
// eventually seals every exact body with per-pass actual native RPC
// at or under the readiness cap — never a subset seal, never a paid
// choice before complete.
func TestCompletionDriveServingCallsAndResume(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	event := factory.CompletionEvent{
		ID: factory.NewID(), PublicationID: factory.NewID(),
		MergedCommit: strings.Repeat("c", 40), ClosingOccurrence: "budget-resume-130",
		AcceptanceID: "d" + strings.Repeat("a", 24), Repository: 7, Issue: 3, Generation: 1,
	}
	rec := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: event.ID, CycleID: factory.NewID(),
			State: factory.ReassessmentCollecting, Repository: 7, Generation: 1,
		},
		Completion: event, Issue: 3,
	}
	if err = rec.Validate(); err != nil {
		t.Fatal(err)
	}
	eventRaw, _ := json.Marshal(event)
	recordRaw, _ := json.Marshal(rec)
	if _, err = raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,$3,$4)`,
		event.ID, event.PublicationID, event.ClosingOccurrence, string(eventRaw)); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,7,1,$2,'collecting',$3)`,
		rec.Cycle.CycleID, event.ID, string(recordRaw)); err != nil {
		t.Fatal(err)
	}
	reader := &scriptedBudgetReader{total: 130}
	source := &CompletionSnapshotSource{Reader: reader}
	coord := &control.Coordinator{Store: db}
	prior := 0
	for pass := 0; pass < 10; pass++ {
		before := reader.calls
		report := coord.DriveCompletions(ctx, control.CompletionDriveDeps{
			Enumerator: source,
			CaptureLocal: func(context.Context, int64) (factory.CompletionLocal, error) {
				return completionTestLocal(), nil
			},
		})
		used := reader.calls - before
		t.Logf("pass%d actual nativeRPC=%d", pass, used)
		if used > 156 {
			t.Fatalf("production completion drive used %d actual native RPC in one pass, cap 156", used)
		}
		stored, err := db.ReassessmentRecord(ctx, rec.Cycle.CycleID)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Sealed) > 0 {
			if len(stored.Evidence) != 130 || stored.EvidenceIssues != 130 {
				t.Fatalf("sealed incomplete set: bodies %d count %d", len(stored.Evidence), stored.EvidenceIssues)
			}
			for i, v := range stored.Evidence {
				if v.Index != int64(i+1) || v.Title != "title "+strconv.Itoa(i+1) || v.Body != "body "+strconv.Itoa(i+1) {
					t.Fatal("sealed body/index lost or changed")
				}
			}
			if _, err := control.DecisionStateFromRecord(stored); err != nil {
				t.Fatalf("resumed seal does not validate: %v", err)
			}
			if pass == 0 {
				t.Fatal("130 issues improperly sealed in a single bounded pass")
			}
			t.Logf("all 130 exact bodies persisted/sealed after %d passes, no chooser invoked", pass+1)
			return
		}
		if stored.Cycle.State != factory.ReassessmentCollecting || len(stored.CollectorProgress) == 0 {
			t.Fatalf("no resumable collecting prefix: %+v", report)
		}
		var progress control.InventoryCollector
		if err = json.Unmarshal(stored.CollectorProgress, &progress); err != nil {
			t.Fatal(err)
		}
		if len(progress.Issues) <= prior {
			t.Fatalf("prefix did not advance: prior %d current %d", prior, len(progress.Issues))
		}
		prior = len(progress.Issues)
	}
	t.Fatal("stable 130-issue inventory never finished within bounded passes")
}
