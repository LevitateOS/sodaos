package main

import (
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/web"
)

func TestOperatorEndpointRequiresExplicitPeer(t *testing.T) {
	app := web.New(config.Config{ForgejoURL: "https://forgejo.example.test"}, nil)
	server, listener, err := operatorEndpoint("", 0, app)
	if err != nil || server != nil || listener != nil {
		t.Fatal("empty operator socket served")
	}
	if _, _, err = operatorEndpoint("/tmp/operator.sock", -1, app); err == nil {
		t.Fatal("operator socket without explicit peer UID admitted")
	}
}
