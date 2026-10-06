use super::common::{deadline, inspect_json, make_service, muse_lease, ok, FakeExec, CID};
use crate::terminal::factory::tcodex;
use crate::terminal::factory::tmuse::*;
use crate::terminal::{Lease, ERR_DENIED};

#[test]
fn start_flows() {
    let lease = muse_lease();
    let p = factory_muse_binding(&lease).unwrap();
    let cred = b"{\"schema_version\":1,\"providers\":{}}";
    // Denials before exec. The credential is opaque: any valid JSON
    // stages (even `{}`), only non-credential bytes refuse.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_muse_start(&Lease::default(), cred, b"prompt", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, b"nope", b"prompt", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, b"", b"prompt", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, cred, b"", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Stale incarnation.
    let other = "f".repeat(64);
    let stale_json = inspect_json().replace(CID, &other);
    let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
    assert_eq!(
        svc.factory_muse_start(&lease, cred, b"prompt", deadline())
            .unwrap_err(),
        crate::terminal::ERR_STALE
    );
    // Success stages the verbatim bytes twice (echo copy + CLI
    // lookup), then prompt, marker, then the gate.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_muse_start(&lease, cred, b"do work", deadline())
        .unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 6);
    assert_eq!(calls[1].0, cred);
    assert_eq!(calls[2].0, cred);
    assert_eq!(calls[3].0, b"do work");
    assert!(calls[4].0.is_empty());
    assert!(
        calls[1].2[6].contains(&tcodex::shell_quote(&p.credential)),
        "{}",
        calls[1].2[6]
    );
    assert!(
        calls[2].2[6].contains(&tcodex::shell_quote(&p.auth)),
        "{}",
        calls[2].2[6]
    );
    assert!(
        calls[3].2[6].contains(&tcodex::shell_quote(&p.prompt)),
        "{}",
        calls[3].2[6]
    );
    assert!(
        calls[4].2[6].contains(&tcodex::shell_quote(&p.marker)),
        "{}",
        calls[4].2[6]
    );
    assert_eq!(calls[5].2[5], muse_start_gate_script(&p));
}
