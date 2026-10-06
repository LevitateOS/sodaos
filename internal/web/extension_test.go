package web

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestExtensionUsernamePolicy(t *testing.T) {
	application, closeTransport := Extension("/unused-soda-service")
	defer closeTransport()
	policy := application.Policies[extensions.PolicyForgejoUsername]
	if policy == nil {
		t.Fatal("Soda extension did not register the username policy")
	}
	manifest, err := extensions.LoadManifest(filepath.Join("..", "..", "system", "containers", "extension"))
	if err != nil || !slices.Contains(manifest.Policies, extensions.PolicyForgejoUsername) {
		t.Fatalf("Soda package does not declare its username policy: %v", err)
	}
	for _, test := range []struct {
		name, operation, username, userID, reason string
		allowed                                   bool
	}{
		{"create", "create", "soda-tester", "", "", true},
		{"rename", "rename", "member_2", "42", "", true},
		{"max-length", "create", strings.Repeat("a", 31), "", "", true},
		{"reserved", "create", "root", "", "unsupported_linux_login", false},
		{"uppercase", "rename", "Root", "42", "unsupported_linux_login", false},
		{"invalid-prefix", "create", "-tester", "", "unsupported_linux_login", false},
		{"too-long", "create", strings.Repeat("a", 32), "", "unsupported_linux_login", false},
		{"unknown-operation", "delete", "soda-tester", "42", "invalid_request", false},
		{"create-with-id", "create", "soda-tester", "42", "invalid_request", false},
		{"rename-without-id", "rename", "soda-tester", "", "invalid_request", false},
		{"zero-id", "rename", "soda-tester", "0", "invalid_request", false},
		{"leading-zero-id", "rename", "soda-tester", "042", "invalid_request", false},
		{"overflow-id", "rename", "soda-tester", "9223372036854775808", "invalid_request", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			decision, err := policy(context.Background(), extensions.PolicyRequest{Operation: test.operation, Username: test.username, UserID: test.userID})
			if err != nil || decision.Allowed != test.allowed || decision.ReasonCode != test.reason {
				t.Fatalf("decision = %+v, %v; want allowed=%t reason=%q", decision, err, test.allowed, test.reason)
			}
		})
	}
}
