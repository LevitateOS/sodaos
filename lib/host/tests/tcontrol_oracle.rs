//! Oracle tests for the PR26 tailnet control-plane port.
//!
//! The `tcontrol*.rs` modules are compiled into the lib AND included here
//! via `#[path]`; shared crate items are re-exported through shim modules
//! so the `crate::` paths inside the ported files resolve unchanged. The
//! duplication is deliberate: the oracle drives `pub(crate)` test seams
//! (stub transports, sync hook) that are unreachable through the public
//! surface, so it must compile the same sources as its own crate.
//!
//! Test state lives under `target/tcontrol-test/` (never `/tmp`).

mod json {
    pub use soda_host::json::*;
}
mod project {
    pub use soda_host::project::*;
}
mod sha256 {
    pub use soda_host::sha256::*;
}
mod tailnet_companion {
    pub use soda_host::tailnet_companion::*;
}
mod tailnet_domain {
    pub use soda_host::tailnet_domain::*;
}

#[path = "../src/tcontrol.rs"]
mod tcontrol;
#[path = "../src/tcontrol_enroll.rs"]
mod tcontrol_enroll;
#[path = "../src/tcontrol_native.rs"]
mod tcontrol_native;
#[path = "../src/tcontrol_policy.rs"]
mod tcontrol_policy;
#[path = "../src/tcontrol_provider.rs"]
mod tcontrol_provider;
#[path = "../src/tcontrol_wire.rs"]
mod tcontrol_wire;

use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

static SCRATCH_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Unique scratch directory under the package `target/` dir.
fn scratch(name: &str) -> PathBuf {
    let id = SCRATCH_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir =
        PathBuf::from("target/tcontrol-test").join(format!("{}-{}-{id}", name, std::process::id()));
    std::fs::create_dir_all(&dir).expect("scratch dir");
    dir
}

use tcontrol as control;
use tcontrol_enroll as enroll;
use tcontrol_native as native;
use tcontrol_policy as policy;
use tcontrol_provider as provider;
use tcontrol_wire as wire;

fn soon(ms: u64) -> Instant {
    Instant::now() + Duration::from_millis(ms)
}

fn policy_store(name: &str, runtime: bool) -> (policy::PolicyStore, PathBuf) {
    let parent = scratch(name);
    let dir = parent.join("soda-tailnet");
    (
        policy::PolicyStore::new(dir, policy::current_uid(), runtime),
        parent,
    )
}

fn accept_check(_: Instant, _: &wire::EnrollmentRequest) -> Result<(), String> {
    Ok(())
}

// ---------- Patterns ----------

#[test]
fn wire_revision_and_hex_patterns() {
    assert!(wire::valid_revision("0"));
    assert!(wire::valid_revision(&"a".repeat(32)));
    assert!(!wire::valid_revision(""));
    assert!(!wire::valid_revision("00"));
    assert!(!wire::valid_revision(&"a".repeat(31)));
    assert!(!wire::valid_revision(&"a".repeat(33)));
    assert!(!wire::valid_revision(&"A".repeat(32)));
    assert!(!wire::valid_revision(&"g".repeat(32)));
    assert!(wire::is_hex32(&"f".repeat(32)));
    assert!(!wire::is_hex32("0"));
}

#[test]
fn wire_tag_client_and_network_patterns() {
    assert!(wire::valid_tag("tag:soda-project"));
    assert!(wire::valid_tag("tag:A"));
    assert!(!wire::valid_tag("tag:"));
    assert!(!wire::valid_tag("tag:1abc"));
    assert!(!wire::valid_tag("tag:a_b"));
    assert!(!wire::valid_tag("--flag"));
    assert!(!wire::valid_tag(&format!("tag:a{}", "b".repeat(63))));
    assert!(wire::valid_client_secret(
        "tskey-client-soda-synthetic-secret"
    ));
    assert!(!wire::valid_client_secret("tskey-client-short"));
    assert!(!wire::valid_client_secret(
        "tskey-client-soda-synthetic-secret?baseURL=https://attacker.invalid"
    ));
    assert!(!wire::valid_client_secret(
        "tskey-client-soda-synthetic-secret?ephemeral=false"
    ));
    assert!(!wire::valid_client_secret(
        "tskey-auth-soda-synthetic-secret"
    ));
    assert!(wire::valid_auth_key("tskey-auth-synthetic-only"));
    assert!(!wire::valid_auth_key("tskey-client-synthetic-only"));
    assert!(wire::valid_client_id("synthetic-client"));
    assert!(!wire::valid_client_id(""));
    assert!(!wire::valid_client_id(&"a".repeat(129)));
    assert!(!wire::valid_client_id("has space"));
    assert!(wire::valid_network("soda.example.test"));
    assert!(wire::valid_network("a"));
    assert!(!wire::valid_network(""));
    assert!(!wire::valid_network("-"));
    assert!(!wire::valid_network("../other"));
    assert!(!wire::valid_network(".leading-dot"));
    assert!(!wire::valid_network(&"a".repeat(254)));
}

// ---------- DNS ----------

#[test]
fn wire_magic_dns_vectors() {
    assert_eq!(
        wire::canonical_magic_dns_name("Atlas.Example.ts.net.").unwrap(),
        "atlas.example.ts.net"
    );
    assert_eq!(
        wire::canonical_magic_dns_name("  host.example.ts.net  ").unwrap(),
        "host.example.ts.net"
    );
    for bad in [
        "",
        "nodot",
        "atlas.local",
        "atlas.LOCAL.",
        ".leading.example.test",
        "trailing..example.test",
        "-lead.example.test",
        "trail-.example.test",
        "under_score.example.test",
        &"a".repeat(64),
        &format!("{}.example.test", "a".repeat(64)),
        &format!("{}.com", "a".repeat(250)),
    ] {
        assert!(
            wire::canonical_magic_dns_name(bad).is_err(),
            "accepted {bad:?}"
        );
    }
}

// ---------- Addresses ----------

#[test]
fn wire_address_vectors() {
    let ok = |addrs: &[&str]| {
        let v: Vec<String> = addrs.iter().map(|s| s.to_string()).collect();
        wire::check_addresses(&v).expect("accept");
    };
    ok(&["100.64.0.1"]);
    ok(&["10.8.0.1", "192.168.1.1"]);
    ok(&["fd7a:115c:a1e0::1"]);
    ok(&["::ffff:1.2.3.4"]);
    ok(&["fd7a:115c:a1e0::1%eth0"]);
    let bad = |addrs: &[&str]| {
        let v: Vec<String> = addrs.iter().map(|s| s.to_string()).collect();
        assert!(wire::check_addresses(&v).is_err(), "accepted {addrs:?}");
    };
    bad(&["1.2.3.04"]);
    bad(&["256.1.1.1"]);
    bad(&["127.0.0.1"]);
    bad(&["0.0.0.0"]);
    bad(&["255.255.255.255"]);
    bad(&["224.0.0.1"]);
    bad(&["169.254.1.1"]);
    bad(&["::1"]);
    bad(&["::"]);
    bad(&["ff02::1"]);
    bad(&["fe80::1"]);
    bad(&["1.2.3.4%eth0"]);
    bad(&["fe80::1%"]);
    bad(&["fe80:0000::1"]);
    bad(&["not-an-ip"]);
    bad(&[""]);
    let seventeen: Vec<String> = (0..17).map(|i| format!("100.64.0.{i}")).collect();
    assert!(wire::check_addresses(&seventeen).is_err());
    assert!(wire::check_addresses(&[]).unwrap().is_empty());
}

#[test]
fn wire_prefix_exit_ip_and_first_ipv4() {
    for good in [
        "10.8.0.0/16",
        "0.0.0.0/0",
        "::/0",
        "10.8.0.1/16",
        "1.2.3.4/32",
        "fe80:0000:0000:0000:0000:0000:0000:0001/128",
        "::ffff:1.2.3.4/128",
    ] {
        assert!(wire::valid_prefix(good), "rejected {good}");
    }
    for bad in [
        "",
        "abc",
        "1.2.3.4/33",
        "1.2.3.4/-1",
        "1.2.3.4/",
        "/16",
        "fe80::1%eth0/64",
        "1.2.3.4%eth0/24",
        "10.08.0.0/16",
        "1.2.3.4/16/8",
        "::/129",
    ] {
        assert!(!wire::valid_prefix(bad), "accepted {bad}");
    }
    assert!(wire::valid_exit_node_ip("100.64.0.2"));
    assert!(!wire::valid_exit_node_ip(""));
    assert!(!wire::valid_exit_node_ip("127.0.0.1"));
    assert!(!wire::valid_exit_node_ip("1.2.3.04"));
    assert!(wire::parseable_addr("100.64.0.2"));
    assert!(wire::parseable_addr("fe80::1%eth0"));
    assert!(!wire::parseable_addr("nope"));
    let ips = ["fd7a:115c:a1e0::1".to_string(), "100.88.77.66".to_string()];
    assert_eq!(wire::first_ipv4(&ips).as_deref(), Some("100.88.77.66"));
    assert!(wire::first_ipv4(&ips[..1]).is_none());
    assert!(wire::first_ipv4(&[]).is_none());
}

#[test]
fn wire_peer_view() {
    let p = wire::peer_view(
        "peer",
        "Exit.Example.ts.net.",
        &["100.64.0.2".to_string()],
        true,
        true,
        false,
    )
    .unwrap();
    assert_eq!(p.dns_name, "exit.example.ts.net");
    assert!(p.online && p.exit_node && !p.expired);
    assert!(wire::peer_view(&"i".repeat(129), "", &[], false, false, false).is_err());
    assert!(wire::peer_view("a\nb", "", &[], false, false, false).is_err());
    assert!(wire::peer_view("", "bad.local", &[], false, false, false).is_err());
    assert!(wire::peer_view("", "", &["127.0.0.1".to_string()], false, false, false).is_err());
}

// ---------- Strict decoders ----------

#[test]
fn wire_host_request_decode() {
    let rev = "a".repeat(64);
    let r = wire::decode_host_request(
        format!(r#"{{"action":"exit-node","revision":"{rev}","confirm":"exit-node","exit_node":"100.64.0.2","allow_lan":true}}"#)
            .as_bytes(),
    )
    .unwrap();
    assert_eq!(r.exit_node.as_deref(), Some("100.64.0.2"));
    assert_eq!(r.allow_lan, Some(true));
    assert_eq!(r.advertise, None);
    // Null collapses to absent, exactly like Go pointer binding.
    let r = wire::decode_host_request(
        format!(r#"{{"action":"signin","revision":"{rev}","exit_node":null,"allow_lan":null}}"#)
            .as_bytes(),
    )
    .unwrap();
    assert_eq!(r.exit_node, None);
    assert_eq!(r.allow_lan, None);
    for bad in [
        format!(r#"{{"action":"signin","revision":"{rev}","bogus":1}}"#),
        format!(r#"{{"action":"signin","action":"logout","revision":"{rev}"}}"#),
        format!(r#"{{"action":5,"revision":"{rev}"}}"#),
        format!(r#"{{"action":"signin","revision":"{rev}","allow_lan":"yes"}}"#),
        r#"["action"]"#.to_string(),
        "null".to_string(),
        String::new(),
        format!(r#"{{"action":"signin","revision":"{rev}"}} trailing"#),
    ] {
        let e = wire::decode_host_request(bad.as_bytes()).expect_err("accepted");
        assert!(e.contains("invalid request"), "wrong cause: {e}");
    }
    let big = format!(
        r#"{{"action":"signin","revision":"{rev}","confirm":"{}"}}"#,
        "x".repeat(65536)
    );
    assert!(wire::decode_host_request(big.as_bytes()).is_err());
}

#[test]
fn wire_enrollment_request_decode() {
    let r = wire::decode_enrollment_request(
        br#"{"action":"save","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-soda-synthetic-secret"}"#,
    )
    .unwrap();
    assert_eq!(r.tags, Some(vec!["tag:soda-project".to_string()]));
    assert_eq!(r.preauthorized, Some(false));
    assert_eq!(r.default, None);
    // `"tags":null` decodes to nil, `[]` to a present empty list.
    let r = wire::decode_enrollment_request(
        br#"{"action":"disable","revision":"0","tags":null,"default":null}"#,
    )
    .unwrap();
    assert_eq!(r.tags, None);
    assert_eq!(r.default, None);
    let r = wire::decode_enrollment_request(br#"{"action":"disable","revision":"0","tags":[]}"#)
        .unwrap();
    assert_eq!(r.tags, Some(vec![]));
    for bad in [
        r#"{"action":"save","revision":"0","tags":["tag:a",5]}"#,
        r#"{"action":"save","revision":"0","tags":"tag:a"}"#,
        r#"{"action":"save","revision":"0","preauthorized":"false"}"#,
        r#"{"action":"save","revision":"0","extra":null}"#,
    ] {
        assert!(wire::decode_enrollment_request(bad.as_bytes()).is_err());
    }
    let mut r = wire::decode_enrollment_request(
        br#"{"action":"save","revision":"0","client_secret":"tskey-client-soda-synthetic-secret"}"#,
    )
    .unwrap();
    assert!(!r.client_secret.is_empty());
    r.zero_secret();
    assert!(r.client_secret.is_empty());
}

#[test]
fn wire_project_selection_decode() {
    // NOTE: ProjectRequest decode lives in the daemon adapter (dbackend.rs),
    // not here; only the create-time selection shape is decoded locally.
    let s = wire::decode_project_selection(br#"{"enabled":true,"revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","binding":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}"#).unwrap();
    assert!(s.enabled);
    assert!(
        wire::decode_project_selection(br#"{"enabled":true,"revision":"x","nope":1}"#).is_err()
    );
}

// ---------- Request validators ----------

#[test]
fn wire_host_request_validate() {
    let rev = "a".repeat(64);
    let base = wire::HostRequest {
        action: "signin".to_string(),
        revision: rev.clone(),
        ..Default::default()
    };
    assert!(base.validate().is_ok());
    let mut auth = base.clone();
    auth.action = "authentication".to_string();
    assert!(auth.validate().is_ok());
    for action in ["logout", "refresh-forgejo"] {
        let mut r = base.clone();
        r.action = action.to_string();
        assert!(r.validate().is_err(), "unconfirmed {action}");
        r.confirm = action.to_string();
        assert!(r.validate().is_ok(), "{action}");
    }
    let mut r = base.clone();
    r.action = "exit-node".to_string();
    r.confirm = "exit-node".to_string();
    r.exit_node = Some("100.64.0.2".to_string());
    r.allow_lan = Some(true);
    assert!(r.validate().is_ok());
    r.exit_node = Some(String::new());
    assert!(r.validate().is_err(), "clear with LAN");
    r.allow_lan = Some(false);
    assert!(r.validate().is_ok());
    r.exit_node = Some("127.0.0.1".to_string());
    assert!(r.validate().is_err());
    r.exit_node = Some("1.2.3.04".to_string());
    assert!(r.validate().is_err());
    r.exit_node = Some("100.64.0.2".to_string());
    r.advertise = Some(false);
    assert!(r.validate().is_err(), "mixed fields");
    let mut r = base.clone();
    r.action = "advertise-exit-node".to_string();
    r.confirm = "advertise-exit-node".to_string();
    r.advertise = Some(true);
    assert!(r.validate().is_ok());
    r.exit_node = Some(String::new());
    assert!(r.validate().is_err());
    let mut r = base.clone();
    r.action = "bogus".to_string();
    assert!(r.validate().is_err());
    let mut r = base.clone();
    r.revision = "short".to_string();
    assert!(r.validate().is_err());
    let mut r = base.clone();
    r.confirm = "signin".to_string();
    assert!(r.validate().is_err(), "signin takes no confirm");
}

fn enrollment_fixture() -> wire::EnrollmentRequest {
    wire::EnrollmentRequest {
        action: "save".to_string(),
        revision: "0".to_string(),
        tailnet: "soda.example.test".to_string(),
        tags: Some(vec!["tag:soda-project".to_string()]),
        preauthorized: Some(false),
        client_id: "synthetic-client".to_string(),
        client_secret: "tskey-client-soda-synthetic-secret".to_string(),
        default: None,
    }
}

#[test]
fn wire_enrollment_request_validate() {
    assert!(enrollment_fixture().validate().is_ok());
    // The eight policy-override mutations from Go's
    // TestTailnetInputRefusesCredentialEndpointAndPolicyOverrides.
    let mut r = enrollment_fixture();
    r.client_secret += "?baseURL=https://attacker.invalid";
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.client_secret += "?ephemeral=false";
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.tailnet = "-".to_string();
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.tailnet = "../other".to_string();
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.tags = Some(vec!["tag:z".to_string(), "tag:a".to_string()]);
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.tags = Some(vec!["tag:a".to_string(), "tag:a".to_string()]);
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.tags = Some(vec!["--flag".to_string()]);
    assert!(r.validate().is_err());
    let mut r = enrollment_fixture();
    r.action = "disable".to_string();
    assert!(r.validate().is_err());
    // Toggles.
    let t = wire::EnrollmentRequest {
        action: "default".to_string(),
        revision: "0".to_string(),
        default: Some(true),
        ..Default::default()
    };
    assert!(t.validate().is_ok());
    let mut t = t.clone();
    t.default = None;
    assert!(t.validate().is_err());
    let mut t = t.clone();
    t.action = "disable".to_string();
    assert!(t.validate().is_ok());
    t.default = Some(false);
    assert!(t.validate().is_err());
    let mut r = enrollment_fixture();
    r.action = "rotate".to_string();
    r.default = Some(true);
    assert!(r.validate().is_err(), "mutation with default flag");
}

#[test]
fn wire_project_selection_validate() {
    // NOTE: ProjectRequest validation is enforced inside Control::project
    // (covered by the policy tests with constructed requests); the
    // adapter-facing shape lives in dbackend.rs.
    let sel = wire::ProjectSelection {
        enabled: true,
        revision: "a".repeat(32),
        binding: "b".repeat(32),
    };
    assert!(sel.validate().is_ok());
    let sel = wire::ProjectSelection::default();
    assert!(sel.validate().is_ok());
    let sel = wire::ProjectSelection {
        revision: "a".repeat(32),
        ..Default::default()
    };
    assert!(sel.validate().is_err());
}

// ---------- Auth URL ----------

#[test]
fn wire_authentication_url() {
    assert_eq!(
        wire::authentication_url("https://login.tailscale.com/a/synthetic"),
        "https://login.tailscale.com/a/synthetic"
    );
    // Uppercase scheme parses and normalizes, like Go's url.Parse.
    assert_eq!(
        wire::authentication_url("HTTPS://login.tailscale.com/a/synthetic"),
        "https://login.tailscale.com/a/synthetic"
    );
    for bad in [
        "https://evil.test/a/secret",
        "http://login.tailscale.com/a/secret",
        "https://login.tailscale.com.evil.test/a/secret",
        "https://login.tailscale.com/a/secret?token=secret",
        "https://login.tailscale.com/a/secret?",
        "https://user@login.tailscale.com/a/secret",
        "https://login.tailscale.com:443/a/secret",
        "https://LOGIN.tailscale.com/a/secret",
        "https://login.tailscale.com/a/%41bc",
        "https://login.tailscale.com/a/bc#frag",
        "https://login.tailscale.com/a/",
        "https://login.tailscale.com/a",
        "https://login.tailscale.com//a/bc",
        "https://login.tailscale.com/a/b c",
        "https://login.tailscale.com/a/bc\n",
        "",
    ] {
        assert_eq!(wire::authentication_url(bad), "", "accepted {bad:?}");
    }
    assert_eq!(
        wire::authentication_url(&format!(
            "https://login.tailscale.com/a/{}",
            "x".repeat(2048)
        )),
        ""
    );
}

// ---------- View validators ----------

fn host_fixture() -> wire::HostView {
    wire::HostView {
        tailnet: "soda.example.test".to_string(),
        magic_dns_enabled: true,
        revision: "a".repeat(64),
        state: "Running".to_string(),
        have_node_key: true,
        dns_name: "host.example.ts.net".to_string(),
        addresses: vec!["100.64.0.1".to_string()],
        peers: vec![wire::Peer {
            id: "peer".to_string(),
            dns_name: "exit.example.ts.net".to_string(),
            addresses: vec!["100.64.0.2".to_string()],
            online: true,
            exit_node: true,
            expired: false,
        }],
        health_issues: 1,
        preferences: wire::HostPreferences {
            want_running: true,
            ..Default::default()
        },
        ..Default::default()
    }
}

fn enrollment_view_fixture() -> wire::EnrollmentView {
    wire::EnrollmentView {
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        tailnet: "soda.example.test".to_string(),
        tags: vec!["tag:soda-project".to_string()],
        configured: true,
        admission: true,
        preauthorized: false,
        credential_checked: true,
        ..Default::default()
    }
}

#[test]
fn wire_view_validators() {
    assert!(host_fixture().validate().is_ok());
    assert!(enrollment_view_fixture().validate().is_ok());
    let fresh = wire::EnrollmentView {
        revision: "0".to_string(),
        ..Default::default()
    };
    assert!(fresh.validate().is_ok());
    let settings = wire::SettingsView {
        host: Some(host_fixture()),
        host_unavailable: false,
        enrollment: enrollment_view_fixture(),
    };
    assert!(settings.validate().is_ok());
    let mut bad = settings.clone();
    bad.host_unavailable = true;
    assert!(bad.validate().is_err());
    let mut bad = host_fixture();
    bad.state = "UnknownState".to_string();
    assert!(bad.validate().is_err());
    let mut bad = host_fixture();
    bad.peers.push(bad.peers[0].clone());
    assert!(bad.validate().is_err(), "duplicate peer id");
    let mut bad = host_fixture();
    bad.preferences.exit_node_ip = "127.0.0.1".to_string();
    assert!(bad.validate().is_err());
    let mut bad = host_fixture();
    bad.tailnet = String::new();
    assert!(bad.validate().is_err(), "Running without tailnet");
    let mut bad = enrollment_view_fixture();
    bad.tags = vec!["tag:z".to_string(), "tag:a".to_string()];
    assert!(bad.validate().is_err());
    let mut bad = enrollment_view_fixture();
    bad.default = true;
    bad.admission = false;
    assert!(bad.validate().is_err());
    let result = wire::HostResult {
        outcome: "pending".to_string(),
        host: Some(host_fixture()),
        auth_url: "https://login.tailscale.com/a/synthetic".to_string(),
        ..Default::default()
    };
    assert!(result.validate().is_ok());
    let mut bad = result.clone();
    bad.auth_url = String::new();
    assert!(bad.validate().is_err(), "pending without URL");
    let mut bad = result.clone();
    bad.auth_url = "https://evil.test/a/x".to_string();
    assert!(bad.validate().is_err());
    let result = wire::HostResult {
        outcome: "confirmed".to_string(),
        readback_unavailable: true,
        ..Default::default()
    };
    assert!(result.validate().is_ok());
    let result = wire::EnrollmentResult {
        outcome: "confirmed".to_string(),
        saved: true,
        credential_checked: true,
        enrollment: enrollment_view_fixture(),
    };
    assert!(result.validate().is_ok());
    let mut bad = result.clone();
    bad.credential_checked = false;
    assert!(bad.validate().is_err());
    let opts = wire::ProjectOptions {
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        tailnet: "soda.example.test".to_string(),
        available: true,
        default: true,
    };
    assert!(opts.validate().is_ok());
    let mut bad = opts.clone();
    bad.available = false;
    assert!(bad.validate().is_err(), "default without available");
    let zero = wire::ProjectOptions {
        revision: "0".to_string(),
        ..Default::default()
    };
    assert!(zero.validate().is_ok());
    let mut bad = zero.clone();
    bad.tailnet = "x".to_string();
    assert!(bad.validate().is_err());
}

#[test]
fn wire_project_view_validator() {
    use soda_host::tailnet_domain::ProjectView;
    let pid = "p0123456789abcdef01234567";
    let off = ProjectView {
        project: pid.to_string(),
        revision: "0".to_string(),
        state: "runtime-unsupported".to_string(),
        outcome: "observed".to_string(),
        ..Default::default()
    };
    assert!(wire::validate_project_view(&off).is_ok());
    let saved = ProjectView {
        project: pid.to_string(),
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        enabled: true,
        saved: true,
        state: "unconfirmed".to_string(),
        outcome: "runtime-unconfirmed".to_string(),
        available_binding: "b".repeat(32),
        available_network: "soda.example.test".to_string(),
        ..Default::default()
    };
    assert!(wire::validate_project_view(&saved).is_ok());
    let connected = ProjectView {
        project: pid.to_string(),
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        enabled: true,
        saved: true,
        state: "connected".to_string(),
        outcome: "queued".to_string(),
        addresses: vec!["100.64.0.2".to_string()],
        dns_name: "project.soda.ts.net".to_string(),
        ..Default::default()
    };
    assert!(wire::validate_project_view(&connected).is_ok());
    let mut bad = connected.clone();
    bad.addresses.clear();
    assert!(wire::validate_project_view(&bad).is_err());
    let mut bad = saved.clone();
    bad.available_network = String::new();
    assert!(wire::validate_project_view(&bad).is_err());
    let mut bad = saved.clone();
    bad.state = "bogus".to_string();
    assert!(wire::validate_project_view(&bad).is_err());
    let mut bad = off.clone();
    bad.outcome = "queued".to_string();
    assert!(wire::validate_project_view(&bad).is_err());
}

// ---------- Encoder goldens ----------

#[test]
fn wire_encoder_goldens() {
    // Hand-derived from the Go struct tags; byte-compared against Go in the
    // differential tests below.
    let opts = wire::ProjectOptions {
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        tailnet: "soda.example.test".to_string(),
        available: true,
        default: false,
    };
    assert_eq!(
        opts.encode(),
        format!(
            r#"{{"revision":"{}","binding":"{}","tailnet":"soda.example.test","available":true,"default":false}}"#,
            "a".repeat(32),
            "b".repeat(32)
        )
    );
    let result = wire::HostResult {
        outcome: "confirmed".to_string(),
        readback_unavailable: true,
        ..Default::default()
    };
    assert_eq!(
        result.encode(),
        r#"{"outcome":"confirmed","host":null,"readback_unavailable":true}"#
    );
    let result = wire::EnrollmentResult {
        outcome: "confirmed".to_string(),
        credential_checked: true,
        enrollment: wire::EnrollmentView {
            revision: "0".to_string(),
            ..Default::default()
        },
        ..Default::default()
    };
    assert_eq!(
        result.encode(),
        r#"{"outcome":"confirmed","saved":false,"credential_checked":true,"enrollment":{"revision":"0","binding":"","tailnet":"","tags":[],"configured":false,"admission":false,"default":false,"preauthorized":false,"credential_checked":false,"enrollment_verified":false,"runtime_supported":false}}"#
    );
    // HTML escaping matches encoding/json.
    let mut host = host_fixture();
    host.tailnet = "a<b>&\"c\"".to_string();
    let encoded = wire::SettingsView {
        host: Some(host),
        host_unavailable: false,
        enrollment: enrollment_view_fixture(),
    }
    .encode();
    assert!(
        encoded.contains(r#""tailnet":"a\u003cb\u003e\u0026\"c\"""#),
        "{encoded}"
    );
}

// ---------- Policy: revisions, reads, checks ----------

#[test]
fn policy_revision_format() {
    for _ in 0..8 {
        let r = policy::new_revision().unwrap();
        assert!(wire::is_hex32(&r), "{r}");
    }
    assert_ne!(
        policy::new_revision().unwrap(),
        policy::new_revision().unwrap()
    );
}

#[test]
fn policy_reads_and_checks_are_non_mutating() {
    let (p, parent) = policy_store("policy-read", false);
    let v = p.enrollment(soon(5000)).unwrap();
    assert!(v.validate().is_ok() && !v.configured && v.revision == "0");
    let mut input = enrollment_fixture();
    input.action = "check".to_string();
    let r = p.update(soon(5000), &input, &accept_check).unwrap();
    assert!(!r.saved && r.credential_checked && !r.enrollment.configured);
    assert!(r.validate().is_ok());
    assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 0);
}

#[test]
fn policy_unsupported_default_has_no_effects() {
    let (p, parent) = policy_store("policy-unsupported", false);
    let r = wire::EnrollmentRequest {
        action: "default".to_string(),
        revision: "0".to_string(),
        default: Some(true),
        ..Default::default()
    };
    let e = p
        .update(soon(5000), &r, &|_, _| {
            panic!("provider called");
        })
        .expect_err("accepted");
    assert!(e.contains("unsupported"), "{e}");
    assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 0);
}

#[test]
fn policy_rotation_cas_and_secret_projection() {
    use std::os::unix::fs::PermissionsExt;
    let (p, parent) = policy_store("policy-rotate", false);
    let input = enrollment_fixture();
    let first = p.update(soon(5000), &input, &accept_check).unwrap();
    assert!(first.saved && first.validate().is_ok());
    let e = p
        .update(soon(5000), &input, &accept_check)
        .expect_err("stale save");
    assert!(e.contains("conflict"), "{e}");
    let mut rotate = input.clone();
    rotate.action = "rotate".to_string();
    rotate.revision.clone_from(&first.enrollment.revision);
    rotate.client_secret = "tskey-client-another-synthetic-secret".to_string();
    let mut bad = rotate.clone();
    bad.tailnet = "other.example.test".to_string();
    let e = p
        .update(soon(5000), &bad, &accept_check)
        .expect_err("cross-network");
    assert!(e.contains("conflict"), "{e}");
    let second = p.update(soon(5000), &rotate, &accept_check).unwrap();
    assert_eq!(second.enrollment.binding, first.enrollment.binding);
    assert_ne!(second.enrollment.revision, first.enrollment.revision);
    let dir = parent.join("soda-tailnet");
    let entries: Vec<_> = std::fs::read_dir(&dir).unwrap().collect();
    assert_eq!(entries.len(), 1);
    for entry in &entries {
        let entry = entry.as_ref().unwrap();
        assert_eq!(
            entry.metadata().unwrap().permissions().mode() & 0o777,
            0o600
        );
        assert!(!entry
            .file_name()
            .to_string_lossy()
            .starts_with("credential-"));
    }
    let active = std::fs::read(dir.join("policy.json")).unwrap();
    let text = String::from_utf8(active).unwrap();
    assert!(text.contains("tskey-client-another-synthetic-secret"));
    assert!(!text.contains("tskey-client-soda-synthetic-secret"));
    let encoded = second.encode();
    assert!(!encoded.contains("synthetic") && !encoded.contains("credential-"));
    let disabled = p
        .update(
            soon(5000),
            &wire::EnrollmentRequest {
                action: "disable".to_string(),
                revision: second.enrollment.revision.clone(),
                ..Default::default()
            },
            &accept_check,
        )
        .unwrap();
    assert!(!disabled.enrollment.admission && !disabled.enrollment.default);
    let v = p.enrollment(soon(5000)).unwrap();
    assert_eq!(v.revision, disabled.enrollment.revision);
}

#[test]
fn policy_does_not_convert_or_delete_retained_credentials() {
    let (p, parent) = policy_store("policy-retained", false);
    let first = p
        .update(soon(5000), &enrollment_fixture(), &accept_check)
        .unwrap();
    let dir = parent.join("soda-tailnet");
    let retained = dir.join(format!("credential-{}.json", "a".repeat(32)));
    let secret =
        br#"{"client_id":"synthetic-client","secret":"tskey-client-retained-synthetic-secret"}"#;
    std::fs::write(&retained, secret).unwrap();
    let mut rotate = enrollment_fixture();
    rotate.action = "rotate".to_string();
    rotate.revision = first.enrollment.revision.clone();
    p.update(soon(5000), &rotate, &accept_check).unwrap();
    assert_eq!(std::fs::read(&retained).unwrap(), secret);
    // Downgrade the stored policy to v1 with a credential reference: reads
    // fail, updates refuse to overwrite, nothing is altered.
    let path = dir.join("policy.json");
    let body = std::fs::read(&path).unwrap();
    let mut text = String::from_utf8(body).unwrap();
    text = text.replacen("\"version\":2", "\"version\":1", 1);
    let cred_start = text.find("\"credential\":").unwrap();
    let cred_end = text.rfind('}').unwrap();
    text.replace_range(
        cred_start..cred_end,
        &format!("\"credential\":\"{}\"", "a".repeat(32)),
    );
    std::fs::write(&path, &text).unwrap();
    assert!(p.enrollment(soon(5000)).is_err());
    assert!(p.update(soon(5000), &rotate, &accept_check).is_err());
    assert_eq!(std::fs::read(&path).unwrap(), text.as_bytes());
    assert_eq!(std::fs::read(&retained).unwrap(), secret);
}

#[test]
fn policy_concurrency_and_cancelled_waiter() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let (p, _) = policy_store("policy-conc", false);
    let seed = p
        .update(soon(5000), &enrollment_fixture(), &accept_check)
        .unwrap();
    let mut input = enrollment_fixture();
    input.action = "rotate".to_string();
    input.revision = seed.enrollment.revision;
    let calls = Arc::new(AtomicU32::new(0));
    let run = |p: &policy::PolicyStore, input: &wire::EnrollmentRequest, calls: &Arc<AtomicU32>| {
        let calls = calls.clone();
        p.update(soon(10000), input, &|_, _| {
            calls.fetch_add(1, Ordering::SeqCst);
            Ok(())
        })
    };
    let (a, b) = std::thread::scope(|s| {
        let ha = s.spawn(|| run(&p, &input, &calls).map(|_| ()));
        let hb = s.spawn(|| run(&p, &input, &calls).map(|_| ()));
        (ha.join().unwrap(), hb.join().unwrap())
    });
    let (ok, conflict) = [&a, &b].iter().fold((0, 0), |(ok, cf), r| match r {
        Ok(()) => (ok + 1, cf),
        Err(e) if e.contains("conflict") => (ok, cf + 1),
        Err(e) => panic!("unexpected {e}"),
    });
    assert_eq!((ok, conflict), (1, 1));
    assert_eq!(calls.load(Ordering::SeqCst), 1);
    // A waiter with an expired deadline acquires nothing.
    let _held = p.lock(soon(5000), false).unwrap().unwrap();
    assert!(p.enrollment(Instant::now()).is_err());
}

#[test]
fn policy_refuses_unsafe_and_ambiguous_state() {
    use std::os::unix::fs::PermissionsExt;
    for kind in [
        "root-link",
        "policy-link",
        "policy-hardlink",
        "permissions",
        "corrupt",
        "missing-credential",
    ] {
        let (p, parent) = policy_store(&format!("policy-unsafe-{kind}"), false);
        if kind == "root-link" {
            std::os::unix::fs::symlink(scratch("policy-link-target"), parent.join("soda-tailnet"))
                .unwrap();
        } else {
            p.update(soon(5000), &enrollment_fixture(), &accept_check)
                .unwrap();
            let dir = parent.join("soda-tailnet");
            let path = dir.join("policy.json");
            match kind {
                "policy-link" => {
                    std::fs::rename(&path, dir.join("policy.json.kept")).unwrap();
                    std::os::unix::fs::symlink(dir.join("policy.json.kept"), &path).unwrap();
                }
                "policy-hardlink" => {
                    std::fs::hard_link(&path, dir.join("policy.json.link")).unwrap();
                }
                "permissions" => {
                    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644))
                        .unwrap();
                }
                "corrupt" => {
                    std::fs::write(&path, b"{}").unwrap();
                }
                "missing-credential" => {
                    let body = std::fs::read(&path).unwrap();
                    let mut text = String::from_utf8(body).unwrap();
                    let start = text.find("\"credential\":").unwrap();
                    let end = text.rfind("}}").unwrap();
                    text.replace_range(
                        start..end,
                        "\"credential\":{\"client_id\":\"\",\"secret\":\"\"}",
                    );
                    std::fs::write(&path, &text).unwrap();
                }
                _ => unreachable!(),
            }
        }
        assert!(p.enrollment(soon(5000)).is_err(), "accepted {kind}");
    }
}

#[test]
fn policy_publication_failure_neither_rolls_back_nor_replays() {
    let (mut p, _) = policy_store("policy-pubfail", false);
    let first = p
        .update(soon(5000), &enrollment_fixture(), &accept_check)
        .unwrap();
    p.sync_hook = Some(Box::new(|| Err("synthetic fsync failure".to_string())));
    let input = wire::EnrollmentRequest {
        action: "disable".to_string(),
        revision: first.enrollment.revision.clone(),
        ..Default::default()
    };
    let e = p
        .update(soon(5000), &input, &accept_check)
        .expect_err("accepted");
    assert!(
        !e.contains("conflict") && !e.contains("invalid request"),
        "{e}"
    );
    p.sync_hook = None;
    let after = p.enrollment(soon(5000)).unwrap();
    assert!(!after.admission && after.revision != first.enrollment.revision);
    let e = p
        .update(soon(5000), &input, &accept_check)
        .expect_err("replayed");
    assert!(e.contains("conflict"), "{e}");
}

// ---------- Policy: projects ----------

use soda_host::tailnet_domain::{ProjectRequest, RunTarget};

const PID: &str = "p0123456789abcdef01234567";

fn inspect_req() -> ProjectRequest {
    ProjectRequest {
        project: PID.to_string(),
        action: "inspect".to_string(),
        ..Default::default()
    }
}

#[test]
fn policy_project_disabled_until_runtime_exists() {
    let (p, parent) = policy_store("policy-projoff", false);
    let cid = "a".repeat(64);
    let v = p.project(soon(5000), &inspect_req(), &cid).unwrap();
    assert!(!v.enabled && wire::validate_project_view(&v).is_ok());
    assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 0);
    let enable = ProjectRequest {
        project: PID.to_string(),
        action: "enable".to_string(),
        revision: "0".to_string(),
        binding: "b".repeat(32),
        confirm_id: PID.to_string(),
    };
    let e = p.project(soon(5000), &enable, &cid).expect_err("enabled");
    assert!(e.contains("unsupported"), "{e}");
    let disable = ProjectRequest {
        action: "disable".to_string(),
        binding: String::new(),
        ..enable.clone()
    };
    let v = p.project(soon(5000), &disable, &cid).unwrap();
    assert!(!v.enabled && v.outcome == "disconnect-unconfirmed");
    assert!(wire::validate_project_view(&v).is_ok());
    let e = p
        .project(soon(5000), &inspect_req(), &"c".repeat(64))
        .expect_err("adopted");
    assert!(e.contains("conflict"), "{e}");
    let e = p
        .project(soon(5000), &disable, &cid)
        .expect_err("dup disable");
    assert!(e.contains("conflict"), "{e}");
    // Malformed requests never touch state.
    for bad in [
        ProjectRequest {
            project: "bogus".to_string(),
            ..inspect_req()
        },
        ProjectRequest {
            revision: "0".to_string(),
            ..inspect_req()
        },
        ProjectRequest {
            action: "enable".to_string(),
            ..inspect_req()
        },
        ProjectRequest {
            action: "disable".to_string(),
            confirm_id: "pffffffffffffffffffffffff".to_string(),
            revision: "0".to_string(),
            ..inspect_req()
        },
    ] {
        let e = p.project(soon(5000), &bad, &cid).expect_err("accepted");
        assert!(e.contains("invalid request"), "{e}");
    }
    let e = p
        .project(soon(5000), &inspect_req(), "short")
        .expect_err("bad cid");
    assert!(e.contains("invalid request"), "{e}");
}

fn runtime_seed(name: &str) -> (policy::PolicyStore, PathBuf, wire::EnrollmentView) {
    let (p, parent) = policy_store(name, true);
    let saved = p
        .update(soon(5000), &enrollment_fixture(), &accept_check)
        .unwrap();
    (p, parent, saved.enrollment)
}

#[test]
fn policy_project_runtime_selection_and_bindings() {
    let (p, _, enrollment) = runtime_seed("policy-runtime");
    let yes = true;
    let result = p
        .update(
            soon(5000),
            &wire::EnrollmentRequest {
                action: "default".to_string(),
                revision: enrollment.revision.clone(),
                default: Some(yes),
                ..Default::default()
            },
            &accept_check,
        )
        .unwrap();
    assert!(result.validate().is_ok() && result.enrollment.default);
    for index in 0..2 {
        let project = format!("p{}", char::from(b'a' + index).to_string().repeat(24));
        let cid = char::from(b'c' + index).to_string().repeat(64);
        p.project(
            soon(5000),
            &ProjectRequest {
                project: project.clone(),
                action: "enable".to_string(),
                revision: "0".to_string(),
                binding: enrollment.binding.clone(),
                confirm_id: project.clone(),
            },
            &cid,
        )
        .unwrap();
        let view = p
            .project(
                soon(5000),
                &ProjectRequest {
                    project: project.clone(),
                    action: "inspect".to_string(),
                    ..Default::default()
                },
                &cid,
            )
            .unwrap();
        assert!(wire::validate_project_view(&view).is_ok());
        assert!(view.enabled && view.binding == enrollment.binding);
        assert_eq!(view.available_binding, enrollment.binding);
        let binding = p
            .run_binding(
                soon(5000),
                &RunTarget {
                    project: project.clone(),
                    container: cid.clone(),
                    run: "e".repeat(64),
                },
            )
            .unwrap();
        assert!(binding.enabled && binding.admission);
        assert_eq!(binding.tailnet, "soda.example.test");
        if index == 0 {
            let off = p
                .project(
                    soon(5000),
                    &ProjectRequest {
                        project: project.clone(),
                        action: "disable".to_string(),
                        revision: view.revision.clone(),
                        confirm_id: project.clone(),
                        ..Default::default()
                    },
                    &cid,
                )
                .unwrap();
            assert!(wire::validate_project_view(&off).is_ok());
            assert!(!off.enabled && off.saved);
        }
    }
    // A configured default does not enroll old projects or legacy omissions.
    let old = p
        .project(
            soon(5000),
            &ProjectRequest {
                project: format!("p{}", "f".repeat(24)),
                action: "inspect".to_string(),
                ..Default::default()
            },
            &"a".repeat(64),
        )
        .unwrap();
    assert!(!old.enabled && old.revision == "0");
}

#[test]
fn policy_project_closed_admission_has_no_reservation() {
    let (p, parent, enrollment) = runtime_seed("policy-closed");
    let project = format!("p{}", "a".repeat(24));
    let result = p
        .update(
            soon(5000),
            &wire::EnrollmentRequest {
                action: "disable".to_string(),
                revision: enrollment.revision.clone(),
                ..Default::default()
            },
            &accept_check,
        )
        .unwrap();
    let e = p
        .update(
            soon(5000),
            &wire::EnrollmentRequest {
                action: "default".to_string(),
                revision: result.enrollment.revision.clone(),
                default: Some(true),
                ..Default::default()
            },
            &accept_check,
        )
        .expect_err("default on closed");
    assert!(e.contains("conflict"), "{e}");
    let e = p
        .project(
            soon(5000),
            &ProjectRequest {
                project: project.clone(),
                action: "enable".to_string(),
                revision: "0".to_string(),
                binding: enrollment.binding.clone(),
                confirm_id: project.clone(),
            },
            &"b".repeat(64),
        )
        .expect_err("enabled on closed");
    assert!(e.contains("conflict"), "{e}");
    let entries: Vec<_> = std::fs::read_dir(parent.join("soda-tailnet"))
        .unwrap()
        .collect();
    assert_eq!(entries.len(), 1);
}

#[test]
fn policy_project_enable_cas_and_no_implicit_retarget() {
    let (p, _, enrollment) = runtime_seed("policy-cas");
    let project = format!("p{}", "a".repeat(24));
    let cid = "b".repeat(64);
    let enable = ProjectRequest {
        project: project.clone(),
        action: "enable".to_string(),
        revision: "0".to_string(),
        binding: enrollment.binding.clone(),
        confirm_id: project.clone(),
    };
    let first = p.project(soon(5000), &enable, &cid).unwrap();
    assert!(wire::validate_project_view(&first).is_ok() && first.enabled);
    let e = p.project(soon(5000), &enable, &cid).expect_err("stale");
    assert!(e.contains("conflict"), "{e}");
    let mut retarget = enable.clone();
    retarget.revision.clone_from(&first.revision);
    retarget.binding = "f".repeat(32);
    let e = p
        .project(soon(5000), &retarget, &cid)
        .expect_err("retarget");
    assert!(e.contains("conflict"), "{e}");
    let mut retry = retarget.clone();
    retry.action = "retry".to_string();
    retry.binding.clone_from(&enrollment.binding);
    let retried = p.project(soon(5000), &retry, &cid).unwrap();
    assert!(wire::validate_project_view(&retried).is_ok());
    assert_ne!(retried.revision, first.revision);
    let inspect = ProjectRequest {
        project: project.clone(),
        action: "inspect".to_string(),
        ..Default::default()
    };
    let e = p
        .project(soon(5000), &inspect, &"c".repeat(64))
        .expect_err("adopted");
    assert!(e.contains("conflict"), "{e}");
    let cancelled = ProjectRequest {
        project: project.clone(),
        action: "disable".to_string(),
        revision: retried.revision.clone(),
        confirm_id: project.clone(),
        ..Default::default()
    };
    assert!(p.project(Instant::now(), &cancelled, &cid).is_err());
}

#[test]
fn policy_project_missing_is_off_while_malformed_fails_safely() {
    let (p, parent, enrollment) = runtime_seed("policy-malformed");
    let project = format!("p{}", "a".repeat(24));
    let cid = "b".repeat(64);
    let target = RunTarget {
        project: project.clone(),
        container: cid.clone(),
        run: "e".repeat(64),
    };
    let view = p
        .project(
            soon(5000),
            &ProjectRequest {
                project: project.clone(),
                action: "inspect".to_string(),
                ..Default::default()
            },
            &cid,
        )
        .unwrap();
    assert!(!view.enabled && wire::validate_project_view(&view).is_ok());
    let binding = p.run_binding(soon(5000), &target).unwrap();
    assert!(!binding.enabled);
    let enable = ProjectRequest {
        project: project.clone(),
        action: "enable".to_string(),
        revision: "0".to_string(),
        binding: enrollment.binding.clone(),
        confirm_id: project.clone(),
    };
    p.project(soon(5000), &enable, &cid).unwrap();
    let e = p
        .run_binding(
            soon(5000),
            &RunTarget {
                container: "c".repeat(64),
                ..target.clone()
            },
        )
        .expect_err("adopted");
    assert!(e.contains("conflict"), "{e}");
    let path = parent
        .join("soda-tailnet")
        .join(format!("project-{project}.json"));
    std::fs::write(&path, b"{}").unwrap();
    let e = p
        .project(
            soon(5000),
            &ProjectRequest {
                project: project.clone(),
                action: "inspect".to_string(),
                ..Default::default()
            },
            &cid,
        )
        .expect_err("malformed accepted");
    assert!(e.contains("unavailable"), "{e}");
    assert!(p.run_binding(soon(5000), &target).is_err());
    assert_eq!(std::fs::read(&path).unwrap(), b"{}");
    // Invalid run targets never touch the lock.
    let e = p
        .run_binding(
            soon(5000),
            &RunTarget {
                run: "../other".to_string(),
                ..target.clone()
            },
        )
        .expect_err("bad run");
    assert!(e.contains("invalid request"), "{e}");
}

// ---------- Native observation ----------

use soda_host::project::Executor;

const NATIVE_STATUS: &str = r#"{"Version":"1.102.4","CurrentTailnet":{"Name":"soda.example.test","MagicDNSEnabled":true},"BackendState":"Running","HaveNodeKey":true,"Self":{"ID":"self","DNSName":"host.example.ts.net.","TailscaleIPs":["100.64.0.1"]},"Peer":{"peer":{"ID":"peer","DNSName":"exit.example.ts.net.","TailscaleIPs":["100.64.0.2"],"Online":true,"ExitNodeOption":true}},"AuthURL":"https://login.tailscale.com/a/synthetic","Health":["sensitive native diagnostic"],"PrivateKey":"must-not-project"}"#;
const NATIVE_PREFS: &str = r#"{"WantRunning":true,"ExitNodeID":"","ExitNodeIP":"","ExitNodeAllowLANAccess":false,"AdvertiseRoutes":["10.8.0.0/16"],"Persist":{"PrivateNodeKey":"must-not-project"}}"#;

// Byte-exact Go goldens (transient `go test` dump of `observe`/`Settings`
// over the fixtures above; helper deleted after capture).
const GO_REVISION: &str = "c2325fcf010801018acfa617389a67457360b7763a3b7343869ff73f5c4ad304";
const GO_FRESH_REVISION: &str = "2a70710052e2a0eccf94ce7d4d5051fb35d5ff241253fcd702d7afe92cb3d443";
const GO_HOSTVIEW: &str = r#"{"tailnet":"soda.example.test","magic_dns_enabled":true,"revision":"c2325fcf010801018acfa617389a67457360b7763a3b7343869ff73f5c4ad304","state":"Running","have_node_key":true,"expired":false,"dns_name":"host.example.ts.net","addresses":["100.64.0.1"],"peers":[{"id":"peer","dns_name":"exit.example.ts.net","addresses":["100.64.0.2"],"online":true,"exit_node":true,"expired":false}],"health_issues":1,"preferences":{"want_running":true,"exit_node_id":"","exit_node_ip":"","allow_lan":false,"advertise_exit_node":false}}"#;

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

#[test]
fn native_observe_fixture_matches_go_goldens() {
    let t = fixture_transport(NATIVE_STATUS.to_string(), NATIVE_PREFS.to_string());
    for _ in 0..2 {
        let (view, auth) = native::observe(&t, soon(5000)).unwrap();
        assert!(view.validate().is_ok());
        assert_eq!(view.revision, GO_REVISION);
        assert_eq!(auth, "https://login.tailscale.com/a/synthetic");
        let mut encoded = String::new();
        view.encode_into(&mut encoded);
        assert_eq!(encoded, GO_HOSTVIEW);
        for secret in [
            "sensitive native diagnostic",
            "PrivateNodeKey",
            "must-not-project",
            "login.tailscale.com",
        ] {
            assert!(!encoded.contains(secret), "projected {secret}");
        }
        assert_eq!(view.health_issues, 1);
        assert_eq!(view.peers.len(), 1);
        assert!(!view.preferences.advertise_exit_node);
    }
}

#[test]
fn native_fresh_daemon_omits_false_node_key() {
    for field in ["", r#","HaveNodeKey":false"#] {
        let status = format!(
            r#"{{"Version":"1.102.4-t3caf7d9e7-g084ee3b64","BackendState":"NeedsLogin","Self":{{"ID":"","DNSName":"","TailscaleIPs":null}},"Peer":null{field}}}"#
        );
        let prefs = r#"{"WantRunning":false,"ExitNodeID":"","ExitNodeIP":"","ExitNodeAllowLANAccess":false,"AdvertiseRoutes":null}"#;
        let t = fixture_transport(status, prefs.to_string());
        let (view, _) = native::observe(&t, soon(5000)).unwrap();
        assert!(view.validate().is_ok());
        assert_eq!(view.revision, GO_FRESH_REVISION);
        assert_eq!(view.state, "NeedsLogin");
        assert!(!view.have_node_key && !view.preferences.want_running);
        assert!(view.addresses.is_empty());
    }
}

#[test]
fn native_node_key_optional_but_strict() {
    for field in [
        "",
        r#""HaveNodeKey":null,"#,
        r#""HaveNodeKey":"false","#,
        r#""HaveNodeKey":0,"#,
        r#""haveNodeKey":true,"#,
    ] {
        let status = NATIVE_STATUS.replacen(r#""HaveNodeKey":true,"#, field, 1);
        let t = fixture_transport(status, NATIVE_PREFS.to_string());
        assert!(
            native::observe(&t, soon(5000)).is_err(),
            "accepted field {field:?}"
        );
    }
}

#[test]
fn native_unavailable_kinds() {
    for kind in [
        "state",
        "prefs",
        "oversize",
        "null",
        "duplicate",
        "status-code",
    ] {
        let mut status = NATIVE_STATUS.to_string();
        let mut prefs = NATIVE_PREFS.to_string();
        match kind {
            "state" => status = status.replacen("Running", "UnknownState", 1),
            "prefs" => prefs = "{}".to_string(),
            "oversize" => status = format!("{status}{}", " ".repeat(65536)),
            "null" => status = "null".to_string(),
            "duplicate" => {
                status = status.replacen(
                    r#""HaveNodeKey":true"#,
                    r#""HaveNodeKey":true,"HaveNodeKey":false"#,
                    1,
                )
            }
            _ => {}
        }
        let t: Box<native::Transport> = if kind == "status-code" {
            Box::new(|_, _, _, _| Ok((500, Vec::new())))
        } else {
            fixture_transport(status, prefs)
        };
        assert!(native::observe(&t, soon(5000)).is_err(), "accepted {kind}");
    }
    assert!(native::finish_local(200, &[0u8; 65537]).is_err());
    assert!(native::finish_local(204, b"").is_ok());
    assert!(native::finish_local(301, b"").is_err());
}

#[test]
fn native_revision_tracks_identity_without_release_veto() {
    let t = fixture_transport(NATIVE_STATUS.to_string(), NATIVE_PREFS.to_string());
    let (before, _) = native::observe(&t, soon(5000)).unwrap();
    let status = NATIVE_STATUS
        .replacen(r#""ID":"self""#, r#""ID":"replacement""#, 1)
        .replacen("1.102.4", "1.1.0", 1);
    let t2 = fixture_transport(status, NATIVE_PREFS.to_string());
    let (after, _) = native::observe(&t2, soon(5000)).unwrap();
    assert_ne!(after.revision, before.revision);
}

#[test]
fn native_sha256_matches_reference() {
    assert_eq!(
        soda_host::sha256::hex_lower(&soda_host::sha256::digest(b"abc")),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
}

#[test]
fn native_signin_reauth_preserves_prefs() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    let patches = Arc::new(AtomicU32::new(0));
    let logins = Arc::new(AtomicU32::new(0));
    let status = NATIVE_STATUS.replacen(r#""Running""#, r#""NeedsLogin""#, 1);
    let t: Box<native::Transport> = Box::new({
        let (patches, logins) = (patches.clone(), logins.clone());
        move |method: &str, path: &str, body: Option<&[u8]>, _d: Instant| match (method, path) {
            ("GET", "status") => Ok((200, status.as_bytes().to_vec())),
            ("GET", "prefs") => Ok((200, NATIVE_PREFS.as_bytes().to_vec())),
            ("PATCH", "prefs") => {
                patches.fetch_add(1, Ordering::SeqCst);
                assert_eq!(
                    body.unwrap_or_default(),
                    br#"{"WantRunning":true,"WantRunningSet":true}"#
                );
                Ok((200, b"{}".to_vec()))
            }
            ("POST", "login-interactive") => {
                logins.fetch_add(1, Ordering::SeqCst);
                Ok((204, Vec::new()))
            }
            _ => panic!("unexpected {method} {path}"),
        }
    });
    let (before, _) = native::observe(&t, soon(5000)).unwrap();
    let exec = boom_exec();
    native::execute_signin(
        &t,
        &exec,
        native::DEFAULT_CLI,
        std::path::Path::new(native::HOST_SOCKET),
        &before,
        soon(5000),
    )
    .unwrap();
    assert_eq!(patches.load(Ordering::SeqCst), 1);
    assert_eq!(logins.load(Ordering::SeqCst), 1);
    // Readback is pending while the auth URL is live.
    let req = wire::HostRequest {
        action: "signin".to_string(),
        revision: before.revision.clone(),
        ..Default::default()
    };
    let result = native::readback_host_action(&t, &req, "", None, soon(5000));
    assert!(result.validate().is_ok());
    assert_eq!(result.outcome, "pending");
    assert!(!result.auth_url.is_empty());
}

#[test]
fn native_exit_node_and_logout_confirm() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    for action in ["exit-node", "logout"] {
        let mutations = Arc::new(AtomicU32::new(0));
        let changed = Arc::new(std::sync::atomic::AtomicBool::new(false));
        let t: Box<native::Transport> = Box::new({
            let (mutations, changed) = (mutations.clone(), changed.clone());
            move |method: &str, path: &str, _body: Option<&[u8]>, _d: Instant| match (method, path)
            {
                ("GET", "status") => {
                    let mut status = NATIVE_STATUS.to_string();
                    if changed.load(Ordering::SeqCst) && action == "logout" {
                        status = status
                            .replacen(r#""Running""#, r#""NeedsLogin""#, 1)
                            .replacen(r#""HaveNodeKey":true"#, r#""HaveNodeKey":false"#, 1);
                    }
                    Ok((200, status.into_bytes()))
                }
                ("GET", "prefs") => {
                    let mut prefs = NATIVE_PREFS.to_string();
                    if changed.load(Ordering::SeqCst) && action == "exit-node" {
                        prefs = prefs
                            .replacen(r#""ExitNodeID":"""#, r#""ExitNodeID":"peer""#, 1)
                            .replacen(
                                r#""ExitNodeAllowLANAccess":false"#,
                                r#""ExitNodeAllowLANAccess":true"#,
                                1,
                            );
                    }
                    Ok((200, prefs.into_bytes()))
                }
                ("POST", "logout") => {
                    assert_eq!(action, "logout");
                    mutations.fetch_add(1, Ordering::SeqCst);
                    changed.store(true, Ordering::SeqCst);
                    Ok((204, Vec::new()))
                }
                _ => panic!("unexpected {method} {path}"),
            }
        });
        let exec = FakeExec {
            f: Box::new({
                let (mutations, changed) = (mutations.clone(), changed.clone());
                move |_, cmd: &str, args: &[&str], _| {
                    assert_eq!(cmd, native::DEFAULT_CLI);
                    let sock = format!("--socket={}", native::HOST_SOCKET);
                    assert_eq!(
                        args,
                        &[
                            sock.as_str(),
                            "set",
                            "--exit-node=100.64.0.2",
                            "--exit-node-allow-lan-access=true"
                        ][..]
                    );
                    mutations.fetch_add(1, Ordering::SeqCst);
                    changed.store(true, Ordering::SeqCst);
                    Ok(Vec::new())
                }
            }),
        };
        let (before, _) = native::observe(&t, soon(5000)).unwrap();
        let mut req = wire::HostRequest {
            action: action.to_string(),
            revision: before.revision.clone(),
            confirm: action.to_string(),
            ..Default::default()
        };
        let selected;
        let action_err;
        if action == "exit-node" {
            req.exit_node = Some("100.64.0.2".to_string());
            req.allow_lan = Some(true);
            let socket = std::path::Path::new(native::HOST_SOCKET);
            match native::execute_exit_node(
                &exec,
                native::DEFAULT_CLI,
                socket,
                &req,
                &before,
                soon(5000),
            ) {
                Ok(id) => {
                    selected = id;
                    action_err = None;
                }
                Err(e) => {
                    selected = String::new();
                    action_err = Some(e);
                }
            }
        } else {
            selected = String::new();
            action_err = native::execute_logout(&t, soon(5000)).err();
        }
        let result = native::readback_host_action(&t, &req, &selected, action_err, soon(5000));
        assert!(result.validate().is_ok(), "{action}: {result:?}");
        assert_eq!(result.outcome, "confirmed", "{action}");
        assert_eq!(mutations.load(Ordering::SeqCst), 1);
        assert!(result.auth_url.is_empty());
    }
}

#[test]
fn native_offline_exit_conflicts_and_retained_id_stays_unconfirmed() {
    // Offline peer: no command runs, conflict surfaces.
    let status = NATIVE_STATUS.replacen(r#""Online":true"#, r#""Online":false"#, 1);
    let t = fixture_transport(status, NATIVE_PREFS.to_string());
    let (before, _) = native::observe(&t, soon(5000)).unwrap();
    let req = wire::HostRequest {
        action: "exit-node".to_string(),
        revision: before.revision.clone(),
        confirm: "exit-node".to_string(),
        exit_node: Some("100.64.0.2".to_string()),
        allow_lan: Some(false),
        ..Default::default()
    };
    let exec = boom_exec();
    let socket = std::path::Path::new(native::HOST_SOCKET);
    let e = native::execute_exit_node(
        &exec,
        native::DEFAULT_CLI,
        socket,
        &req,
        &before,
        soon(5000),
    )
    .expect_err("offline selected");
    assert!(e.contains("conflict"), "{e}");
    // Clear with a retained native ID: command runs, readback unconfirmed.
    let prefs = NATIVE_PREFS.replacen(r#""ExitNodeID":"""#, r#""ExitNodeID":"peer""#, 1);
    let t = fixture_transport(NATIVE_STATUS.to_string(), prefs);
    let (before, _) = native::observe(&t, soon(5000)).unwrap();
    let req = wire::HostRequest {
        action: "exit-node".to_string(),
        revision: before.revision.clone(),
        confirm: "exit-node".to_string(),
        exit_node: Some(String::new()),
        allow_lan: Some(false),
        ..Default::default()
    };
    let exec = FakeExec {
        f: Box::new(|_, cmd, args, _| {
            assert_eq!(cmd, native::DEFAULT_CLI);
            assert!(args.iter().any(|a| a.contains("--exit-node=")));
            Ok(Vec::new())
        }),
    };
    let selected = native::execute_exit_node(
        &exec,
        native::DEFAULT_CLI,
        socket,
        &req,
        &before,
        soon(5000),
    )
    .unwrap();
    let result = native::readback_host_action(&t, &req, &selected, None, soon(5000));
    assert_eq!(result.outcome, "unconfirmed");
}

#[test]
fn native_up_notifications_and_command_bounds() {
    assert!(native::decode_up_notifications(b"").is_ok());
    assert!(native::decode_up_notifications(
        b"{\n\"AuthURL\":\"https://login.tailscale.com/a/synthetic\"\n}\n{\"BackendState\":\"NeedsLogin\"}\n"
    )
    .is_ok());
    for bad in [
        &b"not json"[..],
        b"{\"Error\":\"must-not-escape-private-diagnostic\"}",
        b"{\"AuthURL\":5}",
        b"{} {",
        b"5",
    ] {
        let e = native::decode_up_notifications(bad).expect_err("accepted");
        assert!(!e.contains("must-not-escape"), "{e}");
    }
    let exec = FakeExec {
        f: Box::new(|_, _, _, _| Ok(vec![0u8; 65537])),
    };
    assert!(native::run_command(&exec, "/bin/true", &[], soon(5000)).is_err());
    let exec = FakeExec {
        f: Box::new(|_, _, _, _| Err("synthetic secret diagnostic".to_string())),
    };
    let e = native::run_command(&exec, "/bin/false", &[], soon(5000)).expect_err("accepted");
    assert!(!e.contains("synthetic"), "{e}");
}

#[test]
fn native_local_request_over_real_socket() {
    use std::io::{Read, Write};
    use std::os::unix::net::UnixListener;
    let dir = scratch("sock");
    let path = dir.join("s");
    let listener = UnixListener::bind(&path).unwrap();
    let server = std::thread::spawn(move || {
        let mut framing = 0;
        let mut boom_requests = 0;
        for _ in 0..9 {
            let (mut conn, _) = listener.accept().unwrap();
            let mut head = vec![0u8; 4096];
            let mut total = Vec::new();
            loop {
                let n = conn.read(&mut head).unwrap();
                if n == 0 {
                    break;
                }
                total.extend_from_slice(&head[..n]);
                if total.windows(4).any(|w| w == b"\r\n\r\n") {
                    break;
                }
            }
            let text = String::from_utf8_lossy(&total).into_owned();
            let is_status = text.contains("GET /localapi/v0/status ");
            let is_prefs = text.contains("GET /localapi/v0/prefs ");
            let response = if is_status || is_prefs {
                let body = if is_status {
                    NATIVE_STATUS
                } else {
                    NATIVE_PREFS
                };
                let response = match framing {
                    0 => format!(
                        "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                        body.len(),
                        body
                    ),
                    1 => format!(
                        "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n{:x}\r\n{}\r\n0\r\n\r\n",
                        body.len(),
                        body
                    ),
                    3 if is_status => format!(
                        "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                        body.len() + 1,
                        body
                    ),
                    _ => format!(
                        "HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n{}",
                        body
                    ),
                };
                if is_prefs {
                    framing += 1;
                }
                response
            } else if text.contains("GET /localapi/v0/boom ") {
                boom_requests += 1;
                if boom_requests == 1 {
                    "HTTP/1.1 500 Internal Server Error\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n0\r\n\r\n".to_string()
                } else {
                    "HTTP/1.1 500 Internal Server Error\r\nConnection: close\r\n\r\nsynthetic"
                        .to_string()
                }
            } else {
                "HTTP/1.1 200 OK\r\nConnection: close\r\n\r\nnot json".to_string()
            };
            conn.write_all(response.as_bytes()).unwrap();
        }
    });
    let t: Box<native::Transport> = Box::new({
        let path = path.clone();
        move |method: &str, p: &str, body: Option<&[u8]>, d: Instant| {
            native::local_request(&path, method, p, body, d)
        }
    });
    for _ in 0..3 {
        let (view, auth) = native::observe(&t, soon(5000)).unwrap();
        assert_eq!(view.revision, GO_REVISION);
        assert_eq!(auth, "https://login.tailscale.com/a/synthetic");
    }
    assert!(native::observe(&t, soon(5000)).is_err());
    assert!(native::local_request(&path, "GET", "boom", None, soon(5000)).is_ok());
    let (status, body) = native::local_request(&path, "GET", "boom", None, soon(5000)).unwrap();
    assert_eq!(status, 500);
    assert_eq!(body, b"synthetic");
    server.join().unwrap();
    assert!(
        native::local_request(&dir.join("missing"), "GET", "status", None, soon(5000)).is_err()
    );
    assert!(native::local_request(&path, "GET", "status", None, Instant::now()).is_err());
}

#[test]
fn native_run_status_validate() {
    let live = native::RunStatus {
        enabled: true,
        admission: true,
        tailnet: "soda.example.test".to_string(),
        tags: vec!["tag:soda-project".to_string()],
        addresses: vec!["100.64.0.2".to_string()],
        dns_name: "project.soda.ts.net".to_string(),
    };
    assert!(live.validate().is_ok());
    let off = native::RunStatus {
        tailnet: "soda.example.test".to_string(),
        tags: vec!["tag:soda-project".to_string()],
        ..Default::default()
    };
    assert!(off.validate().is_ok());
    let mut bad = live.clone();
    bad.enabled = false;
    assert!(bad.validate().is_err(), "admission without enablement");
    let mut bad = live.clone();
    bad.tailnet = "../other".to_string();
    assert!(bad.validate().is_err());
    let mut bad = live.clone();
    bad.tags = vec!["tag:z".to_string(), "tag:a".to_string()];
    assert!(bad.validate().is_err());
    let mut bad = live.clone();
    bad.addresses = vec!["127.0.0.1".to_string()];
    assert!(bad.validate().is_err());
    let mut bad = live.clone();
    bad.dns_name = "bad.local".to_string();
    assert!(bad.validate().is_err());
}

#[test]
fn native_cli_client_vectors() {
    let status_exec = |output: &'static str| FakeExec {
        f: Box::new(move |stdin, cmd, args, _| {
            assert!(stdin.is_empty());
            assert_eq!(args, ["status", "--json"]);
            assert_eq!(cmd, "/bin/tailscale");
            Ok(output.as_bytes().to_vec())
        }),
    };
    let status = native::cli_status(
        &status_exec(r#"{"BackendState":"Running","Self":{"DNSName":"Atlas.Example.ts.net."}}"#),
        "/bin/tailscale",
        soon(5000),
    )
    .unwrap();
    assert_eq!(status.identity, "atlas.example.ts.net");
    let endpoint = native::cli_endpoint(
        &status_exec(r#"{"BackendState":"Running","Self":{"DNSName":"Atlas.Example.ts.net.","TailscaleIPs":["fd7a:115c:a1e0::1","100.88.77.66"]},"CurrentTailnet":{"MagicDNSEnabled":true}}"#),
        "/bin/tailscale",
        soon(5000),
    )
    .unwrap();
    assert_eq!(endpoint.identity, "atlas.example.ts.net");
    assert_eq!(endpoint.ipv4, "100.88.77.66");
    for dns in ["", "atlas.example.ts.net."] {
        let endpoint = native::cli_endpoint(
            &status_exec(Box::leak(
                format!(r#"{{"BackendState":"Running","Self":{{"DNSName":"{dns}","TailscaleIPs":["100.88.77.66"]}}}}"#)
                    .into_boxed_str(),
            )),
            "/bin/tailscale",
            soon(5000),
        )
        .unwrap();
        assert_eq!(endpoint.identity, "100.88.77.66");
    }
    for (output, want) in [
        (r#"{"BackendState":"NeedsLogin","Self":{}}"#, "not enrolled"),
        (
            r#"{"BackendState":"Stopped","Self":{"TailscaleIPs":["100.88.77.66"]}}"#,
            "not enrolled",
        ),
        (r#"{"BackendState":"Running","Self":{}}"#, "IPv4"),
        (
            r#"{"BackendState":"Running","Self":{"DNSName":"atlas.example.ts.net","TailscaleIPs":["fd7a:115c:a1e0::1"]}}"#,
            "IPv4",
        ),
        (
            r#"{"BackendState":"Running","Self":{"Expired":true,"DNSName":"atlas.example.ts.net.","TailscaleIPs":["100.88.77.66"]}}"#,
            "not enrolled",
        ),
    ] {
        let e = native::cli_endpoint(&status_exec(output), "/bin/tailscale", soon(5000))
            .expect_err("advertised");
        assert!(e.contains(want), "{e}");
    }
    for bad in ["", "{}", r#"{"BackendState":"Running""#, "{} {}"] {
        assert!(native::cli_status(&status_exec(bad), "/bin/tailscale", soon(5000)).is_err());
    }
    let status = native::cli_status(
        &status_exec(r#"{"BackendState":"NeedsLogin","AuthURL":"https://fixture.invalid/auth"}"#),
        "/bin/tailscale",
        soon(5000),
    )
    .unwrap();
    assert!(status.auth_pending);
    assert!(native::cli_status(
        &status_exec(r#"{"BackendState":"Running","Self":{"DNSName":"atlas.local"}}"#),
        "/bin/tailscale",
        soon(5000)
    )
    .is_err());
    // CLI failures carry the native diagnostic, like Go.
    let exec = FakeExec {
        f: Box::new(|_, _, _, _| Err("exit 7: daemon unavailable".to_string())),
    };
    let e = native::cli_status(&exec, "/bin/tailscale", soon(5000)).expect_err("accepted");
    assert!(
        e.contains("unavailable") && e.contains("daemon unavailable"),
        "{e}"
    );
}

// ---------- Provider ----------

#[test]
fn provider_request_bodies_match_go_recipes() {
    assert_eq!(
        provider::token_form_body(&["tag:soda-project".to_string()]),
        "grant_type=client_credentials&scope=auth_keys&tags=tag%3Asoda-project"
    );
    assert_eq!(
        provider::token_form_body(&["tag:a".to_string(), "tag:b".to_string()]),
        "grant_type=client_credentials&scope=auth_keys&tags=tag%3Aa+tag%3Ab"
    );
    assert_eq!(
        provider::key_create_body(&["tag:soda-project".to_string()], false),
        r#"{"capabilities":{"devices":{"create":{"reusable":false,"ephemeral":true,"tags":["tag:soda-project"],"preauthorized":false}}},"expirySeconds":300,"description":"Soda ephemeral project run"}"#
    );
}

#[test]
fn provider_curl_recipe_keeps_secrets_out_of_argv() {
    use std::sync::{Arc, Mutex};
    let seen: Arc<Mutex<(Vec<String>, Vec<u8>)>> = Arc::new(Mutex::new((Vec::new(), Vec::new())));
    let exec = FakeExec {
        f: Box::new({
            let seen = seen.clone();
            move |stdin: &[u8], cmd: &str, args: &[&str], _| {
                let mut seen = seen.lock().unwrap();
                seen.0 = args.iter().map(|s| s.to_string()).collect();
                seen.1 = stdin.to_vec();
                assert_eq!(cmd, "/usr/bin/curl");
                Ok(b"{\"access_token\":\"synthetic-bearer\",\"token_type\":\"Bearer\",\"expires_in\":3600}\n200".to_vec())
            }
        }),
    };
    let (status, body) = provider::fetch_token(
        &exec,
        "/usr/bin/curl",
        "synthetic-client",
        "tskey-client-soda-synthetic-secret",
        &["tag:soda-project".to_string()],
        soon(5000),
    )
    .unwrap();
    assert_eq!(status, 200);
    assert!(body.starts_with(b"{\"access_token\""));
    let seen = seen.lock().unwrap();
    assert_eq!(
        seen.0,
        [
            "--silent",
            "--config",
            "-",
            "--write-out",
            "\\n%{http_code}"
        ]
    );
    let config = String::from_utf8(seen.1.clone()).unwrap();
    assert!(config.contains("user = \"synthetic-client:tskey-client-soda-synthetic-secret\""));
    assert!(config.contains("tags=tag%3Asoda-project"));
    // Status suffix split survives suffix-like bodies.
    let exec = FakeExec {
        f: Box::new(|_, _, _, _| Ok(b"x\n200\n201".to_vec())),
    };
    let (status, body) =
        provider::run_curl(&exec, "/usr/bin/curl", "url = \"x\"\n", soon(5000)).unwrap();
    assert_eq!((status, body), (201, b"x\n200".to_vec()));
    // Failures are neutral: no keyword, no secret, no diagnostics.
    for out in [
        Err("curl: (7) synthetic".to_string()),
        Ok(b"no-suffix".to_vec()),
        Ok(b"body\n20X".to_vec()),
        Ok(vec![0u8; 65541]),
    ] {
        let exec = FakeExec {
            f: Box::new(move |_, _, _, _| out.clone()),
        };
        let e = provider::run_curl(&exec, "/usr/bin/curl", "", soon(5000)).expect_err("accepted");
        assert!(
            !e.contains("synthetic") && !e.contains("invalid request"),
            "{e}"
        );
    }
}

#[test]
fn provider_capture_uses_the_protocol_stdout_ceiling() {
    struct BoundedExec;
    impl project::Executor for BoundedExec {
        fn run(&self, _: &[u8], _: &str, _: &[&str], _: Instant) -> Result<Vec<u8>, String> {
            panic!("provider must select bounded capture")
        }

        fn run_bounded(
            &self,
            _: &[u8],
            _: &str,
            _: &[&str],
            _: Instant,
            stdout_limit: usize,
            stderr_limit: usize,
        ) -> Result<Vec<u8>, String> {
            assert_eq!(stdout_limit, native::RESPONSE_LIMIT + 4);
            assert_eq!(stderr_limit, 1024 * 1024);
            Ok(b"body\n200".to_vec())
        }
    }

    let (status, body) = provider::run_curl(
        &BoundedExec,
        "/usr/bin/curl",
        "url = \"https://example.invalid\"\n",
        soon(5000),
    )
    .unwrap();
    assert_eq!((status, body), (200, b"body".to_vec()));
}

#[test]
fn provider_token_vectors() {
    let unavailable = || wire::err_unavailable();
    let token = provider::validate_token(
        200,
        br#"{"access_token":"synthetic-bearer","token_type":"Bearer","expires_in":3600}"#,
        false,
        &unavailable,
    )
    .unwrap();
    assert_eq!(token, "synthetic-bearer");
    // Case-insensitive type, string expiry, unknown fields, last-wins dupes.
    assert!(provider::validate_token(
        200,
        br#"{"access_token":"a","token_type":"bearer","expires_in":"3600","extra":1}"#,
        true,
        &unavailable,
    )
    .is_ok());
    assert!(provider::validate_token(
        200,
        br#"{"access_token":"","access_token":"a","token_type":"Bearer","expires_in":3600}"#,
        false,
        &unavailable,
    )
    .is_ok());
    for (status, body, strict) in [
        (403, &br#"{"error":"synthetic secret"}"#[..], false),
        (200, &br#"{}"#[..], false),
        (
            200,
            &br#"{"access_token":"","token_type":"Bearer","expires_in":3600}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":5,"token_type":"Bearer","expires_in":3600}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"other","expires_in":3600}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"Bearer"}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"Bearer","expires_in":0}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"Bearer","expires_in":-5}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"Bearer","expires_in":36.5}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a","token_type":"Bearer","expires_in":true}"#[..],
            false,
        ),
        (
            200,
            &br#"{"access_token":"a\rb","token_type":"Bearer","expires_in":3600}"#[..],
            true,
        ),
        (200, &b"null"[..], false),
        (200, &b"{}"[..], false),
    ] {
        let e = provider::validate_token(status, body, strict, &unavailable).expect_err("accepted");
        assert!(e.contains("unavailable"), "{e}");
        assert!(!e.contains("synthetic"), "{e}");
    }
    // The check path skips the control-character rule, like Go.
    assert!(provider::validate_token(
        200,
        br#"{"access_token":"a\rb","token_type":"Bearer","expires_in":3600}"#,
        false,
        &unavailable,
    )
    .is_ok());
}

#[test]
fn provider_rfc3339_matches_go_vectors() {
    // (input, expected nanos); values verified against time.Parse above.
    for (s, want) in [
        ("2024-01-02T15:04:05Z", Some(1704207845000000000i128)),
        ("2024-01-02T15:04:05.123456789Z", Some(1704207845123456789)),
        ("2024-01-02T15:04:05.123456789123Z", None),
        ("2024-01-02T5:04:05Z", None),
        ("2024-01-02T15:04:05+02:00", Some(1704200645000000000)),
        ("2024-01-02T15:04:05-05:30", Some(1704227645000000000)),
        ("2024-01-02T15:04:05+24:00", None),
        ("2024-01-02T15:04:05+02:60", None),
        ("2024-02-29T00:00:00Z", Some(1709164800000000000)),
        ("1969-12-31T23:59:59Z", Some(-1000000000)),
        ("2024-01-02T15:04:05+25:00", None),
        ("2024-01-02T15:04:05+02:61", None),
        ("2024-13-02T15:04:05Z", None),
        ("2024-01-02T24:04:05Z", None),
        ("2023-02-29T00:00:00Z", None),
        ("2024-01-02T15:04:05", None),
        ("2024-01-02T15:04:05.", None),
        ("2024-01-02T15:04:05.Z", None),
        ("2024-01-02T15:04:05z", None),
        ("2024-01-02t15:04:05Z", None),
        ("+2024-01-02T15:04:05Z", None),
        ("", None),
    ] {
        assert_eq!(provider::parse_rfc3339_nanos(s), want, "{s}");
    }
    // The Go zero time (Go's own UnixNano overflows int64 here; the true
    // value is arithmetic: -719162 days).
    assert_eq!(
        provider::parse_rfc3339_nanos("0001-01-01T00:00:00Z"),
        Some(-(719162i128 * 86_400 * 1_000_000_000))
    );
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

#[test]
fn provider_key_vectors() {
    use std::time::{Duration, SystemTime};
    assert_eq!(
        rfc3339(SystemTime::UNIX_EPOCH + Duration::from_secs(1704207845)),
        "2024-01-02T15:04:05Z"
    );
    let now = SystemTime::now();
    let tags = ["tag:soda-project".to_string()];
    let now_s = rfc3339(now);
    let exp_s = rfc3339(now + Duration::from_secs(300));
    let good = key_response(&now_s, &exp_s);
    let key = provider::validate_key(200, good.as_bytes(), &tags, false, now, now).unwrap();
    assert_eq!(key, "tskey-auth-synthetic-only");
    let mutate = |mode: &str| -> (u16, Vec<u8>) {
        let mut body = good.clone();
        let mut status = 200;
        match mode {
            "redirect" => status = 307,
            "error" => {
                status = 403;
                body = r#"{"message":"private provider text"}"#.to_string();
            }
            "oversize" => body.push_str(&" ".repeat(65536)),
            "null" => body = "null".to_string(),
            "missing reusable" => body = body.replacen(r#""reusable":false,"#, "", 1),
            "duplicate" => {
                body = body.replacen(
                    r#""reusable":false"#,
                    r#""reusable":false,"reusable":false"#,
                    1,
                )
            }
            "reusable" => body = body.replacen(r#""reusable":false"#, r#""reusable":true"#, 1),
            "persistent" => body = body.replacen(r#""ephemeral":true"#, r#""ephemeral":false"#, 1),
            "wrong tags" => body = body.replacen("tag:soda-project", "tag:other", 1),
            "wrong preauthorization" => {
                body = body.replacen(r#""preauthorized":false"#, r#""preauthorized":true"#, 1)
            }
            "wrong key" => body = body.replacen("tskey-auth-", "tskey-client-", 1),
            "expired" => {
                body = key_response(&rfc3339(now), &rfc3339(now - Duration::from_secs(60)))
            }
            "long expiry" => {
                body = key_response(&rfc3339(now), &rfc3339(now + Duration::from_secs(3600)))
            }
            "invalid flag" => body = body.replacen(r#""id""#, r#""invalid":true,"id""#, 1),
            "revoked" => {
                body = body.replacen(
                    r#""id""#,
                    &format!(r#""revoked":"{}","id"#, rfc3339(now)),
                    1,
                )
            }
            "bad created" => body = body.replacen(&now_s, "not-a-time", 1),
            _ => unreachable!(),
        }
        (status, body.into_bytes())
    };
    for mode in [
        "redirect",
        "error",
        "oversize",
        "null",
        "missing reusable",
        "duplicate",
        "reusable",
        "persistent",
        "wrong tags",
        "wrong preauthorization",
        "wrong key",
        "expired",
        "long expiry",
        "invalid flag",
        "revoked",
        "bad created",
    ] {
        let (status, body) = mutate(mode);
        let e = provider::validate_key(status, &body, &tags, false, now, now).expect_err(mode);
        assert!(!e.contains("private"), "{mode}: {e}");
    }
    // Preauthorized policies admit preauthorized keys only.
    let pre = good.replacen(r#""preauthorized":false"#, r#""preauthorized":true"#, 1);
    assert!(provider::validate_key(200, pre.as_bytes(), &tags, true, now, now).is_ok());
}

// ---------- Control contract ----------

use soda_host::tailnet_companion::TailnetControl;

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
fn enroll_credential_and_key_guards() {
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;
    // Transport failures and bad bodies map to unavailable on the check path.
    for (reply, want) in [
        (Err("down".to_string()), "unavailable"),
        (Ok((403, br#"{"error":"x"}"#.to_vec())), "unavailable"),
        (Ok((200, br#"{}"#.to_vec())), "unavailable"),
    ] {
        let stub: Box<provider::ProviderTransport> = Box::new(move |_, _| reply.clone());
        let p = enroll::Provider::Stub(&stub);
        let e =
            enroll::check_credential(&p, &enrollment_fixture(), soon(5000)).expect_err("accepted");
        assert!(e.contains(want), "{e}");
    }
    // An expired deadline makes no provider call.
    let hit = Arc::new(AtomicU32::new(0));
    let stub: Box<provider::ProviderTransport> = Box::new({
        let hit = hit.clone();
        move |_, _| {
            hit.fetch_add(1, Ordering::SeqCst);
            Ok(token_ok())
        }
    });
    let p = enroll::Provider::Stub(&stub);
    let e =
        enroll::check_credential(&p, &enrollment_fixture(), Instant::now()).expect_err("accepted");
    assert!(e.contains("unavailable"), "{e}");
    assert_eq!(hit.load(Ordering::SeqCst), 0);
    // The key-minting probe rejects malformed stored policy as unavailable,
    // and an expired deadline as unconfirmed without provider calls.
    let stored = policy::EnrollmentPolicy {
        revision: "bogus".to_string(),
        ..Default::default()
    };
    let e = enroll::project_key(&p, &stored, &policy::Credential::default(), soon(5000))
        .expect_err("accepted");
    assert!(e.contains("unavailable"), "{e}");
    let stored = policy::EnrollmentPolicy {
        revision: "a".repeat(32),
        binding: "b".repeat(32),
        tailnet: "soda.example.test".to_string(),
        tags: vec!["tag:soda-project".to_string()],
        admission: true,
        credential: policy::Credential {
            client_id: "synthetic-client".to_string(),
            secret: "tskey-client-soda-synthetic-secret".to_string(),
        },
        ..Default::default()
    };
    let e =
        enroll::project_key(&p, &stored, &stored.credential, Instant::now()).expect_err("accepted");
    assert!(!e.contains("unavailable") && !e.contains("conflict"), "{e}");
    assert_eq!(hit.load(Ordering::SeqCst), 0);
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

#[test]
fn control_cli_passthrough() {
    let (opts, _) = control_options("control-cli", false);
    let c = control::Control::new(
        FakeExec {
            f: Box::new(|_, _, _, _| {
                Ok(br#"{"BackendState":"Running","Self":{"DNSName":"Atlas.Example.ts.net.","TailscaleIPs":["100.88.77.66"]},"CurrentTailnet":{"MagicDNSEnabled":true}}"#.to_vec())
            }),
        },
        opts,
    );
    let status = c.cli_status(soon(5000)).unwrap();
    assert_eq!(status.identity, "atlas.example.ts.net");
    let endpoint = c.cli_endpoint(soon(5000)).unwrap();
    assert_eq!(endpoint.ipv4, "100.88.77.66");
}
