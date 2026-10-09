//! Factory preparation operations: the host side of the fixed
//! `project-factory-roles` helper exchange. Ports `prepare.go` and the
//! candidate-preparation half of `factory_candidate.go`
//! (`PrepareCandidate`; run-receipt inspection stays for the factory port).
//! Helper payloads are built byte-for-byte like the Go map marshals.

use std::time::Instant;

use crate::json;
use crate::preparation::{
    self, HoldState, Prepare, PrepareHold, PrepareInspect, PrepareState, PrepareStop,
};
use crate::project::{Executor, Runtime};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

mod candidate;
mod helper;
mod paths;
mod source;
mod state;
mod tools;

pub use self::paths::{
    path_join, preparation_paths, prepare_id_map, single_line, valid_resolved_tool_path,
};
pub use self::state::{map_preparation_state, LauncherEvidence, PreparationObservationError};

#[derive(Default)]
struct StopResponse {
    stopped: String,
    retirement: String,
    known: Option<bool>,
}

impl<'de> Deserialize<'de> for StopResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct StopVisitor;
        impl<'de> Visitor<'de> for StopVisitor {
            type Value = StopResponse;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a preparation stop response")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = StopResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("stopped") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.stopped = v;
                        }
                    } else if key.eq_ignore_ascii_case("retirement") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.retirement = v;
                        }
                    } else if key.eq_ignore_ascii_case("known") {
                        out.known = map.next_value::<Option<bool>>()?;
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["stopped", "retirement", "known"],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(StopVisitor)
    }
}

#[derive(Default)]
struct HoldResponse {
    hold: HoldState,
}

impl<'de> Deserialize<'de> for HoldResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct HoldVisitor;
        impl<'de> Visitor<'de> for HoldVisitor {
            type Value = HoldResponse;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a maintenance hold response")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = HoldResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("hold") {
                        if let Some(v) = map.next_value::<Option<HoldState>>()? {
                            out.hold = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(&key, &["hold"]));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(HoldVisitor)
    }
}

impl<E: Executor> Runtime<E> {
    /// `inspectPreparationState`: authoritative observed state for one identity.
    pub(crate) fn inspect_preparation_state(
        &self,
        project: &str,
        id: &str,
        container: &str,
        deadline: Instant,
        worker_deadline: Option<&str>,
    ) -> Result<PrepareState, PreparationObservationError> {
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(id));
        if let Some(worker_deadline) = worker_deadline {
            body.push_str(",\"deadline\":");
            body.push_str(&json::quote(worker_deadline));
        }
        body.push_str(",\"op\":\"inspect\"}");
        let raw = self
            .factory_helper(project, &body, deadline)
            .map_err(PreparationObservationError::Invalid)?;
        let mut state = map_preparation_state(container, &raw)?;
        state.id = id.to_string();
        state.project = project.to_string();
        Ok(state)
    }

    /// `Prepare`: approve, clone, verify, resolve tools, record, start.
    pub fn prepare(&self, input: &Prepare, deadline: Instant) -> Result<PrepareState, String> {
        self.prepare_inner(input, deadline, None)
    }

    fn prepare_inner(
        &self,
        input: &Prepare,
        deadline: Instant,
        worker_deadline: Option<&str>,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let prep = &input.preparation;
        let container = self.prepare_container(&prep.project, true, deadline)?;
        self.factory_helper(&prep.project, "{\"op\":\"ensure\"}", deadline)?;
        if let Err(err) = self.approve_preparation(prep, &input.setup, deadline) {
            if let Ok(stopped) = self.inspect_preparation_state(
                &prep.project,
                &prep.id,
                &container,
                deadline,
                worker_deadline,
            ) {
                if stopped.stopped {
                    return Ok(stopped);
                }
            }
            return Err(err);
        }
        let state = self
            .inspect_preparation_state(
                &prep.project,
                &prep.id,
                &container,
                deadline,
                worker_deadline,
            )
            .map_err(|error| error.to_string())?;
        if state.stopped || state.ready || state.phase == preparation::PREPARE_FAILED {
            return Ok(state);
        }
        self.clone_preparation_source(prep, deadline)?;
        let verified = self.verify_launcher_environment(prep, deadline)?;
        let (tools, missing) = self.resolve_preparation_tools(prep, deadline)?;
        self.record_preparation_tools(prep, &tools, &missing, &verified, deadline)?;
        if !missing.is_empty() || !verified.refusal.is_empty() {
            return self
                .inspect_preparation_state(
                    &prep.project,
                    &prep.id,
                    &container,
                    deadline,
                    worker_deadline,
                )
                .map_err(|error| error.to_string());
        }
        let mut body = String::from("{");
        if let Some(worker_deadline) = worker_deadline {
            body.push_str("\"deadline\":");
            body.push_str(&json::quote(worker_deadline));
            body.push(',');
        }
        body.push_str("\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"op\":\"start\"}");
        if let Err(err) = self.factory_helper(&prep.project, &body, deadline) {
            // Go returns the re-observed state alongside the error; every
            // caller drops the state on error, so only the error crosses.
            let _ = self.inspect_preparation_state(
                &prep.project,
                &prep.id,
                &container,
                deadline,
                worker_deadline,
            );
            return Err(err);
        }
        self.inspect_preparation_state(
            &prep.project,
            &prep.id,
            &container,
            deadline,
            worker_deadline,
        )
        .map_err(|error| error.to_string())
    }

    /// `InspectPreparation`: authoritative state, never mutating.
    pub fn inspect_preparation(
        &self,
        input: &PrepareInspect,
        deadline: Instant,
    ) -> Result<PrepareState, PreparationObservationError> {
        input
            .validate()
            .map_err(PreparationObservationError::Invalid)?;
        let container = self
            .prepare_container(&input.project, false, deadline)
            .map_err(PreparationObservationError::Invalid)?;
        self.inspect_preparation_state(&input.project, &input.id, &container, deadline, None)
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
        let response: StopResponse = json::decode_strict_as(&raw).map_err(|_| ERR.to_string())?;
        let retirement = response.retirement;
        if response.stopped != input.id
            || response.known.is_none()
            || (retirement != "confirmed" && retirement != "uncertain")
        {
            return Err(ERR.to_string());
        }
        if response.known == Some(false) {
            return Ok(PrepareState {
                id: input.id.clone(),
                project: input.project.clone(),
                phase: preparation::PREPARE_STOPPED.to_string(),
                container,
                stopped: true,
                retirement,
                ..Default::default()
            });
        }
        let mut state = self
            .inspect_preparation_state(&input.project, &input.id, &container, deadline, None)
            .map_err(|error| error.to_string())?;
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
        let response: HoldResponse =
            json::decode_strict_as(&raw).map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let hold = response.hold;
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
