package forgejo

import (
	"context"
	"errors"

	sdk "forgejo.org/extension-sdk"
)

// ApplyIssueIntake presents only the request-scoped native grant. It does not
// use, expand or enroll a Factory bot's repository permissions.
func (b *ServiceBackground) ApplyIssueIntake(ctx context.Context, request sdk.IssueIntakeMutation) (sdk.IssueIntakeReceipt, error) {
	if !sdk.ValidIssueIntakeMutation(request) || request.Grant == "" {
		return sdk.IssueIntakeReceipt{}, errors.New("invalid native issue intake")
	}
	var receipt sdk.IssueIntakeReceipt
	if err := b.call(ctx, sdk.BackgroundIssueIntakePath, request, &receipt); err != nil {
		return receipt, err
	}
	if receipt.Index <= 0 || receipt.IssueID <= 0 || receipt.Slot != request.Slot || receipt.Title != request.Title || receipt.BodyDigest != ContentDigest(request.Body) {
		return sdk.IssueIntakeReceipt{}, ErrInvalidResponse
	}
	return receipt, nil
}
