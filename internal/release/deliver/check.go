package deliver

import (
	"os"
)

// CheckCandidate binds one delivered candidate directory to the requested
// architecture and Soda/Fountain revisions, then verifies the delivered
// archives through the existing candidate/archive owners. It reports the
// first mismatch; it never builds, installs, publishes or changes trust.
func CheckCandidate(candidate, arch, sodaRevision, forgejoRevision string) error {
	p, c, _, err := loadAdmittedCandidate(candidate)
	if err != nil {
		return err
	}
	if p.Architecture != arch {
		return errorAt("candidate architecture")
	}
	if p.Revision != sodaRevision {
		return errorAt("candidate soda revision")
	}
	if c.ForgejoRevision != forgejoRevision {
		return errorAt("candidate forgejo revision")
	}
	root, err := os.OpenRoot(candidate)
	if err != nil {
		return err
	}
	defer root.Close()
	_, err = VerifyCandidateImages(root, candidate, p, c)
	return err
}
