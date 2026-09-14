package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerUsesSeparateIdentityAndNativeServiceCustody(t *testing.T) {
	w := Worker{Name: "soda-qualify-test", User: "soda-qualifier", Executable: "/usr/bin/true", Directory: "/run/soda-qualification", ReadOnly: []string{"/inputs:/run/soda-inputs"}, Writable: []string{"/evidence:/run/soda-qualification"}, Environment: []string{"HOME=/run/soda-qualification"}}
	args, err := w.arguments()
	require.NoError(t, err)
	joined := strings.Join(args, "\n")
	for _, value := range []string{"--property=User=soda-qualifier", "--property=ProtectHome=tmpfs", "--property=ProtectSystem=strict", "--property=KillMode=control-group", "--property=InaccessiblePaths=-/var/lib/soda-release -/root", "--property=BindReadOnlyPaths=/inputs:/run/soda-inputs"} {
		require.Contains(t, joined, value)
	}
	require.NotContains(t, joined, "--replace")
	w.User = "root"
	_, err = w.arguments()
	require.Error(t, err)
	w.User = "soda-qualifier"
	w.Environment = []string{"AWS_SECRET_ACCESS_KEY=synthetic"}
	_, err = w.arguments()
	require.Error(t, err)
	w.Environment = nil
	w.Writable = []string{"/safe:/run/safe /secret:/exposed"}
	_, err = w.arguments()
	require.Error(t, err)
}

func TestWorkerRefusesUntrustedExecutable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "driver")
	require.NoError(t, os.WriteFile(file, []byte("#!/bin/false\n"), 0755))
	require.Error(t, TrustedExecutable(file))
	require.Error(t, TrustedExecutable("relative"))
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink("/usr/bin/true", link))
	require.Error(t, TrustedExecutable(link))
}
