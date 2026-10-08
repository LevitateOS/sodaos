use super::*;
use crate::project::Executor;
use crate::terminal;
use std::os::fd::AsRawFd;
use std::os::unix::io::RawFd;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

const PID: &str = "p0123456789abcdef01234567";
const TID: &str = "0123456789abcdef0123456789abcdef";
const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const CID: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const PIN: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);
static ENV_LOCK: Mutex<()> = Mutex::new(());

fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn test_tmp(slug: &str) -> std::path::PathBuf {
    let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("target/tmp/muse-tests")
        .join(format!("t26m-{}-{n}-{slug}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
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
    select_out: Mutex<Result<String, String>>,
    acquire_out: Mutex<Result<Lease, String>>,
    attach_out: Mutex<Result<Delivery, String>>,
    end_calls: Mutex<Vec<(i64, String)>>,
    end_out: Mutex<Result<(), String>>,
    authorize_out: Mutex<Result<(), String>>,
    nested_out: Mutex<Result<(), String>>,
    selects: Mutex<Vec<(i64, String, String)>>,
    acquires: Mutex<Vec<AcquireRequest>>,
    attaches: Mutex<Vec<(String, Binding)>>,
}

impl FakeHooks {
    fn new() -> Self {
        FakeHooks {
            select_out: Mutex::new(Ok("conn-1".to_string())),
            acquire_out: Mutex::new(Err("no acquire".to_string())),
            attach_out: Mutex::new(Err("no attach".to_string())),
            end_calls: Mutex::new(Vec::new()),
            end_out: Mutex::new(Ok(())),
            authorize_out: Mutex::new(Ok(())),
            nested_out: Mutex::new(Ok(())),
            selects: Mutex::new(Vec::new()),
            acquires: Mutex::new(Vec::new()),
            attaches: Mutex::new(Vec::new()),
        }
    }
}

impl MuseHooks for FakeHooks {
    fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, String> {
        self.acquires.lock().unwrap().push(req.clone());
        self.acquire_out.lock().unwrap().clone()
    }
    fn attach(
        &self,
        lease_id: &str,
        binding: &Binding,
        _deadline: Instant,
    ) -> Result<Delivery, String> {
        self.attaches
            .lock()
            .unwrap()
            .push((lease_id.to_string(), binding.clone()));
        self.attach_out.lock().unwrap().clone()
    }
    fn end(&self, actor: i64, lease_id: &str, _deadline: Instant) -> Result<(), String> {
        self.end_calls
            .lock()
            .unwrap()
            .push((actor, lease_id.to_string()));
        self.end_out.lock().unwrap().clone()
    }
    fn authorize(&self, _actor: i64, _project: &str, _deadline: Instant) -> Result<(), String> {
        self.authorize_out.lock().unwrap().clone()
    }
    fn nested_authorize(
        &self,
        _actor: i64,
        _project: &str,
        _deadline: Instant,
    ) -> Result<(), String> {
        self.nested_out.lock().unwrap().clone()
    }
    fn select(
        &self,
        actor: i64,
        project: &str,
        selected: &str,
        _deadline: Instant,
    ) -> Result<String, String> {
        self.selects
            .lock()
            .unwrap()
            .push((actor, project.to_string(), selected.to_string()));
        self.select_out.lock().unwrap().clone()
    }
}

fn runtime(exec: FakeExec) -> MuseRuntime<FakeExec, FakeHooks> {
    MuseRuntime::new(exec, FakeHooks::new(), "1.0".to_string(), PIN.to_string())
}

fn caller() -> MuseCaller {
    MuseCaller {
        project: PID.to_string(),
        container: CID.to_string(),
        login: "dev".to_string(),
        home: "/home/dev".to_string(),
        actor: 7,
        uid: 1000,
        gid: 1000,
        ..Default::default()
    }
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

// ----- launch protocol -----

#[test]
fn launch_request_validate_matrix() {
    // Vectors from Go `TestLaunchRequestHasNoAuthority`.
    for req in [
        LaunchRequest {
            cwd: "relative".to_string(),
            ..Default::default()
        },
        LaunchRequest {
            cwd: "/workspace".to_string(),
            args: vec!["a\0b".to_string()],
            ..Default::default()
        },
        LaunchRequest {
            cwd: "/workspace".to_string(),
            tty: true,
            ..Default::default()
        },
        LaunchRequest {
            cwd: "/workspace".to_string(),
            config_home: "relative".to_string(),
            ..Default::default()
        },
        LaunchRequest {
            cwd: "/workspace".to_string(),
            register: Some(NestedRegistration {
                actor_id: 1,
                ..Default::default()
            }),
            ..Default::default()
        },
    ] {
        assert!(req.validate().is_err(), "{req:?}");
    }
    assert!(LaunchRequest {
        cwd: "/workspace".to_string(),
        args: vec![
            "--model".to_string(),
            "meta/test".to_string(),
            "prompt with spaces".to_string()
        ],
        ..Default::default()
    }
    .validate()
    .is_ok());
    // Limits.
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        args: vec!["x".repeat(32769)],
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        args: vec!["x".repeat(200); 200],
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        args: vec!["x".to_string(); 257],
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        term: "t".repeat(129),
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        connection_id: "c".repeat(129),
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        home: "relative".to_string(),
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        tty: true,
        cols: 80,
        rows: 0,
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        cwd: "/w".to_string(),
        tty: true,
        cols: 80,
        rows: 24,
        ..Default::default()
    }
    .validate()
    .is_ok());
    // Registration variant ignores CWD/args/connection/TTY presence rules.
    let reg = NestedRegistration {
        child_id: CID.to_string(),
        actor_id: 7,
        registration_id: TID.to_string(),
        muse: true,
    };
    assert!(LaunchRequest {
        register: Some(reg.clone()),
        ..Default::default()
    }
    .validate()
    .is_ok());
    assert!(LaunchRequest {
        register: Some(reg.clone()),
        tty: true,
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        register: Some(reg.clone()),
        args: vec!["x".to_string()],
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        register: Some(NestedRegistration {
            muse: false,
            ..reg.clone()
        }),
        ..Default::default()
    }
    .validate()
    .is_err());
    assert!(LaunchRequest {
        register: Some(NestedRegistration { actor_id: 0, ..reg }),
        ..Default::default()
    }
    .validate()
    .is_err());
}

#[test]
fn launch_decode_matrix() {
    let body = format!(
        "{{\"home\":\"/h\",\"register\":{{\"child_id\":{CID:?},\"actor_id\":\"7\",\"registration_id\":{TID:?},\"muse\":true}},\"config_home\":\"/c\",\"term\":\"xterm\",\"connection_id\":\"conn\",\"cwd\":\"/w\",\"args\":[\"a\",\"b\"],\"tty\":true,\"cols\":80,\"rows\":24}}"
    );
    let req = LaunchRequest::decode(body.as_bytes()).unwrap();
    assert_eq!(req.register.as_ref().unwrap().actor_id, 7);
    assert_eq!((req.cols, req.rows), (80, 24));
    assert!(req.tty);
    let negative_zero = LaunchRequest::decode(br#"{"cols":-0,"rows":-0}"#).unwrap();
    assert_eq!((negative_zero.cols, negative_zero.rows), (0, 0));
    let negative_zero_control = LaunchControl::decode(br#"{"signal":-0}"#).unwrap();
    assert_eq!(negative_zero_control.signal, 0);
    assert!(LaunchRequest::decode(br#"{"cols":1.0}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"rows":1e0}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"cols":70000}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"cols":-1}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"bogus":1}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"register":{"actor_id":""}}"#).is_err());
    assert!(LaunchRequest::decode(br#"{"register":null}"#)
        .unwrap()
        .register
        .is_none());
    assert_eq!(
        LaunchRequest::decode(br#"{}"#).unwrap(),
        LaunchRequest::default()
    );
    let control = LaunchControl::decode(br#"{"signal":15}"#).unwrap();
    assert_eq!(
        control,
        LaunchControl {
            signal: 15,
            cols: 0,
            rows: 0
        }
    );
    assert!(LaunchControl::decode(br#"{"cols":80,"rows":24,"bogus":1}"#).is_err());
    assert!(LaunchControl::decode(br#"{"rows":65536}"#).is_err());
    assert_eq!(
        LaunchExit::denied().encode(),
        r#"{"code":1,"error":"Muse launch denied"}"#
    );
    assert_eq!(
        LaunchExit::cleanup_unconfirmed().encode(),
        r#"{"code":1,"error":"Muse cleanup unconfirmed"}"#
    );
    assert_eq!(
        LaunchExit {
            code: 0,
            error: String::new()
        }
        .encode(),
        r#"{"code":0}"#
    );
    assert!(LaunchExit::denied().encode_line().ends_with('\n'));
}

#[test]
fn muse_arguments_matrix() {
    let args = |list: &[&str]| list.iter().map(|s| s.to_string()).collect::<Vec<_>>();
    // Vectors from Go `TestMuseNativeCommandProviderPlacement`.
    assert_eq!(
        muse_arguments(&args(&[
            "exec",
            "--model",
            "native/model",
            "prompt with spaces"
        ]))
        .unwrap(),
        args(&[
            "exec",
            "--provider",
            "meta",
            "--model",
            "native/model",
            "prompt with spaces"
        ])
    );
    assert_eq!(
        muse_arguments(&args(&["prompt"])).unwrap(),
        args(&["--provider", "meta", "prompt"])
    );
    assert_eq!(
        muse_arguments(&args(&["config", "--help"])).unwrap(),
        args(&["config", "--help"])
    );
    assert_eq!(
        muse_arguments(&args(&[])).unwrap(),
        args(&["--provider", "meta"])
    );
    // Auth surface denied.
    for denied in [
        vec!["auth"],
        vec!["login"],
        vec!["logout"],
        vec!["--model", "m", "auth"],
        vec!["--provider", "evil"],
        vec!["exec", "--base-url=https://evil.example"],
    ] {
        assert_eq!(
            muse_arguments(&args(&denied)).unwrap_err(),
            terminal::ERR_DENIED,
            "{denied:?}"
        );
    }
    // Value flags hide their operand from the positional scan.
    assert!(muse_arguments(&args(&["--model", "auth"])).is_ok());
    assert!(muse_arguments(&args(&["--model"])).is_ok());
    assert_eq!(
        muse_arguments(&args(&["resume", "abc"])).unwrap(),
        args(&["resume", "--provider", "meta", "abc"])
    );
}

#[test]
fn connection_directory_matrix() {
    let ready = MuseConnection {
        id: "a".to_string(),
        provider_id: "muse".to_string(),
        state: "ready".to_string(),
    };
    let reauth = MuseConnection {
        id: "b".to_string(),
        provider_id: "muse".to_string(),
        state: "reauth".to_string(),
    };
    let codex = MuseConnection {
        id: "c".to_string(),
        provider_id: "codex".to_string(),
        state: "ready".to_string(),
    };
    assert!(muse_connection_authorized(&[reauth.clone(), codex.clone(), ready.clone()]).is_ok());
    assert_eq!(
        muse_connection_authorized(&[reauth, codex]).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(
        muse_connection_authorized(&[]).unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(
        select_muse_connection(std::slice::from_ref(&ready), "").unwrap(),
        "a"
    );
    let two = vec![
        ready.clone(),
        MuseConnection {
            id: "z".to_string(),
            provider_id: "muse".to_string(),
            state: "ready".to_string(),
        },
    ];
    assert_eq!(select_muse_connection(&two, "z").unwrap(), "z");
    assert_eq!(
        select_muse_connection(&two, "").unwrap_err(),
        "choose an authorized muse connection with SODA_MUSE_CONNECTION"
    );
    assert_eq!(
        select_muse_connection(&two, "absent").unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(
        select_muse_connection(&[], "").unwrap_err(),
        terminal::ERR_DENIED
    );
}

// ----- pure resolution helpers -----

#[test]
fn cgroup_matrix() {
    assert_eq!(
        muse_project_cgroup(&format!(
            "0::/user.slice/user-1000.slice/session-1.scope/libpod-{CID}.scope"
        ))
        .unwrap(),
        CID
    );
    assert_eq!(
        muse_project_cgroup(&format!("12:memory:/kubepods/libpod-{CID}.scope")).unwrap(),
        CID
    );
    assert!(muse_project_cgroup("0::/init.scope").is_err());
    assert!(muse_project_cgroup(&format!("0::/libpod-{CID}.scope.extra")).is_err());
    assert!(muse_project_cgroup(
        "0::/libpod-0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF.scope"
    )
    .is_err());
    assert!(muse_project_cgroup("").is_err());
}

#[test]
fn uidmap_matrix() {
    assert_eq!(
        muse_mapped_uid("         0          1      65536\n", 1).unwrap(),
        0
    );
    assert_eq!(muse_mapped_uid("0 100000 262144\n", 100500).unwrap(), 500);
    assert_eq!(
        muse_mapped_uid("0 1 1\n1 100000 65536\n", 100000).unwrap(),
        1
    );
    assert!(muse_mapped_uid("0 100000 262144\n", 500000).is_err());
    assert!(muse_mapped_uid("0 0 10\n", 5).is_err()); // outside 0 skipped
    assert!(muse_mapped_uid("garbage\n", 1).is_err());
    assert!(muse_mapped_uid("", 1).is_err());
}

#[test]
fn passwd_and_modes_matrix() {
    let row = |parts: &[&str]| parts.iter().map(|s| s.to_string()).collect::<Vec<_>>();
    assert!(muse_passwd_valid(
        &row(&["dev", "x", "1000", "1000", "", "/home/dev", "/bin/sh"]),
        1000
    ));
    assert!(!muse_passwd_valid(
        &row(&["root", "x", "0", "0", "", "/root", "/bin/sh"]),
        0
    ));
    assert!(!muse_passwd_valid(
        &row(&["dev", "x", "1001", "1000", "", "/home/dev", "/bin/sh"]),
        1000
    ));
    assert!(!muse_passwd_valid(
        &row(&["dev", "x", "1000", "1000", "", "home/dev", "/bin/sh"]),
        1000
    ));
    assert!(!muse_passwd_valid(&row(&["dev", "x", "1000"]), 1000));
    assert!(!muse_passwd_valid(
        &row(&["Dev", "x", "1000", "1000", "", "/home/dev", "/bin/sh"]),
        1000
    ));
    let good = "0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:600:regular file\n";
    assert!(muse_account_modes(good));
    assert!(!muse_account_modes(
        "0:0:755:directory\n0:0:600:regular file\n"
    ));
    assert!(!muse_account_modes(
        &good.replace("0:0:600:regular file", "0:0:644:regular file")
    ));
    assert!(!muse_account_modes(
        &good.replace("0:0:600:regular file", "0:0:600:directory")
    ));
    assert!(!muse_account_modes(
        &good.replace("0:0:755:directory", "1:0:755:directory")
    ));
    assert!(!muse_account_modes(
        &good.replace("0:0:755:directory", "0:0:775:directory")
    ));
}

#[test]
fn registration_and_child_matrix() {
    let reg = NestedRegistration {
        child_id: CID.to_string(),
        actor_id: 7,
        registration_id: TID.to_string(),
        muse: true,
    };
    assert!(muse_registration_valid(&reg));
    assert!(!muse_registration_valid(&NestedRegistration {
        muse: false,
        ..reg.clone()
    }));
    assert!(!muse_registration_valid(&NestedRegistration {
        actor_id: 0,
        ..reg.clone()
    }));
    assert!(!muse_registration_valid(&NestedRegistration {
        child_id: "short".to_string(),
        ..reg
    }));
    assert_eq!(
        muse_child_pid(
            format!("{{\"id\":{CID:?},\"pid\":4242,\"running\":true}}").as_bytes(),
            false,
            CID
        )
        .unwrap(),
        4242
    );
    assert!(muse_child_pid(b"{}", true, CID).is_err());
    assert!(muse_child_pid(&vec![b'x'; 4097], false, CID).is_err());
    assert!(muse_child_pid(
        format!("{{\"id\":{CID:?},\"pid\":0,\"running\":true}}").as_bytes(),
        false,
        CID
    )
    .is_err());
    assert!(muse_child_pid(
        format!("{{\"id\":{CID:?},\"pid\":42,\"running\":false}}").as_bytes(),
        false,
        CID
    )
    .is_err());
    assert!(muse_child_pid(br#"{"id":"other","pid":42,"running":true}"#, false, CID).is_err());
    assert!(muse_child_pid(b"nope", false, CID).is_err());
}

#[test]
fn readonly_mount_matrix() {
    let src = "/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    let good =
        format!(r#"[{{"Source":{src:?},"Destination":"/run/soda-muse/credentials","RW":false}}]"#);
    assert!(muse_readonly_mount(good.as_bytes(), false, src, "/run/soda-muse/credentials").is_ok());
    // Decoded tolerantly: unknown fields and ordering do not matter.
    let extra = format!(
        r#"[{{"RW":false,"Extra":1,"Source":{src:?},"Destination":"/run/soda-muse/credentials"}}]"#
    );
    assert!(
        muse_readonly_mount(extra.as_bytes(), false, src, "/run/soda-muse/credentials").is_ok()
    );
    let rw =
        format!(r#"[{{"Source":{src:?},"Destination":"/run/soda-muse/credentials","RW":true}}]"#);
    assert!(muse_readonly_mount(rw.as_bytes(), false, src, "/run/soda-muse/credentials").is_err());
    assert!(muse_readonly_mount(br#"[]"#, false, src, "/run/soda-muse/credentials").is_err());
    assert!(muse_readonly_mount(good.as_bytes(), true, src, "/run/soda-muse/credentials").is_err());
    assert!(muse_readonly_mount(b"nope", false, src, "/run/soda-muse/credentials").is_err());
}

fn elf_header(machine: u16) -> Vec<u8> {
    let mut header = vec![0u8; 64];
    header[0..4].copy_from_slice(&[0x7f, b'E', b'L', b'F']);
    header[4] = 2;
    header[5] = 1;
    header[18..20].copy_from_slice(&machine.to_le_bytes());
    header
}

#[test]
fn elf_matrix() {
    assert!(muse_elf(&elf_header(62), "amd64").is_ok());
    assert!(muse_elf(&elf_header(183), "arm64").is_ok());
    assert!(muse_elf(&elf_header(183), "amd64").is_err());
    assert!(muse_elf(&elf_header(62), "arm64").is_err());
    assert!(muse_elf(&elf_header(62)[..63], "amd64").is_err());
    assert!(muse_elf(&[0u8; 64], "amd64").is_err());
    let mut bad = elf_header(62);
    bad[4] = 1;
    assert!(muse_elf(&bad, "amd64").is_err());
    assert!(["amd64", "arm64", "unknown"].contains(&host_go_arch()));
}

#[test]
fn signal_and_root_matrix() {
    for sig in [1, 2, 3, 10, 12, 15, 18, 20] {
        assert!(muse_signal(sig), "{sig}");
    }
    for sig in [0, 9, 19, 64, 65, -1, 1 << 40] {
        assert!(!muse_signal(sig), "{sig}");
    }
    let lease = lease_fixture();
    let binding = lease.binding.clone().unwrap();
    assert!(muse_project_credential_root(&binding));
    let mut nested = binding.clone();
    nested.child_id = CID.to_string();
    nested.credential_root = format!("/run/soda-muse/nested/{IID}/{TID}");
    assert!(muse_project_credential_root(&nested));
    for root in [
        format!("/run/soda-muse/{TID}/../escape"),
        format!("/run/soda-muse//{TID}"),
        format!("/run/soda-muse/./{TID}"),
        format!("/run/soda-muse/{TID}/"),
        "/run/other/x".to_string(),
        "/run/soda-muse/other".to_string(),
        format!("/run/soda-muse/nested/{IID}"),
    ] {
        let mut bad = binding.clone();
        bad.credential_root = root;
        assert!(
            !muse_project_credential_root(&bad),
            "{:?}",
            bad.credential_root
        );
    }
    assert!(muse_delivery_valid(&Delivery {
        lease: lease.clone(),
        credential: None
    }));
    assert!(!muse_delivery_valid(&Delivery::default()));
    let mut bad = lease.clone();
    bad.provider_id = "codex".to_string();
    assert!(!muse_delivery_valid(&Delivery {
        lease: bad,
        credential: None
    }));
}

// ----- argv goldens (byte-exact vs Go `museCommand`) -----

#[test]
fn muse_command_goldens() {
    let request = LaunchRequest {
        term: "xterm-256color".to_string(),
        cwd: "/workspace".to_string(),
        args: vec![
            "--provider".to_string(),
            "meta".to_string(),
            "prompt".to_string(),
        ],
        tty: true,
        cols: 80,
        rows: 24,
        ..Default::default()
    };
    let unit = "soda-muse-0123456789abcdef0123456789abcdef.service";
    let path = "/run/soda-muse/0123456789abcdef0123456789abcdef";
    let mut want = vec!["/usr/bin/podman".to_string()];
    want.extend(muse_command_argv(&caller(), &request, unit, path));
    assert_eq!(
        want,
        [
            "/usr/bin/podman", "--remote=false", "exec", "--interactive", "--tty", CID,
            "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec",
            "--unit=soda-muse-0123456789abcdef0123456789abcdef.service",
            "--property=KillMode=control-group", "--property=TimeoutStopSec=10",
            "--property=RuntimeMaxSec=43200", "--property=UMask=0077",
            "--setenv=HOME=/home/dev", "--setenv=USER=dev", "--setenv=LOGNAME=dev",
            "--setenv=TERM=xterm-256color",
            "--setenv=XDG_CONFIG_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/config",
            "--setenv=XDG_STATE_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/state",
            "--setenv=XDG_CACHE_HOME=/run/soda-muse/0123456789abcdef0123456789abcdef/cache",
            "--setenv=TBH_CREDENTIAL_BACKEND=file", "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
            "--uid=dev", "--gid=1000", "--working-directory=/workspace",
            "--property=ReadOnlyPaths=/run/soda-muse/0123456789abcdef0123456789abcdef/auth.json",
            "--property=BindReadOnlyPaths=/run/soda-muse/0123456789abcdef0123456789abcdef/auth.json:/run/soda-muse/0123456789abcdef0123456789abcdef/config/muse/auth.json",
            "--pty", "--", "/usr/local/bin/muse", "--soda-exec",
            "/run/soda-muse/0123456789abcdef0123456789abcdef", "/workspace",
            "--provider", "meta", "prompt",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>()
    );
    // Nested variant: no uid/gid/workdir/mounts, nsenter chain, remapped config path.
    let mut nested = caller();
    nested.child = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210".to_string();
    nested.registration = IID.to_string();
    nested.nested_pid = 4242;
    let nested_request = LaunchRequest {
        term: "dumb".to_string(),
        cwd: "/work".to_string(),
        args: vec![
            "exec".to_string(),
            "--provider".to_string(),
            "meta".to_string(),
        ],
        ..Default::default()
    };
    let nested_argv = muse_command_argv(
        &nested,
        &nested_request,
        "soda-muse-bbbb.service",
        "/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb",
    );
    let mut want_nested = vec!["/usr/bin/podman".to_string()];
    want_nested.extend(nested_argv.clone());
    assert_eq!(
        want_nested,
        [
            "/usr/bin/podman", "--remote=false", "exec", "--interactive", CID,
            "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec",
            "--unit=soda-muse-bbbb.service", "--property=KillMode=control-group",
            "--property=TimeoutStopSec=10", "--property=RuntimeMaxSec=43200",
            "--property=UMask=0077", "--setenv=HOME=/home/dev", "--setenv=USER=dev",
            "--setenv=LOGNAME=dev", "--setenv=TERM=dumb",
            "--setenv=XDG_CONFIG_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/config",
            "--setenv=XDG_STATE_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/state",
            "--setenv=XDG_CACHE_HOME=/run/soda-muse/nested/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbb/cache",
            "--setenv=TBH_CREDENTIAL_BACKEND=file", "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
            "--pipe", "--", "/usr/bin/nsenter", "--target=4242", "--mount", "--pid", "--uts",
            "--ipc", "--net", "--root", "--setuid=1000", "--setgid=1000", "--",
            "/usr/local/bin/muse", "--soda-exec", "/run/soda-muse/credentials/bbbb", "/work",
            "exec", "--provider", "meta",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>()
    );
    // Explicit HOME wins over the caller home.
    let home_request = LaunchRequest {
        home: "/custom/home".to_string(),
        ..request.clone()
    };
    let home_argv = muse_command_argv(
        &caller(),
        &home_request,
        "soda-muse-cccc.service",
        "/run/soda-muse/cccc",
    );
    assert!(home_argv.contains(&"--setenv=HOME=/custom/home".to_string()));
    assert!(!home_argv.contains(&"--setenv=HOME=/home/dev".to_string()));
}

// ----- exec flows -----

#[test]
fn inspect_flows() {
    let good = format!(
        "{{\"id\":{CID:?},\"pid\":1234,\"project\":{PID:?},\"running\":true,\"privileged\":false,\"userns\":\"private\"}}"
    );
    let rt = runtime(FakeExec::new(vec![ok(&good)]));
    let out = rt.inspect(CID, deadline()).unwrap();
    assert_eq!(
        (out.project, out.project_pid, out.container),
        (PID.to_string(), 1234, CID.to_string())
    );
    assert_eq!(rt.exec.calls()[0].2[3], MUSE_INSPECT);
    // userns is NOT gated here (Go checks only labels/privilege).
    let host_userns = good.replace("\"userns\":\"private\"", "\"userns\":\"host\"");
    let rt = runtime(FakeExec::new(vec![ok(&host_userns)]));
    assert!(rt.inspect(CID, deadline()).is_ok());
    for body in [
        good.replace("\"running\":true", "\"running\":false"),
        good.replace("\"pid\":1234", "\"pid\":0"),
        good.replace(PID, "nope"),
        good.replace("\"privileged\":false", "\"privileged\":true"),
        good.replace(CID, &"f".repeat(64)),
        "{nope".to_string(),
    ] {
        let rt = runtime(FakeExec::new(vec![ok(&body)]));
        assert_eq!(
            rt.inspect(CID, deadline()).unwrap_err(),
            terminal::ERR_DENIED,
            "{body}"
        );
    }
    let rt = runtime(FakeExec::new(vec![err("boom")]));
    assert_eq!(
        rt.inspect(CID, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    let rt = runtime(FakeExec::new(vec![Ok(vec![b'x'; 4097])]));
    assert_eq!(
        rt.inspect(CID, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
}

#[test]
fn actor_account_flows() {
    let rt = runtime(FakeExec::new(vec![ok(
        r#"{"Username":"dev","Uid":1000,"Gid":1000,"HomeDir":"/home/dev"}"#,
    )]));
    assert_eq!(
        rt.actor_account(CID, 7, deadline()).unwrap(),
        ("dev".to_string(), "/home/dev".to_string())
    );
    let calls = rt.exec.calls();
    assert_eq!(
        calls[0].2[4..],
        [
            "/usr/local/bin/muse".to_string(),
            "--soda-account".to_string(),
            "7".to_string()
        ]
    );
    let rt = runtime(FakeExec::new(vec![ok(
        r#"{"Username":"root","HomeDir":"/root"}"#,
    )]));
    assert_eq!(
        rt.actor_account(CID, 7, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    let rt = runtime(FakeExec::new(vec![ok("nope")]));
    assert_eq!(
        rt.actor_account(CID, 7, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    let rt = runtime(FakeExec::new(vec![err("boom")]));
    assert_eq!(
        rt.actor_account(CID, 7, deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
}

#[test]
fn reserve_execution_flows() {
    let rt = runtime(FakeExec::new(vec![]));
    *rt.hooks.acquire_out.lock().unwrap() = Ok(Lease {
        id: "lease-new".to_string(),
        generation: 4,
        ..Default::default()
    });
    let request = LaunchRequest {
        cwd: "/workspace".to_string(),
        connection_id: "conn-9".to_string(),
        ..Default::default()
    };
    let execution = rt
        .reserve_execution(&caller(), &request, deadline())
        .unwrap();
    assert_eq!(execution.lease.id, "lease-new");
    assert_eq!(execution.binding.generation, 4);
    assert_eq!(execution.binding.project, CID);
    assert_eq!(execution.binding.scope, terminal::SCOPE_MUSE_PROJECT);
    assert_eq!(execution.binding.credential_root, execution.path);
    assert!(execution.path.starts_with("/run/soda-muse/"));
    assert_eq!(
        execution.unit,
        format!("soda-muse-{}.service", execution.binding.id)
    );
    assert_eq!(execution.binding.id.len(), 32);
    {
        // Scoped: the nested reserve below re-locks `acquires`.
        let acquires = rt.hooks.acquires.lock().unwrap();
        assert_eq!(acquires.len(), 1);
        assert_eq!(acquires[0].provider_id, "muse");
        assert_eq!(acquires[0].actor_id, 7);
        assert_eq!(acquires[0].connection_id, "conn-1");
        assert_eq!(acquires[0].project_id, PID);
        assert_eq!(acquires[0].kind, "terminal");
        assert!((acquires[0].deadline_secs - (terminal::now_unix() + 12 * 3600)).abs() <= 5);
    }
    assert_eq!(
        rt.hooks.selects.lock().unwrap()[0],
        (7, PID.to_string(), "conn-9".to_string())
    );
    // Nested path nests the credential root.
    let mut nested = caller();
    nested.child = CID.to_string();
    nested.registration = IID.to_string();
    let execution = rt.reserve_execution(&nested, &request, deadline()).unwrap();
    assert!(execution
        .path
        .starts_with(&format!("/run/soda-muse/nested/{IID}/")));
    assert_eq!(execution.binding.child_id, CID);
    // Select failure propagates.
    *rt.hooks.select_out.lock().unwrap() = Err(terminal::ERR_DENIED.to_string());
    assert_eq!(
        rt.reserve_execution(&caller(), &request, deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
}

#[test]
fn stage_flows() {
    let path = format!("/run/soda-muse/{TID}");
    let rt = runtime(FakeExec::new(vec![ok(""); 10]));
    rt.stage(&caller(), &path, "", deadline()).unwrap();
    let calls = rt.exec.calls();
    assert_eq!(calls.len(), 10);
    let guest_tail = |call: &(Vec<u8>, String, Vec<String>)| call.2[4..].to_vec();
    assert_eq!(
        guest_tail(&calls[0]),
        ["/usr/bin/install", "--directory", "--mode=0700", &path]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
    );
    assert_eq!(
        guest_tail(&calls[1]),
        [
            "/usr/bin/mount",
            "--types=tmpfs",
            "--options=mode=0700,size=4M",
            "tmpfs",
            &path
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>()
    );
    assert!(calls[2].2[9].ends_with("/config/muse"));
    assert_eq!(
        &calls[2].2[7..9],
        &["--owner=1000".to_string(), "--group=1000".to_string()]
    );
    assert_eq!(
        calls[3].2[4..],
        [
            "/usr/bin/chmod".to_string(),
            "0711".to_string(),
            path.clone()
        ]
    );
    assert_eq!(calls[4].0, b"{}");
    assert!(calls[4].2[5].ends_with("/auth.json"));
    assert_eq!(
        calls[5].2[4..],
        [
            "/usr/bin/chown".to_string(),
            "1000:1000".to_string(),
            format!("{path}/auth.json")
        ]
    );
    assert_eq!(
        calls[6].2[4..],
        [
            "/usr/bin/chmod".to_string(),
            "0600".to_string(),
            format!("{path}/auth.json")
        ]
    );
    // Config copy runs without --interactive via plain podman exec.
    assert_eq!(calls[7].1, "/usr/bin/podman");
    assert_eq!(
        calls[7].2,
        [
            "--remote=false",
            "exec",
            "--user=1000:1000",
            CID,
            "/usr/local/bin/muse",
            "--soda-copy-config",
            "/home/dev/.config/muse",
            &format!("{path}/config/muse"),
        ]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>()
    );
    assert_eq!(
        calls[8].2[4..],
        [
            "/usr/bin/chmod".to_string(),
            "0711".to_string(),
            format!("{path}/config")
        ]
    );
    assert_eq!(
        calls[9].2[4..],
        [
            "/usr/bin/touch".to_string(),
            format!("{path}/config/muse/auth.json")
        ]
    );
    // Explicit config home wins.
    let rt = runtime(FakeExec::new(vec![ok(""); 10]));
    rt.stage(&caller(), &path, "/custom/config", deadline())
        .unwrap();
    assert_eq!(rt.exec.calls()[7].2[6], "/custom/config/muse");
    // Nested stage skips the tmpfs mount and reads config through nsenter.
    let mut nested = caller();
    nested.child = CID.to_string();
    nested.registration = IID.to_string();
    nested.nested_pid = 4242;
    let nested_path = format!("/run/soda-muse/nested/{IID}/{TID}");
    let rt = runtime(FakeExec::new(vec![
        ok(""),                                           // install dir
        ok(""),                                           // install config dir
        ok(""),                                           // chmod path
        ok(""),                                           // dd auth
        ok(""),                                           // chown auth
        ok(""),                                           // chmod auth
        ok(r#"{"settings.json":"e30=","other":"e30="}"#), // nsenter read-config
        ok(""),                                           // dd settings
        ok(""),                                           // chown settings
        ok(""),                                           // chmod settings
        ok(""),                                           // chmod config
        ok(""),                                           // ln auth target
    ]));
    rt.stage(&nested, &nested_path, "", deadline()).unwrap();
    let calls = rt.exec.calls();
    assert_eq!(calls.len(), 12);
    assert_eq!(calls[6].2[4], "/usr/bin/nsenter");
    assert!(calls[6].2.contains(&"--target=4242".to_string()));
    assert_eq!(calls[7].0, b"{}");
    assert!(calls[7].2[5].ends_with("/config/muse/settings.json"));
    assert_eq!(
        calls[10].2[4..],
        [
            "/usr/bin/chmod".to_string(),
            "0711".to_string(),
            format!("{nested_path}/config")
        ]
    );
    assert_eq!(
        calls[11].2[4..],
        [
            "/usr/bin/ln".to_string(),
            "--symbolic".to_string(),
            "../../auth.json".to_string(),
            format!("{nested_path}/config/muse/auth.json")
        ]
    );
    // Mount failure propagates raw.
    let rt = runtime(FakeExec::new(vec![ok(""), err("exit status 32")]));
    assert_eq!(
        rt.stage(&caller(), &path, "", deadline()).unwrap_err(),
        "exit status 32"
    );
}

#[test]
fn deliver_execution_flows() {
    let lease = lease_fixture();
    let mut execution = MuseExecution {
        caller: caller(),
        request: LaunchRequest::default(),
        lease: lease.clone(),
        binding: lease.binding.clone().unwrap(),
        path: format!("/run/soda-muse/{TID}"),
        unit: format!("soda-muse-{TID}.service"),
    };
    execution.binding.invocation_id = String::new();
    let rt = runtime(FakeExec::new(vec![
        ok("active\n"),
        ok(&format!("{IID}\n")),
        ok(""),
        ok(""),
    ]));
    *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
        lease: lease.clone(),
        credential: Some(b"{\"k\":1}".to_vec()),
    });
    rt.deliver_execution(&mut execution, deadline()).unwrap();
    assert_eq!(execution.binding.invocation_id, IID);
    let calls = rt.exec.calls();
    assert_eq!(calls.len(), 4);
    assert_eq!(calls[2].0, b"{\"k\":1}");
    assert!(calls[2].2[5].ends_with("/auth.json"));
    assert!(calls[3].2[5].ends_with("/ready"));
    let attaches = rt.hooks.attaches.lock().unwrap();
    assert_eq!(attaches[0].0, "lease-m");
    assert_eq!(attaches[0].1.invocation_id, IID);
    // Malformed invocation denies before attach.
    let mut execution = execution.clone();
    execution.binding.invocation_id = String::new();
    let rt = runtime(FakeExec::new(vec![ok("active\n"), ok("short\n")]));
    assert_eq!(
        rt.deliver_execution(&mut execution, deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.hooks.attaches.lock().unwrap().is_empty());
    // Invalid credential denies after attach.
    let mut execution = execution.clone();
    execution.binding.invocation_id = String::new();
    let rt = runtime(FakeExec::new(vec![ok("active\n"), ok(&format!("{IID}\n"))]));
    *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
        lease: lease.clone(),
        credential: Some(b"nope".to_vec()),
    });
    assert_eq!(
        rt.deliver_execution(&mut execution, deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    // dd failure propagates raw.
    let mut execution = execution.clone();
    execution.binding.invocation_id = String::new();
    let rt = runtime(FakeExec::new(vec![
        ok("active\n"),
        ok(&format!("{IID}\n")),
        err("exit status 1"),
    ]));
    *rt.hooks.attach_out.lock().unwrap() = Ok(Delivery {
        lease: lease.clone(),
        credential: Some(b"{}".to_vec()),
    });
    assert_eq!(
        rt.deliver_execution(&mut execution, deadline())
            .unwrap_err(),
        "exit status 1"
    );
    // A unit that never activates denies (tight deadline, no 5s wait).
    let mut execution = execution.clone();
    execution.binding.invocation_id = String::new();
    let rt = runtime(FakeExec::new(vec![ok("inactive\n")]));
    let past = Instant::now() - Duration::from_secs(1);
    assert!(rt.deliver_execution(&mut execution, past).is_err());
}

#[test]
fn stop_execution_flows() {
    let lease = lease_fixture();
    let binding = lease.binding.clone().unwrap();
    // Full stop with custody: stop, inactive, exists, rm state,
    // test state, umount, rm path.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    rt.stop_execution(&binding, 7, "lease-m", true, deadline())
        .unwrap();
    let calls = rt.exec.calls();
    assert_eq!(calls.len(), 7);
    assert_eq!(
        calls[0].2[4..],
        [
            "/usr/bin/systemctl".to_string(),
            "stop".to_string(),
            format!("soda-muse-{TID}.service")
        ]
    );
    assert_eq!(
        calls[1].2[4..8],
        [
            "/usr/bin/systemctl".to_string(),
            "show".to_string(),
            "--property=ActiveState".to_string(),
            "--value".to_string()
        ]
    );
    assert_eq!(
        calls[2].2[1..4],
        [
            "container".to_string(),
            "exists".to_string(),
            CID.to_string()
        ]
    );
    assert!(calls[3].2.contains(&"--user=1000:1000".to_string()));
    assert_eq!(
        calls[5].2[4..],
        [
            "/usr/bin/umount".to_string(),
            format!("/run/soda-muse/{TID}")
        ]
    );
    assert_eq!(
        rt.hooks.end_calls.lock().unwrap().as_slice(),
        &[(7, "lease-m".to_string())]
    );
    // Unit still active is uncertain.
    let rt = runtime(FakeExec::new(vec![ok(""), ok("active\n")]));
    assert_eq!(
        rt.stop_execution(&binding, 7, "lease-m", true, deadline())
            .unwrap_err(),
        terminal::ERR_UNCERTAIN
    );
    // Failed stop with a gone cgroup still retires.
    let rt = runtime(FakeExec::new(vec![
        err("exit status 1"),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    rt.stop_execution(&binding, 7, "lease-m", false, deadline())
        .unwrap();
    assert_eq!(rt.exec.calls().len(), 8);
    assert!(rt.hooks.end_calls.lock().unwrap().is_empty());
    // Failed stop with a live cgroup is uncertain.
    let rt = runtime(FakeExec::new(vec![
        err("exit status 1"),
        ok("inactive\n"),
        err("exit status 1"),
    ]));
    assert_eq!(
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap_err(),
        terminal::ERR_UNCERTAIN
    );
    // Removed container skips the rm/test pair.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        err("exit status 1"),
        ok(""),
        ok(""),
    ]));
    rt.stop_execution(&binding, 7, "lease-m", false, deadline())
        .unwrap();
    assert_eq!(rt.exec.calls().len(), 5);
    // Retire failure is uncertain.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        err("exit status 1"),
    ]));
    assert_eq!(
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap_err(),
        terminal::ERR_UNCERTAIN
    );
    // Nested bindings skip the tmpfs unmount and wrap state podman.
    let mut nested = binding.clone();
    nested.child_id = "f".repeat(64);
    nested.credential_root = format!("/run/soda-muse/nested/{IID}/{TID}");
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        ok(""),
    ]));
    rt.stop_execution(&nested, 7, "lease-m", false, deadline())
        .unwrap();
    let calls = rt.exec.calls();
    assert_eq!(calls.len(), 6);
    assert_eq!(
        calls[2].2[4..8],
        [
            "/usr/bin/podman".to_string(),
            "--remote=false".to_string(),
            "container".to_string(),
            "exists".to_string()
        ]
    );
    assert_eq!(calls[2].2[8], "f".repeat(64));
    assert!(!calls
        .iter()
        .any(|c| c.2.contains(&"/usr/bin/umount".to_string())));
}

#[test]
fn retire_mount_matrix() {
    let lease = lease_fixture();
    let binding = lease.binding.clone().unwrap();
    // umount failure + vanished path retires.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        err("exit status 32"),
        ok(""),
        ok(""),
    ]));
    rt.stop_execution(&binding, 7, "lease-m", false, deadline())
        .unwrap();
    assert_eq!(rt.exec.calls().len(), 8);
    // umount + test failures + mountpoint exit 32 retires.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        err("exit status 1"),
        err("exit status 1"),
        err("exit status 32"),
        ok(""),
    ]));
    rt.stop_execution(&binding, 7, "lease-m", false, deadline())
        .unwrap();
    assert_eq!(rt.exec.calls().len(), 9);
    // A live mountpoint (exit 0) is uncertain.
    let rt = runtime(FakeExec::new(vec![
        ok(""),
        ok("inactive\n"),
        ok(""),
        ok(""),
        ok(""),
        err("exit status 1"),
        err("exit status 1"),
        ok(""),
    ]));
    assert_eq!(
        rt.stop_execution(&binding, 7, "lease-m", false, deadline())
            .unwrap_err(),
        terminal::ERR_UNCERTAIN
    );
    assert_eq!(rt.exec.calls().len(), 8);
}

#[test]
fn validate_and_ops_matrix() {
    let lease = lease_fixture();
    let binding = lease.binding.clone().unwrap();
    let rt = runtime(FakeExec::new(vec![ok("active\n")]));
    assert!(rt.validate_muse_binding(&binding, deadline()).is_ok());
    let rt = runtime(FakeExec::new(vec![ok("inactive\n")]));
    assert_eq!(
        rt.validate_muse_binding(&binding, deadline()).unwrap_err(),
        terminal::ERR_STALE
    );
    let rt = runtime(FakeExec::new(vec![err("boom")]));
    assert_eq!(
        rt.validate_muse_binding(&binding, deadline()).unwrap_err(),
        terminal::ERR_STALE
    );
    let rt = runtime(FakeExec::new(vec![]));
    assert_eq!(
        rt.validate_muse_binding(&Binding::default(), deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.exec.calls().is_empty());
    // Ops dispatch.
    let delivery = Delivery {
        lease: lease.clone(),
        credential: None,
    };
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n")), ok("active\n")]));
    assert_eq!(
        rt.muse_operation("validate", &delivery, deadline())
            .unwrap(),
        delivery
    );
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{IID}\n")),
        ok(""),
        ok("inactive\n"),
        err("exit status 1"),
        ok(""),
        ok(""),
    ]));
    assert_eq!(
        rt.muse_operation("stop", &delivery, deadline()).unwrap(),
        delivery
    );
    assert!(rt.hooks.end_calls.lock().unwrap().is_empty()); // no custody on broker stop
                                                            // Invocation mismatch is stale, checked before the action.
    let rt = runtime(FakeExec::new(vec![ok(&format!("{}\n", "b".repeat(32)))]));
    assert_eq!(
        rt.muse_operation("validate", &delivery, deadline())
            .unwrap_err(),
        terminal::ERR_STALE
    );
    // Empty observation is tolerated (unit may be gone).
    let rt = runtime(FakeExec::new(vec![ok("\n"), ok("active\n")]));
    assert!(rt.muse_operation("validate", &delivery, deadline()).is_ok());
    // Unknown actions deny after the invocation check.
    let rt = runtime(FakeExec::new(vec![ok(&format!("{IID}\n"))]));
    assert_eq!(
        rt.muse_operation("launch", &delivery, deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert_eq!(rt.exec.calls().len(), 1);
    // Malformed deliveries never call out.
    let rt = runtime(FakeExec::new(vec![]));
    assert_eq!(
        rt.muse_operation("validate", &Delivery::default(), deadline())
            .unwrap_err(),
        terminal::ERR_DENIED
    );
    let mut other = lease.clone();
    other.binding.as_mut().unwrap().scope = "other".to_string();
    assert_eq!(
        rt.muse_operation(
            "validate",
            &Delivery {
                lease: other,
                credential: None
            },
            deadline()
        )
        .unwrap_err(),
        terminal::ERR_DENIED
    );
    assert!(rt.exec.calls().is_empty());
}

#[test]
fn state_container_matrix() {
    let lease = lease_fixture();
    let binding = lease.binding.clone().unwrap();
    assert_eq!(state_container(&binding).unwrap(), CID);
    let mut nested = binding.clone();
    nested.child_id = "f".repeat(64);
    assert_eq!(state_container(&nested).unwrap(), "f".repeat(64));
    let mut bad = binding.clone();
    bad.id = "short".to_string();
    assert!(state_container(&bad).is_err());
    bad = binding.clone();
    bad.uid = -1;
    assert!(state_container(&bad).is_err());
    bad = binding.clone();
    bad.project = "short".to_string();
    assert!(state_container(&bad).is_err());
}

#[test]
fn config_view_matrix() {
    let view = decode_config_view(br#"{"settings.json":"e30=","trust.json":[123,125]}"#).unwrap();
    assert_eq!(view["settings.json"], b"{}");
    assert_eq!(view["trust.json"], b"{}");
    assert!(decode_config_view(b"[]").is_err());
    assert!(decode_config_view(br#"{"a":7}"#).is_err());
    assert!(decode_config_view(br#"{"a":"!!!"}"#).is_err());
    assert!(decode_config_view(br#"{"a":[256]}"#).is_err());
    assert!(decode_config_view(b"nope").is_err());
}

#[test]
fn control_execution_flows() {
    let lease = lease_fixture();
    let execution = MuseExecution {
        caller: caller(),
        request: LaunchRequest::default(),
        lease,
        binding: lease_fixture().binding.clone().unwrap(),
        path: format!("/run/soda-muse/{TID}"),
        unit: format!("soda-muse-{TID}.service"),
    };
    // Signal path pins the guest kill argv.
    let rt = runtime(FakeExec::new(vec![ok("")]));
    rt.control_execution(
        &execution,
        -1,
        &LaunchControl {
            signal: 15,
            cols: 0,
            rows: 0,
        },
        deadline(),
    )
    .unwrap();
    let calls = rt.exec.calls();
    assert_eq!(
        calls[0].2[4..],
        [
            "/usr/bin/systemctl".to_string(),
            "kill".to_string(),
            "--kill-whom=all".to_string(),
            "--signal=15".to_string(),
            format!("soda-muse-{TID}.service")
        ]
    );
    // Mixed signal+resize and bad signals deny without calling out.
    let rt = runtime(FakeExec::new(vec![]));
    assert!(rt
        .control_execution(
            &execution,
            -1,
            &LaunchControl {
                signal: 15,
                cols: 80,
                rows: 24
            },
            deadline()
        )
        .is_err());
    assert!(rt
        .control_execution(
            &execution,
            -1,
            &LaunchControl {
                signal: 9,
                cols: 0,
                rows: 0
            },
            deadline()
        )
        .is_err());
    assert!(rt.exec.calls().is_empty());
    // Resize requires a TTY session.
    assert!(rt
        .control_execution(
            &execution,
            -1,
            &LaunchControl {
                signal: 0,
                cols: 80,
                rows: 24
            },
            deadline()
        )
        .is_err());
    // Resize over a real PTY applies the window size.
    let tty = MuseExecution {
        request: LaunchRequest {
            tty: true,
            cols: 80,
            rows: 24,
            ..Default::default()
        },
        ..execution.clone()
    };
    let (master, slave) = open_pty_pair();
    let rt = runtime(FakeExec::new(vec![]));
    rt.control_execution(
        &tty,
        master,
        &LaunchControl {
            signal: 0,
            cols: 100,
            rows: 40,
        },
        deadline(),
    )
    .unwrap();
    assert_eq!(pty_size(master), (40, 100));
    unsafe {
        libc::close(master);
        libc::close(slave);
    }
    // Resize on a non-TTY denies.
    let fds = open_pipe_pair();
    let rt = runtime(FakeExec::new(vec![]));
    assert!(rt
        .control_execution(
            &tty,
            fds.0,
            &LaunchControl {
                signal: 0,
                cols: 80,
                rows: 24
            },
            deadline()
        )
        .is_err());
    unsafe {
        libc::close(fds.0);
        libc::close(fds.1);
    }
}

fn open_pty_pair() -> (RawFd, RawFd) {
    let mut master = -1;
    let mut slave = -1;
    // SAFETY: openpty with default termios/winsize.
    let result = unsafe {
        libc::openpty(
            &mut master,
            &mut slave,
            std::ptr::null_mut(),
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    assert_eq!(result, 0, "openpty unavailable");
    (master, slave)
}

fn pty_size(fd: RawFd) -> (u16, u16) {
    let mut ws: libc::winsize = unsafe { std::mem::zeroed() };
    // SAFETY: TIOCGWINSZ with a valid winsize pointer.
    let result = unsafe { libc::ioctl(fd, libc::TIOCGWINSZ, &mut ws) };
    assert_eq!(result, 0);
    (ws.ws_row, ws.ws_col)
}

fn open_pipe_pair() -> (RawFd, RawFd) {
    let mut fds = [-1, -1];
    // SAFETY: pipe with a valid fd pair.
    assert_eq!(unsafe { libc::pipe(fds.as_mut_ptr()) }, 0);
    (fds[0], fds[1])
}

#[test]
fn verify_guest_binary_flows() {
    let machine = if host_go_arch() == "arm64" {
        183u16
    } else {
        62u16
    };
    let header = elf_header(machine);
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
        Ok(header.clone()),
        ok(""),
    ]));
    rt.verify_guest_binary(CID, "", deadline()).unwrap();
    let calls = rt.exec.calls();
    assert_eq!(
        calls[0].2[4..],
        [
            "/usr/bin/sha256sum".to_string(),
            "/usr/local/libexec/soda/muse".to_string()
        ]
    );
    assert_eq!(
        calls[1].2[4..],
        [
            "/usr/bin/head".to_string(),
            "--bytes=64".to_string(),
            "/usr/local/libexec/soda/muse".to_string()
        ]
    );
    assert_eq!(
        calls[2].2[4..],
        [
            "/usr/local/bin/muse".to_string(),
            "--soda-check".to_string(),
            "1.0".to_string()
        ]
    );
    // Unpinned digest denies before any exec.
    let rt = MuseRuntime::new(
        FakeExec::new(vec![]),
        FakeHooks::new(),
        "1.0".to_string(),
        "short".to_string(),
    );
    assert_eq!(
        rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    // Digest mismatch denies.
    let rt = runtime(FakeExec::new(vec![ok(&format!(
        "{}  /usr/local/libexec/soda/muse\n",
        "f".repeat(64)
    ))]));
    assert_eq!(
        rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    // Bad ELF header denies.
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
        Ok(vec![0u8; 64]),
    ]));
    assert_eq!(
        rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    // Empty version denies after the header check.
    let rt = MuseRuntime::new(
        FakeExec::new(vec![
            ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
            Ok(header),
        ]),
        FakeHooks::new(),
        String::new(),
        PIN.to_string(),
    );
    assert_eq!(
        rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
        terminal::ERR_DENIED
    );
    // Version-handshake failure propagates raw.
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
        Ok(elf_header(machine)),
        err("exit status 1"),
    ]));
    assert_eq!(
        rt.verify_guest_binary(CID, "", deadline()).unwrap_err(),
        "exit status 1"
    );
    // Nested children route through the parent podman.
    let child = "f".repeat(64);
    let rt = runtime(FakeExec::new(vec![
        ok(&format!("{PIN}  /usr/local/libexec/soda/muse\n")),
        Ok(elf_header(machine)),
        ok(""),
    ]));
    rt.verify_guest_binary(CID, &child, deadline()).unwrap();
    let calls = rt.exec.calls();
    assert_eq!(
        calls[0].2[4..8],
        [
            "/usr/bin/podman".to_string(),
            "--remote=false".to_string(),
            "exec".to_string(),
            child
        ]
    );
}

// ----- socket protocol -----

fn seqpacket_pair() -> (RawFd, RawFd) {
    let mut fds = [-1, -1];
    // SAFETY: socketpair with a valid fd pair.
    let result =
        unsafe { libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, fds.as_mut_ptr()) };
    assert_eq!(result, 0, "socketpair unavailable");
    (fds[0], fds[1])
}

fn seqpacket_listener(path: &std::path::Path, backlog: i32) -> RawFd {
    use std::os::unix::ffi::OsStrExt;
    let fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    assert!(fd >= 0);
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as _;
    let bytes = path.as_os_str().as_bytes();
    assert!(bytes.len() < addr.sun_path.len());
    addr.sun_path[..bytes.len()]
        .copy_from_slice(unsafe { std::mem::transmute::<&[u8], &[libc::c_char]>(bytes) });
    let addr_len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    unsafe {
        assert_eq!(
            libc::bind(fd, &addr as *const _ as *const libc::sockaddr, addr_len),
            0
        );
        assert_eq!(libc::listen(fd, backlog), 0);
    }
    fd
}

fn seqpacket_connect(path: &std::path::Path) -> RawFd {
    use std::os::unix::ffi::OsStrExt;
    let fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    assert!(fd >= 0);
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as _;
    let bytes = path.as_os_str().as_bytes();
    addr.sun_path[..bytes.len()]
        .copy_from_slice(unsafe { std::mem::transmute::<&[u8], &[libc::c_char]>(bytes) });
    let addr_len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    let result = unsafe { libc::connect(fd, &addr as *const _ as *const libc::sockaddr, addr_len) };
    assert_eq!(result, 0);
    fd
}

fn send_with_fds(fd: RawFd, body: &[u8], fds: &[RawFd]) {
    unsafe {
        let mut iov = libc::iovec {
            iov_base: body.as_ptr() as *mut libc::c_void,
            iov_len: body.len(),
        };
        let mut hdr: libc::msghdr = std::mem::zeroed();
        hdr.msg_iov = &mut iov;
        hdr.msg_iovlen = 1;
        let cmsg_len = libc::CMSG_SPACE((fds.len() * 4) as _) as usize;
        let mut control = vec![0u8; cmsg_len.max(1)];
        if !fds.is_empty() {
            hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
            hdr.msg_controllen = control.len() as _;
            let cmsg = libc::CMSG_FIRSTHDR(&hdr);
            assert!(!cmsg.is_null());
            (*cmsg).cmsg_level = libc::SOL_SOCKET;
            (*cmsg).cmsg_type = libc::SCM_RIGHTS;
            (*cmsg).cmsg_len = libc::CMSG_LEN((fds.len() * 4) as _) as _;
            let data = libc::CMSG_DATA(cmsg) as *mut RawFd;
            for (i, fd) in fds.iter().enumerate() {
                *data.add(i) = *fd;
            }
        }
        let n = libc::sendmsg(fd, &hdr, 0);
        assert_eq!(n as usize, body.len(), "sendmsg short");
    }
}

#[test]
fn peer_attestation() {
    let (a, b) = seqpacket_pair();
    let peer = muse_peer_from_fd(a).expect("kernel must offer SO_PEERPIDFD on 6.5+");
    assert_eq!(peer.pid, std::process::id() as i32);
    assert_eq!(peer.uid, unsafe { libc::geteuid() });
    assert_eq!(peer.gid, unsafe { libc::getegid() });
    assert!(peer.pidfd.as_raw_fd() >= 0);
    assert!(muse_peer_alive(&peer));
    drop(peer);
    unsafe {
        libc::close(a);
        libc::close(b);
    }
}

#[test]
fn accepted_muse_connection_is_cloexec() {
    use std::os::fd::AsRawFd;

    let address = rustix::net::SocketAddrUnix::new_abstract_name(
        format!("sodaos-muse-{}", std::process::id()).as_bytes(),
    )
    .unwrap();
    let listener = rustix::net::socket_with(
        rustix::net::AddressFamily::UNIX,
        rustix::net::SocketType::SEQPACKET,
        rustix::net::SocketFlags::CLOEXEC,
        None,
    )
    .unwrap();
    rustix::net::bind(&listener, &address).unwrap();
    rustix::net::listen(&listener, 1).unwrap();
    let client = rustix::net::socket_with(
        rustix::net::AddressFamily::UNIX,
        rustix::net::SocketType::SEQPACKET,
        rustix::net::SocketFlags::CLOEXEC,
        None,
    )
    .unwrap();
    rustix::net::connect(&client, &address).unwrap();
    let accepted = rustix::net::accept_with(&listener, rustix::net::SocketFlags::CLOEXEC).unwrap();
    let flags = unsafe { libc::fcntl(accepted.as_raw_fd(), libc::F_GETFD) };
    assert!(flags >= 0);
    assert_ne!(flags & libc::FD_CLOEXEC, 0);
    drop((accepted, client, listener));
}

#[test]
fn request_parsing_matrix() {
    // Register without descriptors.
    let (a, b) = seqpacket_pair();
    let body = format!(
        "{{\"register\":{{\"child_id\":{CID:?},\"actor_id\":\"7\",\"registration_id\":{TID:?},\"muse\":true}}}}"
    );
    send_with_fds(b, body.as_bytes(), &[]);
    let received = muse_request_from_fd(a).unwrap();
    assert!(received.request.register.is_some());
    assert!(received.files.iter().all(|f| f.is_none()));
    unsafe {
        libc::close(a);
        libc::close(b);
    }
    // Launch with three descriptors.
    let (a, b) = seqpacket_pair();
    let (p1, p2) = open_pipe_pair();
    let (p3, p4) = open_pipe_pair();
    let (p5, p6) = open_pipe_pair();
    send_with_fds(b, br#"{"cwd":"/w","tty":false}"#, &[p1, p3, p5]);
    let received = muse_request_from_fd(a).unwrap();
    assert_eq!(received.request.cwd, "/w");
    assert!(received.files.iter().all(|f| f.is_some()));
    for file in received.files.iter().flatten() {
        // SCM_RIGHTS receipt sets CLOEXEC atomically via MSG_CMSG_CLOEXEC.
        let flags = unsafe { libc::fcntl(file.as_raw_fd(), libc::F_GETFD) };
        assert!(flags >= 0);
        assert_ne!(flags & libc::FD_CLOEXEC, 0);
    }
    drop(received);
    unsafe {
        libc::close(a);
        libc::close(b);
        libc::close(p1);
        libc::close(p2);
        libc::close(p3);
        libc::close(p4);
        libc::close(p5);
        libc::close(p6);
    }
    // Launch without descriptors is invalid.
    let (a, b) = seqpacket_pair();
    send_with_fds(b, br#"{"cwd":"/w"}"#, &[]);
    assert_eq!(
        muse_request_from_fd(a).unwrap_err(),
        "invalid launch descriptors"
    );
    unsafe {
        libc::close(a);
        libc::close(b);
    }
    // Bad JSON and invalid requests fail.
    let (a, b) = seqpacket_pair();
    send_with_fds(b, b"nope", &[]);
    assert!(muse_request_from_fd(a).is_err());
    unsafe {
        libc::close(a);
        libc::close(b);
    }
    let (a, b) = seqpacket_pair();
    send_with_fds(b, br#"{"cwd":"relative"}"#, &[]);
    assert!(muse_request_from_fd(a).is_err());
    unsafe {
        libc::close(a);
        libc::close(b);
    }
}

#[test]
fn rejected_received_rights_are_closed() {
    // Four copies exceed the admitted three-descriptor launch contract. Once
    // the rejected request returns, closing the sender's writer must make the
    // pipe readable as EOF, proving the receiver dropped every SCM_RIGHTS fd.
    let (socket, sender) = seqpacket_pair();
    let mut pipe = [-1; 2];
    assert_eq!(unsafe { libc::pipe(pipe.as_mut_ptr()) }, 0);
    send_with_fds(sender, br#"{"cwd":"/w"}"#, &[pipe[1]; 4]);
    unsafe { libc::close(pipe[1]) };
    assert!(muse_request_from_fd(socket).is_err());
    let mut byte = 0u8;
    assert_eq!(
        unsafe { libc::read(pipe[0], (&mut byte as *mut u8).cast(), 1) },
        0
    );
    unsafe {
        libc::close(pipe[0]);
        libc::close(socket);
        libc::close(sender);
    }
}

#[test]
fn truncated_rights_are_rejected_and_closed() {
    let (socket, sender) = seqpacket_pair();
    let mut pipe = [-1; 2];
    assert_eq!(unsafe { libc::pipe(pipe.as_mut_ptr()) }, 0);
    send_with_fds(sender, br#"{"cwd":"/w"}"#, &[pipe[1]; 5]);
    unsafe { libc::close(pipe[1]) };
    assert!(muse_request_from_fd(socket).is_err());
    let mut byte = 0u8;
    assert_eq!(
        unsafe { libc::read(pipe[0], (&mut byte as *mut u8).cast(), 1) },
        0
    );
    unsafe {
        libc::close(pipe[0]);
        libc::close(socket);
        libc::close(sender);
    }
}

#[test]
fn oversized_request_datagram_is_rejected() {
    let (socket, sender) = seqpacket_pair();
    let body = vec![b' '; 65537];
    send_with_fds(sender, &body, &[]);
    assert!(muse_request_from_fd(socket).is_err());
    unsafe {
        libc::close(socket);
        libc::close(sender);
    }
}

#[test]
fn request_validation_failure_closes_received_rights() {
    let (socket, sender) = seqpacket_pair();
    let mut pipe = [-1; 2];
    assert_eq!(unsafe { libc::pipe(pipe.as_mut_ptr()) }, 0);
    send_with_fds(sender, br#"{"cwd":"relative"}"#, &[pipe[1]; 3]);
    unsafe { libc::close(pipe[1]) };
    assert!(muse_request_from_fd(socket).is_err());
    let mut byte = 0u8;
    assert_eq!(
        unsafe { libc::read(pipe[0], (&mut byte as *mut u8).cast(), 1) },
        0
    );
    unsafe {
        libc::close(pipe[0]);
        libc::close(socket);
        libc::close(sender);
    }
}

#[test]
fn command_exit_matrix() {
    use std::os::unix::process::ExitStatusExt;
    let exited = |code| {
        muse_command_exit(Ok(std::process::Command::new("/bin/sh")
            .arg("-c")
            .arg(format!("exit {code}"))
            .status()
            .unwrap()))
    };
    assert_eq!(
        exited(0),
        LaunchExit {
            code: 0,
            error: String::new()
        }
    );
    assert_eq!(
        exited(3),
        LaunchExit {
            code: 3,
            error: String::new()
        }
    );
    let signaled = std::process::Command::new("/bin/sh")
        .arg("-c")
        .arg("kill -TERM $$")
        .status()
        .unwrap();
    assert!(signaled.signal() == Some(15));
    assert_eq!(
        muse_command_exit(Ok(signaled)),
        LaunchExit {
            code: 143,
            error: String::new()
        }
    );
    assert_eq!(
        muse_command_exit(Err(std::io::Error::new(
            std::io::ErrorKind::BrokenPipe,
            "gone"
        ))),
        LaunchExit {
            code: 1,
            error: String::new()
        }
    );
}

#[test]
fn json_split_matrix() {
    assert_eq!(split_json_object(br#"{"a":1}"#), JsonPacket::Complete(7));
    assert_eq!(
        split_json_object(br#"{"a":{"b":[1,2]}}{"c":3}"#),
        JsonPacket::Complete(17)
    );
    assert_eq!(
        split_json_object(br#"{"a":"}{"}"#),
        JsonPacket::Complete(10)
    );
    assert_eq!(split_json_object(br#"{"a":"}"}"#), JsonPacket::Complete(9));
    assert_eq!(
        split_json_object(br#"{"a":"\"}"}"#),
        JsonPacket::Complete(11)
    );
    assert_eq!(split_json_object(br#"{"a":"}"#), JsonPacket::Incomplete);
    assert_eq!(split_json_object(b"{\"a\":"), JsonPacket::Incomplete);
    assert_eq!(split_json_object(b"{]"), JsonPacket::Invalid);
    assert_eq!(split_json_object(b""), JsonPacket::Incomplete);
    assert_eq!(split_json_object(b"   "), JsonPacket::Incomplete);
    assert_eq!(
        split_json_object(br#"{"signal":15} {"cols":1}"#),
        JsonPacket::Complete(13)
    );
}

#[test]
fn control_loop_matrix() {
    use std::sync::Arc;
    let lease = lease_fixture();
    let execution = MuseExecution {
        caller: caller(),
        request: LaunchRequest {
            tty: true,
            cols: 80,
            rows: 24,
            ..Default::default()
        },
        lease,
        binding: lease_fixture().binding.clone().unwrap(),
        path: format!("/run/soda-muse/{TID}"),
        unit: format!("soda-muse-{TID}.service"),
    };
    // One signal message reaches the guest, then EOF ends the loop.
    let rt = runtime(FakeExec::new(vec![ok("")]));
    let service = Arc::new(MuseLaunch::new(rt));
    let (a, b) = seqpacket_pair();
    let (master, slave) = open_pty_pair();
    let peer_service = service.clone();
    let execution_clone = execution.clone();
    let control_cancel = Arc::new(AtomicBool::new(false));
    let shutdown = Arc::new(AtomicBool::new(false));
    let control_cancel_worker = control_cancel.clone();
    let shutdown_worker = shutdown.clone();
    let worker = std::thread::spawn(move || {
        peer_service.control_loop(
            b,
            master,
            424242,
            &control_cancel_worker,
            &shutdown_worker,
            &execution_clone,
            Instant::now() + Duration::from_secs(30),
        );
        unsafe {
            libc::close(b);
            libc::close(master);
        }
    });
    unsafe {
        let body = br#"{"signal":15}"#;
        assert_eq!(
            libc::send(
                a,
                body.as_ptr() as *const libc::c_void,
                body.len(),
                libc::MSG_NOSIGNAL
            ) as usize,
            body.len()
        );
    }
    // Wait for the guest call, then close to end the loop.
    for _ in 0..100 {
        if !service.runtime.exec.calls().is_empty() {
            break;
        }
        std::thread::sleep(Duration::from_millis(10));
    }
    unsafe {
        libc::close(a);
        libc::close(slave);
    }
    worker.join().unwrap();
    let calls = service.runtime.exec.calls();
    assert_eq!(calls.len(), 1);
    assert!(calls[0].2.contains(&"--signal=15".to_string()));
    // Garbage ends the loop without guest calls.
    let rt = runtime(FakeExec::new(vec![]));
    let service = Arc::new(MuseLaunch::new(rt));
    let (a, b) = seqpacket_pair();
    let peer_service = service.clone();
    let control_cancel = AtomicBool::new(false);
    let shutdown = AtomicBool::new(false);
    let worker = std::thread::spawn(move || {
        peer_service.control_loop(
            b,
            -1,
            424242,
            &control_cancel,
            &shutdown,
            &execution,
            Instant::now() + Duration::from_secs(30),
        );
        unsafe {
            libc::close(b);
        }
    });
    unsafe {
        let body = b"nope";
        libc::send(
            a,
            body.as_ptr() as *const libc::c_void,
            body.len(),
            libc::MSG_NOSIGNAL,
        );
        libc::close(a);
    }
    worker.join().unwrap();
    assert!(service.runtime.exec.calls().is_empty());
}

#[test]
fn serve_shutdown_and_listener_setup() {
    use std::sync::Arc;
    // Listener dir setup pins.
    let dir = test_tmp("listener");
    let socket = dir.join("launch.sock");
    prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap();
    // Missing parents are created.
    let fresh = test_tmp("fresh");
    let nested = fresh.join("a").join("b").join("launch.sock");
    prepare_muse_listener_dir(nested.to_str().unwrap()).unwrap();
    assert!(fresh.join("a").join("b").is_dir());
    // Non-empty dir fails even when the socket path itself is absent;
    // an occupied socket inside a non-empty dir reports emptiness
    // first, since Go checks ReadDir before Lstat.
    std::fs::write(dir.join("junk"), b"x").unwrap();
    assert_eq!(
        prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap_err(),
        "muse interface directory must be empty before launch service startup"
    );
    std::fs::write(&socket, b"x").unwrap();
    assert_eq!(
        prepare_muse_listener_dir(socket.to_str().unwrap()).unwrap_err(),
        "muse interface directory must be empty before launch service startup"
    );
    std::fs::remove_file(dir.join("junk")).unwrap();
    std::fs::remove_file(&socket).unwrap();
    // Accept loop starts and stops cleanly on a short socket path.
    let path = dir.join("s").to_str().unwrap().to_string();
    let listen_fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    assert!(listen_fd >= 0);
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as _;
    let bytes = path.as_bytes();
    addr.sun_path[..bytes.len()]
        .copy_from_slice(unsafe { std::mem::transmute::<&[u8], &[libc::c_char]>(bytes) });
    let addr_len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    unsafe {
        assert_eq!(
            libc::bind(
                listen_fd,
                &addr as *const _ as *const libc::sockaddr,
                addr_len
            ),
            0
        );
        assert_eq!(libc::listen(listen_fd, 8), 0);
    }
    let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let peer = service.clone();
    let flag = shutdown.clone();
    let worker = std::thread::spawn(move || peer.serve(listen_fd, flag));
    std::thread::sleep(Duration::from_millis(50));
    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    unsafe {
        libc::close(listen_fd);
    }
    std::fs::remove_file(&path).ok();
}

#[test]
fn pidfd_signal_stays_bound_after_child_reap() {
    let shutdown = AtomicBool::new(false);
    let mut finished = std::process::Command::new("/bin/sh")
        .args(["-c", "exit 0"])
        .spawn()
        .unwrap();
    let identity = super::launch::pidfd_open(finished.id()).unwrap();
    assert!(finished.wait().unwrap().success());

    let mut later = std::process::Command::new("/bin/sleep")
        .arg("5")
        .spawn()
        .unwrap();
    // ESRCH for the already-exited identity is benign and cannot start host
    // shutdown or signal a later process that reuses the numeric PID.
    super::launch::signal_supervised_child(identity.as_raw_fd(), &shutdown).unwrap();
    assert!(!shutdown.load(Ordering::SeqCst));
    assert!(later.try_wait().unwrap().is_none());

    let later_identity = super::launch::pidfd_open(later.id()).unwrap();
    super::launch::signal_supervised_child(later_identity.as_raw_fd(), &shutdown).unwrap();
    assert!(!shutdown.load(Ordering::SeqCst));
    assert!(!later.wait().unwrap().success());
}

#[test]
fn failed_supervisor_signal_sets_shared_shutdown_and_refuses_success() {
    use std::sync::Arc;

    let shutdown = Arc::new(AtomicBool::new(false));
    let worker_shutdown = shutdown.clone();
    let supervisor = std::thread::spawn(move || {
        // An invalid pidfd deterministically exercises a non-benign signal
        // failure through the same production helper used by the supervisor.
        super::launch::signal_supervised_child(-1, &worker_shutdown)
    });
    let result = super::launch::finish_supervisor(
        LaunchExit {
            code: 0,
            error: String::new(),
        },
        supervisor.join(),
    );
    assert!(shutdown.load(Ordering::SeqCst));
    assert_eq!(result, LaunchExit::cleanup_unconfirmed());
}

#[test]
fn terminal_listener_failure_sets_the_main_shared_shutdown_flag() {
    use std::sync::Arc;

    let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let invalid_listener = std::fs::File::open("/dev/null").unwrap();
    let error = service.serve(invalid_listener.as_raw_fd(), shutdown.clone());
    assert!(error.is_err());
    assert!(shutdown.load(Ordering::SeqCst));
}

#[test]
fn control_loop_shutdown_cancels_idle_peer() {
    use std::sync::Arc;
    let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let execution = MuseExecution {
        caller: caller(),
        request: LaunchRequest::default(),
        lease: lease_fixture(),
        binding: lease_fixture().binding.clone().unwrap(),
        path: format!("/run/soda-muse/{TID}"),
        unit: format!("soda-muse-{TID}.service"),
    };
    let (client, control) = seqpacket_pair();
    let control_cancel = Arc::new(AtomicBool::new(false));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker_service = service.clone();
    let worker_cancel = control_cancel.clone();
    let worker_shutdown = shutdown.clone();
    let worker = std::thread::spawn(move || {
        worker_service.control_loop(
            control,
            -1,
            -1,
            &worker_cancel,
            &worker_shutdown,
            &execution,
            Instant::now() + Duration::from_secs(3600),
        );
        unsafe { libc::close(control) };
    });
    std::thread::sleep(Duration::from_millis(20));
    shutdown.store(true, Ordering::SeqCst);
    let start = Instant::now();
    worker.join().unwrap();
    assert!(start.elapsed() < Duration::from_millis(500));
    unsafe { libc::close(client) };
}

#[test]
fn listener_reclaims_completed_workers_during_churn() {
    use std::sync::Arc;
    let dir = test_tmp("worker-churn");
    let path = dir.join("launch.sock");
    let listener = seqpacket_listener(&path, 256);
    let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker_service = service.clone();
    let worker_shutdown = shutdown.clone();
    let worker = std::thread::spawn(move || worker_service.serve(listener, worker_shutdown));

    // More than the worker cap of short-lived malformed requests must keep
    // succeeding; stale completed handles would otherwise fill admission.
    for _ in 0..160 {
        let client = seqpacket_connect(&path);
        let body = b"not-json";
        let sent = unsafe {
            libc::send(
                client,
                body.as_ptr() as *const libc::c_void,
                body.len(),
                libc::MSG_NOSIGNAL,
            )
        };
        assert_eq!(sent as usize, body.len());
        let mut response = [0u8; 256];
        let received = unsafe {
            libc::recv(
                client,
                response.as_mut_ptr() as *mut libc::c_void,
                response.len(),
                0,
            )
        };
        assert!(
            received > 0,
            "worker admission stopped during completed-request churn"
        );
        unsafe { libc::close(client) };
    }
    shutdown.store(true, Ordering::SeqCst);
    assert!(worker.join().unwrap().is_ok());
    unsafe { libc::close(listener) };
    let _ = std::fs::remove_file(path);
}

#[test]
fn listener_caps_silent_peers_and_joins_them_on_shutdown() {
    use std::sync::Arc;
    let dir = test_tmp("silent-cap");
    let path = dir.join("launch.sock");
    let listener = seqpacket_listener(&path, 256);
    let service = Arc::new(MuseLaunch::new(runtime(FakeExec::new(vec![]))));
    let shutdown = Arc::new(AtomicBool::new(false));
    let worker_service = service.clone();
    let worker_shutdown = shutdown.clone();
    let worker = std::thread::spawn(move || worker_service.serve(listener, worker_shutdown));

    let mut peers = Vec::new();
    for _ in 0..128 {
        peers.push(seqpacket_connect(&path));
    }
    let overflow = seqpacket_connect(&path);
    let mut pfd = libc::pollfd {
        fd: overflow,
        events: libc::POLLIN,
        revents: 0,
    };
    assert!(
        unsafe { libc::poll(&mut pfd, 1, 2000) } > 0,
        "overflow peer was not refused promptly"
    );
    let mut byte = 0u8;
    assert_eq!(
        unsafe { libc::recv(overflow, &mut byte as *mut u8 as *mut libc::c_void, 1, 0) },
        0
    );
    unsafe { libc::close(overflow) };

    shutdown.store(true, Ordering::SeqCst);
    let start = Instant::now();
    let joined = worker.join();
    assert!(joined.unwrap().is_ok());
    assert!(
        start.elapsed() < Duration::from_secs(6),
        "silent request workers were not bounded at shutdown"
    );
    for peer in peers {
        unsafe { libc::close(peer) };
    }
    unsafe { libc::close(listener) };
    let _ = std::fs::remove_file(path);
}

#[test]
fn muse_host_environment_filters_meta_key() {
    let _guard = ENV_LOCK.lock().unwrap();
    unsafe {
        std::env::set_var("META_API_KEY", "secret");
        std::env::set_var("T26_MUSE_PROBE", "kept");
    }
    let env = muse_host_environment();
    assert!(!env.iter().any(|(k, _)| k == "META_API_KEY"));
    assert!(env
        .iter()
        .any(|(k, v)| k == "T26_MUSE_PROBE" && v == "kept"));
    // No secrets leak into error strings: the key never appears.
    unsafe {
        std::env::remove_var("META_API_KEY");
        std::env::remove_var("T26_MUSE_PROBE");
    }
}
