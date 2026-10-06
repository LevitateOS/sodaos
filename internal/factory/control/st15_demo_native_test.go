// ST15 composed development-factory demonstration (development-only).
// One real issue and its dependant complete the product loop through the
// actual factory chain: accepted input with blockers, automatic dispatch
// of real provider runs, native PR, separate review, complete CI,
// conditional merge, confirmed outcome and dependent pickup, watched
// through real Spaces and driven through real controls. Nothing here
// qualifies shipping bytes.
package control_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

const st15Image = "localhost/soda-m2-st01:20261001"

type st15Config struct {
	FountainURL       string `json:"fountain_url"`
	BrowserURL        string `json:"browser_url"`
	Socket            string `json:"socket"`
	Owner             string `json:"owner"`
	Repo              string `json:"repo"`
	Repository        int64  `json:"repository"`
	ActorID           int64  `json:"actor_id"`
	TokenID           int64  `json:"token_id"`
	BaseBranch        string `json:"base_branch"`
	TokenFile         string `json:"token_file"`
	CreatorID         int64  `json:"creator_id"`
	CreatorTokenFile  string `json:"creator_token_file"`
	ReviewerID        int64  `json:"reviewer_id"`
	ReviewerTokenID   int64  `json:"reviewer_token_id"`
	ReviewerTokenFile string `json:"reviewer_token_file"`
	MaintainerPass    string `json:"maintainer_pass_file"`
	FountainDB        string `json:"fountain_db"`
	SodaDB            string `json:"soda_db"`
	GrantKeyFile      string `json:"grant_key_file"`
	HostSocket        string `json:"host_socket"`
	BrokerSocket      string `json:"broker_socket"`
}

func loadST15(t *testing.T) st15Config {
	t.Helper()
	raw := os.Getenv("SODA_ST15_NATIVE")
	if raw == "" {
		t.Skip("ST15 NOT RUN: SODA_ST15_NATIVE is not configured")
	}
	var cfg st15Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.BrowserURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 ||
		cfg.ActorID <= 0 || cfg.TokenID <= 0 || cfg.CreatorTokenFile == "" || cfg.ReviewerTokenFile == "" ||
		cfg.MaintainerPass == "" || cfg.FountainDB == "" || cfg.SodaDB == "" || cfg.GrantKeyFile == "" ||
		cfg.HostSocket == "" || cfg.BrokerSocket == "" {
		t.Fatal("ST15 fixture requires the full native/daemon/browser configuration")
	}
	return cfg
}

func st15ReceiptDir(t *testing.T) string {
	t.Helper()
	root := os.Getenv("ST15_RECEIPT_DIR")
	if root == "" || !filepath.IsAbs(root) {
		t.Fatal("ST15_RECEIPT_DIR must be an absolute retained directory")
	}
	nativeMust(t, os.MkdirAll(root, 0o700))
	return root
}

func st15Receipt(t *testing.T, label string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(st15ReceiptDir(t), label+".json"), append(raw, '\n'), 0o600))
}

// st15Fixture is the composed journey state: one native repository, one
// project container, one daemon/broker pair and one coordinator over the
// fixture dashboard store (the live browser reads the same store).
type st15Fixture struct {
	t   *testing.T
	ctx context.Context
	cfg st15Config

	scratch  string
	db       *store.Store
	client   *identityclient.Client
	coord    *control.Coordinator
	rest     *forgejo.Client
	observer *forgejo.ServiceObserver
	bg       *forgejo.ServiceBackground
	intake   api.IntakeHandler

	projectID   string
	container   string
	containerID string
	image       string
	versions    string
	harnessSHA  string
	coderPrep   string
	reviewPrep  string
	sourceHead  string
	synthetic   bool

	issueP, issueA, issueB, issueC int64
	issuePIndex, issueAIndex       int64
	issueBIndex, issueCIndex       int64
	acceptP, acceptA, acceptB      string
	assignA, assignB               string
	runA                           string
	runC                           string
	answerA                        int64
	pubA                           string
	prNumber                       int64
	head1, head2                   string
	review1, review2               factory.ReviewOutcome
	verdict1, verdict2             string
	ciFailRev, ciPassRev           int64

	// controlBrowser carries the background dependant-stop verdict:
	// the browser boots during review-2 and polls for B's run, so
	// the stop lands within seconds of record.
	controlBrowser chan error

	checks []st15Check
}

type st15Check struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

func (fx *st15Fixture) check(name string, err error) {
	fx.t.Helper()
	if err != nil {
		fx.checks = append(fx.checks, st15Check{Name: name, Result: "fail: " + err.Error()})
		fx.t.Fatalf("ST15 %s: %v", name, err)
	}
	fx.checks = append(fx.checks, st15Check{Name: name, Result: "pass"})
	fx.t.Logf("ST15 pass: %s", name)
}

// st15Podman drives the project container through root podman: the
// production daemon runs as root and only sees root container storage,
// so the fixture manages the container in that same storage.
func st15Podman(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sudo", append([]string{"-n", "/usr/bin/podman"}, args...)...)
	env := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TMPDIR=") {
			env = append(env, entry)
		}
	}
	cmd.Env = env
	if stdin != nil {
		cmd.Stdin = strings.NewReader(string(stdin))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("podman %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func st15Pexec(ctx context.Context, name, user string, stdin []byte, args ...string) ([]byte, error) {
	full := []string{"exec"}
	if user != "" {
		full = append(full, "--user", user)
	}
	if stdin != nil {
		full = append(full, "-i")
	}
	full = append(full, name)
	full = append(full, args...)
	return st15Podman(ctx, stdin, full...)
}

func st15Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=soda-tester", "GIT_AUTHOR_EMAIL=soda-tester@example.test",
		"GIT_COMMITTER_NAME=soda-tester", "GIT_COMMITTER_EMAIL=soda-tester@example.test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func st15SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func st15RandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
