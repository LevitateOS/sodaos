use super::super::tcodex::*;

use crate::project::Executor;
use crate::sha256;
use crate::terminal::ERR_DENIED;
use crate::terminal::{self, Binding, Lease, Service, KIND_FACTORY};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

pub(in crate::terminal) const PID: &str = "p0123456789abcdef01234567";
pub(in crate::terminal) const RID: &str = "0123456789abcdef0123456789abcdef";
pub(in crate::terminal) const IID: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
pub(in crate::terminal) const CID: &str =
    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
pub(in crate::terminal) const PREP: &str = "f0123456789abcdef01234567";
pub(in crate::terminal) const ROLE: &str = "soda-coder";
pub(in crate::terminal) const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
pub(in crate::terminal) const PIN: &str =
    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);

pub(in crate::terminal) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn test_tmp(slug: &str) -> std::path::PathBuf {
    let n = TMP_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("t26c-{}-{n}-{slug}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

pub(in crate::terminal) type RecordedCall = (Vec<u8>, String, Vec<String>);

pub(in crate::terminal) struct FakeExec {
    calls: Mutex<Vec<RecordedCall>>,
    script: Mutex<Vec<Result<Vec<u8>, String>>>,
}

impl FakeExec {
    pub(in crate::terminal) fn new(script: Vec<Result<Vec<u8>, String>>) -> Self {
        FakeExec {
            calls: Mutex::new(Vec::new()),
            script: Mutex::new(script),
        }
    }

    pub(in crate::terminal) fn calls(&self) -> Vec<RecordedCall> {
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

pub(in crate::terminal) fn ok(body: &str) -> Result<Vec<u8>, String> {
    Ok(body.as_bytes().to_vec())
}

pub(in crate::terminal) fn err(body: &str) -> Result<Vec<u8>, String> {
    Err(body.to_string())
}

pub(in crate::terminal) fn make_service(exec: FakeExec) -> Service<FakeExec> {
    Service {
        exec,
        codex_harness: "/opt/harness".to_string(),
        codex_harness_sha256: PIN.to_string(),
        codex_harness_version: "1.2.3".to_string(),
        muse_harness: String::new(),
        muse_harness_sha256: String::new(),
        muse_harness_version: String::new(),
    }
}

pub(in crate::terminal) fn inspect_json() -> String {
    format!(
        "{{\"id\":{CID:?},\"running\":true,\"project\":{PID:?},\"owner\":\"7\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[],\"GidMap\":[]}}}}"
    )
}

pub(in crate::terminal) fn factory_run() -> FactoryRun {
    FactoryRun {
        deadline_raw: "2030-01-01T00:00:00Z".to_string(),
        actor: 7,
        id: RID.to_string(),
        project: PID.to_string(),
        role: ROLE.to_string(),
        preparation: PREP.to_string(),
        harness: FACTORY_HARNESS_CODEX.to_string(),
        harness_vers: "1.2.3".to_string(),
        model: String::new(),
        assignment: PIN.to_string(),
        source_commit: COMMIT.to_string(),
        connection: "conn".to_string(),
    }
}

fn run_dir() -> String {
    factory_run_paths(ROLE, PREP, RID).unwrap().1
}

pub(in crate::terminal) fn factory_lease() -> Lease {
    Lease {
        provider_id: terminal::PROVIDER_CODEX.to_string(),
        id: "lease-f".to_string(),
        connection_id: "conn".to_string(),
        generation: 5,
        actor_id: 7,
        project_id: PID.to_string(),
        execution_id: RID.to_string(),
        kind: KIND_FACTORY.to_string(),
        binding: Some(Binding {
            kind: KIND_FACTORY.to_string(),
            id: RID.to_string(),
            project: CID.to_string(),
            login: ROLE.to_string(),
            uid: 1001,
            gid: 1001,
            scope: FACTORY_SCOPE_CODEX.to_string(),
            invocation_id: IID.to_string(),
            credential_root: run_dir(),
            generation: 5,
            child_id: PREP.to_string(),
        }),
        ..Default::default()
    }
}

fn write_harness(dir: &std::path::Path) {
    let bin = dir.join("bin");
    std::fs::create_dir_all(&bin).unwrap();
    std::fs::write(bin.join("codex"), b"").unwrap(); // sha256("") == PIN
    std::fs::write(bin.join("codex-code-mode-host"), b"host-bytes").unwrap();
    use std::os::unix::fs::PermissionsExt;
    for name in ["codex", "codex-code-mode-host"] {
        std::fs::set_permissions(bin.join(name), std::fs::Permissions::from_mode(0o755)).unwrap();
    }
}

pub(in crate::terminal) fn host_digest() -> String {
    sha256::hex_lower(&sha256::digest(b"host-bytes"))
}

// ----- domain -----

#[test]
fn domain_predicates() {
    assert!(valid_factory_role("soda-coder"));
    assert!(valid_factory_role("soda-reviewer"));
    assert!(!valid_factory_role("dev"));
    assert!(valid_preparation_id(PREP));
    assert!(!valid_preparation_id("p0123456789abcdef01234567"));
    assert!(valid_digest(PIN));
    assert!(!valid_digest("xyz"));
    assert!(valid_commit(COMMIT));
    assert!(!valid_commit(&COMMIT[..39]));
    assert!(valid_factory_run_id(RID));
    assert!(!valid_factory_run_id("short"));
    assert!(valid_harness_version("1.2.3"));
    assert!(valid_harness_version("v1_beta-2.x"));
    assert!(!valid_harness_version(""));
    assert!(!valid_harness_version(".1"));
    assert!(!valid_harness_version(&"a".repeat(33)));
    assert!(!valid_harness_version("a/b"));
}

#[test]
fn run_validate_pins() {
    assert!(factory_run().validate().is_ok());
    let cases = [
        (
            FactoryRun {
                id: "short".to_string(),
                ..factory_run()
            },
            "invalid factory run identity",
        ),
        (
            FactoryRun {
                project: "nope".to_string(),
                ..factory_run()
            },
            "invalid factory run identity",
        ),
        (
            FactoryRun {
                role: "dev".to_string(),
                ..factory_run()
            },
            "invalid factory run identity",
        ),
        (
            FactoryRun {
                preparation: "nope".to_string(),
                ..factory_run()
            },
            "invalid run preparation reference",
        ),
        (
            // The retired parallel family name stays denied: the
            // Muse CLI family is `muse`, never `muse-code`.
            FactoryRun {
                harness: "muse-code".to_string(),
                ..factory_run()
            },
            "unsupported factory harness",
        ),
        (
            FactoryRun {
                harness_vers: "../x".to_string(),
                ..factory_run()
            },
            "unsupported factory harness",
        ),
        (
            FactoryRun {
                model: "a".repeat(129),
                ..factory_run()
            },
            "invalid run model selection",
        ),
        (
            FactoryRun {
                model: "a\nb".to_string(),
                ..factory_run()
            },
            "invalid run model selection",
        ),
        (
            FactoryRun {
                model: "a\x7fb".to_string(),
                ..factory_run()
            },
            "invalid run model selection",
        ),
        (
            FactoryRun {
                assignment: "short".to_string(),
                ..factory_run()
            },
            "invalid run assignment or source identity",
        ),
        (
            FactoryRun {
                source_commit: "short".to_string(),
                ..factory_run()
            },
            "invalid run assignment or source identity",
        ),
        (
            FactoryRun {
                connection: String::new(),
                ..factory_run()
            },
            "invalid run sponsorship",
        ),
        (
            FactoryRun {
                actor: 0,
                ..factory_run()
            },
            "invalid run sponsorship",
        ),
        (
            FactoryRun {
                deadline_raw: String::new(),
                ..factory_run()
            },
            "run deadline is required",
        ),
    ];
    for (run, want) in cases {
        assert_eq!(run.validate().unwrap_err(), want, "{run:?}");
    }
    // Model bytes are checked raw: multibyte UTF-8 is fine.
    assert!(FactoryRun {
        model: "gpt-5é".to_string(),
        ..factory_run()
    }
    .validate()
    .is_ok());
    // Strict decode pins.
    assert!(FactoryRun::decode(br#"{"id":"x","bogus":1}"#).is_err());
    assert!(FactoryRun::decode(br#"{"deadline":"nope"}"#).is_err());
    let run = FactoryRun::decode(
        format!(
            "{{\"deadline\":\"2030-01-01T00:00:00Z\",\"actor\":7,\"id\":{RID:?},\"project\":{PID:?},\"role\":\"soda-coder\",\"preparation\":{PREP:?},\"harness\":\"codex\",\"harness_version\":\"1.2.3\",\"assignment\":{PIN:?},\"source_commit\":{COMMIT:?},\"connection\":\"conn\"}}"
        )
        .as_bytes(),
    )
    .unwrap();
    assert_eq!(run, factory_run());
}

#[test]
fn path_vectors() {
    let (checkout, run_dir, home, codex) = factory_run_paths(ROLE, PREP, RID).unwrap();
    assert_eq!(checkout, format!("/home/{ROLE}/checkouts/{PREP}"));
    assert_eq!(run_dir, format!("{checkout}/.soda-home/runs/{RID}"));
    assert_eq!(home, format!("{run_dir}/home"));
    assert_eq!(codex, format!("{home}/.codex"));
    assert!(factory_run_paths("dev", PREP, RID).is_none());
    assert!(factory_run_paths(ROLE, "nope", RID).is_none());
    assert!(factory_run_paths(ROLE, PREP, "nope").is_none());
    assert_eq!(
        factory_codex_guest("1.2.3").unwrap(),
        "/usr/local/bin/codex-factory-1.2.3"
    );
    assert!(factory_codex_guest("../x").is_none());
    assert_eq!(
        factory_unit_name(RID).unwrap(),
        format!("soda-factory-{RID}.service")
    );
    assert!(factory_unit_name("nope").is_none());
    assert_eq!(
        takeover_destination("dev", RID).unwrap(),
        format!("/home/dev/factory-takeover/{RID}")
    );
    assert!(takeover_destination("root", RID).is_none());
    assert!(takeover_source(
        &format!("/home/{ROLE}/checkouts/{PREP}"),
        ROLE,
        PREP
    ));
    assert!(!takeover_source(
        "/home/soda-coder/checkouts/other",
        ROLE,
        PREP
    ));
    assert!(!takeover_source(
        &format!("/home/{ROLE}/checkouts/{PREP}"),
        "dev",
        PREP
    ));
    let p = factory_codex_paths(&factory_run()).unwrap();
    assert_eq!(p.prompt, format!("{}/prompt", p.run_dir));
    assert_eq!(p.auth, format!("{}/auth.json", p.codex));
    assert_eq!(p.guest, "/usr/local/bin/codex-factory-1.2.3");
    assert!(factory_codex_paths(&FactoryRun::default()).is_err());
}

#[test]
fn quote_vectors() {
    assert_eq!(shell_quote("abc"), "'abc'");
    assert_eq!(shell_quote("a'b"), "'a'\\''b'");
    assert_eq!(systemd_escape("a$b$$c"), "a$$b$$$$c");
    assert_eq!(systemd_escape("plain"), "plain");
}

// ----- script goldens (byte-exact vs Go) -----

const GOLDEN_SUPERVISOR: &str = r#"RUNDIR='/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef'
exec >"$RUNDIR/stdout.log" 2>&1
STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo "$$ ${20}" >"$RUNDIR/supervisor.pid"
fail() { echo "$1" >"$RUNDIR/exit"; exit "$1"; }
i=0; while [ ! -f "$RUNDIR/marker" ]; do [ -f "$RUNDIR/stop" ] && fail 44; i=$((i+1)); [ "$i" -gt 600 ] && fail 42; sleep 1; done
mv "$RUNDIR/marker" "$RUNDIR/started" || fail 43
'/usr/local/bin/codex-factory-1.2.3' exec --color never --sandbox danger-full-access --skip-git-repo-check --config 'model_reasoning_effort="low"' --output-last-message '/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef/last-message.txt' - <'/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef/prompt'
CODE=$?; echo "$CODE" >"$RUNDIR/exit"; exit "$CODE"
"#;

const GOLDEN_RETIRE: &str = r#"RUNDIR='/home/soda-coder/checkouts/f0123456789abcdef01234567/.soda-home/runs/0123456789abcdef0123456789abcdef'
: >"$RUNDIR/stop"
[ -f "$RUNDIR/supervisor.pid" ] || exit 0
read PID START <"$RUNDIR/supervisor.pid"
case "$PID" in ''|*[!0-9]*) exit 0;; esac
case "$START" in ''|*[!0-9]*) exit 0;; esac
if [ -d "/proc/$PID" ]; then
  if STAT=$(cat "/proc/$PID/stat" 2>/dev/null); then
    REST=${STAT##*)}; set -- $REST
    if [ "${20}" = "$START" ]; then
      CMDLINE=$(tr '\000' ' ' <"/proc/$PID/cmdline" 2>/dev/null) || CMDLINE=""
      case "$CMDLINE" in *"$RUNDIR"*)
        if [ "$3" = "$PID" ]; then kill -KILL -- "-$PID" 2>/dev/null || true; else kill -KILL -- "$PID" 2>/dev/null || true; fi
      ;; esac
    fi
  fi
fi
sleep 1
for S in /proc/[0-9]*/stat; do
  STAT=$(cat "$S" 2>/dev/null) || continue
  REST=${STAT##*)}; set -- $REST
  if [ "$3" = "$PID" ]; then echo "lingering: $S"; exit 1; fi
done
exit 0
"#;

#[test]
fn script_goldens() {
    let p = factory_codex_paths(&factory_run()).unwrap();
    assert_eq!(factory_supervisor(&p, &p.guest, ""), GOLDEN_SUPERVISOR);
    let with_model = factory_supervisor(&p, &p.guest, "gpt-5");
    assert!(
        with_model.contains("--model 'gpt-5' --output-last-message"),
        "{with_model}"
    );
    assert_eq!(
        with_model,
        GOLDEN_SUPERVISOR.replace(
            "--config 'model_reasoning_effort=\"low\"' --output-last-message",
            "--config 'model_reasoning_effort=\"low\"' --model 'gpt-5' --output-last-message"
        )
    );
    assert_eq!(factory_retire(&p), GOLDEN_RETIRE);
    // Escaped supervisor doubles every dollar.
    let escaped = systemd_escape(&with_model);
    assert_eq!(escaped, with_model.replace('$', "$$"));
    assert!(escaped.contains("exec >\"$$RUNDIR/stdout.log\" 2>&1"));
    assert!(escaped.contains("'/usr/local/bin/codex-factory-1.2.3' exec"));
    assert_eq!(
        FACTORY_EXPORT_SCRIPT,
        "set -eu\nsrc=$1\ncandidate=$2\nlimit=$3\nif ! /usr/bin/test -d \"$src/.git/objects\"; then\n  printf 'soda-export-missing\\n'\n  exit 0\nfi\ndir=$(/usr/bin/mktemp -d \"$TMPDIR/.soda-export-XXXXXX\")\ntrap '/usr/bin/rm -rf \"$dir\"' EXIT HUP INT TERM\n/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= \"$dir/repo.git\" >/dev/null 2>/dev/null\nexport GIT_OBJECT_DIRECTORY=\"$src/.git/objects\"\ngit_export() {\n  /usr/bin/git -c core.hooksPath=/dev/null --git-dir=\"$dir/repo.git\" \"$@\"\n}\nif ! actual=$(git_export rev-parse --verify \"$candidate^{commit}\" 2>/dev/null); then\n  printf 'soda-export-invalid\\n'\n  exit 0\nfi\nif [ \"$actual\" != \"$candidate\" ]; then\n  printf 'soda-export-invalid\\n'\n  exit 0\nfi\ngit_export update-ref HEAD \"$candidate\" 2>/dev/null\ngit_export bundle create \"$dir/candidate.bundle\" HEAD 2>/dev/null\n/usr/bin/head -c \"$limit\" \"$dir/candidate.bundle\"\n"
    );
}

#[test]
fn binding_matrix() {
    let lease = factory_lease();
    let p = factory_codex_binding(&lease).unwrap();
    assert_eq!(p.run_dir, run_dir());
    assert!(p.guest.is_empty()); // harness fields are not in the binding
    let mut broken = lease.clone();
    broken.binding = None;
    assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
    for mutate in [
        Box::new(|b: &mut Binding| b.kind = "terminal".to_string()) as Box<dyn Fn(&mut Binding)>,
        Box::new(|b: &mut Binding| b.scope = "other".to_string()),
        Box::new(|b: &mut Binding| b.id = "short".to_string()),
        Box::new(|b: &mut Binding| b.project = "short".to_string()),
        Box::new(|b: &mut Binding| b.login = "root".to_string()),
        Box::new(|b: &mut Binding| b.uid = 0),
        Box::new(|b: &mut Binding| b.generation = 99),
        Box::new(|b: &mut Binding| b.invocation_id = "short".to_string()),
        Box::new(|b: &mut Binding| b.child_id = "nope".to_string()),
        Box::new(|b: &mut Binding| b.credential_root = "/elsewhere".to_string()),
    ] {
        let mut broken = lease.clone();
        mutate(broken.binding.as_mut().unwrap());
        assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
    }
    let mut broken = lease.clone();
    broken.provider_id = "muse".to_string();
    assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
    broken = lease.clone();
    broken.kind = "terminal".to_string();
    assert_eq!(factory_codex_binding(&broken).unwrap_err(), ERR_DENIED);
}

#[test]
fn unit_show_vectors() {
    assert_eq!(
        parse_factory_unit_show(b"ActiveState=active\nInvocationID=abc\n"),
        FactoryUnitShow {
            active: true,
            invocation: "abc".to_string()
        }
    );
    assert_eq!(
        parse_factory_unit_show(b"ActiveState=inactive\nInvocationID=\n"),
        FactoryUnitShow {
            active: false,
            invocation: String::new()
        }
    );
    // Last occurrence wins.
    assert!(parse_factory_unit_show(b"ActiveState=inactive\nActiveState=active\n").active);
    assert_eq!(parse_factory_unit_show(b""), FactoryUnitShow::default());
    assert_eq!(factory_role_id(b"1001\n").unwrap(), 1001);
    assert_eq!(factory_role_id(b"0").unwrap_err(), "invalid role identity");
    assert_eq!(factory_role_id(b"-5").unwrap_err(), "invalid role identity");
    assert_eq!(
        factory_role_id(b"nope").unwrap_err(),
        "invalid role identity"
    );
    assert_eq!(factory_output_size(b"123\n"), Some(123));
    assert_eq!(factory_output_size(b"-1"), None);
    assert_eq!(factory_output_size(b"nope"), None);
    assert_eq!(factory_output_size(&[b'1'; 65]), None);
}

// ----- reserve/start/wait state machine -----

pub(in crate::terminal) fn euid() -> u32 {
    unsafe { libc::geteuid() }
}

pub(in crate::terminal) fn reserve_harness() -> (std::path::PathBuf, String) {
    let dir = test_tmp("reserve");
    write_harness(&dir);
    let path = dir.to_str().unwrap().to_string();
    (dir, path)
}

#[test]
fn reserve_denial_pins() {
    let (_dir, harness) = reserve_harness();
    let run = factory_run();
    let lease = Lease {
        kind: KIND_FACTORY.to_string(),
        execution_id: RID.to_string(),
        generation: 5,
        ..Default::default()
    };
    // Each denial fires before any exec call.
    let bad_run = FactoryRun {
        id: "short".to_string(),
        ..factory_run()
    };
    let svc = make_service(FakeExec::new(vec![]));
    assert_eq!(
        svc.factory_codex_reserve(&bad_run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        "invalid factory run identity"
    );
    let svc = make_service(FakeExec::new(vec![]));
    let mut svc = svc;
    svc.codex_harness = harness.clone();
    let bad_lease = Lease {
        kind: "terminal".to_string(),
        ..lease.clone()
    };
    assert_eq!(
        svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let bad_lease = Lease {
        execution_id: "other".to_string(),
        ..lease.clone()
    };
    assert_eq!(
        svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    let bad_lease = Lease {
        generation: 0,
        ..lease.clone()
    };
    assert_eq!(
        svc.factory_codex_reserve(&run, &bad_lease, PIN, 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, "", 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, &"f".repeat(64), 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    svc.codex_harness = "relative/path".to_string();
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 600, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    svc.codex_harness = harness;
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 59, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert_eq!(
        svc.factory_codex_reserve(&run, &lease, PIN, 10801, deadline())
            .unwrap_err(),
        ERR_DENIED
    );
    assert!(svc.exec.calls().is_empty());
}
