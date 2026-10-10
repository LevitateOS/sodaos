package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

func TestLifecycleRetryIneligibleIsConflict(t *testing.T) {
	w := httptest.NewRecorder()
	lifecycleControlError(w, fmt.Errorf("retry: %w", store.ErrExplicitRetryIneligible))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"code":"retry_ineligible"`) {
		t.Fatalf("retry refusal = %d %s", w.Code, w.Body.String())
	}
}
