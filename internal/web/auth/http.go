package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/levitateos/sodaos/internal/strictjson"
)

// APIBodyLimit is the maximum JSON request body accepted by protected APIs.
const APIBodyLimit = 65536

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSONResponse writes a JSON body after marshaling so partial writes cannot look successful.
func JSONResponse(w http.ResponseWriter, status int, value any) {
	// Encode before committing headers so malformed server values cannot produce
	// an apparently successful, truncated JSON response.
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"internal_error","message":"Cannot encode response."}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// JSONError writes a standard API error envelope.
func JSONError(w http.ResponseWriter, status int, code, message string) {
	JSONResponse(w, status, struct {
		Error apiError `json:"error"`
	}{apiError{Code: code, Message: message}})
}

// AllowAPIMethod rejects unsupported methods with a JSON 405.
func AllowAPIMethod(w http.ResponseWriter, r *http.Request, methods []string) bool {
	allowed := false
	for _, method := range methods {
		allowed = allowed || r.Method == method
	}
	if !allowed {
		w.Header().Set("Allow", strings.Join(methods, ", "))
		JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP method not supported.")
		return false
	}
	return true
}

// DecodeAPIObject reads one strict JSON object within APIBodyLimit.
func DecodeAPIObject(w http.ResponseWriter, r *http.Request, out any) bool {
	body, err := io.ReadAll(r.Body)
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		JSONError(w, http.StatusRequestEntityTooLarge, "body_too_large", "Request exceeds 64 KiB.")
		return false
	}
	body = bytes.TrimSpace(body)
	if err != nil || len(body) == 0 || body[0] != '{' {
		JSONError(w, http.StatusBadRequest, "invalid_json", "Send exactly one JSON object.")
		return false
	}
	// Keep the outer 64-KiB limit and sanitized errors, while rejecting ambiguous
	// repeated fields and invalid UTF-8 through the existing strict decoder.
	if err = strictjson.Decode(bytes.NewReader(body), out); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid_json", "JSON fields or values are invalid.")
		return false
	}
	return true
}
