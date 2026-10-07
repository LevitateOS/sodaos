//! `soda-candidate` loopback fixture server and rootfs filing (Go
//! `tools/soda-candidate` `fixture.go`).

use std::ffi::CString;
use std::net::TcpListener;
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::pin::Pin;
use std::task::{Context, Poll};
use std::time::Duration;

use hyper::body::{Body, Bytes, Frame, Incoming, SizeHint};
use hyper::service::service_fn;
use hyper::{Request, Response, StatusCode};
use hyper_util::rt::{TokioIo, TokioTimer};
use tokio::io::{AsyncRead, ReadBuf};

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

const MAX_HEADERS: usize = 64;
const MAX_HEADER_BYTES: usize = 64 * 1024;
const MAX_CONNECTIONS: usize = 16;
const CHUNK_SIZE: usize = 64 * 1024;
const HEADER_DEADLINE: Duration = Duration::from_secs(5);

struct FixtureStop {
    shutdown: Option<tokio::sync::oneshot::Sender<()>>,
    thread: Option<std::thread::JoinHandle<()>>,
}

impl Drop for FixtureStop {
    fn drop(&mut self) {
        if let Some(shutdown) = self.shutdown.take() {
            let _ = shutdown.send(());
        }
        if let Some(thread) = self.thread.take() {
            let _ = thread.join();
        }
    }
}

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
    let dir = std::fs::File::open(dir).map_err(|e| e.to_string())?;
    let (shutdown, shutdown_rx) = tokio::sync::oneshot::channel();
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .map_err(|e| e.to_string())?;
    let (ready_tx, ready_rx) = std::sync::mpsc::sync_channel(1);
    let thread = std::thread::Builder::new()
        .name("soda-candidate-fixture".to_owned())
        .spawn(move || {
            runtime.block_on(async move {
                let listener = match tokio::net::TcpListener::from_std(listener) {
                    Ok(listener) => listener,
                    Err(error) => {
                        let _ = ready_tx.send(Err(error.to_string()));
                        return;
                    }
                };
                let _ = ready_tx.send(Ok(()));
                let dir = std::sync::Arc::new(dir);
                let slots = std::sync::Arc::new(tokio::sync::Semaphore::new(MAX_CONNECTIONS));
                let mut shutdown_rx = shutdown_rx;
                let mut tasks = tokio::task::JoinSet::new();
                loop {
                    tokio::select! {
                        _ = &mut shutdown_rx => break,
                        accepted = listener.accept() => match accepted {
                            Ok((stream, _)) => {
                                let Ok(permit) = std::sync::Arc::clone(&slots).try_acquire_owned() else {
                                    drop(stream);
                                    continue;
                                };
                                let dir = std::sync::Arc::clone(&dir);
                                tasks.spawn(async move {
                                    let _permit = permit;
                                    let service = service_fn(move |request| {
                                        let dir = std::sync::Arc::clone(&dir);
                                        async move { fixture_response(request, dir).await }
                                    });
                                    let connection = hyper::server::conn::http1::Builder::new()
                                        .timer(TokioTimer::new())
                                        .header_read_timeout(HEADER_DEADLINE)
                                        .max_headers(MAX_HEADERS)
                                        .max_header_size(MAX_HEADER_BYTES)
                                        .max_buf_size(MAX_HEADER_BYTES)
                                        .keep_alive(false)
                                        .serve_connection(TokioIo::new(stream), service);
                                    let _ = connection.await;
                                });
                            }
                            Err(_) => break,
                        },
                        _ = tasks.join_next(), if !tasks.is_empty() => {}
                    }
                }
                tasks.abort_all();
                while tasks.join_next().await.is_some() {}
            });
        })
        .map_err(|e| e.to_string())?;
    match ready_rx.recv() {
        Ok(Ok(())) => {}
        Ok(Err(error)) => {
            let _ = thread.join();
            return Err(error);
        }
        Err(error) => {
            let _ = thread.join();
            return Err(error.to_string());
        }
    }
    let stop = FixtureStop {
        shutdown: Some(shutdown),
        thread: Some(thread),
    };
    Ok((Box::new(move || drop(stop)), listen))
}

async fn fixture_response(
    request: Request<Incoming>,
    dir: std::sync::Arc<std::fs::File>,
) -> Result<Response<FixtureBody>, std::convert::Infallible> {
    let name = percent_decode(request.uri().path().trim_start_matches('/'));
    if name.contains('/') || name.contains('\\') || name.is_empty() || name == "." || name == ".." {
        return Ok(not_found());
    }
    let Ok(name) = CString::new(name) else {
        return Ok(not_found());
    };
    let fd = unsafe {
        libc::openat(
            dir.as_raw_fd(),
            name.as_ptr(),
            libc::O_RDONLY | libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK,
        )
    };
    if fd < 0 {
        return Ok(not_found());
    }
    let file = unsafe { std::fs::File::from_raw_fd(fd) };
    let Ok(metadata) = file.metadata() else {
        return Ok(not_found());
    };
    if !metadata.file_type().is_file() {
        return Ok(not_found());
    }
    let file = tokio::fs::File::from_std(file);
    let body = FixtureBody::File {
        file,
        buffer: Box::new([0; CHUNK_SIZE]),
        done: false,
    };
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(hyper::header::CONTENT_LENGTH, metadata.len())
        .header(hyper::header::CONTENT_TYPE, "application/octet-stream")
        .header(hyper::header::CONNECTION, "close")
        .body(body)
        .unwrap_or_else(|_| not_found()))
}

fn not_found() -> Response<FixtureBody> {
    Response::builder()
        .status(StatusCode::NOT_FOUND)
        .header(hyper::header::CONTENT_LENGTH, "9")
        .header(hyper::header::CONTENT_TYPE, "text/plain")
        .header(hyper::header::CONNECTION, "close")
        .body(FixtureBody::Bytes(Some(Bytes::from_static(b"not found"))))
        .expect("static response is valid")
}

enum FixtureBody {
    Bytes(Option<Bytes>),
    File {
        file: tokio::fs::File,
        buffer: Box<[u8; CHUNK_SIZE]>,
        done: bool,
    },
}

impl Body for FixtureBody {
    type Data = Bytes;
    type Error = std::io::Error;

    fn poll_frame(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Option<Result<Frame<Bytes>, Self::Error>>> {
        match self.as_mut().get_mut() {
            Self::Bytes(body) => Poll::Ready(body.take().map(|bytes| Ok(Frame::data(bytes)))),
            Self::File { file, buffer, done } => {
                if *done {
                    return Poll::Ready(None);
                }
                let mut read_buf = ReadBuf::new(buffer.as_mut_slice());
                match Pin::new(file).poll_read(cx, &mut read_buf) {
                    Poll::Pending => Poll::Pending,
                    Poll::Ready(Err(error)) => {
                        *done = true;
                        Poll::Ready(Some(Err(error)))
                    }
                    Poll::Ready(Ok(())) if read_buf.filled().is_empty() => {
                        *done = true;
                        Poll::Ready(None)
                    }
                    Poll::Ready(Ok(())) => Poll::Ready(Some(Ok(Frame::data(
                        Bytes::copy_from_slice(read_buf.filled()),
                    )))),
                }
            }
        }
    }

    fn is_end_stream(&self) -> bool {
        matches!(self, Self::Bytes(None) | Self::File { done: true, .. })
    }

    fn size_hint(&self) -> SizeHint {
        match self {
            Self::Bytes(Some(bytes)) => SizeHint::with_exact(bytes.len() as u64),
            Self::Bytes(None) | Self::File { .. } => SizeHint::new(),
        }
    }
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
