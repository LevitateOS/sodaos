use super::node_tests::{binding, prefs_doc, status_doc};
use super::*;

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
        let body =
            format!(r#"{{"Version":"1.102.4","BackendState":"NeedsLogin","HaveNodeKey":{value}}}"#);
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
    let (state, _, _) = project_status(body.as_bytes(), prefs_doc().as_bytes(), &unsorted).unwrap();
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
    let body = r#"{"BackendState":"Running","haveNodeKey":true,"HaveNodeKey":null}"#;
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
