package publish

import (
	"context"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// fakeBackgroundOps doubles the native-operation boundary for the restored
// publication integration tests. Git execution itself is real; only the
// Fountain operation transport is faked.

type fakeBackgroundOps struct {
	revision    extensions.NativeRevisionObservation
	revErr      error
	submit      extensions.OperationRecord
	submitErr   error
	lookup      extensions.OperationLookup
	lookupErr   error
	cancel      extensions.OperationRecord
	cancelErr   error
	pushEnv     []string
	pushErr     error
	submits     int
	pushRebinds []bool
	intents     []extensions.OperationIntent
}

func (f *fakeBackgroundOps) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if f.revErr != nil {
		return extensions.NativeRevisionObservation{}, f.revErr
	}
	return f.revision, nil
}

func (f *fakeBackgroundOps) SubmitOperation(_ context.Context, _ extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error) {
	f.submits++
	f.intents = append(f.intents, intent)
	if f.submitErr != nil {
		return extensions.OperationRecord{}, f.submitErr
	}
	return f.submit, nil
}

func (f *fakeBackgroundOps) GetOperation(_ context.Context, _ string) (extensions.OperationLookup, error) {
	if f.lookupErr != nil {
		return extensions.OperationLookup{}, f.lookupErr
	}
	return f.lookup, nil
}

func (f *fakeBackgroundOps) CancelOperation(_ context.Context, _ string) (extensions.OperationRecord, error) {
	if f.cancelErr != nil {
		return extensions.OperationRecord{}, f.cancelErr
	}
	return f.cancel, nil
}

func (f *fakeBackgroundOps) PublishPushEnv(_ context.Context, _ string, rebind bool) ([]string, error) {
	f.pushRebinds = append(f.pushRebinds, rebind)
	if f.pushErr != nil {
		return nil, f.pushErr
	}
	return f.pushEnv, nil
}

func pendingRecord(opID string) extensions.OperationRecord {
	return extensions.OperationRecord{
		InstallationID: "install-1", OperationID: opID, Outcome: factory.OpEffectPending,
		ActorID: "5", RepositoryID: "7", Kind: factory.OpRefPublish,
		EffectState: factory.OpEffectPending, CancellationStatus: factory.OpCancelNone,
		CompletionState: factory.OpCompletionPending,
	}
}
