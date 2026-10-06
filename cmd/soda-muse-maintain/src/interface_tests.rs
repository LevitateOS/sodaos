use crate::interface_admission::public_socket_directory;
use crate::test_support::{is_root, TestDir};
use std::fs;
use std::os::unix::fs::PermissionsExt;

#[test]
fn public_interface_admission() {
    use std::os::unix::net::UnixListener;
    let dir = TestDir::make("sock");
    let sock = dir.path("launch.sock");
    let listener = UnixListener::bind(&sock).unwrap();
    fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
    fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
    if is_root() {
        assert_eq!(public_socket_directory(&sock).unwrap(), dir.dir_str());
        fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o775)).unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch directory must be a protected directory"
        );
        fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o700)).unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch directory must be root-owned and accessible"
        );
        fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
    }
    assert_eq!(
        public_socket_directory("relative/launch.sock").unwrap_err(),
        "explicit public launch socket required"
    );
    assert_eq!(
        public_socket_directory("/tmp/other.sock").unwrap_err(),
        "explicit public launch socket required"
    );
    fs::write(dir.path("credential"), b"synthetic private data").unwrap();
    assert_eq!(
        public_socket_directory(&sock).unwrap_err(),
        "launch directory must contain only the public socket"
    );
    fs::remove_file(dir.path("credential")).unwrap();
    // Same message whether the uid or the mode check fires.
    fs::set_permissions(&sock, fs::Permissions::from_mode(0o600)).unwrap();
    assert_eq!(
        public_socket_directory(&sock).unwrap_err(),
        "public launch socket must be root-owned and public"
    );
    fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
    drop(listener);
    fs::remove_file(&sock).unwrap();
    fs::write(&sock, b"not a socket").unwrap();
    assert_eq!(
        public_socket_directory(&sock).unwrap_err(),
        "public launch socket is unavailable"
    );
}
