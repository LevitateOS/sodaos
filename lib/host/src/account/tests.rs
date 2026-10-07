use super::*;
use crate::project::Config;
use std::cell::RefCell;
use std::collections::VecDeque;

pub(super) const ED: &str =
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";

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
    format!("p{}", "e".repeat(24))
}

fn inspect_payload(id: &str, running: bool) -> Vec<u8> {
    let mut s = String::from("[{\"Image\":\"\",\"Config\":{\"Labels\":{\"org.soda.project\":");
    s.push_str(&crate::json::quote(id));
    s.push_str(",\"org.soda.owner\":\"42\"}},\"State\":{\"Running\":");
    s.push_str(if running { "true" } else { "false" });
    s.push_str(",\"NetworkSettings\":{\"Networks\":{\"sodanet\":{\"IPAddress\":\"10.0.0.5\"}}}}}]");
    s.into_bytes()
}

pub(super) fn container_payload(id: &str) -> Vec<u8> {
    let cid = "f".repeat(64);
    format!(
        "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
    )
    .into_bytes()
}

#[test]
fn key_revision_shape() {
    assert!(valid_key_revision(&"a".repeat(64)));
    assert!(!valid_key_revision(&"a".repeat(63)));
    assert!(!valid_key_revision(&"A".repeat(64)));
    assert!(!valid_key_revision(""));
}

#[test]
fn account_keys_drop_comments_but_keep_newline() {
    let out = canonicalize_account_keys(&[format!("{ED} alice@host"), ED.to_string()]).unwrap();
    assert_eq!(out, vec![format!("{ED}\n"), format!("{ED}\n")]);
    // Options rejected.
    assert_eq!(
        canonicalize_account_keys(&[format!("restrict {ED}")]).unwrap_err(),
        "invalid public key"
    );
    // Garbage rejected.
    assert_eq!(
        canonicalize_account_keys(&["not-a-key".to_string()]).unwrap_err(),
        "invalid public key"
    );
    // Blank and comment lines around the key are skipped.
    let out = canonicalize_account_keys(&[format!("# hello\n\n{ED}\n")]).unwrap();
    assert_eq!(out, vec![format!("{ED}\n")]);
    // Trailing non-blank data after the key line is rejected.
    assert_eq!(
        canonicalize_account_keys(&[format!("{ED}\n{ED}")]).unwrap_err(),
        "invalid public key"
    );
}

#[test]
fn development_keys_require_canonical_unique_lines() {
    assert_eq!(
        canonical_keys(&[ED.to_string()]).unwrap(),
        vec![ED.to_string()]
    );
    // Comments make the line noncanonical (parsed, then mismatch).
    assert_eq!(
        canonical_keys(&[format!("{ED} alice")]).unwrap_err(),
        "noncanonical or duplicate development key"
    );
    // Duplicates rejected.
    assert_eq!(
        canonical_keys(&[ED.to_string(), ED.to_string()]).unwrap_err(),
        "noncanonical or duplicate development key"
    );
    // Options rejected.
    assert_eq!(
        canonical_keys(&[format!("restrict {ED}")]).unwrap_err(),
        "invalid development key"
    );
    // Empty candidate options (`,`) carry no options, so the line parses
    // but is noncanonical (like Go: never "invalid development key").
    assert_eq!(
        canonical_keys(&[format!(", {ED}")]).unwrap_err(),
        "noncanonical or duplicate development key"
    );
    // Too many keys.
    let many = vec![ED.to_string(); 33];
    assert_eq!(
        canonical_keys(&many).unwrap_err(),
        "too many development keys"
    );
    // Key set too large: 32 distinct synthetic RSA keys with 1500-byte moduli.
    let mut big = Vec::new();
    for i in 0u32..32 {
        let mut n = vec![0x42u8; 1500];
        n[0] = 0x01;
        n[1496..].copy_from_slice(&i.to_be_bytes());
        let mut blob = Vec::new();
        blob.extend_from_slice(&(7u32.to_be_bytes()));
        blob.extend_from_slice(b"ssh-rsa");
        blob.extend_from_slice(&3u32.to_be_bytes());
        blob.extend_from_slice(&[0x01, 0x00, 0x01]);
        blob.extend_from_slice(&(1500u32.to_be_bytes()));
        blob.extend_from_slice(&n);
        big.push(format!("ssh-rsa {}", ssh::b64_encode(&blob)));
    }
    assert_eq!(
        canonical_keys(&big).unwrap_err(),
        "development key set too large"
    );
}

#[test]
fn access_key_state_decode_uses_plain_json_semantics() {
    let rev = "b".repeat(64);
    let body = format!("{{\"revision\":{rev:?},\"keys\":[{ED:?}]}}");
    let state = decode_access_key_state(body.as_bytes()).unwrap();
    assert_eq!(state.revision, rev);
    assert_eq!(state.keys, vec![ED.to_string()]);
    // Oversized.
    assert_eq!(
        decode_access_key_state(&vec![b'x'; 65537]).unwrap_err(),
        "native key operation not confirmed"
    );
    // Bad revision shape.
    assert_eq!(
        decode_access_key_state(br#"{"revision":"zz","keys":[]}"#).unwrap_err(),
        "invalid native key observation"
    );
    // Truncated JSON is invalid.
    assert_eq!(
        decode_access_key_state(br#"{"revision":"#).unwrap_err(),
        "invalid native key observation"
    );
    // Missing keys (null) is invalid, but empty is fine.
    let rev = "c".repeat(64);
    assert_eq!(
        decode_access_key_state(format!("{{\"revision\":{rev:?},\"keys\":null}}").as_bytes())
            .unwrap_err(),
        "invalid native key observation"
    );
    let state = decode_access_key_state(format!("{{\"revision\":{rev:?},\"keys\":[]}}").as_bytes())
        .unwrap();
    assert!(state.keys.is_empty());
    let body = format!(
        "{{\"revision\":{rev:?},\"keys\":[{ED:?}],\"Keys\":null}}"
    );
    let state = decode_access_key_state(body.as_bytes()).unwrap();
    assert_eq!(state.keys, vec![ED.to_string()]);
    // Unknown fields ignored, last duplicate wins (plain Unmarshal).
    let body =
        format!("{{\"revision\":{rev:?},\"extra\":1,\"keys\":[{ED:?}],\"revision\":{rev:?}}}");
    let state = decode_access_key_state(body.as_bytes()).unwrap();
    assert_eq!(state.keys, vec![ED.to_string()]);
    // Noncanonical member keys surface the canonical error.
    let body = format!("{{\"revision\":{rev:?},\"keys\":[\"{ED} x\"]}}");
    assert_eq!(
        decode_access_key_state(body.as_bytes()).unwrap_err(),
        "noncanonical or duplicate development key"
    );
}

#[test]
fn agent_program_path_matches_go() {
    // Mirrors Go `terminal.AgentProgram`: the fixed in-container agent.
    assert_eq!(AGENT_PROGRAM, "/usr/libexec/soda/project-terminal");
}

#[test]
fn account_provisions_with_exact_body_and_argv() {
    let id = pid();
    let mock = Mock::new(vec![
        Ok(inspect_payload(&id, true)),
        Ok(br#"{"login":"alice","identity":1}"#.to_vec()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = Account {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        keys: vec![format!("{ED} alice@host")],
    };
    rt.account(&input, deadline()).unwrap();
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1].1, "/usr/bin/podman");
    assert_eq!(
        calls[1].2,
        vec![
            "exec".to_string(),
            "--interactive".to_string(),
            format!("soda-{id}"),
            "/usr/libexec/soda/project-account".to_string(),
        ]
    );
    // Sorted map keys; the stored key keeps its trailing newline (escaped).
    assert_eq!(
        String::from_utf8(calls[1].0.clone()).unwrap(),
        format!(
            "{{\"admin\":false,\"identity\":1,\"keys\":[{ED:?}],\"login\":\"alice\"}}",
            ED = format!("{ED}\n")
        )
    );
}

#[test]
fn account_rejects_bad_requests_and_unconfirmed_helpers() {
    let id = pid();
    // Validation never execs.
    for input in [
        Account {
            project: id.clone(),
            login: "Root;id".to_string(),
            identity: 1,
            keys: vec![],
        },
        Account {
            project: id.clone(),
            login: "root".to_string(),
            identity: 1,
            keys: vec![],
        },
        Account {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 0,
            keys: vec![],
        },
        Account {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            keys: vec![ED.to_string(); 33],
        },
    ] {
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.account(&input, deadline()).unwrap_err(),
            "invalid project account"
        );
        assert!(mock.calls.borrow().is_empty());
    }
    // Stopped project.
    let mock = Mock::new(vec![Ok(inspect_payload(&id, false))]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = Account {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        keys: vec![],
    };
    assert_eq!(
        rt.account(&input, deadline()).unwrap_err(),
        "project is stopped"
    );
    // Helper echo mismatch.
    let mock = Mock::new(vec![
        Ok(inspect_payload(&id, true)),
        Ok(br#"{"login":"bob","identity":1}"#.to_vec()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.account(&input, deadline()).unwrap_err(),
        "native account identity was not confirmed"
    );
    // Podman failures pass through raw.
    let mock = Mock::new(vec![
        Ok(inspect_payload(&id, true)),
        Err("exit status 1".to_string()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(rt.account(&input, deadline()).unwrap_err(), "exit status 1");
}
