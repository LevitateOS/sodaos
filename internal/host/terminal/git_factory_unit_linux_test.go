//go:build linux

package terminal

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestFactoryGitUnitPinsInvocationAndHostMainPID(t *testing.T) {
	id := strings.Repeat("a", 32)
	b := identity.Binding{Kind: identity.Factory, Scope: "git-factory", ID: id, Project: "p" + strings.Repeat("b", 24), ContainerID: strings.Repeat("c", 64), Generation: 1, UID: 1000, GID: 1000, ChildID: "3210", InvocationID: strings.Repeat("d", 32)}
	unit := parseFactoryGitUnit([]byte("ActiveState=active\nInvocationID=" + b.InvocationID + "\nControlGroup=/user.slice/soda-git-factory-" + id + ".service\nMainPID=3210\n"))
	if !validFactoryGitBinding(b) || !factoryGitUnitMatches(unit, b) {
		t.Fatal("valid factory Git incarnation denied")
	}
	for _, change := range []func(*factoryGitUnit){
		func(u *factoryGitUnit) { u.state = "inactive" },
		func(u *factoryGitUnit) { u.invocation = strings.Repeat("e", 32) },
		func(u *factoryGitUnit) { u.mainPID++ },
		func(u *factoryGitUnit) { u.group = "/user.slice/other.service" },
	} {
		other := unit
		change(&other)
		if factoryGitUnitMatches(other, b) {
			t.Fatal("replaced unit accepted")
		}
	}
	b.Scope = "git"
	if validFactoryGitBinding(b) {
		t.Fatal("write-capable scope accepted")
	}
}
