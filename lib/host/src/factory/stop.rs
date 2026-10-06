use std::time::{Duration, Instant};

use crate::project::Executor;

use super::receipt::{receipt_state, receipt_stop_owned, receipt_terminal, FactoryReceipt};
use super::{
    Factory, FactoryBroker, FactoryError, FactoryRun, FactoryState, FactoryStop, FactoryTerminal,
    Secret, FACTORY_FAILED, FACTORY_STOPPED, FACTORY_UNCERTAIN, IDENTITY_FACTORY,
};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `Stop`: persist the stop tombstone for one run identity and retire
    /// its native boundary and broker execution. It works before the run
    /// is ever observed and refuses all subsequent work under that ID.
    pub fn stop(&self, req: &FactoryStop, deadline: Instant) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (mut receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            receipt = FactoryReceipt {
                run: FactoryRun {
                    id: req.id.clone(),
                    project: req.project.clone(),
                    // Go zero-time encoding, byte-exact in the tombstone.
                    deadline: "0001-01-01T00:00:00Z".to_string(),
                    ..FactoryRun::default()
                },
                phase: FACTORY_STOPPED.to_string(),
                retirement: "confirmed".to_string(),
                reason: "stop-before-start".to_string(),
                ..FactoryReceipt::default()
            };
            // A tombstone for an unknown run carries no validated run
            // record; store it directly so later launches refuse without
            // a full identity.
            drop(lock);
            self.store_tombstone(&receipt)?;
            if self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.id, deadline)
                .is_err()
            {
                receipt.reason = "broker-close-uncertain".to_string();
                let _ = self.store_tombstone(&receipt);
                return Ok(receipt_state(&receipt, false));
            }
            return Ok(receipt_state(&receipt, false));
        }
        if receipt_terminal(&receipt.phase) {
            // A tombstone whose broker fence never confirmed retries it;
            // every other terminal receipt is final.
            let retry_close = receipt.phase == FACTORY_STOPPED
                && receipt.reason == "broker-close-uncertain"
                && receipt.run.validate().is_err();
            drop(lock);
            if retry_close
                && self
                    .broker
                    .close_execution(IDENTITY_FACTORY, &req.id, deadline)
                    .is_ok()
            {
                receipt.reason = "stop-before-start".to_string();
                let _ = self.store_tombstone(&receipt);
            }
            return Ok(receipt_state(&receipt, false));
        }
        receipt.phase = FACTORY_STOPPED.to_string();
        self.store_receipt(&receipt)?;
        drop(lock);
        let mut native_uncertain = false;
        match (&receipt.binding, &receipt.lease) {
            (Some(binding), Some(lease)) => {
                if self
                    .terminal
                    .stop(&lease.with_binding(binding), deadline)
                    .is_err()
                {
                    native_uncertain = true;
                }
            }
            _ => {
                if receipt.run.validate().is_ok()
                    && self.terminal.stop_unbound(&receipt.run, deadline).is_err()
                {
                    native_uncertain = true;
                }
            }
        }
        let credential_ok = self.reconcile_run_credential(&mut receipt, deadline);
        // The broker fence is idempotent: retry transient close failures
        // before declaring the stop uncertain.
        let mut closed = false;
        for attempt in 0..3 {
            if self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.id, deadline)
                .is_ok()
            {
                closed = true;
                break;
            }
            if attempt < 2 {
                std::thread::sleep(Duration::from_millis(200));
            }
        }
        let mut uncertain = native_uncertain || !credential_ok || !closed;
        // Stop-before-start owns its outcome: a run that never started
        // and never took delivery holds no CLI effect and no credential,
        // so a confirmed fence settles it even when the native unit was
        // already absent (reserve race). The racing launch's own failure
        // path reaps any unit its reserve just created.
        if closed && !receipt.started && !receipt.delivered {
            uncertain = false;
        }
        if uncertain {
            eprintln!(
                "factory run {} stop uncertain: native={native_uncertain} credential_ok={credential_ok} closed={closed}",
                req.id
            );
        }
        let reason = if uncertain {
            "stop-uncertain"
        } else {
            "stopped"
        };
        self.record_stop_outcome(&receipt, uncertain, reason, true, deadline)
    }

    /// `reconcileRunCredential`: return or reconcile the run's broker lease
    /// after native retirement. Reports whether custody is settled.
    fn reconcile_run_credential(&self, receipt: &mut FactoryReceipt, deadline: Instant) -> bool {
        let (lease, binding) = match (&receipt.lease, &receipt.binding) {
            (Some(lease), binding) if !receipt.credential_returned => {
                (lease.clone(), binding.clone())
            }
            _ => return true,
        };
        if receipt.delivered {
            if let Some(binding) = binding {
                let auth = match self
                    .terminal
                    .capture(&lease.with_binding(&binding), deadline)
                {
                    Ok(auth) => Secret(auth),
                    Err(_) => {
                        self.broker.reconcile_lease(&lease.id, deadline);
                        return false;
                    }
                };
                if self
                    .broker
                    .return_lease(&lease.id, &binding, &auth.0, deadline)
                    .is_err()
                {
                    // The racing launch may have returned first: a terminal
                    // execution means custody settled without this stop.
                    // CredentialReturned stays false: this stop did not return.
                    if let Ok(true) = self.broker.execution_is_terminal(
                        IDENTITY_FACTORY,
                        &receipt.run.id,
                        deadline,
                    ) {
                        return true;
                    }
                    return false;
                }
                receipt.credential_returned = true;
                return true;
            }
        }
        self.broker.reconcile_lease(&lease.id, deadline);
        true
    }

    pub(in crate::factory) fn record_stop_outcome(
        &self,
        receipt: &FactoryReceipt,
        uncertain: bool,
        reason: &str,
        overwrite: bool,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        // A concurrent operator stop owns the outcome of a launch; only
        // the stop path itself overwrites the tombstone it just persisted.
        if receipt_stop_owned(&current.phase) && !overwrite {
            return Ok(receipt_state(&current, false));
        }
        current.credential_returned = receipt.credential_returned;
        current.output = receipt.output.clone();
        current.exit_code = receipt.exit_code;
        if uncertain {
            current.phase = FACTORY_UNCERTAIN.to_string();
            current.retirement = "uncertain".to_string();
            current.reason = reason.to_string();
        } else if overwrite {
            current.phase = FACTORY_STOPPED.to_string();
            current.retirement = "confirmed".to_string();
            current.reason = reason.to_string();
        } else {
            current.phase = FACTORY_FAILED.to_string();
            current.retirement = "confirmed".to_string();
            current.reason = reason.to_string();
        }
        self.store_receipt(&current)?;
        Ok(receipt_state(&current, false))
    }
}
