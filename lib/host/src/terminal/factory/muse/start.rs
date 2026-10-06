use super::commands::muse_start_gate_script;
use super::paths::factory_muse_binding;
use crate::project::Executor;
use crate::terminal::factory::run::MAX_FACTORY_PROMPT;
use crate::terminal::{self, Lease, Service};
use std::time::Instant;

impl<E: Executor> Service<E> {
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
}
