// Oracle + unit tests for the pops JSON adapter (`src/pops.rs`).
//
// `pops` is not wired into `lib.rs` yet (the integrator owns that), so this
// harness includes it directly, the same arrangement as `gmux_smoke.rs`. The
// test root re-exports the crate modules `pops` refers to as `crate::...`.
// Unit tests drive every op through a scripted fake `Executor` (no live
// containers); oracle tests assert byte-for-byte parity with Go.
// Golden vectors were captured from this branch with `go run ./popsgolden`
// (`json.Marshal` of the domain structs plus stdin-body maps and strictjson
// errors; helper removed after capture):
use soda_host::{account, domain, json, preparation, project};

#[path = "../src/pops.rs"]
mod pops;

use pops::{
    AccessKeysReq, AccountReq, HoldPreparationReq, InspectPreparationReq, Ops, PrepareCandidateReq,
    PrepareReq, StopPreparationReq,
};
use std::cell::RefCell;
use std::collections::VecDeque;
use std::time::{Duration, Instant};

// ---------- Go-captured goldens ----------

const PID: &str = "p0123456789abcdef01234567";
const FID: &str = "f0123456789abcdef01234567";
const REV: &str = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789";
const COMMIT: &str = "abcdef0123456789abcdef0123456789abcdef01";
const ED: &str = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";

const GO_ACCESS_KEY_STATE: &str = "{\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\"]}";
const GO_ACCESS_KEY_STATE_EMPTY: &str = "{\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"keys\":[]}";
const GO_HOLD_TRUE: &str = "{\"active\":true,\"revision\":2}";
const GO_HOLD_FALSE: &str = "{\"active\":false,\"revision\":0}";
const GO_PREPARE_FULL: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"running\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"tools\":[{\"name\":\"python3\",\"path\":\"/usr/bin/python3\",\"version\":\"9.9-test\"}],\"setup_exit\":0,\"check_exit\":1,\"output\":\"--- setup ---\\nok\\n--- check ---\\nfail\",\"ready\":false,\"stopped\":false}";
const GO_PREPARE_READY: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"ready\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"ready\":true,\"stopped\":false}";
const GO_PREPARE_STOPPED: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"stopped\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"ready\":false,\"stopped\":true,\"retirement\":\"confirmed\"}";
const GO_PREPARE_MISSING: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"waiting\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"missing\":\"node\",\"ready\":false,\"stopped\":false}";
// Stdin bodies the ops post to podman (Go map marshal => sorted keys).
const GO_KEYS_BODY_APPLY: &str = "{\"apply\":true,\"identity\":7,\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\"],\"login\":\"alice\",\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\"}";
const GO_ACCOUNT_BODY: &str = "{\"admin\":true,\"identity\":7,\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\\n\"],\"login\":\"alice\"}";
const GO_HELPER_INSPECT: &str = "{\"id\":\"f0123456789abcdef01234567\",\"op\":\"inspect\"}";
// strictjson failures, captured verbatim.
const GO_UNKNOWN_FIELD: &str = "decode request: json: unknown field \"bogus\"";
const GO_NESTED_TYPE_ERROR: &str = "decode request: json: cannot unmarshal number into Go struct field Preparation.preparation.id of type string";
const GO_BAD_BASE64: &str = "decode request: illegal base64 data at input byte 0";

// ---------- fake executor ----------

type MockCall = (Vec<u8>, String, Vec<String>);

struct Mock {
    calls: RefCell<Vec<MockCall>>,
    script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
}

impl Mock {
    fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
        Mock {
            calls: RefCell::new(Vec::new()),
            script: RefCell::new(responses.into()),
        }
    }
}

impl project::Executor for Mock {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.borrow_mut().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        self.script
            .borrow_mut()
            .pop_front()
            .unwrap_or(Err("no scripted response".to_string()))
    }
}

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn test_config() -> project::Config {
    project::Config {
        muse_socket: String::new(),
        image: "img".to_string(),
        network: "sodanet".to_string(),
        subnet: "10.0.0.0/24".to_string(),
        bridge: "sodabr".to_string(),
    }
}

fn ops(mock: &Mock) -> Ops<&Mock> {
    Ops {
        exec: mock,
        config: test_config(),
    }
}

// ---------- fixtures ----------

/// `--format` inspection for the container-binding paths (container = REV so
/// state renders match the goldens byte for byte).
fn format_inspect(running: bool) -> Vec<u8> {
    format!(
        "{{\"id\":{REV:?},\"running\":{running},\"project\":{PID:?},\"owner\":\"3\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:524288:262144\"],\"GidMap\":[\"0:524288:262144\"]}}}}"
    )
    .into_bytes()
}

/// Full `podman inspect` for the `inspect` path (account op).
fn container_inspect(owner: &str, running: bool, ip: &str) -> Vec<u8> {
    format!(
        "[{{\"Image\":\"\",\"Config\":{{\"Labels\":{{\"org.soda.project\":{PID:?},\"org.soda.owner\":{owner:?}}}}},\"State\":{{\"Running\":{running}}},\"NetworkSettings\":{{\"Networks\":{{\"sodanet\":{{\"IPAddress\":{ip:?}}}}}}}}}]"
    )
    .into_bytes()
}

/// Factory-helper `inspect` observation, mirroring Go's `execHelperState`.
#[allow(clippy::too_many_arguments)]
fn helper_state(
    role: &str,
    phase: &str,
    digest: &str,
    tools_json: &str,
    missing: &str,
    stopped: bool,
    ready: bool,
    exits: Option<(i64, i64)>,
    setup_log: &str,
    check_log: &str,
) -> Vec<u8> {
    let mut out = format!(
        "{{\"hold\":{{\"active\":false,\"revision\":-1}},\"known\":true,\"phase\":{phase:?},\"role\":{role:?},\"setup_digest\":{digest:?},\"source_commit\":{COMMIT:?},\"tools\":[{tools_json}],\"verified\":{{\"uid\":\"1001\",\"login\":{role:?},\"groups\":{role:?}}},\"missing\":{missing:?},\"stopped\":{stopped},\"ready\":{ready}"
    );
    if let Some((setup_exit, check_exit)) = exits {
        out.push_str(&format!(
            ",\"setup_exit\":{setup_exit},\"check_exit\":{check_exit}"
        ));
    }
    out.push_str(&format!(
        ",\"setup_log\":{setup_log:?},\"check_log\":{check_log:?}}}"
    ));
    out.into_bytes()
}

fn approve_response(fid: &str, role: &str) -> Vec<u8> {
    format!(
        "{{\"approved\":{fid:?},\"repeated\":false,\"checkout\":\"/home/{role}/checkouts/{fid}\",\"credential_file\":\"\"}}"
    )
    .into_bytes()
}

/// Files all prepare fixtures share: setup.sh + check.sh holding `true\n`.
fn fixture_files() -> std::collections::HashMap<String, Vec<u8>> {
    std::collections::HashMap::from([
        ("setup.sh".to_string(), b"true\n".to_vec()),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ])
}

fn fixture_digest() -> String {
    preparation::setup_digest_of(&fixture_files())
}

/// Valid `/prepare` request body (base64 `dHJ1ZQo=` = `true\n`,
/// `YnVuZGxlLWJ5dGVz` = `bundle-bytes`).
fn prepare_body(fid: &str, role: &str, digest: &str) -> Vec<u8> {
    format!(
        "{{\"preparation\":{{\"id\":{fid:?},\"project\":{PID:?},\"role\":{role:?},\"revision\":1,\
        \"requirements\":{{\"id\":\"d0123456789abcdef01234567\",\"revision\":1,\"approver\":7,\"source_commit\":{COMMIT:?},\"digest\":{REV:?}}},\
        \"approval\":{{\"id\":\"d123456789abcdef012345678\",\"revision\":1,\"approver\":9,\"effects_digest\":{REV:?}}},\
        \"source_commit\":{COMMIT:?},\"setup_digest\":{digest:?},\"tools\":[\"python3\"],\"credential\":\"\"}},\
        \"setup\":{{\"files\":{{\"setup.sh\":\"dHJ1ZQo=\",\"check.sh\":\"dHJ1ZQo=\"}},\"bundle\":\"YnVuZGxlLWJ5dGVz\"}}}}"
    )
    .into_bytes()
}

// ---------- strict request decode ----------

#[test]
fn access_keys_req_strict_shape() {
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(req.0.project, PID);
    assert_eq!(req.0.login, "alice");
    assert_eq!(req.0.identity, 7);
    assert_eq!(req.0.revision, REV);
    assert_eq!(req.0.keys, vec![ED.to_string()]);
    assert!(req.0.apply);
    // Missing revision/keys/apply decode as empty (observe shape).
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert!(req.0.revision.is_empty() && req.0.keys.is_empty() && !req.0.apply);
    // Explicit nulls behave like missing fields, as in encoding/json.
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":null}}")
            .as_bytes(),
    )
    .unwrap();
    assert!(req.0.keys.is_empty());
    // Unknown fields rejected with the exact Go strictjson text.
    let err = AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1,"bogus":true}"#)
        .unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD);
    // Non-integer identity rejected.
    assert!(AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1.5}"#).is_err());
    // Duplicates rejected at the strict layer.
    assert!(AccessKeysReq::decode(br#"{"project":"p","project":"q"}"#).is_err());
}

#[test]
fn account_req_strict_shape() {
    let raw =
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[{ED:?}]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!((&req.0.login, req.0.identity), (&"alice".to_string(), 7));
    assert_eq!(req.0.keys, vec![ED.to_string()]);
    // Missing keys decodes as empty.
    let req = AccountReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert!(req.0.keys.is_empty());
    let err = AccountReq::decode(br#"{"project":"p","login":"a","identity":1,"admin":true}"#)
        .unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD.replace("bogus", "admin"));
}

#[test]
fn prepare_req_strict_shape() {
    let digest = fixture_digest();
    let req = PrepareReq::decode(&prepare_body(FID, "soda-coder", &digest)).unwrap();
    assert_eq!(req.0.preparation.id, FID);
    assert_eq!(req.0.preparation.role, "soda-coder");
    assert_eq!(req.0.setup.files.len(), 2);
    assert_eq!(req.0.setup.files["setup.sh"], b"true\n");
    assert_eq!(req.0.setup.bundle, b"bundle-bytes");
    req.0.validate().unwrap();
    // Unknown nested field rejected.
    let bad = prepare_body(FID, "soda-coder", &digest);
    let bad = String::from_utf8(bad).unwrap();
    let bad = bad.replacen(
        "\"role\":\"soda-coder\"",
        "\"role\":\"soda-coder\",\"bogus\":1",
        1,
    );
    let err = PrepareReq::decode(bad.as_bytes()).unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD);
    // Nested type error names the innermost struct, exactly like Go.
    let err = PrepareReq::decode(br#"{"preparation":{"id":1}}"#).unwrap_err();
    assert_eq!(err, GO_NESTED_TYPE_ERROR);
    // Bad base64 in a []byte field reports the Go offset text.
    let err =
        PrepareReq::decode(br#"{"preparation":{},"setup":{"files":{},"bundle":"!!!not-base64"}}"#)
            .unwrap_err();
    assert_eq!(err, GO_BAD_BASE64);
}

#[test]
fn prepare_candidate_req_strict_shape() {
    let digest = fixture_digest();
    let raw = String::from_utf8(candidate_body(FID, "f123456789abcdef012345678", &digest)).unwrap();
    let req = PrepareCandidateReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(req.0.source_preparation, "f123456789abcdef012345678");
    assert_eq!(req.0.bundle, b"candidate-bundle");
    assert_eq!(req.0.preparation.role, "soda-reviewer");
    req.0.validate().unwrap();
    // Unknown fields rejected, including inside the nested preparation.
    let bad = raw.replacen("\"bundle\":", "\"bogus\":1,\"bundle\":", 1);
    assert_eq!(
        PrepareCandidateReq::decode(bad.as_bytes()).unwrap_err(),
        GO_UNKNOWN_FIELD
    );
}

#[test]
fn inspect_stop_hold_req_shapes() {
    let addr = format!("{{\"project\":{PID:?},\"id\":{FID:?}}}");
    let ins = InspectPreparationReq::decode(addr.as_bytes()).unwrap();
    assert_eq!(
        (&ins.0.project, &ins.0.id),
        (&PID.to_string(), &FID.to_string())
    );
    ins.0.validate().unwrap();
    let stop = StopPreparationReq::decode(addr.as_bytes()).unwrap();
    stop.0.validate().unwrap();
    let hold = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    assert!(hold.0.hold && hold.0.revision == 2);
    hold.0.validate().unwrap();
    assert_eq!(
        InspectPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"id\":{FID:?},\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        StopPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"id\":{FID:?},\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        HoldPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2,\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
}

#[test]
fn oversize_body_rejected_before_shape() {
    let big = vec![b'x'; (1 << 20) + 1];
    assert_eq!(
        json::decode_strict(&big).unwrap_err().0,
        "request exceeds 1 MiB"
    );
    assert_eq!(
        AccessKeysReq::decode(&big).unwrap_err(),
        "request exceeds 1 MiB"
    );
}

#[test]
fn ops_constructor_mirrors_runtime_fields() {
    let mock = Mock::new(vec![]);
    let o = ops(&mock);
    assert_eq!(o.config.network, "sodanet");
    assert_eq!(o.config.subnet, "10.0.0.0/24");
}

// ---------- op unit tests (scripted executor) ----------

fn binding_argv() -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        project::PROJECT_INSPECT_FORMAT.to_string(),
        format!("soda-{PID}"),
    ]
}

fn helper_argv() -> Vec<String> {
    vec![
        "exec".to_string(),
        "--interactive".to_string(),
        format!("soda-{PID}"),
        preparation::FACTORY_HELPER.to_string(),
    ]
}

/// `{"op":...}` values posted to the factory helper, in call order.
fn helper_ops(mock: &Mock) -> Vec<String> {
    let helper = helper_argv();
    mock.calls
        .borrow()
        .iter()
        .filter(|(_, cmd, args)| cmd == "/usr/bin/podman" && *args == helper)
        .map(|(stdin, _, _)| {
            let text = String::from_utf8_lossy(stdin);
            let start = text.find("\"op\":\"").unwrap() + "\"op\":\"".len();
            text[start..].split('"').next().unwrap().to_string()
        })
        .collect()
}

#[test]
fn access_keys_observe_matches_golden() {
    let observed = format!("{{\"revision\":{REV:?},\"keys\":[{ED:?}]}}").into_bytes();
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(observed)]);
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).access_keys(&req, deadline()).unwrap();
    assert_eq!(out, GO_ACCESS_KEY_STATE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].1, "/usr/bin/podman");
    assert_eq!(calls[0].2, binding_argv());
    assert_eq!(
        calls[1].2,
        vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            "--interactive".to_string(),
            REV.to_string(),
            account::AGENT_PROGRAM.to_string(),
            "keys".to_string(),
        ]
    );
    // Sorted map keys, exactly like the Go request body.
    assert_eq!(
        String::from_utf8_lossy(&calls[1].0),
        "{\"apply\":false,\"identity\":7,\"keys\":[],\"login\":\"alice\",\"revision\":\"\"}"
    );
}

#[test]
fn access_keys_apply_rechecks_revision() {
    let observed = format!("{{\"revision\":{REV:?},\"keys\":[{ED:?}]}}").into_bytes();
    // Apply binds the container before the preview re-observes it.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(format_inspect(true)),
        Ok(observed.clone()),
        Ok(observed),
    ]);
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    let out = ops(&mock).access_keys(&req, deadline()).unwrap();
    assert_eq!(out, GO_ACCESS_KEY_STATE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 4);
    assert_eq!(String::from_utf8_lossy(&calls[3].0), GO_KEYS_BODY_APPLY);
}

#[test]
fn access_keys_rejects_revision_drift() {
    let drifted = format!("{{\"revision\":{:?},\"keys\":[{ED:?}]}}", "0".repeat(64)).into_bytes();
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(drifted)]);
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "native keys changed or are not managed canonical keys"
    );
    // The apply exec never runs after a drifted preview.
    assert_eq!(mock.calls.borrow().len(), 2);
}

#[test]
fn access_keys_validates_before_exec() {
    // Trailing comment: noncanonical (apply shape passes request validation first).
    let commented = format!("{ED} alice@host");
    let raw = format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{commented:?}],\"apply\":true}}");
    let mock = Mock::new(vec![]);
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "noncanonical or duplicate development key"
    );
    // 33 keys exceed the bound (apply shape passes request validation first).
    let many = vec![format!("{ED:?}"); 33].join(",");
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{many}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "too many development keys"
    );
    assert!(mock.calls.borrow().is_empty());
}

#[test]
fn account_provisions_login() {
    let mock = Mock::new(vec![
        Ok(container_inspect("7", true, "10.0.0.5")),
        Ok(b"{\"login\":\"alice\",\"identity\":7}".to_vec()),
    ]);
    let raw =
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[{ED:?}]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    ops(&mock).account(&req, deadline()).unwrap();
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(
        calls[1].2,
        vec![
            "exec".to_string(),
            "--interactive".to_string(),
            format!("soda-{PID}"),
            "/usr/libexec/soda/project-account".to_string(),
        ]
    );
    // Identity equals owner, so admin:true; keys carry MarshalAuthorizedKey newlines.
    assert_eq!(String::from_utf8_lossy(&calls[1].0), GO_ACCOUNT_BODY);
}

#[test]
fn account_refuses_stopped_and_root() {
    let mock = Mock::new(vec![Ok(container_inspect("7", false, ""))]);
    let raw = format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).account(&req, deadline()).unwrap_err(),
        "project is stopped"
    );
    assert_eq!(mock.calls.borrow().len(), 1);
    let mock = Mock::new(vec![]);
    let raw = format!("{{\"project\":{PID:?},\"login\":\"root\",\"identity\":7,\"keys\":[]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).account(&req, deadline()).unwrap_err(),
        "invalid project account"
    );
    assert!(mock.calls.borrow().is_empty());
}

#[test]
fn prepare_ready_short_circuit_matches_golden() {
    let digest = fixture_digest();
    let ready = helper_state(
        "soda-coder",
        "ready",
        REV,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{}".to_vec()),
        Ok(approve_response(FID, "soda-coder")),
        Ok(ready),
    ]);
    let req = PrepareReq::decode(&prepare_body(FID, "soda-coder", &digest)).unwrap();
    let out = ops(&mock).prepare(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_READY);
    // ensure, approve, inspect: no clone, no start for an already-ready state.
    assert_eq!(helper_ops(&mock), vec!["ensure", "approve", "inspect"]);
    assert_eq!(mock.calls.borrow().len(), 4);
}

#[test]
fn prepare_validates_before_exec() {
    let digest = fixture_digest();
    let mock = Mock::new(vec![]);
    let bad = prepare_body("../escape", "soda-coder", &digest);
    let req = PrepareReq::decode(&bad).unwrap();
    assert_eq!(
        ops(&mock).prepare(&req, deadline()).unwrap_err(),
        "invalid preparation identity"
    );
    let bad = prepare_body(FID, "soda-coder", REV);
    let req = PrepareReq::decode(&bad).unwrap();
    assert_eq!(
        ops(&mock).prepare(&req, deadline()).unwrap_err(),
        "approved inputs do not match their digest"
    );
    assert!(mock.calls.borrow().is_empty());
}

/// Valid `/prepare-candidate` body reusing the shared fixture files.
fn candidate_body(new_fid: &str, src_fid: &str, digest: &str) -> Vec<u8> {
    let raw = String::from_utf8(prepare_body(new_fid, "soda-reviewer", digest)).unwrap();
    let prep_start = raw.find("\"preparation\":").unwrap() + "\"preparation\":".len();
    let prep_end = raw.find(",\"setup\":").unwrap();
    let prep = &raw[prep_start..prep_end];
    format!("{{\"preparation\":{prep},\"source_preparation\":{src_fid:?},\"bundle\":\"Y2FuZGlkYXRlLWJ1bmRsZQ==\"}}")
        .into_bytes()
}

#[test]
fn prepare_candidate_reuses_protected_snapshot() {
    const SRC: &str = "f123456789abcdef012345678";
    const NEW: &str = "f223456789abcdef012345678";
    let digest = fixture_digest();
    let source = helper_state(
        "soda-reviewer",
        "ready",
        &digest,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let request = format!(
        "{{\"id\":{SRC:?},\"role\":\"soda-reviewer\",\"setup_digest\":{digest:?},\"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
    );
    let meta_ok = b"0:0:644:1:regular file\n".to_vec();
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(source),
        Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(request.into_bytes()),
        Ok(b"check.sh\nsetup.sh\nsource.bundle\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(b"true\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(b"true\n".to_vec()),
        Ok(format_inspect(true)),
        Ok(b"{}".to_vec()),
        Ok(approve_response(NEW, "soda-reviewer")),
        Ok(helper_state(
            "soda-reviewer",
            "ready",
            &digest,
            "",
            "",
            false,
            true,
            None,
            "",
            "",
        )),
    ]);
    let req = PrepareCandidateReq::decode(&candidate_body(NEW, SRC, &digest)).unwrap();
    let out = ops(&mock).prepare_candidate(&req, deadline()).unwrap();
    // Go `PrepareState` field order with the fresh identity.
    assert_eq!(
        out,
        format!(
            "{{\"id\":{NEW:?},\"project\":{PID:?},\"role\":\"soda-reviewer\",\"phase\":\"ready\",\"container\":{REV:?},\"source_commit\":{COMMIT:?},\"setup_digest\":{digest:?},\"ready\":true,\"stopped\":false}}"
        )
    );
    assert_eq!(
        helper_ops(&mock),
        vec!["inspect", "ensure", "approve", "inspect"]
    );
    assert_eq!(mock.calls.borrow().len(), 14);
}

#[test]
fn prepare_candidate_rejects_unready_source() {
    const SRC: &str = "f123456789abcdef012345678";
    const NEW: &str = "f223456789abcdef012345678";
    let digest = fixture_digest();
    let source = helper_state(
        "soda-reviewer",
        "running",
        &digest,
        "",
        "",
        false,
        false,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(source)]);
    let req = PrepareCandidateReq::decode(&candidate_body(NEW, SRC, &digest)).unwrap();
    assert_eq!(
        ops(&mock).prepare_candidate(&req, deadline()).unwrap_err(),
        "candidate source preparation is not ready for this role and setup"
    );
    assert_eq!(mock.calls.borrow().len(), 2);
}

#[test]
fn inspect_preparation_matches_golden() {
    let ready = helper_state(
        "soda-coder",
        "ready",
        REV,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(false)), Ok(ready)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    let out = ops(&mock).inspect_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_READY);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].2, binding_argv());
    assert_eq!(calls[1].2, helper_argv());
    assert_eq!(String::from_utf8_lossy(&calls[1].0), GO_HELPER_INSPECT);
}

#[test]
fn stop_preparation_confirms_retirement() {
    let stopped = helper_state(
        "soda-coder",
        "stopped",
        REV,
        "",
        "",
        true,
        false,
        None,
        "",
        "",
    );
    let stop_ok = format!("{{\"stopped\":{FID:?},\"retirement\":\"confirmed\",\"known\":true}}");
    let mock = Mock::new(vec![
        Ok(format_inspect(false)),
        Ok(stop_ok.into_bytes()),
        Ok(stopped),
    ]);
    let req =
        StopPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    let out = ops(&mock).stop_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_STOPPED);
    assert_eq!(helper_ops(&mock), vec!["stop", "inspect"]);
    // An unrecognized retirement marker stays an error, not a state.
    let stop_bad = format!("{{\"stopped\":{FID:?},\"retirement\":\"bogus\",\"known\":true}}");
    let mock = Mock::new(vec![Ok(format_inspect(false)), Ok(stop_bad.into_bytes())]);
    assert_eq!(
        ops(&mock).stop_preparation(&req, deadline()).unwrap_err(),
        "preparation stop unconfirmed"
    );
}

#[test]
fn hold_preparation_confirms_marker() {
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":true,\"revision\":2}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).hold_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_HOLD_TRUE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1].2, helper_argv());
    assert_eq!(
        String::from_utf8_lossy(&calls[1].0),
        "{\"op\":\"hold\",\"revision\":2}"
    );
    // Release clears the marker.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":false,\"revision\":0}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":false,\"revision\":0}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).hold_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_HOLD_FALSE);
    // A mismatched outcome is not confirmed.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":false,\"revision\":2}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    assert_eq!(
        ops(&mock).hold_preparation(&req, deadline()).unwrap_err(),
        "maintenance hold outcome not confirmed"
    );
    // A stopped project cannot take the hold path.
    let mock = Mock::new(vec![Ok(format_inspect(false))]);
    assert_eq!(
        ops(&mock).hold_preparation(&req, deadline()).unwrap_err(),
        "preparation target not ready or isolated"
    );
}

// ---------- response-shape oracles (Go-captured goldens) ----------

#[test]
fn oracle_prepare_state_full_encoding() {
    let tools = "{\"name\":\"python3\",\"path\":\"/usr/bin/python3\",\"version\":\"9.9-test\"}";
    let rich = helper_state(
        "soda-coder",
        "running",
        REV,
        tools,
        "",
        false,
        false,
        Some((0, 1)),
        "ok",
        "fail",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(rich)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    assert_eq!(
        ops(&mock).inspect_preparation(&req, deadline()).unwrap(),
        GO_PREPARE_FULL
    );
}

#[test]
fn oracle_prepare_state_missing_encoding() {
    let waiting = helper_state(
        "soda-coder",
        "waiting",
        REV,
        "",
        "node",
        false,
        false,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(waiting)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    assert_eq!(
        ops(&mock).inspect_preparation(&req, deadline()).unwrap(),
        GO_PREPARE_MISSING
    );
}

#[test]
fn oracle_access_key_state_empty_encoding() {
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(format!("{{\"revision\":{REV:?},\"keys\":[]}}").into_bytes()),
    ]);
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap(),
        GO_ACCESS_KEY_STATE_EMPTY
    );
}

#[test]
fn oracle_decode_errors_match_go_verbatim() {
    assert_eq!(
        AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1,"bogus":true}"#)
            .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        PrepareReq::decode(br#"{"preparation":{"id":1}}"#).unwrap_err(),
        GO_NESTED_TYPE_ERROR
    );
    assert_eq!(
        PrepareCandidateReq::decode(
            br#"{"preparation":{},"source_preparation":"f0123456789abcdef01234567","bundle":"!!!not-base64"}"#
        )
        .unwrap_err(),
        GO_BAD_BASE64
    );
}
