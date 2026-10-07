// All-project before/after snapshot and comparison with no lifecycle action.
// Go port of the retired tests/installed/lifecycle-state.py. Snapshot mode
// must run with the working directory inside the source checkout so the VM
// driver, retained bindings and project snapshot payload resolve; compare
// mode needs only the fixture directory. It is not appliance runtime code.
package acceptance

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// LifecycleStateUsage documents the probe's CLI surface.
const LifecycleStateUsage = "soda-installed-probes lifecycle-state FIXTURE_DIR {snapshot LABEL|compare BEFORE AFTER}"

// hostSodaTables is the dashboard observation surface, unchanged from the
// retired probe.
var hostSodaTables = []string{"users", "keys", "projects", "memberships"}

// shQuote single-quotes one remote-shell word; single quotes inside are
// closed, escaped and reopened, so arbitrary operator paths stay one word.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// snapReportError marks lifecycle failures whose operational message is safe
// to report alongside the failure type.
type snapReportError struct{ msg string }

func (e *snapReportError) Error() string { return e.msg }

// lifecycleRun executes one VM observation with the probe's bound.
func lifecycleRun(dir string, args []string, stdin []byte) ([]byte, error) {
	outcome, err := runBoundedDirEnv(args[0], args[1:], stdin, nil, dir, 180*time.Second)
	if err != nil {
		return nil, err
	}
	if outcome.exitCode != 0 {
		detail := "operator observation failed"
		if bytes.HasPrefix(outcome.stderr, []byte("Project snapshot failed:")) {
			detail = string(bytes.TrimSpace(outcome.stderr))
		}
		return nil, &snapReportError{msg: "Snapshot command failed; no empty substitution: " + detail}
	}
	return outcome.stdout, nil
}

// repoRoot resolves the source checkout through git.
func repoRoot() (string, error) {
	outcome, err := runBounded("git", []string{"rev-parse", "--show-toplevel"}, nil, 30*time.Second)
	if err != nil {
		return "", err
	}
	if outcome.exitCode != 0 {
		return "", errors.New("snapshot must run inside the source checkout")
	}
	root := string(bytes.TrimSpace(outcome.stdout))
	if !filepath.IsAbs(root) {
		return "", errors.New("snapshot must run inside the source checkout")
	}
	return root, nil
}

// snapshotEntry selects one project snapshot.
type snapshotEntry struct {
	identifier string
	login      string
	workloads  bool
}

// snapshotInputs bundles the validated snapshot prerequisites.
type snapshotInputs struct {
	targetID string
	vm       string
	entries  []snapshotEntry
}

// loadSnapshotInputs reads the fixture target and retained bindings.
func loadSnapshotInputs(root, repo string) (snapshotInputs, error) {
	var inputs snapshotInputs
	data, err := os.ReadFile(filepath.Join(root, "target.json"))
	if err != nil {
		return inputs, err
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return inputs, errors.New("invalid snapshot target")
	}
	idRaw, ok := raw["environment_id"]
	if !ok || json.Unmarshal(idRaw, &inputs.targetID) != nil {
		return inputs, errors.New("snapshot environment required")
	}
	bindings, err := os.ReadFile(filepath.Join(repo, ".artifacts/test-vm/u08-8417a90/observed-bindings.json"))
	if err != nil {
		return inputs, err
	}
	var old []map[string]json.RawMessage
	if json.Unmarshal(bindings, &old) != nil {
		return inputs, errors.New("invalid retained bindings")
	}
	entries, err := snapshotEntries(old, inputs.targetID)
	if err != nil {
		return inputs, err
	}
	inputs.vm = filepath.Join(repo, "target/debug/soda-test-vm")
	inputs.entries = entries
	return inputs, nil
}

// snapshotEntries builds the three project entries with unique identities.
func snapshotEntries(old []map[string]json.RawMessage, targetID string) ([]snapshotEntry, error) {
	var entries []snapshotEntry
	for _, binding := range old {
		idRaw, idOK := binding["environmentID"]
		loginRaw, loginOK := binding["login"]
		var identifier, login string
		if !idOK || !loginOK || json.Unmarshal(idRaw, &identifier) != nil || json.Unmarshal(loginRaw, &login) != nil {
			return nil, errors.New("invalid retained binding")
		}
		entries = append(entries, snapshotEntry{identifier: identifier, login: login, workloads: login == "u08-alice-8417"})
	}
	entries = append(entries, snapshotEntry{identifier: targetID, login: "u08-alice-8417", workloads: true})
	if len(entries) != 3 {
		return nil, errors.New("three project snapshots required")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if seen[entry.identifier] {
			return nil, errors.New("three project snapshots required")
		}
		seen[entry.identifier] = true
	}
	return entries, nil
}

// queryHostSoda observes the host associations. The config and the
// database file travel over plain remote reads; the dump runs locally
// against a private copy, so no interpreter is needed on the target. A
// torn copy fails closed on the same integrity check the retired probe
// ran. Row order is deterministic per run; it sorts by the canonical
// JSON rendering rather than Python repr, so snapshots from the retired
// probe are not order-identical (same tables, same rows).
func queryHostSoda(vm, repo string) (any, error) {
	configRaw, err := lifecycleRun(repo, []string{vm, "ssh", "cat /etc/soda/dashboard.json"}, nil)
	if err != nil {
		return nil, err
	}
	var config struct {
		Database string `json:"database"`
	}
	if json.Unmarshal(configRaw, &config) != nil || config.Database == "" || strings.ContainsAny(config.Database, "\x00\r\n") {
		return nil, errors.New("invalid host observation")
	}
	dbRaw, err := lifecycleRun(repo, []string{vm, "ssh", "cat " + shQuote(config.Database)}, nil)
	if err != nil {
		return nil, err
	}
	data, err := dumpHostSoda(dbRaw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// dumpHostSoda queries one dashboard database copy: integrity first, then
// the four tables with canonically ordered rows.
func dumpHostSoda(dbRaw []byte) (map[string]any, error) {
	dir, err := os.MkdirTemp("", "soda-snapshot-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	dbPath := filepath.Join(dir, "dashboard.db")
	if err := os.WriteFile(dbPath, dbRaw, 0o600); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	var check string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&check); err != nil || check != "ok" {
		return nil, errors.New("invalid host observation")
	}
	data := map[string]any{}
	for _, table := range hostSodaTables {
		rows, err := dumpHostTable(db, table)
		if err != nil {
			return nil, err
		}
		data[table] = rows
	}
	return data, nil
}

// dumpHostTable reads one table with canonically ordered rows. Values
// normalize to JSON natives; blobs decode as text like the retired probe's
// str path.
func dumpHostTable(db *sql.DB, table string) ([]any, error) {
	rows, err := db.Query("select * from " + table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []any{}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make([]any, len(columns))
		for i, value := range values {
			switch value := value.(type) {
			case []byte:
				row[i] = string(value)
			case time.Time:
				row[i] = value.UTC().Format(time.RFC3339Nano)
			default:
				row[i] = value
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := json.Marshal(out[i])
		right, _ := json.Marshal(out[j])
		return string(left) < string(right)
	})
	return out, nil
}

// snapshotProject captures one project's identity and state.
func snapshotProject(vm, repo string, program []byte, entry snapshotEntry) (string, any, error) {
	if !regexp.MustCompile(`^p[0-9a-f]{24}$`).MatchString(entry.identifier) {
		return "", nil, errors.New("invalid project identifier")
	}
	if entry.login != "u08-alice-8417" && entry.login != "u08-bob-8417" {
		return "", nil, errors.New("invalid project login")
	}
	identity, ip, err := observeProjectEndpoint(vm, repo, entry.identifier)
	if err != nil {
		return "", nil, err
	}
	return execProjectSnapshot(vm, repo, program, entry, identity, ip)
}

// observeProjectEndpoint reads a project's container identity and IP.
func observeProjectEndpoint(vm, repo, identifier string) (string, string, error) {
	identityRaw, err := lifecycleRun(repo, []string{vm, "ssh", "podman inspect --format \"{{.Id}} {{.Image}} {{.Name}}\" soda-" + identifier}, nil)
	if err != nil {
		return "", "", err
	}
	networkRaw, err := lifecycleRun(repo, []string{vm, "ssh", "podman inspect --format \"{{json .NetworkSettings.Networks}}\" soda-" + identifier}, nil)
	if err != nil {
		return "", "", err
	}
	ip, err := projectSnapshotIP(networkRaw)
	if err != nil {
		return "", "", err
	}
	return string(bytes.TrimSpace(identityRaw)), ip, nil
}

// remoteSnapshotWrapper stages piped stdin as an executable temporary,
// runs the project-state subcommand, and always removes the temporary.
// It mirrors the Rust driver's remote template, so the probe is piped
// over the session and never installed. Double quotes only: the wrapper
// travels single-quoted through the remote shell.
const remoteSnapshotWrapper = `tmp=$(mktemp) && cat > "$tmp" && chmod 700 "$tmp" && "$tmp" project-state; rc=$?; rm -f "$tmp"; exit $rc`

// remotePayload reads the prebuilt acceptance probe. Like soda-test-vm,
// the operator builds it once per checkout:
//
//	cargo build -p soda-acceptance --bin soda-acceptance-remote
func remotePayload(repo string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(repo, "target/debug/soda-acceptance-remote"))
	if err != nil {
		return nil, errors.New("prebuilt soda-acceptance-remote missing; run: cargo build -p soda-acceptance --bin soda-acceptance-remote")
	}
	return data, nil
}

// execProjectSnapshot runs the project snapshot payload in the container.
func execProjectSnapshot(vm, repo string, program []byte, entry snapshotEntry, identity, ip string) (string, any, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !projectSubnet.Contains(addr) {
		return "", nil, errors.New("invalid project IP")
	}
	workloads := "0"
	if entry.workloads {
		workloads = "1"
	}
	command := "podman exec -i --env SODA_PROJECT_IP=" + addr.String() + " --env SODA_EXPECT_WORKLOADS=" + workloads + " soda-" + entry.identifier + " sh -c '" + remoteSnapshotWrapper + "'"
	stateRaw, err := lifecycleRun(repo, []string{vm, "ssh", command}, program)
	if err != nil {
		return "", nil, err
	}
	var state map[string]any
	if json.Unmarshal(stateRaw, &state) != nil {
		return "", nil, errors.New("invalid project state")
	}
	if len(asStateMap(state["people"])) == 0 || len(asStateMap(state["files"])) == 0 {
		return "", nil, errors.New("empty project state")
	}
	return identity, state, nil
}

// projectSubnet bounds the container addresses the snapshot accepts.
var projectSubnet = netip.MustParsePrefix("10.89.0.0/24")

// projectSnapshotIP extracts the validated project IP.
func projectSnapshotIP(networkRaw []byte) (string, error) {
	var network map[string]struct {
		IPAddress string `json:"IPAddress"`
	}
	if json.Unmarshal(networkRaw, &network) != nil {
		return "", errors.New("invalid project network")
	}
	entry, ok := network["soda-projects"]
	if !ok {
		return "", errors.New("invalid project network")
	}
	addr, err := netip.ParseAddr(entry.IPAddress)
	if err != nil || !addr.Is4() || !netip.MustParsePrefix("10.89.0.0/24").Contains(addr) {
		return "", errors.New("project IP outside the selected project network")
	}
	return entry.IPAddress, nil
}

// asStateMap reads a state section as a map.
func asStateMap(section any) map[string]any {
	mapped, _ := section.(map[string]any)
	return mapped
}

// writeSnapshot records the snapshot with sorted keys and no trailing data.
func writeSnapshot(root, label string, stable map[string]any, boot string) error {
	path := filepath.Join(root, label+".json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]any{"stable": stable, "boot_id": boot}); err != nil {
		_ = file.Close()
		return err
	}
	_, writeErr := file.Write(bytes.TrimSuffix(encoded.Bytes(), []byte("\n")))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// runSnapshot captures and records a fresh snapshot.
func runSnapshot(root, label string, stdout io.Writer) error {
	repo, err := repoRoot()
	if err != nil {
		return err
	}
	return runSnapshotAt(root, label, repo, stdout)
}

// runSnapshotAt captures a snapshot against an explicit checkout root.
func runSnapshotAt(root, label, repo string, stdout io.Writer) error {
	inputs, err := loadSnapshotInputs(root, repo)
	if err != nil {
		return err
	}
	soda, err := queryHostSoda(inputs.vm, repo)
	if err != nil {
		return err
	}
	program, err := remotePayload(repo)
	if err != nil {
		return err
	}
	projects := map[string]any{}
	for _, entry := range inputs.entries {
		identity, state, err := snapshotProject(inputs.vm, repo, program, entry)
		if err != nil {
			return err
		}
		projects[entry.identifier] = map[string]any{"identity": identity, "state": state}
	}
	bootRaw, err := lifecycleRun(repo, []string{inputs.vm, "ssh", "head -c 64 /proc/sys/kernel/random/boot_id"}, nil)
	if err != nil {
		return err
	}
	boot := string(bytes.TrimSpace(bootRaw))
	if !regexp.MustCompile(`^[a-f0-9-]{36}$`).MatchString(boot) {
		return errors.New("invalid boot identity")
	}
	if err := writeSnapshot(root, label, map[string]any{"soda": soda, "projects": projects}, boot); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "Complete bounded snapshot captured for three projects and Soda associations; no secret/private-key contents exported.")
	return err
}

// runCompare requires two snapshots to share stable state.
func runCompare(root string, labels []string, stdout io.Writer) error {
	if len(labels) != 2 {
		return errors.New(LifecycleStateUsage)
	}
	snapshots := make([]map[string]any, 2)
	for i, label := range labels {
		data, err := os.ReadFile(filepath.Join(root, label+".json"))
		if err != nil {
			return err
		}
		if json.Unmarshal(data, &snapshots[i]) != nil {
			return errors.New("invalid snapshot")
		}
	}
	a, aOK := snapshots[0]["stable"]
	b, bOK := snapshots[1]["stable"]
	if !aOK || !bOK {
		return errors.New("snapshot stable state required")
	}
	if !reflect.DeepEqual(a, b) {
		return &snapReportError{msg: "Stable state differs; inspect private snapshots, do not repair by reset"}
	}
	_, err := fmt.Fprintln(stdout, "All declared project/account/Git/tool/workload/database state matches; boot identity compared separately.")
	return err
}

// validSnapshotLabel reports whether a label matches the probe's shape.
func validSnapshotLabel(label string) bool {
	return regexp.MustCompile(`^[a-z][a-z0-9-]{0,50}$`).MatchString(label)
}

// RunLifecycleState is the lifecycle-state entrypoint.
func RunLifecycleState(args []string, stdout io.Writer) error {
	err := runLifecycleState(args, stdout)
	if err == nil {
		return nil
	}
	var report *snapReportError
	if errors.As(err, &report) {
		return failDetail("Lifecycle snapshot/comparison incomplete: ", report.msg, err)
	}
	return fail("Lifecycle snapshot/comparison incomplete: ", err)
}

func runLifecycleState(args []string, stdout io.Writer) error {
	if os.Getenv("SODA_NATIVE_VALIDATE") != "soda-test" {
		return errors.New("explicit soda-test validation required")
	}
	if len(args) < 2 {
		return errors.New(LifecycleStateUsage)
	}
	if err := privateDir(args[0], true); err != nil {
		return err
	}
	mode, labels := args[1], args[2:]
	for _, label := range labels {
		if !validSnapshotLabel(label) {
			return errors.New(LifecycleStateUsage)
		}
	}
	if mode == "compare" {
		return runCompare(args[0], labels, stdout)
	}
	if mode != "snapshot" || len(labels) != 1 {
		return errors.New(LifecycleStateUsage)
	}
	return runSnapshot(args[0], labels[0], stdout)
}
