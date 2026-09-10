package runners

import (
	"context"
	"fmt"
	"io"

	"github.com/levitateos/sodaos/internal/linuxhost"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type LocalReader interface {
	List(context.Context) ([]RunnerView, error)
}

type Coordinator struct {
	ForgejoURL       string
	ForgejoPublicURL string
	Authorizer       Authorizer
	Local            LocalReader
	Lifecycle        Lifecycle
}

func (coordinator Coordinator) Execute(ctx context.Context, actor linuxhost.PKExecIdentity, action string, input io.Reader) (any, error) {
	if err := coordinator.Authorizer.RequireAdministrator(ctx, actor); err != nil {
		return nil, err
	}
	switch action {
	case "list":
		return coordinator.list(ctx, input)
	case "create":
		return coordinator.create(ctx, input)
	case "start", "stop", "restart", "remove":
		return coordinator.mutate(ctx, action, input)
	default:
		return nil, fmt.Errorf("unsupported soda-runners action %q", action)
	}
}

func (coordinator Coordinator) list(ctx context.Context, input io.Reader) (ListResponse, error) {
	var request EmptyRequest
	if err := strictjson.Decode(input, &request); err != nil {
		return ListResponse{}, err
	}
	views, err := coordinator.Local.List(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	response := ListResponse{ForgejoURL: coordinator.ForgejoPublicURL, Runners: views, RunnerCount: len(views), TotalCapacity: len(views) * RunnerCapacity}
	for _, view := range views {
		if view.Service.Active == "active" && view.Service.Sub == "running" {
			response.ActiveListeners++
		}
	}
	return response, nil
}

func (coordinator Coordinator) create(ctx context.Context, input io.Reader) (MutationResponse, error) {
	var request CreateRequest
	if err := strictjson.Decode(input, &request); err != nil {
		return MutationResponse{}, err
	}
	if request.Provider == ProviderForgejo {
		request.RegistrationURL = coordinator.ForgejoURL
		if request.RegistrationURL == "" {
			request.RegistrationURL = BundledForgejoURL
		}
	}
	if err := request.Validate(); err != nil {
		return MutationResponse{}, err
	}
	if err := coordinator.Lifecycle.Create(ctx, request); err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{OK: true}, nil
}

func (coordinator Coordinator) mutate(ctx context.Context, action string, input io.Reader) (MutationResponse, error) {
	var request RunnerRequest
	if err := strictjson.Decode(input, &request); err != nil {
		return MutationResponse{}, err
	}
	if err := ValidateID(request.ID); err != nil {
		return MutationResponse{}, err
	}
	var err error
	switch action {
	case "start":
		err = coordinator.Lifecycle.Start(ctx, request.ID)
	case "stop":
		err = coordinator.Lifecycle.Stop(ctx, request.ID)
	case "restart":
		err = coordinator.Lifecycle.Restart(ctx, request.ID)
	case "remove":
		err = coordinator.Lifecycle.Remove(ctx, request.ID)
	}
	if err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{OK: true}, nil
}
