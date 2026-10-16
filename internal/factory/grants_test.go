package factory

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func grantTestPolicy() RepositoryPolicy {
	roles := map[string]RoleSelection{
		project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test-model"},
		project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test-model"},
	}
	return RepositoryPolicy{
		Repository: 42, GrantedBy: 7, Enabled: true,
		TargetBranch: "refs/heads/main", Roles: roles,
		Checks:              []string{"native-ci/build"},
		MergeMethod:         MergeFastForward,
		Publish:             ActorBindingRef{TokenID: 11, ActorID: 12, Kind: OpRefPublish},
		Create:              ActorBindingRef{TokenID: 11, ActorID: 12, Kind: OpPRCreate},
		Review:              ActorBindingRef{TokenID: 13, ActorID: 14, Kind: OpReviewSubmit},
		Merge:               ActorBindingRef{TokenID: 15, ActorID: 16, Kind: OpMerge},
		AttemptLimits:       DefaultAttemptLimits(),
		ReassessmentMinutes: DefaultReassessmentMinutes,
		MaxConcurrent:       2,
	}
}

func TestRepositoryPolicyValidation(t *testing.T) {
	valid := grantTestPolicy()
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	mutate := func(f func(*RepositoryPolicy)) RepositoryPolicy {
		p := grantTestPolicy()
		f(&p)
		return p
	}
	for name, bad := range map[string]RepositoryPolicy{
		"empty checks":     mutate(func(p *RepositoryPolicy) { p.Checks = nil }),
		"duplicate checks": mutate(func(p *RepositoryPolicy) { p.Checks = []string{"a", "a"} }),
		"bare branch":      mutate(func(p *RepositoryPolicy) { p.TargetBranch = "main" }),
		"missing role":     mutate(func(p *RepositoryPolicy) { delete(p.Roles, project.RoleReviewer) }),
		"extra role": mutate(func(p *RepositoryPolicy) {
			p.Roles["soda-admin"] = RoleSelection{Harness: "x", HarnessVers: "x", Model: "y"}
		}),
		"missing family": mutate(func(p *RepositoryPolicy) {
			p.Roles[project.RoleCoder] = RoleSelection{HarnessVers: "0.157.1", Model: "m"}
		}),
		"missing version": mutate(func(p *RepositoryPolicy) {
			p.Roles[project.RoleCoder] = RoleSelection{Harness: project.FactoryHarnessCodex, Model: "m"}
		}),
		"unsupported family": mutate(func(p *RepositoryPolicy) {
			p.Roles[project.RoleCoder] = RoleSelection{Harness: "unknown", HarnessVers: "0.157.1", Model: "m"}
		}),
		"merge method":                   mutate(func(p *RepositoryPolicy) { p.MergeMethod = "merge" }),
		"swappedKind":                    mutate(func(p *RepositoryPolicy) { p.Merge.Kind = OpRefPublish }),
		"zero token":                     mutate(func(p *RepositoryPolicy) { p.Publish.TokenID = 0 }),
		"zero concurrency":               mutate(func(p *RepositoryPolicy) { p.MaxConcurrent = 0 }),
		"high concurrency":               mutate(func(p *RepositoryPolicy) { p.MaxConcurrent = 99 }),
		"zero active attempt time":       mutate(func(p *RepositoryPolicy) { p.AttemptLimits.ActiveMinutes = 0 }),
		"negative correction cycles":     mutate(func(p *RepositoryPolicy) { p.AttemptLimits.CorrectionCycles = -1 }),
		"zero review rounds":             mutate(func(p *RepositoryPolicy) { p.AttemptLimits.ReviewRounds = 0 }),
		"excessive review rounds":        mutate(func(p *RepositoryPolicy) { p.AttemptLimits.ReviewRounds = 17 }),
		"zero child concurrency":         mutate(func(p *RepositoryPolicy) { p.AttemptLimits.MaxChildren = 0 }),
		"excessive child concurrency":    mutate(func(p *RepositoryPolicy) { p.AttemptLimits.MaxChildren = 5 }),
		"negative delivery calls":        mutate(func(p *RepositoryPolicy) { p.AttemptLimits.DeliveryCalls = -1 }),
		"excessive delivery calls":       mutate(func(p *RepositoryPolicy) { p.AttemptLimits.DeliveryCalls = 17 }),
		"omitted reassessment minutes":   mutate(func(p *RepositoryPolicy) { p.ReassessmentMinutes = 0 }),
		"excessive reassessment minutes": mutate(func(p *RepositoryPolicy) { p.ReassessmentMinutes = 121 }),
		"negative reassessment minutes":  mutate(func(p *RepositoryPolicy) { p.ReassessmentMinutes = -5 }),
		"contributor on codex": mutate(func(p *RepositoryPolicy) {
			roles := p.Roles
			coder := roles[project.RoleCoder]
			coder.Profile = ExecutionProfile{Effort: EffortLow, Contributor: true}
			roles[project.RoleCoder] = coder
		}),
		"unknown effort": mutate(func(p *RepositoryPolicy) {
			roles := p.Roles
			coder := roles[project.RoleCoder]
			coder.Profile = ExecutionProfile{Effort: "turbo"}
			roles[project.RoleCoder] = coder
		}),
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("%s: policy accepted", name)
		}
	}
}

// D13 (FW06A scope): admission resolves an omitted reassessment allowance
// to the default so the stored record always carries its value explicitly;
// the admitted range is 1-120.
func TestReassessmentMinutesAdmission(t *testing.T) {
	if got := EffectiveReassessmentMinutes(0); got != DefaultReassessmentMinutes {
		t.Fatalf("omitted reassessment minutes misresolved: %d", got)
	}
	for _, minutes := range []int{MinReassessmentMinutes, DefaultReassessmentMinutes, MaxReassessmentMinutes} {
		p := grantTestPolicy()
		p.ReassessmentMinutes = minutes
		if err := p.Validate(); err != nil {
			t.Fatalf("reassessment minutes %d refused: %v", minutes, err)
		}
		if got := EffectiveReassessmentMinutes(minutes); got != minutes {
			t.Fatalf("explicit reassessment minutes rewritten: %d -> %d", minutes, got)
		}
	}
}

func TestRoleExecutionProfile(t *testing.T) {
	// An omitted profile validates: creation resolves the product defaults.
	omitted := RoleSelection{Harness: project.FactoryHarnessMuse, HarnessVers: "1.4.9", Model: "spark"}
	if err := omitted.Validate(); err != nil {
		t.Fatalf("omitted profile refused: %v", err)
	}
	crew := RoleSelection{
		Harness: project.FactoryHarnessMuse, HarnessVers: "1.4.9", Model: "spark",
		Profile: ExecutionProfile{Effort: EffortXHigh, Contributor: true},
	}
	if err := crew.Validate(); err != nil {
		t.Fatalf("qualified crew profile refused: %v", err)
	}
	encoded, err := json.Marshal(crew)
	if err != nil || !strings.Contains(string(encoded), `"effort":"xhigh"`) || !strings.Contains(string(encoded), `"contributor":true`) {
		t.Fatalf("profile JSON is not snake_case: %s %v", encoded, err)
	}
	if !SettingsCommandType(CommandPlanIntent) {
		t.Fatal("plan-intent is not a settings command")
	}
}

func TestGrantValidation(t *testing.T) {
	if err := (Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Capacity{UpdatedBy: 1}).Validate(); err == nil {
		t.Fatal("empty capacity accepted")
	}
	if err := (OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (OperatorGrant{Repository: 42, GrantedBy: 1, Active: true}).Validate(); err == nil {
		t.Fatal("unbounded operator grant accepted")
	}
	sponsorship := Sponsorship{
		Repository: 42, GrantedBy: 9, ActorID: 9, ProjectID: "p" + strings.Repeat("a", 24), Connection: "conn-1", GrantID: "grant-1",
		Generation: 3, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
	}
	if err := sponsorship.Validate(); err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = []string{"soda-admin"}
	if err := sponsorship.Validate(); err == nil {
		t.Fatal("foreign sponsorship role accepted")
	}
}

func TestSettingsCommandDigest(t *testing.T) {
	payload := `{"enabled":true}`
	cmd := Command{
		ID: NewID(), Type: CommandPolicy, Target: "repository/42/policy",
		Principal: "native:7", Payload: payload, Digest: SettingsDigest(CommandPolicy, "repository/42/policy", payload),
	}
	if err := cmd.Validate(); err != nil {
		t.Fatal(err)
	}
	cmd.Digest = CommandDigest(cmd.Type, cmd.Target)
	if err := cmd.Validate(); err == nil {
		t.Fatal("settings command accepted the operator digest")
	}
	stop := Command{
		ID: NewID(), Type: CommandStop, Target: NewID(), Principal: "os-uid:0", Payload: payload,
		Digest: SettingsDigest(CommandStop, "x", payload),
	}
	if err := stop.Validate(); err == nil {
		t.Fatal("operator command accepted a settings payload")
	}
}

func TestEvaluateAuthority(t *testing.T) {
	policy := grantTestPolicy()
	operator := OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}
	capacity := Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2}
	sponsorship := Sponsorship{
		Repository: 42, GrantedBy: 9, ActorID: 9, ProjectID: "p" + strings.Repeat("a", 24), Connection: "c", GrantID: "g",
		Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
	}
	env := project.EnvironmentGrant{Repository: 42, Owner: 7, Active: true}
	budget := ConnectionUsageBudget{Connection: "c", Revision: 2, RollingMinutes: 720}
	full := AuthorityInput{
		Policy: &policy, Operator: &operator, Appliance: &capacity,
		Sponsorship: &sponsorship, ConnectionUsageBudget: &budget, Environment: &env, DispatchOpen: true,
	}
	if got := EvaluateAuthority(full); !got.Effective || len(got.Missing) != 0 {
		t.Fatal(got)
	}
	if got := EvaluateAuthority(AuthorityInput{DispatchOpen: true}); got.Effective || len(got.Missing) != 5 {
		t.Fatalf("empty authority: %+v", got)
	}
	paused := policy
	paused.Paused = true
	got := EvaluateAuthority(AuthorityInput{
		Policy: &paused, Operator: &operator, Appliance: &capacity,
		Sponsorship: &sponsorship, ConnectionUsageBudget: &budget, Environment: &env, DispatchOpen: true,
	})
	if got.Effective || len(got.Missing) != 1 || got.Missing[0] != MissingPolicyPaused {
		t.Fatalf("paused policy: %+v", got)
	}
	closed := full
	closed.DispatchOpen = false
	if got := EvaluateAuthority(closed); got.Effective || got.Missing[len(got.Missing)-1] != MissingDispatch {
		t.Fatalf("closed dispatch: %+v", got)
	}
	serial, err := json.Marshal(EvaluateAuthority(closed))
	if err != nil || strings.Contains(string(serial), "secret") {
		t.Fatal(string(serial), err)
	}
}

func TestConnectionUsageBudgetMicrosecondRange(t *testing.T) {
	valid := ConnectionUsageBudget{Connection: "c", RollingMinutes: 720}
	if err := valid.Validate(); err != nil {
		t.Fatal("owner increase above the initial value rejected:", err)
	}
	tooLarge := valid
	tooLarge.RollingMinutes = int64(^uint64(0)>>1)/UsageMicrosPerMinute + 1
	if err := tooLarge.Validate(); err == nil {
		t.Fatal("unrepresentable microsecond budget accepted")
	}
	sponsorship := Sponsorship{Connection: "c", Active: true}
	missing := EvaluateAuthority(AuthorityInput{Sponsorship: &sponsorship, DispatchOpen: true})
	budgetMissing := false
	for _, reason := range missing.Missing {
		budgetMissing = budgetMissing || reason == MissingConnectionUsageBudget
	}
	if missing.Effective || !budgetMissing {
		t.Fatalf("sponsorship without canonical budget remained effective: %+v", missing)
	}
}

func TestWithdrawalValidation(t *testing.T) {
	good := Withdrawal{Repository: 42, Cause: "policy_paused", ClosedBy: "native:7", Captured: []string{NewID()}, ActiveCauses: []string{"policy_paused"}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	good.Captured = []string{"not-an-id"}
	if err := good.Validate(); err == nil {
		t.Fatal("withdrawal captured an invalid identity")
	}
}

func TestResolveBindingSlot(t *testing.T) {
	unenrolled := grantTestPolicy()
	if err := unenrolled.Validate(); err != nil {
		t.Fatalf("policy without resolve enrollment refused: %v", err)
	}
	if _, err := unenrolled.ResolveBinding(); err == nil {
		t.Fatal("missing resolve binding resolved")
	}
	if ValidOperationKind(OpThreadResolve) != true {
		t.Fatal("thread resolve kind rejected")
	}
	enrolled := grantTestPolicy()
	enrolled.Resolve = ActorBindingRef{TokenID: 13, ActorID: 14, Kind: OpThreadResolve}
	if err := enrolled.Validate(); err != nil {
		t.Fatalf("enrolled resolve binding refused: %v", err)
	}
	binding, err := enrolled.ResolveBinding()
	if err != nil || binding != enrolled.Resolve {
		t.Fatalf("enrolled resolve binding not returned: %v", err)
	}
	borrowed := grantTestPolicy()
	borrowed.Resolve = ActorBindingRef{TokenID: 13, ActorID: 14, Kind: OpReviewSubmit}
	if err := borrowed.Validate(); err == nil {
		t.Fatal("review binding accepted in the resolve slot")
	}
	if _, err := borrowed.ResolveBinding(); err == nil {
		t.Fatal("wrong-kind resolve binding returned")
	}
}
