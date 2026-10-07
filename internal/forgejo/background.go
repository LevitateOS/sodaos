package forgejo

import (
	"context"
	"errors"
	"sync"

	extensions "forgejo.org/extension-sdk"
)

// ServiceBackground is the Soda backend's single native background
// transport: one lazily bootstrapped service admission shared by snapshot
// readers and conditional-operation publishers. The native host revokes
// the previous service admission on every bootstrap, so splitting
// readers and publishers across two bootstraps would invalidate each
// other; this transport serializes bootstrap and rebinds once after a
// revocation. All calls are idempotent-safe under a retried admission:
// submit replays by identity, and every other call mutates nothing new.
type ServiceBackground struct {
	socket        string
	installation  string
	hostUID       uint32
	mu            sync.Mutex
	admission     string
	pinned        string
	bootstrapCall *backgroundBootstrap
}

type backgroundBootstrap struct {
	done      chan struct{}
	force     bool
	admission string
	err       error
}

// NewServiceBackground builds the shared service transport over the
// operator-configured shared service callback socket and expected native
// host peer UID. Installation optionally pins the expected installation
// and must equal every bootstrap answer when set. No I/O happens before
// the first call.
func NewServiceBackground(socket string, hostUID uint32, installation string) *ServiceBackground {
	return &ServiceBackground{socket: socket, hostUID: hostUID, installation: installation}
}

// ReadNativeRevision observes the atomic native revision and idle/busy
// state through owning-installation admission.
func (b *ServiceBackground) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	var observation extensions.NativeRevisionObservation
	if err := b.call(ctx, extensions.BackgroundRevisionPath, struct{}{}, &observation); err != nil {
		return extensions.NativeRevisionObservation{}, err
	}
	if observation.Revision < 1 {
		return extensions.NativeRevisionObservation{}, errors.New("invalid background response")
	}
	return observation, nil
}

// ReadSnapshot reads one permission-checked native snapshot. The secret
// travels privately alongside the validated request; the host echoes the
// requested repository. The caller brackets the read between two equal
// idle ReadNativeRevision observations.
func (b *ServiceBackground) ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req extensions.SnapshotRequest) (extensions.NativeSnapshot, error) {
	if err := extensions.ValidateSnapshotRequest(req); err != nil {
		return extensions.NativeSnapshot{}, err
	}
	secret, err := backgroundSecret(credential)
	if err != nil {
		return extensions.NativeSnapshot{}, err
	}
	request := struct {
		extensions.SnapshotRequest
		Token string `json:"token"`
	}{SnapshotRequest: req, Token: secret}
	var snapshot extensions.NativeSnapshot
	if err := b.call(ctx, extensions.BackgroundSnapshotPath, request, &snapshot); err != nil {
		return extensions.NativeSnapshot{}, err
	}
	if snapshot.RepositoryID != req.RepositoryID {
		return extensions.NativeSnapshot{}, errors.New("invalid background response")
	}
	return snapshot, nil
}

// SubmitOperation submits one conditional operation intent under its
// persisted identity. The host derives installation identity from the
// authenticated transport; a payload installation must equal it.
func (b *ServiceBackground) SubmitOperation(ctx context.Context, credential extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error) {
	b.mu.Lock()
	pinned := b.pinned
	b.mu.Unlock()
	if intent.InstallationID != "" && pinned != "" && intent.InstallationID != pinned {
		return extensions.OperationRecord{}, errors.New("background installation mismatch")
	}
	secret, err := backgroundSecret(credential)
	if err != nil {
		return extensions.OperationRecord{}, err
	}
	request := struct {
		extensions.OperationIntent
		Token string `json:"token"`
	}{OperationIntent: intent, Token: secret}
	var record extensions.OperationRecord
	if err := b.call(ctx, extensions.BackgroundSubmitPath, request, &record); err != nil {
		return extensions.OperationRecord{}, err
	}
	if err := checkBackgroundRecord(b, intent.OperationID, record); err != nil {
		return extensions.OperationRecord{}, err
	}
	return record, nil
}

// GetOperation looks up one installation-owned operation. Absence reports
// not_observed; it never proves that an earlier request cannot arrive.
func (b *ServiceBackground) GetOperation(ctx context.Context, operationID string) (extensions.OperationLookup, error) {
	if !validBackgroundOperationID(operationID) {
		return extensions.OperationLookup{}, errors.New("invalid operation id")
	}
	var lookup extensions.OperationLookup
	if err := b.call(ctx, extensions.BackgroundGetPath, struct {
		OperationID string `json:"operation_id"`
	}{OperationID: operationID}, &lookup); err != nil {
		return extensions.OperationLookup{}, err
	}
	if err := checkBackgroundLookup(b, operationID, lookup); err != nil {
		return extensions.OperationLookup{}, err
	}
	return lookup, nil
}

// CancelOperation cancels one installation-owned operation, persisting
// cancellation even when submission has not arrived yet.
func (b *ServiceBackground) CancelOperation(ctx context.Context, operationID string) (extensions.OperationRecord, error) {
	if !validBackgroundOperationID(operationID) {
		return extensions.OperationRecord{}, errors.New("invalid operation id")
	}
	var record extensions.OperationRecord
	if err := b.call(ctx, extensions.BackgroundCancelPath, struct {
		OperationID string `json:"operation_id"`
	}{OperationID: operationID}, &record); err != nil {
		return extensions.OperationRecord{}, err
	}
	if err := checkBackgroundRecord(b, operationID, record); err != nil {
		return extensions.OperationRecord{}, err
	}
	return record, nil
}

// PublishPushEnv binds one registered operation to its push headers under
// the current admission. Rebind refreshes a revoked admission first; the
// caller retries a pre-launch push failure only after a lookup still
// shows it pending.
func (b *ServiceBackground) PublishPushEnv(ctx context.Context, operationID string, rebind bool) ([]string, error) {
	if !validBackgroundOperationID(operationID) {
		return nil, errors.New("invalid operation id")
	}
	b.mu.Lock()
	admission := b.admission
	b.mu.Unlock()
	if admission == "" || rebind {
		var err error
		admission, err = b.bootstrap(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return extensions.PublishPushEnv(operationID, admission)
}
