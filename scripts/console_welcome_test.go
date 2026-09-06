package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Command doubles exercise the preserved native discovery without reading a
// real appliance configuration, querying Tailscale or changing network state.
func consoleFixture(t *testing.T, uid, configuration string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	commands := map[string]string{
		"id":          "printf '%s\\n' '" + uid + "'",
		"hostnamectl": "echo native-appliance",
		"nmcli":       "printf 'eth0:ethernet:connected\\nwifi0:wifi:connected\\ntailscale0:tun:connected\\n'",
		"ip": `if [ "$3" = route ]; then
printf 'default via 192.168.1.1 dev eth0\ndefault dev tailscale0\n'
elif [ "$6" = eth0 ]; then
printf '2: eth0 inet 192.168.1.10/24 scope global eth0\n'
elif [ "$6" = wifi0 ]; then
printf '3: wifi0 inet 192.168.2.10/24 scope global wifi0\n'
fi`,
		"soda-tailnet": "echo 'Tailnet: native status fixture'",
	}
	for name, source := range commands {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+source+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(dir, "dashboard.json")
	if err := os.WriteFile(config, []byte(configuration), 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "PATH=") {
			env = append(env, v)
		}
	}
	env = append(env, "PATH="+dir+":"+os.Getenv("PATH"))
	return config, env
}

func TestConsoleUsesConfiguredOriginsAndNativeUplinks(t *testing.T) {
	config, env := consoleFixture(t, "0", `{"public_url":"https://soda.example.test","forgejo_url":"https://forgejo.example.test","oauth_secret":"never-print-this"}`)
	cmd := exec.Command("sh", "../appliance/bin/soda-console-welcome", config)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(output))
	}
	text := string(output)
	for _, want := range []string{"native-appliance", "Observed local IPv4 (eth0): 192.168.1.10", "Observed local IPv4 (wifi0): 192.168.2.10", "https://soda.example.test", "https://forgejo.example.test", "ssh -N -L 9090:127.0.0.1:9090 root@HOST", "not a listener"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	for _, bad := range []string{"never-print-this", ":30000", "https://192.168.1.10:9090", "Observed local IPv4 (tailscale0)"} {
		if strings.Contains(text, bad) {
			t.Fatalf("unexpected %q", bad)
		}
	}
	if strings.Count(text, "Observed local IPv4 (eth0)") != 1 {
		t.Fatal("duplicate uplink")
	}
}

func TestConsoleDoesNotRenderNonOperatorOrUnsafeOrigins(t *testing.T) {
	config, env := consoleFixture(t, "1000", `{}`)
	cmd := exec.Command("sh", "../appliance/bin/soda-console-welcome", config)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatal("non-operator banner", err, string(out))
	}
	for _, origin := range []string{"https://name:private-value@soda.example.test", "https://@soda.example.test", "https://soda.example.test:bad-port", "https://soda.example.test:65536"} {
		config, env = consoleFixture(t, "0", `{"public_url":"`+origin+`","forgejo_url":"https://forgejo.example.test"}`)
		cmd = exec.Command("sh", "../appliance/bin/soda-console-welcome", config)
		cmd.Env = env
		out, err = cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "complete operator setup") || strings.Contains(string(out), origin) {
			t.Fatal("unsafe origin handling", err, string(out))
		}
	}
}

func TestConsoleHookKeepsNoninteractiveSSHQuiet(t *testing.T) {
	cmd := exec.Command("sh", "-c", ". ../appliance/config/console-welcome.sh")
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatal("noninteractive output", err, string(out))
	}
}
