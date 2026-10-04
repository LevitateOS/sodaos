// Real client/member HTTP and committed PostgreSQL data checks. Go port of
// the retired tests/installed/workload-access.py: this proves the retained
// project-network diagnostic-workload mode only, not the blocked default
// nested bridge. Client database operations use the psql CLI with a
// passfile, exactly like the member operations, instead of psycopg. It is
// not appliance runtime code.
package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// WorkloadAccessUsage documents the probe's CLI surface.
const WorkloadAccessUsage = "soda-installed-probes workload-access FIXTURE_DIR"

// Default project target when the fixture carries no target.json.
const (
	defaultAccessIP   = "10.89.0.2"
	defaultAccessMode = "project namespace (nested host mode)"
)

// Client SQL statements. Uppercase PostgreSQL DML against the ephemeral test
// workload; the SQL-locality gate explicitly excludes this file because it
// guards product SQLite, which this probe never touches.
const createProbeTableSQL = "CREATE TABLE IF NOT EXISTS soda_u08_probe (run_id text PRIMARY KEY, value text NOT NULL)"

func insertProbeSQL(run string) string {
	return "INSERT INTO soda_u08_probe VALUES ('" + run + "', 'client-committed')"
}

func selectProbeSQL(run string) string {
	return "SELECT value FROM soda_u08_probe WHERE run_id='" + run + "'"
}

func updateProbeSQL(run string) string {
	return "UPDATE soda_u08_probe SET value='bob-committed' WHERE run_id='" + run + "';\n"
}

// accessTarget is the probe target; network mode presence is tracked so a
// target.json without it fails at results time, exactly like the Python.
type accessTarget struct {
	ip      string
	mode    string
	modeSet bool
}

// loadAccessTarget reads the fixture target or the documented default.
func loadAccessTarget(root string) (accessTarget, error) {
	path := filepath.Join(root, "target.json")
	if _, err := os.Stat(path); err != nil {
		return accessTarget{ip: defaultAccessIP, mode: defaultAccessMode, modeSet: true}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return accessTarget{}, err
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return accessTarget{}, errors.New("invalid workload target")
	}
	target, err := parseAccessTarget(raw)
	if err != nil {
		return accessTarget{}, err
	}
	return target, nil
}

func parseAccessTarget(raw map[string]json.RawMessage) (accessTarget, error) {
	var target accessTarget
	var ip string
	ipRaw, ok := raw["ip"]
	if !ok || json.Unmarshal(ipRaw, &ip) != nil {
		return accessTarget{}, errors.New("project IP required")
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil || !addr.Is4() || !netip.MustParsePrefix("10.89.0.0/24").Contains(addr) {
		return accessTarget{}, errors.New("project IP outside the selected project network")
	}
	target.ip = ip
	if modeRaw, ok := raw["network_mode"]; ok {
		if json.Unmarshal(modeRaw, &target.mode) != nil {
			return accessTarget{}, errors.New("invalid workload network mode")
		}
		target.modeSet = true
	}
	return target, nil
}

// accessProbe carries the run state shared by the probe phases.
type accessProbe struct {
	output   string
	config   string
	ip       string
	run      string
	passfile string
}

// accessRemote runs a member command over the fixture SSH transport.
func (p *accessProbe) accessRemote(who, command string, stdin []byte) ([]byte, error) {
	args := []string{"-F", p.config, "u08-" + who + "-8417@" + p.ip, command}
	outcome, err := runBounded("ssh", args, stdin, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if outcome.exitCode != 0 {
		return nil, errors.New("native member operation failed")
	}
	return outcome.stdout, nil
}

// clientPSQL runs one client database statement through the psql CLI.
func (p *accessProbe) clientPSQL(statement string) ([]byte, error) {
	args := []string{"-X", "-w", "-h", p.ip, "-p", "5432", "-U", "developer", "-d", "soda_example", "-At", "-v", "ON_ERROR_STOP=1", "-c", statement}
	env := []string{"PGPASSFILE=" + p.passfile, "PGCONNECT_TIMEOUT=10"}
	outcome, err := runBoundedEnv("psql", args, nil, env, 60*time.Second)
	if err != nil {
		return nil, err
	}
	if outcome.exitCode != 0 {
		return nil, errors.New("client database operation failed")
	}
	return outcome.stdout, nil
}

// memberPSQLPrefix builds the remote psql invocation for member checks.
func (p *accessProbe) memberPSQLPrefix() string {
	return "PGPASSFILE=\"$HOME/.config/u08-db-client-" + p.run + "/pgpass\" psql -X -w -h " + p.ip + " -p 5432 -U developer -d soda_example -At -v ON_ERROR_STOP=1"
}

// setupAccessOutput creates the run-owned output directory.
func setupAccessOutput(root, run string) (string, error) {
	output := filepath.Join(root, "u08-workload-access-"+run)
	if err := os.Mkdir(output, 0o700); err != nil {
		return "", err
	}
	return output, nil
}

// writePassfile stores the client credential record with exclusive 0600.
func writePassfile(output, ip, password string) (string, error) {
	passfile := filepath.Join(output, "pgpass")
	record := ip + ":5432:soda_example:developer:" + password + "\n"
	file, err := os.OpenFile(passfile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	_, writeErr := file.WriteString(record)
	closeErr := file.Close()
	if writeErr != nil {
		return "", writeErr
	}
	return passfile, closeErr
}

// fetchAccessURL GETs a URL with no proxy and a 10s timeout.
func fetchAccessURL(target string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		},
	}
	response, err := client.Get(target)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New("unexpected workload HTTP status")
	}
	return io.ReadAll(io.LimitReader(response.Body, 1048577))
}

// provisionMemberCredentials fetches the database password and pushes the
// client credential record to both members.
func (p *accessProbe) provisionMemberCredentials() error {
	password, err := p.accessRemote("alice", "head -c 128 \"$HOME/.config/soda-u08-workload/database-password\"", nil)
	if err != nil {
		return err
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).Match(password) {
		return errors.New("invalid database password shape")
	}
	passfile, err := writePassfile(p.output, p.ip, string(password))
	if err != nil {
		return err
	}
	p.passfile = passfile
	push := "umask 077; mkdir -p \"$HOME/.config\"; mkdir -m700 \"$HOME/.config/u08-db-client-" + p.run + "\"; cat > \"$HOME/.config/u08-db-client-" + p.run + "/pgpass\""
	record := []byte(p.ip + ":5432:soda_example:developer:" + string(password) + "\n")
	for _, who := range []string{"alice", "bob"} {
		if _, err := p.accessRemote(who, push, record); err != nil {
			return err
		}
	}
	return nil
}

// verifyClientWrites commits the run row and reads it back on a fresh connection.
func (p *accessProbe) verifyClientWrites() error {
	if _, err := p.clientPSQL(createProbeTableSQL); err != nil {
		return err
	}
	if _, err := p.clientPSQL(insertProbeSQL(p.run)); err != nil {
		return err
	}
	observed, err := p.clientPSQL(selectProbeSQL(p.run))
	if err != nil {
		return err
	}
	if string(bytes.TrimSpace(observed)) != "client-committed" {
		return errors.New("client-committed value not observed back")
	}
	return nil
}

// verifyMemberReads requires both members to read the committed value.
func (p *accessProbe) verifyMemberReads(expect string) error {
	sql := selectProbeSQL(p.run) + ";\n"
	for _, who := range []string{"alice", "bob"} {
		observed, err := p.accessRemote(who, p.memberPSQLPrefix(), []byte(sql))
		if err != nil {
			return err
		}
		if string(bytes.TrimSpace(observed)) != expect {
			return errors.New("member did not observe the committed value")
		}
	}
	return nil
}

// verifyHTTPBind edits the live bind mount and checks it from three clients.
func (p *accessProbe) verifyHTTPBind() error {
	previous, err := p.accessRemote("alice", "head -c 1048576 \"$HOME/u08-personal-checkout/workload/public/index.html\"", nil)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(p.output, "previous-index.html"), previous, 0o666); err != nil {
		return err
	}
	content := []byte("<!doctype html><title>U08</title><h1>Live bind mount " + p.run + "</h1>\n")
	if _, err := p.accessRemote("alice", "cat > \"$HOME/u08-personal-checkout/workload/public/index.html\"", content); err != nil {
		return err
	}
	observed, err := fetchAccessURL("http://" + p.ip + ":8000/")
	if err != nil {
		return err
	}
	if !bytes.Equal(observed, content) {
		return errors.New("client did not observe the live bind-mount edit")
	}
	curl := "curl --noproxy \"*\" --fail --silent --show-error --max-time 10 http://" + p.ip + ":8000/"
	for _, who := range []string{"alice", "bob"} {
		member, err := p.accessRemote(who, curl, nil)
		if err != nil {
			return err
		}
		if !bytes.Equal(member, content) {
			return errors.New("member did not observe the live bind-mount edit")
		}
	}
	return nil
}

// workloadAccessResults preserves the retired results.json key order.
type workloadAccessResults struct {
	RunID               string `json:"run_id"`
	NetworkMode         string `json:"network_mode"`
	HTTPBindEdit        bool   `json:"http_bind_edit_client_and_members"`
	Postgres            bool   `json:"postgres_client_and_members"`
	CommittedValue      string `json:"committed_value"`
	RestartPersistence  bool   `json:"restart_persistence_verified"`
	DefaultNestedBridge bool   `json:"default_nested_bridge_verified"`
}

// finishAccessResults writes results.json and prints the closing lines.
func (p *accessProbe) finishAccessResults(target accessTarget, stdout io.Writer) error {
	if !target.modeSet {
		return errors.New("workload network mode required")
	}
	results := workloadAccessResults{
		RunID:               p.run,
		NetworkMode:         target.mode,
		HTTPBindEdit:        true,
		Postgres:            true,
		CommittedValue:      "bob-committed",
		RestartPersistence:  false,
		DefaultNestedBridge: target.mode == "bridge",
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(p.output, "results.json"), encoded, 0o666); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Client and Alice/Bob verified live HTTP bind-mount edit and committed PostgreSQL read/write/readback."); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "Network mode: "+target.mode+"; restart/reboot persistence is a separate check.")
	return err
}

// RunWorkloadAccess is the workload-access entrypoint: FIXTURE_DIR.
func RunWorkloadAccess(args []string, stdout io.Writer) error {
	if err := runWorkloadAccess(args, stdout); err != nil {
		return fail("Workload access incomplete; retained state; failure type: ", err)
	}
	return nil
}

func runWorkloadAccess(args []string, stdout io.Writer) error {
	probe, target, err := setupAccessProbe(args)
	if err != nil {
		return err
	}
	if err := runAccessDatabase(probe); err != nil {
		return err
	}
	if err := probe.verifyHTTPBind(); err != nil {
		return err
	}
	return probe.finishAccessResults(target, stdout)
}

// setupAccessProbe validates the invocation and creates the run output.
func setupAccessProbe(args []string) (*accessProbe, accessTarget, error) {
	if os.Getenv("SODA_NATIVE_VALIDATE") != "soda-test" {
		return nil, accessTarget{}, errors.New("explicit soda-test validation required")
	}
	if len(args) != 1 {
		return nil, accessTarget{}, errors.New(WorkloadAccessUsage)
	}
	if err := privateDir(args[0], true); err != nil {
		return nil, accessTarget{}, err
	}
	target, err := loadAccessTarget(args[0])
	if err != nil {
		return nil, accessTarget{}, err
	}
	run, err := uuidHex()
	if err != nil {
		return nil, accessTarget{}, err
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(run) {
		return nil, accessTarget{}, errors.New("invalid run identifier")
	}
	output, err := setupAccessOutput(args[0], run)
	if err != nil {
		return nil, accessTarget{}, err
	}
	probe := &accessProbe{output: output, config: filepath.Join(args[0], "developer-client.conf"), ip: target.ip, run: run}
	return probe, target, nil
}

// runAccessDatabase commits the run row and verifies the member readback.
func runAccessDatabase(probe *accessProbe) error {
	if err := probe.provisionMemberCredentials(); err != nil {
		return err
	}
	if err := probe.verifyClientWrites(); err != nil {
		return err
	}
	if err := probe.verifyMemberReads("client-committed"); err != nil {
		return err
	}
	if _, err := probe.accessRemote("bob", probe.memberPSQLPrefix(), []byte(updateProbeSQL(probe.run))); err != nil {
		return err
	}
	observed, err := probe.clientPSQL(selectProbeSQL(probe.run))
	if err != nil {
		return err
	}
	if string(bytes.TrimSpace(observed)) != "bob-committed" {
		return errors.New("bob-committed value not observed back")
	}
	return nil
}
