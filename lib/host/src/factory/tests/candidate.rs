use super::common::{
    container_id, deadline, preparation_target, project_id, sample_launch, wired_factory,
    wired_factory_exec,
};
use super::mocks::script_success;
use crate::factory::*;

#[test]
fn candidate_script_pins_cutover_shape() {
    // The Go oracle is retired at executor cutover: the embedded
    // script is canonical now, pinned by its security-critical shape
    // (behavior stays covered by inspect_candidate_reports_clean_and_dirty).
    for marker in [
        "set -eu\numask 077",
        "core.hooksPath=/dev/null",
        "GIT_OBJECT_DIRECTORY=\"$src/.git/objects\"",
        "trap '/usr/bin/rm -rf \"$dir\"' EXIT HUP INT TERM",
        "diff-index --cached --quiet --no-ext-diff --no-textconv",
        "diff-files --quiet --no-ext-diff --no-textconv",
        "printf '%s %s\\n' \"$head\" \"$dirty\"",
    ] {
        assert!(
            CANDIDATE_INSPECT_SCRIPT.contains(marker),
            "candidate script lost {marker:?}"
        );
    }
}

#[test]
fn inspect_candidate_reports_clean_and_dirty() {
    for (word, dirty) in [("clean", false), ("dirty", true)] {
        let candidate = "2".repeat(40);
        let (_dir, factory, exec, term, broker) = wired_factory_exec(
            "candidate-ok",
            vec![
                Ok(preparation_target(&project_id(), &container_id())),
                Ok(format!("{candidate} {word}\n").into_bytes()),
            ],
        );
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        let state = factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.id, req.run.id);
        assert_eq!(state.project, req.run.project);
        assert_eq!(state.container, container_id());
        assert_eq!(state.candidate, candidate);
        assert_eq!(state.dirty, dirty);
        assert_eq!(
            state.encode(),
            format!(
                "{{\"id\":\"{}\",\"project\":\"{}\",\"container\":\"{}\",\"candidate\":\"{candidate}\",\"dirty\":{dirty}}}",
                req.run.id,
                req.run.project,
                container_id()
            )
        );
        // Full argv vectors, byte-exact.
        let calls = exec.calls.borrow();
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0].1, "/usr/bin/podman");
        assert_eq!(
            calls[0].2,
            vec![
                "--remote=false".to_string(),
                "inspect".to_string(),
                "--format".to_string(),
                crate::project::PROJECT_INSPECT_FORMAT.to_string(),
                format!("soda-{}", req.run.project),
            ]
        );
        assert_eq!(calls[1].1, "/usr/bin/podman");
        assert_eq!(
            calls[1].2,
            vec![
                "exec".to_string(),
                "--user".to_string(),
                "soda-coder".to_string(),
                container_id(),
                "/usr/bin/env".to_string(),
                "-i".to_string(),
                "PATH=/usr/bin:/bin".to_string(),
                "HOME=/home/soda-coder".to_string(),
                "LC_ALL=C".to_string(),
                "TMPDIR=/home/soda-coder/checkouts".to_string(),
                "GIT_CONFIG_NOSYSTEM=1".to_string(),
                "GIT_CONFIG_GLOBAL=/dev/null".to_string(),
                "GIT_NO_REPLACE_OBJECTS=1".to_string(),
                "GIT_TERMINAL_PROMPT=0".to_string(),
                "GIT_OPTIONAL_LOCKS=0".to_string(),
                "/usr/bin/sh".to_string(),
                "-c".to_string(),
                CANDIDATE_INSPECT_SCRIPT.to_string(),
                "soda-candidate".to_string(),
                format!("/home/soda-coder/checkouts/{}", req.run.preparation),
            ]
        );
        assert!(calls[0].0.is_empty() && calls[1].0.is_empty());
    }
}

#[test]
fn inspect_candidate_incarnation_matrix_is_stale() {
    // A replacement container refuses.
    let (_dir, factory, _exec, term, broker) = wired_factory_exec(
        "candidate-replaced",
        vec![Ok(preparation_target(&project_id(), &"e".repeat(64)))],
    );
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::Stale
    );
    // Any observation failure refuses stale too.
    let (_dir, factory, _exec, term, broker) =
        wired_factory_exec("candidate-unobs", vec![Err("podman blew up".to_string())]);
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::Stale
    );
    // Unsettled runs refuse before any exec.
    let (_dir, factory, _exec, _term, broker) = wired_factory("candidate-busy");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::Busy));
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "candidate inspection requires a settled run"
    );
    // Unknown runs are not found.
    assert_eq!(
        factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: "b".repeat(32),
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::NotFound
    );
}

#[test]
fn inspect_candidate_output_shapes() {
    // (raw bytes, expected error or dirty flag)
    let commit = "2".repeat(40);
    let cases: Vec<(Vec<u8>, Result<bool, &str>)> = vec![
        (format!("{commit} clean\n").into_bytes(), Ok(false)),
        (format!("  {commit}   dirty  \n").into_bytes(), Ok(true)),
        (
            vec![b'x'; 65],
            Err("candidate checkout inspection unconfirmed"),
        ),
        (
            b"nope\n".to_vec(),
            Err("candidate checkout observation is invalid"),
        ),
        (
            format!("{} bogus\n", "2".repeat(40)).into_bytes(),
            Err("candidate checkout observation is invalid"),
        ),
        (
            format!("{} clean extra\n", "2".repeat(40)).into_bytes(),
            Err("candidate checkout observation is invalid"),
        ),
        (
            b"\n".to_vec(),
            Err("candidate checkout observation is invalid"),
        ),
    ];
    for (raw, expected) in cases {
        let (_dir, factory, _exec, term, broker) = wired_factory_exec(
            "candidate-shape",
            vec![
                Ok(preparation_target(&project_id(), &container_id())),
                Ok(raw),
            ],
        );
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        let out = factory.inspect_candidate(
            &FactoryCandidateInspect {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
            },
            deadline(),
        );
        match expected {
            Ok(dirty) => assert_eq!(out.unwrap().dirty, dirty),
            Err(msg) => assert_eq!(out.unwrap_err().message(), msg),
        }
    }
    // Podman failure on the script exec is unconfirmed.
    let (_dir, factory, _exec, term, broker) = wired_factory_exec(
        "candidate-execfail",
        vec![
            Ok(preparation_target(&project_id(), &container_id())),
            Err("podman blew up".to_string()),
        ],
    );
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "candidate checkout inspection unconfirmed"
    );
}
