package api

import (
	"testing"
)

func TestParseFeedSubscriptionBody(t *testing.T) {
	sub, expected, err := parseFeedSubscriptionBody([]byte(`{"repositories":[7,9],"kinds":["record"],"expected_revision":2}`), "reader-1")
	if err != nil {
		t.Fatal(err)
	}
	if sub.Subscriber != "reader-1" || expected != 2 || sub.Revision != 3 {
		t.Fatalf("subscription: %+v expected=%d", sub, expected)
	}
	if len(sub.Repositories) != 2 || len(sub.Kinds) != 1 {
		t.Fatalf("scope lost: %+v", sub)
	}
	for _, bad := range []struct {
		name string
		body string
		user string
	}{
		{"malformed", "{", "reader-1"},
		{"empty subscriber", `{"repositories":[7],"expected_revision":0}`, ""},
		{"negative revision", `{"repositories":[7],"expected_revision":-1}`, "reader-1"},
		{"empty scope", `{"repositories":[],"expected_revision":0}`, "reader-1"},
		{"bad repository", `{"repositories":[0],"expected_revision":0}`, "reader-1"},
		{"bad kind", `{"repositories":[7],"kinds":["digest"],"expected_revision":0}`, "reader-1"},
	} {
		if _, _, err := parseFeedSubscriptionBody([]byte(bad.body), bad.user); err == nil {
			t.Fatalf("body admitted: %s", bad.name)
		}
	}
}
