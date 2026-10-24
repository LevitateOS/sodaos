//! Binary-level parity oracle: every request kind through the compiled
//! `project-factory-roles` binary with the factory redirected to scratch.
//! Success stdout is asserted byte-exact; every refusal asserts the exit
//! code plus the fixed stderr contract.

use soda_host::factory::{self, FactoryBroker, FactoryTerminal};
use soda_host::project::Executor;
use soda_host::terminal::Binding;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{
    atomic::{AtomicBool, AtomicU8},
    Arc, Mutex,
};
use std::time::{Duration, Instant};

static NEXT: AtomicU64 = AtomicU64::new(0);

const PID: &str = "f0123456789abcdef01234567";
const PID2: &str = "f123456789abcdef012345678";
const COMMIT: &str = "cccccccccccccccccccccccccccccccccccccccc";
const FIXED_STDERR: &str =
    "factory preparation unconfirmed; inspect native state and managed files\n";

fn binary() -> PathBuf {
    for key in [
        "CARGO_BIN_EXE_project-factory-roles",
        "CARGO_BIN_EXE_project_factory_roles",
    ] {
        if let Ok(path) = std::env::var(key) {
            return PathBuf::from(path);
        }
    }
    let target = std::env::var("CARGO_TARGET_DIR").unwrap_or_else(|_| "target".to_string());
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../..")
        .join(target)
        .join("debug")
        .join("project-factory-roles")
}

struct Scratch {
    root: PathBuf,
    factory: PathBuf,
    git: PathBuf,
    record: PathBuf,
}

impl Scratch {
    fn fresh() -> Scratch {
        let id = NEXT.fetch_add(1, Ordering::SeqCst);
        let root = std::env::temp_dir().join(format!("factory-oracle-{}-{id}", std::process::id()));
        let _ = std::fs::remove_dir_all(&root);
        std::fs::create_dir_all(&root).unwrap();
        let factory = root.join("factory");
        let record = root.join("git.record");
        let git = root.join("git");
        let body = format!(
            "#!/bin/sh\n{{ echo '---'; printf '<%s>\\n' \"$@\"; }} >> '{}'\nexit 0\n",
            record.to_string_lossy()
        );
        std::fs::write(&git, body).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&git, std::fs::Permissions::from_mode(0o755)).unwrap();
        Scratch {
            root,
            factory,
            git,
            record,
        }
    }

    fn run(&self, stdin: &[u8]) -> (i32, String, String) {
        let mut child = Command::new(binary())
            .env("SODA_FACTORY_DIR", &self.factory)
            .env("SODA_FACTORY_GIT", &self.git)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .expect("spawn helper");
        child
            .stdin
            .as_mut()
            .unwrap()
            .write_all(stdin)
            .expect("stdin");
        let output = child.wait_with_output().expect("wait");
        (
            output.status.code().unwrap_or(-1),
            String::from_utf8_lossy(&output.stdout).into_owned(),
            String::from_utf8_lossy(&output.stderr).into_owned(),
        )
    }

    fn ok(&self, stdin: &str) -> String {
        let (code, stdout, stderr) = self.run(stdin.as_bytes());
        assert_eq!(code, 0, "stdout={stdout:?} stderr={stderr:?}");
        assert_eq!(stderr, "");
        stdout
    }

    fn refused(&self, stdin: &[u8]) {
        let (code, stdout, stderr) = self.run(stdin);
        assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
        assert_eq!(stdout, "");
        assert_eq!(stderr, FIXED_STDERR);
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.root);
    }
}

// Real Host orchestration and native helper/Git/processes. Only the external
// broker/provider boundary is observed locally; no account or model is used.
#[derive(Clone)]
struct NativeFactoryFixture {
    factory: PathBuf,
    events: Arc<Mutex<Vec<String>>>,
    crash: Arc<AtomicU8>,
    refuse_stop: Arc<AtomicBool>,
    requires_verification: bool,
}

impl Executor for NativeFactoryFixture {
    fn run(
        &self,
        input: &[u8],
        command: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.run_bounded(input, command, args, deadline, 512 * 1024, 4096)
    }
    fn run_bounded(
        &self,
        input: &[u8],
        command: &str,
        args: &[&str],
        deadline: Instant,
        stdout: usize,
        stderr: usize,
    ) -> Result<Vec<u8>, String> {
        assert_eq!(command, "podman");
        assert_eq!(
            args,
            [
                "--remote=false",
                "exec",
                "--interactive",
                &"b".repeat(64),
                soda_host::preparation::FACTORY_HELPER
            ]
        );
        let request: serde_json::Value = serde_json::from_slice(input).unwrap();
        let op = request["op"].as_str().unwrap();
        let crash = if op == "verify" {
            self.crash.swap(0, Ordering::SeqCst)
        } else {
            0
        };
        if crash == 1 {
            panic!("fixture Host loss before native helper dispatch");
        }
        let raw = soda_host::project::Native.run_bounded(
            input,
            "/usr/bin/env",
            &[
                "-i",
                "PATH=/usr/bin:/bin",
                &format!("SODA_FACTORY_DIR={}", self.factory.display()),
                "SODA_FACTORY_GIT=/usr/bin/git",
                binary().to_str().unwrap(),
            ],
            deadline,
            stdout,
            stderr,
        )?;
        let response: serde_json::Value = serde_json::from_slice(&raw).unwrap();
        self.events.lock().unwrap().push(format!(
            "helper:{op}:{}",
            response["phase"].as_str().unwrap()
        ));
        if crash == 2 {
            panic!("fixture Host loss after actual native helper dispatch");
        }
        Ok(raw)
    }
}

impl FactoryBroker for NativeFactoryFixture {
    fn acquire(
        &self,
        request: &factory::AcquireRequest,
        _: Instant,
    ) -> Result<factory::Lease, factory::FactoryError> {
        self.events.lock().unwrap().push("acquire".into());
        Ok(factory::Lease {
            id: request.execution_id.clone(),
            execution_id: request.execution_id.clone(),
            kind: "factory".into(),
            generation: 1,
            provider_id: "codex".into(),
            actor_id: 1,
            project_id: request.project_id.clone(),
            connection_id: request.connection_id.clone(),
            role: request.role.clone(),
            deadline: request.deadline.clone(),
            ..Default::default()
        })
    }
    fn register(
        &self,
        run: &str,
        _: &Binding,
        _: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        let dir = self
            .factory
            .join("preparations")
            .join(PID)
            .join("verifications")
            .join(run);
        if self.requires_verification {
            let finished: serde_json::Value =
                serde_json::from_slice(&std::fs::read(dir.join("finished.json")).unwrap()).unwrap();
            assert_eq!(finished["setup_exit"], 0);
            assert_eq!(finished["check_exit"], 0);
            assert!(!dir.join("work").exists() && !dir.join("home").exists());
        } else {
            assert!(
                !dir.exists(),
                "research must not stage an execution manifest"
            );
        }
        self.events.lock().unwrap().push("register".into());
        Ok(b"local-provider-boundary".to_vec())
    }
    fn return_lease(
        &self,
        _: &str,
        _: &Binding,
        _: &[u8],
        _: Instant,
    ) -> Result<(), factory::FactoryError> {
        self.events.lock().unwrap().push("return".into());
        Ok(())
    }
    fn reconcile_lease(&self, _: &str, _: Instant) {
        self.events.lock().unwrap().push("reconcile".into());
    }
    fn execution_is_terminal(
        &self,
        _: &str,
        _: &str,
        _: Instant,
    ) -> Result<bool, factory::FactoryError> {
        Ok(true)
    }
    fn close_execution(&self, _: &str, _: &str, _: Instant) -> Result<(), factory::FactoryError> {
        self.events.lock().unwrap().push("close".into());
        Ok(())
    }
}

impl FactoryTerminal for NativeFactoryFixture {
    fn harness_pin(&self, _: &str) -> Result<factory::FactoryHarnessPin, String> {
        Ok(factory::FactoryHarnessPin {
            harness: "codex".into(),
            version: "native-fixture".into(),
            sha256: "a".repeat(64),
            image: String::new(),
        })
    }
    fn reserve(
        &self,
        run: &factory::FactoryRun,
        _: &factory::Lease,
        _: &str,
        _: i64,
        _: Instant,
    ) -> Result<Binding, factory::FactoryError> {
        self.events.lock().unwrap().push("reserve".into());
        Ok(Binding {
            kind: "factory".into(),
            id: run.id.clone(),
            project: "b".repeat(64),
            login: run.role.clone(),
            uid: unsafe { libc::geteuid() }.into(),
            gid: unsafe { libc::getegid() }.into(),
            scope: "codex".into(),
            invocation_id: "a".repeat(32),
            generation: 1,
            child_id: PID.into(),
            credential_root: factory::factory_run_paths(&run.role, &run.preparation, &run.id).1,
            ..Default::default()
        })
    }
    fn start(
        &self,
        _: &factory::Lease,
        _: &[u8],
        _: &[u8],
        _: Instant,
        _: &str,
    ) -> Result<(), factory::FactoryError> {
        let mut events = self.events.lock().unwrap();
        assert!(events.iter().any(|e| e == "register"));
        events.push("cli-start".into());
        Ok(())
    }
    fn wait(&self, _: &factory::Lease, _: Instant) -> Result<(i64, String), factory::FactoryError> {
        Ok((0, String::new()))
    }
    fn stop(&self, _: &factory::Lease, _: Instant) -> Result<(), factory::FactoryError> {
        self.events.lock().unwrap().push("native-stop".into());
        if self.refuse_stop.load(Ordering::SeqCst) {
            Err(factory::FactoryError::Uncertain)
        } else {
            Ok(())
        }
    }
    fn stop_unbound(
        &self,
        _: &factory::FactoryRun,
        _: Instant,
    ) -> Result<(), factory::FactoryError> {
        Ok(())
    }
    fn capture(&self, _: &factory::Lease, _: Instant) -> Result<Vec<u8>, factory::FactoryError> {
        self.events.lock().unwrap().push("capture".into());
        Ok(b"local-provider-boundary".to_vec())
    }
    fn live(&self, _: &Binding, _: Instant) -> bool {
        false
    }
    fn output(
        &self,
        _: &str,
        _: &Binding,
        _: i64,
        _: i64,
        _: Instant,
    ) -> Result<factory::OutputSlice, factory::FactoryError> {
        Err(factory::FactoryError::NotFound)
    }
    fn takeover_copy(
        &self,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
        _: Instant,
    ) -> Result<(String, bool), factory::FactoryError> {
        Err(factory::FactoryError::NotFound)
    }
    fn export_bundle(
        &self,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
        _: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        Err(factory::FactoryError::NotFound)
    }
}

fn b64_encode(data: &[u8]) -> String {
    const ALPHA: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::new();
    for chunk in data.chunks(3) {
        let mut block = [0u8; 3];
        block[..chunk.len()].copy_from_slice(chunk);
        let triple = (u32::from(block[0]) << 16) | (u32::from(block[1]) << 8) | u32::from(block[2]);
        out.push(ALPHA[(triple >> 18) as usize] as char);
        out.push(ALPHA[((triple >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(ALPHA[((triple >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHA[(triple & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

/// Canonical digest recomputed independently (framing: sorted names,
/// NUL separators) via the system `sha256sum` over a temp file.
fn canonical_digest(files: &[(&str, &[u8])]) -> String {
    let mut ordered: Vec<(&str, &[u8])> = files.to_vec();
    ordered.sort_by(|a, b| a.0.cmp(b.0));
    let mut framed = Vec::new();
    for (name, contents) in ordered {
        framed.extend_from_slice(name.as_bytes());
        framed.push(0);
        framed.extend_from_slice(contents);
    }
    let path = std::env::temp_dir().join(format!(
        "factory-digest-{}-{}.bin",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::SeqCst)
    ));
    std::fs::write(&path, &framed).unwrap();
    let output = Command::new("sha256sum").arg(&path).output().unwrap();
    let _ = std::fs::remove_file(&path);
    assert!(output.status.success());
    String::from_utf8(output.stdout).unwrap()[..64].to_string()
}

fn approve_request(pid: &str, files: &[(&str, &[u8])]) -> String {
    let digest = canonical_digest(files);
    let files_json: Vec<String> = files
        .iter()
        .map(|(name, contents)| format!("\"{name}\": \"{}\"", b64_encode(contents)))
        .collect();
    format!(
        "{{\"op\": \"approve\", \"id\": \"{pid}\", \"role\": \"soda-coder\", \
         \"setup_digest\": \"{digest}\", \"source_commit\": \"{COMMIT}\", \
         \"files\": {{{}}}, \"bundle\": \"{}\", \"credential\": \"\"}}",
        files_json.join(", "),
        b64_encode(b"bundle"),
    )
}

fn record_request(pid: &str, missing: &str, refusal: Option<&str>) -> String {
    let verified = match refusal {
        Some(text) => format!(
            "\"uid\": \"1\", \"login\": \"soda-coder\", \"groups\": \"soda-coder\", \"refusal\": \"{text}\""
        ),
        None => "\"uid\": \"1\", \"login\": \"soda-coder\", \"groups\": \"soda-coder\"".to_string(),
    };
    format!(
        "{{\"op\": \"record\", \"id\": \"{pid}\", \"tools\": [], \
         \"missing\": \"{missing}\", \"verified\": {{{verified}}}}}"
    )
}

fn fixture_request(pid: &str) -> String {
    approve_request(pid, &[("setup.sh", b"true\n"), ("check.sh", b"true\n")])
}

/// Extract a top-level string field from a flat response (test-only).
fn field(stdout: &str, key: &str) -> String {
    let needle = format!("\"{key}\": \"");
    let start = stdout
        .find(&needle)
        .unwrap_or_else(|| panic!("{key} in {stdout}"))
        + needle.len();
    stdout[start..].split('"').next().unwrap().to_string()
}

fn wait_for(path: &Path, timeout_secs: u64) {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(timeout_secs);
    while !path.exists() {
        assert!(
            std::time::Instant::now() < deadline,
            "timed out waiting for {}",
            path.display()
        );
        std::thread::sleep(std::time::Duration::from_millis(50));
    }
}

#[test]
fn oracle_ensure_exact_bytes() {
    let scratch = Scratch::fresh();
    assert_eq!(
        scratch.ok("{\"op\": \"ensure\"}"),
        "{\"roles\": [\"soda-coder\", \"soda-reviewer\"]}\n"
    );
    // Idempotent rerun, byte-identical.
    assert_eq!(
        scratch.ok("{\"op\": \"ensure\"}"),
        "{\"roles\": [\"soda-coder\", \"soda-reviewer\"]}\n"
    );
}

#[test]
fn oracle_approve_record_inspect_exact_bytes() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    let approved = scratch.ok(&fixture_request(PID));
    let checkout = scratch
        .factory
        .join("test-homes/soda-coder/checkouts")
        .join(PID)
        .to_string_lossy()
        .into_owned();
    assert_eq!(
        approved,
        format!(
            "{{\"approved\": \"{PID}\", \"repeated\": false, \"checkout\": \"{checkout}\", \
             \"credential_file\": \"\"}}\n"
        )
    );
    assert_eq!(
        scratch.ok(&fixture_request(PID)),
        format!("{{\"approved\": \"{PID}\", \"repeated\": true}}\n")
    );
    // The bundle verification ran with the exact argv, then cleaned up.
    let log = std::fs::read_to_string(&scratch.record).unwrap();
    let snapshot = scratch
        .factory
        .join("preparations")
        .join(PID)
        .join("snapshot");
    let bundle = snapshot
        .join("source.bundle")
        .to_string_lossy()
        .into_owned();
    let repo = snapshot
        .join("verify-tmp")
        .join("repo")
        .to_string_lossy()
        .into_owned();
    assert!(
        log.contains(&format!(
            "---\n<clone>\n<-q>\n<--no-checkout>\n<{bundle}>\n<{repo}>\n"
        )),
        "{log}"
    );
    assert!(
        log.contains(&format!(
            "---\n<-C>\n<{repo}>\n<cat-file>\n<-e>\n<{COMMIT}>\n"
        )),
        "{log}"
    );
    assert!(!snapshot.join("verify-tmp").exists());
    assert_eq!(
        scratch.ok(&record_request(PID, "", None)),
        format!("{{\"recorded\": \"{PID}\", \"waiting\": false}}\n")
    );
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.starts_with(
        "{\"hold\": {\"active\": false, \"revision\": -1}, \"known\": true, \
         \"phase\": \"approved\", \"role\": \"soda-coder\", "
    ));
    assert!(state.contains("\"stopped\": false, \"ready\": false"));
    assert!(
        state.ends_with(", \"setup_log\": \"\", \"check_log\": \"\"}\n"),
        "{state}"
    );
}

#[test]
fn oracle_start_runs_detached_to_ready() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    scratch.ok(&approve_request(
        PID,
        &[("setup.sh", b"echo hello"), ("check.sh", b"true\n")],
    ));
    scratch.ok(&record_request(PID, "", None));
    let started = scratch.ok(&format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}"));
    assert!(started.contains("\"repeated\": false"), "{started}");
    let directory = scratch.factory.join("preparations").join(PID);
    wait_for(&directory.join("finished.json"), 30);
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.contains("\"phase\": \"ready\""), "{state}");
    assert!(
        state.contains("\"setup_exit\": 0, \"check_exit\": 0"),
        "{state}"
    );
    assert!(state.contains("\"setup_log\": \"hello\\n\""), "{state}");

    // The approved verification route runs real Git/source and shell checks
    // in a distinct disposable clone, never the retained preparation tree.
    let mut verified = Scratch::fresh();
    verified.git = PathBuf::from("/usr/bin/git");
    verified.ok("{\"op\":\"ensure\"}");
    let source = verified.root.join("source");
    std::fs::create_dir(&source).unwrap();
    for args in [
        vec!["init", "-q"],
        vec!["config", "user.email", "native@example.invalid"],
        vec!["config", "user.name", "Native fixture"],
    ] {
        assert!(Command::new("/usr/bin/git")
            .arg("-C")
            .arg(&source)
            .args(args)
            .status()
            .unwrap()
            .success());
    }
    std::fs::write(source.join("record.txt"), "retained\n").unwrap();
    assert!(Command::new("/usr/bin/git")
        .arg("-C")
        .arg(&source)
        .args(["add", "record.txt"])
        .status()
        .unwrap()
        .success());
    assert!(Command::new("/usr/bin/git")
        .arg("-C")
        .arg(&source)
        .args(["commit", "-qm", "native verification fixture"])
        .status()
        .unwrap()
        .success());
    let head = Command::new("/usr/bin/git")
        .arg("-C")
        .arg(&source)
        .args(["rev-parse", "HEAD"])
        .output()
        .unwrap();
    assert!(head.status.success());
    let commit = String::from_utf8(head.stdout).unwrap().trim().to_string();
    let bundle = verified.root.join("source.bundle");
    assert!(Command::new("/usr/bin/git")
        .arg("-C")
        .arg(&source)
        .args(["bundle", "create"])
        .arg(&bundle)
        .arg("HEAD")
        .status()
        .unwrap()
        .success());
    let setup = b"echo setup-private-marker\n";
    let check = b"set -eu\ntest -z \"${SODA_CREDENTIAL_FILE-}\"\ntest \"$(cat record.txt)\" = retained\nprintf isolated > record.txt\ncase \"${SODA_VERIFICATION_RUN_ID-}\" in 222*|ccc*) printf 'failed-check\\n'; exit 7;; 333*|666*|777*|ddd*|eee*|fff*) echo waiting; sleep 120;; 444*) yes x | head -c 70000;; bbb*) sleep 1; printf 'long-check-ok\\n\\tprobe\\033[31m\\r'; i=0; while [ \"$i\" -lt 3000 ]; do printf '\\303\\251'; i=$((i+1)); done;; *) echo check-ok;; esac\n";
    let mut approve: serde_json::Value = serde_json::from_str(&approve_request(
        PID,
        &[("setup.sh", setup), ("check.sh", check)],
    ))
    .unwrap();
    approve["role"] = "soda-reviewer".into();
    approve["source_commit"] = commit.clone().into();
    approve["bundle"] = b64_encode(&std::fs::read(&bundle).unwrap()).into();
    verified.ok(&approve.to_string());
    let checkout = verified
        .factory
        .join("test-homes/soda-reviewer/checkouts")
        .join(PID);
    assert!(Command::new("/usr/bin/git")
        .args(["clone", "-q", "--no-hardlinks"])
        .arg(&source)
        .arg(&checkout)
        .status()
        .unwrap()
        .success());
    verified.ok(&record_request(PID, "", None));
    verified.ok(&format!("{{\"op\":\"start\",\"id\":\"{PID}\"}}"));
    let prep = verified.factory.join("preparations").join(PID);
    wait_for(&prep.join("finished.json"), 30);
    assert!(verified
        .ok(&format!("{{\"op\":\"inspect\",\"id\":\"{PID}\"}}"))
        .contains("\"phase\": \"ready\""));
    let retained = std::fs::read(checkout.join("record.txt")).unwrap();
    let check_file = verified.root.join("check.digest");
    std::fs::write(&check_file, check).unwrap();
    let hashed = Command::new("sha256sum").arg(&check_file).output().unwrap();
    assert!(hashed.status.success());
    let check_digest = &String::from_utf8(hashed.stdout).unwrap()[..64];
    let deadline = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
        + 120;
    let mut request = serde_json::json!({"op":"verify","id":PID,"run_id":"1".repeat(32),"invocation_id":"a".repeat(32),"manifest_digest":"b".repeat(64),"check_digest":check_digest,"source_commit":commit,"deadline":soda_wire_time::format(deadline,0).unwrap()});
    let first = verified.ok(&request.to_string());
    assert!(first.contains("\"phase\": \"running\""), "{first}");
    let run_dir = prep.join("verifications").join("1".repeat(32));
    wait_for(&run_dir.join("finished.json"), 30);
    request["op"] = "verify_stop".into();
    let passed: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(passed["phase"], "completed");
    assert_eq!(passed["retirement"], "confirmed");
    assert_eq!(passed["check_exit"], 0);
    assert_eq!(passed["output"], "check-ok\n");
    assert!(!run_dir.join("stopped.json").exists());
    assert!(!run_dir.join("work").exists());
    assert!(!run_dir.join("home").exists());
    assert_eq!(
        std::fs::read(checkout.join("record.txt")).unwrap(),
        retained
    );
    request["op"] = "verify".into();
    let replay: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(replay, passed);
    request["invocation_id"] = "c".repeat(32).into();
    verified.refused(request.to_string().as_bytes());
    request["invocation_id"] = "a".repeat(32).into();
    request["check_digest"] = "e".repeat(64).into();
    request["run_id"] = "5".repeat(32).into();
    verified.refused(request.to_string().as_bytes());
    assert!(!prep.join("verifications").join("5".repeat(32)).exists());
    request["check_digest"] = check_digest.to_string().into();
    request["run_id"] = "2".repeat(32).into();
    verified.ok(&request.to_string());
    let failed_dir = prep.join("verifications").join("2".repeat(32));
    wait_for(&failed_dir.join("finished.json"), 30);
    request["op"] = "verify_stop".into();
    let failed: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(failed["phase"], "failed");
    assert_eq!(failed["check_exit"], 7);
    assert_eq!(failed["retirement"], "confirmed");
    // Stop the unfinished exact check; no source/fixture resurrection on replay.
    request["op"] = "verify".into();
    request["run_id"] = "3".repeat(32).into();
    verified.ok(&request.to_string());
    let stopped_dir = prep.join("verifications").join("3".repeat(32));
    wait_for(&stopped_dir.join("check.log"), 30);
    verified.ok("{\"op\":\"hold\",\"revision\":1}");
    verified.refused(b"{\"op\":\"release\",\"revision\":1}");
    request["op"] = "verify_stop".into();
    let mut stopped: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    eprintln!("initial native fixture Stop receipt: {stopped}");
    // Stop may conservatively observe an unfinished child/supervisor join.
    // Reconcile only this same tuple; never start another check or relax
    // confirmed retirement, cleanup or the immutable stopped replay below.
    let until = Instant::now() + Duration::from_secs(2);
    while stopped["retirement"] == "unconfirmed" && Instant::now() < until {
        std::thread::sleep(Duration::from_millis(20));
        stopped = serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    }
    assert_eq!(stopped["phase"], "stopped");
    assert_eq!(stopped["retirement"], "confirmed");
    assert!(!stopped_dir.join("work").exists());
    verified.ok("{\"op\":\"release\",\"revision\":1}");
    request["op"] = "verify".into();
    let replay: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(replay, stopped);
    assert_eq!(
        std::fs::read(checkout.join("record.txt")).unwrap(),
        retained
    );
    request["run_id"] = "4".repeat(32).into();
    verified.ok(&request.to_string());
    let capped_dir = prep.join("verifications").join("4".repeat(32));
    wait_for(&capped_dir.join("finished.json"), 30);
    request["op"] = "verify_stop".into();
    let capped: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(capped["phase"], "completed");
    assert_eq!(capped["truncated"], true);
    assert_eq!(capped["output"].as_str().unwrap().len(), 65536);
    let raw = std::fs::read(capped_dir.join("check.log")).unwrap();
    std::fs::write(&check_file, &raw[..65536]).unwrap();
    let digest = Command::new("sha256sum").arg(&check_file).output().unwrap();
    assert_eq!(
        capped["output_digest"],
        &String::from_utf8(digest.stdout).unwrap()[..64]
    );
    // Deadline retirement is a retained failed check, not an implicit retry.
    let expires = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
        + 2;
    request["deadline"] = soda_wire_time::format(expires, 0).unwrap().into();
    request["run_id"] = "6".repeat(32).into();
    request["op"] = "verify".into();
    verified.ok(&request.to_string());
    let expired_dir = prep.join("verifications").join("6".repeat(32));
    wait_for(&expired_dir.join("finished.json"), 30);
    request["op"] = "verify_stop".into();
    let expired: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(expired["phase"], "failed");
    assert_eq!(expired["check_exit"], 124);
    assert_eq!(expired["retirement"], "confirmed");
    request["op"] = "verify".into();
    let replay: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(replay, expired);
    // Drive the actual Host orchestration against the same native helper.
    // The observed external boundary records order/custody calls only.
    for (digit, crash, parent_uncertain, phase) in [
        ('a', 0, false, "research"),
        ('b', 0, false, "verify"),
        ('c', 0, false, "verify"),
        ('d', 2, false, "verify"),
        ('e', 1, false, "verify"),
        ('f', 2, true, "verify"),
    ] {
        let run_id = digit.to_string().repeat(32);
        let host_state = verified.root.join(format!("host-{digit}"));
        std::fs::create_dir(&host_state).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&host_state, std::fs::Permissions::from_mode(0o700)).unwrap();
        let fixture = NativeFactoryFixture {
            factory: verified.factory.clone(),
            events: Arc::new(Mutex::new(Vec::new())),
            crash: Arc::new(AtomicU8::new(crash)),
            refuse_stop: Arc::new(AtomicBool::new(parent_uncertain)),
            requires_verification: phase == "verify",
        };
        let mut manifest = soda_host::factory::VerificationManifest {
            version: "project-check-v1".into(),
            run: run_id.clone(),
            project: "p0123456789abcdef01234567".into(),
            preparation: PID.into(),
            source_commit: commit.clone(),
            deadline: soda_wire_time::format(deadline, 0).unwrap(),
            intent: "d0123456789abcdef01234567".into(),
            accepted_revision: "d123456789abcdef012345678".into(),
            scope_digest: "1".repeat(64),
            adopted_manifest_digest: "2".repeat(64),
            requirements: "d23456789abcdef0123456789".into(),
            approval: "d3456789abcdef0123456789a".into(),
            preparation_digest: canonical_digest(&[("setup.sh", setup), ("check.sh", check)]),
            check_digest: check_digest.into(),
            outcomes: vec!["d456789abcdef0123456789ab".into()],
            checks: Vec::new(),
        };
        manifest.checks = serde_json::from_value(serde_json::json!([{
            "catalog_id":"project.check", "outcomes":manifest.outcomes,
            "artifacts":["project.check.receipt"], "fixtures":["project.preparation"]
        }]))
        .unwrap();
        let bytes = serde_json::to_vec(&manifest).unwrap();
        let digest = soda_host::sha256::hex_lower(&soda_host::sha256::digest(&bytes));
        let prompt = b"Native approved check orchestration fixture; no model/provider.".to_vec();
        let launch = factory::FactoryLaunch {
            run: factory::FactoryRun {
                deadline: manifest.deadline.clone(),
                actor: 1,
                id: run_id.clone(),
                project: manifest.project.clone(),
                role: "soda-reviewer".into(),
                preparation: PID.into(),
                harness: "codex".into(),
                harness_vers: "native-fixture".into(),
                assignment: soda_host::sha256::hex_lower(&soda_host::sha256::digest(&prompt)),
                source_commit: commit.clone(),
                connection: "local-observed-boundary".into(),
                execution_phase: phase.into(),
                verification_manifest_digest: if phase == "verify" {
                    digest.clone()
                } else {
                    String::new()
                },
                ..Default::default()
            },
            prompt,
            harness_sha256: "a".repeat(64),
            verification_manifest: if phase == "verify" { bytes } else { Vec::new() },
            ..Default::default()
        };
        let open = || {
            factory::Factory::open_factory(
                host_state.to_str().unwrap(),
                fixture.clone(),
                fixture.clone(),
                fixture.clone(),
            )
            .unwrap()
        };
        if phase == "research" {
            assert!(launch.validate().is_ok());
            for wrong in ["soda-coder", "soda-researcher"] {
                let mut invalid = launch.clone();
                invalid.run.role = wrong.into();
                assert!(invalid.validate().is_err(), "research admitted {wrong}");
            }
        }
        if crash == 0 {
            let began = Instant::now();
            let state = open()
                .launch(&launch, Instant::now() + Duration::from_millis(250))
                .unwrap();
            if phase == "research" {
                assert_eq!(state.phase, "completed");
                assert!(state.credential_returned);
                assert!(state.verification_checks.is_none());
                let replay = open()
                    .launch(&launch, Instant::now() + Duration::from_secs(10))
                    .unwrap();
                assert_eq!(replay.phase, "completed");
                let events = fixture.events.lock().unwrap();
                assert_eq!(events.iter().filter(|e| *e == "cli-start").count(), 1);
                assert_eq!(events.iter().filter(|e| *e == "register").count(), 1);
                assert!(
                    events.iter().any(|e| e == "capture") && events.iter().any(|e| e == "return")
                );
                assert!(!events.iter().any(|e| e.starts_with("helper:verify")));
            } else if digit == 'b' {
                assert!(
                    began.elapsed() >= Duration::from_secs(1),
                    "check did not exceed short observation deadline"
                );
                assert_eq!(state.phase, "completed");
                assert!(state.credential_returned);
                let checks = state.verification_checks.as_ref().unwrap();
                assert_eq!(checks.phase, "passed");
                let wire = serde_json::to_value(checks).unwrap();
                let excerpt = wire["output_excerpt"]
                    .as_str()
                    .expect("approved CHECK excerpt");
                assert!(excerpt.starts_with("long-check-ok\n\tprobe"));
                assert!(excerpt.contains("\\u{1b}[31m\\r"));
                assert!(!excerpt.contains("setup-private-marker"));
                assert!(!excerpt
                    .chars()
                    .any(|c| c.is_control() && c != '\n' && c != '\t'));
                assert!(excerpt.len() <= 4096 && excerpt.ends_with('é'));
                assert_eq!(wire["output_truncated"], true);
                let output = format!("long-check-ok\n\tprobe\u{1b}[31m\r{}", "é".repeat(3000));
                assert_eq!(
                    checks.output_digest,
                    soda_host::sha256::hex_lower(&soda_host::sha256::digest(output.as_bytes()))
                );
                let reopened = open()
                    .inspect(
                        &factory::FactoryInspect {
                            project: manifest.project.clone(),
                            id: run_id.clone(),
                        },
                        Instant::now() + Duration::from_secs(10),
                    )
                    .unwrap();
                assert_eq!(
                    serde_json::to_value(reopened.verification_checks.unwrap()).unwrap(),
                    wire
                );
                let events = fixture.events.lock().unwrap();
                let registered = events.iter().position(|e| e == "register").unwrap();
                assert!(events[..registered]
                    .iter()
                    .any(|e| e.ends_with(":completed")));
                assert!(
                    events.iter().any(|e| e == "cli-start")
                        && events.iter().any(|e| e == "capture")
                        && events.iter().any(|e| e == "return")
                );
            } else {
                assert_eq!(state.phase, "failed");
                assert_eq!(state.retirement, "confirmed");
                assert!(!state.credential_returned);
                let checks = state.verification_checks.as_ref().unwrap();
                assert_eq!(checks.phase, "failed");
                let wire = serde_json::to_value(checks).unwrap();
                assert_eq!(wire["output_excerpt"], "failed-check\n");
                assert!(!checks.output_truncated);
                assert_eq!(
                    checks.output_digest,
                    soda_host::sha256::hex_lower(&soda_host::sha256::digest(b"failed-check\n"))
                );
                let reopened = open()
                    .inspect(
                        &factory::FactoryInspect {
                            project: manifest.project.clone(),
                            id: run_id.clone(),
                        },
                        Instant::now() + Duration::from_secs(10),
                    )
                    .unwrap();
                assert_eq!(
                    serde_json::to_value(reopened.verification_checks.unwrap()).unwrap(),
                    wire
                );
            }
        } else {
            let worker = open();
            let request = launch.clone();
            let lost = std::thread::spawn(move || {
                worker.launch(&request, Instant::now() + Duration::from_millis(250))
            })
            .join();
            assert!(lost.is_err(), "Host-loss boundary was not reached");
            // A newly opened Factory uses the durable pending identity and
            // only inspects/stops. It never submits a second verify start.
            let fresh = open();
            let request = factory::FactoryVerificationChecks {
                project: manifest.project.clone(),
                run: run_id.clone(),
                incarnation: "a".repeat(32),
                manifest_digest: digest,
            };
            let observed =
                fresh.verification_checks(&request, Instant::now() + Duration::from_secs(10));
            if crash == 2 {
                assert_eq!(observed.unwrap().phase, "running");
            } else {
                assert!(observed.is_err());
            }
            let stop = factory::FactoryStop {
                project: manifest.project.clone(),
                id: run_id.clone(),
            };
            let stopped = fresh
                .stop(&stop, Instant::now() + Duration::from_secs(10))
                .unwrap();
            if parent_uncertain {
                assert_eq!(stopped.phase, "uncertain");
                assert_eq!(stopped.retirement, "uncertain");
                fixture.refuse_stop.store(false, Ordering::SeqCst);
            } else {
                assert_eq!(stopped.phase, "stopped");
                assert_eq!(stopped.retirement, "confirmed");
            }
            let stopped = open()
                .stop(&stop, Instant::now() + Duration::from_secs(10))
                .unwrap();
            assert_eq!(stopped.phase, "stopped");
            assert_eq!(stopped.retirement, "confirmed");
            assert!(!stopped.credential_returned);
            let late = open()
                .launch(&launch, Instant::now() + Duration::from_secs(10))
                .unwrap();
            assert_eq!(late.phase, "stopped");
            assert!(!late.delivered);
            let events = fixture.events.lock().unwrap();
            assert_eq!(
                events
                    .iter()
                    .filter(|e| e.starts_with("helper:verify:"))
                    .count(),
                usize::from(crash == 2)
            );
            assert!(events.iter().any(|e| e == "reconcile") && events.iter().any(|e| e == "close"));
        }
        if phase == "verify" && digit != 'b' {
            let events = fixture.events.lock().unwrap();
            for forbidden in ["register", "cli-start", "capture", "return"] {
                assert!(
                    !events.iter().any(|e| e == forbidden),
                    "pre-CLI path invoked {forbidden}: {events:?}"
                );
            }
        }
        assert_eq!(
            std::fs::read(checkout.join("record.txt")).unwrap(),
            retained
        );
        let own = prep.join("verifications").join(run_id);
        assert!(!own.join("active-child.json").exists());
        assert!(!own.join("work").exists());
        assert!(!own.join("home").exists());
    }
    // The enclosing preparation's ordinary Stop also joins verification work.
    request["deadline"] = soda_wire_time::format(deadline, 0).unwrap().into();
    request["run_id"] = "7".repeat(32).into();
    verified.ok(&request.to_string());
    let final_dir = prep.join("verifications").join("7".repeat(32));
    wait_for(&final_dir.join("check.log"), 30);
    let stop = verified.ok(&format!("{{\"op\":\"stop\",\"id\":\"{PID}\"}}"));
    assert!(stop.contains("\"retirement\": \"confirmed\""), "{stop}");
    request["op"] = "verify_inspect".into();
    let final_state: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(final_state["phase"], "stopped");
    assert_eq!(final_state["retirement"], "confirmed");
    assert!(!final_dir.join("work").exists());
    assert_eq!(
        std::fs::read(checkout.join("record.txt")).unwrap(),
        retained
    );
    // Stop before dispatch durably fences the exact tuple. A later delivery
    // of the original start cannot launch or bind a different incarnation.
    request["op"] = "verify_stop".into();
    request["run_id"] = "9".repeat(32).into();
    let absent: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(absent["phase"], "stopped");
    assert_eq!(absent["retirement"], "confirmed");
    request["op"] = "verify".into();
    let delayed: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(delayed, absent);
    assert!(!prep
        .join("verifications")
        .join("9".repeat(32))
        .join("started.json")
        .exists());
    request["invocation_id"] = "f".repeat(32).into();
    verified.refused(request.to_string().as_bytes());
    request["invocation_id"] = "a".repeat(32).into();
    // Retry also completes a stop which crashed after binding the request
    // but before creating its absence marker. There was no launch or fixture.
    request["run_id"] = "a".repeat(32).into();
    let interrupted_stop = prep.join("verifications").join("a".repeat(32));
    std::fs::create_dir(&interrupted_stop).unwrap();
    let mut stored = request.clone();
    stored.as_object_mut().unwrap().remove("op");
    std::fs::write(interrupted_stop.join("request.json"), stored.to_string()).unwrap();
    request["op"] = "verify_stop".into();
    let stopped: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(stopped["phase"], "stopped");
    assert_eq!(stopped["retirement"], "confirmed");
    assert!(!interrupted_stop.join("started.json").exists());
    // An unrecorded fixture is observable uncertainty. Replay never launches,
    // and Stop cannot remove unknown data or pretend maintenance is safe.
    request["op"] = "verify".into();
    request["run_id"] = "8".repeat(32).into();
    let partial = prep.join("verifications").join("8".repeat(32));
    std::fs::create_dir(&partial).unwrap();
    let mut stored = request.clone();
    stored.as_object_mut().unwrap().remove("op");
    std::fs::write(partial.join("request.json"), stored.to_string()).unwrap();
    std::fs::create_dir(partial.join("work")).unwrap();
    std::fs::write(partial.join("work/unknown-data"), "retain").unwrap();
    let uncertain: serde_json::Value =
        serde_json::from_str(&verified.ok(&request.to_string())).unwrap();
    assert_eq!(uncertain["phase"], "uncertain");
    assert_eq!(uncertain["retirement"], "unconfirmed");
    assert!(!partial.join("started.json").exists());
    request["op"] = "verify_stop".into();
    verified.refused(request.to_string().as_bytes());
    assert!(partial.join("request.json").exists());
    assert_eq!(
        std::fs::read_to_string(partial.join("work/unknown-data")).unwrap(),
        "retain"
    );
    verified.ok("{\"op\":\"hold\",\"revision\":2}");
    verified.refused(b"{\"op\":\"release\",\"revision\":2}");
}

#[test]
fn oracle_setup_failure_skips_check() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    scratch.ok(&approve_request(
        PID,
        &[("setup.sh", b"echo out; exit 3"), ("check.sh", b"echo ran")],
    ));
    scratch.ok(&record_request(PID, "", None));
    scratch.ok(&format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}"));
    let directory = scratch.factory.join("preparations").join(PID);
    wait_for(&directory.join("finished.json"), 30);
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.contains("\"phase\": \"failed\""), "{state}");
    assert!(
        state.contains("\"setup_exit\": 3, \"check_exit\": null"),
        "{state}"
    );
    assert!(state.contains("\"setup_log\": \"out\\n\""), "{state}");
    assert!(state.contains("\"check_log\": \"\""), "{state}");
}

#[test]
fn oracle_output_truncates_at_cap() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    scratch.ok(&approve_request(
        PID,
        &[
            ("setup.sh", b"yes | head -c 70000"),
            ("check.sh", b"true\n"),
        ],
    ));
    scratch.ok(&record_request(PID, "", None));
    scratch.ok(&format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}"));
    let directory = scratch.factory.join("preparations").join(PID);
    wait_for(&directory.join("finished.json"), 30);
    let log = std::fs::read(directory.join("setup.log")).unwrap();
    assert_eq!(log.len(), 65536 + "\n[output truncated]\n".len());
    assert!(log.ends_with(b"\n[output truncated]\n"));
}

#[test]
fn oracle_running_stop_lifecycle() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    scratch.ok(&approve_request(
        PID,
        &[("setup.sh", b"sleep 120"), ("check.sh", b"true\n")],
    ));
    scratch.ok(&record_request(PID, "", None));
    let started = scratch.ok(&format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}"));
    let pgid: i32 = started
        .split("\"pgid\": ")
        .nth(1)
        .and_then(|tail| tail.split('}').next())
        .unwrap()
        .trim()
        .parse()
        .unwrap();
    // Guard: never leak the sleeper even if the stop path fails.
    struct Guard(i32);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe {
                libc::killpg(self.0, libc::SIGKILL);
            }
        }
    }
    let _guard = Guard(pgid);
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.contains("\"phase\": \"running\""), "{state}");
    // Repeat start reports the running supervisor without a pgid.
    assert_eq!(
        scratch.ok(&format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}")),
        format!("{{\"started\": \"{PID}\", \"repeated\": true}}\n")
    );
    let stopped = scratch.ok(&format!("{{\"op\": \"stop\", \"id\": \"{PID}\"}}"));
    assert_eq!(
        stopped,
        format!("{{\"stopped\": \"{PID}\", \"retirement\": \"confirmed\", \"known\": true}}\n")
    );
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.contains("\"phase\": \"stopped\""), "{state}");
}

#[test]
fn oracle_stop_hold_release_bytes() {
    let scratch = Scratch::fresh();
    assert_eq!(
        scratch.ok(&format!("{{\"op\": \"stop\", \"id\": \"{PID}\"}}")),
        format!("{{\"stopped\": \"{PID}\", \"retirement\": \"confirmed\", \"known\": false}}\n")
    );
    assert_eq!(
        scratch.ok("{\"op\": \"hold\", \"revision\": 4}"),
        "{\"hold\": {\"active\": true, \"revision\": 4}}\n"
    );
    // Hold denies new preparation work.
    scratch.refused(fixture_request(PID2).as_bytes());
    scratch.refused(b"{\"op\": \"release\", \"revision\": 5}");
    assert_eq!(
        scratch.ok("{\"op\": \"release\", \"revision\": 4}"),
        "{\"hold\": {\"active\": false, \"revision\": -1}}\n"
    );
}

#[test]
fn oracle_dead_supervisor_is_interrupted() {
    let scratch = Scratch::fresh();
    scratch.ok("{\"op\": \"ensure\"}");
    scratch.ok(&fixture_request(PID));
    scratch.ok(&record_request(PID, "", None));
    // Record a supervisor group that just exited.
    use std::os::unix::process::CommandExt;
    let mut child = Command::new("/bin/true").process_group(0).spawn().unwrap();
    let pgid = child.id();
    child.wait().unwrap();
    let directory = scratch.factory.join("preparations").join(PID);
    std::fs::write(
        directory.join("started.json"),
        format!("{{\"pid\": {pgid}, \"pgid\": {pgid}}}"),
    )
    .unwrap();
    let state = scratch.ok(&format!("{{\"op\": \"inspect\", \"id\": \"{PID}\"}}"));
    assert!(state.contains("\"phase\": \"interrupted\""), "{state}");
    scratch.refused(format!("{{\"op\": \"start\", \"id\": \"{PID}\"}}").as_bytes());
}

#[test]
fn oracle_refusal_contract() {
    let scratch = Scratch::fresh();
    // Not JSON, not an object, unknown op, missing op, wrong shapes.
    for body in [
        "not json".to_string(),
        String::new(),
        "[1, 2]".to_string(),
        "null".to_string(),
        "{\"op\": \"frobnicate\"}".to_string(),
        "{\"id\": \"x\"}".to_string(),
        "{\"op\": \"ensure\", \"extra\": 1}".to_string(),
        "{\"op\": \"hold\", \"revision\": true}".to_string(),
        "{\"op\": \"approve\"}".to_string(),
    ] {
        scratch.refused(body.as_bytes());
    }
    // Oversized requests are refused before parsing.
    let big = vec![b'x'; 4 * 1024 * 1024 + 65536 + 1];
    scratch.refused(&big);
    // Bad identities and untrusted inputs are refused with no effects.
    scratch.refused(b"{\"op\": \"stop\", \"id\": \"../escape\"}");
    // field() helper pins exact response values on the success path.
    scratch.ok("{\"op\": \"ensure\"}");
    let approved = scratch.ok(&fixture_request(PID));
    assert_eq!(field(&approved, "approved"), PID);
}
