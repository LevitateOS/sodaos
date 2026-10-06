// Oracle + loopback tests for the muse launch/serve loop (muse_serve.rs).
//
// `muse_serve` is NOT wired into lib.rs yet (the integrator owns that
// one-liner, same as the gmux skeleton), so this harness pulls the module
// closure directly. If these includes compile, the module compiles against
// the already-ported runtime; if these tests pass, the daemon-called
// surface matches Go.
//
// Scratch sockets live under the package target dir inside the worktree,
// never under /tmp.

#[path = "../src/account/mod.rs"]
#[allow(dead_code)]
mod account;
#[path = "../src/domain/mod.rs"]
#[allow(dead_code)]
mod domain;
#[path = "../src/json.rs"]
#[allow(dead_code)]
mod json;
#[path = "../src/muse.rs"]
#[allow(dead_code)]
mod muse;
#[path = "../src/muse_serve.rs"]
mod muse_serve;
#[path = "../src/net.rs"]
#[allow(dead_code)]
mod net;
#[path = "../src/nist.rs"]
#[allow(dead_code)]
mod nist;
#[path = "../src/project/mod.rs"]
#[allow(dead_code)]
mod project;
#[path = "../src/sha256.rs"]
#[allow(dead_code)]
mod sha256;
#[path = "../src/ssh/mod.rs"]
#[allow(dead_code)]
mod ssh;
#[path = "../src/terminal/mod.rs"]
#[allow(dead_code)]
mod terminal;

use muse::{
    muse_arguments, muse_command_argv, LaunchExit, LaunchRequest, MuseCaller, MuseHooks,
    MuseLaunch, MusePeer, MuseRuntime,
};
use muse_serve::open_muse_listener;
use project::Executor;
use std::os::unix::fs::PermissionsExt;
use std::os::unix::io::{AsRawFd, RawFd};
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use terminal::{AcquireRequest, Binding, Delivery, Lease};

// ---------- fixtures ----------

const PID: &str = "p0123456789abcdef01234567";
const TID: &str = "0123456789abcdef0123456789abcdef";
const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const CHID: &str = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
const RID: &str = "cccccccccccccccccccccccccccccccc";
const PIN: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

type RecordedCall = (Vec<u8>, String, Vec<String>);

struct FakeExec {
    calls: Mutex<Vec<RecordedCall>>,
    script: Mutex<Vec<Result<Vec<u8>, String>>>,
}

impl FakeExec {
    fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
        FakeExec {
            calls: Mutex::new(Vec::new()),
            script: Mutex::new(script),
        }
    }

    fn calls(&self) -> Vec<RecordedCall> {
        self.calls.lock().unwrap().clone()
    }
}

impl Executor for FakeExec {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.lock().unwrap().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        let mut script = self.script.lock().unwrap();
        if script.is_empty() {
            return Err("unexpected call".to_string());
        }
        script.remove(0)
    }
}

fn ok(body: &str) -> Result<Vec<u8>, String> {
    Ok(body.as_bytes().to_vec())
}

fn err(body: &str) -> Result<Vec<u8>, String> {
    Err(body.to_string())
}

struct FakeHooks {
    end_calls: Mutex<Vec<(i64, String)>>,
}

impl FakeHooks {
    fn new() -> Self {
        FakeHooks {
            end_calls: Mutex::new(Vec::new()),
        }
    }
}

impl MuseHooks for FakeHooks {
    fn acquire(&self, _req: &AcquireRequest, _deadline: Instant) -> Result<Lease, String> {
        Err("no acquire".to_string())
    }
    fn attach(
        &self,
        _lease_id: &str,
        _binding: &Binding,
        _deadline: Instant,
    ) -> Result<Delivery, String> {
        Err("no attach".to_string())
    }
    fn end(&self, actor: i64, lease_id: &str, _deadline: Instant) -> Result<(), String> {
        self.end_calls
            .lock()
            .unwrap()
            .push((actor, lease_id.to_string()));
        Ok(())
    }
    fn authorize(&self, _actor: i64, _project: &str, _deadline: Instant) -> Result<(), String> {
        Ok(())
    }
    fn nested_authorize(
        &self,
        _actor: i64,
        _project: &str,
        _deadline: Instant,
    ) -> Result<(), String> {
        Ok(())
    }
    fn select(
        &self,
        _actor: i64,
        _project: &str,
        _selected: &str,
        _deadline: Instant,
    ) -> Result<String, String> {
        Ok("conn-1".to_string())
    }
}

fn runtime(exec: FakeExec) -> MuseRuntime<FakeExec, FakeHooks> {
    MuseRuntime::new(exec, FakeHooks::new(), "1.0".to_string(), PIN.to_string())
}

fn lease_fixture() -> Lease {
    Lease {
        provider_id: terminal::PROVIDER_MUSE.to_string(),
        id: "lease-m".to_string(),
        connection_id: "conn-1".to_string(),
        generation: 4,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: TID.to_string(),
        kind: terminal::KIND_TERMINAL.to_string(),
        binding: Some(Binding {
            kind: terminal::KIND_TERMINAL.to_string(),
            id: TID.to_string(),
            project: CID.to_string(),
            login: "dev".to_string(),
            uid: 1000,
            gid: 1000,
            scope: terminal::SCOPE_MUSE_PROJECT.to_string(),
            invocation_id: IID.to_string(),
            credential_root: format!("/run/soda-muse/{TID}"),
            generation: 4,
            ..Default::default()
        }),
        ..Default::default()
    }
}

fn delivery_fixture() -> Delivery {
    Delivery {
        lease: lease_fixture(),
        credential: None,
    }
}

fn valid_launch_request() -> LaunchRequest {
    LaunchRequest {
        cwd: "/work".to_string(),
        term: "xterm".to_string(),
        ..Default::default()
    }
}

fn self_pid() -> i32 {
    std::process::id() as i32
}

/// Peer whose pidfd is already dead: `poll(-1)` errors, so the runtime
/// treats the caller as gone before touching the executor.
fn dead_peer() -> MusePeer {
    MusePeer {
        pid: self_pid(),
        uid: 1000,
        gid: 1000,
        pidfd: -1,
    }
}

/// Peer with a live (poll-quiet) pidfd stand-in: the read end of an open
/// pipe. Returns the peer plus both pipe ends for the caller to close.
fn live_peer() -> (MusePeer, [RawFd; 2]) {
    let mut pair = [0; 2];
    assert_eq!(unsafe { libc::pipe(pair.as_mut_ptr()) }, 0);
    (
        MusePeer {
            pid: self_pid(),
            uid: 1000,
            gid: 1000,
            pidfd: pair[0],
        },
        pair,
    )
}

// ---------- worktree scratch ----------

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Unique scratch dir under the package target dir (short components: unix
/// socket paths must stay under 108 bytes).
fn work_tmp() -> PathBuf {
    let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("target")
        .join(format!("m{}", std::process::id()))
        .join(format!("{n}"));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

fn sock_path(dir: &std::path::Path, name: &str) -> String {
    let path = dir.join(name).to_str().unwrap().to_string();
    assert!(path.len() < 100, "socket path too long: {path}");
    path
}

fn cleanup(dir: &std::path::Path) {
    std::fs::remove_dir_all(dir).ok();
}

// ---------- seqpacket client plumbing ----------

fn seqpacket_connect(path: &str) -> RawFd {
    unsafe {
        let fd = libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET | libc::SOCK_CLOEXEC, 0);
        assert!(fd >= 0, "client socket failed");
        let mut addr: libc::sockaddr_un = std::mem::zeroed();
        addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
        let bytes = path.as_bytes();
        assert!(bytes.len() < addr.sun_path.len());
        for (i, b) in bytes.iter().enumerate() {
            addr.sun_path[i] = *b as libc::c_char;
        }
        let len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
        let rc = libc::connect(fd, &addr as *const _ as *const libc::sockaddr, len);
        assert_eq!(rc, 0, "connect failed");
        fd
    }
}

fn close_fd(fd: RawFd) {
    unsafe {
        libc::close(fd);
    }
}

fn devnull() -> RawFd {
    let fd = unsafe { libc::open(c"/dev/null".as_ptr(), libc::O_RDWR) };
    assert!(fd >= 0, "/dev/null open failed");
    fd
}

/// One datagram plus SCM_RIGHTS descriptors, like the Go launcher client.
fn send_with_fds(fd: RawFd, body: &[u8], fds: &[RawFd]) {
    unsafe {
        let mut iov = libc::iovec {
            iov_base: body.as_ptr() as *mut libc::c_void,
            iov_len: body.len(),
        };
        let mut hdr: libc::msghdr = std::mem::zeroed();
        hdr.msg_iov = &mut iov;
        hdr.msg_iovlen = 1;
        let mut control;
        if !fds.is_empty() {
            control = vec![0u8; libc::CMSG_SPACE((fds.len() * 4) as libc::c_uint) as usize];
            hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
            hdr.msg_controllen = control.len() as _;
            let cmsg = libc::CMSG_FIRSTHDR(&hdr);
            assert!(!cmsg.is_null());
            (*cmsg).cmsg_len = libc::CMSG_LEN((fds.len() * 4) as libc::c_uint) as _;
            (*cmsg).cmsg_level = libc::SOL_SOCKET;
            (*cmsg).cmsg_type = libc::SCM_RIGHTS;
            let data = libc::CMSG_DATA(cmsg) as *mut RawFd;
            for (i, f) in fds.iter().enumerate() {
                *data.add(i) = *f;
            }
            let n = libc::sendmsg(fd, &hdr, 0);
            assert_eq!(n, body.len() as isize, "sendmsg short");
        } else {
            let n = libc::send(
                fd,
                body.as_ptr() as *const libc::c_void,
                body.len(),
                libc::MSG_NOSIGNAL,
            );
            assert_eq!(n, body.len() as isize, "send short");
        }
    }
}

fn recv_line(fd: RawFd) -> String {
    let mut out = Vec::new();
    let mut chunk = [0u8; 4096];
    loop {
        let mut pfd = libc::pollfd {
            fd,
            events: libc::POLLIN,
            revents: 0,
        };
        let ready = unsafe { libc::poll(&mut pfd, 1, 5000) };
        assert!(ready > 0, "response timeout");
        let n = unsafe { libc::recv(fd, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len(), 0) };
        assert!(n > 0, "recv failed");
        out.extend_from_slice(&chunk[..n as usize]);
        if out.contains(&b'\n') {
            break;
        }
        assert!(out.len() < 65536, "response too long");
    }
    String::from_utf8(out).unwrap()
}

// ---------- `start` unit tests (fake Executor/Hooks) ----------

#[test]
fn start_rejects_invalid_request_without_touching_exec() {
    let rt = runtime(FakeExec::new(vec![]));
    let req = LaunchRequest {
        cwd: "relative".to_string(),
        ..Default::default()
    };
    assert_eq!(
        rt.start(&dead_peer(), &req, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.exec.calls().is_empty());
}

#[test]
fn start_denies_dead_peer_without_touching_exec() {
    let rt = runtime(FakeExec::new(vec![]));
    assert_eq!(
        rt.start(&dead_peer(), &valid_launch_request(), deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.exec.calls().is_empty());
}

#[test]
fn start_denies_live_peer_outside_any_project() {
    // Attested but unresolvable: this test process has no libpod scope (and
    // even under one, the empty exec script denies at inspect). Either way
    // the outcome is denied without spawning anything.
    let rt = runtime(FakeExec::new(vec![]));
    let (peer, pipe) = live_peer();
    assert_eq!(
        rt.start(&peer, &valid_launch_request(), deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    close_fd(pipe[0]);
    close_fd(pipe[1]);
}

// ---------- `muse` action dispatch (fake Executor/Hooks) ----------

#[test]
fn muse_validate_ok_echoes_delivery() {
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n")), ok("active\n")]));
    assert_eq!(
        rt.muse("validate", &delivery, deadline()).unwrap(),
        delivery
    );
    assert!(rt.hooks.end_calls.lock().unwrap().is_empty());
}

#[test]
fn muse_validate_stale_on_invocation_mismatch() {
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![ok(&format!("{}\n", "b".repeat(32)))]));
    assert_eq!(
        rt.muse("validate", &delivery, deadline()).unwrap_err(),
        terminal::ERR_STALE
    );
    assert_eq!(rt.exec.calls().len(), 1);
}

#[test]
fn muse_stop_ok_without_custody_return() {
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{IID}\n")),
        ok(""),
        ok("inactive\n"),
        err("exit status 1"),
        ok(""),
        ok(""),
    ]));
    assert_eq!(rt.muse("stop", &delivery, deadline()).unwrap(), delivery);
    assert!(rt.hooks.end_calls.lock().unwrap().is_empty());
}

#[test]
fn muse_start_action_denied_like_go() {
    // Go `projectOperation` serves only validate/stop; start denies after
    // the invocation check.
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n"))]));
    assert_eq!(
        rt.muse("start", &delivery, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(rt.exec.calls().len(), 1);
}

#[test]
fn muse_finish_action_denied_like_go() {
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n"))]));
    assert_eq!(
        rt.muse("finish", &delivery, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(rt.exec.calls().len(), 1);
}

#[test]
fn muse_unknown_action_denied_after_invocation_check() {
    let delivery = delivery_fixture();
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n"))]));
    assert_eq!(
        rt.muse("launch", &delivery, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(rt.exec.calls().len(), 1);
}

#[test]
fn muse_malformed_delivery_never_calls_out() {
    let rt = runtime(FakeExec::new(vec![]));
    assert_eq!(
        rt.muse("validate", &Delivery::default(), deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    let mut other = lease_fixture();
    other.binding.as_mut().unwrap().scope = "other".to_string();
    assert_eq!(
        rt.muse(
            "validate",
            &Delivery {
                lease: other,
                credential: None,
            },
            deadline()
        )
        .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.exec.calls().is_empty());
}

// ---------- listener setup ----------

#[test]
fn open_listener_binds_seqpacket_world_writable() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    // Unixpacket type, like Go's "unixpacket" net.
    let mut typ: libc::c_int = 0;
    let mut len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
    let rc = unsafe {
        libc::getsockopt(
            listener.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_TYPE,
            &mut typ as *mut _ as *mut libc::c_void,
            &mut len,
        )
    };
    assert_eq!(rc, 0);
    assert_eq!(typ, libc::SOCK_SEQPACKET);
    // Go's chmod 0666, exact.
    let mode = std::fs::metadata(&sock).unwrap().permissions().mode() & 0o777;
    assert_eq!(mode, 0o666);
    // A seqpacket client can connect.
    let client = seqpacket_connect(&sock);
    close_fd(client);
    drop(listener);
    cleanup(&dir);
}

#[test]
fn open_listener_refuses_nonempty_dir() {
    let dir = work_tmp();
    std::fs::write(dir.join("junk"), b"x").unwrap();
    let sock = sock_path(&dir, "l.sock");
    assert_eq!(
        open_muse_listener(&sock).unwrap_err(),
        "muse interface directory must be empty before launch service startup"
    );
    cleanup(&dir);
}

#[test]
fn open_listener_reports_empty_before_occupied() {
    // Go checks ReadDir before Lstat: an occupied socket inside a non-empty
    // dir reports emptiness first (the occupied refusal itself is only
    // reachable via a create-between-checks race).
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    std::fs::write(&sock, b"x").unwrap();
    std::fs::write(dir.join("junk"), b"y").unwrap();
    assert_eq!(
        open_muse_listener(&sock).unwrap_err(),
        "muse interface directory must be empty before launch service startup"
    );
    cleanup(&dir);
}

#[test]
fn open_listener_rejects_overlong_path() {
    let dir = work_tmp();
    let sock = dir.join("x".repeat(120)).to_str().unwrap().to_string();
    assert_eq!(
        open_muse_listener(&sock).unwrap_err(),
        "muse launch socket path too long"
    );
    cleanup(&dir);
}

// ---------- serve loopback through the existing muse::MuseLaunch ----------

fn spawn_existing_serve(
    svc: Arc<MuseLaunch<FakeExec, FakeHooks>>,
    listen_fd: RawFd,
    shutdown: Arc<AtomicBool>,
) -> std::thread::JoinHandle<Result<(), String>> {
    std::thread::spawn(move || {
        let flag = shutdown.clone();
        svc.serve(listen_fd, &flag)
    })
}

fn launch_wire() -> Vec<u8> {
    br#"{"home":"","config_home":"","term":"xterm","connection_id":"","cwd":"/work","args":[],"tty":false,"cols":0,"rows":0}"#.to_vec()
}

fn register_wire() -> Vec<u8> {
    let mut s = String::from(r#"{"home":"","register":{"child_id":""#);
    s.push_str(CHID);
    s.push_str(r#"","actor_id":"42","registration_id":""#);
    s.push_str(RID);
    s.push_str(
        r#"","muse":true},"config_home":"","term":"","connection_id":"","cwd":"","args":[],"tty":false,"cols":0,"rows":0}"#,
    );
    s.into_bytes()
}

const DENIED_LINE: &str = "{\"code\":1,\"error\":\"Muse launch denied\"}\n";

/// Full launch path through the real accept loop: attested peer, decoded
/// request, dispatched register — denied here because the test process is
/// not a project container, but the loop mechanics are fully exercised.
#[test]
fn serve_loopback_register_denied() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    let svc = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker = spawn_existing_serve(svc, listener.as_raw_fd(), shutdown.clone());

    let client = seqpacket_connect(&sock);
    send_with_fds(client, &register_wire(), &[]);
    assert_eq!(recv_line(client), DENIED_LINE);
    close_fd(client);

    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    drop(listener);
    cleanup(&dir);
}

/// Launch dispatch with real SCM_RIGHTS stdio: prepare denies before any
/// spawn, and the denied outcome is encoded on the wire.
#[test]
fn serve_loopback_launch_denied() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    let svc = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker = spawn_existing_serve(svc, listener.as_raw_fd(), shutdown.clone());

    let client = seqpacket_connect(&sock);
    let stdio = [devnull(), devnull(), devnull()];
    send_with_fds(client, &launch_wire(), &stdio);
    for fd in stdio {
        close_fd(fd);
    }
    assert_eq!(recv_line(client), DENIED_LINE);
    close_fd(client);

    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    drop(listener);
    cleanup(&dir);
}

#[test]
fn serve_loopback_garbage_denied() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    let svc = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker = spawn_existing_serve(svc, listener.as_raw_fd(), shutdown.clone());

    let client = seqpacket_connect(&sock);
    send_with_fds(client, b"not json{{{", &[]);
    assert_eq!(recv_line(client), DENIED_LINE);
    close_fd(client);

    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    drop(listener);
    cleanup(&dir);
}

#[test]
fn serve_loopback_never_echoes_request_bytes() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    let svc = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker = spawn_existing_serve(svc, listener.as_raw_fd(), shutdown.clone());

    let client = seqpacket_connect(&sock);
    send_with_fds(client, b"{\"x\":\"SECRET-MARKER-9f3c7a\"", &[]);
    let response = recv_line(client);
    assert_eq!(response, DENIED_LINE);
    assert!(!response.contains("SECRET-MARKER"));
    close_fd(client);

    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    drop(listener);
    cleanup(&dir);
}

#[test]
fn serve_loopback_shutdown_clean_without_connections() {
    let dir = work_tmp();
    let sock = sock_path(&dir, "l.sock");
    let listener = open_muse_listener(&sock).unwrap();
    let svc = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker = spawn_existing_serve(svc, listener.as_raw_fd(), shutdown.clone());
    std::thread::sleep(Duration::from_millis(150));
    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    drop(listener);
    cleanup(&dir);
}

// ---------- Go parity oracles ----------

#[test]
fn oracle_launch_exit_wire_bytes() {
    assert_eq!(
        LaunchExit::denied().encode(),
        "{\"code\":1,\"error\":\"Muse launch denied\"}"
    );
    assert_eq!(
        LaunchExit {
            code: 0,
            error: String::new(),
        }
        .encode(),
        "{\"code\":0}"
    );
    assert_eq!(
        LaunchExit::cleanup_unconfirmed().encode(),
        "{\"code\":1,\"error\":\"Muse cleanup unconfirmed\"}"
    );
    assert!(LaunchExit::denied().encode_line().ends_with('\n'));
}

#[test]
fn oracle_launch_request_go_shapes() {
    // Full Go wire shape decodes field-exact.
    let req = LaunchRequest::decode(
        br#"{"home":"/h","config_home":"/c","term":"xterm","connection_id":"conn","cwd":"/work","args":["a","b"],"tty":true,"cols":80,"rows":24}"#,
    )
    .unwrap();
    assert_eq!(req.home, "/h");
    assert_eq!(req.config_home, "/c");
    assert_eq!(req.term, "xterm");
    assert_eq!(req.connection_id, "conn");
    assert_eq!(req.cwd, "/work");
    assert_eq!(req.args, vec!["a".to_string(), "b".to_string()]);
    assert!(req.tty);
    assert_eq!((req.cols, req.rows), (80, 24));
    assert!(req.register.is_none());
    assert!(req.validate().is_ok());
    // Missing keys default like Go zero values.
    let empty = LaunchRequest::decode(b"{}").unwrap();
    assert_eq!(empty, LaunchRequest::default());
    // Register shape with `,string` actor id.
    let reg = LaunchRequest::decode(&register_wire()).unwrap();
    let nested = reg.register.unwrap();
    assert_eq!(nested.child_id, CHID);
    assert_eq!(nested.actor_id, 42);
    assert_eq!(nested.registration_id, RID);
    assert!(nested.muse);
    // Strict: unknown fields rejected.
    assert!(LaunchRequest::decode(br#"{"cwd":"/work","bogus":1}"#).is_err());
    // Register requests must not carry launch preferences.
    let mut bad = LaunchRequest::decode(&register_wire()).unwrap();
    bad.tty = true;
    assert!(bad.validate().is_err());
    // Launch requests need absolute cwd.
    let mut rel = valid_launch_request();
    rel.cwd = "work".to_string();
    assert!(rel.validate().is_err());
}

#[test]
fn oracle_muse_arguments_vectors() {
    // Mirrors Go MuseArguments/museProviderArguments/musePositional.
    let v = |args: &[&str]| -> Vec<String> { args.iter().map(|s| s.to_string()).collect() };
    assert_eq!(muse_arguments(&v(&[])).unwrap(), v(&["--provider", "meta"]));
    assert_eq!(
        muse_arguments(&v(&["exec"])).unwrap(),
        v(&["exec", "--provider", "meta"])
    );
    assert_eq!(
        muse_arguments(&v(&["serve", "--agents", "3"])).unwrap(),
        v(&["serve", "--provider", "meta", "--agents", "3"])
    );
    assert_eq!(
        muse_arguments(&v(&["config", "get"])).unwrap(),
        v(&["config", "get"])
    );
    // Auth surfaces denied; only the FIRST positional counts.
    assert!(muse_arguments(&v(&["auth"])).is_err());
    assert!(muse_arguments(&v(&["--model", "x", "login"])).is_err());
    assert_eq!(
        muse_arguments(&v(&["resume", "--preset", "p", "auth"])).unwrap(),
        v(&["resume", "--provider", "meta", "--preset", "p", "auth"])
    );
    // Provider overrides denied.
    assert!(muse_arguments(&v(&["--provider", "meta"])).is_err());
    assert!(muse_arguments(&v(&["--base-url=x"])).is_err());
}

#[test]
fn oracle_muse_command_argv_project_and_nested() {
    // Hand-derived from Go museCommand (muse_execution_linux.go).
    let unit = format!("soda-muse-{TID}.service");
    let path = format!("/run/soda-muse/{TID}");
    let caller = MuseCaller {
        container: CID.to_string(),
        login: "dev".to_string(),
        home: "/home/dev".to_string(),
        gid: 1000,
        ..Default::default()
    };
    let req = LaunchRequest {
        cwd: "/work/proj".to_string(),
        term: "xterm-256color".to_string(),
        tty: true,
        args: vec!["--provider".to_string(), "meta".to_string()],
        ..Default::default()
    };
    let unit_arg = format!("--unit={unit}");
    let config_home = format!("--setenv=XDG_CONFIG_HOME={path}/config");
    let state_home = format!("--setenv=XDG_STATE_HOME={path}/state");
    let cache_home = format!("--setenv=XDG_CACHE_HOME={path}/cache");
    let readonly = format!("--property=ReadOnlyPaths={path}/auth.json");
    let bind_readonly =
        format!("--property=BindReadOnlyPaths={path}/auth.json:{path}/config/muse/auth.json");
    let project_argv = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        "--tty".to_string(),
        CID.to_string(),
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--wait".to_string(),
        "--collect".to_string(),
        "--service-type=exec".to_string(),
        unit_arg,
        "--property=KillMode=control-group".to_string(),
        "--property=TimeoutStopSec=10".to_string(),
        "--property=RuntimeMaxSec=43200".to_string(),
        "--property=UMask=0077".to_string(),
        "--setenv=HOME=/home/dev".to_string(),
        "--setenv=USER=dev".to_string(),
        "--setenv=LOGNAME=dev".to_string(),
        "--setenv=TERM=xterm-256color".to_string(),
        config_home,
        state_home,
        cache_home,
        "--setenv=TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--setenv=PATH=/usr/local/bin:/usr/bin:/bin".to_string(),
        "--uid=dev".to_string(),
        "--gid=1000".to_string(),
        "--working-directory=/work/proj".to_string(),
        readonly,
        bind_readonly,
        "--pty".to_string(),
        "--".to_string(),
        "/usr/local/bin/muse".to_string(),
        "--soda-exec".to_string(),
        path.clone(),
        "/work/proj".to_string(),
        "--provider".to_string(),
        "meta".to_string(),
    ];
    assert_eq!(muse_command_argv(&caller, &req, &unit, &path), project_argv);

    // Nested: no --tty/uid/gid/mount props, --pipe, nsenter hop, remapped
    // credential path, custom HOME.
    let nested_path = format!("/run/soda-muse/nested/{RID}/{TID}");
    let nested = MuseCaller {
        container: CID.to_string(),
        child: CHID.to_string(),
        registration: RID.to_string(),
        login: "nested".to_string(),
        home: "/home/nested".to_string(),
        uid: 2000,
        gid: 2000,
        nested_pid: 4242,
        ..Default::default()
    };
    let nested_req = LaunchRequest {
        home: "/custom/home".to_string(),
        cwd: "/jobs".to_string(),
        term: "dumb".to_string(),
        ..Default::default()
    };
    let nested_unit = format!("--unit={unit}");
    let nested_config = format!("--setenv=XDG_CONFIG_HOME={nested_path}/config");
    let nested_state = format!("--setenv=XDG_STATE_HOME={nested_path}/state");
    let nested_cache = format!("--setenv=XDG_CACHE_HOME={nested_path}/cache");
    let nested_argv = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        CID.to_string(),
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--wait".to_string(),
        "--collect".to_string(),
        "--service-type=exec".to_string(),
        nested_unit,
        "--property=KillMode=control-group".to_string(),
        "--property=TimeoutStopSec=10".to_string(),
        "--property=RuntimeMaxSec=43200".to_string(),
        "--property=UMask=0077".to_string(),
        "--setenv=HOME=/custom/home".to_string(),
        "--setenv=USER=nested".to_string(),
        "--setenv=LOGNAME=nested".to_string(),
        "--setenv=TERM=dumb".to_string(),
        nested_config,
        nested_state,
        nested_cache,
        "--setenv=TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--setenv=PATH=/usr/local/bin:/usr/bin:/bin".to_string(),
        "--pipe".to_string(),
        "--".to_string(),
        "/usr/bin/nsenter".to_string(),
        "--target=4242".to_string(),
        "--mount".to_string(),
        "--pid".to_string(),
        "--uts".to_string(),
        "--ipc".to_string(),
        "--net".to_string(),
        "--root".to_string(),
        "--setuid=2000".to_string(),
        "--setgid=2000".to_string(),
        "--".to_string(),
        "/usr/local/bin/muse".to_string(),
        "--soda-exec".to_string(),
        format!("/run/soda-muse/credentials/{TID}"),
        "/jobs".to_string(),
    ];
    assert_eq!(
        muse_command_argv(&nested, &nested_req, &unit, &nested_path),
        nested_argv
    );
}
