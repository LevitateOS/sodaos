use std::cell::RefCell;
use std::collections::VecDeque;
use std::time::Instant;

use super::common::{sample_binding, sample_lease};
use crate::factory::*;
use crate::project::Executor;
use crate::terminal::Binding;

pub(in crate::factory) type ExecCall = (Vec<u8>, String, Vec<String>);

pub(in crate::factory) struct FakeExec {
    pub(in crate::factory) calls: RefCell<Vec<ExecCall>>,
    pub(in crate::factory) script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
}

impl FakeExec {
    pub(in crate::factory) fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
        FakeExec {
            calls: RefCell::new(Vec::new()),
            script: RefCell::new(responses.into()),
        }
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
        self.calls.borrow_mut().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        self.script
            .borrow_mut()
            .pop_front()
            .unwrap_or(Err("no scripted exec response".to_string()))
    }
}

pub(in crate::factory) type TakeoverCall = (String, String, String, String, String, String);
pub(in crate::factory) type ExportCall = (String, String, String, String, String);

pub(in crate::factory) struct FakeTerminal {
    pub(in crate::factory) version: String,
    pub(in crate::factory) sha256: String,
    pub(in crate::factory) muse_version: String,
    pub(in crate::factory) muse_sha256: String,
    pub(in crate::factory) reserve: RefCell<VecDeque<Result<Binding, FactoryError>>>,
    pub(in crate::factory) start: RefCell<VecDeque<Result<(), FactoryError>>>,
    pub(in crate::factory) wait: RefCell<VecDeque<Result<(i64, String), FactoryError>>>,
    pub(in crate::factory) stop: RefCell<VecDeque<Result<(), FactoryError>>>,
    pub(in crate::factory) stop_unbound: RefCell<VecDeque<Result<(), FactoryError>>>,
    pub(in crate::factory) capture: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
    pub(in crate::factory) live: RefCell<VecDeque<bool>>,
    pub(in crate::factory) output: RefCell<VecDeque<Result<OutputSlice, FactoryError>>>,
    pub(in crate::factory) takeover: RefCell<VecDeque<Result<(String, bool), FactoryError>>>,
    pub(in crate::factory) export: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
    pub(in crate::factory) reserve_calls: RefCell<Vec<(String, i64)>>,
    pub(in crate::factory) takeover_calls: RefCell<Vec<TakeoverCall>>,
    pub(in crate::factory) export_calls: RefCell<Vec<ExportCall>>,
    pub(in crate::factory) output_calls: RefCell<Vec<(String, i64, i64)>>,
}

impl FakeTerminal {
    pub(in crate::factory) fn new() -> Self {
        FakeTerminal {
            version: "v1".to_string(),
            sha256: "f".repeat(64),
            muse_version: "m2".to_string(),
            muse_sha256: "e".repeat(64),
            reserve: RefCell::new(VecDeque::new()),
            start: RefCell::new(VecDeque::new()),
            wait: RefCell::new(VecDeque::new()),
            stop: RefCell::new(VecDeque::new()),
            stop_unbound: RefCell::new(VecDeque::new()),
            capture: RefCell::new(VecDeque::new()),
            live: RefCell::new(VecDeque::new()),
            output: RefCell::new(VecDeque::new()),
            takeover: RefCell::new(VecDeque::new()),
            export: RefCell::new(VecDeque::new()),
            reserve_calls: RefCell::new(Vec::new()),
            takeover_calls: RefCell::new(Vec::new()),
            export_calls: RefCell::new(Vec::new()),
            output_calls: RefCell::new(Vec::new()),
        }
    }

    fn pop<T>(queue: &RefCell<VecDeque<T>>, what: &str) -> T {
        queue
            .borrow_mut()
            .pop_front()
            .unwrap_or_else(|| panic!("no scripted {what}"))
    }
}

impl FactoryTerminal for FakeTerminal {
    fn harness_pin(&self, family: &str) -> Result<FactoryHarnessPin, String> {
        let (version, sha256) = match family {
            FACTORY_HARNESS_CODEX => (&self.version, &self.sha256),
            FACTORY_HARNESS_MUSE => (&self.muse_version, &self.muse_sha256),
            _ => return Err("unsupported factory harness".to_string()),
        };
        Ok(FactoryHarnessPin {
            harness: family.to_string(),
            version: version.clone(),
            sha256: sha256.clone(),
            image: String::new(),
        })
    }
    fn reserve(
        &self,
        _run: &FactoryRun,
        _lease: &Lease,
        pin: &str,
        max_secs: i64,
        _deadline: Instant,
    ) -> Result<Binding, FactoryError> {
        self.reserve_calls
            .borrow_mut()
            .push((pin.to_string(), max_secs));
        Self::pop(&self.reserve, "reserve")
    }
    fn start(
        &self,
        _lease: &Lease,
        _credential: &[u8],
        _prompt: &[u8],
        _deadline: Instant,
    ) -> Result<(), FactoryError> {
        Self::pop(&self.start, "start")
    }
    fn wait(&self, _lease: &Lease, _deadline: Instant) -> Result<(i64, String), FactoryError> {
        Self::pop(&self.wait, "wait")
    }
    fn stop(&self, _lease: &Lease, _deadline: Instant) -> Result<(), FactoryError> {
        Self::pop(&self.stop, "stop")
    }
    fn stop_unbound(&self, _run: &FactoryRun, _deadline: Instant) -> Result<(), FactoryError> {
        Self::pop(&self.stop_unbound, "stop_unbound")
    }
    fn capture(&self, _lease: &Lease, _deadline: Instant) -> Result<Vec<u8>, FactoryError> {
        Self::pop(&self.capture, "capture")
    }
    fn live(&self, _binding: &Binding, _deadline: Instant) -> bool {
        Self::pop(&self.live, "live")
    }
    fn output(
        &self,
        project: &str,
        _binding: &Binding,
        offset: i64,
        limit: i64,
        _deadline: Instant,
    ) -> Result<OutputSlice, FactoryError> {
        self.output_calls
            .borrow_mut()
            .push((project.to_string(), offset, limit));
        Self::pop(&self.output, "output")
    }
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        _deadline: Instant,
    ) -> Result<(String, bool), FactoryError> {
        self.takeover_calls.borrow_mut().push((
            project.to_string(),
            recorded.to_string(),
            role.to_string(),
            preparation.to_string(),
            member.to_string(),
            run.to_string(),
        ));
        Self::pop(&self.takeover, "takeover")
    }
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        _deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError> {
        self.export_calls.borrow_mut().push((
            project.to_string(),
            recorded.to_string(),
            role.to_string(),
            preparation.to_string(),
            candidate.to_string(),
        ));
        Self::pop(&self.export, "export")
    }
}

pub(in crate::factory) struct FakeBroker {
    pub(in crate::factory) acquire: RefCell<VecDeque<Result<Lease, FactoryError>>>,
    pub(in crate::factory) register: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
    pub(in crate::factory) returns: RefCell<VecDeque<Result<(), FactoryError>>>,
    pub(in crate::factory) terminal: RefCell<VecDeque<Result<bool, FactoryError>>>,
    pub(in crate::factory) close: RefCell<VecDeque<Result<(), FactoryError>>>,
    pub(in crate::factory) acquire_calls: RefCell<Vec<AcquireRequest>>,
    pub(in crate::factory) reconcile_calls: RefCell<Vec<String>>,
    pub(in crate::factory) close_calls: RefCell<Vec<(String, String)>>,
}

impl FakeBroker {
    pub(in crate::factory) fn new() -> Self {
        FakeBroker {
            acquire: RefCell::new(VecDeque::new()),
            register: RefCell::new(VecDeque::new()),
            returns: RefCell::new(VecDeque::new()),
            terminal: RefCell::new(VecDeque::new()),
            close: RefCell::new(VecDeque::new()),
            acquire_calls: RefCell::new(Vec::new()),
            reconcile_calls: RefCell::new(Vec::new()),
            close_calls: RefCell::new(Vec::new()),
        }
    }

    fn pop<T>(queue: &RefCell<VecDeque<T>>, what: &str) -> T {
        queue
            .borrow_mut()
            .pop_front()
            .unwrap_or_else(|| panic!("no scripted {what}"))
    }
}

impl FactoryBroker for FakeBroker {
    fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, FactoryError> {
        self.acquire_calls.borrow_mut().push(req.clone());
        Self::pop(&self.acquire, "acquire")
    }
    fn register(
        &self,
        _lease_id: &str,
        _binding: &Binding,
        _deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError> {
        Self::pop(&self.register, "register")
    }
    fn return_lease(
        &self,
        _lease_id: &str,
        _binding: &Binding,
        _credential: &[u8],
        _deadline: Instant,
    ) -> Result<(), FactoryError> {
        Self::pop(&self.returns, "return")
    }
    fn reconcile_lease(&self, lease_id: &str, _deadline: Instant) {
        self.reconcile_calls.borrow_mut().push(lease_id.to_string());
    }
    fn execution_is_terminal(
        &self,
        _kind: &str,
        _execution_id: &str,
        _deadline: Instant,
    ) -> Result<bool, FactoryError> {
        Self::pop(&self.terminal, "get-execution")
    }
    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        _deadline: Instant,
    ) -> Result<(), FactoryError> {
        self.close_calls
            .borrow_mut()
            .push((kind.to_string(), execution_id.to_string()));
        Self::pop(&self.close, "close")
    }
}

/// Script a full successful launch: acquire, reserve, register, start,
/// wait, stop (retire), capture, return, close.
pub(in crate::factory) fn script_success(
    term: &FakeTerminal,
    broker: &FakeBroker,
    run: &FactoryRun,
    exit: i64,
    output: &str,
) {
    broker.acquire.borrow_mut().push_back(Ok(sample_lease(run)));
    term.reserve.borrow_mut().push_back(Ok(sample_binding(run)));
    broker
        .register
        .borrow_mut()
        .push_back(Ok(b"{\"auth\":1}".to_vec()));
    term.start.borrow_mut().push_back(Ok(()));
    term.wait
        .borrow_mut()
        .push_back(Ok((exit, output.to_string())));
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture
        .borrow_mut()
        .push_back(Ok(b"{\"auth\":2}".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
}

// Shared handles: the factory owns its seams, tests keep a clone.
impl Executor for std::rc::Rc<FakeExec> {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        (**self).run(stdin, cmd, args, deadline)
    }
}

impl FactoryTerminal for std::rc::Rc<FakeTerminal> {
    fn harness_pin(&self, family: &str) -> Result<FactoryHarnessPin, String> {
        (**self).harness_pin(family)
    }
    fn reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<Binding, FactoryError> {
        (**self).reserve(run, lease, pin, max_secs, deadline)
    }
    fn start(
        &self,
        lease: &Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        (**self).start(lease, credential, prompt, deadline)
    }
    fn wait(&self, lease: &Lease, deadline: Instant) -> Result<(i64, String), FactoryError> {
        (**self).wait(lease, deadline)
    }
    fn stop(&self, lease: &Lease, deadline: Instant) -> Result<(), FactoryError> {
        (**self).stop(lease, deadline)
    }
    fn stop_unbound(&self, run: &FactoryRun, deadline: Instant) -> Result<(), FactoryError> {
        (**self).stop_unbound(run, deadline)
    }
    fn capture(&self, lease: &Lease, deadline: Instant) -> Result<Vec<u8>, FactoryError> {
        (**self).capture(lease, deadline)
    }
    fn live(&self, binding: &Binding, deadline: Instant) -> bool {
        (**self).live(binding, deadline)
    }
    fn output(
        &self,
        project: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<OutputSlice, FactoryError> {
        (**self).output(project, binding, offset, limit, deadline)
    }
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), FactoryError> {
        (**self).takeover_copy(project, recorded, role, preparation, member, run, deadline)
    }
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError> {
        (**self).export_bundle(project, recorded, role, preparation, candidate, deadline)
    }
}

impl FactoryBroker for std::rc::Rc<FakeBroker> {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, FactoryError> {
        (**self).acquire(req, deadline)
    }
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError> {
        (**self).register(lease_id, binding, deadline)
    }
    fn return_lease(
        &self,
        lease_id: &str,
        binding: &Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        (**self).return_lease(lease_id, binding, credential, deadline)
    }
    fn reconcile_lease(&self, lease_id: &str, deadline: Instant) {
        (**self).reconcile_lease(lease_id, deadline);
    }
    fn execution_is_terminal(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<bool, FactoryError> {
        (**self).execution_is_terminal(kind, execution_id, deadline)
    }
    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        (**self).close_execution(kind, execution_id, deadline)
    }
}
