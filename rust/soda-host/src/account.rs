//! Project account operations.
//!
//! Ports the account tail of `create.go` (`Account`, `canonicalizeAccountKeys`,
//! `confirmAccount`) and `access_keys.go` (`AccessKeys`, `canonicalKeys`,
//! `decodeAccessKeyState`). In-container key operations run through the fixed
//! `project-terminal` agent binary (`keys` action), like the Go `AgentExec`.

use std::time::Instant;

use crate::domain::{self, AccessKeyState, AccessKeys, Account};
use crate::json::{self, Kind, Spec};
use crate::project::{Executor, Runtime};
use crate::ssh;

/// Fixed in-container terminal agent, mirroring Go `terminal.AgentProgram`.
pub const AGENT_PROGRAM: &str = "/usr/libexec/soda/project-terminal";

/// `keyRevision = ^[0-9a-f]{64}$`.
pub fn valid_key_revision(s: &str) -> bool {
    s.len() == 64
        && s.bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

/// `canonicalizeAccountKeys`: parse every key, drop comments, keep the
/// `MarshalAuthorizedKey` rendering (with its trailing newline) for the
/// container-side `project-account` request.
pub fn canonicalize_account_keys(values: &[String]) -> Result<Vec<String>, String> {
    let mut keys = Vec::with_capacity(values.len());
    for value in values {
        let (key, has_options) = ssh::parse_authorized_key(value.as_bytes())
            .map_err(|_| "invalid public key".to_string())?;
        if has_options {
            return Err("invalid public key".to_string());
        }
        keys.push(ssh::marshal_authorized_key(&key.key_type, &key.blob));
    }
    Ok(keys)
}

/// `confirmAccount`: the helper must echo the login identity back; every
/// shape mismatch collapses into one confirmation error.
const CONFIRM_SPECS: &[Spec] = &[
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
];

fn confirm_account(out: &[u8], login: &str, identity: i64) -> Result<(), String> {
    const ERR: &str = "native account identity was not confirmed";
    if out.len() > 4096 {
        return Err(ERR.to_string());
    }
    let v = json::decode_strict(out).map_err(|_| ERR.to_string())?;
    let m = json::bind_root(&v, "struct", CONFIRM_SPECS, false).map_err(|_| ERR.to_string())?;
    let got_login = m.take_string("login");
    let got_identity = m.take_i64("identity");
    if got_login != login || got_identity != identity {
        return Err(ERR.to_string());
    }
    Ok(())
}

/// `canonicalKeys`: canonical `type base64` lines only, no options, no
/// comments, no duplicates, at most 32 keys and 48000 bytes.
pub fn canonical_keys(values: &[String]) -> Result<Vec<String>, String> {
    if values.len() > 32 {
        return Err("too many development keys".to_string());
    }
    let mut out = Vec::with_capacity(values.len());
    let mut seen = std::collections::HashSet::with_capacity(values.len());
    let mut size = 0usize;
    for value in values {
        let (key, has_options) = ssh::parse_authorized_key(value.as_bytes())
            .map_err(|_| "invalid development key".to_string())?;
        if has_options {
            return Err("invalid development key".to_string());
        }
        // `MarshalAuthorizedKey` ends with "\n"; `TrimSpace` strips it.
        let canonical = ssh::marshal_authorized_key(&key.key_type, &key.blob);
        let canonical = canonical.trim_end_matches('\n');
        if value.trim() != canonical || !seen.insert(canonical.to_string()) {
            return Err("noncanonical or duplicate development key".to_string());
        }
        size += canonical.len();
        if size > 48000 {
            return Err("development key set too large".to_string());
        }
        out.push(canonical.to_string());
    }
    Ok(out)
}

fn valid_access_keys_request(input: &AccessKeys) -> bool {
    if !domain::valid_login(&input.login) || input.login == "root" || input.identity <= 0 {
        return false;
    }
    if input.apply {
        return valid_key_revision(&input.revision);
    }
    input.revision.is_empty() && input.keys.is_empty()
}

/// `decodeAccessKeyState`: plain `encoding/json` semantics (unknown fields
/// ignored, last duplicate wins); only the revision shape, key presence and
/// key canonicality are enforced.
const KEY_STATE_SPECS: &[Spec] = &[
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
];

pub fn decode_access_key_state(data: &[u8]) -> Result<AccessKeyState, String> {
    if data.len() > 65536 {
        return Err("native key operation not confirmed".to_string());
    }
    const ERR: &str = "invalid native key observation";
    let v = json::decode_tolerant(data).map_err(|_| ERR.to_string())?;
    // Tolerant binding: plain Unmarshal ignores unknown fields.
    let m = json::bind_root(&v, "AccessKeyState", KEY_STATE_SPECS, true)
        .map_err(|_| ERR.to_string())?;
    if !m.contains("keys") {
        return Err(ERR.to_string());
    }
    let revision = m.take_string("revision");
    let keys = m.take_str_list("keys");
    if !valid_key_revision(&revision) {
        return Err(ERR.to_string());
    }
    let keys = canonical_keys(&keys)?;
    Ok(AccessKeyState { revision, keys })
}

impl<E: Executor> Runtime<E> {
    /// `Account`: provision a project login via the in-container helper.
    pub fn account(&self, input: &Account, deadline: Instant) -> Result<(), String> {
        if !domain::valid_login(&input.login)
            || input.login == "root"
            || input.identity <= 0
            || input.keys.len() > 32
        {
            return Err("invalid project account".to_string());
        }
        let (env, owner) = self.inspect(&input.project, deadline)?;
        if !env.running {
            return Err("project is stopped".to_string());
        }
        let keys = canonicalize_account_keys(&input.keys)?;
        // Go map marshal emits keys sorted: admin, identity, keys, login.
        let mut body = String::from("{\"admin\":");
        body.push_str(if input.identity == owner {
            "true"
        } else {
            "false"
        });
        body.push_str(",\"identity\":");
        body.push_str(&input.identity.to_string());
        body.push_str(",\"keys\":[");
        for (i, key) in keys.iter().enumerate() {
            if i > 0 {
                body.push(',');
            }
            body.push_str(&json::quote(key));
        }
        body.push_str("],\"login\":");
        body.push_str(&json::quote(&input.login));
        body.push('}');
        let target = format!("soda-{}", input.project);
        // No `--remote=false` on this path, exactly like the Go call.
        let args = [
            "exec",
            "--interactive",
            &target,
            "/usr/libexec/soda/project-account",
        ];
        let out = self.podman(body.as_bytes(), &args, deadline)?;
        confirm_account(&out, &input.login, input.identity)
    }

    /// `AccessKeys`: preview or revision-checked key-set replacement.
    pub fn access_keys(
        &self,
        input: &AccessKeys,
        deadline: Instant,
    ) -> Result<AccessKeyState, String> {
        if !valid_access_keys_request(input) {
            return Err("invalid own-account key operation".to_string());
        }
        let keys = canonical_keys(&input.keys)?;
        let cid = self.project_container(&input.project, true, deadline)?;
        // `previewAccessKeys`: re-observe without applying; any observation
        // failure or revision drift collapses into one error.
        if input.apply {
            let preview = AccessKeys {
                project: input.project.clone(),
                login: input.login.clone(),
                identity: input.identity,
                revision: String::new(),
                keys: Vec::new(),
                apply: false,
            };
            let observed = self
                .access_keys(&preview, deadline)
                .map_err(|_| "native keys changed or are not managed canonical keys".to_string())?;
            if observed.revision != input.revision {
                return Err("native keys changed or are not managed canonical keys".to_string());
            }
        }
        // Go map marshal emits keys sorted: apply, identity, keys, login, revision.
        let mut body = String::from("{\"apply\":");
        body.push_str(if input.apply { "true" } else { "false" });
        body.push_str(",\"identity\":");
        body.push_str(&input.identity.to_string());
        body.push_str(",\"keys\":[");
        for (i, key) in keys.iter().enumerate() {
            if i > 0 {
                body.push(',');
            }
            body.push_str(&json::quote(key));
        }
        body.push_str("],\"login\":");
        body.push_str(&json::quote(&input.login));
        body.push_str(",\"revision\":");
        body.push_str(&json::quote(&input.revision));
        body.push('}');
        let args = [
            "--remote=false",
            "exec",
            "--interactive",
            &cid,
            AGENT_PROGRAM,
            "keys",
        ];
        let data = self
            .podman(body.as_bytes(), &args, deadline)
            .map_err(|_| "native key operation not confirmed".to_string())?;
        let out = decode_access_key_state(&data)?;
        if input.apply && out.keys.join("\n") != keys.join("\n") {
            return Err("native key result differs from requested set".to_string());
        }
        Ok(out)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::project::Config;
    use std::cell::RefCell;
    use std::collections::VecDeque;

    const ED: &str =
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";

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

    fn deadline() -> Instant {
        Instant::now() + std::time::Duration::from_secs(5)
    }

    fn test_config() -> Config {
        Config {
            muse_socket: String::new(),
            image: "img".to_string(),
            network: "sodanet".to_string(),
            subnet: "10.0.0.0/24".to_string(),
            bridge: "sodabr".to_string(),
        }
    }

    fn pid() -> String {
        format!("p{}", "e".repeat(24))
    }

    fn inspect_payload(id: &str, running: bool) -> Vec<u8> {
        let mut s = String::from("[{\"Image\":\"\",\"Config\":{\"Labels\":{\"org.soda.project\":");
        s.push_str(&crate::json::quote(id));
        s.push_str(",\"org.soda.owner\":\"42\"}},\"State\":{\"Running\":");
        s.push_str(if running { "true" } else { "false" });
        s.push_str(
            ",\"NetworkSettings\":{\"Networks\":{\"sodanet\":{\"IPAddress\":\"10.0.0.5\"}}}}}]",
        );
        s.into_bytes()
    }

    fn container_payload(id: &str) -> Vec<u8> {
        let cid = "f".repeat(64);
        format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        )
        .into_bytes()
    }

    fn hex(bytes: &[u8]) -> String {
        bytes.iter().map(|b| format!("{b:02x}")).collect()
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
        let state =
            decode_access_key_state(format!("{{\"revision\":{rev:?},\"keys\":[]}}").as_bytes())
                .unwrap();
        assert!(state.keys.is_empty());
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

    #[test]
    fn access_keys_preview_observes_without_applying() {
        let id = pid();
        let rev = "d".repeat(64);
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(format!("{{\"revision\":{rev:?},\"keys\":[{ED:?}]}}").into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: String::new(),
            keys: vec![],
            apply: false,
        };
        let state = rt.access_keys(&input, deadline()).unwrap();
        assert_eq!(state.revision, rev);
        assert_eq!(state.keys, vec![ED.to_string()]);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[1].1, "/usr/bin/podman");
        assert_eq!(
            &calls[1].2[..6],
            &[
                "--remote=false".to_string(),
                "exec".to_string(),
                "--interactive".to_string(),
                "f".repeat(64),
                AGENT_PROGRAM.to_string(),
                "keys".to_string(),
            ]
        );
        assert_eq!(
            String::from_utf8(calls[1].0.clone()).unwrap(),
            "{\"apply\":false,\"identity\":1,\"keys\":[],\"login\":\"alice\",\"revision\":\"\"}"
        );
    }

    #[test]
    fn access_keys_apply_round_trips_preview_and_confirms_set() {
        let id = pid();
        let rev = "e".repeat(64);
        let observed = format!("{{\"revision\":{rev:?},\"keys\":[{ED:?}]}}");
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(container_payload(&id)),
            Ok(observed.clone().into_bytes()),
            Ok(observed.into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: rev.clone(),
            keys: vec![ED.to_string()],
            apply: true,
        };
        let state = rt.access_keys(&input, deadline()).unwrap();
        assert_eq!(state.revision, rev);
        assert_eq!(mock.calls.borrow().len(), 4);
        // The preview call carries an empty revision and key set.
        assert!(String::from_utf8(mock.calls.borrow()[2].0.clone())
            .unwrap()
            .contains("\"revision\":\"\""));
    }

    #[test]
    fn access_keys_rejects_drift_and_bad_requests() {
        let id = pid();
        // Invalid request shapes never exec.
        for input in [
            AccessKeys {
                project: id.clone(),
                login: "root".to_string(),
                identity: 1,
                revision: String::new(),
                keys: vec![],
                apply: false,
            },
            AccessKeys {
                project: id.clone(),
                login: "alice".to_string(),
                identity: 1,
                revision: "zz".to_string(),
                keys: vec![],
                apply: true,
            },
            AccessKeys {
                project: id.clone(),
                login: "alice".to_string(),
                identity: 1,
                revision: String::new(),
                keys: vec![ED.to_string()],
                apply: false,
            },
        ] {
            let mock = Mock::new(vec![]);
            let rt = Runtime {
                exec: &mock,
                config: test_config(),
            };
            assert_eq!(
                rt.access_keys(&input, deadline()).unwrap_err(),
                "invalid own-account key operation"
            );
            assert!(mock.calls.borrow().is_empty());
        }
        // Preview revision drift.
        let rev = "f".repeat(64);
        let other = "0".repeat(64);
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(container_payload(&id)),
            Ok(format!("{{\"revision\":{other:?},\"keys\":[]}}").into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: rev,
            keys: vec![],
            apply: true,
        };
        assert_eq!(
            rt.access_keys(&input, deadline()).unwrap_err(),
            "native keys changed or are not managed canonical keys"
        );
        // Applied result differs from the requested set.
        let rev = "1".repeat(64);
        let observed = format!("{{\"revision\":{rev:?},\"keys\":[]}}");
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(container_payload(&id)),
            Ok(observed.clone().into_bytes()),
            Ok(observed.into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: rev,
            keys: vec![ED.to_string()],
            apply: true,
        };
        assert_eq!(
            rt.access_keys(&input, deadline()).unwrap_err(),
            "native key result differs from requested set"
        );
    }
}
