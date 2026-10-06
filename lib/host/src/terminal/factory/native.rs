//! Shared native execution substrate for supervised factory runs:
//! transient-unit naming, shell/systemd escaping, user-bus and
//! systemctl/systemd-run argv, harness install and stage-file
//! commands, unit observation and wait, and the native Service
//! methods (env/Podman/systemd/unit/invocation/role/stage-file).

use super::run::valid_factory_run_id;
use crate::project::Executor;
use crate::terminal::{self, Binding, Service};
use std::time::{Duration, Instant};

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

/// `systemdEscape`: double every dollar for transport through systemd-run.
pub fn systemd_escape(script: &str) -> String {
    script.replace('$', "$$")
}

/// Single-quote one shell word.
pub fn shell_quote(value: &str) -> String {
    format!("'{}'", value.replace('\'', "'\\''"))
}

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
}
