use std::time::Instant;

use crate::project::Executor;

use super::deadline::{
    drive_deadline, execution_deadline, parse_deadline, seconds_until, system_nanos_now,
};
use super::receipt::{receipt_state, receipt_stop_owned, FactoryReceipt};
use super::run::FACTORY_DEADLINE_BOUND_NANOS;
use super::{
    AcquireRequest, Factory, FactoryBroker, FactoryError, FactoryLaunch, FactoryState,
    FactoryTerminal, Secret, FACTORY_APPROVED, FACTORY_COMPLETED, FACTORY_FAILED, FACTORY_RUNNING,
    IDENTITY_FACTORY,
};

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
        let lock = self.lock_run(&req.run.project, &req.run.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.run.project, &req.run.id)?;
        if !exists {
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
            return Ok(state);
        }
        if !req.verification_manifest.is_empty()
            || !receipt.run.verification_manifest_digest.is_empty()
        {
            if req.run != receipt.run {
                return Err(FactoryError::Stale);
            }
            self.store_verification_manifest(&req.run, &req.verification_manifest)?;
        }
        drop(lock);
        let mut state = receipt_state(&receipt, false);
        state.verification_checks = self.saved_verification_checks(&receipt)?;
        Ok(state)
    }

    /// `drive`: broker acquisition, native reservation, attested delivery,
    /// single-use start, bounded wait and credential return.
    fn drive(
        &self,
        req: &FactoryLaunch,
        receipt: &FactoryReceipt,
        deadline_nanos: i128,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let run_deadline = drive_deadline(deadline, deadline_nanos);
        // The validated harness family names its broker provider
        // ("codex"/"muse" are both); the lease must come from the run's
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
}
