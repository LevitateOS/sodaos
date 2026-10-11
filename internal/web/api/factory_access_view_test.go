package api

import (
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestAccessEditableDTO(t *testing.T) {
	admin := accessViewerView{CodeWrite: true, OwnerOrAdmin: true, EnvironmentOwner: true, SessionCurrent: true}
	full := accessEditableDTO(accessEditableInput{Viewer: admin, Connections: []string{"conn-1"}, Owned: map[string]bool{"conn-1": true}, IdentityOK: true})
	for name, entry := range map[string]accessEntryView{
		"policy": full.Policy, "activity": full.ActivitySettings, "environment": full.EnvironmentGrant,
	} {
		if !entry.Editable {
			t.Fatalf("%s not editable: %+v", name, entry)
		}
	}
	if full.OperatorGrant.Editable || full.OperatorGrant.Reason != accessReasonOperator {
		t.Fatalf("operator grant: %+v", full.OperatorGrant)
	}
	if full.Capacity.Editable || full.Capacity.Reason != accessReasonOperator {
		t.Fatalf("capacity: %+v", full.Capacity)
	}
	if len(full.Sponsorships) != 1 || !full.Sponsorships[0].Editable || len(full.UsageBudgets) != 1 || !full.UsageBudgets[0].Editable {
		t.Fatalf("connections: %+v %+v", full.Sponsorships, full.UsageBudgets)
	}

	operator := accessViewerView{Operator: true, SessionCurrent: true}
	grants := accessEditableDTO(accessEditableInput{Viewer: operator})
	if !grants.OperatorGrant.Editable || !grants.Capacity.Editable {
		t.Fatalf("operator entries: %+v %+v", grants.OperatorGrant, grants.Capacity)
	}
	if grants.Policy.Editable || grants.Policy.Reason != accessReasonOwnerAdmin {
		t.Fatalf("operator policy: %+v", grants.Policy)
	}
	if grants.EnvironmentGrant.Editable || grants.EnvironmentGrant.Reason != accessReasonRepoOwner {
		t.Fatalf("operator environment: %+v", grants.EnvironmentGrant)
	}

	viewer := accessViewerView{CodeWrite: true, SessionCurrent: true}
	denied := accessEditableDTO(accessEditableInput{Viewer: viewer, Connections: []string{"conn-1"}, Owned: map[string]bool{}, IdentityOK: true})
	if denied.Policy.Editable || denied.EnvironmentGrant.Editable || denied.OperatorGrant.Editable {
		t.Fatalf("viewer powers: %+v", denied)
	}
	if denied.Sponsorships[0].Editable || denied.Sponsorships[0].Reason != accessReasonConnOwner {
		t.Fatalf("viewer sponsorship: %+v", denied.Sponsorships[0])
	}

	stale := accessEditableDTO(accessEditableInput{Viewer: accessViewerView{OwnerOrAdmin: true, Operator: true}, Connections: []string{"conn-1"}, Owned: map[string]bool{"conn-1": true}, IdentityOK: true})
	for name, entry := range map[string]accessEntryView{
		"policy": stale.Policy, "operator": stale.OperatorGrant, "environment": stale.EnvironmentGrant,
	} {
		if entry.Editable || entry.Reason != accessReasonSession {
			t.Fatalf("%s stale session: %+v", name, entry)
		}
	}
	if stale.Sponsorships[0].Editable || stale.Sponsorships[0].Reason != accessReasonSession {
		t.Fatalf("stale sponsorship: %+v", stale.Sponsorships[0])
	}

	blind := accessEditableDTO(accessEditableInput{Viewer: admin, Connections: []string{"conn-1"}})
	if blind.Sponsorships[0].Editable || blind.Sponsorships[0].Reason != accessReasonIdentity {
		t.Fatalf("identity down: %+v", blind.Sponsorships[0])
	}
	if !blind.Policy.Editable {
		t.Fatalf("identity down policy: %+v", blind.Policy)
	}
}

func TestAccessRequiredActions(t *testing.T) {
	actions := accessRequiredActions([]string{
		factory.MissingPolicy, factory.MissingPolicyDisabled, factory.MissingPolicyPaused,
		factory.MissingOperatorGrant, factory.MissingOperatorWithdrawn, factory.MissingCapacity,
		factory.MissingSponsorship, factory.MissingSponsorshipGone, factory.MissingConnectionUsageBudget,
		factory.MissingEnvironment, factory.MissingEnvironmentGone,
		factory.MissingPreparation, factory.MissingDispatch,
	})
	expect := map[string][2]string{
		factory.MissingPolicy: {"owner_or_admin", "policy"}, factory.MissingPolicyDisabled: {"owner_or_admin", "policy"},
		factory.MissingPolicyPaused:  {"owner_or_admin", "policy"},
		factory.MissingOperatorGrant: {"operator", "operator_grant"}, factory.MissingOperatorWithdrawn: {"operator", "operator_grant"},
		factory.MissingCapacity:    {"operator", "capacity"},
		factory.MissingSponsorship: {"connection_owner", "sponsorships"}, factory.MissingSponsorshipGone: {"connection_owner", "sponsorships"},
		factory.MissingConnectionUsageBudget: {"connection_owner", "usage_budgets"},
		factory.MissingEnvironment:           {"owner_or_admin", "environment_grant"}, factory.MissingEnvironmentGone: {"owner_or_admin", "environment_grant"},
		factory.MissingPreparation: {"owner_or_admin", "project"}, factory.MissingDispatch: {"maintainer", "lifecycle"},
	}
	if len(actions) != len(expect) {
		t.Fatalf("actions: %+v", actions)
	}
	for _, action := range actions {
		want, ok := expect[action.Code]
		if !ok || action.ResolveBy != want[0] || action.Setting != want[1] {
			t.Fatalf("action: %+v", action)
		}
	}
	unknown := accessRequiredActions([]string{"future_code"})
	if len(unknown) != 1 || unknown[0].ResolveBy != "" || unknown[0].Code != "future_code" {
		t.Fatalf("unknown code: %+v", unknown)
	}
	if got := accessRequiredActions(nil); len(got) != 0 {
		t.Fatalf("no missing: %+v", got)
	}
}

func TestAccessUsageDTO(t *testing.T) {
	shown := accessUsageDTO(accessUsageInput{Connection: "conn-1", Owned: true, BudgetOK: true, UsedOK: true, Budget: 500, Used: 0})
	if shown.Status != "" || shown.Unit != "minutes" || shown.BudgetMinutes != 500 || shown.UsedMinutes == nil || *shown.UsedMinutes != 0 {
		t.Fatalf("owner usage: %+v", shown)
	}
	hidden := accessUsageDTO(accessUsageInput{Connection: "conn-1"})
	if hidden.Status != accessUsageOwnerOnly || hidden.BudgetMinutes != 0 || hidden.UsedMinutes != nil {
		t.Fatalf("foreign usage: %+v", hidden)
	}
	broken := accessUsageDTO(accessUsageInput{Connection: "conn-1", Owned: true, BudgetOK: true})
	if broken.Status != accessUsageUnavailable || broken.UsedMinutes != nil {
		t.Fatalf("unavailable usage: %+v", broken)
	}
}
