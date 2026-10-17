package api

import (
	"context"
	"errors"
	"sort"
	"strconv"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// CompletionSnapshotSource adapts the thin F-read transport to the
// coordinator's completion inventory enumerator (D13/FW31): paged
// open-issue enumeration through the repository_issues family plus
// exact per-issue bodies through the existing individual issue and
// dependency families. One page brackets its enumeration between two
// equal idle revision observations; every body read must agree on
// that same revision, so a concurrent insertion, edit or permission
// change surfaces as a revision hold instead of a silently partial
// page. Closed, pull-request, hidden or digest-mismatched bodies
// refuse instead of sealing a subset. Payloads encode the canonical
// sealed issue substance the funded decision assesses.
// completionSourceStore is the NATIVE95 store authority behind one
// serving capture: the local-authority context plus the
// source-qualified local capture at a native tip. *store.Store
// satisfies it once NATIVE95 publishes the exports; the control
// package stays adapter-free so the interface lives here, not in
// control. No SourceInputs cross this seam in either direction.
type completionSourceStore interface {
	CompletionSourceContext(ctx context.Context, repository int64) (factory.CompletionAuthority, error)
	CaptureCompletionLocalAtSource(ctx context.Context, repository int64, expected factory.CompletionAuthority, tip string) (factory.CompletionLocal, error)
}

type CompletionSnapshotSource struct {
	Reader     forgejo.SnapshotReader
	Credential extensions.CredentialFile
	Store      completionSourceStore
}

var _ control.CompletionIssueEnumerator = (*CompletionSnapshotSource)(nil)

// NewCompletionSnapshotSource binds one service observer to the
// completion inventory enumerator behind the D13 completion drive.
func NewCompletionSnapshotSource(observer *forgejo.ServiceObserver) *CompletionSnapshotSource {
	if observer == nil {
		return &CompletionSnapshotSource{}
	}
	return &CompletionSnapshotSource{
		Reader: observer.SnapshotReader(), Credential: observer.Credential(),
	}
}

// completionPageReader is a tiny purpose-specific SnapshotReader
// wrapper that charges actual completion-page native calls one EACH
// at the ReadNativeRevision/ReadSnapshot boundary against the
// existing readiness-pass budget. The reservation charges the
// attempt itself, so denied or transiently failed reads keep their
// charge: called work is never refunded and fictitious body reads
// are never charged. A nil budget (store-free fixtures) carries no
// charge.
//
// Ownership: Source.ObserveRevision calls the raw Reader directly
// and stays charged BY ITS CONTROL OWNER — the outer bracket's
// upfront-2 plus the collector's upfront-2 and restart reservations
// — never by this wrapper. The wrapper covers page-bracket and
// body-bracket reads only; no generic forgejo, service or
// background transport changes.
type completionPageReader struct {
	inner forgejo.SnapshotReader
}

func (w completionPageReader) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	if err := control.ReserveCompletionPageCall(ctx); err != nil {
		return extensions.NativeRevisionObservation{}, err
	}
	return w.inner.ReadNativeRevision(ctx)
}

func (w completionPageReader) ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	if err := control.ReserveCompletionPageCall(ctx); err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	return w.inner.ReadSnapshot(ctx, credential, req)
}

// ObserveRevision observes the atomic native revision and idle/busy
// state backing the inventory bracket.
func (s *CompletionSnapshotSource) ObserveRevision(ctx context.Context) (int64, bool, error) {
	if s == nil || s.Reader == nil {
		return 0, false, forgejo.ErrUnavailable
	}
	observation, err := s.Reader.ReadNativeRevision(ctx)
	if err != nil {
		return 0, false, err
	}
	return observation.Revision, observation.Idle, nil
}

// EnumerateIssues reads one bounded open-issue page with exact bodies.
// Revision movement across the page bracket reports a revision hold
// for the collector's restart path; anything else that prevents an
// exact page reports unavailable or the exact producer hold.
func (s *CompletionSnapshotSource) EnumerateIssues(ctx context.Context, repository int64, cursor string) (control.NativeInventoryPage, error) {
	if s == nil || s.Reader == nil {
		return control.NativeInventoryPage{}, forgejo.ErrUnavailable
	}
	// The page budget bounds every page before any native call: at
	// most min(native 50, floor((remaining-3)/3)) items fit beside
	// the page's own 3-call bracket, with the fixed-8 bracket
	// prepayment (two outer observations plus both 3-call serving
	// captures) and the collector's own two already reserved, and
	// the closing 3 held here before the limit calc, never after
	// full collection. Below 1 the source holds pending before
	// spending a single call.
	requestLimit := control.CompletionPageItemLimit(ctx)
	if requestLimit < 1 {
		return control.NativeInventoryPage{}, control.ErrReadinessPassPending
	}
	repositoryID := strconv.FormatInt(repository, 10)
	reader := completionPageReader{inner: s.Reader}
	before, err := reader.ReadNativeRevision(ctx)
	if err != nil {
		return control.NativeInventoryPage{}, err
	}
	if before.Revision < 1 || !before.Idle {
		return control.NativeInventoryPage{}, errors.New(control.HoldInventoryRevision)
	}
	snapshot, err := reader.ReadSnapshot(ctx, s.Credential, forgejo.SnapshotRequest{
		RepositoryID: repositoryID,
		Families:     []forgejo.SnapshotFamily{forgejo.FamilyRepositoryIssues},
		Limit:        requestLimit, Cursor: cursor,
	})
	if err != nil {
		return control.NativeInventoryPage{}, err
	}
	after, err := reader.ReadNativeRevision(ctx)
	if err != nil {
		return control.NativeInventoryPage{}, err
	}
	if !after.Idle || after.Revision != before.Revision {
		return control.NativeInventoryPage{}, errors.New(control.HoldInventoryRevision)
	}
	if snapshot.RepositoryIssues == nil {
		return control.NativeInventoryPage{}, errors.New(control.HoldInventoryUnavailable)
	}
	// The producer must honor the requested bound: more items than
	// requested refuse instead of truncating silently, and an
	// incomplete prefix is never treated as terminal.
	if len(snapshot.RepositoryIssues.Items) > requestLimit {
		return control.NativeInventoryPage{}, errors.New(control.HoldInventoryPage)
	}
	if err := forgejo.ValidateRepositoryIssuesPage(snapshot.RepositoryIssues, repositoryID, cursor); err != nil {
		return control.NativeInventoryPage{}, errors.New(control.HoldInventoryPage)
	}
	issues := make([]control.NativeInventoryIssue, 0, len(snapshot.RepositoryIssues.Items))
	for _, item := range snapshot.RepositoryIssues.Items {
		body, err := s.readIssueBody(ctx, repositoryID, item, before.Revision)
		if err != nil {
			return control.NativeInventoryPage{}, err
		}
		data, err := body.CanonicalData()
		if err != nil {
			return control.NativeInventoryPage{}, errors.New(control.HoldInventoryPage)
		}
		if len(data) > factory.MaxReassessmentEvidenceBytes {
			return control.NativeInventoryPage{}, errors.New(control.HoldInventoryOversize)
		}
		index, err := strconv.ParseInt(item.IssueIndex, 10, 64)
		if err != nil || index < 1 || body.Index != index {
			return control.NativeInventoryPage{}, errors.New(control.HoldInventoryPage)
		}
		issues = append(issues, control.NativeInventoryIssue{Index: index, Data: data})
	}
	return control.NativeInventoryPage{
		Issues: issues, NextCursor: snapshot.RepositoryIssues.NextCursor,
		Revision: before.Revision, Complete: snapshot.RepositoryIssues.Complete, Idle: true,
	}, nil
}

// readIssueBody fetches one enumerated issue's exact title, body and
// dependency edges under the page's revision bracket. The bracketed
// read verifies digests against content; a body at another revision,
// a hidden body, or a closed or pull-request body refuses rather than
// sealing a subset or redacted content.
func (s *CompletionSnapshotSource) readIssueBody(ctx context.Context, repositoryID string, item forgejo.RepositoryIssueItem, revision int64) (factory.CompletionIssueEvidence, error) {
	snapshot, err := forgejo.BracketedRead(ctx, completionPageReader{inner: s.Reader}, s.Credential, forgejo.SnapshotRequest{
		RepositoryID: repositoryID, IssueIndex: item.IssueIndex,
		Families: []forgejo.SnapshotFamily{forgejo.FamilyIssue, forgejo.FamilyDependencies},
		Limit:    forgejo.SnapshotPageLimit,
	})
	if err != nil {
		switch {
		case errors.Is(err, forgejo.ErrStaleSnapshot), errors.Is(err, forgejo.ErrNativeBusy):
			return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryRevision)
		case errors.Is(err, forgejo.ErrIncompleteSnapshot):
			return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryUnavailable)
		default:
			return factory.CompletionIssueEvidence{}, err
		}
	}
	if snapshot.Revision != revision {
		return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryRevision)
	}
	if snapshot.Issue == nil || snapshot.Dependencies == nil {
		return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryUnavailable)
	}
	issue := snapshot.Issue
	if !issue.Visible || issue.IsClosed || issue.IsPull {
		return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryUnavailable)
	}
	index, err := strconv.ParseInt(issue.Index, 10, 64)
	if err != nil || index < 1 || issue.Index != item.IssueIndex {
		return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryPage)
	}
	dependencies := make([]string, 0, len(snapshot.Dependencies.Items))
	for _, edge := range snapshot.Dependencies.Items {
		if !edge.Visible || !edge.Complete || !factory.ValidDependencyRef(edge.DependencyID) {
			return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryUnavailable)
		}
		dependencies = append(dependencies, edge.DependencyID)
	}
	sort.Slice(dependencies, func(i, j int) bool {
		if len(dependencies[i]) != len(dependencies[j]) {
			return len(dependencies[i]) < len(dependencies[j])
		}
		return dependencies[i] < dependencies[j]
	})
	for i := 1; i < len(dependencies); i++ {
		if dependencies[i] == dependencies[i-1] {
			return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryPage)
		}
	}
	body := factory.CompletionIssueEvidence{
		Dependencies: dependencies, Index: index,
		Title: issue.Title, Body: issue.Content,
		TitleDigest: issue.TitleDigest, ContentDigest: issue.ContentDigest,
	}
	if err := body.Validate(); err != nil {
		return factory.CompletionIssueEvidence{}, errors.New(control.HoldInventoryPage)
	}
	return body, nil
}

// CaptureLocal captures the serving local half of D13 sealed evidence
// at the CURRENT native tip for the D13 completion drive. It composes
// (1) the local-authority context from the store, (2) exactly one
// bounded refs-only BracketedRead on that context's target branch, and
// (3) the source-qualified capture at the observed tip. The requested
// ref must be complete, visible, existing and a full OID; a hidden,
// missing, busy, malformed or moved ref refuses without any fallback
// to an unqualified local capture, and no SourceInputs are ever
// returned. The read goes through the RAW reader: the capture's native
// cost is at most 3 reads, prepaid by the controller's fixed-8
// reservation, never charged per call against the page counter.
func (s *CompletionSnapshotSource) CaptureLocal(ctx context.Context, repository int64) (factory.CompletionLocal, error) {
	if s == nil || s.Reader == nil || s.Store == nil {
		return factory.CompletionLocal{}, forgejo.ErrUnavailable
	}
	expected, err := s.Store.CompletionSourceContext(ctx, repository)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	if err := expected.Validate(); err != nil {
		return factory.CompletionLocal{}, err
	}
	snapshot, err := forgejo.BracketedRead(ctx, s.Reader, s.Credential, forgejo.SnapshotRequest{
		RepositoryID: strconv.FormatInt(repository, 10),
		Families:     []forgejo.SnapshotFamily{forgejo.FamilyRefs},
		Refs:         []string{expected.TargetBranch},
	})
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	var tip string
	for _, ref := range snapshot.Refs {
		if ref.Ref != expected.TargetBranch {
			continue
		}
		if !ref.Complete || !ref.Visible || !ref.Exists || !factory.ValidCommit(ref.OID) {
			return factory.CompletionLocal{}, forgejo.ErrIncompleteSnapshot
		}
		tip = ref.OID
		break
	}
	if tip == "" {
		return factory.CompletionLocal{}, forgejo.ErrIncompleteSnapshot
	}
	local, err := s.Store.CaptureCompletionLocalAtSource(ctx, repository, expected, tip)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	if err := local.Validate(); err != nil {
		return factory.CompletionLocal{}, err
	}
	return local, nil
}
