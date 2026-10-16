//! T13/T16 daemon scope: typed task-event record and paged reads against
//! a running implement receipt. Chain binding, duplicate idempotency,
//! byte bounds, wrong-identity refusal and the terminal cursor are all
//! proved here; the P1 fixture proves the native spawn shapes beneath.

use super::common::*;
use crate::factory::*;

fn implement_run() -> FactoryRun {
    let mut run = sample_run();
    run.execution_phase = EXECUTION_IMPLEMENT.to_string();
    run.manifest_digest = "1".repeat(64);
    run
}

fn event(sequence: i64, prev: &str, event_type: &str, task: &str) -> TaskEvent {
    TaskEvent {
        sequence,
        prev_digest: prev.to_string(),
        event_type: event_type.to_string(),
        task_key: task.to_string(),
        child_id: if event_type == TASK_EVENT_CHILD_START_INTENT {
            String::new()
        } else {
            "b".repeat(32)
        },
        invocation: "09".repeat(16),
        plan_digest: "1".repeat(64),
        detail: String::new(),
    }
}

#[test]
fn record_and_page_chained_events() {
    let (dir, factory, _, _, _) = wired_factory("task-events-page");
    let run = implement_run();
    write_running(&dir, &run, true);
    let first = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .unwrap();
    let second = event(2, &first.digest(), TASK_EVENT_CHILD_STARTED, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &second, deadline())
        .unwrap();
    // Exact duplicate of the head is a no-op: a lost reply is looked up,
    // not replayed.
    factory
        .record_task_event(&run.project, &run.id, &second, deadline())
        .unwrap();
    let req = FactoryTaskEvents {
        project: run.project.clone(),
        id: run.id.clone(),
        after_sequence: 0,
        byte_limit: 4096,
    };
    let page = factory.task_events(&req, deadline()).unwrap();
    assert_eq!(page.events.len(), 2);
    assert_eq!(page.next_sequence, 3);
    assert!(!page.terminal);
    assert!(confirm_factory_task_events(&req, &page).is_ok());
    // Cursor paging: the second page holds nothing new but still advances.
    let req = FactoryTaskEvents {
        after_sequence: 2,
        ..req
    };
    let page = factory.task_events(&req, deadline()).unwrap();
    assert!(page.events.is_empty());
    assert_eq!(page.next_sequence, 3);
}

#[test]
fn record_refuses_wrong_identities() {
    let (dir, factory, _, _, _) = wired_factory("task-events-refuse");
    let run = implement_run();
    write_running(&dir, &run, true);
    // Unknown run: fenced, never fabricated.
    let first = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    assert!(factory
        .record_task_event(&run.project, &"c".repeat(32), &first, deadline())
        .is_err());
    assert!(factory
        .task_events(
            &FactoryTaskEvents {
                project: run.project.clone(),
                id: "c".repeat(32),
                after_sequence: 0,
                byte_limit: 4096,
            },
            deadline()
        )
        .is_err());
    // Wrong plan digest: agent bytes cannot forge accepted events.
    let mut forged = first.clone();
    forged.plan_digest = "2".repeat(64);
    assert!(factory
        .record_task_event(&run.project, &run.id, &forged, deadline())
        .is_err());
    // Wrong invocation: a replay under another incarnation refuses.
    let mut replayed = first.clone();
    replayed.invocation = "08".repeat(16);
    assert!(factory
        .record_task_event(&run.project, &run.id, &replayed, deadline())
        .is_err());
    // Sequence gap and chain break refuse without advancing.
    let gap = event(3, "", TASK_EVENT_CHILD_STARTED, "design-api");
    assert!(factory
        .record_task_event(&run.project, &run.id, &gap, deadline())
        .is_err());
    factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .unwrap();
    let mut conflict = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    conflict.detail = "different bytes".to_string();
    assert!(factory
        .record_task_event(&run.project, &run.id, &conflict, deadline())
        .is_err());
    let mut broken = event(2, &"0".repeat(64), TASK_EVENT_CHILD_STARTED, "design-api");
    broken.child_id = "b".repeat(32);
    assert!(factory
        .record_task_event(&run.project, &run.id, &broken, deadline())
        .is_err());
}

#[test]
fn record_refuses_non_implement_run() {
    let (dir, factory, _, _, _) = wired_factory("task-events-phase");
    let run = sample_run();
    write_running(&dir, &run, true);
    let first = TaskEvent {
        sequence: 1,
        prev_digest: String::new(),
        event_type: TASK_EVENT_CHILD_START_INTENT.to_string(),
        task_key: "design-api".to_string(),
        child_id: String::new(),
        invocation: "09".repeat(16),
        plan_digest: "0".repeat(64),
        detail: String::new(),
    };
    assert!(factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .is_err());
}

#[test]
fn terminal_event_ends_stream_but_keeps_history() {
    let (dir, factory, _, _, _) = wired_factory("task-events-terminal");
    let run = implement_run();
    write_running(&dir, &run, true);
    let first = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .unwrap();
    let stopped = event(2, &first.digest(), TASK_EVENT_RUN_STOPPED, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &stopped, deadline())
        .unwrap();
    let req = FactoryTaskEvents {
        project: run.project.clone(),
        id: run.id.clone(),
        after_sequence: 0,
        byte_limit: 4096,
    };
    let page = factory.task_events(&req, deadline()).unwrap();
    assert!(page.terminal);
    assert_eq!(page.events.len(), 2);
    // Nothing appends past the terminal event.
    let late = event(
        3,
        &stopped.digest(),
        TASK_EVENT_CHILD_START_INTENT,
        "code-task",
    );
    assert!(factory
        .record_task_event(&run.project, &run.id, &late, deadline())
        .is_err());
}

#[test]
fn tiny_page_is_a_bounded_size_error() {
    let (dir, factory, _, _, _) = wired_factory("task-events-size");
    let run = implement_run();
    write_running(&dir, &run, true);
    let first = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .unwrap();
    let req = FactoryTaskEvents {
        project: run.project.clone(),
        id: run.id.clone(),
        after_sequence: 0,
        byte_limit: 1,
    };
    assert!(factory.task_events(&req, deadline()).is_err());
    // The cursor did not advance: a full page still serves sequence 1.
    let req = FactoryTaskEvents {
        byte_limit: 4096,
        ..req
    };
    let page = factory.task_events(&req, deadline()).unwrap();
    assert_eq!(page.events.len(), 1);
    assert_eq!(page.next_sequence, 2);
}

#[test]
fn journal_survives_daemon_restart() {
    let (dir, factory, _, _, _) = wired_factory("task-events-restart");
    let run = implement_run();
    write_running(&dir, &run, true);
    let first = event(1, "", TASK_EVENT_CHILD_START_INTENT, "design-api");
    factory
        .record_task_event(&run.project, &run.id, &first, deadline())
        .unwrap();
    drop(factory);
    let factory = dummy_factory(&dir);
    let page = factory
        .task_events(
            &FactoryTaskEvents {
                project: run.project.clone(),
                id: run.id.clone(),
                after_sequence: 0,
                byte_limit: 4096,
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(page.events.len(), 1);
    assert_eq!(page.events[0].digest(), first.digest());
}
