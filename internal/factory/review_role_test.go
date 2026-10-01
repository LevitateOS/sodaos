package factory

import (
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestST10ReviewerCannotBecomePublication(t *testing.T) {
	p := testPublication()
	if err := p.Validate(); err != nil {
		t.Fatalf("coder fixture: %v", err)
	}
	p.Role = project.RoleReviewer
	if err := p.Validate(); err == nil {
		t.Fatal("reviewer candidate accepted for publication")
	}
}
