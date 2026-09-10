package runners

import (
	"context"
	"fmt"
	"io"

	"github.com/levitateos/sodaos/internal/strictjson"
)

type Lifecycle interface {
	Create(context.Context, CreateRequest) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	Remove(context.Context, string) error
}

// Operations is the fixed native protocol for the root:soda socket service.
// The caller enforces transport authority; the web adapter authorizes the operator;
// this layer never accepts arbitrary commands, accounts, paths or unit names.
type Operations struct {
	Local     LocalReader
	Lifecycle Lifecycle
}

func (operations Operations) Execute(ctx context.Context, action string, input io.Reader) (any, error) {
	if action == "list" {
		var request EmptyRequest
		if err := strictjson.Decode(input, &request); err != nil {
			return nil, err
		}
		return operations.Local.List(ctx)
	}
	if action == "create" {
		return operations.create(ctx, input)
	}
	return operations.mutate(ctx, action, input)
}

func (operations Operations) create(ctx context.Context, input io.Reader) (MutationResponse, error) {
	var request CreateRequest
	if err := strictjson.Decode(input, &request); err != nil {
		return MutationResponse{}, err
	}
	if err := request.Validate(); err != nil {
		return MutationResponse{}, err
	}
	if err := operations.Lifecycle.Create(ctx, request); err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{OK: true}, nil
}

func (operations Operations) mutate(ctx context.Context, action string, input io.Reader) (MutationResponse, error) {
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
		err = operations.Lifecycle.Start(ctx, request.ID)
	case "stop":
		err = operations.Lifecycle.Stop(ctx, request.ID)
	case "restart":
		err = operations.Lifecycle.Restart(ctx, request.ID)
	case "remove":
		err = operations.Lifecycle.Remove(ctx, request.ID)
	default:
		return MutationResponse{}, fmt.Errorf("unsupported runner helper action %q", action)
	}
	if err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{OK: true}, nil
}
