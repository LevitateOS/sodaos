package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// CurrentVerificationEvidence is the native supervisor's approved fixture and
// receipt. State is for controller adoption only, never the model projection.
type CurrentVerificationEvidence struct {
	Manifest          ApprovedAppCheck         `json:"manifest"`
	ManifestDigest    string                   `json:"manifest_digest"`
	PreparationDigest string                   `json:"preparation_digest"`
	SourceCommit      string                   `json:"source_commit"`
	Receipt           VerificationCheckReceipt `json:"check_receipt"`
	State             project.PrepareState     `json:"-"`
}

func (e CurrentVerificationEvidence) ValidateFor(d DeliverySubject) error {
	if e.Manifest.Validate() != nil || e.Manifest.Digest() != d.VerificationManifestDigest || e.ManifestDigest != d.VerificationManifestDigest || e.PreparationDigest != d.PreparationDigest || e.SourceCommit != d.SourceCommit || e.Manifest.IntentID != d.IntentID || e.Manifest.AcceptedRevision != d.AcceptedRevision || e.Receipt.CheckID != ApprovedAppCheckID || e.Receipt.Outcome != CheckReceiptPassed || e.Receipt.Validate() != nil {
		return errors.New("current native verification differs from admitted delivery")
	}
	return nil
}
