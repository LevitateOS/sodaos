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

use super::native::{
    factory_unit_name_or_denied, harness_install_script, reserve_run_argv, shell_quote,
    systemd_escape,
};
use super::run::{
    valid_factory_role, valid_factory_run_id, valid_harness_version, valid_preparation_id,
    FACTORY_SCOPE_CODEX, MAX_FACTORY_PROMPT,
};
use crate::domain;
use crate::project::Executor;
use crate::sha256;
use crate::terminal::{self, Binding, Lease, Service, KIND_FACTORY};

// `muse_serve_oracle` compiles this module through a private `#[path]` copy
// that never touches some re-exported names; they serve the real library
// (dbackend conversions until the serial caller join).
#[allow(unused_imports)]
pub use super::output::FactoryCodexOutputSlice;
#[allow(unused_imports)]
pub use super::run::{FactoryRun, FACTORY_SCOPE_MUSE};

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
/// `binding` checks.
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
    let (checkout, run_dir, home, codex) = super::binding::checked_binding_paths(
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

impl<E: Executor> Service<E> {
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
        super::output::check_output_range(offset, limit)?;
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

#[cfg(test)]
mod source_head_tests {
    use super::*;
    use crate::project::Native;
    use std::cell::Cell;
    use std::fs;
    use std::path::PathBuf;

    struct Scratch(PathBuf);

    impl Drop for Scratch {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }

    struct GitCheckoutExec {
        calls: Cell<usize>,
        checkout: String,
    }

    impl Executor for GitCheckoutExec {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            assert_eq!(cmd, "/usr/bin/podman");
            let call = self.calls.get();
            self.calls.set(call + 1);
            if call % 2 == 0 {
                assert_eq!(
                    &args[..5],
                    &[
                        "--remote=false",
                        "exec",
                        "soda-test-container",
                        "/usr/bin/sh",
                        "-c"
                    ]
                );
                return Ok(Vec::new());
            }
            assert_eq!(
                args,
                [
                    "--remote=false",
                    "exec",
                    "--user",
                    "soda-coder",
                    "soda-test-container",
                    "/usr/bin/git",
                    "-C",
                    self.checkout.as_str(),
                    "rev-parse",
                    "HEAD"
                ]
            );
            Native.run(stdin, "/usr/bin/git", &args[6..], deadline)
        }
    }

    #[test]
    fn preparation_id_follows_current_checkout_head_and_rejects_stale_source() {
        let nonce = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let name = format!("soda-codex-head-{}-{nonce}", std::process::id());
        let scratch = Scratch(std::env::temp_dir().join(name));
        let checkout = scratch.0.join("checkout");
        fs::create_dir_all(&checkout).unwrap();
        let git = |args: &[&str]| {
            Native
                .run(
                    &[],
                    "/usr/bin/git",
                    args,
                    Instant::now() + Duration::from_secs(10),
                )
                .unwrap()
        };
        let path = checkout.to_str().unwrap();
        git(&["-C", path, "init", "--quiet"]);
        git(&["-C", path, "config", "--local", "user.name", "soda-test"]);
        git(&[
            "-C",
            path,
            "config",
            "--local",
            "user.email",
            "soda-test@example.invalid",
        ]);
        git(&[
            "-C",
            path,
            "-c",
            "commit.gpgsign=false",
            "commit",
            "--allow-empty",
            "-m",
            "first",
        ]);
        let first = String::from_utf8(git(&["-C", path, "rev-parse", "HEAD"]))
            .unwrap()
            .trim()
            .to_string();
        git(&[
            "-C",
            path,
            "-c",
            "commit.gpgsign=false",
            "commit",
            "--allow-empty",
            "-m",
            "second",
        ]);
        let second = String::from_utf8(git(&["-C", path, "rev-parse", "HEAD"]))
            .unwrap()
            .trim()
            .to_string();
        assert_ne!(first, second);

        let run = FactoryRun {
            role: "soda-coder".into(),
            preparation: "f0123456789abcdef01234567".into(),
            ..Default::default()
        };
        let paths = |checkout: &str| FactoryCodexPaths {
            checkout: checkout.into(),
            run_dir: scratch.0.join("run").to_string_lossy().into_owned(),
            home: scratch.0.join("home").to_string_lossy().into_owned(),
            codex: scratch.0.join("codex").to_string_lossy().into_owned(),
            ..Default::default()
        };
        for (source, expected) in [
            (&second, None),
            (&first, Some("factory checkout is not the assigned commit")),
        ] {
            let exec = GitCheckoutExec {
                calls: Cell::new(0),
                checkout: path.into(),
            };
            let svc = Service {
                exec,
                codex_harness: String::new(),
                codex_harness_sha256: String::new(),
                codex_harness_version: String::new(),
                muse_harness: String::new(),
                muse_harness_sha256: String::new(),
                muse_harness_version: String::new(),
            };
            let assigned = FactoryRun {
                source_commit: source.clone(),
                ..run.clone()
            };
            let result = svc.factory_codex_setup(
                "soda-test-container",
                &assigned,
                &paths(path),
                1001,
                1001,
                Instant::now() + Duration::from_secs(10),
            );
            match expected {
                None => assert!(result.is_ok(), "current candidate head refused: {result:?}"),
                Some(message) => assert_eq!(result.unwrap_err(), message),
            }
            assert_eq!(svc.exec.calls.get(), 2);
        }
    }
}
