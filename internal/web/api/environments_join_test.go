package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func TestJoinRejectsNonProjectLoginBeforeProvisioning(t *testing.T) {
	for _, login := range []string{"root", "Root", "-tester", "tester.name"} {
		t.Run(login, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/api/environments/p0123456789abcdef01234567/join", nil)
			_, _, ok := new(API).admitNewJoin(w, r, store.Session{}, store.Project{Ready: true}, repositoryAccess{actor: forgejo.User{Login: login}}, "none")
			if ok || w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("invalid login %q reached provisioning: status=%d", login, w.Code)
			}
		})
	}
}
