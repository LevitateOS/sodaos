package control_test

// Native merge proof (ST12, S-merge). Fixture candidates, PRs, reviews and
// commit statuses are prepared through ordinary native Git/REST writes plus
// the production conditional review client; merges run through the
// production conditional merge client and MergePass. Absence of
// SODA_ST12_NATIVE skips native work and is never passing evidence.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

type nativeMergePR struct {
	branch   string
	head     string
	base     string
	prID     int64
	prNumber int64
	issueID  int64
}

// nativeMergeAPI posts one native mutation, retrying only the transient
// 503 a busy host returns while a native mutation reservation is held.
// Any other refusal fails fast; reads stay on the shared helper.
func nativeMergeAPI(t *testing.T, c nativeST12Config, method, path string, body, target any) {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		nativeMust(t, err)
	}
	deadline := time.Now().Add(60 * time.Second)
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for {
		var input io.Reader
		if raw != nil {
			input = bytes.NewReader(raw)
		}
		req, err := http.NewRequestWithContext(context.Background(), method, strings.TrimRight(c.FountainURL, "/")+path, input)
		nativeMust(t, err)
		req.Header.Set("Authorization", "token "+nativeSecret(t, c.CreatorTokenFile))
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			t.Fatal("native API transport unavailable")
		}
		if response.StatusCode != http.StatusServiceUnavailable || !time.Now().Before(deadline) {
			defer response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				t.Fatalf("native fixture API %s %s: status %d", method, path, response.StatusCode)
			}
			if target != nil {
				nativeMust(t, json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target))
			}
			return
		}
		response.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

func nativeMergePublish(t *testing.T, c nativeST12Config, label string) (nativeMergePR, nativeCandidate) {
	t.Helper()
	n := nativeNewCandidate(t, c.nativeST09Config)
	branch := "st12-" + label + "-" + factory.NewID()[:12]
	nativeGitOK(t, c.nativeST09Config, n.dir, "push", nativeRepoURL(c.nativeST09Config), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls",
		map[string]any{"head": branch, "base": strings.TrimPrefix(c.BaseBranch, "refs/heads/"), "title": "ST12 " + label, "body": "Exact merge fixture", "allow_maintainer_edit": false}, &pr)
	if pr.ID <= 0 || pr.Number <= 0 {
		t.Fatal("native PR identity unconfirmed")
	}
	var issue struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(pr.Number, 10), nil, &issue)
	if issue.ID <= 0 || issue.Number != pr.Number {
		t.Fatal("native PR issue identity unconfirmed")
	}
	return nativeMergePR{branch: branch, head: n.head, base: n.base, prID: pr.ID, prNumber: pr.Number, issueID: issue.ID}, n
}

// TestNativeMergeWire pins the pull_request.merge receipt wire shape from
// the staged native producer. AdoptMerge must match this shape exactly.
func TestNativeMergeWire(t *testing.T) {
	c := loadNativeST12(t)
	p, _ := nativeMergePublish(t, c, "wire")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	payload := extensions.MergePayload{
		PullRequestNumber: p.prNumber, HeadRepositoryID: c.Repository,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch,
		ExpectedHeadOID: p.head, ExpectedBaseOID: p.base,
		Method: extensions.MergeMethodFastForwardOnly,
	}
	raw, err := json.Marshal(payload)
	nativeMust(t, err)
	record := nativeMergeSubmitRawCommitted(t, bg, c.TokenFile, "st12-native-wire", c.ActorID, c.Repository, raw)
	nativeMergeReceipt(t, "merge-wire", record)
	var fields map[string]any
	nativeMust(t, json.Unmarshal(record.Receipt, &fields))
	t.Logf("receipt wire: %v", fields)
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.head {
		t.Fatal("base tip does not carry the merged head")
	}
}

// TestNativeMergeReviewSnapshot pins the reviews-family evidence Soda
// consumes as merge review evidence: the exact type strings and commit
// binding of conditional approvals and change requests.
func TestNativeMergeReviewSnapshot(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "reviewsnap")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	outcome := nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-snapshot",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.prNumber, PRID: p.prID,
		IssueID: p.issueID, PRAuthorID: c.CreatorID,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Event: "APPROVED", Body: "ST12 snapshot probe approval.",
	})
	nativeMergeReceipt(t, "reviewsnap-submit", outcome)
	for _, family := range []string{"reviews", "pull", "issue"} {
		req := extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			Families: []string{family},
		}
		if family == "issue" {
			req.IssueIndex = strconv.FormatInt(p.prNumber, 10)
		} else {
			req.PullNumber = strconv.FormatInt(p.prNumber, 10)
		}
		observed := nativeMergeCall(t, family+" snapshot", func() (extensions.NativeSnapshot, error) {
			return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), req)
		})
		nativeMergeReceipt(t, "reviewsnap-"+family, observed)
	}
	// Pin the change-request snapshot shape on a second PR.
	p2, _ := nativeMergePublish(t, c, "reqchanges")
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-snapshot",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p2.prNumber, PRID: p2.prID,
		IssueID: p2.issueID, PRAuthorID: c.CreatorID,
		HeadRef: "refs/heads/" + p2.branch, BaseRef: c.BaseBranch, HeadOID: p2.head, BaseOID: p2.base,
		Event: "REQUEST_CHANGES", Body: "ST12 snapshot probe findings.",
	})
	observed := nativeMergeCall(t, "reviews snapshot", func() (extensions.NativeSnapshot, error) {
		return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			PullNumber: strconv.FormatInt(p2.prNumber, 10), Families: []string{"reviews"},
		})
	})
	nativeMergeReceipt(t, "reviewsnap-reqchanges", observed)
}

// TestNativeMergeConfirmationSnapshot pins the post-merge pull and issue
// evidence Soda consumes as completion confirmation.
func TestNativeMergeConfirmationSnapshot(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "confsnap")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	payload := extensions.MergePayload{
		PullRequestNumber: p.prNumber, HeadRepositoryID: c.Repository,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch,
		ExpectedHeadOID: p.head, ExpectedBaseOID: p.base,
		Method: extensions.MergeMethodFastForwardOnly,
	}
	raw, err := json.Marshal(payload)
	nativeMust(t, err)
	nativeMergeSubmitRawCommitted(t, bg, c.TokenFile, "st12-native-snapshot", c.ActorID, c.Repository, raw)
	nativeMergeIdle(t, bg)
	for _, family := range []string{"pull", "issue", "refs"} {
		req := extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			Families: []string{family},
		}
		if family == "issue" {
			req.IssueIndex = strconv.FormatInt(p.prNumber, 10)
		} else {
			req.PullNumber = strconv.FormatInt(p.prNumber, 10)
		}
		if family == "refs" {
			req.Refs = []string{"refs/heads/" + p.branch, c.BaseBranch}
		}
		observed := nativeMergeCall(t, family+" snapshot", func() (extensions.NativeSnapshot, error) {
			return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), req)
		})
		nativeMergeReceipt(t, "confsnap-"+family, observed)
	}
}
