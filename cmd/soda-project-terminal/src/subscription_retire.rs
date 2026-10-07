use std::fs::File;

use crate::state_json::StateValue;

use crate::account::Account;
use crate::fs;
use crate::subscription_cgroup::{
    subscription_cgroup, subscription_freeze, subscription_kernel_write,
};
use crate::subscription_credentials::subscription_capture;
use crate::subscription_profile::{lease_execution_id, subscription_check_unit, subscription_path};
use crate::subscription_wire::result_object;
use crate::svc;
use crate::sys;
use crate::term;

/// Unlink one name, ignoring absence (other errors propagate).
fn unlink_missing_ok(dir: &File, name: &str) -> Result<(), String> {
    match fs::unlink_at(dir, name) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(err.to_string()),
    }
}

fn rmdir_missing_ok(dir: &File, name: &str) -> Result<(), String> {
    match fs::rmdir_at(dir, name) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(err.to_string()),
    }
}

/// `subscription_remove_model`: subscription records plus the model tree.
/// The `.py` removes `model/auth` and `model/harness` through slash paths
/// relative to the held dir fd; the equivalent here opens `model` once.
pub fn subscription_remove_model(directory: &File) -> Result<(), String> {
    for name in ["subscription", "subscription-started", "subscription-unit"] {
        unlink_missing_ok(directory, name)?;
    }
    match sys::open_child_dir(directory, "model") {
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(err) => return Err(err.to_string()),
        Ok(model) => {
            rmdir_missing_ok(&model, "auth")?;
            rmdir_missing_ok(&model, "harness")?;
        }
    }
    rmdir_missing_ok(directory, "model")
}

/// `os.path.ismount`: device differs from the parent, or same file (binds
/// and filesystem roots); every stat failure is "not a mount".
pub fn is_mount(path: &str) -> bool {
    let target = match std::ffi::CString::new(path) {
        Ok(target) => target,
        Err(_) => return false,
    };
    let mut first: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(target.as_ptr(), &mut first) } != 0 {
        return false;
    }
    let parent = match path.rsplit_once('/') {
        Some((head, _)) if !head.is_empty() => head,
        _ => "/",
    };
    let parent = match std::ffi::CString::new(parent) {
        Ok(parent) => parent,
        Err(_) => return false,
    };
    let mut second: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(parent.as_ptr(), &mut second) } != 0 {
        return false;
    }
    first.st_dev != second.st_dev || first.st_ino == second.st_ino
}

/// `subscription_retire`: unmount the model mounts, remove the model tree
/// and the exact run-owned files.
pub fn subscription_retire(lease: &StateValue, account: &Account) -> Result<(), String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let path = term::terminal_path(&identifier)?;
    let model = subscription_path(&identifier)?;
    for name in ["auth", "harness"] {
        let mount = format!("{model}/{name}");
        if is_mount(&mount) {
            sys::run_checked(&["/usr/bin/umount".to_string(), mount], 5)
                .map_err(|e| format!("umount: {e}"))?;
        }
    }
    let directory = term::checked_chain(&path)?;
    subscription_remove_model(&directory)?;
    term::remove_owned_files(&path, &directory, account)?;
    Ok(())
}

/// `subscription_finish`: freeze a live unit, capture on `finish`, always
/// kill/close/stop, retire on `stop`. The `.py` `finally` order is exact:
/// a failed kill skips the stop, and a `finally` failure replaces the body
/// error.
pub fn subscription_finish(
    action: &str,
    lease: &StateValue,
    _profile: &StateValue,
    account: &Account,
) -> Result<StateValue, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let live = subscription_check_unit(lease, account, false)?;
    let group = if live {
        Some(subscription_cgroup(&identifier)?)
    } else {
        None
    };
    let body: Result<Vec<u8>, String> = (|| {
        if let Some(group) = group.as_ref() {
            subscription_freeze(group)?;
        }
        if action == "finish" {
            return subscription_capture(lease, account);
        }
        Ok(Vec::new())
    })();
    let mut raised: Result<(), String> = Ok(());
    if let Some(group) = group {
        let killed = subscription_kernel_write(&group, "cgroup.kill", b"1\n");
        drop(group);
        if let Err(err) = killed {
            raised = Err(err);
        }
        // A failed kill aborts the rest of the `.py` finally body.
        if raised.is_ok() {
            if let Err(err) = svc::stop_service(&identifier, account) {
                raised = Err(err);
            }
        }
    } else if let Err(err) = svc::stop_service(&identifier, account) {
        raised = Err(err);
    }
    // A `finally` failure replaces the body error.
    raised?;
    let state = body?;
    if action == "stop" {
        subscription_retire(lease, account)?;
    }
    Ok(result_object(lease, &crate::b64::encode(&state)))
}
