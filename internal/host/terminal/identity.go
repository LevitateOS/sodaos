package terminal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type identityRequest struct {
	Action        string                `json:"action"`
	Delivery      identity.DeliveryWire `json:"delivery"`
	Login         string                `json:"login,omitempty"`
	Scope         string                `json:"scope,omitempty"`
	Cols          int                   `json:"cols,omitempty"`
	Rows          int                   `json:"rows,omitempty"`
	SourceHash    string                `json:"source_hash,omitempty"`
	Container     string                `json:"container,omitempty"`
	HarnessSHA256 string                `json:"harness_sha256,omitempty"`
}

func terminalLease(lease identity.Lease, preparing bool) bool {
	if !terminalReservation(lease) {
		return false
	}
	if preparing {
		return lease.Binding == nil && lease.Deadline.After(time.Now()) && !lease.Deadline.After(time.Now().Add(12*time.Hour))
	}
	return terminalBinding(lease)
}

func terminalReservation(lease identity.Lease) bool {
	return lease.Kind == identity.Terminal && terminalID.MatchString(lease.ExecutionID) && projectID.MatchString(lease.ProjectID) && lease.ActorID > 0 && lease.ID != "" && lease.Generation > 0
}

func terminalBinding(lease identity.Lease) bool {
	b := lease.Binding
	if b == nil {
		return false
	}
	return b.Kind == identity.Terminal && b.ID == lease.ExecutionID && containerID.MatchString(b.Project) && loginName.MatchString(b.Login) && b.Login != "root" && b.Generation == lease.Generation
}

func (s *Service) identityCall(ctx context.Context, container string, request identityRequest) (identity.Delivery, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return identity.Delivery{}, err
	}
	// The agent is fixed product code. Secrets travel only over stdin.
	agent := AgentExec(container, body, "broker")
	out, err := s.podman(ctx, body, agent.Args[1:]...)
	if err != nil {
		return identity.Delivery{}, errors.New("managed Codex operation failed")
	}
	var wire identity.DeliveryWire
	if len(out) > 384<<10 || strictjson.Decode(bytes.NewReader(out), &wire) != nil {
		return identity.Delivery{}, errors.New("invalid managed Codex response")
	}
	return identity.Delivery(wire), nil
}

// PrepareIdentity reserves one native terminal before the broker releases bytes.
func (s *Service) PrepareIdentity(ctx context.Context, lease identity.Lease, login, scope string, cols, rows int) (identity.Binding, error) {
	if !terminalPreparation(lease, login, scope, cols, rows) {
		return identity.Binding{}, identity.ErrDenied
	}
	container, err := s.projectContainer(ctx, lease.ProjectID, true)
	if err != nil {
		return identity.Binding{}, err
	}
	hash, err := AgentProgramHash()
	if err != nil {
		return identity.Binding{}, err
	}
	request := identityRequest{Container: container, Action: "prepare", Delivery: identity.DeliveryWire{Lease: lease}, Login: login, Scope: scope, Cols: cols, Rows: rows, SourceHash: hash}
	result, err := s.identityCall(ctx, container, request)
	if err != nil {
		return identity.Binding{}, err
	}
	if !terminalPrepared(result, lease, container, login) {
		return identity.Binding{}, errors.New("managed terminal reservation differs")
	}
	return *result.Lease.Binding, nil
}

// Identity executes fixed model-session operations against an immutable project.
func (s *Service) Identity(ctx context.Context, action string, delivery identity.Delivery) (identity.Delivery, error) {
	if !terminalLease(delivery.Lease, false) {
		return identity.Delivery{}, identity.ErrDenied
	}
	if !s.identityAction(action, delivery) {
		return identity.Delivery{}, identity.ErrDenied
	}
	container, done, err := s.identityTarget(ctx, action, delivery.Lease)
	if err != nil {
		return identity.Delivery{}, err
	}
	if done {
		return delivery, nil
	}
	if action == "start" {
		if err = s.identityStage(ctx, container, delivery); err != nil {
			return identity.Delivery{}, err
		}
	}
	result, err := s.identityCall(ctx, container, identityRequest{Action: action, Delivery: identity.DeliveryWire(delivery), HarnessSHA256: s.CodexHarnessSHA256})
	if err != nil {
		return result, err
	}
	return identityResult(action, delivery, result)
}

func identityResult(action string, delivery, result identity.Delivery) (identity.Delivery, error) {
	if result.Lease.ID != delivery.Lease.ID || result.Lease.Binding == nil {
		return result, identity.ErrStale
	}
	if *result.Lease.Binding != *delivery.Lease.Binding {
		return result, identity.ErrStale
	}
	if action == "finish" {
		if !identity.CredentialValid(result.Credential) {
			return result, identity.ErrUncertain
		}
	} else if len(result.Credential) != 0 {
		return identity.Delivery{}, errors.New("unexpected credential response")
	}
	return result, nil
}

func (s *Service) managedEnd(ctx context.Context, container string, request TerminalRequest) error {
	if request.Action != "end" {
		return nil
	}
	lookup := identityRequest{Action: "lookup", Login: request.Login, Delivery: identity.DeliveryWire{Lease: identity.Lease{ExecutionID: request.ID, ActorID: request.Identity}}}
	result, err := s.identityCall(ctx, container, lookup)
	if err != nil {
		return err
	}
	if result.Lease.ID == "" {
		return nil
	}
	if result.Lease.Binding == nil || result.Lease.Binding.Project != container || result.Lease.Binding.ID != request.ID || result.Lease.ActorID != request.Identity {
		return identity.ErrStale
	}
	if s.EndIdentity == nil {
		return identity.ErrDenied
	}
	return s.EndIdentity(ctx, request.Identity, result.Lease.ID)
}

func (s *Service) verifyIdentityHarness() error {
	path := filepath.Join(s.CodexHarness, "bin", "codex")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.New("verified Codex executable required")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != s.CodexHarnessSHA256 {
		return errors.New("codex harness digest differs")
	}
	return nil
}

func (s *Service) identityTarget(ctx context.Context, action string, lease identity.Lease) (string, bool, error) {
	if action != "stop" {
		target, err := s.projectContainer(ctx, lease.ProjectID, true)
		if err != nil {
			return "", false, err
		}
		if target != lease.Binding.Project {
			return "", false, identity.ErrStale
		}
		return target, false, nil
	}
	return s.identityStopTarget(ctx, lease)
}

func (s *Service) identityStopTarget(ctx context.Context, lease identity.Lease) (string, bool, error) {
	target := lease.Binding.Project
	_, err := s.podman(ctx, nil, "--remote=false", "container", "exists", target)
	if err != nil {
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return target, true, nil
		}
		return "", false, identity.ErrUncertain
	}
	data, err := s.podman(ctx, nil, "--remote=false", "inspect", "--format", terminalInspect, target)
	if err != nil || len(data) > 4096 {
		return "", false, identity.ErrUncertain
	}
	var v terminalInspection
	if strictjson.Decode(bytes.NewReader(data), &v) != nil {
		return "", false, identity.ErrUncertain
	}
	if v.ID != target || !terminalTargetReady(v, lease.ProjectID, false) {
		return "", false, identity.ErrStale
	}
	return target, !v.Running, nil
}

func (s *Service) identityStage(ctx context.Context, container string, delivery identity.Delivery) error {
	if err := s.verifyIdentityHarness(); err != nil {
		return err
	}
	if _, err := s.identityCall(ctx, container, identityRequest{Action: "stage", Delivery: identity.DeliveryWire(delivery)}); err != nil {
		return err
	}
	path := "/run/soda-terminals/" + delivery.Lease.ExecutionID + "/model/harness"
	return s.streamIdentityHarness(ctx, container, path)
}

func terminalPreparation(lease identity.Lease, login, scope string, cols, rows int) bool {
	return terminalLease(lease, true) && loginName.MatchString(login) && login != "root" && containerID.MatchString(scope) && terminalDimensions(cols, rows)
}

func terminalPrepared(result identity.Delivery, lease identity.Lease, container, login string) bool {
	if !terminalLease(result.Lease, false) {
		return false
	}
	return result.Lease.ID == lease.ID && result.Lease.Binding.Project == container && result.Lease.Binding.Login == login && len(result.Credential) == 0
}

func (s *Service) identityAction(action string, delivery identity.Delivery) bool {
	switch action {
	case "start":
		return identity.CredentialValid(delivery.Credential) && filepath.IsAbs(s.CodexHarness) && containerID.MatchString(s.CodexHarnessSHA256)
	case "validate", "finish", "stop":
		return len(delivery.Credential) == 0
	default:
		return false
	}
}
