package factory

import (
	"strings"
	"testing"
)

func TestControlActionsValidate(t *testing.T) {
	for _, action := range []string{ActionPause, ActionResume} {
		if !ValidRepositoryAction(action) || ValidRunAction(action) {
			t.Fatalf("repository action %q misclassified", action)
		}
	}
	for _, action := range []string{ActionStop, ActionRetry, ActionTakeover} {
		if !ValidRunAction(action) || ValidRepositoryAction(action) {
			t.Fatalf("run action %q misclassified", action)
		}
	}
	if ValidRepositoryAction("takeover") || ValidRunAction("pause") || ValidRunAction("") {
		t.Fatal("unknown actions admitted")
	}
}

func TestLifecycleCommandsRideSettingsLedger(t *testing.T) {
	for _, typ := range []string{CommandPause, CommandResume, CommandRetry, CommandTakeover} {
		if !SettingsCommandType(typ) {
			t.Fatalf("lifecycle command %q not on the ledger", typ)
		}
		cmd := Command{ID: NewID(), Type: typ, Target: "repository/7/dispatch", Principal: "native:1", Payload: `{"cause":"control_paused"}`}
		cmd.Digest = SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
		if err := cmd.Validate(); err != nil {
			t.Fatalf("lifecycle command %q: %v", typ, err)
		}
	}
}

func pauseFixture() PauseReceipt {
	return PauseReceipt{
		CommandID:  NewID(),
		Paused:     true,
		Withdrawal: Withdrawal{Repository: 7, Revision: 1, Cause: CauseControlPaused, ClosedBy: "native:1", Captured: []string{}, ActiveCauses: []string{CauseControlPaused}},
		Runs:       []RunStopOutcome{},
	}
}

func TestPauseReceiptKeepsActionAndEffectSeparate(t *testing.T) {
	receipt := pauseFixture()
	receipt.Runs = []RunStopOutcome{
		{ID: NewID(), Outcome: string(Cancelled), Reason: "stopped", Confirmed: true},
		{ID: NewID(), Reason: "host stop unconfirmed", Uncertain: true},
	}
	if err := receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	uncertain := receipt
	uncertain.Runs[1].Confirmed = true
	if uncertain.Validate() == nil {
		t.Fatal("uncertain effect reported confirmed")
	}
	blank := receipt
	blank.Runs[0].Confirmed, blank.Runs[0].Uncertain = false, false
	if blank.Validate() == nil {
		t.Fatal("effect without a state admitted")
	}
	none := pauseFixture()
	none.Runs = nil
	if none.Validate() == nil {
		t.Fatal("pause without a run list admitted")
	}
}

func TestRetryDecisionStaysQueued(t *testing.T) {
	decision := RetryDecision{CommandID: NewID(), Prior: NewID(), Queued: true, Reason: "dispatch automation unavailable; retry recorded without launching"}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	launched := decision
	launched.Queued = false
	if launched.Validate() == nil {
		t.Fatal("retry without a queued decision admitted")
	}
}

func TestResumeReceiptValidates(t *testing.T) {
	receipt := ResumeReceipt{CommandID: NewID(), Revision: 2, Reopened: true, Effective: EffectiveAuthority{Missing: []string{}, DispatchOpen: true, Effective: true}}
	if err := receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	closed := receipt
	closed.Reopened = false
	closed.Effective.DispatchOpen = false
	closed.Effective.Effective = false
	if err := closed.Validate(); err != nil {
		t.Fatal("honest still-closed resume rejected", err)
	}
}

func TestTakeoverRecordDerivesMemberDestination(t *testing.T) {
	record := TakeoverRecord{Run: NewID(), Member: "alice", Project: "p765432109876543210987654", Dest: "/home/alice/factory-takeover/" + strings.Repeat("a", 32), CommandID: NewID(), Copied: "2026-10-01T00:00:00Z"}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	foreign := record
	foreign.Dest = "/home/bob/factory-takeover/" + strings.Repeat("a", 32)
	if foreign.Validate() == nil {
		t.Fatal("takeover into another member's home admitted")
	}
	escape := record
	escape.Dest = "/tmp/takeover"
	if escape.Validate() == nil {
		t.Fatal("takeover outside the member home admitted")
	}
}

func TestStartVerificationValidates(t *testing.T) {
	verification := StartVerification{Started: true, Revived: []string{}, Unverified: []string{}}
	if err := verification.Validate(); err != nil {
		t.Fatal(err)
	}
	unstarted := verification
	unstarted.Started = false
	if unstarted.Validate() == nil {
		t.Fatal("unstarted verification admitted")
	}
	bad := verification
	bad.Unverified = []string{"short"}
	if bad.Validate() == nil {
		t.Fatal("unverified non-identity admitted")
	}
}
