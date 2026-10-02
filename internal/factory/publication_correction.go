package factory

// CorrectionOps is one publication's ordered correction chain: branch
// republications after the linked PR exists. Each entry chains its
// predecessor tip and binds the same PR; the publication head follows
// the latest committed correction. See Publication.Validate.
type CorrectionOps []PublicationOperation
