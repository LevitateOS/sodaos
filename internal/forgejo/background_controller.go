package forgejo

import (
	"context"
	"errors"

	sdk "forgejo.org/extension-sdk"
)

var _ sdk.ControllerBackground = (*ServiceBackground)(nil)

func (b *ServiceBackground) ReadControllerGrant(ctx context.Context, id string) (sdk.ApplianceOperationGrant, error) {
	var g sdk.ApplianceOperationGrant
	if !sdk.ValidControllerReference(sdk.ControllerReference{GrantID: id}) {
		return g, errors.New("invalid controller reference")
	}
	if err := b.call(ctx, sdk.BackgroundControllerGrantPath, sdk.ControllerReference{GrantID: id}, &g); err != nil {
		return g, err
	}
	b.mu.Lock()
	installation := b.pinned
	b.mu.Unlock()
	return g, sdk.ValidateControllerGrant(g, id, installation)
}

func (b *ServiceBackground) ClaimControllerExecution(ctx context.Context, credential sdk.CredentialFile, id string, q sdk.ExecutionClaimRequest) (sdk.ExecutionClaimReceipt, error) {
	var r sdk.ExecutionClaimReceipt
	ref := sdk.ControllerReference{GrantID: id, CommandID: q.CommandID}
	if !sdk.ValidControllerReference(ref) || q.CommandID == "" {
		return r, errors.New("invalid execution claim reference")
	}
	secret, err := backgroundSecret(credential)
	if err != nil {
		return r, err
	}
	if err = b.call(ctx, sdk.BackgroundControllerClaimPath, sdk.ControllerClaim{ExecutionClaimRequest: q, GrantID: id, Token: secret}, &r); err != nil {
		return r, err
	}
	if r.ManifestDigest != q.ManifestDigest || r.NotAfter != q.NotAfter {
		return r, errors.New("execution claim changed")
	}
	return r, sdk.ValidateControllerClaim(r, ref)
}

func (b *ServiceBackground) ReadControllerExecution(ctx context.Context, id, command string) (sdk.ExecutionClaimReceipt, error) {
	ref := sdk.ControllerReference{GrantID: id, CommandID: command}
	var r sdk.ExecutionClaimReceipt
	if !sdk.ValidControllerReference(ref) || command == "" {
		return r, errors.New("invalid execution claim reference")
	}
	if err := b.call(ctx, sdk.BackgroundControllerClaimGetPath, ref, &r); err != nil {
		return r, err
	}
	if r.Acquired {
		return r, errors.New("lookup cannot acquire execution")
	}
	return r, sdk.ValidateControllerClaim(r, ref)
}

func (b *ServiceBackground) RetireControllerExecution(ctx context.Context, id, command string, q sdk.ExecutionRetirement) (sdk.ExecutionClaimReceipt, error) {
	ref := sdk.ControllerReference{GrantID: id, CommandID: command}
	var r sdk.ExecutionClaimReceipt
	if !sdk.ValidControllerReference(ref) || command == "" {
		return r, errors.New("invalid execution retirement reference")
	}
	if err := b.call(ctx, sdk.BackgroundControllerRetirePath, sdk.ControllerRetirement{ControllerReference: ref, ExecutionRetirement: q}, &r); err != nil {
		return r, err
	}
	if r.Acquired || r.ManifestDigest != q.ManifestDigest || r.RetirementDigest != q.ReceiptDigest || r.State != q.State {
		return r, errors.New("execution retirement changed")
	}
	return r, sdk.ValidateControllerClaim(r, ref)
}
