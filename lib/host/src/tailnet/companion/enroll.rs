use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_domain::ERR_UNCONFIRMED;
use crate::tailnet_files::{retire_pending_run_key, retire_run_key, write_run_key, Root};
use crate::tailnet_runtime::{recheck_project_run, ProjectRun};

use super::{stage_error, Companion, TailnetControl};

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    fn enroll_companion(
        &self,
        run: &ProjectRun,
        root: &Root,
        deadline: Instant,
    ) -> Result<(), String> {
        let rec = self
            .inspect_companion(run, deadline)
            .map_err(|_| stage_error("enrollment unconfirmed", ERR_UNCONFIRMED.to_string()))?;
        if !rec.execs.is_empty() {
            return Err(stage_error(
                "enrollment unconfirmed",
                ERR_UNCONFIRMED.to_string(),
            ));
        }
        retire_pending_run_key(root, run).map_err(|e| stage_error("enrollment unconfirmed", e))?;
        let recheck = |at: Instant| recheck_project_run(&self.exec, run, at);
        let consume = |at: Instant, key: &str| self.consume_run_key(run, root, key, at);
        self.tailnet
            .enroll_run(&run.target, &recheck, &consume, deadline)
            .map_err(|e| stage_error("enrollment unconfirmed", e))
    }

    pub(crate) fn ensure_companion_enrolled(
        &self,
        run: &ProjectRun,
        root: &Root,
        admission: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        let has_node = self.wait_companion_node(run, deadline)?;
        if has_node || !admission {
            return Ok(());
        }
        self.enroll_companion(run, root, deadline)
    }

    fn consume_run_key(
        &self,
        run: &ProjectRun,
        root: &Root,
        key: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let file = write_run_key(root, run, key)?;
        let cli_err = self
            .companion_cli(
                run,
                &[
                    "up".to_string(),
                    "--json".to_string(),
                    "--timeout=5s".to_string(),
                    "--auth-key=file:/run/soda-enrollment/key".to_string(),
                    format!("--hostname=soda-{}", run.target.project),
                    "--accept-dns=true".to_string(),
                    "--accept-routes=false".to_string(),
                    "--ssh=false".to_string(),
                    "--advertise-exit-node=false".to_string(),
                ],
                deadline,
            )
            .err();
        // A cancelled Podman observer does not cancel the native exec. The CLI
        // has its own five-second deadline. Do not retire its input until all
        // execs have ended.
        let end = Instant::now() + Duration::from_secs(8);
        loop {
            if let Ok(rec) = self.inspect_companion(run, end) {
                if rec.execs.is_empty() {
                    retire_run_key(root, &file).map_err(|_| ERR_UNCONFIRMED.to_string())?;
                    return cli_err.map_or(Ok(()), Err);
                }
            }
            if Instant::now() >= end {
                return Err(ERR_UNCONFIRMED.to_string());
            }
            std::thread::sleep(Duration::from_millis(100));
        }
    }
}
