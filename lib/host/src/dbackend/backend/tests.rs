use crate::dbackend::test_support::{backend, backend_with_muse};

use super::*;

// -- decode/gate layer: invalid bodies fail before any executor touch --

#[test]
fn profile_rejects_unknown_fields() {
    assert!(matches!(
        backend().profile(b"{\"unexpected\":1}"),
        Err(BackendError::Internal)
    ));
    assert!(matches!(
        backend().profile(b"[]"),
        Err(BackendError::Internal)
    ));
}

#[test]
fn create_checks_identity_before_validate() {
    // Bad ID with good owner: Go's "invalid project identity" path.
    assert!(matches!(
        backend().create(b"{\"id\":\"bad id!\",\"owner\":1}"),
        Err(BackendError::Internal)
    ));
    // Good ID with bad owner: same path (never reaches Validate).
    assert!(matches!(
        backend().create(b"{\"id\":\"p0123456789abcdef01234567\",\"owner\":0}"),
        Err(BackendError::Internal)
    ));
    // Unknown fields rejected even with valid identity.
    assert!(matches!(
        backend().create(b"{\"id\":\"p0123456789abcdef01234567\",\"owner\":1,\"x\":1}"),
        Err(BackendError::Internal)
    ));
}

#[test]
fn targeted_routes_validate_id_only() {
    for route in ["inspect", "os", "connection"] {
        let result = match route {
            "inspect" => backend().inspect(b"{\"id\":\"bad\"}"),
            "os" => backend().observe_os(b"{\"id\":\"bad\"}"),
            _ => backend().connection(b"{\"id\":\"bad\"}"),
        };
        assert!(matches!(result, Err(BackendError::Internal)), "{route}");
    }
}

#[test]
fn lifecycle_rejects_bad_action_without_exec() {
    // Unknown action fails in the runtime gate before any container
    // inspection; unknown fields fail at decode.
    assert!(matches!(
        backend().lifecycle(b"{\"project\":\"p0123456789abcdef01234567\",\"action\":\"bogus\"}"),
        Err(BackendError::Internal)
    ));
    assert!(matches!(
        backend()
            .lifecycle(b"{\"project\":\"p0123456789abcdef01234567\",\"action\":\"start\",\"x\":1}"),
        Err(BackendError::Internal)
    ));
}

#[test]
fn factory_routes_report_unavailable_root() {
    // Factory opens lazily; the file-as-directory root fails the open,
    // mirroring Go's `factoryRun` 503 before any body decode.
    assert!(matches!(
        backend().factory_launch(b"{}"),
        Err(BackendError::Unavailable)
    ));
    assert!(matches!(
        backend().factory_harness(b"{}", "sha256:0"),
        Err(BackendError::Unavailable)
    ));
}
#[test]
fn identity_launch_without_broker_is_internal() {
    // No broker listens at the test socket: the real client fails to
    // connect and the launch reports 500, exactly like Go's
    // `identityHandler` on broker errors.
    let start = br#"{"connection_id":"c","project_id":"p0123456789abcdef01234567","actor_id":"1","login":"l","scope":"s","cols":80,"rows":24}"#;
    assert!(matches!(
        backend().identity_launch(start),
        Err(BackendError::Internal)
    ));
    // The muse-configured backend constructs (real mserve runtime, no
    // I/O at open).
    let _ = backend_with_muse();
}
#[test]
fn factory_error_mapping() {
    use crate::factory::FactoryError;
    assert!(matches!(
        map_factory_err(FactoryError::NotFound, BackendError::OutputStale),
        BackendError::NotFound
    ));
    assert!(matches!(
        map_factory_err(FactoryError::Stale, BackendError::OutputStale),
        BackendError::OutputStale
    ));
    assert!(matches!(
        map_factory_err(FactoryError::Stale, BackendError::ExportStale),
        BackendError::ExportStale
    ));
    assert!(matches!(
        map_factory_err(FactoryError::ExportCandidate, BackendError::OutputStale),
        BackendError::ExportCandidate
    ));
    assert!(matches!(
        map_factory_err(FactoryError::ExportBounds, BackendError::OutputStale),
        BackendError::ExportBounds
    ));
    assert!(matches!(
        map_factory_err(
            FactoryError::Msg("boom".to_string()),
            BackendError::OutputStale
        ),
        BackendError::Internal
    ));
}
#[test]
fn terminal_accept_mints_unique_sessions() {
    let backend = backend();
    let first = backend.terminal_accept("k").unwrap().id;
    let second = backend.terminal_accept("k").unwrap().id;
    assert_ne!(first, second);
}
