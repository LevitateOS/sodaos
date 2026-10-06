use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_domain::{valid_project_id, ERR_CONFLICT, ERR_INVALID, ERR_UNCONFIRMED};
use crate::tailnet_files::open_runtime_project;
use crate::tailnet_runtime::{project_container, project_run, ProjectRun};

use super::{is_companion_run_fresh, stage_error, Companion, TailnetControl, RUNTIME_ROOT};

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    pub(crate) fn retire_previous_companion(
        &self,
        id: &str,
        run: &ProjectRun,
        previous: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        if previous.target.run.is_empty() {
            return Ok(());
        }
        if previous.target.project != id || previous.target.container != run.target.container {
            return Err(stage_error(
                "project runtime changed",
                ERR_CONFLICT.to_string(),
            ));
        }
        self.stop_tailnet_run(previous, deadline)
            .map_err(|e| stage_error("companion stop unconfirmed", e))
    }

    pub(crate) fn reconcile_previous_run(
        &self,
        id: &str,
        run: &ProjectRun,
        previous: &ProjectRun,
        current_err: Option<String>,
        deadline: Instant,
    ) -> Result<bool, String> {
        if let Some(cause) = &current_err {
            if cause != "runtime record not found" {
                return Err(stage_error("companion runtime unconfirmed", cause.clone()));
            }
        }
        let fresh = is_companion_run_fresh(current_err.as_ref(), previous, run);
        if !fresh && previous != run {
            return Err(stage_error(
                "project runtime changed",
                ERR_CONFLICT.to_string(),
            ));
        }
        if fresh {
            self.retire_previous_companion(id, run, previous, deadline)?;
        }
        Ok(fresh)
    }

    pub fn stop_tailnet(&self, id: &str, deadline: Instant) -> Result<(), String> {
        if !valid_project_id(id) {
            return Err(ERR_INVALID.to_string());
        }
        // The unit is conditioned on an existing runtime record; no scan.
        let files = open_runtime_project(RUNTIME_ROOT, id, deadline)?;
        let run = files.current()?;
        if run.target.project != id {
            return Err(ERR_CONFLICT.to_string());
        }
        self.stop_tailnet_run(&run, deadline)
    }

    pub(crate) fn confirm_stopped_resolver(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        match project_run(&self.exec, &run.target.project, deadline) {
            Ok(current) if current == *run => {}
            _ => return Ok(()),
        }
        let data = std::fs::read(&run.resolver).map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if data.len() > 16384
            || String::from_utf8_lossy(&data)
                .to_lowercase()
                .contains("tailscale")
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        Ok(())
    }

    pub(crate) fn stop_tailnet_run(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let id = run.target.project.clone();
        match project_container(&self.exec, &id, false, deadline) {
            Ok(cid) if cid == run.target.container => {}
            _ => return Err(ERR_CONFLICT.to_string()),
        }
        let rec = self.inspect_companion(run, deadline)?;
        if !rec.running {
            return Ok(());
        }
        let result = self.logout_and_stop_companion(run, &rec.id, deadline);
        let after = self
            .inspect_companion(run, deadline)
            .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if after.id != rec.id || after.running {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        self.confirm_stopped_resolver(run, deadline)?;
        result
    }

    pub(crate) fn logout_and_stop_companion(
        &self,
        run: &ProjectRun,
        id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let logout_end = Instant::now() + Duration::from_secs(5);
        let logout = self.companion_cli(run, &["logout".to_string()], logout_end);
        let stop = self.runtime_podman(
            &["stop".to_string(), "--time=8".to_string(), id.to_string()],
            deadline,
        );
        if stop.is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        logout.map(|_| ()).map_err(|_| ERR_UNCONFIRMED.to_string())
    }
}
