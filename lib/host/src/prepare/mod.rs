//! Factory preparation operations: the host side of the fixed
//! `project-factory-roles` helper exchange. Ports `prepare.go` and the
//! candidate-preparation half of `factory_candidate.go`
//! (`PrepareCandidate`; run-receipt inspection stays for the factory port).
//! Helper payloads are built byte-for-byte like the Go map marshals.

use std::time::Instant;

use crate::json::{self, Kind, Spec};
use crate::preparation::{
    self, HoldState, Prepare, PrepareHold, PrepareInspect, PrepareState, PrepareStop,
};
use crate::project::{Executor, Runtime};

mod candidate;
mod helper;
mod paths;
mod source;
mod state;
mod tools;

pub use self::paths::{
    path_clean, path_join, preparation_paths, prepare_id_map, single_line, valid_resolved_tool_path,
};
pub use self::state::{map_preparation_state, LauncherEvidence};

const STOP_RESPONSE_SPECS: &[Spec] = &[
    Spec {
        name: "stopped",
        kind: Kind::Str,
    },
    Spec {
        name: "retirement",
        kind: Kind::Str,
    },
    Spec {
        name: "known",
        kind: Kind::Bool,
    },
];

const HOLD_RESPONSE_SPECS: &[Spec] = &[Spec {
    name: "hold",
    kind: Kind::Object {
        go_type: "project.HoldState",
        struct_name: "HoldState",
        specs: preparation::HOLD_STATE_SPECS,
    },
}];

impl<E: Executor> Runtime<E> {
    /// `inspectPreparationState`: authoritative observed state for one identity.
    pub(crate) fn inspect_preparation_state(
        &self,
        project: &str,
        id: &str,
        container: &str,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(id));
        body.push_str(",\"op\":\"inspect\"}");
        let raw = self.factory_helper(project, &body, deadline)?;
        let mut state = map_preparation_state(container, &raw)?;
        state.id = id.to_string();
        state.project = project.to_string();
        Ok(state)
    }

    /// `Prepare`: approve, clone, verify, resolve tools, record, start.
    pub fn prepare(&self, input: &Prepare, deadline: Instant) -> Result<PrepareState, String> {
        input.validate()?;
        let prep = &input.preparation;
        let container = self.prepare_container(&prep.project, true, deadline)?;
        self.factory_helper(&prep.project, "{\"op\":\"ensure\"}", deadline)?;
        if let Err(err) = self.approve_preparation(prep, &input.setup, deadline) {
            if let Ok(stopped) =
                self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
            {
                if stopped.stopped {
                    return Ok(stopped);
                }
            }
            return Err(err);
        }
        let state =
            self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)?;
        if state.stopped || state.ready || state.phase == preparation::PREPARE_FAILED {
            return Ok(state);
        }
        self.clone_preparation_source(prep, deadline)?;
        let verified = self.verify_launcher_environment(prep, deadline)?;
        let (tools, missing) = self.resolve_preparation_tools(prep, deadline)?;
        self.record_preparation_tools(prep, &tools, &missing, &verified, deadline)?;
        if !missing.is_empty() || !verified.refusal.is_empty() {
            return self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
        }
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"op\":\"start\"}");
        if let Err(err) = self.factory_helper(&prep.project, &body, deadline) {
            // Go returns the re-observed state alongside the error; every
            // caller drops the state on error, so only the error crosses.
            let _ = self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
            return Err(err);
        }
        self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
    }

    /// `InspectPreparation`: authoritative state, never mutating.
    pub fn inspect_preparation(
        &self,
        input: &PrepareInspect,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        self.inspect_preparation_state(&input.project, &input.id, &container, deadline)
    }

    /// `StopPreparation`: persist the stop tombstone, report the state.
    pub fn stop_preparation(
        &self,
        input: &PrepareStop,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&input.id));
        body.push_str(",\"op\":\"stop\"}");
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        const ERR: &str = "preparation stop unconfirmed";
        let v = json::decode_strict(&raw).map_err(|_| ERR.to_string())?;
        let m = json::bind_root(&v, "struct", STOP_RESPONSE_SPECS, false)
            .map_err(|_| ERR.to_string())?;
        let retirement = m.take_string("retirement");
        if m.take_string("stopped") != input.id
            || (retirement != "confirmed" && retirement != "uncertain")
        {
            return Err(ERR.to_string());
        }
        let mut state =
            self.inspect_preparation_state(&input.project, &input.id, &container, deadline)?;
        state.retirement = retirement;
        Ok(state)
    }

    /// `HoldPreparation`: enforce the maintenance hold marker natively.
    pub fn hold_preparation(
        &self,
        input: &PrepareHold,
        deadline: Instant,
    ) -> Result<HoldState, String> {
        input.validate()?;
        self.prepare_container(&input.project, true, deadline)?;
        let op = if input.hold { "hold" } else { "release" };
        let body = format!("{{\"op\":\"{op}\",\"revision\":{}}}", input.revision);
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        let v =
            json::decode_strict(&raw).map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let m = json::bind_root(&v, "struct", HOLD_RESPONSE_SPECS, false)
            .map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let hold = HoldState::from_map(&m.take_map("hold"));
        if hold.active != input.hold {
            return Err("maintenance hold outcome not confirmed".to_string());
        }
        Ok(hold)
    }
}

#[cfg(test)]
mod candidate_tests;
#[cfg(test)]
mod execution_tests;
#[cfg(test)]
mod tests;
