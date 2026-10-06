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
use crate::project::Executor;
use crate::sha256;
use crate::terminal::{self, Binding, Lease, Service, KIND_FACTORY};

// `muse_serve_oracle` compiles this module through a private `#[path]` copy
// that never touches some re-exported names; they serve the real library.
#[allow(unused_imports)]
pub use super::artifacts::{
    export_argv, takeover_destination, takeover_source, takeover_steps, ERR_FACTORY_EXPORT_BOUNDS,
    ERR_FACTORY_EXPORT_CANDIDATE, FACTORY_EXPORT_SCRIPT, MAX_FACTORY_EXPORT_BUNDLE,
    TAKEOVER_DIR_NAME,
};
pub use super::lifecycle::factory_retire;
pub use super::output::{
    factory_output_size, output_read_command, FactoryCodexOutputSlice, MAX_FACTORY_OUTPUT_OFFSET,
    MAX_FACTORY_OUTPUT_READ, MAX_FACTORY_OUTPUT_WINDOW,
};
#[allow(unused_imports)]
pub use super::run::{
    valid_commit, valid_digest, valid_factory_role, valid_factory_run_id, valid_harness_family,
    valid_harness_version, valid_preparation_id, FactoryRun, FACTORY_HARNESS_CODEX,
    FACTORY_HARNESS_MUSE, FACTORY_SCOPE_CODEX, FACTORY_SCOPE_MUSE, MAX_FACTORY_PROMPT, ROLE_CODER,
    ROLE_REVIEWER,
};

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
    let (checkout, run_dir, home, codex) =
        crate::terminal::factory::tfactory::checked_binding_paths(
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
        crate::terminal::factory::tfactory::check_output_range(offset, limit)?;
        let stdout = self.factory_output_stdout(
            project_id,
            binding,
            FACTORY_SCOPE_CODEX,
            factory_run_paths,
            deadline,
        )?;
        self.factory_output_window(&binding.project, &stdout, offset, limit, deadline)
    }
}
