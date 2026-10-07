use super::*;

fn sample_profile() -> Profile {
    Profile {
        id: ROCKY_HEADLESS.to_string(),
        distribution: "rocky".to_string(),
        version: "9.7".to_string(),
        interface: "headless".to_string(),
        architecture: "amd64".to_string(),
        image: format!("sha256:{}", "a".repeat(64)),
        revision: "b".repeat(40),
    }
}

#[test]
fn validators_match_go_regexps() {
    assert!(valid_id(&format!("p{}", "a".repeat(24))));
    assert!(!valid_id("p123"));
    assert!(!valid_id(&format!("P{}", "a".repeat(24))));
    assert!(!valid_id(&format!("p{}", "A".repeat(24))));
    assert!(valid_login("soda-tester"));
    assert!(valid_login("root-1")); // exact "root" is rejected at the op layer, not here
    assert!(!valid_login("1abc"));
    assert!(!valid_login("Root"));
    assert!(valid_image_ref(&"c".repeat(64)));
    assert!(valid_image_ref(&format!("sha256:{}", "c".repeat(64))));
    assert!(!valid_image_ref("c"));
    assert!(valid_container_id(&"d".repeat(64)));
    assert!(!valid_container_id("xyz"));
}

#[test]
fn profile_validation_matches() {
    assert!(sample_profile().validate().is_ok());
    let mut p = sample_profile();
    p.architecture = "x86_64".to_string();
    assert!(p.validate().is_err());
    let mut p = sample_profile();
    p.version = "9.7.1.2".to_string();
    assert!(p.validate().is_err());
    let mut p = sample_profile();
    p.version = "9".to_string();
    assert!(p.validate().is_ok());
}

#[test]
fn profile_wire_round_trip() {
    let p = sample_profile();
    let enc = p.encode();
    assert_eq!(decode_profile(&enc).unwrap(), p);
    // Unknown fields rejected, like strictjson.
    assert!(decode_profile(&enc.replace('}', ",\"x\":1}")).is_err());
    // Oversized labels rejected before parsing.
    assert!(decode_profile(&"x".repeat(1025)).is_err());
}

#[test]
fn create_decode_and_validate() {
    let raw = format!(
        "{{\"profile\":{},\"id\":\"p{}\",\"owner\":42}}",
        sample_profile().encode(),
        "e".repeat(24)
    );
    let c = Create::decode(raw.as_bytes()).unwrap();
    assert!(c.validate().is_ok());
    let c = Create::decode(br#"{"id":"pAAAAAAAAAAAAAAAAAAAAAAAA"}"#).unwrap();
    assert!(c.validate().is_err()); // missing profile + bad id
}

#[test]
fn environment_omits_empty_like_go() {
    let env = Environment {
        image: String::new(),
        profile: None,
        id: "x".to_string(),
        ip: String::new(),
        running: false,
    };
    assert_eq!(env.encode(), r#"{"id":"x","ip":"","running":false}"#);
    let env = Environment {
        image: "sha256:1".to_string(),
        profile: Some(sample_profile()),
        id: "x".to_string(),
        ip: "10.0.0.2".to_string(),
        running: true,
    };
    let enc = env.encode();
    assert!(enc.starts_with(r#"{"image":"sha256:1","profile":{"id":"rocky-headless""#));
    assert!(enc.ends_with(r#""ip":"10.0.0.2","running":true}"#));
}

#[test]
fn os_release_validation_matches_go() {
    let good = OsRelease {
        id: "rocky".to_string(),
        version: "9.7".to_string(),
        name: "Rocky Linux 9.7".to_string(),
    };
    assert!(valid_os_release(&good));
    assert!(!valid_os_release(&OsRelease {
        id: "Rocky".to_string(),
        ..good.clone()
    }));
    assert!(!valid_os_release(&OsRelease {
        name: "".to_string(),
        ..good.clone()
    }));
    assert!(!valid_os_release(&OsRelease {
        name: "a\u{200b}b".to_string(),
        ..good.clone()
    })); // Cf ZWSP
    assert!(!valid_os_release(&OsRelease {
        name: "a\u{7f}b".to_string(),
        ..good.clone()
    })); // Cc DEL
    assert!(!valid_os_release(&OsRelease {
        name: "a\u{ad}b".to_string(),
        ..good.clone()
    })); // Cf soft hyphen
    assert!(valid_os_release(&OsRelease {
        name: "Rocky Linux \u{e9}".to_string(),
        ..good.clone()
    }));
}

#[test]
fn account_dto_strict_shape() {
    let a = Account::decode(
        br#"{"project":"p0123456789abcdef01234567","login":"alice","identity":1,"keys":["a","b"]}"#,
    ).unwrap();
    assert_eq!(a.login, "alice");
    assert_eq!(a.identity, 1);
    assert_eq!(a.keys, vec!["a".to_string(), "b".to_string()]);
    // Missing keys decodes as empty (browser-only account).
    assert!(Account::decode(
        br#"{"project":"p0123456789abcdef01234567","login":"alice","identity":1}"#,
    ).unwrap().keys.is_empty());
    // Unknown fields rejected.
    assert!(Account::decode(br#"{"project":"p","login":"a","identity":1,"admin":true}"#).is_err());
    // Non-integer identity rejected.
    assert!(Account::decode(br#"{"project":"p","login":"a","identity":1.5}"#).is_err());
}

#[test]
fn access_keys_dto_strict_shape() {
    let a = AccessKeys::decode(
        br#"{"project":"p","login":"a","identity":7,"revision":"r","keys":["k"],"apply":true}"#,
    ).unwrap();
    assert_eq!(a.identity, 7);
    assert!(a.apply);
    assert_eq!(a.keys, vec!["k".to_string()]);
    // Duplicates rejected at the strict layer.
    assert!(Account::decode(br#"{"project":"p","project":"q"}"#).is_err());
}

#[test]
fn access_key_state_encodes_struct_order() {
    let s = AccessKeyState {
        revision: "r".to_string(),
        keys: vec!["a\"b".to_string()],
    };
    assert_eq!(s.encode(), "{\"revision\":\"r\",\"keys\":[\"a\\\"b\"]}");
}
