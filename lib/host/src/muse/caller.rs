//! Caller authority resolution: kernel/project/account/registered
//! caller methods plus the cgroup/account pure helpers.

use std::time::Instant;

use super::{muse_peer_alive, MuseCaller, MuseHooks, MusePeer, MuseRuntime};
use crate::domain;
use crate::project::Executor;
use crate::terminal;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// `museKernelCaller` + project binding + registration switch.
    pub fn resolve_project(
        &self,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let mut caller = self.kernel_caller(peer)?;
        let inspected = self.inspect(&caller.container, deadline)?;
        caller.project = inspected.project;
        caller.project_pid = inspected.project_pid;
        // Re-verify through the terminal isolation gate.
        let svc = terminal::Service {
            exec: &self.exec,
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
        };
        match svc.project_container(&caller.project, true, deadline) {
            Ok(verified) if verified == caller.container => {}
            _ => return Err(terminal::err_denied()),
        }
        let project_ns = std::fs::read_link(format!("/proc/{}/ns/pid", caller.project_pid))
            .map_err(|_| terminal::err_denied())?;
        if caller.namespace != project_ns.to_string_lossy().into_owned() {
            return self.registered_caller(caller, peer, deadline);
        }
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }

    fn kernel_caller(&self, peer: &MusePeer) -> Result<MuseCaller, String> {
        let mut caller = MuseCaller::default();
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        let proc = format!("/proc/{}", peer.pid);
        let cg = std::fs::read_to_string(format!("{proc}/cgroup"))
            .map_err(|_| terminal::err_denied())?;
        caller.container = muse_project_cgroup(&cg)?;
        let mappings = std::fs::read_to_string(format!("{proc}/uid_map"))
            .map_err(|_| terminal::err_denied())?;
        caller.uid = muse_mapped_uid(&mappings, peer.uid)?;
        caller.namespace = std::fs::read_link(format!("{proc}/ns/pid"))
            .map_err(|_| terminal::err_denied())?
            .to_string_lossy()
            .into_owned();
        Ok(caller)
    }

    fn registered_caller(
        &self,
        mut caller: MuseCaller,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let registered = self.nested.lock().unwrap().get(&caller.namespace).cloned();
        let Some(registered) = registered else {
            return Err(terminal::err_denied());
        };
        if registered.parent != caller.container || registered.project != caller.project {
            return Err(terminal::err_denied());
        }
        self.validate_nested(&registered, deadline)?;
        caller.actor = registered.actor;
        caller.child = registered.child;
        caller.registration = registered.registration;
        caller.nested_pid = registered.pid;
        caller.muse_allowed = registered.muse;
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }

    /// `MuseRuntime.resolve`: full caller authority resolution.
    pub fn resolve(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        match self.resolve_inner(peer, deadline) {
            Ok(caller) => Ok(caller),
            Err(_) => Err(terminal::err_denied()),
        }
    }

    fn resolve_inner(&self, peer: &MusePeer, deadline: Instant) -> Result<MuseCaller, String> {
        let caller = self.resolve_project(peer, deadline)?;
        if !caller.child.is_empty() {
            if !caller.muse_allowed {
                return Err(terminal::err_denied());
            }
            return self.nested_caller(caller, peer, deadline);
        }
        if caller.uid == 0 {
            return Err(terminal::err_denied());
        }
        let caller = self.project_account(caller, deadline)?;
        let mut caller = caller;
        caller.actor = self.project_actor(&caller, deadline)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }

    fn project_account(
        &self,
        mut caller: MuseCaller,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/getent", "passwd", &caller.uid.to_string()],
            deadline,
        );
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        let account: Vec<String> = String::from_utf8_lossy(&body)
            .trim()
            .split(':')
            .map(|s| s.to_string())
            .collect();
        if !muse_passwd_valid(&account, caller.uid) {
            return Err(terminal::err_denied());
        }
        caller.login = account[0].clone();
        caller.home = account[5].clone();
        caller.gid = terminal::parse_go_int(&account[3]).ok_or_else(terminal::err_denied)?;
        Ok(caller)
    }

    fn project_actor(&self, caller: &MuseCaller, deadline: Instant) -> Result<i64, String> {
        let marker = format!("/var/lib/soda/accounts/{}", caller.login);
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/stat",
                "--format=%u:%g:%a:%F",
                "/var",
                "/var/lib",
                "/var/lib/soda",
                "/var/lib/soda/accounts",
                &marker,
            ],
            deadline,
        );
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if !muse_account_modes(&String::from_utf8_lossy(&body)) {
            return Err(terminal::err_denied());
        }
        let body = self.guest_refs(&caller.container, &[], &["/usr/bin/cat", &marker], deadline);
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if body.len() > 64 {
            return Err(terminal::err_denied());
        }
        match terminal::parse_go_int(String::from_utf8_lossy(&body).trim()) {
            Some(actor) if actor > 0 => Ok(actor),
            _ => Err(terminal::err_denied()),
        }
    }

    pub(in crate::muse) fn authorized_caller(
        &self,
        caller: &MuseCaller,
        peer: &MusePeer,
        deadline: Instant,
    ) -> bool {
        self.hooks
            .authorize(caller.actor, &caller.project, deadline)
            .is_ok()
            && muse_peer_alive(peer)
    }
}

/// `museProjectCgroup`: the complete OCI scope component from
/// `/proc/<pid>/cgroup`, never a substring guess.
pub fn muse_project_cgroup(value: &str) -> Result<String, String> {
    for part in value.trim().split('/') {
        if let Some(id) = part
            .strip_prefix("libpod-")
            .and_then(|s| s.strip_suffix(".scope"))
        {
            if domain::valid_container_id(id) {
                return Ok(id.to_string());
            }
        }
    }
    Err(terminal::err_denied())
}

/// `museMappedUID`: inside-UID for a host UID from a uid/gid map.
pub fn muse_mapped_uid(data: &str, uid: u32) -> Result<i64, String> {
    for line in data.split('\n') {
        let p: Vec<&str> = line.split_whitespace().collect();
        if p.len() != 3 {
            continue;
        }
        let (Some(inside), Some(outside), Some(count)) = (
            terminal::parse_go_uint(p[0], 32),
            terminal::parse_go_uint(p[1], 32),
            terminal::parse_go_uint(p[2], 32),
        ) else {
            continue;
        };
        if outside > 0 && u64::from(uid) >= outside && u64::from(uid) < outside + count {
            return Ok((inside + u64::from(uid) - outside) as i64);
        }
    }
    Err(terminal::err_denied())
}

/// `musePasswdValid`: seven-field passwd row for the mapped UID.
pub fn muse_passwd_valid(account: &[String], uid: i64) -> bool {
    if account.len() != 7 {
        return false;
    }
    domain::valid_login(&account[0])
        && account[0] != "root"
        && account[2] == uid.to_string()
        && account[5].starts_with('/')
}

/// `museAccountNode`: one root-owned account-path node.
pub fn muse_account_node(line: &str, marker: bool) -> bool {
    let parts: Vec<&str> = line.splitn(4, ':').collect();
    if parts.len() != 4 || parts[0] != "0" || parts[1] != "0" {
        return false;
    }
    let Ok(mode) = u32::from_str_radix(parts[2], 8) else {
        return false;
    };
    if mode & 0o022 != 0 {
        return false;
    }
    if marker {
        return parts[3] == "regular file" && mode == 0o600;
    }
    parts[3] == "directory"
}

/// `museAccountModes`: five account-path nodes, marker last.
pub fn muse_account_modes(body: &str) -> bool {
    let lines: Vec<&str> = body.trim().split('\n').collect();
    if lines.len() != 5 {
        return false;
    }
    for (i, line) in lines.iter().enumerate() {
        if !muse_account_node(line, i == 4) {
            return false;
        }
    }
    true
}
