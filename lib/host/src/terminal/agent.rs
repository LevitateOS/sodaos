use std::time::Instant;

use crate::account::AGENT_PROGRAM;
use crate::domain;
use crate::project::Executor;

use super::{
    err_denied, err_stale, parse_go_int, terminal_target_ready, TerminalInspection,
    TERMINAL_INSPECT,
};

// ---------- terminal service ----------

/// Broker custody callback behind `end` reconciliation.
pub type EndIdentityHook<'a> = &'a dyn Fn(i64, &str) -> Result<(), String>;

/// Privileged project-terminal executor (`terminal.Service`).
/// `EndIdentity` is passed per call so the service stays dependency-free.
pub struct Service<E> {
    pub exec: E,
    pub codex_harness: String,
    pub codex_harness_sha256: String,
    pub codex_harness_version: String,
    pub muse_harness: String,
    pub muse_harness_sha256: String,
    pub muse_harness_version: String,
}

/// `podman --remote=false inspect --format <terminalInspect> <name>`.
pub fn inspect_argv(name: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        TERMINAL_INSPECT.to_string(),
        name.to_string(),
    ]
}

/// `podman --remote=false container exists <target>`.
pub fn container_exists_argv(target: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "container".to_string(),
        "exists".to_string(),
        target.to_string(),
    ]
}

/// Fixed `podman exec` of the project terminal agent (`AgentExec` argv).
/// Python is gone: the agent is the native `project-terminal` binary.
pub fn agent_argv(container: &str, args: &[&str]) -> Vec<String> {
    let mut argv = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        AGENT_PROGRAM.to_string(),
    ];
    argv.extend(args.iter().map(|s| s.to_string()));
    argv
}

impl<E: Executor> Service<E> {
    fn podman(&self, stdin: &[u8], args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
        self.exec.run(stdin, "/usr/bin/podman", args, deadline)
    }

    pub(in crate::terminal) fn podman_owned(
        &self,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        self.podman(stdin, &refs, deadline)
    }

    /// `Service.projectContainer`: fixed-name inspect with the full
    /// isolation gate. Native lifecycle may inspect stopped containers.
    pub fn project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if data.len() <= 4096 => data,
            _ => return Err("terminal inspection unavailable".to_string()),
        };
        let v = TerminalInspection::decode(&data)
            .map_err(|_| "invalid terminal inspection".to_string())?;
        if !terminal_target_ready(&v, id, require_running) {
            return Err("terminal target not ready or isolated".to_string());
        }
        Ok(v.id)
    }

    /// `Service.factoryProjectContainer`: supervised factory runs bind the
    /// exact container incarnation instead of the 262144-mapping profile.
    pub fn factory_project_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err(err_denied());
        }
        let argv = inspect_argv(&format!("soda-{id}"));
        let data = match self.podman_owned(&[], &argv, deadline) {
            Ok(data) if !data.is_empty() && data.len() <= 16384 => data,
            _ => return Err(err_stale()),
        };
        let v = TerminalInspection::decode(&data).map_err(|_| err_stale())?;
        if !domain::valid_container_id(&v.id)
            || v.project != id
            || v.privileged
            || v.userns != "private"
        {
            return Err(err_denied());
        }
        let Some(owner) = parse_go_int(&v.owner) else {
            return Err(err_denied());
        };
        if owner <= 0 {
            return Err(err_denied());
        }
        if require_running && !v.running {
            return Err(err_stale());
        }
        Ok(v.id)
    }
}
