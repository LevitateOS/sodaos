//! Native access to a stopped retained root without starting its workloads.
use std::time::{Duration, Instant};

use super::{is_clean_absolute_path, terminal_target_ready, Service, TerminalInspection};
use crate::project::Executor;

/// Build a rooted native command using the already-admitted Project mappings.
/// The caller supplies a trusted command and holds Project admission exclusion.
/// No Project-controlled string is interpolated into shell source.
pub fn stopped_namespace_argv(
    inspection: &TerminalInspection,
    root: &str,
    command: &[&str],
) -> Result<Vec<String>, String> {
    if inspection.running
        || !crate::domain::valid_id(&inspection.project)
        || !terminal_target_ready(inspection, &inspection.project, false)
        || !is_clean_absolute_path(root)
        || root.chars().any(char::is_control)
        || !command
            .first()
            .is_some_and(|program| is_clean_absolute_path(program))
    {
        return Err("stopped Project root unavailable".to_string());
    }
    let uid_base = inspection.uid_map[0].split(':').nth(1).unwrap_or_default();
    let gid_base = inspection.gid_map[0].split(':').nth(1).unwrap_or_default();
    let mut args = vec![
        "--user".to_string(),
        format!("--map-users=0:{uid_base}:262144"),
        format!("--map-groups=0:{gid_base}:262144"),
        "--setuid=0".to_string(),
        "--setgid=0".to_string(),
        "--root".to_string(),
        root.to_string(),
        "--wd".to_string(),
        "/".to_string(),
    ];
    args.extend(command.iter().map(|arg| (*arg).to_string()));
    Ok(args)
}

fn mounted_root(data: &[u8]) -> Result<&str, String> {
    let path = std::str::from_utf8(data)
        .ok()
        .and_then(|text| text.strip_suffix('\n'))
        .filter(|text| !text.is_empty() && !text.chars().any(char::is_control))
        .filter(|text| is_clean_absolute_path(text));
    path.ok_or_else(|| "stopped Project mount unavailable".to_string())
}

impl<E: Executor> Service<E> {
    fn confirm_stopped(
        &self,
        inspection: &TerminalInspection,
        deadline: Instant,
    ) -> Result<(), String> {
        let current = self.project_inspection(&inspection.project, false, deadline)?;
        if &current != inspection || current.running {
            return Err("stopped Project identity or state changed".to_string());
        }
        Ok(())
    }

    /// Caller retains admission throughout mount, observation and cleanup.
    pub(super) fn in_stopped_root(
        &self,
        inspection: &TerminalInspection,
        command: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        // Validate identity/mappings before the first mount effect.
        stopped_namespace_argv(inspection, "/", command)?;
        let mounted = self.exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &["--remote=false", "mount", &inspection.id],
            deadline,
            4096,
            1024,
        );
        let result = (|| {
            let data = mounted?;
            let root = mounted_root(&data)?;
            self.confirm_stopped(inspection, deadline)?;
            let args = stopped_namespace_argv(inspection, root, command)?;
            let args: Vec<&str> = args.iter().map(String::as_str).collect();
            let output = self.exec.run_bounded(
                &[],
                "/usr/bin/unshare",
                &args,
                deadline,
                stdout_limit,
                stderr_limit,
            );
            self.confirm_stopped(inspection, deadline)?;
            output
        })();
        // Mount can succeed before its observer fails. Always reconcile this
        // exact target, including malformed replies, command errors and timeouts.
        // Cleanup has its own small bound after the observation deadline.
        let cleanup = self.exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &["--remote=false", "unmount", &inspection.id],
            Instant::now() + Duration::from_secs(2),
            1024,
            1024,
        );
        cleanup.map_err(|_| "stopped Project mount cleanup unavailable".to_string())?;
        result
    }
}
