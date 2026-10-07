//! Shared prepare fixtures (one private fixture owner) plus
//! paths/idmap/single-line/tool/map-state/container admission cases.

use super::*;
use crate::preparation::Preparation;
use crate::project::Config;
use std::cell::RefCell;
use std::collections::VecDeque;

pub(super) const DIGEST: &str = "32c794ef2201b76b757bfba2c23bba06dcc5a8c6121f6fabc99115ef12043ced";
pub(super) const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
const EFFECTS: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

type MockCall = (Vec<u8>, String, Vec<String>);

pub(super) struct Mock {
    pub(super) calls: RefCell<Vec<MockCall>>,
    script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
}

impl Mock {
    pub(super) fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
        Mock {
            calls: RefCell::new(Vec::new()),
            script: RefCell::new(responses.into()),
        }
    }
}

impl Executor for Mock {
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

pub(super) fn deadline() -> Instant {
    Instant::now() + std::time::Duration::from_secs(5)
}

pub(super) fn test_config() -> Config {
    Config {
        muse_socket: String::new(),
        image: "img".to_string(),
        network: "sodanet".to_string(),
        subnet: "10.0.0.0/24".to_string(),
        bridge: "sodabr".to_string(),
    }
}

pub(super) fn pid() -> String {
    format!("p{}", "a".repeat(24))
}

pub(super) fn fid() -> String {
    format!("f{}", "b".repeat(24))
}

pub(super) fn container_payload(id: &str) -> Vec<u8> {
    let cid = "f".repeat(64);
    format!(
        "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
    )
    .into_bytes()
}

pub(super) fn observation(phase: &str, role: &str, ready: bool, stopped: bool) -> Vec<u8> {
    format!(
        "{{\"hold\":{{\"active\":false,\"revision\":0}},\"known\":true,\
         \"phase\":{phase:?},\"role\":{role:?},\"setup_digest\":{DIGEST:?},\
         \"source_commit\":{COMMIT:?},\"tools\":[],\
         \"verified\":{{\"uid\":\"1001\",\"login\":{role:?},\"groups\":{role:?},\
         \"refusal\":\"\"}},\"missing\":\"\",\"stopped\":{stopped},\"ready\":{ready},\
         \"setup_exit\":null,\"check_exit\":null,\"setup_log\":\"\",\"check_log\":\"\"}}"
    )
    .into_bytes()
}

pub(super) fn preparation() -> Preparation {
    Preparation {
        id: fid(),
        project: pid(),
        role: "soda-coder".to_string(),
        revision: 0,
        requirements: preparation::RequirementAcceptance {
            id: format!("d{:024x}", 1),
            revision: 0,
            approver: 1,
            source_commit: COMMIT.to_string(),
            digest: EFFECTS.to_string(),
        },
        approval: preparation::AdminApproval {
            id: format!("d{:024x}", 2),
            revision: 0,
            approver: 1,
            effects_digest: EFFECTS.to_string(),
        },
        source_commit: COMMIT.to_string(),
        setup_digest: DIGEST.to_string(),
        tools: vec![],
        credential: String::new(),
    }
}

pub(super) fn setup() -> preparation::ApprovedSetup {
    preparation::ApprovedSetup {
        files: [
            ("setup.sh".to_string(), b"#!/bin/sh\n".to_vec()),
            ("check.sh".to_string(), b"#!/bin/sh\n".to_vec()),
        ]
        .into_iter()
        .collect(),
        bundle: b"BUNDLE".to_vec(),
    }
}

pub(super) fn approve_response(fid: &str) -> Vec<u8> {
    format!(
        "{{\"approved\":{fid:?},\"repeated\":false,\
         \"checkout\":\"/home/soda-coder/checkouts/{fid}\",\"credential_file\":\"\"}}"
    )
    .into_bytes()
}

#[test]
fn path_join_uses_native_components_and_tool_admission_is_strict() {
    assert_eq!(path_join(&["a", "b", "c"]), "a/b/c");
    assert_eq!(path_join(&["/a", "b", "c"]), "/a/b/c");
    assert_eq!(path_join(&[]), "");
    assert_eq!(path_join(&[""]), "");
    assert_eq!(path_join(&["", "a", "b"]), "a/b");
    assert_eq!(path_join(&["a", "", "b"]), "a/b");
    assert_eq!(path_join(&["", ""]), "");
    assert!(valid_resolved_tool_path("/usr/bin/git"));
    assert!(!valid_resolved_tool_path("/usr/bin/../bin/git"));
    assert!(!valid_resolved_tool_path("/usr//bin/git"));
    assert!(!valid_resolved_tool_path("/usr/bin/./git"));
}

#[test]
fn idmap_requires_shifted_usable_range() {
    assert!(prepare_id_map(&["0:1000000:262144".to_string()]));
    assert!(prepare_id_map(&[
        "1:1:1".to_string(),
        "0:1000000:262144".to_string()
    ]));
    assert!(!prepare_id_map(&[]));
    assert!(!prepare_id_map(&["1:1000000:262144".to_string()]));
    assert!(!prepare_id_map(&["0:1000000:100".to_string()]));
    assert!(!prepare_id_map(&["0:0:262144".to_string()]));
    assert!(!prepare_id_map(&["0:01:262144".to_string()]));
    assert!(!prepare_id_map(&["0:+1:262144".to_string()]));
    assert!(!prepare_id_map(&["0:1".to_string()]));
    assert!(!prepare_id_map(&["0:1:2:3".to_string()]));
    // base + size must fit uint32: 4294705151 + 262144 = max exactly.
    assert!(prepare_id_map(&["0:4294705151:262144".to_string()]));
    assert!(!prepare_id_map(&["0:4294705152:262144".to_string()]));
    assert!(!prepare_id_map(&["0:4294967295:262144".to_string()]));
    // Every entry must parse.
    assert!(!prepare_id_map(&[
        "0:1000000:262144".to_string(),
        "bogus".to_string()
    ]));
}

#[test]
fn single_line_rules() {
    assert_eq!(single_line(b"abc\n", 64).as_deref(), Some("abc"));
    assert_eq!(single_line(b"  abc  ", 64).as_deref(), Some("abc"));
    assert_eq!(single_line(b"a\rb", 64).as_deref(), Some("a\rb"));
    assert_eq!(single_line(b"", 64), None);
    assert_eq!(single_line(b"   ", 64), None);
    assert_eq!(single_line(b"a\nb", 64), None);
    assert_eq!(single_line(b"abc", 2), None);
    assert_eq!(single_line(b"\xff\xfe", 64), None);
}

#[test]
fn tool_paths_must_be_clean_and_pinned() {
    assert!(valid_resolved_tool_path("/usr/bin/git"));
    assert!(valid_resolved_tool_path("/usr/local/bin/tool"));
    assert!(!valid_resolved_tool_path(""));
    assert!(!valid_resolved_tool_path("git"));
    assert!(!valid_resolved_tool_path("/usr/bin/../bin/git"));
    assert!(!valid_resolved_tool_path("/bin/git"));
    assert!(!valid_resolved_tool_path("/usr/bin/"));
    assert!(!valid_resolved_tool_path(&format!(
        "/usr/bin/{}",
        "a".repeat(250)
    )));
}

#[test]
fn preparation_paths_layout() {
    let (checkout, snapshot, bundle, home) = preparation_paths("soda-coder", "fid");
    assert_eq!(checkout, "/home/soda-coder/checkouts/fid");
    assert_eq!(snapshot, "/var/lib/soda/factory/preparations/fid/snapshot");
    assert_eq!(
        bundle,
        "/var/lib/soda/factory/preparations/fid/snapshot/source.bundle"
    );
    assert_eq!(home, "/home/soda-coder/checkouts/fid/.soda-home");
}

#[test]
fn map_state_validates_and_assembles() {
    let raw = observation("ready", "soda-coder", true, false);
    let state = map_preparation_state("cid", &raw).unwrap();
    assert_eq!(state.container, "cid");
    assert!(state.ready);
    assert!(state.id.is_empty()); // caller fills identity
                                  // Ready must agree with the phase.
    let raw = observation("running", "soda-coder", true, false);
    assert_eq!(
        map_preparation_state("cid", &raw).unwrap_err(),
        "preparation observation is inconsistent"
    );
    let raw = observation("ready", "soda-coder", false, false);
    assert_eq!(
        map_preparation_state("cid", &raw).unwrap_err(),
        "preparation observation is inconsistent"
    );
    // Unknown phase / role / digest / commit.
    let mut bad = String::from_utf8(observation("nope", "soda-coder", false, false)).unwrap();
    assert_eq!(
        map_preparation_state("cid", bad.as_bytes()).unwrap_err(),
        "invalid preparation observation"
    );
    bad = bad.replace("\"soda-coder\"", "\"coder\"");
    assert_eq!(
        map_preparation_state("cid", bad.as_bytes()).unwrap_err(),
        "invalid preparation observation"
    );
    // Oversized logs.
    let big = "x".repeat(66561);
    let raw = format!(
        "{{\"known\":true,\"phase\":\"running\",\"role\":\"soda-coder\",\
         \"setup_digest\":{DIGEST:?},\"source_commit\":{COMMIT:?},\
         \"setup_log\":{big:?}}}"
    );
    assert_eq!(
        map_preparation_state("cid", raw.as_bytes()).unwrap_err(),
        "preparation observation exceeds the bounded size"
    );
    // Log assembly, tools, exits.
    let raw = format!(
        "{{\"known\":true,\"phase\":\"running\",\"role\":\"soda-coder\",\
         \"setup_digest\":{DIGEST:?},\"source_commit\":{COMMIT:?},\
         \"tools\":[{{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"v\"}}],\
         \"missing\":\"go\",\"setup_exit\":0,\"check_exit\":3,\
         \"setup_log\":\"s\",\"check_log\":\"c\"}}"
    );
    let state = map_preparation_state("cid", raw.as_bytes()).unwrap();
    assert_eq!(state.output, "--- setup ---\ns\n--- check ---\nc");
    assert_eq!(state.tools.len(), 1);
    assert_eq!(state.setup_exit, Some(0));
    assert_eq!(state.check_exit, Some(3));
    assert_eq!(state.missing, "go");
}

#[test]
fn prepare_container_binds_isolated_target() {
    let id = pid();
    let mock = Mock::new(vec![Ok(container_payload(&id))]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let cid = rt.prepare_container(&id, true, deadline()).unwrap();
    assert_eq!(cid, "f".repeat(64));
    let calls = mock.calls.borrow();
    assert_eq!(calls[0].1, "/usr/bin/podman");
    assert_eq!(
        calls[0].2[..4],
        [
            "--remote=false".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            crate::project::PROJECT_INSPECT_FORMAT.to_string(),
        ]
    );
    assert_eq!(calls[0].2[4], format!("soda-{id}"));
    // Bad project id never execs.
    assert_eq!(
        rt.prepare_container("x", true, deadline()).unwrap_err(),
        "invalid project"
    );
    // Podman failure and oversized output.
    let mock = Mock::new(vec![Err("boom".to_string())]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_container(&id, true, deadline()).unwrap_err(),
        "preparation target unavailable"
    );
    let mock = Mock::new(vec![Ok(vec![b'x'; 4097])]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_container(&id, true, deadline()).unwrap_err(),
        "preparation target unavailable"
    );
    // Unparseable and not-ready targets.
    let mock = Mock::new(vec![Ok(b"{}".to_vec())]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_container(&id, true, deadline()).unwrap_err(),
        "preparation target not ready or isolated"
    );
}
