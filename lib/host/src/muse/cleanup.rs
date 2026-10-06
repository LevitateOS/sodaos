//! Execution-state cleanup owner: guest state-dir removal,
//! container-aware invocation, mount retirement and execution-file
//! retirement with original precedence and ordering.

use std::time::Instant;

use super::{state_container, MuseHooks, MuseRuntime};
use crate::project::Executor;
use crate::terminal::{self, Binding};

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    // ---------- state cleanup (muse_state_linux.go) ----------

    /// `MuseRuntime.cleanupExecutionState`: remove the guest state dir.
    pub fn cleanup_execution_state(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = state_container(binding)?;
        if container.is_empty() {
            return Err(terminal::err_denied());
        }
        match self.invoke_state(
            binding,
            &[
                "container".to_string(),
                "exists".to_string(),
                container.clone(),
            ],
            deadline,
        ) {
            Ok(_) => {}
            Err(err) if terminal::exit_code_of(&err) == Some(1) => return Ok(()),
            Err(_) => return Err(terminal::err_uncertain()),
        }
        let state = format!("/tmp/soda-muse-state-{}", binding.id);
        let owner = format!("--user={}:{}", binding.uid, binding.gid);
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner.clone(),
                container.clone(),
                "/usr/bin/rm".to_string(),
                "--recursive".to_string(),
                "--force".to_string(),
                "--".to_string(),
                state.clone(),
            ],
            deadline,
        )
        .map_err(|_| terminal::err_uncertain())?;
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner,
                container,
                "/usr/bin/test".to_string(),
                "!".to_string(),
                "-e".to_string(),
                state,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }

    fn invoke_state(
        &self,
        binding: &Binding,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if binding.scope == "muse-project" && !binding.child_id.is_empty() {
            let mut full = vec!["/usr/bin/podman".to_string(), "--remote=false".to_string()];
            full.extend(args.iter().cloned());
            return self.guest(&binding.project, &[], &full, deadline);
        }
        self.podman(&[], args, deadline)
    }

    fn retire_mount(&self, container: &str, path: &str, deadline: Instant) -> Result<(), String> {
        if self
            .guest_refs(container, &[], &["/usr/bin/umount", path], deadline)
            .is_ok()
        {
            return Ok(());
        }
        if self
            .guest_refs(
                container,
                &[],
                &["/usr/bin/test", "!", "-e", path],
                deadline,
            )
            .is_ok()
        {
            return Ok(());
        }
        match self.guest_refs(
            container,
            &[],
            &["/usr/bin/mountpoint", "--quiet", path],
            deadline,
        ) {
            Err(err) if terminal::exit_code_of(&err) == Some(32) => Ok(()),
            _ => Err(terminal::err_uncertain()),
        }
    }

    pub(in crate::muse) fn retire_execution_files(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        self.cleanup_execution_state(binding, deadline)?;
        if !binding.credential_root.contains("/nested/") {
            self.retire_mount(&binding.project, &binding.credential_root, deadline)?;
        }
        self.guest_refs(
            &binding.project,
            &[],
            &[
                "/usr/bin/rm",
                "--recursive",
                "--force",
                "--",
                &binding.credential_root,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }
}
