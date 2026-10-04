//! Tailnet companion orchestration: container lifecycle and observation.
//! Lane D owns this file.
//!
//! Rust port of `internal/host/tailnet/companion.go`. Deadlines are absolute
//! [`Instant`]s passed last; sub-deadlines derive from `Instant::now()` plus a
//! fixed duration, mirroring Go's `context.WithTimeout` nesting.

use std::os::unix::fs::MetadataExt;
use std::time::{Duration, Instant};

use crate::json::{bind_root, decode_strict, Kind, Spec};
use crate::project::Executor;
use crate::tailnet_domain::{
    parse_rfc3339_nano, project_has_node, project_status, valid_container_id, valid_project_id,
    ProjectRequest, ProjectView, RunBinding, RunTarget, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE,
    ERR_UNCONFIRMED,
};
use crate::tailnet_files::{
    open_runtime_project, read_companion_id, retire_pending_run_key, retire_run_key,
    validate_run_resolver, write_companion_id, write_run_key, Root, RunFiles,
};
use crate::tailnet_runtime::{
    companion_create_args, project_container, project_run, project_running, recheck_project_run,
    ProjectRun,
};

pub const RUNTIME_ROOT: &str = "/run/soda-tailnet";
pub const COMPANION_INSPECT: &str = "{\"id\":{{json .ID}},\"image\":{{json .Image}},\"command\":{{json .Config.CreateCommand}},\"running\":{{json .State.Running}},\"pid\":{{json .State.Pid}},\"started\":{{json .State.StartedAt}},\"execs\":{{json .ExecIDs}}}";

/// Decoded companion `inspect` record. Field names match the Go struct so
/// binding errors keep Go's `companion.<Field>` paths.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct CompanionRecord {
    pub id: String,
    pub image: String,
    pub command: Vec<String>,
    pub running: bool,
    pub pid: i64,
    pub started: String,
    pub execs: Vec<String>,
}

const COMPANION_SPECS: &[Spec] = &[
    Spec {
        name: "ID",
        kind: Kind::Str,
    },
    Spec {
        name: "Image",
        kind: Kind::Str,
    },
    Spec {
        name: "Command",
        kind: Kind::StrList,
    },
    Spec {
        name: "Running",
        kind: Kind::Bool,
    },
    Spec {
        name: "PID",
        kind: Kind::Int,
    },
    Spec {
        name: "Started",
        kind: Kind::Str,
    },
    Spec {
        name: "Execs",
        kind: Kind::StrList,
    },
];

fn decode_companion_record(data: &[u8]) -> Result<CompanionRecord, String> {
    let value = decode_strict(data).map_err(|e| e.0)?;
    let bound = bind_root(&value, "companion", COMPANION_SPECS, false).map_err(|e| e.0)?;
    Ok(CompanionRecord {
        id: bound.take_string("ID"),
        image: bound.take_string("Image"),
        command: bound.take_str_list("Command"),
        running: bound.take_bool("Running"),
        pid: bound.take_i64("PID"),
        started: bound.take_string("Started"),
        execs: bound.take_str_list("Execs"),
    })
}

/// Policy/enrollment surface the companion needs from the Tailnet control
/// plane. Mirrors `domain.Control.{Project,RunBinding,EnrollRun}`.
pub trait TailnetControl {
    fn project(
        &self,
        req: &ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<ProjectView, String>;
    fn run_binding(&self, target: &RunTarget, deadline: Instant) -> Result<RunBinding, String>;
    fn enroll_run(
        &self,
        target: &RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String>;
}

#[allow(clippy::type_complexity)]
pub struct Companion<E, T> {
    pub exec: E,
    pub tailnet: T,
    pub image: String,
    pub enabled_check: Option<Box<dyn Fn(&str, &str, Instant) -> Result<bool, String>>>,
}

/// Tags the actual failure stage with a fixed label. Native/provider output
/// never enters the chain, so the typed cause survives as a substring.
pub fn preparation_error(stage: &str, err: Option<String>) -> Option<String> {
    err.map(|cause| format!("{stage}: {cause}"))
}

/// Infallible [`preparation_error`] for call sites holding a definite cause.
fn stage_error(stage: &str, err: String) -> String {
    match preparation_error(stage, Some(err)) {
        Some(tagged) => tagged,
        None => unreachable!("preparation_error with a cause always tags"),
    }
}

fn arg_refs(args: &[String]) -> Vec<&str> {
    args.iter().map(String::as_str).collect()
}

pub fn companion_identity_matches(
    out: &CompanionRecord,
    _run: &ProjectRun,
    image: &str,
    id: &str,
) -> bool {
    valid_container_id(id)
        && out.id == id
        && out.image.trim_start_matches("sha256:") == image.trim_start_matches("sha256:")
}

pub fn companion_command_matches(cmd: &[String], args: &[String]) -> bool {
    cmd.len() == args.len() + 1
        && (cmd[0] == "/usr/bin/podman" || cmd[0] == "podman")
        && cmd[1..] == args[..]
}

pub fn companion_execs_valid(execs: &[String]) -> Result<(), String> {
    if execs.len() > 16 {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    for id in execs {
        if !valid_container_id(id) {
            return Err(ERR_UNAVAILABLE.to_string());
        }
    }
    Ok(())
}

pub fn validate_companion_record(
    out: &CompanionRecord,
    run: &ProjectRun,
    image: &str,
    id: &str,
) -> Result<(), String> {
    let args = companion_create_args(run, image)?;
    if !companion_identity_matches(out, run, image, id) {
        return Err(ERR_CONFLICT.to_string());
    }
    if out.running {
        match parse_rfc3339_nano(&out.started) {
            Some(false) => {}
            _ => return Err(ERR_CONFLICT.to_string()),
        }
        if out.pid <= 1 {
            return Err(ERR_CONFLICT.to_string());
        }
    }
    // CreateCommand is retained by Podman. Admit exactly our immutable recipe,
    // allowing only the native argv[0] spelling, never labels/name alone.
    if !companion_command_matches(&out.command, &args) {
        return Err(ERR_CONFLICT.to_string());
    }
    companion_execs_valid(&out.execs)
}

/// Live `/proc` identity reduced to the fields a companion incarnation must
/// match. Start-time and boot-id reads only gate validity, as in Go.
struct ProcessIdentity {
    userns: String,
    netns: String,
    uid: u32,
    gid: u32,
}

fn parse_stat_fields(pid: i64, data: &[u8]) -> Result<Vec<String>, String> {
    if data.len() > 8192 {
        return Err(ERR_CONFLICT.to_string());
    }
    let first = data
        .iter()
        .position(|&b| b == b' ')
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    let end = data
        .iter()
        .rposition(|&b| b == b')')
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    if data[..first] != *pid.to_string().as_bytes() {
        return Err(ERR_CONFLICT.to_string());
    }
    let rest = std::str::from_utf8(&data[end + 1..]).map_err(|_| ERR_CONFLICT.to_string())?;
    let fields: Vec<String> = rest.split_whitespace().map(str::to_string).collect();
    if fields.len() < 20 || fields[0] == "Z" || fields[0] == "X" {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(fields)
}

fn parse_process_start(fields: &[String]) -> Result<(), String> {
    match fields[19].parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == fields[19] => Ok(()),
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

fn read_process_id_map(
    base: &str,
    name: &str,
    read: &dyn Fn(&str) -> Result<Vec<u8>, String>,
) -> Result<u32, String> {
    let data = read(&format!("{base}/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    if data.len() > 4096 {
        return Err(ERR_CONFLICT.to_string());
    }
    let text = String::from_utf8_lossy(&data);
    let fields: Vec<&str> = text.split_whitespace().collect();
    if fields.len() != 3 {
        return Err(ERR_CONFLICT.to_string());
    }
    if fields[0] != "0" || fields[2] != "262144" {
        return Err(ERR_CONFLICT.to_string());
    }
    match fields[1].parse::<u32>() {
        Ok(n)
            if n != 0
                && n.to_string() == fields[1]
                && u64::from(n) + 262144 <= u64::from(u32::MAX) =>
        {
            Ok(n)
        }
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

fn read_process_namespace(
    base: &str,
    name: &str,
    link: &dyn Fn(&str) -> Result<String, String>,
) -> Result<String, String> {
    let value = link(&format!("{base}/ns/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    let prefix = format!("{name}:[");
    let inner = value
        .strip_prefix(&prefix)
        .and_then(|s| s.strip_suffix(']'))
        .ok_or_else(|| ERR_CONFLICT.to_string())?;
    match inner.parse::<u64>() {
        Ok(n) if n != 0 && n.to_string() == inner => {}
        _ => return Err(ERR_CONFLICT.to_string()),
    }
    let host = link(&format!("/proc/1/ns/{name}")).map_err(|_| ERR_CONFLICT.to_string())?;
    if host == value {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(value)
}

fn read_system_boot_id(read: &dyn Fn(&str) -> Result<Vec<u8>, String>) -> Result<(), String> {
    let boot = read("/proc/sys/kernel/random/boot_id").map_err(|_| ERR_CONFLICT.to_string())?;
    let text = String::from_utf8_lossy(&boot);
    let trimmed = text.trim();
    if trimmed.len() != 36 || trimmed.bytes().filter(|&b| b == b'-').count() != 4 {
        return Err(ERR_CONFLICT.to_string());
    }
    let compact: String = trimmed.chars().filter(|&c| c != '-').collect();
    if compact.len() != 32 || !compact.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

fn process_run_identity(
    pid: i64,
    read: &dyn Fn(&str) -> Result<Vec<u8>, String>,
    link: &dyn Fn(&str) -> Result<String, String>,
) -> Result<ProcessIdentity, String> {
    if pid <= 1 {
        return Err(ERR_CONFLICT.to_string());
    }
    let data = read(&format!("/proc/{pid}/stat")).map_err(|_| ERR_CONFLICT.to_string())?;
    let fields = parse_stat_fields(pid, &data)?;
    parse_process_start(&fields)?;
    let base = format!("/proc/{pid}");
    let uid = read_process_id_map(&base, "uid_map", read)?;
    let gid = read_process_id_map(&base, "gid_map", read)?;
    let userns = read_process_namespace(&base, "user", link)?;
    let netns = read_process_namespace(&base, "net", link)?;
    read_system_boot_id(read)?;
    Ok(ProcessIdentity {
        userns,
        netns,
        uid,
        gid,
    })
}

pub fn match_companion_namespaces(out: &CompanionRecord, run: &ProjectRun) -> Result<(), String> {
    if !out.running {
        return Ok(());
    }
    let read = |path: &str| std::fs::read(path).map_err(|_| ERR_CONFLICT.to_string());
    let link = |path: &str| {
        std::fs::read_link(path)
            .map(|p| p.to_string_lossy().into_owned())
            .map_err(|_| ERR_CONFLICT.to_string())
    };
    match process_run_identity(out.pid, &read, &link) {
        Ok(identity)
            if identity.userns == run.userns
                && identity.netns == run.netns
                && identity.uid == run.uid
                && identity.gid == run.gid =>
        {
            Ok(())
        }
        _ => Err(ERR_CONFLICT.to_string()),
    }
}

fn stat_metadata(path: &str) -> Result<std::fs::Metadata, String> {
    std::fs::metadata(path).map_err(|_| ERR_CONFLICT.to_string())
}

/// Enrollment/observations use the actual resolver inode, never Podman's
/// generated `ResolvConfPath` metadata.
fn companion_resolver(
    run: &ProjectRun,
    pid: i64,
    stat: &dyn Fn(&str) -> Result<std::fs::Metadata, String>,
) -> Result<(), String> {
    let original = stat(&run.resolver).map_err(|_| ERR_CONFLICT.to_string())?;
    let actual =
        stat(&format!("/proc/{pid}/root/etc/resolv.conf")).map_err(|_| ERR_CONFLICT.to_string())?;
    if !original.is_file() || original.dev() != actual.dev() || original.ino() != actual.ino() {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

fn companion_still_running(
    before: &CompanionRecord,
    after: Option<&CompanionRecord>,
    cli_ok: bool,
) -> bool {
    cli_ok
        && matches!(after, Some(a) if a.id == before.id && a.pid == before.pid && a.started == before.started && a.running)
}

fn is_companion_run_fresh(
    current_err: Option<&String>,
    previous: &ProjectRun,
    run: &ProjectRun,
) -> bool {
    match current_err {
        Some(cause) if cause == "runtime record not found" => true,
        None => previous.target.run != run.target.run,
        _ => false,
    }
}

fn apply_companion_idle_state(view: &mut ProjectView, running: bool) -> bool {
    if running {
        return view.enabled;
    }
    if !view.enabled {
        view.state = "off".to_string();
    }
    false // Off intent is not confirmed disconnection.
}

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    #[allow(dead_code)]
    fn podman(&self, stdin: &[u8], args: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        self.exec
            .run(stdin, "/usr/bin/podman", &arg_refs(args), deadline)
    }

    fn runtime_command(
        &self,
        cmd: &str,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        // The ordinary host executor predates secret-bearing native protocols.
        // Never collect its stderr for the companion, and never return native
        // error text.
        if self.exec.is_host_native() {
            return self.run_native_command(cmd, args, deadline);
        }
        match self.exec.run(&[], cmd, &arg_refs(args), deadline) {
            Ok(body) if body.len() <= 65536 => Ok(body),
            _ => Err(ERR_UNCONFIRMED.to_string()),
        }
    }

    fn runtime_podman(&self, args: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        let mut full = vec!["--remote=false".to_string()];
        full.extend(args.iter().cloned());
        self.runtime_command("/usr/bin/podman", &full, deadline)
    }

    /// Direct native execution with deadline kill. Stdout is capped at 65536
    /// bytes total; overflow, spawn failure, deadline kill and non-zero exit
    /// all collapse to unconfirmed without leaking native diagnostics.
    fn run_native_command(
        &self,
        cmd: &str,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        use std::io::Read;
        use std::process::Stdio;
        let mut child = std::process::Command::new(cmd)
            .args(args)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        let mut stdout = child.stdout.take();
        std::thread::scope(|scope| {
            let reader = scope.spawn(|| {
                let mut capped = Vec::new();
                let mut overflow = false;
                let mut chunk = [0u8; 8192];
                if let Some(out) = stdout.as_mut() {
                    loop {
                        match out.read(&mut chunk) {
                            Ok(0) => break,
                            Ok(n) => {
                                if capped.len() + n > 65536 {
                                    overflow = true;
                                } else {
                                    capped.extend_from_slice(&chunk[..n]);
                                }
                            }
                            Err(_) => {
                                overflow = true;
                                break;
                            }
                        }
                    }
                }
                (capped, overflow)
            });
            loop {
                match child.try_wait() {
                    Ok(Some(status)) => {
                        let (capped, overflow) =
                            reader.join().map_err(|_| ERR_UNCONFIRMED.to_string())?;
                        if !status.success() || overflow {
                            return Err(ERR_UNCONFIRMED.to_string());
                        }
                        return Ok(capped);
                    }
                    Ok(None) => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
                            let _ = reader.join();
                            return Err(ERR_UNCONFIRMED.to_string());
                        }
                        std::thread::sleep(Duration::from_millis(5));
                    }
                    Err(_) => {
                        let _ = reader.join();
                        return Err(ERR_UNCONFIRMED.to_string());
                    }
                }
            }
        })
    }

    fn inspect_companion(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<CompanionRecord, String> {
        let id = read_companion_id(RUNTIME_ROOT, run, 0, 0)?;
        let body = self
            .runtime_podman(
                &[
                    "inspect".to_string(),
                    "--format".to_string(),
                    COMPANION_INSPECT.to_string(),
                    id.clone(),
                ],
                deadline,
            )
            .map_err(|_| ERR_CONFLICT.to_string())?;
        let out = decode_companion_record(&body).map_err(|_| ERR_CONFLICT.to_string())?;
        validate_companion_record(&out, run, &self.image, &id)?;
        match_companion_namespaces(&out, run)?;
        Ok(out)
    }

    fn companion_cli(
        &self,
        run: &ProjectRun,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if args.is_empty() {
            return Err(ERR_INVALID.to_string());
        }
        let rec = match self.inspect_companion(run, deadline) {
            Ok(rec) if rec.running => rec,
            _ => return Err(ERR_UNAVAILABLE.to_string()),
        };
        if args[0] != "logout" {
            companion_resolver(run, rec.pid, &stat_metadata)?;
        }
        let mut full = vec![
            "exec".to_string(),
            "--user=0:0".to_string(),
            rec.id.clone(),
            "/usr/local/bin/tailscale".to_string(),
            "--socket=/run/tailscale/tailscaled.sock".to_string(),
        ];
        full.extend(args.iter().cloned());
        let output = self.runtime_podman(&full, deadline);
        let after = self.inspect_companion(run, deadline);
        if !companion_still_running(&rec, after.as_ref().ok(), output.is_ok()) {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        if args[0] != "logout" {
            let after = after.map_err(|_| ERR_UNCONFIRMED.to_string())?;
            companion_resolver(run, after.pid, &stat_metadata)
                .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        }
        output.map_err(|_| ERR_UNCONFIRMED.to_string())
    }

    /// Saved policy intent without requiring companion-specific runtime
    /// readiness. Missing policy means off; malformed state fails safely.
    fn project_tailnet_enabled(
        &self,
        project: &str,
        cid: &str,
        deadline: Instant,
    ) -> Result<bool, String> {
        if let Some(check) = &self.enabled_check {
            return check(project, cid, deadline);
        }
        let view = self.tailnet.project(
            &ProjectRequest {
                project: project.to_string(),
                action: "inspect".to_string(),
                revision: String::new(),
                binding: String::new(),
                confirm_id: String::new(),
            },
            cid,
            deadline,
        )?;
        Ok(view.enabled)
    }

    fn should_start_tailnet(&self, id: &str, deadline: Instant) -> Result<bool, String> {
        // `tailnet` is a generic bound, never nil; only the image gate applies.
        if self.image.is_empty() {
            return Ok(false);
        }
        // Off/unconfigured policy is a clean no-op before companion-specific
        // runtime readiness. An unresolvable container falls through so enabled
        // startup races still apply.
        if let Ok(cid) = project_container(&self.exec, id, false, deadline) {
            match self.project_tailnet_enabled(id, &cid, deadline) {
                Err(e) => return Err(stage_error("Tailnet policy unconfirmed", e)),
                Ok(false) => return Ok(false),
                Ok(true) => {}
            }
        }
        Ok(true)
    }

    fn wait_project_runtime(&self, id: &str, deadline: Instant) -> Result<ProjectRun, String> {
        let end = (Instant::now() + Duration::from_secs(10)).min(deadline);
        loop {
            match project_run(&self.exec, id, end) {
                Ok(run) => return Ok(run),
                Err(e) => {
                    if Instant::now() >= end {
                        return Err(stage_error("project runtime not ready", e));
                    }
                    std::thread::sleep(Duration::from_millis(100));
                }
            }
        }
    }

    fn admit_tailnet_run(
        &self,
        id: &str,
        deadline: Instant,
    ) -> Result<(ProjectRun, RunBinding), String> {
        let run = self.wait_project_runtime(id, deadline)?;
        let binding = self
            .tailnet
            .run_binding(&run.target, deadline)
            .map_err(|e| stage_error("Tailnet policy unconfirmed", e))?;
        Ok((run, binding))
    }

    fn retire_previous_companion(
        &self,
        id: &str,
        run: &ProjectRun,
        previous: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        if previous.target.run.is_empty() {
            return Ok(());
        }
        if previous.target.project != id || previous.target.container != run.target.container {
            return Err(stage_error(
                "project runtime changed",
                ERR_CONFLICT.to_string(),
            ));
        }
        self.stop_tailnet_run(previous, deadline)
            .map_err(|e| stage_error("companion stop unconfirmed", e))
    }

    fn reconcile_previous_run(
        &self,
        id: &str,
        run: &ProjectRun,
        previous: &ProjectRun,
        current_err: Option<String>,
        deadline: Instant,
    ) -> Result<bool, String> {
        if let Some(cause) = &current_err {
            if cause != "runtime record not found" {
                return Err(stage_error("companion runtime unconfirmed", cause.clone()));
            }
        }
        let fresh = is_companion_run_fresh(current_err.as_ref(), previous, run);
        if !fresh && previous != run {
            return Err(stage_error(
                "project runtime changed",
                ERR_CONFLICT.to_string(),
            ));
        }
        if fresh {
            self.retire_previous_companion(id, run, previous, deadline)?;
        }
        Ok(fresh)
    }

    fn prepare_companion_state(
        &self,
        id: &str,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(RunFiles, Root, bool), String> {
        let files = open_runtime_project(RUNTIME_ROOT, id, deadline)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        let (previous, current_err) = match files.current() {
            Ok(previous) => (previous, None),
            Err(e) => (ProjectRun::default(), Some(e)),
        };
        let fresh = self.reconcile_previous_run(id, run, &previous, current_err, deadline)?;
        let read = |path: &str| std::fs::read(path).map_err(|_| ERR_UNAVAILABLE.to_string());
        let stat = |path: &str| std::fs::metadata(path).map_err(|_| ERR_UNAVAILABLE.to_string());
        validate_run_resolver(run, Box::new(read), Box::new(stat), fresh)
            .map_err(|e| stage_error("project runtime not ready", e))?;
        let root = files
            .prepare(run, fresh)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok((files, root, fresh))
    }

    fn create_fresh_companion(
        &self,
        files: &RunFiles,
        root: &Root,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        files
            .save_current(run)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        recheck_project_run(&self.exec, run, deadline)
            .map_err(|e| stage_error("project runtime changed", e))?;
        let args = companion_create_args(run, &self.image)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        let created = self
            .runtime_command("/usr/bin/podman", &args, deadline)
            .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        let id = String::from_utf8_lossy(&created).trim().to_string();
        write_companion_id(root, &id)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok(())
    }

    fn start_companion_if_stopped(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        if !rec.running {
            recheck_project_run(&self.exec, run, deadline)
                .map_err(|e| stage_error("project runtime changed", e))?;
            self.runtime_podman(&["start".to_string(), rec.id.clone()], deadline)
                .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        }
        Ok(())
    }

    fn activate_companion_container(
        &self,
        files: &RunFiles,
        root: &Root,
        run: &ProjectRun,
        fresh: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        if fresh {
            self.create_fresh_companion(files, root, run, deadline)?;
        }
        self.start_companion_if_stopped(run, deadline)
    }

    fn wait_companion_node(&self, run: &ProjectRun, deadline: Instant) -> Result<bool, String> {
        let end = (Instant::now() + Duration::from_secs(8)).min(deadline);
        let args = vec![
            "status".to_string(),
            "--json".to_string(),
            "--peers=false".to_string(),
        ];
        loop {
            match self.companion_cli(run, &args, end) {
                Ok(body) => {
                    return project_has_node(&body)
                        .map_err(|e| stage_error("companion status unavailable", e));
                }
                Err(_) => {
                    if Instant::now() >= end {
                        return Err(stage_error(
                            "companion status unavailable",
                            ERR_UNAVAILABLE.to_string(),
                        ));
                    }
                    std::thread::sleep(Duration::from_millis(100));
                }
            }
        }
    }

    fn enroll_companion(
        &self,
        run: &ProjectRun,
        root: &Root,
        deadline: Instant,
    ) -> Result<(), String> {
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|_| stage_error("enrollment unconfirmed", ERR_UNCONFIRMED.to_string()))?;
        if !rec.execs.is_empty() {
            return Err(stage_error(
                "enrollment unconfirmed",
                ERR_UNCONFIRMED.to_string(),
            ));
        }
        retire_pending_run_key(root, run).map_err(|e| stage_error("enrollment unconfirmed", e))?;
        let recheck = |at: Instant| recheck_project_run(&self.exec, run, at);
        let consume = |at: Instant, key: &str| self.consume_run_key(run, root, key, at);
        self.tailnet
            .enroll_run(&run.target, &recheck, &consume, deadline)
            .map_err(|e| stage_error("enrollment unconfirmed", e))
    }

    fn ensure_companion_enrolled(
        &self,
        run: &ProjectRun,
        root: &Root,
        admission: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        let has_node = self.wait_companion_node(run, deadline)?;
        if has_node || !admission {
            return Ok(());
        }
        self.enroll_companion(run, root, deadline)
    }

    fn finalize_companion_run(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<String, String> {
        recheck_project_run(&self.exec, run, deadline)
            .map_err(|e| stage_error("project runtime changed", e))?;
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok(rec.id)
    }

    /// Prepares one exact run. Only systemd/native explicit activation calls
    /// it; HTTP reads and browser focus never enter this path.
    pub fn start_tailnet(&self, id: &str, deadline: Instant) -> Result<String, String> {
        let start = self.should_start_tailnet(id, deadline)?;
        if !start {
            return Ok(String::new());
        }
        let (run, binding) = self.admit_tailnet_run(id, deadline)?;
        if !binding.enabled {
            return Ok(String::new());
        }
        let (files, root, fresh) = self.prepare_companion_state(id, &run, deadline)?;
        self.activate_companion_container(&files, &root, &run, fresh, deadline)?;
        self.ensure_companion_enrolled(&run, &root, binding.admission, deadline)?;
        self.finalize_companion_run(&run, deadline)
    }

    /// Delegates lifetime to native Podman/systemd, not a reenrollment
    /// worker. Each daemon activation has its own in-memory ephemeral identity.
    pub fn wait_tailnet(&self, cid: &str, deadline: Instant) -> Result<(), String> {
        if cid.is_empty() {
            return Ok(());
        }
        if !valid_container_id(cid) {
            return Err(ERR_INVALID.to_string());
        }
        let output = self.runtime_podman(
            &[
                "wait".to_string(),
                "--condition=stopped".to_string(),
                cid.to_string(),
            ],
            deadline,
        );
        if Instant::now() >= deadline {
            return Ok(());
        }
        let body = output.map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if String::from_utf8_lossy(&body).trim() != "0" {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        // Unexpected successful daemon exit is also a supervision failure, not
        // a reason to silently leave an enabled project without its daemon.
        Err(ERR_UNAVAILABLE.to_string())
    }

    pub fn stop_tailnet(&self, id: &str, deadline: Instant) -> Result<(), String> {
        if !valid_project_id(id) {
            return Err(ERR_INVALID.to_string());
        }
        // The unit is conditioned on an existing runtime record; no scan.
        let files = open_runtime_project(RUNTIME_ROOT, id, deadline)?;
        let run = files.current()?;
        if run.target.project != id {
            return Err(ERR_CONFLICT.to_string());
        }
        self.stop_tailnet_run(&run, deadline)
    }

    fn confirm_stopped_resolver(&self, run: &ProjectRun, deadline: Instant) -> Result<(), String> {
        match project_run(&self.exec, &run.target.project, deadline) {
            Ok(current) if current == *run => {}
            _ => return Ok(()),
        }
        let data = std::fs::read(&run.resolver).map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if data.len() > 16384
            || String::from_utf8_lossy(&data)
                .to_lowercase()
                .contains("tailscale")
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        Ok(())
    }

    fn stop_tailnet_run(&self, run: &ProjectRun, deadline: Instant) -> Result<(), String> {
        let id = run.target.project.clone();
        match project_container(&self.exec, &id, false, deadline) {
            Ok(cid) if cid == run.target.container => {}
            _ => return Err(ERR_CONFLICT.to_string()),
        }
        let rec = self.inspect_companion(run, deadline)?;
        if !rec.running {
            return Ok(());
        }
        let result = self.logout_and_stop_companion(run, &rec.id, deadline);
        let after = self
            .inspect_companion(run, deadline)
            .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if after.id != rec.id || after.running {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        self.confirm_stopped_resolver(run, deadline)?;
        result
    }

    fn logout_and_stop_companion(
        &self,
        run: &ProjectRun,
        id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let logout_end = Instant::now() + Duration::from_secs(5);
        let logout = self.companion_cli(run, &["logout".to_string()], logout_end);
        let stop = self.runtime_podman(
            &["stop".to_string(), "--time=8".to_string(), id.to_string()],
            deadline,
        );
        if stop.is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        logout.map(|_| ()).map_err(|_| ERR_UNCONFIRMED.to_string())
    }

    fn consume_run_key(
        &self,
        run: &ProjectRun,
        root: &Root,
        key: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let file = write_run_key(root, run, key)?;
        let cli_err = self
            .companion_cli(
                run,
                &[
                    "up".to_string(),
                    "--json".to_string(),
                    "--timeout=5s".to_string(),
                    "--auth-key=file:/run/soda-enrollment/key".to_string(),
                    format!("--hostname=soda-{}", run.target.project),
                    "--accept-dns=true".to_string(),
                    "--accept-routes=false".to_string(),
                    "--ssh=false".to_string(),
                    "--advertise-exit-node=false".to_string(),
                ],
                deadline,
            )
            .err();
        // A cancelled Podman observer does not cancel the native exec. The CLI
        // has its own five-second deadline. Do not retire its input until all
        // execs have ended.
        let end = Instant::now() + Duration::from_secs(8);
        loop {
            if let Ok(rec) = self.inspect_companion(run, end) {
                if rec.execs.is_empty() {
                    retire_run_key(root, &file).map_err(|_| ERR_UNCONFIRMED.to_string())?;
                    return cli_err.map_or(Ok(()), Err);
                }
            }
            if Instant::now() >= end {
                return Err(ERR_UNCONFIRMED.to_string());
            }
            std::thread::sleep(Duration::from_millis(100));
        }
    }

    fn queue_project_tailnet_disable(
        &self,
        project: &str,
        view: &mut ProjectView,
        deadline: Instant,
    ) {
        // Queue cancellation first. Also handle an owned orphan whose unit is
        // already inactive: stopping an inactive systemd unit does not execute
        // ExecStop.
        let args = vec![
            "stop".to_string(),
            "--no-block".to_string(),
            format!("soda-tailnet@{project}.service"),
        ];
        if self
            .runtime_command("/usr/bin/systemctl", &args, deadline)
            .is_ok()
        {
            view.outcome = "queued".to_string();
        }
        let end = Instant::now() + Duration::from_secs(15);
        if self.stop_tailnet(project, end).is_err() {
            view.outcome = "runtime-unconfirmed".to_string();
        }
    }

    fn mark_stopped_project(&self, project: &str, view: &mut ProjectView, deadline: Instant) {
        // Distinguish a confirmed stopped parent from failed observation.
        if let Ok(false) = project_running(&self.exec, project, deadline) {
            view.state = "stopped".to_string();
        }
    }

    fn queue_project_tailnet_start(
        &self,
        req: &ProjectRequest,
        view: &mut ProjectView,
        deadline: Instant,
    ) -> bool {
        if req.action == "inspect" || !view.enabled {
            return true;
        }
        let args = vec![
            "start".to_string(),
            "--no-block".to_string(),
            format!("soda-tailnet@{}.service", req.project),
        ];
        if self
            .runtime_command("/usr/bin/systemctl", &args, deadline)
            .is_err()
        {
            return false;
        }
        view.outcome = "queued".to_string();
        true
    }

    fn observe_companion_status(
        &self,
        run: &ProjectRun,
        view: &mut ProjectView,
        deadline: Instant,
    ) {
        let binding = match self.tailnet.run_binding(&run.target, deadline) {
            Ok(binding) => binding,
            Err(_) => return,
        };
        let status = match self.companion_cli(
            run,
            &[
                "status".to_string(),
                "--json".to_string(),
                "--peers=false".to_string(),
            ],
            deadline,
        ) {
            Ok(body) => body,
            Err(_) => return,
        };
        let prefs =
            match self.companion_cli(run, &["debug".to_string(), "prefs".to_string()], deadline) {
                Ok(body) => body,
                Err(_) => return,
            };
        let (state, addresses, dns) = match project_status(&status, &prefs, &binding) {
            Ok(triple) => triple,
            Err(_) => return,
        };
        if recheck_project_run(&self.exec, run, deadline).is_err() {
            return;
        }
        view.state = state;
        view.addresses = addresses;
        view.dns_name = dns;
    }

    pub fn observe_project_tailnet(
        &self,
        req: &ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<ProjectView, String> {
        let mut view = self.tailnet.project(req, cid, deadline)?;
        if self.image.is_empty() {
            return Ok(view);
        }
        if req.action == "disable" {
            self.queue_project_tailnet_disable(&req.project, &mut view, deadline);
        }
        let run = match project_run(&self.exec, &req.project, deadline) {
            Ok(run) => run,
            Err(_) => {
                self.mark_stopped_project(&req.project, &mut view, deadline);
                return Ok(view);
            }
        };
        if !self.queue_project_tailnet_start(req, &mut view, deadline) {
            return Ok(view);
        }
        let rec = match self.inspect_companion(&run, deadline) {
            Ok(rec) => rec,
            Err(_) => return Ok(view),
        };
        if !apply_companion_idle_state(&mut view, rec.running) {
            return Ok(view);
        }
        self.observe_companion_status(&run, &mut view, deadline);
        Ok(view)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

    fn temp_dir(name: &str) -> std::path::PathBuf {
        let id = TEMP_COUNTER.fetch_add(1, Ordering::SeqCst);
        let dir =
            std::env::temp_dir().join(format!("soda-laneD-{name}-{}-{id}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn project_id() -> String {
        format!("p{}", "a".repeat(24))
    }

    /// Mirrors Go's `fileRun` fixture.
    fn file_run() -> ProjectRun {
        ProjectRun {
            target: RunTarget {
                project: project_id(),
                container: "b".repeat(64),
                run: "c".repeat(64),
            },
            pid: 77,
            started: String::new(),
            userns: "user:[4026531837]".to_string(),
            netns: "net:[4026531840]".to_string(),
            uid: 300000,
            gid: 300000,
            resolver: String::new(),
        }
    }

    fn test_view() -> ProjectView {
        ProjectView {
            available_binding: String::new(),
            available_network: String::new(),
            addresses: Vec::new(),
            dns_name: String::new(),
            saved: false,
            project: project_id(),
            revision: String::new(),
            binding: String::new(),
            enabled: false,
            state: String::new(),
            outcome: String::new(),
        }
    }

    fn test_binding() -> RunBinding {
        RunBinding {
            enabled: true,
            admission: false,
            tailnet: "tail-abc".to_string(),
            tags: vec!["tag:soda".to_string()],
        }
    }

    fn test_request(action: &str) -> ProjectRequest {
        ProjectRequest {
            project: project_id(),
            action: action.to_string(),
            revision: String::new(),
            binding: String::new(),
            confirm_id: String::new(),
        }
    }

    fn project_inspect_json(id: &str, cid: &str, running: bool) -> Vec<u8> {
        format!(
            "{{\"id\":\"{cid}\",\"running\":{running},\"project\":\"{id}\",\"owner\":\"1000\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:100000:262144\"],\"GidMap\":[\"0:100000:262144\"]}}}}"
        )
        .into_bytes()
    }

    #[allow(clippy::type_complexity)]
    struct MockExec {
        calls: RefCell<Vec<(String, Vec<String>)>>,
        handler: Box<dyn Fn(&str, &[String]) -> Result<Vec<u8>, String>>,
        native: bool,
    }

    impl MockExec {
        fn with_handler(
            handler: impl Fn(&str, &[String]) -> Result<Vec<u8>, String> + 'static,
        ) -> Self {
            MockExec {
                calls: RefCell::new(Vec::new()),
                handler: Box::new(handler),
                native: false,
            }
        }

        fn calls(&self) -> Vec<(String, Vec<String>)> {
            self.calls.borrow().clone()
        }
    }

    impl Executor for MockExec {
        fn run(
            &self,
            _stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            let owned: Vec<String> = args.iter().map(|s| s.to_string()).collect();
            self.calls
                .borrow_mut()
                .push((cmd.to_string(), owned.clone()));
            (self.handler)(cmd, &owned)
        }

        fn is_host_native(&self) -> bool {
            self.native
        }
    }

    #[allow(clippy::type_complexity)]
    struct MockTailnet {
        project_fn: Box<dyn Fn(&ProjectRequest, &str) -> Result<ProjectView, String>>,
        binding_fn: Box<dyn Fn(&RunTarget) -> Result<RunBinding, String>>,
        enroll_targets: RefCell<Vec<String>>,
        enroll_err: Option<String>,
    }

    impl MockTailnet {
        fn inert() -> Self {
            MockTailnet {
                project_fn: Box::new(|_, _| panic!("unexpected tailnet.project call")),
                binding_fn: Box::new(|_| panic!("unexpected tailnet.run_binding call")),
                enroll_targets: RefCell::new(Vec::new()),
                enroll_err: None,
            }
        }
    }

    impl TailnetControl for MockTailnet {
        fn project(
            &self,
            req: &ProjectRequest,
            cid: &str,
            _deadline: Instant,
        ) -> Result<ProjectView, String> {
            (self.project_fn)(req, cid)
        }

        fn run_binding(
            &self,
            target: &RunTarget,
            _deadline: Instant,
        ) -> Result<RunBinding, String> {
            (self.binding_fn)(target)
        }

        fn enroll_run(
            &self,
            target: &RunTarget,
            _recheck: &dyn Fn(Instant) -> Result<(), String>,
            _consume: &dyn Fn(Instant, &str) -> Result<(), String>,
            _deadline: Instant,
        ) -> Result<(), String> {
            self.enroll_targets.borrow_mut().push(target.run.clone());
            match &self.enroll_err {
                Some(e) => Err(e.clone()),
                None => Ok(()),
            }
        }
    }

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    fn companion_with(
        exec: MockExec,
        tailnet: MockTailnet,
        image: &str,
    ) -> Companion<MockExec, MockTailnet> {
        Companion {
            exec,
            tailnet,
            image: image.to_string(),
            enabled_check: None,
        }
    }

    fn image_id() -> String {
        format!("sha256:{}", "d".repeat(64))
    }

    #[test]
    fn resolver_uses_actual_inode_rather_than_generated_metadata() {
        let dir = temp_dir("resolver");
        let owned = dir.join("resolver");
        let other = dir.join("other");
        std::fs::write(&owned, b"nameserver 10.89.0.1\n").unwrap();
        std::fs::write(&other, b"nameserver 10.89.0.1\n").unwrap();
        let mut run = file_run();
        run.resolver = owned.to_str().unwrap().to_string();
        for same in [true, false] {
            let stat = |path: &str| -> Result<std::fs::Metadata, String> {
                let mapped = if path == "/proc/99/root/etc/resolv.conf" {
                    if same {
                        &owned
                    } else {
                        &other
                    }
                } else {
                    std::path::Path::new(path)
                };
                std::fs::metadata(mapped).map_err(|e| e.to_string())
            };
            let result = companion_resolver(&run, 99, &stat);
            assert_eq!(result.is_ok(), same, "resolver identity same={same}");
        }
        // A missing owned resolver never matches.
        run.resolver = dir.join("absent").to_str().unwrap().to_string();
        let stat = |path: &str| std::fs::metadata(path).map_err(|_| ERR_CONFLICT.to_string());
        assert_eq!(
            companion_resolver(&run, 99, &stat),
            Err(ERR_CONFLICT.to_string())
        );
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn record_requires_immutable_cid_recipe_and_running_incarnation() {
        let mut run = file_run();
        run.resolver = format!(
            "/var/lib/containers/storage/overlay-containers/{}/userdata/resolv.conf",
            run.target.container
        );
        let image = image_id();
        let id = "e".repeat(64);
        let args = companion_create_args(&run, &image).unwrap();
        let mut command = vec!["/usr/bin/podman".to_string()];
        command.extend(args);
        let original = CompanionRecord {
            id: id.clone(),
            image: image.clone(),
            command,
            running: true,
            pid: 99,
            started: "2026-09-12T12:00:00Z".to_string(),
            execs: Vec::new(),
        };
        validate_companion_record(&original, &run, &image, &id).unwrap();
        // A new release default is not authority to adopt an existing run's
        // companion.
        assert!(validate_companion_record(
            &original,
            &run,
            &format!("sha256:{}", "f".repeat(64)),
            &id
        )
        .is_err());
        // Before first native start there is no daemon PID.
        let created = CompanionRecord {
            running: false,
            pid: 0,
            started: String::new(),
            ..original.clone()
        };
        validate_companion_record(&created, &run, &image, &id).unwrap();
        // The native argv[0] spelling is admitted.
        let native_spelling = CompanionRecord {
            command: {
                let mut cmd = vec!["podman".to_string()];
                cmd.extend(original.command[1..].iter().cloned());
                cmd
            },
            ..original.clone()
        };
        validate_companion_record(&native_spelling, &run, &image, &id).unwrap();
        // The digest prefix spelling is immaterial.
        let bare_image = CompanionRecord {
            image: "d".repeat(64),
            ..original.clone()
        };
        validate_companion_record(&bare_image, &run, &image, &id).unwrap();
        for kind in [
            "cid",
            "image",
            "namespace",
            "secret-env",
            "missing-command",
            "foreign-exec",
            "many-execs",
            "pid",
            "started",
            "started-text",
        ] {
            let mut changed = original.clone();
            match kind {
                "cid" => changed.id = "f".repeat(64),
                "image" => changed.image = format!("sha256:{}", "f".repeat(64)),
                "namespace" => {
                    for arg in &mut changed.command {
                        if arg.starts_with("--network=") {
                            *arg = "--network=host".to_string();
                        }
                    }
                }
                "secret-env" => changed
                    .command
                    .push("--env=TS_AUTHKEY=synthetic-secret".to_string()),
                "missing-command" => changed.command.clear(),
                "foreign-exec" => changed.execs = vec!["unknown".to_string()],
                "many-execs" => changed.execs = vec![id.clone(); 17],
                "pid" => changed.pid = 1,
                "started" => changed.started = "0001-01-01T00:00:00Z".to_string(),
                "started-text" => changed.started = "not-a-time".to_string(),
                _ => unreachable!(),
            }
            assert!(
                validate_companion_record(&changed, &run, &image, &id).is_err(),
                "changed companion admitted: {kind}"
            );
        }
    }

    #[test]
    fn exec_identity_and_command_matchers() {
        let run = file_run();
        let id = "e".repeat(64);
        let rec = CompanionRecord {
            id: id.clone(),
            image: image_id(),
            ..CompanionRecord::default()
        };
        assert!(companion_identity_matches(&rec, &run, &image_id(), &id));
        assert!(!companion_identity_matches(
            &rec,
            &run,
            &image_id(),
            &"f".repeat(64)
        ));
        assert!(!companion_identity_matches(
            &rec,
            &run,
            &format!("sha256:{}", "f".repeat(64)),
            &id
        ));
        assert!(!companion_identity_matches(
            &rec,
            &run,
            &image_id(),
            "short"
        ));
        let args = vec!["a".to_string(), "b".to_string()];
        assert!(companion_command_matches(
            &[
                "/usr/bin/podman".to_string(),
                "a".to_string(),
                "b".to_string()
            ],
            &args
        ));
        assert!(companion_command_matches(
            &["podman".to_string(), "a".to_string(), "b".to_string()],
            &args
        ));
        assert!(!companion_command_matches(&["a".to_string()], &args));
        assert!(!companion_command_matches(&Vec::new(), &args));
        assert!(!companion_command_matches(
            &["/bin/podman".to_string(), "a".to_string(), "b".to_string()],
            &args
        ));
        assert!(companion_execs_valid(&[]).is_ok());
        assert!(companion_execs_valid(std::slice::from_ref(&id)).is_ok());
        assert_eq!(
            companion_execs_valid(&["unknown".to_string()]),
            Err(ERR_UNAVAILABLE.to_string())
        );
        assert_eq!(
            companion_execs_valid(&vec![id.clone(); 17]),
            Err(ERR_UNAVAILABLE.to_string())
        );
    }

    #[test]
    fn inspect_decode_is_strict() {
        let good = br#"{"id":"abc","image":"sha256:00","command":["podman","a"],"running":true,"pid":99,"started":"2026-09-12T12:00:00Z","execs":[]}"#;
        let rec = decode_companion_record(good).unwrap();
        assert_eq!(rec.id, "abc");
        assert_eq!(rec.pid, 99);
        assert!(rec.running);
        assert_eq!(rec.command, vec!["podman".to_string(), "a".to_string()]);
        // Nulls bind zero values like encoding/json.
        let nulls = br#"{"id":null,"image":null,"command":null,"running":null,"pid":null,"started":null,"execs":null}"#;
        let rec = decode_companion_record(nulls).unwrap();
        assert_eq!(rec, CompanionRecord::default());
        for bad in [
            "not json",
            "[]",
            "{\"id\":\"a\",\"id\":\"b\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[]}",
            "{\"id\":\"a\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[],\"extra\":1}",
            "{\"id\":\"a\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":1.5,\"started\":\"\",\"execs\":[]}",
            "{\"id\":\"a\",\"image\":\"i\",\"command\":{},\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[]}",
        ] {
            assert!(
                decode_companion_record(bad.as_bytes()).is_err(),
                "decoded: {bad}"
            );
        }
    }

    #[test]
    fn preparation_stages_preserve_typed_causes() {
        for (stage, cause) in [
            ("project runtime not ready", ERR_UNAVAILABLE),
            ("Tailnet policy unconfirmed", ERR_CONFLICT),
            ("companion startup unconfirmed", ERR_UNCONFIRMED),
            ("companion status unavailable", ERR_UNAVAILABLE),
            ("enrollment unconfirmed", ERR_UNCONFIRMED),
        ] {
            let tagged = preparation_error(stage, Some(cause.to_string())).expect("tagged");
            assert!(tagged.contains(stage), "stage lost: {tagged}");
            assert!(tagged.contains(cause), "cause lost: {tagged}");
        }
        assert_eq!(preparation_error("stage", None), None);
    }

    #[test]
    fn run_freshness_and_idle_state() {
        let run = file_run();
        let mut other = run.clone();
        other.target.run = "d".repeat(64);
        assert!(!is_companion_run_fresh(None, &run, &run));
        assert!(is_companion_run_fresh(None, &other, &run));
        let missing = "runtime record not found".to_string();
        assert!(is_companion_run_fresh(Some(&missing), &run, &run));
        let other_err = ERR_UNAVAILABLE.to_string();
        assert!(!is_companion_run_fresh(Some(&other_err), &other, &run));

        let mut view = test_view();
        view.enabled = true;
        assert!(apply_companion_idle_state(&mut view, true));
        assert_eq!(view.state, "");
        assert!(!apply_companion_idle_state(&mut view, false));
        assert_eq!(view.state, "");
        view.enabled = false;
        assert!(!apply_companion_idle_state(&mut view, true));
        assert!(!apply_companion_idle_state(&mut view, false));
        assert_eq!(view.state, "off");
    }

    #[test]
    fn namespaces_match_only_exact_incarnation() {
        let run = file_run();
        let stopped = CompanionRecord {
            running: false,
            ..CompanionRecord::default()
        };
        assert!(match_companion_namespaces(&stopped, &run).is_ok());
        // PID 1 or below can never be a companion incarnation.
        let init = CompanionRecord {
            running: true,
            pid: 1,
            ..CompanionRecord::default()
        };
        assert_eq!(
            match_companion_namespaces(&init, &run),
            Err(ERR_CONFLICT.to_string())
        );
        // An absent process never matches.
        let absent = CompanionRecord {
            running: true,
            pid: 4242424242,
            ..CompanionRecord::default()
        };
        assert_eq!(
            match_companion_namespaces(&absent, &run),
            Err(ERR_CONFLICT.to_string())
        );
        // A live process with synthetic namespaces never matches either.
        let live = CompanionRecord {
            running: true,
            pid: std::process::id() as i64,
            ..CompanionRecord::default()
        };
        assert_eq!(
            match_companion_namespaces(&live, &run),
            Err(ERR_CONFLICT.to_string())
        );
    }

    #[test]
    fn wait_tailnet_output_matrix() {
        // Empty CID is a no-op without any call.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert!(c.wait_tailnet("", deadline()).is_ok());
        assert!(c.exec.calls().is_empty());
        // Malformed CIDs are rejected before any call.
        assert_eq!(
            c.wait_tailnet("short", deadline()),
            Err(ERR_INVALID.to_string())
        );
        assert!(c.exec.calls().is_empty());

        for (output, want) in [
            (Ok(b"0\n".to_vec()), Err(ERR_UNAVAILABLE.to_string())),
            (Ok(b"0".to_vec()), Err(ERR_UNAVAILABLE.to_string())),
            (Ok(b"1\n".to_vec()), Err(ERR_UNCONFIRMED.to_string())),
            (Ok(b"stopped\n".to_vec()), Err(ERR_UNCONFIRMED.to_string())),
            (Ok(vec![b'x'; 65537]), Err(ERR_UNCONFIRMED.to_string())),
            (Err("boom".to_string()), Err(ERR_UNCONFIRMED.to_string())),
        ] {
            let exec = MockExec::with_handler(move |cmd, args| {
                assert_eq!(cmd, "/usr/bin/podman");
                assert_eq!(args[0], "--remote=false");
                assert_eq!(args[1], "wait");
                assert_eq!(args[2], "--condition=stopped");
                output.clone()
            });
            let c = companion_with(exec, MockTailnet::inert(), &image_id());
            assert_eq!(c.wait_tailnet(&"e".repeat(64), deadline()), want);
        }
        // An expired deadline reports success like Go's ctx.Err check, even
        // when the supervisor output disagrees.
        let exec = MockExec::with_handler(|_, _| Ok(b"1\n".to_vec()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert!(c.wait_tailnet(&"e".repeat(64), Instant::now()).is_ok());
    }

    #[test]
    fn native_commands_never_leak_diagnostics() {
        let exec = MockExec {
            calls: RefCell::new(Vec::new()),
            handler: Box::new(|_, _| panic!("native path must not use the executor")),
            native: true,
        };
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let out = c
            .run_native_command("/bin/echo", &["hello".to_string()], deadline())
            .unwrap();
        assert_eq!(out, b"hello\n");
        assert_eq!(
            c.run_native_command("/bin/false", &[], deadline()),
            Err(ERR_UNCONFIRMED.to_string())
        );
        assert_eq!(
            c.run_native_command("/nonexistent-soda-binary", &[], deadline()),
            Err(ERR_UNCONFIRMED.to_string())
        );
        // Exactly at the cap passes; one byte over does not.
        let out = c
            .run_native_command(
                "/bin/sh",
                &["-c".to_string(), "head -c 65536 /dev/zero".to_string()],
                deadline(),
            )
            .unwrap();
        assert_eq!(out.len(), 65536);
        assert_eq!(
            c.run_native_command(
                "/bin/sh",
                &["-c".to_string(), "head -c 70000 /dev/zero".to_string()],
                deadline(),
            ),
            Err(ERR_UNCONFIRMED.to_string())
        );
        // Deadline kill collapses to unconfirmed.
        assert_eq!(
            c.run_native_command(
                "/bin/sleep",
                &["30".to_string()],
                Instant::now() + Duration::from_millis(100)
            ),
            Err(ERR_UNCONFIRMED.to_string())
        );
    }

    #[test]
    fn start_skipped_when_image_empty() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected exec call"));
        let c = companion_with(exec, MockTailnet::inert(), "");
        assert_eq!(
            c.start_tailnet(&project_id(), deadline()),
            Ok(String::new())
        );
        assert!(c.exec.calls().is_empty());
        assert_eq!(c.should_start_tailnet(&project_id(), deadline()), Ok(false));
    }

    #[test]
    fn should_start_tailnet_policy_gates() {
        // An unresolvable container falls through to startup.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(c.should_start_tailnet(&project_id(), deadline()), Ok(true));

        let id = project_id();
        let cid = "b".repeat(64);
        for enabled in [false, true] {
            let id_clone = id.clone();
            let cid_clone = cid.clone();
            let exec = MockExec::with_handler(move |cmd, args| {
                assert_eq!(cmd, "/usr/bin/podman");
                assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
                Ok(project_inspect_json(&id_clone, &cid_clone, true))
            });
            let mut c = companion_with(exec, MockTailnet::inert(), &image_id());
            let want_id = id.clone();
            let want_cid = cid.clone();
            c.enabled_check = Some(Box::new(move |project, seen_cid, _| {
                assert_eq!(project, want_id);
                assert_eq!(seen_cid, want_cid);
                Ok(enabled)
            }));
            assert_eq!(
                c.should_start_tailnet(&project_id(), deadline()),
                Ok(enabled)
            );
        }
        // Without an override the policy service decides.
        let id_clone = id.clone();
        let cid_clone = cid.clone();
        let exec = MockExec::with_handler(move |_, args| {
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &cid_clone, true))
        });
        let tailnet = MockTailnet {
            project_fn: Box::new(|req, _| {
                assert_eq!(req.action, "inspect");
                let mut view = test_view();
                view.enabled = true;
                Ok(view)
            }),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        assert_eq!(c.project_tailnet_enabled(&id, &cid, deadline()), Ok(true));
        assert_eq!(c.should_start_tailnet(&id, deadline()), Ok(true));
    }

    #[test]
    fn companion_cli_rejects_empty_args_and_missing_state() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let run = file_run();
        assert_eq!(
            c.companion_cli(&run, &[], deadline()),
            Err(ERR_INVALID.to_string())
        );
        // No runtime record exists in the test environment, so the CLI is
        // unavailable before any supervisor call.
        assert_eq!(
            c.companion_cli(&run, &["status".to_string()], deadline()),
            Err(ERR_UNAVAILABLE.to_string())
        );
        assert!(c.exec.calls().is_empty());
    }

    #[test]
    fn logout_and_stop_reports_unconfirmed_logout() {
        // Logout fails without runtime state; the stop argv is still exact.
        let exec = MockExec::with_handler(|cmd, args| {
            assert_eq!(cmd, "/usr/bin/podman");
            assert_eq!(
                args,
                &[
                    "--remote=false".to_string(),
                    "stop".to_string(),
                    "--time=8".to_string(),
                    "e".repeat(64),
                ]
            );
            Ok(Vec::new())
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(
            c.logout_and_stop_companion(&file_run(), &"e".repeat(64), deadline()),
            Err(ERR_UNCONFIRMED.to_string())
        );
        let exec = MockExec::with_handler(|_, _| Err("stop failed".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(
            c.logout_and_stop_companion(&file_run(), &"e".repeat(64), deadline()),
            Err(ERR_UNCONFIRMED.to_string())
        );
    }

    #[test]
    fn retire_previous_companion_guards_identity() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let run = file_run();
        let empty = ProjectRun {
            target: RunTarget {
                project: String::new(),
                container: String::new(),
                run: String::new(),
            },
            pid: 0,
            started: String::new(),
            userns: String::new(),
            netns: String::new(),
            uid: 0,
            gid: 0,
            resolver: String::new(),
        };
        assert!(c
            .retire_previous_companion(&project_id(), &run, &empty, deadline())
            .is_ok());
        assert!(c.exec.calls().is_empty());
        let mut foreign = run.clone();
        foreign.target.project = format!("p{}", "f".repeat(24));
        let err = c
            .retire_previous_companion(&project_id(), &run, &foreign, deadline())
            .unwrap_err();
        assert!(err.starts_with("project runtime changed: "), "{err}");
        assert!(err.contains(ERR_CONFLICT), "{err}");
        // A live mismatch against the supervisor is a bare conflict.
        let exec = MockExec::with_handler(|_, _| {
            Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(
            c.retire_previous_companion(&project_id(), &run, &run, deadline())
                .unwrap_err(),
            stage_error("companion stop unconfirmed", ERR_CONFLICT.to_string())
        );
    }

    #[test]
    fn reconcile_previous_run_freshness() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let run = file_run();
        // A missing record with no previous run is fresh.
        let empty = ProjectRun {
            target: RunTarget {
                project: String::new(),
                container: String::new(),
                run: String::new(),
            },
            pid: 0,
            started: String::new(),
            userns: String::new(),
            netns: String::new(),
            uid: 0,
            gid: 0,
            resolver: String::new(),
        };
        assert_eq!(
            c.reconcile_previous_run(
                &project_id(),
                &run,
                &empty,
                Some("runtime record not found".to_string()),
                deadline()
            ),
            Ok(true)
        );
        // Any other current-record failure is unconfirmed runtime.
        let err = c
            .reconcile_previous_run(
                &project_id(),
                &run,
                &empty,
                Some(ERR_UNAVAILABLE.to_string()),
                deadline(),
            )
            .unwrap_err();
        assert!(err.starts_with("companion runtime unconfirmed: "), "{err}");
        // A present record for the same run with drifted fields is a change.
        let mut drifted = run.clone();
        drifted.pid = 78;
        let err = c
            .reconcile_previous_run(&project_id(), &run, &drifted, None, deadline())
            .unwrap_err();
        assert!(err.starts_with("project runtime changed: "), "{err}");
        assert_eq!(
            c.reconcile_previous_run(&project_id(), &run, &run, None, deadline()),
            Ok(false)
        );
        // A present record for another run retires the previous companion.
        let mut other = run.clone();
        other.target.run = "d".repeat(64);
        let exec = MockExec::with_handler(|_, _| {
            Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let err = c
            .reconcile_previous_run(&project_id(), &run, &other, None, deadline())
            .unwrap_err();
        assert!(err.starts_with("companion stop unconfirmed: "), "{err}");
    }

    #[test]
    fn prepare_and_finalize_stage_fast_failures() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let err = c
            .prepare_companion_state(&project_id(), &file_run(), deadline())
            .expect_err("prepare fails without runtime state");
        assert!(err.starts_with("companion runtime unconfirmed: "), "{err}");
        let err = c
            .start_companion_if_stopped(&file_run(), deadline())
            .unwrap_err();
        assert!(err.starts_with("companion startup unconfirmed: "), "{err}");

        // Finalize rechecks first; an unresolvable parent fails fast.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let err = c
            .finalize_companion_run(&file_run(), deadline())
            .unwrap_err();
        assert!(err.starts_with("project runtime changed: "), "{err}");
    }

    #[test]
    fn stop_tailnet_validates_and_needs_state() {
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(
            c.stop_tailnet("bad-id", deadline()),
            Err(ERR_INVALID.to_string())
        );
        assert!(c.exec.calls().is_empty());
        // No runtime record exists in the test environment.
        assert!(c.stop_tailnet(&project_id(), deadline()).is_err());
        // A supervisor mismatch against the recorded container conflicts.
        let exec = MockExec::with_handler(|_, _| {
            Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(
            c.stop_tailnet_run(&file_run(), deadline()),
            Err(ERR_CONFLICT.to_string())
        );
    }

    #[test]
    fn disable_flow_marks_runtime_unconfirmed() {
        let id = project_id();
        let id_clone = id.clone();
        let exec = MockExec::with_handler(move |cmd, args| {
            if cmd == "/usr/bin/systemctl" {
                return Ok(Vec::new());
            }
            assert_eq!(cmd, "/usr/bin/podman");
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &"b".repeat(64), false))
        });
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| Ok(test_view())),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let view = c
            .observe_project_tailnet(&test_request("disable"), &"c".repeat(64), deadline())
            .unwrap();
        // systemctl accepted the stop, then the missing runtime record
        // overwrote the outcome.
        assert_eq!(view.outcome, "runtime-unconfirmed");
        // The parent container is confirmed stopped.
        assert_eq!(view.state, "stopped");
        let calls = c.exec.calls();
        assert_eq!(
            calls[0],
            (
                "/usr/bin/systemctl".to_string(),
                vec![
                    "stop".to_string(),
                    "--no-block".to_string(),
                    format!("soda-tailnet@{id}.service"),
                ]
            )
        );
        // Queueing the disable directly behaves the same.
        let mut direct = test_view();
        c.queue_project_tailnet_disable(&id, &mut direct, deadline());
        assert_eq!(direct.outcome, "runtime-unconfirmed");
    }

    #[test]
    fn queue_start_gates_on_action_and_systemd() {
        let id = project_id();
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_eq!(cmd, "/usr/bin/systemctl");
            assert_eq!(
                args,
                &[
                    "start".to_string(),
                    "--no-block".to_string(),
                    format!("soda-tailnet@{id}.service"),
                ]
            );
            Ok(Vec::new())
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        assert!(c.queue_project_tailnet_start(&test_request("inspect"), &mut view, deadline()));
        assert_eq!(view.outcome, "");
        view.enabled = true;
        assert!(c.queue_project_tailnet_start(&test_request("enable"), &mut view, deadline()));
        assert_eq!(view.outcome, "queued");

        let exec = MockExec::with_handler(|_, _| Err("systemd down".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        view.enabled = true;
        assert!(!c.queue_project_tailnet_start(&test_request("enable"), &mut view, deadline()));
        assert_eq!(view.outcome, "");
    }

    #[test]
    fn mark_stopped_distinguishes_confirmed_stop() {
        let id = project_id();
        for (running, want) in [(false, "stopped"), (true, "")] {
            let id_clone = id.clone();
            let exec = MockExec::with_handler(move |_, _| {
                Ok(project_inspect_json(&id_clone, &"b".repeat(64), running))
            });
            let c = companion_with(exec, MockTailnet::inert(), &image_id());
            let mut view = test_view();
            c.mark_stopped_project(&id, &mut view, deadline());
            assert_eq!(view.state, want);
        }
        // Failed observation leaves the state untouched.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        c.mark_stopped_project(&id, &mut view, deadline());
        assert_eq!(view.state, "");
    }

    #[test]
    fn observe_passthrough_and_fast_paths() {
        // Policy errors propagate with the caller's view dropped, as in Go.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| Err("policy down".to_string())),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        assert_eq!(
            c.observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline()),
            Err("policy down".to_string())
        );
        // An empty image passes the policy view through untouched.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| {
                let mut view = test_view();
                view.enabled = true;
                view.state = "policy-state".to_string();
                Ok(view)
            }),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, "");
        let view = c
            .observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline())
            .unwrap();
        assert_eq!(view.state, "policy-state");
        assert!(c.exec.calls().is_empty());
        // An unresolvable parent marks a confirmed stop without queueing.
        let id = project_id();
        let id_clone = id.clone();
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_ne!(cmd, "/usr/bin/systemctl");
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &"b".repeat(64), false))
        });
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| {
                let mut view = test_view();
                view.enabled = true;
                Ok(view)
            }),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let view = c
            .observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline())
            .unwrap();
        assert_eq!(view.state, "stopped");
        assert_eq!(view.outcome, "");
    }

    #[test]
    fn observe_companion_status_fails_closed() {
        let run = file_run();
        // A binding failure leaves the view untouched without any call.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| panic!("unexpected project call")),
            binding_fn: Box::new(|_| Err("binding down".to_string())),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let mut view = test_view();
        c.observe_companion_status(&run, &mut view, deadline());
        assert_eq!(view.state, "");
        assert!(c.exec.calls().is_empty());
        // A status failure (no runtime state here) also leaves it untouched.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| panic!("unexpected project call")),
            binding_fn: Box::new(|_| Ok(test_binding())),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let mut view = test_view();
        c.observe_companion_status(&run, &mut view, deadline());
        assert_eq!(view.state, "");
        assert!(view.addresses.is_empty());
        assert_eq!(view.dns_name, "");
    }

    #[test]
    fn project_status_maps_observations() {
        // Pre-login states short-circuit before prefs and binding checks.
        let (state, addresses, dns) = project_status(
            br#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#,
            b"{}",
            &test_binding(),
        )
        .unwrap();
        assert_eq!(state, "needs-login");
        assert!(addresses.is_empty());
        assert_eq!(dns, "");
        // A matched running node reports connected with addresses and DNS.
        let (state, addresses, dns) = project_status(
            br#"{"BackendState":"Running","HaveNodeKey":true,"CurrentTailnet":{"Name":"tail-abc"},"Self":{"ID":"node1","DNSName":"soda-abc.tail-abc.ts.net","TailscaleIPs":["100.64.0.5"],"Tags":["tag:soda"],"Online":true,"Expired":false}}"#,
            br#"{"WantRunning":true,"CorpDNS":true,"RouteAll":false,"RunSSH":false,"ExitNodeID":"","ExitNodeIP":"","AdvertiseRoutes":[]}"#,
            &test_binding(),
        )
        .unwrap();
        assert_eq!(state, "connected");
        assert_eq!(addresses, vec!["100.64.0.5".to_string()]);
        assert_eq!(dns, "soda-abc.tail-abc.ts.net");
    }

    #[test]
    fn project_has_node_reports_presence() {
        assert_eq!(
            project_has_node(br#"{"BackendState":"Running","HaveNodeKey":true}"#),
            Ok(true)
        );
        assert_eq!(
            project_has_node(br#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#),
            Ok(false)
        );
    }

    #[test]
    fn confirm_stopped_resolver_ignores_gone_parent() {
        // An unresolvable parent needs no resolver confirmation.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let run = file_run();
        assert!(c.confirm_stopped_resolver(&run, deadline()).is_ok());
    }
}
