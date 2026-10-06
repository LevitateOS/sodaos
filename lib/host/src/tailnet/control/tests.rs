use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_companion::TailnetControl;
use crate::tailnet_domain::{ProjectRequest, RunTarget};
use crate::tcontrol as control;
use crate::tcontrol_native as native;
use crate::tcontrol_policy as policy;
use crate::tcontrol_provider as provider;
use crate::tcontrol_wire as wire;

const GO_REVISION: &str = "c2325fcf010801018acfa617389a67457360b7763a3b7343869ff73f5c4ad304";

fn soon(ms: u64) -> Instant {
    Instant::now() + Duration::from_millis(ms)
}

static SCRATCH_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Unique scratch directory under the package `target/` dir.
fn scratch(name: &str) -> PathBuf {
    let id = SCRATCH_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir =
        PathBuf::from("target/tcontrol-test").join(format!("{}-{}-{id}", name, std::process::id()));
    std::fs::create_dir_all(&dir).expect("scratch dir");
    dir
}

const NATIVE_STATUS: &str = r#"{"Version":"1.102.4","CurrentTailnet":{"Name":"soda.example.test","MagicDNSEnabled":true},"BackendState":"Running","HaveNodeKey":true,"Self":{"ID":"self","DNSName":"host.example.ts.net.","TailscaleIPs":["100.64.0.1"]},"Peer":{"peer":{"ID":"peer","DNSName":"exit.example.ts.net.","TailscaleIPs":["100.64.0.2"],"Online":true,"ExitNodeOption":true}},"AuthURL":"https://login.tailscale.com/a/synthetic","Health":["sensitive native diagnostic"],"PrivateKey":"must-not-project"}"#;
const NATIVE_PREFS: &str = r#"{"WantRunning":true,"ExitNodeID":"","ExitNodeIP":"","ExitNodeAllowLANAccess":false,"AdvertiseRoutes":["10.8.0.0/16"],"Persist":{"PrivateNodeKey":"must-not-project"}}"#;

fn fixture_transport(status: String, prefs: String) -> Box<native::Transport> {
    Box::new(
        move |method: &str, path: &str, _body: Option<&[u8]>, _d: Instant| {
            if method == "GET" && path == "status" {
                Ok((200, status.as_bytes().to_vec()))
            } else if method == "GET" && path == "prefs" {
                Ok((200, prefs.as_bytes().to_vec()))
            } else {
                Ok((500, Vec::new()))
            }
        },
    )
}

type FakeRun = dyn Fn(&[u8], &str, &[&str], Instant) -> Result<Vec<u8>, String> + Send + Sync;

struct FakeExec {
    f: Box<FakeRun>,
}

impl Executor for FakeExec {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        (self.f)(stdin, cmd, args, deadline)
    }
}

fn boom_exec() -> FakeExec {
    FakeExec {
        f: Box::new(|_, _, _, _| panic!("unexpected native mutation")),
    }
}

fn control_options(name: &str, runtime: bool) -> (control::Options, PathBuf) {
    let parent = scratch(name);
    (
        control::Options {
            state_dir: parent.join("soda-tailnet"),
            uid: policy::current_uid(),
            runtime,
            socket: parent.join("sock"),
            cli: "/usr/bin/tailscale".to_string(),
            libexec: "/usr/libexec/soda".to_string(),
            curl: "/usr/bin/curl".to_string(),
        },
        parent,
    )
}

fn token_ok() -> (u16, Vec<u8>) {
    (
        200,
        br#"{"access_token":"synthetic-bearer","token_type":"Bearer","expires_in":3600}"#.to_vec(),
    )
}

fn rfc3339(t: std::time::SystemTime) -> String {
    let secs = t.duration_since(std::time::UNIX_EPOCH).unwrap().as_secs() as i64;
    let days = secs.div_euclid(86400);
    let rem = secs.rem_euclid(86400);
    let z = days + 719468;
    let era = z.div_euclid(146097);
    let doe = z.rem_euclid(146097);
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let mut y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    if m <= 2 {
        y += 1;
    }
    format!(
        "{y:04}-{m:02}-{d:02}T{:02}:{:02}:{:02}Z",
        rem / 3600,
        rem % 3600 / 60,
        rem % 60
    )
}

fn key_response(created: &str, expires: &str) -> String {
    format!(
        r#"{{"id":"synthetic-id","key":"tskey-auth-synthetic-only","created":"{created}","expires":"{expires}","capabilities":{{"devices":{{"create":{{"reusable":false,"ephemeral":true,"preauthorized":false,"tags":["tag:soda-project"]}}}}}}}}"#
    )
}

fn key_ok() -> (u16, Vec<u8>) {
    use std::time::{Duration, SystemTime};
    let now = SystemTime::now();
    (
        200,
        key_response(&rfc3339(now), &rfc3339(now + Duration::from_secs(300))).into_bytes(),
    )
}

fn provider_fixture(
    calls: std::sync::Arc<std::sync::atomic::AtomicU32>,
) -> Box<provider::ProviderTransport> {
    use std::sync::atomic::Ordering;
    Box::new(
        move |req: provider::ProviderRequest, _d: Instant| match req {
            provider::ProviderRequest::Token {
                client_id,
                client_secret,
                tags,
            } => {
                assert_eq!(client_id, "synthetic-client");
                assert_eq!(client_secret, "tskey-client-soda-synthetic-secret");
                assert_eq!(tags, ["tag:soda-project".to_string()]);
                Ok(token_ok())
            }
            provider::ProviderRequest::KeyCreate {
                tailnet,
                tags,
                preauthorized,
                token,
            } => {
                calls.fetch_add(1, Ordering::SeqCst);
                assert_eq!(tailnet, "soda.example.test");
                assert_eq!(tags, ["tag:soda-project".to_string()]);
                assert!(!preauthorized);
                assert_eq!(token, "synthetic-bearer");
                Ok(key_ok())
            }
        },
    )
}

const GO_SETTINGS: &str = r#"{"host":{"tailnet":"soda.example.test","magic_dns_enabled":true,"revision":"c2325fcf010801018acfa617389a67457360b7763a3b7343869ff73f5c4ad304","state":"Running","have_node_key":true,"expired":false,"dns_name":"host.example.ts.net","addresses":["100.64.0.1"],"peers":[{"id":"peer","dns_name":"exit.example.ts.net","addresses":["100.64.0.2"],"online":true,"exit_node":true,"expired":false}],"health_issues":1,"preferences":{"want_running":true,"exit_node_id":"","exit_node_ip":"","allow_lan":false,"advertise_exit_node":false}},"host_unavailable":false,"enrollment":{"revision":"0","binding":"","tailnet":"","tags":[],"configured":false,"admission":false,"default":false,"preauthorized":false,"credential_checked":false,"enrollment_verified":false,"runtime_supported":false}}"#;

const GO_OPTIONS: &str =
    r#"{"revision":"0","binding":"","tailnet":"","available":false,"default":false}"#;

#[test]
fn control_settings_and_options_match_go_bytes() {
    let (opts, _) = control_options("control-settings", false);
    let mut c = control::Control::new(boom_exec(), opts);
    c.local_stub = Some(fixture_transport(
        NATIVE_STATUS.to_string(),
        NATIVE_PREFS.to_string(),
    ));
    assert_eq!(c.settings(soon(5000)).unwrap(), GO_SETTINGS.as_bytes());
    assert_eq!(c.options(soon(5000)).unwrap(), GO_OPTIONS.as_bytes());
    // Unavailable LocalAPI is projected, not an error.
    c.local_stub = Some(Box::new(|_, _, _, _| Ok((500, Vec::new()))));
    let body = c.settings(soon(5000)).unwrap();
    let text = String::from_utf8(body).unwrap();
    assert!(
        text.contains(r#""host":null,"host_unavailable":true"#),
        "{text}"
    );
}

#[test]
fn control_host_action_cycle() {
    let (opts, _) = control_options("control-host", false);
    let mut c = control::Control::new(boom_exec(), opts);
    c.local_stub = Some(fixture_transport(
        NATIVE_STATUS.to_string(),
        NATIVE_PREFS.to_string(),
    ));
    // Malformed bodies and stale revisions fail before any native work.
    let e = c.host_action(b"{}", soon(5000)).expect_err("accepted");
    assert!(e.contains("invalid request"), "{e}");
    let e = c
        .host_action(
            format!(r#"{{"action":"signin","revision":"{}"}}"#, "f".repeat(64)).as_bytes(),
            soon(5000),
        )
        .expect_err("stale");
    assert!(e.contains("conflict"), "{e}");
    let auth = c
        .host_action(
            format!(r#"{{"action":"authentication","revision":"{GO_REVISION}"}}"#).as_bytes(),
            soon(5000),
        )
        .unwrap();
    let text = String::from_utf8(auth).unwrap();
    assert!(text.contains(r#""outcome":"observed""#), "{text}");
    assert!(
        text.contains("https://login.tailscale.com/a/synthetic"),
        "{text}"
    );
    // Confirmed mutation with a failing readback: confirmed + unavailable.
    let (opts, _) = control_options("control-host2", false);
    let c = control::Control::new(
        FakeExec {
            f: Box::new(|_, cmd, args, _| {
                assert_eq!(cmd, "/usr/bin/tailscale");
                assert_eq!(args[1], "set");
                Ok(Vec::new())
            }),
        },
        opts,
    );
    // First observe succeeds, then the daemon goes away.
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let reads = Arc::new(AtomicU32::new(0));
    let mut c = c;
    c.local_stub = Some(Box::new({
        let reads = reads.clone();
        move |method: &str, path: &str, _body: Option<&[u8]>, _d: Instant| {
            if reads.fetch_add(1, Ordering::SeqCst) < 2 {
                if path == "status" {
                    return Ok((200, NATIVE_STATUS.as_bytes().to_vec()));
                }
                return Ok((200, NATIVE_PREFS.as_bytes().to_vec()));
            }
            assert_eq!(method, "GET");
            Err("synthetic secret error".to_string())
        }
    }));
    let out = c
        .host_action(
            format!(
                r#"{{"action":"advertise-exit-node","revision":"{GO_REVISION}","confirm":"advertise-exit-node","advertise":true}}"#
            )
            .as_bytes(),
            soon(5000),
        )
        .unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(text.contains(r#""outcome":"confirmed""#), "{text}");
    assert!(text.contains(r#""readback_unavailable":true"#), "{text}");
    assert!(!text.contains("synthetic"), "{text}");
}

#[test]
fn control_enrollment_save_check_rotate_disable() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let (opts, _) = control_options("control-enroll", false);
    let mut c = control::Control::new(boom_exec(), opts);
    let calls = Arc::new(AtomicU32::new(0));
    c.provider_stub = Some(provider_fixture(calls.clone()));
    let save = br#"{"action":"save","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}"#;
    let out = c.enrollment(save, soon(5000)).unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(text.contains(r#""saved":true"#), "{text}");
    assert!(!text.contains("synthetic"), "{text}");
    // Save again with the same body: stale revision conflicts.
    let e = c.enrollment(save, soon(5000)).expect_err("stale");
    assert!(e.contains("conflict"), "{e}");
    // Rotate + check + disable against the live revision.
    let rev = extract_revision(&text);
    let rotate = format!(
        r#"{{"action":"rotate","revision":"{rev}","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}}"#
    );
    let out = c.enrollment(rotate.as_bytes(), soon(5000)).unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(text.contains(r#""saved":true"#), "{text}");
    let rev2 = extract_revision(&text);
    assert_ne!(rev2, rev);
    let check = format!(
        r#"{{"action":"check","revision":"{rev2}","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}}"#
    );
    let out = c.enrollment(check.as_bytes(), soon(5000)).unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(text.contains(r#""saved":false"#), "{text}");
    assert!(text.contains(r#""credential_checked":true"#), "{text}");
    let out = c
        .enrollment(
            format!(r#"{{"action":"disable","revision":"{rev2}"}}"#).as_bytes(),
            soon(5000),
        )
        .unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(text.contains(r#""admission":false"#), "{text}");
    // Default-on without the runtime is unsupported before any provider call.
    let e = c
        .enrollment(
            br#"{"action":"default","revision":"0","default":true}"#,
            soon(5000),
        )
        .expect_err("unsupported default");
    assert!(e.contains("unsupported"), "{e}");
    assert_eq!(calls.load(Ordering::SeqCst), 0);
    // Unknown fields and bad credentials are invalid, without provider calls.
    for bad in [
        br#"{"action":"save","revision":"0","bogus":1}"#.as_slice(),
        br#"{"action":"save","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"short"}"#.as_slice(),
    ] {
        let e = c.enrollment(bad, soon(5000)).expect_err("accepted");
        assert!(e.contains("invalid request"), "{e}");
    }
}

// Minimal `revision` extractor for test plumbing (not a product decoder).
fn extract_revision(text: &str) -> String {
    let key = r#""revision":""#;
    let start = text.find(key).unwrap() + key.len();
    text[start..start + 32].to_string()
}

fn seed_enrollment(c: &control::Control<FakeExec>) -> String {
    let out = c
        .enrollment(
            br#"{"action":"save","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}"#,
            soon(10000),
        )
        .unwrap();
    let text = String::from_utf8(out).unwrap();
    let key = r#""binding":""#;
    let start = text.find(key).unwrap() + key.len();
    text[start..start + 32].to_string()
}

#[test]
fn control_project_and_binding_through_trait() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let (opts, _) = control_options("control-trait", true);
    let mut c = control::Control::new_project_control(boom_exec(), opts);
    let calls = Arc::new(AtomicU32::new(0));
    c.provider_stub = Some(provider_fixture(calls.clone()));
    let binding = seed_enrollment(&c);
    let project = format!("p{}", "a".repeat(24));
    let cid = "b".repeat(64);
    // Drive the companion surface through the trait object, as the adapter
    // and companion do.
    let t: &dyn TailnetControl = &c;
    let view = t
        .project(
            &ProjectRequest {
                project: project.clone(),
                action: "enable".to_string(),
                revision: "0".to_string(),
                binding: binding.clone(),
                confirm_id: project.clone(),
            },
            &cid,
            soon(5000),
        )
        .unwrap();
    assert!(view.enabled && wire::validate_project_view(&view).is_ok());
    let target = RunTarget {
        project: project.clone(),
        container: cid.clone(),
        run: "e".repeat(64),
    };
    let rb = t.run_binding(&target, soon(5000)).unwrap();
    assert!(rb.enabled && rb.admission);
    let consumed = Arc::new(AtomicU32::new(0));
    t.enroll_run(
        &target,
        &|_| Ok(()),
        &|_, key| {
            assert_eq!(key, "tskey-auth-synthetic-only");
            consumed.fetch_add(1, Ordering::SeqCst);
            Ok(())
        },
        soon(10000),
    )
    .unwrap();
    assert_eq!(consumed.load(Ordering::SeqCst), 1);
    assert_eq!(calls.load(Ordering::SeqCst), 1);
}

#[test]
fn control_enroll_run_fences_identity_and_uncertainty() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    for mode in [
        "replaced cid",
        "invalid incarnation",
        "binding replaced",
        "admission closed",
        "provider failure",
        "changed during key",
        "consume failure",
    ] {
        let (opts, _) = control_options(&format!("control-fence-{mode}"), true);
        let mut c = control::Control::new_project_control(boom_exec(), opts);
        let calls = Arc::new(AtomicU32::new(0));
        c.provider_stub = Some(provider_fixture(calls.clone()));
        let binding = seed_enrollment(&c);
        let project = format!("p{}", "a".repeat(24));
        let cid = "b".repeat(64);
        c.project(
            &ProjectRequest {
                project: project.clone(),
                action: "enable".to_string(),
                revision: "0".to_string(),
                binding: binding.clone(),
                confirm_id: project.clone(),
            },
            &cid,
            soon(5000),
        )
        .unwrap();
        let mut target = RunTarget {
            project: project.clone(),
            container: cid.clone(),
            run: "e".repeat(64),
        };
        if mode == "replaced cid" {
            target.container = "f".repeat(64);
        }
        if mode == "invalid incarnation" {
            target.run = "../other".to_string();
        }
        if mode == "binding replaced" {
            let opts_text = c.options(soon(5000)).unwrap();
            let rev = extract_revision(&String::from_utf8(opts_text).unwrap());
            let save = format!(
                r#"{{"action":"save","revision":"{rev}","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}}"#
            );
            c.enrollment(save.as_bytes(), soon(5000)).unwrap();
        }
        if mode == "admission closed" {
            let opts_text = c.options(soon(5000)).unwrap();
            let rev = extract_revision(&String::from_utf8(opts_text).unwrap());
            c.enrollment(
                format!(r#"{{"action":"disable","revision":"{rev}"}}"#).as_bytes(),
                soon(5000),
            )
            .unwrap();
        }
        if mode == "provider failure" {
            c.provider_stub = Some(Box::new({
                let calls = calls.clone();
                move |req, _| match req {
                    provider::ProviderRequest::Token { .. } => Ok(token_ok()),
                    provider::ProviderRequest::KeyCreate { .. } => {
                        calls.fetch_add(1, Ordering::SeqCst);
                        Err("private provider text".to_string())
                    }
                }
            }));
        }
        let consumed = Arc::new(AtomicU32::new(0));
        let checks = Arc::new(AtomicU32::new(0));
        let recheck = {
            let checks = checks.clone();
            move |_: Instant| {
                let n = checks.fetch_add(1, Ordering::SeqCst) + 1;
                if mode == "changed during key" && n > 1 {
                    return Err("changed".to_string());
                }
                Ok(())
            }
        };
        let consume = {
            let consumed = consumed.clone();
            move |_: Instant, _: &str| {
                consumed.fetch_add(1, Ordering::SeqCst);
                if mode == "consume failure" {
                    return Err("synthetic".to_string());
                }
                Ok(())
            }
        };
        let e = c
            .enroll_run(&target, &recheck, &consume, soon(10000))
            .expect_err(mode);
        assert!(
            !e.contains("private") && !e.contains("synthetic"),
            "{mode}: {e}"
        );
        if mode != "consume failure" {
            assert_eq!(consumed.load(Ordering::SeqCst), 0, "{mode}");
        }
        if ["provider failure", "changed during key", "consume failure"].contains(&mode) {
            assert_eq!(calls.load(Ordering::SeqCst), 1, "{mode}");
        } else {
            assert_eq!(calls.load(Ordering::SeqCst), 0, "{mode}");
        }
    }
}

#[test]
fn control_enroll_run_serial_explicit_requests_without_journal() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let (opts, parent) = control_options("control-serial", true);
    let mut c = control::Control::new_project_control(boom_exec(), opts);
    let calls = Arc::new(AtomicU32::new(0));
    c.provider_stub = Some(provider_fixture(calls.clone()));
    let binding = seed_enrollment(&c);
    let project = format!("p{}", "a".repeat(24));
    let cid = "b".repeat(64);
    c.project(
        &ProjectRequest {
            project: project.clone(),
            action: "enable".to_string(),
            revision: "0".to_string(),
            binding: binding.clone(),
            confirm_id: project.clone(),
        },
        &cid,
        soon(5000),
    )
    .unwrap();
    let target = RunTarget {
        project: project.clone(),
        container: cid.clone(),
        run: "e".repeat(64),
    };
    let consumed = Arc::new(AtomicU32::new(0));
    std::thread::scope(|s| {
        for _ in 0..8 {
            s.spawn(|| {
                c.enroll_run(
                    &target,
                    &|_| Ok(()),
                    &|_, key| {
                        assert_eq!(key, "tskey-auth-synthetic-only");
                        consumed.fetch_add(1, Ordering::SeqCst);
                        Ok(())
                    },
                    soon(15000),
                )
                .unwrap();
            });
        }
    });
    assert_eq!(calls.load(Ordering::SeqCst), 8);
    assert_eq!(consumed.load(Ordering::SeqCst), 8);
    let dir = parent.join("soda-tailnet");
    for entry in std::fs::read_dir(&dir).unwrap() {
        let name = entry.unwrap().file_name().to_string_lossy().into_owned();
        assert!(!name.starts_with("attempt-"), "{name}");
    }
    let body = std::fs::read(dir.join(format!("project-{project}.json"))).unwrap();
    let text = String::from_utf8(body).unwrap();
    assert!(!text.contains("active_run") && !text.contains("tskey") && !text.contains("bearer"));
}

#[test]
fn control_failed_enrollment_can_be_explicitly_retried() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    for failure in ["token", "key", "consume"] {
        let (opts, parent) = control_options(&format!("control-retry-{failure}"), true);
        let mut c = control::Control::new_project_control(boom_exec(), opts);
        let calls = Arc::new(AtomicU32::new(0));
        c.provider_stub = Some(provider_fixture(calls.clone()));
        let binding = seed_enrollment(&c);
        let project = format!("p{}", "a".repeat(24));
        let cid = "b".repeat(64);
        c.project(
            &ProjectRequest {
                project: project.clone(),
                action: "enable".to_string(),
                revision: "0".to_string(),
                binding: binding.clone(),
                confirm_id: project.clone(),
            },
            &cid,
            soon(5000),
        )
        .unwrap();
        let target = RunTarget {
            project: project.clone(),
            container: cid.clone(),
            run: "e".repeat(64),
        };
        let dir = parent.join("soda-tailnet");
        let before = std::fs::read(dir.join(format!("project-{project}.json"))).unwrap();
        if failure == "token" {
            c.provider_stub = Some(Box::new(|req, _| match req {
                provider::ProviderRequest::Token { .. } => Err("down".to_string()),
                other => panic!("unexpected {other:?}"),
            }));
        }
        if failure == "key" {
            c.provider_stub = Some(Box::new({
                let calls = calls.clone();
                move |req, _| match req {
                    provider::ProviderRequest::Token { .. } => Ok(token_ok()),
                    provider::ProviderRequest::KeyCreate { .. } => {
                        calls.fetch_add(1, Ordering::SeqCst);
                        Err("down".to_string())
                    }
                }
            }));
        }
        let fail_consume = |_: Instant, _: &str| Err("down".to_string());
        let ok_consume = |_: Instant, key: &str| {
            assert!(key.starts_with("tskey-auth-"));
            Ok(())
        };
        assert!(c
            .enroll_run(
                &target,
                &|_| Ok(()),
                &(if failure == "consume" {
                    fail_consume
                } else {
                    ok_consume
                }),
                soon(10000)
            )
            .is_err());
        // Passive reads neither replay nor change saved policy.
        let count = calls.load(Ordering::SeqCst);
        c.run_binding(&target, soon(5000)).unwrap();
        assert_eq!(calls.load(Ordering::SeqCst), count);
        c.provider_stub = Some(provider_fixture(calls.clone()));
        c.enroll_run(&target, &|_| Ok(()), &ok_consume, soon(10000))
            .unwrap();
        assert_eq!(calls.load(Ordering::SeqCst), count + 1);
        assert_eq!(
            std::fs::read(dir.join(format!("project-{project}.json"))).unwrap(),
            before
        );
        assert_eq!(std::fs::read_dir(&dir).unwrap().count(), 2);
    }
}
