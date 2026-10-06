use crate::project::Executor;
use crate::terminal::factory::tcodex::{self, FactoryRun};
use crate::terminal::factory::tmuse::*;
use crate::terminal::{self, Binding, Lease, Service, KIND_FACTORY};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

pub(super) const PID: &str = "p0123456789abcdef01234567";
pub(super) const RID: &str = "0123456789abcdef0123456789abcdef";
pub(super) const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
pub(super) const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
pub(super) const PREP: &str = "f0123456789abcdef01234567";
pub(super) const ROLE: &str = "soda-coder";
pub(super) const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
pub(super) const PIN: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);

pub(super) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

pub(super) fn test_tmp(slug: &str) -> std::path::PathBuf {
    let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("t26m-{}-{n}-{slug}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

pub(super) type RecordedCall = (Vec<u8>, String, Vec<String>);

pub(super) struct FakeExec {
    calls: Mutex<Vec<RecordedCall>>,
    script: Mutex<Vec<Result<Vec<u8>, String>>>,
}

impl FakeExec {
    pub(super) fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
        FakeExec {
            calls: Mutex::new(Vec::new()),
            script: Mutex::new(script),
        }
    }

    pub(super) fn calls(&self) -> Vec<RecordedCall> {
        self.calls.lock().unwrap().clone()
    }
}

impl Executor for FakeExec {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.lock().unwrap().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        let mut script = self.script.lock().unwrap();
        if script.is_empty() {
            return Err("unexpected call".to_string());
        }
        script.remove(0)
    }
}

pub(super) fn ok(body: &str) -> Result<Vec<u8>, String> {
    Ok(body.as_bytes().to_vec())
}

pub(super) fn err(body: &str) -> Result<Vec<u8>, String> {
    Err(body.to_string())
}

pub(super) fn make_service(exec: FakeExec) -> Service<FakeExec> {
    Service {
        exec,
        codex_harness: String::new(),
        codex_harness_sha256: String::new(),
        codex_harness_version: String::new(),
        muse_harness: "/opt/muse".to_string(),
        muse_harness_sha256: PIN.to_string(),
        muse_harness_version: "1.4.2".to_string(),
    }
}

fn write_harness(dir: &std::path::Path) {
    let bin = dir.join("bin");
    std::fs::create_dir_all(&bin).unwrap();
    std::fs::write(bin.join("muse"), b"").unwrap(); // sha256("") == PIN
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(bin.join("muse"), std::fs::Permissions::from_mode(0o755)).unwrap();
}

pub(super) fn reserve_harness() -> (std::path::PathBuf, String) {
    let dir = test_tmp("reserve");
    write_harness(&dir);
    let path = dir.to_str().unwrap().to_string();
    (dir, path)
}

pub(super) fn inspect_json() -> String {
    format!(
        "{{\"id\":{CID:?},\"running\":true,\"project\":{PID:?},\"owner\":\"7\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
    )
}

pub(super) fn muse_run() -> FactoryRun {
    FactoryRun {
        deadline_raw: "2030-01-01T00:00:00Z".to_string(),
        actor: 7,
        id: RID.to_string(),
        project: PID.to_string(),
        role: ROLE.to_string(),
        preparation: PREP.to_string(),
        harness: tcodex::FACTORY_HARNESS_MUSE.to_string(),
        harness_vers: "1.4.2".to_string(),
        model: "muse-spark-1.3".to_string(),
        assignment: PIN.to_string(),
        source_commit: COMMIT.to_string(),
        connection: "conn".to_string(),
    }
}

pub(super) fn muse_run_dir() -> String {
    factory_muse_run_paths(ROLE, PREP, RID).unwrap().1
}

pub(super) fn muse_lease() -> Lease {
    Lease {
        provider_id: terminal::PROVIDER_MUSE.to_string(),
        id: "lease-f".to_string(),
        connection_id: "conn".to_string(),
        generation: 5,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: RID.to_string(),
        kind: KIND_FACTORY.to_string(),
        binding: Some(Binding {
            kind: KIND_FACTORY.to_string(),
            id: RID.to_string(),
            project: CID.to_string(),
            login: ROLE.to_string(),
            uid: 1001,
            gid: 1001,
            scope: tcodex::FACTORY_SCOPE_MUSE.to_string(),
            invocation_id: IID.to_string(),
            credential_root: muse_run_dir(),
            generation: 5,
            child_id: PREP.to_string(),
        }),
        ..Default::default()
    }
}

pub(super) fn euid() -> u32 {
    unsafe { libc::geteuid() }
}
