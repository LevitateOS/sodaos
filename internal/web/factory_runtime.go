package web

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	sdk "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// Appliance composition translates the execution actor at the native credential
// boundary. The coordinator's captured launch and assignments remain canonical;
// the host and its broker consume native Fountain user identities.
type applianceFactoryHost struct {
	control.HostFactory
	identities          *forgejo.FountainIdentities
	store               *store.Store
	controller          sdk.ControllerBackground
	executionCredential sdk.CredentialFile
	custody             execution.CustodyObserver
}

var _ control.HostFactory = (*applianceFactoryHost)(nil)

func (h *applianceFactoryHost) LaunchRun(ctx context.Context, request execution.LaunchRequest) (execution.RunObservation, error) {
	if h == nil || h.HostFactory == nil || h.identities == nil || h.store == nil || h.controller == nil || h.executionCredential == "" || request.Validate() != nil || request.Kind != collaboration.Appliance || request.Appliance.ControllerGrantID == "" {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	run, err := h.store.FactoryRun(ctx, request.Input.Run.ID)
	if err != nil || run.Validate() != nil || run.ProjectID != request.Input.Run.Project || run.Admission == nil || run.Admission.Policy.Execution == nil || run.Admission.Policy.Execution.ControllerGrantID != request.Appliance.ControllerGrantID || run.Admission.ControllerManifestDigest != request.Appliance.ControllerManifestDigest {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	if _, err := h.store.AuthorizeFactoryLaunch(ctx, request); err != nil {
		return execution.RunObservation{}, err
	}
	g, err := h.controller.ReadControllerGrant(ctx, request.Appliance.ControllerGrantID)
	if err != nil {
		observed, retireErr := h.RetireRun(ctx, request.Address())
		if retireErr != nil {
			return observed, retireErr
		}
		return observed, err
	}
	nativeRepository, err := h.identities.NativeRepository(ctx, collaboration.RepositoryID(run.Admission.Policy.Repository))
	if err != nil {
		return execution.RunObservation{}, err
	}
	grantee, err := h.identities.NativeActor(ctx, collaboration.ActorID(run.Admission.Policy.Execution.ActorID))
	if err != nil {
		return execution.RunObservation{}, err
	}
	if !g.Active || g.RepositoryID != nativeRepository || g.ActorID != grantee || g.TokenID != strconv.FormatInt(run.Admission.Policy.Execution.TokenID, 10) || run.Deadline.After(time.Unix(g.NotAfter, 0)) {
		observed, err := h.RetireRun(ctx, request.Address())
		if err != nil {
			return observed, err
		}
		return observed, execution.ErrLaunch
	}
	claim, err := h.controller.ClaimControllerExecution(ctx, h.executionCredential, g.ID, sdk.ExecutionClaimRequest{CommandID: run.ID, ManifestDigest: run.Admission.ControllerManifestDigest, NotAfter: controllerDeadline(run.Deadline)})
	if err != nil {
		return execution.RunObservation{}, err
	}
	if !claim.Acquired {
		return h.InspectRun(ctx, request.Address())
	}
	current, err := h.controller.ReadControllerGrant(ctx, g.ID)
	if err != nil || !current.Active {
		observed, retireErr := h.RetireRun(ctx, request.Address())
		if retireErr != nil {
			return observed, retireErr
		}
		return observed, execution.ErrLaunch
	}
	if _, err := h.store.AuthorizeFactoryLaunch(ctx, request); err != nil {
		observed, retireErr := h.RetireRun(ctx, request.Address())
		if retireErr != nil {
			return observed, retireErr
		}
		return observed, err
	}
	native, err := h.identities.NativeActor(ctx, collaboration.ActorID(request.Input.Run.Actor))
	if err != nil {
		return execution.RunObservation{}, err
	}
	actor, err := strconv.ParseInt(native, 10, 64)
	if err != nil || actor < 1 {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	// FactoryLaunch.Run is a value: translate the outgoing copy without changing
	// prompt/manifest bytes, custody references or the captured canonical packet.
	request.Input.Run.Actor = actor
	return h.HostFactory.LaunchRun(ctx, request)
}

func controllerDeadline(deadline time.Time) int64 {
	n := deadline.Unix()
	if deadline.Nanosecond() != 0 {
		n++
	}
	return n
}

func (h *applianceFactoryHost) controllerRun(ctx context.Context, address execution.RunAddress) (factory.Run, error) {
	if h.store == nil {
		return factory.Run{}, nil
	}
	r, err := h.store.FactoryRun(ctx, address.Run)
	if errors.Is(err, sql.ErrNoRows) {
		return factory.Run{}, nil
	}
	if err != nil {
		return r, err
	}
	if r.ProjectID != address.Workspace || r.Validate() != nil {
		return r, execution.ErrObservation
	}
	return r, nil
}

func (h *applianceFactoryHost) InspectRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	r, err := h.controllerRun(ctx, address)
	if err != nil {
		return execution.RunObservation{}, err
	}
	if r.Admission == nil || r.Admission.Policy.Execution == nil {
		return h.HostFactory.InspectRun(ctx, address)
	}
	if h.controller == nil {
		return execution.RunObservation{}, execution.ErrObservation
	}
	id := r.Admission.Policy.Execution.ControllerGrantID
	claim, err := h.controller.ReadControllerExecution(ctx, id, r.ID)
	if err != nil {
		return execution.RunObservation{}, err
	} // absence never permits a second launch or releases the source
	g, err := h.controller.ReadControllerGrant(ctx, id)
	if err != nil {
		return execution.RunObservation{}, err
	}
	if !g.Active || claim.State != "claimed" {
		return h.RetireRun(ctx, address)
	}
	observed, err := h.HostFactory.InspectRun(ctx, address)
	if err != nil {
		return observed, fmt.Errorf("claimed native execution remains uncertain: %v", err)
	}
	return observed, nil
}

func (h *applianceFactoryHost) RetireRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	r, err := h.controllerRun(ctx, address)
	if err != nil {
		return execution.RunObservation{}, err
	}
	observed, err := h.HostFactory.RetireRun(ctx, address)
	if err != nil {
		return observed, err
	}
	if r.Admission == nil || r.Admission.Policy.Execution == nil {
		return observed, nil
	}
	if observed.ValidateFor(address) != nil || observed.Live || observed.Retirement != execution.Retired || h.custody == nil || h.controller == nil {
		return observed, execution.ErrObservation
	}
	custody, err := h.custody.ReadCustody(ctx, address)
	if err != nil || custody.ValidateFor(address) != nil || !custody.Sealed || custody.Appliance == nil || custody.Appliance.State != execution.Released {
		return observed, execution.ErrObservation
	}
	raw, err := json.Marshal(struct {
		Native  execution.RunObservation
		Custody execution.CustodyObservation
	}{observed, custody})
	if err != nil {
		return observed, err
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	id := r.Admission.Policy.Execution.ControllerGrantID
	claim, lookupErr := h.controller.ReadControllerExecution(ctx, id, r.ID)
	state := "retired"
	if observed.Reason == "stop-before-start" && observed.InvocationID() == "" {
		state = "not_started"
	}
	if lookupErr == nil && claim.State != "claimed" {
		if claim.ManifestDigest != r.Admission.ControllerManifestDigest || claim.RetirementDigest != digest || claim.State != state {
			return observed, execution.ErrObservation
		}
		return observed, nil
	}
	if lookupErr != nil && state != "not_started" {
		return observed, lookupErr
	}
	_, err = h.controller.RetireControllerExecution(ctx, id, r.ID, sdk.ExecutionRetirement{NotAfter: controllerDeadline(r.Deadline), ManifestDigest: r.Admission.ControllerManifestDigest, State: state, ReceiptDigest: digest})
	return observed, err
}
