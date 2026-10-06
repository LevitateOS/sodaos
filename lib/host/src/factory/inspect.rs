use std::time::Instant;

use crate::project::Executor;

use super::deadline::live_deadline;
use super::receipt::{receipt_state, receipt_terminal};
use super::{
    takeover_source, Factory, FactoryBroker, FactoryError, FactoryInspect, FactoryState,
    FactoryTakeover, FactoryTerminal, TakeoverResult, FACTORY_APPROVED, FACTORY_RUNNING,
};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `Inspect`: report the authoritative recorded state for one run
    /// identity and whether its unit is currently live.
    pub fn inspect(
        &self,
        req: &FactoryInspect,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let mut live = false;
        if let Some(binding) = &receipt.binding {
            if receipt.phase == FACTORY_APPROVED || receipt.phase == FACTORY_RUNNING {
                live = self.terminal.live(binding, live_deadline(deadline));
            }
        }
        Ok(receipt_state(&receipt, live))
    }

    /// `Takeover`: copy one retired run's retained work into the admitted
    /// member's own derived checkout destination. Holds the lock across
    /// the copy to serialize duplicate takeovers.
    pub fn takeover(
        &self,
        req: &FactoryTakeover,
        deadline: Instant,
    ) -> Result<TakeoverResult, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let _lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => return Err(FactoryError::msg("factory run is not retired for takeover")),
        };
        if !takeover_source(
            &format!(
                "/home/{}/checkouts/{}",
                receipt.run.role, receipt.run.preparation
            ),
            &receipt.run.role,
            &receipt.run.preparation,
        ) {
            return Err(FactoryError::msg("invalid takeover source"));
        }
        let (dest, reused) = self.terminal.takeover_copy(
            &req.project,
            &binding.project,
            &receipt.run.role,
            &receipt.run.preparation,
            &req.member,
            &req.id,
            deadline,
        )?;
        let result = TakeoverResult {
            id: req.id.clone(),
            project: req.project.clone(),
            member: req.member.clone(),
            destination: dest,
            reused,
        };
        result.validate().map_err(FactoryError::msg)?;
        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use crate::factory::tests::common::{
        container_id, deadline, sample_launch, sample_run, wired_factory, write_running,
    };
    use crate::factory::*;

    #[test]
    fn inspect_matrix() {
        let (_dir, factory, _exec, _term, broker) = wired_factory("inspect");
        let req = sample_launch();
        // Unknown run.
        assert_eq!(
            factory
                .inspect(
                    &FactoryInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone()
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::NotFound
        );
        // Approved run without binding: no liveness probe.
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        let state = factory
            .inspect(
                &FactoryInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "approved");
        assert!(!state.live);
        // Bad address.
        assert_eq!(
            factory
                .inspect(
                    &FactoryInspect {
                        project: "x".to_string(),
                        id: "y".to_string()
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "invalid factory run address"
        );
    }

    #[test]
    fn inspect_running_probes_liveness() {
        let (dir, factory, _exec, term, _broker) = wired_factory("inspect-live");
        let run = sample_run();
        write_running(&dir, &run, true);
        term.live.borrow_mut().push_back(true);
        let state = factory
            .inspect(
                &FactoryInspect {
                    project: run.project.clone(),
                    id: run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "running");
        assert!(state.live);
        assert_eq!(state.container, container_id());
    }
}
