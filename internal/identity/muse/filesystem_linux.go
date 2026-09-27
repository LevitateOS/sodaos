package muse

import (
	"errors"

	"golang.org/x/sys/unix"
)

func privateTmpfs(root string) error {
	var fs unix.Statfs_t
	if err := unix.Statfs(root, &fs); err != nil {
		return err
	}
	if int64(fs.Type) != 0x01021994 {
		return errors.New("enrollment root must be tmpfs")
	}
	return nil
}
