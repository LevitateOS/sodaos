//! Binary-level parity oracle: every request kind through the compiled
//! `project-factory-roles` binary with the factory redirected to scratch.
//! Success stdout is asserted byte-exact; every refusal asserts the exit
//! code plus the fixed stderr contract.

use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

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
