//! FW14 task-runner scope: typed events around real native children —
//! overlap with chained journal, release while active, failure, parent
//! stop and lifecycle refusals. The Go importer folds these same bytes.

use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

use crate::factory::*;
use crate::terminal::factory::children::{ChildProfile, ChildSpec};

static TAG: AtomicU64 = AtomicU64::new(0);

fn scratch(tag: &str) -> PathBuf {
    let dir = PathBuf::from(std::env::var("HOME").unwrap()).join(format!(
        ".cache/fw14-tasks-{}-{}-{}",
        tag,
        std::process::id(),
        TAG.fetch_add(1, Ordering::SeqCst)
    ));
    std::fs::create_dir_all(dir.join("work")).unwrap();
    dir
}

fn profile() -> ChildProfile {
    ChildProfile {
        effort: "xhigh".to_string(),
        contributor: true,
        role: "soda-coder".to_string(),
        account_home: "/home/soda-coder".to_string(),
    }
}

fn runner(dir: &Path, max_children: i64) -> TaskRunner {
    TaskRunner::open(
        &"a".repeat(32),
        &"09".repeat(16),
        &"ab".repeat(32),
        "muse",
        &profile(),
        max_children,
        dir,
    )
    .unwrap()
}

fn spec(dir: &Path, task: &str, argv: &[&str]) -> ChildSpec {
    ChildSpec {
        task_key: task.to_string(),
        argv: argv.iter().map(|s| s.to_string()).collect(),
        workdir: dir.join("work"),
    }
}

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn cleanup(dir: &Path) {
    let _ = std::fs::remove_dir_all(dir);
}

fn output_digest() -> String {
    "cd".repeat(32)
}

/// FW14: two independent children overlap under one journal, a completion
/// releases a dependent while the unrelated child stays active, and parent
/// stop retires everything with a terminal event. Real processes.
#[test]
fn overlap_record_release_and_stop() {
    let dir = scratch("overlap");
    let mut runner = runner(&dir, 3);
    let slow = spec(&dir, "slow-lane", &["/bin/sleep", "5"]);
    let fast = spec(&dir, "fast-lane", &["/bin/echo", "fast-output"]);
    let slow_child = runner.start_task(&slow, deadline()).unwrap();
    let fast_child = runner.start_task(&fast, deadline()).unwrap();
    assert_ne!(slow_child, fast_child);
    assert_eq!(runner.active_child_count(), 2);
    // The journal chains intent+started per child with contiguous
    // sequences binding run, invocation and plan digest.
    let journal = runner.journal_snapshot();
    assert_eq!(journal.len(), 4);
    let mut prev = String::new();
    for (index, event) in journal.iter().enumerate() {
        assert!(event.validate().is_ok());
        assert_eq!(event.sequence, index as i64 + 1);
        assert_eq!(event.prev_digest, prev);
        assert_eq!(event.invocation, "09".repeat(16));
        assert_eq!(event.plan_digest, "ab".repeat(32));
        prev = event.digest();
    }
    assert_eq!(journal[0].event_type, "child_start_intent");
    assert_eq!(journal[1].event_type, "child_started");
    assert_eq!(journal[1].child_id, slow_child);
    // Fast completes with its output digest while slow stays active; the
    // dependent releases immediately with no unrelated-lane barrier.
    assert_eq!(
        runner
            .join_task("fast-lane", Duration::from_secs(10))
            .unwrap(),
        0
    );
    assert_eq!(
        runner.report_task("fast-lane", &output_digest()).unwrap(),
        0
    );
    runner
        .checkpoint_task("fast-lane", &["check-one".to_string()])
        .unwrap();
    runner
        .complete_task("fast-lane", Some(&"be".repeat(20)), Some(&"ef".repeat(20)))
        .unwrap();
    let downstream = spec(&dir, "downstream", &["/bin/echo", "downstream-output"]);
    runner.start_task(&downstream, deadline()).unwrap();
    assert_eq!(
        runner
            .join_task("downstream", Duration::from_secs(10))
            .unwrap(),
        0
    );
    assert_eq!(
        runner.report_task("downstream", &output_digest()).unwrap(),
        0
    );
    // Parent stop retires the slow child and seals the journal.
    let report = runner.stop("slow-lane", "parent stop confirmed").unwrap();
    assert!(report.uncertain.is_empty());
    assert_eq!(report.retired.len(), 1);
    let journal = runner.journal_snapshot();
    let stopped = journal.last().unwrap();
    assert_eq!(stopped.event_type, "run_stopped");
    let (page, next, terminal) = {
        let mut journal = EventJournal::default();
        for event in &runner.journal_snapshot() {
            journal.append(event).unwrap();
        }
        journal.query(0, 24576 - 256).unwrap()
    };
    assert!(terminal);
    assert_eq!(next, journal.len() as i64 + 1);
    assert_eq!(page.len(), journal.len());
    assert!(runner
        .start_task(&spec(&dir, "late-task", &["/bin/echo", "late"]), deadline())
        .is_err());
    cleanup(&dir);
}

/// FW14: a terminal failure records its task without retiring siblings;
/// the caller stops the run from the active set.
#[test]
fn failure_records_task_and_lists_active() {
    let dir = scratch("failure");
    let mut runner = runner(&dir, 3);
    runner
        .start_task(&spec(&dir, "doomed", &["/bin/sleep", "60"]), deadline())
        .unwrap();
    runner
        .start_task(&spec(&dir, "witness", &["/bin/sleep", "60"]), deadline())
        .unwrap();
    runner
        .fail_task("doomed", "check-one refused the output")
        .unwrap();
    assert_eq!(runner.active_tasks(), vec!["witness".to_string()]);
    let journal = runner.journal_snapshot();
    let failed = journal.last().unwrap();
    assert_eq!(failed.event_type, "task_failed");
    assert_eq!(failed.task_key, "doomed");
    assert_eq!(failed.detail, "check-one refused the output");
    let report = runner.stop("witness", "failure containment").unwrap();
    assert!(report.uncertain.is_empty());
    assert_eq!(report.retired.len(), 2);
    cleanup(&dir);
}

/// FW14: lifecycle refusals keep the journal honest. A lost start reply is
/// looked up — the same child returns with no new events.
#[test]
fn lifecycle_refusals_and_start_lookup() {
    let dir = scratch("refusals");
    let mut runner = runner(&dir, 2);
    let first = spec(&dir, "only-task", &["/bin/echo", "hi"]);
    let child = runner.start_task(&first, deadline()).unwrap();
    let before = runner.journal_snapshot().len();
    assert_eq!(runner.start_task(&first, deadline()).unwrap(), child);
    assert_eq!(runner.journal_snapshot().len(), before);
    // A live child is progress, not a report.
    let live = spec(&dir, "live-task", &["/bin/sleep", "30"]);
    runner.start_task(&live, deadline()).unwrap();
    assert!(runner.report_task("live-task", &output_digest()).is_err());
    // Only a reported task completes; only an executing task fails.
    assert_eq!(
        runner
            .join_task("only-task", Duration::from_secs(10))
            .unwrap(),
        0
    );
    assert!(runner.complete_task("only-task", None, None).is_err());
    assert_eq!(
        runner.report_task("only-task", &output_digest()).unwrap(),
        0
    );
    assert!(runner.fail_task("unknown-task", "nope").is_err());
    assert!(runner
        .start_task(&spec(&dir, "Bad-Key", &["/bin/echo", "x"]), deadline())
        .is_err());
    assert!(runner
        .report_task("live-task", "prose, not a digest")
        .is_err());
    assert!(runner
        .checkpoint_task("only-task", &["Not A Check".to_string()])
        .is_err());
    assert!(runner
        .complete_task("live-task", Some("short"), None)
        .is_err());
    let _ = runner.stop("only-task", "done");
    assert!(runner
        .start_task(&spec(&dir, "late-task", &["/bin/echo", "x"]), deadline())
        .is_err());
    cleanup(&dir);
}

/// The journal encoding round-trips through the strict decoder with Go
/// wire field names, so the Go importer parses these exact bytes.
#[test]
fn journal_encoding_matches_go_wire() {
    let dir = scratch("wire");
    let mut runner = runner(&dir, 1);
    runner
        .start_task(&spec(&dir, "wire-task", &["/bin/echo", "wire"]), deadline())
        .unwrap();
    for event in runner.journal_snapshot() {
        let encoded = event.encode();
        for field in [
            "\"sequence\"",
            "\"prev_digest\"",
            "\"type\"",
            "\"task_key\"",
            "\"invocation\"",
            "\"plan_digest\"",
        ] {
            assert!(encoded.contains(field), "missing {field} in {encoded}");
        }
        let decoded = TaskEvent::decode(encoded.as_bytes()).unwrap();
        assert_eq!(decoded, event);
        assert!(decoded.validate().is_ok());
    }
    let _ = runner.stop("wire-task", "done");
    cleanup(&dir);
}

#[test]
fn open_rejects_bad_bindings() {
    let dir = scratch("admit");
    let bad_profile = ChildProfile {
        effort: "turbo".to_string(),
        contributor: false,
        role: "soda-coder".to_string(),
        account_home: "/home/soda-coder".to_string(),
    };
    assert!(TaskRunner::open(
        "short",
        &"09".repeat(16),
        &"ab".repeat(32),
        "muse",
        &profile(),
        3,
        &dir
    )
    .is_err());
    assert!(TaskRunner::open(
        &"a".repeat(32),
        "",
        &"ab".repeat(32),
        "muse",
        &profile(),
        3,
        &dir
    )
    .is_err());
    assert!(TaskRunner::open(
        &"a".repeat(32),
        &"09".repeat(16),
        "short",
        "muse",
        &profile(),
        3,
        &dir
    )
    .is_err());
    assert!(TaskRunner::open(
        &"a".repeat(32),
        &"09".repeat(16),
        &"ab".repeat(32),
        "turbo-harness",
        &profile(),
        3,
        &dir
    )
    .is_err());
    assert!(TaskRunner::open(
        &"a".repeat(32),
        &"09".repeat(16),
        &"ab".repeat(32),
        "muse",
        &bad_profile,
        3,
        &dir
    )
    .is_err());
    assert!(TaskRunner::open(
        &"a".repeat(32),
        &"09".repeat(16),
        &"ab".repeat(32),
        "muse",
        &profile(),
        5,
        &dir
    )
    .is_err());
    cleanup(&dir);
}
