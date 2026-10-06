use super::*;
use std::io::{Read, Write};
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

fn temp_dir(tag: &str) -> std::path::PathBuf {
    let id = COUNTER.fetch_add(1, Ordering::SeqCst);
    let path = std::env::temp_dir().join(format!(
        "soda-reltools-fixture-{tag}-{}-{id}",
        std::process::id()
    ));
    let _ = std::fs::remove_dir_all(&path);
    std::fs::create_dir_all(&path).unwrap();
    path
}

#[test]
fn fixture_wanted_matrix() {
    for u in [
        "http://127.0.0.1:8080",
        "http://localhost:8080",
        "http://[::1]:8080",
    ] {
        assert!(fixture_wanted("media", u), "{u}");
    }
    assert!(!fixture_wanted("media", "https://example.invalid/rootfs"));
    assert!(!fixture_wanted("candidate", "http://127.0.0.1:8080"));
    assert!(!fixture_wanted("", "http://127.0.0.1:8080"));
    assert!(!fixture_wanted("media", "://bogus"));
}

#[test]
fn fixture_addr_matrix() {
    assert_eq!(
        fixture_addr("http://localhost:8080").unwrap(),
        "127.0.0.1:8080"
    );
    assert!(fixture_addr("http://127.0.0.1/").is_err());
}

fn http_get(addr: &str, path: &str) -> (u16, Vec<u8>) {
    let mut stream = std::net::TcpStream::connect(addr).unwrap();
    stream
        .set_read_timeout(Some(std::time::Duration::from_secs(5)))
        .unwrap();
    write!(
        stream,
        "GET {path} HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"
    )
    .unwrap();
    let mut body = Vec::new();
    stream.read_to_end(&mut body).unwrap();
    let header_end = body.windows(4).position(|w| w == b"\r\n\r\n").unwrap() + 4;
    let header = String::from_utf8_lossy(&body[..header_end]).into_owned();
    let code = header
        .split_whitespace()
        .nth(1)
        .unwrap_or("0")
        .parse()
        .unwrap_or(0);
    (code, body[header_end..].to_vec())
}

#[test]
fn serve_and_file_rootfs() {
    let serve_dir = temp_dir("serve");
    let (stop, listen) = serve_fixture("127.0.0.1:0", serve_dir.to_str().unwrap()).unwrap();
    let out = temp_dir("out");
    let media = out.join("artifacts").join("media");
    std::fs::create_dir_all(&media).unwrap();
    std::fs::write(media.join("a-rootfs.img"), b"payload").unwrap();
    let name = copy_built_rootfs(out.to_str().unwrap(), serve_dir.to_str().unwrap()).unwrap();
    assert_eq!(name, "a-rootfs.img");
    let (code, body) = http_get(&listen, &format!("/{name}"));
    assert_eq!(code, 200);
    assert_eq!(body, b"payload");
    let (code, _) = http_get(&listen, "/missing.img");
    assert_eq!(code, 404);
    stop();
    let _ = std::fs::remove_dir_all(&serve_dir);
    let _ = std::fs::remove_dir_all(&out);
}

#[test]
fn serve_fixture_busy_port_is_not_an_error() {
    let serve_dir = temp_dir("busy");
    let (stop, listen) = serve_fixture("127.0.0.1:0", serve_dir.to_str().unwrap()).unwrap();
    let (stop2, listen2) = serve_fixture(&listen, serve_dir.to_str().unwrap()).unwrap();
    assert_eq!(listen2, listen);
    stop();
    stop2();
    let _ = std::fs::remove_dir_all(&serve_dir);
}

#[test]
fn copy_file_refuses_occupied_pickup() {
    let dir = temp_dir("copy");
    let src = dir.join("src.img");
    let dst = dir.join("served.img");
    std::fs::write(&src, b"new").unwrap();
    copy_file(src.to_str().unwrap(), dst.to_str().unwrap()).unwrap();
    std::fs::write(&src, b"changed").unwrap();
    assert!(copy_file(src.to_str().unwrap(), dst.to_str().unwrap()).is_err());
    assert_eq!(std::fs::read(&dst).unwrap(), b"new");
    std::fs::remove_file(&dst).unwrap();
    std::os::unix::fs::symlink(&src, &dst).unwrap();
    assert!(copy_file(src.to_str().unwrap(), dst.to_str().unwrap()).is_err());
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn copy_built_rootfs_needs_an_image() {
    let out = temp_dir("noimg");
    let serve = temp_dir("noserve");
    assert!(copy_built_rootfs(out.to_str().unwrap(), serve.to_str().unwrap()).is_err());
    let _ = std::fs::remove_dir_all(&out);
    let _ = std::fs::remove_dir_all(&serve);
}
