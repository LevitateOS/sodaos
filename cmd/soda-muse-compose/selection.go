package main

import (
	"encoding/hex"
	"errors"
	"strings"
)

// Only inspect immutable IDs returned by this Compose project's ps command.
func selectComposeChild(output, service string, inspect func(string) (string, error)) (string, error) {
	child := ""
	for _, id := range strings.Fields(output) {
		if !immutableComposeID(id) {
			return "", errors.New("invalid Compose container identity")
		}
		label, err := inspect(id)
		if err != nil {
			return "", errors.New("compose service attribution failed")
		}
		if strings.TrimSpace(label) != id+" "+service {
			continue
		}
		if child != "" {
			return "", errors.New("exactly one immutable Compose container required")
		}
		child = id
	}
	if child == "" {
		return "", errors.New("requested Compose service container missing")
	}
	return child, nil
}

func immutableComposeID(id string) bool {
	_, err := hex.DecodeString(id)
	return len(id) == 64 && err == nil
}
