package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	extensions "forgejo.org/extension-sdk"

	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// maxBackgroundStatusBody bounds a captured dispatch status body: refusal
// codes are short lines, never bulk content.
const maxBackgroundStatusBody = 1024

// ServiceBackground is the Soda backend's single native background
// transport: one lazily bootstrapped service admission shared by snapshot
// readers and conditional-operation publishers. The native host revokes
// the previous service admission on every bootstrap, so splitting
// readers and publishers across two bootstraps would invalidate each
// other; this transport serializes bootstrap and rebinds once after a
// revocation. All calls are idempotent-safe under a retried admission:
// submit replays by identity, and every other call mutates nothing new.
type ServiceBackground struct {
	socket       string
	installation string
	hostUID      uint32
	mu           sync.Mutex
	admission    string
	pinned       string
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

// backgroundClientAdapter implements the SDK background client over the
// shared transport so snapshot readers keep their existing shape while
// sharing one admission with publishers.
type backgroundClientAdapter struct {
	background *ServiceBackground
}

func (a *backgroundClientAdapter) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	return a.background.ReadNativeRevision(ctx)
}

func (a *backgroundClientAdapter) ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req extensions.SnapshotRequest) (extensions.NativeSnapshot, error) {
	return a.background.ReadSnapshot(ctx, credential, req)
}

func (a *backgroundClientAdapter) SubmitOperation(ctx context.Context, credential extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error) {
	return a.background.SubmitOperation(ctx, credential, intent)
}

func (a *backgroundClientAdapter) GetOperation(ctx context.Context, operationID string) (extensions.OperationLookup, error) {
	return a.background.GetOperation(ctx, operationID)
}

func (a *backgroundClientAdapter) CancelOperation(ctx context.Context, operationID string) (extensions.OperationRecord, error) {
	return a.background.CancelOperation(ctx, operationID)
}

func checkBackgroundLookup(b *ServiceBackground, operationID string, lookup extensions.OperationLookup) error {
	b.mu.Lock()
	pinned := b.pinned
	b.mu.Unlock()
	if pinned == "" || lookup.InstallationID != pinned || lookup.OperationID != operationID || lookup.Status == "" {
		return errors.New("invalid background response")
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		if lookup.Record != nil {
			return errors.New("invalid background response")
		}
		return nil
	}
	if lookup.Record == nil {
		return errors.New("invalid background response")
	}
	return checkBackgroundRecord(b, operationID, *lookup.Record)
}

// call posts one background request under the current admission. A 401
// rebinds once and retries once: either the admission was revoked and
// the retry succeeds, or the retry's verdict stands.
func (b *ServiceBackground) call(ctx context.Context, path string, payload any, target any) error {
	admission, err := b.admissionForCall(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := b.post(ctx, path, data, admission, target); err != nil {
		var status *forgejopublish.StatusError
		if !errors.As(err, &status) || status.Status != http.StatusUnauthorized {
			return err
		}
		rebound, rebindErr := b.bootstrap(ctx, true)
		if rebindErr != nil {
			return rebindErr
		}
		if rebound == admission {
			return err
		}
		return b.post(ctx, path, data, rebound, target)
	}
	return nil
}

func (b *ServiceBackground) post(ctx context.Context, path string, data []byte, admission string, target any) error {
	b.mu.Lock()
	socket := b.socket
	b.mu.Unlock()
	transport := &http.Transport{DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socket)
		if err != nil {
			return nil, err
		}
		return b.verifiedPeer(conn)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://extensionHost"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(extensions.AdmissionHeader, admission)
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("background request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &forgejopublish.StatusError{Status: response.StatusCode, Body: readStatusBody(response.Body)}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBackgroundBodyBytes+1))
	if err != nil || len(body) > maxBackgroundBodyBytes {
		return errors.New("invalid background response")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid background response")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("invalid background response")
	}
	return nil
}

func readStatusBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxBackgroundStatusBody+1))
	if err != nil || len(data) > maxBackgroundStatusBody {
		return ""
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return string(bytes.TrimSpace(line))
}
