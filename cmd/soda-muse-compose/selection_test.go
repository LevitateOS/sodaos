package main

import (
	"errors"
	"strings"
	"testing"
)

func TestComposeSelectsOnlyRequestedServiceWithinProject(t *testing.T) {
	first, second := strings.Repeat("a", 64), strings.Repeat("b", 64)
	child, err := selectComposeChild(first+"\n"+second+"\n", "development", func(id string) (string, error) {
		if id == first {
			return id + " database\n", nil
		}
		return id + " development\n", nil
	})
	if err != nil || child != second {
		t.Fatalf("wrong service selected: %q %v", child, err)
	}
}

func TestComposeRejectsAmbiguousOrUnconfirmedIdentity(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, output := range []string{"", "--all", id + "\n" + id} {
		if _, err := selectComposeChild(output, "development", func(value string) (string, error) { return value + " development", nil }); err == nil {
			t.Fatal("unconfirmed unique container admitted")
		}
	}
	if _, err := selectComposeChild(id, "development", func(string) (string, error) { return "", errors.New("runtime unavailable") }); err == nil {
		t.Fatal("failed native observation admitted")
	}
}
