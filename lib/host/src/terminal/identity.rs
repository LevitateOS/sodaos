use std::os::unix::fs::PermissionsExt;
use std::time::Instant;

use crate::domain;
use crate::json;
use crate::project::Executor;
use crate::sha256;

use super::{
    agent_argv, agent_program_hash, container_exists_argv, credential_valid, err_denied, err_stale,
    err_uncertain, exit_code_of, inspect_argv, now_unix, terminal_dimensions,
    terminal_target_ready, valid_terminal_id, Binding, Delivery, EndIdentityHook, Lease, Service,
    TerminalInspection, TerminalRequest, BROKER_RESPONSE_LIMIT, KIND_TERMINAL,
};

// ---------- identity broker calls (identity.go) ----------

/// Fixed agent request envelope (`identityRequest`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct IdentityRequest {
    pub action: String,
    pub delivery: Delivery,
    pub login: String,
    pub scope: String,
    pub cols: i64,
    pub rows: i64,
    pub source_hash: String,
    pub container: String,
    pub harness_sha256: String,
}

impl IdentityRequest {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"action\":");
        out.push_str(&json::quote(&self.action));
        out.push_str(",\"delivery\":");
        out.push_str(&self.delivery.encode());
        if !self.login.is_empty() {
            out.push_str(",\"login\":");
            out.push_str(&json::quote(&self.login));
        }
        if !self.scope.is_empty() {
            out.push_str(",\"scope\":");
            out.push_str(&json::quote(&self.scope));
        }
        if self.cols != 0 {
            out.push_str(",\"cols\":");
            out.push_str(&self.cols.to_string());
        }
        if self.rows != 0 {
            out.push_str(",\"rows\":");
            out.push_str(&self.rows.to_string());
        }
        if !self.source_hash.is_empty() {
            out.push_str(",\"source_hash\":");
            out.push_str(&json::quote(&self.source_hash));
        }
        if !self.container.is_empty() {
            out.push_str(",\"container\":");
            out.push_str(&json::quote(&self.container));
        }
        if !self.harness_sha256.is_empty() {
            out.push_str(",\"harness_sha256\":");
            out.push_str(&json::quote(&self.harness_sha256));
        }
        out.push('}');
        out
    }
}

fn terminal_reservation(lease: &Lease) -> bool {
    lease.kind == KIND_TERMINAL
        && valid_terminal_id(&lease.execution_id)
        && domain::valid_id(&lease.project_id)
        && lease.actor_id > 0
        && !lease.id.is_empty()
        && lease.generation > 0
}

fn terminal_binding(lease: &Lease) -> bool {
    let Some(b) = &lease.binding else {
        return false;
    };
    b.kind == KIND_TERMINAL
        && b.id == lease.execution_id
        && domain::valid_container_id(&b.project)
        && domain::valid_login(&b.login)
        && b.login != "root"
        && b.generation == lease.generation
}

/// `terminalLease`: reservation shape, plus binding (live) or deadline
/// window (preparing).
pub fn terminal_lease(lease: &Lease, preparing: bool, now: i64) -> bool {
    if !terminal_reservation(lease) {
        return false;
    }
    if preparing {
        return lease.binding.is_none()
            && matches!(lease.deadline, Some((secs, _)) if secs > now && secs <= now + 12 * 3600);
    }
    terminal_binding(lease)
}

fn terminal_preparation(
    lease: &Lease,
    login: &str,
    scope: &str,
    cols: i64,
    rows: i64,
    now: i64,
) -> bool {
    terminal_lease(lease, true, now)
        && domain::valid_login(login)
        && login != "root"
        && domain::valid_container_id(scope)
        && terminal_dimensions(cols, rows)
}

fn terminal_prepared(
    result: &Delivery,
    lease: &Lease,
    container: &str,
    login: &str,
    now: i64,
) -> bool {
    if !terminal_lease(&result.lease, false, now) {
        return false;
    }
    result.lease.id == lease.id
        && result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.login == login)
            .unwrap_or(false)
        && result
            .credential
            .as_ref()
            .map(|c| c.is_empty())
            .unwrap_or(true)
}

impl<E: Executor> Service<E> {
    /// `Service.identityCall`: fixed `broker` agent invocation over stdin.
    pub fn identity_call(
        &self,
        container: &str,
        request: &IdentityRequest,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let body = request.encode().into_bytes();
        let argv = agent_argv(container, &["broker"]);
        let out = match self.podman_owned(&body, &argv, deadline) {
            Ok(out) => out,
            Err(_) => return Err("managed Codex operation failed".to_string()),
        };
        if out.len() > BROKER_RESPONSE_LIMIT {
            return Err("invalid managed Codex response".to_string());
        }
        Delivery::decode(&out).map_err(|_| "invalid managed Codex response".to_string())
    }

    /// `Service.PrepareIdentity`: reserve one native terminal before the
    /// broker releases bytes.
    pub fn prepare_identity(
        &self,
        lease: &Lease,
        login: &str,
        scope: &str,
        cols: i64,
        rows: i64,
        deadline: Instant,
    ) -> Result<Binding, String> {
        let now = now_unix();
        if !terminal_preparation(lease, login, scope, cols, rows, now) {
            return Err(err_denied());
        }
        let container = self.project_container(&lease.project_id, true, deadline)?;
        let hash = agent_program_hash(0)?;
        let request = IdentityRequest {
            action: "prepare".to_string(),
            delivery: Delivery {
                lease: lease.clone(),
                credential: None,
            },
            login: login.to_string(),
            scope: scope.to_string(),
            cols,
            rows,
            source_hash: hash,
            container: container.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        if !terminal_prepared(&result, lease, &container, login, now_unix()) {
            return Err("managed terminal reservation differs".to_string());
        }
        result
            .lease
            .binding
            .clone()
            .ok_or_else(|| "managed terminal reservation differs".to_string())
    }

    fn identity_action(&self, action: &str, delivery: &Delivery) -> bool {
        match action {
            "start" => {
                delivery
                    .credential
                    .as_ref()
                    .map(|c| credential_valid(c))
                    .unwrap_or(false)
                    && self.codex_harness.starts_with('/')
                    && domain::valid_container_id(&self.codex_harness_sha256)
            }
            "validate" | "finish" | "stop" => delivery
                .credential
                .as_ref()
                .map(|c| c.is_empty())
                .unwrap_or(true),
            _ => false,
        }
    }

    /// `Service.Identity`: fixed model-session operations.
    pub fn identity(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !terminal_lease(&delivery.lease, false, now_unix()) {
            return Err(err_denied());
        }
        if !self.identity_action(action, delivery) {
            return Err(err_denied());
        }
        let (container, done) = self.identity_target(action, &delivery.lease, deadline)?;
        if done {
            return Ok(delivery.clone());
        }
        if action == "start" {
            self.identity_stage(&container, delivery, deadline)?;
        }
        let request = IdentityRequest {
            action: action.to_string(),
            delivery: delivery.clone(),
            harness_sha256: self.codex_harness_sha256.clone(),
            ..Default::default()
        };
        let result = self.identity_call(&container, &request, deadline)?;
        identity_result(action, delivery, &result)
    }

    /// `Service.managedEnd`: reconcile the managed lease behind an `end`
    /// before the launcher tears the terminal down.
    pub fn managed_end(
        &self,
        container: &str,
        request: &TerminalRequest,
        end_identity: Option<EndIdentityHook<'_>>,
        deadline: Instant,
    ) -> Result<(), String> {
        if request.action != "end" {
            return Ok(());
        }
        let lookup = IdentityRequest {
            action: "lookup".to_string(),
            delivery: Delivery {
                lease: Lease {
                    execution_id: request.id.clone(),
                    actor_id: request.identity,
                    ..Default::default()
                },
                credential: None,
            },
            login: request.login.clone(),
            ..Default::default()
        };
        let result = self.identity_call(container, &lookup, deadline)?;
        if result.lease.id.is_empty() {
            return Ok(());
        }
        let bound = result
            .lease
            .binding
            .as_ref()
            .map(|b| b.project == container && b.id == request.id)
            .unwrap_or(false);
        if !bound || result.lease.actor_id != request.identity {
            return Err(err_stale());
        }
        match end_identity {
            None => Err(err_denied()),
            Some(end) => end(request.identity, &result.lease.id),
        }
    }

    /// `Service.verifyIdentityHarness`: the staged host Codex bytes must
    /// match the pinned digest.
    pub fn verify_identity_harness(&self) -> Result<(), String> {
        let path = format!("{}/bin/codex", self.codex_harness.trim_end_matches('/'));
        // Go gates on `Lstat` regularity plus any exec bit before opening.
        let lstat = std::fs::symlink_metadata(&path)
            .map_err(|_| "verified Codex executable required".to_string())?;
        if !lstat.file_type().is_file() || lstat.permissions().mode() & 0o111 == 0 {
            return Err("verified Codex executable required".to_string());
        }
        let mut data = Vec::new();
        use std::io::Read;
        std::fs::File::open(&path)
            .map_err(|_| "verified Codex executable required".to_string())?
            .read_to_end(&mut data)
            .map_err(|e| format!("codex harness unreadable: {e}"))?;
        if sha256::hex_lower(&sha256::digest(&data)) != self.codex_harness_sha256 {
            return Err("codex harness digest differs".to_string());
        }
        Ok(())
    }

    fn identity_target(
        &self,
        action: &str,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        if action != "stop" {
            let target = self.project_container(&lease.project_id, true, deadline)?;
            let bound = lease
                .binding
                .as_ref()
                .map(|b| b.project.clone())
                .unwrap_or_default();
            if target != bound {
                return Err(err_stale());
            }
            return Ok((target, false));
        }
        self.identity_stop_target(lease, deadline)
    }

    fn identity_stop_target(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        let target = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        let argv = container_exists_argv(&target);
        if let Err(err) = self.podman_owned(&[], &argv, deadline) {
            if exit_code_of(&err) == Some(1) {
                return Ok((target, true));
            }
            return Err(err_uncertain());
        }
        let argv = inspect_argv(&target);
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err(err_uncertain()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_uncertain())?;
        if v.id != target || !terminal_target_ready(&v, &lease.project_id, false) {
            return Err(err_stale());
        }
        Ok((target, !v.running))
    }

    fn identity_stage(
        &self,
        container: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        self.verify_identity_harness()?;
        let request = IdentityRequest {
            action: "stage".to_string(),
            delivery: delivery.clone(),
            ..Default::default()
        };
        self.identity_call(container, &request, deadline)?;
        let path = format!(
            "/run/soda-terminals/{}/model/harness",
            delivery.lease.execution_id
        );
        self.stream_identity_harness(container, &path, deadline)
    }
}

/// `identityResult`: the agent must echo the binding; only `finish`
/// returns credential bytes.
pub fn identity_result(
    action: &str,
    delivery: &Delivery,
    result: &Delivery,
) -> Result<Delivery, String> {
    // Go returns the partial result alongside the error; the daemon drops
    // it on every error path, so only the verdict is preserved here.
    if result.lease.id != delivery.lease.id || result.lease.binding.is_none() {
        return Err(err_stale());
    }
    if result.lease.binding != delivery.lease.binding {
        return Err(err_stale());
    }
    if action == "finish" {
        match &result.credential {
            Some(c) if credential_valid(c) => {}
            _ => return Err(err_uncertain()),
        }
    } else if result
        .credential
        .as_ref()
        .map(|c| !c.is_empty())
        .unwrap_or(false)
    {
        return Err("unexpected credential response".to_string());
    }
    Ok(result.clone())
}
