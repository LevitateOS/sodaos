package control_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/web/api"
)

// st15JourneyChooser answers the demo's funded judgments the way the
// prompt-entry serving journey does: provisional plans while scope is
// missing, incremental acceptance once admitted, and the funded
// continue behind a sealed reassessment. It answers nothing else, so
// any funded reassessment decision behind this chooser is a continue.
type st15JourneyChooser struct{}

func (st15JourneyChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	switch in := state.(type) {
	case factory.PlanningInput:
		selected := "incremental_acceptance"
		if len(in.Missing) > 0 {
			selected = "plan_stable_scope"
		}
		answer := factory.PriorityAnswer{Selected: selected, Model: "st15-synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
		for id := range choices {
			answer.Probabilities[id] = 0
		}
		answer.Probabilities[selected] = 1
		return answer, nil
	case control.ReassessmentDecisionState:
		if len(in.Evidence) == 0 || !in.Local.Sealed() || in.Completion.ID == "" {
			return factory.PriorityAnswer{}, fmt.Errorf("st15 reassessment undecidable without sealed substance")
		}
		for _, body := range in.Evidence {
			if err := body.Validate(); err != nil {
				return factory.PriorityAnswer{}, err
			}
		}
		if _, ok := choices[factory.ReassessmentOptionContinue].(map[string]any); !ok {
			return factory.PriorityAnswer{}, fmt.Errorf("st15 reassessment offers no continue")
		}
		share := 0.9
		decided := factory.PriorityAnswer{
			Selected: factory.ReassessmentOptionContinue, Confidence: share,
			Probabilities: map[string]float64{}, Model: "st15-synthetic-model",
		}
		for id := range choices {
			decided.Probabilities[id] = (1 - share) / float64(len(choices)-1)
		}
		decided.Probabilities[factory.ReassessmentOptionContinue] = share
		return decided, nil
	default:
		return factory.PriorityAnswer{}, fmt.Errorf("st15 demo answers no judgment for %T", state)
	}
}

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
	// The D13 fence holds the same-repository dependant behind A's merged
	// completion until its funded reassessment decision releases. The
	// demo scripts that decision deterministically: sealed native
	// substance judges continue, opening the fence for B after the
	// release. The inventory enumerator is the production snapshot
	// source, so the seal carries the real fixture issues.
	coord.CompletionEnumerator = api.NewCompletionSnapshotSource(observer)
	coord.Store.PriorityCallBudget = 100
	coord.Planning = &control.PlanningJudgments{Journal: fx.db, Funding: fx.db, Chooser: &st15JourneyChooser{}}
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
		"issue":      map[string]any{"number": issue.Number},
	})
	fx.t.Logf("ST15 intake opened #%d: %v", issue.Number, outcome)
}

func (fx *st15Fixture) commentHint(index int64) {
	fx.t.Helper()
	outcome := fx.postIntake("issue_comment", map[string]any{
		"repository": map[string]any{"id": fx.cfg.Repository},
		"issue":      map[string]any{"number": index},
	})
	fx.t.Logf("ST15 intake comment #%d: %v", index, outcome)
}
