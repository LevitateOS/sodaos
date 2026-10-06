pub const TEST_KEY: &str =
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV";
pub const TEST_KEY_2: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCoDZAt7ewKwwXBa7tCvC9/+p//wMupqSjGVnTOvoYOUt1UNbromMK5hBZGq2xIlqJQ0rRZoTEtsE7w5BUgkNzvsioTnnjD48UeqMvJhBfSrJYTVMZeM/ttm1jmlNhbjs6nt98R/KmdsyK+2a+z5BQ+KRlr4kbKfd/UDOPj8XuA2/vW4K2301UUdDk9Jh2r/bcjRnrIyHUX1Rmga608tAWRZtJQRo+8/28JqnjQM5s4qcu1d1N2Y823P4YGaLYhRLoKhV1/gCMRRhD9ZTZpn58sVmiEGQ90YeHE/8vETm+Q2IkjZvx2vobzhvdsA3LGs3B1EN2kdbCGJRqaltrJLK+t";
pub const TEST_KEY_3: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBLtQL7Kq7o6tng5YEyRN3ICkkd2BErzxr+p3zkns1Apc0BJ7BDcTZqzWuusUsWLZxRtnOVtz2FT2vd0GlCz20RI=";

/// Serializes tests that mutate process environment, which Rust runs
/// in parallel threads unlike Go's sequential test binary.
pub static ENV_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

pub struct TempDir {
    pub path: String,
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.path);
    }
}

/// Go `t.TempDir`: fresh 0700 directory, removed on drop.
pub fn temp_dir() -> TempDir {
    let template = std::ffi::CString::new("/tmp/soda-enroll-test-XXXXXX").unwrap();
    let raw = template.into_raw();
    let ok = unsafe { libc::mkdtemp(raw) };
    let template = unsafe { std::ffi::CString::from_raw(raw) };
    assert!(!ok.is_null());
    TempDir {
        path: template.to_string_lossy().into_owned(),
    }
}

pub fn test_uid() -> u32 {
    unsafe { libc::geteuid() }
}

pub struct TestPty {
    pub master: std::fs::File,
    pub slave_path: String,
}

pub fn open_test_pty() -> TestPty {
    use std::os::unix::io::FromRawFd;
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
    TestPty {
        master: unsafe { std::fs::File::from_raw_fd(master) },
        slave_path,
    }
}

/// Drain whatever the console already wrote, stopping after 300 ms idle.
pub fn drain_available(master: &mut std::fs::File) -> Vec<u8> {
    use std::io::Read;
    use std::os::unix::io::AsRawFd;
    let mut transcript = Vec::new();
    loop {
        let mut fd = libc::pollfd {
            fd: master.as_raw_fd(),
            events: libc::POLLIN,
            revents: 0,
        };
        if unsafe { libc::poll(&mut fd, 1, 300) } <= 0 || fd.revents & libc::POLLIN == 0 {
            break;
        }
        let mut buffer = [0u8; 4096];
        match master.read(&mut buffer) {
            Ok(0) => break,
            Ok(n) => transcript.extend_from_slice(&buffer[..n]),
            Err(_) => break,
        }
    }
    transcript
}
