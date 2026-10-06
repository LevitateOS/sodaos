use super::common::{
    container_id, deadline, prep_id, project_id, run_id, sample_launch, wired_factory,
};
use super::mocks::script_success;
use crate::factory::*;

#[test]
fn takeover_success_and_refusals() {
    let (_dir, factory, _exec, term, broker) = wired_factory("takeover-ok");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    let dest = format!("/home/alice/factory-takeover/{}", req.run.id);
    term.takeover
        .borrow_mut()
        .push_back(Ok((dest.clone(), false)));
    let result = factory
        .takeover(
            &FactoryTakeover {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
                member: "alice".to_string(),
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(result.destination, dest);
    assert!(!result.reused);
    assert_eq!(
        result.encode(),
        format!(
            "{{\"id\":\"{}\",\"project\":\"{}\",\"member\":\"alice\",\"destination\":\"{dest}\",\"reused\":false}}",
            req.run.id, req.run.project
        )
    );
    let calls = term.takeover_calls.borrow();
    assert_eq!(calls.len(), 1);
    assert_eq!(
        calls[0],
        (
            req.run.project.clone(),
            container_id(),
            "soda-coder".to_string(),
            req.run.preparation.clone(),
            "alice".to_string(),
            req.run.id.clone()
        )
    );

    // A run that never retired refuses.
    let (_dir, factory, _exec, _term, broker) = wired_factory("takeover-busy");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::Busy));
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .takeover(
                &FactoryTakeover {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    member: "alice".to_string(),
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "factory run is not retired for takeover"
    );
    // Unknown runs are not found.
    assert_eq!(
        factory
            .takeover(
                &FactoryTakeover {
                    project: req.run.project.clone(),
                    id: "b".repeat(32),
                    member: "alice".to_string(),
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::NotFound
    );
    // A wrong destination from the boundary fails validation.
    let (_dir, factory, _exec, term, broker) = wired_factory("takeover-baddest");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    term.takeover
        .borrow_mut()
        .push_back(Ok(("/elsewhere".to_string(), false)));
    assert_eq!(
        factory
            .takeover(
                &FactoryTakeover {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    member: "alice".to_string(),
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "takeover destination does not match its identities"
    );
    // Bad member.
    assert_eq!(
        FactoryTakeover {
            project: project_id(),
            id: run_id(),
            member: "root".to_string(),
        }
        .validate()
        .unwrap_err(),
        "invalid takeover member"
    );
}

#[test]
fn output_without_binding_reports_phase_only() {
    let (_dir, factory, _exec, _term, broker) = wired_factory("output-nobind");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::Busy));
    factory.launch(&req, deadline()).unwrap();
    // No output script: unbound runs never reach the boundary.
    let state = factory
        .output(
            &FactoryOutput {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
                offset: 0,
                limit: 100,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(state.phase, "approved");
    assert!(!state.terminal);
    assert!(state.container.is_empty());
    assert!(state.data.is_empty());
    assert_eq!(state.encode(), format!(
        "{{\"live\":false,\"terminal\":false,\"truncated\":false,\"gap\":false,\"id\":\"{}\",\"project\":\"{}\",\"phase\":\"approved\",\"container\":\"\",\"unit\":\"\",\"invocation\":\"\",\"total\":0,\"offset\":0,\"next\":0,\"data\":\"\",\"reason\":\"broker-busy\"}}",
        req.run.id, req.run.project
    ));
}

#[test]
fn output_slice_reports_binding_and_cursors() {
    let (_dir, factory, _exec, term, broker) = wired_factory("output-slice");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    term.output.borrow_mut().push_back(Ok(OutputSlice {
        data: b"hello".to_vec(),
        total: 100,
        offset: 40,
        truncated: true,
        gap: false,
    }));
    let state = factory
        .output(
            &FactoryOutput {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
                offset: 40,
                limit: 100,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(state.phase, "completed");
    assert!(state.terminal);
    assert!(!state.live);
    assert_eq!(state.container, container_id());
    assert_eq!(state.unit, factory_unit_name(&req.run.id));
    assert_eq!(state.invocation, "09".repeat(16));
    assert_eq!(state.total, 100);
    assert_eq!(state.offset, 40);
    assert_eq!(state.next, 45);
    assert!(state.truncated);
    assert_eq!(state.data, "aGVsbG8=");
    assert_eq!(state.exit_code, Some(0));
    {
        let calls = term.output_calls.borrow();
        assert_eq!(calls.as_slice(), &[(req.run.project.clone(), 40, 100)]);
    }
    // Unknown runs are not found; stale incarnations refuse.
    assert_eq!(
        factory
            .output(
                &FactoryOutput {
                    project: req.run.project.clone(),
                    id: "b".repeat(32),
                    offset: 0,
                    limit: 1,
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::NotFound
    );
    term.output.borrow_mut().push_back(Err(FactoryError::Stale));
    assert_eq!(
        factory
            .output(
                &FactoryOutput {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    offset: 0,
                    limit: 1,
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::Stale
    );
    // Cursor validation.
    assert_eq!(
        FactoryOutput {
            project: project_id(),
            id: run_id(),
            offset: -1,
            limit: 1,
        }
        .validate()
        .unwrap_err(),
        "invalid output cursor"
    );
    assert_eq!(
        FactoryOutput {
            project: project_id(),
            id: run_id(),
            offset: 0,
            limit: 0,
        }
        .validate()
        .unwrap_err(),
        "invalid output read bound"
    );
}

#[test]
fn export_success_denied_and_bounds() {
    let (_dir, factory, _exec, term, broker) = wired_factory("export-ok");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    let candidate = "1".repeat(40);
    term.export
        .borrow_mut()
        .push_back(Ok(b"bundle-bytes".to_vec()));
    let state = factory
        .export(
            &FactoryExport {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
                role: "soda-coder".to_string(),
                preparation: req.run.preparation.clone(),
                candidate: candidate.clone(),
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(state.phase, "completed");
    assert_eq!(state.container, container_id());
    assert_eq!(state.candidate, candidate);
    assert_eq!(state.bundle, crate::ssh::b64_encode(b"bundle-bytes"));
    {
        let calls = term.export_calls.borrow();
        assert_eq!(
            calls.as_slice(),
            &[(
                req.run.project.clone(),
                container_id(),
                "soda-coder".to_string(),
                req.run.preparation.clone(),
                candidate.clone()
            )]
        );
    }
    // Role mismatch is denied (the daemon maps it to a generic 500).
    assert_eq!(
        factory
            .export(
                &FactoryExport {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    role: "soda-reviewer".to_string(),
                    preparation: req.run.preparation.clone(),
                    candidate: candidate.clone(),
                },
                deadline()
            )
            .unwrap_err(),
        FactoryError::Denied
    );
    // Empty and oversized bundles exceed bounds.
    term.export.borrow_mut().push_back(Ok(Vec::new()));
    let export_req = FactoryExport {
        project: req.run.project.clone(),
        id: req.run.id.clone(),
        role: "soda-coder".to_string(),
        preparation: req.run.preparation.clone(),
        candidate: candidate.clone(),
    };
    assert_eq!(
        factory.export(&export_req, deadline()).unwrap_err(),
        FactoryError::ExportBounds
    );
    term.export
        .borrow_mut()
        .push_back(Ok(vec![0u8; MAX_FACTORY_EXPORT_BUNDLE + 1]));
    assert_eq!(
        factory.export(&export_req, deadline()).unwrap_err(),
        FactoryError::ExportBounds
    );
    // Terminal verdicts propagate for the route layer to map.
    term.export
        .borrow_mut()
        .push_back(Err(FactoryError::ExportCandidate));
    assert_eq!(
        factory.export(&export_req, deadline()).unwrap_err(),
        FactoryError::ExportCandidate
    );
    // Unsettled runs refuse.
    let (_dir, factory, _exec, _term, broker) = wired_factory("export-busy");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::Busy));
    factory.launch(&req, deadline()).unwrap();
    assert_eq!(
        factory
            .export(
                &FactoryExport {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    role: "soda-coder".to_string(),
                    preparation: req.run.preparation.clone(),
                    candidate: "1".repeat(40),
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "factory run is not settled for export"
    );
    // Request validation.
    assert_eq!(
        FactoryExport {
            project: "x".to_string(),
            id: run_id(),
            role: "soda-coder".to_string(),
            preparation: prep_id(),
            candidate: "1".repeat(40),
        }
        .validate()
        .unwrap_err(),
        "invalid factory export address"
    );
    assert_eq!(
        FactoryExport {
            project: project_id(),
            id: run_id(),
            role: "root".to_string(),
            preparation: prep_id(),
            candidate: "1".repeat(40),
        }
        .validate()
        .unwrap_err(),
        "invalid factory export role"
    );
}
