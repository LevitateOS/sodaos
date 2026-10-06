use super::super::codex::tests::{
    deadline, err, euid, factory_lease, factory_run, host_digest, inspect_json, make_service, ok,
    reserve_harness, FakeExec, CID, COMMIT, IID, PID, PIN, PREP, RID, ROLE,
};
use super::super::lifecycle::factory_retire;
use super::super::native::{factory_unit_name, factory_user_bus, reserve_run_argv, shell_quote};
use super::super::output::{MAX_FACTORY_OUTPUT_READ, MAX_FACTORY_OUTPUT_WINDOW};
use super::super::run::{FactoryRun, FACTORY_SCOPE_CODEX, MAX_FACTORY_PROMPT};
use super::super::tcodex::{factory_codex_binding, reserve_exec_argv, start_gate_script};

use crate::terminal::{self, Delivery, Lease, KIND_FACTORY};
use crate::terminal::{ERR_DENIED, ERR_STALE, ERR_UNCERTAIN};
use std::time::{Duration, Instant};

#[test]
fn reserve_success_argv_sequence() {
    let (_dir, harness) = reserve_harness();
    let host = host_digest();
    let guest = "/usr/local/bin/codex-factory-1.2.3";
    let unit = factory_unit_name(RID).unwrap();
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),              // project container
        ok("1001\n"),                     // id -u
        ok("1002\n"),                     // id -g
        ok(""),                           // setup script
        ok(&format!("{COMMIT}\n")),       // git rev-parse
        ok(&format!("{PIN}  {guest}\n")), // guest sha256sum (present)
        ok(&format!("{host}  /usr/local/bin/codex-code-mode-host\n")), // host probe (present)
        ok(""),                           // systemd-run
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")), // attestation
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness;
    let run = factory_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    let (binding, p) = svc
        .factory_codex_reserve(&run, &lease, PIN, 600, deadline())
        .unwrap();
    assert_eq!(binding.id, RID);
    assert_eq!(binding.project, CID);
    assert_eq!(binding.login, ROLE);
    assert_eq!((binding.uid, binding.gid), (1001, 1002));
    assert_eq!(binding.scope, FACTORY_SCOPE_CODEX);
    assert_eq!(binding.invocation_id, IID);
    assert_eq!(binding.credential_root, p.run_dir);
    assert_eq!(binding.generation, 5);
    assert_eq!(binding.child_id, PREP);
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 9);
    // systemd-run argv is byte-exact: header + podman payload.
    let bus = factory_user_bus(euid());
    let mut want_run = vec![
        "/usr/bin/env".to_string(),
        bus,
        "/usr/bin/systemd-run".to_string(),
        "--user".to_string(),
    ];
    want_run.extend(reserve_run_argv(&unit, 600));
    want_run.extend(reserve_exec_argv(CID, &run, &p, guest));
    let got_run: Vec<String> = std::iter::once(calls[7].1.clone())
        .chain(calls[7].2.clone())
        .collect();
    assert_eq!(got_run, want_run);
    // Setup script pins directories and ownership.
    assert_eq!(calls[3].1, "/usr/bin/podman");
    assert!(
        calls[3].2[5].starts_with("set -u\nmkdir -p -m 700 "),
        "{}",
        calls[3].2[5]
    );
    assert!(
        calls[3].2[5].contains("chown 1001:1002 "),
        "{}",
        calls[3].2[5]
    );
    // Attestation show pins the unit.
    assert_eq!(
        calls[8].2[..4],
        [
            factory_user_bus(euid()),
            "/usr/bin/systemctl".to_string(),
            "--user".to_string(),
            "show".to_string()
        ]
    );
    assert_eq!(calls[8].2[5], unit);
}

#[test]
fn reserve_stage_and_failure_paths() {
    let (_dir, harness) = reserve_harness();
    let guest = "/usr/local/bin/codex-factory-1.2.3";
    // Guest harness absent: cp + install path with digest check.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        err("exit status 1"),             // guest probe misses
        ok(""),                           // podman cp
        ok(&format!("{PIN}  {guest}\n")), // install digest
        err("exit status 1"),             // host probe misses
        ok(""),                           // host cp
        ok(&format!(
            "{}  /usr/local/bin/codex-code-mode-host\n",
            host_digest()
        )), // host install
        ok(""),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness.clone();
    let run = factory_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    let (binding, _) = svc
        .factory_codex_reserve(&run, &lease, PIN, 600, deadline())
        .unwrap();
    assert_eq!(binding.uid, 1001);
    let calls = svc.exec.calls();
    assert_eq!(calls[6].2[1], "cp");
    assert_eq!(calls[6].2[3], format!("{CID}:{guest}.new"));
    assert!(
        calls[7].2[5].contains(&format!("mv '{guest}.new' '{guest}'")),
        "{}",
        calls[7].2[5]
    );
    // Digest mismatch after install.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        ok(&format!("{}  {guest}\n", "f".repeat(64))),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness.clone();
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "guest harness digest differs"
    );
    // Wrong checkout commit.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(""),
        ok(&format!("{}xxx\n", &COMMIT[..37])),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness.clone();
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "factory checkout is not the assigned commit"
    );
    // systemd-run failure.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        ok(&format!("{PIN}  {guest}\n")),
        ok(&format!(
            "{}  /usr/local/bin/codex-code-mode-host\n",
            host_digest()
        )),
        err("exit status 1"),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness;
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "factory unit start unconfirmed"
    );
}

#[test]
fn reserve_unattested_unit_is_stopped() {
    let (_dir, harness) = reserve_harness();
    let guest = "/usr/local/bin/codex-factory-1.2.3";
    // Attestation never arrives: the started unit is stopped before
    // the error returns. A short deadline keeps the test fast.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        ok(&format!("{PIN}  {guest}\n")),
        ok(&format!(
            "{}  /usr/local/bin/codex-code-mode-host\n",
            host_digest()
        )),
        ok(""),
        ok("ActiveState=activating\nInvocationID=\n"),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = harness;
    let run = factory_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    let tight = Instant::now() + Duration::from_millis(150);
    let result = svc.factory_codex_reserve(&run, &lease, PIN, 600, tight);
    assert!(result.is_err());
    let calls = svc.exec.calls();
    let last = calls.last().unwrap();
    assert_eq!(last.1, "/usr/bin/env");
    assert_eq!(last.2[1], "/usr/bin/systemctl");
    assert_eq!(last.2[3], "stop");
    assert_eq!(last.2[4], factory_unit_name(RID).unwrap());
}

#[test]
fn start_flows() {
    let lease = factory_lease();
    let p = factory_codex_binding(&lease).unwrap();
    // Denials before exec.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_start(&Lease::default(), b"{}", b"prompt", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_start(&lease, b"nope", b"prompt", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_start(&lease, b"{}", b"", deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_start(
            &lease,
            b"{}",
            &vec![b'x'; MAX_FACTORY_PROMPT + 1],
            deadline()
        )
        .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Stale incarnation.
    let other = "f".repeat(64);
    let stale_json = inspect_json().replace(CID, &other);
    let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
    assert_eq!(
        svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
            .unwrap_err(),
        ERR_STALE
    );
    // Success stages credential, prompt, marker, then the gate.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_codex_start(&lease, b"{\"t\":1}", b"do work", deadline())
        .unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 5);
    assert_eq!(calls[1].0, b"{\"t\":1}");
    assert_eq!(calls[2].0, b"do work");
    assert!(calls[3].0.is_empty());
    assert!(
        calls[1].2[6].contains(&shell_quote(&p.auth)),
        "{}",
        calls[1].2[6]
    );
    assert!(
        calls[1].2[6].contains("chown 1001:1001 "),
        "{}",
        calls[1].2[6]
    );
    assert!(
        calls[2].2[6].contains(&shell_quote(&p.prompt)),
        "{}",
        calls[2].2[6]
    );
    assert!(
        calls[3].2[6].contains(&shell_quote(&p.marker)),
        "{}",
        calls[3].2[6]
    );
    assert_eq!(calls[4].2[5], start_gate_script(&p));
    // Gate failure.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok(""),
        ok(""),
        ok(""),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
            .unwrap_err(),
        "factory start staging unconfirmed"
    );
    // Stage failure.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_codex_start(&lease, b"{}", b"prompt", deadline())
            .unwrap_err(),
        "factory file staging unconfirmed"
    );
}

#[test]
fn wait_flows() {
    let lease = factory_lease();
    let p = factory_codex_binding(&lease).unwrap();
    // Active then inactive, exit 3, bounded output.
    let svc = make_service(FakeExec::new(vec![
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok("3\n"),
        ok("hello output"),
    ]));
    assert_eq!(
        svc.factory_codex_wait(&lease, deadline()).unwrap(),
        (3, "hello output".to_string())
    );
    let calls = svc.exec.calls();
    assert_eq!(calls[2].2[4], format!("{}/exit", p.run_dir));
    assert_eq!(calls[3].2[4], "-c");
    assert_eq!(calls[3].2[5], "65537");
    // Missing/garbage exit records read -1 without failing.
    let svc = make_service(FakeExec::new(vec![
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_codex_wait(&lease, deadline()).unwrap(),
        (-1, String::new())
    );
    let svc = make_service(FakeExec::new(vec![
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok("not-a-number\n"),
        ok("out"),
    ]));
    assert_eq!(
        svc.factory_codex_wait(&lease, deadline()).unwrap(),
        (-1, "out".to_string())
    );
    let svc = make_service(FakeExec::new(vec![
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok("999\n"),
        ok("out"),
    ]));
    assert_eq!(
        svc.factory_codex_wait(&lease, deadline()).unwrap(),
        (-1, "out".to_string())
    );
    // Observation failure propagates.
    let svc = make_service(FakeExec::new(vec![err("boom")]));
    assert_eq!(
        svc.factory_codex_wait(&lease, deadline()).unwrap_err(),
        "factory unit observation unavailable"
    );
    // Binding failure first.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_wait(&Lease::default(), deadline())
            .unwrap_err(),
        ERR_DENIED
    );
}

// ----- validate/stop/capture/finish -----

#[test]
fn validate_flows() {
    let lease = factory_lease();
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]));
    assert!(svc.factory_codex_validate(&lease, deadline()).is_ok());
    assert_eq!(svc.exec.calls().len(), 4);
    // Role mismatch denies.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("9999\n"),
    ]));
    assert_eq!(
        svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
        ERR_DENIED
    );
    // Inactive unit denies.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok("ActiveState=inactive\nInvocationID=\n"),
    ]));
    assert_eq!(
        svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
        ERR_DENIED
    );
    // Invocation mismatch denies.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(&format!(
            "ActiveState=active\nInvocationID={}\n",
            "b".repeat(32)
        )),
    ]));
    assert_eq!(
        svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
        ERR_DENIED
    );
    // Observation failure denies (not the raw error).
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        err("boom"),
    ]));
    assert_eq!(
        svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
        ERR_DENIED
    );
}

#[test]
fn stop_flows() {
    let lease = factory_lease();
    let p = factory_codex_binding(&lease).unwrap();
    // Full retire: pid, stop, inactive, exists, retire, pid (unchanged).
    let svc = make_service(FakeExec::new(vec![
        ok("4242 99999\n"),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        ok(""),
        ok("4242 99999\n"),
    ]));
    svc.factory_codex_stop(&lease, deadline()).unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 6);
    assert_eq!(calls[4].2[5], factory_retire(&p.run_dir));
    // Removed container retires the unit only.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
    ]));
    svc.factory_codex_stop(&lease, deadline()).unwrap();
    assert_eq!(svc.exec.calls().len(), 4);
    // Uncertain container probe.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 2"),
    ]));
    assert_eq!(
        svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    // Retire failure is uncertain.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    // Changed pid re-runs the retire script once; still-changed is uncertain.
    let svc = make_service(FakeExec::new(vec![
        ok("4242 1\n"),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        ok(""),
        ok("5151 2\n"),
        ok(""),
        ok("5151 2\n"),
    ]));
    svc.factory_codex_stop(&lease, deadline()).unwrap();
    assert_eq!(svc.exec.calls().len(), 8);
    let svc = make_service(FakeExec::new(vec![
        ok("4242 1\n"),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        ok(""),
        ok("5151 2\n"),
        ok(""),
        ok("6161 3\n"),
    ]));
    assert_eq!(
        svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    // A unit that never dies is uncertain (expired deadline, no sleep).
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]));
    let past = Instant::now() - Duration::from_secs(1);
    assert_eq!(
        svc.factory_codex_stop(&lease, past).unwrap_err(),
        ERR_UNCERTAIN
    );
}

#[test]
fn capture_and_finish_flows() {
    let lease = factory_lease();
    let p = factory_codex_binding(&lease).unwrap();
    let svc = make_service(FakeExec::new(vec![ok("{\"maintained\":true}")]));
    assert_eq!(
        svc.factory_codex_capture(&lease, deadline()).unwrap(),
        b"{\"maintained\":true}".to_vec()
    );
    let calls = svc.exec.calls();
    assert_eq!(calls[0].2[3], "/usr/bin/head");
    assert_eq!(calls[0].2[4], "-c");
    assert_eq!(calls[0].2[5], "262145");
    assert_eq!(calls[0].2[6], p.auth);
    let svc = make_service(FakeExec::new(vec![err("exit status 1")]));
    assert_eq!(
        svc.factory_codex_capture(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    let svc = make_service(FakeExec::new(vec![ok("not-json")]));
    assert_eq!(
        svc.factory_codex_capture(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    // Finish stops, then captures.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        ok("{\"m\":1}"),
    ]));
    assert_eq!(
        svc.factory_codex_finish(&lease, deadline()).unwrap(),
        b"{\"m\":1}".to_vec()
    );
    // Finish propagates stop failures without capturing.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(""),
        err("exit status 1"),
    ]));
    assert_eq!(
        svc.factory_codex_finish(&lease, deadline()).unwrap_err(),
        ERR_UNCERTAIN
    );
    assert_eq!(svc.exec.calls().len(), 5);
}

#[test]
fn stop_unbound_flows() {
    let run = factory_run();
    let stopped_json = inspect_json().replace("\"running\":true", "\"running\":false");
    // Stopped containers still retire (requireRunning=false).
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        ok(&stopped_json),
        ok(""),
        ok(""),
        ok(""),
    ]));
    svc.factory_codex_stop_unbound(&run, deadline()).unwrap();
    assert_eq!(svc.exec.calls().len(), 6);
    // Missing container resolves to success.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        err("exit status 1"),
    ]));
    svc.factory_codex_stop_unbound(&run, deadline()).unwrap();
    let calls = svc.exec.calls();
    assert_eq!(
        calls[3].2,
        terminal::container_exists_argv(&format!("soda-{PID}"))
    );
    // Present-but-uninspectable container is uncertain.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        ok(""),
    ]));
    assert_eq!(
        svc.factory_codex_stop_unbound(&run, deadline())
            .unwrap_err(),
        ERR_UNCERTAIN
    );
    // Invalid run never calls out.
    let svc = make_service(FakeExec::new(vec![]));
    assert!(svc
        .factory_codex_stop_unbound(&FactoryRun::default(), deadline())
        .is_err());
    assert!(svc.exec.calls().is_empty());
}

#[test]
fn live_matrix() {
    let lease = factory_lease();
    let binding = lease.binding.clone().unwrap();
    let svc = make_service(FakeExec::new(vec![ok(&format!(
        "ActiveState=active\nInvocationID={IID}\n"
    ))]));
    assert!(svc.factory_codex_live(&binding, deadline()));
    let svc = make_service(FakeExec::new(vec![ok(
        "ActiveState=inactive\nInvocationID=\n",
    )]));
    assert!(!svc.factory_codex_live(&binding, deadline()));
    let svc = make_service(FakeExec::new(vec![err("boom")]));
    assert!(!svc.factory_codex_live(&binding, deadline()));
    // Shape faults never call out.
    let svc = make_service(FakeExec::new(vec![]));
    let mut bad = binding.clone();
    bad.scope = "other".to_string();
    assert!(!svc.factory_codex_live(&bad, deadline()));
    bad = binding;
    bad.invocation_id = "short".to_string();
    assert!(!svc.factory_codex_live(&bad, deadline()));
    assert!(svc.exec.calls().is_empty());
}

// ----- output/export/takeover -----

#[test]
fn output_flows() {
    let lease = factory_lease();
    let binding = lease.binding.clone().unwrap();
    // Cursor validation first.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_output(PID, &binding, -1, 100, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_output(PID, &binding, 0, 0, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_output(PID, &binding, 0, MAX_FACTORY_OUTPUT_READ + 1, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Gap past the recorded size.
    let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("10\n")]));
    let slice = svc
        .factory_codex_output(PID, &binding, 11, 100, deadline())
        .unwrap();
    assert_eq!((slice.total, slice.offset, slice.gap), (10, 10, true));
    assert!(slice.data.is_empty());
    // Zero cursor on an over-window log serves the trailing window.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("300000\n"),
        ok("tail-bytes"),
    ]));
    let slice = svc
        .factory_codex_output(PID, &binding, 0, 100, deadline())
        .unwrap();
    assert_eq!(
        (slice.total, slice.offset, slice.truncated),
        (300000, 300000 - MAX_FACTORY_OUTPUT_WINDOW, true)
    );
    assert_eq!(slice.data, b"tail-bytes");
    let calls = svc.exec.calls();
    assert!(
        calls[2].2[5].contains(&format!(
            "tail -c +{}",
            300000 - MAX_FACTORY_OUTPUT_WINDOW + 1
        )),
        "{}",
        calls[2].2[5]
    );
    // Cursor at end reads nothing further.
    let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok("10\n")]));
    let slice = svc
        .factory_codex_output(PID, &binding, 10, 100, deadline())
        .unwrap();
    assert_eq!(
        (slice.total, slice.offset, slice.truncated),
        (10, 10, false)
    );
    assert_eq!(svc.exec.calls().len(), 2);
    // Read failures return the cursor without data or error.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("10\n"),
        err("exit status 1"),
    ]));
    let slice = svc
        .factory_codex_output(PID, &binding, 0, 100, deadline())
        .unwrap();
    assert_eq!((slice.total, slice.offset), (10, 0));
    assert!(slice.data.is_empty());
    // Overlong reads truncate to the limit.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1000\n"),
        ok(&"x".repeat(200)),
    ]));
    let slice = svc
        .factory_codex_output(PID, &binding, 0, 100, deadline())
        .unwrap();
    assert_eq!(slice.data.len(), 100);
    // Missing log reads empty; stat failure leaves total zero.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        err("exit status 1"),
    ]));
    let slice = svc
        .factory_codex_output(PID, &binding, 0, 100, deadline())
        .unwrap();
    assert_eq!((slice.total, slice.offset), (0, 0));
    // Stale incarnation.
    let other = "f".repeat(64);
    let stale_json = inspect_json().replace(CID, &other);
    let svc = make_service(FakeExec::new(vec![ok(&stale_json)]));
    assert_eq!(
        svc.factory_codex_output(PID, &binding, 0, 100, deadline())
            .unwrap_err(),
        ERR_STALE
    );
}

#[test]
fn factory_identity_operation_matrix() {
    let lease = factory_lease();
    let delivery = Delivery {
        lease: lease.clone(),
        credential: Some(b"{}".to_vec()),
    };
    // Validate clears any credential and attests.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]));
    let out = svc
        .factory_identity_operation("validate", &delivery, deadline())
        .unwrap();
    assert!(out.credential.is_none());
    assert_eq!(out.lease, lease);
    // Stop retires.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
    ]));
    let out = svc
        .factory_identity_operation("stop", &delivery, deadline())
        .unwrap();
    assert!(out.credential.is_none());
    // Finish returns the captured credential.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        ok("{\"m\":1}"),
    ]));
    let out = svc
        .factory_identity_operation("finish", &delivery, deadline())
        .unwrap();
    assert_eq!(out.credential, Some(b"{\"m\":1}".to_vec()));
    // Unknown actions deny without calling out.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_identity_operation("launch", &delivery, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Errors propagate.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("9999\n"),
    ]));
    assert_eq!(
        svc.factory_identity_operation("validate", &delivery, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
}
