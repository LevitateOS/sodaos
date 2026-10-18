package control

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

var updateWire = flag.Bool("update-wire", false, "regenerate closed-history wire packet goldens")

const (
	closedKeyA = "daaaaaaaaaaaaaaaaaaaaaaaa"
	closedKeyB = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	closedKeyC = "dcccccccccccccccccccccccc"
	closedKeyD = "ddddddddddddddddddddddddd"
)

// closedHistoryRevisions builds the three-revision closed-history flow: r1
// creates three tasks, r2 leaves A unchanged while B changes, C drops, and
// D arrives, and r3 recreates A under two delivered dependents. Same shape
// as the delivered-retention test so bytes and behavior cross-check.
func closedHistoryRevisions(r factory.IntentRevision) (first, changed, third factory.IntentRevision) {
	first = r
	first.ID = "d666666666666666666666666"
	first.Predecessor = r.ID
	first.Objective = "Build a request tracker"
	first.Tasks = []factory.IntentTask{
		{Key: closedKeyA, Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: closedKeyB, Objective: "Request list", Outcomes: []string{"Show requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{closedKeyA}},
		{Key: closedKeyC, Objective: "Request search", Outcomes: []string{"Find requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
	}
	changed = first
	changed.ID = "d777777777777777777777777"
	changed.Predecessor = first.ID
	changed.Objective = "Build a request tracker with exports"
	changed.Tasks = []factory.IntentTask{
		{Key: closedKeyA, Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: closedKeyB, Objective: "Request list with filters", Outcomes: []string{"Show requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{closedKeyA}},
		{Key: closedKeyD, Objective: "Request export", Outcomes: []string{"Export requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{closedKeyA}},
	}
	third = changed
	third.ID = "d888888888888888888888888"
	third.Predecessor = changed.ID
	third.Objective = "Build a request tracker with exports and audit"
	third.Tasks = []factory.IntentTask{
		{Key: closedKeyA, Objective: "Request entry with validation", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: closedKeyB, Objective: "Request list with filters", Outcomes: []string{"Show requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{closedKeyA}},
		{Key: closedKeyD, Objective: "Request export", Outcomes: []string{"Export requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{closedKeyA}},
	}
	return first, changed, third
}

func closeNativeIssues(f *generatedGraphFixture, bindings []factory.IntentTaskBinding, skip string) {
	for _, b := range bindings {
		if b.TaskKey == skip {
			continue
		}
		snap := f.issues[b.IssueIndex]
		snap.Issue.IsClosed = true
		f.issues[b.IssueIndex] = snap
	}
}

// TestGeneratedGraphFixtureRefusesClosedEditAtomically proves the fake keeps
// native atomicity: a packet mixing a valid create with a closed edit is
// refused whole, leaving no created issue, no revision bump, no record.
func TestGeneratedGraphFixtureRefusesClosedEditAtomically(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	if _, err := generateGraphMechanics(t, c, ctx, r.Intent, r.Owner); err != nil {
		t.Fatal(err)
	}
	bindings, err := c.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(bindings) == 0 {
		t.Fatalf("bindings=%v %v", bindings, err)
	}
	target := bindings[0]
	snap := f.issues[target.IssueIndex].Issue
	p := sdk.IssueGraphPayload{
		Creates: []sdk.IssueGraphCreate{{Key: "d000000000000000000000000", Title: "Probe", Body: "probe"}},
		Edits:   []sdk.IssueGraphEdit{{IssueID: target.IssueID, ExpectedTitle: snap.Title, ExpectedBody: snap.Content, ExpectedContentVersion: snap.ContentVersion, Title: snap.Title, Body: snap.Content}},
	}
	snap.IsClosed = true
	raw, _ := json.Marshal(p)
	hook := f.beforeSubmit
	f.beforeSubmit = nil
	defer func() { f.beforeSubmit = hook }()
	issuesBefore, revisionBefore, submissionsBefore := len(f.issues), f.revision, f.submissions
	_, err = f.SubmitGraph(ctx, factory.IntentGeneration{OperationID: "atomic-probe", ActorID: 91, Payload: raw}, r.Repository)
	if err == nil || !strings.Contains(err.Error(), "invalid operation intent") {
		t.Fatalf("closed edit not refused: %v", err)
	}
	if len(f.issues) != issuesBefore || f.revision != revisionBefore || f.submissions != submissionsBefore {
		t.Fatal("refused packet committed partial effects")
	}
	lookup, err := f.LookupGraph(ctx, factory.IntentGeneration{OperationID: "atomic-probe"}, r.Repository)
	if err != nil || lookup.Status != sdk.BackgroundOutcomeNotObserved {
		t.Fatalf("refused packet left a record: %+v %v", lookup, err)
	}
}

// TestIntentGraphDeltaRefusesTamperedClosedHistory is the manual-close
// distinction: a closed record whose content drifted from the pinned
// acceptance is history Soda cannot vouch for, so the delta refuses with
// objective_changed and submits nothing. Only pinned-stable closed
// records rebind, and rebinding preserves history without claiming
// verified merged work.
func TestIntentGraphDeltaRefusesTamperedClosedHistory(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	if _, err := generateGraphMechanics(t, c, ctx, r.Intent, r.Owner); err != nil {
		t.Fatal(err)
	}
	bindings, err := c.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(bindings) == 0 {
		t.Fatalf("bindings=%v %v", bindings, err)
	}
	snap := f.issues[bindings[0].IssueIndex]
	snap.Issue.Content += "\nManual note."
	snap.Issue.ContentDigest = forgejo.ContentDigest(snap.Issue.Content)
	snap.Issue.ContentVersion++
	snap.Issue.IsClosed = true
	f.issues[bindings[0].IssueIndex] = snap
	next := r
	next.ID = "d666666666666666666666666"
	next.Predecessor = r.ID
	if err := c.Store.SaveIntentRevision(ctx, next); err != nil {
		t.Fatal(err)
	}
	_, err = generateGraphMechanics(t, c, ctx, next.Intent, next.Owner)
	refusal := &AcceptanceRefusal{}
	if !errors.As(err, &refusal) || refusal.Reason != RefusalObjectiveChanged {
		t.Fatalf("tampered closed history did not refuse: %v", err)
	}
	if f.submissions != 1 {
		t.Fatalf("refused delta submitted a packet: %d", f.submissions)
	}
}

type wireNativeIssue struct {
	IssueID        int64   `json:"issue_id"`
	IssueIndex     int64   `json:"issue_index"`
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	ContentVersion int     `json:"content_version"`
	IsClosed       bool    `json:"is_closed"`
	DependsOn      []int64 `json:"depends_on"`
	TitleDigest    string  `json:"title_digest"`
	ContentDigest  string  `json:"content_digest"`
}

type wireCase struct {
	Name            string                            `json:"name"`
	Revision        string                            `json:"revision"`
	Predecessor     string                            `json:"predecessor,omitempty"`
	Operation       string                            `json:"operation"`
	Actor           int64                             `json:"actor_id"`
	Repository      int64                             `json:"repository"`
	SubmittedKeys   []string                          `json:"submitted_keys"`
	Delivered       map[string]factory.DeliveredIssue `json:"delivered,omitempty"`
	PacketFile      string                            `json:"packet_file,omitempty"`
	ReceiptFile     string                            `json:"receipt_file,omitempty"`
	NativeFile      string                            `json:"native_file,omitempty"`
	ExpectedRefusal string                            `json:"expected_refusal,omitempty"`
	SodaAssertions  []string                          `json:"soda_assertions"`
}

type wireManifest struct {
	Note            string     `json:"note"`
	FixtureIDsLocal bool       `json:"fixture_ids_local"`
	Cases           []wireCase `json:"cases"`
}

func dumpWireNative(f *generatedGraphFixture) []wireNativeIssue {
	var out []wireNativeIssue
	for index, snap := range f.issues {
		issue := snap.Issue
		id, _ := strconv.ParseInt(issue.ID, 10, 64)
		entry := wireNativeIssue{IssueID: id, IssueIndex: index, Title: issue.Title, Body: issue.Content, ContentVersion: issue.ContentVersion, IsClosed: issue.IsClosed, DependsOn: []int64{}, TitleDigest: issue.TitleDigest, ContentDigest: issue.ContentDigest}
		if snap.Dependencies != nil {
			for _, dep := range snap.Dependencies.Items {
				if depID, err := strconv.ParseInt(dep.DependencyID, 10, 64); err == nil {
					entry.DependsOn = append(entry.DependsOn, depID)
				}
			}
			sort.Slice(entry.DependsOn, func(i, j int) bool { return entry.DependsOn[i] < entry.DependsOn[j] })
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IssueIndex < out[j].IssueIndex })
	return out
}

func checkWireGolden(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if *updateWire {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s: run with -update-wire", name)
	}
	if string(want) != string(data) {
		at := 0
		for at < len(want) && at < len(data) && want[at] == data[at] {
			at++
		}
		t.Fatalf("golden %s differs at byte %d (want %d bytes, got %d)", name, at, len(want), len(data))
	}
}

func wirePacketKeys(p sdk.IssueGraphPayload) map[string]bool {
	keys := map[string]bool{}
	for _, entry := range p.Creates {
		keys[entry.Key] = true
	}
	return keys
}

// TestIntentGraphClosedHistoryWirePackets emits the exact production wire
// packets for the I07 native join: L replays these bytes against the real
// writer in its isolated companion and Q joins them against native
// receipts. Goldens live in testdata/closed-history; byte equality pins
// the wire shape while live assertions pin the join semantics. Fixture
// issue IDs are local to these goldens; the manifest tells L/Q how to
// translate them onto native allocations.
func TestIntentGraphClosedHistoryWirePackets(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	first, changed, third := closedHistoryRevisions(r)
	if err := c.Store.SaveIntentRevision(ctx, first); err != nil {
		t.Fatal(err)
	}
	var packets []sdk.IssueGraphPayload
	submit := f.beforeSubmit
	f.beforeSubmit = func(g factory.IntentGeneration) error {
		if err := submit(g); err != nil {
			return err
		}
		var p sdk.IssueGraphPayload
		if err := json.Unmarshal(g.Payload, &p); err != nil {
			return err
		}
		packets = append(packets, p)
		return nil
	}
	if _, err := generateGraphMechanics(t, c, ctx, first.Intent, first.Owner); err != nil {
		t.Fatal(err)
	}
	b1, err := c.Store.IntentTaskBindings(ctx, first.ID)
	if err != nil || len(b1) != 3 {
		t.Fatalf("bindings=%v %v", b1, err)
	}
	closeNativeIssues(f, b1, "")
	preR2 := dumpWireNative(f)
	if err := c.Store.SaveIntentRevision(ctx, changed); err != nil {
		t.Fatal(err)
	}
	if _, err := generateGraphMechanics(t, c, ctx, changed.Intent, changed.Owner); err != nil {
		t.Fatal(err)
	}
	b2, err := c.Store.IntentTaskBindings(ctx, changed.ID)
	if err != nil || len(b2) != 3 {
		t.Fatalf("bindings=%v %v", b2, err)
	}
	closeNativeIssues(f, b2, closedKeyA)
	preR3 := dumpWireNative(f)
	if err := c.Store.SaveIntentRevision(ctx, third); err != nil {
		t.Fatal(err)
	}
	if _, err := generateGraphMechanics(t, c, ctx, third.Intent, third.Owner); err != nil {
		t.Fatal(err)
	}
	if len(packets) != 3 {
		t.Fatalf("packets=%d", len(packets))
	}
	if len(packets[0].Creates) != 3 || len(packets[0].Edits) != 0 {
		t.Fatal("fresh packet shape changed")
	}
	if created := wirePacketKeys(packets[1]); len(packets[1].Edits) != 0 || len(created) != 2 || !created[closedKeyB] || !created[closedKeyD] {
		t.Fatal("delta packet shape changed")
	}
	if created := wirePacketKeys(packets[2]); len(packets[2].Edits) != 0 || len(created) != 1 || !created[closedKeyA] {
		t.Fatal("redelta packet shape changed")
	}
	dir := filepath.Join("testdata", "closed-history")
	packetFiles := []string{"r1-fresh.packet.json", "r2-delta.packet.json", "r3-redelta.packet.json"}
	for i, p := range packets {
		raw, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		checkWireGolden(t, dir, packetFiles[i], append(raw, '\n'))
	}
	revisions := []factory.IntentRevision{first, changed, third}
	natives := []struct {
		name string
		dump []wireNativeIssue
	}{{"pre-r2.native.json", preR2}, {"pre-r3.native.json", preR3}}
	for _, native := range natives {
		raw, err := json.MarshalIndent(native.dump, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		checkWireGolden(t, dir, native.name, append(raw, '\n'))
	}
	manifest := wireManifest{
		Note:            "Production Soda packets for the I07 native join. packet bytes are the durable submitted payloads; receipt files are the fixture receipts recorded by Soda (shape reference only, never native proof). Fixture issue IDs are local: L seeds its writer by applying r1-fresh, builds key->native-ID from its own receipt, and translates ID-bearing entries (dependency removals) in r2/r3 onto native allocations before applying. Operation IDs are deterministic intent-<revision>; resubmission must return the recorded result. Authority context per case: the submitting actor must own the repository; native writer validates every edit anchor before any create (atomic).",
		FixtureIDsLocal: true,
	}
	receiptFiles := []string{"r1-fresh.receipt.json", "r2-delta.receipt.json", "r3-redelta.receipt.json"}
	caseNames := []string{"r1-fresh", "r2-delta", "r3-redelta"}
	nativeFiles := []string{"", "pre-r2.native.json", "pre-r3.native.json"}
	assertions := [][]string{
		{"3 creates for submitted keys A/B/C", "1 dependency edge B->A", "receipt carries 3 keyed entries"},
		{"0 edits: unchanged closed A omitted, changed closed B recreated, removed closed C silent", "creates exactly keys B/D", "delivered map pins A only", "dependency removal drops old B->A edge; new B/D->A edges added", "receipt carries 2 keyed entries and no row for the delivered issue"},
		{"0 edits across the second delta", "creates exactly key A", "delivered map pins B/D", "dependent edges rewired onto the recreated A", "prior closed natives byte-identical"},
	}
	for i, rev := range revisions {
		stored, err := c.Store.IntentGeneration(ctx, rev.ID)
		if err != nil {
			t.Fatal(err)
		}
		checkWireGolden(t, dir, receiptFiles[i], stored.Receipt)
		keys := []string{}
		for _, task := range stored.SubmittedTasks {
			keys = append(keys, task.Key)
		}
		manifest.Cases = append(manifest.Cases, wireCase{Name: caseNames[i], Revision: rev.ID, Predecessor: rev.Predecessor, Operation: stored.OperationID, Actor: stored.ActorID, Repository: rev.Repository, SubmittedKeys: keys, Delivered: stored.Delivered, PacketFile: packetFiles[i], ReceiptFile: receiptFiles[i], NativeFile: nativeFiles[i], SodaAssertions: assertions[i]})
	}
	manifest.Cases = append(manifest.Cases, wireCase{Name: "tampered-closed", ExpectedRefusal: RefusalObjectiveChanged, SodaAssertions: []string{"closed record with drifted content refuses objective_changed", "refused delta submits zero packets", "only pinned-stable closed records rebind; rebinding preserves history without claiming verified merged work"}})
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWireGolden(t, dir, "manifest.json", append(raw, '\n'))
}
