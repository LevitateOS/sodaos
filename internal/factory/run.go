package factory

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidDigest(value string) bool { return digest.MatchString(value) }

func (r Run) Validate() error {
	if !ValidID(r.ID) || !ValidID(r.AttemptID) || !ValidCommit(r.InputSHA) {
		return errors.New("invalid factory execution identity")
	}
	if err := r.validateDeadline(); err != nil {
		return err
	}
	if err := r.validateProvenance(); err != nil {
		return err
	}
	if err := r.validateOutcome(); err != nil {
		return err
	}
	if err := r.validateCredentials(); err != nil {
		return err
	}
	return r.validateResources()
}

func (r Run) validateDeadline() error {
	limit := 30 * time.Minute
	switch r.Role {
	case Implementation:
		limit = 90 * time.Minute
	case Review, Repair:
	default:
		return errors.New("invalid factory role")
	}
	if r.Started.IsZero() || !r.Deadline.After(r.Started) || r.Deadline.Sub(r.Started) > limit {
		return errors.New("invalid execution deadline")
	}
	return nil
}

func (r Run) validateProvenance() error {
	value, pinned := strings.CutPrefix(r.Image, "sha256:")
	if !pinned || !ValidDigest(value) {
		return errors.New("execution image is not pinned")
	}
	if r.Harness == "" || r.Model == "" || len(r.Harness) > 128 || len(r.Model) > 128 {
		return errors.New("execution harness/model provenance is incomplete")
	}
	return nil
}

func (r Run) validateOutcome() error {
	if !validOutcome(r.Outcome) || len(r.Summary) > 16<<10 || (r.Outcome == "" && r.CleanupComplete) {
		return errors.New("invalid execution outcome")
	}
	return nil
}

func (r Run) Authority(attemptID string, now time.Time) error {
	if r.AttemptID != attemptID || r.Outcome != "" || !now.Before(r.Deadline) {
		return errors.New("run authority is inactive or expired")
	}
	return nil
}

func (r Run) reviewEvidence(attemptID, sha string) (Evidence, error) {
	if r.Role != Review || r.AttemptID != attemptID || r.InputSHA != sha || r.Outcome != Succeeded || !r.CleanupComplete || !ValidID(r.ID) {
		return Evidence{}, errors.New("review does not bind to a fresh completed candidate run")
	}
	return Evidence{ID: r.ID, Commit: r.InputSHA}, nil
}

func validOutcome(outcome Outcome) bool {
	return outcome == "" || outcome == Succeeded || outcome == Failed || outcome == Cancelled || outcome == NeedsHuman
}

func (r Run) validateResources() error {
	if len(r.Resources) > 3 {
		return errors.New("too many workspace resources")
	}
	seen := map[string]bool{}
	for _, resource := range r.Resources {
		if seen[resource.Kind] || resource.Name != ResourceName(r.ID, resource.Kind) {
			return errors.New("resource is not owned by this run")
		}
		switch resource.Kind {
		case "workspace", "proxy", "network":
		default:
			return errors.New("unknown workspace resource kind")
		}
		if resource.ID != "" && !ValidDigest(resource.ID) {
			return errors.New("invalid resource ID")
		}
		seen[resource.Kind] = true
	}
	return nil
}

func ResourceName(runID, kind string) string { return "soda-factory-" + kind + "-" + runID }

func (r Run) validateCredentials() error {
	if r.CredentialReturned && !r.CredentialDelegated {
		return errors.New("credential return has no delegation")
	}
	if r.CredentialDelegated && (!r.CredentialClaimed || !ValidDigest(r.CredentialSeedSHA)) {
		return errors.New("invalid credential delegation")
	}
	return nil
}
