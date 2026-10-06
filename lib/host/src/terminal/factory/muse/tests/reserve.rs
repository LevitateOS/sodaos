use super::super::commands::muse_exec_argv;
use super::common::{
    deadline, err, euid, inspect_json, make_service, muse_run, ok, reserve_harness, test_tmp,
    FakeExec, CID, COMMIT, IID, PIN, PREP, RID, ROLE,
};
use crate::terminal::factory::native::{factory_unit_name, factory_user_bus, reserve_run_argv};
use crate::terminal::factory::run::{FactoryRun, FACTORY_HARNESS_CODEX, FACTORY_SCOPE_MUSE};
use crate::terminal::ERR_DENIED;
use crate::terminal::{Lease, Service, KIND_FACTORY};

#[test]
fn verify_harness_matrix() {
    let (dir, harness) = reserve_harness();
    let svc = Service {
        exec: FakeExec::new(vec![]),
        codex_harness: String::new(),
        codex_harness_sha256: String::new(),
        codex_harness_version: String::new(),
        muse_harness: harness.clone(),
        muse_harness_sha256: PIN.to_string(),
        muse_harness_version: "1.4.2".to_string(),
    };
    svc.verify_muse_harness().unwrap();
    // Digest mismatch.
    let bad = Service {
        muse_harness_sha256: "f".repeat(64),
        ..Service {
            exec: FakeExec::new(vec![]),
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: harness.clone(),
            muse_harness_sha256: PIN.to_string(),
            muse_harness_version: "1.4.2".to_string(),
        }
    };
    assert_eq!(
        bad.verify_muse_harness().unwrap_err(),
        "muse harness digest differs"
    );
    // Missing binary.
    let missing = test_tmp("missing");
    let svc = make_service(FakeExec::new(vec![]));
    let mut svc = svc;
    svc.muse_harness = missing.to_str().unwrap().to_string();
    assert_eq!(
        svc.verify_muse_harness().unwrap_err(),
        "verified Muse executable required"
    );
    let _ = dir;
}

#[test]
fn reserve_denial_pins() {
    let (_dir, harness) = reserve_harness();
    let run = muse_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    // Each denial fires before any exec call.
    let bad_run = FactoryRun {
        id: "short".to_string(),
        ..muse_run()
    };
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_muse_reserve(&bad_run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "invalid factory run identity"
    );
    // Wrong family takes the codex path, never muse.
    let codex = FactoryRun {
        harness: FACTORY_HARNESS_CODEX.to_string(),
        harness_vers: "0.153.4".to_string(),
        ..muse_run()
    };
    assert_eq!(
        svc.factory_muse_reserve(&codex, &lease, PIN, 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_reserve(&run, &lease, "f".repeat(64).as_str(), 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_muse_reserve(&run, &lease, PIN, 59, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let mut svc = make_service(FakeExec::new(vec![]));
    svc.muse_harness = harness;
    assert_eq!(
        svc.factory_muse_reserve(&run, &lease, PIN, 10801, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
}

#[test]
fn reserve_success_argv_sequence() {
    let (_dir, harness) = reserve_harness();
    let guest = "/usr/local/bin/muse-factory-1.4.2";
    let unit = factory_unit_name(RID).unwrap();
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),                                      // project container
        ok("1001\n"),                                             // id -u
        ok("1002\n"),                                             // id -g
        ok(""),                                                   // setup script
        ok(&format!("{COMMIT}\n")),                               // git rev-parse
        ok(&format!("{PIN}  {guest}\n")),                         // guest sha256sum (present)
        ok(""),                                                   // systemd-run
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")), // attestation
    ]);
    let mut svc = make_service(exec);
    svc.muse_harness = harness;
    let run = muse_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    let (binding, p) = svc
        .factory_muse_reserve(&run, &lease, PIN, 600, deadline())
        .unwrap();
    assert_eq!(binding.id, RID);
    assert_eq!(binding.project, CID);
    assert_eq!(binding.login, ROLE);
    assert_eq!((binding.uid, binding.gid), (1001, 1002));
    assert_eq!(binding.scope, FACTORY_SCOPE_MUSE);
    assert_eq!(binding.invocation_id, IID);
    assert_eq!(binding.credential_root, p.run_dir);
    assert_eq!(binding.generation, 5);
    assert_eq!(binding.child_id, PREP);
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 8);
    // systemd-run argv is byte-exact: header + podman payload.
    let bus = factory_user_bus(euid());
    let mut want_run = vec![
        "/usr/bin/env".to_string(),
        bus,
        "/usr/bin/systemd-run".to_string(),
        "--user".to_string(),
    ];
    want_run.extend(reserve_run_argv(&unit, 600));
    want_run.extend(muse_exec_argv(CID, &run, &p, guest));
    let got_run: Vec<String> = std::iter::once(calls[6].1.clone())
        .chain(calls[6].2.clone())
        .collect();
    assert_eq!(got_run, want_run);
    // Muse env: no CODEX_HOME, update check off, file backend pinned.
    let payload = &calls[6].2;
    assert!(!payload.iter().any(|a| a.contains("CODEX_HOME")));
    assert!(payload.iter().any(|a| a == "MUSE_NO_AUTO_UPDATE=1"));
    assert!(payload.iter().any(|a| a == &format!("HOME={}", p.home)));
    assert!(payload
        .iter()
        .any(|a| a == &format!("XDG_CONFIG_HOME={}/.config", p.home)));
    assert!(payload.iter().any(|a| a == "TBH_CREDENTIAL_BACKEND=file"));
    assert!(!payload.iter().any(|a| a.starts_with("META_API_KEY")));
}

#[test]
fn reserve_stage_missing_guest() {
    let (_dir, harness) = reserve_harness();
    let guest = "/usr/local/bin/muse-factory-1.4.2";
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1002\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        err("exit status 1"),             // guest sha256sum (absent)
        ok(""),                           // podman cp
        ok(&format!("{PIN}  {guest}\n")), // install digest check
        ok(""),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]);
    let mut svc = make_service(exec);
    svc.muse_harness = harness.clone();
    let run = muse_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    let (binding, _) = svc
        .factory_muse_reserve(&run, &lease, PIN, 600, deadline())
        .unwrap();
    assert_eq!(binding.scope, FACTORY_SCOPE_MUSE);
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 10);
    assert_eq!(calls[6].2[1], "cp");
    assert!(calls[6].2[2].ends_with("/bin/muse"));
    // Digest mismatch refuses.
    let exec = FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1002\n"),
        ok(""),
        ok(&format!("{COMMIT}\n")),
        err("exit status 1"),
        ok(""),
        ok(&format!("{}  {guest}\n", "f".repeat(64))),
    ]);
    let mut svc = make_service(exec);
    svc.muse_harness = harness;
    assert_eq!(
        svc.factory_muse_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "guest harness digest differs"
    );
}
