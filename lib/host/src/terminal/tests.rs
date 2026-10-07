use super::*;
use crate::project::Executor;
use crate::sha256;
use std::fs::File;
use std::io::BufReader;
use std::os::unix::io::AsRawFd;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Mutex, MutexGuard};
use std::time::{Duration, Instant};

const PID: &str = "p0123456789abcdef01234567";
const TID: &str = "0123456789abcdef0123456789abcdef";
const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);
static ENV_LOCK: Mutex<()> = Mutex::new(());

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

/// Unique short-lived scratch dir (no fixed `/tmp` paths).
fn test_tmp(slug: &str) -> std::path::PathBuf {
    let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("t26-{}-{}-{slug}", std::process::id(), n));
    std::fs::create_dir_all(&dir).unwrap();
    dir
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

    fn argvs(&self) -> Vec<Vec<String>> {
        self.calls()
            .into_iter()
            .map(|(_, cmd, args)| {
                let mut full = vec![cmd];
                full.extend(args);
                full
            })
            .collect()
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

fn ok(body: &str) -> Result<Vec<u8>, String> {
    Ok(body.as_bytes().to_vec())
}

fn inspect_json(
    id: &str,
    running: bool,
    project: &str,
    owner: &str,
    privileged: bool,
    userns: &str,
) -> String {
    format!(
        "{{\"id\":{id:?},\"running\":{running},\"project\":{project:?},\"owner\":{owner:?},\"privileged\":{privileged},\"userns\":{userns:?},\"mappings\":{{\"UidMap\":[\"0:100000:262144\"],\"GidMap\":[\"0:100000:262144\"]}}}}"
    )
}

// ----- predicates -----

#[test]
fn terminal_name_matrix() {
    assert!(valid_terminal_name(""));
    assert!(valid_terminal_name("plain name-1_2"));
    assert!(valid_terminal_name(&"x".repeat(80)));
    assert!(!valid_terminal_name(&"x".repeat(81)));
    assert!(valid_terminal_name(&"é".repeat(80)));
    assert!(!valid_terminal_name(&"é".repeat(81)));
    assert!(!valid_terminal_name("a\nb"));
    assert!(!valid_terminal_name("a\tb"));
    assert!(!valid_terminal_name("\u{0}"));
    assert!(!valid_terminal_name("\u{7f}"));
    // Cf format characters rejected.
    for c in [
        "\u{ad}",
        "\u{200b}",
        "\u{200f}",
        "\u{202a}",
        "\u{2060}",
        "\u{feff}",
        "\u{61c}",
        "\u{1d173}",
        "\u{e0001}",
    ] {
        assert!(!valid_terminal_name(c), "Cf {c:?} must be rejected");
    }
    // Nearby non-Cf characters accepted.
    for c in ["\u{a0}", "\u{2000}", "\u{2028}", "é", "中"] {
        assert!(valid_terminal_name(c), "{c:?} must be accepted");
    }
}

#[test]
fn id_predicates() {
    assert!(valid_terminal_id(TID));
    assert!(!valid_terminal_id(&TID[..31]));
    assert!(!valid_terminal_id(&format!("{TID}0")));
    assert!(!valid_terminal_id("0123456789ABCDEF0123456789ABCDEF"));
    assert!(terminal_dimensions(2, 2));
    assert!(terminal_dimensions(500, 300));
    assert!(!terminal_dimensions(1, 2));
    assert!(!terminal_dimensions(501, 2));
    assert!(!terminal_dimensions(2, 301));
}

fn base_request() -> TerminalRequest {
    TerminalRequest {
        action: "create".to_string(),
        id: TID.to_string(),
        project: PID.to_string(),
        login: "dev".to_string(),
        identity: 7,
        cols: 80,
        rows: 24,
        expires: now_unix() + 60,
        name: "term".to_string(),
        scope: CID.to_string(),
    }
}

#[test]
fn request_valid_matrix() {
    let now = now_unix();
    assert!(base_request().valid(now));
    // Actor faults.
    for broke in [
        TerminalRequest {
            project: "nope".to_string(),
            ..base_request()
        },
        TerminalRequest {
            login: "Root".to_string(),
            ..base_request()
        },
        TerminalRequest {
            login: "root".to_string(),
            ..base_request()
        },
        TerminalRequest {
            identity: 0,
            ..base_request()
        },
        TerminalRequest {
            name: "a\nb".to_string(),
            ..base_request()
        },
    ] {
        assert!(!broke.valid(now), "{broke:?}");
    }
    // Window edges.
    assert!(!TerminalRequest {
        expires: now,
        ..base_request()
    }
    .valid(now));
    assert!(TerminalRequest {
        expires: now + 12 * 3600,
        ..base_request()
    }
    .valid(now));
    assert!(!TerminalRequest {
        expires: now + 12 * 3600 + 1,
        ..base_request()
    }
    .valid(now));
    // Scope rules.
    assert!(!TerminalRequest {
        scope: String::new(),
        ..base_request()
    }
    .valid(now));
    assert!(!TerminalRequest {
        scope: TID.to_string(),
        ..base_request()
    }
    .valid(now));
    let mut attach = base_request();
    attach.action = "attach".to_string();
    attach.scope = String::new();
    attach.name = String::new();
    assert!(attach.valid(now));
    attach.scope = CID.to_string();
    assert!(!attach.valid(now));
    // List shape.
    let list = TerminalRequest {
        action: "list".to_string(),
        id: String::new(),
        project: PID.to_string(),
        login: "dev".to_string(),
        identity: 7,
        cols: 0,
        rows: 0,
        expires: now + 60,
        name: String::new(),
        scope: String::new(),
    };
    assert!(list.valid(now));
    assert!(!TerminalRequest {
        id: TID.to_string(),
        ..list.clone()
    }
    .valid(now));
    assert!(!TerminalRequest {
        cols: 80,
        ..list.clone()
    }
    .valid(now));
    assert!(!TerminalRequest {
        scope: CID.to_string(),
        ..list.clone()
    }
    .valid(now));
    // Idle actions.
    for action in ["inspect", "end", "rename"] {
        let mut idle = base_request();
        idle.action = action.to_string();
        idle.cols = 0;
        idle.rows = 0;
        idle.scope = String::new();
        if action != "rename" {
            idle.name = String::new();
        }
        assert!(idle.valid(now), "{action}");
        idle.cols = 80;
        assert!(!idle.valid(now), "{action} sized");
    }
    let mut rename = base_request();
    rename.action = "rename".to_string();
    rename.cols = 0;
    rename.rows = 0;
    rename.scope = String::new();
    assert!(rename.valid(now));
    // Unknown action and bad id.
    assert!(!TerminalRequest {
        action: "kill".to_string(),
        ..base_request()
    }
    .valid(now));
    assert!(!TerminalRequest {
        id: "short".to_string(),
        ..base_request()
    }
    .valid(now));
}

#[test]
fn strict_b64_vectors() {
    // Probed against Go `Strict()`: trailing bits checked, newlines skipped.
    assert_eq!(strict_b64_decode("AB/C").unwrap(), vec![0x00, 0x1f, 0xc2]);
    assert_eq!(strict_b64_decode("AB/D").unwrap(), vec![0x00, 0x1f, 0xc3]);
    assert_eq!(strict_b64_decode("AQ==").unwrap(), vec![0x01]);
    assert!(strict_b64_decode("AR==").is_none());
    assert!(strict_b64_decode("AZ==").is_none());
    assert!(strict_b64_decode("A/==").is_none());
    assert!(strict_b64_decode("A+/=").is_none());
    assert!(strict_b64_decode("AB=C").is_none());
    assert!(strict_b64_decode("ABC").is_none());
    assert!(strict_b64_decode("ABCD====").is_none());
    assert!(strict_b64_decode("AB=CDEF").is_none());
    assert_eq!(strict_b64_decode("AB\nCD").unwrap(), vec![0x00, 0x10, 0x83]);
    assert_eq!(strict_b64_decode("").unwrap(), Vec::<u8>::new());
    assert!(strict_b64_decode("====").is_none());
    assert!(strict_b64_decode("AB").is_none());
}

#[test]
fn frame_input_matrix() {
    let input = |data: &str| TerminalFrame {
        frame_type: "input".to_string(),
        data: data.to_string(),
        ..Default::default()
    };
    assert!(input("aGk=").input_valid());
    assert!(!input("").input_valid());
    assert!(!input("AR==").input_valid());
    assert!(!input("AB\nCD").input_valid());
    assert!(!input("AB\rCD").input_valid());
    assert!(input(&"QUFB".repeat(5461)).input_valid()); // 16383 bytes
    assert!(!input(&"QUFB".repeat(5462)).input_valid()); // 16386 bytes
    assert!(!TerminalFrame {
        cols: 1,
        ..input("aGk=")
    }
    .input_valid());
    assert!(!TerminalFrame {
        reason: "x".to_string(),
        ..input("aGk=")
    }
    .input_valid());
    assert!(!TerminalFrame {
        terminals: Some(vec![]),
        ..input("aGk=")
    }
    .input_valid());
    let resize = |cols, rows| TerminalFrame {
        frame_type: "resize".to_string(),
        cols,
        rows,
        ..Default::default()
    };
    assert!(resize(80, 24).input_valid());
    assert!(!resize(1, 24).input_valid());
    assert!(!TerminalFrame {
        data: "aGk=".to_string(),
        ..resize(80, 24)
    }
    .input_valid());
    for t in ["heartbeat", "close"] {
        assert!(TerminalFrame {
            frame_type: t.to_string(),
            ..Default::default()
        }
        .input_valid());
        assert!(!TerminalFrame {
            frame_type: t.to_string(),
            data: "aGk=".to_string(),
            ..Default::default()
        }
        .input_valid());
    }
    assert!(!TerminalFrame {
        frame_type: "output".to_string(),
        ..Default::default()
    }
    .input_valid());
    assert!(!TerminalFrame::default().input_valid());
}

#[test]
fn frame_output_matrix() {
    let output = |data: &str| TerminalFrame {
        frame_type: "output".to_string(),
        data: data.to_string(),
        ..Default::default()
    };
    assert!(output("aGk=").output_valid());
    // Newlines accepted on the output path (Go `Strict` skips them).
    assert!(output("AB\nCD").output_valid());
    assert!(!output("").output_valid());
    assert!(!output("AR==").output_valid());
    assert!(output(&"QUFB".repeat(1365)).output_valid()); // 4095 bytes
    assert!(!output(&"QUFB".repeat(1366)).output_valid()); // 4098 bytes
    assert!(!TerminalFrame {
        reason: "x".to_string(),
        ..output("aGk=")
    }
    .output_valid());
    assert!(!TerminalFrame {
        cols: 1,
        ..output("aGk=")
    }
    .output_valid());
    assert!(TerminalFrame {
        frame_type: "ready".to_string(),
        ..Default::default()
    }
    .output_valid());
    assert!(!TerminalFrame {
        frame_type: "ready".to_string(),
        data: "aGk=".to_string(),
        ..Default::default()
    }
    .output_valid());
    for reason in [
        "disconnected",
        "expired",
        "exited",
        "launch_failed",
        "stream_failed",
        "cleanup_unconfirmed",
    ] {
        assert!(
            TerminalFrame {
                frame_type: "closed".to_string(),
                reason: reason.to_string(),
                ..Default::default()
            }
            .output_valid(),
            "{reason}"
        );
    }
    assert!(!TerminalFrame {
        frame_type: "closed".to_string(),
        reason: "nope".to_string(),
        ..Default::default()
    }
    .output_valid());
    assert!(!TerminalFrame {
        frame_type: "closed".to_string(),
        ..Default::default()
    }
    .output_valid());
    assert!(!TerminalFrame {
        frame_type: "input".to_string(),
        ..Default::default()
    }
    .output_valid());
}

fn meta_state(id: &str, ready: bool, attached: bool, state: &str) -> TerminalState {
    TerminalState {
        id: id.to_string(),
        name: "t".to_string(),
        created_at: 1700000000,
        ready,
        attached,
        state: state.to_string(),
    }
}

#[test]
fn metadata_output_matrix() {
    let meta = |items: Vec<TerminalState>| TerminalFrame {
        frame_type: "metadata".to_string(),
        terminals: Some(items),
        ..Default::default()
    };
    assert!(meta(vec![meta_state(TID, true, true, "ready")]).output_valid());
    assert!(meta(vec![meta_state(TID, false, false, "opening")]).output_valid());
    assert!(meta(vec![]).output_valid());
    assert!(!TerminalFrame {
        frame_type: "metadata".to_string(),
        ..Default::default()
    }
    .output_valid());
    assert!(!TerminalFrame {
        frame_type: "metadata".to_string(),
        data: "aGk=".to_string(),
        ..Default::default()
    }
    .output_valid());
    // Shape: terminals forbidden off metadata.
    assert!(!TerminalFrame {
        frame_type: "ready".to_string(),
        terminals: Some(vec![]),
        ..Default::default()
    }
    .output_valid());
    // Item faults.
    assert!(!meta(vec![meta_state("short", true, true, "ready")]).output_valid());
    assert!(!meta(vec![
        meta_state(TID, true, true, "ready"),
        meta_state(TID, true, true, "ready")
    ])
    .output_valid());
    assert!(!meta(vec![TerminalState {
        name: "x\ny".to_string(),
        ..meta_state(TID, true, true, "ready")
    }])
    .output_valid());
    assert!(!meta(vec![TerminalState {
        created_at: 0,
        ..meta_state(TID, true, true, "ready")
    }])
    .output_valid());
    assert!(!meta(vec![TerminalState {
        created_at: 9007199254740992,
        ..meta_state(TID, true, true, "ready")
    }])
    .output_valid());
    assert!(meta(vec![TerminalState {
        created_at: 9007199254740991,
        ..meta_state(TID, true, true, "ready")
    }])
    .output_valid());
    assert!(!meta(vec![meta_state(TID, false, true, "ready")]).output_valid());
    assert!(!meta(vec![meta_state(TID, true, false, "opening")]).output_valid());
    assert!(!meta(vec![meta_state(TID, false, false, "bogus")]).output_valid());
    // 64-row cap.
    let many: Vec<TerminalState> = (0..65)
        .map(|i| meta_state(&format!("{i:032x}"), true, false, "ready"))
        .collect();
    assert!(!meta(many).output_valid());
    let edge: Vec<TerminalState> = (0..64)
        .map(|i| meta_state(&format!("{i:032x}"), true, false, "ready"))
        .collect();
    assert!(meta(edge).output_valid());
}

// ----- wire codec goldens (byte-exact vs Go `encoding/json`) -----

#[test]
fn frame_encode_goldens() {
    let cases: Vec<(TerminalFrame, &str)> = vec![
        (
            TerminalFrame {
                frame_type: "input".to_string(),
                data: "aGk=".to_string(),
                ..Default::default()
            },
            r#"{"type":"input","data":"aGk="}"#,
        ),
        (
            TerminalFrame {
                frame_type: "resize".to_string(),
                cols: 80,
                rows: 24,
                ..Default::default()
            },
            r#"{"type":"resize","cols":80,"rows":24}"#,
        ),
        (
            TerminalFrame {
                frame_type: "close".to_string(),
                ..Default::default()
            },
            r#"{"type":"close"}"#,
        ),
        (
            TerminalFrame {
                frame_type: "output".to_string(),
                data: "aGkK".to_string(),
                ..Default::default()
            },
            r#"{"type":"output","data":"aGkK"}"#,
        ),
        (
            TerminalFrame {
                frame_type: "closed".to_string(),
                reason: "exited".to_string(),
                ..Default::default()
            },
            r#"{"type":"closed","reason":"exited"}"#,
        ),
        (
            TerminalFrame {
                frame_type: "ready".to_string(),
                ..Default::default()
            },
            r#"{"type":"ready"}"#,
        ),
        (
            TerminalFrame {
                frame_type: "metadata".to_string(),
                terminals: Some(vec![
                    TerminalState {
                        id: TID.to_string(),
                        name: "a<b>&\"c".to_string(),
                        created_at: 1700000000,
                        ready: true,
                        attached: true,
                        state: "ready".to_string(),
                    },
                    TerminalState {
                        id: "fedcba9876543210fedcba9876543210".to_string(),
                        created_at: 1,
                        state: "opening".to_string(),
                        ..Default::default()
                    },
                ]),
                ..Default::default()
            },
            r#"{"type":"metadata","terminals":[{"id":"0123456789abcdef0123456789abcdef","name":"a\u003cb\u003e\u0026\"c","created_at":1700000000,"ready":true,"attached":true,"state":"ready"},{"id":"fedcba9876543210fedcba9876543210","name":"","created_at":1,"ready":false,"attached":false,"state":"opening"}]}"#,
        ),
    ];
    for (frame, want) in cases {
        assert_eq!(frame.encode(), want, "{frame:?}");
        assert_eq!(
            TerminalFrame::decode(frame.encode().as_bytes()).unwrap(),
            frame
        );
    }
}

#[test]
fn request_encode_golden() {
    let req = TerminalRequest {
        action: "create".to_string(),
        id: TID.to_string(),
        project: PID.to_string(),
        login: "dev".to_string(),
        identity: 7,
        cols: 80,
        rows: 24,
        expires: 1900000000,
        name: "n".to_string(),
        scope: CID.to_string(),
    };
    assert_eq!(
        req.encode(),
        r#"{"action":"create","id":"0123456789abcdef0123456789abcdef","project":"p0123456789abcdef01234567","login":"dev","identity":7,"cols":80,"rows":24,"expires":1900000000,"name":"n","scope":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}"#
    );
    assert_eq!(
        TerminalRequest::decode(req.encode().as_bytes()).unwrap(),
        req
    );
}

fn golden_binding() -> Binding {
    Binding {
        child_id: "f0123456789abcdef01234567".to_string(),
        uid: 1000,
        gid: 1000,
        scope: "muse-project".to_string(),
        credential_root: "/run/soda-muse/abcd".to_string(),
        invocation_id: TID.to_string(),
        kind: "terminal".to_string(),
        id: "abcd".to_string(),
        project: CID.to_string(),
        login: "dev".to_string(),
        generation: 3,
    }
}

fn golden_lease() -> Lease {
    Lease {
        repository_id: 9,
        provider_id: "muse".to_string(),
        id: "lease-1".to_string(),
        connection_id: "conn-1".to_string(),
        generation: 3,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: TID.to_string(),
        kind: "terminal".to_string(),
        role: "r".to_string(),
        deadline_raw: "2026-10-04T12:00:00.123456789Z".to_string(),
        deadline: parse_rfc3339("2026-10-04T12:00:00.123456789Z"),
        grant_id: "g".to_string(),
        grant_revision: 2,
        binding: Some(golden_binding()),
    }
}

#[test]
fn identity_encode_goldens() {
    assert_eq!(
        golden_binding().encode(),
        r#"{"child_id":"f0123456789abcdef01234567","uid":1000,"gid":1000,"scope":"muse-project","credential_root":"/run/soda-muse/abcd","invocation_id":"0123456789abcdef0123456789abcdef","kind":"terminal","id":"abcd","project":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","login":"dev","generation":3}"#
    );
    let lease_golden = r#"{"repository_id":"9","provider_id":"muse","id":"lease-1","connection_id":"conn-1","generation":3,"actor_id":"7","project_id":"p0123456789abcdef01234567","execution_id":"0123456789abcdef0123456789abcdef","kind":"terminal","role":"r","deadline":"2026-10-04T12:00:00.123456789Z","grant_id":"g","grant_revision":2,"binding":{"child_id":"f0123456789abcdef01234567","uid":1000,"gid":1000,"scope":"muse-project","credential_root":"/run/soda-muse/abcd","invocation_id":"0123456789abcdef0123456789abcdef","kind":"terminal","id":"abcd","project":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","login":"dev","generation":3}}"#;
    assert_eq!(golden_lease().encode(), lease_golden);
    for (credential, suffix) in [
        (None, "null"),
        (Some(vec![]), r#""""#),
        (Some(b"{\"a\":1}".to_vec()), r#""eyJhIjoxfQ==""#),
    ] {
        let delivery = Delivery {
            lease: golden_lease(),
            credential,
        };
        assert_eq!(
            delivery.encode(),
            format!("{{\"lease\":{lease_golden},\"credential\":{suffix}}}")
        );
        assert_eq!(
            Delivery::decode(delivery.encode().as_bytes()).unwrap(),
            delivery
        );
    }
    // Lean lease: omitempty drops zero fields.
    let lean = Lease {
        provider_id: "codex".to_string(),
        id: "l".to_string(),
        connection_id: "c".to_string(),
        generation: 1,
        actor_id: 1,
        project_id: PID.to_string(),
        execution_id: TID.to_string(),
        kind: "terminal".to_string(),
        deadline_raw: "2026-01-01T00:00:00Z".to_string(),
        deadline: parse_rfc3339("2026-01-01T00:00:00Z"),
        ..Default::default()
    };
    assert_eq!(
        lean.encode(),
        r#"{"provider_id":"codex","id":"l","connection_id":"c","generation":1,"actor_id":"1","project_id":"p0123456789abcdef01234567","execution_id":"0123456789abcdef0123456789abcdef","kind":"terminal","deadline":"2026-01-01T00:00:00Z"}"#
    );
    assert_eq!(
        Delivery::decode(format!("{{\"lease\":{},\"credential\":null}}", lean.encode()).as_bytes())
            .unwrap()
            .lease,
        lean
    );
}

#[test]
fn strict_decode_matrix() {
    assert!(TerminalRequest::decode(br#"{"action":"list","bogus":1}"#).is_err());
    assert!(TerminalFrame::decode(br#"{"type":"close","bogus":1}"#).is_err());
    assert!(TerminalFrame::decode(br#"{"type":"close","terminals":"x"}"#).is_err());
    assert!(TerminalFrame::decode(br#"{"type":"close","terminals":[7]}"#).is_err());
    // terminals tri-state: missing/null -> None, [] -> Some(empty).
    assert_eq!(
        TerminalFrame::decode(br#"{"type":"close"}"#)
            .unwrap()
            .terminals,
        None
    );
    assert_eq!(
        TerminalFrame::decode(br#"{"type":"close","terminals":null}"#)
            .unwrap()
            .terminals,
        None
    );
    assert_eq!(
        TerminalFrame::decode(br#"{"type":"metadata","terminals":[]}"#)
            .unwrap()
            .terminals,
        Some(vec![])
    );
    // Folded duplicate terminals keys are an unknown field.
    assert!(
        TerminalFrame::decode(br#"{"type":"close","terminals":null,"Terminals":null}"#).is_err()
    );
    // Null items decode to zero states.
    let f = TerminalFrame::decode(br#"{"type":"metadata","terminals":[null]}"#).unwrap();
    assert_eq!(f.terminals, Some(vec![TerminalState::default()]));
    // Trailing data rejected.
    assert!(TerminalFrame::decode(br#"{"type":"close"} {}"#).is_err());
    // Delivery credential shapes.
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":"aGk="}"#)
            .unwrap()
            .credential,
        Some(b"hi".to_vec())
    );
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":[104,105]}"#)
            .unwrap()
            .credential,
        Some(b"hi".to_vec())
    );
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":null}"#)
            .unwrap()
            .credential,
        None
    );
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"},"CREDENTIAL":"aGk=","Credential":null}"#)
            .unwrap()
            .credential,
        Some(b"hi".to_vec())
    );
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"},"credential":""}"#)
            .unwrap()
            .credential,
        Some(vec![])
    );
    assert_eq!(
        Delivery::decode(br#"{"lease":{"provider_id":"x"}}"#)
            .unwrap()
            .credential,
        None
    );
    assert!(Delivery::decode(br#"{"lease":{},"credential":"!!!"}"#).is_err());
    // ,string edges.
    assert!(Delivery::decode(br#"{"lease":{"actor_id":5}}"#).is_err());
    assert!(Delivery::decode(br#"{"lease":{"actor_id":""}}"#).is_err());
    assert!(Delivery::decode(br#"{"lease":{"deadline":"not-a-time"}}"#).is_err());
    let d = Delivery::decode(br#"{"lease":{"actor_id":"007","deadline":"2026-01-01T00:00:00Z"}}"#)
        .unwrap();
    assert_eq!(d.lease.actor_id, 7);
}

#[test]
fn credential_valid_vectors() {
    // Probed against Go `json.Valid`.
    for (body, want) in [
        ("01", false),
        (" 1 ", true),
        ("", false),
        ("null", true),
        ("{},", false),
        ("{}", true),
        ("[1,2]", true),
        ("\"a\"", true),
        ("1e3", true),
        ("-0", true),
        ("0.5", true),
        ("--1", false),
        ("{\"a\":1}", true),
        ("[", false),
    ] {
        assert_eq!(credential_valid(body.as_bytes()), want, "{body:?}");
    }
    assert!(!credential_valid(&vec![b'{'; CREDENTIAL_LIMIT + 1]));
    assert!(!credential_valid(&vec![b' '; CREDENTIAL_LIMIT - 2])); // whitespace is not a value
    assert!(json_valid(b"{}"));
}

#[test]
fn string_i64_vectors() {
    for (raw, want) in [
        ("5", Some(5)),
        ("-5", Some(-5)),
        ("05", Some(5)),
        ("00", Some(0)),
        ("-0", Some(0)),
        ("0", Some(0)),
        ("9223372036854775807", Some(i64::MAX)),
        ("-9223372036854775808", Some(i64::MIN)),
    ] {
        assert_eq!(parse_string_i64(raw), want, "{raw:?}");
    }
    for raw in [
        "",
        " 5",
        "+5",
        "5 ",
        "5.0",
        "5e1",
        "0x5",
        "9223372036854775808",
        "-9223372036854775809",
        "-",
        "５",
    ] {
        assert!(parse_string_i64(raw).is_none(), "{raw:?}");
    }
}

#[test]
fn rfc3339_vectors() {
    assert_eq!(parse_rfc3339("1970-01-01T00:00:00Z"), Some((0, 0)));
    assert_eq!(parse_rfc3339("2026-01-01T00:00:00Z"), Some((1767225600, 0)));
    assert_eq!(
        parse_rfc3339("2026-10-04T12:00:00.123456789Z").unwrap().1,
        123456789
    );
    assert_eq!(
        parse_rfc3339("2026-10-04T12:00:00.5Z").unwrap().1,
        500000000
    );
    let a = parse_rfc3339("2026-10-04T12:00:00Z").unwrap();
    assert_eq!(
        parse_rfc3339("2026-10-04T14:00:00+02:00").unwrap(),
        (a.0, 0)
    );
    assert_eq!(
        parse_rfc3339("2026-10-04T07:00:00-05:00").unwrap(),
        (a.0, 0)
    );
    assert_eq!(parse_rfc3339("2026-10-04T07:00:00-0500"), None);
    assert_eq!(parse_rfc3339("2026-10-04T07:00:00-05"), None);
    assert_eq!(parse_rfc3339("2024-02-29T00:00:00Z").unwrap().0, 1709164800);
    for bad in [
        "",
        "not-a-time",
        "2026-13-01T00:00:00Z",
        "2026-00-01T00:00:00Z",
        "2026-02-30T00:00:00Z",
        "2025-02-29T00:00:00Z",
        "2026-01-01T24:00:00Z",
        "2026-01-01T00:60:00Z",
        "2026-01-01T00:00:61Z",
        "2026-01-01T00:00:00",
        "2026-01-01T00:00:00.",
        "2026-01-01T00:00:00.Z",
        "2026-01-01t00:00:00Z",
        "2026-01-01T00:00:00z",
        "2026-01-01T00:00:00+25:00",
        "2026-01-01T00:00:00+02:60",
        "2026-01-01T00:00:00.1234567890Z",
    ] {
        assert_eq!(parse_rfc3339(bad), None, "{bad:?}");
    }
}

// ----- inspection + argv -----

#[test]
fn id_map_matrix() {
    let yes = |s: &str| terminal_id_map(&[s.to_string()]);
    assert!(yes("0:100000:262144"));
    assert!(yes("0:1:262144"));
    assert!(yes("0:4294705151:262144")); // max base: 4294967295-262144
    assert!(!yes("0:4294705152:262144"));
    assert!(!yes("0:0:262144"));
    assert!(!yes("0:0100000:262144"));
    assert!(!yes("1:100000:262144"));
    assert!(!yes("0:100000:262145"));
    assert!(!yes("0:100000"));
    assert!(!yes("0:100000:262144:extra"));
    assert!(!yes("0:+100000:262144"));
    assert!(!yes("0:4294967296:262144"));
    assert!(!terminal_id_map(&[]));
    assert!(!terminal_id_map(&[
        "0:1:262144".to_string(),
        "0:1:262144".to_string()
    ]));
}

fn inspection() -> TerminalInspection {
    TerminalInspection {
        id: CID.to_string(),
        running: true,
        project: PID.to_string(),
        owner: "7".to_string(),
        privileged: false,
        userns: "private".to_string(),
        uid_map: vec!["0:100000:262144".to_string()],
        gid_map: vec!["0:100000:262144".to_string()],
    }
}

#[test]
fn isolation_matrix() {
    assert!(terminal_isolation(&inspection(), PID));
    assert!(!terminal_isolation(
        &TerminalInspection {
            id: TID.to_string(),
            ..inspection()
        },
        PID
    ));
    assert!(!terminal_isolation(
        &TerminalInspection {
            project: "pffffffffffffffffffffffff".to_string(),
            ..inspection()
        },
        PID
    ));
    assert!(!terminal_isolation(
        &TerminalInspection {
            privileged: true,
            ..inspection()
        },
        PID
    ));
    assert!(!terminal_isolation(
        &TerminalInspection {
            userns: "host".to_string(),
            ..inspection()
        },
        PID
    ));
    assert!(!terminal_isolation(
        &TerminalInspection {
            gid_map: vec![],
            ..inspection()
        },
        PID
    ));
    assert!(terminal_target_ready(&inspection(), PID, true));
    assert!(terminal_target_ready(
        &TerminalInspection {
            running: false,
            ..inspection()
        },
        PID,
        false
    ));
    assert!(!terminal_target_ready(
        &TerminalInspection {
            running: false,
            ..inspection()
        },
        PID,
        true
    ));
    assert!(!terminal_target_ready(
        &TerminalInspection {
            owner: "0".to_string(),
            ..inspection()
        },
        PID,
        true
    ));
    assert!(!terminal_target_ready(
        &TerminalInspection {
            owner: "-3".to_string(),
            ..inspection()
        },
        PID,
        true
    ));
    assert!(!terminal_target_ready(
        &TerminalInspection {
            owner: "no".to_string(),
            ..inspection()
        },
        PID,
        true
    ));
    assert!(terminal_target_ready(
        &TerminalInspection {
            owner: "+7".to_string(),
            ..inspection()
        },
        PID,
        true
    ));
    assert!(terminal_target_ready(
        &TerminalInspection {
            owner: "007".to_string(),
            ..inspection()
        },
        PID,
        true
    ));
    // Strict decode pins.
    assert!(TerminalInspection::decode(br#"{"id":"x","unknown":1}"#).is_err());
    let v =
        TerminalInspection::decode(inspect_json(CID, true, PID, "7", false, "private").as_bytes())
            .unwrap();
    assert_eq!(v, inspection());
}

#[test]
fn exit_code_pins() {
    assert_eq!(exit_code_of("exit status 1"), Some(1));
    assert_eq!(
        exit_code_of("/usr/bin/podman failed: exit status 32: boom"),
        Some(32)
    );
    assert_eq!(exit_code_of("exit status 0"), Some(0));
    assert_eq!(exit_code_of("signal: killed"), None);
    assert_eq!(exit_code_of("exit status "), None);
    assert_eq!(exit_code_of(""), None);
}

#[test]
fn argv_vectors() {
    assert_eq!(
        inspect_argv("soda-abc"),
        vec![
            "--remote=false",
            "inspect",
            "--format",
            TERMINAL_INSPECT,
            "soda-abc"
        ]
    );
    assert_eq!(
        container_exists_argv(CID),
        vec!["--remote=false", "container", "exists", CID]
    );
    assert_eq!(
        agent_argv(CID, &["broker"]),
        vec![
            "--remote=false",
            "exec",
            "--interactive",
            CID,
            "/usr/libexec/soda/project-terminal",
            "broker"
        ]
    );
    // Current native attach argv: python is gone, fixed agent binary.
    assert_eq!(
        native_argv(CID, "create", TID, "dev", 7, 80, 24, 3600, "hash", "n", "scope"),
        vec![
            "--remote=false",
            "exec",
            "--interactive",
            CID,
            "/usr/libexec/soda/project-terminal",
            "create",
            TID,
            "dev",
            "7",
            "80",
            "24",
            "3600",
            "hash",
            "n",
            "scope",
        ]
    );
    assert_eq!(
        tar_producer_argv("/opt/harness//x/../"),
        vec!["--create", "--file=-", "--directory", "/opt/harness", "."]
    );
    assert_eq!(
        tar_consumer_argv(CID, "/run/x"),
        vec![
            "--remote=false",
            "exec",
            "--interactive",
            CID,
            "/usr/bin/tar",
            "--extract",
            "--file=-",
            "--directory",
            "/run/x",
            "--no-same-owner",
            "--same-permissions",
        ]
    );
}

#[test]
fn clean_path_vectors() {
    for (input, want) in [
        ("/a/b/c", "/a/b/c"),
        ("/a//b/./c/", "/a/b/c"),
        ("/a/b/../c", "/a/c"),
        ("/../a", "/a"),
        ("", "."),
        (".", "."),
        ("a/../../b", "../b"),
        ("/", "/"),
        ("a/b/", "a/b"),
        ("/opt/harness//x/../", "/opt/harness"),
    ] {
        assert_eq!(clean_path(input), want, "{input:?}");
    }
}

// ----- service exec flows -----

fn make_service(exec: FakeExec) -> Service<FakeExec> {
    Service {
        exec,
        codex_harness: "/opt/harness".to_string(),
        codex_harness_sha256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
        codex_harness_version: "1.0".to_string(),
        muse_harness: String::new(),
        muse_harness_sha256: String::new(),
        muse_harness_version: String::new(),
    }
}

#[test]
fn project_container_flows() {
    // Invalid project: no exec call.
    let exec = FakeExec::new(vec![]);
    let svc = make_service(exec);
    assert_eq!(
        svc.project_container("nope", true, deadline()).unwrap_err(),
        "invalid project"
    );
    // Exec failure and oversize.
    let exec = FakeExec::new(vec![Err("boom".to_string())]);
    let svc = make_service(exec);
    assert_eq!(
        svc.project_container(PID, true, deadline()).unwrap_err(),
        "terminal inspection unavailable"
    );
    let exec = FakeExec::new(vec![Ok(vec![b'x'; 4097])]);
    let svc = make_service(exec);
    assert_eq!(
        svc.project_container(PID, true, deadline()).unwrap_err(),
        "terminal inspection unavailable"
    );
    // Bad JSON and not-ready target.
    let exec = FakeExec::new(vec![ok("{nope")]);
    let svc = make_service(exec);
    assert_eq!(
        svc.project_container(PID, true, deadline()).unwrap_err(),
        "invalid terminal inspection"
    );
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, false, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.project_container(PID, true, deadline()).unwrap_err(),
        "terminal target not ready or isolated"
    );
    // Success pins argv and container id.
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, true, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(svc.project_container(PID, true, deadline()).unwrap(), CID);
    assert_eq!(
        svc.exec.argvs(),
        vec![{
            let mut v = vec!["/usr/bin/podman".to_string()];
            v.extend(inspect_argv(&format!("soda-{PID}")));
            v
        }]
    );
    assert!(svc.exec.calls()[0].0.is_empty());
    // Stopped container admitted when running is not required.
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, false, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(svc.project_container(PID, false, deadline()).unwrap(), CID);
}

#[test]
fn factory_project_container_flows() {
    let exec = FakeExec::new(vec![]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container("nope", true, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let exec = FakeExec::new(vec![Err("boom".to_string())]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_STALE
    );
    let exec = FakeExec::new(vec![ok("")]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_STALE
    );
    let exec = FakeExec::new(vec![ok("{nope")]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_STALE
    );
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, true, PID, "7", true, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, true, PID, "0", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, false, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap_err(),
        ERR_STALE
    );
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, true, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.factory_project_container(PID, true, deadline())
            .unwrap(),
        CID
    );
}

fn live_lease() -> Lease {
    Lease {
        provider_id: "codex".to_string(),
        id: "lease-1".to_string(),
        connection_id: "conn".to_string(),
        generation: 2,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: TID.to_string(),
        kind: KIND_TERMINAL.to_string(),
        deadline_raw: "2026-10-05T00:00:00Z".to_string(),
        deadline: parse_rfc3339("2026-10-05T00:00:00Z"),
        binding: Some(Binding {
            kind: KIND_TERMINAL.to_string(),
            id: TID.to_string(),
            project: CID.to_string(),
            login: "dev".to_string(),
            generation: 2,
            ..Default::default()
        }),
        ..Default::default()
    }
}

#[test]
fn identity_call_flows() {
    let req = IdentityRequest {
        action: "validate".to_string(),
        ..Default::default()
    };
    let exec = FakeExec::new(vec![Err("boom".to_string())]);
    let svc = make_service(exec);
    assert_eq!(
        svc.identity_call(CID, &req, deadline()).unwrap_err(),
        "managed Codex operation failed"
    );
    let exec = FakeExec::new(vec![Ok(vec![b'x'; BROKER_RESPONSE_LIMIT + 1])]);
    let svc = make_service(exec);
    assert_eq!(
        svc.identity_call(CID, &req, deadline()).unwrap_err(),
        "invalid managed Codex response"
    );
    let exec = FakeExec::new(vec![ok("{nope")]);
    let svc = make_service(exec);
    assert_eq!(
        svc.identity_call(CID, &req, deadline()).unwrap_err(),
        "invalid managed Codex response"
    );
    // Success pins stdin bytes and broker argv.
    let body = Delivery {
        lease: live_lease(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![ok(&body)]);
    let svc = make_service(exec);
    let out = svc.identity_call(CID, &req, deadline()).unwrap();
    assert_eq!(out.lease, live_lease());
    let calls = svc.exec.calls();
    assert_eq!(calls[0].0, req.encode().as_bytes());
    let mut want = vec!["/usr/bin/podman".to_string()];
    want.extend(agent_argv(CID, &["broker"]));
    assert_eq!(svc.exec.argvs(), vec![want]);
}

#[test]
fn terminal_lease_matrix() {
    let now = now_unix();
    let mut lease = live_lease();
    assert!(terminal_lease(&lease, false, now));
    lease.binding = None;
    assert!(!terminal_lease(&lease, false, now));
    lease.deadline_raw = "2020-01-01T00:00:00Z".to_string();
    lease.deadline = parse_rfc3339(&lease.deadline_raw);
    assert!(!terminal_lease(&lease, true, now)); // not after now
    lease.deadline = Some((now + 3600, 0));
    assert!(terminal_lease(&lease, true, now)); // live window
    lease.deadline = Some((now + 12 * 3600 + 1, 0));
    assert!(!terminal_lease(&lease, true, now)); // beyond 12h
    lease.deadline = Some((now, 0));
    assert!(!terminal_lease(&lease, true, now)); // not strictly after now
    lease = live_lease();
    lease.execution_id = "short".to_string();
    assert!(!terminal_lease(&lease, false, now));
    lease = live_lease();
    lease.binding.as_mut().unwrap().generation = 99;
    assert!(!terminal_lease(&lease, false, now));
}

#[test]
fn identity_result_matrix() {
    let delivery = Delivery {
        lease: live_lease(),
        credential: None,
    };
    assert_eq!(
        identity_result("validate", &delivery, &delivery).unwrap(),
        delivery
    );
    let mut other = delivery.clone();
    other.lease.id = "other".to_string();
    assert_eq!(
        identity_result("validate", &delivery, &other).unwrap_err(),
        ERR_STALE
    );
    other = delivery.clone();
    other.lease.binding = None;
    assert_eq!(
        identity_result("validate", &delivery, &other).unwrap_err(),
        ERR_STALE
    );
    other = delivery.clone();
    other.lease.binding.as_mut().unwrap().login = "mallory".to_string();
    assert_eq!(
        identity_result("validate", &delivery, &other).unwrap_err(),
        ERR_STALE
    );
    other = delivery.clone();
    other.credential = Some(b"{}".to_vec());
    assert_eq!(
        identity_result("validate", &delivery, &other).unwrap_err(),
        "unexpected credential response"
    );
    assert_eq!(identity_result("finish", &delivery, &other).unwrap(), other);
    other = delivery.clone();
    other.credential = Some(b"{}".to_vec());
    let mut bad = other.clone();
    bad.credential = Some(b"nope".to_vec());
    assert_eq!(
        identity_result("finish", &delivery, &bad).unwrap_err(),
        ERR_UNCERTAIN
    );
    assert_eq!(
        identity_result("finish", &delivery, &delivery).unwrap_err(),
        ERR_UNCERTAIN
    );
}

#[test]
fn identity_dispatch_matrix() {
    // Denied lease and denied action: no exec calls.
    let exec = FakeExec::new(vec![]);
    let svc = make_service(exec);
    let bad = Delivery::default();
    assert_eq!(
        svc.identity("validate", &bad, deadline()).unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.identity(
            "bogus",
            &Delivery {
                lease: live_lease(),
                credential: None
            },
            deadline()
        )
        .unwrap_err(),
        ERR_DENIED
    );
    let with_cred = Delivery {
        lease: live_lease(),
        credential: Some(b"{}".to_vec()),
    };
    assert_eq!(
        svc.identity("validate", &with_cred, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // Validate round-trip: inspect + broker call.
    let body = Delivery {
        lease: live_lease(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![
        ok(&inspect_json(CID, true, PID, "7", false, "private")),
        ok(&body),
    ]);
    let svc = make_service(exec);
    let out = svc
        .identity(
            "validate",
            &Delivery {
                lease: live_lease(),
                credential: None,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(out.lease, live_lease());
    assert_eq!(svc.exec.calls().len(), 2);
    // Stale container incarnation.
    let exec = FakeExec::new(vec![ok(&inspect_json(
        "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
        true,
        PID,
        "7",
        false,
        "private",
    ))]);
    let svc = make_service(exec);
    assert_eq!(
        svc.identity(
            "validate",
            &Delivery {
                lease: live_lease(),
                credential: None
            },
            deadline()
        )
        .unwrap_err(),
        ERR_STALE
    );
    // Stop on a removed container short-circuits without a broker call.
    let exec = FakeExec::new(vec![Err("exit status 1".to_string())]);
    let svc = make_service(exec);
    let out = svc
        .identity(
            "stop",
            &Delivery {
                lease: live_lease(),
                credential: None,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(out.lease, live_lease());
    assert_eq!(svc.exec.calls().len(), 1);
    // Stop on a stopped container also short-circuits.
    let exec = FakeExec::new(vec![
        ok(""),
        ok(&inspect_json(CID, false, PID, "7", false, "private")),
    ]);
    let svc = make_service(exec);
    let out = svc
        .identity(
            "stop",
            &Delivery {
                lease: live_lease(),
                credential: None,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(out.lease, live_lease());
    // Uncertain stop target.
    let exec = FakeExec::new(vec![Err("exit status 2".to_string())]);
    let svc = make_service(exec);
    assert_eq!(
        svc.identity(
            "stop",
            &Delivery {
                lease: live_lease(),
                credential: None
            },
            deadline()
        )
        .unwrap_err(),
        ERR_UNCERTAIN
    );
}

#[test]
fn managed_end_matrix() {
    let end = TerminalRequest {
        action: "end".to_string(),
        id: TID.to_string(),
        project: PID.to_string(),
        login: "dev".to_string(),
        identity: 7,
        ..Default::default()
    };
    // Non-end requests never call out.
    let exec = FakeExec::new(vec![]);
    let svc = make_service(exec);
    let mut other = end.clone();
    other.action = "attach".to_string();
    assert!(svc.managed_end(CID, &other, None, deadline()).is_ok());
    assert!(svc.exec.calls().is_empty());
    // Lookup failure propagates.
    let exec = FakeExec::new(vec![Err("boom".to_string())]);
    let svc = make_service(exec);
    assert_eq!(
        svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
        "managed Codex operation failed"
    );
    // No lease recorded: success without a callback.
    let body = Delivery::default().encode();
    let exec = FakeExec::new(vec![ok(&body)]);
    let svc = make_service(exec);
    assert!(svc.managed_end(CID, &end, None, deadline()).is_ok());
    // Binding mismatch is stale.
    let mut lease = live_lease();
    lease.binding.as_mut().unwrap().id = "ffffffffffffffffffffffffffffffff".to_string();
    let body = Delivery {
        lease,
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![ok(&body)]);
    let svc = make_service(exec);
    assert_eq!(
        svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
        ERR_STALE
    );
    // Missing callback denies; present callback runs.
    let body = Delivery {
        lease: live_lease(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![ok(&body)]);
    let svc = make_service(exec);
    assert_eq!(
        svc.managed_end(CID, &end, None, deadline()).unwrap_err(),
        ERR_DENIED
    );
    let body = Delivery {
        lease: live_lease(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![ok(&body)]);
    let svc = make_service(exec);
    let seen = Mutex::new(Vec::new());
    let end_identity = |actor: i64, lease_id: &str| {
        seen.lock().unwrap().push((actor, lease_id.to_string()));
        Ok(())
    };
    assert!(svc
        .managed_end(CID, &end, Some(&end_identity), deadline())
        .is_ok());
    assert_eq!(
        seen.lock().unwrap().as_slice(),
        &[(7, "lease-1".to_string())]
    );
    // Lookup stdin pins the request envelope.
    let calls = svc.exec.calls();
    let body_json = String::from_utf8(calls[0].0.clone()).unwrap();
    assert!(body_json.contains(r#""action":"lookup""#), "{body_json}");
    assert!(body_json.contains(r#""login":"dev""#), "{body_json}");
}

// ----- agent + harness files -----

fn euid() -> u32 {
    unsafe { libc::geteuid() }
}

fn with_agent_env(path: &std::path::Path) -> MutexGuard<'static, ()> {
    let guard = ENV_LOCK.lock().unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", path) };
    guard
}

#[test]
fn agent_program_hash_matrix() {
    let dir = test_tmp("agent");
    let path = dir.join("project-terminal");
    std::fs::write(&path, b"agent-bytes").unwrap();
    let _guard = with_agent_env(&path);
    assert_eq!(agent_program_path(), path.to_str().unwrap().to_string());
    // Happy path under the test uid.
    assert_eq!(
        agent_program_hash(euid()).unwrap(),
        sha256::hex_lower(&sha256::digest(b"agent-bytes"))
    );
    // Ownership gate.
    assert_eq!(
        agent_program_hash(euid().wrapping_add(1)).unwrap_err(),
        "project terminal agent has unexpected ownership"
    );
    // Writable gate: 0o644 passes (no write bits outside owner).
    use std::os::unix::fs::PermissionsExt;
    assert!(agent_program_hash(euid()).is_ok());
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o664)).unwrap();
    assert_eq!(
        agent_program_hash(euid()).unwrap_err(),
        "project terminal agent is group- or world-writable"
    );
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o755)).unwrap();
    assert!(agent_program_hash(euid()).is_ok());
    // Symlinks and directories are not regular files.
    let link = dir.join("link");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &link) };
    assert_eq!(
        agent_program_hash(euid()).unwrap_err(),
        "project terminal agent is not a regular file"
    );
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &dir) };
    assert_eq!(
        agent_program_hash(euid()).unwrap_err(),
        "project terminal agent is not a regular file"
    );
    // Missing file.
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", dir.join("absent")) };
    assert!(
        agent_program_hash(euid())
            .unwrap_err()
            .starts_with("project terminal agent unavailable: "),
        "missing file must fail with unavailable"
    );
    // Empty file.
    let empty = dir.join("empty");
    std::fs::write(&empty, b"").unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &empty) };
    assert_eq!(
        agent_program_hash(euid()).unwrap_err(),
        "project terminal agent has unexpected size"
    );
    // Oversize file (sparse).
    let big = dir.join("big");
    let f = std::fs::File::create(&big).unwrap();
    f.set_len((32 << 20) + 1).unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &big) };
    assert_eq!(
        agent_program_hash(euid()).unwrap_err(),
        "project terminal agent has unexpected size"
    );
    unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
}

fn write_harness(dir: &std::path::Path, bytes: &[u8], mode: u32) -> String {
    let bin = dir.join("bin");
    std::fs::create_dir_all(&bin).unwrap();
    let path = bin.join("codex");
    std::fs::write(&path, bytes).unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(mode)).unwrap();
    sha256::hex_lower(&sha256::digest(bytes))
}

#[test]
fn verify_identity_harness_matrix() {
    let dir = test_tmp("harness");
    let digest = write_harness(&dir, b"codex-bytes", 0o755);
    let mut svc = make_service(FakeExec::new(vec![]));
    svc.codex_harness = dir.to_str().unwrap().to_string();
    svc.codex_harness_sha256 = digest;
    assert!(svc.verify_identity_harness().is_ok());
    svc.codex_harness_sha256 = "0".repeat(64);
    assert_eq!(
        svc.verify_identity_harness().unwrap_err(),
        "codex harness digest differs"
    );
    svc.codex_harness = dir.join("absent").to_str().unwrap().to_string();
    assert_eq!(
        svc.verify_identity_harness().unwrap_err(),
        "verified Codex executable required"
    );
    let dir2 = test_tmp("harness-noexec");
    write_harness(&dir2, b"codex-bytes", 0o644);
    svc.codex_harness = dir2.to_str().unwrap().to_string();
    assert_eq!(
        svc.verify_identity_harness().unwrap_err(),
        "verified Codex executable required"
    );
}

#[test]
fn stream_identity_harness_flows() {
    let svc = make_service(FakeExec::new(vec![Err("no tar".to_string())]));
    assert_eq!(
        svc.stream_identity_harness(CID, "/run/x", deadline())
            .unwrap_err(),
        "codex harness stream unavailable"
    );
    let svc = make_service(FakeExec::new(vec![
        ok("tar-bytes"),
        Err("no podman".to_string()),
    ]));
    assert_eq!(
        svc.stream_identity_harness(CID, "/run/x", deadline())
            .unwrap_err(),
        "codex harness staging failed"
    );
    let svc = make_service(FakeExec::new(vec![ok("tar-bytes"), ok("")]));
    assert!(svc
        .stream_identity_harness(CID, "/run/x", deadline())
        .is_ok());
    let calls = svc.exec.calls();
    assert_eq!(calls[0].1, "/usr/bin/tar");
    assert_eq!(calls[0].2, tar_producer_argv("/opt/harness"));
    assert_eq!(calls[1].1, "/usr/bin/podman");
    assert_eq!(calls[1].2, tar_consumer_argv(CID, "/run/x"));
    assert_eq!(calls[1].0, b"tar-bytes");
}

fn preparing_lease() -> Lease {
    // Binding-free lease with a live 12h deadline window.
    Lease {
        provider_id: "codex".to_string(),
        id: "lease-9".to_string(),
        connection_id: "conn".to_string(),
        generation: 1,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: TID.to_string(),
        kind: KIND_TERMINAL.to_string(),
        deadline_raw: "live-window".to_string(),
        deadline: Some((now_unix() + 3600, 0)),
        ..Default::default()
    }
}

#[test]
fn prepare_identity_flows() {
    // Denied input never calls out.
    let exec = FakeExec::new(vec![]);
    let svc = make_service(exec);
    assert_eq!(
        svc.prepare_identity(&Lease::default(), "dev", CID, 80, 24, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
    // The prepare path verifies the production agent binary (uid 0):
    // as non-root the ownership gate fires after the inspect call.
    let _guard = ENV_LOCK.lock().unwrap();
    let dir = test_tmp("prep-agent");
    let path = dir.join("project-terminal");
    std::fs::write(&path, b"agent-bytes").unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &path) };
    let exec = FakeExec::new(vec![ok(&inspect_json(
        CID, true, PID, "7", false, "private",
    ))]);
    let svc = make_service(exec);
    if euid() == 0 {
        let prepared = Delivery {
            lease: live_lease(),
            credential: None,
        }
        .encode();
        let exec = FakeExec::new(vec![
            ok(&inspect_json(CID, true, PID, "7", false, "private")),
            ok(&prepared),
        ]);
        let svc = make_service(exec);
        let mut lease = preparing_lease();
        lease.id = "lease-1".to_string();
        let binding = svc
            .prepare_identity(&lease, "dev", CID, 80, 24, deadline())
            .unwrap();
        assert_eq!(binding, live_lease().binding.unwrap());
        let calls = svc.exec.calls();
        assert_eq!(calls.len(), 2);
        let stdin = String::from_utf8(calls[1].0.clone()).unwrap();
        assert!(stdin.contains(r#""action":"prepare""#), "{stdin}");
        assert!(stdin.contains(&format!("\"container\":{CID:?}")), "{stdin}");
    } else {
        assert_eq!(
            svc.prepare_identity(&preparing_lease(), "dev", CID, 80, 24, deadline())
                .unwrap_err(),
            "project terminal agent has unexpected ownership"
        );
        assert_eq!(svc.exec.calls().len(), 1);
    }
    unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
}

#[test]
fn identity_start_flow() {
    // Full start sequence: inspect, stage broker call, tar producer,
    // tar consumer, start broker call.
    let dir = test_tmp("start-harness");
    let digest = write_harness(&dir, b"codex-bytes", 0o755);
    let lease = live_lease();
    let delivery = Delivery {
        lease: lease.clone(),
        credential: Some(b"{}".to_vec()),
    };
    let stage_out = Delivery {
        lease: lease.clone(),
        credential: None,
    }
    .encode();
    let start_out = Delivery {
        lease: lease.clone(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![
        ok(&inspect_json(CID, true, PID, "7", false, "private")),
        ok(&stage_out),
        ok("tar-bytes"),
        ok(""),
        ok(&start_out),
    ]);
    let mut svc = make_service(exec);
    svc.codex_harness = dir.to_str().unwrap().to_string();
    svc.codex_harness_sha256 = digest;
    let out = svc.identity("start", &delivery, deadline()).unwrap();
    assert_eq!(out.lease, lease);
    let calls = svc.exec.calls();
    assert_eq!(calls.len(), 5);
    assert_eq!(calls[0].1, "/usr/bin/podman");
    assert_eq!(calls[1].1, "/usr/bin/podman");
    assert_eq!(calls[2].1, "/usr/bin/tar");
    assert_eq!(calls[3].1, "/usr/bin/podman");
    assert_eq!(
        calls[3].2,
        tar_consumer_argv(CID, &format!("/run/soda-terminals/{TID}/model/harness"))
    );
    let stage_stdin = String::from_utf8(calls[1].0.clone()).unwrap();
    assert!(stage_stdin.contains(r#""action":"stage""#), "{stage_stdin}");
    let start_stdin = String::from_utf8(calls[4].0.clone()).unwrap();
    assert!(start_stdin.contains(r#""action":"start""#), "{start_stdin}");
    assert!(
        start_stdin.contains(&format!(
            "\"harness_sha256\":{:?}",
            svc.codex_harness_sha256
        )),
        "{start_stdin}"
    );
}

#[test]
fn private_request_matrix() {
    assert!(valid_identity_request("POST", "", false, "", 0));
    assert!(!valid_identity_request("GET", "", false, "", 0));
    assert!(!valid_identity_request("POST", "x", false, "", 0));
    assert_eq!(identity_action("/identity/validate"), "validate");
    assert_eq!(identity_action("/identity/launch"), "launch");
    assert_eq!(identity_action("/other"), "/other");
}

#[test]
fn terminal_start_decode() {
    let body = br#"{"connection_id":"c","project_id":"p0123456789abcdef01234567","actor_id":"7","login":"dev","scope":"s","cols":80,"rows":24}"#;
    let start = TerminalStart::decode(body).unwrap();
    assert_eq!(start.actor_id, 7);
    assert_eq!(start.cols, 80);
    let negative_zero = TerminalStart::decode(br#"{"cols":-0,"rows":-0}"#).unwrap();
    assert_eq!((negative_zero.cols, negative_zero.rows), (0, 0));
    assert!(TerminalStart::decode(br#"{"cols":1e0}"#).is_err());
    assert!(TerminalStart::decode(br#"{"actor_id":""}"#).is_err());
    assert!(TerminalStart::decode(br#"{"actor_id":7}"#).is_err());
    assert_eq!(TerminalStart::decode(br#"{}"#).unwrap().actor_id, 0);
    assert!(TerminalStart::decode(br#"{"bogus":1}"#).is_err());
}

#[test]
fn zero_deadline_encode_pin() {
    // Go marshals the zero time, never an empty string (probed).
    let encoded = Lease::default().encode();
    assert!(
        encoded.contains(r#""deadline":"0001-01-01T00:00:00Z""#),
        "{encoded}"
    );
    let round = Delivery::decode(format!("{{\"lease\":{encoded},\"credential\":null}}").as_bytes())
        .unwrap();
    assert_eq!(round.lease.deadline_raw, "0001-01-01T00:00:00Z");
    assert_eq!(round.lease.deadline, parse_rfc3339("0001-01-01T00:00:00Z"));
}

#[test]
fn output_line_and_attach_pins() {
    let f = parse_output_line(br#"{"type":"output","data":"aGk="}"#).unwrap();
    assert_eq!(f.frame_type, "output");
    assert_eq!(
        parse_output_line(br#"{"type":"input","data":"aGk="}"#).unwrap_err(),
        "invalid terminal response"
    );
    assert_eq!(
        parse_output_line(b"nope").unwrap_err(),
        "invalid terminal response"
    );
    // Attach pre-checks fire before any spawn.
    assert_eq!(
        NativeAttach::attach("short", &base_request()).unwrap_err(),
        "invalid terminal target"
    );
    let mut expired = base_request();
    expired.expires = now_unix() - 1;
    assert_eq!(
        NativeAttach::attach(CID, &expired).unwrap_err(),
        "invalid terminal target"
    );
}

#[test]
fn rand_id_shape() {
    let a = rand_id().unwrap();
    let b = rand_id().unwrap();
    assert_eq!(a.len(), 32);
    assert!(a.bytes().all(|c| c.is_ascii_hexdigit()));
    assert_ne!(a, b);
}

// ----- daemon dispatch -----

struct FakeBroker {
    acquire_result: Mutex<Result<Lease, String>>,
    register_result: Mutex<Result<Delivery, String>>,
    reconciled: Mutex<Vec<String>>,
    acquires: Mutex<Vec<AcquireRequest>>,
}

impl IdentityBroker for FakeBroker {
    fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, String> {
        self.acquires.lock().unwrap().push(req.clone());
        self.acquire_result.lock().unwrap().clone()
    }
    fn register(
        &self,
        _lease_id: &str,
        _binding: &Binding,
        _deadline: Instant,
    ) -> Result<Delivery, String> {
        self.register_result.lock().unwrap().clone()
    }
    fn reconcile_lease(&self, lease_id: &str) -> Result<(), String> {
        self.reconciled.lock().unwrap().push(lease_id.to_string());
        Ok(())
    }
}

fn launch_input() -> TerminalStart {
    TerminalStart {
        connection_id: "conn".to_string(),
        project_id: PID.to_string(),
        actor_id: 7,
        login: "dev".to_string(),
        scope: CID.to_string(),
        cols: 80,
        rows: 24,
    }
}

#[test]
fn identity_launch_flows() {
    let input = launch_input();
    // Unconfigured harness denies before acquiring.
    let broker = FakeBroker {
        acquire_result: Mutex::new(Err("unreachable".to_string())),
        register_result: Mutex::new(Err("unreachable".to_string())),
        reconciled: Mutex::new(Vec::new()),
        acquires: Mutex::new(Vec::new()),
    };
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        identity_launch(&broker, &svc, &input, false, deadline()).unwrap_err(),
        ERR_DENIED
    );
    assert!(broker.acquires.lock().unwrap().is_empty());
    // Acquire failure: no reconcile.
    let broker = FakeBroker {
        acquire_result: Mutex::new(Err("broker down".to_string())),
        register_result: Mutex::new(Err("unreachable".to_string())),
        reconciled: Mutex::new(Vec::new()),
        acquires: Mutex::new(Vec::new()),
    };
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
        "broker down"
    );
    assert!(broker.reconciled.lock().unwrap().is_empty());
    let acquire = broker.acquires.lock().unwrap()[0].clone();
    assert_eq!(acquire.provider_id, "codex");
    assert_eq!(acquire.actor_id, 7);
    assert_eq!(acquire.kind, "terminal");
    assert_eq!(acquire.execution_id.len(), 32);
    assert!((acquire.deadline_secs - (now_unix() + 12 * 3600)).abs() <= 5);
    // Prepare failure reconciles.
    let broker = FakeBroker {
        acquire_result: Mutex::new(Ok(preparing_lease())),
        register_result: Mutex::new(Err("unreachable".to_string())),
        reconciled: Mutex::new(Vec::new()),
        acquires: Mutex::new(Vec::new()),
    };
    let mut bad = preparing_lease();
    bad.project_id = "nope".to_string();
    *broker.acquire_result.lock().unwrap() = Ok(bad);
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        broker.reconciled.lock().unwrap().as_slice(),
        &["lease-9".to_string()]
    );
    // Register failure reconciles (prepare succeeds only as root; as
    // non-root the agent ownership gate fires first and still reconciles).
    let _guard = ENV_LOCK.lock().unwrap();
    let dir = test_tmp("launch-agent");
    let path = dir.join("project-terminal");
    std::fs::write(&path, b"agent-bytes").unwrap();
    unsafe { std::env::set_var("SODA_PROJECT_TERMINAL", &path) };
    let broker = FakeBroker {
        acquire_result: Mutex::new(Ok(preparing_lease())),
        register_result: Mutex::new(Err("register down".to_string())),
        reconciled: Mutex::new(Vec::new()),
        acquires: Mutex::new(Vec::new()),
    };
    let prepared = Delivery {
        lease: live_lease(),
        credential: None,
    }
    .encode();
    let exec = FakeExec::new(vec![
        ok(&inspect_json(CID, true, PID, "7", false, "private")),
        ok(&prepared),
    ]);
    let svc = make_service(exec);
    if euid() == 0 {
        // prepare would still fail: lease id differs from prepared id.
        assert_eq!(
            identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
            "managed terminal reservation differs"
        );
    } else {
        assert_eq!(
            identity_launch(&broker, &svc, &input, true, deadline()).unwrap_err(),
            "project terminal agent has unexpected ownership"
        );
    }
    assert_eq!(
        broker.reconciled.lock().unwrap().as_slice(),
        &["lease-9".to_string()]
    );
    unsafe { std::env::remove_var("SODA_PROJECT_TERMINAL") };
}

#[test]
fn identity_route_matrix() {
    let factory = Lease {
        kind: KIND_FACTORY.to_string(),
        ..Default::default()
    };
    assert_eq!(
        identity_route("/identity/launch", &factory, true),
        IdentityRoute::Launch
    );
    assert_eq!(
        identity_route("/identity/validate", &factory, true),
        IdentityRoute::Factory
    );
    let mut muse = live_lease();
    muse.provider_id = "muse".to_string();
    muse.binding.as_mut().unwrap().scope = SCOPE_MUSE_PROJECT.to_string();
    assert_eq!(
        identity_route("/identity/stop", &muse, true),
        IdentityRoute::Muse
    );
    assert_eq!(
        identity_route("/identity/stop", &muse, false),
        IdentityRoute::Terminal
    );
    muse.binding.as_mut().unwrap().scope = "other".to_string();
    assert_eq!(
        identity_route("/identity/stop", &muse, true),
        IdentityRoute::Terminal
    );
    assert_eq!(
        identity_route("/identity/finish", &live_lease(), true),
        IdentityRoute::Terminal
    );
}

fn piped_file() -> (std::os::unix::net::UnixStream, File) {
    use std::os::unix::io::{FromRawFd, IntoRawFd};
    use std::os::unix::net::UnixStream;
    let (peer, end) = UnixStream::pair().unwrap();
    // SAFETY: the fd is owned by the new File exactly once.
    let file = unsafe { File::from_raw_fd(end.into_raw_fd()) };
    (peer, file)
}

#[test]
fn take_reader_detaches_output() {
    let (out_peer, out_file) = piped_file();
    let (_in_peer, in_file) = piped_file();
    let mut attach = NativeAttach {
        child: None,
        stdin: Some(in_file),
        reader: Some(BufReader::new(out_file)),
        closed: false,
    };
    let mut reader = attach.take_reader().expect("reader detached");
    assert!(attach.take_reader().is_none());
    std::io::Write::write_all(&mut &out_peer, b"{\"type\":\"ready\"}\n").unwrap();
    let frame = NativeAttach::output_frame(&mut reader).unwrap();
    assert_eq!(frame.frame_type, "ready");
    drop(out_peer);
    assert_eq!(
        NativeAttach::output_frame(&mut reader).unwrap_err(),
        "terminal output ended"
    );
    attach.close();
    assert!(attach.closed);
}

#[test]
fn quiet_output_never_blocks_input() {
    let (out_peer, out_file) = piped_file();
    let (_in_peer, in_file) = piped_file();
    let mut attach = NativeAttach {
        child: None,
        stdin: Some(in_file),
        reader: Some(BufReader::new(out_file)),
        closed: false,
    };
    let mut reader = attach.take_reader().unwrap();
    let out = std::thread::spawn(move || NativeAttach::output_frame(&mut reader));
    // Child quiet (peer open, no data): input still flows.
    let frame = TerminalFrame {
        frame_type: "input".to_string(),
        data: "eA==".to_string(),
        cols: 0,
        rows: 0,
        reason: String::new(),
        terminals: None,
    };
    attach.input_frame(&frame).unwrap();
    drop(out_peer);
    assert_eq!(out.join().unwrap().unwrap_err(), "terminal output ended");
    attach.close();
}

/// Spawn a real quiet child owned by a test `NativeAttach` (reader
/// detached, stdin open): the task-owned `Some(child)` close path.
/// Also returns the PID plus its `/proc` starttime identity, captured
/// while the child is known alive, so the reap assertion can tell our
/// entry from a recycled PID.
fn attach_with_child(argv: &[&str]) -> (NativeAttach, u32, u64) {
    use std::os::unix::io::FromRawFd;
    let mut child = std::process::Command::new(argv[0])
        .args(&argv[1..])
        .stdin(std::process::Stdio::piped())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null())
        .spawn()
        .unwrap();
    let pid = child.id();
    let (_, starttime) = proc_stat(pid).expect("spawned child must have a /proc entry");
    let stdin = child.stdin.take().unwrap();
    // Same ownership transfer as `attach`: exactly-once raw fds.
    let fd = stdin.as_raw_fd();
    std::mem::forget(stdin);
    let stdin = unsafe { File::from_raw_fd(fd) };
    (
        NativeAttach {
            child: Some(child),
            stdin: Some(stdin),
            reader: None,
            closed: false,
        },
        pid,
        starttime,
    )
}

/// `/proc/<pid>/stat` (state, starttime), or `None` once the PID is
/// reaped and gone. Starttime (field 22) identifies the process
/// across PID recycling: a zombie holds its PID, so a live entry
/// with a different starttime proves our child was reaped.
fn proc_stat(pid: u32) -> Option<(char, u64)> {
    let text = std::fs::read_to_string(format!("/proc/{pid}/stat")).ok()?;
    let close = text.rfind(')')?;
    let mut fields = text[close + 2..].split_whitespace();
    let state = fields.next()?.chars().next()?;
    let starttime: u64 = fields.nth(18)?.parse().ok()?;
    Some((state, starttime))
}

/// Assert the child was reaped (STEER-A-007-1: no timing-dependent
/// null-signal check — a recycled PID also answers it, and our own
/// dying child is still alive in the instant after kill). Every
/// branch is decided by starttime identity, not timing: gone, or an
/// entry with a foreign starttime, proves reap; only our own entry
/// persisting to the deadline fails.
fn assert_pid_reaped(pid: u32, our_start: u64) {
    let deadline = Instant::now() + Duration::from_secs(2);
    loop {
        match proc_stat(pid) {
            None => return,
            Some((_, start)) if start != our_start => return,
            Some((state, _)) => {
                if Instant::now() >= deadline {
                    panic!("pid {pid} still ours (state {state}): child unreaped");
                }
                std::thread::sleep(Duration::from_millis(50));
            }
        }
    }
}

#[test]
fn close_reaps_eof_exited_child() {
    // CODEX-H01-REAP-1: `cat` exits on the stdin EOF that `close`
    // causes by dropping stdin; the exit must be reaped.
    let (mut attach, pid, starttime) = attach_with_child(&["cat"]);
    attach.close();
    assert!(attach.child.is_none());
    assert_pid_reaped(pid, starttime);
    // Once-only close stays idempotent.
    attach.close();
}

#[test]
fn close_reaps_killed_child() {
    // CODEX-H01-REAP-1: `sleep` ignores stdin EOF, so `close` must
    // kill after the 3s grace and then reap the forced termination.
    let (mut attach, pid, starttime) = attach_with_child(&["sleep", "30"]);
    let start = Instant::now();
    attach.close();
    assert!(start.elapsed() >= Duration::from_secs(3), "grace skipped");
    assert!(attach.child.is_none());
    assert_pid_reaped(pid, starttime);
    attach.close();
}
