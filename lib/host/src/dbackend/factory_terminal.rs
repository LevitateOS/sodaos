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
    fn take_native_channels(
        &self,
        run: &str,
    ) -> Result<Option<terminal::factory::native_io::NativeChannels>, factory::FactoryError> {
        self.native_channels
            .lock()
            .map_err(|_| factory::FactoryError::Msg("native channel owner poisoned".into()))
            .map(|mut channels| channels.remove(run))
    }
    fn write_native_output(
        &self,
        lease: &factory::Lease,
        output: &[u8],
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        self.service()
            .factory_codex_native_output(&cv_lease_to_texec(lease), output, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn harness_pin(&self, family: &str) -> Result<factory::FactoryHarnessPin, String> {
        let (binary, version, sha256) = match family {
            factory::FACTORY_HARNESS_CODEX => {
                (&self.harness, &self.harness_version, &self.harness_sha256)
            }
            factory::FACTORY_HARNESS_MUSE => (
                &self.muse_harness,
                &self.muse_harness_version,
                &self.muse_harness_sha256,
            ),
            _ => return Err("unsupported factory harness".to_string()),
        };
        if binary.is_empty() {
            return Err("factory harness is not configured".to_string());
        }
        Ok(factory::FactoryHarnessPin {
            harness: family.to_string(),
            version: version.clone(),
            sha256: sha256.clone(),
            image: String::new(),
        })
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
        if run.harness == factory::FACTORY_HARNESS_CODEX
            && run.execution_phase == factory::EXECUTION_IMPLEMENT
            && (self.model_catalog.is_empty() || self.model_catalog_sha256.is_empty())
        {
            return Err(factory::FactoryError::Msg(
                "qualified native Codex task profile is not configured".into(),
            ));
        }
        let out = if run.harness == factory::FACTORY_HARNESS_MUSE {
            service
                .factory_muse_reserve(&trun, &tlease, pin, max_secs, deadline)
                .map(|(binding, _paths)| binding)
        } else {
            let channels = terminal::factory::native_io::NativeChannels::create(
                &self.state_dir,
                &run.project,
                &run.id,
            )
            .map_err(factory::FactoryError::Msg)?;
            match service.factory_codex_controlled_reserve(
                &trun, &tlease, pin, max_secs, &channels, deadline,
            ) {
                Ok((binding, paths)) => {
                    if let Some(profile) = paths.native_profile {
                        let retained =
                            factory::PrivateState::open(&self.state_dir).and_then(|state| {
                                state.retain_native_profile(run, &binding.invocation_id, profile)
                            });
                        if retained.is_err() {
                            if service.factory_codex_stop_unbound(&trun, deadline).is_ok() {
                                let _ = channels.retire();
                            }
                            return Err(factory::FactoryError::Msg(
                                "native profile retention unconfirmed".into(),
                            ));
                        }
                    }
                    self.native_channels
                        .lock()
                        .map_err(|_| {
                            factory::FactoryError::Msg("native channel owner poisoned".into())
                        })?
                        .insert(run.id.clone(), channels);
                    Ok(binding)
                }
                Err(reason) => {
                    if service.factory_codex_stop_unbound(&trun, deadline).is_ok() {
                        let _ = channels.retire();
                    }
                    Err(reason)
                }
            }
        };
        out.map_err(factory::FactoryError::Msg)
    }
    fn stage_mcp(
        &self,
        lease: &factory::Lease,
        binding: &factory::FactoryMCPBinding,
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        binding.validate().map_err(factory::FactoryError::Msg)?;
        let data = serde_json::to_vec(binding)
            .map_err(|_| factory::FactoryError::Msg("invalid Factory MCP staging".to_string()))?;
        self.service()
            .factory_stage_mcp(&cv_lease_to_texec(lease), &data, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn stage_execution_manifest(
        &self,
        lease: &factory::Lease,
        manifest: &[u8],
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        self.service()
            .factory_stage_execution_manifest(&cv_lease_to_texec(lease), manifest, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn start(
        &self,
        lease: &factory::Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
        execution_phase: &str,
    ) -> Result<(), factory::FactoryError> {
        let service = self.service();
        let tlease = cv_lease_to_texec(lease);
        if muse_scoped_lease(&tlease) {
            service
                .factory_muse_start(&tlease, credential, prompt, deadline, execution_phase)
                .map_err(factory::FactoryError::Msg)
        } else {
            service
                .factory_codex_start(&tlease, credential, prompt, deadline, execution_phase)
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
                .map_err(factory::FactoryError::Msg)?;
            self.retire_native_channels(&lease.execution_id)
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
                .map_err(factory::FactoryError::Msg)?;
            self.retire_native_channels(&run.id)
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
    fn retire_native_channels(&self, run: &str) -> Result<(), factory::FactoryError> {
        let channels = self
            .native_channels
            .lock()
            .map_err(|_| factory::FactoryError::Msg("native channel owner poisoned".into()))?
            .remove(run);
        if let Some(channels) = channels {
            channels.retire().map_err(factory::FactoryError::Msg)?;
        }
        Ok(())
    }

    /// The seam owns no service state: it rebuilds the value service (pure
    /// configuration + the shared executor shape) per call. Service methods
    /// take `&self` and hold no interior state, so this is free.
    fn service(&self) -> terminal::Service<project::Native> {
        terminal::Service {
            exec: project::Native,
            codex_harness: self.harness.clone(),
            codex_harness_sha256: self.harness_sha256.clone(),
            codex_harness_version: self.harness_version.clone(),
            codex_model_catalog: self.model_catalog.clone(),
            codex_model_catalog_sha256: self.model_catalog_sha256.clone(),
            muse_harness: self.muse_harness.clone(),
            muse_harness_sha256: self.muse_harness_sha256.clone(),
            muse_harness_version: self.muse_harness_version.clone(),
        }
    }
}
