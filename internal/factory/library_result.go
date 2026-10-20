package factory

// LibraryMaintenanceResult binds terminal reporting to the accepted MCP
// operation. It carries no candidate or permission to change work policy.
type LibraryMaintenanceResult struct {
	SubmissionID string `json:"submission_id"`
}

func (o PhaseOutput) nilTypedOutputExceptMaintenance() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil && o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil && o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil && o.Research == nil && o.Assessment == nil
}
