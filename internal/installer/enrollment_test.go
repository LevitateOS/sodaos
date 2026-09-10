package installer

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
)

func enrollmentTestKey(t *testing.T) string {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}

func TestEnrollmentInput(t *testing.T) {
	key := enrollmentTestKey(t)
	for _, value := range []string{key, key + "\n", key + " laptop\n"} {
		got, err := enrollmentPublicKey(strings.NewReader(value))
		if err != nil || got != key {
			t.Fatalf("public input rejected: %v", err)
		}
	}
	for _, value := range []string{"", key + "\n\n", key + "\n" + key + "\n", "command=\"sh\" " + key, "-----BEGIN OPENSSH PRIVATE KEY-----", strings.Repeat("x", enrollmentKeyLimit+1), key + "\r\n", key + "\x00"} {
		if _, err := enrollmentPublicKey(strings.NewReader(value)); err == nil {
			t.Fatal("unsafe input accepted")
		}
	}
}

func TestEnrollmentPrivateAddressAndClient(t *testing.T) {
	for _, address := range []string{"10.0.0.8", "172.16.3.4", "192.168.1.20"} {
		command, err := enrollmentClientCommand(address)
		if err != nil || !strings.Contains(command, "root@"+address+" < ~/.ssh/id_ed25519.pub") || !strings.Contains(command, "ControlPath=none") || !strings.Contains(command, "ClearAllForwardings=yes") {
			t.Fatalf("invalid client instruction: %v", err)
		}
	}
	for _, address := range []string{"", "0.0.0.0", "127.0.0.1", "169.254.1.1", "8.8.8.8", "100.64.0.1", "::1", "fd00::1", "192.168.1.1\nPermitTTY yes", "192.168.1.01"} {
		if _, err := enrollmentClientCommand(address); err == nil {
			t.Fatalf("unsafe address accepted: %q", address)
		}
	}
}

func TestEnrollmentNativePolicy(t *testing.T) {
	config := enrollmentConfig()
	for _, line := range []string{"PermitRootLogin yes", "AuthenticationMethods password", "PasswordAuthentication yes", "UsePAM no", "KbdInteractiveAuthentication no", "PubkeyAuthentication no", "PermitEmptyPasswords no", "DisableForwarding yes", "PermitTunnel no", "PermitTTY no", "PermitUserRC no", "PermitUserEnvironment no", "MaxSessions 1", "MaxAuthTries 3", "AuthorizedKeysFile none", "ForceCommand " + enrollmentBinary + " enrollment-receive"} {
		if !strings.Contains(config, "\n"+line+"\n") {
			t.Fatalf("missing native restriction %s", line)
		}
	}
	for _, forbidden := range []string{"Include ", "AcceptEnv ", "Subsystem ", "ListenAddress ", "Match ", "SetEnv "} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("unexpected configurable SSH surface %q", forbidden)
		}
	}
	args := strings.Join(enrollmentStartArgs(), " ")
	for _, required := range []string{"--unit=" + enrollmentUnit, "--property=RuntimeMaxSec=300s", "--property=KillMode=control-group", "--property=SendSIGKILL=yes", "--property=Restart=no", "--property=TimeoutStopSec=2s"} {
		if !strings.Contains(args, required) {
			t.Fatalf("missing lifetime property %s", required)
		}
	}
}

func TestEnrollmentNativeSocketOwnership(t *testing.T) {
	config, err := enrollmentSocketUnitConfig("192.168.1.20")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"BindsTo=" + enrollmentUnit, "After=" + enrollmentUnit, "ListenStream=192.168.1.20:" + enrollmentPort, "Accept=yes", "MaxConnections=2"} {
		if !strings.Contains(config, "\n"+line+"\n") {
			t.Fatalf("missing native socket requirement: %s", line)
		}
	}
	if _, err := enrollmentSocketUnitConfig("0.0.0.0\nAccept=no"); err == nil {
		t.Fatal("unsafe native socket address accepted")
	}
	connection := enrollmentTemplateUnitConfig()
	for _, line := range []string{"BindsTo=" + enrollmentUnit + " " + enrollmentSocketUnit, "After=" + enrollmentUnit + " " + enrollmentSocketUnit, "StandardInput=socket", "StandardOutput=socket", "Slice=system.slice", "RuntimeMaxSec=300s", "KillMode=control-group", "TimeoutStopSec=2s", "SendSIGKILL=yes", "Restart=no", "ExecStart=/usr/sbin/sshd -i -e -f " + enrollmentConfigPath} {
		if !strings.Contains(connection, "\n"+line+"\n") {
			t.Fatalf("missing native connection requirement: %s", line)
		}
	}
	if strings.Contains(config+connection, "[Install]") {
		t.Fatal("temporary units must never be enabled at boot")
	}
}

func TestEnrollmentPeerServiceProvenance(t *testing.T) {
	unit, err := enrollmentPeerUnit("0::/system.slice/" + enrollmentUnit + "\n")
	if err != nil || unit != enrollmentUnit || enrollmentReceiverUnit(unit) {
		t.Fatal("fixed broker identity was confused with a connection")
	}
	unit, err = enrollmentPeerUnit("0::/system.slice/soda-key-enrollment@0-192.168.1.20:22222-192.168.1.30:43123.service\n")
	if err != nil || !enrollmentReceiverUnit(unit) {
		t.Fatal("native connection service identity refused")
	}
	for _, group := range []string{"0::/user.slice/user-0.slice/session-1.scope\n", "0::/system.slice/soda-key-enrollment@x.service/subgroup\n", "0::/system.slice/soda-key-enrollment@x.service\n0::/system.slice/sshd.service\n"} {
		if _, err := enrollmentPeerUnit(group); err == nil {
			t.Fatalf("unrelated or ambiguous cgroup accepted: %q", group)
		}
	}
	for _, unit := range []string{enrollmentTemplateUnit, "sshd.service", "soda-key-enrollment-other@x.service", "soda-key-enrollment@x.service/child", "soda-key-enrollment@x.service\n"} {
		if enrollmentReceiverUnit(unit) {
			t.Fatalf("unrelated root service accepted: %q", unit)
		}
	}
}

// sshd -T only parses/evaluates configuration: it never binds or authenticates.
// The key is synthetic, generated under t.TempDir; no native host key is read.
func TestEnrollmentNativeConfigParse(t *testing.T) {
	sshd, err := exec.LookPath("sshd")
	if err != nil {
		t.Skip("native sshd unavailable for parse-only check")
	}
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("native ssh-keygen unavailable for synthetic fixture")
	}
	dir := t.TempDir()
	key := filepath.Join(dir, "host_key")
	if err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", key).Run(); err != nil {
		t.Fatal("synthetic host key creation failed")
	}
	config := strings.Replace(enrollmentConfig(), "/etc/ssh/ssh_host_ed25519_key", key, 1)
	file := filepath.Join(dir, "sshd_config")
	if err := os.WriteFile(file, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(sshd, "-T", "-f", file).Output()
	if err != nil {
		t.Fatal("native sshd rejected isolated enrollment configuration (no listener opened)")
	}
	for _, line := range []string{"usepam no", "permitrootlogin yes", "authenticationmethods password", "disableforwarding yes", "permittty no", "permituserrc no", "permituserenvironment no", "passwordauthentication yes", "pubkeyauthentication no", "kbdinteractiveauthentication no", "maxsessions 1", "forcecommand " + enrollmentBinary + " enrollment-receive"} {
		if !strings.Contains(string(output), "\n"+line+"\n") {
			t.Fatalf("native parsed restriction missing: %s", line)
		}
	}
}

func TestEnrollmentPreservesAuthorizedKeys(t *testing.T) {
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	existingKey := enrollmentTestKey(t)
	existing := []byte("# existing native policy\nrestrict " + existingKey + " original comment")
	dir := filepath.Join(home, ".ssh")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "authorized_keys")
	if err := os.WriteFile(path, existing, 0600); err != nil {
		t.Fatal(err)
	}
	key := enrollmentTestKey(t)
	if err := appendEnrollmentKey(context.Background(), home, key, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, append(append(bytes.Clone(existing), '\n'), []byte(key+"\n")...)) {
		t.Fatal("existing authorized keys were not preserved exactly")
	}
	st, err := os.Stat(path)
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("native file permissions changed unsafely")
	}
	if err := appendEnrollmentKey(context.Background(), home, existingKey, uint32(os.Geteuid())); err == nil {
		t.Fatal("restricted key was duplicated as unrestricted")
	}
	if err := appendEnrollmentKey(context.Background(), home, key, uint32(os.Geteuid())); err == nil {
		t.Fatal("duplicate accepted")
	}
	again, _ := os.ReadFile(path)
	if !bytes.Equal(again, got) {
		t.Fatal("refusal changed existing keys")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("temporary key files remained")
	}
}

func TestEnrollmentCreatesAuthorizedKeys(t *testing.T) {
	home := t.TempDir()
	key := enrollmentTestKey(t)
	if err := appendEnrollmentKey(context.Background(), home, key, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		name string
		mode os.FileMode
	}{{".ssh", 0700}, {".ssh/authorized_keys", 0600}} {
		st, err := os.Stat(filepath.Join(home, entry.name))
		if err != nil || st.Mode().Perm() != entry.mode {
			t.Fatalf("wrong ownership/mode for %s", entry.name)
		}
	}
}

func TestEnrollmentRefusesUnsafeKeyPaths(t *testing.T) {
	for _, scenario := range []string{"home-symlink", "ssh-symlink", "key-symlink", "key-hardlink", "key-directory", "key-writable", "ssh-writable", "home-writable", "oversized", "wrong-owner"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			if err := os.Mkdir(home, 0700); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(home, ".ssh")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "authorized_keys")
			sentinel := filepath.Join(root, "sentinel")
			original := []byte("preserve this unrelated file\n")
			if err := os.WriteFile(sentinel, original, 0600); err != nil {
				t.Fatal(err)
			}
			uid := uint32(os.Geteuid())
			var err error
			switch scenario {
			case "home-symlink":
				alias := filepath.Join(root, "alias")
				err = os.Symlink(home, alias)
				home = alias
			case "ssh-symlink":
				err = os.Remove(dir)
				if err == nil {
					err = os.Symlink(root, dir)
				}
			case "key-symlink":
				err = os.Symlink(sentinel, path)
			case "key-hardlink":
				err = os.Link(sentinel, path)
			case "key-directory":
				err = os.Mkdir(path, 0700)
			case "key-writable":
				err = os.WriteFile(path, original, 0666)
				if err == nil {
					err = os.Chmod(path, 0666)
				}
			case "ssh-writable":
				err = os.Chmod(dir, 0777)
			case "home-writable":
				err = os.Chmod(home, 0777)
			case "oversized":
				err = os.WriteFile(path, bytes.Repeat([]byte{'x'}, (1<<20)+1), 0600)
			case "wrong-owner":
				uid++
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := appendEnrollmentKey(context.Background(), home, enrollmentTestKey(t), uid); err == nil {
				t.Fatal("unsafe key target accepted")
			}
			got, err := os.ReadFile(sentinel)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatal("unrelated file was changed")
			}
		})
	}
}

func TestEnrollmentCancelledAppendPreservesKey(t *testing.T) {
	home := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := appendEnrollmentKey(ctx, home, enrollmentTestKey(t), uint32(os.Geteuid())); err == nil {
		t.Fatal("cancelled key append proceeded")
	}
	if _, err := os.Lstat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("cancelled append changed the key directory")
	}
}

func TestEnrollmentConcurrentWriterPreservesKey(t *testing.T) {
	home := t.TempDir()
	key := enrollmentTestKey(t)
	if err := appendEnrollmentKey(context.Background(), home, key, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(filepath.Join(home, ".ssh"), unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer unix.Flock(fd, unix.LOCK_UN)
	if err := appendEnrollmentKey(context.Background(), home, enrollmentTestKey(t), uint32(os.Geteuid())); err == nil {
		t.Fatal("a second writer bypassed the active enrollment write lock")
	}
	got, err := os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil || string(got) != key+"\n" {
		t.Fatal("concurrent refusal changed the existing key")
	}
}

func TestEnrollmentNativeEditorRacePreservesNewerFile(t *testing.T) {
	for _, replacement := range []bool{true, false} {
		name := "in-place edit"
		if replacement {
			name = "atomic pathname replacement"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".ssh")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "authorized_keys")
			original := enrollmentTestKey(t) + " original\n"
			newer := enrollmentTestKey(t) + " native editor without trailing newline"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			key := enrollmentTestKey(t)
			calls := 0
			write := func(file *os.File, data []byte) (int, error) {
				calls++
				// This is the former Fstatat-to-Renameat race window: a native
				// editor acts after Soda's last pre-write pathname check.
				target := path
				if replacement {
					target = filepath.Join(dir, "native-editor-new-file")
				}
				if err := os.WriteFile(target, []byte(newer), 0600); err != nil {
					return 0, err
				}
				if replacement {
					if err := os.Rename(target, path); err != nil {
						return 0, err
					}
				}
				return file.Write(data)
			}
			err := appendEnrollmentKeyWithWriter(context.Background(), home, key, uint32(os.Geteuid()), write)
			if !errors.Is(err, errEnrollmentWriteUncertain) || calls != 1 {
				t.Fatalf("concurrent native edit was not reported as uncertain: %v", err)
			}
			got, err := os.ReadFile(path)
			want := newer
			if !replacement {
				want += "\n" + key + "\n"
			}
			if err != nil || string(got) != want {
				t.Fatal("Soda overwrote or merged into the native editor's newer data")
			}
		})
	}
}

func TestEnrollmentConcurrentCreationIsNeverReplaced(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".ssh", "authorized_keys")
	native := enrollmentTestKey(t) + " created by native editor\n"
	write := func(file *os.File, data []byte) (int, error) {
		if err := os.WriteFile(path, []byte(native), 0600); err != nil {
			return 0, err
		}
		return file.Write(data)
	}
	if err := appendEnrollmentKeyWithWriter(context.Background(), home, enrollmentTestKey(t), uint32(os.Geteuid()), write); err == nil {
		t.Fatal("concurrent key-file creation was replaced")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != native {
		t.Fatal("native-created key file was changed")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatal("unpublished temporary key file remained")
	}
}

func TestEnrollmentPartialAppendDoesNotRollBackNativeData(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".ssh")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "authorized_keys")
	original := enrollmentTestKey(t) + " preserved original\n"
	if err := os.WriteFile(path, []byte(original), 0640); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var partial []byte
	write := func(file *os.File, data []byte) (int, error) {
		partial = bytes.Clone(data[:len(data)/2])
		n, err := file.Write(partial)
		if err != nil {
			return n, err
		}
		return n, errors.New("synthetic partial-write failure")
	}
	err = appendEnrollmentKeyWithWriter(context.Background(), home, enrollmentTestKey(t), uint32(os.Geteuid()), write)
	if !errors.Is(err, errEnrollmentWriteUncertain) {
		t.Fatalf("partial append did not report uncertainty: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, append([]byte(original), partial...)) {
		t.Fatal("partial failure rewrote or rolled back existing bytes")
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		t.Fatal("append replaced the native inode or changed its mode")
	}
}
