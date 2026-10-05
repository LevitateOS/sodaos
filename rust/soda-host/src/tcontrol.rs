//! Tailnet control plane for the soda-host daemon.
//!
//! PR26 port of Go `internal/tailnet/*` (control plane). The daemon adapter
//! programs against exactly this surface:
//!
//! - [`Control`] with [`settings`](Control::settings),
//!   [`options`](Control::options), [`host_action`](Control::host_action),
//!   [`enrollment`](Control::enrollment) and [`project`](Control::project)
//! - `impl TailnetControl for Control` (project/run_binding/enroll_run for
//!   the companion)
//!
//! Error strings carry typed-cause substrings the adapter matches, mirroring
//! Go's `tailnetErrorStatus`: `invalid request` (400), `conflict` (409),
//! `unsupported` (422), `unavailable` (503); anything else is 502.
//!
//! Supporting layers: [`tcontrol_wire`](crate::tcontrol_wire) (DTOs, strict
//! decoders, validators, encoders),
//! [`tcontrol_policy`](crate::tcontrol_policy) (locked policy files),
//! [`tcontrol_native`](crate::tcontrol_native) (LocalAPI observation, host
//! actions, CLI client), [`tcontrol_provider`](crate::tcontrol_provider)
//! (provider HTTPS via host curl) and
//! [`tcontrol_enroll`](crate::tcontrol_enroll) (run enrollment).

use std::path::PathBuf;
use std::time::Instant;

use crate::project::{Executor, Native};
use crate::tailnet_companion::TailnetControl;
use crate::tailnet_domain;
use crate::tcontrol_enroll as enroll;
use crate::tcontrol_native as native;
use crate::tcontrol_policy as policy;
use crate::tcontrol_provider as provider;
use crate::tcontrol_wire as wire;

/// Production executor: direct native process execution with deadline kill.
pub type NativeExec = Native;

/// Control-plane configuration. Paths default to the production appliance
/// layout; tests override them with scratch state.
#[derive(Debug, Clone)]
pub struct Options {
    /// Policy directory holding `policy.json` (`/var/lib/soda-tailnet`).
    pub state_dir: PathBuf,
    /// Owner uid enforced on all policy state (`0` in production).
    pub uid: u32,
    /// Companion runtime configured (mirrors `NewProjectControl`).
    pub runtime: bool,
    /// Tailscaled LocalAPI socket (`/var/run/tailscale/tailscaled.sock`).
    pub socket: PathBuf,
    /// Tailscale CLI (`/usr/bin/tailscale`).
    pub cli: String,
    /// Soda libexec dir (`/usr/libexec/soda`).
    pub libexec: String,
    /// Host curl binary for provider HTTPS (`/usr/bin/curl`).
    pub curl: String,
}

impl Default for Options {
    fn default() -> Self {
        Options {
            state_dir: PathBuf::from("/var/lib/soda-tailnet"),
            uid: 0,
            runtime: false,
            socket: PathBuf::from(native::HOST_SOCKET),
            cli: native::DEFAULT_CLI.to_string(),
            libexec: native::DEFAULT_LIBEXEC.to_string(),
            curl: provider::DEFAULT_CURL.to_string(),
        }
    }
}

/// Tailnet control plane. Generic over the crate [`Executor`], like the
/// companion; production passes [`NativeExec`].
pub struct Control<E: Executor = NativeExec> {
    exec: E,
    policy: policy::PolicyStore,
    socket: PathBuf,
    cli: String,
    libexec: String,
    curl: String,
    /// Test seam: canned LocalAPI `(status, body)` rounds.
    pub(crate) local_stub: Option<Box<native::Transport>>,
    /// Test seam: canned provider `(status, body)` rounds.
    pub(crate) provider_stub: Option<Box<provider::ProviderTransport>>,
}

impl<E: Executor> Control<E> {
    /// Build a control plane. `exec` runs native commands (`tailscale`,
    /// `soda-forgejo-tailnet`, `curl`); `opts` carries paths, uid and the
    /// runtime flag.
    pub fn new(exec: E, opts: Options) -> Self {
        Control {
            exec,
            policy: policy::PolicyStore::new(opts.state_dir, opts.uid, opts.runtime),
            socket: opts.socket,
            cli: opts.cli,
            libexec: opts.libexec,
            curl: opts.curl,
            local_stub: None,
            provider_stub: None,
        }
    }

    /// Build a control plane with the companion runtime enabled. Used only
    /// when the immutable companion image is explicitly configured; like
    /// Go's `NewProjectControl`, construction performs no I/O.
    pub fn new_project_control(exec: E, opts: Options) -> Self {
        let mut opts = opts;
        opts.runtime = true;
        Self::new(exec, opts)
    }

    fn local_round_trip(
        &self,
        method: &str,
        path: &str,
        body: Option<&[u8]>,
        deadline: Instant,
    ) -> Result<(u16, Vec<u8>), String> {
        if let Some(stub) = &self.local_stub {
            stub(method, path, body, deadline)
        } else {
            native::local_request(&self.socket, method, path, body, deadline)
        }
    }

    #[allow(clippy::type_complexity)]
    fn transport(
        &self,
    ) -> impl Fn(&str, &str, Option<&[u8]>, Instant) -> Result<(u16, Vec<u8>), String> + '_ {
        |method, path, body, deadline| self.local_round_trip(method, path, body, deadline)
    }

    fn provider(&self) -> enroll::Provider<'_> {
        match &self.provider_stub {
            Some(stub) => enroll::Provider::Stub(stub),
            None => enroll::Provider::Curl {
                exec: &self.exec,
                curl: &self.curl,
            },
        }
    }

    fn check_credential(
        &self,
        deadline: Instant,
        r: &wire::EnrollmentRequest,
    ) -> Result<(), String> {
        enroll::check_credential(&self.provider(), r, deadline)
    }

    /// Take the policy lock for a host action. `Ok(None)` is a missing
    /// policy directory (authentication observes without initializing).
    fn lock_policy(
        &self,
        deadline: Instant,
        write: bool,
    ) -> Result<Option<policy::PolicyLock>, String> {
        self.policy.lock(deadline, write)
    }

    fn observe(&self, deadline: Instant) -> Result<(wire::HostView, String), String> {
        native::observe(&self.transport(), deadline)
    }

    /// JSON bytes of the settings view, same shape as Go `Control.Settings`.
    pub fn settings(&self, deadline: Instant) -> Result<Vec<u8>, String> {
        let enrollment = self.policy.enrollment(deadline)?;
        let mut result = wire::SettingsView {
            host: None,
            host_unavailable: false,
            enrollment,
        };
        match self.observe(deadline) {
            Ok((host, _)) => result.host = Some(host),
            Err(_) => result.host_unavailable = true,
        }
        capped(result.encode().into_bytes())
    }

    /// JSON bytes of the project options, same shape as Go `Control.Options`.
    pub fn options(&self, deadline: Instant) -> Result<Vec<u8>, String> {
        let v = self.policy.enrollment(deadline)?;
        let opts = wire::ProjectOptions {
            revision: v.revision,
            binding: v.binding,
            tailnet: v.tailnet,
            available: v.runtime_supported && v.configured && v.admission,
            default: v.runtime_supported && v.admission && v.default,
        };
        capped(opts.encode().into_bytes())
    }

    /// Strict-decode a host-action body and run it, same result shape as Go
    /// `HostAction`.
    pub fn host_action(&self, body: &[u8], deadline: Instant) -> Result<Vec<u8>, String> {
        let r = wire::decode_host_request(body)?;
        if r.validate().is_err() {
            return Err(wire::err_invalid());
        }
        let _held = self.lock_policy(deadline, r.action != "authentication")?;
        let (before, auth) = self.observe(deadline)?;
        if before.revision != r.revision {
            return Err(wire::err_conflict());
        }
        if r.action == "authentication" {
            let result = wire::HostResult {
                outcome: "observed".to_string(),
                host: Some(before),
                readback_unavailable: false,
                auth_url: wire::authentication_url(&auth),
            };
            return capped(result.encode().into_bytes());
        }
        if Instant::now() >= deadline {
            return Err(wire::err_unconfirmed());
        }
        let transport = self.transport();
        let socket = self.socket.as_path();
        let (selected, action_err) = match r.action.as_str() {
            "signin" => (
                String::new(),
                native::execute_signin(
                    &transport, &self.exec, &self.cli, socket, &before, deadline,
                )
                .err(),
            ),
            "logout" => (
                String::new(),
                native::execute_logout(&transport, deadline).err(),
            ),
            "exit-node" => match native::execute_exit_node(
                &self.exec, &self.cli, socket, &r, &before, deadline,
            ) {
                Ok(id) => (id, None),
                Err(e) if e.contains("conflict") => return Err(e),
                Err(e) => (String::new(), Some(e)),
            },
            "advertise-exit-node" => (
                String::new(),
                native::execute_advertise_exit_node(&self.exec, &self.cli, socket, &r, deadline)
                    .err(),
            ),
            "refresh-forgejo" => (
                String::new(),
                native::execute_refresh_forgejo(&self.exec, &self.libexec, deadline).err(),
            ),
            _ => (String::new(), None),
        };
        let result = native::readback_host_action(&transport, &r, &selected, action_err, deadline);
        capped(result.encode().into_bytes())
    }

    /// Strict-decode an enrollment body and apply it, same result shape as
    /// Go `Enrollment`. The transient client secret is zeroed after use.
    pub fn enrollment(&self, body: &[u8], deadline: Instant) -> Result<Vec<u8>, String> {
        let mut r = wire::decode_enrollment_request(body)?;
        let out = self
            .policy
            .update(deadline, &r, &|d, req| self.check_credential(d, req));
        r.zero_secret();
        capped(out?.encode().into_bytes())
    }

    /// Inspect or mutate one project's tailnet selection, same semantics as
    /// Go `Control.Project`. `cid` is the project container id.
    pub fn project(
        &self,
        req: &tailnet_domain::ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<tailnet_domain::ProjectView, String> {
        self.policy.project(deadline, req, cid)
    }

    /// Native-only public-metadata projection for one run incarnation,
    /// mirroring `Control.RunBinding`.
    pub fn run_binding(
        &self,
        target: &tailnet_domain::RunTarget,
        deadline: Instant,
    ) -> Result<tailnet_domain::RunBinding, String> {
        self.policy.run_binding(deadline, target)
    }

    /// Enroll one run incarnation, mirroring `Control.EnrollRun`.
    pub fn enroll_run(
        &self,
        target: &tailnet_domain::RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String> {
        enroll::enroll_run(
            &self.policy,
            &self.provider(),
            target,
            recheck,
            consume,
            deadline,
        )
    }

    /// Read the local node's authoritative Tailscale status via the CLI.
    pub fn cli_status(&self, deadline: Instant) -> Result<native::CliStatus, String> {
        native::cli_status(&self.exec, &self.cli, deadline)
    }

    /// Resolve the advertised host and IPv4 address used by Forgejo.
    pub fn cli_endpoint(&self, deadline: Instant) -> Result<native::CliEndpoint, String> {
        native::cli_endpoint(&self.exec, &self.cli, deadline)
    }
}

impl<E: Executor> TailnetControl for Control<E> {
    fn project(
        &self,
        req: &tailnet_domain::ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<tailnet_domain::ProjectView, String> {
        Control::project(self, req, cid, deadline)
    }

    fn run_binding(
        &self,
        target: &tailnet_domain::RunTarget,
        deadline: Instant,
    ) -> Result<tailnet_domain::RunBinding, String> {
        Control::run_binding(self, target, deadline)
    }

    fn enroll_run(
        &self,
        target: &tailnet_domain::RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String> {
        Control::enroll_run(self, target, recheck, consume, deadline)
    }
}

/// Enforce the 64 KiB wire response cap, mirroring `writeTailnetResponse`
/// (oversize is a 502, i.e. unconfirmed here).
fn capped(body: Vec<u8>) -> Result<Vec<u8>, String> {
    if body.len() > wire::WIRE_RESPONSE_LIMIT {
        return Err(wire::err_unconfirmed());
    }
    Ok(body)
}
