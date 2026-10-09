//! Prepare execution cases: full chain, short-circuit recovery,
//! blocked start, and the mixed inspect/stop/hold paths.

use super::tests::{
    approve_response, container_payload, deadline, fid, observation, pid, preparation, setup,
    test_config, Mock, COMMIT, DIGEST,
};
use super::*;
use crate::preparation::{setup_digest_of, PrepareContextRead, MAX_CONTEXT_FILE_BYTES};
use crate::project::Native;
use std::cell::RefCell;
use std::collections::HashMap;
use std::path::PathBuf;
use std::time::Duration;

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

struct ContextGitExec {
    calls: RefCell<Vec<(Vec<String>, Instant, usize, usize)>>,
    checkout: String,
    container: String,
    prep_id: String,
    source_commit: String,
    setup_digest: String,
    request_file: Vec<u8>,
    setup_file: Vec<u8>,
    check_file: Vec<u8>,
}

impl Executor for ContextGitExec {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        assert_eq!(cmd, "/usr/bin/podman");
        self.calls.borrow_mut().push((
            args.iter().map(|arg| arg.to_string()).collect(),
            deadline,
            0,
            0,
        ));
        if args.get(0) == Some(&"--remote=false") && args.get(1) == Some(&"inspect") {
            return Ok(container_payload(&pid()));
        }
        if args.get(0) == Some(&"exec") && args.get(1) == Some(&"--interactive") {
            let operation = b"\"op\":\"inspect\"";
            assert!(stdin.windows(operation.len()).any(|part| part == operation));
            return Ok(format!(
                "{{\"known\":true,\"phase\":\"ready\",\"role\":\"soda-coder\",\
                 \"setup_digest\":{0:?},\"source_commit\":{1:?},\"ready\":true,\
                 \"stopped\":false,\"setup_log\":\"\",\"check_log\":\"\"}}",
                self.setup_digest, self.source_commit
            )
            .into_bytes());
        }
        if args.get(0) == Some(&"exec")
            && (args.get(2) == Some(&"/usr/bin/stat") || args.get(2) == Some(&"/usr/bin/env"))
        {
            if args.iter().any(|arg| *arg == "%u:%g:%a") {
                return Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec());
            }
            return Ok(b"0:0:644:1:regular file\n".to_vec());
        }
        if args.get(0) == Some(&"exec") && args.get(2) == Some(&"/usr/bin/ls") {
            return Ok(b"check.sh\nsetup.sh\nsource.bundle\n".to_vec());
        }
        if args.get(0) == Some(&"exec") && args.get(2) == Some(&"/usr/bin/head") {
            return match args.last().copied() {
                Some(path) if path.ends_with("request.json") => Ok(self.request_file.clone()),
                Some(path) if path.ends_with("setup.sh") => Ok(self.setup_file.clone()),
                Some(path) if path.ends_with("check.sh") => Ok(self.check_file.clone()),
                _ => Err("unexpected preparation file read".into()),
            };
        }
        Err(format!("unexpected context command: {args:?}"))
    }

    fn run_bounded(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        assert_eq!(cmd, "/usr/bin/podman");
        assert_eq!(args.get(0), Some(&"exec"));
        assert_eq!(args.get(1), Some(&"--user"));
        assert_eq!(args.get(2), Some(&"soda-coder"));
        assert_eq!(args.get(3), Some(&self.container.as_str()));
        assert_eq!(args.get(4), Some(&"/usr/bin/env"));
        assert_eq!(args.get(5), Some(&"-i"));
        assert_eq!(args.get(14), Some(&"/usr/bin/git"));
        assert_eq!(args.get(19), Some(&"-C"));
        assert_eq!(
            args.get(20),
            Some(&format!("/home/soda-coder/checkouts/{}", self.prep_id).as_str())
        );
        assert!(deadline > Instant::now());
        self.calls.borrow_mut().push((
            args.iter().map(|arg| arg.to_string()).collect(),
            deadline,
            stdout_limit,
            stderr_limit,
        ));
        // Transport substitution only: run the exact production env/Git argv
        // against the owned local fixture checkout instead of Podman.
        let mut local_args: Vec<String> = args[5..].iter().map(|arg| arg.to_string()).collect();
        let checkout_at = local_args.iter().position(|arg| arg == "-C").unwrap() + 1;
        local_args[checkout_at] = self.checkout.clone();
        let refs: Vec<&str> = local_args.iter().map(String::as_str).collect();
        Native.run_bounded(
            stdin,
            "/usr/bin/env",
            &refs,
            deadline,
            stdout_limit,
            stderr_limit,
        )
    }
}

struct ContextScratch(PathBuf);

impl Drop for ContextScratch {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

fn context_fixture() -> (ContextScratch, ContextGitExec, String, String) {
    let root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../.artifacts");
    std::fs::create_dir_all(&root).unwrap();
    let nonce = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    let scratch =
        ContextScratch(root.join(format!("prepare-context-{}-{nonce}", std::process::id())));
    let checkout = scratch.0.join("checkout");
    std::fs::create_dir_all(checkout.join("src")).unwrap();
    let git = |args: &[&str]| {
        Native
            .run(
                &[],
                "/usr/bin/git",
                args,
                Instant::now() + Duration::from_secs(10),
            )
            .unwrap()
    };
    let path = checkout.to_str().unwrap();
    git(&["-C", path, "init", "--quiet"]);
    git(&["-C", path, "config", "--local", "user.name", "soda-tester"]);
    git(&[
        "-C",
        path,
        "config",
        "--local",
        "user.email",
        "soda-tester@example.invalid",
    ]);
    std::fs::write(
        checkout.join("src/lib.go"),
        b"package demo\nconst Value = 1\n",
    )
    .unwrap();
    std::fs::write(
        checkout.join("src/large.txt"),
        vec![b'x'; MAX_CONTEXT_FILE_BYTES + 1],
    )
    .unwrap();
    std::fs::write(checkout.join("src/huge.txt"), b"base\n").unwrap();
    std::fs::write(checkout.join("src/nul.txt"), b"base\0text\n").unwrap();
    std::fs::write(checkout.join("src/binary.txt"), b"base text\n").unwrap();
    git(&[
        "-C",
        path,
        "add",
        "src/lib.go",
        "src/large.txt",
        "src/huge.txt",
        "src/nul.txt",
        "src/binary.txt",
    ]);
    git(&[
        "-C",
        path,
        "-c",
        "commit.gpgsign=false",
        "commit",
        "-m",
        "approved base",
    ]);
    let base = String::from_utf8(git(&["-C", path, "rev-parse", "HEAD"]))
        .unwrap()
        .trim()
        .to_string();
    std::fs::write(
        checkout.join("src/lib.go"),
        b"package demo\nconst Value = 2\n",
    )
    .unwrap();
    std::fs::write(checkout.join("src/huge.txt"), vec![b'y'; 40 * 1024]).unwrap();
    std::fs::write(checkout.join("src/binary.txt"), b"candidate\0binary\n").unwrap();
    git(&[
        "-C",
        path,
        "add",
        "src/lib.go",
        "src/huge.txt",
        "src/binary.txt",
    ]);
    git(&[
        "-C",
        path,
        "-c",
        "commit.gpgsign=false",
        "commit",
        "-m",
        "candidate",
    ]);
    let candidate = String::from_utf8(git(&["-C", path, "rev-parse", "HEAD"]))
        .unwrap()
        .trim()
        .to_string();

    let setup_file = b"#!/bin/sh\ntrue\n".to_vec();
    let check_file = b"#!/bin/sh\ntrue\n".to_vec();
    let approved = HashMap::from([
        ("setup.sh".to_string(), setup_file.clone()),
        ("check.sh".to_string(), check_file.clone()),
    ]);
    let setup_digest = setup_digest_of(&approved);
    let prep_id = fid();
    let request_file = format!(
        "{{\"id\":{prep_id:?},\"role\":\"soda-coder\",\"setup_digest\":{setup_digest:?},\
         \"source_commit\":{base:?},\"credential\":\"never-return-this\"}}"
    )
    .into_bytes();
    let exec = ContextGitExec {
        calls: RefCell::new(Vec::new()),
        checkout: path.to_string(),
        container: "f".repeat(64),
        prep_id,
        source_commit: base.clone(),
        setup_digest,
        request_file,
        setup_file,
        check_file,
    };
    (scratch, exec, base, candidate)
}

#[test]
fn preparation_context_reads_exact_base_and_candidate_diff_with_bounds() {
    let (_scratch, exec, base, candidate) = context_fixture();
    let project_id = pid();
    let input = PrepareContextRead {
        project: project_id,
        id: exec.prep_id.clone(),
        source_commit: base.clone(),
        approved_base: base.clone(),
        candidate: candidate.clone(),
        paths: vec!["src/lib.go".to_string()],
    };
    let deadline = Instant::now() + Duration::from_secs(10);
    let runtime = Runtime {
        exec: &exec,
        config: test_config(),
    };
    let material = runtime.read_preparation_context(&input, deadline).unwrap();
    assert_eq!(material.approved_base, base);
    assert_eq!(material.candidate, candidate);
    assert_eq!(
        material.files[0].content,
        b"package demo\nconst Value = 1\n"
    );
    assert!(String::from_utf8_lossy(&material.diff).contains("const Value = 2"));
    assert!(!material.diff.windows(6).any(|part| part == b"never-"));
    assert_eq!(material.setup, b"#!/bin/sh\ntrue\n");
    assert_eq!(material.check, b"#!/bin/sh\ntrue\n");
    assert!(exec.calls.borrow().iter().all(|call| call.1 == deadline));
    assert!(exec
        .calls
        .borrow()
        .iter()
        .filter(|call| call.2 > 0)
        .all(|call| call.3 == 1024));

    let oversized = PrepareContextRead {
        paths: vec!["src/large.txt".to_string()],
        ..input.clone()
    };
    assert!(runtime
        .read_preparation_context(&oversized, deadline)
        .is_err());
    let oversized_diff = PrepareContextRead {
        paths: vec!["src/huge.txt".to_string()],
        ..input.clone()
    };
    assert!(runtime
        .read_preparation_context(&oversized_diff, deadline)
        .is_err());
    let nul_base = PrepareContextRead {
        paths: vec!["src/nul.txt".to_string()],
        ..input.clone()
    };
    assert!(runtime
        .read_preparation_context(&nul_base, deadline)
        .unwrap_err()
        .contains("not UTF-8 text"));
    let binary_candidate = PrepareContextRead {
        paths: vec!["src/binary.txt".to_string()],
        ..input.clone()
    };
    assert!(runtime
        .read_preparation_context(&binary_candidate, deadline)
        .unwrap_err()
        .contains("binary file"));
    let stale = PrepareContextRead {
        candidate: base.clone(),
        ..input.clone()
    };
    assert!(runtime
        .read_preparation_context(&stale, deadline)
        .unwrap_err()
        .contains("checkout HEAD"));
    for path in [
        "../secret",
        ".env",
        "keys/id_rsa",
        ".codex/auth.json",
        "/absolute",
        "src\\file",
    ] {
        let invalid = PrepareContextRead {
            paths: vec![path.to_string()],
            ..input.clone()
        };
        assert!(invalid.validate().is_err(), "unsafe path admitted: {path}");
    }
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
    assert!(String::from_utf8_lossy(&calls[20].0).contains(
        "\"tools\":[{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"git version 2\"}]"
    ));
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
    // An unknown identity receives a confirmed tombstone before any approval
    // or start, so the caller can safely retain that state as retirement proof.
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(
            format!("{{\"stopped\":{fid:?},\"retirement\":\"confirmed\",\"known\":false}}")
                .into_bytes(),
        ),
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
    assert_eq!(state.id, fid);
    assert_eq!(state.project, id);
    assert_eq!(state.phase, crate::preparation::PREPARE_STOPPED);
    assert!(state.stopped);
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(mock.calls.borrow().len(), 2);
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(format!("{{\"stopped\":{fid:?},\"retirement\":\"confirmed\"}}").into_bytes()),
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
            deadline(),
        )
        .unwrap_err(),
        "preparation stop unconfirmed"
    );
    // Stop with a bad tombstone.
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(format!("{{\"stopped\":{fid:?},\"retirement\":\"maybe\",\"known\":true}}").into_bytes()),
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
