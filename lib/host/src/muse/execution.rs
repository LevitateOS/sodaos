use std::os::unix::io::RawFd;
use std::time::{Duration, Instant};

use super::{
    host_go_arch, muse_arguments, muse_elf, muse_peer_alive, muse_resize, muse_signal,
    unit_invocation_argv, LaunchControl, LaunchRequest, MuseCaller, MuseExecution, MuseHooks,
    MusePeer, MuseRuntime,
};
use crate::domain;
use crate::project::Executor;
use crate::terminal::{self, AcquireRequest, Binding};

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// `MuseRuntime.verifyGuestBinary`: pinned dispatcher digest, ELF
    /// header and version handshake.
    pub fn verify_guest_binary(
        &self,
        container: &str,
        child: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !domain::valid_container_id(&self.binary_sha256) {
            return Err(terminal::err_denied());
        }
        let mut prefix = Vec::new();
        if !child.is_empty() {
            prefix.extend([
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "exec".to_string(),
                child.to_string(),
            ]);
        }
        let mut hash_argv = prefix.clone();
        hash_argv.extend([
            "/usr/bin/sha256sum".to_string(),
            "/usr/local/libexec/soda/muse".to_string(),
        ]);
        let hash = match self.guest(container, &[], &hash_argv, deadline) {
            Ok(hash) => hash,
            Err(_) => return Err(terminal::err_denied()),
        };
        let text = String::from_utf8_lossy(&hash);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != self.binary_sha256 {
            return Err(terminal::err_denied());
        }
        let mut head_argv = prefix.clone();
        head_argv.extend([
            "/usr/bin/head".to_string(),
            "--bytes=64".to_string(),
            "/usr/local/libexec/soda/muse".to_string(),
        ]);
        let header = self
            .guest(container, &[], &head_argv, deadline)
            .map_err(|_| terminal::err_denied())?;
        muse_elf(&header, host_go_arch())?;
        if self.binary_version.is_empty() {
            return Err(terminal::err_denied());
        }
        let mut check_argv = prefix;
        check_argv.extend([
            "/usr/local/bin/muse".to_string(),
            "--soda-check".to_string(),
            self.binary_version.clone(),
        ]);
        self.guest(container, &[], &check_argv, deadline)
            .map(|_| ())
    }

    // ---------- execution lifecycle (muse_execution_linux.go) ----------

    /// `MuseRuntime.Start` preparation: validate, sanitize, resolve,
    /// verify, reserve, stage. Returns the staged execution; the caller
    /// spawns [`muse_command_argv`], then calls [`deliver_execution`].
    pub fn prepare_execution(
        &self,
        peer: &MusePeer,
        request: &LaunchRequest,
        deadline: Instant,
    ) -> Result<MuseExecution, String> {
        let budget = deadline.min(Instant::now() + Duration::from_secs(30));
        let mut request = request.clone();
        if request.validate().is_err() {
            return Err(terminal::err_denied());
        }
        request.args = muse_arguments(&request.args)?;
        let caller = self.resolve(peer, budget)?;
        self.verify_guest_binary(&caller.container, &caller.child, budget)?;
        let execution = self.reserve_execution(&caller, &request, budget)?;
        if !muse_peer_alive(peer) {
            let cleanup = Instant::now() + Duration::from_secs(30);
            let _ = self.hooks.end(caller.actor, &execution.lease.id, cleanup);
            return Err(terminal::err_denied());
        }
        if let Err(err) = self.stage(&caller, &execution.path, &request.config_home, budget) {
            let cleanup = Instant::now() + Duration::from_secs(30);
            let _ = self.stop_execution(
                &execution.binding,
                caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            );
            return Err(err);
        }
        Ok(execution)
    }

    /// `MuseRuntime.reserveExecution`: broker connection, lease, binding.
    pub fn reserve_execution(
        &self,
        caller: &MuseCaller,
        request: &LaunchRequest,
        deadline: Instant,
    ) -> Result<MuseExecution, String> {
        let connection = self.hooks.select(
            caller.actor,
            &caller.project,
            &request.connection_id,
            deadline,
        )?;
        let id = terminal::rand_id().map_err(|_| terminal::err_denied())?;
        let lease = self.hooks.acquire(
            &AcquireRequest {
                execution_id: id.clone(),
                actor_id: caller.actor,
                connection_id: connection,
                project_id: caller.project.clone(),
                kind: terminal::KIND_TERMINAL.to_string(),
                provider_id: terminal::PROVIDER_MUSE.to_string(),
                deadline_secs: terminal::now_unix() + 12 * 3600,
                ..Default::default()
            },
            deadline,
        )?;
        let path = format!("/run/soda-muse/{id}");
        let path = if caller.child.is_empty() {
            path
        } else {
            format!("/run/soda-muse/nested/{}/{id}", caller.registration)
        };
        let binding = Binding {
            kind: terminal::KIND_TERMINAL.to_string(),
            id: id.clone(),
            project: caller.container.clone(),
            child_id: caller.child.clone(),
            uid: caller.uid,
            gid: caller.gid,
            login: caller.login.clone(),
            generation: lease.generation,
            scope: "muse-project".to_string(),
            credential_root: path.clone(),
            ..Default::default()
        };
        Ok(MuseExecution {
            caller: caller.clone(),
            request: request.clone(),
            lease,
            binding,
            path: path.clone(),
            unit: format!("soda-muse-{id}.service"),
        })
    }

    /// `MuseRuntime.deliverExecution`: await the unit, bind the invocation,
    /// attach the lease, deliver the credential, plant the ready marker.
    pub fn deliver_execution(
        &self,
        execution: &mut MuseExecution,
        deadline: Instant,
    ) -> Result<(), String> {
        let budget = deadline.min(Instant::now() + Duration::from_secs(30));
        self.await_unit(&execution.caller.container, &execution.unit, budget)?;
        let body = self.guest(
            &execution.caller.container,
            &[],
            &unit_invocation_argv(&execution.unit),
            budget,
        )?;
        execution.binding.invocation_id = String::from_utf8_lossy(&body).trim().to_string();
        if !terminal::valid_terminal_id(&execution.binding.invocation_id) {
            return Err(terminal::err_denied());
        }
        let mut delivery = self
            .hooks
            .attach(&execution.lease.id, &execution.binding, budget)?;
        let mut credential = delivery.credential.take().unwrap_or_default();
        if !terminal::credential_valid(&credential) {
            return Err(terminal::err_denied());
        }
        let target = format!("{}/auth.json", execution.path);
        let dd = [
            "/usr/bin/dd".to_string(),
            format!("of={target}"),
            "status=none".to_string(),
        ];
        let dd_result = self.guest(&execution.caller.container, &credential, &dd, budget);
        for byte in credential.iter_mut() {
            *byte = 0;
        }
        dd_result.map(|_| ())?;
        self.guest_refs(
            &execution.caller.container,
            &[],
            &["/usr/bin/touch", &format!("{}/ready", execution.path)],
            budget,
        )
        .map(|_| ())
    }

    /// `MuseRuntime.controlExecution`: unit signal or PTY resize.
    pub fn control_execution(
        &self,
        execution: &MuseExecution,
        stdin_fd: RawFd,
        child_pid: Option<i32>,
        control: &LaunchControl,
        deadline: Instant,
    ) -> Result<(), String> {
        if control.signal != 0 {
            if control.cols != 0 || control.rows != 0 || !muse_signal(control.signal) {
                return Err(terminal::err_denied());
            }
            return self
                .guest_refs(
                    &execution.caller.container,
                    &[],
                    &[
                        "/usr/bin/systemctl",
                        "kill",
                        "--kill-whom=all",
                        &format!("--signal={}", control.signal),
                        &execution.unit,
                    ],
                    deadline,
                )
                .map(|_| ());
        }
        if !execution.request.tty {
            return Err(terminal::err_denied());
        }
        muse_resize(stdin_fd, control)?;
        if let Some(pid) = child_pid {
            // SAFETY: kill with a valid signal number; ESRCH is ignored.
            unsafe {
                libc::kill(pid, libc::SIGWINCH);
            }
        }
        Ok(())
    }
}
