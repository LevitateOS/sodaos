package control

// D14 capability/submission admission (FW38).
//
// Every MCP request validates capability plus profile/tool/input
// binding, including tools absent from the caller's discovery set.
// Mutations repeat freshness/authority checks at admission.
// Caller-supplied role, child ID or environment variable proves
// nothing: the recorded capability row and the observed run state
// decide. Expiry, stop intent, withdrawal and terminal retirement
// revoke access.

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

var (
	// ErrMCPCapabilityInactive reports a call under a capability that
	// is not active: staged, revoked or retired.
	ErrMCPCapabilityInactive = errors.New("factory capability is not active")
	// ErrMCPCapabilityExpired reports a call past the recorded
	// deadline. Rate limits never extend deadlines.
	ErrMCPCapabilityExpired = errors.New("factory capability is expired")
	// ErrMCPCapabilityStopped reports a call racing current stop
	// intent: activation compares the observed binding, and stop
	// ordered before start permanently prevents admission.
	ErrMCPCapabilityStopped = errors.New("factory run is stopped")
	// ErrMCPCapabilityBinding reports a profile/tool/input binding
	// the recorded capability does not authorize, including tools
	// the caller's discovery set never advertised.
	ErrMCPCapabilityBinding = errors.New("capability does not authorize this call")
	// ErrMCPSubmissionFrozen reports a new submission identity after
	// the assignment recorded a terminal proposal: exact replay of
	// the terminal remains allowed, new identities refuse.
	ErrMCPSubmissionFrozen = errors.New("assignment already recorded a terminal proposal")
	// ErrMCPHandoffCap reports the 256-handoff bound for one
	// assignment: terminal receipt capacity stays reserved
	// regardless of task traffic.
	ErrMCPHandoffCap = errors.New("assignment handoff bound reached")
)

// MCPRunObservation is the host-observed run state read outside SQL:
// run identity, incarnation, assignment, authority, deadline and stop
// state. CAS activation checks this observation against the packet
// and the recorded capability inside the transaction.
type MCPRunObservation struct {
	RunID        string
	Incarnation  uint64
	AssignmentID string
	Authority    string
	Stopped      bool
}

// AdmitMCPCapability gates one tool call on the recorded capability
// and the observed run state. now is the admission instant; the
// caller supplies the observed run/incarnation from outside SQL so a
// restarted or replaced run cannot borrow the old authority.
func AdmitMCPCapability(capability factory.MCPCapability, tool factory.MCPTool, now time.Time, observed MCPRunObservation) error {
	if err := capability.Validate(); err != nil {
		return err
	}
	if capability.State != factory.MCPCapabilityActive {
		return ErrMCPCapabilityInactive
	}
	if capability.Expired(now) {
		return ErrMCPCapabilityExpired
	}
	if observed.Stopped {
		return ErrMCPCapabilityStopped
	}
	if observed.RunID != "" && observed.RunID != capability.RunID {
		return ErrMCPCapabilityBinding
	}
	if observed.AssignmentID != "" && observed.AssignmentID != capability.AssignmentID {
		return ErrMCPCapabilityBinding
	}
	if observed.Authority != "" && observed.Authority != capability.Authority {
		return ErrMCPCapabilityBinding
	}
	if !factory.MCPToolAdmitted(capability.Profile, tool) {
		return ErrMCPCapabilityBinding
	}
	return nil
}

// MCPSubmissionJournal is the assignment submission state admission
// reads: the recorded terminal proposal, if any, and the handoff
// count for cap enforcement. *store.Store implements it directly;
// tests stub it.
type MCPSubmissionJournal interface {
	// TerminalMCPProposal returns the recorded terminal receipt for
	// the assignment, or false when none is recorded yet.
	TerminalMCPProposal(ctx context.Context, assignmentID string) (factory.MCPReceipt, bool, error)
	// MCPHandoffCount returns the recorded handoff submissions for
	// the assignment.
	MCPHandoffCount(ctx context.Context, assignmentID string) (int, error)
}

// AdmitMCPSubmission gates one mutation after capability admission:
// submission shape, idempotency-key presence, terminal freeze and the
// handoff cap. Exact replay of a recorded submission stays allowed;
// conflicting bytes under a recorded ID conflict at persistence.
func AdmitMCPSubmission(ctx context.Context, journal MCPSubmissionJournal, capability factory.MCPCapability, submission factory.MCPSubmission) error {
	if err := submission.Validate(); err != nil {
		return err
	}
	if submission.AssignmentID != capability.AssignmentID {
		return ErrMCPCapabilityBinding
	}
	terminal, frozen, err := journal.TerminalMCPProposal(ctx, submission.AssignmentID)
	if err != nil {
		return err
	}
	if frozen {
		if submission.SubmissionID == terminal.SubmissionID {
			return nil
		}
		if factory.MCPIsTerminal(submission.Tool) {
			return ErrMCPSubmissionFrozen
		}
		// Nonterminal proposals after a terminal record nothing
		// new: the assignment is settled. Reads still serve the
		// recorded receipt through factory.receipt.
		return ErrMCPSubmissionFrozen
	}
	if submission.Tool == factory.MCPToolRequestHandoff {
		count, err := journal.MCPHandoffCount(ctx, submission.AssignmentID)
		if err != nil {
			return err
		}
		if count >= factory.MCPMaxHandoffs {
			return ErrMCPHandoffCap
		}
	}
	return nil
}
