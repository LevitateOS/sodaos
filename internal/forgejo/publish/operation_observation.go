package publish

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ObserveForPublish brackets one idle native revision and reads the exact
// target and comparison tips behind it. The tips are observed between two
// equal idle observations; a moved or busy revision, a missing comparison
// or an ambiguous advertisement refuses instead of binding a stale read.
// An empty target tip means the branch is absent.
func (c Config) ObserveForPublish(ctx context.Context, ops BackgroundOperations, targetRef, comparisonRef string) (factory.PublicationObservation, error) {
	var empty factory.PublicationObservation
	if err := c.Validate(); err != nil {
		return empty, err
	}
	if !validPublishRef(targetRef) || !validPublishRef(comparisonRef) || targetRef == comparisonRef {
		return empty, &Refusal{Reason: "invalid_refs"}
	}
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	first, err := bracketRevision(ctx, ops)
	if err != nil {
		return empty, err
	}
	target, comparison, err := c.observeTips(ctx, targetRef, comparisonRef)
	if err != nil {
		return empty, err
	}
	second, err := ops.ReadNativeRevision(ctx)
	if err != nil {
		return empty, mapRevisionError(err)
	}
	if !second.Idle || second.Revision != first {
		return empty, &Wait{Reason: "revision_moved"}
	}
	return factory.PublicationObservation{
		TargetRef: targetRef, TargetTip: target, Comparison: comparison,
		NativeRev: first, ObservedUnix: time.Now().Unix(),
	}, nil
}

func validPublishRef(ref string) bool {
	name, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok || name == "" || len(ref) > 512 || ref != strings.TrimSpace(ref) {
		return false
	}
	return !strings.Contains(ref, "..") && !strings.ContainsAny(ref, " ~^:?*\\")
}

func bracketRevision(ctx context.Context, ops BackgroundOperations) (int64, error) {
	observation, err := ops.ReadNativeRevision(ctx)
	if err != nil {
		return 0, mapRevisionError(err)
	}
	if !observation.Idle {
		return 0, &Wait{Reason: "native_busy"}
	}
	if observation.Revision < 1 {
		return 0, errors.New("invalid native revision observation")
	}
	return observation.Revision, nil
}

func mapRevisionError(err error) error {
	var wait *Wait
	if errors.As(err, &wait) {
		return err
	}
	var refusal *Refusal
	if errors.As(err, &refusal) {
		return err
	}
	return err
}

func (c Config) observeTips(ctx context.Context, targetRef, comparisonRef string) (target, comparison string, err error) {
	git, cleanup, err := c.sourceRepository(ctx)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	target, err = observeTip(ctx, git, c.Remote, targetRef, true)
	if err != nil {
		return "", "", err
	}
	comparison, err = observeTip(ctx, git, c.Remote, comparisonRef, false)
	if err != nil {
		return "", "", err
	}
	return target, comparison, nil
}

// observeTip reads one exact advertised tip. A missing target reads as
// empty for branch creation; any other absence, ambiguity or malformed
// advertisement refuses.
func observeTip(ctx context.Context, git *repository, remote, ref string, absentOK bool) (string, error) {
	out, err := git.run(ctx, "ls-remote", remote, ref)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		if absentOK {
			return "", nil
		}
		return "", &Refusal{Reason: "comparison_missing"}
	}
	lines := strings.Split(text, "\n")
	if len(lines) != 1 {
		return "", &Refusal{Reason: "ambiguous_refs"}
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 2 || fields[1] != ref || !factory.ValidCommit(fields[0]) {
		return "", &Refusal{Reason: "ambiguous_refs"}
	}
	return fields[0], nil
}
