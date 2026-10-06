use super::*;

pub(crate) fn binding() -> RunBinding {
    RunBinding {
        enabled: true,
        admission: true,
        tailnet: "soda.example.test".to_string(),
        tags: vec!["tag:soda-project".to_string()],
    }
}

pub(crate) fn status_doc(state: &str) -> String {
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

pub(crate) fn prefs_doc() -> String {
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
