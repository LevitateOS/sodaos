//! Oracle tests for the PR26 tailnet control-plane port.
//!
//! The `tcontrol*.rs` modules are not in `lib.rs` yet, so they are included
//! here via `#[path]`; shared crate items are re-exported through shim
//! modules so the `crate::` paths inside the ported files resolve unchanged.
//! At lib wire-up these shims disappear with no changes to `tcontrol*.rs`.
//!
//! Test state lives under `target/tcontrol-test/` (never `/tmp`).

mod domain {
    pub use soda_host::domain::*;
}
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

#[path = "../src/tcontrol_wire.rs"]
mod tcontrol_wire;
#[path = "../src/tcontrol_policy.rs"]
mod tcontrol_policy;

use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

static SCRATCH_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Unique scratch directory under the package `target/` dir.
fn scratch(name: &str) -> PathBuf {
    let id = SCRATCH_COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = PathBuf::from("target/tcontrol-test")
        .join(format!("{}-{}-{id}", name, std::process::id()));
    std::fs::create_dir_all(&dir).expect("scratch dir");
    dir
}

use tcontrol_policy as policy;
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
    assert!(wire::valid_client_secret("tskey-client-soda-synthetic-secret"));
    assert!(!wire::valid_client_secret("tskey-client-short"));
    assert!(!wire::valid_client_secret(
        "tskey-client-soda-synthetic-secret?baseURL=https://attacker.invalid"
    ));
    assert!(!wire::valid_client_secret(
        "tskey-client-soda-synthetic-secret?ephemeral=false"
    ));
    assert!(!wire::valid_client_secret("tskey-auth-soda-synthetic-secret"));
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
    let big = format!(r#"{{"action":"signin","revision":"{rev}","confirm":"{}"}}"#, "x".repeat(65536));
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
    let r = wire::decode_enrollment_request(
        br#"{"action":"disable","revision":"0","tags":[]}"#,
    )
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
    assert!(wire::decode_project_selection(br#"{"enabled":true,"revision":"x","nope":1}"#).is_err());
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
    assert!(encoded.contains(r#""tailnet":"a\u003cb\u003e\u0026\"c\"""#), "{encoded}");
}

// ---------- Policy: revisions, reads, checks ----------

#[test]
fn policy_revision_format() {
    for _ in 0..8 {
        let r = policy::new_revision();
        assert!(wire::is_hex32(&r), "{r}");
    }
    assert_ne!(policy::new_revision(), policy::new_revision());
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
    let e = p.update(soon(5000), &input, &accept_check).expect_err("stale save");
    assert!(e.contains("conflict"), "{e}");
    let mut rotate = input.clone();
    rotate.action = "rotate".to_string();
    rotate.revision.clone_from(&first.enrollment.revision);
    rotate.client_secret = "tskey-client-another-synthetic-secret".to_string();
    let mut bad = rotate.clone();
    bad.tailnet = "other.example.test".to_string();
    let e = p.update(soon(5000), &bad, &accept_check).expect_err("cross-network");
    assert!(e.contains("conflict"), "{e}");
    let second = p.update(soon(5000), &rotate, &accept_check).unwrap();
    assert_eq!(second.enrollment.binding, first.enrollment.binding);
    assert_ne!(second.enrollment.revision, first.enrollment.revision);
    let dir = parent.join("soda-tailnet");
    let entries: Vec<_> = std::fs::read_dir(&dir).unwrap().collect();
    assert_eq!(entries.len(), 1);
    for entry in &entries {
        let entry = entry.as_ref().unwrap();
        assert_eq!(entry.metadata().unwrap().permissions().mode() & 0o777, 0o600);
        assert!(!entry.file_name().to_string_lossy().starts_with("credential-"));
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
    let first = p.update(soon(5000), &enrollment_fixture(), &accept_check).unwrap();
    let dir = parent.join("soda-tailnet");
    let retained = dir.join(format!("credential-{}.json", "a".repeat(32)));
    let secret = br#"{"client_id":"synthetic-client","secret":"tskey-client-retained-synthetic-secret"}"#;
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
    text.replace_range(cred_start..cred_end, &format!("\"credential\":\"{}\"", "a".repeat(32)));
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
    let seed = p.update(soon(5000), &enrollment_fixture(), &accept_check).unwrap();
    let mut input = enrollment_fixture();
    input.action = "rotate".to_string();
    input.revision = seed.enrollment.revision;
    let calls = Arc::new(AtomicU32::new(0));
    let run = |p: &policy::PolicyStore, input: &wire::EnrollmentRequest, calls: &Arc<AtomicU32>| {
        let calls = calls.clone();
        p.update(
            soon(10000),
            input,
            &|_, _| {
                calls.fetch_add(1, Ordering::SeqCst);
                Ok(())
            },
        )
    };
    let (a, b) = std::thread::scope(|s| {
        let ha = s.spawn(|| run(&p, &input, &calls).map(|_| ()).map_err(|e| e));
        let hb = s.spawn(|| run(&p, &input, &calls).map(|_| ()).map_err(|e| e));
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
            p.update(soon(5000), &enrollment_fixture(), &accept_check).unwrap();
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
                    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644)).unwrap();
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
    let first = p.update(soon(5000), &enrollment_fixture(), &accept_check).unwrap();
    p.sync_hook = Some(Box::new(|| Err("synthetic fsync failure".to_string())));
    let input = wire::EnrollmentRequest {
        action: "disable".to_string(),
        revision: first.enrollment.revision.clone(),
        ..Default::default()
    };
    let e = p.update(soon(5000), &input, &accept_check).expect_err("accepted");
    assert!(!e.contains("conflict") && !e.contains("invalid request"), "{e}");
    p.sync_hook = None;
    let after = p.enrollment(soon(5000)).unwrap();
    assert!(!after.admission && after.revision != first.enrollment.revision);
    let e = p.update(soon(5000), &input, &accept_check).expect_err("replayed");
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
    let e = p.project(soon(5000), &disable, &cid).expect_err("dup disable");
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
    let entries: Vec<_> = std::fs::read_dir(parent.join("soda-tailnet")).unwrap().collect();
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
    let e = p.project(soon(5000), &retarget, &cid).expect_err("retarget");
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
    let path = parent.join("soda-tailnet").join(format!("project-{project}.json"));
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

