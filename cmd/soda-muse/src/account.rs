use super::launch_json::serialize_go;
use super::paths::path_error;
use std::ffi::{CStr, CString};
use std::fs;
use std::io;
use std::os::unix::fs::MetadataExt;

pub(crate) fn account_for(actor: &str) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project root required"));
    }
    let parsed: i64 = actor
        .parse()
        .map_err(|_| String::from("invalid account identity"))?;
    if parsed <= 0 {
        return Err(String::from("invalid account identity"));
    }
    let directory = "/var/lib/soda/accounts";
    for path in ["/var", "/var/lib", "/var/lib/soda", directory] {
        account_node(path, true)?;
    }
    let mut names: Vec<String> = Vec::new();
    let entries = fs::read_dir(directory).map_err(|e| path_error("open", directory, e))?;
    for entry in entries {
        let entry = entry.map_err(|e| path_error("open", directory, e))?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    for name in &names {
        if account_entry(directory, name, actor)? {
            return Ok(());
        }
    }
    Err(String::from("provisioned account missing"))
}

#[derive(serde::Serialize)]
struct AccountRecord<'a> {
    #[serde(rename = "Uid")]
    uid: String,
    #[serde(rename = "Gid")]
    gid: String,
    #[serde(rename = "Username")]
    username: &'a str,
    #[serde(rename = "Name")]
    name: &'a str,
    #[serde(rename = "HomeDir")]
    home_dir: &'a str,
}

fn account_entry(directory: &str, name: &str, actor: &str) -> Result<bool, String> {
    let marker = format!("{directory}/{name}");
    if account_node(&marker, false).is_err() {
        return Ok(false);
    }
    let body = fs::read(&marker).map_err(|e| path_error("open", &marker, e))?;
    if String::from_utf8_lossy(&body).trim() != actor {
        return Ok(false);
    }
    let (uid, gid, gecos, dir) = lookup_user(name)?;
    let out = format!(
        "{}\n",
        serialize_go(&AccountRecord {
            uid: uid.to_string(),
            gid: gid.to_string(),
            username: name,
            name: &gecos,
            home_dir: &dir,
        })
    );
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(true)
}

pub(crate) fn account_node(path: &str, directory: bool) -> Result<(), String> {
    let info = fs::symlink_metadata(path).map_err(|e| path_error("lstat", path, e))?;
    if info.uid() != 0 || info.gid() != 0 || info.mode() & 0o022 != 0 {
        return Err(String::from("unsafe account record"));
    }
    if directory {
        if !info.is_dir() {
            return Err(String::from("account ancestor is not a directory"));
        }
        return Ok(());
    }
    if !info.is_file() || info.mode() & 0o777 != 0o600 || info.len() > 64 {
        return Err(String::from("unsafe account marker"));
    }
    Ok(())
}

fn lookup_user(name: &str) -> Result<(u32, u32, String, String), String> {
    let cname = CString::new(name).map_err(|_| format!("user: unknown user {name}"))?;
    let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buf = [0u8; 16384];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let rc = unsafe {
        libc::getpwnam_r(
            cname.as_ptr(),
            &mut pwd,
            buf.as_mut_ptr() as *mut libc::c_char,
            buf.len(),
            &mut result,
        )
    };
    if rc != 0 || result.is_null() {
        return Err(format!("user: unknown user {name}"));
    }
    let gecos = unsafe { CStr::from_ptr(pwd.pw_gecos) }
        .to_string_lossy()
        .into_owned();
    let dir = unsafe { CStr::from_ptr(pwd.pw_dir) }
        .to_string_lossy()
        .into_owned();
    Ok((pwd.pw_uid, pwd.pw_gid, gecos, dir))
}
