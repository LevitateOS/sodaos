package identity

import (
	"testing"
	"time"
)

func TestGrantRequiresBothConfirmations(t *testing.T) {
	r := GrantRequest{ConnectionID: "connection", UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.ConfirmCredentialExposure = false
	if r.Validate() == nil {
		t.Fatal("credential exposure was not confirmed")
	}
	r.ConfirmCredentialExposure = true
	r.ConfirmSubscription = false
	if r.Validate() == nil {
		t.Fatal("subscription use was not confirmed")
	}
}

func TestLeaseRequiresBoundedDeadlineAndExecution(t *testing.T) {
	now := time.Now()
	r := AcquireRequest{ProviderID: Codex, ActorID: 1, ConnectionID: "connection", ExecutionID: "execution", Kind: Factory, Deadline: now.Add(time.Hour)}
	if err := r.Validate(now); err != nil {
		t.Fatal(err)
	}
	r.Deadline = now
	if r.Validate(now) == nil {
		t.Fatal("expired request admitted")
	}
	r.Deadline = now.Add(25 * time.Hour)
	if r.Validate(now) == nil {
		t.Fatal("unbounded request admitted")
	}
}
