// Ordinary exec into a different-UID PostgreSQL workload. Go port of the
// retired tests/installed/workload-exec.py: no lifecycle, credentials,
// process attachment, data writes or workload replacement. It is not
// appliance runtime code.
package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// workloadExecProbe is piped to sh -se over the selected member session. The
// bytes match the retired Python probe exactly, including the lowercase
// select the SQL-locality gate ignores.
const workloadExecProbe = `set -eu
podman exec workload_database_1 grep -Eq '^Uid:[[:space:]]+999[[:space:]]+999[[:space:]]+999[[:space:]]+999$' /proc/1/status
test "$(podman exec workload_database_1 id -u)" = 0
test "$(podman exec --user postgres workload_database_1 id -u)" = 999
podman exec --user postgres workload_database_1 psql -X -U developer -d soda_example -At -c 'select current_database()'
podman exec -t --user postgres workload_database_1 true
`

// WorkloadExecUsage documents the probe's CLI surface.
const WorkloadExecUsage = "soda-installed-probes workload-exec FIXTURE_DIR"

// loadExecTarget reads the plain target.json and validates the project IP.
func loadExecTarget(fixture string) (string, error) {
	data, err := os.ReadFile(filepath.Join(fixture, "target.json"))
	if err != nil {
		return "", err
	}
	var target struct {
		IP string `json:"ip"`
	}
	if json.Unmarshal(data, &target) != nil {
		return "", errors.New("invalid exec target")
	}
	addr, err := netip.ParseAddr(target.IP)
	if err != nil || !addr.Is4() {
		return "", errors.New("project IP required")
	}
	if !netip.MustParsePrefix("10.89.0.0/24").Contains(addr) {
		return "", errors.New("project IP outside the selected project network")
	}
	return target.IP, nil
}

// execBaseArgs builds the exact member SSH argv the Python probe used.
func execBaseArgs(fixture string) []string {
	return []string{"-F", filepath.Join(fixture, "developer-client.conf"), "-o", "BatchMode=yes", "-o", "ForwardAgent=no", "-o", "ClearAllForwardings=yes"}
}

// execDifferentUID runs the UID-999 checks and requires the database name.
func execDifferentUID(base []string, owner string) error {
	args := append(append([]string{}, base...), owner, "sh -se")
	outcome, err := runBounded("ssh", args, []byte(workloadExecProbe), 60*time.Second)
	if err != nil {
		return err
	}
	if outcome.exitCode != 0 {
		return errors.New("different-UID exec failed; keep native state for diagnosis")
	}
	if string(bytes.TrimSpace(outcome.stdout)) != "soda_example" {
		return errors.New("different-UID exec returned an unexpected database")
	}
	return nil
}

// checkMemberDenied requires the ordinary member's engine denial.
func checkMemberDenied(base []string, target string) error {
	args := append(append([]string{}, base...), target, "podman exec workload_database_1 true")
	outcome, err := runBounded("ssh", args, nil, 30*time.Second)
	if err != nil {
		return err
	}
	if outcome.exitCode == 0 || !strings.Contains(strings.ToLower(string(outcome.stderr)), "permission denied") {
		return errors.New("expected socket permission denial, not transport failure")
	}
	return nil
}

// RunWorkloadExec is the workload-exec entrypoint: FIXTURE_DIR.
func RunWorkloadExec(args []string, stdout io.Writer) error {
	if err := runWorkloadExec(args, stdout); err != nil {
		return failParen("Workload exec check incomplete (", ").", err)
	}
	return nil
}

func runWorkloadExec(args []string, stdout io.Writer) error {
	if os.Getenv("SODA_NATIVE_VALIDATE") != "soda-test" {
		return errors.New("explicit soda-test validation required")
	}
	if len(args) != 1 {
		return errors.New(WorkloadExecUsage)
	}
	if err := privateDir(args[0], true); err != nil {
		return err
	}
	ip, err := loadExecTarget(args[0])
	if err != nil {
		return err
	}
	base := execBaseArgs(args[0])
	if err := execDifferentUID(base, "u08-alice-8417@"+ip); err != nil {
		return err
	}
	if err := checkMemberDenied(base, "u08-bob-8417@"+ip); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "Root/default-user, explicit PostgreSQL-user and PTY exec passed against UID-999 PID 1; ordinary member denied engine access.")
	return err
}
