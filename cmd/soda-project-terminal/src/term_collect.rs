use std::collections::HashSet;
use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use crate::account::Account;
use crate::fs;
use crate::svc;
use crate::sys;
use crate::term_binding::{binding_record, permit_live, read_reservation};
use crate::term_paths::{
    checked_chain, fstatat, list_dir_names, s_isreg, s_issock, terminal_path, TERMINALS,
};

pub(crate) fn terminal_directories() -> Result<Vec<String>, String> {
    let mut identifiers = Vec::new();
    for entry in std::fs::read_dir(TERMINALS).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        identifiers.push(
            entry
                .file_name()
                .into_string()
                .map_err(|_| "terminal name encoding".to_string())?,
        );
    }
    identifiers.sort();
    if identifiers.len() > 128 {
        return Err("native terminal inventory bound".to_string());
    }
    for identifier in &identifiers {
        terminal_path(identifier)?;
    }
    Ok(identifiers)
}

fn read_single_byte(file: &File) -> Result<Option<u8>, String> {
    let mut byte = [0u8; 1];
    let got = loop {
        let got =
            unsafe { libc::read(file.as_raw_fd(), byte.as_mut_ptr() as *mut libc::c_void, 1) };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(io::Error::last_os_error().to_string());
        }
        break got;
    };
    Ok(if got == 1 { Some(byte[0]) } else { None })
}

/// Retire a crash stub (missing/corrupt binding, no unit); the bare-raise
/// sites below return the *original* binding failure, message free-form.
fn stub_retire(path: &str, identifier: &str, directory: &File) -> Result<(), String> {
    let names: HashSet<String> = list_dir_names(directory)?.into_iter().collect();
    if names.iter().any(|n| n != "binding") {
        return Err("terminal binding".to_string());
    }
    if svc::service_state(identifier, None)? != "inactive" {
        return Err("terminal binding".to_string());
    }
    if !svc::cgroup_empty(identifier)? {
        return Err("terminal binding".to_string());
    }
    if names.contains("binding") {
        let fd = fs::root_file(directory, "binding", false)?;
        if read_single_byte(&fd)?.is_some() {
            return Err("unknown terminal binding".to_string());
        }
        drop(fd);
        fs::unlink_at(directory, "binding").map_err(|e| e.to_string())?;
    }
    std::fs::remove_dir(path).map_err(|e| e.to_string())?;
    Ok(())
}

fn remove_screen_contents(directory: &File, account: &Account) -> Result<(), String> {
    let screen = sys::open_child_dir(directory, "screen").map_err(|e| e.to_string())?;
    let (uid, mode) = fs::fstat_uid_mode(&screen).map_err(|e| e.to_string())?;
    // Failed startup may precede the root seal; accept only the original account.
    if (uid != 0 && uid != account.pw_uid) || mode & 0o022 != 0 {
        return Err("unexpected terminal screen".to_string());
    }
    for name in list_dir_names(&screen)? {
        let info = fstatat(&screen, &name).map_err(|e| e.to_string())?;
        let expected = (name == "socket" && s_issock(info.st_mode))
            || (name == "socket.lock" && s_isreg(info.st_mode));
        if !expected || info.st_uid != account.pw_uid || info.st_nlink != 1 {
            return Err("unexpected terminal socket".to_string());
        }
        fs::unlink_at(&screen, &name).map_err(|e| e.to_string())?;
    }
    Ok(())
}

pub(crate) fn remove_owned_files(
    path: &str,
    directory: &File,
    account: &Account,
) -> Result<(), String> {
    // Only exact run-owned files after unit shutdown, never recursive removal.
    let names: HashSet<String> = list_dir_names(directory)?.into_iter().collect();
    let allowed: HashSet<&str> = [
        "binding",
        "reservation",
        "name",
        "name.next",
        "writer",
        "tmux.conf",
        "ready",
        "screen",
    ]
    .into_iter()
    .collect();
    if names.iter().any(|n| !allowed.contains(n.as_str())) {
        return Err("unexpected terminal files".to_string());
    }
    if names.contains("screen") {
        remove_screen_contents(directory, account)?;
    }
    for name in names.iter().filter(|n| n.as_str() != "screen") {
        let info = fstatat(directory, name).map_err(|e| e.to_string())?;
        if !s_isreg(info.st_mode) || info.st_uid != 0 || info.st_nlink != 1 {
            return Err("unexpected terminal file".to_string());
        }
    }
    for name in names.iter().filter(|n| n.as_str() != "screen") {
        fs::unlink_at(directory, name).map_err(|e| e.to_string())?;
    }
    if names.contains("screen") {
        fs::rmdir_at(directory, "screen").map_err(|e| e.to_string())?;
    }
    std::fs::remove_dir(path).map_err(|e| e.to_string())?;
    Ok(())
}

/// Retire exact completed runtimes; returns the live-unit count.
pub fn collect_finished() -> Result<usize, String> {
    let mut active = 0usize;
    for identifier in terminal_directories()? {
        let path = format!("{TERMINALS}/{identifier}");
        let directory = checked_chain(&path)?;
        let record = match binding_record(&directory, None, 0) {
            Ok(record) => record,
            Err(_) => {
                // Missing/corrupt binding with no unit: retire the stub.
                stub_retire(&path, &identifier, &directory)?;
                continue;
            }
        };
        if list_dir_names(&directory)?
            .iter()
            .any(|n| n == "subscription")
        {
            active += 1;
            continue; // broker owns credential capture and retirement
        }
        let account = Account {
            pw_name: record.login.clone(),
            pw_uid: record.uid,
            pw_gid: record.gid,
            pw_dir: record.home.clone(),
            pw_shell: record.shell.clone(),
        };
        let state = svc::service_state(&identifier, Some(&account))?;
        if (state == "inactive" || state == "failed") && svc::cgroup_empty(&identifier)? {
            let permit = read_reservation(&directory)?;
            if !permit_live(permit.as_ref()) {
                remove_owned_files(&path, &directory, &account)?;
            }
        } else {
            active += 1;
        }
    }
    Ok(active)
}
