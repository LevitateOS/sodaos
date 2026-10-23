//! Member access uses the fixed guest helper while running and the same file
//! owner against a stopped mounted root. The mutation gate excludes Start/Join.
use crate::project::{Executor, Runtime};
use std::path::Path;
use std::time::Instant;

#[path = "../../../../cmd/soda-project-terminal/src/project_member_access.rs"]
#[allow(dead_code)]
pub(crate) mod files;

pub(crate) fn operate<E: Executor>(
    runtime: &Runtime<E>,
    input: &files::Request,
    deadline: Instant,
) -> Result<files::State, String> {
    if !crate::domain::valid_id(&input.project) {
        return Err("invalid Project identity".into());
    }
    let cid = runtime.project_container(&input.project, false, deadline)?;
    let (environment, _) = runtime.inspect(&input.project, deadline)?;
    if environment.running {
        let body = serde_json::to_vec(input).map_err(|_| "invalid member access request")?;
        let out = runtime.podman(
            &body,
            &[
                "--remote=false",
                "exec",
                "--interactive",
                &cid,
                "/usr/libexec/soda/project-account",
                "--access",
            ],
            deadline,
        )?;
        let state: files::State =
            crate::json::decode_strict_as(&out).map_err(|_| "native member access unconfirmed")?;
        if state.project != input.project
            || state.identity != input.identity
            || state.login != input.login
        {
            return Err("member identity changed".into());
        }
        return Ok(state);
    }
    let mount = runtime.podman(&[], &["--remote=false", "mount", &cid], deadline)?;
    let result = (|| {
        let mount = std::str::from_utf8(&mount)
            .map_err(|_| "Project root unavailable")?
            .trim();
        if !mount.starts_with('/') || mount.lines().count() != 1 {
            return Err("Project root unavailable".into());
        }
        files::operate(Path::new(mount), input, false)
    })();
    let unmounted = runtime.podman(&[], &["--remote=false", "unmount", &cid], deadline);
    let state = result?;
    unmounted.map_err(|_| "Project root unmount unconfirmed")?;
    let (after, _) = runtime.inspect(&input.project, deadline)?;
    if after.running {
        return Err("Project started during offline access operation".into());
    }
    Ok(state)
}
