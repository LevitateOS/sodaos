package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestProviderErrorMapsNativeInvisibleRepositoryToNotFound(t *testing.T) {
	for _, err := range []error{
		extensions.ErrRepositoryNotVisible,
		errors.Join(errors.New("repository authority"), extensions.ErrRepositoryNotVisible),
	} {
		response := httptest.NewRecorder()
		ProviderError(response, err)
		if response.Code != http.StatusNotFound {
			t.Fatalf("native repository denial status = %d, body %s", response.Code, response.Body.String())
		}
	}
}

func TestProviderErrorMapsChangedNativeSessionToConflict(t *testing.T) {
	response := httptest.NewRecorder()
	ProviderError(response, ErrNativeSessionChanged)
	if response.Code != http.StatusConflict {
		t.Fatalf("changed native session status = %d, body %s", response.Code, response.Body.String())
	}
}
