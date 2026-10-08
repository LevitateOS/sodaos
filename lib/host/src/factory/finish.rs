use std::time::Instant;

use crate::project::Executor;

use super::candidate::run_reason;
use super::deadline::cleanup_deadline;
use super::receipt::{receipt_state, receipt_stop_owned, FactoryReceipt};
use super::{
    Factory, FactoryBroker, FactoryError, FactoryState, FactoryTerminal, Lease, Secret,
    FACTORY_COMPLETED, FACTORY_FAILED, IDENTITY_FACTORY,
};
use crate::terminal::Binding;

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `failRun`: record a refusal that never acquired native or broker
    /// resources. A concurrent stop still wins.
    pub(in crate::factory) fn fail_run(
        &self,
        receipt: &FactoryReceipt,
        cause: FactoryError,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, cleanup)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        if receipt_stop_owned(&current.phase) {
            return Ok(receipt_state(&current, false));
        }
        if cause == FactoryError::Busy {
            current.reason = "broker-busy".to_string();
            self.store_receipt(&current)?;
            return Ok(receipt_state(&current, false));
        }
        if matches!(
            cause,
            FactoryError::Denied | FactoryError::Uncertain | FactoryError::DeadlineExceeded
        ) {
            current.phase = FACTORY_FAILED.to_string();
            current.reason = run_reason(&cause).to_string();
            self.store_receipt(&current)?;
            return Ok(receipt_state(&current, false));
        }
        Err(cause)
    }

    /// `abandonRun`: close the broker execution and record failure after a
    /// refused local step. Transport failures stay errors.
    pub(in crate::factory) fn abandon_run(
        &self,
        receipt: &FactoryReceipt,
        reason: &str,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let _ = self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, cleanup)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        if receipt_stop_owned(&current.phase) {
            return Ok(receipt_state(&current, false));
        }
        current.phase = FACTORY_FAILED.to_string();
        current.reason = reason.to_string();
        self.store_receipt(&current)?;
        Ok(receipt_state(&current, false))
    }

    /// `refreshStopped`: re-read the receipt and store progress. A stop
    /// tombstone aborts the launch.
    pub(in crate::factory) fn refresh_stopped(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<Option<FactoryState>, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(None);
        }
        if receipt_stop_owned(&current.phase) {
            return Ok(Some(receipt_state(&current, false)));
        }
        current.lease = receipt.lease.clone();
        current.binding = receipt.binding.clone();
        current.generation = receipt.generation;
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(None)
    }

    /// `launchYielded`: whether a concurrent stop tombstoned the run.
    fn launch_yielded(&self, receipt: &FactoryReceipt, deadline: Instant) -> bool {
        let Ok(_lock) = self.lock_run(&receipt.run.project, &receipt.run.id, deadline) else {
            return false;
        };
        match self.load_receipt(&receipt.run.project, &receipt.run.id) {
            Ok((current, true)) => receipt_stop_owned(&current.phase),
            _ => false,
        }
    }

    /// `stopFailedStart`: retire a run whose start never confirmed: native
    /// stop, credential capture attempt, broker return or reconcile, then
    /// close.
    pub(in crate::factory) fn stop_failed_start(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let _ = self.terminal.stop(&bound, cleanup);
        let mut receipt = receipt.clone();
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, false, "stopped", false, cleanup);
        }
        let mut uncertain = false;
        match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => {
                let auth = Secret(auth);
                if self
                    .broker
                    .return_lease(&lease.id, binding, &auth.0, cleanup)
                    .is_ok()
                {
                    receipt.credential_returned = true;
                } else {
                    uncertain = true;
                }
            }
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                uncertain = true;
            }
        }
        if self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup)
            .is_err()
        {
            uncertain = true;
        }
        self.record_stop_outcome(&receipt, uncertain, "start-unconfirmed", false, cleanup)
    }

    pub(in crate::factory) fn stop_timed_out(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let mut uncertain = false;
        if self.terminal.stop(&bound, cleanup).is_err() {
            uncertain = true;
        }
        let mut receipt = receipt.clone();
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, uncertain, "stopped", false, cleanup);
        }
        match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => {
                let auth = Secret(auth);
                if self
                    .broker
                    .return_lease(&lease.id, binding, &auth.0, cleanup)
                    .is_ok()
                {
                    receipt.credential_returned = true;
                } else {
                    uncertain = true;
                }
            }
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                uncertain = true;
            }
        }
        if self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup)
            .is_err()
        {
            uncertain = true;
        }
        self.record_stop_outcome(&receipt, uncertain, "deadline-exceeded", false, cleanup)
    }

    pub(in crate::factory) fn finish_run(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        exit: i64,
        output: String,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let mut receipt = receipt.clone();
        if exit >= 0 {
            receipt.exit_code = Some(exit);
        }
        receipt.output = output;
        // The CLI exited, but descendants may linger: retire the boundary
        // before capturing, so the captured bytes are final.
        if self.terminal.stop(&bound, cleanup).is_err() {
            self.broker.reconcile_lease(&lease.id, cleanup);
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
            return self.record_stop_outcome(
                &receipt,
                true,
                "retirement-unconfirmed",
                false,
                cleanup,
            );
        }
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, false, "stopped", false, cleanup);
        }
        let auth = Secret(match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => auth,
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                let _ = self
                    .broker
                    .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
                return self.record_stop_outcome(
                    &receipt,
                    true,
                    "credential-capture-unconfirmed",
                    false,
                    cleanup,
                );
            }
        });
        if self
            .broker
            .return_lease(&lease.id, binding, &auth.0, cleanup)
            .is_err()
        {
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
            return self.record_stop_outcome(
                &receipt,
                true,
                "credential-return-unconfirmed",
                false,
                cleanup,
            );
        }
        receipt.credential_returned = true;
        let _ = self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
        receipt.phase = FACTORY_COMPLETED.to_string();
        receipt.retirement = "confirmed".to_string();
        receipt.reason.clear();
        if exit != 0 {
            receipt.phase = FACTORY_FAILED.to_string();
            receipt.reason = "execution-failed".to_string();
        }
        self.update_receipt(&mut receipt, cleanup)?;
        Ok(receipt_state(&receipt, false))
    }
}
