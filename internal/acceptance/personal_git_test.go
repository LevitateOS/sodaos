package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersonalGitPayloadResolutionAndCommand(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "soda-installed-probes")
	payload := []byte("compiled-payload")
	if err := os.WriteFile(personalGitPayloadPath(executable), payload, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := loadPersonalGitPayload(executable)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, %v", got, err)
	}
	for _, tc := range []struct {
		prepare bool
		phase   string
	}{{true, "prepare"}, {false, "unlock"}} {
		command := personalGitRemoteCommand(len(payload), tc.prepare)
		if !strings.Contains(command, `mktemp "$HOME/.soda-personal-git.XXXXXX"`) || !strings.Contains(command, "head -c 16") || !strings.Contains(command, "personal-git "+tc.phase) {
			t.Errorf("command = %q", command)
		}
		if strings.Contains(command, "PASSPHRASE") {
			t.Errorf("command carries secret placeholder: %q", command)
		}
	}
	if got := quoteRemoteShell("a'b"); got != `'a'\''b'` {
		t.Errorf("quoted command = %q", got)
	}
}

func TestPersonalGitPayloadMissingFailsBeforePassphraseCreation(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	fixture := gitFixture(t)
	if err := runPersonalGitWithPayload([]string{"prepare", fixture}, &bytes.Buffer{}, func() ([]byte, error) {
		return nil, errors.New("compiled personal-Git payload unavailable")
	}); err == nil {
		t.Fatal("missing sibling payload accepted")
	}
	if _, err := os.Stat(filepath.Join(fixture, "personal-git", "u08-alice-8417-passphrase")); !os.IsNotExist(err) {
		t.Fatalf("passphrase exists before payload admission: %v", err)
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

func TestLoadPassphraseBoundsAndValidatesPrivateInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passphrase")
	valid := strings.Repeat("A", 43)
	for _, tc := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "valid 43 bytes", value: valid, valid: true},
		{name: "short", value: valid[:42]},
		{name: "long", value: valid + "A"},
		{name: "invalid character", value: valid[:42] + "."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.value), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := loadPassphrase(path)
			if tc.valid {
				if err != nil || got != valid {
					t.Fatalf("loadPassphrase() = %q, %v", got, err)
				}
			} else if err == nil || got != "" {
				t.Fatalf("loadPassphrase() = %q, %v; want refusal", got, err)
			}
		})
	}

	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "passphrase-link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if got, err := loadPassphrase(link); err == nil || got != "" {
		t.Fatalf("symlink loadPassphrase() = %q, %v; want refusal", got, err)
	}

	nonregular := filepath.Join(dir, "directory")
	if err := os.Mkdir(nonregular, 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := loadPassphrase(nonregular); err == nil || got != "" {
		t.Fatalf("nonregular loadPassphrase() = %q, %v; want refusal", got, err)
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
		"case \"$*\" in *'personal-git prepare'*|*'personal-git unlock'*) size=$(wc -c); printf 'STDIN:%s\\n' \"$size\" >> " + log + "; printf '" + pubkey + "\\n'; exit 0;; esac\n" +
		"body=$(cat)\n" +
		"case \"$body\" in *'git clone'*) printf 'noise\\nU08-COMMIT:" + commit + "\\n';; *) exit 0;; esac\n"
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
	if err := runPersonalGitWithPayload([]string{"prepare", fixture}, &stdout, func() ([]byte, error) {
		return []byte("compiled payload bytes"), nil
	}); err != nil {
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
	for _, want := range []string{`mktemp "$HOME/.soda-personal-git.XXXXXX"`, "personal-git prepare", "head -c 22", "IdentitiesOnly=yes", "u08-alice-8417@10.89.0.2", "u08-bob-8417@10.89.0.2", "STDIN:65"} {
		if !strings.Contains(string(argv), want) {
			t.Errorf("argv log misses %q:\n%s", want, argv)
		}
	}
	for _, login := range []string{"u08-alice-8417", "u08-bob-8417"} {
		passphrase, err := os.ReadFile(filepath.Join(directory, login+"-passphrase"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(argv), string(passphrase)) {
			t.Fatal("passphrase appeared in SSH argv log")
		}
	}
	stdout.Reset()
	if err := runPersonalGitWithPayload([]string{"unlock", fixture}, &stdout, func() ([]byte, error) {
		return []byte("compiled payload bytes"), nil
	}); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if err := runPersonalGitWithPayload([]string{"prepare", fixture}, &bytes.Buffer{}, func() ([]byte, error) {
		return []byte("compiled payload bytes"), nil
	}); err == nil {
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
