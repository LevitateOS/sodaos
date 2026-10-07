package control_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	nativeMust(fx.t, seedStagedDependencyEdge(fx.cfg.FountainDB, fx.cfg.CreatorID, blockedID, blockerID))
}
