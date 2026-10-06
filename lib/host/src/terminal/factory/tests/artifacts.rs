use super::super::artifacts::{
    export_argv, takeover_destination, takeover_steps, ERR_FACTORY_EXPORT_BOUNDS,
    ERR_FACTORY_EXPORT_CANDIDATE, MAX_FACTORY_EXPORT_BUNDLE,
};
use super::super::codex::tests::{
    deadline, err, inspect_json, make_service, ok, FakeExec, CID, COMMIT, PID, PREP, RID, ROLE,
};

use crate::terminal::{self, ERR_DENIED, ERR_STALE};

#[test]
fn export_flows() {
    let src = format!("/home/{ROLE}/checkouts/{PREP}");
    // Validation denies.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, "dev", PREP, COMMIT, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, "short", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_export_bundle(PID, "short", ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Success pins the clean-env argv.
    let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("bundle-bytes")]));
    let bundle = svc
        .factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
        .unwrap();
    assert_eq!(bundle, b"bundle-bytes");
    let calls = svc.exec.calls();
    assert_eq!(calls[1].2, export_argv(CID, ROLE, &src, COMMIT));
    assert!(calls[1].2.contains(&"soda-export".to_string()));
    assert!(calls[1]
        .2
        .contains(&(MAX_FACTORY_EXPORT_BUNDLE + 1).to_string()));
    // Sentinel verdicts.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("soda-export-missing\n"),
    ]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        terminal::ERR_NOT_FOUND
    );
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("soda-export-invalid\n"),
    ]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        ERR_FACTORY_EXPORT_CANDIDATE
    );
    // Bounds.
    let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("")]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        ERR_FACTORY_EXPORT_BOUNDS
    );
    // Exec failure with a live deadline.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        "candidate export execution unconfirmed"
    );
    // Stale incarnation.
    let other = "f".repeat(64);
    let stale_json = inspect_json().replace(CID, &other);
    let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
    assert_eq!(
        svc.factory_export_bundle(PID, CID, ROLE, PREP, COMMIT, deadline())
            .unwrap_err(),
        ERR_STALE
    );
}

#[test]
fn takeover_flows() {
    let dest = takeover_destination("dev", RID).unwrap();
    let src = format!("/home/{ROLE}/checkouts/{PREP}");
    // Validation denies.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, "dev", PREP, "dev", RID, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "root", RID, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Existing destination reuses.
    let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("")]));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap(),
        (dest.clone(), true)
    );
    // Missing source.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        err("exit status 1"),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap_err(),
        "takeover found no retained work"
    );
    // Full copy pins every step.
    let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
    script.extend(std::iter::repeat_with(|| ok("")).take(7));
    script.push(err("exit status 1")); // dest still absent
    script.push(ok("")); // mv
    let svc = make_service(FakeExec::new(script));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap(),
        (dest.clone(), false)
    );
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 12);
    let steps = takeover_steps(&src, &dest, "dev");
    for (i, step) in steps.iter().enumerate() {
        let mut want = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            CID.to_string(),
        ];
        want.extend(step.clone());
        assert_eq!(calls[3 + i].2, want, "step {i}");
    }
    assert_eq!(
        calls[10].2[3..],
        ["/usr/bin/test".to_string(), "-e".to_string(), dest.clone()]
    );
    assert_eq!(
        calls[11].2[3..7],
        [
            "/usr/bin/mv".to_string(),
            "-T".to_string(),
            format!("{dest}.partial"),
            dest.clone()
        ]
    );
    // Step failures propagate raw.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        err("exit status 1"),
        ok(""),
        err("exit status 3"),
    ]));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap_err(),
        "exit status 3"
    );
    // Racing destination reuses after cleanup.
    let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
    script.extend(std::iter::repeat_with(|| ok("")).take(7));
    script.push(ok("")); // dest appeared
    script.push(ok("")); // rm partial
    let svc = make_service(FakeExec::new(script));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap(),
        (dest.clone(), true)
    );
    // Move failure reports unconfirmed after cleanup.
    let mut script = vec![ok(&inspect_json()), err("exit status 1"), ok("")];
    script.extend(std::iter::repeat_with(|| ok("")).take(7));
    script.push(err("exit status 1"));
    script.push(err("exit status 1")); // mv fails
    script.push(ok("")); // rm partial
    let svc = make_service(FakeExec::new(script));
    assert_eq!(
        svc.factory_takeover_copy(PID, CID, ROLE, PREP, "dev", RID, deadline())
            .unwrap_err(),
        "takeover destination was not confirmed"
    );
}
