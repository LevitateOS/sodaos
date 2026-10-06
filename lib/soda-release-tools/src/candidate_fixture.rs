//! `soda-candidate` loopback fixture server and rootfs filing (Go
//! `tools/soda-candidate` `fixture.go`).

use std::io::{BufRead, Read, Write};
use std::net::TcpListener;
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};

use crate::candidate::Options;
use crate::worker::lookup_user;

pub fn default_rootfs_dir(o: &mut Options) -> Result<(), String> {
    if !o.rootfs_dir.is_empty() {
        return Ok(());
    }
    let source = std::env::current_dir().map_err(|e| e.to_string())?;
    let dir = source.join(".artifacts").join("rootfs");
    o.rootfs_dir = dir.to_string_lossy().into_owned();
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o755))
        .map_err(|e| e.to_string())?;
    Ok(())
}

fn url_host(url: &str) -> Option<String> {
    let rest = url.split("://").nth(1)?;
    let authority = rest.split('/').next().unwrap_or_default();
    let authority = authority.split(['?', '#']).next().unwrap_or_default();
    if authority.is_empty() {
        return None;
    }
    if let Some(bracketed) = authority.strip_prefix('[') {
        return bracketed.split(']').next().map(str::to_owned);
    }
    Some(
        authority
            .split('@')
            .next_back()
            .unwrap_or_default()
            .split(':')
            .next()
            .unwrap_or_default()
            .to_owned(),
    )
}

fn url_port(url: &str) -> Option<String> {
    let rest = url.split("://").nth(1)?;
    let authority = rest.split('/').next().unwrap_or_default();
    if let Some(bracketed) = authority.strip_prefix('[') {
        let after = bracketed.split(']').nth(1).unwrap_or_default();
        return after
            .strip_prefix(':')
            .map(str::to_owned)
            .filter(|p| !p.is_empty());
    }
    let host_port = authority.split('@').next_back().unwrap_or_default();
    host_port
        .split_once(':')
        .map(|(_, p)| p.to_owned())
        .filter(|p| !p.is_empty())
}

/// Report whether the wrapper should serve the pickup address itself.
pub fn fixture_wanted(mode: &str, rootfs_url: &str) -> bool {
    if mode != "media" {
        return false;
    }
    if !rootfs_url.contains("://") {
        return false;
    }
    match url_host(rootfs_url) {
        Some(host) => host == "127.0.0.1" || host == "localhost" || host == "::1",
        None => false,
    }
}

/// Split the pickup URL into a listen address for the file server.
pub fn fixture_addr(rootfs_url: &str) -> Result<String, String> {
    let mut host = url_host(rootfs_url).unwrap_or_default();
    if host == "localhost" {
        host = "127.0.0.1".to_owned();
    }
    let port = url_port(rootfs_url).unwrap_or_default();
    if host.is_empty() || port.is_empty() {
        return Err("rootfs URL needs an explicit port".to_owned());
    }
    if host.contains(':') {
        Ok(format!("[{host}]:{port}"))
    } else {
        Ok(format!("{host}:{port}"))
    }
}

/// Serve `dir` over HTTP until the returned stop function runs. A busy port
/// means the operator already serves it and is not an error. Reports the
/// actual listen address so tests can use ephemeral ports.
/// Stop handle for a running fixture server plus its listen address.
pub type FixtureServer = (Box<dyn FnOnce() + Send>, String);

pub fn serve_fixture(addr: &str, dir: &str) -> Result<FixtureServer, String> {
    match std::fs::metadata(dir) {
        Ok(st) if st.file_type().is_dir() => {}
        _ => {
            return Err(format!(
                "pickup directory {dir} missing; rerun the setup script"
            ))
        }
    }
    let listener = match TcpListener::bind(addr) {
        Ok(l) => l,
        Err(e) if e.kind() == std::io::ErrorKind::AddrInUse => {
            return Ok((Box::new(|| {}), addr.to_owned()));
        }
        Err(e) => return Err(e.to_string()),
    };
    listener.set_nonblocking(true).map_err(|e| e.to_string())?;
    let listen = listener
        .local_addr()
        .map_err(|e| e.to_string())?
        .to_string();
    let dir = dir.to_owned();
    let running = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(true));
    let flag = std::sync::Arc::clone(&running);
    std::thread::spawn(move || {
        while flag.load(std::sync::atomic::Ordering::SeqCst) {
            match listener.accept() {
                Ok((stream, _)) => {
                    let _ = handle_fixture_request(stream, &dir);
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                    std::thread::sleep(std::time::Duration::from_millis(25));
                }
                Err(_) => break,
            }
        }
    });
    Ok((
        Box::new(move || {
            running.store(false, std::sync::atomic::Ordering::SeqCst);
        }),
        listen,
    ))
}

fn handle_fixture_request(stream: std::net::TcpStream, dir: &str) -> std::io::Result<()> {
    let mut reader = std::io::BufReader::new(stream.try_clone()?);
    let mut request = String::new();
    reader.read_line(&mut request)?;
    let path = request.split_whitespace().nth(1).unwrap_or("/");
    // Drain headers.
    loop {
        let mut line = String::new();
        reader.read_line(&mut line)?;
        if line.trim().is_empty() {
            break;
        }
    }
    let mut stream = reader.into_inner();
    let name = path.trim_start_matches('/');
    let name = percent_decode(name);
    if name.contains('/') || name.contains('\\') || name.is_empty() || name == "." || name == ".." {
        return write_response(&mut stream, 404, b"not found");
    }
    let file = std::path::Path::new(dir).join(&name);
    let mut body = Vec::new();
    match std::fs::File::open(&file) {
        Ok(mut f) => {
            if f.metadata()
                .map(|m| !m.file_type().is_file())
                .unwrap_or(true)
            {
                return write_response(&mut stream, 404, b"not found");
            }
            f.read_to_end(&mut body)?;
        }
        Err(_) => return write_response(&mut stream, 404, b"not found"),
    }
    write_response(&mut stream, 200, &body)
}

fn percent_decode(s: &str) -> String {
    let mut out = Vec::with_capacity(s.len());
    let bytes = s.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' && i + 2 < bytes.len() {
            if let (Some(h), Some(l)) = (hex_val(bytes[i + 1]), hex_val(bytes[i + 2])) {
                out.push(h * 16 + l);
                i += 3;
                continue;
            }
        }
        out.push(bytes[i]);
        i += 1;
    }
    String::from_utf8_lossy(&out).into_owned()
}

fn hex_val(b: u8) -> Option<u8> {
    match b {
        b'0'..=b'9' => Some(b - b'0'),
        b'a'..=b'f' => Some(b - b'a' + 10),
        b'A'..=b'F' => Some(b - b'A' + 10),
        _ => None,
    }
}

fn write_response(stream: &mut std::net::TcpStream, code: u16, body: &[u8]) -> std::io::Result<()> {
    let reason = if code == 200 { "OK" } else { "Not Found" };
    stream.write_all(format!("HTTP/1.1 {code} {reason}\r\nContent-Length: {}\r\nConnection: close\r\nContent-Type: application/octet-stream\r\n\r\n", body.len()).as_bytes())?;
    stream.write_all(body)?;
    stream.flush()
}

/// File the produced image into the served directory; report the pickup name.
pub fn copy_built_rootfs(out_dir: &str, serve_dir: &str) -> Result<String, String> {
    let media = std::path::Path::new(out_dir)
        .join("artifacts")
        .join("media");
    let mut matches = Vec::new();
    if let Ok(entries) = std::fs::read_dir(&media) {
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().into_owned();
            if name.ends_with("-rootfs.img")
                && entry.file_type().map(|t| t.is_file()).unwrap_or(false)
            {
                matches.push(entry.path());
            }
        }
    }
    matches.sort();
    let Some(first) = matches.first() else {
        return Err(
            "no rootfs image in build output; the media build did not produce one".to_owned(),
        );
    };
    let name = first
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_default();
    let dst = std::path::Path::new(serve_dir).join(&name);
    copy_file(&first.to_string_lossy(), &dst.to_string_lossy())?;
    if let Ok(user) = std::env::var("SUDO_USER") {
        if !user.is_empty() {
            let _ = chown_name(&dst.to_string_lossy(), &user);
        }
    }
    Ok(name)
}

fn chown_name(path: &str, name: &str) -> Result<(), String> {
    let (uid, gid) = lookup_user(name)?;
    let path = std::ffi::CString::new(path).map_err(|e| e.to_string())?;
    if unsafe { libc::chown(path.as_ptr(), uid, gid) } != 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    Ok(())
}

pub fn copy_file(src: &str, dst: &str) -> Result<(), String> {
    if std::fs::symlink_metadata(dst).is_ok() {
        return Err(format!("occupied pickup file {dst} refused"));
    }
    let mut input = std::fs::File::open(src).map_err(|e| e.to_string())?;
    let mut output = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o644)
        .open(dst)
        .map_err(|e| e.to_string())?;
    let copy_err = std::io::copy(&mut input, &mut output)
        .err()
        .map(|e| e.to_string());
    let close_err = output.sync_all().err().map(|e| e.to_string());
    match (copy_err, close_err) {
        (None, None) => Ok(()),
        (Some(e), None) | (None, Some(e)) => Err(e),
        (Some(a), Some(b)) => Err(format!("{a}; {b}")),
    }
}

#[cfg(test)]
mod tests;
