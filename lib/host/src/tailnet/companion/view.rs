use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_domain::{project_status, ProjectRequest, ProjectView};
use crate::tailnet_runtime::{project_run, project_running, recheck_project_run, ProjectRun};

use super::{apply_companion_idle_state, Companion, TailnetControl};

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    pub(crate) fn queue_project_tailnet_disable(
        &self,
        project: &str,
        view: &mut ProjectView,
        deadline: Instant,
    ) {
        // Queue cancellation first. Also handle an owned orphan whose unit is
        // already inactive: stopping an inactive systemd unit does not execute
        // ExecStop.
        let args = vec![
            "stop".to_string(),
            "--no-block".to_string(),
            format!("soda-tailnet@{project}.service"),
        ];
        if self
            .runtime_command("/usr/bin/systemctl", &args, deadline)
            .is_ok()
        {
            view.outcome = "queued".to_string();
        }
        let end = Instant::now() + Duration::from_secs(15);
        if self.stop_tailnet(project, end).is_err() {
            view.outcome = "runtime-unconfirmed".to_string();
        }
    }

    pub(crate) fn mark_stopped_project(
        &self,
        project: &str,
        view: &mut ProjectView,
        deadline: Instant,
    ) {
        // Distinguish a confirmed stopped parent from failed observation.
        if let Ok(false) = project_running(&self.exec, project, deadline) {
            view.state = "stopped".to_string();
        }
    }

    pub(crate) fn queue_project_tailnet_start(
        &self,
        req: &ProjectRequest,
        view: &mut ProjectView,
        deadline: Instant,
    ) -> bool {
        if req.action == "inspect" || !view.enabled {
            return true;
        }
        let args = vec![
            "start".to_string(),
            "--no-block".to_string(),
            format!("soda-tailnet@{}.service", req.project),
        ];
        if self
            .runtime_command("/usr/bin/systemctl", &args, deadline)
            .is_err()
        {
            return false;
        }
        view.outcome = "queued".to_string();
        true
    }

    pub(crate) fn observe_companion_status(
        &self,
        run: &ProjectRun,
        view: &mut ProjectView,
        deadline: Instant,
    ) {
        let binding = match self.tailnet.run_binding(&run.target, deadline) {
            Ok(binding) => binding,
            Err(_) => return,
        };
        let status = match self.companion_cli(
            run,
            &[
                "status".to_string(),
                "--json".to_string(),
                "--peers=false".to_string(),
            ],
            deadline,
        ) {
            Ok(body) => body,
            Err(_) => return,
        };
        let prefs =
            match self.companion_cli(run, &["debug".to_string(), "prefs".to_string()], deadline) {
                Ok(body) => body,
                Err(_) => return,
            };
        let (state, addresses, dns) = match project_status(&status, &prefs, &binding) {
            Ok(triple) => triple,
            Err(_) => return,
        };
        if recheck_project_run(&self.exec, run, deadline).is_err() {
            return;
        }
        view.state = state;
        view.addresses = addresses;
        view.dns_name = dns;
    }

    pub fn observe_project_tailnet(
        &self,
        req: &ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<ProjectView, String> {
        let mut view = self.tailnet.project(req, cid, deadline)?;
        if self.image.is_empty() {
            return Ok(view);
        }
        if req.action == "disable" {
            self.queue_project_tailnet_disable(&req.project, &mut view, deadline);
        }
        let run = match project_run(&self.exec, &req.project, deadline) {
            Ok(run) => run,
            Err(_) => {
                self.mark_stopped_project(&req.project, &mut view, deadline);
                return Ok(view);
            }
        };
        if !self.queue_project_tailnet_start(req, &mut view, deadline) {
            return Ok(view);
        }
        let rec = match self.inspect_companion(&run, deadline) {
            Ok(rec) => rec,
            Err(_) => return Ok(view),
        };
        if !apply_companion_idle_state(&mut view, rec.running) {
            return Ok(view);
        }
        self.observe_companion_status(&run, &mut view, deadline);
        Ok(view)
    }
}
