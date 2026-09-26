//go:build !linux

package workspace

import "errors"

func (w *Runtime) CheckCapacity() error { return errors.New("factory execution requires native Linux") }
