use std::time::Instant;

use crate::project::Executor;

use super::deadline::live_deadline;
use super::receipt::receipt_terminal;
use super::{
    factory_unit_name, Factory, FactoryBroker, FactoryError, FactoryExport, FactoryExportState,
    FactoryOutput, FactoryOutputState, FactoryTerminal, FACTORY_APPROVED, FACTORY_RUNNING,
    MAX_FACTORY_EXPORT_BUNDLE,
};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `Output`: report one bounded slice of a run's recorded CLI output
    /// with its run/process binding. A run whose recorded container
    /// incarnation no longer resolves refuses stale.
    pub fn output(
        &self,
        req: &FactoryOutput,
        deadline: Instant,
    ) -> Result<FactoryOutputState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let mut state = FactoryOutputState {
            id: req.id.clone(),
            project: req.project.clone(),
            phase: receipt.phase.clone(),
            terminal: receipt_terminal(&receipt.phase),
            reason: receipt.reason.clone(),
            exit_code: receipt.exit_code,
            ..FactoryOutputState::default()
        };
        let binding = match &receipt.binding {
            Some(binding) => binding.clone(),
            None => return Ok(state),
        };
        // Container, unit and invocation describe the recorded process
        // binding together; before a binding exists all three stay empty.
        state.container = binding.project.clone();
        state.unit = factory_unit_name(&req.id);
        state.invocation = binding.invocation_id.clone();
        if receipt.phase == FACTORY_APPROVED || receipt.phase == FACTORY_RUNNING {
            state.live = self.terminal.live(&binding, live_deadline(deadline));
        }
        let slice = self.terminal.output(
            &receipt.run.project,
            &binding,
            req.offset,
            req.limit,
            deadline,
        )?;
        state.total = slice.total;
        state.offset = slice.offset;
        state.truncated = slice.truncated;
        state.gap = slice.gap;
        state.next = slice.offset + slice.data.len() as i64;
        if !slice.data.is_empty() {
            state.data = crate::ssh::b64_encode(&slice.data);
        }
        Ok(state)
    }

    /// `Export`: read one settled run's exact candidate as a bounded Git
    /// bundle from its recorded role checkout.
    pub fn export(
        &self,
        req: &FactoryExport,
        deadline: Instant,
    ) -> Result<FactoryExportState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => return Err(FactoryError::msg("factory run is not settled for export")),
        };
        if receipt.run.role != req.role || receipt.run.preparation != req.preparation {
            return Err(FactoryError::Denied);
        }
        let bundle = self.terminal.export_bundle(
            &req.project,
            &binding.project,
            &req.role,
            &req.preparation,
            &req.candidate,
            deadline,
        )?;
        if bundle.is_empty() || bundle.len() > MAX_FACTORY_EXPORT_BUNDLE {
            return Err(FactoryError::ExportBounds);
        }
        Ok(FactoryExportState {
            id: req.id.clone(),
            project: req.project.clone(),
            phase: receipt.phase.clone(),
            container: binding.project.clone(),
            candidate: req.candidate.clone(),
            bundle: crate::ssh::b64_encode(&bundle),
        })
    }
}
