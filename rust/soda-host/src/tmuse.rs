//! Muse Code CLI factory runner (`muse exec`).
//!
//! Parallel runner to tcodex for the `muse` harness family: the same
//! supervised lifecycle (reserve/start/wait/stop/output/capture), a
//! different guest (the staged static `muse` binary — never the
//! auto-updating shell launcher), credential (the broker's opaque
//! `auth.json` bytes staged verbatim at the CLI's file-backend lookup
//! path, exactly like the interactive muse runtime and the enrollment
//! fixture; never parsed, never exported), and supervisor argv.
//! Muse borrows: the broker forgets the lease on return and never calls
//! finish, so the daemon denies `finish` for muse leases (capture only
//! echoes the staged copy for the in-process pfactory return path, whose
//! bytes the broker ignores). tcodex.rs stays behavior-identical;
//! shared podman/unit/pid helpers and script builders are reused
//! crate-internally.

use crate::domain;
use crate::project::Executor;
use crate::sha256;
use crate::tcodex::{
    self, valid_factory_role, valid_factory_run_id, valid_harness_version, valid_preparation_id,
    FactoryRun, MAX_FACTORY_OUTPUT_OFFSET, MAX_FACTORY_OUTPUT_READ, MAX_FACTORY_OUTPUT_WINDOW,
    MAX_FACTORY_PROMPT,
};
use crate::texec::{self, Binding, Lease, Service, KIND_FACTORY};
use std::os::unix::fs::PermissionsExt;
use std::time::{Duration, Instant};

// ---------- run paths, scripts, bindings ----------

/// Fixed container paths for one supervised Muse Code run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryMusePaths {
    pub checkout: String,
    pub run_dir: String,
    pub home: String,
    pub muse_config: String,
    pub prompt: String,
    pub marker: String,
    pub started: String,
    pub stop: String,
    pub pid_file: String,
    pub output: String,
    pub stdout: String,
    pub auth: String,
    pub credential: String,
    pub guest: String,
}

/// Validated run identities to fixed checkout/run/home/config paths.
pub fn factory_muse_run_paths(
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
    let muse_config = format!("{home}/.config/muse");
    Some((checkout, run_dir, home, muse_config))
}

/// `factoryCodexPaths` shape for Muse runs: validated run identities to
/// fixed paths. The guest pins the run's harness version.
pub fn factory_muse_paths(run: &FactoryRun) -> Result<FactoryMusePaths, String> {
    run.validate()?;
    if run.harness != tcodex::FACTORY_HARNESS_MUSE {
        return Err(texec::err_denied());
    }
    let (checkout, run_dir, home, muse_config) =
        factory_muse_run_paths(&run.role, &run.preparation, &run.id)
            .ok_or_else(texec::err_denied)?;
    let guest = factory_muse_guest(&run.harness_vers).ok_or_else(texec::err_denied)?;
    Ok(FactoryMusePaths {
        checkout,
        run_dir: run_dir.clone(),
        home: home.clone(),
        muse_config,
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{home}/.config/muse/auth.json"),
        credential: format!("{run_dir}/muse-auth.json"),
        guest,
    })
}

/// `factoryCodexBinding` shape for Muse runs: supervised
/// factory-muse binding check plus derived run paths. The recorded
/// credential root must equal the derived run directory.
pub fn factory_muse_binding(lease: &Lease) -> Result<FactoryMusePaths, String> {
    let Some(b) = &lease.binding else {
        return Err(texec::err_denied());
    };
    if lease.provider_id != texec::PROVIDER_MUSE || lease.kind != KIND_FACTORY {
        return Err(texec::err_denied());
    }
    if b.kind != KIND_FACTORY
        || b.scope != tcodex::FACTORY_SCOPE_MUSE
        || !texec::valid_terminal_id(&b.id)
    {
        return Err(texec::err_denied());
    }
    if !domain::valid_container_id(&b.project)
        || !valid_factory_role(&b.login)
        || !texec::valid_terminal_id(&b.invocation_id)
    {
        return Err(texec::err_denied());
    }
    if !valid_preparation_id(&b.child_id) {
        return Err(texec::err_denied());
    }
    let (checkout, run_dir, home, muse_config) =
        factory_muse_run_paths(&b.login, &b.child_id, &b.id).ok_or_else(texec::err_denied)?;
    if checkout.is_empty() || texec::clean_path(&b.credential_root) != run_dir {
        return Err(texec::err_denied());
    }
    Ok(FactoryMusePaths {
        checkout,
        run_dir: run_dir.clone(),
        home: home.clone(),
        muse_config,
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{home}/.config/muse/auth.json"),
        credential: format!("{run_dir}/muse-auth.json"),
        guest: String::new(),
    })
}

/// `FactoryCodexGuest` shape for Muse runs: fixed versioned guest path
/// for staged harness bytes.
pub fn factory_muse_guest(version: &str) -> Option<String> {
    if !valid_harness_version(version) {
        return None;
    }
    Some(format!("/usr/local/bin/muse-factory-{version}"))
}

/// `factorySupervisor` shape for Muse runs: marker-gated fixed
/// `muse exec` entrypoint. Stdout carries exactly the final answer
/// (headless contract: diagnostics go to stderr), so the answer lands
/// in last-message.txt and diagnostics in stdout.log. The CLI reads the
/// staged `auth.json` through the file backend (same env as enrollment);
/// any inherited `META_API_KEY` is unset so no container env can smuggle
/// a key past the staged file. Exit 45 is the missing-credential refusal
/// (42/43/44 keep the codex gate meanings).
pub fn factory_muse_supervisor(p: &FactoryMusePaths, guest: &str, model: &str) -> String {
    let mut command = format!(
        "{} exec --provider meta --reasoning-effort low --workspace {} --trust-workspace --no-session-log --disable-approval --disable-sandbox",
        tcodex::shell_quote(guest),
        tcodex::shell_quote(&p.checkout),
    );
    if !model.is_empty() {
        command.push_str(&format!(" --model {}", tcodex::shell_quote(model)));
    }
    command.push_str(&format!(
        " --prompt-file {} >{} 2>>{}",
        tcodex::shell_quote(&p.prompt),
        tcodex::shell_quote(&p.output),
        tcodex::shell_quote(&p.stdout),
    ));
    let steps = [
        format!("RUNDIR={}", tcodex::shell_quote(&p.run_dir)),
        ": >\"$RUNDIR/stdout.log\"; : >\"$RUNDIR/last-message.txt\"".to_string(),
        "exec 2>>\"$RUNDIR/stdout.log\"".to_string(),
        "STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo \"$$ ${20}\" >\"$RUNDIR/supervisor.pid\"".to_string(),
        "fail() { echo \"$1\" >\"$RUNDIR/exit\"; exit \"$1\"; }".to_string(),
        "i=0; while [ ! -f \"$RUNDIR/marker\" ]; do [ -f \"$RUNDIR/stop\" ] && fail 44; i=$((i+1)); [ \"$i\" -gt 600 ] && fail 42; sleep 1; done".to_string(),
        "mv \"$RUNDIR/marker\" \"$RUNDIR/started\" || fail 43".to_string(),
        "unset META_API_KEY".to_string(),
        format!("[ -s {} ] || fail 45", tcodex::shell_quote(&p.auth)),
        command,
        "CODE=$?; echo \"$CODE\" >\"$RUNDIR/exit\"; exit \"$CODE\"".to_string(),
    ];
    steps.join("\n") + "\n"
}

/// `factoryCodexSetup` directory-preparation shape for Muse runs.
pub fn muse_setup_script(p: &FactoryMusePaths, uid: i64, gid: i64) -> String {
    format!(
        "set -u\nmkdir -p -m 700 {} {}\nchown {uid}:{gid} {} {} {}\nchmod 700 {} {} {}\n",
        tcodex::shell_quote(&p.home),
        tcodex::shell_quote(&p.muse_config),
        tcodex::shell_quote(&p.run_dir),
        tcodex::shell_quote(&p.home),
        tcodex::shell_quote(&p.muse_config),
        tcodex::shell_quote(&p.run_dir),
        tcodex::shell_quote(&p.home),
        tcodex::shell_quote(&p.muse_config),
    )
}

/// `FactoryCodexStart` staging-gate shape for Muse runs: a non-empty
/// staged `auth.json` and prompt plus the started-or-pending marker.
pub fn muse_start_gate_script(p: &FactoryMusePaths) -> String {
    format!(
        "test -s {} && test -s {} && {{ test -f {} || test -f {}; }}\n",
        tcodex::shell_quote(&p.auth),
        tcodex::shell_quote(&p.prompt),
        tcodex::shell_quote(&p.marker),
        tcodex::shell_quote(&p.started),
    )
}

/// `FactoryCodexReserve` podman-supervisor argv shape for Muse runs (the
/// unit's exec payload). No `CODEX_HOME`: the staged `auth.json` carries
/// the credential through the CLI file backend (`XDG_CONFIG_HOME` pins
/// the lookup; the launcher is bypassed so no update check can run).
pub fn muse_exec_argv(
    container: &str,
    run: &FactoryRun,
    p: &FactoryMusePaths,
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
        format!("XDG_CONFIG_HOME={}/.config", p.home),
        "--env".to_string(),
        "TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--env".to_string(),
        "MUSE_NO_AUTO_UPDATE=1".to_string(),
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
        tcodex::systemd_escape(&factory_muse_supervisor(p, guest, &run.model)),
    ]
}

// ---------- service operations ----------

impl<E: Executor> Service<E> {
    /// Staged-host Muse harness check: regular exec-bit file whose digest
    /// matches the configured pin. Mirrors `verify_identity_harness`.
    pub fn verify_muse_harness(&self) -> Result<(), String> {
        let path = format!("{}/bin/muse", self.muse_harness.trim_end_matches('/'));
        let lstat = std::fs::symlink_metadata(&path)
            .map_err(|_| "verified Muse executable required".to_string())?;
        if !lstat.file_type().is_file() || lstat.permissions().mode() & 0o111 == 0 {
            return Err("verified Muse executable required".to_string());
        }
        let mut data = Vec::new();
        use std::io::Read;
        std::fs::File::open(&path)
            .map_err(|_| "verified Muse executable required".to_string())?
            .read_to_end(&mut data)
            .map_err(|e| format!("muse harness unreadable: {e}"))?;
        if sha256::hex_lower(&sha256::digest(&data)) != self.muse_harness_sha256 {
            return Err("muse harness digest differs".to_string());
        }
        Ok(())
    }

    /// `Service.FactoryCodexReserve` shape for Muse runs.
    pub fn factory_muse_reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin_sha256: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<(crate::texec::Binding, FactoryMusePaths), String> {
        let p = factory_muse_paths(run)?;
        if lease.kind != KIND_FACTORY || lease.execution_id != run.id || lease.generation <= 0 {
            return Err(texec::err_denied());
        }
        if pin_sha256.is_empty()
            || pin_sha256 != self.muse_harness_sha256
            || !self.muse_harness.starts_with('/')
        {
            return Err(texec::err_denied());
        }
        if !(60..=3 * 3600).contains(&max_secs) {
            return Err(texec::err_denied());
        }
        self.verify_muse_harness()?;
        let container = self.factory_project_container(&run.project, true, deadline)?;
        let (uid, gid) = self.factory_role_ids(&container, &run.role, deadline)?;
        self.factory_muse_setup(&container, run, &p, uid, gid, deadline)?;
        let guest = self.factory_muse_stage(&container, &run.harness_vers, deadline)?;
        let unit = tcodex::factory_unit_name_or_denied(&run.id)?;
        let mut args = tcodex::reserve_run_argv(&unit, max_secs);
        args.extend(muse_exec_argv(&container, run, &p, &guest));
        if self.factory_systemd_run(&args, deadline).is_err() {
            return Err("factory unit start unconfirmed".to_string());
        }
        match self.factory_active_invocation(&unit, Duration::from_secs(10), deadline) {
            Ok(invocation) => Ok((
                crate::texec::Binding {
                    kind: KIND_FACTORY.to_string(),
                    id: run.id.clone(),
                    project: container,
                    login: run.role.clone(),
                    uid,
                    gid,
                    scope: tcodex::FACTORY_SCOPE_MUSE.to_string(),
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

    fn factory_muse_setup(
        &self,
        container: &str,
        run: &FactoryRun,
        p: &FactoryMusePaths,
        uid: i64,
        gid: i64,
        deadline: Instant,
    ) -> Result<(), String> {
        let setup = muse_setup_script(p, uid, gid);
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
                    return Err("factory source changed under preparation".to_string());
                }
            }
            _ => return Err("factory source unconfirmed".to_string()),
        }
        Ok(())
    }

    fn factory_muse_stage(
        &self,
        container: &str,
        version: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let guest = factory_muse_guest(version).ok_or_else(texec::err_denied)?;
        let probe = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sha256sum".to_string(),
            guest.clone(),
        ];
        match self.run_podman(&[], &probe, deadline) {
            Err(_) => {
                let host_bin = texec::clean_path(&format!("{}/bin/muse", self.muse_harness));
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
                    tcodex::harness_install_script(&guest),
                ];
                match self.run_podman(&[], &install, deadline) {
                    Ok(out) => {
                        let text = String::from_utf8_lossy(&out);
                        let fields: Vec<&str> = text.split_whitespace().collect();
                        if fields.len() != 2 || fields[0] != self.muse_harness_sha256 {
                            return Err("guest harness digest differs".to_string());
                        }
                    }
                    Err(_) => return Err("factory harness install unconfirmed".to_string()),
                }
            }
            Ok(out) => {
                let text = String::from_utf8_lossy(&out);
                let fields: Vec<&str> = text.split_whitespace().collect();
                if fields.len() != 2 || fields[0] != self.muse_harness_sha256 {
                    return Err("guest harness digest differs".to_string());
                }
            }
        }
        Ok(guest)
    }

    /// `Service.FactoryCodexStart` shape for Muse runs: stage the opaque
    /// broker `auth.json` bytes verbatim (both the run-dir copy capture
    /// echoes and the CLI lookup copy), then prompt, then the start
    /// marker the supervisor gates on. Marker order is load-bearing.
    /// The credential is never parsed: opaque-valid JSON, like codex.
    pub fn factory_muse_start(
        &self,
        lease: &Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let p = factory_muse_binding(lease)?;
        if !texec::credential_valid(credential) {
            return Err(texec::err_denied());
        }
        if prompt.is_empty() || prompt.len() > MAX_FACTORY_PROMPT {
            return Err(texec::err_denied());
        }
        let binding = lease.binding.as_ref().ok_or_else(texec::err_denied)?;
        let container = self.factory_project_container(&lease.project_id, true, deadline)?;
        if container != binding.project {
            return Err(texec::err_stale());
        }
        self.factory_stage_file(&container, binding, &p.credential, credential, deadline)?;
        self.factory_stage_file(&container, binding, &p.auth, credential, deadline)?;
        self.factory_stage_file(&container, binding, &p.prompt, prompt, deadline)?;
        self.factory_stage_file(&container, binding, &p.marker, &[], deadline)?;
        let gate = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container,
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            muse_start_gate_script(&p),
        ];
        match self.run_podman(&[], &gate, deadline) {
            Ok(out) if out.len() <= 1024 => Ok(()),
            _ => Err("factory start staging unconfirmed".to_string()),
        }
    }

    /// `Service.FactoryCodexWait` shape for Muse runs.
    pub fn factory_muse_wait(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(i32, String), String> {
        let p = factory_muse_binding(lease)?;
        let unit = tcodex::factory_unit_name_or_denied(&lease.execution_id)?;
        loop {
            let show = self.factory_unit_state(&unit, deadline)?;
            if !show.active {
                break;
            }
            if tcodex::sleep_until(Instant::now() + Duration::from_millis(500), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        let mut exit = -1;
        let cat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.clone(),
            "/usr/bin/cat".to_string(),
            format!("{}/exit", p.run_dir),
        ];
        if let Ok(out) = self.run_podman(&[], &cat, deadline) {
            if let Some(code) = texec::parse_go_int(String::from_utf8_lossy(&out).trim()) {
                if (0..=255).contains(&code) {
                    exit = code as i32;
                }
            }
        }
        let mut output = String::new();
        let head = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project,
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "65537".to_string(),
            p.output.clone(),
        ];
        if let Ok(out) = self.run_podman(&[], &head, deadline) {
            output = String::from_utf8_lossy(&out).into_owned();
        }
        Ok((exit, output))
    }

    /// `Service.FactoryCodexValidate` shape for Muse runs.
    pub fn factory_muse_validate(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        factory_muse_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(texec::err_denied)?;
        let container = self.factory_project_container(&lease.project_id, true, deadline)?;
        if container != binding.project {
            return Err(texec::err_denied());
        }
        let (uid, gid) = self.factory_role_ids(&container, &binding.login, deadline)?;
        if uid != binding.uid || gid != binding.gid {
            return Err(texec::err_denied());
        }
        let unit = tcodex::factory_unit_name_or_denied(&lease.execution_id)?;
        let show = self
            .factory_unit_state(&unit, deadline)
            .map_err(|_| texec::err_denied())?;
        if !show.active || show.invocation != binding.invocation_id {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    /// `Service.FactoryCodexStop` shape for Muse runs. Idempotent.
    pub fn factory_muse_stop(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        let p = factory_muse_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(texec::err_denied)?;
        let unit = tcodex::factory_unit_name_or_denied(&lease.execution_id)?;
        let before = self.factory_read_pid(&binding.project, &p.pid_file, deadline);
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        match self.factory_container_exists(&binding.project, deadline) {
            Ok(true) => {}
            Ok(false) => return Ok(()),
            Err(err) => return Err(err),
        }
        self.factory_retire(&binding.project, &p.run_dir, deadline)?;
        let after = self.factory_read_pid(&binding.project, &p.pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(&binding.project, &p.run_dir, deadline)?;
            if self.factory_read_pid(&binding.project, &p.pid_file, deadline) != after {
                return Err(texec::err_uncertain());
            }
        }
        Ok(())
    }

    /// `Service.FactoryCodexStopUnbound` shape for Muse runs.
    pub fn factory_muse_stop_unbound(
        &self,
        run: &FactoryRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let p = factory_muse_paths(run)?;
        let unit = tcodex::factory_unit_name_or_denied(&run.id)?;
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        let container = match self.factory_project_container(&run.project, false, deadline) {
            Ok(container) => container,
            Err(_) => {
                match self.factory_container_exists(&format!("soda-{}", run.project), deadline) {
                    Ok(false) => return Ok(()),
                    Ok(true) => return Err(texec::err_uncertain()),
                    Err(err) => return Err(err),
                }
            }
        };
        let before = self.factory_read_pid(&container, &p.pid_file, deadline);
        self.factory_retire(&container, &p.run_dir, deadline)?;
        let after = self.factory_read_pid(&container, &p.pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(&container, &p.run_dir, deadline)?;
            if self.factory_read_pid(&container, &p.pid_file, deadline) != after {
                return Err(texec::err_uncertain());
            }
        }
        Ok(())
    }

    /// `Service.FactoryCodexCapture` shape for Muse runs: echo the
    /// daemon-staged `auth.json` copy (never the CLI's live lookup file)
    /// for the in-process pfactory return path. The broker ignores these
    /// bytes for muse (borrow: forget on return), so capture failure here
    /// only reports host-side staging trouble, never rotation state.
    pub fn factory_muse_capture(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let p = factory_muse_binding(lease)?;
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project,
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "262145".to_string(),
            p.credential.clone(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(out) if texec::credential_valid(&out) => Ok(out),
            _ => Err(texec::err_uncertain()),
        }
    }

    /// `Service.FactoryCodexLive` shape for Muse runs.
    pub fn factory_muse_live(&self, binding: &Binding, deadline: Instant) -> bool {
        if binding.kind != KIND_FACTORY
            || binding.scope != tcodex::FACTORY_SCOPE_MUSE
            || !texec::valid_terminal_id(&binding.id)
            || !texec::valid_terminal_id(&binding.invocation_id)
        {
            return false;
        }
        let Some(unit) = tcodex::factory_unit_name(&binding.id) else {
            return false;
        };
        match self.factory_unit_state(&unit, deadline) {
            Ok(show) => show.active && show.invocation == binding.invocation_id,
            Err(_) => false,
        }
    }

    fn factory_muse_output_binding(
        &self,
        project_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<FactoryMusePaths, String> {
        if binding.kind != KIND_FACTORY
            || binding.scope != tcodex::FACTORY_SCOPE_MUSE
            || !texec::valid_terminal_id(&binding.id)
        {
            return Err(texec::err_denied());
        }
        if !domain::valid_container_id(&binding.project)
            || !valid_factory_role(&binding.login)
            || !texec::valid_terminal_id(&binding.invocation_id)
        {
            return Err(texec::err_denied());
        }
        if !valid_preparation_id(&binding.child_id) {
            return Err(texec::err_denied());
        }
        let (checkout, run_dir, home, muse_config) =
            factory_muse_run_paths(&binding.login, &binding.child_id, &binding.id)
                .ok_or_else(texec::err_denied)?;
        if checkout.is_empty() || texec::clean_path(&binding.credential_root) != run_dir {
            return Err(texec::err_denied());
        }
        let container = self.factory_project_container(project_id, true, deadline)?;
        if container != binding.project {
            return Err(texec::err_stale());
        }
        Ok(FactoryMusePaths {
            checkout,
            run_dir: run_dir.clone(),
            home,
            muse_config,
            stdout: format!("{run_dir}/stdout.log"),
            ..Default::default()
        })
    }

    /// `Service.FactoryCodexOutput` shape for Muse runs.
    pub fn factory_muse_output(
        &self,
        project_id: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<tcodex::FactoryCodexOutputSlice, String> {
        if !(0..=MAX_FACTORY_OUTPUT_OFFSET).contains(&offset)
            || !(1..=MAX_FACTORY_OUTPUT_READ).contains(&limit)
        {
            return Err(texec::err_denied());
        }
        let p = self.factory_muse_output_binding(project_id, binding, deadline)?;
        let mut total = 0i64;
        let stat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            binding.project.clone(),
            "/usr/bin/stat".to_string(),
            "-c".to_string(),
            "%s".to_string(),
            p.stdout.clone(),
        ];
        if let Ok(out) = self.run_podman(&[], &stat, deadline) {
            if let Some(size) = tcodex::factory_output_size(&out) {
                total = size;
            }
        }
        if offset > total {
            return Ok(tcodex::FactoryCodexOutputSlice {
                total,
                offset: total,
                gap: true,
                ..Default::default()
            });
        }
        let (mut start, mut truncated) = (offset, false);
        if start == 0 && total > MAX_FACTORY_OUTPUT_WINDOW {
            start = total - MAX_FACTORY_OUTPUT_WINDOW;
            truncated = true;
        }
        if start >= total {
            return Ok(tcodex::FactoryCodexOutputSlice {
                total,
                offset: start,
                truncated,
                ..Default::default()
            });
        }
        let read = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            binding.project.clone(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            tcodex::output_read_command(&p.stdout, start, limit),
        ];
        let mut out = match self.run_podman(&[], &read, deadline) {
            Ok(out) => out,
            Err(_) => {
                return Ok(tcodex::FactoryCodexOutputSlice {
                    total,
                    offset: start,
                    truncated,
                    ..Default::default()
                });
            }
        };
        if out.len() as i64 > limit {
            out.truncate(limit as usize);
        }
        Ok(tcodex::FactoryCodexOutputSlice {
            data: out,
            total,
            offset: start,
            truncated,
            ..Default::default()
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::texec::ERR_DENIED;
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
        let dir = std::env::temp_dir().join(format!("t26m-{}-{n}-{slug}", std::process::id()));
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
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: "/opt/muse".to_string(),
            muse_harness_sha256: PIN.to_string(),
            muse_harness_version: "1.4.2".to_string(),
        }
    }

    fn write_harness(dir: &std::path::Path) {
        let bin = dir.join("bin");
        std::fs::create_dir_all(&bin).unwrap();
        std::fs::write(bin.join("muse"), b"").unwrap(); // sha256("") == PIN
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(bin.join("muse"), std::fs::Permissions::from_mode(0o755)).unwrap();
    }

    fn reserve_harness() -> (std::path::PathBuf, String) {
        let dir = test_tmp("reserve");
        write_harness(&dir);
        let path = dir.to_str().unwrap().to_string();
        (dir, path)
    }

    fn inspect_json() -> String {
        format!(
            "{{\"id\":{CID:?},\"running\":true,\"project\":{PID:?},\"owner\":\"7\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
        )
    }

    fn muse_run() -> FactoryRun {
        FactoryRun {
            deadline_raw: "2030-01-01T00:00:00Z".to_string(),
            actor: 7,
            id: RID.to_string(),
            project: PID.to_string(),
            role: ROLE.to_string(),
            preparation: PREP.to_string(),
            harness: tcodex::FACTORY_HARNESS_MUSE.to_string(),
            harness_vers: "1.4.2".to_string(),
            model: "muse-spark-1.3".to_string(),
            assignment: PIN.to_string(),
            source_commit: COMMIT.to_string(),
            connection: "conn".to_string(),
        }
    }

    fn muse_run_dir() -> String {
        factory_muse_run_paths(ROLE, PREP, RID).unwrap().1
    }

    fn muse_lease() -> Lease {
        Lease {
            provider_id: texec::PROVIDER_MUSE.to_string(),
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
                scope: tcodex::FACTORY_SCOPE_MUSE.to_string(),
                invocation_id: IID.to_string(),
                credential_root: muse_run_dir(),
                generation: 5,
                child_id: PREP.to_string(),
            }),
            ..Default::default()
        }
    }

    fn euid() -> u32 {
        unsafe { libc::geteuid() }
    }

    #[test]
    fn paths_and_guest() {
        let run = muse_run();
        let p = factory_muse_paths(&run).unwrap();
        assert_eq!(p.checkout, format!("/home/{ROLE}/checkouts/{PREP}"));
        assert!(p.run_dir.ends_with(&format!("/runs/{RID}")));
        assert_eq!(p.home, format!("{}/home", p.run_dir));
        assert_eq!(p.muse_config, format!("{}/home/.config/muse", p.run_dir));
        assert_eq!(p.prompt, format!("{}/prompt", p.run_dir));
        assert_eq!(p.output, format!("{}/last-message.txt", p.run_dir));
        assert_eq!(p.stdout, format!("{}/stdout.log", p.run_dir));
        assert_eq!(p.auth, format!("{}/home/.config/muse/auth.json", p.run_dir));
        assert_eq!(p.credential, format!("{}/muse-auth.json", p.run_dir));
        assert_eq!(p.guest, "/usr/local/bin/muse-factory-1.4.2");
        assert_eq!(
            factory_muse_guest("1.4.2").unwrap(),
            "/usr/local/bin/muse-factory-1.4.2"
        );
        assert!(factory_muse_guest("bad vers!").is_none());
        // The codex family never takes muse paths.
        let codex = FactoryRun {
            harness: tcodex::FACTORY_HARNESS_CODEX.to_string(),
            harness_vers: "0.153.4".to_string(),
            ..muse_run()
        };
        assert_eq!(factory_muse_paths(&codex).unwrap_err(), ERR_DENIED);
    }

    #[test]
    fn binding_gates() {
        let lease = muse_lease();
        let p = factory_muse_binding(&lease).unwrap();
        assert_eq!(p.run_dir, muse_run_dir());
        for mutate in [
            Box::new(|b: &mut Binding| b.scope = tcodex::FACTORY_SCOPE_CODEX.to_string())
                as Box<dyn Fn(&mut Binding)>,
            Box::new(|b: &mut Binding| b.kind = "terminal".to_string()),
            Box::new(|b: &mut Binding| b.id = "short".to_string()),
            Box::new(|b: &mut Binding| b.project = "x".to_string()),
            Box::new(|b: &mut Binding| b.login = "dev".to_string()),
            Box::new(|b: &mut Binding| b.invocation_id = "x".to_string()),
            Box::new(|b: &mut Binding| b.child_id = "x".to_string()),
            Box::new(|b: &mut Binding| b.credential_root = "/elsewhere".to_string()),
        ] {
            let mut broken = lease.clone();
            mutate(broken.binding.as_mut().unwrap());
            assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
        }
        let mut broken = lease.clone();
        broken.provider_id = "codex".to_string();
        assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
        broken = lease.clone();
        broken.kind = "terminal".to_string();
        assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
    }

    #[test]
    fn supervisor_shape() {
        let p = factory_muse_paths(&muse_run()).unwrap();
        let guest = "/usr/local/bin/muse-factory-1.4.2";
        let script = factory_muse_supervisor(&p, guest, "muse-spark-1.3");
        // Fixed headless entrypoint: meta provider pinned, low effort,
        // owned checkout trusted, no prompts, no sandbox, no session log.
        for marker in [
            "'/usr/local/bin/muse-factory-1.4.2' exec --provider meta",
            "--reasoning-effort low",
            "--trust-workspace",
            "--no-session-log",
            "--disable-approval",
            "--disable-sandbox",
            "--model 'muse-spark-1.3'",
            &format!("--workspace '{}'", p.checkout),
            &format!("--prompt-file '{}'", p.prompt),
            // Stream contract: answer to last-message, diagnostics log.
            &format!(">'{}' 2>>'{}'", p.output, p.stdout),
            // File backend: no inherited key may smuggle past the file.
            "unset META_API_KEY",
            &format!("[ -s '{}' ] || fail 45", p.auth),
            // Marker gate keeps the codex meanings.
            "fail 44",
            "fail 42",
            "fail 43",
            "supervisor.pid",
        ] {
            assert!(script.contains(marker), "supervisor lost {marker:?}");
        }
        assert!(!script.contains("CODEX_HOME"));
        assert!(!script.contains("META_API_KEY=\""));
        assert!(!script.contains("--output-last-message"));
        // Empty model keeps the CLI default.
        let bare = factory_muse_supervisor(&p, guest, "");
        assert!(!bare.contains("--model"));
    }

    #[test]
    fn setup_and_gate_scripts() {
        let p = factory_muse_paths(&muse_run()).unwrap();
        let setup = muse_setup_script(&p, 1001, 1001);
        assert!(setup.contains("mkdir -p -m 700 "));
        assert!(setup.contains(&p.home));
        assert!(setup.contains(&p.muse_config));
        assert!(setup.contains("chown 1001:1001 "));
        let gate = muse_start_gate_script(&p);
        assert!(gate.contains(&p.auth));
        assert!(gate.contains(&p.prompt));
        assert!(gate.contains(&p.marker));
        assert!(gate.contains(&p.started));
    }

    #[test]
    fn verify_harness_matrix() {
        let (dir, harness) = reserve_harness();
        let svc = Service {
            exec: FakeExec::new(vec![]),
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: harness.clone(),
            muse_harness_sha256: PIN.to_string(),
            muse_harness_version: "1.4.2".to_string(),
        };
        svc.verify_muse_harness().unwrap();
        // Digest mismatch.
        let bad = Service {
            muse_harness_sha256: "f".repeat(64),
            ..Service {
                exec: FakeExec::new(vec![]),
                codex_harness: String::new(),
                codex_harness_sha256: String::new(),
                codex_harness_version: String::new(),
                muse_harness: harness.clone(),
                muse_harness_sha256: PIN.to_string(),
                muse_harness_version: "1.4.2".to_string(),
            }
        };
        assert_eq!(
            bad.verify_muse_harness().unwrap_err(),
            "muse harness digest differs"
        );
        // Missing binary.
        let missing = test_tmp("missing");
        let svc = make_service(FakeExec::new(vec![]));
        let mut svc = svc;
        svc.muse_harness = missing.to_str().unwrap().to_string();
        assert_eq!(
            svc.verify_muse_harness().unwrap_err(),
            "verified Muse executable required"
        );
        let _ = dir;
    }

    #[test]
    fn reserve_denial_pins() {
        let (_dir, harness) = reserve_harness();
        let run = muse_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        // Each denial fires before any exec call.
        let bad_run = FactoryRun {
            id: "short".to_string(),
            ..muse_run()
        };
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_muse_reserve(&bad_run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "invalid factory run identity"
        );
        // Wrong family takes the codex path, never muse.
        let codex = FactoryRun {
            harness: tcodex::FACTORY_HARNESS_CODEX.to_string(),
            harness_vers: "0.153.4".to_string(),
            ..muse_run()
        };
        assert_eq!(
            svc.factory_muse_reserve(&codex, &lease, PIN, 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_muse_reserve(&run, &lease, "f".repeat(64).as_str(), 600, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_muse_reserve(&run, &lease, PIN, 59, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        let mut svc = make_service(FakeExec::new(vec![]));
        svc.muse_harness = harness;
        assert_eq!(
            svc.factory_muse_reserve(&run, &lease, PIN, 10801, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
    }

    #[test]
    fn reserve_success_argv_sequence() {
        let (_dir, harness) = reserve_harness();
        let guest = "/usr/local/bin/muse-factory-1.4.2";
        let unit = tcodex::factory_unit_name(RID).unwrap();
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),                                      // project container
            ok("1001\n"),                                             // id -u
            ok("1002\n"),                                             // id -g
            ok(""),                                                   // setup script
            ok(&format!("{COMMIT}\n")),                               // git rev-parse
            ok(&format!("{PIN}  {guest}\n")),                         // guest sha256sum (present)
            ok(""),                                                   // systemd-run
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")), // attestation
        ]);
        let mut svc = make_service(exec);
        svc.muse_harness = harness;
        let run = muse_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        let (binding, p) = svc
            .factory_muse_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap();
        assert_eq!(binding.id, RID);
        assert_eq!(binding.project, CID);
        assert_eq!(binding.login, ROLE);
        assert_eq!((binding.uid, binding.gid), (1001, 1002));
        assert_eq!(binding.scope, tcodex::FACTORY_SCOPE_MUSE);
        assert_eq!(binding.invocation_id, IID);
        assert_eq!(binding.credential_root, p.run_dir);
        assert_eq!(binding.generation, 5);
        assert_eq!(binding.child_id, PREP);
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 8);
        // systemd-run argv is byte-exact: header + podman payload.
        let bus = tcodex::factory_user_bus(euid());
        let mut want_run = vec![
            "/usr/bin/env".to_string(),
            bus,
            "/usr/bin/systemd-run".to_string(),
            "--user".to_string(),
        ];
        want_run.extend(tcodex::reserve_run_argv(&unit, 600));
        want_run.extend(muse_exec_argv(CID, &run, &p, guest));
        let got_run: Vec<String> = std::iter::once(calls[6].1.clone())
            .chain(calls[6].2.clone())
            .collect();
        assert_eq!(got_run, want_run);
        // Muse env: no CODEX_HOME, update check off, file backend pinned.
        let payload = &calls[6].2;
        assert!(!payload.iter().any(|a| a.contains("CODEX_HOME")));
        assert!(payload.iter().any(|a| a == "MUSE_NO_AUTO_UPDATE=1"));
        assert!(payload.iter().any(|a| a == &format!("HOME={}", p.home)));
        assert!(payload
            .iter()
            .any(|a| a == &format!("XDG_CONFIG_HOME={}/.config", p.home)));
        assert!(payload.iter().any(|a| a == "TBH_CREDENTIAL_BACKEND=file"));
        assert!(!payload.iter().any(|a| a.starts_with("META_API_KEY")));
    }

    #[test]
    fn reserve_stage_missing_guest() {
        let (_dir, harness) = reserve_harness();
        let guest = "/usr/local/bin/muse-factory-1.4.2";
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1002\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            err("exit status 1"),             // guest sha256sum (absent)
            ok(""),                           // podman cp
            ok(&format!("{PIN}  {guest}\n")), // install digest check
            ok(""),
            ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ]);
        let mut svc = make_service(exec);
        svc.muse_harness = harness.clone();
        let run = muse_run();
        let lease = Lease {
            kind: KIND_FACTORY.to_string(),
            execution_id: RID.to_string(),
            generation: 5,
            ..Default::default()
        };
        let (binding, _) = svc
            .factory_muse_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap();
        assert_eq!(binding.scope, tcodex::FACTORY_SCOPE_MUSE);
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 10);
        assert_eq!(calls[6].2[1], "cp");
        assert!(calls[6].2[2].ends_with("/bin/muse"));
        // Digest mismatch refuses.
        let exec = FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1002\n"),
            ok(""),
            ok(&format!("{COMMIT}\n")),
            err("exit status 1"),
            ok(""),
            ok(&format!("{}  {guest}\n", "f".repeat(64))),
        ]);
        let mut svc = make_service(exec);
        svc.muse_harness = harness;
        assert_eq!(
            svc.factory_muse_reserve(&run, &lease, PIN, 600, deadline())
                .unwrap_err(),
            "guest harness digest differs"
        );
    }

    #[test]
    fn start_flows() {
        let lease = muse_lease();
        let p = factory_muse_binding(&lease).unwrap();
        let cred = b"{\"schema_version\":1,\"providers\":{}}";
        // Denials before exec. The credential is opaque: any valid JSON
        // stages (even `{}`), only non-credential bytes refuse.
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_muse_start(&Lease::default(), cred, b"prompt", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_muse_start(&lease, b"nope", b"prompt", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_muse_start(&lease, b"", b"prompt", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert_eq!(
            svc.factory_muse_start(&lease, cred, b"", deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
        // Stale incarnation.
        let other = "f".repeat(64);
        let stale_json = inspect_json().replace(CID, &other);
        let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
        assert_eq!(
            svc.factory_muse_start(&lease, cred, b"prompt", deadline())
                .unwrap_err(),
            crate::texec::ERR_STALE
        );
        // Success stages the verbatim bytes twice (echo copy + CLI
        // lookup), then prompt, marker, then the gate.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
            ok(""),
        ]));
        svc.factory_muse_start(&lease, cred, b"do work", deadline())
            .unwrap();
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 6);
        assert_eq!(calls[1].0, cred);
        assert_eq!(calls[2].0, cred);
        assert_eq!(calls[3].0, b"do work");
        assert!(calls[4].0.is_empty());
        assert!(
            calls[1].2[6].contains(&tcodex::shell_quote(&p.credential)),
            "{}",
            calls[1].2[6]
        );
        assert!(
            calls[2].2[6].contains(&tcodex::shell_quote(&p.auth)),
            "{}",
            calls[2].2[6]
        );
        assert!(
            calls[3].2[6].contains(&tcodex::shell_quote(&p.prompt)),
            "{}",
            calls[3].2[6]
        );
        assert!(
            calls[4].2[6].contains(&tcodex::shell_quote(&p.marker)),
            "{}",
            calls[4].2[6]
        );
        assert_eq!(calls[5].2[5], muse_start_gate_script(&p));
    }

    #[test]
    fn wait_output_live() {
        let lease = muse_lease();
        // Wait: inactive unit, exit file, last message.
        let svc = make_service(FakeExec::new(vec![
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok("0\n"),
            ok("{\"status\":\"completed\"}\n"),
        ]));
        let (code, out) = svc.factory_muse_wait(&lease, deadline()).unwrap();
        assert_eq!((code, out.as_str()), (0, "{\"status\":\"completed\"}\n"));
        // Output slice: container, stat, bounded read.
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("11\n"),
            ok("hello world"),
        ]));
        let binding = lease.binding.clone().unwrap();
        let slice = svc
            .factory_muse_output(PID, &binding, 0, 100, deadline())
            .unwrap();
        assert_eq!(slice.total, 11);
        assert_eq!(slice.data, b"hello world");
        // Live: scope-gated attestation.
        let svc = make_service(FakeExec::new(vec![ok(&format!(
            "ActiveState=active\nInvocationID={IID}\n"
        ))]));
        assert!(svc.factory_muse_live(&binding, deadline()));
        let mut dead = binding.clone();
        dead.scope = tcodex::FACTORY_SCOPE_CODEX.to_string();
        let svc = make_service(FakeExec::new(vec![]));
        assert!(!svc.factory_muse_live(&dead, deadline()));
    }

    #[test]
    fn stop_and_capture() {
        let lease = muse_lease();
        // Stop: pid, systemctl, await, exists, retire, pid.
        let svc = make_service(FakeExec::new(vec![
            ok("111 222\n"),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            ok(""),
            ok(""),
            ok(""),
        ]));
        svc.factory_muse_stop(&lease, deadline()).unwrap();
        let calls = svc.exec.calls();
        assert_eq!(calls[0].2[3], "/usr/bin/cat");
        assert!(calls[0].2[4].ends_with("supervisor.pid"));
        // Capture echoes the daemon-staged auth.json copy (not the
        // CLI's live lookup file).
        let cred = "{\"schema_version\":1,\"providers\":{}}";
        let svc = make_service(FakeExec::new(vec![ok(cred)]));
        let back = svc.factory_muse_capture(&lease, deadline()).unwrap();
        assert_eq!(back, cred.as_bytes());
        let calls = svc.exec.calls();
        assert!(calls[0].2[6].ends_with("muse-auth.json"));
    }

    #[test]
    fn finish_denied_for_muse() {
        // Muse borrows: the broker forgets on return and never calls
        // finish (same denial as the interactive muse runtime).
        let lease = muse_lease();
        let delivery = texec::Delivery {
            lease,
            credential: Some(b"{}".to_vec()),
        };
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_identity_operation("finish", &delivery, deadline())
                .unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
    }
}
