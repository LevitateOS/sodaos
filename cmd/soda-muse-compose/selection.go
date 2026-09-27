package main

import (
	"encoding/hex"
	"errors"
	"strings"
)

// Resolve only this Compose project's handles to native immutable IDs.
func selectComposeChild(output, service string, inspect func(string) (string, error)) (string, error) {
	child := ""
	for _, id := range strings.Fields(output) {
		if !composeContainerHandle(id) {
			return "", errors.New("invalid Compose container identity")
		}
		label, err := inspect(id)
		if err != nil {
			return "", errors.New("compose service attribution failed")
		}
		full, err := composeObservedChild(id, service, label)
		if err != nil {
			return "", err
		}
		if full == "" {
			continue
		}
		if child != "" {
			return "", errors.New("exactly one immutable Compose container required")
		}
		child = full
	}
	if child == "" {
		return "", errors.New("requested Compose service container missing")
	}
	return child, nil
}

func composeContainerHandle(id string) bool {
	_, err := hex.DecodeString(id)
	return (len(id) == 12 || len(id) == 64) && err == nil
}

func composeObservedChild(handle, service, output string) (string, error) {
	parts := strings.Fields(output)
	if len(parts) != 2 || !immutableComposeID(parts[0]) || !strings.HasPrefix(parts[0], handle) {
		return "", errors.New("native Compose container identity unconfirmed")
	}
	if parts[1] != service {
		return "", nil
	}
	return parts[0], nil
}

func immutableComposeID(id string) bool {
	_, err := hex.DecodeString(id)
	return len(id) == 64 && err == nil
}
