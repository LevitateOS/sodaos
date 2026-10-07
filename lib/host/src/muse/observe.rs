use std::time::Instant;

use super::{MuseCaller, MuseHooks, MuseInspection, MuseRuntime, MUSE_INSPECT};
use crate::domain;
use crate::json;
use crate::project::Executor;
use crate::terminal;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    pub(in crate::muse) fn podman(
        &self,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut full = vec!["--remote=false".to_string()];
        full.extend(args.iter().cloned());
        let refs: Vec<&str> = full.iter().map(|s| s.as_str()).collect();
        self.exec.run(stdin, "/usr/bin/podman", &refs, deadline)
    }

    pub(in crate::muse) fn guest(
        &self,
        container: &str,
        stdin: &[u8],
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut full = vec![
            "exec".to_string(),
            "--interactive".to_string(),
            container.to_string(),
        ];
        full.extend(args.iter().cloned());
        self.podman(stdin, &full, deadline)
    }

    pub(in crate::muse) fn guest_refs(
        &self,
        container: &str,
        stdin: &[u8],
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.guest(
            container,
            stdin,
            &args.iter().map(|s| s.to_string()).collect::<Vec<_>>(),
            deadline,
        )
    }

    /// `MuseRuntime.inspect`: running, labeled, unprivileged container.
    pub fn inspect(&self, container: &str, deadline: Instant) -> Result<MuseCaller, String> {
        let body = self.podman(
            &[],
            &[
                "inspect".to_string(),
                "--format".to_string(),
                MUSE_INSPECT.to_string(),
                container.to_string(),
            ],
            deadline,
        );
        let mut out = MuseCaller::default();
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        let inspection: MuseInspection =
            json::decode_strict_as(&body).map_err(|_| terminal::err_denied())?;
        let raw_pid = inspection.pid;
        if inspection.id != container
            || !inspection.running
            || raw_pid <= 0
            || raw_pid > i32::MAX as i64
            || !domain::valid_id(&inspection.project)
            || inspection.privileged
        {
            return Err(terminal::err_denied());
        }
        out.container = container.to_string();
        out.project = inspection.project;
        out.project_pid = raw_pid as i32;
        let _ = inspection.userns;
        Ok(out)
    }
}
