package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// JEV (TypeSafe System One) is the installation's judgment provider: Factory
// routes posts, chooses work and answers worker consultations with it. It is
// connected once for the whole installation by the operator, from the same AI
// accounts dialog as Codex and Muse. The key is checked with one small JEV
// call before it is kept, and is read fresh on every use, so connecting or
// disconnecting needs no restart.

func (s *API) jevConnected() bool {
	return control.TypeSafeKeyReadable(s.Config.TypeSafeAPIKeyFile)
}

func (s *API) jevManager(v store.Session) bool {
	return s.Config.OperatorID > 0 && v.User.ID == s.Config.OperatorID
}

type jevStatus struct {
	Connected bool `json:"connected"`
	CanManage bool `json:"can_manage"`
}

func (s *API) apiJEVStatus(w http.ResponseWriter, r *http.Request, v store.Session) {
	auth.JSONResponse(w, 200, jevStatus{Connected: s.jevConnected(), CanManage: s.jevManager(v)})
}

func (s *API) apiJEVConnect(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.jevManager(v) {
		auth.JSONError(w, 403, "forbidden", "Only the installation operator can connect JEV.")
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 8192))
	if err != nil || json.Unmarshal(data, &body) != nil {
		auth.JSONError(w, 400, "invalid_request", "Send the JEV key as JSON.")
		return
	}
	key := strings.TrimSpace(body.Key)
	if key == "" || len(key) > 4096 || strings.ContainsAny(key, " \t\r\n") {
		auth.JSONError(w, 400, "invalid_key", "Paste the whole TypeSafe API key.")
		return
	}
	path := s.Config.TypeSafeAPIKeyFile
	candidate, err := os.CreateTemp(filepath.Dir(path), ".typesafe-*.key")
	if err != nil {
		auth.JSONError(w, 503, "jev_storage_unavailable", "This server has nowhere to keep the JEV key. Finish Factory setup first.")
		return
	}
	defer func() { _ = os.Remove(candidate.Name()) }()
	if _, err = candidate.WriteString(key); err == nil {
		err = candidate.Chmod(0o600)
	}
	if closeErr := candidate.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		auth.JSONError(w, 503, "jev_storage_unavailable", "The JEV key could not be saved. Try again.")
		return
	}
	if err = checkJEVKey(r.Context(), candidate.Name(), s.Config.TypeSafeModel); err != nil {
		auth.JSONError(w, 422, "jev_key_rejected", "JEV did not accept this key. Check it on typesafe.ai and paste it again.")
		return
	}
	if err = os.Rename(candidate.Name(), path); err != nil {
		auth.JSONError(w, 503, "jev_storage_unavailable", "The JEV key could not be saved. Try again.")
		return
	}
	auth.JSONResponse(w, 200, jevStatus{Connected: true, CanManage: true})
}

func (s *API) apiJEVDisconnect(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.jevManager(v) {
		auth.JSONError(w, 403, "forbidden", "Only the installation operator can disconnect JEV.")
		return
	}
	if err := os.Remove(s.Config.TypeSafeAPIKeyFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		auth.JSONError(w, 503, "jev_storage_unavailable", "The JEV key could not be removed. Try again.")
		return
	}
	auth.JSONResponse(w, 200, jevStatus{Connected: false, CanManage: true})
}

// checkJEVKey asks JEV one trivial yes/no question with the candidate key.
func checkJEVKey(ctx context.Context, keyFile, model string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	judge := &control.TypeSafeSystemOne{APIKeyFile: keyFile, Model: model}
	_, err := judge.Evaluate(ctx, "noul", map[string]any{"check": "SodaOS is confirming this key works."},
		"Is the supplied state a key check?", map[string]any{"true": "It is a key check.", "false": "It is not."})
	return err
}
