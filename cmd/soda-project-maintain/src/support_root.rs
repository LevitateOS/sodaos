//! Exact retained-root execution for explicit support maintenance.
use crate::{
    command,
    filesystem::Tool,
    project::{inspect_project, Observation},
};
use soda_host::project::Native;
use soda_host::terminal::{stopped_namespace_argv, Service, TerminalInspection};
use std::time::{Duration, Instant};

pub(crate) fn service() -> Service<Native> {
    Service {
        exec: Native,
        codex_harness: String::new(),
        codex_harness_sha256: String::new(),
        muse_harness: String::new(),
        muse_harness_sha256: String::new(),
        muse_harness_version: String::new(),
    }
}

pub(crate) struct Root {
    pub(crate) observation: Observation,
    inspection: TerminalInspection,
    mounted: Option<String>,
}
impl Root {
    pub(crate) fn open(project: &str, deadline: Instant) -> Result<Self, String> {
        let observation = inspect_project(&format!("soda-{project}"), project, deadline)?;
        let inspection = service().project_inspection(project, false, deadline)?;
        if observation.id != inspection.id
            || observation.running != inspection.running
            || (!observation.running && observation.pid != 0)
        {
            return Err("Project identity or runtime state changed".into());
        }
        let mut root = Self {
            observation,
            inspection,
            mounted: None,
        };
        if !root.observation.running {
            let data = command::podman(&["mount", &root.observation.id], deadline)?;
            // Mount cleanup is required even when its observer is malformed.
            root.mounted = Some(String::new());
            let path = std::str::from_utf8(&data)
                .map_err(|_| "invalid retained root mount")?
                .trim();
            if data.len() > 4096 || path.chars().any(char::is_control) {
                return Err("invalid retained root mount".into());
            }
            stopped_namespace_argv(&root.inspection, path, &["/usr/bin/true"])?;
            root.mounted = Some(path.into());
        }
        root.confirm(deadline)?;
        Ok(root)
    }
    pub(crate) fn confirm(&self, deadline: Instant) -> Result<(), String> {
        let actual = inspect_project(&self.observation.id, &self.observation.project, deadline)?;
        let by_name = service().project_inspection(&self.observation.project, false, deadline)?;
        if actual != self.observation || by_name != self.inspection {
            return Err(
                "Project incarnation or runtime state changed; admission remains closed".into(),
            );
        }
        Ok(())
    }
    fn argv(
        &self,
        script: &str,
        args: &[&str],
        input: bool,
        private_runtime: bool,
    ) -> Result<(&'static str, Vec<String>), String> {
        let mut command = vec!["/usr/bin/bash", "-ceu", script, "terminal-support"];
        command.extend_from_slice(args);
        if let Some(root) = &self.mounted {
            let mut argv = stopped_namespace_argv(&self.inspection, root, &command)?;
            if private_runtime {
                argv.splice(
                    0..0,
                    [
                        "--mount",
                        "--propagation=private",
                        "--pid",
                        "--fork",
                        "--kill-child=SIGKILL",
                        "--mount-proc=/proc",
                    ]
                    .map(str::to_string),
                );
            }
            Ok(("/usr/bin/unshare", argv))
        } else {
            let mut argv: Vec<String> = ["--remote=false", "exec", "--user", "0:0"]
                .map(str::to_string)
                .into();
            if input {
                argv.push("--interactive".into());
            }
            argv.push(self.observation.id.clone());
            argv.extend(command.iter().map(|s| (*s).to_string()));
            Ok(("/usr/bin/podman", argv))
        }
    }
    pub(crate) fn run(
        &self,
        script: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.confirm(deadline)?;
        let script = if self.mounted.is_some() {
            format!("SODA_STOPPED=1\n{script}")
        } else {
            script.into()
        };
        let (program, argv) = self.argv(&script, args, false, false)?;
        let result = command::run(
            program,
            &argv.iter().map(String::as_str).collect::<Vec<_>>(),
            deadline,
        );
        self.confirm(deadline)?;
        result
    }
    pub(crate) fn install(
        &self,
        script: &str,
        digest: &str,
        old: &str,
        source: &Tool,
        deadline: Instant,
    ) -> Result<(), String> {
        self.confirm(deadline)?;
        let script = if self.mounted.is_some() {
            format!("{}\n{script}", include_str!("stopped_runtime.sh"))
        } else {
            script.into()
        };
        let (program, argv) = self.argv(
            &script,
            &["install", digest, old],
            true,
            self.mounted.is_some(),
        )?;
        command::streamed(
            vec![("project-terminal".into(), source.fd, source.size)],
            program,
            &argv.iter().map(String::as_str).collect::<Vec<_>>(),
            deadline,
        )?;
        self.confirm(deadline)
    }
    pub(crate) fn close(mut self) -> Result<(), String> {
        self.unmount()
    }
    fn unmount(&mut self) -> Result<(), String> {
        if self.mounted.take().is_some() {
            command::podman(
                &["unmount", &self.observation.id],
                Instant::now() + Duration::from_secs(3),
            )
            .map_err(|_| {
                "retained root unmount unconfirmed; admission remains closed".to_string()
            })?;
        }
        Ok(())
    }
}
impl Drop for Root {
    fn drop(&mut self) {
        let _ = self.unmount();
    }
}
