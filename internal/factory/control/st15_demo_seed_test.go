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
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

func (fx *st15Fixture) st09() nativeST09Config {
	return nativeST09Config{
		TokenID: fx.cfg.TokenID, FountainURL: fx.cfg.FountainURL, Socket: fx.cfg.Socket,
		TokenFile: fx.cfg.TokenFile, Owner: fx.cfg.Owner, Repo: fx.cfg.Repo,
		Repository: fx.cfg.Repository, ActorID: fx.cfg.ActorID, BaseBranch: fx.cfg.BaseBranch,
		CreatorID: fx.cfg.CreatorID, CreatorTokenFile: fx.cfg.CreatorTokenFile,
	}
}

func (fx *st15Fixture) api(tokenFile, method, path string, body any, out any) {
	fx.t.Helper()
	nativeAPI(fx.t, fx.st09(), tokenFile, method, path, body, out)
}

func (fx *st15Fixture) git(dir string, args ...string) string {
	fx.t.Helper()
	out, err := st15Git(fx.ctx, dir, args...)
	if err != nil {
		fx.t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func (fx *st15Fixture) repoURL() string {
	token, err := os.ReadFile(fx.cfg.CreatorTokenFile)
	if err != nil {
		fx.t.Fatal(err)
	}
	return strings.Replace(fx.cfg.FountainURL, "://", "://soda-maintainer:"+strings.TrimSpace(string(token))+"@", 1) + "/" + fx.cfg.Owner + "/" + fx.cfg.Repo + ".git"
}

// seedRepository pushes the journey base: the buggy widget, its tests and
// the CI workflow the fixture statuses attest.
func (fx *st15Fixture) seedRepository() error {
	parent := fx.t.TempDir()
	dir := filepath.Join(parent, "seed")
	if _, err := st15Git(fx.ctx, parent, "clone", "-q", fx.repoURL(), dir); err != nil {
		return err
	}
	files := map[string]string{
		"widget.py":                  "def total(items):\n    return sum(items) + 1\n",
		"test_widget.py":             "from widget import total\n\n\ndef test_total_two():\n    assert total([1, 2]) == 4\n",
		".forgejo/workflows/ci.yaml": "name: st15 CI\non: [pull_request]\njobs:\n  build:\n    runs-on: st15\n    steps:\n      - run: python3 -m py_compile widget.py\n  test:\n    runs-on: st15\n    steps:\n      - run: python3 -m pytest -q\n",
	}
	// NOTE: test_widget.py pins the BUGGY total on purpose: the seeded
	// main is internally consistent (bug uncovered), and the journey fix
	// corrects both the widget and its test. The required regression path
	// tests/test_total_regression.py only enters through the correction.
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := st15Git(fx.ctx, dir, "add", "."); err != nil {
		return err
	}
	if _, err := st15Git(fx.ctx, dir, "commit", "-q", "-m", "ST15 journey base"); err != nil {
		return err
	}
	fx.sourceHead = fx.git(dir, "rev-parse", "HEAD")
	if _, err := st15Git(fx.ctx, dir, "push", "-q", "origin", "main"); err != nil {
		return err
	}
	head := fx.sourceHead
	st15Receipt(fx.t, "seed-base", map[string]any{"head": head, "files": []string{"widget.py", "test_widget.py", ".forgejo/workflows/ci.yaml"}})
	return nil
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

type st15Issue struct {
	ID     int64 `json:"id"`
	Number int64 `json:"number"`
}

func (fx *st15Fixture) createIssue(title, body string) st15Issue {
	fx.t.Helper()
	var issue st15Issue
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues",
		map[string]any{"title": title, "body": body}, &issue)
	if issue.ID <= 0 || issue.Number <= 0 {
		fx.t.Fatalf("issue not created: %+v", issue)
	}
	return issue
}

func (fx *st15Fixture) commentIssue(index int64, body string) int64 {
	fx.t.Helper()
	var comment struct {
		ID int64 `json:"id"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(index, 10)+"/comments",
		map[string]any{"body": body}, &comment)
	if comment.ID <= 0 {
		fx.t.Fatalf("comment not created on #%d", index)
	}
	return comment.ID
}

func (fx *st15Fixture) readIssue(index int64) (title, body string, id int64) {
	fx.t.Helper()
	var issue struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(index, 10), nil, &issue)
	return issue.Title, issue.Body, issue.ID
}

func (fx *st15Fixture) readComment(commentID int64) string {
	fx.t.Helper()
	var comment struct {
		Body string `json:"body"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/comments/"+strconv.FormatInt(commentID, 10), nil, &comment)
	return comment.Body
}

func (fx *st15Fixture) insertEdge(blockedID, blockerID int64) {
	fx.t.Helper()
	nativeMust(fx.t, store.SeedStagedDependencyEdge(fx.cfg.FountainDB, fx.cfg.CreatorID, blockedID, blockerID))
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
