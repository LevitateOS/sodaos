use std::fs::File;
use std::os::unix::io::AsRawFd;

use crate::state_json::StateValue;

use crate::account::Account;
use crate::broker::CREDENTIAL_LIMIT;
use crate::fs;
use crate::subscription_profile::{
    lease_execution_id, live_deadline, subscription_check_unit, subscription_path,
};
use crate::subscription_wire::result_object;
use crate::sys;
use crate::term;

/// `fchownat(dir, name, uid, gid, AT_SYMLINK_NOFOLLOW)`.
fn fchownat_no_follow(dir: &File, name: &str, uid: u32, gid: u32) -> std::io::Result<()> {
    let target = std::ffi::CString::new(name)
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "nul byte in name"))?;
    if unsafe {
        libc::fchownat(
            dir.as_raw_fd(),
            target.as_ptr(),
            uid as libc::uid_t,
            gid as libc::gid_t,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

pub(crate) fn is_regular(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
}

/// `subscription_auth_directory`: the `model/auth` dir must be owned by the
/// account with no group/other permission bits.
pub fn subscription_auth_directory(lease: &StateValue, account: &Account) -> Result<File, String> {
    let identifier = lease_execution_id(lease)?.to_string();
    let model = term::checked_chain(&subscription_path(&identifier)?)?;
    let directory = sys::open_child_dir(&model, "auth").map_err(|e| e.to_string())?;
    drop(model);
    let info = fs::fstat_all(&directory).map_err(|e| e.to_string())?;
    if info.st_uid != account.pw_uid || info.st_gid != account.pw_gid || info.st_mode & 0o077 != 0 {
        return Err("unsafe auth directory".to_string());
    }
    Ok(directory)
}

/// `subscription_seed`: validated credential bytes become the account-owned
/// `auth.json` (exclusive create: a second stage fails, like the `.py`).
pub fn subscription_seed(
    lease: &StateValue,
    account: &Account,
    encoded: &str,
) -> Result<(), String> {
    let state = crate::b64::decode(encoded).ok_or_else(|| "credential encoding".to_string())?;
    if state.is_empty() || state.len() > CREDENTIAL_LIMIT {
        return Err("credential size".to_string());
    }
    let text = std::str::from_utf8(&state).map_err(|_| "credential json".to_string())?;
    StateValue::parse(text).map_err(|_| "credential json".to_string())?;
    let directory = subscription_auth_directory(lease, account)?;
    fs::new_file(&directory, "auth.json", &state, 0o600)?;
    fchownat_no_follow(&directory, "auth.json", account.pw_uid, account.pw_gid)
        .map_err(|e| e.to_string())?;
    Ok(())
}

/// `subscription_stage`: deadline, liveness, then seed.
pub fn subscription_stage(
    request: &StateValue,
    lease: &StateValue,
    profile: &StateValue,
    account: &Account,
) -> Result<StateValue, String> {
    live_deadline(profile)?;
    subscription_check_unit(lease, account, true)?;
    let encoded = request
        .get("delivery")
        .and_then(|d| d.get("credential"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| "subscription request".to_string())?;
    subscription_seed(lease, account, encoded)?;
    Ok(result_object(lease, ""))
}

/// `subscription_capture`: validated read-back of the account-owned
/// credential file (regular, account uid, one link, `0o077`-clean,
/// bounded JSON).
pub fn subscription_capture(lease: &StateValue, account: &Account) -> Result<Vec<u8>, String> {
    let directory = subscription_auth_directory(lease, account)?;
    let fd = sys::open_at(
        &directory,
        "auth.json",
        libc::O_RDONLY | libc::O_NONBLOCK,
        0,
    )
    .map_err(|e| e.to_string())?;
    drop(directory);
    let info = fs::fstat_all(&fd).map_err(|e| e.to_string())?;
    if !is_regular(info.st_mode)
        || info.st_uid != account.pw_uid
        || info.st_nlink != 1
        || info.st_mode & 0o077 != 0
    {
        return Err("unsafe credential file".to_string());
    }
    let state = fs::read_up_to(&fd, CREDENTIAL_LIMIT + 1).map_err(|e| e.to_string())?;
    if state.is_empty() || state.len() > CREDENTIAL_LIMIT {
        return Err("credential size".to_string());
    }
    let text = std::str::from_utf8(&state).map_err(|_| "credential json".to_string())?;
    StateValue::parse(text).map_err(|_| "credential json".to_string())?;
    Ok(state)
}
