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
    /// header and command availability. Retained for explicit digest
    /// re-verification; new executions resolve through the store snapshot
    /// and never consult configured pins.
    ///
    /// The expected identity resolves on every call: pinned configuration
    /// verifies against the pin, while empty pins resolve the fresh shared
    /// store selection. Returns the verified bundle digest so the execution
    /// keeps its exact bytes across refreshes. Never caches one hash.
    pub fn verify_guest_binary(
        &self,
        container: &str,
        child: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let (_, expected_sha256) =
            super::provider::expected_muse_identity(&self.binary_version, &self.binary_sha256)?;
        self.verify_guest_digest(container, child, &expected_sha256, deadline)
    }

    /// Guest-side `podman exec` prefix for nested children; empty for the
    /// normal project-namespace path.
    pub(in crate::muse) fn guest_binary_prefix(&self, child: &str) -> Vec<String> {
        let mut prefix = Vec::new();
        if !child.is_empty() {
            prefix.extend([
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "exec".to_string(),
                child.to_string(),
            ]);
        }
        prefix
    }

    /// Verify the guest provider binary against one explicit expected
    /// digest: pinned dispatcher digest, ELF header and command
    /// availability. Returns the verified digest so the execution keeps its
    /// exact bytes across refreshes. Never caches one hash.
    pub(in crate::muse) fn verify_guest_digest(
        &self,
        container: &str,
        child: &str,
        expected_sha256: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_container_id(expected_sha256) {
            return Err(terminal::err_denied());
        }
        let prefix = self.guest_binary_prefix(child);
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
        if fields.len() != 2 || fields[0] != expected_sha256 {
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
        let mut check_argv = prefix;
        check_argv.extend([
            "/usr/local/bin/muse".to_string(),
            "--soda-check".to_string(),
        ]);
        self.guest(container, &[], &check_argv, deadline)
            .map(|_| expected_sha256.to_string())
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
        // Store-authoritative for ALL new executions: legacy configured pins
        // never select bytes for new work (mandatory freshness supersedes
        // them for host launches exactly as for broker sessions). Historical
        // bytes survive only through the bound-digest recovery path, never
        // through startup config. One coherent snapshot per invocation,
        // restaged into the running Project, then bound to this run.
        let snapshot = super::provider::fresh_provider_snapshot()?;
        let verified_digest = match self.ensure_guest_provider(&caller, &snapshot, budget) {
            Ok(digest) => {
                super::provider::release_provider_snapshot(snapshot);
                digest
            }
            Err(err) => {
                super::provider::release_provider_snapshot(snapshot);
                return Err(err);
            }
        };
        let mut execution = self.reserve_execution(&caller, &request, budget)?;
        // Bind this run to its exact verified bundle: recovery re-resolves
        // this digest (fail closed), never the current selection.
        execution.provider_digest = verified_digest;
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
            // Stamped by `prepare_execution` with the verified bundle digest;
            // empty until then, never a cached selection.
            provider_digest: String::new(),
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
        Ok(())
    }
}
