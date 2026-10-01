package deliver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testAdmissionEvidence(p Payload, media MediaBinding) QualificationEvidence {
	checks := []EvidenceCheck{}
	for _, name := range requiredQualificationChecks() {
		checks = append(checks, EvidenceCheck{Name: name, Outcome: "passed", Detail: "synthetic native observation"})
	}
	return QualificationEvidence{
		Format: 1, Outcome: "passed", Scope: QualificationScope,
		Revision: p.Revision, Architecture: p.Architecture,
		HostManifest: "", ISOSHA256: media.ISO.SHA256, RootfsSHA256: media.Rootfs.SHA256,
		Checks: checks,
	}
}

func testAdmissionInputs(t *testing.T) (Config, string, string, string, QualificationEvidence) {
	t.Helper()
	tr := testTrust(t)
	r := testRelease(t, tr)
	var p Payload
	var c Candidate
	var media MediaBinding
	require.NoError(t, decode(r.Payload, &p))
	require.NoError(t, decode(r.Candidate, &c))
	require.NoError(t, json.Unmarshal(r.Media, &media))

	root := privateDir(t)
	candidate := filepath.Join(root, "candidate")
	require.NoError(t, os.Mkdir(candidate, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "payload.json"), r.Payload, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "candidate.json"), r.Candidate, 0o600))
	mediaPath := filepath.Join(root, "media.json")
	require.NoError(t, os.WriteFile(mediaPath, r.Media, 0o600))

	sum := sha256.Sum256(r.Payload)
	evidence := testAdmissionEvidence(p, media)
	evidence.PayloadSHA256 = hex.EncodeToString(sum[:])
	evidence.HostManifest = c.Host.Manifest
	evidencePath := filepath.Join(root, "qualification.json")
	raw, err := marshal(evidence)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(evidencePath, raw, 0o600))
	return Config{Serial: 3, Class: "normal", Notes: "fixture notes"}, candidate, mediaPath, evidencePath, evidence
}

func rewriteAdmissionEvidence(t *testing.T, path string, evidence QualificationEvidence) []byte {
	t.Helper()
	raw, err := marshal(evidence)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	return raw
}

func hashAdmissionFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestAdmitQualificationAcceptsExactNativeEvidence(t *testing.T) {
	c, candidate, media, evidencePath, _ := testAdmissionInputs(t)
	before := []string{
		hashAdmissionFile(t, filepath.Join(candidate, "payload.json")),
		hashAdmissionFile(t, filepath.Join(candidate, "candidate.json")),
		hashAdmissionFile(t, media),
		hashAdmissionFile(t, evidencePath),
	}
	q, err := AdmitQualification(c, candidate, media, evidencePath)
	require.NoError(t, err)
	require.Equal(t, uint64(3), q.Serial)
	require.Equal(t, "normal", q.Class)
	require.Equal(t, QualificationScope, q.Scope)
	require.Equal(t, "fixture notes", q.Notes)
	require.Equal(t, map[string]string{"qualification.json": "sha256:" + before[3]}, q.Evidence)
	require.Equal(t, before[0], hashAdmissionFile(t, filepath.Join(candidate, "payload.json")))
	require.Equal(t, before[1], hashAdmissionFile(t, filepath.Join(candidate, "candidate.json")))
	require.Equal(t, before[2], hashAdmissionFile(t, media))
	require.Equal(t, before[3], hashAdmissionFile(t, evidencePath))
}

func TestAdmitQualificationRefusesNonPassingOutcomes(t *testing.T) {
	c, candidate, media, evidencePath, evidence := testAdmissionInputs(t)
	for _, outcome := range []string{"failed", "cancelled", "", "passed-with-warnings"} {
		mutated := evidence
		mutated.Outcome = outcome
		rewriteAdmissionEvidence(t, evidencePath, mutated)
		_, err := AdmitQualification(c, candidate, media, evidencePath)
		require.Error(t, err, outcome)
	}
}

func TestAdmitQualificationRefusesIncompleteEvidence(t *testing.T) {
	c, candidate, media, evidencePath, evidence := testAdmissionInputs(t)
	cases := map[string]func(*QualificationEvidence){
		"missing check":   func(e *QualificationEvidence) { e.Checks = e.Checks[:3] },
		"no checks":       func(e *QualificationEvidence) { e.Checks = nil },
		"failed check":    func(e *QualificationEvidence) { e.Checks[1].Outcome = "failed" },
		"cancelled check": func(e *QualificationEvidence) { e.Checks[2].Outcome = "cancelled" },
		"unexpected check": func(e *QualificationEvidence) {
			e.Checks = append(e.Checks, EvidenceCheck{Name: "bonus", Outcome: "passed"})
		},
		"duplicated check": func(e *QualificationEvidence) { e.Checks = append(e.Checks, e.Checks[0]) },
		"overlong detail":  func(e *QualificationEvidence) { e.Checks[0].Detail = strings.Repeat("x", 1025) },
		"wrong scope":      func(e *QualificationEvidence) { e.Scope = "local-only" },
		"empty scope":      func(e *QualificationEvidence) { e.Scope = "" },
		"fixture run":      func(e *QualificationEvidence) { e.Fixture = true },
		"wrong format":     func(e *QualificationEvidence) { e.Format = 2 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			mutated := evidence
			mutated.Checks = append([]EvidenceCheck{}, evidence.Checks...)
			mutate(&mutated)
			rewriteAdmissionEvidence(t, evidencePath, mutated)
			_, err := AdmitQualification(c, candidate, media, evidencePath)
			require.Error(t, err)
		})
	}
}

func TestAdmitQualificationRefusesMismatchedEvidence(t *testing.T) {
	c, candidate, media, evidencePath, evidence := testAdmissionInputs(t)
	cases := map[string]func(*QualificationEvidence){
		"wrong revision":      func(e *QualificationEvidence) { e.Revision = strings.Repeat("9", 40) },
		"wrong architecture":  func(e *QualificationEvidence) { e.Architecture = "aarch64" },
		"wrong payload hash":  func(e *QualificationEvidence) { e.PayloadSHA256 = strings.Repeat("0", 64) },
		"wrong host manifest": func(e *QualificationEvidence) { e.HostManifest = "sha256:" + strings.Repeat("0", 64) },
		"wrong iso hash":      func(e *QualificationEvidence) { e.ISOSHA256 = strings.Repeat("0", 64) },
		"wrong rootfs hash":   func(e *QualificationEvidence) { e.RootfsSHA256 = strings.Repeat("0", 64) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			mutated := evidence
			mutate(&mutated)
			rewriteAdmissionEvidence(t, evidencePath, mutated)
			_, err := AdmitQualification(c, candidate, media, evidencePath)
			require.Error(t, err)
		})
	}
}

func rewriteAdmissionCandidateRevision(t *testing.T, candidate, revision string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(candidate, "payload.json"))
	require.NoError(t, err)
	var p Payload
	require.NoError(t, decode(raw, &p))
	p.Revision = revision
	p.ID = p.CoreOS + ".soda-" + revision[:12]
	payload, err := marshal(p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "payload.json"), payload, 0o600))
	raw, err = os.ReadFile(filepath.Join(candidate, "candidate.json"))
	require.NoError(t, err)
	var c Candidate
	require.NoError(t, decode(raw, &c))
	c.Host.Revision = revision
	sum := sha256.Sum256(payload)
	c.PayloadSHA256 = hex.EncodeToString(sum[:])
	updated, err := marshal(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "candidate.json"), updated, 0o600))
}

func TestAdmitQualificationRefusesReplayedAndBrokenCandidates(t *testing.T) {
	c, candidate, media, evidencePath, _ := testAdmissionInputs(t)
	_, other, _, _, _ := testAdmissionInputs(t)
	rewriteAdmissionCandidateRevision(t, other, strings.Repeat("9", 40))
	_, err := AdmitQualification(c, other, media, evidencePath)
	require.Error(t, err, "evidence for one candidate cannot admit another")
	raw, err := os.ReadFile(media)
	require.NoError(t, err)
	var binding MediaBinding
	require.NoError(t, json.Unmarshal(raw, &binding))
	binding.ISO.SHA256 = strings.Repeat("0", 64)
	changed, err := json.Marshal(binding)
	require.NoError(t, err)
	substituted := filepath.Join(t.TempDir(), "media.json")
	require.NoError(t, os.WriteFile(substituted, changed, 0o600))
	_, err = AdmitQualification(c, candidate, substituted, evidencePath)
	require.Error(t, err, "substituted media cannot reuse the evidence")
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "candidate.json"), []byte(`{"Format":1}`), 0o600))
	_, err = AdmitQualification(c, candidate, media, evidencePath)
	require.Error(t, err, "invalid candidate metadata refuses")
}

func TestAdmitQualificationRefusesAdvertisedUpgradeSource(t *testing.T) {
	c, candidate, media, evidencePath, _ := testAdmissionInputs(t)
	raw, err := os.ReadFile(filepath.Join(candidate, "payload.json"))
	require.NoError(t, err)
	var p Payload
	require.NoError(t, decode(raw, &p))
	p.UpgradeFrom = []string{"unproved-source"}
	changed, err := marshal(p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(candidate, "payload.json"), changed, 0o600))
	_, err = AdmitQualification(c, candidate, media, evidencePath)
	require.Error(t, err, "an advertised upgrade source without qualified evidence refuses")
}

func TestAdmitQualificationRefusesForgedEvidence(t *testing.T) {
	c, candidate, media, evidencePath, evidence := testAdmissionInputs(t)
	raw, err := marshal(evidence)
	require.NoError(t, err)
	forged := map[string]string{
		"unknown field": string(raw[:len(raw)-2]) + `,"Verdict":"passed"}` + "\n",
		"duplicate key": `{"Format":1,"Format":1,` + string(raw[len(`{"Format":1,`):]),
		"trailing data": string(raw) + `{"Format":1}` + "\n",
		"not json":      "passed\n",
		"empty":         "",
	}
	for name, body := range forged {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(evidencePath, []byte(body), 0o600))
			_, err := AdmitQualification(c, candidate, media, evidencePath)
			require.Error(t, err)
		})
	}
	require.NoError(t, os.WriteFile(evidencePath, []byte(strings.Repeat("x", (1<<20)+1)), 0o600))
	_, err = AdmitQualification(c, candidate, media, evidencePath)
	require.Error(t, err, "oversized evidence refuses")
}

func TestAdmitQualificationRequiresOperatorIdentity(t *testing.T) {
	_, candidate, media, evidencePath, _ := testAdmissionInputs(t)
	for _, c := range []Config{
		{Serial: 0, Class: "normal", Notes: "n"},
		{Serial: 1, Class: "other", Notes: "n"},
		{Serial: 1, Class: "normal", Notes: ""},
		{Serial: 1, Class: "normal", Notes: strings.Repeat("n", 16385)},
	} {
		_, err := AdmitQualification(c, candidate, media, evidencePath)
		require.Error(t, err)
	}
}
