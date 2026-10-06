//! Tailnet domain mirrors: error texts, DTOs, matchers, time/JSON helpers.
//! Lane A owns this file.
//!
//! Pure port of the `internal/tailnet` project-status surface: `ProjectHasNode`
//! and `ProjectStatus` (`project_status.go`), the `nativeObject` / `peerView` /
//! `addresses` helpers (`control.go`), `CanonicalMagicDNSName` (`tailnet.go`),
//! the project DTOs (`control_types.go`), `RunTarget` (`enrollment.go`) and
//! `RunBinding` (`project_runtime.go`). Go stdlib edge semantics (JSON,
//! `netip`, `time`) were verified against the pinned toolchain with probes.

use crate::domain;

pub const ERR_INVALID: &str = "invalid Tailnet request";
pub const ERR_CONFLICT: &str = "tailnet revision or identity changed";
pub const ERR_UNSUPPORTED: &str = "tailnet runtime is not supported";
pub const ERR_UNCONFIRMED: &str = "tailnet outcome is unconfirmed";
pub const ERR_UNAVAILABLE: &str = "tailscale status is unavailable";

/// Native LocalAPI body cap (`responseLimit` in control.go).
const RESPONSE_LIMIT: usize = 65536;

pub type ReadFile = Box<dyn Fn(&str) -> Result<Vec<u8>, String>>;
pub type LinkFile = Box<dyn Fn(&str) -> Result<String, String>>;
pub type StatFile = Box<dyn Fn(&str) -> Result<std::fs::Metadata, String>>;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunTarget {
    pub project: String,
    pub container: String,
    pub run: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunBinding {
    pub enabled: bool,
    pub admission: bool,
    pub tailnet: String,
    pub tags: Vec<String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectRequest {
    pub project: String,
    pub action: String,
    pub revision: String,
    pub binding: String,
    pub confirm_id: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectView {
    pub available_binding: String,
    pub available_network: String,
    pub addresses: Vec<String>,
    pub dns_name: String,
    pub saved: bool,
    pub project: String,
    pub revision: String,
    pub binding: String,
    pub enabled: bool,
    pub state: String,
    pub outcome: String,
}

/// `^p[0-9a-f]{24}$`
pub fn valid_project_id(s: &str) -> bool {
    domain::valid_id(s)
}

/// `^[0-9a-f]{64}$`
pub fn valid_container_id(s: &str) -> bool {
    domain::valid_container_id(s)
}

/// `^(sha256:)?[0-9a-f]{64}$`
pub fn valid_image_id(s: &str) -> bool {
    domain::valid_image_ref(s)
}

fn unavailable<T>() -> Result<T, String> {
    Err(ERR_UNAVAILABLE.to_string())
}
#[path = "tailnet/domain/time.rs"]
mod time;

pub use time::{go_escape, parse_rfc3339_nano};

#[path = "tailnet/domain/native.rs"]
mod native;

use native::{
    bind_bool_into, bind_string_into, decode_native_prefs, decode_native_status, fold_eq,
    lower_char, native_object, NativeStatus, SelfPeer,
};

#[path = "tailnet/domain/addresses.rs"]
mod addresses;

use addresses::{canonical_magic_dns_name, resolve_project_peer};

/// Mirror of `parseProjectStatus`: the decoded status plus a terminal outcome
/// (`""` means Running, continue to preferences and binding checks).
fn parse_project_status(data: &[u8]) -> Result<(NativeStatus, String), String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let s = decode_native_status(&v)?;
    let outcome = match s.backend_state.as_str() {
        "NeedsLogin" => "needs-login",
        "NeedsMachineAuth" => "approval-required",
        "Starting" | "NoState" | "Stopped" => "pending",
        "Running" => "",
        _ => "unconfirmed",
    };
    Ok((s, outcome.to_string()))
}

/// Mirror of `validateProjectPreferences`.
fn validate_project_preferences(preferences: &[u8]) -> Result<(), String> {
    if preferences.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(
        preferences,
        &[
            "WantRunning",
            "CorpDNS",
            "RouteAll",
            "RunSSH",
            "ExitNodeID",
            "ExitNodeIP",
            "AdvertiseRoutes",
        ],
    )?;
    let p = decode_native_prefs(&v)?;
    if !p.want_running || !p.corp_dns || p.route_all || p.run_ssh {
        return Err(ERR_CONFLICT.to_string());
    }
    if !p.exit_node_id.is_empty() || !p.exit_node_ip.is_empty() || !p.advertise_routes.is_empty() {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

/// Mirror of `matchProjectSelf` (binding tags compared as-is, like Go).
fn match_project_self(peer: Option<&SelfPeer>, tags: &[String]) -> bool {
    let p = match peer {
        Some(p) => p,
        None => return false,
    };
    if p.id.is_empty() || !p.online || p.expired {
        return false;
    }
    let mut cloned = p.tags.clone();
    cloned.sort();
    cloned == tags
}

/// Mirror of `matchProjectBinding`.
fn match_project_binding(s: &NativeStatus, binding: &RunBinding) -> bool {
    if !binding.enabled || !s.have_node_key {
        return false;
    }
    match &s.tailnet {
        Some(name) if name == &binding.tailnet => {}
        _ => return false,
    }
    match_project_self(s.peer.as_ref(), &binding.tags)
}

/// Mirror of `ProjectStatus`: `(outcome, addresses, dns_name)`.
pub fn project_status(
    data: &[u8],
    prefs: &[u8],
    binding: &RunBinding,
) -> Result<(String, Vec<String>, String), String> {
    let (s, outcome) = parse_project_status(data)?;
    if !outcome.is_empty() {
        return Ok((outcome, Vec::new(), String::new()));
    }
    validate_project_preferences(prefs)?;
    if !match_project_binding(&s, binding) {
        return Ok(("unconfirmed".to_string(), Vec::new(), String::new()));
    }
    let peer = match &s.peer {
        Some(p) => p,
        None => return Ok(("unconfirmed".to_string(), Vec::new(), String::new())),
    };
    let (addrs, dns_name) = resolve_project_peer(&peer.dns_name, &peer.ips)?;
    Ok(("connected".to_string(), addrs, dns_name))
}

/// Mirror of `ProjectHasNode`.
pub fn project_has_node(data: &[u8]) -> Result<bool, String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut backend = String::new();
    let mut key = false;
    bind_string_into(fields, "BackendState", &mut backend)?;
    bind_bool_into(fields, "HaveNodeKey", &mut key)?;
    match backend.as_str() {
        "NoState" | "NeedsLogin" | "NeedsMachineAuth" | "Stopped" | "Starting" => {}
        "Running" => {
            if !key {
                return unavailable();
            }
        }
        _ => return unavailable(),
    }
    Ok(key)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn binding() -> RunBinding {
        RunBinding {
            enabled: true,
            admission: true,
            tailnet: "soda.example.test".to_string(),
            tags: vec!["tag:soda-project".to_string()],
        }
    }

    fn status_doc(state: &str) -> String {
        format!(
            concat!(
                r#"{{"Version":"1.102.4","BackendState":"{state}","HaveNodeKey":true,"#,
                r#""CurrentTailnet":{{"Name":"soda.example.test"}},"#,
                r#""Self":{{"ID":"node-project-a","Online":true,"DNSName":"project.soda.ts.net.","#,
                r#""TailscaleIPs":["100.64.0.2"],"Tags":["tag:soda-project"]}},"#,
                r#""AuthURL":"private","Health":["private"]}}"#
            ),
            state = state
        )
    }

    fn prefs_doc() -> String {
        r#"{"WantRunning":true,"CorpDNS":true,"RouteAll":false,"RunSSH":false,"ExitNodeID":"","ExitNodeIP":"","AdvertiseRoutes":null,"Persist":{"PrivateNodeKey":"private"}}"#
            .to_string()
    }

    #[test]
    fn has_node_vectors() {
        // Ported from TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey.
        for (body, node, invalid) in [
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#,
                false,
                false,
            ),
            (
                r#"{"Version":"other","BackendState":"Running","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"NeedsMachineAuth","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":true}"#,
                true,
                false,
            ),
            (
                r#"{"BackendState":"Running","HaveNodeKey":false}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"Unknown","HaveNodeKey":false}"#,
                false,
                true,
            ),
            (r#"{"BackendState":"NeedsLogin"}"#, false, false),
            (r#"{"BackendState":"Running"}"#, false, true),
            (r#"{"HaveNodeKey":false}"#, false, true),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":null}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":"false"}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":0}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","haveNodeKey":false}"#,
                false,
                true,
            ),
            (
                r#"{"BackendState":"NeedsLogin","HaveNodeKey":false,"HaveNodeKey":true}"#,
                false,
                true,
            ),
            ("null", false, true),
        ] {
            let r = project_has_node(body.as_bytes());
            match r {
                Ok(v) => assert!(
                    !invalid && v == node,
                    "{body}: got node={v}, want node={node} invalid={invalid}"
                ),
                Err(e) => assert!(
                    invalid && e == ERR_UNAVAILABLE,
                    "{body}: got err={e:?}, want node={node} invalid={invalid}"
                ),
            }
        }
    }

    #[test]
    fn has_node_state_table() {
        for (state, key, ok, node) in [
            ("NoState", false, true, false),
            ("NeedsLogin", false, true, false),
            ("NeedsMachineAuth", true, true, true),
            ("Stopped", true, true, true),
            ("Starting", false, true, false),
            ("Running", true, true, true),
            ("Running", false, false, false),
            ("InUseOtherUser", true, false, false),
            ("", true, false, false),
            ("Bogus", true, false, false),
        ] {
            let body = format!(r#"{{"BackendState":"{state}","HaveNodeKey":{key}}}"#);
            let r = project_has_node(body.as_bytes());
            if ok {
                assert_eq!(r.unwrap(), node, "{body}");
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{body}");
            }
        }
    }

    #[test]
    fn status_fresh_daemon_omits_node_key() {
        // Ported from TestProjectStatusFreshDaemonOmitsFalseNodeKey.
        let (state, ips, dns) = project_status(
            br#"{"Version":"1.102.4","BackendState":"NeedsLogin"}"#,
            &[],
            &RunBinding::default(),
        )
        .unwrap();
        assert_eq!(state, "needs-login");
        assert!(ips.is_empty());
        assert!(dns.is_empty());
        for value in ["null", r#""false""#, "0"] {
            let body = format!(
                r#"{{"Version":"1.102.4","BackendState":"NeedsLogin","HaveNodeKey":{value}}}"#
            );
            assert_eq!(
                project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{value}"
            );
        }
    }

    #[test]
    fn status_connected_and_unsafe_prefs() {
        // Ported from TestProjectStatusRequiresExactNetworkTagsAndNativePreferences.
        let (state, ips, dns) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &binding(),
        )
        .unwrap();
        assert_eq!(state, "connected");
        assert_eq!(ips, vec!["100.64.0.2".to_string()]);
        assert_eq!(dns, "project.soda.ts.net");
        for prefs in [
            prefs_doc().replace(r#""CorpDNS":true"#, r#""CorpDNS":false"#),
            prefs_doc().replace(r#""RouteAll":false"#, r#""RouteAll":true"#),
            prefs_doc().replace(r#""RunSSH":false"#, r#""RunSSH":true"#),
            prefs_doc().replace(r#""ExitNodeID":"""#, r#""ExitNodeID":"foreign""#),
            prefs_doc().replace(r#""ExitNodeIP":"""#, r#""ExitNodeIP":"1.2.3.4""#),
            prefs_doc().replace(
                r#""AdvertiseRoutes":null"#,
                r#""AdvertiseRoutes":["0.0.0.0/0"]"#,
            ),
            prefs_doc().replace(r#""WantRunning":true"#, r#""WantRunning":false"#),
        ] {
            assert_eq!(
                project_status(
                    status_doc("Running").as_bytes(),
                    prefs.as_bytes(),
                    &binding()
                )
                .unwrap_err(),
                ERR_CONFLICT,
                "{prefs}"
            );
        }
    }

    #[test]
    fn status_binding_mismatch_is_unconfirmed() {
        let mut bad = binding();
        bad.tags = vec!["tag:other".to_string()];
        let (state, ips, _) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &bad,
        )
        .unwrap();
        assert_eq!(state, "unconfirmed");
        assert!(ips.is_empty());
        // Unsorted binding tags never match (Go compares the raw slice).
        let mut unsorted = binding();
        unsorted.tags = vec!["tag:soda-project".to_string(), "tag:aaa".to_string()];
        let body = status_doc("Running").replace(
            r#""Tags":["tag:soda-project"]"#,
            r#""Tags":["tag:aaa","tag:soda-project"]"#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &unsorted).unwrap();
        assert_eq!(state, "unconfirmed");
        // Disabled binding, wrong tailnet, missing tailnet, offline/expired/empty self.
        for body in [
            status_doc("Running").replace(r#""Online":true"#, r#""Online":false"#),
            status_doc("Running").replace(
                r#""ID":"node-project-a""#,
                r#""ID":"node-project-a","Expired":true"#,
            ),
            status_doc("Running").replace(r#""ID":"node-project-a""#, r#""ID":"""#),
            status_doc("Running").replace("soda.example.test", "other.example.test"),
            status_doc("Running").replace(
                r#""CurrentTailnet":{"Name":"soda.example.test"}"#,
                r#""CurrentTailnet":null"#,
            ),
        ] {
            let (state, _, _) =
                project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
            assert_eq!(state, "unconfirmed", "{body}");
        }
        let (state, _, _) = project_status(
            status_doc("Running").as_bytes(),
            prefs_doc().as_bytes(),
            &RunBinding::default(),
        )
        .unwrap();
        assert_eq!(state, "unconfirmed");
    }

    #[test]
    fn status_non_running_outcomes() {
        for (native, want) in [
            ("NeedsMachineAuth", "approval-required"),
            ("NeedsLogin", "needs-login"),
            ("Starting", "pending"),
            ("NoState", "pending"),
            ("Stopped", "pending"),
            ("Bogus", "unconfirmed"),
            ("InUseOtherUser", "unconfirmed"),
        ] {
            let (state, ips, dns) =
                project_status(status_doc(native).as_bytes(), &[], &RunBinding::default()).unwrap();
            assert_eq!(state, want, "{native}");
            assert!(ips.is_empty());
            assert!(dns.is_empty());
        }
        // Release numbers are not a runtime veto.
        let body = status_doc("NeedsLogin").replace("1.102.4", "a different release");
        let (state, _, _) = project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap();
        assert_eq!(state, "needs-login");
    }

    #[test]
    fn status_malformed_inputs() {
        // Running with missing prefs body is unavailable, not unconfirmed.
        assert_eq!(
            project_status(status_doc("Running").as_bytes(), &[], &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        // Prefs missing a required key, or mistyped, is unavailable.
        for prefs in [
            prefs_doc().replace(r#""CorpDNS":true,"#, ""),
            prefs_doc().replace(r#""WantRunning":true"#, r#""WantRunning":"yes""#),
            prefs_doc().replace(r#""AdvertiseRoutes":null"#, r#""AdvertiseRoutes":"x""#),
            prefs_doc().replace(
                r#""AdvertiseRoutes":null"#,
                r#""AdvertiseRoutes":null,"AdvertiseRoutes":[]"#,
            ),
        ] {
            assert_eq!(
                project_status(
                    status_doc("Running").as_bytes(),
                    prefs.as_bytes(),
                    &binding()
                )
                .unwrap_err(),
                ERR_UNAVAILABLE,
                "{prefs}"
            );
        }
        // `[null]` routes decode to `[""]`, which is unsafe -> conflict.
        let prefs = prefs_doc().replace(r#""AdvertiseRoutes":null"#, r#""AdvertiseRoutes":[null]"#);
        assert_eq!(
            project_status(
                status_doc("Running").as_bytes(),
                prefs.as_bytes(),
                &binding()
            )
            .unwrap_err(),
            ERR_CONFLICT
        );
        // Status mistyped optionals fail closed.
        for body in [
            status_doc("Running").replace(r#""Self":{"#, r#""Self":"x","Self2":{"#),
            status_doc("Running").replace(r#""Online":true"#, r#""Online":"yes""#),
            status_doc("Running").replace(
                r#""TailscaleIPs":["100.64.0.2"]"#,
                r#""TailscaleIPs":[true]"#,
            ),
            status_doc("Running").replace(
                r#""CurrentTailnet":{"Name":"soda.example.test"}"#,
                r#""CurrentTailnet":[]"#,
            ),
            status_doc("Running").replace(r#""BackendState":"Running""#, r#""BackendState":123"#),
        ] {
            assert_eq!(
                project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{body}"
            );
        }
        // Nested exact-duplicate keys are rejected by the strict gate.
        let body = status_doc("Running").replace(r#""ID":"#, r#""ID":"x","ID":"#);
        assert_eq!(
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        // Trailing data and non-objects are rejected.
        for raw in ["null", "[]", "42", "", "{},", "{\"BackendState\""] {
            assert_eq!(
                project_status(raw.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
                ERR_UNAVAILABLE,
                "{raw}"
            );
        }
        let big_ok = format!(
            r#"{{"BackendState":"NeedsLogin","HaveNodeKey":false{}}}"#,
            ""
        );
        assert!(project_status(big_ok.as_bytes(), &[], &RunBinding::default()).is_ok());
    }

    #[test]
    fn status_case_fold_binding() {
        // Case-variant keys bind with last-in-document-wins; pointer structs merge.
        let body = status_doc("Running").replace(
            r#""ID":"node-project-a","#,
            r#""id":"lower","ID":"node-project-a","#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "connected");
        // Reversed order: the fold-equal key comes last and wins (empty ID).
        let body = status_doc("Running").replace(
            r#""ID":"node-project-a","#,
            r#""ID":"node-project-a","id":"","#,
        );
        let (state, _, _) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "unconfirmed");
        // Pointer structs merge across case-variant keys (Go reuses the struct).
        let body = r#"{"Version":"1.102.4","BackendState":"Running","HaveNodeKey":true,"CurrentTailnet":{"Name":"soda.example.test"},"Self":{"ID":"node-project-a","Online":true,"TailscaleIPs":["100.64.0.2"],"Tags":["tag:soda-project"]},"self":{"DNSName":"project.soda.ts.net."},"AuthURL":"private","Health":["private"]}"#;
        let (state, ips, dns) =
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap();
        assert_eq!(state, "connected");
        assert_eq!(ips, vec!["100.64.0.2".to_string()]);
        assert_eq!(dns, "project.soda.ts.net");
        // A case-aliased HaveNodeKey blocks the omission rule.
        let body = r#"{"BackendState":"NeedsLogin","havenodekey":false}"#;
        assert_eq!(
            project_status(body.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
            ERR_UNAVAILABLE
        );
    }

    #[test]
    fn status_response_limit() {
        let mut big = format!(
            r#"{{"BackendState":"NeedsLogin","Pad":"{}"}}"#,
            "x".repeat(65536)
        );
        assert!(big.len() > RESPONSE_LIMIT);
        assert_eq!(
            project_status(big.as_bytes(), &[], &RunBinding::default()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        assert_eq!(
            project_has_node(big.as_bytes()).unwrap_err(),
            ERR_UNAVAILABLE
        );
        big = format!(
            r#"{{"BackendState":"NeedsLogin","Pad":"{}"}}"#,
            "x".repeat(1000)
        );
        assert!(project_status(big.as_bytes(), &[], &RunBinding::default()).is_ok());
    }

    #[test]
    fn status_peer_address_rules() {
        // Each address list replaces the fixture's single IPv4; all must be
        // canonical global-unicast forms or the whole status is unavailable.
        for (ips, ok) in [
            (r#"["100.64.0.2"]"#, true),
            (r#"["10.0.0.1","192.168.1.1"]"#, true),
            (r#"["fd7a:115c:a1e0::1"]"#, true),
            (r#"["2001:db8::1"]"#, true),
            (r#"["::ffff:1.2.3.4"]"#, true),
            (r#"["1:2:3:4:5:6:7:8"]"#, true),
            (r#"["fd7a::1%ETH0"]"#, true),
            (r#"["127.0.0.1"]"#, false),
            (r#"["0.0.0.0"]"#, false),
            (r#"["169.254.1.1"]"#, false),
            (r#"["224.0.0.1"]"#, false),
            (r#"["255.255.255.255"]"#, false),
            (r#"["::1"]"#, false),
            (r#"["::"]"#, false),
            (r#"["fe80::1"]"#, false),
            (r#"["ff02::1"]"#, false),
            (r#"["::ffff:127.0.0.1"]"#, false),
            (r#"["01.2.3.4"]"#, false),
            (r#"["FD7A::1"]"#, false),
            (r#"["fd7a:115c:a1e0:0:0:0:0:1"]"#, false),
            (r#"["1.2.3.4%eth0"]"#, false),
            (r#"["100.64.0.2%"]"#, false),
            (r#"["1.2.3.4 "]"#, false),
            (r#"["abc"]"#, false),
            (r#"[]"#, false),
            (r#"null"#, false),
        ] {
            let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, ips);
            let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
            if ok {
                assert_eq!(r.unwrap().0, "connected", "{ips}");
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{ips}");
            }
        }
        // More than 16 addresses is unavailable.
        let many = "[".to_string() + &vec!["\"100.64.0.2\""; 17].join(",") + "]";
        let body = status_doc("Running").replace(r#"["100.64.0.2"]"#, &many);
        assert_eq!(
            project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding()).unwrap_err(),
            ERR_UNAVAILABLE
        );
    }

    #[test]
    fn status_dns_name_rules() {
        let long_label = "a".repeat(64);
        let long_name = format!("{}.example.ts.net", "a".repeat(240));
        for (dns, ok) in [
            ("project.soda.ts.net.", true),
            ("Atlas.Example.ts.net.", true),
            ("  atlas.example.ts.net  ", true),
            ("a.b", true),
            ("a.b..", false),
            ("atlas.local", false),
            ("atlas.LOCAL.", false),
            ("nodot", false),
            ("", true),
            ("-a.b", false),
            ("a-.b", false),
            ("a.B_c", false),
            ("a..b", false),
            (".a.b", false),
            ("exa mple.ts.net", false),
            (&long_label, false),
            (&long_name, false),
        ] {
            let body = status_doc("Running").replace("project.soda.ts.net.", dns);
            let r = project_status(body.as_bytes(), prefs_doc().as_bytes(), &binding());
            if ok {
                let (state, _, name) = r.unwrap();
                assert_eq!(state, "connected", "{dns:?}");
                assert_eq!(name, dns.trim().trim_end_matches('.').to_lowercase());
            } else {
                assert_eq!(r.unwrap_err(), ERR_UNAVAILABLE, "{dns:?}");
            }
        }
        assert_eq!(
            canonical_magic_dns_name("Atlas.Example.ts.net.").unwrap(),
            "atlas.example.ts.net"
        );
    }

    #[test]
    fn dns_trims_nel_and_fold_covers_simple_fold_orbits() {
        // Go's strings.TrimSpace also trims U+0085 (NEL).
        assert_eq!(
            canonical_magic_dns_name("atlas.example.ts.net").unwrap(),
            "atlas.example.ts.net"
        );
        // Go's non-ASCII mates of ASCII letters: exactly ſ (S/s) and
        // Kelvin K (K/k). İ (U+0130) does NOT fold with i.
        assert!(fold_eq("ſ", "s"));
        assert!(fold_eq("S", "ſ"));
        assert!(fold_eq("Backendſtate", "BackendState"));
        assert!(fold_eq("\u{212a}", "k"));
        assert!(fold_eq("K", "\u{212a}"));
        assert!(!fold_eq("İ", "i"));
        assert!(!fold_eq("BackendState", "BackendStates"));
        assert!(!fold_eq("ß", "ss"));
    }

    #[test]
    fn id_matchers() {
        assert!(valid_project_id(&("p".to_string() + &"a".repeat(24))));
        for bad in [
            "".to_string(),
            "p".to_string(),
            "P".to_string() + &"a".repeat(24),
            "p".to_string() + &"a".repeat(23),
            "p".to_string() + &"a".repeat(25),
            "p".to_string() + &"b".repeat(23) + "q",
            "p".to_string() + &"B".repeat(24),
        ] {
            assert!(!valid_project_id(&bad), "{bad}");
        }
        assert!(valid_container_id(&"c".repeat(64)));
        for bad in [
            "",
            &"c".repeat(63),
            &("c".repeat(63) + "C"),
            &"c".repeat(65),
        ] {
            assert!(!valid_container_id(bad), "{bad}");
        }
        assert!(valid_image_id(&"d".repeat(64)));
        assert!(valid_image_id(&("sha256:".to_string() + &"d".repeat(64))));
        for bad in [
            "",
            "sha256:",
            &("sha256:".to_string() + &"d".repeat(63)),
            &("SHA256:".to_string() + &"d".repeat(64)),
            &("e".repeat(65)),
        ] {
            assert!(!valid_image_id(bad), "{bad}");
        }
        assert_eq!(ERR_INVALID, "invalid Tailnet request");
        assert_eq!(ERR_CONFLICT, "tailnet revision or identity changed");
        assert_eq!(ERR_UNSUPPORTED, "tailnet runtime is not supported");
        assert_eq!(ERR_UNCONFIRMED, "tailnet outcome is unconfirmed");
        assert_eq!(ERR_UNAVAILABLE, "tailscale status is unavailable");
    }

    #[test]
    fn rfc3339_vectors() {
        for (s, want) in [
            ("0001-01-01T00:00:00Z", Some(true)),
            ("0001-01-01T00:00:00.000000000Z", Some(true)),
            ("0001-01-01T00:00:00+00:00", Some(true)),
            ("0001-01-01T01:00:00+01:00", Some(true)),
            ("0000-12-31T23:00:00-01:00", Some(true)),
            ("0001-01-01T00:00:00.000000001Z", Some(false)),
            ("0001-01-01T00:30:00+01:00", Some(false)),
            ("2024-02-29T00:00:00Z", Some(false)),
            ("2000-02-29T12:30:45.123Z", Some(false)),
            ("1900-02-28T00:00:00Z", Some(false)),
            ("0000-01-01T00:00:00Z", Some(false)),
            ("9999-12-31T23:59:59Z", Some(false)),
            ("2024-01-01T00:00:00.5Z", Some(false)),
            // Go keeps nanosecond precision and ignores further digits.
            ("2024-01-01T00:00:00.1234567891Z", Some(false)),
            ("2024-01-01T00:00:00.0000000001Z", Some(false)),
            (
                "2024-01-01T00:00:00.123456789012345678901234567890Z",
                Some(false),
            ),
            (
                "0001-01-01T00:00:00.000000000000000000000000000000Z",
                Some(true),
            ),
            ("2024-01-01T00:00:00+24:00", Some(false)),
            ("2024-01-01T00:00:00+24:60", Some(false)),
            ("2024-01-01T00:00:00-24:60", Some(false)),
            ("2024-01-01T00:00:00+00:60", Some(false)),
            ("2024-01-01T00:00:00-00:00", Some(false)),
            ("2023-02-29T00:00:00Z", None),
            ("1900-02-29T00:00:00Z", None),
            ("2024-13-01T00:00:00Z", None),
            ("2024-00-10T00:00:00Z", None),
            ("2024-01-00T00:00:00Z", None),
            ("2024-01-32T00:00:00Z", None),
            ("2024-04-31T00:00:00Z", None),
            ("2024-01-01T24:00:00Z", None),
            ("2024-01-01T00:00:60Z", None),
            ("2024-01-01T00:00:00", None),
            ("2024-01-01T00:00:00.Z", None),
            ("2024-01-01T00:00:00.", None),
            ("2024-01-01T00:00:00+07", None),
            ("2024-01-01T00:00:00+0700", None),
            ("2024-01-01T00:00:00+07:00:00", None),
            ("2024-01-01T00:00:00+25:00", None),
            ("2024-01-01T00:00:00+24:61", None),
            ("2024-01-01T00:00:00+00:61", None),
            ("2024-01-01T00:00:00+99:99", None),
            ("2024-01-01t00:00:00z", None),
            ("2024-01-01T00:00:00z", None),
            ("2024-1-1T00:00:00Z", None),
            ("2024-01-01T1:00:00Z", Some(false)),
            ("2024-06-15T0:30:45.123456789+05:30", Some(false)),
            ("2024-01-01T123:00:00Z", None),
            ("2024-01-01T01:2:03Z", None),
            ("2024-01-01T01:02:3Z", None),
            ("24-01-01T00:00:00Z", None),
            ("20240-01-01T00:00:00Z", None),
            ("2024-01-01", None),
            ("2024-01-01T00:00", None),
            ("2024-01-01T00:00:00Z ", None),
            (" 2024-01-01T00:00:00Z", None),
            ("", None),
        ] {
            assert_eq!(parse_rfc3339_nano(s), want, "{s}");
        }
        // Go accepts 10+ fraction digits and keeps nanosecond precision.
        assert_eq!(
            parse_rfc3339_nano("2024-01-01T00:00:00.1234567890Z"),
            Some(false)
        );
    }

    #[test]
    fn escape_vectors() {
        for (raw, want) in [
            ("", r#""""#),
            ("abc", r#""abc""#),
            ("a\"b\\c", r#""a\"b\\c""#),
            ("a\nb\rc\td", r#""a\nb\rc\td""#),
            ("\u{8}\u{c}", r#""\b\f""#),
            ("\u{1}\u{1f}", r#""\u0001\u001f""#),
            ("<>&", r#""\u003c\u003e\u0026""#),
            ("é☃", r#""é☃""#),
            ("\u{7f}", "\"\u{7f}\""),
        ] {
            assert_eq!(go_escape(raw), want, "{raw:?}");
        }
        assert_eq!(go_escape("\u{2028}"), r#""\u2028""#);
        assert_eq!(go_escape("\u{2029}"), r#""\u2029""#);
    }
}
