package identity

import (
	"reflect"
	"testing"
)

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

func TestMuseNativeCommandProviderPlacement(t *testing.T) {
	for _, test := range []struct{ input, expected []string }{
		{[]string{"exec", "--model", "native/model", "prompt with spaces"}, []string{"exec", "--provider", "meta", "--model", "native/model", "prompt with spaces"}},
		{[]string{"prompt"}, []string{"--provider", "meta", "prompt"}},
		{[]string{"config", "--help"}, []string{"config", "--help"}},
	} {
		actual, err := MuseArguments(test.input)
		if err != nil || !reflect.DeepEqual(actual, test.expected) {
			t.Fatalf("native argv placement: got %v, %v; want %v", actual, err, test.expected)
		}
	}
}
