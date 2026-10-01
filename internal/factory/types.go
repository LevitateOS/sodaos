// Package factory owns supervised software-work run records and operator
// command validation. It does not execute containers, persist rows or
// duplicate Forgejo collaboration. ST02 removed the manual attempt admission
// and fixed repair/merge lifecycle; runs stand alone, keyed by execution ID.
package factory

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"

	"github.com/levitateos/sodaos/internal/project"
)

type Outcome string

const (
	Succeeded  Outcome = "succeeded"
	Failed     Outcome = "failed"
	Cancelled  Outcome = "cancelled"
	NeedsHuman Outcome = "needs-human"
)

var (
	identifier = regexp.MustCompile(`^[a-f0-9]{32}$`)
	commit     = regexp.MustCompile(`^[a-f0-9]{40}$`)
	role       = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
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

// ValidProjectID reuses the canonical Project identity; factory records never
// invent a second Project DTO.
func ValidProjectID(id string) bool { return project.ValidID(id) }

func validOutcome(outcome Outcome) bool {
	return outcome == "" || outcome == Succeeded || outcome == Failed || outcome == Cancelled || outcome == NeedsHuman
}

// Command types served by the private operator endpoint. Status is a read;
// stop and reconcile are durable idempotent mutations.
const (
	CommandStatus    = "status"
	CommandStop      = "stop"
	CommandReconcile = "reconcile"
)

// Command is one admitted operator command. The client generates ID; the
// coordinator records the digest and durable outcome. Same ID with a changed
// payload conflicts instead of executing twice. Commands cannot supply a
// human native identity, accept requirements or change policy.
type Command struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Target    string `json:"target,omitempty"`
	Principal string `json:"principal"`
	Digest    string `json:"digest"`
	Outcome   string `json:"outcome,omitempty"`
	Created   string `json:"created,omitempty"`
	Finished  string `json:"finished,omitempty"`
}

func (c Command) Validate() error {
	if !ValidID(c.ID) {
		return errors.New("invalid command identity")
	}
	switch c.Type {
	case CommandStop:
		if !ValidID(c.Target) {
			return errors.New("stop requires its recorded run")
		}
	case CommandReconcile:
		if c.Target != "" {
			return errors.New("reconcile addresses all recorded runs")
		}
	default:
		return errors.New("command is not a durable operator mutation")
	}
	if c.Principal == "" || len(c.Principal) > 128 {
		return errors.New("invalid command principal")
	}
	if c.Digest != CommandDigest(c.Type, c.Target) {
		return errors.New("command digest differs from its payload")
	}
	if len(c.Outcome) > 64<<10 {
		return errors.New("command outcome exceeds retained output limit")
	}
	return nil
}

// CommandDigest binds a command ID to its exact payload. Status reads carry
// no durable command and never use this digest.
func CommandDigest(typ, target string) string {
	sum := sha256.Sum256([]byte(typ + "\x00" + target))
	return hex.EncodeToString(sum[:])
}
