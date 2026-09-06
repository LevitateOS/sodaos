package host

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"golang.org/x/crypto/ssh"
	"testing"
)

type accountExecutor struct{ body []byte }

func (e *accountExecutor) Run(_ context.Context, in []byte, command string, args ...string) ([]byte, error) {
	if args[0] == "inspect" {
		return []byte(`[{"Config":{"Labels":{"org.soda.project":"p0123456789abcdef01234567","org.soda.owner":"1"}},"State":{"Running":true},"NetworkSettings":{"Networks":{"soda-projects":{"IPAddress":"10.89.0.2"}}}}]`), nil
	}
	e.body = append([]byte(nil), in...)
	return []byte(`{"ok":true}`), nil
}
func TestNativeProjectAdministrationComesFromOwnerLabel(t *testing.T) {
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, uid := range []int64{1, 2} {
		exec := &accountExecutor{}
		d := Daemon{Exec: exec, Config: Config{Network: "soda-projects", Subnet: "10.89.0.0/24"}}
		err = d.account(context.Background(), Account{Project: "p0123456789abcdef01234567", Login: "alice", Identity: uid, Keys: []string{string(ssh.MarshalAuthorizedKey(key))}})
		if err != nil {
			t.Fatal(err)
		}
		var body struct{ Admin bool }
		if err = json.Unmarshal(exec.body, &body); err != nil {
			t.Fatal(err)
		}
		if body.Admin != (uid == 1) {
			t.Fatal("caller acquired incorrect native privilege", uid, body)
		}
	}
}
