use super::binding::RunPathsFn;
use super::native::shell_quote;
use super::run::{valid_factory_role, valid_preparation_id};
use crate::domain;
use crate::project::Executor;
use crate::terminal::{self, Binding, Service, KIND_FACTORY};
use std::time::Instant;

/// One output slice bound (`24*1024-256`).
pub const MAX_FACTORY_OUTPUT_READ: i64 = 24 * 1024 - 256;
/// Trailing attach window (`256*1024`).
pub const MAX_FACTORY_OUTPUT_WINDOW: i64 = 256 * 1024;
/// Output cursor bound (`256<<20`).
pub const MAX_FACTORY_OUTPUT_OFFSET: i64 = 256 << 20;

/// One observed byte slice of recorded CLI output with its cursor.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCodexOutputSlice {
    pub data: Vec<u8>,
    pub total: i64,
    pub offset: i64,
    pub truncated: bool,
    pub gap: bool,
}

/// Family-neutral alias for the shared output-slice record.
pub type FactoryOutputSlice = FactoryCodexOutputSlice;

/// Shared output-cursor gate: adapters run this before binding
/// attestation so a bad cursor never shells out.
pub fn check_output_range(offset: i64, limit: i64) -> Result<(), String> {
    if !(0..=MAX_FACTORY_OUTPUT_OFFSET).contains(&offset)
        || !(1..=MAX_FACTORY_OUTPUT_READ).contains(&limit)
    {
        return Err(terminal::err_denied());
    }
    Ok(())
}

/// `FactoryCodexOutput` bounded read pipeline.
pub fn output_read_command(stdout: &str, start: i64, limit: i64) -> String {
    format!(
        "/usr/bin/tail -c +{} {} | /usr/bin/head -c {limit}\n",
        start.wrapping_add(1),
        shell_quote(stdout),
    )
}

/// `factoryOutputSize`: non-negative stat size from a short read.
pub fn factory_output_size(out: &[u8]) -> Option<i64> {
    let size = terminal::parse_go_int(String::from_utf8_lossy(out).trim())?;
    if size >= 0 && out.len() <= 64 {
        Some(size)
    } else {
        None
    }
}

impl<E: Executor> Service<E> {
    /// Shared output-binding core: scope/shape checks plus container
    /// attestation, returning the derived stdout path.
    pub(crate) fn factory_output_stdout(
        &self,
        project_id: &str,
        binding: &Binding,
        scope: &str,
        run_paths: RunPathsFn,
        deadline: Instant,
    ) -> Result<String, String> {
        if binding.kind != KIND_FACTORY
            || binding.scope != scope
            || !terminal::valid_terminal_id(&binding.id)
        {
            return Err(terminal::err_denied());
        }
        if !domain::valid_container_id(&binding.project)
            || !valid_factory_role(&binding.login)
            || !terminal::valid_terminal_id(&binding.invocation_id)
        {
            return Err(terminal::err_denied());
        }
        if !valid_preparation_id(&binding.child_id) {
            return Err(terminal::err_denied());
        }
        let (checkout, run_dir, _, _) = run_paths(&binding.login, &binding.child_id, &binding.id)
            .ok_or_else(terminal::err_denied)?;
        if checkout.is_empty() || terminal::clean_path(&binding.credential_root) != run_dir {
            return Err(terminal::err_denied());
        }
        let container = self.factory_project_container(project_id, true, deadline)?;
        if container != binding.project {
            return Err(terminal::err_stale());
        }
        Ok(format!("{run_dir}/stdout.log"))
    }

    /// Shared output core: one bounded slice at a byte cursor. The
    /// cursor gate stays in the caller (`check_output_range` runs
    /// before binding attestation, so a bad cursor never shells out).
    pub(crate) fn factory_output_window(
        &self,
        project: &str,
        stdout: &str,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<FactoryOutputSlice, String> {
        check_output_range(offset, limit)?;
        let mut total = 0i64;
        let stat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/stat".to_string(),
            "-c".to_string(),
            "%s".to_string(),
            stdout.to_string(),
        ];
        if let Ok(out) = self.run_podman(&[], &stat, deadline) {
            if let Some(size) = factory_output_size(&out) {
                total = size;
            }
        }
        if offset > total {
            return Ok(FactoryOutputSlice {
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
            return Ok(FactoryOutputSlice {
                total,
                offset: start,
                truncated,
                ..Default::default()
            });
        }
        let read = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            output_read_command(stdout, start, limit),
        ];
        let mut out = match self.run_podman(&[], &read, deadline) {
            Ok(out) => out,
            Err(_) => {
                return Ok(FactoryOutputSlice {
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
        Ok(FactoryOutputSlice {
            data: out,
            total,
            offset: start,
            truncated,
            ..Default::default()
        })
    }
}
