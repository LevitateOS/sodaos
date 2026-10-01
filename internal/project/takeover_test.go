package project

import (
	"strings"
	"testing"
)

func TestFactoryTakeoverValidates(t *testing.T) {
	run := strings.Repeat("a", 32)
	in := FactoryTakeover{Project: "p765432109876543210987654", ID: run, Member: "alice"}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []FactoryTakeover{
		{Project: "nope", ID: run, Member: "alice"},
		{Project: in.Project, ID: "short", Member: "alice"},
		{Project: in.Project, ID: run, Member: "Root"},
		{Project: in.Project, ID: run, Member: "root"},
		{Project: in.Project, ID: run, Member: "../../etc"},
		{Project: in.Project, ID: run, Member: ""},
	} {
		if bad.Validate() == nil {
			t.Fatalf("takeover admitted: %+v", bad)
		}
	}
}

func TestTakeoverDestinationDerivesMemberPath(t *testing.T) {
	run := strings.Repeat("a", 32)
	if got := TakeoverDestination("alice", run); got != "/home/alice/factory-takeover/"+run {
		t.Fatalf("destination %q", got)
	}
	if TakeoverDestination("root", run) != "" || TakeoverDestination("alice", "short") != "" || TakeoverDestination("../x", run) != "" {
		t.Fatal("destination derived for invalid identities")
	}
}

func TestTakeoverResultBindsDestination(t *testing.T) {
	run := strings.Repeat("a", 32)
	result := TakeoverResult{ID: run, Project: "p765432109876543210987654", Member: "alice", Destination: "/home/alice/factory-takeover/" + run}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	moved := result
	moved.Destination = "/home/alice/factory-takeover/" + strings.Repeat("b", 32)
	if moved.Validate() == nil {
		t.Fatal("takeover result for another run admitted")
	}
	foreign := result
	foreign.Destination = "/home/bob/factory-takeover/" + run
	if foreign.Validate() == nil {
		t.Fatal("takeover result in another home admitted")
	}
}

func TestTakeoverExclusions(t *testing.T) {
	if !TakeoverExcluded(".git") || !TakeoverExcluded(".soda-home") {
		t.Fatal("credential-adjacent entries not excluded")
	}
	if TakeoverExcluded("src") || TakeoverExcluded(".gitignore") || TakeoverExcluded("") {
		t.Fatal("retained work excluded")
	}
}

func TestTakeoverSourcePinsFixedLayout(t *testing.T) {
	prep := "f765432109876543210987654"
	if !TakeoverSource("/home/soda-coder/checkouts/"+prep, RoleCoder, prep) {
		t.Fatal("fixed checkout source refused")
	}
	if TakeoverSource("/home/soda-coder/checkouts/"+prep+"/../x", RoleCoder, prep) {
		t.Fatal("checkout escape admitted")
	}
	if TakeoverSource("/home/soda-coder/checkouts/"+prep, "alice", prep) {
		t.Fatal("non-role source admitted")
	}
}
