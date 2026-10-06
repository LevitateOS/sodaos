use super::paths::{factory_muse_binding, factory_muse_paths, factory_muse_run_paths};
use crate::project::Executor;
use crate::terminal::factory::tcodex::{self, FactoryRun};
use crate::terminal::{self, Binding, Lease, Service};
use std::time::Instant;

impl<E: Executor> Service<E> {
    /// `Service.FactoryCodexWait` shape for Muse runs.
    pub fn factory_muse_wait(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<(i32, String), String> {
        let p = factory_muse_binding(lease)?;
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        self.factory_wait_result(
            &lease.execution_id,
            &project,
            &p.run_dir,
            &p.output,
            deadline,
        )
    }

    /// `Service.FactoryCodexValidate` shape for Muse runs.
    pub fn factory_muse_validate(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        factory_muse_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        self.factory_attest_live(&lease.project_id, binding, &lease.execution_id, deadline)
    }

    /// `Service.FactoryCodexStop` shape for Muse runs. Idempotent.
    pub fn factory_muse_stop(&self, lease: &Lease, deadline: Instant) -> Result<(), String> {
        let p = factory_muse_binding(lease)?;
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        self.factory_stop_confirmed(
            &binding.project,
            &lease.execution_id,
            &p.pid_file,
            &p.run_dir,
            deadline,
        )
    }

    /// `Service.FactoryCodexStopUnbound` shape for Muse runs.
    pub fn factory_muse_stop_unbound(
        &self,
        run: &FactoryRun,
        deadline: Instant,
    ) -> Result<(), String> {
        let p = factory_muse_paths(run)?;
        self.factory_stop_unbound_confirmed(
            &run.project,
            &run.id,
            &p.pid_file,
            &p.run_dir,
            deadline,
        )
    }

    /// `Service.FactoryCodexCapture` shape for Muse runs: echo the
    /// daemon-staged `auth.json` copy (never the CLI's live lookup file)
    /// for the in-process pfactory return path. The broker ignores these
    /// bytes for muse (borrow: forget on return), so capture failure here
    /// only reports host-side staging trouble, never rotation state.
    pub fn factory_muse_capture(
        &self,
        lease: &Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let p = factory_muse_binding(lease)?;
        let project = lease
            .binding
            .as_ref()
            .map(|b| b.project.clone())
            .unwrap_or_default();
        self.factory_capture_valid(&project, &p.credential, deadline)
    }

    /// `Service.FactoryCodexLive` shape for Muse runs.
    pub fn factory_muse_live(&self, binding: &Binding, deadline: Instant) -> bool {
        self.factory_live_scoped(binding, tcodex::FACTORY_SCOPE_MUSE, deadline)
    }

    /// `Service.FactoryCodexOutput` shape for Muse runs.
    pub fn factory_muse_output(
        &self,
        project_id: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<crate::terminal::factory::tfactory::FactoryOutputSlice, String> {
        crate::terminal::factory::tfactory::check_output_range(offset, limit)?;
        let stdout = self.factory_output_stdout(
            project_id,
            binding,
            tcodex::FACTORY_SCOPE_MUSE,
            factory_muse_run_paths,
            deadline,
        )?;
        self.factory_output_window(&binding.project, &stdout, offset, limit, deadline)
    }
}
