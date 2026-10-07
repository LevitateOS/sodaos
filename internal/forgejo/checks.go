package forgejo

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// CheckAssessor observes native check evidence for one exact candidate
// through the shared background admission (ST11, S-checks). One bracketed
// pull/checks/refs read binds the PR linkage, the latest status per
// context on the exact head, and both ref tips; factory.VerifyChecks
// owns the verdict, including stale-head/base refusals, so every outcome
// stays a recorded assessment. The assessor never schedules CI, manages
// runners, or touches workflow definitions: it reads observed native
// state only. Its actor secret stays outside the Project and its agents.
type CheckAssessor struct {
	background *ServiceBackground
	rest       *Client
	tokenFile  string
}

// NewCheckAssessor builds the check observation client over the shared
// service background, the internal REST client and the assessor's
// restricted token file.
func NewCheckAssessor(background *ServiceBackground, rest *Client, tokenFile string) *CheckAssessor {
	return &CheckAssessor{background: background, rest: rest, tokenFile: tokenFile}
}

func (a *CheckAssessor) checkActor(ctx context.Context, actor int64) error {
	if a.background == nil || a.rest == nil || a.tokenFile == "" {
		return &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	token, err := config.Secret(a.tokenFile)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	user, err := a.rest.Current(ctx, token)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	if user.ID != actor {
		return &factory.PublicationWait{Reason: "authority_lost"}
	}
	return nil
}

// ObserveChecks reads the PR linkage, exact-head statuses and both ref
// tips in one bracket. Tip mismatches stay in the returned observation
// for the verdict; only a misidentified PR or an unreadable bracket
// refuses or waits without recording.
func (a *CheckAssessor) ObserveChecks(ctx context.Context, target factory.CheckTarget, actorID int64) (factory.ObservedChecks, error) {
	var empty factory.ObservedChecks
	if err := target.Validate(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if actorID <= 0 {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if err := a.checkActor(ctx, actorID); err != nil {
		return empty, err
	}
	reader := &BackgroundSnapshotReader{Client: a.background, ActorID: strconv.FormatInt(actorID, 10)}
	snapshot, err := BracketedRead(ctx, reader, extensions.CredentialFile(a.tokenFile), SnapshotRequest{
		RepositoryID: strconv.FormatInt(target.Repository, 10),
		PullNumber:   strconv.FormatInt(target.PRNumber, 10),
		SHA:          target.HeadOID,
		Refs:         []string{target.HeadRef, target.BaseRef},
		Families:     []SnapshotFamily{FamilyPull, FamilyChecks, FamilyRefs},
	})
	if err != nil {
		return empty, checkReadError(err)
	}
	return matchCheckTarget(target, snapshot)
}

func checkReadError(err error) error {
	if errors.Is(err, ErrNativeBusy) {
		return &factory.PublicationWait{Reason: "native_busy"}
	}
	if errors.Is(err, ErrStaleSnapshot) {
		return &factory.PublicationWait{Reason: "revision_moved"}
	}
	// A busy host refuses snapshot reads with 503 while native writers
	// settle; the attempt waits for the resource instead of weakening
	// verification. Any other failure stays unavailable, never a guess.
	var status *forgejopublish.StatusError
	if errors.As(err, &status) && status.Status == http.StatusServiceUnavailable {
		return &factory.PublicationWait{Reason: "native_busy"}
	}
	return &factory.PublicationWait{Reason: "checks_unavailable"}
}

// matchCheckTarget verifies the PR linkage and maps the observation. The
// latest status per context wins, ordered by native update stamp then
// native ID, matching combined-status semantics; hidden check items or
// hidden head/base refs mark the observation hidden for the verdict.
func matchCheckTarget(target factory.CheckTarget, snapshot NativeSnapshot) (factory.ObservedChecks, error) {
	var empty factory.ObservedChecks
	pull := snapshot.Pull
	if pull == nil || !pull.Visible || pull.ID != strconv.FormatInt(target.PRID, 10) ||
		pull.Number != strconv.FormatInt(target.PRNumber, 10) || pull.IssueID != strconv.FormatInt(target.IssueID, 10) ||
		pull.HeadRepoID != strconv.FormatInt(target.Repository, 10) ||
		pull.HeadBranch != factory.RefHead(target.HeadRef) || pull.BaseBranch != factory.RefHead(target.BaseRef) {
		return empty, &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	if pull.HasMerged {
		return empty, &factory.PublicationRefusal{Reason: "pr_merged"}
	}
	if snapshot.Checks == nil {
		return empty, &factory.PublicationWait{Reason: "checks_unavailable"}
	}
	observed := factory.ObservedChecks{NativeRev: snapshot.Revision, Complete: true}
	for _, ref := range snapshot.Refs {
		switch ref.Ref {
		case target.HeadRef:
			if !ref.Visible {
				observed.Hidden = true
				continue
			}
			if ref.Exists {
				observed.HeadTip = ref.OID
			}
		case target.BaseRef:
			if !ref.Visible {
				observed.Hidden = true
				continue
			}
			if ref.Exists {
				observed.BaseTip = ref.OID
			}
		}
	}
	type latest struct {
		state   string
		updated int64
		id      int64
	}
	winners := make(map[string]latest)
	contexts := make(map[string]bool)
	for _, item := range snapshot.Checks.Items {
		if !item.Visible {
			observed.Hidden = true
			continue
		}
		contexts[item.Context] = true
		id, err := strconv.ParseInt(item.ID, 10, 64)
		if err != nil || id <= 0 {
			return empty, &factory.PublicationWait{Reason: "checks_unavailable"}
		}
		prior, seen := winners[item.Context]
		if !seen || item.UpdatedUnix > prior.updated || item.UpdatedUnix == prior.updated && id > prior.id {
			winners[item.Context] = latest{state: item.State, updated: item.UpdatedUnix, id: id}
		}
	}
	observed.ObservedContexts = len(contexts)
	for context, winner := range winners {
		observed.Checks = append(observed.Checks, factory.ObservedCheck{Context: context, State: winner.state})
	}
	sort.Slice(observed.Checks, func(i, j int) bool { return observed.Checks[i].Context < observed.Checks[j].Context })
	if err := observed.Validate(); err != nil {
		return empty, &factory.PublicationWait{Reason: "checks_unavailable"}
	}
	return observed, nil
}
