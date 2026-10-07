//! Fixed role accounts: exact ports of `role_record` and `ensure_role`.
//! Production resolves the system passwd/group databases and creates
//! missing roles via `useradd`; the test seam (active only when the
//! factory root is redirected) maps both roles onto the invoking user
//! with homes under the scratch factory.

use crate::error::{fail, Error};
use crate::fsx;
use crate::validate;
use std::collections::HashSet;
use std::ffi::{CStr, CString};
use std::path::{Path, PathBuf};

pub const NOLOGIN: &str = "/usr/sbin/nologin";
const SYSTEM_KEYDIR: &str = "/etc/ssh/authorized_keys";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub name: String,
    pub dir: PathBuf,
    pub uid: u32,
    pub gid: u32,
    pub shell: String,
}

/// Exact `useradd` recipe, kept as a pure function so tests pin the argv.
pub fn useradd_argv(login: &str) -> Vec<String> {
    [
        "useradd",
        "--create-home",
        "--shell",
        NOLOGIN,
        "--password",
        "!",
        login,
    ]
    .iter()
    .map(|word| word.to_string())
    .collect()
}

/// Inherit-stdio checked execution (`subprocess.run(..., check=True)`).
pub fn run_check(argv: &[String]) -> Result<(), Error> {
    let (head, rest) = argv
        .split_first()
        .ok_or_else(|| Error::io_msg("empty argv"))?;
    let status = std::process::Command::new(head)
        .args(rest)
        .status()
        .map_err(|err| Error::io("spawn", &err))?;
    if status.success() {
        Ok(())
    } else {
        Err(Error::io_msg(format!("command failed: {head}")))
    }
}

fn bytes_to_string(bytes: &[u8]) -> String {
    String::from_utf8_lossy(bytes).into_owned()
}

/// Reentrant `getpwnam`; missing accounts yield `None` like the `.py`
/// `KeyError` arm. `ERANGE` retries with a larger buffer.
fn system_account(login: &str) -> Result<Option<Account>, Error> {
    let name = CString::new(login).map_err(|_| Error::fail("unsupported factory role"))?;
    let mut size: usize = 16384;
    loop {
        let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
        let mut buf = vec![0u8; size];
        let mut result: *mut libc::passwd = std::ptr::null_mut();
        let rc = unsafe {
            libc::getpwnam_r(
                name.as_ptr(),
                &mut pwd,
                buf.as_mut_ptr() as *mut libc::c_char,
                buf.len(),
                &mut result,
            )
        };
        if result.is_null() {
            if rc == libc::ERANGE && size < 1_048_576 {
                size *= 2;
                continue;
            }
            return Ok(None);
        }
        let account = unsafe {
            Account {
                name: bytes_to_string(CStr::from_ptr(pwd.pw_name).to_bytes()),
                dir: PathBuf::from(bytes_to_string(CStr::from_ptr(pwd.pw_dir).to_bytes())),
                uid: pwd.pw_uid,
                gid: pwd.pw_gid,
                shell: bytes_to_string(CStr::from_ptr(pwd.pw_shell).to_bytes()),
            }
        };
        return Ok(Some(account));
    }
}

/// Every group carrying `login` as a member (`grp.getgrall` scan).
fn member_groups(login: &str) -> Result<Vec<String>, Error> {
    let mut names = Vec::new();
    unsafe {
        libc::setgrent();
        loop {
            let entry = libc::getgrent();
            if entry.is_null() {
                break;
            }
            let mut mem = (*entry).gr_mem;
            let mut member = false;
            while !mem.is_null() && !(*mem).is_null() {
                if CStr::from_ptr(*mem).to_bytes() == login.as_bytes() {
                    member = true;
                    break;
                }
                mem = mem.add(1);
            }
            if member {
                names.push(bytes_to_string(CStr::from_ptr((*entry).gr_name).to_bytes()));
            }
        }
        libc::endgrent();
    }
    Ok(names)
}

fn primary_group(gid: u32) -> Result<String, Error> {
    let mut size: usize = 16384;
    loop {
        let mut grp: libc::group = unsafe { std::mem::zeroed() };
        let mut buf = vec![0u8; size];
        let mut result: *mut libc::group = std::ptr::null_mut();
        let rc = unsafe {
            libc::getgrgid_r(
                gid,
                &mut grp,
                buf.as_mut_ptr() as *mut libc::c_char,
                buf.len(),
                &mut result,
            )
        };
        if result.is_null() {
            if rc == libc::ERANGE && size < 1_048_576 {
                size *= 2;
                continue;
            }
            return Err(Error::io_msg("primary group unknown"));
        }
        return Ok(unsafe { bytes_to_string(CStr::from_ptr((*result).gr_name).to_bytes()) });
    }
}

/// Test-seam account: both roles map onto the invoking user with scratch
/// homes. An optional `test-accounts/<login>.json` `{"shell": ...}` file
/// overrides the shell so tests can plant interactive accounts.
fn test_account(ctx: &crate::Ctx, login: &str) -> Result<Option<Account>, Error> {
    if !validate::is_role(login) {
        return Ok(None);
    }
    let mut shell = NOLOGIN.to_string();
    let overlay = ctx
        .factory
        .join("test-accounts")
        .join(format!("{login}.json"));
    match std::fs::read(&overlay) {
        Ok(raw) => {
            let text =
                std::str::from_utf8(&raw).map_err(|_| Error::fail("unsupported test account"))?;
            let value = crate::state_json::StateValue::parse(text)
                .map_err(|_| Error::fail("unsupported test account"))?;
            match value.get("shell") {
                None => {}
                Some(shell_value) => match shell_value.as_str() {
                    Some(text) => shell = text.to_string(),
                    None => return Err(Error::fail("unsupported test account")),
                },
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {}
        Err(err) => return Err(Error::classify(err)),
    }
    Ok(Some(Account {
        name: login.to_string(),
        dir: ctx.factory.join("test-homes").join(login),
        uid: unsafe { libc::geteuid() },
        gid: unsafe { libc::getegid() },
        shell,
    }))
}

pub fn role_record(ctx: &crate::Ctx, login: &str) -> Result<Option<Account>, Error> {
    if ctx.test.is_some() {
        return test_account(ctx, login);
    }
    system_account(login)
}

fn check_home(account: &Account, login: &str) -> Result<(), Error> {
    let meta = std::fs::symlink_metadata(&account.dir).map_err(Error::classify)?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_dir() || meta.uid() != account.uid || meta.mode() & 0o077 != 0 {
        return fail(format!("unsafe factory home: {login}"));
    }
    Ok(())
}

fn ensure_checkouts(account: &Account, login: &str) -> Result<(), Error> {
    let checkouts = account.dir.join("checkouts");
    let is_link = std::fs::symlink_metadata(&checkouts)
        .map(|meta| meta.file_type().is_symlink())
        .unwrap_or(false);
    if is_link {
        return fail(format!("unsafe factory checkout root: {login}"));
    }
    fsx::mkdir_p(&checkouts, 0o755)?;
    fsx::chown(&checkouts, account.uid, account.gid)?;
    Ok(())
}

fn ensure_creds(ctx: &crate::Ctx, login: &str) -> Result<(), Error> {
    let creds = ctx.credentials.join(login);
    fsx::mkdir_p(&creds, 0o755)?;
    fsx::owned_dir(ctx, &creds, 0o755)?;
    Ok(())
}

fn key_path(ctx: &crate::Ctx, login: &str) -> PathBuf {
    if ctx.test.is_some() {
        ctx.factory.join("test-ssh-keys").join(login)
    } else {
        Path::new(SYSTEM_KEYDIR).join(login)
    }
}

fn refuse_keyfile(ctx: &crate::Ctx, login: &str) -> Result<(), Error> {
    if fsx::lexists(&key_path(ctx, login)) {
        return fail(format!(
            "factory account must not hold external SSH keys: {login}"
        ));
    }
    Ok(())
}

fn ensure_role_production(ctx: &crate::Ctx, login: &str) -> Result<Account, Error> {
    let mut account = role_record(ctx, login)?;
    if account.is_none() {
        run_check(&useradd_argv(login))?;
        account = role_record(ctx, login)?;
        let created = account.ok_or_else(|| Error::io_msg("useradd left no account"))?;
        fsx::chmod(&created.dir, 0o700)?;
        account = Some(created);
    }
    let account = account.expect("account resolved");
    if account.shell != NOLOGIN {
        return fail(format!(
            "existing factory account has an interactive shell: {login}"
        ));
    }
    let mut groups = member_groups(login)?;
    groups.push(primary_group(account.gid)?);
    let unique: HashSet<&str> = groups.iter().map(|name| name.as_str()).collect();
    if unique.len() != 1 || !unique.contains(login) {
        return fail(format!(
            "existing factory account has unexpected groups: {login}"
        ));
    }
    refuse_keyfile(ctx, login)?;
    check_home(&account, login)?;
    ensure_checkouts(&account, login)?;
    ensure_creds(ctx, login)?;
    Ok(account)
}

fn ensure_role_test(ctx: &crate::Ctx, login: &str) -> Result<Account, Error> {
    let account = test_account(ctx, login)?.expect("test role resolves");
    if std::fs::symlink_metadata(&account.dir).map(|_| ()).is_err() {
        fsx::mkdir_p(&account.dir, 0o700)?;
    }
    if account.shell != NOLOGIN {
        return fail(format!(
            "existing factory account has an interactive shell: {login}"
        ));
    }
    // The single-user test mapping carries no supplementary groups.
    refuse_keyfile(ctx, login)?;
    check_home(&account, login)?;
    ensure_checkouts(&account, login)?;
    ensure_creds(ctx, login)?;
    Ok(account)
}

/// Provision (or re-verify) one fixed role account.
pub fn ensure_role(ctx: &crate::Ctx, login: &str) -> Result<Account, Error> {
    validate::check_role_str(login)?;
    if ctx.test.is_some() {
        return ensure_role_test(ctx, login);
    }
    ensure_role_production(ctx, login)
}

#[cfg(test)]
#[path = "accounts_tests.rs"]
mod tests;
