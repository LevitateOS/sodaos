//go:build !linux

package muse

import "errors"

func privateTmpfs(string) error { return errors.New("identity enrollment requires Linux tmpfs") }
