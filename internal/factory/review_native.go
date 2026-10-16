package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
)

// ReviewWork is one immutable body-only review authorization. Native PR and
// issue identities come from the recorded creation receipt, not a PR search.
// The reviewer owns findings; this work grants no ref publication.
type ReviewWork struct {
	OperationID  string `json:"operation_id"`
	AuthRevision string `json:"authorization_revision"`
	Repository   int64  `json:"repository"`
	ActorID      int64  `json:"actor_id"`
	PRNumber     int64  `json:"pr_number"`
	PRID         int64  `json:"pr_id"`
	IssueID      int64  `json:"issue_id"`
	PRAuthorID   int64  `json:"pr_author_id"`
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
	NativeRev    int64  `json:"native_revision"`
	NotAfter     int64  `json:"not_after"`
	Event        string `json:"event"`
	Body         string `json:"body"`
}

// ReviewOperation retains one review run's exact immutable native work and
// the latest attributable operation outcome inside its publication record.
// The work itself is the native tuple; this wrapper owns only its run binding
// and reconciliation state.
type ReviewOperation struct {
	RunID   string           `json:"run_id"`
	Work    ReviewWork       `json:"work"`
	Outcome OperationOutcome `json:"outcome"`
}

// ValidateForPublication checks the durable operation against the exact PR
// identity and candidate history that admitted it. Historical actor, head and
// deadline values remain immutable when current policy or publication state
// later changes.
func (op ReviewOperation) ValidateForPublication(p Publication) error {
	if !ValidID(op.RunID) || op.Work.OperationID != "review-"+op.RunID {
		return errors.New("invalid review run operation identity")
	}
	if err := op.Work.Validate(); err != nil {
		return err
	}
	if op.Work.Repository != p.Repository || op.Work.PRNumber != p.PRNumber || op.Work.PRID != p.PRID ||
		op.Work.IssueID != p.PRCreate.IssueID || op.Work.PRAuthorID != p.PRCreate.Work.ActorID ||
		op.Work.HeadRef != p.PRCreate.HeadRef || op.Work.BaseRef != p.PRCreate.BaseRef || op.Work.BaseOID != p.PRCreate.BaseOID ||
		op.Work.AuthRevision != ReviewAuthRevision(p.AssignmentID, op.RunID) {
		return errors.New("review work differs from its linked publication")
	}
	knownHead := p.Publish.Work != nil && op.Work.HeadOID == p.Publish.Work.Candidate
	for _, correction := range p.Corrections {
		if correction.Effect == OpEffectCommitted && correction.Work != nil && op.Work.HeadOID == correction.Work.Candidate {
			knownHead = true
		}
	}
	if !knownHead {
		return errors.New("review head is outside the publication candidate history")
	}
	if op.Outcome.NotObserved {
		return errors.New("review outcome cannot persist an absent lookup")
	}
	if op.Outcome.OperationID == "" {
		if op.Outcome.InstallationID != "" || op.Outcome.Kind != "" || op.Outcome.ActorID != 0 || op.Outcome.RepositoryID != 0 ||
			op.Outcome.Effect != "" || op.Outcome.Cancellation != "" || op.Outcome.Completion != "" || op.Outcome.Reason != "" || len(op.Outcome.Receipt) != 0 {
			return errors.New("review outcome is incomplete")
		}
	} else if op.Outcome.OperationID != op.Work.OperationID {
		return errors.New("review outcome identity differs")
	}
	if op.Outcome.InstallationID != "" && len(op.Outcome.InstallationID) > 256 {
		return errors.New("review installation identity exceeds bounds")
	}
	if op.Outcome.Effect != "" && !ValidOpEffect(op.Outcome.Effect) {
		return errors.New("invalid review operation effect")
	}
	if op.Outcome.Cancellation != "" && !ValidOpCancellation(op.Outcome.Cancellation) {
		return errors.New("invalid review operation cancellation")
	}
	if op.Outcome.Completion != "" && !ValidOpCompletion(op.Outcome.Completion) {
		return errors.New("invalid review operation completion")
	}
	if len(op.Outcome.Reason) > 64 || len(op.Outcome.Receipt) > MaxReviewReceiptBytes {
		return errors.New("review operation outcome exceeds bounds")
	}
	if op.Outcome.Effect == OpEffectCommitted && (op.Outcome.InstallationID == "" || len(op.Outcome.Receipt) == 0 || op.Outcome.Completion == "") {
		return errors.New("committed review lacks its receipt or completion")
	}
	tombstone := op.Outcome.OperationID != "" && op.Outcome.Kind == "" && op.Outcome.ActorID == 0 && op.Outcome.RepositoryID == 0 &&
		op.Outcome.Effect == OpEffectNotCommitted && op.Outcome.Cancellation == OpCancelCancelled
	if op.Outcome.OperationID != "" && !tombstone && (op.Outcome.InstallationID == "" || op.Outcome.Kind != OpReviewSubmit || op.Outcome.ActorID != op.Work.ActorID || op.Outcome.RepositoryID != op.Work.Repository) {
		return errors.New("review operation outcome scope differs")
	}
	return nil
}

// WithOutcome adopts one native lookup/cancel/submit response without
// permitting the stable review identity or a terminal result to move.
func (op ReviewOperation) WithOutcome(outcome OperationOutcome) (ReviewOperation, error) {
	if outcome.NotObserved || outcome.OperationID != op.Work.OperationID {
		return ReviewOperation{}, errors.New("review outcome identity differs")
	}
	tombstone := outcome.Kind == "" && outcome.ActorID == 0 && outcome.RepositoryID == 0 &&
		outcome.Effect == OpEffectNotCommitted && outcome.Cancellation == OpCancelCancelled
	if !tombstone && (outcome.InstallationID == "" || outcome.Kind != OpReviewSubmit ||
		outcome.ActorID != op.Work.ActorID || outcome.RepositoryID != op.Work.Repository) {
		return ReviewOperation{}, errors.New("review outcome scope differs")
	}
	prior := op.Outcome
	if (prior.Effect == OpEffectCommitted || prior.Effect == OpEffectNotCommitted) && prior.Effect != outcome.Effect {
		return ReviewOperation{}, errors.New("review terminal effect changed")
	}
	if prior.InstallationID != "" && prior.InstallationID != outcome.InstallationID {
		return ReviewOperation{}, errors.New("review installation identity changed")
	}
	if len(prior.Receipt) > 0 && string(prior.Receipt) != string(outcome.Receipt) {
		return ReviewOperation{}, errors.New("review receipt changed")
	}
	op.Outcome = outcome
	return op, nil
}

// ValidateTarget checks the exact target before findings or operation identity
// exist. The native actor must be separate from the recorded PR author.
func (w ReviewWork) ValidateTarget() error {
	if w.Repository <= 0 || w.ActorID <= 0 || w.PRNumber <= 0 || w.PRID <= 0 || w.IssueID <= 0 || w.PRAuthorID <= 0 || w.ActorID == w.PRAuthorID {
		return errors.New("invalid independent review target")
	}
	if !ValidTargetBranch(w.HeadRef) || !ValidTargetBranch(w.BaseRef) || w.HeadRef == w.BaseRef || !ValidCommit(w.HeadOID) || !ValidCommit(w.BaseOID) {
		return errors.New("invalid exact review candidate")
	}
	return nil
}

// Validate checks persisted semantic inputs. Deadline freshness belongs to the
// caller at dispatch; expired historical work remains valid for reconciliation.
func (w ReviewWork) Validate() error {
	if err := w.ValidateTarget(); err != nil {
		return err
	}
	if !ValidPublicationOperationID(w.OperationID) || strings.TrimSpace(w.AuthRevision) == "" || len(w.AuthRevision) > 512 || w.NativeRev < 1 || w.NotAfter <= 0 || len(w.Body) > 65536 {
		return errors.New("invalid review authorization")
	}
	switch w.Event {
	case "APPROVED":
	case "REQUEST_CHANGES":
		if strings.TrimSpace(w.Body) == "" {
			return errors.New("requested changes need findings")
		}
	default:
		return errors.New("invalid review disposition")
	}
	return nil
}

// ReviewObservation reports the exact current native target after equal idle
// revision observations. Native review history is checked in that same bracket.
type ReviewObservation struct {
	NativeRev    int64  `json:"native_revision"`
	ObservedUnix int64  `json:"observed_unix"`
	PRID         int64  `json:"pr_id"`
	PRNumber     int64  `json:"pr_number"`
	IssueID      int64  `json:"issue_id"`
	PRAuthorID   int64  `json:"pr_author_id"`
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
}

// ReviewAuthRevision binds one submitted review to its assignment and run:
// the opaque native authorization revision, never a credential.
func ReviewAuthRevision(assignmentID, runID string) string {
	return "soda-assignment:" + assignmentID + ":review-run:" + runID
}

// ReviewReportFence delimits the reviewer report block: the review prompt
// instructs the CLI to close its work with exactly one fenced review-json
// block, and the supervisor parses the last such block strictly.
const ReviewReportFence = "review-json"

// ReviewReport is one reviewer agent's genuine verdict on an exact
// candidate: approve it or request changes with concrete findings.
type ReviewReport struct {
	Verdict  string   `json:"verdict"`
	Summary  string   `json:"summary"`
	Body     string   `json:"body"`
	Findings []string `json:"findings"`
}

// Validate checks the verdict shape. Approvals carry a summary; requested
// changes carry findings in the body, never an empty report.
func (r ReviewReport) Validate() error {
	switch r.Verdict {
	case "approve":
	case "request-changes":
		if strings.TrimSpace(r.Body) == "" {
			return errors.New("requested changes need findings")
		}
	default:
		return errors.New("invalid review verdict")
	}
	if len(r.Body) > 65536 || len(r.Summary) > 4096 {
		return errors.New("review report exceeds its bound")
	}
	if len(r.Findings) > 64 {
		return errors.New("review report has too many findings")
	}
	for _, finding := range r.Findings {
		if len(finding) > 4096 {
			return errors.New("review finding is too large")
		}
	}
	return nil
}

// ParseReviewReport extracts the last fenced review block from reviewer
// output and validates it strictly. Unknown fields refuse; anything
// unparseable reports false instead of guessing a verdict.
func ParseReviewReport(output string) (ReviewReport, bool) {
	start := strings.LastIndex(output, "```"+ReviewReportFence)
	if start < 0 {
		return ReviewReport{}, false
	}
	rest := output[start+len("```"+ReviewReportFence):]
	// The JSON payload may itself contain nested fences (quoted diffs,
	// code samples), so the first fence after the opener is unreliable:
	// try every closing fence in order until one decodes. The first
	// valid payload wins; trailing chatter never mis-terminates it.
	for offset := 0; offset < len(rest); {
		rel := strings.Index(rest[offset:], "```")
		if rel < 0 {
			return ReviewReport{}, false
		}
		body := strings.TrimSpace(rest[:offset+rel])
		offset += rel + len("```")
		if body == "" || len(body) > 65536+8192 {
			continue
		}
		var report ReviewReport
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&report); err != nil {
			continue
		}
		if report.Validate() != nil {
			continue
		}
		return report, true
	}
	return ReviewReport{}, false
}

// Frozen D04 review/finding bounds (FW06C). At most MaxFindingsPerReport
// findings per report and MaxOutstandingFindings outstanding carried
// findings; one finding's combined defect/evidence/outcome prose fits
// MaxFindingProseBytes; the encoded native review payload fits
// MaxReviewPayloadBytes, enforced on the native operation by FW16 and on
// the Soda adapter by FW18 against this same ceiling; one review-specific
// native receipt fits MaxReviewReceiptBytes. The 1 MiB review-history cap
// (MaxPublicationReviewBytes) counts full encoded work plus reserved
// receipt capacity at the review-specific size. Other operation receipt
// limits stay unchanged. The N adapter moves to the review-specific
// receipt cap with FW18; until then it still admits only 4 KiB receipts.
const (
	MaxFindingsPerReport   = 64
	MaxOutstandingFindings = 64
	MaxFindingProseBytes   = 4096
	MaxReviewPayloadBytes  = 48 << 10
	MaxReviewReceiptBytes  = 16 << 10
)

// Finding severities and fix kinds. Fix verification covers checks-only
// outcomes; informational findings use verification with an explicit
// no-action required outcome, never an empty field.
const (
	FindingSeverityCritical = "critical"
	FindingSeverityMajor    = "major"
	FindingSeverityMinor    = "minor"
	FindingFixCode          = "code"
	FindingFixVerification  = "verification"
)

// ValidFindingKey reports whether key is a stable 32-hex finding identity.
// Keys share the factory identity alphabet; they are preserved through
// review rounds, and new defects get new keys.
func ValidFindingKey(key string) bool { return ValidID(key) }

// FindingLocation is one finding's optional native anchor: diff path,
// old|new side and one-based actual diff line.
type FindingLocation struct {
	Path string `json:"path"`
	Side string `json:"side"`
	Line int64  `json:"line"`
}

// Validate checks the anchor shape. Non-line findings stay unlocated and
// must be represented in the review summary instead.
func (l FindingLocation) Validate() error {
	if l.Path == "" || len(l.Path) > 1024 || l.Path != strings.TrimSpace(l.Path) {
		return errors.New("invalid finding location path")
	}
	switch l.Side {
	case "old", "new":
	default:
		return errors.New("invalid finding location side")
	}
	if l.Line < 1 {
		return errors.New("invalid finding location line")
	}
	return nil
}

// Finding is one Soda keyed finding (D06): stable key, origin review run
// and native review, severity, actionability, defect, evidence, required
// outcome, fix kind and optional native location. Soda severity,
// dispositions and Jev state never enter Fountain policy; the native
// comment carries the key plus its prose.
type Finding struct {
	Location        *FindingLocation `json:"location,omitempty"`
	Key             string           `json:"key"`
	OriginRun       string           `json:"origin_run"`
	Severity        string           `json:"severity"`
	Defect          string           `json:"defect"`
	Evidence        string           `json:"evidence"`
	RequiredOutcome string           `json:"required_outcome"`
	FixKind         string           `json:"fix_kind"`
	OriginReview    int64            `json:"origin_review,omitempty"`
	Actionable      bool             `json:"actionable"`
}

// Validate checks the complete finding shape. Every field except the
// location and the not-yet-natively-recorded origin review is required;
// informational findings record their rationale as evidence with an
// explicit no-action outcome.
func (f Finding) Validate() error {
	if !ValidFindingKey(f.Key) {
		return errors.New("invalid finding key")
	}
	if !ValidID(f.OriginRun) {
		return errors.New("invalid finding origin run")
	}
	if f.OriginReview < 0 {
		return errors.New("invalid finding origin review")
	}
	switch f.Severity {
	case FindingSeverityCritical, FindingSeverityMajor, FindingSeverityMinor:
	default:
		return errors.New("invalid finding severity")
	}
	if strings.TrimSpace(f.Defect) == "" || strings.TrimSpace(f.Evidence) == "" || strings.TrimSpace(f.RequiredOutcome) == "" {
		return errors.New("finding lacks its defect, evidence or required outcome")
	}
	if len(f.Defect)+len(f.Evidence)+len(f.RequiredOutcome) > MaxFindingProseBytes {
		return errors.New("finding prose exceeds its bound")
	}
	switch f.FixKind {
	case FindingFixCode, FindingFixVerification:
	default:
		return errors.New("invalid finding fix kind")
	}
	if f.Location != nil {
		if err := f.Location.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// FindingSetDigest binds one finding set to its sorted keys: the input
// digest carried by review cycles and the exact digest covered by
// correction dispositions. Key order never affects the digest; any added,
// dropped or re-keyed finding changes it.
func FindingSetDigest(keys []string) (string, error) {
	if len(keys) > MaxOutstandingFindings {
		return "", errors.New("finding set exceeds its bound")
	}
	sorted := append([]string(nil), keys...)
	for _, key := range sorted {
		if !ValidFindingKey(key) {
			return "", errors.New("finding set carries an invalid key")
		}
	}
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return "", errors.New("finding set carries a duplicate key")
		}
	}
	sum := sha256.Sum256([]byte("finding-set-v1\x00" + strings.Join(sorted, "\x00")))
	return hex.EncodeToString(sum[:]), nil
}

// ResolveConversationMember is one thread member behind a resolution
// digest: member ID, body, content version, location and resolver state,
// in native (created_unix, id) order with the anchored root first.
type ResolveConversationMember struct {
	Body          string
	Location      string
	ResolverState string
	ID            int64
	ContentVer    int64
}

// ResolveConversationLine renders one member in the canonical digest
// encoding shared with the X SDK twin: decimal ID, body, decimal content
// version, location and resolver state joined by NUL bytes. Both sides
// must reproduce these bytes exactly; the digest comparison refuses any
// divergence as changed content.
func ResolveConversationLine(m ResolveConversationMember) string {
	return strconv.FormatInt(m.ID, 10) + "\x00" + m.Body + "\x00" +
		strconv.FormatInt(m.ContentVer, 10) + "\x00" + m.Location + "\x00" + m.ResolverState
}

// ConversationDigest binds the expected complete conversation behind a
// resolve request: member count plus the ordered member lines. Any reply,
// edit, unresolve or resolver-state change alters the digest and refuses
// the stale resolution. Review snapshots (FW17) record this digest;
// resolution (FW17) recomputes it from live native state.
func ConversationDigest(lines []string) (string, error) {
	if len(lines) == 0 {
		return "", errors.New("conversation digest needs its members")
	}
	sum := sha256.Sum256([]byte(strconv.Itoa(len(lines)) + "\n" + strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

// ThreadEvidence is the recorded conversation a resolve request must
// match: repository, PR, anchored root comment, complete digest and
// whether the root mapping was deleted. Deletion invalidates the mapping
// and cannot silently transfer accepted resolution to another root.
type ThreadEvidence struct {
	Digest        string
	Repository    int64
	PRNumber      int64
	RootCommentID int64
	Deleted       bool
}

// ResolveRequest is one conditional thread-resolve intent (D06): exact
// PR/head/base, anchored root comment, expected complete conversation
// digest and resolved=true. Replies are members, never resolution
// targets; only resolution is supported, never unresolve via this
// operation.
type ResolveRequest struct {
	HeadRef       string `json:"head_ref"`
	BaseRef       string `json:"base_ref"`
	HeadOID       string `json:"head_oid"`
	BaseOID       string `json:"base_oid"`
	Digest        string `json:"digest"`
	Repository    int64  `json:"repository"`
	PRNumber      int64  `json:"pr_number"`
	PRID          int64  `json:"pr_id"`
	RootCommentID int64  `json:"root_comment_id"`
	ActorID       int64  `json:"actor_id"`
	Resolved      bool   `json:"resolved"`
}

// ValidateForEvidence checks the resolve intent against its enrolled
// binding, reviewer independence and recorded thread evidence. A missing
// or wrong-kind binding, a wrong actor, the PR author's own identity, a
// reply passed as root, a deleted root or changed content all refuse;
// exact replay against unchanged evidence validates.
func (r ResolveRequest) ValidateForEvidence(binding ActorBindingRef, prAuthorID int64, evidence ThreadEvidence) error {
	if binding == (ActorBindingRef{}) || binding.Kind != OpThreadResolve {
		return errors.New("missing resolve binding")
	}
	if err := binding.Validate(); err != nil {
		return errors.New("invalid resolve binding")
	}
	if r.ActorID <= 0 || r.ActorID != binding.ActorID {
		return errors.New("resolve actor differs from its binding")
	}
	if prAuthorID <= 0 || r.ActorID == prAuthorID {
		return errors.New("resolve actor needs its independent reviewer")
	}
	if r.Repository <= 0 || r.PRNumber <= 0 || r.PRID <= 0 || r.RootCommentID <= 0 {
		return errors.New("invalid resolve target")
	}
	if !ValidTargetBranch(r.HeadRef) || !ValidTargetBranch(r.BaseRef) || r.HeadRef == r.BaseRef {
		return errors.New("invalid resolve refs")
	}
	if !ValidCommit(r.HeadOID) || !ValidCommit(r.BaseOID) || r.HeadOID == r.BaseOID {
		return errors.New("resolve candidate is not a fresh exact commit")
	}
	if !r.Resolved {
		return errors.New("resolve request must resolve")
	}
	if !ValidDigest(r.Digest) {
		return errors.New("invalid resolve conversation digest")
	}
	if evidence.Deleted {
		return errors.New("resolve root deleted")
	}
	if evidence.Repository != r.Repository || evidence.PRNumber != r.PRNumber || evidence.RootCommentID != r.RootCommentID {
		return errors.New("resolve target differs from its recorded root")
	}
	if !ValidDigest(evidence.Digest) || evidence.Digest != r.Digest {
		return errors.New("resolve conversation changed under its request")
	}
	return nil
}

// ReviewCycleID derives the deterministic review-cycle identity from
// publication, attempt root, round and phase. New rounds on the same head
// are distinct; replay of a round reuses its identity.
func ReviewCycleID(publicationID, attemptRoot string, round int, phase string) string {
	sum := sha256.Sum256([]byte("review-cycle-v1\x00" + publicationID + "\x00" + attemptRoot + "\x00" +
		strconv.Itoa(round) + "\x00" + phase))
	return hex.EncodeToString(sum[:])[:32]
}

// ReviewOutcome is the attributable native review identity adopted from its
// operation's committed receipt. It is historical evidence for this exact head
// and base; callers must re-observe before using it for a later decision.
type ReviewOutcome struct {
	Operation  OperationOutcome
	ReviewID   int64
	CommentID  int64
	ReviewerID int64
	PRID       int64
	PRNumber   int64
	IssueID    int64
	HeadOID    string
	BaseOID    string
	CommitID   string
	Event      string
}
