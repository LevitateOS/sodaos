use std::fs::File;
use std::os::unix::io::AsRawFd;

use crate::fs;
use crate::svc;
use crate::sys;

/// `subscription_cgroup`: the owned service cgroup, root-owned and
/// group/other write-free.
pub fn subscription_cgroup(identifier: &str) -> Result<File, String> {
    let parent = svc::cgroup_parent()?;
    let group =
        sys::open_child_dir(&parent, &svc::unit_name(identifier)).map_err(|e| e.to_string())?;
    drop(parent);
    let (uid, mode) = fs::fstat_uid_mode(&group).map_err(|e| e.to_string())?;
    if uid != 0 || mode & 0o022 != 0 {
        return Err("unsafe cgroup".to_string());
    }
    Ok(group)
}

/// `subscription_kernel_write`: one full write to a cgroup control file.
pub fn subscription_kernel_write(directory: &File, name: &str, value: &[u8]) -> Result<(), String> {
    let fd = sys::open_at(
        directory,
        name,
        sys::OFlags::WRONLY,
        rustix::fs::Mode::empty(),
    )
    .map_err(|e| e.to_string())?;
    let wrote = loop {
        let wrote = unsafe {
            libc::write(
                fd.as_raw_fd(),
                value.as_ptr() as *const libc::c_void,
                value.len(),
            )
        };
        if wrote < 0 {
            let err = std::io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err.to_string());
        }
        break wrote as usize;
    };
    if wrote != value.len() {
        return Err("short cgroup write".to_string());
    }
    Ok(())
}

/// `KEY value` rows of a `cgroup.events` body, with the `.py`
/// `dict(row.split() ...)` shape (anything but a pair raises).
pub fn parse_cgroup_events(content: &[u8]) -> Result<Vec<(String, String)>, String> {
    if !content.is_ascii() {
        return Err("cgroup events encoding".to_string());
    }
    let text = std::str::from_utf8(content).map_err(|_| "cgroup events encoding".to_string())?;
    let body = text.strip_suffix('\n').unwrap_or(text);
    let mut values = Vec::new();
    if !body.is_empty() {
        for row in body.split('\n') {
            let row = row.strip_suffix('\r').unwrap_or(row);
            let parts: Vec<&str> = row.split_whitespace().collect();
            if parts.len() != 2 {
                return Err("cgroup events shape".to_string());
            }
            values.push((parts[0].to_string(), parts[1].to_string()));
        }
    }
    Ok(values)
}

/// `subscription_freeze`: freeze the cgroup and confirm `frozen 1` within 5s.
pub fn subscription_freeze(group: &File) -> Result<(), String> {
    subscription_kernel_write(group, "cgroup.freeze", b"1\n")?;
    let until = sys::monotonic() + 5.0;
    loop {
        if sys::monotonic() >= until {
            return Err("freeze unconfirmed".to_string());
        }
        let fd = sys::open_at(
            group,
            "cgroup.events",
            sys::OFlags::RDONLY,
            rustix::fs::Mode::empty(),
        )
        .map_err(|e| e.to_string())?;
        let content = fs::read_up_to(&fd, 4096).map_err(|e| e.to_string())?;
        drop(fd);
        let values = parse_cgroup_events(&content)?;
        if values.iter().any(|(k, v)| k == "frozen" && v == "1") {
            return Ok(());
        }
        std::thread::sleep(std::time::Duration::from_millis(20));
    }
}
