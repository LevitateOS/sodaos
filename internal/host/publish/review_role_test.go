package publish

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestST10PublisherRejectsReviewerBundle(t *testing.T) {
	config, request := candidateFixture(t, "README.md")
	request.Run.Role = project.RoleCoder
	if err := config.ValidateCandidate(context.Background(), request); err != nil {
		t.Fatalf("coder candidate: %v", err)
	}
	request.Run.Role = project.RoleReviewer
	if err := config.ValidateCandidate(context.Background(), request); err == nil {
		t.Fatal("reviewer bundle accepted by publisher")
	}
}
