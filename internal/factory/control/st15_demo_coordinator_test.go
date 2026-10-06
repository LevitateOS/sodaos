package control_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/web/api"
)

// wireCoordinator wires the journey coordinator over production readers
// and native executors, plus the in-process signed intake handler.
func (fx *st15Fixture) wireCoordinator() error {
	uid := uint32(os.Getuid())
	rest := forgejo.New(fx.cfg.FountainURL)
	fx.rest = rest
	background := forgejo.NewServiceBackground(fx.cfg.Socket, uid, "")
	observer := forgejo.NewServiceObserver(fx.cfg.Socket, uid, fx.cfg.CreatorTokenFile, rest)
	observer.ShareBackground(background)
	fx.observer, fx.bg = observer, background
	source := api.NewServiceReadinessSource(observer)
	coord := control.NewCoordinator(fx.db, hostexec.NewClient(fx.cfg.HostSocket), fx.client)
	coord.AcceptanceReads = source
	coord.Readiness = source
	coord.DispatchReads = source
	// The publisher requires an existing private root (ST09 precedent:
	// t.TempDir + 0700); a bare join path fails privateInputs and the
	// pass reports native_unavailable without attempting anything.
	publishRoot := filepath.Join(fx.scratch, "publish")
	if err := os.MkdirAll(publishRoot, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(publishRoot, 0o700); err != nil {
		return err
	}
	coord.Publication = forgejo.NewPublisher(background, rest, fx.cfg.FountainURL, publishRoot, fx.cfg.TokenFile)
	coord.Reviews = forgejo.NewReviewer(background, rest, fx.cfg.ReviewerTokenFile)
	coord.Checks = forgejo.NewCheckAssessor(background, rest, fx.cfg.TokenFile)
	coord.Merges = forgejo.NewMerger(background, rest, fx.cfg.TokenFile)
	fx.coord = coord
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	fx.intake = api.IntakeHandler{Coordinator: coord, Secret: secret}
	return nil
}

// postIntake delivers one signed native webhook to the real intake
// handler and returns the decoded outcome.
func (fx *st15Fixture) postIntake(event string, payload any) map[string]any {
	fx.t.Helper()
	body, err := json.Marshal(payload)
	nativeMust(fx.t, err)
	mac := hmac.New(sha256.New, fx.intake.Secret)
	mac.Write(body)
	request := httptest.NewRequest(http.MethodPost, "/api/factory/intake", bytes.NewReader(body))
	request.Header.Set("X-Forgejo-Delivery", "st15-"+st15RandHex(8))
	request.Header.Set("X-Forgejo-Signature", hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("X-Forgejo-Event", event)
	recorder := httptest.NewRecorder()
	fx.intake.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		fx.t.Fatalf("intake %s rejected: %d %s", event, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	var outcome map[string]any
	nativeMust(fx.t, json.Unmarshal(recorder.Body.Bytes(), &outcome))
	return outcome
}

// issueOpenedHint delivers the creation webhook for one maintainer issue.
func (fx *st15Fixture) issueOpenedHint(issue st15Issue) {
	fx.t.Helper()
	outcome := fx.postIntake("issues", map[string]any{
		"action": "opened", "number": issue.Number,
		"repository": map[string]any{"id": fx.cfg.Repository, "permissions": map[string]any{"push": true, "admin": true}},
		"sender":     map[string]any{"id": fx.cfg.CreatorID},
		"issue":      map[string]any{"index": issue.Number},
	})
	fx.t.Logf("ST15 intake opened #%d: %v", issue.Number, outcome)
}

func (fx *st15Fixture) commentHint(index int64) {
	fx.t.Helper()
	outcome := fx.postIntake("issue_comment", map[string]any{
		"repository": map[string]any{"id": fx.cfg.Repository},
		"issue":      map[string]any{"index": index},
	})
	fx.t.Logf("ST15 intake comment #%d: %v", index, outcome)
}
