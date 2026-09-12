package scripts

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestTerminalBrandingMatchesCanonicalSymbol(t *testing.T) {
	cmd := exec.Command("python3", "render-terminal-logo.py", "--check")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("terminal symbol drift: %v\n%s", err, out)
	}
	motd, err := os.ReadFile("../assets/branding/terminal/motd.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(string(motd), "$\x1b") {
		t.Fatal("MOTD must not leak fastfetch tags or terminal escape codes")
	}
}

func TestFastfetchRendersTerminalBranding(t *testing.T) {
	binary, err := exec.LookPath("fastfetch")
	if err != nil {
		t.Skip("fastfetch is not installed on this test host")
	}
	cmd := exec.Command(binary, "--config", "../assets/branding/terminal/fastfetch.jsonc", "--logo", "../assets/branding/terminal/sodaos.txt", "--structure", "break", "--pipe", "false")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fastfetch: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "\x1b[38;2;223;0;27m") || !strings.Contains(text, "\x1b[39m") {
		t.Fatal("missing canonical red or terminal foreground reset")
	}
	plain := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(text, "")
	rows := strings.Split(strings.TrimRight(plain, "\n"), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	motd, err := os.ReadFile("../assets/branding/terminal/motd.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSuffix(string(motd), "\n\nSODA OS\n")
	if strings.Join(rows, "\n") != want {
		t.Fatal("fastfetch changed the symbol geometry or leaked color placeholders")
	}
}
