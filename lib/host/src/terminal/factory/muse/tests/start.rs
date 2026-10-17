use super::super::commands::muse_start_gate_script;
use super::super::paths::factory_muse_binding;
use super::common::{deadline, inspect_json, make_service, muse_lease, ok, FakeExec, CID};
use crate::terminal::factory::native::shell_quote;
use crate::terminal::factory::run::{MAX_FACTORY_PROMPT, MAX_REASSESS_PROMPT};
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
        svc.factory_muse_start(&Lease::default(), cred, b"prompt", deadline(), "implement")
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, b"nope", b"prompt", deadline(), "implement")
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, b"", b"prompt", deadline(), "implement")
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_start(&lease, cred, b"", deadline(), "implement")
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Stale incarnation.
    let other = "f".repeat(64);
    let stale_json = inspect_json().replace(CID, &other);
    let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
    assert_eq!(
        svc.factory_muse_start(&lease, cred, b"prompt", deadline(), "implement")
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
    svc.factory_muse_start(&lease, cred, b"do work", deadline(), "implement")
        .unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 6);
    assert_eq!(calls[1].0, cred);
    assert_eq!(calls[2].0, cred);
    assert_eq!(calls[3].0, b"do work");
    assert!(calls[4].0.is_empty());
    assert!(
        calls[1].2[6].contains(&shell_quote(&p.credential)),
        "{}",
        calls[1].2[6]
    );
    assert!(
        calls[2].2[6].contains(&shell_quote(&p.auth)),
        "{}",
        calls[2].2[6]
    );
    assert!(
        calls[3].2[6].contains(&shell_quote(&p.prompt)),
        "{}",
        calls[3].2[6]
    );
    assert!(
        calls[4].2[6].contains(&shell_quote(&p.marker)),
        "{}",
        calls[4].2[6]
    );
    assert_eq!(calls[5].2[5], muse_start_gate_script(&p));
}

/// Staged-prompt bounds follow the admitted phase: repair prompts may
/// carry the full inventory while other phases keep the standard bound.
#[test]
fn start_phase_prompt_bounds() {
    let lease = muse_lease();
    let cred = b"{\"schema_version\":1,\"providers\":{}}";
    let big = vec![b'x'; MAX_REASSESS_PROMPT];
    // Repair at the inventory bound stages the verbatim bytes.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_muse_start(&lease, cred, &big, deadline(), "repair")
        .unwrap();
    assert_eq!(svc.exec.calls()[3].0, big);
    // Standard phases refuse the inventory size without exec.
    let svc = make_service(FakeExec::new(vec![]));
    for phase in ["implement", "verify", "publish"] {
        assert_eq!(
            svc.factory_muse_start(&lease, cred, &big, deadline(), phase)
                .unwrap_err(),
            ERR_DENIED
        );
    }
    // One byte past the exception refuses for every phase.
    let over = vec![b'x'; MAX_REASSESS_PROMPT + 1];
    for phase in ["repair", "reassess", "implement"] {
        assert_eq!(
            svc.factory_muse_start(&lease, cred, &over, deadline(), phase)
                .unwrap_err(),
            ERR_DENIED
        );
    }
    // The standard bound itself still stages for ordinary phases.
    let ordinary = vec![b'x'; MAX_FACTORY_PROMPT];
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_muse_start(&lease, cred, &ordinary, deadline(), "implement")
        .unwrap();
    assert_eq!(svc.exec.calls()[3].0, ordinary);
}
