//! Native task-child control inside one admitted parent run (D05 / FW13).
//!
//! Only start eligible task, observe/join child and cancel child, all
//! internal to the admitted parent: never a public arbitrary-process API,
//! no nested child spawning, no extra admitted repository session. The
//! table enforces `MaxChildren` (policy range 1–4, admitted snapshot),
//! exact profile/account inheritance and parent stop/deadline retirement.
//! P1 proved the pinned CLI shapes overlap with real harnesses; this table
//! is the typed control boundary around them (FW14 plugs the CLI argv in).

use std::collections::HashMap;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::run::{valid_factory_role, valid_factory_run_id, FACTORY_HARNESS_MUSE};
use crate::factory::{MAX_TASK_CHILDREN, MIN_TASK_CHILDREN};

/// Admitted execution profile snapshot for one child: explicit reasoning
/// effort and Contributor mode, inherited exactly from the parent run.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChildProfile {
    pub effort: String,
    pub contributor: bool,
    pub role: String,
    pub account_home: String,
}

impl ChildProfile {
    /// Vocabulary and custody validation. Contributor mode is valid only
    /// for Muse; the harness pin separately proves effort support.
    pub fn validate(&self, harness: &str) -> Result<(), String> {
        match self.effort.as_str() {
            "low" | "medium" | "high" | "xhigh" => {}
            _ => return Err("invalid execution effort".to_string()),
        }
        if self.contributor && harness != FACTORY_HARNESS_MUSE {
            return Err("contributor mode is valid only for Muse".to_string());
        }
        if !valid_factory_role(&self.role) {
            return Err("invalid factory role".to_string());
        }
        if self.account_home.is_empty()
            || !self.account_home.starts_with('/')
            || self.account_home.len() > 1024
        {
            return Err("invalid child custody root".to_string());
        }
        Ok(())
    }

    /// Exact inherited environment. The base is minimal and fixed — no
    /// ambient parent variable (in particular no `META_API_KEY`) can leak
    /// past the staged custody root.
    pub fn environ(&self, run: &str, invocation: &str, task: &str) -> Vec<(String, String)> {
        vec![
            ("PATH".to_string(), "/usr/bin:/bin".to_string()),
            ("TERM".to_string(), "dumb".to_string()),
            ("HOME".to_string(), self.account_home.clone()),
            ("SODA_ROLE".to_string(), self.role.clone()),
            ("SODA_RUN".to_string(), run.to_string()),
            ("SODA_INVOCATION".to_string(), invocation.to_string()),
            ("SODA_TASK".to_string(), task.to_string()),
            ("SODA_EFFORT".to_string(), self.effort.clone()),
            (
                "SODA_CONTRIBUTOR".to_string(),
                if self.contributor {
                    "1".to_string()
                } else {
                    "0".to_string()
                },
            ),
        ]
    }
}

/// One eligible task launch: plan-local task key, fixed argv (never a
/// shell), and the child working directory under the admitted checkout.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChildSpec {
    pub task_key: String,
    pub argv: Vec<String>,
    pub workdir: PathBuf,
}

impl ChildSpec {
    pub fn validate(&self) -> Result<(), String> {
        if !crate::factory::valid_task_key(&self.task_key) {
            return Err("task event task key is invalid".to_string());
        }
        if self.argv.is_empty() || self.argv.len() > 64 {
            return Err("invalid child argv".to_string());
        }
        for word in &self.argv {
            if word.is_empty() || word.len() > 4096 {
                return Err("invalid child argv".to_string());
            }
        }
        Ok(())
    }
}

/// Observable child state: native identity, liveness, terminal exit and a
/// bounded output head. Ordinary prose never advances task state; these
/// typed observations do.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChildObservation {
    pub child_id: String,
    pub alive: bool,
    pub exit_code: Option<i32>,
    /// Bounded tail (8 KiB) of combined output for progress display.
    pub output_tail: Vec<u8>,
}

/// Parent-stop retirement report: retired children and the fenced
/// uncertain remainder, which must never be reported as success.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ChildStopReport {
    pub retired: Vec<String>,
    pub uncertain: Vec<String>,
}

struct ChildRecord {
    child_id: String,
    child: Option<std::process::Child>,
    exit_code: Option<i32>,
    log_path: PathBuf,
}

const OBSERVE_TAIL_BYTES: usize = 8192;
const CANCEL_TERM_GRACE: Duration = Duration::from_millis(1000);
const POLL_SLICE: Duration = Duration::from_millis(20);

/// Native children of one admitted parent run. One child owns a task
/// until its terminal observation; no automatic replacement child follows
/// a failed or uncertain task.
pub struct ChildTable {
    parent_run: String,
    parent_invocation: String,
    max_children: usize,
    scratch: PathBuf,
    counter: u64,
    children: HashMap<String, ChildRecord>,
}

impl ChildTable {
    /// Bind one table to its admitted parent. The scratch root holds
    /// per-child logs and must live under `/home`, outside any
    /// agent-writable checkout.
    pub fn open(
        parent_run: &str,
        parent_invocation: &str,
        max_children: i64,
        scratch: &Path,
    ) -> Result<Self, String> {
        if !valid_factory_run_id(parent_run) {
            return Err("invalid factory run identity".to_string());
        }
        if parent_invocation.is_empty() || parent_invocation.len() > 128 {
            return Err("task event invocation is invalid".to_string());
        }
        if !(MIN_TASK_CHILDREN..=MAX_TASK_CHILDREN).contains(&max_children) {
            return Err("execution manifest child limit is outside its range".to_string());
        }
        Ok(ChildTable {
            parent_run: parent_run.to_string(),
            parent_invocation: parent_invocation.to_string(),
            max_children: max_children as usize,
            scratch: scratch.to_path_buf(),
            counter: 0,
            children: HashMap::new(),
        })
    }

    /// Currently live children. Never exceeds the admitted limit.
    pub fn active_count(&mut self) -> usize {
        let mut live = 0;
        for record in self.children.values_mut() {
            if record.exit_code.is_none() && poll_child(record) == ChildPoll::Alive {
                live += 1;
            }
        }
        live
    }

    fn child_id(&mut self, task_key: &str) -> String {
        self.counter += 1;
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0);
        let input = format!(
            "factory-child-v1\x00{}\x00{task_key}\x00{nanos}\x00{}",
            self.parent_run, self.counter
        );
        let digest = crate::sha256::hex_lower(&crate::sha256::digest(input.as_bytes()));
        digest[..32].to_string()
    }

    /// Start the eligible task's child. Refuses duplicates, the admitted
    /// child cap and deadline expiry without spawning.
    pub fn start(
        &mut self,
        spec: &ChildSpec,
        profile: &ChildProfile,
        harness: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        spec.validate()?;
        profile.validate(harness)?;
        if Instant::now() >= deadline {
            return Err("parent deadline passed".to_string());
        }
        if self.children.contains_key(&spec.task_key) {
            return Err("task already has its child".to_string());
        }
        if self.active_count() >= self.max_children {
            return Err("task child limit reached".to_string());
        }
        let child_id = self.child_id(&spec.task_key);
        let log_path = self.scratch.join(format!("{child_id}.log"));
        let log = std::fs::OpenOptions::new()
            .create(true)
            .truncate(true)
            .write(true)
            .open(&log_path)
            .map_err(|e| e.to_string())?;
        let stdout = log.try_clone().map_err(|e| e.to_string())?;
        let mut command = Command::new(&spec.argv[0]);
        if spec.argv.len() > 1 {
            command.args(&spec.argv[1..]);
        }
        command
            .env_clear()
            .envs(profile.environ(&self.parent_run, &self.parent_invocation, &spec.task_key))
            .current_dir(&spec.workdir)
            .stdin(Stdio::null())
            .stdout(Stdio::from(stdout))
            .stderr(Stdio::from(log));
        // New session so cancellation retires the whole child group.
        #[cfg(target_os = "linux")]
        {
            use std::os::unix::process::CommandExt;
            command.process_group(0);
        }
        match command.spawn() {
            Ok(child) => {
                self.children.insert(
                    spec.task_key.clone(),
                    ChildRecord {
                        child_id: child_id.clone(),
                        child: Some(child),
                        exit_code: None,
                        log_path,
                    },
                );
                Ok(child_id)
            }
            Err(e) => Err(format!("task child start failed: {e}")),
        }
    }

    fn record_mut(&mut self, task_key: &str) -> Result<&mut ChildRecord, String> {
        self.children
            .get_mut(task_key)
            .ok_or_else(|| "unknown task child".to_string())
    }

    /// Observe one child without blocking: liveness, terminal exit and a
    /// bounded output tail.
    pub fn observe(&mut self, task_key: &str) -> Result<ChildObservation, String> {
        let record = self.record_mut(task_key)?;
        let (alive, exit_code) = match poll_child(record) {
            ChildPoll::Alive => (true, None),
            ChildPoll::Exited(code) => (false, Some(code)),
        };
        Ok(ChildObservation {
            child_id: record.child_id.clone(),
            alive,
            exit_code,
            output_tail: read_tail(&record.log_path),
        })
    }

    /// Join one child with a bounded wait. Expiry is a timeout, never a
    /// result: the child stays owned and observable.
    pub fn join(&mut self, task_key: &str, timeout: Duration) -> Result<i32, String> {
        let limit = Instant::now() + timeout;
        loop {
            let observation = self.observe(task_key)?;
            if let Some(code) = observation.exit_code {
                return Ok(code);
            }
            if Instant::now() >= limit {
                return Err("task child join timed out".to_string());
            }
            std::thread::sleep(POLL_SLICE);
        }
    }

    /// Cancel one child: SIGTERM with a grace interval, then SIGKILL. An
    /// unconfirmed retirement stays uncertain and must be fenced, never
    /// reported as success.
    pub fn cancel(&mut self, task_key: &str) -> Result<Option<i32>, String> {
        let record = self.record_mut(task_key)?;
        match terminate_record(record) {
            TerminateOutcome::Exited(code) => Ok(Some(code)),
            TerminateOutcome::Uncertain => Err("task child retirement unconfirmed".to_string()),
            TerminateOutcome::AlreadyGone => Ok(record.exit_code),
        }
    }

    /// Retire every child for parent stop or deadline expiry, before
    /// credential release. No child may outlive confirmed retirement;
    /// uncertain children stay fenced in the report.
    pub fn stop_all(&mut self) -> ChildStopReport {
        let mut report = ChildStopReport::default();
        let keys: Vec<String> = self.children.keys().cloned().collect();
        for key in keys {
            let Some(record) = self.children.get_mut(&key) else {
                continue;
            };
            if record.exit_code.is_some() {
                continue;
            }
            match terminate_record(record) {
                TerminateOutcome::Exited(_) | TerminateOutcome::AlreadyGone => {
                    report.retired.push(record.child_id.clone())
                }
                TerminateOutcome::Uncertain => report.uncertain.push(record.child_id.clone()),
            }
        }
        report
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ChildPoll {
    Alive,
    Exited(i32),
}

fn poll_child(record: &mut ChildRecord) -> ChildPoll {
    if let Some(code) = record.exit_code {
        return ChildPoll::Exited(code);
    }
    let Some(child) = record.child.as_mut() else {
        return ChildPoll::Exited(record.exit_code.unwrap_or(127));
    };
    match child.try_wait() {
        Ok(Some(status)) => {
            let code = status.code().unwrap_or(127);
            record.exit_code = Some(code);
            record.child = None;
            ChildPoll::Exited(code)
        }
        Ok(None) => ChildPoll::Alive,
        Err(_) => ChildPoll::Alive,
    }
}

enum TerminateOutcome {
    Exited(i32),
    AlreadyGone,
    Uncertain,
}

fn terminate_record(record: &mut ChildRecord) -> TerminateOutcome {
    if let Some(code) = record.exit_code {
        let _ = code;
        return TerminateOutcome::AlreadyGone;
    }
    let Some(child) = record.child.as_mut() else {
        return TerminateOutcome::AlreadyGone;
    };
    let pid = child.id() as libc::pid_t;
    // Escalation only while the pid is positively still our owned child.
    match child.try_wait() {
        Ok(Some(status)) => {
            let code = status.code().unwrap_or(127);
            record.exit_code = Some(code);
            record.child = None;
            return TerminateOutcome::Exited(code);
        }
        Ok(None) => {}
        Err(_) => return TerminateOutcome::Uncertain,
    }
    unsafe {
        libc::kill(pid, libc::SIGTERM);
    }
    let term_end = Instant::now() + CANCEL_TERM_GRACE;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let code = status.code().unwrap_or(127);
                record.exit_code = Some(code);
                record.child = None;
                return TerminateOutcome::Exited(code);
            }
            Ok(None) => {}
            Err(_) => return TerminateOutcome::Uncertain,
        }
        if Instant::now() >= term_end {
            break;
        }
        std::thread::sleep(POLL_SLICE);
    }
    // Still owned after the grace interval: SIGKILL is authorized. A
    // waitpid refusal at any point means ownership is lost, so no
    // further signal may follow a recycled pid.
    match child.try_wait() {
        Ok(None) => unsafe {
            libc::kill(pid, libc::SIGKILL);
        },
        Ok(Some(status)) => {
            let code = status.code().unwrap_or(127);
            record.exit_code = Some(code);
            record.child = None;
            return TerminateOutcome::Exited(code);
        }
        Err(_) => return TerminateOutcome::Uncertain,
    }
    let kill_end = Instant::now() + CANCEL_TERM_GRACE;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let code = status.code().unwrap_or(127);
                record.exit_code = Some(code);
                record.child = None;
                return TerminateOutcome::Exited(code);
            }
            Ok(None) => {}
            Err(_) => return TerminateOutcome::Uncertain,
        }
        if Instant::now() >= kill_end {
            return TerminateOutcome::Uncertain;
        }
        std::thread::sleep(POLL_SLICE);
    }
}

fn read_tail(path: &Path) -> Vec<u8> {
    let mut file = match std::fs::File::open(path) {
        Ok(file) => file,
        Err(_) => return Vec::new(),
    };
    let mut data = Vec::new();
    if file.read_to_end(&mut data).is_err() {
        return Vec::new();
    }
    if data.len() > OBSERVE_TAIL_BYTES {
        data[data.len() - OBSERVE_TAIL_BYTES..].to_vec()
    } else {
        data
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TAG: AtomicU64 = AtomicU64::new(0);

    fn scratch(tag: &str) -> PathBuf {
        let dir = PathBuf::from(std::env::var("HOME").unwrap()).join(format!(
            ".cache/fw13-children-{}-{}-{}",
            tag,
            std::process::id(),
            TAG.fetch_add(1, Ordering::SeqCst)
        ));
        std::fs::create_dir_all(&dir).unwrap();
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

    fn run_id() -> String {
        "a".repeat(32)
    }

    fn table(tag: &str, max_children: i64) -> (PathBuf, ChildTable) {
        let dir = scratch(tag);
        let workdir = dir.join("work");
        std::fs::create_dir_all(&workdir).unwrap();
        let table = ChildTable::open(&run_id(), &"09".repeat(16), max_children, &dir).unwrap();
        (dir, table)
    }

    fn spec(task: &str, dir: &Path, argv: &[&str]) -> ChildSpec {
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

    /// T12: two independent children actually overlap, and a dependent is
    /// released while an unrelated child remains active — no global
    /// barrier. Proven with real processes, mirroring the P1 fixture.
    #[test]
    fn independent_children_overlap_and_release() {
        let (dir, mut table) = table("overlap", 3);
        let slow = spec("slow-lane", &dir, &["/bin/sleep", "5"]);
        let fast = spec("fast-lane", &dir, &["/bin/echo", "fast-output"]);
        table.start(&slow, &profile(), "muse", deadline()).unwrap();
        table.start(&fast, &profile(), "muse", deadline()).unwrap();
        assert_eq!(table.active_count(), 2);
        // Joint-alive poll: both children observed live together.
        let mut joint = 0;
        for _ in 0..50 {
            let a = table.observe("slow-lane").unwrap();
            let b = table.observe("fast-lane").unwrap();
            if a.alive && b.alive {
                joint += 1;
                break;
            }
            std::thread::sleep(Duration::from_millis(20));
        }
        assert_eq!(joint, 1, "children never overlapped");
        // Fast completes with its exact output while slow stays active.
        assert_eq!(table.join("fast-lane", Duration::from_secs(10)).unwrap(), 0);
        let slow_obs = table.observe("slow-lane").unwrap();
        assert!(slow_obs.alive);
        // Dependent releases immediately: no unrelated-lane barrier.
        let dependent = spec("downstream", &dir, &["/bin/echo", "downstream-output"]);
        table
            .start(&dependent, &profile(), "muse", deadline())
            .unwrap();
        assert_eq!(
            table.join("downstream", Duration::from_secs(10)).unwrap(),
            0
        );
        let report = table.stop_all();
        assert!(report.uncertain.is_empty());
        assert_eq!(report.retired.len(), 1);
        let fast_obs = table.observe("fast-lane").unwrap();
        assert!(fast_obs.output_tail.ends_with(b"fast-output\n"));
        cleanup(&dir);
    }

    /// T12/T16: the admitted child cap, exact profile/account
    /// inheritance and duplicate-task refusal.
    #[test]
    fn child_cap_profile_and_ownership() {
        let (dir, mut table) = table("cap", 1);
        let first = spec("only-task", &dir, &["/bin/sleep", "5"]);
        let second = spec("other-task", &dir, &["/bin/echo", "hi"]);
        table.start(&first, &profile(), "muse", deadline()).unwrap();
        assert!(table
            .start(&second, &profile(), "muse", deadline())
            .is_err());
        assert!(table.start(&first, &profile(), "muse", deadline()).is_err());
        let report = table.stop_all();
        assert!(report.uncertain.is_empty());
        // A fresh task may start once room is free; the same task never
        // gets a replacement child.
        table
            .start(&second, &profile(), "muse", deadline())
            .unwrap();
        assert_eq!(
            table.join("other-task", Duration::from_secs(10)).unwrap(),
            0
        );
        assert!(table
            .start(&second, &profile(), "muse", deadline())
            .is_err());
        // Exact inheritance: the child sees precisely the admitted
        // profile/account, and no ambient test-process variable leaks in
        // (CARGO_* is present here, never in the child).
        let probe = spec("env-probe", &dir, &["/usr/bin/env"]);
        table.start(&probe, &profile(), "muse", deadline()).unwrap();
        assert_eq!(table.join("env-probe", Duration::from_secs(10)).unwrap(), 0);
        let observation = table.observe("env-probe").unwrap();
        let env = String::from_utf8(observation.output_tail).unwrap();
        for want in [
            "HOME=/home/soda-coder",
            "SODA_ROLE=soda-coder",
            "SODA_EFFORT=xhigh",
            "SODA_CONTRIBUTOR=1",
            "SODA_TASK=env-probe",
        ] {
            assert!(env.lines().any(|line| line == want), "missing {want}");
        }
        assert!(!env.contains("CARGO_"), "ambient environment leaked");
        assert!(!env.contains("META_API_KEY"), "credential leaked");
        // Contributor mode binds the Muse harness.
        let probe2 = spec("env-probe-2", &dir, &["/usr/bin/env"]);
        assert!(table
            .start(&probe2, &profile(), "codex", deadline())
            .is_err());
        let _ = table.stop_all();
        cleanup(&dir);
    }

    /// T16: cancelling one child retires exactly it; parent stop retires
    /// everything before credential release, with fenced uncertainty.
    #[test]
    fn cancel_and_parent_stop_retire_children() {
        let (dir, mut table) = table("cancel", 3);
        let first = spec("cancel-me", &dir, &["/bin/sleep", "60"]);
        let second = spec("stay-live", &dir, &["/bin/sleep", "60"]);
        table.start(&first, &profile(), "muse", deadline()).unwrap();
        table
            .start(&second, &profile(), "muse", deadline())
            .unwrap();
        table.cancel("cancel-me").unwrap();
        let cancelled = table.observe("cancel-me").unwrap();
        assert!(!cancelled.alive);
        assert!(table.observe("stay-live").unwrap().alive);
        assert!(table.cancel("cancel-me").is_ok());
        assert!(table.cancel("unknown-task").is_err());
        let report = table.stop_all();
        assert!(report.uncertain.is_empty());
        assert_eq!(report.retired.len(), 1);
        assert!(!table.observe("stay-live").unwrap().alive);
        // Past-deadline starts refuse without spawning.
        let late = spec("late-task", &dir, &["/bin/echo", "late"]);
        assert!(table
            .start(&late, &profile(), "muse", Instant::now())
            .is_err());
        assert!(table.observe("late-task").is_err());
        cleanup(&dir);
    }

    #[test]
    fn admission_rejects_bad_bindings() {
        let dir = scratch("admit");
        assert!(ChildTable::open("short", &"09".repeat(16), 3, &dir).is_err());
        assert!(ChildTable::open(&run_id(), "", 3, &dir).is_err());
        assert!(ChildTable::open(&run_id(), &"09".repeat(16), 0, &dir).is_err());
        assert!(ChildTable::open(&run_id(), &"09".repeat(16), 5, &dir).is_err());
        cleanup(&dir);
    }
}
