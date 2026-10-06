use std::os::unix::fs::PermissionsExt;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::mocks::{FakeBroker, FakeExec, FakeTerminal};
use crate::factory::receipt::FactoryReceipt;
use crate::factory::*;
use crate::preparation::{self, Preparation, PrepareState};

pub(in crate::factory) static TAG_COUNTER: AtomicU64 = AtomicU64::new(0);

pub(in crate::factory) fn test_state_dir(tag: &str) -> std::path::PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "soda-pf26-{}-{}-{}",
        tag,
        std::process::id(),
        TAG_COUNTER.fetch_add(1, Ordering::SeqCst)
    ));
    std::fs::create_dir_all(&dir).unwrap();
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
    dir
}

pub(in crate::factory) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

/// Inverse of `days_from_civil` for test deadlines.
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719468;
    let era = if z >= 0 { z } else { z - 146096 } / 146097;
    let doe = z - era * 146097;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32;
    (if m <= 2 { y + 1 } else { y }, m, d)
}

/// RFC3339 `Z` deadline `offset_secs` in the future.
pub(in crate::factory) fn deadline_text(offset_secs: i64) -> String {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
        + offset_secs;
    let (y, m, d) = civil_from_days(now.div_euclid(86400));
    let secs = now.rem_euclid(86400);
    format!(
        "{y:04}-{m:02}-{d:02}T{:02}:{:02}:{:02}Z",
        secs / 3600,
        secs % 3600 / 60,
        secs % 60
    )
}

pub(in crate::factory) fn run_id() -> String {
    "a".repeat(32)
}

pub(in crate::factory) fn project_id() -> String {
    format!("p{}", "b".repeat(24))
}

pub(in crate::factory) fn prep_id() -> String {
    format!("f{}", "c".repeat(24))
}

pub(in crate::factory) fn container_id() -> String {
    "d".repeat(64)
}

pub(in crate::factory) fn sample_run() -> FactoryRun {
    let prompt = b"do the thing";
    FactoryRun {
        deadline: deadline_text(3600),
        actor: 7,
        id: run_id(),
        project: project_id(),
        role: "soda-coder".to_string(),
        preparation: prep_id(),
        harness: "codex".to_string(),
        harness_vers: "v1".to_string(),
        model: String::new(),
        assignment: crate::sha256::hex_lower(&crate::sha256::digest(prompt)),
        source_commit: "e".repeat(40),
        connection: "conn-1".to_string(),
    }
}

pub(in crate::factory) fn sample_launch() -> FactoryLaunch {
    FactoryLaunch {
        run: sample_run(),
        prompt: b"do the thing".to_vec(),
        harness_sha256: "f".repeat(64),
    }
}

pub(in crate::factory) fn sample_lease(run: &FactoryRun) -> Lease {
    Lease {
        provider_id: "codex".to_string(),
        id: "lease-1".to_string(),
        connection_id: run.connection.clone(),
        generation: 3,
        actor_id: run.actor,
        project_id: run.project.clone(),
        execution_id: run.id.clone(),
        kind: "factory".to_string(),
        role: run.role.clone(),
        deadline: run.deadline.clone(),
        ..Lease::default()
    }
}

pub(in crate::factory) fn sample_binding(run: &FactoryRun) -> Binding {
    Binding {
        scope: "factory-codex".to_string(),
        invocation_id: "09".repeat(16),
        kind: "factory".to_string(),
        id: run.id.clone(),
        project: container_id(),
        login: run.role.clone(),
        generation: 3,
        uid: 1001,
        gid: 1001,
        ..Binding::default()
    }
}

pub(in crate::factory) type TestFactory =
    Factory<std::rc::Rc<FakeExec>, std::rc::Rc<FakeTerminal>, std::rc::Rc<FakeBroker>>;

fn test_factory(
    dir: &std::path::Path,
    exec: std::rc::Rc<FakeExec>,
    term: std::rc::Rc<FakeTerminal>,
    broker: std::rc::Rc<FakeBroker>,
) -> TestFactory {
    Factory::open_factory(dir.to_str().unwrap(), exec, term, broker).unwrap()
}

pub(in crate::factory) fn receipt_bytes(
    dir: &std::path::Path,
    project: &str,
    run: &str,
) -> Vec<u8> {
    std::fs::read(dir.join(format!("{project}-{run}.json"))).unwrap()
}

pub(in crate::factory) fn fixed_run() -> FactoryRun {
    FactoryRun {
        deadline: "2030-06-07T08:09:10Z".to_string(),
        actor: 7,
        id: "a".repeat(32),
        project: format!("p{}", "b".repeat(24)),
        role: "soda-coder".to_string(),
        preparation: format!("f{}", "c".repeat(24)),
        harness: "codex".to_string(),
        harness_vers: "v1".to_string(),
        model: String::new(),
        assignment: "0".repeat(64),
        source_commit: "e".repeat(40),
        connection: "conn-1".to_string(),
    }
}

pub(in crate::factory) fn fixed_run_json() -> String {
    format!(
        "{{\"deadline\":\"2030-06-07T08:09:10Z\",\"actor\":7,\"id\":\"{}\",\"project\":\"p{}\",\"role\":\"soda-coder\",\"preparation\":\"f{}\",\"harness\":\"codex\",\"harness_version\":\"v1\",\"assignment\":\"{}\",\"source_commit\":\"{}\",\"connection\":\"conn-1\"}}",
        "a".repeat(32),
        "b".repeat(24),
        "c".repeat(24),
        "0".repeat(64),
        "e".repeat(40)
    )
}

pub(in crate::factory) fn fixed_binding() -> Binding {
    Binding {
        scope: "factory-codex".to_string(),
        invocation_id: "09".repeat(16),
        kind: "factory".to_string(),
        id: "a".repeat(32),
        project: "d".repeat(64),
        login: "soda-coder".to_string(),
        generation: 3,
        uid: 1001,
        gid: 1001,
        ..Binding::default()
    }
}

pub(in crate::factory) fn fixed_binding_json() -> String {
    format!(
        "{{\"uid\":1001,\"gid\":1001,\"scope\":\"factory-codex\",\"invocation_id\":\"{}\",\"kind\":\"factory\",\"id\":\"{}\",\"project\":\"{}\",\"login\":\"soda-coder\",\"generation\":3}}",
        "09".repeat(16),
        "a".repeat(32),
        "d".repeat(64)
    )
}

pub(in crate::factory) fn fixed_lease() -> Lease {
    Lease {
        provider_id: "codex".to_string(),
        id: "lease-1".to_string(),
        connection_id: "conn-1".to_string(),
        generation: 3,
        actor_id: 7,
        project_id: format!("p{}", "b".repeat(24)),
        execution_id: "a".repeat(32),
        kind: "factory".to_string(),
        role: "soda-coder".to_string(),
        deadline: "2030-06-07T08:09:10Z".to_string(),
        ..Lease::default()
    }
}

pub(in crate::factory) fn fixed_lease_json() -> String {
    format!(
        "{{\"provider_id\":\"codex\",\"id\":\"lease-1\",\"connection_id\":\"conn-1\",\"generation\":3,\"actor_id\":\"7\",\"project_id\":\"p{}\",\"execution_id\":\"{}\",\"kind\":\"factory\",\"role\":\"soda-coder\",\"deadline\":\"2030-06-07T08:09:10Z\"}}",
        "b".repeat(24),
        "a".repeat(32)
    )
}

pub(in crate::factory) fn dummy_factory(dir: &std::path::Path) -> TestFactory {
    use std::rc::Rc;
    test_factory(
        dir,
        Rc::new(FakeExec::new(Vec::new())),
        Rc::new(FakeTerminal::new()),
        Rc::new(FakeBroker::new()),
    )
}

pub(in crate::factory) fn wired_factory(
    tag: &str,
) -> (
    std::path::PathBuf,
    TestFactory,
    std::rc::Rc<FakeExec>,
    std::rc::Rc<FakeTerminal>,
    std::rc::Rc<FakeBroker>,
) {
    use std::rc::Rc;
    let dir = test_state_dir(tag);
    let exec = Rc::new(FakeExec::new(Vec::new()));
    let term = Rc::new(FakeTerminal::new());
    let broker = Rc::new(FakeBroker::new());
    let factory = test_factory(&dir, exec.clone(), term.clone(), broker.clone());
    (dir, factory, exec, term, broker)
}

pub(in crate::factory) fn drive_to_start(
    term: &std::rc::Rc<FakeTerminal>,
    broker: &std::rc::Rc<FakeBroker>,
    run: &FactoryRun,
) {
    broker.acquire.borrow_mut().push_back(Ok(sample_lease(run)));
    term.reserve.borrow_mut().push_back(Ok(sample_binding(run)));
    broker
        .register
        .borrow_mut()
        .push_back(Ok(b"{\"auth\":1}".to_vec()));
}

/// Hand-write a running receipt with lease and binding, as if a launch
/// were mid-flight.
pub(in crate::factory) fn write_running(dir: &std::path::Path, run: &FactoryRun, delivered: bool) {
    let receipt = FactoryReceipt {
        run: run.clone(),
        lease: Some(sample_lease(run)),
        binding: Some(sample_binding(run)),
        generation: 3,
        phase: "running".to_string(),
        started: true,
        delivered,
        ..FactoryReceipt::default()
    };
    receipt.validate().unwrap();
    std::fs::write(
        dir.join(format!("{}-{}.json", run.project, run.id)),
        receipt.encode().as_bytes(),
    )
    .unwrap();
}

pub(in crate::factory) fn wired_factory_exec(
    tag: &str,
    responses: Vec<Result<Vec<u8>, String>>,
) -> (
    std::path::PathBuf,
    TestFactory,
    std::rc::Rc<FakeExec>,
    std::rc::Rc<FakeTerminal>,
    std::rc::Rc<FakeBroker>,
) {
    use std::rc::Rc;
    let dir = test_state_dir(tag);
    let exec = Rc::new(FakeExec::new(responses));
    let term = Rc::new(FakeTerminal::new());
    let broker = Rc::new(FakeBroker::new());
    let factory = test_factory(&dir, exec.clone(), term.clone(), broker.clone());
    (dir, factory, exec, term, broker)
}

pub(in crate::factory) fn preparation_target(project: &str, cid: &str) -> Vec<u8> {
    format!(
        "{{\"id\":{cid:?},\"running\":true,\"project\":{project:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
    )
    .into_bytes()
}

pub(in crate::factory) fn sample_state(run: &FactoryRun) -> FactoryState {
    FactoryState {
        id: run.id.clone(),
        project: run.project.clone(),
        role: run.role.clone(),
        phase: "completed".to_string(),
        ..FactoryState::default()
    }
}

pub(in crate::factory) fn sample_preparation() -> Preparation {
    Preparation {
        id: format!("f{}", "c".repeat(24)),
        project: project_id(),
        role: "soda-coder".to_string(),
        revision: 1,
        requirements: preparation::RequirementAcceptance {
            id: format!("d{}", "c".repeat(24)),
            revision: 1,
            approver: 7,
            source_commit: "e".repeat(40),
            digest: "0".repeat(64),
        },
        approval: preparation::AdminApproval {
            id: format!("d{}", "c".repeat(24)),
            revision: 1,
            approver: 7,
            effects_digest: "0".repeat(64),
        },
        source_commit: "e".repeat(40),
        setup_digest: "0".repeat(64),
        tools: Vec::new(),
        credential: String::new(),
    }
}

pub(in crate::factory) fn sample_prepare_state(prep: &Preparation) -> PrepareState {
    PrepareState {
        id: prep.id.clone(),
        project: prep.project.clone(),
        role: prep.role.clone(),
        phase: "ready".to_string(),
        container: container_id(),
        source_commit: prep.source_commit.clone(),
        setup_digest: prep.setup_digest.clone(),
        ..PrepareState::default()
    }
}
