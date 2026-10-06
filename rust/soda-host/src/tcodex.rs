//! Supervised factory-Codex executor.
//!
//! Port of `internal/host/terminal/factory_codex.go`,
//! `factory_codex_linux.go`, `factory_export_linux.go`,
//! `factory_output_linux.go`, `factory_takeover_linux.go`, plus the pure
//! `internal/project` factory domain the executor validates against
//! (`factory.go`, `factory_output.go`, `factory_export.go`, `takeover.go`
//! and the role/preparation/digest/commit predicates from
//! `preparation.go`).
//!
//! Methods extend [`Service`](crate::terminal::Service); argv builders are
//! plain `pub` fns. Polling loops bound sleeps by the caller deadline so
//! a cancelled context ends the wait promptly, mirroring Go's `select`
//! on `ctx.Done()`.

use std::time::{Duration, Instant};

use crate::domain;
use crate::json::{self, Kind, Spec};
use crate::project::Executor;
use crate::sha256;
use crate::terminal::{self, Binding, Delivery, Lease, Service, KIND_FACTORY};

// ---------- factory domain (internal/project) ----------

/// Fixed factory execution discriminator for supervised Codex runs.
pub const FACTORY_SCOPE_CODEX: &str = "factory-codex";
/// First harness family.
pub const FACTORY_HARNESS_CODEX: &str = "codex";
/// Fixed factory execution discriminator for supervised Muse Code runs.
pub const FACTORY_SCOPE_MUSE: &str = "factory-muse";
/// Muse Code CLI harness family (`muse exec` runs backed by the native
/// "muse" provider).
pub const FACTORY_HARNESS_MUSE: &str = "muse";

/// Supported supervised CLI families.
pub fn valid_harness_family(family: &str) -> bool {
    family == FACTORY_HARNESS_CODEX || family == FACTORY_HARNESS_MUSE
}
/// Fixed factory role logins.
pub const ROLE_CODER: &str = "soda-coder";
pub const ROLE_REVIEWER: &str = "soda-reviewer";
/// Prompt byte bound (`64*1024`).
pub const MAX_FACTORY_PROMPT: usize = 64 * 1024;
/// One output slice bound (`24*1024-256`).
pub const MAX_FACTORY_OUTPUT_READ: i64 = 24 * 1024 - 256;
/// Trailing attach window (`256*1024`).
pub const MAX_FACTORY_OUTPUT_WINDOW: i64 = 256 * 1024;
/// Output cursor bound (`256<<20`).
pub const MAX_FACTORY_OUTPUT_OFFSET: i64 = 256 << 20;
/// Exported bundle bound (`4<<20`).
pub const MAX_FACTORY_EXPORT_BUNDLE: usize = 4 << 20;
/// Takeover checkout directory inside a member home.
pub const TAKEOVER_DIR_NAME: &str = "factory-takeover";

/// `project.ErrFactoryExportCandidate`.
pub const ERR_FACTORY_EXPORT_CANDIDATE: &str = "export candidate is not recorded";
/// `project.ErrFactoryExportBounds`.
pub const ERR_FACTORY_EXPORT_BOUNDS: &str = "candidate export exceeds bounds";

/// `ValidFactoryRole`: the two fixed factory roles only.
pub fn valid_factory_role(role: &str) -> bool {
    role == ROLE_CODER || role == ROLE_REVIEWER
}

/// `ValidPreparationID = ^f[0-9a-f]{24}$`.
pub fn valid_preparation_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'f' && domain::is_hex_lower(&id[1..])
}

/// `ValidDigest = ^[0-9a-f]{64}$`.
pub fn valid_digest(d: &str) -> bool {
    domain::valid_container_id(d)
}

/// `ValidCommit = ^[0-9a-f]{40}$`.
pub fn valid_commit(c: &str) -> bool {
    c.len() == 40 && domain::is_hex_lower(c)
}

/// `ValidFactoryRunID = ^[a-f0-9]{32}$`.
pub fn valid_factory_run_id(id: &str) -> bool {
    terminal::valid_terminal_id(id)
}

/// `ValidHarnessVersion = ^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`.
pub fn valid_harness_version(version: &str) -> bool {
    let b = version.as_bytes();
    !b.is_empty()
        && b.len() <= 32
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
}

/// One supervised CLI execution (`project.FactoryRun`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryRun {
    pub deadline_raw: String,
    pub actor: i64,
    pub id: String,
    pub project: String,
    pub role: String,
    pub preparation: String,
    pub harness: String,
    pub harness_vers: String,
    pub model: String,
    pub assignment: String,
    pub source_commit: String,
    pub connection: String,
}

const FACTORY_RUN_SPECS: &[Spec] = &[
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "actor",
        kind: Kind::I64,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "preparation",
        kind: Kind::Str,
    },
    Spec {
        name: "harness",
        kind: Kind::Str,
    },
    Spec {
        name: "harness_version",
        kind: Kind::Str,
    },
    Spec {
        name: "model",
        kind: Kind::Str,
    },
    Spec {
        name: "assignment",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "connection",
        kind: Kind::Str,
    },
];

impl FactoryRun {
    /// `FactoryRun.Validate()` with exact error strings.
    pub fn validate(&self) -> Result<(), String> {
        if !valid_factory_run_id(&self.id)
            || !domain::valid_id(&self.project)
            || !valid_factory_role(&self.role)
        {
            return Err("invalid factory run identity".to_string());
        }
        if !valid_preparation_id(&self.preparation) {
            return Err("invalid run preparation reference".to_string());
        }
        if !valid_harness_family(&self.harness) || !valid_harness_version(&self.harness_vers) {
            return Err("unsupported factory harness".to_string());
        }
        if !self.model.is_empty() {
            if self.model.len() > 128 {
                return Err("invalid run model selection".to_string());
            }
            // Go checks raw bytes, not runes.
            for byte in self.model.bytes() {
                if byte < 0x20 || byte == 0x7f {
                    return Err("invalid run model selection".to_string());
                }
            }
        }
        if !valid_digest(&self.assignment) || !valid_commit(&self.source_commit) {
            return Err("invalid run assignment or source identity".to_string());
        }
        if self.connection.is_empty() || self.connection.len() > 128 || self.actor <= 0 {
            return Err("invalid run sponsorship".to_string());
        }
        if self.deadline_raw.is_empty() {
            return Err("run deadline is required".to_string());
        }
        Ok(())
    }

    /// Strict decode of one run object.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m = json::bind_root(&v, "FactoryRun", FACTORY_RUN_SPECS, false).map_err(|e| e.0)?;
        let deadline_raw = m.take_string("deadline");
        if m.contains("deadline") {
            terminal::parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?;
        }
        Ok(FactoryRun {
            deadline_raw,
            actor: m.take_i64("actor"),
            id: m.take_string("id"),
            project: m.take_string("project"),
            role: m.take_string("role"),
            preparation: m.take_string("preparation"),
            harness: m.take_string("harness"),
            harness_vers: m.take_string("harness_version"),
            model: m.take_string("model"),
            assignment: m.take_string("assignment"),
            source_commit: m.take_string("source_commit"),
            connection: m.take_string("connection"),
        })
    }
}

/// `FactoryRunPaths`: fixed container paths for one run; `None` on an
/// invalid identity. Returns `(checkout, run_dir, home, codex_home)`.
pub fn factory_run_paths(
    role: &str,
    preparation: &str,
    run: &str,
) -> Option<(String, String, String, String)> {
    if !valid_factory_role(role) || !valid_preparation_id(preparation) || !valid_factory_run_id(run)
    {
        return None;
    }
    let checkout = format!("/home/{role}/checkouts/{preparation}");
    let run_dir = format!("{checkout}/.soda-home/runs/{run}");
    let home = format!("{run_dir}/home");
    let codex = format!("{home}/.codex");
    Some((checkout, run_dir, home, codex))
}

/// `FactoryCodexGuest`: fixed versioned guest path for staged harness bytes.
pub fn factory_codex_guest(version: &str) -> Option<String> {
    if !valid_harness_version(version) {
        return None;
    }
    Some(format!("/usr/local/bin/codex-factory-{version}"))
}

/// `FactoryUnitName`: deterministic transient host unit per run identity.
pub fn factory_unit_name(run: &str) -> Option<String> {
    if !valid_factory_run_id(run) {
        return None;
    }
    Some(format!("soda-factory-{run}.service"))
}

pub(crate) fn factory_unit_name_or_denied(run: &str) -> Result<String, String> {
    factory_unit_name(run).ok_or_else(terminal::err_denied)
}

/// `TakeoverDestination`: member-owned checkout destination for one run.
pub fn takeover_destination(member: &str, run: &str) -> Option<String> {
    if !domain::valid_login(member) || member == "root" || !valid_factory_run_id(run) {
        return None;
    }
    Some(format!("/home/{member}/{TAKEOVER_DIR_NAME}/{run}"))
}

/// `TakeoverSource`: the exact fixed role-checkout layout, nothing else.
pub fn takeover_source(path: &str, role: &str, preparation: &str) -> bool {
    if !valid_factory_role(role) || !valid_preparation_id(preparation) {
        return false;
    }
    path == format!("/home/{role}/checkouts/{preparation}")
}

// ---------- run paths, scripts, bindings ----------

/// Fixed container paths for one supervised Codex run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCodexPaths {
    pub checkout: String,
    pub run_dir: String,
    pub home: String,
    pub codex: String,
    pub prompt: String,
    pub marker: String,
    pub started: String,
    pub stop: String,
    pub pid_file: String,
    pub output: String,
    pub stdout: String,
    pub auth: String,
    pub guest: String,
}

/// `factoryCodexPaths`: validated run identities to fixed paths.
pub fn factory_codex_paths(run: &FactoryRun) -> Result<FactoryCodexPaths, String> {
    run.validate()?;
    let (checkout, run_dir, home, codex) =
        factory_run_paths(&run.role, &run.preparation, &run.id).ok_or_else(terminal::err_denied)?;
    let guest = factory_codex_guest(&run.harness_vers).ok_or_else(terminal::err_denied)?;
    Ok(FactoryCodexPaths {
        checkout,
        run_dir: run_dir.clone(),
        home,
        codex: codex.clone(),
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{codex}/auth.json"),
        guest,
    })
}

/// One observed byte slice of recorded CLI output with its cursor.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCodexOutputSlice {
    pub data: Vec<u8>,
    pub total: i64,
    pub offset: i64,
    pub truncated: bool,
    pub gap: bool,
}

/// `systemdEscape`: double every dollar for transport through systemd-run.
pub fn systemd_escape(script: &str) -> String {
    script.replace('$', "$$")
}

/// Single-quote one shell word.
pub fn shell_quote(value: &str) -> String {
    format!("'{}'", value.replace('\'', "'\\''"))
}

/// `factorySupervisor`: marker-gated fixed Codex entrypoint. Golden-pinned
/// against the Go output byte for byte.
pub fn factory_supervisor(p: &FactoryCodexPaths, guest: &str, model: &str) -> String {
    let mut command = format!(
        "{} exec --color never --sandbox danger-full-access --skip-git-repo-check --config {}",
        shell_quote(guest),
        shell_quote("model_reasoning_effort=\"low\"")
    );
    if !model.is_empty() {
        command.push_str(&format!(" --model {}", shell_quote(model)));
    }
    command.push_str(&format!(
        " --output-last-message {} - <{}",
        shell_quote(&p.output),
        shell_quote(&p.prompt)
    ));
    let steps = [
        format!("RUNDIR={}", shell_quote(&p.run_dir)),
        "exec >\"$RUNDIR/stdout.log\" 2>&1".to_string(),
        "STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo \"$$ ${20}\" >\"$RUNDIR/supervisor.pid\"".to_string(),
        "fail() { echo \"$1\" >\"$RUNDIR/exit\"; exit \"$1\"; }".to_string(),
        "i=0; while [ ! -f \"$RUNDIR/marker\" ]; do [ -f \"$RUNDIR/stop\" ] && fail 44; i=$((i+1)); [ \"$i\" -gt 600 ] && fail 42; sleep 1; done".to_string(),
        "mv \"$RUNDIR/marker\" \"$RUNDIR/started\" || fail 43".to_string(),
        command,
        "CODE=$?; echo \"$CODE\" >\"$RUNDIR/exit\"; exit \"$CODE\"".to_string(),
    ];
    steps.join("\n") + "\n"
}

/// `factoryRetire`: identity-verified supervisor process-group kill plus a
/// lingering-member scan. Golden-pinned against the Go output.
pub fn factory_retire(p: &FactoryCodexPaths) -> String {
    let script = [
        format!("RUNDIR={}", shell_quote(&p.run_dir)),
        ": >\"$RUNDIR/stop\"".to_string(),
        "[ -f \"$RUNDIR/supervisor.pid\" ] || exit 0".to_string(),
        "read PID START <\"$RUNDIR/supervisor.pid\"".to_string(),
        "case \"$PID\" in ''|*[!0-9]*) exit 0;; esac".to_string(),
        "case \"$START\" in ''|*[!0-9]*) exit 0;; esac".to_string(),
        "if [ -d \"/proc/$PID\" ]; then".to_string(),
        "  if STAT=$(cat \"/proc/$PID/stat\" 2>/dev/null); then".to_string(),
        "    REST=${STAT##*)}; set -- $REST".to_string(),
        "    if [ \"${20}\" = \"$START\" ]; then".to_string(),
        "      CMDLINE=$(tr '\\000' ' ' <\"/proc/$PID/cmdline\" 2>/dev/null) || CMDLINE=\"\"".to_string(),
        "      case \"$CMDLINE\" in *\"$RUNDIR\"*)".to_string(),
        "        if [ \"$3\" = \"$PID\" ]; then kill -KILL -- \"-$PID\" 2>/dev/null || true; else kill -KILL -- \"$PID\" 2>/dev/null || true; fi".to_string(),
        "      ;; esac".to_string(),
        "    fi".to_string(),
        "  fi".to_string(),
        "fi".to_string(),
        "sleep 1".to_string(),
        "for S in /proc/[0-9]*/stat; do".to_string(),
        "  STAT=$(cat \"$S\" 2>/dev/null) || continue".to_string(),
        "  REST=${STAT##*)}; set -- $REST".to_string(),
        "  if [ \"$3\" = \"$PID\" ]; then echo \"lingering: $S\"; exit 1; fi".to_string(),
        "done".to_string(),
        "exit 0".to_string(),
    ];
    script.join("\n") + "\n"
}

/// `factoryExportScript`: objects-only candidate export through a clean
/// bare repository. Golden-pinned against the Go output.
pub const FACTORY_EXPORT_SCRIPT: &str = r#"set -eu
src=$1
candidate=$2
limit=$3
if ! /usr/bin/test -d "$src/.git/objects"; then
  printf 'soda-export-missing\n'
  exit 0
fi
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-export-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_export() {
  /usr/bin/git -c core.hooksPath=/dev/null --git-dir="$dir/repo.git" "$@"
}
if ! actual=$(git_export rev-parse --verify "$candidate^{commit}" 2>/dev/null); then
  printf 'soda-export-invalid\n'
  exit 0
fi
if [ "$actual" != "$candidate" ]; then
  printf 'soda-export-invalid\n'
  exit 0
fi
git_export update-ref HEAD "$candidate" 2>/dev/null
git_export bundle create "$dir/candidate.bundle" HEAD 2>/dev/null
/usr/bin/head -c "$limit" "$dir/candidate.bundle"
"#;

/// `factoryCodexBinding`: supervised factory-Codex binding check plus
/// derived run paths. The recorded credential root must equal the derived
/// run directory. Family policy (login, generation) wraps the shared
/// `tfactory` checks.
pub fn factory_codex_binding(lease: &Lease) -> Result<FactoryCodexPaths, String> {
    let Some(b) = &lease.binding else {
        return Err(terminal::err_denied());
    };
    if !domain::valid_login(&b.login) || b.login == "root" {
        return Err(terminal::err_denied());
    }
    if b.uid <= 0 || b.gid <= 0 || b.generation != lease.generation || b.generation <= 0 {
        return Err(terminal::err_denied());
    }
    let (checkout, run_dir, home, codex) = crate::tfactory::checked_binding_paths(
        lease,
        terminal::PROVIDER_CODEX,
        FACTORY_SCOPE_CODEX,
        factory_run_paths,
    )?;
    Ok(FactoryCodexPaths {
        checkout,
        run_dir: run_dir.clone(),
        home,
        codex: codex.clone(),
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{codex}/auth.json"),
        guest: String::new(),
    })
}

// ---------- argv builders ----------

/// `factoryUserBus` for an explicit euid (production passes `geteuid`).
pub fn factory_user_bus(euid: u32) -> String {
    format!("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/{euid}/bus")
}

fn current_euid() -> u32 {
    unsafe { libc::geteuid() }
}

/// Full `env ... systemctl --user` argv.
pub fn factory_systemctl_argv(euid: u32, args: &[&str]) -> Vec<String> {
    let mut argv = vec![
        factory_user_bus(euid),
        "/usr/bin/systemctl".to_string(),
        "--user".to_string(),
    ];
    argv.extend(args.iter().map(|s| s.to_string()));
    argv
}

/// Full `env ... systemd-run --user` argv.
pub fn factory_systemd_run_argv(euid: u32, args: &[&str]) -> Vec<String> {
    let mut argv = vec![
        factory_user_bus(euid),
        "/usr/bin/systemd-run".to_string(),
        "--user".to_string(),
    ];
    argv.extend(args.iter().map(|s| s.to_string()));
    argv
}

/// `FactoryCodexReserve` podman-supervisor argv (the unit's exec payload).
pub fn reserve_exec_argv(
    container: &str,
    run: &FactoryRun,
    p: &FactoryCodexPaths,
    guest: &str,
) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--user".to_string(),
        run.role.clone(),
        "--workdir".to_string(),
        p.checkout.clone(),
        "--env".to_string(),
        format!("HOME={}", p.home),
        "--env".to_string(),
        format!("CODEX_HOME={}", p.codex),
        "--env".to_string(),
        "TERM=dumb".to_string(),
        "--env".to_string(),
        format!("GIT_AUTHOR_NAME={}", run.role),
        "--env".to_string(),
        format!("GIT_AUTHOR_EMAIL={}@localhost", run.role),
        "--env".to_string(),
        format!("GIT_COMMITTER_NAME={}", run.role),
        "--env".to_string(),
        format!("GIT_COMMITTER_EMAIL={}@localhost", run.role),
        container.to_string(),
        "/usr/bin/setsid".to_string(),
        "--wait".to_string(),
        "/usr/bin/sh".to_string(),
        "-c".to_string(),
        systemd_escape(&factory_supervisor(p, guest, &run.model)),
    ]
}

/// `FactoryCodexReserve` systemd-run header argv.
pub fn reserve_run_argv(unit: &str, max_secs: i64) -> Vec<String> {
    vec![
        format!("--unit={unit}"),
        "--collect".to_string(),
        "--property=KillMode=control-group".to_string(),
        format!("--property=RuntimeMaxSec={max_secs}"),
        "--property=TimeoutStopSec=10".to_string(),
        "--".to_string(),
        "/usr/bin/podman".to_string(),
    ]
}

/// `factoryCodexSetup` directory-preparation script.
pub fn codex_setup_script(p: &FactoryCodexPaths, uid: i64, gid: i64) -> String {
    format!(
        "set -u\nmkdir -p -m 700 {} {}\nchown {uid}:{gid} {} {} {}\nchmod 700 {} {} {}\n",
        shell_quote(&p.home),
        shell_quote(&p.codex),
        shell_quote(&p.run_dir),
        shell_quote(&p.home),
        shell_quote(&p.codex),
        shell_quote(&p.run_dir),
        shell_quote(&p.home),
        shell_quote(&p.codex),
    )
}

/// Harness install script after `podman cp`.
pub fn harness_install_script(guest: &str) -> String {
    format!(
        "mv {} {} && chmod 755 {} && /usr/bin/sha256sum {}\n",
        shell_quote(&format!("{guest}.new")),
        shell_quote(guest),
        shell_quote(guest),
        shell_quote(guest),
    )
}

/// `factoryStageFile` install command.
pub fn stage_file_command(path: &str, uid: i64, gid: i64) -> String {
    format!(
        "/usr/bin/install -m 600 /dev/stdin {} && chown {uid}:{gid} {}\n",
        shell_quote(path),
        shell_quote(path),
    )
}

/// `FactoryCodexStart` staging-gate script.
pub fn start_gate_script(p: &FactoryCodexPaths) -> String {
    format!(
        "test -s {} && test -s {} && {{ test -f {} || test -f {}; }}\n",
        shell_quote(&p.auth),
        shell_quote(&p.prompt),
        shell_quote(&p.marker),
        shell_quote(&p.started),
    )
}

/// `FactoryCodexOutput` bounded read pipeline.
pub fn output_read_command(stdout: &str, start: i64, limit: i64) -> String {
    format!(
        "/usr/bin/tail -c +{} {} | /usr/bin/head -c {limit}\n",
        start.wrapping_add(1),
        shell_quote(stdout),
    )
}

/// `FactoryTakeoverCopy` fixed copy steps.
pub fn takeover_steps(src: &str, dest: &str, member: &str) -> Vec<Vec<String>> {
    let partial = format!("{dest}.partial");
    let parent = format!("/home/{member}/{TAKEOVER_DIR_NAME}");
    vec![
        vec!["/usr/bin/mkdir".to_string(), "-p".to_string(), parent],
        vec![
            "/usr/bin/rm".to_string(),
            "-rf".to_string(),
            partial.clone(),
        ],
        vec!["/usr/bin/mkdir".to_string(), partial.clone()],
        vec![
            "/usr/bin/cp".to_string(),
            "-a".to_string(),
            format!("{src}/."),
            format!("{partial}/"),
        ],
        vec![
            "/usr/bin/rm".to_string(),
            "-rf".to_string(),
            format!("{partial}/.git"),
            format!("{partial}/.soda-home"),
        ],
        vec![
            "/usr/bin/git".to_string(),
            "-C".to_string(),
            partial.clone(),
            "init".to_string(),
            "-q".to_string(),
        ],
        vec![
            "/usr/bin/chown".to_string(),
            "-R".to_string(),
            format!("--reference=/home/{member}"),
            partial,
        ],
    ]
}

/// `FactoryExportBundle` guest env (fixed, clean) plus argv tail.
pub fn export_argv(current: &str, role: &str, src: &str, candidate: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--user".to_string(),
        role.to_string(),
        current.to_string(),
        "/usr/bin/env".to_string(),
        "-i".to_string(),
        "PATH=/usr/bin:/bin".to_string(),
        format!("HOME=/home/{role}"),
        "LC_ALL=C".to_string(),
        format!("TMPDIR=/home/{role}/checkouts"),
        "GIT_CONFIG_NOSYSTEM=1".to_string(),
        "GIT_CONFIG_GLOBAL=/dev/null".to_string(),
        "GIT_NO_REPLACE_OBJECTS=1".to_string(),
        "GIT_TERMINAL_PROMPT=0".to_string(),
        "/usr/bin/sh".to_string(),
        "-c".to_string(),
        FACTORY_EXPORT_SCRIPT.to_string(),
        "soda-export".to_string(),
        src.to_string(),
        candidate.to_string(),
        (MAX_FACTORY_EXPORT_BUNDLE + 1).to_string(),
    ]
}

// ---------- executor ----------

/// Observed transient-unit state (`factoryUnitShow`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryUnitShow {
    pub active: bool,
    pub invocation: String,
}

/// `parseFactoryUnitShow`: last `ActiveState=`/`InvocationID=` wins.
pub fn parse_factory_unit_show(body: &[u8]) -> FactoryUnitShow {
    let mut out = FactoryUnitShow::default();
    for line in String::from_utf8_lossy(body).trim().split('\n') {
        if let Some(value) = line.strip_prefix("ActiveState=") {
            out.active = value == "active";
        }
        if let Some(value) = line.strip_prefix("InvocationID=") {
            out.invocation = value.trim().to_string();
        }
    }
    out
}

/// `factoryRoleID`: positive role identity or `invalid role identity`.
pub fn factory_role_id(out: &[u8]) -> Result<i64, String> {
    match terminal::parse_go_int(String::from_utf8_lossy(out).trim()) {
        Some(id) if id > 0 => Ok(id),
        _ => Err("invalid role identity".to_string()),
    }
}

/// Sleep until `target` in slices, failing if `deadline` passes first.
/// Mirrors Go's `select` on `ctx.Done()` vs `time.After`.
pub(crate) fn sleep_until(target: Instant, deadline: Instant) -> Result<(), ()> {
    loop {
        let now = Instant::now();
        if now >= deadline {
            return Err(());
        }
        if now >= target {
            return Ok(());
        }
        let slice = (target - now)
            .min(deadline - now)
            .min(Duration::from_millis(10));
        std::thread::sleep(slice);
    }
}

impl<E: Executor> Service<E> {
    fn run_env(&self, argv: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = argv.iter().map(|s| s.as_str()).collect();
        self.exec.run(&[], "/usr/bin/env", &refs, deadline)
    }

    pub(crate) fn run_podman(
        &self,
        stdin: &[u8],
        argv: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = argv.iter().map(|s| s.as_str()).collect();
        self.exec.run(stdin, "/usr/bin/podman", &refs, deadline)
    }

    /// `Service.factorySystemctl`.
    pub fn factory_systemctl(&self, args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
        self.run_env(&factory_systemctl_argv(current_euid(), args), deadline)
    }

    /// `Service.factorySystemdRun`.
    pub fn factory_systemd_run(
        &self,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        self.run_env(&factory_systemd_run_argv(current_euid(), &refs), deadline)
    }

    /// `Service.factoryUnitState`.
    pub fn factory_unit_state(
        &self,
        unit: &str,
        deadline: Instant,
    ) -> Result<FactoryUnitShow, String> {
        match self.factory_systemctl(
            &["show", "--property=ActiveState,InvocationID", unit],
            deadline,
        ) {
            Ok(body) if body.len() <= 4096 => Ok(parse_factory_unit_show(&body)),
            _ => Err("factory unit observation unavailable".to_string()),
        }
    }

    /// `Service.factoryActiveInvocation`: active unit with a well-formed
    /// invocation within `wait`, else stale.
    pub fn factory_active_invocation(
        &self,
        unit: &str,
        wait: Duration,
        deadline: Instant,
    ) -> Result<String, String> {
        let end = Instant::now() + wait;
        loop {
            if let Ok(show) = self.factory_unit_state(unit, deadline) {
                if show.active && terminal::valid_terminal_id(&show.invocation) {
                    return Ok(show.invocation);
                }
            }
            let now = Instant::now();
            if now >= end || now >= deadline {
                return Err(terminal::err_stale());
            }
            if sleep_until(now + Duration::from_millis(100), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
    }

    /// `Service.factoryRoleIDs`: guest uid/gid for a factory role.
    pub fn factory_role_ids(
        &self,
        container: &str,
        role: &str,
        deadline: Instant,
    ) -> Result<(i64, i64), String> {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/id".to_string(),
            "-u".to_string(),
            role.to_string(),
        ];
        let uid_out = match self.run_podman(&[], &argv, deadline) {
            Ok(out) if out.len() <= 64 => out,
            _ => return Err("factory role is not resolvable".to_string()),
        };
        let uid = factory_role_id(&uid_out)?;
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/id".to_string(),
            "-g".to_string(),
            role.to_string(),
        ];
        let gid_out = match self.run_podman(&[], &argv, deadline) {
            Ok(out) if out.len() <= 64 => out,
            _ => return Err("factory role is not resolvable".to_string()),
        };
        let gid = factory_role_id(&gid_out)?;
        Ok((uid, gid))
    }

    /// `Service.FactoryCodexReserve`: prepare run directories, stage the
    /// verified harness, start the waiting supervisor unit. No credentials.
    pub fn factory_codex_reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin_sha256: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<(Binding, FactoryCodexPaths), String> {
        let p = factory_codex_paths(run)?;
        if lease.kind != KIND_FACTORY || lease.execution_id != run.id || lease.generation <= 0 {
            return Err(terminal::err_denied());
        }
        if pin_sha256.is_empty()
            || pin_sha256 != self.codex_harness_sha256
            || !self.codex_harness.starts_with('/')
        {
            return Err(terminal::err_denied());
        }
        if !(60..=3 * 3600).contains(&max_secs) {
            return Err(terminal::err_denied());
        }
        self.verify_identity_harness()?;
        let container = self.factory_project_container(&run.project, true, deadline)?;
        let (uid, gid) = self.factory_role_ids(&container, &run.role, deadline)?;
        self.factory_codex_setup(&container, run, &p, uid, gid, deadline)?;
        let guest = self.factory_codex_stage(&container, &run.harness_vers, deadline)?;
        let unit = factory_unit_name_or_denied(&run.id)?;
        let mut args = reserve_run_argv(&unit, max_secs);
        args.extend(reserve_exec_argv(&container, run, &p, &guest));
        if self.factory_systemd_run(&args, deadline).is_err() {
            return Err("factory unit start unconfirmed".to_string());
        }
        match self.factory_active_invocation(&unit, Duration::from_secs(10), deadline) {
            Ok(invocation) => Ok((
                Binding {
                    kind: KIND_FACTORY.to_string(),
                    id: run.id.clone(),
                    project: container,
                    login: run.role.clone(),
                    uid,
                    gid,
                    scope: FACTORY_SCOPE_CODEX.to_string(),
                    invocation_id: invocation,
                    credential_root: p.run_dir.clone(),
                    generation: lease.generation,
                    child_id: run.preparation.clone(),
                },
                p,
            )),
            Err(err) => {
                let _ = self.factory_systemctl(&["stop", &unit], deadline);
                Err(err)
            }
        }
    }

    fn factory_codex_setup(
        &self,
        container: &str,
        run: &FactoryRun,
        p: &FactoryCodexPaths,
        uid: i64,
        gid: i64,
        deadline: Instant,
    ) -> Result<(), String> {
        let setup = codex_setup_script(p, uid, gid);
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            setup,
        ];
        if self.run_podman(&[], &argv, deadline).is_err() {
            return Err("factory run directories unconfirmed".to_string());
        }
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            "--user".to_string(),
            run.role.clone(),
            container.to_string(),
            "/usr/bin/git".to_string(),
            "-C".to_string(),
            p.checkout.clone(),
            "rev-parse".to_string(),
            "HEAD".to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(head) if head.len() <= 1024 => {
                if String::from_utf8_lossy(&head).trim() != run.source_commit {
                    return Err("factory checkout is not the assigned commit".to_string());
                }
            }
            _ => return Err("factory checkout identity unconfirmed".to_string()),
        }
        Ok(())
    }

    fn factory_codex_stage(
        &self,
        container: &str,
        version: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let guest = factory_codex_guest(version).ok_or_else(terminal::err_denied)?;
        let probe = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sha256sum".to_string(),
            guest.clone(),
        ];
        match self.run_podman(&[], &probe, deadline) {
            Err(_) => {
                let host_bin = terminal::clean_path(&format!("{}/bin/codex", self.codex_harness));
                let cp = vec![
                    "--remote=false".to_string(),
                    "cp".to_string(),
                    host_bin,
                    format!("{container}:{guest}.new"),
                ];
                if self.run_podman(&[], &cp, deadline).is_err() {
                    return Err("factory harness staging unconfirmed".to_string());
                }
                let install = vec![
                    "--remote=false".to_string(),
                    "exec".to_string(),
                    container.to_string(),
                    "/usr/bin/sh".to_string(),
                    "-c".to_string(),
                    harness_install_script(&guest),
                ];
                match self.run_podman(&[], &install, deadline) {
                    Ok(out) => {
                        let text = String::from_utf8_lossy(&out);
                        let fields: Vec<&str> = text.split_whitespace().collect();
                        if fields.len() != 2 || fields[0] != self.codex_harness_sha256 {
                            return Err("guest harness digest differs".to_string());
                        }
                    }
                    Err(_) => return Err("factory harness install unconfirmed".to_string()),
                }
            }
            Ok(out) => {
                let text = String::from_utf8_lossy(&out);
                let fields: Vec<&str> = text.split_whitespace().collect();
                if fields.len() != 2 || fields[0] != self.codex_harness_sha256 {
                    return Err("guest harness digest differs".to_string());
                }
            }
        }
        self.factory_codex_stage_host(container, deadline)?;
        Ok(guest)
    }

    fn factory_codex_stage_host(&self, container: &str, deadline: Instant) -> Result<(), String> {
        const HOST_GUEST: &str = "/usr/local/bin/codex-code-mode-host";
        let host_bin =
            terminal::clean_path(&format!("{}/bin/codex-code-mode-host", self.codex_harness));
        let want =
            std::fs::read(&host_bin).map_err(|_| "factory code host is not staged".to_string())?;
        let digest = sha256::hex_lower(&sha256::digest(&want));
        let probe = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sha256sum".to_string(),
            HOST_GUEST.to_string(),
        ];
        if let Ok(out) = self.run_podman(&[], &probe, deadline) {
            let text = String::from_utf8_lossy(&out);
            let fields: Vec<&str> = text.split_whitespace().collect();
            if fields.len() == 2 && fields[0] == digest {
                return Ok(());
            }
        }
        let cp = vec![
            "--remote=false".to_string(),
            "cp".to_string(),
            host_bin,
            format!("{container}:{HOST_GUEST}.new"),
        ];
        if self.run_podman(&[], &cp, deadline).is_err() {
            return Err("factory code host staging unconfirmed".to_string());
        }
        let install = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            harness_install_script(HOST_GUEST),
        ];
        match self.run_podman(&[], &install, deadline) {
            Ok(out) => {
                let text = String::from_utf8_lossy(&out);
                let fields: Vec<&str> = text.split_whitespace().collect();
                if fields.len() != 2 || fields[0] != digest {
                    return Err("guest code host digest differs".to_string());
                }
            }
            Err(_) => return Err("factory code host install unconfirmed".to_string()),
        }
        Ok(())
    }

    /// `Service.FactoryCodexStart`: stage credential, prompt, then the
    /// start marker the supervisor gates on. Marker order is load-bearing.
    pub fn factory_codex_start(
        &self,
        lease: &Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let p = factory_codex_binding(lease)?;
        if !terminal::credential_valid(credential) {
            return Err(terminal::err_denied());
        }
        if prompt.is_empty() || prompt.len() > MAX_FACTORY_PROMPT {
            return Err(terminal::err_denied());
        }
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        let container = self.factory_project_container(&lease.project_id, true, deadline)?;
        if container != binding.project {
            return Err(terminal::err_stale());
        }
        self.factory_stage_file(&container, binding, &p.auth, credential, deadline)?;
        self.factory_stage_file(&container, binding, &p.prompt, prompt, deadline)?;
        self.factory_stage_file(&container, binding, &p.marker, &[], deadline)?;
        let gate = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container,
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            start_gate_script(&p),
        ];
        match self.run_podman(&[], &gate, deadline) {
            Ok(out) if out.len() <= 1024 => Ok(()),
            _ => Err("factory start staging unconfirmed".to_string()),
        }
    }

    pub(crate) fn factory_stage_file(
        &self,
        container: &str,
        binding: &Binding,
        path: &str,
        data: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            "--interactive".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            stage_file_command(path, binding.uid, binding.gid),
        ];
        self.run_podman(data, &argv, deadline)
            .map(|_| ())
            .map_err(|_| "factory file staging unconfirmed".to_string())
    }

    /// `Service.FactoryCodexWait`: block until the unit leaves active
    /// state, then read the recorded exit and bounded output.
    pub fn factory_codex_wait(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(i32, String), String> {
        let p = factory_codex_binding(lease)?;
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        self.factory_wait_result(
            &lease.execution_id,
            &project,
            &p.run_dir,
            &p.output,
            deadline,
        )
    }

    /// `Service.FactoryCodexValidate`: attest the live supervised boundary
    /// before the broker releases credential bytes.
    pub fn factory_codex_validate(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        factory_codex_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        self.factory_attest_live(&lease.project_id, binding, &lease.execution_id, deadline)
    }

    /// `Service.FactoryCodexStop`: retire the recorded unit and container
    /// process group. Idempotent.
    pub fn factory_codex_stop(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        let p = factory_codex_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        self.factory_stop_confirmed(
            &binding.project,
            &lease.execution_id,
            &p.pid_file,
            &p.run_dir,
            deadline,
        )
    }

    pub(crate) fn factory_await_inactive(
        &self,
        unit: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(15);
        loop {
            if let Ok(show) = self.factory_unit_state(unit, deadline) {
                if !show.active {
                    return Ok(());
                }
            }
            let now = Instant::now();
            if now >= end || now >= deadline {
                return Err(terminal::err_uncertain());
            }
            if sleep_until(now + Duration::from_millis(250), deadline).is_err() {
                return Err(terminal::err_uncertain());
            }
        }
    }

    pub(crate) fn factory_retire(
        &self,
        container: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        // The retire script only needs the run directory; build the shim
        // paths so the golden-pinned builder stays untouched.
        let p = FactoryCodexPaths {
            run_dir: run_dir.to_string(),
            ..Default::default()
        };
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            factory_retire(&p),
        ];
        self.run_podman(&[], &argv, deadline)
            .map(|_| ())
            .map_err(|_| terminal::err_uncertain())
    }

    pub(crate) fn factory_read_pid(
        &self,
        container: &str,
        pid_file: &str,
        deadline: Instant,
    ) -> String {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/cat".to_string(),
            pid_file.to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(out) if out.len() <= 256 && out.contains(&b' ') => {
                String::from_utf8_lossy(&out).trim().to_string()
            }
            _ => String::new(),
        }
    }

    pub(crate) fn factory_container_exists(
        &self,
        container: &str,
        deadline: Instant,
    ) -> Result<bool, String> {
        match self.run_podman(&[], &terminal::container_exists_argv(container), deadline) {
            Ok(_) => Ok(true),
            Err(err) if terminal::exit_code_of(&err) == Some(1) => Ok(false),
            Err(_) => Err(terminal::err_uncertain()),
        }
    }

    /// `Service.FactoryCodexCapture`: read back the maintained credential
    /// after confirmed retirement.
    pub fn factory_codex_capture(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let p = factory_codex_binding(lease)?;
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        self.factory_capture_valid(&project, &p.auth, deadline)
    }

    /// `Service.FactoryCodexFinish`: stop, then capture.
    pub fn factory_codex_finish(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.factory_codex_stop(lease, deadline)?;
        self.factory_codex_capture(lease, deadline)
    }

    /// `Service.FactoryCodexStopUnbound`: retire a run whose binding was
    /// never recorded.
    pub fn factory_codex_stop_unbound(
        &self,
        run: &FactoryRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let p = factory_codex_paths(run)?;
        self.factory_stop_unbound_confirmed(
            &run.project,
            &run.id,
            &p.pid_file,
            &p.run_dir,
            deadline,
        )
    }

    /// `Service.FactoryCodexLive`: recorded unit currently active with the
    /// recorded invocation. Observation only.
    pub fn factory_codex_live(&self, binding: &Binding, deadline: Instant) -> bool {
        self.factory_live_scoped(binding, FACTORY_SCOPE_CODEX, deadline)
    }

    /// `Service.FactoryCodexOutput`: one bounded slice at a byte cursor.
    pub fn factory_codex_output(
        &self,
        project_id: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<FactoryCodexOutputSlice, String> {
        crate::tfactory::check_output_range(offset, limit)?;
        let stdout = self.factory_output_stdout(
            project_id,
            binding,
            FACTORY_SCOPE_CODEX,
            factory_run_paths,
            deadline,
        )?;
        self.factory_output_window(&binding.project, &stdout, offset, limit, deadline)
    }

    /// `Service.FactoryExportBundle`: exact-candidate bundle export.
    pub fn factory_export_bundle(
        &self,
        project_id: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let src = format!("/home/{role}/checkouts/{preparation}");
        if !takeover_source(&src, role, preparation)
            || !valid_commit(candidate)
            || !domain::valid_container_id(recorded)
        {
            return Err(terminal::err_denied());
        }
        let current = self.factory_project_container(project_id, true, deadline)?;
        if current != recorded {
            return Err(terminal::err_stale());
        }
        let argv = export_argv(&current, role, &src, candidate);
        let bundle = match self.run_podman(&[], &argv, deadline) {
            Ok(bundle) => bundle,
            Err(_) => {
                if Instant::now() >= deadline {
                    return Err("context deadline exceeded".to_string());
                }
                return Err("candidate export execution unconfirmed".to_string());
            }
        };
        if bundle == b"soda-export-missing\n" {
            return Err(terminal::ERR_NOT_FOUND.to_string());
        }
        if bundle == b"soda-export-invalid\n" {
            return Err(ERR_FACTORY_EXPORT_CANDIDATE.to_string());
        }
        if bundle.is_empty() || bundle.len() > MAX_FACTORY_EXPORT_BUNDLE {
            return Err(ERR_FACTORY_EXPORT_BOUNDS.to_string());
        }
        Ok(bundle)
    }

    /// `Service.FactoryTakeoverCopy`: copy retained role-checkout work
    /// into the admitted member's derived destination.
    #[allow(clippy::too_many_arguments)] // one parameter per Go argument, in order
    pub fn factory_takeover_copy(
        &self,
        project_id: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        let dest = takeover_destination(member, run).ok_or_else(terminal::err_denied)?;
        let src = format!("/home/{role}/checkouts/{preparation}");
        if !takeover_source(&src, role, preparation) || !domain::valid_container_id(recorded) {
            return Err(terminal::err_denied());
        }
        let current = self.factory_project_container(project_id, true, deadline)?;
        if current != recorded {
            return Err(terminal::err_stale());
        }
        let probe = |args: &[String]| self.run_podman(&[], args, deadline);
        let test_dir = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-d".to_string(),
            dest.clone(),
        ];
        if probe(&test_dir).is_ok() {
            return Ok((dest, true));
        }
        let partial = format!("{dest}.partial");
        let test_src = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-d".to_string(),
            src.clone(),
        ];
        if probe(&test_src).is_err() {
            return Err("takeover found no retained work".to_string());
        }
        for step in takeover_steps(&src, &dest, member) {
            let mut argv = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current.clone(),
            ];
            argv.extend(step);
            match probe(&argv) {
                Ok(out) if out.len() <= 65536 => {}
                Ok(_) => return Err("takeover step returned excessive output".to_string()),
                Err(err) => return Err(err),
            }
        }
        let test_dest = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-e".to_string(),
            dest.clone(),
        ];
        if probe(&test_dest).is_ok() {
            let cleanup = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current,
                "/usr/bin/rm".to_string(),
                "-rf".to_string(),
                partial,
            ];
            let _ = probe(&cleanup);
            return Ok((dest, true));
        }
        let mv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/mv".to_string(),
            "-T".to_string(),
            partial.clone(),
            dest.clone(),
        ];
        if probe(&mv).is_err() {
            let cleanup = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current,
                "/usr/bin/rm".to_string(),
                "-rf".to_string(),
                partial,
            ];
            let _ = probe(&cleanup);
            return Err("takeover destination was not confirmed".to_string());
        }
        Ok((dest, false))
    }

    /// `Daemon.factoryIdentityOperation`: broker validate/stop/finish
    /// callbacks for supervised factory runs.
    pub fn factory_identity_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let mut out = delivery.clone();
        out.credential = None;
        let muse = delivery
            .lease
            .binding
            .as_ref()
            .is_some_and(|b| b.scope == FACTORY_SCOPE_MUSE);
        match action {
            "validate" if muse => self.factory_muse_validate(&delivery.lease, deadline)?,
            "validate" => self.factory_codex_validate(&delivery.lease, deadline)?,
            "stop" if muse => self.factory_muse_stop(&delivery.lease, deadline)?,
            "stop" => self.factory_codex_stop(&delivery.lease, deadline)?,
            // Muse borrows: the broker forgets the lease on return and
            // never calls finish (same denial as the interactive muse
            // runtime, which allows only validate and stop).
            "finish" if muse => return Err(terminal::err_denied()),
            "finish" => {
                out.credential = Some(self.factory_codex_finish(&delivery.lease, deadline)?);
            }
            _ => return Err(terminal::err_denied()),
        }
        Ok(out)
    }
}

/// `factoryOutputSize`: non-negative stat size from a short read.
pub fn factory_output_size(out: &[u8]) -> Option<i64> {
    let size = terminal::parse_go_int(String::from_utf8_lossy(out).trim())?;
    if size >= 0 && out.len() <= 64 {
        Some(size)
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::terminal::{ERR_DENIED, ERR_STALE, ERR_UNCERTAIN};
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::sync::Mutex;

    const PID: &str = "p0123456789abcdef01234567";
    const RID: &str = "0123456789abcdef0123456789abcdef";
    const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    const PREP: &str = "f0123456789abcdef01234567";
    const ROLE: &str = "soda-coder";
    const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
    const PIN: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

    static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    fn test_tmp(slug: &str) -> std::path::PathBuf {
        let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
        let dir = std::env::temp_dir().join(format!("t26c-{}-{n}-{slug}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    type RecordedCall = (Vec<u8>, String, Vec<String>);

    struct FakeExec {
        calls: Mutex<Vec<RecordedCall>>,
        script: Mutex<Vec<Result<Vec<u8>, String>>>,
    }

    impl FakeExec {
        fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
            FakeExec {
                calls: Mutex::new(Vec::new()),
                script: Mutex::new(script),
            }
        }

        fn calls(&self) -> Vec<RecordedCall> {
            self.calls.lock().unwrap().clone()
        }
    }

    impl Executor for FakeExec {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.lock().unwrap().push((
                stdin.to_vec(),
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            let mut script = self.script.lock().unwrap();
            if script.is_empty() {
                return Err("unexpected call".to_string());
            }
            script.remove(0)
        }
    }

    fn ok(body: &str) -> Result<Vec<u8>, String> {
        Ok(body.as_bytes().to_vec())
    }

    fn err(body: &str) -> Result<Vec<u8>, String> {
        Err(body.to_string())
    }

    fn make_service(exec: FakeExec) -> Service<FakeExec> {
        Service {
            exec,
            codex_harness: "/opt/harness".to_string(),
            codex_harness_sha256: PIN.to_string(),
            codex_harness_version: "1.2.3".to_string(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
        }
    }

    fn inspect_json() -> String {
        format!(
            "{{\"id\":{CID:?},\"running\":true,\"project\":{PID:?},\"owner\":\"7\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
        )
    }

    fn factory_run() -> FactoryRun {
        FactoryRun {
            deadline_raw: "2030-01-01T00:00:00Z".to_string(),
            actor: 7,
            id: RID.to_string(),
            project: PID.to_string(),
            role: ROLE.to_string(),
            preparation: PREP.to_string(),
            harness: FACTORY_HARNESS_CODEX.to_string(),
            harness_vers: "1.2.3".to_string(),
            model: String::new(),
            assignment: PIN.to_string(),
            source_commit: COMMIT.to_string(),
            connection: "conn".to_string(),
        }
    }

    fn run_dir() -> String {
        factory_run_paths(ROLE, PREP, RID).unwrap().1
    }

    fn factory_lease() -> Lease {
        Lease {
            provider_id: terminal::PROVIDER_CODEX.to_string(),
            id: "lease-f".to_string(),
            connection_id: "conn".to_string(),
            generation: 5,
            actor_id: 7,
            project_id: PID.to_string(),
            execution_id: RID.to_string(),
            kind: KIND_FACTORY.to_string(),
            binding: Some(Binding {
                kind: KIND_FACTORY.to_string(),
                id: RID.to_string(),
                project: CID.to_string(),
                login: ROLE.to_string(),
                uid: 1001,
                gid: 1001,
                scope: FACTORY_SCOPE_CODEX.to_string(),
                invocation_id: IID.to_string(),
                credential_root: run_dir(),
                generation: 5,
                child_id: PREP.to_string(),
            }),
            ..Default::default()
        }
    }

    fn write_harness(dir: &std::path::Path) {
        let bin = dir.join("bin");
        std::fs::create_dir_all(&bin).unwrap();
        std::fs::write(bin.join("codex"), b"").unwrap(); // sha256("") == PIN
        std::fs::write(bin.join("codex-code-mode-host"), b"host-bytes").unwrap();
        use std::os::unix::fs::PermissionsExt;
        for name in ["codex", "codex-code-mode-host"] {
            std::fs::set_permissions(bin.join(name), std::fs::Permissions::from_mode(0o755))
                .unwrap();
        }
    }

    fn host_digest() -> String {
        sha256::hex_lower(&sha256::digest(b"host-bytes"))
    }

    // ----- domain -----

    #[test]
    fn domain_predicates() {
        assert!(valid_factory_role("soda-coder"));
        assert!(valid_factory_role("soda-reviewer"));
        assert!(!valid_factory_role("dev"));
        assert!(valid_preparation_id(PREP));
        assert!(!valid_preparation_id("p0123456789abcdef01234567"));
        assert!(valid_digest(PIN));
        assert!(!valid_digest("xyz"));
        assert!(valid_commit(COMMIT));
        assert!(!valid_commit(&COMMIT[..39]));
        assert!(valid_factory_run_id(RID));
        assert!(!valid_factory_run_id("short"));
        assert!(valid_harness_version("1.2.3"));
        assert!(valid_harness_version("v1_beta-2.x"));
        assert!(!valid_harness_version(""));
        assert!(!valid_harness_version(".1"));
        assert!(!valid_harness_version(&"a".repeat(33)));
        assert!(!valid_harness_version("a/b"));
    }

    #[test]
    fn run_validate_pins() {
        assert!(factory_run().validate().is_ok());
        let cases = [
            (
                FactoryRun {
                    id: "short".to_string(),
                    ..factory_run()
                },
                "invalid factory run identity",
            ),
            (
                FactoryRun {
                    project: "nope".to_string(),
                    ..factory_run()
                },
                "invalid factory run identity",
            ),
            (
                FactoryRun {
                    role: "dev".to_string(),
                    ..factory_run()
                },
                "invalid factory run identity",
            ),
            (
                FactoryRun {
                    preparation: "nope".to_string(),
                    ..factory_run()
                },
                "invalid run preparation reference",
            ),
            (
                // The retired parallel family name stays denied: the
                // Muse CLI family is `muse`, never `muse-code`.
                FactoryRun {
                    harness: "muse-code".to_string(),
                    ..factory_run()
                },
                "unsupported factory harness",
            ),
            (
                FactoryRun {
                    harness_vers: "../x".to_string(),
                    ..factory_run()
                },
                "unsupported factory harness",
            ),
            (
                FactoryRun {
                    model: "a".repeat(129),
                    ..factory_run()
                },
                "invalid run model selection",
            ),
            (
                FactoryRun {
                    model: "a\nb".to_string(),
                    ..factory_run()
                },
                "invalid run model selection",
            ),
            (
                FactoryRun {
                    model: "a\x7fb".to_string(),
                    ..factory_run()
                },
                "invalid run model selection",
            ),
            (
                FactoryRun {
                    assignment: "short".to_string(),
                    ..factory_run()
                },
                "invalid run assignment or source identity",
            ),
            (
                FactoryRun {
                    source_commit: "short".to_string(),
                    ..factory_run()
                },
                "invalid run assignment or source identity",
            ),
            (
                FactoryRun {
                    connection: String::new(),
                    ..factory_run()
                },
                "invalid run sponsorship",
            ),
            (
                FactoryRun {
                    actor: 0,
                    ..factory_run()
                },
                "invalid run sponsorship",
            ),
            (
                FactoryRun {
                    deadline_raw: String::new(),
                    ..factory_run()
                },
                "run deadline is required",
            ),
        ];
        for (run, want) in cases {
            assert_eq!(run.validate().unwrap_err(), want, "{run:?}");
        }
        // Model bytes are checked raw: multibyte UTF-8 is fine.
        assert!(FactoryRun {
            model: "gpt-5é".to_string(),
            ..factory_run()
        }
        .validate()
        .is_ok());
        // Strict decode pins.
        assert!(FactoryRun::decode(br#"{"id":"x","bogus":1}"#).is_err());
        assert!(FactoryRun::decode(br#"{"deadline":"nope"}"#).is_err());
        let run = FactoryRun::decode(
            format!(
                "{{\"deadline\":\"2030-01-01T00:00:00Z\",\"actor\":7,\"id\":{RID:?},\"project\":{PID:?},\"role\":\"soda-coder\",\"preparation\":{PREP:?},\"harness\":\"codex\",\"harness_version\":\"1.2.3\",\"assignment\":{PIN:?},\"source_commit\":{COMMIT:?},\"connection\":\"conn\"}}"
            )
            .as_bytes(),
        )
        .unwrap();
        assert_eq!(run, factory_run());
    }

    #[test]
    fn path_vectors() {
        let (checkout, run_dir, home, codex) = factory_run_paths(ROLE, PREP, RID).unwrap();
        assert_eq!(checkout, format!("/home/{ROLE}/checkouts/{PREP}"));
        assert_eq!(run_dir, format!("{checkout}/.soda-home/runs/{RID}"));
        assert_eq!(home, format!("{run_dir}/home"));
        assert_eq!(codex, format!("{home}/.codex"));
        assert!(factory_run_paths("dev", PREP, RID).is_none());
        assert!(factory_run_paths(ROLE, "nope", RID).is_none());
        assert!(factory_run_paths(ROLE, PREP, "nope").is_none());
        assert_eq!(
            factory_codex_guest("1.2.3").unwrap(),
            "/usr/local/bin/codex-factory-1.2.3"
        );
        assert!(factory_codex_guest("../x").is_none());
        assert_eq!(
            factory_unit_name(RID).unwrap(),
            format!("soda-factory-{RID}.service")
        );
        assert!(factory_unit_name("nope").is_none());
        assert_eq!(
            takeover_destination("dev", RID).unwrap(),
            format!("/home/dev/factory-takeover/{RID}")
        );
        assert!(takeover_destination("root", RID).is_none());
        assert!(takeover_source(
            &format!("/home/{ROLE}/checkouts/{PREP}"),
            ROLE,
            PREP
        ));
        assert!(!takeover_source(
            "/home/soda-coder/checkouts/other",
            ROLE,
            PREP
        ));
        assert!(!takeover_source(
            &format!("/home/{ROLE}/checkouts/{PREP}"),
            "dev",
            PREP
        ));
        let p = factory_codex_paths(&factory_run()).unwrap();
        assert_eq!(p.prompt, format!("{}/prompt", p.run_dir));
        assert_eq!(p.auth, format!("{}/auth.json", p.codex));
        assert_eq!(p.guest, "/usr/local/bin/codex-factory-1.2.3");
        assert!(factory_codex_paths(&FactoryRun::default()).is_err());
    }

    #[test]
    fn quote_vectors() {
        assert_eq!(shell_quote("abc"), "'abc'");
        assert_eq!(shell_quote("a'b"), "'a'\\''b'");
        assert_eq!(systemd_escape("a$b$$c"), "a$$b$$$$c");
        assert_eq!(systemd_escape("plain"), "plain");
    }

    // ----- script goldens (byte-exact vs Go) -----

    const GOLDEN_SUPERVISOR: &str = r#"RUNDIR='/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef'
exec >"$RUNDIR/stdout.log" 2>&1
STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo "$$ ${20}" >"$RUNDIR/supervisor.pid"
fail() { echo "$1" >"$RUNDIR/exit"; exit "$1"; }
i=0; while [ ! -f "$RUNDIR/marker" ]; do [ -f "$RUNDIR/stop" ] && fail 44; i=$((i+1)); [ "$i" -gt 600 ] && fail 42; sleep 1; done
mv "$RUNDIR/marker" "$RUNDIR/started" || fail 43
'/usr/local/bin/codex-factory-1.2.3' exec --color never --sandbox danger-full-access --skip-git-repo-check --config 'model_reasoning_effort="low"' --output-last-message '/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef/last-message.txt' - <'/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef/prompt'
CODE=$?; echo "$CODE" >"$RUNDIR/exit"; exit "$CODE"
"#;

    const GOLDEN_RETIRE: &str = r#"RUNDIR='/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef'
: >"$RUNDIR/stop"
[ -f "$RUNDIR/supervisor.pid" ] || exit 0
read PID START <"$RUNDIR/supervisor.pid"
case "$PID" in ''|*[!0-9]*) exit 0;; esac
case "$START" in ''|*[!0-9]*) exit 0;; esac
if [ -d "/proc/$PID" ]; then
  if STAT=$(cat "/proc/$PID/stat" 2>/dev/null); then
    REST=${STAT##*)}; set -- $REST
    if [ "${20}" = "$START" ]; then
      CMDLINE=$(tr '\000' ' ' <"/proc/$PID/cmdline" 2>/dev/null) || CMDLINE=""
      case "$CMDLINE" in *"$RUNDIR"*)
        if [ "$3" = "$PID" ]; then kill -KILL -- "-$PID" 2>/dev/null || true; else kill -KILL -- "$PID" 2>/dev/null || true; fi
      ;; esac
    fi
  fi
fi
sleep 1
for S in /proc/[0-9]*/stat; do
  STAT=$(cat "$S" 2>/dev/null) || continue
  REST=${STAT##*)}; set -- $REST
  if [ "$3" = "$PID" ]; then echo "lingering: $S"; exit 1; fi
done
exit 0
"#;

    #[test]
    fn script_goldens() {
        let p = factory_codex_paths(&factory_run()).unwrap();
        assert_eq!(factory_supervisor(&p, &p.guest, ""), GOLDEN_SUPERVISOR);
        let with_model = factory_supervisor(&p, &p.guest, "gpt-5");
        assert!(
            with_model.contains("--model 'gpt-5' --output-last-message"),
            "{with_model}"
        );
        assert_eq!(
            with_model,
            GOLDEN_SUPERVISOR.replace(
                "--config 'model_reasoning_effort=\"low\"' --output-last-message",
                "--config 'model_reasoning_effort=\"low\"' --model 'gpt-5' --output-last-message"
            )
        );
        assert_eq!(factory_retire(&p), GOLDEN_RETIRE);
        // Escaped supervisor doubles every dollar.
        let escaped = systemd_escape(&with_model);
        assert_eq!(escaped, with_model.replace('$', "$$"));
        assert!(escaped.contains("exec >\"$$RUNDIR/stdout.log\" 2>&1"));
        assert!(escaped.contains("'/usr/local/bin/codex-factory-1.2.3' exec"));
        assert_eq!(
            FACTORY_EXPORT_SCRIPT,
            "set -eu\nsrc=$1\ncandidate=$2\nlimit=$3\nif ! /usr/bin/test -d \"$src/.git/objects\"; then\n  printf 'soda-export-missing\\n'\n  exit 0\nfi\ndir=$(/usr/bin/mktemp -d \"$TMPDIR/.soda-export-XXXXXX\")\ntrap '/usr/bin/rm -rf \"$dir\"' EXIT HUP INT TERM\n/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= \"$dir/repo.git\" >/dev/null 2>/dev/null\nexport GIT_OBJECT_DIRECTORY=\"$src/.git/objects\"\ngit_export() {\n  /usr/bin/git -c core.hooksPath=/dev/null --git-dir=\"$dir/repo.git\" \"$@\"\n}\nif ! actual=$(git_export rev-parse --verify \"$candidate^{commit}\" 2>/dev/null); then\n  printf 'soda-export-invalid\\n'\n  exit 0\nfi\nif [ \"$actual\" != \"$candidate\" ]; then\n  printf 'soda-export-invalid\\n'\n  exit 0\nfi\ngit_export update-ref HEAD \"$candidate\" 2>/dev/null\ngit_export bundle create \"$dir/candidate.bundle\" HEAD 2>/dev/null\n/usr/bin/head -c \"$limit\" \"$dir/candidate.bundle\"\n"
        );
    }

    #[test]
    fn binding_matrix() {
        let lease = factory_lease();
        let p = factory_codex_binding(&lease).unwrap();
        assert_eq!(p.run_dir, run_dir());
        assert!(p.guest.is_empty()); // harness fields are not in the binding
        let mut broken = lease.clone();
        broken.binding = None;
        assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
        for mutate in [
            Box::new(|b: &mut Binding| b.kind = "terminal".to_string())
                as Box<dyn Fn(&mut Binding)>,
            Box::new(|b: &mut Binding| b.scope = "other".to_string()),
            Box::new(|b: &mut Binding| b.id = "short".to_string()),
            Box::new(|b: &mut Binding| b.project = "short".to_string()),
            Box::new(|b: &mut Binding| b.login = "root".to_string()),
            Box::new(|b: &mut Binding| b.uid = 0),
            Box::new(|b: &mut Binding| b.generation = 99),
            Box::new(|b: &mut Binding| b.invocation_id = "short".to_string()),
            Box::new(|b: &mut Binding| b.child_id = "nope".to_string()),
            Box::new(|b: &mut Binding| b.credential_root = "/elsewhere".to_string()),
        ] {
            let mut broken = lease.clone();
            mutate(broken.binding.as_mut().unwrap());
            assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
        }
        let mut broken = lease.clone();
        broken.provider_id = "muse".to_string();
        assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
        broken = lease.clone();
        broken.kind = "terminal".to_string();
        assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
    }

    #[test]
    fn unit_show_vectors() {
        assert_eq!(
            parse_factory_unit_show(b"ActiveState=active\nInvocationID=abc\n"),
            FactoryUnitShow {
                active: true,
                invocation: "abc".to_string()
            }
        );
        assert_eq!(
            parse_factory_unit_show(b"ActiveState=inactive\nInvocationID=\n"),
            FactoryUnitShow {
                active: false,
                invocation: String::new()
            }
        );
        // Last occurrence wins.
        assert!(parse_factory_unit_show(b"ActiveState=inactive\nActiveState=active\n").active);
        assert_eq!(parse_factory_unit_show(b""), FactoryUnitShow::default());
        assert_eq!(factory_role_id(b"1001\n").unwrap(), 1001);
        assert_eq!(factory_role_id(b"0").unwrap_err(), "invalid role identity");
        assert_eq!(factory_role_id(b"-5").unwrap_err(), "invalid role identity");
        assert_eq!(
            factory_role_id(b"nope").unwrap_err(),
            "invalid role identity"
        );
        assert_eq!(factory_output_size(b"123\n"), Some(123));
        assert_eq!(factory_output_size(b"-1"), None);
        assert_eq!(factory_output_size(b"nope"), None);
        assert_eq!(factory_output_size(&[b'1'; 65]), None);
    }

    // ----- reserve/start/wait state machine -----

    fn euid() -> u32 {
        unsafe { libc::geteuid() }
    }

    fn reserve_harness() -> (std::path::PathBuf, String) {
        let dir = test_tmp("reserve");
        write_harness(&dir);
        let path = dir.to_str().unwrap().to_string();
        (dir, path)
    }

    #[test]
    fn reserve_denial_pins() {
        let (_dir, harness) = reserve_harness();
        let run = factory_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        // Each denial fires before any exec call.
        let bad_run = FactoryRun {
            id: "short".to_string(),
            ..factory_run()
        };
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_codex_reserve(&bad_run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "invalid factory run identity"
        );
        let svc = make_service(FakeExec::new(vec![]));
        let mut svc = svc;
        svc.codex_harness = harness.clone();
        let bad_lease = Lease {
            kind: "terminal".to_string(),
            ..lease.clone()
        };
        assert_eq!(
            svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let bad_lease = Lease {
            execution_id: "other".to_string(),
            ..lease.clone()
        };
        assert_eq!(
            svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let bad_lease = Lease {
            generation: 0,
            ..lease.clone()
        };
        assert_eq!(
            svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, "", 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, &"f".repeat(64), 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        svc.codex_harness = "relative/path".to_string();
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        svc.codex_harness = harness;
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 59, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 10801, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
    }

    #[test]
    fn reserve_success_argv_sequence() {
        let (_dir, harness) = reserve_harness();
        let host = host_digest();
        let guest = "/usr/local/bin/codex-factory-1.2.3";
        let unit = factory_unit_name(RID).unwrap();
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),              // project container
            ok("1001\n"),                     // id -u
            ok("1002\n"),                     // id -g
            ok(""),                           // setup script
            ok(&format!("{COMMIT}\n")),       // git rev-parse
            ok(&format!("{PIN}  {guest}\n")), // guest sha256sum (present)
            ok(&format!("{host}  /usr/local/bin/codex-code-mode-host\n")), // host probe (present)
            ok(""),                           // systemd-run
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")), // attestation
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness;
        let run = factory_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        let (binding, p) = svc
            .factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap();
        assert_eq!(binding.id, RID);
        assert_eq!(binding.project, CID);
        assert_eq!(binding.login, ROLE);
        assert_eq!((binding.uid, binding.gid), (1001, 1002));
        assert_eq!(binding.scope, FACTORY_SCOPE_CODEX);
        assert_eq!(binding.invocation_id, IID);
        assert_eq!(binding.credential_root, p.run_dir);
        assert_eq!(binding.generation, 5);
        assert_eq!(binding.child_id, PREP);
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 9);
        // systemd-run argv is byte-exact: header + podman payload.
        let bus = factory_user_bus(euid());
        let mut want_run = vec![
            "/usr/bin/env".to_string(),
            bus,
            "/usr/bin/systemd-run".to_string(),
            "--user".to_string(),
        ];
        want_run.extend(reserve_run_argv(&unit, 600));
        want_run.extend(reserve_exec_argv(CID, &run, &p, guest));
        let got_run: Vec<String> = std::iter::once(calls[7].1.clone())
            .chain(calls[7].2.clone())
            .collect();
        assert_eq!(got_run, want_run);
        // Setup script pins directories and ownership.
        assert_eq!(calls[3].1, "/usr/bin/podman");
        assert!(
            calls[3].2[5].starts_with("set -u\nmkdir -p -m 700 "),
            "{}",
            calls[3].2[5]
        );
        assert!(
            calls[3].2[5].contains("chown 1001:1002 "),
            "{}",
            calls[3].2[5]
        );
        // Attestation show pins the unit.
        assert_eq!(
            calls[8].2[..4],
            [
                factory_user_bus(euid()),
                "/usr/bin/systemctl".to_string(),
                "--user".to_string(),
                "show".to_string()
            ]
        );
        assert_eq!(calls[8].2[5], unit);
    }

    #[test]
    fn reserve_stage_and_failure_paths() {
        let (_dir, harness) = reserve_harness();
        let guest = "/usr/local/bin/codex-factory-1.2.3";
        // Guest harness absent: cp + install path with digest check.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            err("exit status 1"),             // guest probe misses
            ok(""),                           // podman cp
            ok(&format!("{PIN}  {guest}\n")), // install digest
            err("exit status 1"),             // host probe misses
            ok(""),                           // host cp
            ok(&format!(
                "{}  /usr/local/bin/codex-code-mode-host\n",
                host_digest()
            )), // host install
            ok(""),
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness.clone();
        let run = factory_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        let (binding, _) = svc
            .factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap();
        assert_eq!(binding.uid, 1001);
        let calls = svc.exec.calls();
        assert_eq!(calls[6].2[1], "cp");
        assert_eq!(calls[6].2[3], format!("{CID}:{guest}.new"));
        assert!(
            calls[7].2[5].contains(&format!("mv '{guest}.new' '{guest}'")),
            "{}",
            calls[7].2[5]
        );
        // Digest mismatch after install.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            ok(&format!("{}  {guest}\n", "f".repeat(64))),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness.clone();
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "guest harness digest differs"
        );
        // Wrong checkout commit.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(""),
            ok(&format!("{}xxx\n", &COMMIT[..37])),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness.clone();
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "factory checkout is not the assigned commit"
        );
        // systemd-run failure.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            ok(&format!("{PIN}  {guest}\n")),
            ok(&format!(
                "{}  /usr/local/bin/codex-code-mode-host\n",
                host_digest()
            )),
            err("exit status 1"),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness;
        assert_eq!(
            svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "factory unit start unconfirmed"
        );
    }

    #[test]
    fn reserve_unattested_unit_is_stopped() {
        let (_dir, harness) = reserve_harness();
        let guest = "/usr/local/bin/codex-factory-1.2.3";
        // Attestation never arrives: the started unit is stopped before
        // the error returns. A short deadline keeps the test fast.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            ok(&format!("{PIN}  {guest}\n")),
            ok(&format!(
                "{}  /usr/local/bin/codex-code-mode-host\n",
                host_digest()
            )),
            ok(""),
            ok("ActiveState=activating\nInvocationID=\n"),
        ]);
        let mut svc = make_service(exec);
        svc.codex_harness = harness;
        let run = factory_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        let tight = Instant::now() + Duration::from_millis(150);
        let result = svc.factory_codex_reserve(&run, &lease, PIN, 600, tight);
        assert!(result.is_err());
        let calls = svc.exec.calls();
        let last = calls.last().unwrap();
        assert_eq!(last.1, "/usr/bin/env");
        assert_eq!(last.2[1], "/usr/bin/systemctl");
        assert_eq!(last.2[3], "stop");
        assert_eq!(last.2[4], factory_unit_name(RID).unwrap());
    }

    #[test]
    fn start_flows() {
        let lease = factory_lease();
        let p = factory_codex_binding(&lease).unwrap();
        // Denials before exec.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_codex_start(&Lease::default(), b"{}", b"prompt", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_start(&lease, b"nope", b"prompt", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_start(&lease, b"{}", b"", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_start(
                &lease,
                b"{}",
                &vec![b'x'; MAX_FACTORY_PROMPT + 1],
                deadline()
            )
            .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Stale incarnation.
        let other = "f".repeat(64);
        let stale_json = inspect_json().replace(CID, &other);
        let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
        assert_eq!(
            svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
                .unwrap_err(),
            ERR_STALE
        );
        // Success stages credential, prompt, marker, then the gate.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
        ]));
        svc.factory_codex_start(&lease, b"{\"t\":1}", b"do work", deadline())
            .unwrap();
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 5);
        assert_eq!(calls[1].0, b"{\"t\":1}");
        assert_eq!(calls[2].0, b"do work");
        assert!(calls[3].0.is_empty());
        assert!(
            calls[1].2[6].contains(&shell_quote(&p.auth)),
            "{}",
            calls[1].2[6]
        );
        assert!(
            calls[1].2[6].contains("chown 1001:1001 "),
            "{}",
            calls[1].2[6]
        );
        assert!(
            calls[2].2[6].contains(&shell_quote(&p.prompt)),
            "{}",
            calls[2].2[6]
        );
        assert!(
            calls[3].2[6].contains(&shell_quote(&p.marker)),
            "{}",
            calls[3].2[6]
        );
        assert_eq!(calls[4].2[5], start_gate_script(&p));
        // Gate failure.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok(""),
            ok(""),
            ok(""),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
                .unwrap_err(),
            "factory start staging unconfirmed"
        );
        // Stage failure.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
                .unwrap_err(),
            "factory file staging unconfirmed"
        );
    }

    #[test]
    fn wait_flows() {
        let lease = factory_lease();
        let p = factory_codex_binding(&lease).unwrap();
        // Active then inactive, exit 3, bounded output.
        let svc = make_service(FakeExec::new(vec![
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok("3\n"),
            ok("hello output"),
        ]));
        assert_eq!(
            svc.factory_codex_wait(&lease, deadline()).unwrap(),
            (3, "hello output".to_string())
        );
        let calls = svc.exec.calls();
        assert_eq!(calls[2].2[4], format!("{}/exit", p.run_dir));
        assert_eq!(calls[3].2[4], "-c");
        assert_eq!(calls[3].2[5], "65537");
        // Missing/garbage exit records read -1 without failing.
        let svc = make_service(FakeExec::new(vec![
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_codex_wait(&lease, deadline()).unwrap(),
            (-1, String::new())
        );
        let svc = make_service(FakeExec::new(vec![
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok("not-a-number\n"),
            ok("out"),
        ]));
        assert_eq!(
            svc.factory_codex_wait(&lease, deadline()).unwrap(),
            (-1, "out".to_string())
        );
        let svc = make_service(FakeExec::new(vec![
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok("999\n"),
            ok("out"),
        ]));
        assert_eq!(
            svc.factory_codex_wait(&lease, deadline()).unwrap(),
            (-1, "out".to_string())
        );
        // Observation failure propagates.
        let svc = make_service(FakeExec::new(vec![err("boom")]));
        assert_eq!(
            svc.factory_codex_wait(&lease, deadline()).unwrap_err(),
            "factory unit observation unavailable"
        );
        // Binding failure first.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_codex_wait(&Lease::default(), deadline())
                .unwrap_err(),
            ERR_DENIED
        );
    }

    // ----- validate/stop/capture/finish -----

    #[test]
    fn validate_flows() {
        let lease = factory_lease();
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ]));
        assert!(svc.factory_codex_validate(&lease, deadline()).is_ok());
        assert_eq!(svc.exec.calls().len(), 4);
        // Role mismatch denies.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("9999\n"),
        ]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
        // Inactive unit denies.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok("ActiveState=inactive\nInvocationID=\n"),
        ]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
        // Invocation mismatch denies.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(&format!(
                "ActiveState=active\nInvocationID={}\n",
                "b".repeat(32)
            )),
        ]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
        // Observation failure denies (not the raw error).
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            err("boom"),
        ]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
    }

    #[test]
    fn stop_flows() {
        let lease = factory_lease();
        let p = factory_codex_binding(&lease).unwrap();
        // Full retire: pid, stop, inactive, exists, retire, pid (unchanged).
        let svc = make_service(FakeExec::new(vec![
            ok("4242 99999\n"),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            ok(""),
            ok("4242 99999\n"),
        ]));
        svc.factory_codex_stop(&lease, deadline()).unwrap();
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 6);
        assert_eq!(calls[4].2[5], factory_retire(&p));
        // Removed container retires the unit only.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
        ]));
        svc.factory_codex_stop(&lease, deadline()).unwrap();
        assert_eq!(svc.exec.calls().len(), 4);
        // Uncertain container probe.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 2"),
        ]));
        assert_eq!(
            svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        // Retire failure is uncertain.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        // Changed pid re-runs the retire script once; still-changed is uncertain.
        let svc = make_service(FakeExec::new(vec![
            ok("4242 1\n"),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            ok(""),
            ok("5151 2\n"),
            ok(""),
            ok("5151 2\n"),
        ]));
        svc.factory_codex_stop(&lease, deadline()).unwrap();
        assert_eq!(svc.exec.calls().len(), 8);
        let svc = make_service(FakeExec::new(vec![
            ok("4242 1\n"),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            ok(""),
            ok("5151 2\n"),
            ok(""),
            ok("6161 3\n"),
        ]));
        assert_eq!(
            svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        // A unit that never dies is uncertain (expired deadline, no sleep).
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ]));
        let past = Instant::now() - Duration::from_secs(1);
        assert_eq!(
            svc.factory_codex_stop(&lease, past).unwrap_err(),
            ERR_UNCERTAIN
        );
    }

    #[test]
    fn capture_and_finish_flows() {
        let lease = factory_lease();
        let p = factory_codex_binding(&lease).unwrap();
        let svc = make_service(FakeExec::new(vec![ok("{\"maintained\":true}")]));
        assert_eq!(
            svc.factory_codex_capture(&lease, deadline()).unwrap(),
            b"{\"maintained\":true}".to_vec()
        );
        let calls = svc.exec.calls();
        assert_eq!(calls[0].2[3], "/usr/bin/head");
        assert_eq!(calls[0].2[4], "-c");
        assert_eq!(calls[0].2[5], "262145");
        assert_eq!(calls[0].2[6], p.auth);
        let svc = make_service(FakeExec::new(vec![err("exit status 1")]));
        assert_eq!(
            svc.factory_codex_capture(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        let svc = make_service(FakeExec::new(vec![ok("not-json")]));
        assert_eq!(
            svc.factory_codex_capture(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        // Finish stops, then captures.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            ok("{\"m\":1}"),
        ]));
        assert_eq!(
            svc.factory_codex_finish(&lease, deadline()).unwrap(),
            b"{\"m\":1}".to_vec()
        );
        // Finish propagates stop failures without capturing.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_codex_finish(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
        assert_eq!(svc.exec.calls().len(), 5);
    }

    #[test]
    fn stop_unbound_flows() {
        let run = factory_run();
        let stopped_json = inspect_json().replace("\"running\":true", "\"running\":false");
        // Stopped containers still retire (requireRunning=false).
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(&stopped_json),
            ok(""),
            ok(""),
            ok(""),
        ]));
        svc.factory_codex_stop_unbound(&run, deadline()).unwrap();
        assert_eq!(svc.exec.calls().len(), 6);
        // Missing container resolves to success.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            err("exit status 1"),
        ]));
        svc.factory_codex_stop_unbound(&run, deadline()).unwrap();
        let calls = svc.exec.calls();
        assert_eq!(
            calls[3].2,
            terminal::container_exists_argv(&format!("soda-{PID}"))
        );
        // Present-but-uninspectable container is uncertain.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            ok(""),
        ]));
        assert_eq!(
            svc.factory_codex_stop_unbound(&run, deadline())
                .unwrap_err(),
            ERR_UNCERTAIN
        );
        // Invalid run never calls out.
        let svc = make_service(FakeExec::new(vec![]));
        assert!(svc
            .factory_codex_stop_unbound(&FactoryRun::default(), deadline())
            .is_err());
        assert!(svc.exec.calls().is_empty());
    }

    #[test]
    fn live_matrix() {
        let lease = factory_lease();
        let binding = lease.binding.clone().unwrap();
        let svc = make_service(FakeExec::new(vec![ok(&format!(
            "ActiveState=active\nInvocationID={IID}\n"
        ))]));
        assert!(svc.factory_codex_live(&binding, deadline()));
        let svc = make_service(FakeExec::new(vec![ok(
            "ActiveState=inactive\nInvocationID=\n",
        )]));
        assert!(!svc.factory_codex_live(&binding, deadline()));
        let svc = make_service(FakeExec::new(vec![err("boom")]));
        assert!(!svc.factory_codex_live(&binding, deadline()));
        // Shape faults never call out.
        let svc = make_service(FakeExec::new(vec![]));
        let mut bad = binding.clone();
        bad.scope = "other".to_string();
        assert!(!svc.factory_codex_live(&bad, deadline()));
        bad = binding;
        bad.invocation_id = "short".to_string();
        assert!(!svc.factory_codex_live(&bad, deadline()));
        assert!(svc.exec.calls().is_empty());
    }

    // ----- output/export/takeover -----

    #[test]
    fn output_flows() {
        let lease = factory_lease();
        let binding = lease.binding.clone().unwrap();
        // Cursor validation first.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_codex_output(PID, &binding, -1, 100, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_output(PID, &binding, 0, 0, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_codex_output(PID, &binding, 0, MAX_FACTORY_OUTPUT_READ + 1, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Gap past the recorded size.
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("10\n")]));
        let slice = svc
            .factory_codex_output(PID, &binding, 11, 100, deadline())
            .unwrap();
        assert_eq!((slice.total, slice.offset, slice.gap), (10, 10, true));
        assert!(slice.data.is_empty());
        // Zero cursor on an over-window log serves the trailing window.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("300000\n"),
            ok("tail-bytes"),
        ]));
        let slice = svc
            .factory_codex_output(PID, &binding, 0, 100, deadline())
            .unwrap();
        assert_eq!(
            (slice.total, slice.offset, slice.truncated),
            (300000, 300000 - MAX_FACTORY_OUTPUT_WINDOW, true)
        );
        assert_eq!(slice.data, b"tail-bytes");
        let calls = svc.exec.calls();
        assert!(
            calls[2].2[5].contains(&format!(
                "tail -c +{}",
                300000 - MAX_FACTORY_OUTPUT_WINDOW + 1
            )),
            "{}",
            calls[2].2[5]
        );
        // Cursor at end reads nothing further.
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("10\n")]));
        let slice = svc
            .factory_codex_output(PID, &binding, 10, 100, deadline())
            .unwrap();
        assert_eq!(
            (slice.total, slice.offset, slice.truncated),
            (10, 10, false)
        );
        assert_eq!(svc.exec.calls().len(), 2);
        // Read failures return the cursor without data or error.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("10\n"),
            err("exit status 1"),
        ]));
        let slice = svc
            .factory_codex_output(PID, &binding, 0, 100, deadline())
            .unwrap();
        assert_eq!((slice.total, slice.offset), (10, 0));
        assert!(slice.data.is_empty());
        // Overlong reads truncate to the limit.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1000\n"),
            ok(&"x".repeat(200)),
        ]));
        let slice = svc
            .factory_codex_output(PID, &binding, 0, 100, deadline())
            .unwrap();
        assert_eq!(slice.data.len(), 100);
        // Missing log reads empty; stat failure leaves total zero.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            err("exit status 1"),
        ]));
        let slice = svc
            .factory_codex_output(PID, &binding, 0, 100, deadline())
            .unwrap();
        assert_eq!((slice.total, slice.offset), (0, 0));
        // Stale incarnation.
        let other = "f".repeat(64);
        let stale_json = inspect_json().replace(CID, &other);
        let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
        assert_eq!(
            svc.factory_codex_output(PID, &binding, 0, 100, deadline())
                .unwrap_err(),
            ERR_STALE
        );
    }

    #[test]
    fn export_flows() {
        let src = format!("/home/{ROLE}/checkouts/{PREP}");
        // Validation denies.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, "dev", PREP, COMMIT, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, "short", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_export_bundle(PID, "short", ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Success pins the clean-env argv.
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("bundle-bytes")]));
        let bundle = svc
            .factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap();
        assert_eq!(bundle, b"bundle-bytes");
        let calls = svc.exec.calls();
        assert_eq!(calls[1].2, export_argv(CID, ROLE, &src, COMMIT));
        assert!(calls[1].2.contains(&"soda-export".to_string()));
        assert!(calls[1]
            .2
            .contains(&(MAX_FACTORY_EXPORT_BUNDLE + 1).to_string()));
        // Sentinel verdicts.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("soda-export-missing\n"),
        ]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            terminal::ERR_NOT_FOUND
        );
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("soda-export-invalid\n"),
        ]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            ERR_FACTORY_EXPORT_CANDIDATE
        );
        // Bounds.
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("")]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            ERR_FACTORY_EXPORT_BOUNDS
        );
        // Exec failure with a live deadline.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            "candidate export execution unconfirmed"
        );
        // Stale incarnation.
        let other = "f".repeat(64);
        let stale_json = inspect_json().replace(CID, &other);
        let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
        assert_eq!(
            svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
                .unwrap_err(),
            ERR_STALE
        );
    }

    #[test]
    fn takeover_flows() {
        let dest = takeover_destination("dev", RID).unwrap();
        let src = format!("/home/{ROLE}/checkouts/{PREP}");
        // Validation denies.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, "dev", PREP, "dev", RID, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "root", RID, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Existing destination reuses.
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("")]));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap(),
            (dest.clone(), true)
        );
        // Missing source.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            err("exit status 1"),
            err("exit status 1"),
        ]));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap_err(),
            "takeover found no retained work"
        );
        // Full copy pins every step.
        let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
        script.extend(std::iter::repeat_with(|| ok("")).take(7));
        script.push(err("exit status 1")); // dest still absent
        script.push(ok("")); // mv
        let svc = make_service(FakeExec::new(script));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap(),
            (dest.clone(), false)
        );
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 12);
        let steps = takeover_steps(&src, &dest, "dev");
        for (i, step) in steps.iter().enumerate() {
            let mut want = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                CID.to_string(),
            ];
            want.extend(step.clone());
            assert_eq!(calls[3 + i].2, want, "step {i}");
        }
        assert_eq!(
            calls[10].2[3..],
            ["/usr/bin/test".to_string(), "-e".to_string(), dest.clone()]
        );
        assert_eq!(
            calls[11].2[3..7],
            [
                "/usr/bin/mv".to_string(),
                "-T".to_string(),
                format!("{dest}.partial"),
                dest.clone()
            ]
        );
        // Step failures propagate raw.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            err("exit status 1"),
            ok(""),
            err("exit status 3"),
        ]));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap_err(),
            "exit status 3"
        );
        // Racing destination reuses after cleanup.
        let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
        script.extend(std::iter::repeat_with(|| ok("")).take(7));
        script.push(ok("")); // dest appeared
        script.push(ok("")); // rm partial
        let svc = make_service(FakeExec::new(script));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap(),
            (dest.clone(), true)
        );
        // Move failure reports unconfirmed after cleanup.
        let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
        script.extend(std::iter::repeat_with(|| ok("")).take(7));
        script.push(err("exit status 1"));
        script.push(err("exit status 1")); // mv fails
        script.push(ok("")); // rm partial
        let svc = make_service(FakeExec::new(script));
        assert_eq!(
            svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
                .unwrap_err(),
            "takeover destination was not confirmed"
        );
    }

    #[test]
    fn factory_identity_operation_matrix() {
        let lease = factory_lease();
        let delivery = Delivery {
            lease: lease.clone(),
            credential: Some(b"{}".to_vec()),
        };
        // Validate clears any credential and attests.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ]));
        let out = svc
            .factory_identity_operation("validate", &delivery, deadline())
            .unwrap();
        assert!(out.credential.is_none());
        assert_eq!(out.lease, lease);
        // Stop retires.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
        ]));
        let out = svc
            .factory_identity_operation("stop", &delivery, deadline())
            .unwrap();
        assert!(out.credential.is_none());
        // Finish returns the captured credential.
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            ok("{\"m\":1}"),
        ]));
        let out = svc
            .factory_identity_operation("finish", &delivery, deadline())
            .unwrap();
        assert_eq!(out.credential, Some(b"{\"m\":1}".to_vec()));
        // Unknown actions deny without calling out.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_identity_operation("launch", &delivery, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Errors propagate.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("9999\n"),
        ]));
        assert_eq!(
            svc.factory_identity_operation("validate", &delivery, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
    }
}
