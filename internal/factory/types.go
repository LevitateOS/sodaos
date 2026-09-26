// Package factory owns bounded software-work identities and lifecycle policy.
// It does not execute containers, persist rows or duplicate Forgejo collaboration.
package factory

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

type Role string

const (
	Implementation Role = "implementation"
	Review         Role = "review"
	Repair         Role = "repair"
)

type Outcome string

const (
	Succeeded  Outcome = "succeeded"
	Failed     Outcome = "failed"
	Cancelled  Outcome = "cancelled"
	NeedsHuman Outcome = "needs-human"
)

type Phase string

const (
	Implement Phase = "implementation"
	Verify    Phase = "verification"
	Fix       Phase = "repair"
	Reverify  Phase = "final-verification"
	Finished  Phase = "finished"
)

var (
	identifier = regexp.MustCompile(`^[a-f0-9]{32}$`)
	commit     = regexp.MustCompile(`^[a-f0-9]{40}$`)
	delivery   = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
)

func NewID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw[:])
}
func ValidID(id string) bool     { return identifier.MatchString(id) }
func ValidCommit(id string) bool { return commit.MatchString(id) }

// WorkItem records the objective and revisions admitted by a human. It is not
// a mirror of the issue's discussion or a credential-bearing request.
type WorkItem struct {
	RepositoryID int64  `json:"repository_id"`
	Issue        int64  `json:"issue"`
	HumanID      int64  `json:"human_id"`
	Objective    string `json:"objective"`
	BaseSHA      string `json:"base_sha"`
	PolicySHA    string `json:"policy_sha"`
}

// Attempt is one admission of a WorkItem. Each agent execution is a separate Run.
// Cleanup is deliberately independent of the terminal software-work outcome.
type Attempt struct {
	ID              string    `json:"id"`
	Delivery        string    `json:"delivery"`
	Work            WorkItem  `json:"work"`
	Admitted        time.Time `json:"admitted"`
	Deadline        time.Time `json:"deadline"`
	Phase           Phase     `json:"phase"`
	Outcome         Outcome   `json:"outcome,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Candidate       string    `json:"candidate,omitempty"`
	Pull            int64     `json:"pull,omitempty"`
	Executions      int       `json:"executions"`
	CIEvaluations   int       `json:"ci_evaluations"`
	CI              *Evidence `json:"ci,omitempty"`
	Review          *Evidence `json:"review,omitempty"`
	CleanupComplete bool      `json:"cleanup_complete"`
	Revision        int64     `json:"revision"`
}

type Evidence struct {
	ID     string `json:"id"`
	Commit string `json:"commit"`
	Passed bool   `json:"passed"`
}

// Resource intent is recorded before provisioning. Restart reconciliation uses
// these exact names/IDs plus the run ownership label; it never scans for intent.
type Resource struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type Run struct {
	CredentialClaimed bool       `json:"credential_claimed"`
	ID                string     `json:"id"`
	AttemptID         string     `json:"attempt_id"`
	Role              Role       `json:"role"`
	InputSHA          string     `json:"input_sha"`
	Started           time.Time  `json:"started"`
	Deadline          time.Time  `json:"deadline"`
	Outcome           Outcome    `json:"outcome,omitempty"`
	Summary           string     `json:"summary,omitempty"`
	Resources         []Resource `json:"resources"`
	CleanupComplete   bool       `json:"cleanup_complete"`
	Image             string     `json:"image"`
	Harness           string     `json:"harness"`
	Model             string     `json:"model"`
}

func New(work WorkItem, event string, now time.Time) (Attempt, error) {
	a := Attempt{ID: NewID(), Delivery: event, Work: work, Admitted: now, Deadline: now.Add(3 * time.Hour), Phase: Implement, CleanupComplete: true}
	return a, a.Validate()
}

func (w WorkItem) Validate() error {
	if w.RepositoryID <= 0 || w.Issue <= 0 || w.HumanID <= 0 {
		return errors.New("invalid human work identity")
	}
	if len(w.Objective) == 0 || len(w.Objective) > 64<<10 {
		return errors.New("invalid work objective")
	}
	if !ValidCommit(w.BaseSHA) || !ValidDigest(w.PolicySHA) {
		return errors.New("invalid admitted source or policy revision")
	}
	return nil
}

func (a Attempt) Validate() error {
	if !ValidID(a.ID) || !delivery.MatchString(a.Delivery) {
		return errors.New("invalid factory admission identity")
	}
	if err := a.Work.Validate(); err != nil {
		return err
	}
	if err := a.validateDeadline(); err != nil {
		return err
	}
	if err := a.validateLimits(); err != nil {
		return err
	}
	if a.Candidate != "" && !ValidCommit(a.Candidate) {
		return errors.New("invalid candidate commit")
	}
	return a.validatePhase()
}

func (a Attempt) validateDeadline() error {
	if a.Admitted.IsZero() || !a.Deadline.After(a.Admitted) || a.Deadline.Sub(a.Admitted) > 3*time.Hour {
		return errors.New("invalid attempt deadline")
	}
	return nil
}

func (a Attempt) validateLimits() error {
	if a.Executions < 0 || a.Executions > 4 || a.CIEvaluations < 0 || a.CIEvaluations > 2 || len(a.Summary) > 16<<10 || a.Pull < 0 {
		return errors.New("invalid factory limits")
	}
	return nil
}

func (a Attempt) validatePhase() error {
	switch a.Phase {
	case Implement, Verify, Fix, Reverify:
		if a.Outcome != "" {
			return errors.New("active attempt has terminal outcome")
		}
	case Finished:
		if a.Outcome == "" || !validOutcome(a.Outcome) {
			return errors.New("invalid terminal outcome")
		}
	default:
		return errors.New("invalid factory phase")
	}
	return nil
}
