package runners

import "context"

// Create cannot register execution capacity until isolated jobs are supported.
func (*Native) Create(context.Context, CreateRequest) error { return ErrUnavailable }
