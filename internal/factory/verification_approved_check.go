package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// ApprovedAppCheck is an explicit whole-app assertion in the administrator's
// protected check.sh, not a worker claim or an ordinary setup success. The
// native preparation executor runs that exact approved script on the delivery
// source. Each listed outcome is asserted by the script's zero exit status.
// External fixture grants and arbitrary check catalogs are not implied.
type ApprovedAppCheck struct {
	IntentID         string   `json:"intent_id"`
	AcceptedRevision string   `json:"accepted_revision"`
	Outcomes         []string `json:"outcomes"`
}

const ApprovedAppCheckPrefix = "# soda-app-verification: "
const ApprovedAppCheckID = project.VerificationProjectCheck

func ParseApprovedAppCheck(script []byte) (ApprovedAppCheck, error) {
	var manifest ApprovedAppCheck
	found := false
	for _, line := range strings.Split(string(script), "\n") {
		if !strings.HasPrefix(line, ApprovedAppCheckPrefix) {
			continue
		}
		if found {
			return manifest, errors.New("approved check declares multiple verification manifests")
		}
		found = true
		decoder := json.NewDecoder(bytes.NewBufferString(strings.TrimPrefix(line, ApprovedAppCheckPrefix)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&manifest); err != nil {
			return manifest, errors.New("approved app verification manifest is invalid")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return manifest, errors.New("approved app verification manifest has trailing content")
		}
	}
	if !found {
		return manifest, errors.New("approved check does not declare application outcomes")
	}
	return manifest, manifest.Validate()
}

func (manifest ApprovedAppCheck) Validate() error {
	if !project.ValidDecisionID(manifest.IntentID) || !project.ValidDecisionID(manifest.AcceptedRevision) || len(manifest.Outcomes) == 0 || len(manifest.Outcomes) > MCPMaxManifestOutcomes {
		return errors.New("approved app verification manifest lacks its accepted scope")
	}
	seen := map[string]bool{}
	for _, outcome := range manifest.Outcomes {
		if !project.ValidDecisionID(outcome) || seen[outcome] {
			return errors.New("invalid or duplicate approved application outcome")
		}
		seen[outcome] = true
	}
	return nil
}

func (m ApprovedAppCheck) Digest() string {
	raw, _ := json.Marshal(m)
	return MCPInputDigest(raw)
}

func (m ApprovedAppCheck) BindsIntent(revision IntentRevision) bool {
	want, got := slices.Clone(revision.EffectiveOutcomes), slices.Clone(m.Outcomes)
	slices.Sort(want)
	slices.Sort(got)
	return revision.State == "accepted" && m.IntentID == revision.Intent && m.AcceptedRevision == revision.ID && slices.Equal(want, got)
}

// DeliveryPreparationDigest identifies immutable preparation inputs, excluding
// the observation revision which advances when native state is refreshed.
func DeliveryPreparationDigest(preparation project.Preparation) string {
	preparation.Revision = 0
	raw, _ := json.Marshal(preparation)
	return MCPInputDigest(raw)
}
