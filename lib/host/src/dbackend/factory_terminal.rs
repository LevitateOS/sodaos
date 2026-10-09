use std::time::Instant;

use crate::daemon::broker::cv_lease_to_texec;
use crate::{
    factory, project,
    terminal::{self, factory::tcodex},
};

use super::TerminalSeam;

fn cv_run_to_tcodex(r: &factory::FactoryRun) -> crate::terminal::factory::tcodex::FactoryRun {
    crate::terminal::factory::tcodex::FactoryRun {
        deadline_raw: r.deadline.clone(),
        actor: r.actor,
        id: r.id.clone(),
        project: r.project.clone(),
        role: r.role.clone(),
        preparation: r.preparation.clone(),
        harness: r.harness.clone(),
        harness_vers: r.harness_vers.clone(),
        model: r.model.clone(),
        assignment: r.assignment.clone(),
        source_commit: r.source_commit.clone(),
        connection: r.connection.clone(),
    }
}

fn cv_slice_to_factory(
    s: &crate::terminal::factory::tcodex::FactoryCodexOutputSlice,
) -> factory::OutputSlice {
    factory::OutputSlice {
        data: s.data.clone(),
        total: s.total,
        offset: s.offset,
        truncated: s.truncated,
        gap: s.gap,
    }
}

// -- seam implementations --

impl factory::FactoryTerminal for TerminalSeam {
    fn harness_family(&self) -> String {
        self.pin_family().to_string()
    }
    fn harness_version(&self) -> String {
        if self.pin_family() == factory::FACTORY_HARNESS_MUSE {
            self.muse_harness_version.clone()
        } else {
            self.harness_version.clone()
        }
    }
    fn harness_sha256(&self) -> String {
        if self.pin_family() == factory::FACTORY_HARNESS_MUSE {
            self.muse_harness_sha256.clone()
        } else {
            self.harness_sha256.clone()
        }
    }
    fn reserve(
        &self,
        run: &factory::FactoryRun,
        lease: &factory::Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<terminal::Binding, factory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        let tlease = cv_lease_to_texec(lease);
        let out = if run.harness == factory::FACTORY_HARNESS_MUSE {
            service
                .factory_muse_reserve(&trun, &tlease, pin, max_secs, deadline)
                .map(|(binding, _paths)| binding)
        } else {
            service
                .factory_codex_reserve(&trun, &tlease, pin, max_secs, deadline)
                .map(|(binding, _paths)| binding)
        };
        out.map_err(factory::FactoryError::Msg)
    }
    fn start(
        &self,
        lease: &factory::Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_start(&tlease, credential, prompt, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_start(&tlease, credential, prompt, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn wait(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<(i64, String), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_wait(&tlease, deadline)
                .map(|(code, out)| (code as i64, out))
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_wait(&tlease, deadline)
                .map(|(code, out)| (code as i64, out))
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn stop(&self, lease: &factory::Lease, deadline: Instant) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_stop(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_stop(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn stop_unbound(
        &self,
        run: &factory::FactoryRun,
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let trun = cv_run_to_tcodex(run);
        if run.harness == factory::FACTORY_HARNESS_MUSE {
            service
                .factory_muse_stop_unbound(&trun, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_stop_unbound(&trun, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn capture(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_capture(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_capture(&tlease, deadline)
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn live(&self, binding: &terminal::Binding, deadline: Instant) -> bool {
        let service = self.service();
        if binding.scope == tcodex::FACTORY_SCOPE_MUSE {
            service.factory_muse_live(binding, deadline)
        } else {
            service.factory_codex_live(binding, deadline)
        }
    }
    fn output(
        &self,
        project: &str,
        binding: &terminal::Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<factory::OutputSlice, factory::FactoryError> {
        let service = self.service();
        if binding.scope == tcodex::FACTORY_SCOPE_MUSE {
            service
                .factory_muse_output(project, binding, offset, limit, deadline)
                .map(|s| cv_slice_to_factory(&s))
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_output(project, binding, offset, limit, deadline)
                .map(|s| cv_slice_to_factory(&s))
                .map_err(factory::FactoryError::Msg)
        }
    }
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), factory::FactoryError> {
        let service = self.service();
        service
            .factory_takeover_copy(project, recorded, role, preparation, member, run, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let service = self.service();
        service
            .factory_export_bundle(project, recorded, role, preparation, candidate, deadline)
            .map_err(factory::FactoryError::Msg)
    }
}

/// A lease takes the muse path when its binding carries the factory-muse
/// scope. Anything else (including unbound) stays on the codex path,
/// which denies what it does not recognize.
fn muse_scoped_lease(lease: &terminal::Lease) -> bool {
    lease
        .binding
        .as_ref()
        .is_some_and(|b| b.scope == tcodex::FACTORY_SCOPE_MUSE)
}

impl TerminalSeam {
    /// The seam owns no service state: it rebuilds the value service (pure
    /// configuration + the shared executor shape) per call. Service methods
    /// take `&self` and hold no interior state, so this is free.
    fn service(&self) -> terminal::Service<project::Native> {
        terminal::Service {
            exec: project::Native,
            codex_harness: self.harness.clone(),
            codex_harness_sha256: self.harness_sha256.clone(),
            codex_harness_version: self.harness_version.clone(),
            muse_harness: self.muse_harness.clone(),
            muse_harness_sha256: self.muse_harness_sha256.clone(),
            muse_harness_version: self.muse_harness_version.clone(),
        }
    }

    /// The pin advertises the codex harness when configured, else the
    /// muse harness when configured, else nothing usable.
    fn pin_family(&self) -> &'static str {
        if !self.harness.is_empty() {
            factory::FACTORY_HARNESS_CODEX
        } else if !self.muse_harness.is_empty() {
            factory::FACTORY_HARNESS_MUSE
        } else {
            factory::FACTORY_HARNESS_CODEX
        }
    }
}
