use super::common::{
    container_id, sample_preparation, sample_prepare_state, sample_run, sample_state,
};
use crate::factory::*;
use crate::preparation::{HoldState, PrepareHold, PrepareInspect, PrepareState, PrepareStop};

#[test]
fn factory_facade_confirmations() {
    let run = sample_run();
    let state = sample_state(&run);
    assert!(confirm_factory_launch(&run, &state).is_ok());
    let mut bad = state.clone();
    bad.role = "soda-reviewer".to_string();
    assert_eq!(
        confirm_factory_launch(&run, &bad).unwrap_err(),
        "native factory launch did not return the admitted run"
    );

    let inspect = FactoryInspect {
        project: run.project.clone(),
        id: run.id.clone(),
    };
    assert!(confirm_factory_inspect(&inspect, &state).is_ok());
    assert_eq!(
        confirm_factory_inspect(&inspect, &FactoryState::default()).unwrap_err(),
        "native factory observation does not match its identity"
    );

    let stop = FactoryStop {
        project: run.project.clone(),
        id: run.id.clone(),
    };
    let mut stopped = state.clone();
    stopped.phase = "stopped".to_string();
    stopped.retirement = "confirmed".to_string();
    assert!(confirm_factory_stop(&stop, &stopped).is_ok());
    stopped.retirement = "bogus".to_string();
    assert_eq!(
        confirm_factory_stop(&stop, &stopped).unwrap_err(),
        "native factory stop was not confirmed"
    );

    let output_req = FactoryOutput {
        project: run.project.clone(),
        id: run.id.clone(),
        offset: 0,
        limit: 1,
    };
    let output_state = FactoryOutputState {
        id: run.id.clone(),
        project: run.project.clone(),
        phase: "completed".to_string(),
        ..FactoryOutputState::default()
    };
    assert!(confirm_factory_output(&output_req, &output_state).is_ok());
    let mut bad_output = output_state.clone();
    bad_output.phase = "bogus".to_string();
    assert_eq!(
        confirm_factory_output(&output_req, &bad_output).unwrap_err(),
        "native factory output does not match its identity"
    );

    let export_req = FactoryExport {
        project: run.project.clone(),
        id: run.id.clone(),
        role: run.role.clone(),
        preparation: run.preparation.clone(),
        candidate: "1".repeat(40),
    };
    let export_state = FactoryExportState {
        id: run.id.clone(),
        project: run.project.clone(),
        phase: "completed".to_string(),
        container: container_id(),
        candidate: "1".repeat(40),
        bundle: "eA==".to_string(),
    };
    assert!(confirm_factory_export(&export_req, &export_state).is_ok());
    let mut bad_export = export_state.clone();
    bad_export.bundle = String::new();
    assert_eq!(
        confirm_factory_export(&export_req, &bad_export).unwrap_err(),
        "native factory export does not match its identity"
    );

    let takeover_req = FactoryTakeover {
        project: run.project.clone(),
        id: run.id.clone(),
        member: "alice".to_string(),
    };
    let takeover_result = TakeoverResult {
        id: run.id.clone(),
        project: run.project.clone(),
        member: "alice".to_string(),
        destination: takeover_destination("alice", &run.id),
        reused: false,
    };
    assert!(confirm_factory_takeover(&takeover_req, &takeover_result).is_ok());
    assert_eq!(
        confirm_factory_takeover(
            &takeover_req,
            &TakeoverResult {
                member: "bob".to_string(),
                ..takeover_result.clone()
            }
        )
        .unwrap_err(),
        "native factory takeover did not return the admitted destination"
    );

    let cand_req = FactoryCandidateInspect {
        project: run.project.clone(),
        id: run.id.clone(),
    };
    let cand_state = FactoryCandidateState {
        id: run.id.clone(),
        project: run.project.clone(),
        container: container_id(),
        candidate: "1".repeat(40),
        dirty: false,
    };
    assert!(confirm_factory_candidate_inspect(&cand_req, &cand_state).is_ok());
    let mut bad_cand = cand_state.clone();
    bad_cand.container = "short".to_string();
    assert_eq!(
        confirm_factory_candidate_inspect(&cand_req, &bad_cand).unwrap_err(),
        "native candidate observation does not match its identity"
    );
}

#[test]
fn factory_status_mappings() {
    assert_eq!(
        factory_not_found_status(404).as_deref(),
        Some(ERR_RUN_NOT_FOUND)
    );
    assert_eq!(factory_not_found_status(500), None);
    assert_eq!(
        factory_output_status_error(404).as_deref(),
        Some(ERR_RUN_NOT_FOUND)
    );
    assert_eq!(
        factory_output_status_error(409).as_deref(),
        Some(ERR_RUN_STALE)
    );
    assert_eq!(factory_output_status_error(200), None);
    assert_eq!(
        factory_export_status_error(404).as_deref(),
        Some(ERR_RUN_NOT_FOUND)
    );
    assert_eq!(
        factory_export_status_error(409).as_deref(),
        Some(ERR_RUN_STALE)
    );
    assert_eq!(
        factory_export_status_error(422).as_deref(),
        Some("export candidate is not recorded")
    );
    assert_eq!(
        factory_export_status_error(413).as_deref(),
        Some("candidate export exceeds bounds")
    );
    assert_eq!(factory_export_status_error(500), None);
    assert_eq!(
        factory_candidate_status_error(409).as_deref(),
        Some(ERR_RUN_STALE)
    );
    assert_eq!(
        factory_candidate_status_error(404).as_deref(),
        Some(ERR_RUN_NOT_FOUND)
    );
    assert_eq!(factory_candidate_status_error(200), None);
}

#[test]
fn prepare_facade_confirmations() {
    let prep = sample_preparation();
    let state = sample_prepare_state(&prep);
    assert!(confirm_prepare(&prep, &state).is_ok());
    assert!(confirm_prepare_candidate(&prep, &state).is_ok());
    let mut bad = state.clone();
    bad.setup_digest = "1".repeat(64);
    assert_eq!(
        confirm_prepare(&prep, &bad).unwrap_err(),
        "native preparation result does not match its identity"
    );
    assert_eq!(
        confirm_prepare_candidate(&prep, &bad).unwrap_err(),
        "native candidate preparation does not match its identity"
    );

    let inspect = PrepareInspect {
        project: prep.project.clone(),
        id: prep.id.clone(),
    };
    assert!(confirm_inspect_preparation(&inspect, &state).is_ok());
    assert_eq!(
        confirm_inspect_preparation(&inspect, &PrepareState::default()).unwrap_err(),
        "native preparation observation does not match its identity"
    );

    let stop = PrepareStop {
        project: prep.project.clone(),
        id: prep.id.clone(),
    };
    let mut stopped = state.clone();
    stopped.stopped = true;
    stopped.retirement = "uncertain".to_string();
    assert!(confirm_stop_preparation(&stop, &stopped).is_ok());
    assert_eq!(
        confirm_stop_preparation(&stop, &state).unwrap_err(),
        "native preparation stop was not confirmed"
    );

    let hold = PrepareHold {
        project: prep.project.clone(),
        hold: true,
        revision: 2,
    };
    assert!(confirm_hold_preparation(
        &hold,
        &HoldState {
            active: true,
            revision: 2
        }
    )
    .is_ok());
    assert_eq!(
        confirm_hold_preparation(&hold, &HoldState::default()).unwrap_err(),
        "native maintenance hold outcome not confirmed"
    );
}
