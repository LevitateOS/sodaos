use std::fs::File;
use std::io::Read;
use std::os::unix::io::{AsRawFd, FromRawFd};

pub struct Pty {
    pub master: File,
    pub slave_path: String,
}

pub fn open_pty() -> Pty {
    let mut master = 0;
    let mut slave = 0;
    assert_eq!(
        unsafe {
            libc::openpty(
                &mut master,
                &mut slave,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
                std::ptr::null_mut(),
            )
        },
        0
    );
    // ptsname_r: ptsname's static buffer races across test threads.
    let mut name = [0 as libc::c_char; 64];
    assert_eq!(
        unsafe { libc::ptsname_r(master, name.as_mut_ptr(), name.len()) },
        0
    );
    let slave_path = unsafe {
        std::ffi::CStr::from_ptr(name.as_ptr())
            .to_string_lossy()
            .into_owned()
    };
    unsafe { libc::close(slave) };
    Pty {
        master: unsafe { File::from_raw_fd(master) },
        slave_path,
    }
}

fn poll_read(master: &mut File, timeout_ms: i32) -> Option<Vec<u8>> {
    let fd = master.as_raw_fd();
    let mut fdset: libc::pollfd = libc::pollfd {
        fd,
        events: libc::POLLIN,
        revents: 0,
    };
    let ready = unsafe { libc::poll(&mut fdset, 1, timeout_ms) };
    if ready > 0 && fdset.revents & libc::POLLIN != 0 {
        let mut chunk = [0u8; 4096];
        match master.read(&mut chunk) {
            Ok(0) => None,
            Ok(n) => Some(chunk[..n].to_vec()),
            Err(_) => None,
        }
    } else {
        None
    }
}

/// Drain until `needle` appears (panics on timeout), returning everything
/// received. Synchronizes with the console before sending input.
/// Wall-clock deadlines: an unopened slave reports instant HUP, which
/// must not spin the wait budget.
pub fn read_until(master: &mut File, needle: &[u8], timeout_ms: u64) -> Vec<u8> {
    let start = std::time::Instant::now();
    let budget = std::time::Duration::from_millis(timeout_ms.max(100));
    let mut output = Vec::new();
    while !output.windows(needle.len()).any(|w| w == needle) {
        match poll_read(master, 50) {
            Some(chunk) => output.extend(chunk),
            None => {
                assert!(
                    start.elapsed() < budget,
                    "timed out waiting for console output"
                );
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
    }
    output
}

/// Drain until the master idles, returning everything received.
pub fn drain_idle(master: &mut File, idle_ms: u64) -> Vec<u8> {
    let budget = std::time::Duration::from_millis(idle_ms.max(100));
    let mut output = Vec::new();
    let mut quiet_since = std::time::Instant::now();
    loop {
        match poll_read(master, 50) {
            Some(chunk) => {
                output.extend(chunk);
                quiet_since = std::time::Instant::now();
            }
            None => {
                if quiet_since.elapsed() >= budget {
                    break;
                }
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
    }
    output
}
