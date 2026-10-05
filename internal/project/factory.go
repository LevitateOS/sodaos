// Fixed supervised factory-run identities and wire records. This file is pure
// validation: no I/O, no SQL, no privilege. Run/source/assignment changes
// always require a new run identity; the same ID with different content
// conflicts instead of reusing approval. Only harnesses with their own M3
// proof are selectable; the executor refuses every other harness pin.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

const (
	// FactoryHarnessCodex is the first harness family. Each released
	// version needs its own proof before the executor admits its pin.
	FactoryHarnessCodex = "codex"
	// FactoryScopeCodex is the fixed factory execution discriminator carried
	// in the broker binding for supervised Codex runs.
	FactoryScopeCodex = "factory-codex"
	// FactoryHarnessMuse is the Muse Code CLI harness family
	// (`muse exec` runs backed by the native "muse" provider).
	FactoryHarnessMuse = "muse"
	// FactoryScopeMuse is the fixed factory execution discriminator
	// carried in the broker binding for supervised Muse Code runs.
	FactoryScopeMuse = "factory-muse"
)

const (
	// FactoryApproved means the run intent is recorded but nothing started.
	FactoryApproved = "approved"
	// FactoryRunning means the CLI is executing under the recorded identity.
	FactoryRunning = "running"
	// FactoryCompleted means the CLI exited and credentials were returned.
	FactoryCompleted = "completed"
	// FactoryFailed means launch or execution failed without uncertainty.
	FactoryFailed = "failed"
	// FactoryStopped means a stop tombstone retired or barred this identity.
	FactoryStopped = "stopped"
	// FactoryUncertain means retirement or credential return is unconfirmed;
	// the run and its connection stay fenced until reconciled.
	FactoryUncertain = "uncertain"
)

// ValidFactoryPhase reports whether phase is a known factory-run phase.
func ValidFactoryPhase(phase string) bool {
	switch phase {
	case FactoryApproved, FactoryRunning, FactoryCompleted, FactoryFailed, FactoryStopped, FactoryUncertain:
		return true
	default:
		return false
	}
}

var factoryRunID = regexp.MustCompile(`^[a-f0-9]{32}$`)

// ValidFactoryRunID reports whether id is a well-formed run identity.
func ValidFactoryRunID(id string) bool { return factoryRunID.MatchString(id) }

var harnessVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)

// ValidHarnessVersion reports whether version is an admissible harness pin.
// The executor admits only versions with their own proof; the charset keeps
// staged guest paths fixed.
func ValidHarnessVersion(version string) bool { return harnessVersion.MatchString(version) }

// ValidHarnessFamily identifies the supported supervised CLI families.
func ValidHarnessFamily(family string) bool {
	return family == FactoryHarnessCodex || family == FactoryHarnessMuse
}

const (
	// MaxFactoryPrompt bounds one assignment's prompt bytes. Prompts travel
	// the private daemon socket and are staged verbatim for the fixed CLI.
	MaxFactoryPrompt = 64 * 1024
	// MaxFactoryOutput bounds retained last-message output per run.
	MaxFactoryOutput = 64 * 1024
)

// FactoryRun binds one supervised CLI execution to its Project, role,
// preparation, source and broker sponsorship. The ID is also the broker
// execution ID; the assignment digest binds the immutable prompt bytes.
type FactoryRun struct {
	Deadline     time.Time `json:"deadline"`
	Actor        int64     `json:"actor"`
	ID           string    `json:"id"`
	Project      string    `json:"project"`
	Role         string    `json:"role"`
	Preparation  string    `json:"preparation"`
	Harness      string    `json:"harness"`
	HarnessVers  string    `json:"harness_version"`
	Model        string    `json:"model,omitempty"`
	Assignment   string    `json:"assignment"`
	SourceCommit string    `json:"source_commit"`
	Connection   string    `json:"connection"`
}

func (r FactoryRun) Validate() error {
	if !ValidFactoryRunID(r.ID) || !ValidID(r.Project) || !ValidFactoryRole(r.Role) {
		return errors.New("invalid factory run identity")
	}
	if !ValidPreparationID(r.Preparation) {
		return errors.New("invalid run preparation reference")
	}
	if !ValidHarnessFamily(r.Harness) || !ValidHarnessVersion(r.HarnessVers) {
		return errors.New("unsupported factory harness")
	}
	// Empty keeps the CLI default for older launches; a set model pins
	// spend to the dispatch-selected policy choice.
	if r.Model != "" {
		if len(r.Model) > 128 {
			return errors.New("invalid run model selection")
		}
		for i := 0; i < len(r.Model); i++ {
			if r.Model[i] < 0x20 || r.Model[i] == 0x7f {
				return errors.New("invalid run model selection")
			}
		}
	}
	if !ValidDigest(r.Assignment) || !ValidCommit(r.SourceCommit) {
		return errors.New("invalid run assignment or source identity")
	}
	if r.Connection == "" || len(r.Connection) > 128 || r.Actor <= 0 {
		return errors.New("invalid run sponsorship")
	}
	if r.Deadline.IsZero() {
		return errors.New("run deadline is required")
	}
	return nil
}

// FactoryPromptDigest returns the canonical digest over prompt bytes. The
// executor requires it to equal the run's recorded assignment digest.
func FactoryPromptDigest(prompt []byte) string {
	sum := sha256.Sum256(prompt)
	return hex.EncodeToString(sum[:])
}

// FactoryLaunch is the host run request: the recorded run identity, the exact
// prompt bytes and the caller-observed harness pin. The executor admits only
// its own pinned harness bytes.
type FactoryLaunch struct {
	Run           FactoryRun `json:"run"`
	Prompt        []byte     `json:"prompt"`
	HarnessSHA256 string     `json:"harness_sha256"`
}

func (l FactoryLaunch) Validate() error {
	if err := l.Run.Validate(); err != nil {
		return err
	}
	if len(l.Prompt) == 0 || len(l.Prompt) > MaxFactoryPrompt {
		return errors.New("invalid run prompt size")
	}
	if FactoryPromptDigest(l.Prompt) != l.Run.Assignment {
		return errors.New("prompt bytes do not match their digest")
	}
	if !ValidDigest(l.HarnessSHA256) {
		return errors.New("invalid harness pin")
	}
	return nil
}

// FactoryState is the observed run outcome for one identity.
type FactoryState struct {
	ExitCode           *int   `json:"exit_code,omitempty"`
	Generation         int64  `json:"generation"`
	UID                int    `json:"uid"`
	GID                int    `json:"gid"`
	CredentialReturned bool   `json:"credential_returned"`
	Live               bool   `json:"live"`
	Delivered          bool   `json:"delivered"`
	ID                 string `json:"id"`
	Project            string `json:"project"`
	Role               string `json:"role"`
	Phase              string `json:"phase"`
	Container          string `json:"container"`
	Unit               string `json:"unit"`
	Invocation         string `json:"invocation"`
	Login              string `json:"login"`
	LeaseID            string `json:"lease_id,omitempty"`
	Output             string `json:"output,omitempty"`
	Retirement         string `json:"retirement,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

// FactoryRunPaths derives the fixed container paths for one run. The checkout
// comes from the run's preparation; the run home is fresh per run and holds
// the staged credential, prompt, gate marker and bounded output. Empty
// strings report an invalid identity; no caller-supplied path is accepted.
func FactoryRunPaths(role, preparation, run string) (checkout, runDir, home, codexHome string) {
	if !ValidFactoryRole(role) || !ValidPreparationID(preparation) || !ValidFactoryRunID(run) {
		return "", "", "", ""
	}
	checkout = "/home/" + role + "/checkouts/" + preparation
	runDir = checkout + "/.soda-home/runs/" + run
	home = runDir + "/home"
	codexHome = home + "/.codex"
	return checkout, runDir, home, codexHome
}

// FactoryCodexGuest is the fixed versioned guest path for staged harness
// bytes. One immutable path per proved version; concurrent runs share it.
func FactoryCodexGuest(version string) string {
	if !ValidHarnessVersion(version) {
		return ""
	}
	return "/usr/local/bin/codex-factory-" + version
}

// FactoryUnitName is the transient host unit supervising one run. The name is
// deterministic per run identity, so a second start refuses and recovery can
// address a unit whose binding was never recorded.
func FactoryUnitName(run string) string {
	if !ValidFactoryRunID(run) {
		return ""
	}
	return "soda-factory-" + run + ".service"
}

// FactoryInspect addresses a run before or after completion.
type FactoryInspect struct {
	Project string `json:"project"`
	ID      string `json:"id"`
}

func (p FactoryInspect) Validate() error {
	if !ValidID(p.Project) || !ValidFactoryRunID(p.ID) {
		return errors.New("invalid factory run address")
	}
	return nil
}

// FactoryStop retires one run identity, even before it is ever observed.
type FactoryStop struct {
	Project string `json:"project"`
	ID      string `json:"id"`
}

func (p FactoryStop) Validate() error {
	if !ValidID(p.Project) || !ValidFactoryRunID(p.ID) {
		return errors.New("invalid factory run address")
	}
	return nil
}
