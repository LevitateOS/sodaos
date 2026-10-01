// Check assessments bind one native PR's exact head and verified base to
// the required-check verdict behind a merge decision (ST11, S-checks).
// Soda evaluates the nonempty complete configured check set from observed
// native state; native Actions schedules work on separately managed
// capacity, and no agent narrative can mark checks passing: VerifyChecks
// takes no success-claim input at all, only the exact target, the adopted
// approval definitions, the current policy and one bracketed native
// observation. ST12 consumes the persisted assessment as check evidence.
package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// Check verdicts. Only CheckPass authorizes a merge decision; every other
// verdict names its reason and, for pending checks, what CI still owes.
const (
	// CheckPass means every adopted required check succeeded on the exact head.
	CheckPass = "pass"
	// CheckPending means only missing or pending evidence blocks; CI is
	// still running and the attempt waits without weakening verification.
	CheckPending = "pending"
	// CheckFailed means a required check reported a definitive non-pass
	// state; the failure is correction or intervention input, never a wait.
	CheckFailed = "failed"
	// CheckRefused means the assessment itself cannot authorize: the
	// head/base moved, the approval definitions changed without adoption,
	// or the evidence is incomplete or unreadable.
	CheckRefused = "refused"
)

// ValidCheckVerdict reports whether verdict is a known check verdict.
func ValidCheckVerdict(verdict string) bool {
	switch verdict {
	case CheckPass, CheckPending, CheckFailed, CheckRefused:
		return true
	default:
		return false
	}
}

// Check reason codes. Bounded; the view renders the static resolution.
const (
	CheckReasonPass               = "checks_pass"
	CheckReasonPending            = "checks_pending"
	CheckReasonFailed             = "check_failed"
	CheckReasonError              = "check_error"
	CheckReasonCancelled          = "check_cancelled"
	CheckReasonSkipped            = "check_skipped"
	CheckReasonWarning            = "check_warning"
	CheckReasonUnknownState       = "check_state_unknown"
	CheckReasonStaleHead          = "stale_head"
	CheckReasonStaleBase          = "stale_base"
	CheckReasonPolicyEmpty        = "policy_empty"
	CheckReasonDefinitionsChanged = "definitions_changed"
	CheckReasonEvidenceIncomplete = "evidence_incomplete"
	CheckReasonHidden             = "checks_hidden"
)

func validCheckReason(reason string) bool {
	switch reason {
	case CheckReasonPass, CheckReasonPending, CheckReasonFailed,
		CheckReasonError, CheckReasonCancelled, CheckReasonSkipped,
		CheckReasonWarning, CheckReasonUnknownState, CheckReasonStaleHead,
		CheckReasonStaleBase, CheckReasonPolicyEmpty,
		CheckReasonDefinitionsChanged, CheckReasonEvidenceIncomplete,
		CheckReasonHidden:
		return true
	default:
		return false
	}
}

// CheckResolution is the static maintainer-facing resolution for one check
// reason: what would resolve it. No free text enters resolutions.
func CheckResolution(reason string) string {
	switch reason {
	case CheckReasonPass:
		return "Every required check succeeded on the exact head."
	case CheckReasonPending:
		return "Wait for the pending checks to report; missing checks need a CI run on the exact head."
	case CheckReasonFailed:
		return "Correct the candidate so the failed check passes on a new head."
	case CheckReasonError:
		return "Resolve the check error and rerun CI on the exact head."
	case CheckReasonCancelled:
		return "Rerun the cancelled check on the exact head; cancellation is not a pass."
	case CheckReasonSkipped:
		return "A required check was skipped; run it on the exact head instead of waiving it."
	case CheckReasonWarning:
		return "Resolve the check warning so the check reports success on the exact head."
	case CheckReasonUnknownState:
		return "A required check reported an unrecognized state; rerun CI so it reports success."
	case CheckReasonStaleHead:
		return "The candidate moved; reassess the new head with fresh check evidence."
	case CheckReasonStaleBase:
		return "The target base moved; reverify the candidate against the new base."
	case CheckReasonPolicyEmpty:
		return "Configure a nonempty required-check set before automatic merge can proceed."
	case CheckReasonDefinitionsChanged:
		return "The approval definitions changed; a repository administrator must adopt the new set."
	case CheckReasonEvidenceIncomplete:
		return "Native check evidence is incomplete; reassess once the full set is readable."
	case CheckReasonHidden:
		return "The factory reader cannot see required check evidence; adjust native visibility."
	default:
		return ""
	}
}

// MaxObservedCheckContext bounds one observed native check context.
const MaxObservedCheckContext = 256

// MaxObservedCheckState bounds one observed native check state.
const MaxObservedCheckState = 64

// CheckTarget is the exact candidate and verified base one assessment
// binds: the PR linkage plus the exact head and base commits.
type CheckTarget struct {
	Repository int64  `json:"repository,string"`
	PRNumber   int64  `json:"pr_number,string"`
	PRID       int64  `json:"pr_id,string"`
	IssueID    int64  `json:"issue_id,string"`
	HeadRef    string `json:"head_ref"`
	BaseRef    string `json:"base_ref"`
	HeadOID    string `json:"head_oid"`
	BaseOID    string `json:"base_oid"`
}

// Validate rejects malformed targets. Shape only: observation verifies
// the PR linkage and tips against bracketed native evidence.
func (t CheckTarget) Validate() error {
	if t.Repository <= 0 || t.PRNumber <= 0 || t.PRID <= 0 || t.IssueID <= 0 {
		return errors.New("invalid check target scope")
	}
	if !ValidTargetBranch(t.HeadRef) || !ValidTargetBranch(t.BaseRef) || t.HeadRef == t.BaseRef {
		return errors.New("invalid check target refs")
	}
	if !ValidCommit(t.HeadOID) || !ValidCommit(t.BaseOID) {
		return errors.New("invalid check target commits")
	}
	return nil
}

// ChecksDigest binds one required-check set independent of order, so an
// adopted definition compares exactly against the current policy.
func ChecksDigest(checks []string) string {
	ordered := append([]string(nil), checks...)
	sort.Strings(ordered)
	sum := sha256.Sum256([]byte(strings.Join(ordered, "\x00")))
	return hex.EncodeToString(sum[:])
}

// AdoptedChecks are the approval definitions one attempt adopted: the
// policy revision and required-check set a repository administrator's
// adoption bound. Assessment refuses when the current policy differs.
type AdoptedChecks struct {
	Checks         []string `json:"checks"`
	PolicyRevision int64    `json:"policy_revision"`
	Digest         string   `json:"digest"`
}

// Validate rejects malformed adoptions. The set is nonempty, bounded and
// unique like the policy check set, and the digest binds exactly it.
func (a AdoptedChecks) Validate() error {
	if a.PolicyRevision < 0 {
		return errors.New("invalid adopted policy revision")
	}
	if len(a.Checks) == 0 || len(a.Checks) > MaxRequiredChecks {
		return errors.New("adopted checks require a nonempty bounded set")
	}
	seen := make(map[string]bool, len(a.Checks))
	for _, check := range a.Checks {
		if check == "" || len(check) > MaxCheckName || check != strings.TrimSpace(check) ||
			strings.IndexFunc(check, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 || seen[check] {
			return errors.New("invalid adopted check name")
		}
		seen[check] = true
	}
	if !ValidDigest(a.Digest) || a.Digest != ChecksDigest(a.Checks) {
		return errors.New("adopted checks digest differs from their set")
	}
	return nil
}

// ObservedCheck is the latest native status for one context on the exact
// head. Empty State means the context has no status on this head.
type ObservedCheck struct {
	Context string `json:"context"`
	State   string `json:"state,omitempty"`
}

func (c ObservedCheck) Validate() error {
	if c.Context == "" || len(c.Context) > MaxObservedCheckContext ||
		strings.IndexFunc(c.Context, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return errors.New("invalid observed check context")
	}
	if len(c.State) > MaxObservedCheckState ||
		strings.IndexFunc(c.State, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return errors.New("invalid observed check state")
	}
	return nil
}

// ObservedChecks is one bracketed native observation for a check target:
// the bound idle revision, the observed head/base tips, the latest status
// per context on the exact head, and whether anything required was hidden.
type ObservedChecks struct {
	Checks           []ObservedCheck `json:"checks,omitempty"`
	NativeRev        int64           `json:"native_revision"`
	ObservedContexts int             `json:"observed_contexts"`
	HeadTip          string          `json:"head_tip,omitempty"`
	BaseTip          string          `json:"base_tip,omitempty"`
	Complete         bool            `json:"complete"`
	Hidden           bool            `json:"hidden,omitempty"`
}

// Validate rejects malformed observations. Shape only: the reader binds
// the revision, tips and latest-per-context statuses from one bracket.
func (o ObservedChecks) Validate() error {
	if o.NativeRev < 1 || o.ObservedContexts < 0 || len(o.Checks) > SnapshotPageBound {
		return errors.New("invalid observed check evidence")
	}
	if o.HeadTip != "" && !ValidCommit(o.HeadTip) {
		return errors.New("invalid observed head tip")
	}
	if o.BaseTip != "" && !ValidCommit(o.BaseTip) {
		return errors.New("invalid observed base tip")
	}
	seen := make(map[string]bool, len(o.Checks))
	for _, check := range o.Checks {
		if err := check.Validate(); err != nil {
			return err
		}
		if seen[check.Context] {
			return errors.New("duplicate observed check context")
		}
		seen[check.Context] = true
	}
	return nil
}

// SnapshotPageBound mirrors the native snapshot page bound so
// observations stay within one complete native page.
const SnapshotPageBound = 50

// CheckResult is one required check's verdict: its latest observed native
// state on the exact head and whether that state passes. Empty State
// means the check is missing on this head.
type CheckResult struct {
	Context string `json:"context"`
	State   string `json:"state,omitempty"`
	Passed  bool   `json:"passed"`
}

func (r CheckResult) Validate() error {
	if r.Context == "" || len(r.Context) > MaxCheckName {
		return errors.New("invalid check result context")
	}
	if len(r.State) > MaxObservedCheckState {
		return errors.New("invalid check result state")
	}
	if r.Passed != (r.State == CheckStateSuccess) {
		return errors.New("check result pass differs from its success state")
	}
	return nil
}

// CheckStateSuccess is the only native state that passes a required
// check. Every other state, including a missing status, is not a pass.
const CheckStateSuccess = "success"

// checkStateSeverity ranks non-pass states for reason selection. Higher
// ranks are definitive failures; pending and missing only wait for CI.
func checkStateSeverity(state string) (rank int, reason string) {
	switch state {
	case CheckStateSuccess:
		return 0, ""
	case "failure":
		return 70, CheckReasonFailed
	case "error":
		return 60, CheckReasonError
	case "cancelled":
		return 50, CheckReasonCancelled
	case "skipped":
		return 40, CheckReasonSkipped
	case "warning":
		return 20, CheckReasonWarning
	case "pending", "":
		return 10, CheckReasonPending
	default:
		return 30, CheckReasonUnknownState
	}
}

// CheckAssessment is one recorded check verdict for an exact head and
// verified base: the adopted definitions, the bound native revision and
// the per-check results behind the verdict. ST12 consumes it as check
// evidence and must re-verify its head/base/policy bindings before use.
type CheckAssessment struct {
	Results          []CheckResult `json:"results"`
	Checks           []string      `json:"checks"`
	Repository       int64         `json:"repository,string"`
	PRNumber         int64         `json:"pr_number,string"`
	PRID             int64         `json:"pr_id,string"`
	IssueID          int64         `json:"issue_id,string"`
	PolicyRevision   int64         `json:"policy_revision"`
	NativeRev        int64         `json:"native_revision"`
	Revision         int64         `json:"revision"`
	AssessedUnix     int64         `json:"assessed_unix"`
	ObservedContexts int           `json:"observed_contexts"`
	HeadRef          string        `json:"head_ref"`
	BaseRef          string        `json:"base_ref"`
	HeadOID          string        `json:"head_oid"`
	BaseOID          string        `json:"base_oid"`
	ChecksDigest     string        `json:"checks_digest"`
	Verdict          string        `json:"verdict"`
	Reason           string        `json:"reason"`
}

// Validate rejects malformed assessments. A pass carries only passing
// results; any other verdict carries the results behind it.
func (a CheckAssessment) Validate() error {
	if a.Repository <= 0 || a.PRNumber <= 0 || a.PRID <= 0 || a.IssueID <= 0 {
		return errors.New("invalid check assessment scope")
	}
	if !ValidTargetBranch(a.HeadRef) || !ValidTargetBranch(a.BaseRef) || a.HeadRef == a.BaseRef {
		return errors.New("invalid check assessment refs")
	}
	if !ValidCommit(a.HeadOID) || !ValidCommit(a.BaseOID) {
		return errors.New("invalid check assessment commits")
	}
	if a.PolicyRevision < 0 || a.NativeRev < 1 || a.Revision < 0 || a.AssessedUnix <= 0 || a.ObservedContexts < 0 {
		return errors.New("invalid check assessment binding")
	}
	if len(a.Checks) == 0 || len(a.Checks) > MaxRequiredChecks {
		return errors.New("check assessment requires its nonempty check set")
	}
	seen := make(map[string]bool, len(a.Checks))
	for _, check := range a.Checks {
		if check == "" || len(check) > MaxCheckName || seen[check] {
			return errors.New("invalid check assessment check name")
		}
		seen[check] = true
	}
	if !ValidDigest(a.ChecksDigest) || a.ChecksDigest != ChecksDigest(a.Checks) {
		return errors.New("check assessment digest differs from its set")
	}
	if !ValidCheckVerdict(a.Verdict) || !validCheckReason(a.Reason) {
		return errors.New("invalid check assessment verdict")
	}
	if len(a.Results) != len(a.Checks) {
		return errors.New("check assessment results differ from its set")
	}
	for i, result := range a.Results {
		if err := result.Validate(); err != nil {
			return err
		}
		if result.Context != a.Checks[i] {
			return errors.New("check assessment results differ from its set")
		}
	}
	switch a.Verdict {
	case CheckPass:
		if a.Reason != CheckReasonPass {
			return errors.New("check pass names its reason")
		}
		for _, result := range a.Results {
			if !result.Passed {
				return errors.New("check pass carries only passing results")
			}
		}
	case CheckPending:
		if a.Reason != CheckReasonPending {
			return errors.New("pending check assessment names its reason")
		}
	case CheckFailed:
		switch a.Reason {
		case CheckReasonFailed, CheckReasonError, CheckReasonCancelled,
			CheckReasonSkipped, CheckReasonWarning, CheckReasonUnknownState:
		default:
			return errors.New("failed check assessment names its failure")
		}
	case CheckRefused:
		switch a.Reason {
		case CheckReasonStaleHead, CheckReasonStaleBase, CheckReasonPolicyEmpty,
			CheckReasonDefinitionsChanged, CheckReasonEvidenceIncomplete,
			CheckReasonHidden:
		default:
			return errors.New("refused check assessment names its cause")
		}
	}
	return nil
}

// VerifyChecks evaluates the adopted required-check set against one
// bracketed native observation of the exact head and verified base. The
// current policy must still carry exactly the adopted definitions;
// anything else refuses without consulting agent claims, which have no
// input here. Malformed inputs error; every semantic outcome is a
// recorded verdict, including refusals.
func VerifyChecks(target CheckTarget, adopted AdoptedChecks, current RepositoryPolicy, observed ObservedChecks, nowUnix int64) (CheckAssessment, error) {
	var empty CheckAssessment
	if err := target.Validate(); err != nil {
		return empty, err
	}
	if err := adopted.Validate(); err != nil {
		return empty, err
	}
	if err := observed.Validate(); err != nil {
		return empty, err
	}
	if nowUnix <= 0 {
		return empty, errors.New("check assessment needs its record time")
	}
	assessment := CheckAssessment{
		Checks: adopted.Checks, Repository: target.Repository,
		PRNumber: target.PRNumber, PRID: target.PRID, IssueID: target.IssueID,
		PolicyRevision: adopted.PolicyRevision, NativeRev: observed.NativeRev,
		AssessedUnix: nowUnix, ObservedContexts: observed.ObservedContexts,
		HeadRef: target.HeadRef, BaseRef: target.BaseRef,
		HeadOID: target.HeadOID, BaseOID: target.BaseOID,
		ChecksDigest: adopted.Digest,
	}
	for _, check := range adopted.Checks {
		assessment.Results = append(assessment.Results, CheckResult{Context: check})
	}
	// A missing policy arrives as the zero record: zero configured checks
	// can never authorize a merge.
	if current.Repository != target.Repository || len(current.Checks) == 0 {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonPolicyEmpty
		return assessment, nil
	}
	if current.Revision != adopted.PolicyRevision || ChecksDigest(current.Checks) != adopted.Digest {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonDefinitionsChanged
		return assessment, nil
	}
	if !observed.Complete {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonEvidenceIncomplete
		return assessment, nil
	}
	if observed.Hidden {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonHidden
		return assessment, nil
	}
	if observed.HeadTip != target.HeadOID {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonStaleHead
		return assessment, nil
	}
	if observed.BaseTip != target.BaseOID {
		assessment.Verdict, assessment.Reason = CheckRefused, CheckReasonStaleBase
		return assessment, nil
	}
	latest := make(map[string]string, len(observed.Checks))
	for _, check := range observed.Checks {
		latest[check.Context] = check.State
	}
	worst, reason := 0, CheckReasonPass
	for i, result := range assessment.Results {
		state := latest[result.Context]
		assessment.Results[i].State = state
		assessment.Results[i].Passed = state == CheckStateSuccess
		if rank, cause := checkStateSeverity(state); rank > worst {
			worst, reason = rank, cause
		}
	}
	switch {
	case worst == 0:
		assessment.Verdict, assessment.Reason = CheckPass, CheckReasonPass
	case worst <= 10:
		assessment.Verdict, assessment.Reason = CheckPending, CheckReasonPending
	default:
		assessment.Verdict, assessment.Reason = CheckFailed, reason
	}
	return assessment, nil
}
