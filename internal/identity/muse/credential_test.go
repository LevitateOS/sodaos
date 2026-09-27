package muse

import (
	"os"
	"path/filepath"
	"testing"
)

const subscription = `{"schema_version":1,"providers":{"meta":{"access_token":"synthetic-oauth","api_key":"synthetic-subscription","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code","account_name":"synthetic presentation"}}}`

func TestNativeSubscriptionAndPrivateFile(t *testing.T) {
	if !CredentialValid([]byte(subscription)) {
		t.Fatal("native device subscription refused")
	}
	for _, value := range []string{
		`{"schema_version":1,"providers":{"meta":{"api_key":"synthetic-payg","mechanism":"api_key"}}}`,
		`{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://other.invalid","mechanism":"oauth","obtained_via":"device_code"}}}`,
	} {
		if CredentialValid([]byte(value)) {
			t.Fatal("non-subscription credential admitted")
		}
	}
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(subscription), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := credentialFile(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := credentialFile(path); err == nil {
		t.Fatal("public credential accepted")
	}
	link := filepath.Join(t.TempDir(), "auth.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := credentialFile(link); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestEnrollmentEnvironmentAndDevicePrompt(t *testing.T) {
	t.Setenv("META_API_KEY", "synthetic-payg")
	t.Setenv("TBH_CREDENTIAL_BACKEND", "keychain")
	env := environment("/private-enrollment")
	for _, entry := range env {
		if entry == "META_API_KEY=synthetic-payg" || entry == "TBH_CREDENTIAL_BACKEND=keychain" {
			t.Fatal("ambient authentication inherited")
		}
	}
	match := deviceURL.FindStringSubmatch("Open https://auth.meta.com/oauth/device/?code=ABCD-1234 to enroll")
	if len(match) != 2 || match[1] != "ABCD-1234" {
		t.Fatal("verified prompt not recognized")
	}
}
