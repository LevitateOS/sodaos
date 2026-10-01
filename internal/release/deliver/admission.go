// Qualification admission binds protected final signing to exact native
// install/upgrade/recovery evidence. Failed, cancelled, incomplete,
// mismatched or forged records cannot become releasable, and fixture records
// are explicitly non-qualifying. Nativeness itself is established by protected
// custody (root-admitted config, restricted inputs, isolated worker identity),
// never by these bytes alone — the same trust model as Permit.
package deliver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
)

// QualificationScope is the only evidence scope protected finalization admits.
const QualificationScope = "native-install-upgrade-recovery"

// EvidenceCheck is one required native observation inside qualification
// evidence. Only Outcome "passed" admits; Detail is a bounded observation
// pointer, never a credential.
type EvidenceCheck struct {
	Name, Outcome, Detail string
}

// QualificationEvidence is the exact record a protected qualification worker
// writes after native install/upgrade/recovery runs against one candidate and
// its media. Hashes bind exact bytes; revision and architecture bind the
// payload identity. Serial, class and notes stay operator admission in Config:
// evidence never self-authorizes a release.
type QualificationEvidence struct {
	Format                                 int
	Outcome, Scope, Revision, Architecture string
	PayloadSHA256, HostManifest            string
	ISOSHA256, RootfsSHA256                string
	Checks                                 []EvidenceCheck
	Fixture                                bool
}

func requiredQualificationChecks() []string {
	return []string{"install", "upgrade", "recovery", "preservation"}
}

func admitReleaseIdentity(c Config) error {
	if c.Serial == 0 || (c.Class != "normal" && c.Class != "emergency") || c.Notes == "" || len(c.Notes) > 16384 {
		return errors.New("exact release serial, class and notes required")
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func decodeQualificationEvidence(path string) (QualificationEvidence, []byte, error) {
	var evidence QualificationEvidence
	raw, err := ReadFile(path, 1<<20)
	if err != nil {
		return evidence, nil, err
	}
	if err := decode(raw, &evidence); err != nil {
		return evidence, nil, errors.New("qualification evidence refused")
	}
	return evidence, raw, nil
}

func admitEvidenceShape(evidence QualificationEvidence) error {
	if evidence.Format != 1 {
		return errors.New("qualification evidence format refused")
	}
	if evidence.Fixture {
		return errors.New("fixture evidence is explicitly non-qualifying")
	}
	switch evidence.Outcome {
	case "passed":
	default:
		return errors.New("qualification evidence outcome is not passed")
	}
	if evidence.Scope != QualificationScope {
		return errors.New("qualification evidence scope is not native install/upgrade/recovery")
	}
	return nil
}

func admitEvidenceCheck(check EvidenceCheck, seen map[string]bool) error {
	if !seen[check.Name] {
		return errors.New("qualification evidence names an unexpected check")
	}
	delete(seen, check.Name)
	if check.Outcome != "passed" {
		return errors.New("qualification evidence check did not pass: " + check.Name)
	}
	if len(check.Detail) > 1024 {
		return errors.New("qualification evidence check detail too long: " + check.Name)
	}
	return nil
}

func admitEvidenceChecks(evidence QualificationEvidence) error {
	seen := map[string]bool{}
	for _, name := range requiredQualificationChecks() {
		seen[name] = true
	}
	for _, check := range evidence.Checks {
		if err := admitEvidenceCheck(check, seen); err != nil {
			return err
		}
	}
	if len(seen) != 0 {
		return errors.New("qualification evidence check set is incomplete")
	}
	return nil
}

func loadAdmittedCandidate(candidate string) (Payload, Candidate, []byte, error) {
	var p Payload
	var c Candidate
	root, err := os.OpenRoot(candidate)
	if err != nil {
		return p, c, nil, err
	}
	defer root.Close()
	payload, err := readAt(root, "payload.json", 1<<20)
	if err != nil {
		return p, c, nil, err
	}
	raw, err := readAt(root, "candidate.json", 1<<20)
	if err != nil {
		return p, c, nil, err
	}
	if decode(payload, &p) != nil || decode(raw, &c) != nil {
		return p, c, nil, errors.New("candidate metadata refused")
	}
	if err := c.Validate(p, payload); err != nil {
		return p, c, nil, err
	}
	return p, c, payload, nil
}

func admitEvidenceCandidate(evidence QualificationEvidence, p Payload, c Candidate, payload []byte) error {
	if evidence.Revision != p.Revision || evidence.Architecture != p.Architecture {
		return errorAt("qualification evidence candidate identity")
	}
	if evidence.PayloadSHA256 != sha256Hex(payload) {
		return errorAt("qualification evidence payload binding")
	}
	if evidence.HostManifest != c.Host.Manifest {
		return errorAt("qualification evidence host binding")
	}
	return nil
}

func admitEvidenceMedia(evidence QualificationEvidence, media string, p Payload, c Candidate) error {
	raw, err := readMediaBinding(media)
	if err != nil {
		return err
	}
	var binding MediaBinding
	if json.Unmarshal(raw, &binding) != nil || !validMediaBinding(binding, p, c) {
		return errorAt("qualification evidence media binding")
	}
	if evidence.ISOSHA256 != binding.ISO.SHA256 || evidence.RootfsSHA256 != binding.Rootfs.SHA256 {
		return errorAt("qualification evidence media identity")
	}
	return nil
}

// AdmitQualification enforces the release contract on one evidence record: it
// must claim a passed native install/upgrade/recovery run, name the complete
// check set, and bind the exact candidate and media bytes. It reads only; the
// admitted bytes reach Prepare unchanged.
func AdmitQualification(c Config, candidate, media, evidencePath string) (Qualification, error) {
	if err := admitReleaseIdentity(c); err != nil {
		return Qualification{}, err
	}
	evidence, raw, err := decodeQualificationEvidence(evidencePath)
	if err != nil {
		return Qualification{}, err
	}
	if err := admitEvidenceShape(evidence); err != nil {
		return Qualification{}, err
	}
	if err := admitEvidenceChecks(evidence); err != nil {
		return Qualification{}, err
	}
	p, cand, payload, err := loadAdmittedCandidate(candidate)
	if err != nil {
		return Qualification{}, err
	}
	if err := admitEvidenceCandidate(evidence, p, cand, payload); err != nil {
		return Qualification{}, err
	}
	if err := admitEvidenceMedia(evidence, media, p, cand); err != nil {
		return Qualification{}, err
	}
	return Qualification{
		Serial:   c.Serial,
		Class:    c.Class,
		Scope:    QualificationScope,
		Notes:    c.Notes,
		Evidence: map[string]string{"qualification.json": "sha256:" + sha256Hex(raw)},
	}, nil
}
