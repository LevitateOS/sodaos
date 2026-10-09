// Personal Git on the exact retained fixtures: prepare, then exercise. Go
// port of the retired tests/installed/personal-git.py. Prepare generates
// encrypted Git keys inside each user's project home and loads local (not
// forwarded) SSH agents; only public keys leave the project. Existing
// keys/checkouts are never replaced or automatically retried. It is not
// appliance runtime code.
package acceptance

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PersonalGitUsage documents the probe's CLI surface.
const PersonalGitUsage = "soda-installed-probes personal-git {prepare|exercise|unlock} FIXTURE_DIR"

// gitRemoteTemplate is the remote key/agent program in POSIX shell,
// piped to sh -se like gitFetchProbe. It replaces the retired Python
// template command for command: same directories, modes, umask, askpass
// shape, agent-singleton probe (ssh-add -l must exit 2 for a stale
// socket), key add, temporary cleanup and git-ssh writer. Every child
// closes stdin: the program itself arrives over stdin, so an inheriting
// child would consume the script. The placeholders are replaced exactly
// as before.
const gitRemoteTemplate = `set -eu
base="$HOME/.ssh/u08-personal-git"
mkdir -p -m 0700 "$HOME/.ssh"
PREPARE_DIRECTORY
umask 077
password="$base/temporary-passphrase"
ask="$base/temporary-askpass"
printf '%s' PASSPHRASE_INPUT > "$password"
printf '#!/bin/sh\nexec /usr/bin/head -c 128 %s\n' "$password" > "$ask"
chmod 0700 "$ask"
export SSH_ASKPASS="$ask" SSH_ASKPASS_REQUIRE=force DISPLAY=soda-u08
PREPARE_KEY
# A socket without a live agent is a retained run-owned transient.
if [ -e "$base/agent" ]; then
	set +e
	SSH_AUTH_SOCK="$base/agent" ssh-add -l </dev/null >/dev/null 2>&1
	code=$?
	set -e
	if [ "$code" != 2 ]; then echo 'Agent already live; no duplicate start' >&2; exit 1; fi
	rm -f "$base/agent"
fi
agent=$(ssh-agent -a "$base/agent" -s </dev/null)
pid=$(printf '%s' "$agent" | sed -n 's/.*SSH_AGENT_PID=\([0-9][0-9]*\).*/\1/p')
[ -n "$pid" ] || { echo 'Git key/agent operation failed' >&2; exit 1; }
printf '%s\n' "$pid" > "$base/agent.pid"
export SSH_AUTH_SOCK="$base/agent"
ssh-add "$base/identity" </dev/null
# Only these exact run-owned temporary secret inputs are removed.
# The encrypted key and live agent remain in this user's home.
rm -f "$password" "$ask"
{
	printf '#!/bin/sh\n'
	printf 'export SSH_AUTH_SOCK=%s\n' "$base/agent"
	printf 'exec /usr/bin/ssh -F /dev/null -o BatchMode=yes -o ForwardAgent=no -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=%s -i %s "$@"\n' "$base/known_hosts" "$base/identity"
} > "$base/git-ssh"
chmod 0700 "$base/git-ssh"
cat "$base/identity.pub"
`

// gitFetchProbe is the collaborator fetch check piped to sh -se.
const gitFetchProbe = `set -eu
export GIT_SSH="$HOME/.ssh/u08-personal-git/git-ssh"
cd "$HOME/u08-personal-checkout"
git fetch origin u08-native-alice
git show FETCH_HEAD:alice-native-git.txt | grep -qx 'u08-alice-8417 personal native Git proof'
`

// gitTarget selects the project IP and repository.
type gitTarget struct {
	ip         string
	repository string
}

// loadGitTarget reads the fixture target or the documented default.
func loadGitTarget(fixture string) (gitTarget, error) {
	path := filepath.Join(fixture, "target.json")
	if _, err := os.Stat(path); err != nil {
		return gitTarget{ip: "10.89.0.2", repository: "shared-alice"}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return gitTarget{}, err
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return gitTarget{}, errors.New("invalid personal-Git target")
	}
	return parseGitTarget(raw)
}

func parseGitTarget(raw map[string]json.RawMessage) (gitTarget, error) {
	var target gitTarget
	ipRaw, ok := raw["ip"]
	if !ok || json.Unmarshal(ipRaw, &target.ip) != nil {
		return gitTarget{}, errors.New("project IP required")
	}
	addr, err := netip.ParseAddr(target.ip)
	if err != nil || !addr.Is4() || !netip.MustParsePrefix("10.89.0.0/24").Contains(addr) {
		return gitTarget{}, errors.New("project IP outside the selected project network")
	}
	repoRaw, ok := raw["repository"]
	if !ok || json.Unmarshal(repoRaw, &target.repository) != nil {
		return gitTarget{}, errors.New("repository name required")
	}
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,99}$`).MatchString(target.repository) {
		return gitTarget{}, errors.New("invalid repository name")
	}
	return target, nil
}

// gitProbe carries the run state shared by the probe phases.
type gitProbe struct {
	fixture   string
	directory string
	target    gitTarget
}

// gitSSHBase builds the exact per-user SSH argv the Python probe used.
func (p *gitProbe) gitSSHBase(login, who string) []string {
	return []string{
		"-F", "/dev/null",
		"-o", "ForwardAgent=no",
		"-o", "ClearAllForwardings=yes",
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + filepath.Join(p.fixture, login+"-known-hosts"),
		"-i", filepath.Join(p.fixture, who, "development"),
		login + "@" + p.target.ip,
	}
}

// gitChecked runs one native command with the probe's timeout and refusal.
func gitChecked(args []string, stdin []byte) ([]byte, error) {
	outcome, err := runBounded(args[0], args[1:], stdin, 90*time.Second)
	if err != nil {
		return nil, err
	}
	if outcome.exitCode != 0 {
		return nil, fmt.Errorf("native personal Git operation failed; no retry (exit %d)", outcome.exitCode)
	}
	return outcome.stdout, nil
}

// tokenPassphrase generates a 43-character URL-safe passphrase, the Go
// equivalent of secrets.token_urlsafe(32).
func tokenPassphrase() (string, error) {
	var secret [32]byte
	if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret[:]), nil
}

// gitRemoteProgram renders the remote program for a validated passphrase.
// The passphrase charset ([A-Za-z0-9_-]) admits no quoting, so
// single-quote wrapping is exact.
func gitRemoteProgram(passphrase string, prepare bool) string {
	program := strings.ReplaceAll(gitRemoteTemplate, "PASSPHRASE_INPUT", "'"+passphrase+"'")
	if prepare {
		program = strings.ReplaceAll(program, "PREPARE_DIRECTORY", `mkdir -m 0700 "$base"`)
		return strings.ReplaceAll(program, "PREPARE_KEY", `ssh-keygen -q -t ed25519 -f "$base/identity" -C 'U08 personal project Git' </dev/null`)
	}
	program = strings.ReplaceAll(program, "PREPARE_DIRECTORY", `[ -d "$base" ] || exit 1`)
	return strings.ReplaceAll(program, "PREPARE_KEY", `[ -f "$base/identity" ] || exit 1`)
}

// preparePassfile creates a fresh passphrase file with exclusive 0600.
func preparePassfile(path string) error {
	passphrase, err := tokenPassphrase()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString(passphrase)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// loadPassphrase reads and validates a stored passphrase.
func loadPassphrase(path string) (passphrase string, resultErr error) {
	file, limit, err := openPrivateInput(path, 43)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			passphrase = ""
			resultErr = errors.Join(resultErr, closeErr)
		}
	}()
	st, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !ownedByCaller(st) {
		return "", errors.New("passphrase must be a caller-owned restricted file")
	}
	data, err := readPrivateInput(file, limit, path)
	if err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).Match(data) {
		return "", errors.New("invalid passphrase shape")
	}
	return string(data), nil
}

// gitKeyUser runs the prepare/unlock phase for one user.
func (p *gitProbe) gitKeyUser(login, who string, prepare bool, stdout io.Writer) error {
	base := p.gitSSHBase(login, who)
	passfile := filepath.Join(p.directory, login+"-passphrase")
	if prepare {
		if err := preparePassfile(passfile); err != nil {
			return err
		}
	}
	public, err := p.fetchExportedKey(base, passfile, prepare)
	if err != nil {
		return err
	}
	pubPath := filepath.Join(p.directory, login+".pub")
	if prepare {
		if err := os.WriteFile(pubPath, public, 0o666); err != nil {
			return err
		}
	} else if err := checkUnlockedKey(pubPath, public); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, login+" encrypted project-local Git key/agent ready; only public part exported")
	return err
}

// fetchExportedKey renders the remote program and returns the public part.
func (p *gitProbe) fetchExportedKey(base []string, passfile string, prepare bool) ([]byte, error) {
	passphrase, err := loadPassphrase(passfile)
	if err != nil {
		return nil, err
	}
	program := gitRemoteProgram(passphrase, prepare)
	public, err := gitChecked(append(append([]string{"ssh"}, base...), "sh -se"), []byte(program))
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(public, []byte("ssh-ed25519 ")) || bytes.Count(public, []byte("\n")) != 1 {
		return nil, errors.New("invalid exported public key")
	}
	return public, nil
}

// checkUnlockedKey requires the re-exported key to match prepare's.
func checkUnlockedKey(pubPath string, public []byte) error {
	expected, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(public, expected) {
		return errors.New("exported public key differs from prepare")
	}
	return nil
}

// validateGitURL checks the actual repository advertisement.
func validateGitURL(sshURL, repository string) error {
	parsed, err := url.Parse(sshURL)
	if err != nil {
		return errors.New("invalid repository advertisement")
	}
	if err := checkGitEndpoint(parsed); err != nil {
		return err
	}
	if err := checkGitUser(parsed); err != nil {
		return err
	}
	return checkGitPath(parsed, repository)
}

// checkGitEndpoint requires the loopback Git transport and port.
func checkGitEndpoint(parsed *url.URL) error {
	if !strings.EqualFold(parsed.Scheme, "ssh") || parsed.Hostname() != "127.0.0.1" {
		return errors.New("invalid repository advertisement")
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port != 2222 {
		return errors.New("invalid repository advertisement")
	}
	return nil
}

// checkGitUser requires the bare git identity without extras.
func checkGitUser(parsed *url.URL) error {
	if parsed.User == nil || parsed.User.Username() != "git" {
		return errors.New("invalid repository advertisement")
	}
	if password, _ := parsed.User.Password(); password != "" {
		return errors.New("invalid repository advertisement")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("invalid repository advertisement")
	}
	return nil
}

// checkGitPath requires the exact retained repository path.
func checkGitPath(parsed *url.URL, repository string) error {
	if parsed.EscapedPath() != "/u08-alice-8417/"+repository+".git" {
		return errors.New("invalid repository advertisement")
	}
	return nil
}

// exerciseCommand builds the per-user clone/commit/push proof script.
func exerciseCommand(login, who, branch, clone string) string {
	lines := []string{
		"set -eu",
		`export GIT_SSH="$HOME/.ssh/u08-personal-git/git-ssh"`,
		`test ! -e "$HOME/u08-personal-checkout"`,
		"git clone '" + clone + "' \"$HOME/u08-personal-checkout\"",
		`cd "$HOME/u08-personal-checkout"`,
		"git config user.name '" + login + "'",
		"git config user.email '" + login + "@example.test'",
		"git switch -c " + branch,
		"printf '%s\\n' '" + login + " personal native Git proof' > " + who + "-native-git.txt",
		"git add " + who + "-native-git.txt",
		"git commit -m 'U08 personal Git proof for " + who + "'",
		"git push --set-upstream origin " + branch,
		"local=$(git rev-parse HEAD)",
		"remote=$(git ls-remote origin refs/heads/" + branch + " | cut -f1)",
		`test "$local" = "$remote"`,
		`printf 'U08-COMMIT:%s\n' "$local"`,
	}
	return strings.Join(lines, "\n") + "\n"
}

// extractCommit reads the first commit readback line.
func extractCommit(output []byte) (string, error) {
	for _, line := range strings.Split(string(output), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSuffix(line, "\r"), "U08-COMMIT:")
		if !ok {
			continue
		}
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(rest) {
			return "", errors.New("invalid commit readback")
		}
		return rest, nil
	}
	return "", errors.New("missing commit readback")
}

// gitOutcome preserves the retired results.json entry order.
type gitOutcome struct {
	Login        string          `json:"login"`
	Branch       string          `json:"branch"`
	Commit       string          `json:"commit"`
	RepositoryID json.RawMessage `json:"repository_id"`
}

// loadExerciseRepo reads one user's repository advertisement.
func (p *gitProbe) loadExerciseRepo(login string) (string, json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(p.directory, login+"-repository.json"))
	if err != nil {
		return "", nil, err
	}
	var repo struct {
		SSHURL string          `json:"ssh_url"`
		ID     json.RawMessage `json:"id"`
	}
	if json.Unmarshal(data, &repo) != nil {
		return "", nil, errors.New("invalid repository record")
	}
	if err := validateGitURL(repo.SSHURL, p.target.repository); err != nil {
		return "", nil, err
	}
	return repo.SSHURL, repo.ID, nil
}

// exerciseUser runs the clone/commit/push proof for one user.
func (p *gitProbe) exerciseUser(login, who string, outcomes []gitOutcome, stdout io.Writer) ([]gitOutcome, error) {
	clone, id, err := p.loadExerciseRepo(login)
	if err != nil {
		return outcomes, err
	}
	base := p.gitSSHBase(login, who)
	branch := "u08-native-" + who
	result, err := gitChecked(append(append([]string{"ssh"}, base...), "sh -se"), []byte(exerciseCommand(login, who, branch, clone)))
	if err != nil {
		return outcomes, err
	}
	commit, err := extractCommit(result)
	if err != nil {
		return outcomes, err
	}
	outcomes = append(outcomes, gitOutcome{Login: login, Branch: branch, Commit: commit, RepositoryID: id})
	if err := writeGitOutcomes(p.directory, outcomes); err != nil {
		return outcomes, err
	}
	_, err = fmt.Fprintln(stdout, login+" personal clone/commit/push/native ref readback verified")
	return outcomes, err
}

// writeGitOutcomes records the outcomes after each user, as before.
func writeGitOutcomes(directory string, outcomes []gitOutcome) error {
	encoded, err := json.MarshalIndent(outcomes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "results.json"), encoded, 0o666)
}

// phaseKey runs prepare or unlock for both users.
func (p *gitProbe) phaseKey(prepare bool, stdout io.Writer) error {
	for _, who := range []string{"alice", "bob"} {
		if err := p.gitKeyUser("u08-"+who+"-8417", who, prepare, stdout); err != nil {
			return err
		}
	}
	return nil
}

// phaseExercise runs the per-user proofs plus the collaborator readback. The
// final fetch runs over bob's session, matching the Python loop's last ssh
// binding.
func (p *gitProbe) phaseExercise(stdout io.Writer) error {
	var outcomes []gitOutcome
	var err error
	for _, who := range []string{"alice", "bob"} {
		if outcomes, err = p.exerciseUser("u08-"+who+"-8417", who, outcomes, stdout); err != nil {
			return err
		}
	}
	base := p.gitSSHBase("u08-bob-8417", "bob")
	if _, err := gitChecked(append(append([]string{"ssh"}, base...), "sh -se"), []byte(gitFetchProbe)); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "Independent collaborator fetch/readback verified; no merge or environment promotion performed.")
	return err
}

// gitPhase validates the phase argument.
func gitPhase(arg string) (string, error) {
	if arg != "prepare" && arg != "exercise" && arg != "unlock" {
		return "", errors.New(PersonalGitUsage)
	}
	return arg, nil
}

// RunPersonalGit is the personal-git entrypoint: PHASE FIXTURE_DIR.
func RunPersonalGit(args []string, stdout io.Writer) error {
	if err := runPersonalGit(args, stdout); err != nil {
		return fail("Personal Git incomplete; retained state; failure type: ", err)
	}
	return nil
}

func runPersonalGit(args []string, stdout io.Writer) error {
	if os.Getenv("SODA_NATIVE_VALIDATE") != "soda-test" {
		return errors.New("explicit soda-test validation required")
	}
	if len(args) != 2 {
		return errors.New(PersonalGitUsage)
	}
	phase, err := gitPhase(args[0])
	if err != nil {
		return err
	}
	if err := privateDir(args[1], true); err != nil {
		return err
	}
	target, err := loadGitTarget(args[1])
	if err != nil {
		return err
	}
	directory := filepath.Join(args[1], "personal-git")
	if phase == "prepare" {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return err
		}
	}
	probe := &gitProbe{fixture: args[1], directory: directory, target: target}
	if phase == "exercise" {
		return probe.phaseExercise(stdout)
	}
	return probe.phaseKey(phase == "prepare", stdout)
}
