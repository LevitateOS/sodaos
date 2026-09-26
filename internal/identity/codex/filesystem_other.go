//go:build !linux

package codex

import "errors"

func privateTmpfs(string) error { return errors.New("identity enrollment requires Linux tmpfs") }
