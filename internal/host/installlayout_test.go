package host

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/installlayout"
)

func TestLifecycleRefusesOtherPackagingLayout(t *testing.T) {
	other := "/usr/lib/systemd/system/soda-project@.service"
	if other == installlayout.ProjectUnit {
		other = "/etc/systemd/system/soda-project@.service"
	}
	exec := managementExec(func(_ context.Context, _ []byte, cmd string, args ...string) ([]byte, error) {
		if cmd == "/usr/bin/systemctl" {
			if args[0] != "show" {
				t.Fatal("mutated unit from another packaging layout")
			}
			return []byte("LoadState=loaded\nFragmentPath=" + other + "\nDropInPaths=\nUnitFileState=enabled\n"), nil
		}
		return []byte(fmt.Sprintf(`{"id":%q,"running":true,"project":"p0123456789abcdef01234567","owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64))), nil
	})
	d := testDaemon(exec, Config{})
	if _, err := d.lifecycle(t.Context(), Lifecycle{Project: "p0123456789abcdef01234567", Action: "stop"}); err == nil {
		t.Fatal("other packaging layout accepted")
	}
}
