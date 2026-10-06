use std::ffi::CString;
use std::os::unix::io::RawFd;
use std::time::Instant;

use super::command::podman;
use super::filesystem::{go_errno, last_errno};
use super::project::{confirm_project, Observation};
use super::public_socket_directory;

const INTERFACE_SCRIPT: &str = "set -eu; test ! -L /run; test -d /run; test ! -L /run/soda-muse-interface; if test -e /run/soda-muse-interface; then test -d /run/soda-muse-interface; fi; mkdir -p /run/soda-muse-interface";

pub(crate) fn prepare_interface(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec",
            "--user",
            "0:0",
            &target.id,
            "/bin/sh",
            "-ceu",
            INTERFACE_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

pub(crate) struct FdGuard(pub(crate) RawFd);

impl Drop for FdGuard {
    fn drop(&mut self) {
        if self.0 >= 0 {
            unsafe { libc::close(self.0) };
        }
    }
}

pub(crate) fn attach_interface(
    target: &Observation,
    muse_socket: &str,
    deadline: Instant,
) -> Result<(), String> {
    // attachLaunchInterface with kind "muse".
    let source = public_socket_directory(muse_socket)?;
    let c = CString::new(source).map_err(|_| go_errno(libc::EINVAL))?;
    let source_fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if source_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _source_guard = FdGuard(source_fd);
    let tree = syscall_open_tree(source_fd)?;
    let _tree_guard = FdGuard(tree);
    restrict_interface_mount(tree)?;
    attach_project_mount(target, tree, deadline)
}

fn syscall_open_tree(source_fd: RawFd) -> Result<RawFd, String> {
    let empty = c"";
    let tree = unsafe {
        libc::syscall(
            libc::SYS_open_tree,
            source_fd as libc::c_long,
            empty.as_ptr(),
            (libc::OPEN_TREE_CLONE | libc::OPEN_TREE_CLOEXEC | libc::AT_EMPTY_PATH as u32)
                as libc::c_long,
        )
    };
    if tree < 0 {
        return Err(format!(
            "clone public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(tree as RawFd)
}

fn restrict_interface_mount(tree: RawFd) -> Result<(), String> {
    let mut attr: libc::mount_attr = unsafe { std::mem::zeroed() };
    attr.attr_set = libc::MOUNT_ATTR_RDONLY
        | libc::MOUNT_ATTR_NOSUID
        | libc::MOUNT_ATTR_NODEV
        | libc::MOUNT_ATTR_NOEXEC;
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_mount_setattr,
            tree as libc::c_long,
            empty.as_ptr(),
            libc::AT_EMPTY_PATH as libc::c_long,
            &mut attr as *mut libc::mount_attr,
            std::mem::size_of::<libc::mount_attr>() as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "restrict public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}

fn attach_project_mount(
    target: &Observation,
    tree: RawFd,
    deadline: Instant,
) -> Result<(), String> {
    let root = format!("/proc/{}", target.pid);
    let dest = format!("{root}/root/run/soda-muse-interface");
    let cdest = CString::new(dest).map_err(|_| go_errno(libc::EINVAL))?;
    let target_fd = unsafe {
        libc::open(
            cdest.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if target_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _target_guard = FdGuard(target_fd);
    let ns = format!("{root}/ns/mnt");
    let cns = CString::new(ns).map_err(|_| go_errno(libc::EINVAL))?;
    let namespace = unsafe { libc::open(cns.as_ptr(), libc::O_RDONLY | libc::O_CLOEXEC, 0) };
    if namespace < 0 {
        return Err(go_errno(last_errno()));
    }
    let _ns_guard = FdGuard(namespace);
    // The incarnation check runs after both opens, just before entry.
    confirm_project(target, deadline)?;
    if unsafe { libc::unshare(libc::CLONE_FS) } != 0 {
        return Err(format!(
            "separate maintenance filesystem context: {}",
            go_errno(last_errno())
        ));
    }
    if unsafe { libc::setns(namespace, libc::CLONE_NEWNS) } != 0 {
        return Err(format!(
            "enter project mount namespace: {}",
            go_errno(last_errno())
        ));
    }
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_move_mount,
            tree as libc::c_long,
            empty.as_ptr(),
            target_fd as libc::c_long,
            empty.as_ptr(),
            (libc::MOVE_MOUNT_F_EMPTY_PATH | libc::MOVE_MOUNT_T_EMPTY_PATH) as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "attach public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}
