use std::collections::HashMap;
use std::os::unix::io::RawFd;
use std::sync::Mutex;
use std::time::Instant;

use super::LaunchRequest;
use crate::terminal::{AcquireRequest, Binding, Delivery, Lease};

// ---------- runtime types ----------

/// Kernel-attested socket identity. PIDFD pins the original caller.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MusePeer {
    pub pid: i32,
    pub uid: u32,
    pub gid: u32,
    pub pidfd: RawFd,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseCaller {
    pub project: String,
    pub container: String,
    pub login: String,
    pub home: String,
    pub namespace: String,
    pub child: String,
    pub registration: String,
    pub actor: i64,
    pub uid: i64,
    pub gid: i64,
    pub project_pid: i32,
    pub nested_pid: i32,
    pub muse_allowed: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseNested {
    pub parent: String,
    pub project: String,
    pub child: String,
    pub namespace: String,
    pub registration: String,
    pub actor: i64,
    pub pid: i32,
    pub muse: bool,
}

/// One prepared native execution: resolved caller, sanitized request,
/// broker lease/binding, credential path and unit name.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseExecution {
    pub caller: MuseCaller,
    pub request: LaunchRequest,
    pub lease: Lease,
    pub binding: Binding,
    pub path: String,
    pub unit: String,
}

/// Broker custody surface the Muse runtime needs.
pub trait MuseHooks {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String>;
    fn attach(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String>;
    fn end(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String>;
    fn authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String>;
    fn nested_authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String>;
    fn select(
        &self,
        actor: i64,
        project: &str,
        selected: &str,
        deadline: Instant,
    ) -> Result<String, String>;
}

/// Native caller resolution plus one supervised execution per launch.
pub struct MuseRuntime<E, H> {
    pub exec: E,
    pub hooks: H,
    pub binary_version: String,
    pub binary_sha256: String,
    pub(in crate::muse) nested: Mutex<HashMap<String, MuseNested>>,
}

impl<E, H> MuseRuntime<E, H> {
    pub fn new(exec: E, hooks: H, binary_version: String, binary_sha256: String) -> Self {
        MuseRuntime {
            exec,
            hooks,
            binary_version,
            binary_sha256,
            nested: Mutex::new(HashMap::new()),
        }
    }
}
