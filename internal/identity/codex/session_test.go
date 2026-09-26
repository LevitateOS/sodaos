package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("SODA_IDENTITY_CODEX_TEST_CHILD") == "1" {
		runProtocolFixture()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// The synthetic process implements only the observed enrollment account contract.
func runProtocolFixture() {
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		var req struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(scan.Bytes(), &req) != nil {
			return
		}
		switch req.Method {
		case "initialize":
			fmt.Printf("{\"id\":%d,\"result\":{}}\n", req.ID)
		case "account/login/start":
			fmt.Printf("{\"id\":%d,\"result\":{\"type\":\"chatgptDeviceCode\",\"loginId\":\"synthetic-login\",\"verificationUrl\":\"https://auth.openai.com/codex/device\",\"userCode\":\"ABCD-1234\"}}\n", req.ID)
			if os.Getenv("SODA_IDENTITY_CODEX_TEST_PENDING") != "1" {
				_ = os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "auth.json"), []byte(`{"tokens":{"refresh_token":"synthetic-enrollment"}}`), 0o600)
				fmt.Println(`{"method":"account/login/completed","params":{"loginId":"synthetic-login","success":true,"error":null}}`)
			}
		case "account/read":
			fmt.Printf("{\"id\":%d,\"result\":{\"account\":{\"type\":\"chatgpt\",\"email\":null,\"planType\":\"plus\"},\"requiresOpenaiAuth\":true}}\n", req.ID)
		case "account/login/cancel":
			fmt.Printf("{\"id\":%d,\"result\":{\"status\":\"canceled\"}}\n", req.ID)
		}
	}
}

func fixtureProvider(t *testing.T) *Provider {
	t.Helper()
	t.Setenv("SODA_IDENTITY_CODEX_TEST_CHILD", "1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return &Provider{config: Config{Binary: binary, Root: t.TempDir()}}
}

func TestManagedEnrollmentProtocolPersistsOnlyAfterProcessStop(t *testing.T) {
	p := fixtureProvider(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	session, err := p.Start(ctx, "subscription")
	if err != nil {
		t.Fatal(err)
	}
	s := session.(*Session)
	defer func() { _ = s.Close() }()
	for s.Snapshot().State == "pending" {
		select {
		case <-ctx.Done():
			t.Fatal("fixture enrollment timeout")
		case <-time.After(time.Millisecond):
		}
	}
	conn, data, err := s.Finish(ctx)
	if err != nil || conn.Plan != "plus" || conn.Email != "" || string(data) != `{"tokens":{"refresh_token":"synthetic-enrollment"}}` {
		t.Fatal("typed managed enrollment contract failed", err)
	}
	select {
	case <-s.done:
	default:
		t.Fatal("credential retained before provider process ended")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(s.root); !os.IsNotExist(err) {
		t.Fatal("credential tmpfs root not removed", err)
	}
}

func TestCancelRemovesUnfinishedEnrollment(t *testing.T) {
	p := fixtureProvider(t)
	t.Setenv("SODA_IDENTITY_CODEX_TEST_PENDING", "1")
	session, err := p.Start(t.Context(), "subscription")
	if err != nil {
		t.Fatal(err)
	}
	s := session.(*Session)
	if s.Snapshot().State != "pending" {
		t.Fatal("unexpected enrollment state")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(s.root); !os.IsNotExist(err) {
		t.Fatal("canceled credentials retained", err)
	}
}
