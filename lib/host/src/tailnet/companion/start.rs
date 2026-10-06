use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_domain::{project_has_node, ProjectRequest, RunBinding, ERR_UNAVAILABLE};
use crate::tailnet_files::{
    open_runtime_project, validate_run_resolver, write_companion_id, Root, RunFiles,
};
use crate::tailnet_runtime::{
    companion_create_args, project_container, project_run, recheck_project_run, ProjectRun,
};

use super::{stage_error, Companion, TailnetControl, RUNTIME_ROOT};

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    /// Saved policy intent without requiring companion-specific runtime
    /// readiness. Missing policy means off; malformed state fails safely.
    pub(crate) fn project_tailnet_enabled(
        &self,
        project: &str,
        cid: &str,
        deadline: Instant,
    ) -> Result<bool, String> {
        if let Some(check) = &self.enabled_check {
            return check(project, cid, deadline);
        }
        let view = self.tailnet.project(
            &ProjectRequest {
                project: project.to_string(),
                action: "inspect".to_string(),
                revision: String::new(),
                binding: String::new(),
                confirm_id: String::new(),
            },
            cid,
            deadline,
        )?;
        Ok(view.enabled)
    }

    pub(crate) fn should_start_tailnet(&self, id: &str, deadline: Instant) -> Result<bool, String> {
        // `tailnet` is a generic bound, never nil; only the image gate applies.
        if self.image.is_empty() {
            return Ok(false);
        }
        // Off/unconfigured policy is a clean no-op before companion-specific
        // runtime readiness. An unresolvable container falls through so enabled
        // startup races still apply.
        if let Ok(cid) = project_container(&self.exec, id, false, deadline) {
            match self.project_tailnet_enabled(id, &cid, deadline) {
                Err(e) => return Err(stage_error("Tailnet policy unconfirmed", e)),
                Ok(false) => return Ok(false),
                Ok(true) => {}
            }
        }
        Ok(true)
    }

    fn wait_project_runtime(&self, id: &str, deadline: Instant) -> Result<ProjectRun, String> {
        let end = (Instant::now() + Duration::from_secs(10)).min(deadline);
        loop {
            match project_run(&self.exec, id, end) {
                Ok(run) => return Ok(run),
                Err(e) => {
                    if Instant::now() >= end {
                        return Err(stage_error("project runtime not ready", e));
                    }
                    std::thread::sleep(Duration::from_millis(100));
                }
            }
        }
    }

    fn admit_tailnet_run(
        &self,
        id: &str,
        deadline: Instant,
    ) -> Result<(ProjectRun, RunBinding), String> {
        let run = self.wait_project_runtime(id, deadline)?;
        let binding = self
            .tailnet
            .run_binding(&run.target, deadline)
            .map_err(|e| stage_error("Tailnet policy unconfirmed", e))?;
        Ok((run, binding))
    }

    pub(crate) fn prepare_companion_state(
        &self,
        id: &str,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(RunFiles, Root, bool), String> {
        let files = open_runtime_project(RUNTIME_ROOT, id, deadline)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        let (previous, current_err) = match files.current() {
            Ok(previous) => (previous, None),
            Err(e) => (ProjectRun::default(), Some(e)),
        };
        let fresh = self.reconcile_previous_run(id, run, &previous, current_err, deadline)?;
        let read = |path: &str| std::fs::read(path).map_err(|_| ERR_UNAVAILABLE.to_string());
        let stat = |path: &str| std::fs::metadata(path).map_err(|_| ERR_UNAVAILABLE.to_string());
        validate_run_resolver(run, Box::new(read), Box::new(stat), fresh)
            .map_err(|e| stage_error("project runtime not ready", e))?;
        let root = files
            .prepare(run, fresh)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok((files, root, fresh))
    }

    fn create_fresh_companion(
        &self,
        files: &RunFiles,
        root: &Root,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        files
            .save_current(run)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        recheck_project_run(&self.exec, run, deadline)
            .map_err(|e| stage_error("project runtime changed", e))?;
        let args = companion_create_args(run, &self.image)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        let created = self
            .runtime_command("/usr/bin/podman", &args, deadline)
            .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        let id = String::from_utf8_lossy(&created).trim().to_string();
        write_companion_id(root, &id)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok(())
    }

    pub(crate) fn start_companion_if_stopped(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        if !rec.running {
            recheck_project_run(&self.exec, run, deadline)
                .map_err(|e| stage_error("project runtime changed", e))?;
            self.runtime_podman(&["start".to_string(), rec.id.clone()], deadline)
                .map_err(|e| stage_error("companion startup unconfirmed", e))?;
        }
        Ok(())
    }

    fn activate_companion_container(
        &self,
        files: &RunFiles,
        root: &Root,
        run: &ProjectRun,
        fresh: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        if fresh {
            self.create_fresh_companion(files, root, run, deadline)?;
        }
        self.start_companion_if_stopped(run, deadline)
    }

    pub(crate) fn wait_companion_node(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<bool, String> {
        let end = (Instant::now() + Duration::from_secs(8)).min(deadline);
        let args = vec![
            "status".to_string(),
            "--json".to_string(),
            "--peers=false".to_string(),
        ];
        loop {
            match self.companion_cli(run, &args, end) {
                Ok(body) => {
                    return project_has_node(&body)
                        .map_err(|e| stage_error("companion status unavailable", e));
                }
                Err(_) => {
                    if Instant::now() >= end {
                        return Err(stage_error(
                            "companion status unavailable",
                            ERR_UNAVAILABLE.to_string(),
                        ));
                    }
                    std::thread::sleep(Duration::from_millis(100));
                }
            }
        }
    }

    pub(crate) fn finalize_companion_run(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<String, String> {
        recheck_project_run(&self.exec, run, deadline)
            .map_err(|e| stage_error("project runtime changed", e))?;
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|e| stage_error("companion runtime unconfirmed", e))?;
        Ok(rec.id)
    }

    /// Prepares one exact run. Only systemd/native explicit activation calls
    /// it; HTTP reads and browser focus never enter this path.
    pub fn start_tailnet(&self, id: &str, deadline: Instant) -> Result<String, String> {
        let start = self.should_start_tailnet(id, deadline)?;
        if !start {
            return Ok(String::new());
        }
        let (run, binding) = self.admit_tailnet_run(id, deadline)?;
        if !binding.enabled {
            return Ok(String::new());
        }
        let (files, root, fresh) = self.prepare_companion_state(id, &run, deadline)?;
        self.activate_companion_container(&files, &root, &run, fresh, deadline)?;
        self.ensure_companion_enrolled(&run, &root, binding.admission, deadline)?;
        self.finalize_companion_run(&run, deadline)
    }
}
