use super::*;
use crate::project::Executor;
use std::cell::RefCell;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

pub(crate) fn temp_dir(name: &str) -> std::path::PathBuf {
    let id = TEMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("soda-laneD-{name}-{}-{id}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

pub(crate) fn project_id() -> String {
    format!("p{}", "a".repeat(24))
}

/// Mirrors Go's `fileRun` fixture.
pub(crate) fn file_run() -> ProjectRun {
    ProjectRun {
        target: RunTarget {
            project: project_id(),
            container: "b".repeat(64),
            run: "c".repeat(64),
        },
        pid: 77,
        started: String::new(),
        userns: "user:[4026531837]".to_string(),
        netns: "net:[4026531840]".to_string(),
        uid: 300000,
        gid: 300000,
        resolver: String::new(),
    }
}

pub(crate) fn test_view() -> ProjectView {
    ProjectView {
        available_binding: String::new(),
        available_network: String::new(),
        addresses: Vec::new(),
        dns_name: String::new(),
        saved: false,
        project: project_id(),
        revision: String::new(),
        binding: String::new(),
        enabled: false,
        state: String::new(),
        outcome: String::new(),
    }
}

pub(crate) fn test_binding() -> RunBinding {
    RunBinding {
        enabled: true,
        admission: false,
        tailnet: "tail-abc".to_string(),
        tags: vec!["tag:soda".to_string()],
    }
}

pub(crate) fn test_request(action: &str) -> ProjectRequest {
    ProjectRequest {
        project: project_id(),
        action: action.to_string(),
        revision: String::new(),
        binding: String::new(),
        confirm_id: String::new(),
    }
}

pub(crate) fn project_inspect_json(id: &str, cid: &str, running: bool) -> Vec<u8> {
    format!(
        "{{\"id\":\"{cid}\",\"running\":{running},\"project\":\"{id}\",\"owner\":\"1000\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:100000:262144\"],\"GidMap\":[\"0:100000:262144\"]}}}}"
    )
    .into_bytes()
}

#[allow(clippy::type_complexity)]
pub(crate) struct MockExec {
    pub(crate) calls: RefCell<Vec<(String, Vec<String>)>>,
    pub(crate) handler: Box<dyn Fn(&str, &[String]) -> Result<Vec<u8>, String>>,
    pub(crate) native: bool,
}

impl MockExec {
    pub(crate) fn with_handler(
        handler: impl Fn(&str, &[String]) -> Result<Vec<u8>, String> + 'static,
    ) -> Self {
        MockExec {
            calls: RefCell::new(Vec::new()),
            handler: Box::new(handler),
            native: false,
        }
    }

    pub(crate) fn calls(&self) -> Vec<(String, Vec<String>)> {
        self.calls.borrow().clone()
    }
}

impl Executor for MockExec {
    fn run(
        &self,
        _stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let owned: Vec<String> = args.iter().map(|s| s.to_string()).collect();
        self.calls
            .borrow_mut()
            .push((cmd.to_string(), owned.clone()));
        (self.handler)(cmd, &owned)
    }

    fn is_host_native(&self) -> bool {
        self.native
    }
}

#[allow(clippy::type_complexity)]
pub(crate) struct MockTailnet {
    pub(crate) project_fn: Box<dyn Fn(&ProjectRequest, &str) -> Result<ProjectView, String>>,
    pub(crate) binding_fn: Box<dyn Fn(&RunTarget) -> Result<RunBinding, String>>,
    pub(crate) enroll_targets: RefCell<Vec<String>>,
    pub(crate) enroll_err: Option<String>,
}

impl MockTailnet {
    pub(crate) fn inert() -> Self {
        MockTailnet {
            project_fn: Box::new(|_, _| panic!("unexpected tailnet.project call")),
            binding_fn: Box::new(|_| panic!("unexpected tailnet.run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        }
    }
}

impl TailnetControl for MockTailnet {
    fn project(
        &self,
        req: &ProjectRequest,
        cid: &str,
        _deadline: Instant,
    ) -> Result<ProjectView, String> {
        (self.project_fn)(req, cid)
    }

    fn run_binding(&self, target: &RunTarget, _deadline: Instant) -> Result<RunBinding, String> {
        (self.binding_fn)(target)
    }

    fn enroll_run(
        &self,
        target: &RunTarget,
        _recheck: &dyn Fn(Instant) -> Result<(), String>,
        _consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        _deadline: Instant,
    ) -> Result<(), String> {
        self.enroll_targets.borrow_mut().push(target.run.clone());
        match &self.enroll_err {
            Some(e) => Err(e.clone()),
            None => Ok(()),
        }
    }
}

pub(crate) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

pub(crate) fn companion_with(
    exec: MockExec,
    tailnet: MockTailnet,
    image: &str,
) -> Companion<MockExec, MockTailnet> {
    Companion {
        exec,
        tailnet,
        image: image.to_string(),
        enabled_check: None,
    }
}

pub(crate) fn image_id() -> String {
    format!("sha256:{}", "d".repeat(64))
}
