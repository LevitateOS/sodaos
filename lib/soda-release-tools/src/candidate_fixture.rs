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
use percent_encoding::percent_decode_str;
use tokio::io::{AsyncRead, ReadBuf};
use url::{Host, Url};

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

fn pickup_url(url: &str) -> Option<(Host<String>, Option<u16>)> {
    if url.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') || !valid_percent_escapes(url) {
        return None;
    }
    let (_, rest) = url.split_once("://")?;
    let authority = rest.split(['/', '?', '#']).next()?;
    if authority.is_empty()
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return None;
    }
    let hostport = authority.rsplit('@').next()?;
    let port_text = if hostport.starts_with('[') {
        hostport.split_once(']')?.1.strip_prefix(':')
    } else {
        hostport.rsplit_once(':').map(|(_, port)| port)
    };
    let port = match port_text {
        Some(text) if !text.is_empty() => Some(text.parse::<u16>().ok().filter(|port| *port != 0)?),
        Some(_) => return None,
        None => None,
    };
    let parsed = Url::parse(url).ok()?;
    let host = match parsed.host()? {
        Host::Domain(host) => Host::Domain(host.to_owned()),
        Host::Ipv4(host) => Host::Ipv4(host),
        Host::Ipv6(host) => Host::Ipv6(host),
    };
    Some((host, port))
}

fn valid_percent_escapes(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at] == b'%' {
            if at + 2 >= bytes.len()
                || !bytes[at + 1].is_ascii_hexdigit()
                || !bytes[at + 2].is_ascii_hexdigit()
            {
                return false;
            }
            at += 3;
        } else {
            at += 1;
        }
    }
    true
}

/// Report whether the wrapper should serve the pickup address itself.
pub fn fixture_wanted(mode: &str, rootfs_url: &str) -> bool {
    if mode != "media" {
        return false;
    }
    match pickup_url(rootfs_url)
        .filter(|(_, port)| port.is_none_or(|port| port != 0))
        .map(|(host, _)| host)
    {
        Some(Host::Domain(host)) => host.eq_ignore_ascii_case("localhost"),
        Some(Host::Ipv4(host)) => host.is_loopback(),
        Some(Host::Ipv6(host)) => host.is_loopback(),
        None => false,
    }
}

/// Split the pickup URL into a listen address for the file server.
pub fn fixture_addr(rootfs_url: &str) -> Result<String, String> {
    let (host, port) = pickup_url(rootfs_url)
        .filter(|(host, _)| match host {
            Host::Domain(host) => host.eq_ignore_ascii_case("localhost"),
            Host::Ipv4(host) => host.is_loopback(),
            Host::Ipv6(host) => host.is_loopback(),
        })
        .ok_or_else(|| "rootfs URL needs an explicit loopback port".to_owned())?;
    let port = port.ok_or_else(|| "rootfs URL needs an explicit loopback port".to_owned())?;
    let host = match host {
        Host::Domain(_) | Host::Ipv4(_) => "127.0.0.1".to_owned(),
        Host::Ipv6(host) => host.to_string(),
    };
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
    percent_decode_str(s).decode_utf8_lossy().into_owned()
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
