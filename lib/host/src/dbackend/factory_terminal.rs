use std::time::Instant;

use crate::daemon::broker::cv_lease_to_texec;
use crate::terminal::factory::run::FACTORY_SCOPE_MUSE;
use crate::{factory, project, terminal};

use super::TerminalSeam;

fn cv_run_to_terminal(r: &factory::FactoryRun) -> crate::terminal::factory::run::FactoryRun {
    crate::terminal::factory::run::FactoryRun {
        deadline_raw: r.deadline.clone(),
        actor: r.actor,
        id: r.id.clone(),
        project: r.project.clone(),
        role: r.role.clone(),
        preparation: r.preparation.clone(),
        harness: r.harness.clone(),
        harness_vers: r.harness_vers.clone(),
        model: r.model.clone(),
        effort: r.effort.clone(),
        contributor: r.contributor,
        assignment: r.assignment.clone(),
        source_commit: r.source_commit.clone(),
        connection: r.connection.clone(),
    }
}

fn cv_slice_to_factory(
    s: &crate::terminal::factory::output::FactoryOutputSlice,
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
        let lease = cv_lease_to_texec(lease);
        muse_scoped_lease(&lease)?;
        self.service()
            .factory_muse_native_output(&lease, output, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn harness_pin(&self, family: &str) -> Result<factory::FactoryHarnessPin, String> {
        // NEW-work identity resolves the CURRENT shared selection
        // (due-check + stale refusal). No reference is pinned here: the
        // query reserves nothing, and the exact bytes are verified again at
        // reserve time. `image` stays empty; the daemon route fills it from
        // its configuration, exactly like the Go dispatch.
        let (version, sha256) = terminal::factory::bound::current_pin(family)?;
        Ok(factory::FactoryHarnessPin {
            harness: family.to_string(),
            version,
            sha256,
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
        if run.harness != factory::FACTORY_HARNESS_MUSE {
            return Err(factory::FactoryError::Msg(
                "unsupported factory harness".into(),
            ));
        }
        let service = self.service();
        let trun = cv_run_to_terminal(run);
        let tlease = cv_lease_to_texec(lease);
        // Bind the exact bundle before staging anything: a recorded sidecar
        // wins for recovery and duplicate reserves, otherwise the CURRENT
        // shared selection is snapshotted for this new reservation. Missing
        // history fails closed; the startup-configured path/pin is never
        // consulted.
        let state =
            factory::PrivateState::open(&self.state_dir).map_err(factory::FactoryError::Msg)?;
        let previous = state
            .read_harness_bind(&run.project, &run.id)
            .map_err(factory::FactoryError::Msg)?;
        let mut bound = if let Some(recorded) = previous.clone() {
            terminal::factory::bound::bind_recorded(&recorded)
                .map_err(factory::FactoryError::Msg)?
        } else {
            terminal::factory::bound::resolve_current(&run.harness)
                .map_err(factory::FactoryError::Msg)?
        };
        // Coherence with the admission: the request must name exactly the
        // bound executable digest and version. A moved selection or a stale
        // admission fails closed here instead of staging drifted bytes.
        if pin != bound.executable_sha256 || run.harness_vers != bound.version {
            terminal::factory::bound::release_bound(bound);
            return Err(factory::FactoryError::Msg(terminal::err_denied()));
        }
        let token = bound.take_ref_token().ok_or_else(|| {
            factory::FactoryError::Msg("harness reference unconfirmed".to_string())
        })?;
        let record = terminal::factory::bound::HarnessBind {
            family: run.harness.clone(),
            version: bound.version.clone(),
            package_digest: bound.package_digest.clone(),
            executable_sha256: bound.executable_sha256.clone(),
            ref_token: token.token.clone(),
        };
        if let Err(cause) = state
            .write_harness_bind(&run.project, &run.id, &record)
            .map_err(factory::FactoryError::Msg)
        {
            terminal::factory::bound::release_harness_ref(
                bound.provider,
                &bound.package_digest,
                &token.token,
            );
            return Err(cause);
        }
        let out = {
            let channels = terminal::factory::native_io::NativeChannels::create(
                &self.state_dir,
                &run.project,
                &run.id,
            )
            .map_err(factory::FactoryError::Msg)?;
            match service.factory_muse_controlled_reserve(
                &trun, &tlease, &bound, max_secs, &channels, deadline,
            ) {
                Ok((binding, _paths)) => {
                    self.native_channels
                        .lock()
                        .map_err(|_| {
                            factory::FactoryError::Msg("native channel owner poisoned".into())
                        })?
                        .insert(run.id.clone(), channels);
                    Ok(binding)
                }
                Err(reason) => {
                    if service.factory_muse_stop_unbound(&trun, deadline).is_ok() {
                        let _ = channels.retire();
                    }
                    Err(reason)
                }
            }
        };
        match out {
            Ok(binding) => {
                // The new reference supersedes the recorded one, if any.
                if let Some(old) = previous {
                    terminal::factory::bound::release_harness_ref(
                        bound.provider,
                        &old.package_digest,
                        &old.ref_token,
                    );
                }
                Ok(binding)
            }
            Err(reason) => {
                if !reserve_failure_is_definitive(&reason) {
                    // Unit state is unconfirmed: the service may have
                    // started a unit for this attempt (start unconfirmed,
                    // failed active-invocation check, retention loss after
                    // a successful reserve) while this attempt's binding +
                    // ref are already the durable record. Preserve them;
                    // the single owner releases after confirmed
                    // retirement. A conservative leak beats collecting a
                    // possibly-live bundle.
                    return Err(factory::FactoryError::Msg(reason));
                }
                // Definitive pre-start refusal: this attempt started no
                // unit, so release this attempt and restore the recorded
                // binding, if any.
                let reason = factory::FactoryError::Msg(reason);
                terminal::factory::bound::release_harness_ref(
                    bound.provider,
                    &bound.package_digest,
                    &token.token,
                );
                match previous {
                    Some(old) => {
                        let _ = state.write_harness_bind(&run.project, &run.id, &old);
                        Err(reason)
                    }
                    None => {
                        let _ = state.remove_harness_bind(&run.project, &run.id);
                        Err(reason)
                    }
                }
            }
        }
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
        let tlease = cv_lease_to_texec(lease);
        muse_scoped_lease(&tlease)?;
        self.service()
            .factory_muse_start(&tlease, credential, prompt, deadline, execution_phase)
            .map_err(factory::FactoryError::Msg)
    }
    fn wait(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<(i64, String), factory::FactoryError> {
        let tlease = cv_lease_to_texec(lease);
        muse_scoped_lease(&tlease)?;
        self.service()
            .factory_muse_wait(&tlease, deadline)
            .map(|(code, out)| (code as i64, out))
            .map_err(factory::FactoryError::Msg)
    }
    fn stop(&self, lease: &factory::Lease, deadline: Instant) -> Result<(), factory::FactoryError> {
        let tlease = cv_lease_to_texec(lease);
        muse_scoped_lease(&tlease)?;
        self.service()
            .factory_muse_stop(&tlease, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn stop_unbound(
        &self,
        run: &factory::FactoryRun,
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        if run.harness != factory::FACTORY_HARNESS_MUSE {
            return Err(factory::FactoryError::Msg(terminal::err_denied()));
        }
        self.service()
            .factory_muse_stop_unbound(&cv_run_to_terminal(run), deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn capture(
        &self,
        lease: &factory::Lease,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let tlease = cv_lease_to_texec(lease);
        muse_scoped_lease(&tlease)?;
        self.service()
            .factory_muse_capture(&tlease, deadline)
            .map_err(factory::FactoryError::Msg)
    }
    fn live(&self, binding: &terminal::Binding, deadline: Instant) -> bool {
        binding.scope == FACTORY_SCOPE_MUSE && self.service().factory_muse_live(binding, deadline)
    }
    fn output(
        &self,
        project: &str,
        binding: &terminal::Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<factory::OutputSlice, factory::FactoryError> {
        if binding.scope != FACTORY_SCOPE_MUSE {
            return Err(factory::FactoryError::Msg(terminal::err_denied()));
        }
        self.service()
            .factory_muse_output(project, binding, offset, limit, deadline)
            .map(|s| cv_slice_to_factory(&s))
            .map_err(factory::FactoryError::Msg)
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

/// Whether a reserve error is a definitive pre-start refusal: the service
/// never started a unit for this attempt, so rolling back this attempt's
/// binding + ref is safe. This is exactly the pre-`systemd_run` set
/// (admission, container, checkout, staging and install checks, all of
/// which deny before any unit exists). Anything else — unit start
/// unconfirmed, a failed active-invocation check after the start, retention
/// loss after a successful reserve, or an unrecognized error — leaves unit
/// state unconfirmed and must preserve the durable binding + lifetime ref;
/// release happens only after confirmed retirement through the single
/// owner. Missing/corrupt history still fails closed at the read above.
fn reserve_failure_is_definitive(reason: &str) -> bool {
    matches!(
        reason,
        terminal::ERR_DENIED
            | "factory run directories unconfirmed"
            | "factory checkout identity unconfirmed"
            | "factory checkout is not the assigned commit"
            | "factory source changed under preparation"
            | "factory source unconfirmed"
            | "factory harness staging unconfirmed"
            | "factory harness install unconfirmed"
            | "guest harness digest differs"
            | "factory code host staging unconfirmed"
            | "factory code host install unconfirmed"
            | "guest code host digest differs"
    )
}

/// Factory leases are Muse-only: a lease whose binding does not carry the
/// factory-muse scope (including an unbound lease) is denied.
fn muse_scoped_lease(lease: &terminal::Lease) -> Result<(), factory::FactoryError> {
    if lease
        .binding
        .as_ref()
        .is_some_and(|b| b.scope == FACTORY_SCOPE_MUSE)
    {
        Ok(())
    } else {
        Err(factory::FactoryError::Msg(terminal::err_denied()))
    }
}

impl TerminalSeam {
    /// The seam owns no service state: it rebuilds the value service (pure
    /// configuration + the shared executor shape) per call. Service methods
    /// take `&self` and hold no interior state, so this is free.
    fn service(&self) -> terminal::Service<project::Native> {
        terminal::Service {
            exec: project::Native,
            // The factory never runs Codex; the human-terminal Codex
            // harness is not part of this seam.
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            muse_harness: self.muse_harness.clone(),
            muse_harness_sha256: self.muse_harness_sha256.clone(),
            muse_harness_version: self.muse_harness_version.clone(),
        }
    }
}
