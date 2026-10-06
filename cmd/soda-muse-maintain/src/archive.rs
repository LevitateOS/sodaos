use std::fs;
use std::os::unix::io::RawFd;
use std::time::{Duration, Instant};

use super::filesystem::{go_errno, last_errno};

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
            let no = last_errno();
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

// emit_archive ports archiveTools: one USTAR header per tool, CopyN of
// exactly size bytes, 512 padding, and the two zero trailer blocks.
// A short file ends CopyN with raw io.EOF; read failures keep the
// PathError shape with the tool's archive name.
pub(crate) fn emit_archive(
    emit: &mut dyn FnMut(&[u8]) -> Result<(), String>,
    feeds: &[(String, RawFd, u64)],
) -> Result<(), String> {
    for (name, fd, size) in feeds {
        emit(&tar_header(name, *size)?)?;
        let mut remaining = *size;
        let mut offset: i64 = 0;
        let mut chunk = [0u8; 65536];
        while remaining > 0 {
            let want = remaining.min(chunk.len() as u64) as usize;
            let n =
                unsafe { libc::pread(*fd, chunk.as_mut_ptr() as *mut libc::c_void, want, offset) };
            if n < 0 {
                return Err(format!("read {name}: {}", go_errno(last_errno())));
            }
            if n == 0 {
                return Err(String::from("EOF"));
            }
            emit(&chunk[..n as usize])?;
            offset += n as i64;
            remaining -= n as u64;
        }
        let pad = (512 - (size % 512)) % 512;
        if pad > 0 {
            let zeros = vec![0u8; pad as usize];
            emit(&zeros)?;
        }
    }
    emit(&[0u8; 1024])?;
    Ok(())
}

// tar_header writes the USTAR byte stream Go's archive/tar emits for
// these short regular names: fixed headers, 512-block data, two zero
// blocks. Names are fixed constants, so the length guard never fires.
pub(crate) fn tar_header(name: &str, size: u64) -> Result<[u8; 512], String> {
    let mut header = [0u8; 512];
    if name.len() > 100 || name.contains('\0') {
        return Err(String::from("public tool name exceeds archive limit"));
    }
    header[..name.len()].copy_from_slice(name.as_bytes());
    // Mode 0755, uid/gid 0, size octal, mtime 0, regular file, USTAR.
    header[100..108].copy_from_slice(format!("{:07o}\0", 0o755).as_bytes());
    header[108..116].copy_from_slice(b"0000000\0");
    header[116..124].copy_from_slice(b"0000000\0");
    header[124..136].copy_from_slice(format!("{:011o}\0", size).as_bytes());
    header[136..148].copy_from_slice(b"00000000000\0");
    header[156] = b'0';
    header[329..337].copy_from_slice(b"0000000\0");
    header[337..345].copy_from_slice(b"0000000\0");
    header[257..263].copy_from_slice(b"ustar\0");
    header[263..265].copy_from_slice(b"00");
    // Checksum over spaces, then six octal digits, NUL, space.
    header[148..156].copy_from_slice(b"        ");
    let sum: u32 = header.iter().map(|b| *b as u32).sum();
    header[148..156].copy_from_slice(format!("{sum:06o}\0 ").as_bytes());
    Ok(header)
}
