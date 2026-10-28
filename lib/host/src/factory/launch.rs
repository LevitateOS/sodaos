use std::time::Instant;

use crate::project::Executor;

use super::deadline::{
    drive_deadline, execution_deadline, parse_deadline, seconds_until, system_nanos_now,
};
use super::receipt::{receipt_state, receipt_stop_owned, FactoryReceipt};
use super::run::FACTORY_DEADLINE_BOUND_NANOS;
use super::{
    AcquireRequest, Factory, FactoryBroker, FactoryError, FactoryLaunch, FactoryState,
    FactoryTerminal, PrivateState, Secret, FACTORY_APPROVED, FACTORY_COMPLETED, FACTORY_FAILED,
    FACTORY_RUNNING, IDENTITY_FACTORY,
};
use crate::terminal::factory::bound;

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `Launch`: admit one supervised run and drive it to completion. An
    /// existing receipt is authoritative: duplicates return its state.
    /// Only unconfirmed operations return an error.
    pub fn launch(
        &self,
        req: &FactoryLaunch,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        // Recorded authority first: lock and load the receipt before
        // consulting the CURRENT pin or the execution deadline. An existing
        // receipt takes the recorded path below, which never queries
        // current, never starts a new process, and never re-checks the
        // original execution deadline. Only a genuinely new run (no
        // receipt) must match the current pin within the current deadline
        // bound.
        let lock = self.lock_run(&req.run.project, &req.run.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.run.project, &req.run.id)?;
        if exists {
            if req.run.effort != receipt.run.effort
                || req.run.contributor != receipt.run.contributor
            {
                return Err(FactoryError::Stale);
            }
        } else {
            let pin = self
                .terminal
                .harness_pin(&req.run.harness)
                .map_err(FactoryError::msg)?;
            if pin.harness != req.run.harness
                || pin.sha256.is_empty()
                || req.harness_sha256 != pin.sha256
            {
                return Err(FactoryError::msg(
                    "factory harness is unavailable until its own proof passes",
                ));
            }
            // Validated above: parseable and nonzero.
            let deadline_nanos = parse_deadline(&req.run.deadline).unwrap_or(0);
            let now = system_nanos_now();
            if !(deadline_nanos > now && deadline_nanos <= now + FACTORY_DEADLINE_BOUND_NANOS) {
                return Err(FactoryError::msg(
                    "run deadline is outside the supervised bound",
                ));
            }
            drop(lock);
            return self.launch_new(req, deadline_nanos, deadline);
        }
        if !req.verification_manifest.is_empty()
            || !receipt.run.verification_manifest_digest.is_empty()
        {
            if req.run != receipt.run {
                return Err(FactoryError::Stale);
            }
            self.store_verification_manifest(&req.run, &req.verification_manifest)?;
        }
        // Out-of-drive confirmed retirements (operator stop and its
        // repair paths live in `stop`, outside this lane): the exact-bundle
        // reference is due for release exactly once, same as in-drive.
        if receipt.retirement == "confirmed" {
            self.release_run_harness(&req.run.project, &req.run.id);
        }
        drop(lock);
        let mut state = receipt_state(&receipt, false);
        state.verification_checks = self.saved_verification_checks(&receipt)?;
        Ok(state)
    }

    /// `launchNew`: the genuinely-new path (no receipt existed). The caller
    /// has already matched the CURRENT pin and bound the CURRENT execution
    /// deadline; this stores the fresh receipt and drives it. Never called
    /// for an existing receipt, so recorded runs are never re-executed and
    /// no model call happens on the recorded path.
    fn launch_new(
        &self,
        req: &FactoryLaunch,
        deadline_nanos: i128,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let lock = self.lock_run(&req.run.project, &req.run.id, deadline)?;
        let (_, exists) = self.load_receipt(&req.run.project, &req.run.id)?;
        if exists {
            // A concurrent admission won the race: fall back to the
            // recorded path rather than re-executing. Re-locking keeps
            // the recorded read under the run lock.
            drop(lock);
            return self.launch(req, deadline);
        }
        self.store_verification_manifest(&req.run, &req.verification_manifest)?;
        self.store_execution_manifest(&req.run, &req.execution_manifest)?;
        let receipt = FactoryReceipt {
            run: req.run.clone(),
            phase: FACTORY_APPROVED.to_string(),
            ..FactoryReceipt::default()
        };
        let stored = self.store_receipt(&receipt);
        drop(lock);
        stored?;
        let mut state = self.drive(req, &receipt, deadline_nanos, deadline)?;
        let (current, _) = self.load_receipt(&req.run.project, &req.run.id)?;
        state.verification_checks = self.saved_verification_checks(&current)?;
        Ok(state)
    }

    /// `drive`: broker acquisition, native reservation, attested delivery,
    /// single-use start, bounded wait and credential return. The exact-bundle
    /// reference bound at reserve time releases here once retirement is
    /// confirmed.
    fn drive(
        &self,
        req: &FactoryLaunch,
        receipt: &FactoryReceipt,
        deadline_nanos: i128,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let result = self.drive_inner(req, receipt, deadline_nanos, deadline);
        if result
            .as_ref()
            .is_ok_and(|state| state.retirement == "confirmed")
        {
            self.release_run_harness(&req.run.project, &req.run.id);
        }
        result
    }

    /// `driveInner`: the launch drive itself; the wrapper above owns the
    /// exact-bundle lifetime.
    fn drive_inner(
        &self,
        req: &FactoryLaunch,
        receipt: &FactoryReceipt,
        deadline_nanos: i128,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let run_deadline = drive_deadline(deadline, deadline_nanos);
        // The validated harness family names its broker provider
        // ("muse"); the lease must come from the run's
        // own provider or the broker denies the connection mismatch.
        let lease = match self.broker.acquire(
            &AcquireRequest {
                provider_id: req.run.harness.clone(),
                execution_id: req.run.id.clone(),
                actor_id: req.run.actor,
                connection_id: req.run.connection.clone(),
                project_id: req.run.project.clone(),
                kind: IDENTITY_FACTORY.to_string(),
                deadline: req.run.deadline.clone(),
                role: req.run.role.clone(),
            },
            run_deadline,
        ) {
            Ok(lease) => lease,
            Err(cause) => return self.fail_run(receipt, cause, deadline),
        };
        let mut receipt = receipt.clone();
        receipt.generation = lease.generation;
        receipt.lease = Some(lease.clone());
        if let Some(stopped) = self.refresh_stopped(&mut receipt, deadline)? {
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
            return Ok(stopped);
        }
        let binding = match self.terminal.reserve(
            &req.run,
            &lease,
            &req.harness_sha256,
            seconds_until(deadline_nanos),
            run_deadline,
        ) {
            Ok(binding) => binding,
            Err(cause) => {
                eprintln!("factory run {} reserve refused: {cause:?}", req.run.id);
                return self.abandon_run(&receipt, "reserve-refused", deadline);
            }
        };
        receipt.binding = Some(binding.clone());
        if let Some(stopped) = self.refresh_stopped(&mut receipt, deadline)? {
            let _ = self
                .terminal
                .stop(&lease.with_binding(&binding), run_deadline);
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
            return Ok(stopped);
        }
        // Native verification checks run without model credentials. Keep this
        // lease unregistered until they pass, so a failed/cancelled check can
        // release an unused lease without requiring account reauthentication.
        let credential = if req.run.verification_manifest_digest.is_empty() {
            Some(Secret(
                match self.broker.register(&lease.id, &binding, run_deadline) {
                    Ok(credential) => credential,
                    Err(cause) => {
                        eprintln!("factory run {} register refused: {cause:?}", req.run.id);
                        let _ = self
                            .terminal
                            .stop(&lease.with_binding(&binding), run_deadline);
                        let _ =
                            self.broker
                                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
                        return self.abandon_run(&receipt, "register-refused", deadline);
                    }
                },
            ))
        } else {
            None
        };
        if let Some(mcp) = &req.mcp {
            if self
                .terminal
                .stage_mcp(&lease.with_binding(&binding), mcp, run_deadline)
                .is_err()
            {
                return self.stop_failed_start(&receipt, &lease, &binding, deadline);
            }
        }
        if !req.execution_manifest.is_empty()
            && self
                .terminal
                .stage_execution_manifest(
                    &lease.with_binding(&binding),
                    &req.execution_manifest,
                    run_deadline,
                )
                .is_err()
        {
            return self.stop_failed_start(&receipt, &lease, &binding, deadline);
        }
        if self
            .guard_checkpoint_checkout(&receipt, run_deadline)
            .is_err()
        {
            return self.stop_failed_start(&receipt, &lease, &binding, deadline);
        }
        // The admitted check and CLI share the wire run lifetime. The short
        // launch observation RPC must not cancel a valid longer native check.
        let launch_deadline = if req.run.verification_manifest_digest.is_empty() {
            deadline
        } else {
            execution_deadline(deadline_nanos)
        };
        let start_deadline = drive_deadline(launch_deadline, deadline_nanos);
        if self
            .await_verification_checks(&receipt, start_deadline)
            .is_err()
        {
            return self.stop_failed_start(&receipt, &lease, &binding, deadline);
        }
        match self.consume_start(&mut receipt, launch_deadline) {
            Ok(Some(stopped)) => {
                let _ = self
                    .terminal
                    .stop(&lease.with_binding(&binding), start_deadline);
                let _ = self
                    .broker
                    .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
                return Ok(stopped);
            }
            Ok(None) => {}
            Err(_) => return self.stop_failed_start(&receipt, &lease, &binding, deadline),
        }
        let credential = match credential {
            Some(credential) => credential,
            None => match self.broker.register(&lease.id, &binding, start_deadline) {
                Ok(credential) => Secret(credential),
                Err(_) => return self.stop_failed_start(&receipt, &lease, &binding, deadline),
            },
        };
        if self
            .terminal
            .start(
                &lease.with_binding(&binding),
                &credential.0,
                &req.prompt,
                start_deadline,
                &req.run.execution_phase,
            )
            .is_err()
        {
            return self.stop_failed_start(&receipt, &lease, &binding, deadline);
        }
        receipt.delivered = true;
        self.update_receipt(&mut receipt, launch_deadline)?;
        let mut channels = self.terminal.take_native_channels(&req.run.id)?;
        let native = if let Some(owned_channels) = channels.as_mut() {
            match self.control_native(
                &receipt,
                &req.prompt,
                owned_channels,
                execution_deadline(deadline_nanos),
            ) {
                Ok(output) => Some(output),
                Err(cause) => {
                    eprintln!("factory run {} native control refused: {cause}", req.run.id);
                    let result = self.stop_failed_start(&receipt, &lease, &binding, deadline);
                    if result
                        .as_ref()
                        .is_ok_and(|state| state.retirement == "confirmed")
                    {
                        if let Some(channels) = channels.take() {
                            let _ = channels.retire();
                        }
                    }
                    return result;
                }
            }
        } else {
            None
        };
        match self.terminal.wait(
            &lease.with_binding(&binding),
            execution_deadline(deadline_nanos),
        ) {
            Ok((exit, output)) => {
                let (exit, output) = native.unwrap_or((exit, output));
                let result = self.finish_run(&receipt, &lease, &binding, exit, output, deadline);
                if result
                    .as_ref()
                    .is_ok_and(|state| state.retirement == "confirmed")
                {
                    if let Some(channels) = channels.take() {
                        let _ = channels.retire();
                    }
                }
                result
            }
            Err(_) => {
                let result = self.stop_timed_out(&receipt, &lease, &binding, deadline);
                if result
                    .as_ref()
                    .is_ok_and(|state| state.retirement == "confirmed")
                {
                    if let Some(channels) = channels.take() {
                        let _ = channels.retire();
                    }
                }
                result
            }
        }
    }

    /// `consumeStart`: consume the single-use start marker. A second start,
    /// or a start after the stop tombstone, never reaches the native
    /// boundary.
    pub(in crate::factory) fn consume_start(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<Option<FactoryState>, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(None);
        }
        if receipt_stop_owned(&current.phase) || current.started {
            return Ok(Some(receipt_state(&current, false)));
        }
        if !current.run.verification_manifest_digest.is_empty()
            && !self
                .saved_verification_checks(&current)?
                .is_some_and(|s| s.phase == "passed" && s.retirement == "confirmed")
        {
            return Err(FactoryError::msg(
                "verification checks have not passed before CLI start",
            ));
        }
        current.started = true;
        current.phase = FACTORY_RUNNING.to_string();
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(None)
    }

    pub(in crate::factory) fn update_receipt(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(());
        }
        let stop_owned = receipt_stop_owned(&current.phase);
        current.delivered |= receipt.delivered;
        current.credential_returned |= receipt.credential_returned;
        if matches!(receipt.phase.as_str(), FACTORY_COMPLETED | FACTORY_FAILED) {
            current.output = receipt.output.clone();
            current.exit_code = receipt.exit_code;
        }
        if !stop_owned {
            current.retirement = receipt.retirement.clone();
            current.reason = receipt.reason.clone();
            if !receipt.phase.is_empty() {
                current.phase = receipt.phase.clone();
            }
        }
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(())
    }

    /// Release one run's bound harness reference after confirmed retirement
    /// and drop its sidecar. Best-effort and idempotent: never fails the run
    /// outcome, and a later touch retries whatever is left.
    pub(in crate::factory) fn release_run_harness(&self, project: &str, run: &str) {
        let Ok(state) = PrivateState::open(&self.state_dir) else {
            return;
        };
        let Ok(bind) = state.read_harness_bind(project, run) else {
            return;
        };
        let Some(bind) = bind else { return };
        if let Ok(provider) = bound::provider_for_family(&bind.family) {
            bound::release_harness_ref(provider, &bind.package_digest, &bind.ref_token);
        }
        let _ = state.remove_harness_bind(project, run);
    }
}

impl PrivateState {
    /// Read one run's recorded exact harness binding, if any. A corrupt or
    /// invalid record fails closed: there is no safe exact identity left to
    /// stage, and falling back to current would stage drifted bytes.
    pub(crate) fn read_harness_bind(
        &self,
        project: &str,
        run: &str,
    ) -> Result<Option<bound::HarnessBind>, String> {
        let bytes = match self.read_bytes(&bound::bind_name(project, run)?)? {
            Some(bytes) => bytes,
            None => return Ok(None),
        };
        let bind: bound::HarnessBind =
            serde_json::from_slice(&bytes).map_err(|_| "run harness binding is invalid")?;
        bind.validate()?;
        Ok(Some(bind))
    }

    /// Record one run's exact harness binding. The live store reference in
    /// the record keeps the bundle alive across refresh/cleanup until the
    /// confirmed-retirement release.
    pub(crate) fn write_harness_bind(
        &self,
        project: &str,
        run: &str,
        bind: &bound::HarnessBind,
    ) -> Result<(), String> {
        bind.validate()?;
        self.write(&bound::bind_name(project, run)?, bind)
    }

    /// Drop one run's recorded binding. Idempotent: a missing record is
    /// already gone.
    pub(crate) fn remove_harness_bind(&self, project: &str, run: &str) -> Result<(), String> {
        use std::os::unix::io::AsRawFd as _;
        let name = bound::bind_name(project, run)?;
        let dir = self.try_clone_directory()?;
        let cname = std::ffi::CString::new(name).map_err(|_| "checkpoint state name is invalid")?;
        // SAFETY: unlinkat on an owned directory fd with a validated name.
        let rc = unsafe { libc::unlinkat(dir.as_raw_fd(), cname.as_ptr(), 0) };
        if rc != 0 {
            let err = std::io::Error::last_os_error();
            if err.kind() == std::io::ErrorKind::NotFound {
                return Ok(());
            }
            return Err(err.to_string());
        }
        dir.sync_all().map_err(|e| e.to_string())?;
        Ok(())
    }
}
