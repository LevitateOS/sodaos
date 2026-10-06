use super::*;

#[test]
fn hostname_vectors() {
    // Ported from TestHostname* in installer_test.go plus edges.
    for good in [
        "soda-01",
        "a",
        "a.b.c",
        "host123",
        "x-y-z",
        &"a".repeat(63),
        &format!("{}.{}", "a".repeat(63), "b".repeat(63)),
    ] {
        assert!(hostname(good), "reject {good:?}");
    }
    assert!(hostname(&format!(
        "{}.{}.{}.{}",
        "a".repeat(63),
        "b".repeat(63),
        "c".repeat(63),
        "d".repeat(61)
    )));
    assert!(!hostname(&"a".repeat(253)));
    for bad in [
        "",
        &"a".repeat(254),
        "UPPER",
        "-lead",
        "trail-",
        "under_score",
        "white space",
        "double..dot",
        ".leading",
        "trailing.",
        &"a".repeat(64),
        "host!",
        "ho/st",
        "é",
    ] {
        assert!(!hostname(bad), "accept {bad:?}");
    }
}

#[test]
fn subnet_vectors() {
    assert!(project_subnet("10.89.0.0/24").is_ok());
    assert!(project_subnet("0.0.0.0/0").is_ok());
    assert!(project_subnet("192.168.1.1/32").is_ok());
    for bad in [
        "10.89.0.1/24",
        "fd00::/64",
        "10.0.0.0/33",
        "not-a-subnet",
        "10.0.0.0/8 ",
        "",
    ] {
        assert_eq!(
            project_subnet(bad).unwrap_err().to_string(),
            "canonical IPv4 project subnet required",
            "input {bad:?}"
        );
    }
}

#[test]
fn password_hash_vectors() {
    let good = format!("$6${}${}", "s".repeat(8), "h".repeat(86));
    assert!(valid_password_hash(&good));
    assert!(valid_password_hash(&format!("$6$s${}", "h".repeat(86))));
    assert!(valid_password_hash(&format!(
        "$6${}${}",
        "s".repeat(16),
        "h".repeat(86)
    )));
    for bad in [
        format!("$6$${}", "h".repeat(86)),
        format!("$6${}${}", "s".repeat(17), "h".repeat(86)),
        format!("$6${}${}", "s".repeat(8), "h".repeat(85)),
        format!("$6${}${}", "s".repeat(8), "h".repeat(87)),
        format!("$5${}${}", "s".repeat(8), "h".repeat(86)),
        format!("$6${}$extra${}", "s".repeat(8), "h".repeat(86)),
        format!("$6${}${}", "s!".repeat(4), "h".repeat(86)),
        "$6$salt$".to_string(),
        String::new(),
    ] {
        assert!(!valid_password_hash(&bad), "accept {bad:?}");
    }
}

const TEMPLATE: &str =
    r#"{"ignition":{"version":"3.5.0"},"storage":{"files":[{"path":"/etc/keep","mode":420}]}}"#;
const KEY: &str =
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV";

fn hash() -> String {
    format!("$6${}${}", "s".repeat(8), "h".repeat(86))
}

#[test]
fn destination_assembles_config() {
    let out = destination(TEMPLATE.as_bytes(), "soda-01", KEY, &hash(), "10.89.0.0/24").unwrap();
    let text = String::from_utf8(out).unwrap();
    let parsed = jsongo::parse(text.as_bytes()).unwrap();
    let config = Soft::new(&parsed).unwrap();
    let storage = config.object("storage").unwrap().unwrap();
    let files = storage.array("files").unwrap().unwrap();
    assert_eq!(files.len(), 3);
    let passwd = config.object("passwd").unwrap().unwrap();
    let users = passwd.array("users").unwrap().unwrap();
    assert_eq!(users.len(), 1);
    // Deterministic key order and exact byte shape.
    assert!(
        text.starts_with(r#"{"ignition":{"version":"3.5.0"},"passwd":{"users":[{"name":"root""#)
    );
    assert!(text.contains(r#""mode":384"#));
    assert!(text.contains("data:;base64,"));
    // Hostname file decodes to the name plus newline.
    let host_b64 = crate::sshkey::b64_encode(b"soda-01\n");
    assert!(text.contains(&format!("data:;base64,{host_b64}")));
}

#[test]
fn destination_without_key_omits_authorized_keys() {
    let out = destination(TEMPLATE.as_bytes(), "soda-01", "", &hash(), "10.89.0.0/24").unwrap();
    let text = String::from_utf8(out).unwrap();
    assert!(!text.contains("sshAuthorizedKeys"));
    assert!(text.contains("passwordHash"));
}

#[test]
fn destination_rejects_bad_inputs() {
    assert_eq!(
        destination(
            TEMPLATE.as_bytes(),
            "BAD NAME",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "invalid private provisioning inputs"
    );
    assert_eq!(
        destination(
            TEMPLATE.as_bytes(),
            "soda-01",
            KEY,
            "not-a-hash",
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "invalid private provisioning inputs"
    );
    assert_eq!(
        destination(TEMPLATE.as_bytes(), "soda-01", KEY, &hash(), "10.89.0.1/24")
            .unwrap_err()
            .to_string(),
        "invalid private provisioning inputs"
    );
    assert_eq!(
        destination(
            TEMPLATE.as_bytes(),
            "soda-01",
            "bogus-key",
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "valid SSH public key without authorized_keys options required"
    );
    assert_eq!(
        destination(b"not json", "soda-01", KEY, &hash(), "10.89.0.0/24")
            .unwrap_err()
            .to_string(),
        "invalid public destination template"
    );
    assert_eq!(
        destination(
            br#"{"ignition":{"version":"3.4.0"},"storage":{}}"#,
            "soda-01",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "expected converted Ignition 3.5.0 template"
    );
    assert_eq!(
        destination(
            br#"{"ignition":{"version":"3.5.0"},"storage":{},"passwd":{}}"#,
            "soda-01",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "public template must not contain accounts"
    );
    assert_eq!(
        destination(
            br#"{"ignition":{"version":"3.5.0"}}"#,
            "soda-01",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "invalid public storage template"
    );
    assert_eq!(
        destination(
            br#"{"ignition":{"version":"3.5.0"},"storage":{}}"#,
            "soda-01",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "invalid public files template"
    );
    assert_eq!(
        destination(
            br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[{"path":"/etc/hostname"}]}}"#,
            "soda-01",
            KEY,
            &hash(),
            "10.89.0.0/24"
        )
        .unwrap_err()
        .to_string(),
        "provisioning path collision"
    );
}
