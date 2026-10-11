//! Candidate snapshot cases: ready-source reuse and bad-snapshot
//! rejection.

use super::tests::{
    COMMIT, DIGEST, Mock, approve_response, container_payload, deadline, observation, pid,
    preparation, test_config,
};
use super::*;
use crate::preparation::FactoryCandidate;

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
        deadline: "2099-01-01T00:00:00Z".to_string(),
    };
    let state = rt.prepare_candidate(&input, deadline()).unwrap();
    assert_eq!(state.phase, "running");
    assert_eq!(state.id, fid);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 30);
    assert_eq!(
        String::from_utf8(calls[28].0.clone()).unwrap(),
        format!("{{\"deadline\":\"2099-01-01T00:00:00Z\",\"id\":{fid:?},\"op\":\"start\"}}")
    );
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
        deadline: "2099-01-01T00:00:00Z".to_string(),
    };
    assert_eq!(
        rt.prepare_candidate(&input, deadline()).unwrap_err(),
        "candidate source preparation is not ready for this role and setup"
    );
    assert_eq!(mock.calls.borrow().len(), 2);
}

#[test]
fn coder_retry_clones_only_the_protected_original_bundle() {
    let source = format!("f{}", "c".repeat(24));
    let prep = preparation();
    let input = FactoryCandidate {
        preparation: prep.clone(),
        source_preparation: source.clone(),
        bundle: Vec::new(),
        deadline: "2099-01-01T00:00:00Z".into(),
    };
    let snapshot = || {
        vec![
            Ok(container_payload(&pid())),
            Ok(observation("ready", "soda-coder", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(format!("{{\"id\":{source:?},\"role\":\"soda-coder\",\"setup_digest\":{DIGEST:?},\"source_commit\":{COMMIT:?},\"credential\":\"\"}}").into_bytes()),
            Ok(b"check.sh\nsetup.sh\nsource.bundle\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(b"#!/bin/sh\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(b"#!/bin/sh\n".to_vec()),
        ]
    };
    let mut responses = snapshot();
    responses.extend(vec![
        Ok(b"0:0:644:1:regular file\n".to_vec()), // protected bundle stat
        Ok(b"BUNDLE".to_vec()),                   // original bundle read
        Ok(container_payload(&pid())),
        Ok(b"{}".to_vec()),
        Ok(approve_response(&prep.id)),
        Ok(observation("approved", "soda-coder", false, false)),
        Err("exit status 1".into()), // new checkout has no .git
        Ok(b"".to_vec()),            // no partial effects
        Ok(b"".to_vec()),            // clone into the new identity
        Ok(format!("{COMMIT}\n").into_bytes()),
        Ok(b"".to_vec()),
        Ok(b"1001\n".to_vec()),
        Ok(b"soda-coder\n".to_vec()),
        Ok(b"soda-coder\n".to_vec()),
        Ok(b"".to_vec()),
        Err("exit status 1".into()),
        Err("exit status 1".into()),
        Err("exit status 1".into()),
        Err("exit status 1".into()),
        Err("exit status 1".into()),
        Ok(b"".to_vec()),
        Ok(b"{}".to_vec()),
        Ok(b"{}".to_vec()),
        Ok(observation("running", "soda-coder", false, false)),
    ]);
    let mock = Mock::new(responses);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_candidate(&input, deadline()).unwrap().phase,
        "running"
    );
    let calls = mock.calls.borrow();
    assert_eq!(
        calls[11].2.last().unwrap(),
        &super::paths::preparation_paths("soda-coder", &source).2
    );
    assert!(
        String::from_utf8(calls[14].0.clone())
            .unwrap()
            .contains("\"bundle\":\"QlVORExF\"")
    );
    let clone = &calls[18].2;
    assert!(clone.iter().any(|arg| arg == "clone"));
    assert_eq!(
        clone.last().unwrap(),
        &format!("/home/soda-coder/checkouts/{}", prep.id)
    );
    assert!(!calls.iter().any(|call| {
        call.2
            .iter()
            .any(|arg| arg.contains(&format!("checkouts/{source}")) || arg == "reset")
    }));
    drop(calls);

    let mut changed = input.clone();
    changed.preparation.source_commit = "a".repeat(40);
    changed.preparation.requirements.source_commit = "a".repeat(40);
    let mock = Mock::new(snapshot());
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_candidate(&changed, deadline()).unwrap_err(),
        "coder retry must retain its exact approved source"
    );
    assert_eq!(mock.calls.borrow().len(), 10);

    let mut responses = snapshot();
    responses.push(Ok(b"0:0:600:1:regular file\n".to_vec()));
    let mock = Mock::new(responses);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_candidate(&input, deadline()).unwrap_err(),
        "candidate approved file is not protected"
    );
    assert_eq!(mock.calls.borrow().len(), 11);
}

#[test]
fn candidate_expired_deadline_refuses_before_native_io() {
    let mut prep = preparation();
    prep.role = "soda-reviewer".to_string();
    let input = FactoryCandidate {
        preparation: prep,
        source_preparation: format!("f{}", "c".repeat(24)),
        bundle: b"BUNDLE".to_vec(),
        deadline: "2000-01-01T00:00:00Z".to_string(),
    };
    let mock = Mock::new(Vec::new());
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(
        rt.prepare_candidate(&input, deadline()).unwrap_err(),
        "candidate preparation deadline has expired"
    );
    assert!(mock.calls.borrow().is_empty());
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
        deadline: "2099-01-01T00:00:00Z".to_string(),
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
