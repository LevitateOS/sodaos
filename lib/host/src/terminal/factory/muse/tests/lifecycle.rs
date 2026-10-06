use super::common::{deadline, inspect_json, make_service, muse_lease, ok, FakeExec, IID, PID};
use crate::terminal::factory::run::FACTORY_SCOPE_CODEX;
use crate::terminal::{self, ERR_DENIED};

#[test]
fn wait_output_live() {
    let lease = muse_lease();
    // Wait: inactive unit, exit file, last message.
    let svc = make_service(FakeExec::new(vec![
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok("0\n"),
        ok("{\"status\":\"completed\"}\n"),
    ]));
    let (code, out) = svc.factory_muse_wait(&lease, deadline()).unwrap();
    assert_eq!((code, out.as_str()), (0, "{\"status\":\"completed\"}\n"));
    // Output slice: container, stat, bounded read.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("11\n"),
        ok("hello world"),
    ]));
    let binding = lease.binding.clone().unwrap();
    let slice = svc
        .factory_muse_output(PID, &binding, 0, 100, deadline())
        .unwrap();
    assert_eq!(slice.total, 11);
    assert_eq!(slice.data, b"hello world");
    // Live: scope-gated attestation.
    let svc = make_service(FakeExec::new(vec![ok(&format!(
        "ActiveState=active\nInvocationID={IID}\n"
    ))]));
    assert!(svc.factory_muse_live(&binding, deadline()));
    let mut dead = binding.clone();
    dead.scope = FACTORY_SCOPE_CODEX.to_string();
    let svc = make_service(FakeExec::new(vec![]));
    assert!(!svc.factory_muse_live(&dead, deadline()));
}

#[test]
fn stop_and_capture() {
    let lease = muse_lease();
    // Stop: pid, systemctl, await, exists, retire, pid.
    let svc = make_service(FakeExec::new(vec![
        ok("111 222\n"),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_muse_stop(&lease, deadline()).unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls[0].2[3], "/usr/bin/cat");
    assert!(calls[0].2[4].ends_with("supervisor.pid"));
    // Capture echoes the daemon-staged auth.json copy (not the
    // CLI's live lookup file).
    let cred = "{\"schema_version\":1,\"providers\":{}}";
    let svc = make_service(FakeExec::new(vec![ok(cred)]));
    let back = svc.factory_muse_capture(&lease, deadline()).unwrap();
    assert_eq!(back, cred.as_bytes());
    let calls = svc.exec.calls();
    assert!(calls[0].2[6].ends_with("muse-auth.json"));
}

#[test]
fn finish_denied_for_muse() {
    // Muse borrows: the broker forgets on return and never calls
    // finish (same denial as the interactive muse runtime).
    let lease = muse_lease();
    let delivery = terminal::Delivery {
        lease,
        credential: Some(b"{}".to_vec()),
    };
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_identity_operation("finish", &delivery, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
}
