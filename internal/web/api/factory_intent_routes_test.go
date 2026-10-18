package api

import (
	"net/http/httptest"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestIntentRoutesUseOnlyAdmittedProductMethods(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		want         bool
	}{
		{"GET", "/factory/departments", true},
		{"POST", "/factory/departments", true},
		{"DELETE", "/factory/departments", false},
		{"POST", "/factory/intents", true},
		{"GET", "/factory/intents", true},
		{"GET", "/factory/intents/d111111111111111111111111", true},
		{"POST", "/factory/intents/d111111111111111111111111/explore", true},
		{"POST", "/factory/intents/d111111111111111111111111/generate", true},
		{"POST", "/factory/intents/d111111111111111111111111/plan", true},
		{"GET", "/factory/intents/d111111111111111111111111/generation", true},
		{"GET", "/factory/intents/d111111111111111111111111/history", true},
		{"POST", "/factory/intents/d111111111111111111111111/history", false},
		{"GET", "/factory/intents/d111111111111111111111111/allowance", true},
		{"POST", "/factory/intents/d111111111111111111111111/allowance", false},
		{"GET", "/factory/intents/d111111111111111111111111/overview", true},
		{"POST", "/factory/intents/d111111111111111111111111/overview", false},
		{"GET", "/factory/intents/d111111111111111111111111/revisions", true},
		{"POST", "/factory/intents/d111111111111111111111111/revisions", false},
		{"POST", "/factory/intent-allowances", true},
		{"GET", "/factory/intent-allowances", false},
		{"POST", "/factory/intents/d111111111111111111111/generation", false},
		{"POST", "/factory/intents/d111111111111111111111111/execute", false},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		if got := extensionProductRoute(r); got != tc.want {
			t.Fatalf("%s %s = %v", tc.method, tc.path, got)
		}
	}
	for _, path := range []string{"/api/factory/departments", "/api/factory/intents/d111111111111111111111111/generation"} {
		if !extensionProductContribution(path, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}) {
			t.Fatal("missing native Spaces contribution")
		}
	}
}
