//! User scoped personal Git key preparation and unlock payload.

use crate::error::Error;
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

const BASE: &str = ".ssh/u08-personal-git";
const PASSPHRASE_FILE: &str = "temporary-passphrase";
const ASKPASS_FILE: &str = "temporary-askpass";

fn io(error: std::io::Error) -> Error {
    Error::Io(error)
}

fn private_write(path: &Path, data: &[u8]) -> Result<(), Error> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(io)?;
    file.write_all(data).map_err(io)?;
    file.sync_all().map_err(io)
}

fn private_replace(path: &Path, data: &[u8], mode: u32) -> Result<(), Error> {
    let mut file = OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(mode)
        .open(path)
        .map_err(io)?;
    file.set_permissions(fs::Permissions::from_mode(mode))
        .map_err(io)?;
    file.write_all(data).map_err(io)?;
    file.sync_all().map_err(io)
}

fn run_quiet(
    program: &str,
    args: &[&str],
    env: &[(&str, &str)],
) -> Result<std::process::Output, Error> {
    let mut command = Command::new(program);
    command
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    for (key, value) in env {
        command.env(key, value);
    }
    command.output().map_err(io)
}

fn agent_pid(output: &[u8]) -> Option<String> {
    let text = std::str::from_utf8(output).ok()?;
    let rest = text.split_once("SSH_AGENT_PID=")?.1;
    let digits = rest
        .chars()
        .take_while(|character| character.is_ascii_digit())
        .collect::<String>();
    if digits.is_empty() {
        None
    } else {
        Some(digits)
    }
}

#[cfg(test)]
mod tests {
    use super::agent_pid;

    #[test]
    fn parses_only_numeric_agent_pid() {
        assert_eq!(
            agent_pid(b"SSH_AGENT_PID=4242; export SSH_AGENT_PID;"),
            Some("4242".into())
        );
        assert_eq!(agent_pid(b"agent started"), None);
        assert_eq!(agent_pid(b"SSH_AGENT_PID=x;"), None);
    }
}

fn write_git_ssh() -> Result<(), Error> {
    let script = b"#!/bin/sh\nbase=\"$HOME/.ssh/u08-personal-git\"\nexport SSH_AUTH_SOCK=\"$base/agent\"\nexec /usr/bin/ssh -F /dev/null -o BatchMode=yes -o ForwardAgent=no -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=\"$base/known_hosts\" -i \"$base/identity\" \"$@\"\n";
    let path = std::env::var_os("HOME")
        .map(PathBuf::from)
        .ok_or_else(|| Error::msg("Git key/agent operation failed"))?
        .join(BASE)
        .join("git-ssh");
    private_replace(&path, script, 0o700)
}

/// Read the passphrase from SSH stdin, then prepare or unlock this user’s key.
/// The caller’s shell stages the binary prefix, leaving exactly 43 secret
/// bytes on stdin; no passphrase enters command arguments or output.
pub fn run(phase: &str) -> Result<Vec<u8>, Error> {
    if phase != "prepare" && phase != "unlock" {
        return Err(Error::msg("unknown personal-Git phase"));
    }
    unsafe { libc::umask(0o077) };
    let mut passphrase = [0u8; 43];
    std::io::stdin()
        .read_exact(&mut passphrase)
        .map_err(|_| Error::msg("invalid personal-Git passphrase input"))?;
    if !passphrase
        .iter()
        .all(|byte| byte.is_ascii_alphanumeric() || *byte == b'_' || *byte == b'-')
    {
        return Err(Error::msg("invalid personal-Git passphrase input"));
    }
    let home = std::env::var_os("HOME")
        .map(PathBuf::from)
        .ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
    let ssh_dir = home.join(".ssh");
    fs::create_dir_all(&ssh_dir).map_err(io)?;
    fs::set_permissions(&ssh_dir, fs::Permissions::from_mode(0o700)).map_err(io)?;
    let base = home.join(BASE);
    let identity = base.join("identity");
    if phase == "prepare" {
        fs::create_dir(&base).map_err(io)?;
        fs::set_permissions(&base, fs::Permissions::from_mode(0o700)).map_err(io)?;
    } else if !base.is_dir() || !identity.is_file() {
        return Err(Error::msg("Git key/agent operation failed"));
    }

    let password_path = base.join(PASSPHRASE_FILE);
    let ask_path = base.join(ASKPASS_FILE);
    private_write(&password_path, &passphrase)?;
    let ask_script =
        b"#!/bin/sh\nexec /usr/bin/head -c 128 \"$SODA_PERSONAL_GIT_PASSPHRASE_FILE\"\n";
    private_write(&ask_path, ask_script)?;
    fs::set_permissions(&ask_path, fs::Permissions::from_mode(0o700)).map_err(io)?;
    let password_text = password_path
        .to_str()
        .ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
    let ask_text = ask_path
        .to_str()
        .ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
    let ask_env = [
        ("SSH_ASKPASS", ask_text),
        ("SSH_ASKPASS_REQUIRE", "force"),
        ("DISPLAY", "soda-u08"),
        ("SODA_PERSONAL_GIT_PASSPHRASE_FILE", password_text),
    ];
    if phase == "prepare" {
        let output = run_quiet(
            "ssh-keygen",
            &[
                "-q",
                "-t",
                "ed25519",
                "-f",
                identity
                    .to_str()
                    .ok_or_else(|| Error::msg("Git key/agent operation failed"))?,
                "-C",
                "U08 personal project Git",
            ],
            &ask_env,
        )?;
        if !output.status.success() {
            return Err(Error::msg("Git key/agent operation failed"));
        }
    }

    let socket = base.join("agent");
    if socket.exists() {
        let socket_text = socket
            .to_str()
            .ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
        let probe = run_quiet("ssh-add", &["-l"], &[("SSH_AUTH_SOCK", socket_text)])?;
        if probe.status.code() != Some(2) {
            return Err(Error::msg("Agent already live; no duplicate start"));
        }
        fs::remove_file(&socket).map_err(io)?;
    }
    let socket_text = socket
        .to_str()
        .ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
    let agent = Command::new("ssh-agent")
        .args(["-a", socket_text, "-s"])
        .stdin(Stdio::null())
        .stderr(Stdio::null())
        .output()
        .map_err(io)?;
    if !agent.status.success() {
        return Err(Error::msg("Git key/agent operation failed"));
    }
    let pid =
        agent_pid(&agent.stdout).ok_or_else(|| Error::msg("Git key/agent operation failed"))?;
    private_replace(
        &base.join("agent.pid"),
        format!("{pid}\n").as_bytes(),
        0o600,
    )?;
    let loaded = run_quiet(
        "ssh-add",
        &[identity
            .to_str()
            .ok_or_else(|| Error::msg("Git key/agent operation failed"))?],
        &[
            ("SSH_AUTH_SOCK", socket_text),
            ask_env[0],
            ask_env[1],
            ask_env[2],
            ask_env[3],
        ],
    )?;
    if !loaded.status.success() {
        return Err(Error::msg("Git key/agent operation failed"));
    }
    fs::remove_file(&password_path).map_err(io)?;
    fs::remove_file(&ask_path).map_err(io)?;
    write_git_ssh()?;
    fs::read(base.join("identity.pub")).map_err(io)
}
