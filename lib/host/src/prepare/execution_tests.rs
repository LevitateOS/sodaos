//! Prepare execution cases: full chain, short-circuit recovery,
//! blocked start, and the mixed inspect/stop/hold paths.

use super::tests::{
    approve_response, container_payload, deadline, fid, observation, pid, preparation, setup,
    test_config, Mock, COMMIT, DIGEST,
};
use super::*;

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
