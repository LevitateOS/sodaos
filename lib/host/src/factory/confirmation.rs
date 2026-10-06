use crate::domain;
use crate::preparation::{
    self, HoldState, Preparation, PrepareHold, PrepareInspect, PrepareState, PrepareStop,
};

use super::{
    valid_factory_phase, FactoryCandidateInspect, FactoryCandidateState, FactoryExport,
    FactoryExportState, FactoryHarnessPin, FactoryInspect, FactoryOutput, FactoryOutputState,
    FactoryRun, FactoryState, FactoryStop, FactoryTakeover, TakeoverResult, ERR_RUN_NOT_FOUND,
    ERR_RUN_STALE,
};

// ---------- facade confirmations ----------
//
// Pure client-side logic from `internal/host/{factory_client,
// factory_candidate,prepare}.go`: transport (`c.call`) arrives with the
// route layer, but identity confirmations and status mappings are exact
// here so both sides pin the same wire contract.

/// `factoryStateConfirmed`: the launch returned the admitted run.
pub fn confirm_factory_launch(run: &FactoryRun, state: &FactoryState) -> Result<(), String> {
    if state.id == run.id
        && state.project == run.project
        && state.role == run.role
        && valid_factory_phase(&state.phase)
    {
        Ok(())
    } else {
        Err("native factory launch did not return the admitted run".to_string())
    }
}

/// `FactoryInspect` client confirmation.
pub fn confirm_factory_inspect(req: &FactoryInspect, state: &FactoryState) -> Result<(), String> {
    if state.id == req.id && state.project == req.project && valid_factory_phase(&state.phase) {
        Ok(())
    } else {
        Err("native factory observation does not match its identity".to_string())
    }
}

/// `FactoryHarness` client confirmation.
pub fn confirm_factory_harness(pin: &FactoryHarnessPin) -> Result<(), String> {
    if pin.validate().is_ok() {
        Ok(())
    } else {
        Err("native factory harness pin is not usable".to_string())
    }
}

/// `FactoryStop` client confirmation.
pub fn confirm_factory_stop(req: &FactoryStop, state: &FactoryState) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && valid_factory_phase(&state.phase)
        && (state.retirement.is_empty()
            || state.retirement == "confirmed"
            || state.retirement == "uncertain")
    {
        Ok(())
    } else {
        Err("native factory stop was not confirmed".to_string())
    }
}

/// `FactoryOutput` client identity confirmation (status mapping is
/// [`factory_output_status_error`]).
pub fn confirm_factory_output(
    req: &FactoryOutput,
    state: &FactoryOutputState,
) -> Result<(), String> {
    if state.id == req.id && state.project == req.project && valid_factory_phase(&state.phase) {
        Ok(())
    } else {
        Err("native factory output does not match its identity".to_string())
    }
}

/// `FactoryExport` client identity confirmation.
pub fn confirm_factory_export(
    req: &FactoryExport,
    state: &FactoryExportState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && state.candidate == req.candidate
        && valid_factory_phase(&state.phase)
        && !state.bundle.is_empty()
        && state.bundle.len() <= 8 << 20
    {
        Ok(())
    } else {
        Err("native factory export does not match its identity".to_string())
    }
}

/// `FactoryTakeover` client confirmation.
pub fn confirm_factory_takeover(
    req: &FactoryTakeover,
    result: &TakeoverResult,
) -> Result<(), String> {
    if result.validate().is_ok()
        && result.id == req.id
        && result.project == req.project
        && result.member == req.member
    {
        Ok(())
    } else {
        Err("native factory takeover did not return the admitted destination".to_string())
    }
}

/// `FactoryInspectCandidate` client identity confirmation.
pub fn confirm_factory_candidate_inspect(
    req: &FactoryCandidateInspect,
    state: &FactoryCandidateState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && domain::valid_container_id(&state.container)
        && preparation::valid_commit(&state.candidate)
    {
        Ok(())
    } else {
        Err("native candidate observation does not match its identity".to_string())
    }
}

/// `preparationIdentityConfirmed`, shared by the prepare and
/// prepare-candidate clients.
pub fn confirm_prepare(prep: &Preparation, state: &PrepareState) -> Result<(), String> {
    if state.id == prep.id
        && state.project == prep.project
        && state.role == prep.role
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
        && state.source_commit == prep.source_commit
        && state.setup_digest == prep.setup_digest
    {
        Ok(())
    } else {
        Err("native preparation result does not match its identity".to_string())
    }
}

/// `PrepareCandidate` client confirmation.
pub fn confirm_prepare_candidate(prep: &Preparation, state: &PrepareState) -> Result<(), String> {
    if state.id == prep.id
        && state.project == prep.project
        && state.role == prep.role
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
        && state.source_commit == prep.source_commit
        && state.setup_digest == prep.setup_digest
    {
        Ok(())
    } else {
        Err("native candidate preparation does not match its identity".to_string())
    }
}

/// `InspectPreparation` client confirmation.
pub fn confirm_inspect_preparation(
    req: &PrepareInspect,
    state: &PrepareState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
    {
        Ok(())
    } else {
        Err("native preparation observation does not match its identity".to_string())
    }
}

/// `StopPreparation` client confirmation.
pub fn confirm_stop_preparation(req: &PrepareStop, state: &PrepareState) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && state.stopped
        && (state.retirement == "confirmed" || state.retirement == "uncertain")
    {
        Ok(())
    } else {
        Err("native preparation stop was not confirmed".to_string())
    }
}

/// `HoldPreparation` client confirmation.
pub fn confirm_hold_preparation(req: &PrepareHold, state: &HoldState) -> Result<(), String> {
    if state.active == req.hold {
        Ok(())
    } else {
        Err("native maintenance hold outcome not confirmed".to_string())
    }
}

/// `factoryNotFound`: inspect/takeover 404 mapping.
pub fn factory_not_found_status(status: u16) -> Option<String> {
    if status == 404 {
        Some(ERR_RUN_NOT_FOUND.to_string())
    } else {
        None
    }
}

/// `factoryOutputError` status mapping: 404 and 409.
pub fn factory_output_status_error(status: u16) -> Option<String> {
    match status {
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        409 => Some(ERR_RUN_STALE.to_string()),
        _ => None,
    }
}

/// `factoryExportError` status mapping: 404, 409, 422 and 413.
pub fn factory_export_status_error(status: u16) -> Option<String> {
    match status {
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        409 => Some(ERR_RUN_STALE.to_string()),
        422 => Some("export candidate is not recorded".to_string()),
        413 => Some("candidate export exceeds bounds".to_string()),
        _ => None,
    }
}

/// `FactoryInspectCandidate` status mapping: 409, then 404.
pub fn factory_candidate_status_error(status: u16) -> Option<String> {
    match status {
        409 => Some(ERR_RUN_STALE.to_string()),
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        _ => None,
    }
}
