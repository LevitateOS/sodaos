package control

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
)

// ReviewExecutor observes an exact candidate and submits a body-only review
// through a separate native actor. It provides no publication capability.
type ReviewExecutor interface {
	ObserveReview(ctx context.Context, work factory.ReviewWork) (factory.ReviewObservation, error)
	SubmitReview(ctx context.Context, work factory.ReviewWork) (factory.OperationOutcome, error)
	LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	AdoptReview(work factory.ReviewWork, outcome factory.OperationOutcome) (factory.ReviewOutcome, error)
}
