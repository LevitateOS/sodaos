// Helpers retained for the kept integration tests after unit-test removal.
package control_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

type nativeST10Config struct {
	nativeST09Config
	ReviewerID        int64  `json:"reviewer_id"`
	ReviewerTokenID   int64  `json:"reviewer_token_id"`
	ReviewerTokenFile string `json:"reviewer_token_file"`
}

func nativeReviewObserved(t *testing.T, r *forgejo.Reviewer, w factory.ReviewWork) factory.ReviewWork {
	t.Helper()
	for i := 0; i < 40; i++ {
		observed, err := r.ObserveReview(context.Background(), w)
		if err == nil {
			w.NativeRev = observed.NativeRev
			w.NotAfter = time.Now().Add(5 * time.Minute).Unix()
			return w
		}
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || (wait.Reason != "native_busy" && wait.Reason != "revision_moved") {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("native review observation did not settle")
	return factory.ReviewWork{}
}
