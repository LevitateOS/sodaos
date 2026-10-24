package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// CurrentVerificationEvidence is the native supervisor's approved fixture and
// receipt. State is for controller adoption only, never the model projection.
type CurrentVerificationEvidence struct {
	Manifest          ApprovedAppCheck             `json:"manifest"`
	ManifestDigest    string                       `json:"manifest_digest"`
	PreparationDigest string                       `json:"preparation_digest"`
	SourceCommit      string                       `json:"source_commit"`
	Receipt           VerificationCheckReceipt     `json:"check_receipt"`
	ApprovedContext   *ApprovedVerificationContext `json:"approved_context,omitempty"`
	State             project.PrepareState         `json:"-"`
}

// ApprovedVerificationContext projects the native reader's bounded UTF-8 source
// as text so Test Pilot can inspect actual assertions. Repository text remains
// untrusted input, and a passing receipt does not itself prove outcome coverage.
type ApprovedVerificationContext struct {
	Project      string                       `json:"project"`
	ID           string                       `json:"id"`
	Role         string                       `json:"role"`
	SourceCommit string                       `json:"source_commit"`
	ApprovedBase string                       `json:"approved_base"`
	DiffBase     string                       `json:"diff_base"`
	Candidate    string                       `json:"candidate"`
	Setup        string                       `json:"setup"`
	Check        string                       `json:"check"`
	Files        []ApprovedVerificationSource `json:"files"`
	Diff         string                       `json:"diff"`
}

type ApprovedVerificationSource struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func NewApprovedVerificationContext(c project.FactoryPreparationContext) *ApprovedVerificationContext {
	files := make([]ApprovedVerificationSource, len(c.Files))
	for i, file := range c.Files {
		files[i] = ApprovedVerificationSource{Path: file.Path, Content: string(file.Content)}
	}
	return &ApprovedVerificationContext{Project: c.Project, ID: c.ID, Role: c.Role, SourceCommit: c.SourceCommit, ApprovedBase: c.ApprovedBase, DiffBase: c.DiffBase, Candidate: c.Candidate, Setup: string(c.Setup), Check: string(c.Check), Files: files, Diff: string(c.Diff)}
}

func (c ApprovedVerificationContext) nativeContext() project.FactoryPreparationContext {
	files := make([]project.FactoryContextFile, len(c.Files))
	for i, file := range c.Files {
		files[i] = project.FactoryContextFile{Path: file.Path, Content: []byte(file.Content)}
	}
	return project.FactoryPreparationContext{Project: c.Project, ID: c.ID, Role: c.Role, SourceCommit: c.SourceCommit, ApprovedBase: c.ApprovedBase, DiffBase: c.DiffBase, Candidate: c.Candidate, Setup: []byte(c.Setup), Check: []byte(c.Check), Files: files, Diff: []byte(c.Diff)}
}

func (e CurrentVerificationEvidence) ValidateFor(d DeliverySubject) error {
	if e.Manifest.Validate() != nil || e.Manifest.Digest() != d.VerificationManifestDigest || e.ManifestDigest != d.VerificationManifestDigest || e.PreparationDigest != d.PreparationDigest || e.SourceCommit != d.SourceCommit || e.Manifest.IntentID != d.IntentID || e.Manifest.AcceptedRevision != d.AcceptedRevision || e.Receipt.CheckID != ApprovedAppCheckID || e.Receipt.Outcome != CheckReceiptPassed || e.Receipt.Validate() != nil {
		return errors.New("current native verification differs from admitted delivery")
	}
	if e.ApprovedContext == nil || e.ApprovedContext.nativeContext().ValidateContent(project.RoleReviewer) != nil || e.ApprovedContext.SourceCommit != d.SourceCommit || e.ApprovedContext.ApprovedBase != d.SourceCommit || e.ApprovedContext.DiffBase != d.SourceCommit || e.ApprovedContext.Candidate != d.SourceCommit {
		return errors.New("approved verification assertions differ from admitted source")
	}
	manifest, err := ParseApprovedAppCheck([]byte(e.ApprovedContext.Check))
	if err != nil || manifest.Digest() != d.VerificationManifestDigest {
		return errors.New("approved verification assertions differ from admitted manifest")
	}
	return nil
}
