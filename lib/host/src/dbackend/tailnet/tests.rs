use crate::daemon::backend::ExecBackend;
use crate::dbackend::test_support::backend;
use crate::{project, tcontrol};

use super::*;

// -- integrated tailnet control plane (Lane T, no scaffold) --

/// Backend whose tailnet controls observe scratch state paths that are
/// never created, so the real control plane answers deterministically
/// (default policy view, unavailable host) without appliance paths.
fn backend_with_scratch_tailnet(name: &str) -> DaemonBackend {
    let mut backend = backend();
    let root = std::path::PathBuf::from("target/dbackend-test")
        .join(format!("{name}-{}", std::process::id()));
    let opts = tcontrol::Options {
        state_dir: root.join("soda-tailnet"),
        socket: root.join("tailscaled.sock"),
        ..tcontrol::Options::default()
    };
    backend.tailnet = tcontrol::Control::new(project::Native, opts.clone());
    backend.companion.tailnet = tcontrol::Control::new(project::Native, opts);
    backend
}

#[test]
fn tailnet_settings_and_options_answer_without_appliance_state() {
    let backend = backend_with_scratch_tailnet("settings");
    let settings = backend.tailnet("settings", b"{}").expect("settings");
    let body = String::from_utf8(settings).expect("utf8 settings");
    assert!(body.contains("\"host_unavailable\":true"), "{body}");
    assert!(body.contains("\"revision\":\"0\""), "{body}");
    let options = backend.tailnet("options", b"{}").expect("options");
    let obody = String::from_utf8(options).expect("utf8 options");
    assert!(obody.contains("\"revision\":\"0\""), "{obody}");
}

#[test]
fn tailnet_host_and_enrollment_reject_malformed_bodies() {
    let backend = backend_with_scratch_tailnet("reject");
    assert!(matches!(
        backend.tailnet("host", b"{"),
        Err(BackendError::Invalid)
    ));
    assert!(matches!(
        backend.tailnet("enrollment", b"{"),
        Err(BackendError::Invalid)
    ));
}
#[test]
fn tailnet_rejects_bad_input_as_invalid() {
    assert!(matches!(
        backend().tailnet("settings", b"{\"x\":1}"),
        Err(BackendError::Invalid)
    ));
    assert!(matches!(
        backend().tailnet("bogus", b"{}"),
        Err(BackendError::Invalid)
    ));
    // Bad project id fails validation before any container resolve.
    assert!(matches!(
        backend().tailnet("project", b"{\"project\":\"bad\",\"action\":\"inspect\"}"),
        Err(BackendError::Invalid)
    ));
    // Policy admits inspect only.
    let pid = "p0123456789abcdef01234567";
    let body = format!(
        "{{\"project\":\"{pid}\",\"action\":\"enable\",\"revision\":\"0\",\"binding\":\"{pid}\",\"confirm_id\":\"{pid}\"}}"
    );
    assert!(matches!(
        backend().tailnet("policy", body.as_bytes()),
        Err(BackendError::Invalid)
    ));
}

#[test]
fn tailnet_validate_rules_match_go() {
    use crate::tailnet_domain::ProjectRequest;
    let pid = "p0123456789abcdef01234567".to_string();
    let rev = "0123456789abcdef0123456789abcdef".to_string();
    let base = ProjectRequest {
        project: pid.clone(),
        action: "inspect".to_string(),
        revision: String::new(),
        binding: String::new(),
        confirm_id: String::new(),
    };
    assert!(validate_project_request(&base).is_ok());
    // Inspect forbids mutation fields.
    let mut bad = base.clone();
    bad.revision = rev.clone();
    assert!(validate_project_request(&bad).is_err());
    // Enable needs 32-hex binding + self confirm + valid revision.
    let ok = ProjectRequest {
        action: "enable".to_string(),
        revision: rev.clone(),
        binding: rev.clone(),
        confirm_id: pid.clone(),
        ..base.clone()
    };
    assert!(validate_project_request(&ok).is_ok());
    let mut bad_binding = ok.clone();
    bad_binding.binding = "xyz".to_string();
    assert!(validate_project_request(&bad_binding).is_err());
    let mut bad_confirm = ok.clone();
    bad_confirm.confirm_id = "pffffffffffffffffffffffff".to_string();
    assert!(validate_project_request(&bad_confirm).is_err());
    // Disable forbids binding.
    let mut disable = ok.clone();
    disable.action = "disable".to_string();
    assert!(validate_project_request(&disable).is_err());
    disable.binding.clear();
    assert!(validate_project_request(&disable).is_ok());
    // Unknown action rejected.
    let mut unknown = ok.clone();
    unknown.action = "explode".to_string();
    assert!(validate_project_request(&unknown).is_err());
    assert!(!valid_revision("ABCDEF0123456789abcdef0123456789"));
    assert!(valid_revision("0"));
}

#[test]
fn tailnet_view_encode_matches_go_omitempty() {
    use crate::tailnet_domain::ProjectView;
    let empty = ProjectView::default();
    assert_eq!(
        encode_project_view(&empty),
        "{\"saved\":false,\"project\":\"\",\"revision\":\"\",\"binding\":\"\",\"enabled\":false,\"state\":\"\",\"outcome\":\"\"}"
    );
    let full = ProjectView {
        available_binding: "b".to_string(),
        addresses: vec!["10.0.0.1".to_string()],
        saved: true,
        project: "p".to_string(),
        enabled: true,
        state: "ready".to_string(),
        ..ProjectView::default()
    };
    assert_eq!(
        encode_project_view(&full),
        "{\"available_binding\":\"b\",\"addresses\":[\"10.0.0.1\"],\"saved\":true,\"project\":\"p\",\"revision\":\"\",\"binding\":\"\",\"enabled\":true,\"state\":\"ready\",\"outcome\":\"\"}"
    );
}

#[test]
fn tailnet_error_mapping() {
    assert!(matches!(
        map_tailnet_err("tailscale: invalid request".to_string()),
        BackendError::Invalid
    ));
    assert!(matches!(
        map_tailnet_err("revision conflict".to_string()),
        BackendError::ExportStale
    ));
    assert!(matches!(
        map_tailnet_err("action unsupported".to_string()),
        BackendError::ExportCandidate
    ));
    assert!(matches!(
        map_tailnet_err("tailnet unavailable".to_string()),
        BackendError::Unavailable
    ));
    assert!(matches!(
        map_tailnet_err("boom".to_string()),
        BackendError::Internal
    ));
}
