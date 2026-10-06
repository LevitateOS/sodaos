use crate::options::default_tools;
use crate::sha256::{hex_encode, Sha256};
use crate::test_support::{hex_string, is_root, TestDir, MUSE_VERSION};
use std::ffi::CString;
use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::{go_base, go_clean, go_dir, go_errno, load_tools, open_tool, verify_native, Tool};

#[test]
fn sha256_known_answers() {
    let vectors: &[(&[u8], &str)] = &[
        (
            b"",
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        ),
        (
            b"abc",
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
        ),
        // 55 bytes: padding fits in the final block.
        (
            b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318",
        ),
        // 56 bytes: padding spills into a second block.
        (
            b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a",
        ),
        // 64 bytes: exactly one full block plus padding block.
        (
            b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
        ),
    ];
    for (input, want) in vectors {
        let mut h = Sha256::new();
        // Feed one byte at a time to stress the streaming buffer.
        for b in input.iter() {
            h.update(std::slice::from_ref(b));
        }
        assert_eq!(&hex_encode(&h.finish()), want);
    }
}

#[test]
fn errno_table_spot_checks() {
    assert_eq!(go_errno(libc::ENOENT), "no such file or directory");
    assert_eq!(go_errno(libc::EACCES), "permission denied");
    assert_eq!(go_errno(libc::EPIPE), "broken pipe");
    assert_eq!(go_errno(libc::ENOSYS), "function not implemented");
    assert_eq!(go_errno(libc::ELOOP), "too many levels of symbolic links");
    assert_eq!(go_errno(libc::EINVAL), "invalid argument");
    assert_eq!(go_errno(libc::EISDIR), "is a directory");
    assert_eq!(go_errno(9999), "errno 9999");
}

#[test]
fn go_path_helpers() {
    assert_eq!(
        go_clean("/usr/libexec/soda/../../share/soda/muse-tools"),
        "/usr/share/soda/muse-tools"
    );
    assert_eq!(go_clean("a//b/./c/"), "a/b/c");
    assert_eq!(go_clean(""), ".");
    assert_eq!(go_clean("/"), "/");
    assert_eq!(go_base("/x/launch.sock"), "launch.sock");
    assert_eq!(go_base("/x/launch.sock/"), "launch.sock");
    assert_eq!(go_base("/"), "/");
    assert_eq!(go_base(""), ".");
    assert_eq!(go_dir("/x/launch.sock"), "/x");
    assert_eq!(go_dir("/launch.sock"), "/");
    assert_eq!(go_dir("/x/launch.sock/"), "/x/launch.sock");
    assert_eq!(default_tools(), "/usr/share/soda/muse-tools");
}

#[test]
fn tool_source_and_digest_admission() {
    let dir = TestDir::make("tools");
    let native = dir.path("muse-native");
    let body = b"synthetic native bytes";
    fs::write(&native, body).unwrap();
    fs::set_permissions(&native, fs::Permissions::from_mode(0o755)).unwrap();
    // Symlinks never admit, regardless of owner.
    let link = dir.path("muse");
    std::os::unix::fs::symlink(&native, &link).unwrap();
    assert_eq!(
        open_tool(&dir.dir_str(), "muse").unwrap_err(),
        "public tool source must be a regular root-owned executable"
    );
    // Group-writable and non-executable sources refuse alike.
    let weak = dir.path("weak");
    fs::write(&weak, body).unwrap();
    fs::set_permissions(&weak, fs::Permissions::from_mode(0o775)).unwrap();
    assert!(open_tool(&dir.dir_str(), "weak").is_err());
    fs::set_permissions(&weak, fs::Permissions::from_mode(0o644)).unwrap();
    assert!(open_tool(&dir.dir_str(), "weak").is_err());
    // Digest admission through an open tool fd.
    let c = CString::new(native.clone()).unwrap();
    let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
    assert!(fd >= 0);
    let tool = Tool {
        name: String::from("muse-native"),
        fd,
        size: body.len() as u64,
    };
    assert_eq!(
        verify_native(&tool, &hex_string('0', 64)).unwrap_err(),
        "muse native digest mismatch"
    );
    let mut h = Sha256::new();
    h.update(body);
    verify_native(&tool, &hex_encode(&h.finish())).unwrap();
    // pread verification leaves the offset at zero.
    assert_eq!(unsafe { libc::lseek(fd, 0, libc::SEEK_CUR) }, 0);
    drop(tool);
    if is_root() {
        open_tool(&dir.dir_str(), "muse-native").expect("root-owned source refused");
    } else {
        assert!(open_tool(&dir.dir_str(), "muse-native").is_err());
    }
    // The version gate fires before any tool opens.
    assert_eq!(
        load_tools(
            "definitely-missing",
            &hex_string('0', 64),
            &format!("{MUSE_VERSION}-other")
        )
        .unwrap_err(),
        "muse maintenance version differs from pinned release"
    );
}
