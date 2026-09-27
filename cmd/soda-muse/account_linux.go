//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func accountFor(actor string) error {
	if os.Geteuid() != 0 {
		return errors.New("project root required")
	}
	parsed, err := strconv.ParseInt(actor, 10, 64)
	if err != nil || parsed <= 0 {
		return errors.New("invalid account identity")
	}
	directory := "/var/lib/soda/accounts"
	if err = accountAncestors(directory); err != nil {
		return err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		found, err := accountEntry(directory, entry.Name(), actor)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
	}
	return errors.New("provisioned account missing")
}

func accountAncestors(directory string) error {
	for _, path := range []string{"/var", "/var/lib", "/var/lib/soda", directory} {
		if err := accountNode(path, true); err != nil {
			return err
		}
	}
	return nil
}

func accountEntry(directory, name, actor string) (bool, error) {
	marker := filepath.Join(directory, name)
	if accountNode(marker, false) != nil {
		return false, nil
	}
	body, err := os.ReadFile(marker)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(string(body)) != actor {
		return false, nil
	}
	account, err := user.Lookup(name)
	if err != nil {
		return false, err
	}
	return true, json.NewEncoder(os.Stdout).Encode(account)
}

func accountNode(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	native, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !accountOwner(native, info) {
		return errors.New("unsafe account record")
	}
	if directory {
		if !info.IsDir() {
			return errors.New("account ancestor is not a directory")
		}
		return nil
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 64 {
		return errors.New("unsafe account marker")
	}
	return nil
}

func accountOwner(native *syscall.Stat_t, info os.FileInfo) bool {
	return native.Uid == 0 && native.Gid == 0 && info.Mode().Perm()&0o022 == 0
}
