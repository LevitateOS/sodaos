use std::time::Instant;

use crate::project::Executor;
use crate::tailnet_companion::TailnetControl;
use crate::tailnet_domain;
use crate::tcontrol_enroll as enroll;

use super::Control;

impl<E: Executor> Control<E> {
    /// Inspect or mutate one project's tailnet selection, same semantics as
    /// Go `Control.Project`. `cid` is the project container id.
    pub fn project(
        &self,
        req: &tailnet_domain::ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<tailnet_domain::ProjectView, String> {
        self.policy.project(deadline, req, cid)
    }

    /// Native-only public-metadata projection for one run incarnation,
    /// mirroring `Control.RunBinding`.
    pub fn run_binding(
        &self,
        target: &tailnet_domain::RunTarget,
        deadline: Instant,
    ) -> Result<tailnet_domain::RunBinding, String> {
        self.policy.run_binding(deadline, target)
    }

    /// Enroll one run incarnation, mirroring `Control.EnrollRun`.
    pub fn enroll_run(
        &self,
        target: &tailnet_domain::RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String> {
        enroll::enroll_run(
            &self.policy,
            &self.provider(),
            target,
            recheck,
            consume,
            deadline,
        )
    }
}

impl<E: Executor> TailnetControl for Control<E> {
    fn project(
        &self,
        req: &tailnet_domain::ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<tailnet_domain::ProjectView, String> {
        Control::project(self, req, cid, deadline)
    }

    fn run_binding(
        &self,
        target: &tailnet_domain::RunTarget,
        deadline: Instant,
    ) -> Result<tailnet_domain::RunBinding, String> {
        Control::run_binding(self, target, deadline)
    }

    fn enroll_run(
        &self,
        target: &tailnet_domain::RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String> {
        Control::enroll_run(self, target, recheck, consume, deadline)
    }
}
