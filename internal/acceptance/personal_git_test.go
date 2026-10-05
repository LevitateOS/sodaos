package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitRemoteProgram(t *testing.T) {
	pass := strings.Repeat("B", 43)
	for _, prepare := range []bool{true, false} {
		program := gitRemoteProgram(pass, prepare)
		for _, placeholder := range []string{"PASSPHRASE_INPUT", "PREPARE_DIRECTORY", "PREPARE_KEY"} {
			if strings.Contains(program, placeholder) {
				t.Errorf("prepare=%v leaves %s", prepare, placeholder)
			}
		}
		if !strings.Contains(program, "printf '%s' '"+pass+"'") {
			t.Errorf("prepare=%v misquotes passphrase", prepare)
		}
		if !strings.HasPrefix(program, "set -eu\n") {
			t.Errorf("prepare=%v bad prefix", prepare)
		}
		if !strings.HasSuffix(program, "cat \"$base/identity.pub\"\n") {
			t.Errorf("prepare=%v bad suffix", prepare)
		}
		if strings.Contains(strings.ToLower(program), "python") {
			t.Errorf("prepare=%v carries Python", prepare)
		}
	}
	prepare := gitRemoteProgram(pass, true)
	if !strings.Contains(prepare, "\nmkdir -m 0700 \"$base\"\n") || !strings.Contains(prepare, "ssh-keygen") {
		t.Error("prepare program misses key generation")
	}
	if strings.Contains(prepare, `[ -d "$base" ] || exit 1`) || strings.Contains(prepare, `[ -f "$base/identity" ] || exit 1`) {
		t.Error("prepare program carries unlock assertions")
	}
	unlock := gitRemoteProgram(pass, false)
	if !strings.Contains(unlock, "\n[ -d \"$base\" ] || exit 1\n") || !strings.Contains(unlock, `[ -f "$base/identity" ] || exit 1`) {
		t.Error("unlock program misses assertions")
	}
	if strings.Contains(unlock, "mkdir -m 0700") || strings.Contains(unlock, "ssh-keygen") {
		t.Error("unlock program carries prepare actions")
	}
	// Byte-critical lines: the askpass helper shape and the agent PID
	// capture must stay exact.
	if !strings.Contains(prepare, `printf '#!/bin/sh\nexec /usr/bin/head -c 128 %s\n' "$password" > "$ask"`) {
		t.Error("askpass line differs")
	}
	if !strings.Contains(prepare, `sed -n 's/.*SSH_AGENT_PID=\([0-9][0-9]*\).*/\1/p'`) {
		t.Error("agent pid line differs")
	}
}

func TestGitRemoteProgramExecutesPrepareThenUnlock(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	stub := func(name, body string) {
		t.Helper()
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Stub keygen writes a fixed key pair for whatever -f path it gets.
	stub("ssh-keygen", `while [ $# -gt 0 ]; do if [ "$1" = "-f" ]; then key="$2"; shift 2; else shift; fi; done
printf 'PRIVATE\n' > "$key"
printf 'ssh-ed25519 AAAASTUBKEY U08 personal project Git\n' > "$key.pub"`)
	stub("ssh-agent", `printf 'SSH_AGENT_PID=4242; export SSH_AGENT_PID;\n'`)
	// No live agent anywhere in the fixture: -l always reports none (exit 2).
	stub("ssh-add", `if [ "$1" = "-l" ]; then exit 2; fi
exit 0`)
	run := func(program string) (string, int) {
		t.Helper()
		cmd := exec.Command("sh", "-se")
		cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		cmd.Stdin = strings.NewReader(program)
		out, err := cmd.Output()
		if err == nil {
			return string(out), 0
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), exit.ExitCode()
		}
		t.Fatalf("run: %v", err)
		return "", -1
	}
	pass := strings.Repeat("C", 43)
	base := filepath.Join(home, ".ssh/u08-personal-git")
	out, code := run(gitRemoteProgram(pass, true))
	if code != 0 {
		t.Fatalf("prepare exit = %d", code)
	}
	if out != "ssh-ed25519 AAAASTUBKEY U08 personal project Git\n" {
		t.Fatalf("prepare stdout = %q", out)
	}
	for _, dir := range []string{filepath.Join(home, ".ssh"), base} {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() || st.Mode().Perm() != 0o700 {
			t.Errorf("dir %s = %v %v", dir, st, err)
		}
	}
	for _, gone := range []string{"temporary-passphrase", "temporary-askpass"} {
		if _, err := os.Stat(filepath.Join(base, gone)); !os.IsNotExist(err) {
			t.Errorf("%s retained", gone)
		}
	}
	gitssh, err := os.ReadFile(filepath.Join(base, "git-ssh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SSH_AUTH_SOCK=" + base + "/agent", "IdentitiesOnly=yes", "-i " + base + "/identity", "\"$@\""} {
		if !strings.Contains(string(gitssh), want) {
			t.Errorf("git-ssh misses %q:\n%s", want, gitssh)
		}
	}
	if st, _ := os.Stat(filepath.Join(base, "git-ssh")); st.Mode().Perm() != 0o700 {
		t.Errorf("git-ssh mode = %o", st.Mode().Perm())
	}
	pid, err := os.ReadFile(filepath.Join(base, "agent.pid"))
	if err != nil || string(pid) != "4242\n" {
		t.Errorf("agent.pid = %q %v", pid, err)
	}
	out, code = run(gitRemoteProgram(pass, false))
	if code != 0 || out != "ssh-ed25519 AAAASTUBKEY U08 personal project Git\n" {
		t.Fatalf("unlock = %d %q", code, out)
	}
	if _, code := run(gitRemoteProgram(pass, true)); code == 0 {
		t.Error("second prepare accepted")
	}
}

func TestTokenPassphrase(t *testing.T) {
	first, err := tokenPassphrase()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 43 {
		t.Fatalf("passphrase length = %d", len(first))
	}
	for _, c := range first {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-", c) {
			t.Fatalf("passphrase charset differs: %q", first)
		}
	}
	second, err := tokenPassphrase()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("passphrases collide")
	}
}

func TestLoadGitTarget(t *testing.T) {
	dir := t.TempDir()
	target, err := loadGitTarget(dir)
	if err != nil {
		t.Fatal(err)
	}
	if target.ip != "10.89.0.2" || target.repository != "shared-alice" {
		t.Errorf("default target = %+v", target)
	}
	path := filepath.Join(dir, "target.json")
	if err := os.WriteFile(path, []byte(`{"ip": "10.89.0.5", "repository": "team-repo-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err = loadGitTarget(dir)
	if err != nil || target.ip != "10.89.0.5" || target.repository != "team-repo-1" {
		t.Errorf("file target = %+v %v", target, err)
	}
	for _, body := range []string{
		`{"ip": "10.90.0.2", "repository": "shared-alice"}`,
		`{"ip": "10.89.0.2", "repository": "UPPER"}`,
		`{"ip": "10.89.0.2", "repository": ""}`,
		`{"ip": "10.89.0.2"}`,
		`{"repository": "shared-alice"}`,
		`not json`,
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadGitTarget(dir); err == nil {
			t.Errorf("bad target accepted: %s", body)
		}
	}
}

func TestValidateGitURL(t *testing.T) {
	good := "ssh://git@127.0.0.1:2222/u08-alice-8417/shared-alice.git"
	if err := validateGitURL(good, "shared-alice"); err != nil {
		t.Errorf("valid advertisement rejected: %v", err)
	}
	for _, bad := range []string{
		"http://git@127.0.0.1:2222/u08-alice-8417/shared-alice.git",
		"ssh://git@127.0.0.2:2222/u08-alice-8417/shared-alice.git",
		"ssh://git@127.0.0.1:2223/u08-alice-8417/shared-alice.git",
		"ssh://git@127.0.0.1/u08-alice-8417/shared-alice.git",
		"ssh://root@127.0.0.1:2222/u08-alice-8417/shared-alice.git",
		"ssh://git:secret@127.0.0.1:2222/u08-alice-8417/shared-alice.git",
		"ssh://git@127.0.0.1:2222/u08-alice-8417/shared-alice.git?x=1",
		"ssh://git@127.0.0.1:2222/u08-alice-8417/shared-alice.git#frag",
		"ssh://git@127.0.0.1:2222/u08-alice-8417/other.git",
		"not a url",
		"",
	} {
		if err := validateGitURL(bad, "shared-alice"); err == nil {
			t.Errorf("advertisement accepted: %q", bad)
		}
	}
}

func TestExtractCommit(t *testing.T) {
	commit := strings.Repeat("c", 40)
	got, err := extractCommit([]byte("noise\nU08-COMMIT:" + commit + "\n"))
	if err != nil || got != commit {
		t.Errorf("extract = %q %v", got, err)
	}
	got, err = extractCommit([]byte("U08-COMMIT:" + commit + "\r\n"))
	if err != nil || got != commit {
		t.Errorf("CRLF extract = %q %v", got, err)
	}
	for _, bad := range []string{"no marker here\n", "U08-COMMIT:short\n", "U08-COMMIT:" + strings.Repeat("z", 40) + "\n"} {
		if _, err := extractCommit([]byte(bad)); err == nil {
			t.Errorf("bad readback accepted: %q", bad)
		}
	}
}

func TestGitOutcomesOrder(t *testing.T) {
	outcomes := []gitOutcome{{Login: "u08-alice-8417", Branch: "u08-native-alice", Commit: strings.Repeat("a", 40), RepositoryID: json.RawMessage("42")}}
	encoded, err := json.MarshalIndent(outcomes, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n  {\n" +
		"    \"login\": \"u08-alice-8417\",\n" +
		"    \"branch\": \"u08-native-alice\",\n" +
		"    \"commit\": \"" + strings.Repeat("a", 40) + "\",\n" +
		"    \"repository_id\": 42\n  }\n]"
	if string(encoded) != want {
		t.Errorf("outcomes JSON differs:\n%s", encoded)
	}
}

// stubGitSSH emulates the member sessions: key commands print a canned public
// key, clone commands print a commit readback, fetch commands succeed.
func stubGitSSH(t *testing.T, pubkey, commit string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "argv.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + log + "\n" +
		"body=$(cat)\n" +
		"case \"$body\" in *'ssh-agent'*) printf '" + pubkey + "\\n';; *'git clone'*) printf 'noise\\nU08-COMMIT:" + commit + "\\n';; *) exit 0;; esac\n"
	path := filepath.Join(dir, "ssh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func gitFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunPersonalGitPrepareUnlock(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	fixture := gitFixture(t)
	pubkey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFAKEKEYFORTEST\n"
	commit := strings.Repeat("d", 40)
	log := stubGitSSH(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFAKEKEYFORTEST", commit)
	var stdout bytes.Buffer
	if err := RunPersonalGit([]string{"prepare", fixture}, &stdout); err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	directory := filepath.Join(fixture, "personal-git")
	for _, login := range []string{"u08-alice-8417", "u08-bob-8417"} {
		data, err := os.ReadFile(filepath.Join(directory, login+".pub"))
		if err != nil || string(data) != pubkey {
			t.Errorf("%s pub = %q %v", login, data, err)
		}
		st, err := os.Stat(filepath.Join(directory, login+"-passphrase"))
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Errorf("%s passfile = %v %v", login, st, err)
		}
	}
	if !strings.Contains(stdout.String(), "only public part exported") {
		t.Errorf("prepare output = %q", stdout.String())
	}
	argv, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"sh -se", "IdentitiesOnly=yes", "u08-alice-8417@10.89.0.2", "u08-bob-8417@10.89.0.2"} {
		if !strings.Contains(string(argv), want) {
			t.Errorf("argv log misses %q:\n%s", want, argv)
		}
	}
	stdout.Reset()
	if err := RunPersonalGit([]string{"unlock", fixture}, &stdout); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if err := RunPersonalGit([]string{"prepare", fixture}, &bytes.Buffer{}); err == nil {
		t.Error("second prepare accepted")
	}
}

func TestRunPersonalGitExercise(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	fixture := gitFixture(t)
	directory := filepath.Join(fixture, "personal-git")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	commit := strings.Repeat("e", 40)
	stubGitSSH(t, "unused", commit)
	for _, login := range []string{"u08-alice-8417", "u08-bob-8417"} {
		record := `{"ssh_url": "ssh://git@127.0.0.1:2222/u08-alice-8417/shared-alice.git", "id": 7}`
		if err := os.WriteFile(filepath.Join(directory, login+"-repository.json"), []byte(record), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout bytes.Buffer
	if err := RunPersonalGit([]string{"exercise", fixture}, &stdout); err != nil {
		t.Fatalf("exercise failed: %v", err)
	}
	results, err := os.ReadFile(filepath.Join(directory, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []gitOutcome
	if json.Unmarshal(results, &outcomes) != nil || len(outcomes) != 2 {
		t.Fatalf("outcomes = %s", results)
	}
	if outcomes[0].Login != "u08-alice-8417" || outcomes[0].Commit != commit || string(outcomes[0].RepositoryID) != "7" {
		t.Errorf("alice outcome = %+v", outcomes[0])
	}
	if outcomes[1].Login != "u08-bob-8417" || outcomes[1].Branch != "u08-native-bob" {
		t.Errorf("bob outcome = %+v", outcomes[1])
	}
	if !strings.Contains(stdout.String(), "Independent collaborator fetch/readback verified") {
		t.Errorf("exercise output = %q", stdout.String())
	}
}

func TestRunPersonalGitValidation(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	fixture := gitFixture(t)
	if err := os.WriteFile(filepath.Join(fixture, "target.json"), []byte(`{"SYNTHETIC_PRIVATE_MARKER":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RunPersonalGit([]string{"prepare", fixture}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("bad target accepted")
	}
	if !strings.HasPrefix(err.Error(), "Personal Git incomplete; retained state; failure type: ") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_MARKER") {
		t.Errorf("failure leaks request: %q", err.Error())
	}
	if err := RunPersonalGit([]string{"bogus", fixture}, &bytes.Buffer{}); err == nil {
		t.Error("bad phase accepted")
	}
	if err := RunPersonalGit([]string{"prepare"}, &bytes.Buffer{}); err == nil {
		t.Error("short argv accepted")
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "wrong")
	if err := RunPersonalGit([]string{"prepare", fixture}, &bytes.Buffer{}); err == nil {
		t.Error("wrong env accepted")
	}
}
