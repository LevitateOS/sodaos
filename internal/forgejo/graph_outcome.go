package forgejo

import (
	"bytes"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

// GraphOutcomeFromNative projects an already scoped native operation result.
// Validation of installation, operation identity and known actor/repository
// bindings stays with the transport and IssueGraphWriter. It is not admission.
func GraphOutcomeFromNative(record sdk.OperationRecord) factory.GraphOperationOutcome {
	return factory.GraphOperationOutcome{
		OperationID: record.OperationID,
		Effect:      collaboration.EffectState(record.EffectState), Cancellation: collaboration.CancellationState(record.CancellationStatus),
		Completion: collaboration.CompletionState(record.CompletionState), Reason: record.ReasonCode,
		Receipt: bytes.Clone(record.Receipt),
	}
}

func GraphLookupFromNative(lookup sdk.OperationLookup) factory.GraphOperationLookup {
	result := factory.GraphOperationLookup{State: collaboration.LookupUnknown}
	if lookup.Record != nil {
		record := GraphOutcomeFromNative(*lookup.Record)
		result.State, result.Record = collaboration.LookupFound, &record
	} else if lookup.Status == sdk.BackgroundOutcomeNotObserved {
		result.State = collaboration.LookupNotObserved
	} else if lookup.Status == sdk.BackgroundOutcomeOperationsUnavailable {
		result.State = collaboration.LookupUnavailable
	}
	return result
}
