package project

import (
	"errors"
	"strings"
)

// FactoryHarnessPin is the host's staged-harness identity: the proved
// harness family, its version string and content pin, plus the exact
// container image factory runs execute under. The coordinator dispatches
// only when the pin matches the policy selection exactly; an unversioned
// or unpinned host waits instead of running unattributed work.
type FactoryHarnessPin struct {
	Harness string `json:"harness"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Image   string `json:"image"`
}

// Validate rejects malformed or unpinned harness identities.
func (p FactoryHarnessPin) Validate() error {
	if p.Harness != FactoryHarnessCodex || !ValidHarnessVersion(p.Version) {
		return errors.New("unsupported factory harness")
	}
	if !ValidDigest(p.SHA256) {
		return errors.New("invalid harness pin")
	}
	value, pinned := strings.CutPrefix(p.Image, "sha256:")
	if !pinned || !ValidDigest(value) {
		return errors.New("execution image is not pinned")
	}
	return nil
}
