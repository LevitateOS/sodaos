//! Bounded requests into the one live native parent task owner.
//! The queue transports requests only; NativeTasks remains the sole ledger and
//! admission owner. A stopped controller or expired caller cannot grant work.

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

use super::native_tasks::NativeTasks;
use super::receipt::FactoryReceipt;
use super::{Factory, FactoryBroker, FactoryError, FactoryTerminal, PrivateState};
use crate::project::Executor;

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct NativeTaskAction {
    pub project: String,
    pub run: String,
    pub invocation_id: String,
    pub manifest_digest: String,
    pub action: String,
    pub arguments: TaskArguments,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct TaskArguments {
    pub task_id: String,
}

impl NativeTaskAction {
    pub fn validate(&self) -> Result<(), String> {
        if !crate::domain::valid_id(&self.project)
            || !super::valid_factory_run_id(&self.run)
            || self.invocation_id.is_empty()
            || self.invocation_id.len() > 128
            || self.invocation_id.contains('\0')
            || !crate::preparation::valid_digest(&self.manifest_digest)
            || !matches!(self.action.as_str(), "start_task" | "release_output")
            || !super::valid_task_key(&self.arguments.task_id)
        {
            return Err("invalid native task action".into());
        }
        Ok(())
    }
}

struct Pending {
    request: NativeTaskAction,
    deadline: Instant,
    alive: Arc<AtomicBool>,
    reply: mpsc::SyncSender<Result<Value, FactoryError>>,
}

#[derive(Clone)]
struct Endpoint {
    invocation: String,
    manifest: String,
    sender: mpsc::SyncSender<Pending>,
}

type Key = (String, String, String);
fn endpoints() -> &'static Mutex<HashMap<Key, Endpoint>> {
    static ENDPOINTS: OnceLock<Mutex<HashMap<Key, Endpoint>>> = OnceLock::new();
    ENDPOINTS.get_or_init(|| Mutex::new(HashMap::new()))
}

pub(super) struct NativeTaskOwner {
    key: Key,
    invocation: String,
    receiver: mpsc::Receiver<Pending>,
}

impl Drop for NativeTaskOwner {
    fn drop(&mut self) {
        if let Ok(mut endpoints) = endpoints().lock() {
            if endpoints
                .get(&self.key)
                .is_some_and(|entry| entry.invocation == self.invocation)
            {
                endpoints.remove(&self.key);
            }
        }
    }
}

impl NativeTaskOwner {
    pub(super) fn register(state: &str, receipt: &FactoryReceipt) -> Result<Self, FactoryError> {
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        let key = (
            state.into(),
            receipt.run.project.clone(),
            receipt.run.id.clone(),
        );
        let (sender, receiver) = mpsc::sync_channel(4);
        let mut endpoints = endpoints().lock().map_err(|_| FactoryError::Stale)?;
        if endpoints.contains_key(&key) {
            return Err(FactoryError::Stale);
        }
        endpoints.insert(
            key.clone(),
            Endpoint {
                invocation: binding.invocation_id.clone(),
                manifest: receipt.run.manifest_digest.clone(),
                sender,
            },
        );
        Ok(Self {
            key,
            invocation: binding.invocation_id.clone(),
            receiver,
        })
    }

    pub(super) fn poll<E: Executor, T: FactoryTerminal, B: FactoryBroker>(
        &self,
        factory: &Factory<E, T, B>,
        receipt: &FactoryReceipt,
        private: &PrivateState,
        tasks: &mut NativeTasks,
        session: &str,
        turn: &str,
    ) {
        for _ in 0..4 {
            let Ok(pending) = self.receiver.try_recv() else {
                break;
            };
            if !pending.alive.load(Ordering::Acquire) || Instant::now() >= pending.deadline {
                let _ = pending.reply.try_send(Err(FactoryError::DeadlineExceeded));
                continue;
            }
            let request = &pending.request;
            // Stable identity belongs to this exact run/action/task tuple.
            // An HTTP retry replays the original selection, never a new nonce.
            let bytes = serde_json::to_vec(request).expect("fixed native action serializes");
            let call = crate::sha256::hex_lower(&crate::sha256::digest(&bytes));
            let params = json!({
                "threadId":session, "turnId":turn, "callId":call,
                "tool":if request.action == "start_task" { super::native_tasks::START_TOOL } else { super::native_tasks::RELEASE_TOOL },
                "arguments":{"task_id":request.arguments.task_id},
            });
            let result = tasks
                .tool(
                    factory,
                    receipt,
                    private,
                    &json!(call),
                    &params,
                    turn,
                    pending.deadline,
                )
                .and_then(|text| serde_json::from_str(&text).map_err(|_| FactoryError::Stale));
            let _ = pending.reply.try_send(result);
        }
    }
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    pub fn native_task_action(
        &self,
        request: &NativeTaskAction,
        deadline: Instant,
    ) -> Result<Value, FactoryError> {
        request.validate().map_err(FactoryError::msg)?;
        {
            let _lock = self.lock_run(&request.project, &request.run, deadline)?;
            let (receipt, exists) = self.load_receipt(&request.project, &request.run)?;
            let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
            let phase = if request.action == "start_task" {
                super::EXECUTION_IMPLEMENT
            } else {
                super::EXECUTION_CORRECT
            };
            if !exists
                || receipt.phase != super::FACTORY_RUNNING
                || receipt.run.harness != super::FACTORY_HARNESS_MUSE
                || receipt.run.execution_phase != phase
                || receipt.run.manifest_digest != request.manifest_digest
                || binding.invocation_id != request.invocation_id
                || !self.terminal.live(binding, deadline)
            {
                return Err(FactoryError::Stale);
            }
        }
        let endpoint = endpoints()
            .lock()
            .map_err(|_| FactoryError::Stale)?
            .get(&(
                self.state_dir.clone(),
                request.project.clone(),
                request.run.clone(),
            ))
            .cloned()
            .ok_or(FactoryError::Stale)?;
        if endpoint.invocation != request.invocation_id
            || endpoint.manifest != request.manifest_digest
        {
            return Err(FactoryError::Stale);
        }
        let deadline = deadline.min(Instant::now() + Duration::from_secs(30));
        let alive = Arc::new(AtomicBool::new(true));
        let (reply, receiver) = mpsc::sync_channel(1);
        endpoint
            .sender
            .try_send(Pending {
                request: request.clone(),
                deadline,
                alive: alive.clone(),
                reply,
            })
            .map_err(|_| FactoryError::Stale)?;
        let result = receiver
            .recv_timeout(deadline.saturating_duration_since(Instant::now()))
            .map_err(|_| FactoryError::DeadlineExceeded);
        alive.store(false, Ordering::Release);
        result?
    }
}
