package host

// Opt-in product-owned native proof. Build this package's test binary on the
// matching native builder; run ONLY with a separately approved fixture/inputs.
// It starts a temporary root-private helper, not an installed/public service.
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
)

// Read only the probe's anchored output markers, not echoed command text. Bash
// bracketed-paste mode writes CSI + carriage return before command output; PTY
// bytes are not already a rendered screen. Keep raw terminal bytes out of logs.
func terminalProbeText(value string) string {
	value = regexp.MustCompile("\\x1b\\[[0-?]*[ -/]*[@-~]").ReplaceAllString(value, "")
	return strings.ReplaceAll(value, "\r", "\n")
}

func TestTerminalProbeNativeControlSequences(t *testing.T) {
	text := "prompt$ printf '__SODA_SIZE__%s\\n' ...\r\n\x1b[?2004l\r__SODA_SIZE__37 103\r\n\x1b[?2004hprompt$ "
	if !regexp.MustCompile(`(?m)^__SODA_SIZE__37 103$`).MatchString(terminalProbeText(text)) {
		t.Fatal("native PTY marker was not recognized")
	}
	if regexp.MustCompile(`(?m)^__SODA_SIZE__`).MatchString(terminalProbeText("prompt$ printf '__SODA_SIZE__%s\\n' ...\r\n")) {
		t.Fatal("echo was mistaken for native evidence")
	}
}

func TestInstalledTerminalBoundary(t *testing.T) {
	input := os.Getenv("SODA_TERMINAL_NATIVE_INPUT")
	if input == "" {
		t.Skip("opt-in native fixture proof; no target selected")
	}
	var request struct {
		Target    string `json:"target"`
		Protocol  string `json:"terminal_protocol"`
		Container string `json:"container_id"`
		Project   string `json:"project"`
		Accounts  []struct {
			Login    string `json:"login"`
			Identity int64  `json:"identity"`
			UID      int    `json:"uid"`
			GID      int    `json:"gid"`
			Home     string `json:"home"`
			Groups   []int  `json:"groups"`
			Admin    bool   `json:"admin"`
		} `json:"accounts"`
	}
	hostname, _ := os.Hostname()
	info, err := os.Lstat(input)
	if err != nil || !filepath.IsAbs(input) || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 8192 || os.Geteuid() != 0 {
		t.Fatal("invalid private native input or execution identity")
	}
	f, err := os.Open(input)
	if err != nil {
		t.Fatal("native input unavailable")
	}
	err = strictjson.Decode(f, &request)
	f.Close()
	if err != nil || request.Protocol != "managed-tmux-v1" || !containerID.MatchString(request.Container) || request.Target != hostname || os.Getenv("SODA_NATIVE_VALIDATE") != hostname || !projectID.MatchString(request.Project) || len(request.Accounts) != 2 {
		t.Fatal("native target/account scope mismatch")
	}
	seen := map[int64]bool{}
	for _, a := range request.Accounts {
		if !loginName.MatchString(a.Login) || a.Login == "root" || a.Identity <= 0 || seen[a.Identity] || a.UID <= 0 || a.GID < 0 || !filepath.IsAbs(a.Home) || len(a.Groups) == 0 {
			t.Fatal("invalid declared native account")
		}
		seen[a.Identity] = true
	}
	dir := filepath.Dir(input)
	parent, err := os.Lstat(dir)
	if err != nil || !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		t.Fatal("native input parent must be private")
	}
	marker, err := os.OpenFile(filepath.Join(dir, "terminal-proof-started"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("occupied native proof; do not replay")
	}
	marker.Close()
	socket := filepath.Join(dir, "host.sock")
	if len(socket) > 100 {
		t.Fatal("native socket path too long")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal("native socket unavailable")
	}
	if err = os.Chmod(socket, 0600); err != nil {
		listener.Close()
		t.Fatal("native socket protection failed")
	}
	d := &Daemon{Exec: Native{}}
	server := &http.Server{Handler: d, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Close()
	defer d.CloseTerminals()
	// A parent probe may kill only this exact owned child helper to test abrupt
	// loss. No installed helper/service is ever signalled.
	if os.Getenv("SODA_TERMINAL_HELPER_CHILD") == "1" {
		select {}
	}
	c := NewClient(socket)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cid, err := d.terminalContainer(ctx, request.Project)
	if err != nil || cid != request.Container {
		t.Fatal("native project binding unavailable")
	}
	newID := func() string {
		var value [16]byte
		if _, err := rand.Read(value[:]); err != nil {
			t.Fatal("identifier unavailable")
		}
		return hex.EncodeToString(value[:])
	}
	openManaged := func(client *Client, in TerminalRequest) (*Terminal, *Terminal) {
		in.Action = "create"
		owner, err := client.OpenTerminal(ctx, in)
		if err != nil {
			t.Fatal("managed owner unavailable")
		}
		t.Cleanup(owner.Close)
		if f, err := owner.Receive(ctx); err != nil || f.Type != "ready" {
			t.Fatal("managed creation unconfirmed")
		}
		in.Action = "attach"
		stream, err := client.OpenTerminal(ctx, in)
		if err != nil {
			t.Fatal("managed attach unavailable")
		}
		t.Cleanup(stream.Close)
		if f, err := stream.Receive(ctx); err != nil || f.Type != "ready" {
			t.Fatal("managed attachment unconfirmed")
		}
		return owner, stream
	}

	// Output stays in bounded memory and is never attached to a test failure/log.
	command := func(stream *Terminal, cmd string) {
		if err := stream.Send(ctx, TerminalFrame{Type: "input", Data: base64.StdEncoding.EncodeToString([]byte(cmd + "\n"))}); err != nil {
			t.Fatal("native command input failed")
		}
	}
	waitText := func(stream *Terminal, pattern *regexp.Regexp) []string {
		var output strings.Builder
		deadline, stop := context.WithTimeout(ctx, 10*time.Second)
		defer stop()
		for output.Len() < 65536 {
			frame, err := stream.Receive(deadline)
			if err != nil || frame.Type != "output" {
				t.Fatal("native output not confirmed", frame.Type, frame.Reason, "bytes", output.Len())
			}
			data, _ := base64.StdEncoding.DecodeString(frame.Data)
			output.Write(data)
			if found := pattern.FindStringSubmatch(terminalProbeText(output.String())); found != nil {
				return found
			}
		}
		t.Fatal("native output limit exceeded")
		return nil
	}
	// A genuine marker/actor mismatch must refuse without changing the account.
	wrong := TerminalRequest{Action: "create", ID: newID(), Project: request.Project, Login: request.Accounts[0].Login, Identity: request.Accounts[1].Identity, Cols: 80, Rows: 24, Expires: time.Now().Add(90 * time.Second).Unix()}
	denied, err := c.OpenTerminal(ctx, wrong)
	if err != nil {
		t.Fatal("native refusal transport unavailable")
	}
	frame, err := denied.Receive(ctx)
	denied.Close()
	if err != nil || frame.Type != "closed" || frame.Reason != "launch_failed" {
		t.Fatal("native identity mismatch was not refused")
	}
	results := []map[string]any{}
	for _, a := range request.Accounts {
		in := TerminalRequest{ID: newID(), Project: request.Project, Login: a.Login, Identity: a.Identity, Cols: 80, Rows: 24, Expires: time.Now().Add(90 * time.Second).Unix()}
		owner, stream := openManaged(c, in)
		t.Log("checking native account facts", a.Identity)
		command(stream, `/usr/bin/python3 -c 'import os,json,shutil; print("__SODA_FACTS__"+json.dumps({"uid":os.getuid(),"gid":os.getgid(),"resuid":os.getresuid(),"resgid":os.getresgid(),"groups":os.getgroups(),"home":os.environ.get("HOME"),"cwd":os.getcwd(),"tty":os.isatty(0),"pid":os.getppid(),"shell":os.environ.get("SHELL"),"mise":shutil.which("mise"),"podman":shutil.which("podman"),"mise_data":os.environ.get("MISE_DATA_DIR"),"mise_config":os.environ.get("MISE_GLOBAL_CONFIG_FILE"),"container_host":os.environ.get("CONTAINER_HOST")} ))'`)
		value := waitText(stream, regexp.MustCompile(`(?m)^__SODA_FACTS__(\{[^\r\n]+\})\r?$`))
		var actual struct {
			UID           int
			GID           int
			Groups        []int
			ResUID        []int
			ResGID        []int
			Home          string
			Cwd           string
			TTY           bool
			PID           int
			Shell         string
			Mise          string
			Podman        string
			MiseData      string `json:"mise_data"`
			MiseConfig    string `json:"mise_config"`
			ContainerHost string `json:"container_host"`
		}
		if json.Unmarshal([]byte(value[1]), &actual) != nil {
			t.Fatal("native facts invalid")
		}
		sort.Ints(actual.Groups)
		sort.Ints(a.Groups)
		if actual.UID != a.UID || actual.GID != a.GID || !reflect.DeepEqual(actual.ResUID, []int{a.UID, a.UID, a.UID}) || !reflect.DeepEqual(actual.ResGID, []int{a.GID, a.GID, a.GID}) || actual.Home != a.Home || actual.Cwd != a.Home || !actual.TTY || actual.PID <= 1 || !reflect.DeepEqual(actual.Groups, a.Groups) || actual.Shell != "/bin/bash" || actual.Mise == "" || actual.Podman == "" || actual.MiseData != "/opt/mise" || actual.MiseConfig != "/etc/mise/config.toml" || actual.ContainerHost != "unix:///run/soda-podman/podman.sock" {
			t.Fatal("native account/home/group/TTY mismatch")
		}
		if err = stream.Send(ctx, TerminalFrame{Type: "resize", Cols: 103, Rows: 37}); err != nil {
			t.Fatal("resize transport failed")
		}
		command(stream, `printf '__SODA_SIZE__%s\n' "$(stty size)"`)
		waitText(stream, regexp.MustCompile(`(?m)^__SODA_SIZE__37 103\r?$`))
		command(stream, `sudo -n id -u; printf '__SODA_SUDO__%s\n' "$?"`)
		sudo := waitText(stream, regexp.MustCompile(`(?m)^__SODA_SUDO__([0-9]+)\r?$`))
		if (sudo[1] == "0") != a.Admin {
			t.Fatal("native sudo boundary mismatch")
		}
		command(stream, "sleep 20")
		time.Sleep(200 * time.Millisecond)
		command(stream, "\x03printf '__SODA_INTERRUPT__%s\\n' ok")
		waitText(stream, regexp.MustCompile(`(?m)^__SODA_INTERRUPT__ok\r?$`))
		command(stream, `SODA_RESUME_PROBE=kept; printf '__SODA_SET__ok\n'`)
		waitText(stream, regexp.MustCompile(`(?m)^__SODA_SET__ok$`))
		observe := func() string {
			out, e := (Native{}).Run(ctx, nil, "/usr/bin/podman", "--remote=false", "exec", cid, "/usr/bin/python3", "-I", "-c", `import pathlib,sys; print(pathlib.Path('/proc/'+sys.argv[1]+'/stat').read_text().rsplit(')',1)[1].split()[19])`, strconv.Itoa(actual.PID))
			if e != nil {
				t.Fatal("independent shell start observation failed")
			}
			return strings.TrimSpace(string(out))
		}
		start := observe()
		stream.Close()
		time.Sleep(300 * time.Millisecond)
		if observe() != start {
			t.Fatal("detach replaced shell")
		}
		in.Action = "attach"
		stream, err = c.OpenTerminal(ctx, in)
		if err != nil {
			t.Fatal("reattach unavailable")
		}
		defer stream.Close()
		if f, e := stream.Receive(ctx); e != nil || f.Type != "ready" {
			t.Fatal("reattach unconfirmed")
		}
		command(stream, `printf '__SODA_RESUME__%s:%s\n' "$$" "$SODA_RESUME_PROBE"`)
		resumed := waitText(stream, regexp.MustCompile(`(?m)^__SODA_RESUME__([1-9][0-9]*):kept$`))
		if resumed[1] != strconv.Itoa(actual.PID) || observe() != start {
			t.Fatal("reattach replaced shell state")
		}
		if err = owner.Send(ctx, TerminalFrame{Type: "close"}); err != nil {
			t.Fatal("native End failed")
		}
		closeCtx, closeCancel := context.WithTimeout(ctx, 20*time.Second)
		for {
			frame, e := owner.Receive(closeCtx)
			if e != nil {
				closeCancel()
				t.Fatal("native teardown receipt missing")
			}
			if frame.Type == "closed" {
				if frame.Reason != "disconnected" {
					t.Fatal("native teardown unconfirmed")
				}
				break
			}
		}
		closeCancel()
		stream.Close()
		// Independent native process observation. PID is from the verified own shell,
		// never a browser-supplied kill target. This probe never sends a PID signal.
		out, e := Native{}.Run(ctx, nil, "/usr/bin/podman", "--remote=false", "exec", cid, "/usr/bin/python3", "-I", "-c", `import os,sys; print(int(os.path.exists('/proc/'+sys.argv[1])))`, strconv.Itoa(actual.PID))
		if e != nil || strings.TrimSpace(string(out)) != "0" {
			t.Fatal("native login process disappearance not confirmed")
		}
		absent, e := c.OpenTerminal(ctx, in)
		if e != nil {
			t.Fatal("ended attach refusal transport unavailable")
		}
		f, e := absent.Receive(ctx)
		absent.Close()
		if e != nil || f.Type != "closed" || f.Reason != "launch_failed" {
			t.Fatal("ended attach did not refuse")
		}
		results = append(results, map[string]any{"login": a.Login, "identity": a.Identity, "native_account_tty_resize_interrupt": true, "same_shell_start_and_memory_after_reattach": true, "ended_attach_refused": true, "sudo_boundary": true, "login_exit_observed": true})
	}
	// Exercise real remote teardown, not just a closed WebSocket or dead host CLI.
	// Foreground jobs and login PIDs are read back through an independent exec.
	teardown := []map[string]any{}
	for i, mode := range []string{"transport-eof", "silent-lease", "helper-killed"} {
		t.Log("checking native teardown", mode)
		client := c
		var child *exec.Cmd
		if mode == "helper-killed" {
			childDir := filepath.Join(dir, "loss")
			if err := os.Mkdir(childDir, 0700); err != nil {
				t.Fatal("occupied child proof")
			}
			body, _ := json.Marshal(request)
			childInput := filepath.Join(childDir, "request.json")
			if err := os.WriteFile(childInput, body, 0600); err != nil {
				t.Fatal("child input unavailable")
			}
			child = exec.Command(os.Args[0], "-test.run", "^TestInstalledTerminalBoundary$", "-test.timeout=3m")
			child.Env = append(os.Environ(), "SODA_TERMINAL_NATIVE_INPUT="+childInput, "SODA_TERMINAL_HELPER_CHILD=1")
			if err := child.Start(); err != nil {
				t.Fatal("child helper unavailable")
			}
			defer func() {
				if child.ProcessState == nil {
					_ = child.Process.Kill()
					_ = child.Wait()
				}
			}()
			childSocket := filepath.Join(childDir, "host.sock")
			until := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Lstat(childSocket); err == nil {
					break
				}
				if time.Now().After(until) {
					t.Fatal("child helper not ready")
				}
				time.Sleep(20 * time.Millisecond)
			}
			client = NewClient(childSocket)
		}
		a := request.Accounts[i%2]
		owner, stream := openManaged(client, TerminalRequest{ID: newID(), Project: request.Project, Login: a.Login, Identity: a.Identity, Cols: 80, Rows: 24, Expires: time.Now().Add(90 * time.Second).Unix()})
		command(stream, `printf '__SODA_LOGIN__%s\n' "$$"`)
		login := waitText(stream, regexp.MustCompile(`(?m)^__SODA_LOGIN__([1-9][0-9]*)$`))[1]
		command(stream, `/usr/bin/python3 -u -c 'import os,time,pathlib; parent=pathlib.Path("/proc/"+str(os.getppid())+"/stat").read_text().rsplit(")",1)[1].split()[1]; print("__SODA_JOB__"+str(os.getpid())+":"+parent); time.sleep(180)'`)
		owned := waitText(stream, regexp.MustCompile(`(?m)^__SODA_JOB__([1-9][0-9]*):([1-9][0-9]*)$`))
		job, launcher := owned[1], owned[2]
		started := time.Now()
		switch mode {
		case "transport-eof":
			owner.Close()
		case "silent-lease":
			for {
				frame, err := owner.Receive(ctx)
				if err != nil {
					t.Fatal("lease teardown receipt unavailable")
				}
				if frame.Type == "closed" {
					if frame.Reason != "expired" {
						t.Fatal("lease teardown reason mismatch")
					}
					break
				}
			}
		case "helper-killed":
			if err := child.Process.Kill(); err != nil {
				t.Fatal("owned helper kill failed")
			}
			_ = child.Wait()
		}
		until := started.Add(75 * time.Second)
		for {
			out, err := (Native{}).Run(ctx, nil, "/usr/bin/podman", "--remote=false", "exec", cid, "/usr/bin/python3", "-I", "-c", `import os,sys; print(int(any(os.path.exists('/proc/'+p) for p in sys.argv[1:])))`, login, job, launcher)
			if err != nil {
				t.Fatal("independent teardown observation failed")
			}
			if strings.TrimSpace(string(out)) == "0" {
				break
			}
			if time.Now().After(until) {
				t.Fatal("owned login/foreground job survived native deadline")
			}
			time.Sleep(time.Second)
		}
		stream.Close()
		teardown = append(teardown, map[string]any{"mode": mode, "login_exit_observed": true, "foreground_exit_observed": true, "launcher_exit_observed": true, "elapsed_seconds": time.Since(started).Seconds()})
	}
	// Independent SSH/workload preservation and browser integration are not
	// fabricated by this private-helper test; observe them separately.
	result, _ := json.MarshalIndent(map[string]any{"target": hostname, "project": request.Project, "accounts": results, "identity_mismatch_refused": true, "teardown": teardown, "scope": "managed tmux: account/TTY/profile/same-shell reattach/End/owner EOF/lease/helper-loss; not browser/editor/unrelated-service preservation"}, "", "  ")
	output, err := os.OpenFile(filepath.Join(dir, "terminal-proof.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("native evidence finalization failed")
	}
	_, err = output.Write(append(result, '\n'))
	closeErr := output.Close()
	if err != nil || closeErr != nil {
		t.Fatal("native evidence finalization failed")
	}
}
