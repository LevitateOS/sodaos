package identity

import "testing"

func TestLaunchRequestHasNoAuthority(t *testing.T) {
	for _, r := range []LaunchRequest{{CWD: "relative"}, {CWD: "/workspace", Args: []string{"a\x00b"}}, {CWD: "/workspace", TTY: true}, {CWD: "/workspace", ConfigHome: "relative"}, {Register: &NestedRegistration{ActorID: 1}, CWD: "/workspace"}} {
		if r.Validate() == nil {
			t.Fatalf("invalid request admitted: %#v", r)
		}
	}
	if err := (LaunchRequest{CWD: "/workspace", Args: []string{"--model", "meta/test", "prompt with spaces"}}).Validate(); err != nil {
		t.Fatal(err)
	}
}
