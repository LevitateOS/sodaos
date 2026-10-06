//! Native Muse execution runtime and launch socket.
//!
//! Port of `internal/host/terminal/muse_types.go`,
//! `muse_linux.go`, `muse_execution_linux.go`, `muse_socket_linux.go`,
//! `muse_state_linux.go`, `muse_binary_linux.go`, the launch policy from
//! `internal/identity/launch.go` + `selection.go`, and the daemon Muse
//! wiring from `internal/host/muse.go` (authorization/selection over
//! broker connections plus launch-listener setup).
//!
//! Broker custody callbacks arrive through [`MuseHooks`]; unlike Go's
//! nullable func fields the hooks are mandatory, which collapses the
//! nil-hook denials into the type system (the daemon always wires all
//! six). `MuseRuntime::prepare_execution` returns a plain
//! [`MuseExecution`] record; the argv, delivery, control and finish steps
//! are separate methods so routes can stage them around process spawn.

use std::collections::HashMap;
use std::os::unix::io::{FromRawFd, RawFd};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

use crate::domain;
use crate::json::{self, Value};
use crate::project::Executor;
use crate::terminal::{self, Binding, Delivery};
#[cfg(test)]
use crate::terminal::{AcquireRequest, Lease};

mod wire;

pub use self::wire::{
    LaunchControl, LaunchExit, LaunchRequest, NestedRegistration, MUSE_CONNECTION_SETTING,
    MUSE_LAUNCH_SOCKET,
};

mod args;

pub use self::args::muse_arguments;

mod connection;

pub use self::connection::{muse_connection_authorized, select_muse_connection, MuseConnection};

mod runtime_types;

pub use self::runtime_types::{
    MuseCaller, MuseExecution, MuseHooks, MuseNested, MusePeer, MuseRuntime,
};

mod validate;

pub use self::validate::{
    host_go_arch, muse_account_modes, muse_account_node, muse_child_pid, muse_delivery_valid,
    muse_elf, muse_host_environment, muse_mapped_uid, muse_passwd_valid, muse_project_cgroup,
    muse_project_credential_root, muse_readonly_mount, muse_registration_valid, muse_signal,
    MUSE_CHILD_INSPECT, MUSE_INSPECT,
};

mod argv;

pub use self::argv::{muse_command_argv, unit_active_argv, unit_invocation_argv};

// ---------- runtime ----------

mod execution;
mod inspect;
mod nested;
mod observe;
mod resolve;

pub(in crate::muse) use self::inspect::{
    muse_peer_alive, sleep_until, MuseInspection, MUSE_INSPECTION_SPECS,
};

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// `MuseRuntime.stage`: credential directory plus config population.
    pub fn stage(
        &self,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/install", "--directory", "--mode=0700", path],
            deadline,
        )?;
        if caller.child.is_empty() {
            self.guest_refs(
                &caller.container,
                &[],
                &[
                    "/usr/bin/mount",
                    "--types=tmpfs",
                    "--options=mode=0700,size=4M",
                    "tmpfs",
                    path,
                ],
                deadline,
            )?;
        }
        self.stage_files(caller, path, deadline)?;
        self.stage_config(caller, path, config_home, deadline)
    }

    fn stage_files(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let dir = format!("{path}/config/muse");
        self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/install",
                "--directory",
                "--mode=0700",
                &format!("--owner={}", caller.uid),
                &format!("--group={}", caller.gid),
                &dir,
            ],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", path],
            deadline,
        )?;
        let auth = format!("{path}/auth.json");
        self.guest(
            &caller.container,
            b"{}",
            &[
                "/usr/bin/dd".to_string(),
                format!("of={auth}"),
                "status=none".to_string(),
            ],
            deadline,
        )?;
        let ownership = format!("{}:{}", caller.uid, caller.gid);
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chown", &ownership, &auth],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0600", &auth],
            deadline,
        )
        .map(|_| ())
    }

    fn stage_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let source = if config_home.is_empty() {
            format!("{}/.config", caller.home)
        } else {
            config_home.to_string()
        };
        self.populate_config(caller, path, &source, deadline)?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", &format!("{path}/config")],
            deadline,
        )?;
        self.auth_mount_target(caller, path, deadline)
    }

    fn populate_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self.copy_nested_config(caller, path, source, deadline);
        }
        self.podman(
            &[],
            &[
                "exec".to_string(),
                format!("--user={}:{}", caller.uid, caller.gid),
                caller.container.clone(),
                "/usr/local/bin/muse".to_string(),
                "--soda-copy-config".to_string(),
                format!("{source}/muse"),
                format!("{path}/config/muse"),
            ],
            deadline,
        )
        .map(|_| ())
    }

    fn auth_mount_target(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self
                .guest_refs(
                    &caller.container,
                    &[],
                    &[
                        "/usr/bin/ln",
                        "--symbolic",
                        "../../auth.json",
                        &format!("{path}/config/muse/auth.json"),
                    ],
                    deadline,
                )
                .map(|_| ());
        }
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/touch", &format!("{path}/config/muse/auth.json")],
            deadline,
        )
        .map(|_| ())
    }

    fn copy_nested_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/nsenter",
                &format!("--target={}", caller.nested_pid),
                "--mount",
                "--pid",
                "--uts",
                "--ipc",
                "--net",
                "--root",
                &format!("--setuid={}", caller.uid),
                &format!("--setgid={}", caller.gid),
                "--",
                "/usr/local/bin/muse",
                "--soda-read-config",
                &format!("{source}/muse"),
            ],
            deadline,
        )?;
        if body.len() > 3 << 20 {
            return Err(terminal::err_denied());
        }
        let mut view = decode_config_view(&body)?;
        for name in ["settings.json", "trust.json"] {
            let Some(value) = view.remove(name) else {
                continue;
            };
            let target = format!("{path}/config/muse/{name}");
            let dd = [
                "/usr/bin/dd".to_string(),
                format!("of={target}"),
                "status=none".to_string(),
            ];
            self.guest(&caller.container, &value, &dd, deadline)?;
            let mut value = value;
            for byte in value.iter_mut() {
                *byte = 0;
            }
            let ownership = format!("{}:{}", caller.uid, caller.gid);
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chown", &ownership, &target],
                deadline,
            )?;
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chmod", "0600", &target],
                deadline,
            )?;
        }
        Ok(())
    }

    /// `MuseRuntime.stopExecution`: stop the unit, verify its cgroup is
    /// gone, retire files, return custody.
    pub fn stop_execution(
        &self,
        binding: &Binding,
        actor: i64,
        lease_id: &str,
        return_custody: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = binding.project.clone();
        let unit = format!("soda-muse-{}.service", binding.id);
        let stop_result = self.guest_refs(
            &container,
            &[],
            &["/usr/bin/systemctl", "stop", &unit],
            deadline,
        );
        let body = self
            .guest(&container, &[], &unit_active_argv(&unit), deadline)
            .map_err(|_| terminal::err_uncertain())?;
        if String::from_utf8_lossy(&body).trim() != "inactive" {
            return Err(terminal::err_uncertain());
        }
        if stop_result.is_err() {
            self.guest_refs(
                &container,
                &[],
                &[
                    "/usr/bin/test",
                    "!",
                    "-d",
                    &format!("/sys/fs/cgroup/system.slice/{unit}"),
                ],
                deadline,
            )
            .map_err(|_| terminal::err_uncertain())?;
        }
        self.retire_execution_files(binding, deadline)?;
        if return_custody {
            return self.hooks.end(actor, lease_id, deadline);
        }
        Ok(())
    }

    /// `MuseRuntime.ValidateMuseBinding`: exact incarnation + active unit.
    pub fn validate_muse_binding(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        if binding.validate().is_err()
            || !terminal::valid_terminal_id(&binding.id)
            || !domain::valid_container_id(&binding.project)
            || !domain::valid_login(&binding.login)
        {
            return Err(terminal::err_denied());
        }
        let body = self
            .guest(
                &binding.project,
                &[],
                &unit_active_argv(&format!("soda-muse-{}.service", binding.id)),
                deadline,
            )
            .map_err(|_| terminal::err_stale())?;
        if String::from_utf8_lossy(&body).trim() != "active" {
            return Err(terminal::err_stale());
        }
        Ok(())
    }

    /// `MuseRuntime.awaitUnit`: active state within 5s.
    pub fn await_unit(&self, container: &str, unit: &str, deadline: Instant) -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(5);
        loop {
            if let Ok(body) = self.guest(container, &[], &unit_active_argv(unit), deadline) {
                if String::from_utf8_lossy(&body).trim() == "active" {
                    return Ok(());
                }
            }
            let now = Instant::now();
            if now >= deadline {
                return Err("context deadline exceeded".to_string());
            }
            if now >= end {
                return Err(terminal::err_denied());
            }
            if sleep_until(now + Duration::from_millis(50), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
    }

    /// `MuseRuntime.Muse`: broker-selected operations on recorded boundaries.
    pub fn muse_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !muse_delivery_valid(delivery) {
            return Err(terminal::err_denied());
        }
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        match binding.scope.as_str() {
            "muse-project" => self.project_operation(action, delivery, deadline)?,
            _ => return Err(terminal::err_denied()),
        }
        Ok(delivery.clone())
    }

    fn project_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        let unit = format!("soda-muse-{}.service", binding.id);
        if !muse_project_credential_root(binding) {
            return Err(terminal::err_denied());
        }
        if let Ok(body) = self.guest(
            &binding.project,
            &[],
            &unit_invocation_argv(&unit),
            deadline,
        ) {
            let current = String::from_utf8_lossy(&body).trim().to_string();
            if !current.is_empty() && current != binding.invocation_id {
                return Err(terminal::err_stale());
            }
        }
        if action == "validate" {
            return self.validate_muse_binding(binding, deadline);
        }
        if action != "stop" {
            return Err(terminal::err_denied());
        }
        self.stop_execution(
            binding,
            delivery.lease.actor_id,
            &delivery.lease.id,
            false,
            deadline,
        )
    }

    // ---------- state cleanup (muse_state_linux.go) ----------

    /// `MuseRuntime.cleanupExecutionState`: remove the guest state dir.
    pub fn cleanup_execution_state(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = state_container(binding)?;
        if container.is_empty() {
            return Err(terminal::err_denied());
        }
        match self.invoke_state(
            binding,
            &[
                "container".to_string(),
                "exists".to_string(),
                container.clone(),
            ],
            deadline,
        ) {
            Ok(_) => {}
            Err(err) if terminal::exit_code_of(&err) == Some(1) => return Ok(()),
            Err(_) => return Err(terminal::err_uncertain()),
        }
        let state = format!("/tmp/soda-muse-state-{}", binding.id);
        let owner = format!("--user={}:{}", binding.uid, binding.gid);
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner.clone(),
                container.clone(),
                "/usr/bin/rm".to_string(),
                "--recursive".to_string(),
                "--force".to_string(),
                "--".to_string(),
                state.clone(),
            ],
            deadline,
        )
        .map_err(|_| terminal::err_uncertain())?;
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner,
                container,
                "/usr/bin/test".to_string(),
                "!".to_string(),
                "-e".to_string(),
                state,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }

    fn invoke_state(
        &self,
        binding: &Binding,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if binding.scope == "muse-project" && !binding.child_id.is_empty() {
            let mut full = vec!["/usr/bin/podman".to_string(), "--remote=false".to_string()];
            full.extend(args.iter().cloned());
            return self.guest(&binding.project, &[], &full, deadline);
        }
        self.podman(&[], args, deadline)
    }

    fn retire_mount(&self, container: &str, path: &str, deadline: Instant) -> Result<(), String> {
        if self
            .guest_refs(container, &[], &["/usr/bin/umount", path], deadline)
            .is_ok()
        {
            return Ok(());
        }
        if self
            .guest_refs(
                container,
                &[],
                &["/usr/bin/test", "!", "-e", path],
                deadline,
            )
            .is_ok()
        {
            return Ok(());
        }
        match self.guest_refs(
            container,
            &[],
            &["/usr/bin/mountpoint", "--quiet", path],
            deadline,
        ) {
            Err(err) if terminal::exit_code_of(&err) == Some(32) => Ok(()),
            _ => Err(terminal::err_uncertain()),
        }
    }

    fn retire_execution_files(&self, binding: &Binding, deadline: Instant) -> Result<(), String> {
        self.cleanup_execution_state(binding, deadline)?;
        if !binding.credential_root.contains("/nested/") {
            self.retire_mount(&binding.project, &binding.credential_root, deadline)?;
        }
        self.guest_refs(
            &binding.project,
            &[],
            &[
                "/usr/bin/rm",
                "--recursive",
                "--force",
                "--",
                &binding.credential_root,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }
}

/// `museResize`: PTY resize over the stdin fd.
pub fn muse_resize(stdin_fd: RawFd, control: &LaunchControl) -> Result<(), String> {
    if control.signal != 0 || control.cols == 0 || control.rows == 0 {
        return Err(terminal::err_denied());
    }
    let ws = libc::winsize {
        ws_row: control.rows,
        ws_col: control.cols,
        ws_xpixel: 0,
        ws_ypixel: 0,
    };
    // SAFETY: ioctl with a valid winsize pointer.
    let result = unsafe { libc::ioctl(stdin_fd, libc::TIOCSWINSZ, &ws) };
    if result < 0 {
        return Err(terminal::err_denied());
    }
    Ok(())
}

/// `stateContainer`: execution state owner (child for nested runs).
pub fn state_container(binding: &Binding) -> Result<String, String> {
    if !terminal::valid_terminal_id(&binding.id) || binding.uid < 0 || binding.gid < 0 {
        return Err(terminal::err_denied());
    }
    let mut container = binding.project.clone();
    if binding.scope == "muse-project" && !binding.child_id.is_empty() {
        container = binding.child_id.clone();
    }
    if !domain::valid_container_id(&container) {
        return Err(terminal::err_denied());
    }
    Ok(container)
}

/// Daemon `OpenMuseListener` filesystem setup: create the interface
/// directory, require it empty, require the socket path absent. Returns
/// the directory path for the caller to bind.
pub fn prepare_muse_listener_dir(socket_path: &str) -> Result<(), String> {
    let dir = match socket_path.rfind('/') {
        Some(0) => "/",
        Some(i) => &socket_path[..i],
        None => ".",
    };
    std::fs::create_dir_all(dir)
        .map_err(|e| format!("muse interface directory unavailable: {e}"))?;
    // Go checks `ReadDir` success plus emptiness with one error.
    let empty = std::fs::read_dir(dir)
        .map(|mut entries| entries.next().is_none())
        .unwrap_or(false);
    if !empty {
        return Err(
            "muse interface directory must be empty before launch service startup".to_string(),
        );
    }
    match std::fs::symlink_metadata(socket_path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err("muse launch socket is occupied".to_string()),
    }
}

// ---------- launch socket (muse_socket_linux.go) ----------

/// `SO_PEERPIDFD` (Linux 6.5+), absent from libc: Go uses the literal too.
const SO_PEERPIDFD: i32 = 77;

/// `musePeer`: kernel-attested pid/uid/gid plus a pinning pidfd.
/// Fails closed when the kernel lacks `SO_PEERPIDFD`.
pub fn muse_peer_from_fd(fd: RawFd) -> Result<MusePeer, String> {
    // SAFETY: getsockopt with correctly sized outputs.
    unsafe {
        let mut cred: libc::ucred = std::mem::zeroed();
        let mut cred_len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut libc::c_void,
            &mut cred_len,
        ) != 0
        {
            return Err("muse peer credentials unavailable".to_string());
        }
        let mut pidfd: libc::c_int = -1;
        let mut pidfd_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            SO_PEERPIDFD,
            &mut pidfd as *mut _ as *mut libc::c_void,
            &mut pidfd_len,
        ) != 0
        {
            return Err("muse peer pidfd unavailable".to_string());
        }
        Ok(MusePeer {
            pid: cred.pid,
            uid: cred.uid,
            gid: cred.gid,
            pidfd,
        })
    }
}

fn close_fds(fds: &[RawFd]) {
    for fd in fds {
        // SAFETY: fds came from SCM_RIGHTS; double close is avoided by
        // construction (each fd closed exactly once).
        unsafe {
            libc::close(*fd);
        }
    }
}

/// Parse `SCM_RIGHTS` fds out of one control buffer (glibc `CMSG_NXTHDR`
/// iteration, alignment included).
pub fn parse_unix_rights(control: &[u8], controllen: usize) -> Result<Vec<RawFd>, String> {
    let mut fds = Vec::new();
    if controllen == 0 {
        return Ok(fds);
    }
    // SAFETY: manual cmsg iteration over the kernel-filled prefix.
    unsafe {
        let mhdr = libc::msghdr {
            msg_name: std::ptr::null_mut(),
            msg_namelen: 0,
            msg_iov: std::ptr::null_mut(),
            msg_iovlen: 0,
            msg_control: control.as_ptr() as *mut libc::c_void,
            msg_controllen: controllen as _,
            msg_flags: 0,
        };
        let align = std::mem::size_of::<usize>();
        let mut cmsg = libc::CMSG_FIRSTHDR(&mhdr);
        while !cmsg.is_null() {
            if (*cmsg).cmsg_level == libc::SOL_SOCKET && (*cmsg).cmsg_type == libc::SCM_RIGHTS {
                let data = libc::CMSG_DATA(cmsg);
                let end = (cmsg as *const u8).add((*cmsg).cmsg_len as usize);
                let mut cursor = data;
                while (cursor as *const u8).add(std::mem::size_of::<libc::c_int>()) <= end {
                    fds.push(*(cursor as *const libc::c_int));
                    cursor = cursor.add(std::mem::size_of::<libc::c_int>());
                }
            }
            let next = (cmsg as usize + ((*cmsg).cmsg_len as usize).div_ceil(align) * align)
                as *const libc::cmsghdr;
            let limit = (mhdr.msg_control as usize + mhdr.msg_controllen as usize) as *const u8;
            if (next as *const u8).add(std::mem::size_of::<libc::cmsghdr>()) > limit {
                break;
            }
            cmsg = next as *mut libc::cmsghdr;
        }
    }
    Ok(fds)
}

/// One launch request plus its stdio files (`museRequest`).
/// `files` holds `None`s when the caller passed no descriptors (register).
#[derive(Debug)]
pub struct MuseRequest {
    pub request: LaunchRequest,
    pub files: [Option<std::fs::File>; 3],
}

/// `museRequest` over an already-accepted fd: one 64KB datagram plus up
/// to 3 SCM_RIGHTS descriptors, 5s read budget.
pub fn muse_request_from_fd(fd: RawFd) -> Result<MuseRequest, String> {
    let mut body = vec![0u8; 65536];
    // SAFETY: CMSG_SPACE for 3 ints.
    let cmsg_len = unsafe { libc::CMSG_SPACE(3 * 4) } as usize;
    let mut control = vec![0u8; cmsg_len];
    let (n, controllen, flags) = unsafe {
        let mut iov = libc::iovec {
            iov_base: body.as_mut_ptr() as *mut libc::c_void,
            iov_len: body.len(),
        };
        let mut hdr: libc::msghdr = std::mem::zeroed();
        hdr.msg_iov = &mut iov;
        hdr.msg_iovlen = 1;
        hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
        hdr.msg_controllen = control.len() as _;
        // 5s read budget like Go's `SetReadDeadline`.
        let mut pfd = libc::pollfd {
            fd,
            events: libc::POLLIN,
            revents: 0,
        };
        if libc::poll(&mut pfd, 1, 5000) <= 0 {
            return Err("muse launch request unreadable".to_string());
        }
        let n = libc::recvmsg(fd, &mut hdr, 0);
        if n < 0 {
            return Err("muse launch request unreadable".to_string());
        }
        (n as usize, hdr.msg_controllen as usize, hdr.msg_flags)
    };
    let fds = parse_unix_rights(&control, controllen)?;
    if flags & (libc::MSG_TRUNC | libc::MSG_CTRUNC) != 0 || (!fds.is_empty() && fds.len() != 3) {
        close_fds(&fds);
        return Err("invalid launch descriptors".to_string());
    }
    let mut files: [Option<std::fs::File>; 3] = [None, None, None];
    if fds.len() == 3 {
        for (i, fd) in fds.iter().enumerate() {
            // SAFETY: set CLOEXEC on the received fd, then adopt it.
            unsafe {
                let flags = libc::fcntl(*fd, libc::F_GETFD);
                if flags >= 0 {
                    libc::fcntl(*fd, libc::F_SETFD, flags | libc::FD_CLOEXEC);
                }
                files[i] = Some(std::fs::File::from_raw_fd(*fd));
            }
        }
    }
    let request = LaunchRequest::decode(&body[..n])?;
    if !muse_descriptors_valid(&request, &files) {
        return Err("invalid launch descriptors".to_string());
    }
    request.validate()?;
    Ok(MuseRequest { request, files })
}

/// `museDescriptorsValid`: register carries no stdio, launches carry all three.
pub fn muse_descriptors_valid(request: &LaunchRequest, files: &[Option<std::fs::File>; 3]) -> bool {
    if request.register.is_some() {
        return files[0].is_none();
    }
    files[0].is_some() && files[1].is_some() && files[2].is_some()
}

/// `museCommandExit`: shell wait status to launch outcome.
pub fn muse_command_exit(status: std::io::Result<std::process::ExitStatus>) -> LaunchExit {
    use std::os::unix::process::ExitStatusExt;
    match status {
        Ok(status) => match status.code() {
            Some(code) => LaunchExit {
                code,
                error: String::new(),
            },
            None => LaunchExit {
                code: 128 + status.signal().unwrap_or(0),
                error: String::new(),
            },
        },
        Err(_) => LaunchExit {
            code: 1,
            error: String::new(),
        },
    }
}

/// Split one complete JSON object off the front of `buffer`, tracking
/// strings and escapes like a streaming decoder. Returns the object
/// length when balanced.
pub fn split_json_object(buffer: &[u8]) -> Option<usize> {
    let mut depth = 0i32;
    let mut in_string = false;
    let mut escape = false;
    let mut started = false;
    for (i, &b) in buffer.iter().enumerate() {
        if in_string {
            if escape {
                escape = false;
            } else if b == b'\\' {
                escape = true;
            } else if b == b'"' {
                in_string = false;
            }
            continue;
        }
        match b {
            b'"' => in_string = true,
            b'{' => {
                depth += 1;
                started = true;
            }
            b'}' => {
                depth -= 1;
                if started && depth == 0 {
                    return Some(i + 1);
                }
                if depth < 0 {
                    return None;
                }
            }
            _ => {}
        }
    }
    None
}

/// `MuseLaunch`: dedicated SOCK_SEQPACKET launch listener service.
/// Served behind an [`Arc`] so each connection runs detached, like Go's
/// per-connection goroutine.
pub struct MuseLaunch<E, H> {
    pub runtime: MuseRuntime<E, H>,
}

impl<E, H> MuseLaunch<E, H> {
    pub fn new(runtime: MuseRuntime<E, H>) -> Self {
        MuseLaunch { runtime }
    }
}

impl<E: Executor + Send + Sync + 'static, H: MuseHooks + Send + Sync + 'static> MuseLaunch<E, H> {
    /// `MuseLaunch.Serve`: accept loop until `shutdown` is set, then join
    /// every live connection (Go's `wg.Wait`).
    /// `listen_fd` is a bound SOCK_SEQPACKET listener (owned by the caller).
    pub fn serve(
        self: &std::sync::Arc<Self>,
        listen_fd: RawFd,
        shutdown: &AtomicBool,
    ) -> Result<(), String> {
        // Non-blocking accept with a stop poll, mirroring Go's
        // context-cancelled `AcceptUnix`.
        unsafe {
            let flags = libc::fcntl(listen_fd, libc::F_GETFL);
            if flags >= 0 {
                libc::fcntl(listen_fd, libc::F_SETFL, flags | libc::O_NONBLOCK);
            }
        }
        let mut handles = Vec::new();
        let outcome = loop {
            if shutdown.load(Ordering::SeqCst) {
                break Ok(());
            }
            let mut pfd = libc::pollfd {
                fd: listen_fd,
                events: libc::POLLIN,
                revents: 0,
            };
            // SAFETY: valid one-element pollfd array.
            let ready = unsafe { libc::poll(&mut pfd, 1, 100) };
            if ready < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            if ready == 0 {
                continue;
            }
            // SAFETY: accept on a listening unix socket.
            let conn =
                unsafe { libc::accept(listen_fd, std::ptr::null_mut(), std::ptr::null_mut()) };
            if conn < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
                if errno == libc::EAGAIN || errno == libc::EWOULDBLOCK || errno == libc::EINTR {
                    continue;
                }
                break Err(format!("muse accept failed: errno {errno}"));
            }
            let service = self.clone();
            handles.push(std::thread::spawn(move || {
                service.serve_one(conn, &service)
            }));
        };
        for handle in handles {
            let _ = handle.join();
        }
        outcome
    }

    fn serve_one(&self, conn: RawFd, service: &std::sync::Arc<Self>) {
        let result = self.serve_connection(conn, service);
        // `json.Encoder.Encode(result)`: framed with one newline.
        let line = result.encode_line();
        let bytes = line.as_bytes();
        let mut written = 0;
        while written < bytes.len() {
            // SAFETY: send on the connected fd.
            let n = unsafe {
                libc::send(
                    conn,
                    bytes[written..].as_ptr() as *const libc::c_void,
                    bytes.len() - written,
                    libc::MSG_NOSIGNAL,
                )
            };
            if n <= 0 {
                break;
            }
            written += n as usize;
        }
        // SAFETY: close the accepted fd exactly once.
        unsafe {
            libc::close(conn);
        }
    }

    fn serve_connection(&self, conn: RawFd, service: &std::sync::Arc<Self>) -> LaunchExit {
        let peer = match muse_peer_from_fd(conn) {
            Ok(peer) => peer,
            Err(_) => return LaunchExit::denied(),
        };
        // SAFETY: close the pidfd exactly once at the end.
        struct Closer(RawFd);
        impl Drop for Closer {
            fn drop(&mut self) {
                unsafe {
                    libc::close(self.0);
                }
            }
        }
        let _pidfd = Closer(peer.pidfd);
        let received = match muse_request_from_fd(conn) {
            Ok(received) => received,
            Err(_) => return LaunchExit::denied(),
        };
        if let Some(register) = &received.request.register {
            let deadline = Instant::now() + Duration::from_secs(30);
            return match self.runtime.register_nested(&peer, register, deadline) {
                Ok(()) => LaunchExit {
                    code: 0,
                    error: String::new(),
                },
                Err(_) => LaunchExit::denied(),
            };
        }
        let [stdin, stdout, stderr] = received.files;
        let (Some(stdin), Some(stdout), Some(stderr)) = (stdin, stdout, stderr) else {
            return LaunchExit::denied();
        };
        self.shell(
            conn,
            &peer,
            &received.request,
            [stdin, stdout, stderr],
            service,
        )
    }

    /// `MuseLaunch.shell`: spawn, deliver, supervise, finish.
    pub fn shell(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        service: &std::sync::Arc<Self>,
    ) -> LaunchExit {
        self.shell_inner(conn, peer, request, stdio, Some(service.clone()))
    }

    fn shell_inner(
        &self,
        conn: RawFd,
        peer: &MusePeer,
        request: &LaunchRequest,
        stdio: [std::fs::File; 3],
        supervisor_owner: Option<std::sync::Arc<Self>>,
    ) -> LaunchExit {
        use std::os::unix::io::AsRawFd;
        let session_end = Instant::now() + Duration::from_secs(12 * 3600);
        let mut execution = match self.runtime.prepare_execution(peer, request, session_end) {
            Ok(execution) => execution,
            Err(_) => return LaunchExit::denied(),
        };
        // The supervisor resizes through the parent's stdin handle.
        // SAFETY: dup before spawn transfers ownership of stdio to the child.
        let stdin_fd = unsafe { libc::dup(stdio[0].as_raw_fd()) };
        let mut child = match spawn_execution(&execution, stdio) {
            Ok(child) => child,
            Err(_) => {
                unsafe {
                    libc::close(stdin_fd);
                }
                self.finish_unconfirmed(&execution);
                return LaunchExit::denied();
            }
        };
        if self
            .runtime
            .deliver_execution(&mut execution, session_end)
            .is_err()
        {
            let _ = child.kill();
            let _ = child.wait();
            unsafe {
                libc::close(stdin_fd);
            }
            self.finish_unconfirmed(&execution);
            return LaunchExit::denied();
        }
        // Control supervisor: any channel failure kills the session, like
        // Go's `museControls` cancel.
        if let Some(owner) = supervisor_owner {
            // SAFETY: dup the conn for the supervisor; it closes its copy.
            let conn_dup = unsafe { libc::dup(conn) };
            let child_pid = child.id() as i32;
            let snapshot = execution.clone();
            std::thread::spawn(move || {
                owner.control_loop(conn_dup, stdin_fd, child_pid, &snapshot, session_end);
                unsafe {
                    libc::close(conn_dup);
                    libc::close(stdin_fd);
                    libc::kill(child_pid, libc::SIGKILL);
                }
            });
        } else {
            unsafe {
                libc::close(stdin_fd);
            }
        }
        let mut result = muse_command_exit(child.wait());
        let cleanup = Instant::now() + Duration::from_secs(30);
        if self
            .runtime
            .stop_execution(
                &execution.binding,
                execution.caller.actor,
                &execution.lease.id,
                true,
                cleanup,
            )
            .is_err()
        {
            result = LaunchExit::cleanup_unconfirmed();
        }
        result
    }

    fn finish_unconfirmed(&self, execution: &MuseExecution) {
        let cleanup = Instant::now() + Duration::from_secs(30);
        let _ = self.runtime.stop_execution(
            &execution.binding,
            execution.caller.actor,
            &execution.lease.id,
            true,
            cleanup,
        );
    }

    /// Control supervisor (`museControls`): streaming strict control
    /// messages, 1MB total, unknown fields rejected. Any failure ends the
    /// loop; the spawner kills the session afterwards.
    pub fn control_loop(
        &self,
        conn: RawFd,
        stdin_fd: RawFd,
        child_pid: i32,
        execution: &MuseExecution,
        session_end: Instant,
    ) {
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 4096];
        loop {
            if Instant::now() >= session_end {
                return;
            }
            // SAFETY: recv into the scratch chunk.
            let n = unsafe {
                libc::recv(
                    conn,
                    chunk.as_mut_ptr() as *mut libc::c_void,
                    chunk.len(),
                    0,
                )
            };
            if n <= 0 {
                return;
            }
            buffer.extend_from_slice(&chunk[..n as usize]);
            if buffer.len() > 1 << 20 {
                return;
            }
            loop {
                let trim: Vec<u8> = buffer
                    .iter()
                    .skip_while(|b| b.is_ascii_whitespace())
                    .cloned()
                    .collect();
                if trim.is_empty() {
                    buffer.clear();
                    break;
                }
                if trim[0] != b'{' {
                    return;
                }
                let Some(len) = split_json_object(&trim) else {
                    buffer = trim;
                    break;
                };
                let control = match LaunchControl::decode(&trim[..len]) {
                    Ok(control) => control,
                    Err(_) => return,
                };
                if self
                    .runtime
                    .control_execution(execution, stdin_fd, Some(child_pid), &control, session_end)
                    .is_err()
                {
                    return;
                }
                buffer = trim[len..].to_vec();
            }
        }
    }
}

/// Spawn the prepared execution with owned stdio files.
fn spawn_execution(
    execution: &MuseExecution,
    stdio: [std::fs::File; 3],
) -> Result<std::process::Child, String> {
    let [stdin, stdout, stderr] = stdio;
    let argv = muse_command_argv(
        &execution.caller,
        &execution.request,
        &execution.unit,
        &execution.path,
    );
    std::process::Command::new("/usr/bin/podman")
        .args(&argv)
        .env_clear()
        .envs(muse_host_environment())
        .stdin(std::process::Stdio::from(stdin))
        .stdout(std::process::Stdio::from(stdout))
        .stderr(std::process::Stdio::from(stderr))
        .spawn()
        .map_err(|e| format!("/usr/bin/podman failed: {e}"))
}

/// Tolerant `map[string][]byte` decode for nested config views
/// (`encoding/json` into `map[string][]byte`: base64 strings or numeric
/// arrays, like [`Kind::Bytes`](crate::json::Kind::Bytes)).
pub fn decode_config_view(body: &[u8]) -> Result<HashMap<String, Vec<u8>>, String> {
    let v = json::decode_tolerant(body).map_err(|_| terminal::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(terminal::err_denied());
    };
    let mut out = HashMap::with_capacity(fields.len());
    for (key, value) in fields {
        let bytes = match value {
            Value::Null => Vec::new(),
            Value::Str(s) => {
                crate::ssh::b64_decode_go(s.as_bytes()).map_err(|_| terminal::err_denied())?
            }
            Value::Array(items) => {
                let mut bytes = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Number(lit) => {
                            let n: i64 = lit.parse().map_err(|_| terminal::err_denied())?;
                            if !(0..=255).contains(&n) {
                                return Err(terminal::err_denied());
                            }
                            bytes.push(n as u8);
                        }
                        _ => return Err(terminal::err_denied()),
                    }
                }
                bytes
            }
            _ => return Err(terminal::err_denied()),
        };
        out.insert(key.clone(), bytes);
    }
    Ok(out)
}

#[cfg(test)]
mod tests;
