package main

import (
	"strings"
	"testing"
)

func TestRefreshRequiresActualTailnetListener(t *testing.T) {
	raw := `[{"Config":{"Env":["SECRET=never-log","FORGEJO__server__SSH_DOMAIN=old"]},"State":{"Running":true},"HostConfig":{"PortBindings":{"22/tcp":[{"HostIp":"100.64.0.1","HostPort":"2222"}]}}}]`
	domain, running, err := publishedState([]byte(raw), "100.64.0.1")
	if err != nil || domain != "old" || !running {
		t.Fatal(domain, running, err)
	}
	_, _, err = publishedState([]byte(strings.Replace(raw, "100.64.0.1", "192.168.1.10", 1)), "100.64.0.1")
	if err == nil || strings.Contains(err.Error(), "never-log") {
		t.Fatal("missing private listener guard or credential leakage", err)
	}
}
