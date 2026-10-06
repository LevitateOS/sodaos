//! Factory preparation operations: the host side of the fixed
//! `project-factory-roles` helper exchange. Ports `prepare.go` and the
//! candidate-preparation half of `factory_candidate.go`
//! (`PrepareCandidate`; run-receipt inspection stays for the factory port).
//! Helper payloads are built byte-for-byte like the Go map marshals.

use std::time::Instant;

use crate::json::{self, Kind, Spec};
use crate::preparation::{
    self, HoldState, Prepare, PrepareHold, PrepareInspect, PrepareState, PrepareStop,
};
use crate::project::{Executor, Runtime};

mod candidate;
mod helper;
mod paths;
mod source;
mod state;
mod tools;

pub use self::paths::{
    path_clean, path_join, preparation_paths, prepare_id_map, single_line, valid_resolved_tool_path,
};
pub use self::state::{map_preparation_state, LauncherEvidence};

const STOP_RESPONSE_SPECS: &[Spec] = &[
    Spec {
        name: "stopped",
        kind: Kind::Str,
    },
    Spec {
        name: "retirement",
        kind: Kind::Str,
    },
    Spec {
        name: "known",
        kind: Kind::Bool,
    },
];

const HOLD_RESPONSE_SPECS: &[Spec] = &[Spec {
    name: "hold",
    kind: Kind::Object {
        go_type: "project.HoldState",
        struct_name: "HoldState",
        specs: preparation::HOLD_STATE_SPECS,
    },
}];

impl<E: Executor> Runtime<E> {
    /// `inspectPreparationState`: authoritative observed state for one identity.
    pub(crate) fn inspect_preparation_state(
        &self,
        project: &str,
        id: &str,
        container: &str,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(id));
        body.push_str(",\"op\":\"inspect\"}");
        let raw = self.factory_helper(project, &body, deadline)?;
        let mut state = map_preparation_state(container, &raw)?;
        state.id = id.to_string();
        state.project = project.to_string();
        Ok(state)
    }

    /// `Prepare`: approve, clone, verify, resolve tools, record, start.
    pub fn prepare(&self, input: &Prepare, deadline: Instant) -> Result<PrepareState, String> {
        input.validate()?;
        let prep = &input.preparation;
        let container = self.prepare_container(&prep.project, true, deadline)?;
        self.factory_helper(&prep.project, "{\"op\":\"ensure\"}", deadline)?;
        if let Err(err) = self.approve_preparation(prep, &input.setup, deadline) {
            if let Ok(stopped) =
                self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
            {
                if stopped.stopped {
                    return Ok(stopped);
                }
            }
            return Err(err);
        }
        let state =
            self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)?;
        if state.stopped || state.ready || state.phase == preparation::PREPARE_FAILED {
            return Ok(state);
        }
        self.clone_preparation_source(prep, deadline)?;
        let verified = self.verify_launcher_environment(prep, deadline)?;
        let (tools, missing) = self.resolve_preparation_tools(prep, deadline)?;
        self.record_preparation_tools(prep, &tools, &missing, &verified, deadline)?;
        if !missing.is_empty() || !verified.refusal.is_empty() {
            return self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
        }
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"op\":\"start\"}");
        if let Err(err) = self.factory_helper(&prep.project, &body, deadline) {
            // Go returns the re-observed state alongside the error; every
            // caller drops the state on error, so only the error crosses.
            let _ = self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
            return Err(err);
        }
        self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
    }

    /// `InspectPreparation`: authoritative state, never mutating.
    pub fn inspect_preparation(
        &self,
        input: &PrepareInspect,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        self.inspect_preparation_state(&input.project, &input.id, &container, deadline)
    }

    /// `StopPreparation`: persist the stop tombstone, report the state.
    pub fn stop_preparation(
        &self,
        input: &PrepareStop,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&input.id));
        body.push_str(",\"op\":\"stop\"}");
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        const ERR: &str = "preparation stop unconfirmed";
        let v = json::decode_strict(&raw).map_err(|_| ERR.to_string())?;
        let m = json::bind_root(&v, "struct", STOP_RESPONSE_SPECS, false)
            .map_err(|_| ERR.to_string())?;
        let retirement = m.take_string("retirement");
        if m.take_string("stopped") != input.id
            || (retirement != "confirmed" && retirement != "uncertain")
        {
            return Err(ERR.to_string());
        }
        let mut state =
            self.inspect_preparation_state(&input.project, &input.id, &container, deadline)?;
        state.retirement = retirement;
        Ok(state)
    }

    /// `HoldPreparation`: enforce the maintenance hold marker natively.
    pub fn hold_preparation(
        &self,
        input: &PrepareHold,
        deadline: Instant,
    ) -> Result<HoldState, String> {
        input.validate()?;
        self.prepare_container(&input.project, true, deadline)?;
        let op = if input.hold { "hold" } else { "release" };
        let body = format!("{{\"op\":\"{op}\",\"revision\":{}}}", input.revision);
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        let v =
            json::decode_strict(&raw).map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let m = json::bind_root(&v, "struct", HOLD_RESPONSE_SPECS, false)
            .map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let hold = HoldState::from_map(&m.take_map("hold"));
        if hold.active != input.hold {
            return Err("maintenance hold outcome not confirmed".to_string());
        }
        Ok(hold)
    }
}

#[cfg(test)]
mod tests {
    use super::paths::path_is_abs;
    use super::*;
    use crate::preparation::{FactoryCandidate, Preparation};
    use crate::project::Config;
    use std::cell::RefCell;
    use std::collections::VecDeque;

    const DIGEST: &str = "32c794ef2201b76b757bfba2c23bba06dcc5a8c6121f6fabc99115ef12043ced";
    const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
    const EFFECTS: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

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
        format!("p{}", "a".repeat(24))
    }

    fn fid() -> String {
        format!("f{}", "b".repeat(24))
    }

    fn container_payload(id: &str) -> Vec<u8> {
        let cid = "f".repeat(64);
        format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        )
        .into_bytes()
    }

    fn observation(phase: &str, role: &str, ready: bool, stopped: bool) -> Vec<u8> {
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

    fn preparation() -> Preparation {
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

    fn setup() -> preparation::ApprovedSetup {
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

    #[test]
    fn path_clean_matches_go_vectors() {
        for (input, want) in [
            ("", "."),
            ("/", "/"),
            ("//", "/"),
            ("/a/", "/a"),
            ("a/", "a"),
            ("/a//b", "/a/b"),
            ("/a/./b", "/a/b"),
            ("a/./b", "a/b"),
            ("/a/../b", "/b"),
            ("a/../../b", "../b"),
            ("/../", "/"),
            ("../a", "../a"),
            ("/..", "/"),
            ("..", ".."),
            (".", "."),
            ("/a/b/..", "/a"),
            ("a/b/../..", "."),
            ("/usr/bin/../bin/git", "/usr/bin/git"),
            ("/a/.../b", "/a/.../b"),
            ("a/.../b", "a/.../b"),
        ] {
            assert_eq!(path_clean(input), want, "{input:?}");
        }
        assert_eq!(path_join(&["a", "b", "c"]), "a/b/c");
        assert_eq!(path_join(&["/a", "b/../c"]), "/a/c");
        assert_eq!(path_join(&[]), "");
        assert_eq!(path_join(&[""]), "");
        assert_eq!(path_join(&["", "a", "b"]), "a/b");
        assert_eq!(path_join(&["a", "", "b"]), "a/b");
        assert_eq!(path_join(&["", ""]), "");
        assert!(path_is_abs("/x"));
        assert!(!path_is_abs("x"));
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

    fn approve_response(fid: &str) -> Vec<u8> {
        format!(
            "{{\"approved\":{fid:?},\"repeated\":false,\
             \"checkout\":\"/home/soda-coder/checkouts/{fid}\",\"credential_file\":\"\"}}"
        )
        .into_bytes()
    }

    #[test]
    fn prepare_runs_the_full_chain() {
        let id = pid();
        let fid = fid();
        let mock = Mock::new(vec![
            Ok(container_payload(&id)), // prepareContainer
            Ok(b"{}".to_vec()),         // ensure
            Ok(approve_response(&fid)), // approve
            Ok(observation("approved", "soda-coder", false, false)), // inspect
            Ok(b"".to_vec()),           // test -d (exists)
            Ok(format!("{COMMIT}\n").into_bytes()), // rev-parse
            Ok(b"".to_vec()),           // mkdir
            Ok(b"1001\n".to_vec()),     // id -u
            Ok(b"soda-coder\n".to_vec()), // id -un
            Ok(b"soda-coder\n".to_vec()), // id -Gn
            Ok(b"".to_vec()),           // test -r setup
            Err("exit status 1".to_string()), // test -w setup
            Err("exit status 1".to_string()), // test -w shared
            Err("exit status 1".to_string()), // test -e podman.sock
            Err("exit status 1".to_string()), // test -e docker.sock
            Err("exit status 1".to_string()), // test -x sudo
            Ok(b"".to_vec()),           // test -w home
            Ok(b"{}".to_vec()),         // record
            Ok(b"{}".to_vec()),         // start
            Ok(observation("running", "soda-coder", false, false)), // final inspect
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: preparation(),
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        assert_eq!(state.project, id);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 20);
        // Approve payload: sorted keys, padded base64 files.
        assert_eq!(
            String::from_utf8(calls[2].0.clone()).unwrap(),
            format!(
                "{{\"bundle\":\"QlVORExF\",\"credential\":\"\",\
                 \"files\":{{\"check.sh\":\"IyEvYmluL3NoCg==\",\
                 \"setup.sh\":\"IyEvYmluL3NoCg==\"}},\"id\":{fid:?},\
                 \"op\":\"approve\",\"role\":\"soda-coder\",\"setup_digest\":{DIGEST:?},\
                 \"source_commit\":{COMMIT:?}}}"
            )
        );
        assert_eq!(
            calls[2].2,
            vec![
                "exec".to_string(),
                "--interactive".to_string(),
                format!("soda-{id}"),
                "/usr/libexec/soda/project-factory-roles".to_string(),
            ]
        );
        // Record payload carries the observed evidence.
        assert_eq!(
            String::from_utf8(calls[17].0.clone()).unwrap(),
            format!(
                "{{\"id\":{fid:?},\"missing\":\"\",\"op\":\"record\",\"tools\":[],\
                 \"verified\":{{\"groups\":\"soda-coder\",\"login\":\"soda-coder\",\
                 \"uid\":\"1001\"}}}}"
            )
        );
        // Start payload.
        assert_eq!(
            String::from_utf8(calls[18].0.clone()).unwrap(),
            format!("{{\"id\":{fid:?},\"op\":\"start\"}}")
        );
    }

    #[test]
    fn prepare_short_circuits_and_recovers() {
        let id = pid();
        let fid = fid();
        // Ready state returns right after the first inspect.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Ok(approve_response(&fid)),
            Ok(observation("ready", "soda-coder", true, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: preparation(),
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert!(state.ready);
        assert_eq!(mock.calls.borrow().len(), 4);
        // Approve failure with a stopped tombstone returns the tombstone.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Err("helper blew up".to_string()),
            Ok(observation("stopped", "soda-coder", false, true)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert!(state.stopped);
        // Approve failure without a tombstone surfaces the error.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Err("helper blew up".to_string()),
            Ok(observation("approved", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare(&input, deadline()).unwrap_err(),
            "helper blew up"
        );
        // Validation never execs.
        let mut bad = preparation();
        bad.id = "x".to_string();
        let input = Prepare {
            preparation: bad,
            setup: setup(),
        };
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare(&input, deadline()).unwrap_err(),
            "invalid preparation identity"
        );
        assert!(mock.calls.borrow().is_empty());
    }

    #[test]
    fn prepare_skips_start_when_blocked() {
        let id = pid();
        let fid = fid();
        // Missing tool: record, then re-observe, never start.
        let mut prep = preparation();
        prep.tools = vec!["git".to_string(), "go".to_string()];
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Ok(approve_response(&fid)),
            Ok(observation("approved", "soda-coder", false, false)),
            Ok(b"".to_vec()),
            Ok(format!("{COMMIT}\n").into_bytes()),
            Ok(b"".to_vec()),
            Ok(b"1001\n".to_vec()),
            Ok(b"soda-coder\n".to_vec()),
            Ok(b"soda-coder\n".to_vec()),
            Ok(b"".to_vec()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Ok(b"".to_vec()),
            Ok(b"/usr/bin/git\n".to_vec()),   // command -v git
            Ok(b"git version 2\n".to_vec()),  // git --version
            Err("exit status 1".to_string()), // command -v go missing
            Ok(b"{}".to_vec()),               // record
            Ok(observation("approved", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: prep,
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert_eq!(state.phase, "approved");
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 22);
        assert!(!calls
            .iter()
            .any(|c| String::from_utf8_lossy(&c.0).contains("\"op\":\"start\"")));
        // The missing marker names go.
        assert!(String::from_utf8_lossy(&calls[20].0).contains("\"missing\":\"go\""));
        assert!(
            String::from_utf8_lossy(&calls[20].0)
                .contains("\"tools\":[{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"git version 2\"}]")
        );
    }

    #[test]
    fn inspect_stop_hold_paths() {
        let id = pid();
        let fid = fid();
        // Inspect.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("running", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt
            .inspect_preparation(
                &PrepareInspect {
                    project: id.clone(),
                    id: fid.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        // Stop.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(
                format!("{{\"stopped\":{fid:?},\"retirement\":\"confirmed\",\"known\":true}}")
                    .into_bytes(),
            ),
            Ok(observation("stopped", "soda-coder", false, true)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt
            .stop_preparation(
                &PrepareStop {
                    project: id.clone(),
                    id: fid.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert!(state.stopped);
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(
            String::from_utf8(mock.calls.borrow()[1].0.clone()).unwrap(),
            format!("{{\"id\":{fid:?},\"op\":\"stop\"}}")
        );
        // Stop with a bad tombstone.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(
                format!("{{\"stopped\":{fid:?},\"retirement\":\"maybe\",\"known\":true}}")
                    .into_bytes(),
            ),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.stop_preparation(
                &PrepareStop {
                    project: id.clone(),
                    id: fid.clone()
                },
                deadline()
            )
            .unwrap_err(),
            "preparation stop unconfirmed"
        );
        // Hold and release.
        for (hold, op) in [(true, "hold"), (false, "release")] {
            let mock = Mock::new(vec![
                Ok(container_payload(&id)),
                Ok(format!("{{\"hold\":{{\"active\":{hold},\"revision\":7}}}}").into_bytes()),
            ]);
            let rt = Runtime {
                exec: &mock,
                config: test_config(),
            };
            let state = rt
                .hold_preparation(
                    &PrepareHold {
                        project: id.clone(),
                        hold,
                        revision: 7,
                    },
                    deadline(),
                )
                .unwrap();
            assert_eq!(state.active, hold);
            assert_eq!(state.revision, 7);
            assert_eq!(
                String::from_utf8(mock.calls.borrow()[1].0.clone()).unwrap(),
                format!("{{\"op\":\"{op}\",\"revision\":7}}")
            );
        }
        // Hold outcome mismatch.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{\"hold\":{\"active\":false,\"revision\":7}}".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.hold_preparation(
                &PrepareHold {
                    project: id.clone(),
                    hold: true,
                    revision: 7
                },
                deadline()
            )
            .unwrap_err(),
            "maintenance hold outcome not confirmed"
        );
    }

    #[test]
    fn candidate_reuses_ready_source() {
        let id = pid();
        let source = format!("f{}", "c".repeat(24));
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let fid = prep.id.clone();
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),                             // prepareContainer
            Ok(observation("ready", "soda-reviewer", true, false)), // source inspect
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),   // dir stat
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // request stat
            Ok(request.into_bytes()),                               // request head
            Ok(b"check.sh\nsetup.sh\n".to_vec()),                   // ls
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // check.sh stat
            Ok(b"#!/bin/sh\n".to_vec()),                            // check.sh head
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // setup.sh stat
            Ok(b"#!/bin/sh\n".to_vec()),                            // setup.sh head
            // ... then the full prepare chain for the fresh identity:
            Ok(container_payload(&id)), // prepareContainer
            Ok(b"{}".to_vec()),         // ensure
            Ok(String::from_utf8(approve_response(&fid))
                .unwrap()
                .replace("soda-coder", "soda-reviewer")
                .into_bytes()),
            Ok(observation("approved", "soda-reviewer", false, false)),
            Ok(b"".to_vec()), // test -d
            Ok(format!("{COMMIT}\n").into_bytes()),
            Ok(b"".to_vec()), // mkdir
            Ok(b"1002\n".to_vec()),
            Ok(b"soda-reviewer\n".to_vec()),
            Ok(b"soda-reviewer\n".to_vec()),
            Ok(b"".to_vec()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Ok(b"".to_vec()),
            Ok(b"{}".to_vec()), // record
            Ok(b"{}".to_vec()), // start
            Ok(observation("running", "soda-reviewer", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = FactoryCandidate {
            preparation: prep,
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        let state = rt.prepare_candidate(&input, deadline()).unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 30);
        // The snapshot stat covers all four directories.
        assert_eq!(
            calls[2].2[..7],
            [
                "exec".to_string(),
                "f".repeat(64),
                "/usr/bin/stat".to_string(),
                "-c".to_string(),
                "%u:%g:%a".to_string(),
                "--".to_string(),
                "/var/lib/soda/factory".to_string(),
            ]
        );
        // Source not ready short-circuits before any snapshot reads.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("running", "soda-reviewer", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let input = FactoryCandidate {
            preparation: prep,
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        assert_eq!(
            rt.prepare_candidate(&input, deadline()).unwrap_err(),
            "candidate source preparation is not ready for this role and setup"
        );
        assert_eq!(mock.calls.borrow().len(), 2);
    }

    #[test]
    fn candidate_rejects_bad_snapshots() {
        let id = pid();
        let source = format!("f{}", "c".repeat(24));
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let input = || FactoryCandidate {
            preparation: prep.clone(),
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        // Unprotected directories.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:700\n".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot is not protected"
        );
        // Request that differs from the recorded inputs.
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"other\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(request.into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot differs from its recorded inputs"
        );
        // Invalid file set via an illegal name.
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(request.into_bytes()),
            Ok(b"setup.sh\n../evil\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(b"#!/bin/sh\n".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot has an invalid file set"
        );
    }
}
