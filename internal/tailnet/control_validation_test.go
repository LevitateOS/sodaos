package tailnet

import (
	"strings"
	"testing"
)

func TestHostAdmissionUsesCanonicalUnicastAddresses(t *testing.T) {
	host := HostView{
		Tailnet:      "soda",
		Revision:     strings.Repeat("a", 64),
		State:        "Running",
		DNSName:      "host.example.ts.net.",
		Addresses:    []string{"100.64.0.1"},
		Peers:        []Peer{{ID: "peer", DNSName: "peer.example.ts.net", Addresses: []string{"fd7a:115c:a1e0::1"}}},
		HealthIssues: 0,
	}
	if err := host.Validate(); err != nil {
		t.Fatal(err)
	}

	for _, address := range []string{"100.64.0.01", "127.0.0.1"} {
		invalid := host
		invalid.Addresses = []string{address}
		if err := invalid.Validate(); err == nil {
			t.Fatalf("accepted invalid address %q", address)
		}
	}
	tooMany := host
	tooMany.Addresses = make([]string, 17)
	for i := range tooMany.Addresses {
		tooMany.Addresses[i] = "100.64.0.1"
	}
	if err := tooMany.Validate(); err == nil {
		t.Fatal("accepted more than sixteen addresses")
	}
	badPeer := host
	badPeer.Peers = []Peer{{ID: "peer", Addresses: []string{"100.64.0.2"}, DNSName: "bad..name.ts.net"}}
	if err := badPeer.Validate(); err == nil {
		t.Fatal("accepted invalid peer DNS name")
	}
	badPeer = host
	badPeer.Peers = []Peer{{ID: "bad\npeer", Addresses: []string{"100.64.0.2"}}}
	if err := badPeer.Validate(); err == nil {
		t.Fatal("accepted invalid peer ID")
	}
}

func TestPendingHostResultAdmissionRequiresExactLoginURL(t *testing.T) {
	host := &HostView{
		Tailnet:      "soda",
		Revision:     strings.Repeat("a", 64),
		State:        "NeedsLogin",
		Addresses:    []string{},
		Peers:        []Peer{},
		HealthIssues: 0,
	}
	result := HostResult{
		Outcome: "pending",
		Host:    host,
		AuthURL: "https://login.tailscale.com/a/synthetic",
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"https://evil.test/a/secret",
		"http://login.tailscale.com/a/secret",
		"https://login.tailscale.com.evil.test/a/secret",
		"https://user@login.tailscale.com/a/secret",
		"https://login.tailscale.com/a/secret?token=secret",
		"https://login.tailscale.com/a/secret#fragment",
		"https://login.tailscale.com/a/secret%2Fother",
	} {
		invalid := result
		invalid.AuthURL = raw
		if err := invalid.Validate(); err == nil {
			t.Fatalf("accepted unsafe authentication URL %q", raw)
		}
	}
}
