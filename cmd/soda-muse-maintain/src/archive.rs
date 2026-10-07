use std::fs;
use std::io::{self, Read, Write};
use std::os::unix::io::RawFd;
use std::time::{Duration, Instant};

use tar::{Builder, EntryType, Header};

// feed_archive streams the tar byte sequence to podman's stdin. Writes
// poll non-blocking against the deadline so an abandoned pipe ends with
// the maintenance failure instead of hanging past it; a dead reader
// surfaces Go's closed-pipe error.
pub(crate) fn feed_archive(
    writer: fs::File,
    feeds: &[(String, RawFd, u64)],
    deadline: Instant,
) -> Result<(), String> {
    use std::os::unix::io::AsRawFd;
    let wfd = writer.as_raw_fd();
    let flags = unsafe { libc::fcntl(wfd, libc::F_GETFL) };
    if flags >= 0 {
        unsafe {
            libc::fcntl(wfd, libc::F_SETFL, flags | libc::O_NONBLOCK);
        }
    }
    let mut emit = |buf: &[u8]| -> Result<(), String> {
        let mut rest = buf;
        while !rest.is_empty() {
            let n = unsafe { libc::write(wfd, rest.as_ptr() as *const libc::c_void, rest.len()) };
            if n > 0 {
                rest = &rest[n as usize..];
                continue;
            }
            if n == 0 {
                continue;
            }
            let no = super::filesystem::last_errno();
            if no == libc::EINTR {
                continue;
            }
            if no == libc::EAGAIN {
                if Instant::now() >= deadline {
                    return Err(String::from("project maintenance command failed"));
                }
                std::thread::sleep(Duration::from_millis(10));
                continue;
            }
            return Err(String::from("io: read/write on closed pipe"));
        }
        Ok(())
    };
    emit_archive(&mut emit, feeds)
}

struct ExactPread {
    fd: RawFd,
    name: String,
    offset: i64,
    remaining: u64,
}

impl Read for ExactPread {
    fn read(&mut self, buffer: &mut [u8]) -> io::Result<usize> {
        if self.remaining == 0 || buffer.is_empty() {
            return Ok(0);
        }
        let length = self.remaining.min(buffer.len() as u64) as usize;
        let n = unsafe {
            libc::pread(
                self.fd,
                buffer.as_mut_ptr() as *mut libc::c_void,
                length,
                self.offset,
            )
        };
        if n < 0 {
            return Err(io::Error::other(format!(
                "read {}: {}",
                self.name,
                super::filesystem::go_errno(super::filesystem::last_errno())
            )));
        }
        if n == 0 {
            return Err(io::Error::new(io::ErrorKind::UnexpectedEof, "EOF"));
        }
        self.offset += n as i64;
        self.remaining -= n as u64;
        Ok(n as usize)
    }
}

struct EmitWriter<'a> {
    emit: &'a mut dyn FnMut(&[u8]) -> Result<(), String>,
    error: Option<String>,
}

impl Write for EmitWriter<'_> {
    fn write(&mut self, buffer: &[u8]) -> io::Result<usize> {
        if let Some(error) = &self.error {
            return Err(io::Error::other(error.clone()));
        }
        match (self.emit)(buffer) {
            Ok(()) => Ok(buffer.len()),
            Err(error) => {
                self.error = Some(error.clone());
                Err(io::Error::other(error))
            }
        }
    }

    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

fn archive_error(error: io::Error, emit_error: Option<String>) -> String {
    emit_error.unwrap_or_else(|| error.to_string())
}

fn tool_header(name: &str, size: u64) -> Result<Header, String> {
    if name.len() > 100 || name.contains('\0') {
        return Err(String::from("public tool name exceeds archive limit"));
    }
    // USTAR size is an 11-digit octal field. Header::set_size panics when
    // the value cannot be represented, so reject it before calling upstream.
    if size >= 8u64.pow(11) {
        return Err(String::from("public tool size exceeds archive limit"));
    }
    let mut header = Header::new_ustar();
    header
        .set_path(name)
        .map_err(|error| format!("invalid public tool name: {error}"))?;
    header.set_mode(0o755);
    header.set_uid(0);
    header.set_gid(0);
    header.set_size(size);
    header.set_mtime(0);
    header.set_entry_type(EntryType::Regular);
    header
        .set_link_name_literal(b"")
        .map_err(|error| error.to_string())?;
    header.set_username("").map_err(|error| error.to_string())?;
    header
        .set_groupname("")
        .map_err(|error| error.to_string())?;
    header
        .set_device_major(0)
        .map_err(|error| error.to_string())?;
    header
        .set_device_minor(0)
        .map_err(|error| error.to_string())?;
    header.set_cksum();
    Ok(header)
}

// emit_archive uses tar's USTAR writer while retaining the maintenance pipe
// writer and exact-size FD reads. The bounded reader makes a shrinking tool an
// error rather than letting Builder::append_data write a short entry.
pub(crate) fn emit_archive(
    emit: &mut dyn FnMut(&[u8]) -> Result<(), String>,
    feeds: &[(String, RawFd, u64)],
) -> Result<(), String> {
    let mut archive = Builder::new(EmitWriter { emit, error: None });
    for (name, fd, size) in feeds {
        let mut header = tool_header(name, *size)?;
        let reader = ExactPread {
            fd: *fd,
            name: name.clone(),
            offset: 0,
            remaining: *size,
        };
        if let Err(error) = archive.append_data(&mut header, name, reader) {
            let error = archive_error(error, archive.get_mut().error.clone());
            // Builder finishes on Drop. A failed source must not gain zero
            // trailer bytes that could make its truncated body appear valid.
            archive.get_mut().error = Some(error.clone());
            return Err(error);
        }
    }
    if let Err(error) = archive.finish() {
        let emit_error = archive.get_mut().error.clone();
        return Err(archive_error(error, emit_error));
    }
    Ok(())
}

#[cfg(test)]
#[path = "archive_tests.rs"]
mod archive_tests;
