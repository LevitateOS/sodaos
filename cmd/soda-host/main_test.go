package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPreparationFailureRequiresExplicitRetry(t *testing.T) {
	if exitStatus(nil) != 0 || exitStatus(errors.New("daemon exited")) != 1 || exitStatus(errors.Join(errTailnetPreparation, errors.New("provider refused"))) != 78 {
		t.Fatal("wrong native exit policy")
	}
	unit, err := os.ReadFile("../../appliance/services/soda-tailnet@.service")
	if err != nil || !strings.Contains(string(unit), "RestartPreventExitStatus=78\n") || !strings.Contains(string(unit), "Restart=on-failure\n") {
		t.Fatal("systemd retry policy not wired", err)
	}
}
