//! Supervisor/socket helpers (port of `service_state`, `stop_service`,
//! `cgroup_parent`, `cgroup_empty`, `socket_identity`, `tmux_control`).
//!
//! Merge-safe: uses only the brief-pinned `sys`/`fs` signatures plus `libc`.

use std::collections::HashMap;
use std::io;

use crate::account::Account;
pub(crate) use crate::cgroup::{cgroup_empty, cgroup_parent};
use crate::cgroup::{cstring, exit_ok, wait_status_timeout};
pub(crate) use crate::socket::{socket_identity, socket_identity_kinded, SocketCheck};
use crate::sys;

pub fn unit_name(identifier: &str) -> String {
    format!("soda-terminal-{identifier}.service")
}

pub fn service_properties() -> &'static str {
    "LoadState,ActiveState,Description,FragmentPath,DropInPaths,Type,User,KillMode,Restart,SendSIGKILL,TimeoutStopUSec,StandardInput,StandardOutput,StandardError"
}

/// `systemctl show <unit> --property=<props>` argv.
pub fn systemctl_show_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        unit.to_string(),
        format!("--property={}", service_properties()),
    ]
}

/// `systemctl stop <unit>` argv.
pub fn systemctl_stop_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "stop".to_string(),
        unit.to_string(),
    ]
}

/// `systemctl show --value --property=InvocationID <unit>` argv.
pub fn invocation_show_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--value".to_string(),
        "--property=InvocationID".to_string(),
        unit.to_string(),
    ]
}

/// `infocmp <term>` argv.
pub fn infocmp_argv(term: &str) -> Vec<String> {
    vec!["/usr/bin/infocmp".to_string(), term.to_string()]
}

/// `tmux -N -S <sock> <args...>` argv.
pub fn tmux_argv(sock: &str, args: &[String]) -> Vec<String> {
    let mut argv = vec![
        "/usr/bin/tmux".to_string(),
        "-N".to_string(),
        "-S".to_string(),
        sock.to_string(),
    ];
    argv.extend(args.iter().cloned());
    argv
}

/// Parse `KEY=value` lines (last wins, like `dict(...)`); requires the exact
/// property key set and a ≤4096-byte ASCII body.
pub fn parse_service_fields(
    stdout: &[u8],
    properties: &str,
) -> Result<HashMap<String, String>, String> {
    if stdout.len() > 4096 {
        return Err("unit response size".to_string());
    }
    if !stdout.is_ascii() {
        return Err("unit inspection unavailable".to_string());
    }
    let text =
        std::str::from_utf8(stdout).map_err(|_| "unit inspection unavailable".to_string())?;
    // Mirror `splitlines`: drop one trailing newline, then every remaining
    // segment (including blank middles) must be a `KEY=value` line.
    let body = text.strip_suffix('\n').unwrap_or(text);
    let mut fields = HashMap::new();
    if !body.is_empty() {
        for line in body.split('\n') {
            let line = line.strip_suffix('\r').unwrap_or(line);
            let (key, value) = line
                .split_once('=')
                .ok_or_else(|| "unit inspection unavailable".to_string())?;
            fields.insert(key.to_string(), value.to_string());
        }
    }
    let expected: std::collections::HashSet<&str> = properties.split(',').collect();
    let got: std::collections::HashSet<&str> = fields.keys().map(|k| k.as_str()).collect();
    if got != expected {
        return Err("unit inspection unavailable".to_string());
    }
    Ok(fields)
}

/// Loaded-unit ownership and supervision-field checks.
pub fn verify_loaded_unit(
    fields: &HashMap<String, String>,
    identifier: &str,
    account: &Account,
) -> Result<(), String> {
    let unit = unit_name(identifier);
    if fields.get("Description").map(|s| s.as_str())
        != Some(&format!("Soda terminal {identifier}")[..])
        || fields.get("FragmentPath").map(|s| s.as_str())
            != Some(&format!("/run/systemd/transient/{unit}")[..])
    {
        return Err("not the owned terminal unit".to_string());
    }
    let selected = [
        ("DropInPaths", ""),
        ("Type", "exec"),
        ("User", account.pw_name.as_str()),
        ("KillMode", "control-group"),
        ("Restart", "no"),
        ("SendSIGKILL", "yes"),
        ("TimeoutStopUSec", "3s"),
        ("StandardInput", "null"),
        ("StandardOutput", "null"),
        ("StandardError", "null"),
    ];
    for (key, want) in selected {
        if fields.get(key).map(|s| s.as_str()) != Some(want) {
            return Err("terminal supervision changed".to_string());
        }
    }
    Ok(())
}

/// Inspect the transient unit; returns `ActiveState`.
pub fn service_state(identifier: &str, account: Option<&Account>) -> Result<String, String> {
    let unit = unit_name(identifier);
    let output = sys::run_output(&systemctl_show_argv(&unit), 3)
        .map_err(|e| format!("unit inspection: {e}"))?;
    let fields = parse_service_fields(&output.stdout, service_properties())?;
    if !output.status.success() && fields.get("LoadState").map(|s| s.as_str()) != Some("not-found")
    {
        return Err("unit inspection unavailable".to_string());
    }
    if fields.get("LoadState").map(|s| s.as_str()) != Some("not-found") {
        match account {
            None => {
                // Still refuse foreign units before reporting occupation.
                if fields.get("Description").map(|s| s.as_str())
                    != Some(&format!("Soda terminal {identifier}")[..])
                    || fields.get("FragmentPath").map(|s| s.as_str())
                        != Some(&format!("/run/systemd/transient/{unit}")[..])
                {
                    return Err("not the owned terminal unit".to_string());
                }
                return Err("terminal unit occupied".to_string());
            }
            Some(held) => verify_loaded_unit(&fields, identifier, held)?,
        }
    }
    fields
        .get("ActiveState")
        .cloned()
        .ok_or_else(|| "unit inspection unavailable".to_string())
}

/// Stop the unit and confirm native state plus cgroup emptiness.
pub fn stop_service(identifier: &str, account: &Account) -> Result<(), String> {
    let state = service_state(identifier, Some(account))?;
    if state != "inactive" && state != "failed" {
        // `check=False`: a collected unit may vanish after the observation;
        // native state and cgroup emptiness below are the real confirmation.
        let _ = sys::run_output(&systemctl_stop_argv(&unit_name(identifier)), 8);
    }
    let state = service_state(identifier, Some(account))?;
    if (state != "inactive" && state != "failed") || !cgroup_empty(identifier)? {
        return Err("terminal cleanup unconfirmed".to_string());
    }
    Ok(())
}

/// Run a tmux control command as the account (stdio to /dev/null, 2s).
pub fn tmux_control(account: &Account, sock: &str, args: &[String]) -> Result<(), String> {
    let argv = tmux_argv(sock, args);
    let program = cstring(&argv[0]).map_err(|e| e.to_string())?;
    let params: Vec<std::ffi::CString> = argv
        .iter()
        .map(|a| cstring(a))
        .collect::<io::Result<_>>()
        .map_err(|e| e.to_string())?;
    let env: Vec<(std::ffi::CString, std::ffi::CString)> =
        crate::account::user_environment(account)
            .iter()
            .map(|(k, v)| (cstring(k).unwrap(), cstring(v).unwrap()))
            .collect();
    let name = cstring(&account.pw_name).map_err(|e| e.to_string())?;
    let home = cstring(&account.pw_dir).map_err(|e| e.to_string())?;
    let (uid, gid) = (account.pw_uid, account.pw_gid);
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    if pid == 0 {
        unsafe {
            // `preexec_fn` equivalent: become the user, then a fixed env.
            if libc::initgroups(name.as_ptr(), gid) != 0
                || libc::setresgid(gid, gid, gid) != 0
                || libc::setresuid(uid, uid, uid) != 0
                || libc::chdir(home.as_ptr()) != 0
            {
                libc::_exit(127);
            }
            libc::umask(0o022);
            libc::clearenv();
            for (key, value) in &env {
                if libc::setenv(key.as_ptr(), value.as_ptr(), 1) != 0 {
                    libc::_exit(127);
                }
            }
            let null = libc::open(c"/dev/null".as_ptr(), libc::O_RDWR);
            if null < 0 {
                libc::_exit(127);
            }
            libc::dup2(null, 0);
            libc::dup2(null, 1);
            libc::dup2(null, 2);
            if null > 2 {
                libc::close(null);
            }
            let mut pointers: Vec<*const libc::c_char> =
                params.iter().map(|a| a.as_ptr()).collect();
            pointers.push(std::ptr::null());
            libc::execv(program.as_ptr(), pointers.as_ptr());
            libc::_exit(127);
        }
    }
    let status = wait_status_timeout(pid, 2)?;
    if !exit_ok(status) {
        return Err("tmux control failed".to_string());
    }
    Ok(())
}

#[cfg(test)]
#[path = "svc_tests.rs"]
mod svc_tests;
