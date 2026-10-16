use super::inventory::{scan_mix, scan_order};
use super::tests::{
    companion_with, deadline, file_run, image_id, project_id, MockExec, MockTailnet,
};
use crate::tailnet_domain::{RunTarget, ERR_CONFLICT, ERR_INVALID, ERR_UNCONFIRMED};
use crate::tailnet_runtime::{companion_container_args, ProjectRun};
use std::cell::RefCell;
use std::collections::{BTreeSet, HashMap, HashSet};
use std::rc::Rc;

const INSTALLATION: &str = "550e8400-e29b-41d4-a716-446655440000";
const STARTED: &str = "2026-09-12T12:00:00Z";

fn hex_id(n: u64) -> String {
    format!("{n:064x}")
}

fn project_n(n: u64) -> String {
    format!("p{n:024x}")
}

struct ChildFixture {
    cid: String,
    project: String,
    run: String,
    parent: String,
    image: String,
    generation: String,
    running: bool,
}

fn inspect_json(fixture: &ChildFixture) -> Vec<u8> {
    let target = RunTarget {
        project: fixture.project.clone(),
        run: fixture.run.clone(),
        container: fixture.parent.clone(),
    };
    let args = companion_container_args(&target, &fixture.image, INSTALLATION, &fixture.generation)
        .unwrap();
    let mut command = vec!["/usr/bin/podman".to_string()];
    command.extend(args);
    serde_json::json!({
        "id": fixture.cid,
        "image": fixture.image,
        "command": command,
        "running": fixture.running,
        "pid": if fixture.running { 99 } else { 0 },
        "started": if fixture.running { STARTED } else { "" },
        "execs": Vec::<String>::new(),
        "labels": {
            "project": fixture.project,
            "run": fixture.run,
            "parent": fixture.parent,
            "installation": INSTALLATION,
            "generation": fixture.generation,
        },
    })
    .to_string()
    .into_bytes()
}

/// Mock podman surface: `ps` lists the live IDs, `inspect` serves the fixture
/// map, `stop` succeeds, `rm` retires from the live set and records removals.
fn harness_with(
    fixtures: Vec<ChildFixture>,
    rm_err: Option<String>,
) -> (MockExec, Rc<RefCell<Vec<String>>>) {
    let live: HashSet<String> = fixtures.iter().map(|f| f.cid.clone()).collect();
    let live = Rc::new(RefCell::new(live));
    let map: HashMap<String, ChildFixture> =
        fixtures.into_iter().map(|f| (f.cid.clone(), f)).collect();
    let fixtures = Rc::new(map);
    let rms = Rc::new(RefCell::new(Vec::new()));
    let live_clone = live.clone();
    let fixtures_clone = fixtures.clone();
    let rms_clone = rms.clone();
    let exec = MockExec::with_handler(move |cmd, args| {
        assert_eq!(cmd, "/usr/bin/podman");
        assert_eq!(args[0], "--remote=false");
        match args[1].as_str() {
            "ps" => Ok(live_clone
                .borrow()
                .iter()
                .cloned()
                .collect::<Vec<_>>()
                .join("\n")
                .into_bytes()),
            "inspect" => {
                let cid = args.last().unwrap();
                assert!(live_clone.borrow().contains(cid.as_str()));
                Ok(inspect_json(&fixtures_clone[cid.as_str()]))
            }
            "stop" => Ok(Vec::new()),
            "rm" => {
                let cid = args.last().unwrap().clone();
                if let Some(err) = &rm_err {
                    return Err(err.clone());
                }
                live_clone.borrow_mut().remove(&cid);
                rms_clone.borrow_mut().push(cid);
                Ok(Vec::new())
            }
            other => panic!("unexpected podman subcommand {other}"),
        }
    });
    (exec, rms.clone())
}

fn subcommand_calls(exec: &MockExec, sub: &str) -> Vec<Vec<String>> {
    exec.calls()
        .into_iter()
        .filter(|(_, args)| args.get(1).is_some_and(|s| s == sub))
        .map(|(_, args)| args)
        .collect()
}

#[test]
fn scan_mix_is_deterministic_and_total() {
    let ids: Vec<String> = (0..16).map(hex_id).collect();
    assert_eq!(scan_order(ids.clone()), scan_order(ids.clone()));
    // Unique keys survive exactly once.
    let ordered = scan_order(ids.clone());
    assert_eq!(ordered.len(), ids.len());
    let mut sorted = ordered.clone();
    sorted.sort();
    let mut want = ids.clone();
    want.sort();
    assert_eq!(sorted, want);
    // Distinct inputs mix to distinct order keys in practice.
    let mixes: BTreeSet<u64> = ids.iter().map(|id| scan_mix(id.as_str())).collect();
    assert_eq!(mixes.len(), ids.len());
}

#[test]
fn scan_order_scatters_a_fixed_sorted_prefix() {
    // 128 IDs forming a fixed lexicographic prefix, with obsolete candidates
    // trailing after it: the old sorted take(128) could never reach them.
    let mut all: Vec<String> = (0..128).map(hex_id).collect();
    let trailing = vec![hex_id(128), hex_id(129)];
    all.extend(trailing.clone());
    let mut sorted = all.clone();
    sorted.sort();
    assert_eq!(&sorted[..128], &all[..128]);
    assert_eq!(&sorted[128..], &trailing[..]);
    let mixed = scan_order(all);
    assert_ne!(mixed, sorted, "mixed order must not be the sorted prefix");
    for cid in &trailing {
        assert!(
            mixed[..128].contains(cid),
            "trailing obsolete candidate {cid} must stay reachable within the bound"
        );
    }
}

#[test]
fn bounded_inventory_reaches_trailing_obsolete_and_confirms_after_removal() {
    let project = project_n(1);
    let current_run = hex_id(1000);
    let old_run = hex_id(999);
    let parent = hex_id(500);
    let image = image_id();
    let mut fixtures: Vec<ChildFixture> = (0..127)
        .map(|n| ChildFixture {
            cid: hex_id(n),
            project: project.clone(),
            run: current_run.clone(),
            parent: parent.clone(),
            image: image.clone(),
            generation: "7".to_string(),
            running: true,
        })
        .collect();
    let obsolete: Vec<String> = vec![hex_id(127), hex_id(128)];
    for cid in &obsolete {
        fixtures.push(ChildFixture {
            cid: cid.clone(),
            project: project.clone(),
            run: old_run.clone(),
            parent: parent.clone(),
            image: image.clone(),
            generation: "7".to_string(),
            running: false,
        });
    }
    // One running obsolete incarnation: verified stopped only, never removed.
    let running_obsolete = hex_id(129);
    fixtures.push(ChildFixture {
        cid: running_obsolete.clone(),
        project: project.clone(),
        run: old_run.clone(),
        parent: parent.clone(),
        image: image.clone(),
        generation: "7".to_string(),
        running: true,
    });
    let (exec, rms) = harness_with(fixtures, None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let inventory = c.native_children(INSTALLATION, None, deadline());
    assert!(inventory.unknown, "130 entries exceed the 128 bound");
    assert_eq!(inventory.children.len(), 128);
    let found: BTreeSet<String> = inventory
        .children
        .iter()
        .map(|child| child.record.id.clone())
        .collect();
    for cid in &obsolete {
        assert!(
            found.contains(cid),
            "trailing obsolete {cid} must be reached"
        );
    }
    // Bounded retirement removes exactly the verified stopped obsolete pair:
    // running and current-run children are never named.
    let retired = c
        .retire_stopped_where(
            INSTALLATION,
            None,
            &|child| child.target.project == project && child.target.run != current_run,
            16,
            deadline(),
        )
        .unwrap();
    assert_eq!(retired, 2);
    let removed: BTreeSet<String> = rms.borrow().iter().cloned().collect();
    assert_eq!(removed, obsolete.iter().cloned().collect::<BTreeSet<_>>());
    assert!(!removed.contains(&running_obsolete));
    assert_eq!(
        subcommand_calls(&c.exec, "stop").len(),
        0,
        "already-stopped children need no stop call"
    );
    // The backlog shrank through the bounded pass: the next inventory is
    // complete with every current child surviving.
    let again = c.native_children(INSTALLATION, None, deadline());
    assert!(!again.unknown);
    assert_eq!(again.children.len(), 128);
    assert!(again.children.iter().all(|child| child.record.running));
}

#[test]
fn retire_verified_stopped_child_removes_only_after_reverification() {
    let project = project_id();
    let run = hex_id(7);
    let image = image_id();
    let fixture = ChildFixture {
        cid: hex_id(42),
        project: project.clone(),
        run: run.clone(),
        parent: hex_id(43),
        image: image.clone(),
        generation: "7".to_string(),
        running: false,
    };
    // Stopped child: stop is a verified no-op, rm follows reinspection.
    let (exec, rms) = harness_with(vec![fixture], None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let inventory = c.native_children(INSTALLATION, None, deadline());
    assert!(!inventory.unknown && inventory.children.len() == 1);
    c.retire_verified_stopped_child(&inventory.children[0], deadline())
        .unwrap();
    assert_eq!(subcommand_calls(&c.exec, "stop").len(), 0);
    assert_eq!(rms.borrow().len(), 1);
}

#[test]
fn retire_refuses_a_child_that_will_not_stay_stopped() {
    let project = project_id();
    let image = image_id();
    let fixture = ChildFixture {
        cid: hex_id(44),
        project: project.clone(),
        run: hex_id(8),
        parent: hex_id(45),
        image: image.clone(),
        generation: "7".to_string(),
        running: true,
    };
    let (exec, rms) = harness_with(vec![fixture], None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let inventory = c.native_children(INSTALLATION, None, deadline());
    assert_eq!(inventory.children.len(), 1);
    // Every inspection still reports running, so the stop cannot confirm and
    // removal is never attempted.
    assert_eq!(
        c.retire_verified_stopped_child(&inventory.children[0], deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
    assert_eq!(subcommand_calls(&c.exec, "stop").len(), 1);
    assert!(rms.borrow().is_empty());
}

#[test]
fn retire_reports_unconfirmed_when_removal_fails() {
    let project = project_id();
    let image = image_id();
    let fixture = ChildFixture {
        cid: hex_id(46),
        project: project.clone(),
        run: hex_id(9),
        parent: hex_id(47),
        image: image.clone(),
        generation: "7".to_string(),
        running: false,
    };
    let (exec, rms) = harness_with(vec![fixture], Some("rm failed".to_string()));
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let inventory = c.native_children(INSTALLATION, None, deadline());
    assert_eq!(
        c.retire_verified_stopped_child(&inventory.children[0], deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
    assert!(rms.borrow().is_empty());
}

#[test]
fn retire_refuses_identity_drift_before_removal() {
    let project = project_id();
    let run = hex_id(10);
    let parent = hex_id(11);
    let image = image_id();
    let drifted = format!("sha256:{}", "f".repeat(64));
    let first = ChildFixture {
        cid: hex_id(48),
        project: project.clone(),
        run: run.clone(),
        parent: parent.clone(),
        image: image.clone(),
        generation: "7".to_string(),
        running: false,
    };
    let second = ChildFixture {
        image: drifted,
        ..ChildFixture {
            cid: hex_id(48),
            project: project.clone(),
            run: run.clone(),
            parent: parent.clone(),
            image: image.clone(),
            generation: "7".to_string(),
            running: false,
        }
    };
    let live = Rc::new(RefCell::new(HashSet::from([hex_id(48)])));
    let rms = Rc::new(RefCell::new(Vec::new()));
    let calls = Rc::new(RefCell::new(0));
    let first_json = inspect_json(&first);
    let second_json = inspect_json(&second);
    let live_clone = live.clone();
    let rms_clone = rms.clone();
    let calls_clone = calls.clone();
    let exec = MockExec::with_handler(move |cmd, args| {
        assert_eq!(cmd, "/usr/bin/podman");
        match args[1].as_str() {
            "ps" => Ok(live_clone
                .borrow()
                .iter()
                .cloned()
                .collect::<Vec<_>>()
                .join("\n")
                .into_bytes()),
            "inspect" => {
                let mut n = calls_clone.borrow_mut();
                *n += 1;
                Ok(if *n == 1 {
                    first_json.clone()
                } else {
                    second_json.clone()
                })
            }
            "rm" => {
                rms_clone.borrow_mut().push(args.last().unwrap().clone());
                Ok(Vec::new())
            }
            other => panic!("unexpected podman subcommand {other}"),
        }
    });
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let inventory = c.native_children(INSTALLATION, None, deadline());
    // The stop-path inspection already sees the drifted image, so even the
    // stop refuses before any removal is considered.
    let err = c
        .retire_verified_stopped_child(&inventory.children[0], deadline())
        .unwrap_err();
    assert!(
        err.contains(ERR_CONFLICT),
        "identity drift must refuse, got {err}"
    );
    assert_eq!(*calls.borrow(), 2);
    assert!(rms.borrow().is_empty());
}

#[test]
fn retire_stopped_where_bounds_the_batch_and_keeps_no_ops_quiet() {
    let project = project_id();
    let current_run = hex_id(20);
    let old_run = hex_id(21);
    let parent = hex_id(22);
    let image = image_id();
    // Three stopped obsolete plus one stopped current-run child.
    let mut fixtures: Vec<ChildFixture> = (30..33)
        .map(|n| ChildFixture {
            cid: hex_id(n),
            project: project.clone(),
            run: old_run.clone(),
            parent: parent.clone(),
            image: image.clone(),
            generation: "7".to_string(),
            running: false,
        })
        .collect();
    let current_cid = hex_id(33);
    fixtures.push(ChildFixture {
        cid: current_cid.clone(),
        project: project.clone(),
        run: current_run.clone(),
        parent: parent.clone(),
        image: image.clone(),
        generation: "7".to_string(),
        running: false,
    });
    let (exec, rms) = harness_with(fixtures, None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    let retired = c
        .retire_stopped_where(
            INSTALLATION,
            None,
            &|child| child.target.project == project && child.target.run != current_run,
            2,
            deadline(),
        )
        .unwrap();
    assert_eq!(retired, 2);
    assert_eq!(rms.borrow().len(), 2);
    assert!(!rms.borrow().contains(&current_cid));
    // Empty inventory is a no-op: only the listing call runs.
    let (exec, _) = harness_with(Vec::new(), None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    assert_eq!(
        c.retire_stopped_where(INSTALLATION, None, &|_| true, 16, deadline()),
        Ok(0)
    );
    assert_eq!(c.exec.calls().len(), 1);
    // An invalid installation never selects: no native call at all.
    let (exec, _) = harness_with(Vec::new(), None);
    let c = companion_with(exec, MockTailnet::inert(), &image);
    assert_eq!(
        c.retire_stopped_where("not-a-uuid", None, &|_| true, 16, deadline()),
        Ok(0)
    );
    assert!(c.exec.calls().is_empty());
}

#[test]
fn reconcile_empty_previous_record_is_a_quiet_takeover() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let run = file_run();
    let empty = ProjectRun {
        target: RunTarget {
            project: String::new(),
            container: String::new(),
            run: String::new(),
        },
        pid: 0,
        started: String::new(),
        userns: String::new(),
        netns: String::new(),
        uid: 0,
        gid: 0,
        resolver: String::new(),
    };
    // Missing record with no previous run takes over freshness with no native
    // calls; an identical present record is a no-op with no retirement.
    assert_eq!(
        c.reconcile_previous_run(
            &project_id(),
            &run,
            &empty,
            Some("runtime record not found".to_string()),
            deadline()
        ),
        Ok(true)
    );
    assert!(c.exec.calls().is_empty());
    assert_eq!(
        c.reconcile_previous_run(&project_id(), &run, &run, None, deadline()),
        Ok(false)
    );
    assert!(c.exec.calls().is_empty());
}

#[test]
fn current_run_lookup_refuses_without_state() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(c.current_run_id("bad-id"), Err(ERR_INVALID.to_string()));
    assert!(c.exec.calls().is_empty());
    // No runtime record exists in the test environment: refusal, never absence.
    assert_eq!(
        c.current_run_id(&project_id()),
        Err("runtime record not found".to_string())
    );
    assert!(c.exec.calls().is_empty());
    // Error strings stay typed: no native output enters the chain.
    assert_eq!(
        c.current_run_id(&project_n(9)),
        Err("runtime record not found".to_string())
    );
}
