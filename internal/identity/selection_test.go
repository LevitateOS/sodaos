package identity

import "testing"

func TestSelectMuseConnectionRequiresCurrentAuthorizedProvider(t *testing.T) {
	connections := []Connection{{ID: "codex", ProviderID: Codex, State: Ready}, {ID: "muse", ProviderID: Muse, State: Ready}, {ID: "expired", ProviderID: Muse, State: Reauth}}
	selected, err := SelectMuseConnection(connections, "")
	if err != nil || selected != "muse" {
		t.Fatal(selected, err)
	}
	if _, err = SelectMuseConnection(connections, "expired"); err == nil {
		t.Fatal("selected unavailable connection")
	}
	if _, err = SelectMuseConnection(connections, "unrelated"); err == nil {
		t.Fatal("selected unrelated connection")
	}
	connections = append(connections, Connection{ID: "second", ProviderID: Muse, State: Ready})
	if _, err = SelectMuseConnection(connections, ""); err == nil {
		t.Fatal("arbitrarily selected subscription")
	}
	selected, err = SelectMuseConnection(connections, "second")
	if err != nil || selected != "second" {
		t.Fatal(selected, err)
	}
}
