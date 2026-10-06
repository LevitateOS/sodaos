use super::compose::{select_compose_child, write_override};
use super::launch_wire::{json_string, launch_request_json, parse_launch_exit, NestedRegistration};
use super::options::parse_bool_flag;
use super::registration::go_base;
use super::*;

#[test]
fn selects_only_requested_service_within_project() {
    let first = "a".repeat(64);
    let second = "b".repeat(64);
    let output = format!("{first}\n{second}\n");
    let child = select_compose_child(&output, "development", &|id| {
        if id == first {
            Ok(format!("{id} database\n"))
        } else {
            Ok(format!("{id} development\n"))
        }
    })
    .unwrap();
    assert_eq!(child, second);
}

#[test]
fn rejects_ambiguous_or_unconfirmed_identity() {
    let id = "a".repeat(64);
    for output in ["", "--all", &format!("{id}\n{id}")] {
        let out = if output == "--all" {
            "--all".to_string()
        } else {
            output.to_string()
        };
        assert!(
            select_compose_child(&out, "development", &|v| Ok(format!("{v} development"))).is_err(),
            "unconfirmed unique container admitted for {out:?}"
        );
    }
    assert!(
        select_compose_child(&id, "development", &|_| Err(String::from(
            "runtime unavailable"
        )))
        .is_err()
    );
}

#[test]
fn resolves_upstream_short_handle_to_immutable_identity() {
    let id = "a".repeat(64);
    let child = select_compose_child(&id[..12], "development", &|_| {
        Ok(format!("{id} development"))
    })
    .unwrap();
    assert_eq!(child, id);
    assert!(select_compose_child(&id[..12], "development", &|_| {
        Ok(format!("{} development", "b".repeat(64)))
    })
    .is_err());
}

#[test]
fn compose_muse_mounts_are_explicit() {
    let dir = std::env::temp_dir().join(format!("soda-compose-test-{}", std::process::id()));
    let _ = fs::create_dir_all(&dir);
    let path = dir.join("compose.json");
    write_override(
        path.to_str().unwrap(),
        "development",
        "/run/soda-muse/nested/registration",
    )
    .unwrap();
    let body = fs::read(&path).unwrap();
    let text = String::from_utf8(body).unwrap();
    assert_eq!(
        text,
        "{\"services\":{\"development\":{\"volumes\":[\"/run/soda-muse/nested/registration:/run/soda-muse/credentials:ro\",\"/usr/local/bin/muse:/usr/local/bin/muse:ro\",\"/usr/local/libexec/soda/muse:/usr/local/libexec/soda/muse:ro\",\"/run/soda-muse-interface:/run/soda-muse-interface:ro\"]}}}"
    );
    let _ = fs::remove_dir_all(&dir);
}

#[test]
fn override_escapes_like_go_encoding_json() {
    assert_eq!(
        json_string("a<b>&\"c\\d"),
        "\"a\\u003cb\\u003e\\u0026\\\"c\\\\d\""
    );
    assert_eq!(json_string("line\ntab\t"), "\"line\\ntab\\t\"");
    assert_eq!(json_string("\u{1}"), "\"\\u0001\"");
}

#[test]
fn go_base_matches_filepath_semantics() {
    assert_eq!(go_base("login"), "login");
    assert_eq!(go_base("a/b"), "b");
    assert_eq!(go_base("/"), "/");
    assert_eq!(go_base(""), ".");
}

#[test]
fn bool_flag_values_match_go() {
    assert_eq!(parse_bool_flag("true"), Some(true));
    assert_eq!(parse_bool_flag("1"), Some(true));
    assert_eq!(parse_bool_flag("false"), Some(false));
    assert_eq!(parse_bool_flag("0"), Some(false));
    assert_eq!(parse_bool_flag("yes"), None);
}

#[test]
fn launch_request_wire_matches_go() {
    let req = NestedRegistration {
        child_id: "c".repeat(64),
        actor_id: "42".to_string(),
        registration_id: "r".repeat(32),
        muse: true,
    };
    let wire = launch_request_json(&req);
    let expect = format!(
        "{{\"register\":{{\"child_id\":\"{}\",\"actor_id\":\"42\",\"registration_id\":\"{}\",\"muse\":true}},\"connection_id\":\"\",\"cwd\":\"\",\"args\":null,\"tty\":false,\"cols\":0,\"rows\":0}}",
        "c".repeat(64),
        "r".repeat(32)
    );
    assert_eq!(wire, expect);
}

#[test]
fn launch_exit_parsing_matches_go_unmarshal() {
    assert_eq!(
        parse_launch_exit(b"{\"code\":0}").unwrap(),
        (0, String::new())
    );
    assert_eq!(parse_launch_exit(b"{}").unwrap(), (0, String::new()));
    assert_eq!(
        parse_launch_exit(b"{\"error\":\"denied\",\"code\":1}").unwrap(),
        (1, String::from("denied"))
    );
    assert_eq!(
        parse_launch_exit(b" { \"code\" : 0 , \"extra\" : [1,{\"x\":null}] } ").unwrap(),
        (0, String::new())
    );
    assert_eq!(
        parse_launch_exit(b"{\"code\":0,\"error\":\"a\\\"b\"}").unwrap(),
        (0, String::from("a\"b"))
    );
    for bad in [
        "",
        "{",
        "{\"code\":}",
        "{\"code\":\"0\"}",
        "{\"code\":0.0}",
        "{\"code\":0",
        "{\"error\":0}",
        "{\"code\":0}trailing",
        "{\"code\":0,\"code\":}",
        "{\"code\":0,}",
        "{,}",
    ] {
        assert!(
            parse_launch_exit(bad.as_bytes()).is_err(),
            "admitted {bad:?}"
        );
    }
}
