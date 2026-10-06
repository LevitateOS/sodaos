use std::ffi::CString;
use std::fs;
use std::io::Write as _;
use std::os::unix::io::RawFd;
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

use super::archive::emit_archive;
use super::interface::FdGuard;
use super::stage::INSTALL_SCRIPT;
pub(crate) use super::MUSE_VERSION;

static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(crate) struct TestDir(pub(crate) std::path::PathBuf);

impl TestDir {
    pub(crate) fn make(tag: &str) -> TestDir {
        let id = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
        let dir =
            std::env::temp_dir().join(format!("smm-test-{}-{}-{}", std::process::id(), id, tag));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        TestDir(dir)
    }

    pub(crate) fn path(&self, name: &str) -> String {
        self.0.join(name).to_string_lossy().into_owned()
    }

    pub(crate) fn dir_str(&self) -> String {
        self.0.to_string_lossy().into_owned()
    }
}

impl Drop for TestDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

pub(crate) fn is_root() -> bool {
    unsafe { libc::geteuid() == 0 }
}

pub(crate) const PROJECT: &str = "p0123456789abcdef01234567";

pub(crate) fn hex_string(c: char, len: usize) -> String {
    std::iter::repeat_n(c, len).collect()
}

pub(crate) fn synthetic_feeds(tools_dir: &TestDir) -> (Vec<(String, RawFd, u64)>, Vec<FdGuard>) {
    let mut feeds = Vec::new();
    let mut guards = Vec::new();
    for name in ["muse", "soda-identity-compose", "muse-native"] {
        let p = tools_dir.path(name);
        fs::write(&p, format!("synthetic {name}")).unwrap();
        let c = CString::new(p).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let len = format!("synthetic {name}").len() as u64;
        guards.push(FdGuard(fd));
        feeds.push((name.to_string(), fd, len));
    }
    (feeds, guards)
}

pub(crate) fn emit_synthetic() -> Vec<u8> {
    let tools_dir = TestDir::make("emit");
    let (feeds, guards) = synthetic_feeds(&tools_dir);
    let mut archive = Vec::new();
    emit_archive(
        &mut |b: &[u8]| {
            archive.extend_from_slice(b);
            Ok(())
        },
        &feeds,
    )
    .unwrap();
    drop(guards);
    archive
}

pub(crate) fn run_install_script(archive: &[u8], targets: &[String]) -> (bool, String) {
    let mut cmd = Command::new("/bin/sh");
    cmd.args(["-ceu", INSTALL_SCRIPT, "soda-muse-maintain"]);
    for t in targets {
        cmd.arg(t);
    }
    cmd.stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    let mut child = cmd.spawn().unwrap();
    child.stdin.take().unwrap().write_all(archive).unwrap();
    let out = child.wait_with_output().unwrap();
    (
        out.status.success(),
        String::from_utf8_lossy(&out.stderr).into_owned(),
    )
}
