//! Oracle vectors for the factory-codex terminal callbacks.
//!
//! The three daemon callbacks live in the library as inherent methods on
//! [`Service`](soda_host::terminal::Service) and are dispatched by
//! `factory_identity_operation` (`tcodex.rs`, ported by the texec lane):
//!
//! * `factory_codex_validate(&self, lease, deadline) -> Result<(), String>`
//! * `factory_codex_stop(&self, lease, deadline) -> Result<(), String>`
//! * `factory_codex_finish(&self, lease, deadline) -> Result<Vec<u8>, String>`
//!
//! This file proves argv/parse parity with the Go sources
//! (`internal/host/terminal/factory_codex.go`, `factory_codex_linux.go`)
//! from outside the crate: every expected command line below is transcribed
//! from the Go call sites, and the `Show`-output vectors pin
//! `parseFactoryUnitShow` semantics (last `ActiveState=`/`InvocationID=`
//! wins, whole-body trim, `active` is an exact match).
//!
//! ## Error taxonomy (exact strings, as the daemon adapter maps them)
//!
//! * `validate`: binding failures, incarnation/role/unit mismatches and unit
//!   observation failures all return `identity authority denied` (`denied`).
//! * `stop`: binding/unit-name failures return `denied`; retirement failures
//!   (unit never dies, container probe fails, retire script fails, pid churn)
//!   return `subscription requires reconnection` (`uncertain`). A missing
//!   container (`podman container exists` exit 1) is `Ok` (idempotent).
//! * `finish`: stop errors propagate unchanged; capture failures (missing
//!   file, invalid bytes) return `uncertain`. Success returns the raw
//!   credential bytes (`identity.CredentialValid`: non-empty, <=256KiB,
//!   exactly one JSON value).
//! * `not found` (`identity execution missing`) is absent by design: these
//!   callbacks bind an exact recorded incarnation instead of looking one up.
//!
//! ## Documented deviations from Go (deliberately NOT pinned here)
//!
//! The no-edit rule for this lane forbids touching `tcodex.rs`; the two
//! divergences below need the owning lane's fix, after which they should
//! gain pinned vectors in this file:
//!
//! * D1: Go `FactoryCodexValidate` maps a `factoryProjectContainer` failure
//!   (inspect error, unparsable output, container not running) to `denied`;
//!   the Rust port propagates the helper's `stale`
//!   (`identity generation changed`).
//! * D2: Go maps a `factoryRoleIDs` failure to `denied`; the Rust port
//!   propagates `factory role is not resolvable` / `invalid role identity`.

use soda_host::project::Executor;
use soda_host::terminal::factory::tcodex::{self, FactoryUnitShow};
use soda_host::terminal::{
    self, Binding, Delivery, Lease, Service, ERR_DENIED, ERR_UNCERTAIN, KIND_FACTORY,
    PROVIDER_CODEX, TERMINAL_INSPECT,
};
use std::sync::Mutex;
use std::time::{Duration, Instant};

// ---------- fixtures ----------

const PID: &str = "p0123456789abcdef01234567";
const RID: &str = "0123456789abcdef0123456789abcdef";
const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const PREP: &str = "f0123456789abcdef01234567";
const ROLE: &str = "soda-coder";
const UNIT: &str = "soda-factory-0123456789abcdef0123456789abcdef.service";
const RUN_DIR: &str = "/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef";

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn ok(body: &str) -> Result<Vec<u8>, String> {
    Ok(body.as_bytes().to_vec())
}

fn err(body: &str) -> Result<Vec<u8>, String> {
    Err(body.to_string())
}

fn inspect_json() -> String {
    format!(
        "{{\"id\":{CID:?},\"running\":true,\"project\":{PID:?},\"owner\":\"7\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
    )
}

fn factory_lease() -> Lease {
    Lease {
        provider_id: PROVIDER_CODEX.to_string(),
        id: "lease-f".to_string(),
        connection_id: "conn".to_string(),
        generation: 5,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: RID.to_string(),
        kind: KIND_FACTORY.to_string(),
        binding: Some(Binding {
            kind: KIND_FACTORY.to_string(),
            id: RID.to_string(),
            project: CID.to_string(),
            login: ROLE.to_string(),
            uid: 1001,
            gid: 1001,
            scope: tcodex::FACTORY_SCOPE_CODEX.to_string(),
            invocation_id: IID.to_string(),
            credential_root: RUN_DIR.to_string(),
            generation: 5,
            child_id: PREP.to_string(),
        }),
        ..Default::default()
    }
}

fn make_service(exec: FakeExec) -> Service<FakeExec> {
    Service {
        exec,
        codex_harness: "/opt/harness".to_string(),
        codex_harness_sha256: String::new(),
        codex_harness_version: String::new(),
        muse_harness: String::new(),
        muse_harness_sha256: String::new(),
        muse_harness_version: String::new(),
    }
}

type RecordedCall = (Vec<u8>, String, Vec<String>);

struct FakeExec {
    calls: Mutex<Vec<RecordedCall>>,
    script: Mutex<Vec<Result<Vec<u8>, String>>>,
}

impl FakeExec {
    fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
        FakeExec {
            calls: Mutex::new(Vec::new()),
            script: Mutex::new(script),
        }
    }

    fn calls(&self) -> Vec<RecordedCall> {
        self.calls.lock().unwrap().clone()
    }
}

impl Executor for FakeExec {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.lock().unwrap().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        let mut script = self.script.lock().unwrap();
        if script.is_empty() {
            return Err("unexpected call".to_string());
        }
        script.remove(0)
    }
}

/// `factoryUserBus` argv head: the euid is runtime state, so only the fixed
/// shape is pinned; every following element is asserted exactly.
fn assert_env_systemctl(call: &RecordedCall, tail: &[&str]) {
    assert!(call.0.is_empty(), "no stdin: {call:?}");
    assert_eq!(call.1, "/usr/bin/env");
    let bus = &call.2[0];
    assert!(
        bus.starts_with("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/") && bus.ends_with("/bus"),
        "user bus env: {bus}"
    );
    assert_eq!(call.2[1], "/usr/bin/systemctl");
    assert_eq!(call.2[2], "--user");
    let rest: Vec<&str> = call.2[3..].iter().map(|s| s.as_str()).collect();
    assert_eq!(rest, tail);
}

fn assert_podman(call: &RecordedCall, argv: &[&str]) {
    assert!(call.0.is_empty(), "no stdin: {call:?}");
    assert_eq!(call.1, "/usr/bin/podman");
    let got: Vec<&str> = call.2.iter().map(|s| s.as_str()).collect();
    assert_eq!(got, argv);
}

// ---------- fixed-path goldens ----------

#[test]
fn fixed_paths_match_go_layout() {
    assert_eq!(tcodex::factory_unit_name(RID).as_deref(), Some(UNIT));
    let (_, run_dir, _, _) = tcodex::factory_run_paths(ROLE, PREP, RID).unwrap();
    assert_eq!(run_dir, RUN_DIR);
    let p = tcodex::factory_codex_binding(&factory_lease()).unwrap();
    assert_eq!(p.pid_file, format!("{RUN_DIR}/supervisor.pid"));
    assert_eq!(
        p.auth,
        format!("{RUN_DIR}/home/.codex/auth.json"),
        "auth path golden"
    );
}

// ---------- validate ----------

#[test]
fn validate_ok_argv_golden() {
    let lease = factory_lease();
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]));
    svc.factory_codex_validate(&lease, deadline()).unwrap();
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 4);
    // Go: s.podman(ctx, nil, "--remote=false", "inspect", "--format",
    // terminalInspect, "soda-"+id)
    assert_podman(
        &calls[0],
        &[
            "--remote=false",
            "inspect",
            "--format",
            TERMINAL_INSPECT,
            &format!("soda-{PID}"),
        ],
    );
    // Go: s.podman(ctx, nil, "--remote=false", "exec", container,
    // "/usr/bin/id", "-u"/"-g", role)
    assert_podman(
        &calls[1],
        &["--remote=false", "exec", CID, "/usr/bin/id", "-u", ROLE],
    );
    assert_podman(
        &calls[2],
        &["--remote=false", "exec", CID, "/usr/bin/id", "-g", ROLE],
    );
    // Go: s.factorySystemctl(ctx, "show",
    // "--property=ActiveState,InvocationID", unit)
    assert_env_systemctl(
        &calls[3],
        &["show", "--property=ActiveState,InvocationID", UNIT],
    );
}

#[test]
fn validate_denies_each_gate() {
    let lease = factory_lease();
    // Every binding defect denies before any exec call.
    let binding_cases: Vec<Lease> = vec![
        Lease::default(),
        Lease {
            binding: None,
            ..lease.clone()
        },
        Lease {
            provider_id: terminal::PROVIDER_MUSE.to_string(),
            ..lease.clone()
        },
        Lease {
            kind: terminal::KIND_TERMINAL.to_string(),
            ..lease.clone()
        },
        Lease {
            generation: 6,
            ..lease.clone()
        },
    ];
    for bad in &binding_cases {
        let svc = make_service(FakeExec::new(vec![]));
        assert_eq!(
            svc.factory_codex_validate(bad, deadline()).unwrap_err(),
            ERR_DENIED
        );
        assert!(svc.exec.calls().is_empty());
    }
    // A malformed execution id fails unit-name derivation (after the
    // container/role gates): denied, matching Go's factoryUnitName error.
    let bad_exec = Lease {
        execution_id: "short".to_string(),
        ..lease.clone()
    };
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
    ]));
    assert_eq!(
        svc.factory_codex_validate(&bad_exec, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(svc.exec.calls().len(), 3);
    let mut bad_binding = lease.clone();
    bad_binding.binding.as_mut().unwrap().login = "root".to_string();
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_validate(&bad_binding, deadline())
            .unwrap_err(),
        ERR_DENIED
    );

    // Incarnation mismatch denies (Go compares against l.Binding.Project).
    let svc = make_service(FakeExec::new(vec![ok(
        &inspect_json().replace(CID, &"f".repeat(64))
    )]));
    assert_eq!(
        svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
        ERR_DENIED
    );
    // Role identity mismatch denies.
    for (uid, gid) in [("1001\n", "9999\n"), ("9999\n", "1001\n")] {
        let svc = make_service(FakeExec::new(vec![ok(&inspect_json()), ok(uid), ok(gid)]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
    }
    // Unit gates deny: inactive, invocation mismatch, observation failure.
    let shows = [
        "ActiveState=inactive\nInvocationID=\n".to_string(),
        format!("ActiveState=active\nInvocationID={}\n", "b".repeat(32)),
        "ActiveState=activating\nInvocationID=\n".to_string(),
    ];
    for show in &shows {
        let svc = make_service(FakeExec::new(vec![
            ok(&inspect_json()),
            ok("1001\n"),
            ok("1001\n"),
            ok(show),
        ]));
        assert_eq!(
            svc.factory_codex_validate(&lease, deadline()).unwrap_err(),
            ERR_DENIED
        );
    }
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

// ---------- stop ----------

#[test]
fn stop_ok_argv_golden() {
    let lease = factory_lease();
    let p = tcodex::factory_codex_binding(&lease).unwrap();
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
    let pid_argv = [
        "--remote=false",
        "exec",
        CID,
        "/usr/bin/cat",
        &format!("{RUN_DIR}/supervisor.pid"),
    ];
    // Go factoryReadPID, before the stop so a racing supervisor is seen.
    assert_podman(&calls[0], &pid_argv);
    // Go: s.factorySystemctl(ctx, "stop", unit); errors ignored.
    assert_env_systemctl(&calls[1], &["stop", UNIT]);
    // Go: factoryUnitState poll until inactive.
    assert_env_systemctl(
        &calls[2],
        &["show", "--property=ActiveState,InvocationID", UNIT],
    );
    // Go: s.podman(ctx, nil, "--remote=false", "container", "exists", c).
    assert_podman(&calls[3], &["--remote=false", "container", "exists", CID]);
    // Go: s.podman(ctx, nil, "--remote=false", "exec", c, "/usr/bin/sh",
    // "-c", factoryRetire(p)).
    assert_eq!(calls[4].1, "/usr/bin/podman");
    let retire: Vec<&str> = calls[4].2.iter().map(|s| s.as_str()).collect();
    assert_eq!(
        retire,
        [
            "--remote=false",
            "exec",
            CID,
            "/usr/bin/sh",
            "-c",
            &tcodex::factory_retire(&p),
        ]
    );
    assert_podman(&calls[5], &pid_argv);
}

#[test]
fn stop_is_idempotent_without_container() {
    let lease = factory_lease();
    // `podman container exists` exit 1: unit retired, no group to kill.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
    ]));
    svc.factory_codex_stop(&lease, deadline()).unwrap();
    assert_eq!(svc.exec.calls().len(), 4);
    // A failed `systemctl stop` is ignored: the poll still gates retirement.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        err("exit status 1"),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
    ]));
    svc.factory_codex_stop(&lease, deadline()).unwrap();
}

#[test]
fn stop_uncertain_matrix() {
    let lease = factory_lease();
    // Ambiguous container probe.
    for probe in ["exit status 2", "boom"] {
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err(probe),
        ]));
        assert_eq!(
            svc.factory_codex_stop(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
    }
    // Retire script failure.
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
    // Changed pid re-runs retire once; a second change is uncertain.
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
    assert_eq!(svc.exec.calls().len(), 8);
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
    // Binding failure first: denied, no calls.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_stop(&Lease::default(), deadline())
            .unwrap_err(),
        ERR_DENIED
    );
}

// ---------- finish ----------

#[test]
fn finish_ok_returns_credential_bytes() {
    let lease = factory_lease();
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
        ok("{\"maintained\":true}"),
    ]));
    assert_eq!(
        svc.factory_codex_finish(&lease, deadline()).unwrap(),
        b"{\"maintained\":true}".to_vec()
    );
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 5);
    // Go FactoryCodexCapture: head -c 262145 over the auth file.
    assert_podman(
        &calls[4],
        &[
            "--remote=false",
            "exec",
            CID,
            "/usr/bin/head",
            "-c",
            "262145",
            &format!("{RUN_DIR}/home/.codex/auth.json"),
        ],
    );
}

#[test]
fn finish_runs_capture_only_after_confirmed_stop() {
    let lease = factory_lease();
    // Stop failure propagates; no capture call follows.
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
    // Missing or invalid credential bytes are uncertain, never invented.
    for body in [err("exit status 1"), ok("not-json"), ok("")] {
        let svc = make_service(FakeExec::new(vec![
            ok(""),
            ok(""),
            ok("ActiveState=inactive\nInvocationID=\n"),
            err("exit status 1"),
            body,
        ]));
        assert_eq!(
            svc.factory_codex_finish(&lease, deadline()).unwrap_err(),
            ERR_UNCERTAIN
        );
    }
}

// ---------- Show-output parser oracle ----------

#[test]
fn unit_show_parse_oracle() {
    // Exact ActiveState match; InvocationID trimmed.
    assert_eq!(
        tcodex::parse_factory_unit_show(b"ActiveState=active\nInvocationID=abc\n"),
        FactoryUnitShow {
            active: true,
            invocation: "abc".to_string(),
        }
    );
    assert_eq!(
        tcodex::parse_factory_unit_show(b"ActiveState=inactive\nInvocationID=\n"),
        FactoryUnitShow::default()
    );
    // Last occurrence wins for both keys (Go loops without break).
    assert!(tcodex::parse_factory_unit_show(b"ActiveState=inactive\nActiveState=active\n").active);
    assert!(!tcodex::parse_factory_unit_show(b"ActiveState=active\nActiveState=inactive\n").active);
    assert_eq!(
        tcodex::parse_factory_unit_show(b"InvocationID=one\nInvocationID=two\n").invocation,
        "two"
    );
    assert_eq!(
        tcodex::parse_factory_unit_show(b"InvocationID=  spaced  \n").invocation,
        "spaced"
    );
    // Whole-body trim only: surrounding blank lines are fine.
    assert_eq!(
        tcodex::parse_factory_unit_show(b"\nActiveState=active\nInvocationID=x\n\n"),
        FactoryUnitShow {
            active: true,
            invocation: "x".to_string(),
        }
    );
    // Anything but exactly "active" is inactive.
    assert!(!tcodex::parse_factory_unit_show(b"ActiveState=activating\n").active);
    assert!(!tcodex::parse_factory_unit_show(b"ActiveState=\n").active);
    // Empty, blank, and garbage bodies parse to the zero value.
    assert_eq!(
        tcodex::parse_factory_unit_show(b""),
        FactoryUnitShow::default()
    );
    assert_eq!(
        tcodex::parse_factory_unit_show(b"  \n\t\n"),
        FactoryUnitShow::default()
    );
    assert_eq!(
        tcodex::parse_factory_unit_show(b"MainPID=123\nLoadState=loaded\n"),
        FactoryUnitShow::default()
    );
}

// ---------- daemon adapter dispatch ----------

#[test]
fn adapter_dispatch_calls_each_callback() {
    let lease = factory_lease();
    let delivery = Delivery {
        lease: lease.clone(),
        credential: Some(b"stale-bytes".to_vec()),
    };
    // validate clears any carried credential.
    let svc = make_service(FakeExec::new(vec![
        ok(&inspect_json()),
        ok("1001\n"),
        ok("1001\n"),
        ok(&format!("ActiveState=active\nInvocationID={IID}\n")),
    ]));
    let out = svc
        .factory_identity_operation("validate", &delivery, deadline())
        .unwrap();
    assert_eq!(out.credential, None);
    // stop clears any carried credential.
    let svc = make_service(FakeExec::new(vec![
        ok(""),
        ok(""),
        ok("ActiveState=inactive\nInvocationID=\n"),
        err("exit status 1"),
    ]));
    let out = svc
        .factory_identity_operation("stop", &delivery, deadline())
        .unwrap();
    assert_eq!(out.credential, None);
    // finish delivers fresh credential bytes.
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
    // Unknown actions deny.
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_identity_operation("launch", &delivery, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
}
