package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAccessRequest(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	request := map[string]any{
		"target":         "test-target-01",
		"revision":       strings.Repeat("a", 40),
		"project_id":     "p" + strings.Repeat("b", 24),
		"subnet":         "10.89.0.0/24",
		"browser_result": filepath.Join(root, "browser.json"),
		"host_key_file":  filepath.Join(root, "hostkey"),
		"users": []any{
			map[string]any{"id": "1001", "login": "alice", "key_file": filepath.Join(root, "alice-key"), "administrator": true},
			map[string]any{"id": "1002", "login": "bob", "key_file": filepath.Join(root, "bob-key"), "administrator": false},
		},
	}
	if mutate != nil {
		mutate(request)
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeAccessSupport(t *testing.T, root string) {
	t.Helper()
	project := "p" + strings.Repeat("b", 24)
	revision := strings.Repeat("a", 40)
	user := func(id, login string) map[string]any {
		return map[string]any{"id": id, "login": login, "connection": map[string]any{
			"environment": map[string]any{"id": project, "running": true, "ip": "10.89.0.11"},
			"host_key":    testHostKey(),
			"fingerprint": testFingerprint(),
		}}
	}
	browser := map[string]any{
		"target": "test-target-01", "revision": revision, "outcome": "passed-scoped-journey",
		"access": map[string]any{
			"native_join_confirmed": true, "reservation_id": project,
			"users": []any{user("1001", "alice"), user("1002", "bob")},
		},
	}
	data, err := json.Marshal(browser)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "browser.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hostkey"), []byte(testHostKey()+" test-comment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "alice-key"), []byte("alice-private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bob-key"), []byte("bob-private"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAccessRequest(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "test-target-01")
	root := accessFixture(t)
	writeAccessSupport(t, root)
	writeAccessRequest(t, root, nil)
	request, err := loadAccessRequest(root)
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if request.publicKey != testHostKey() || len(request.users) != 2 || request.sshConfig != "/dev/null" {
		t.Errorf("request = %+v", request)
	}
	cases := map[string]func(map[string]any){
		"bad target":      func(m map[string]any) { m["target"] = "bad target!" },
		"bad revision":    func(m map[string]any) { m["revision"] = "short" },
		"bad project":     func(m map[string]any) { m["project_id"] = "x" },
		"public subnet":   func(m map[string]any) { m["subnet"] = "11.0.0.0/8" },
		"extra key":       func(m map[string]any) { m["extra"] = 1 },
		"missing users":   func(m map[string]any) { delete(m, "users") },
		"one user":        func(m map[string]any) { m["users"] = m["users"].([]any)[:1] },
		"swapped admin":   func(m map[string]any) { m["users"].([]any)[0].(map[string]any)["administrator"] = false },
		"duplicate ids":   func(m map[string]any) { m["users"].([]any)[1].(map[string]any)["id"] = "1001" },
		"missing browser": func(m map[string]any) { m["browser_result"] = "/nonexistent/browser.json" },
	}
	for name, mutate := range cases {
		root := accessFixture(t)
		writeAccessSupport(t, root)
		writeAccessRequest(t, root, mutate)
		if _, err := loadAccessRequest(root); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "other-target")
	valid := accessFixture(t)
	writeAccessSupport(t, valid)
	writeAccessRequest(t, valid, nil)
	if _, err := loadAccessRequest(valid); err == nil {
		t.Error("environment mismatch accepted")
	}
}
