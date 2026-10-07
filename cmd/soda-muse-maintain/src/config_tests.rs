use crate::config_validation::validate_runtime_config;
use crate::config_wire::{decode_host_config, go_quoted};
use crate::test_support::{hex_string, TestDir};
use std::fs;

use super::{load_config, Config};

#[test]
fn go_quoted_vectors() {
    assert_eq!(go_quoted("plain"), "\"plain\"");
    assert_eq!(
        go_quoted("a\x01b\x7f\u{80}é\"\\"),
        "\"a\\x01b\\x7f\\u0080é\\\"\\\\\""
    );
    assert_eq!(go_quoted("\u{a0}"), "\"\\u00a0\"");
    assert_eq!(
        go_quoted("\u{200b}\u{2028}\u{e000}\u{f0000}"),
        "\"\\u200b\\u2028\\ue000\\U000f0000\""
    );
    assert_eq!(go_quoted("\n\r\t"), "\"\\n\\r\\t\"");
    assert_eq!(go_quoted("\u{7}\u{8}\u{c}\u{b}"), "\"\\a\\b\\f\\v\"");
    assert_eq!(go_quoted("space kept"), "\"space kept\"");
}

#[test]
fn config_read_error_shapes() {
    let dir = TestDir::make("cfgread");
    let missing = dir.path("nope.json");
    assert_eq!(
        load_config(&missing).unwrap_err(),
        format!("open {missing}: no such file or directory")
    );
    let sub = dir.path("sub");
    fs::create_dir(&sub).unwrap();
    assert_eq!(
        load_config(&sub).unwrap_err(),
        format!("read {sub}: is a directory")
    );
    // Decode runs before the release lookup.
    let bad = dir.path("bad.json");
    fs::write(&bad, b"{oops").unwrap();
    assert!(load_config(&bad).is_err());
}

#[test]
fn host_config_decode_vectors() {
    let c = decode_host_config(b"null").unwrap();
    assert_eq!(c.image, "");
    // Last duplicate wins; keys match case-insensitively.
    let c = decode_host_config(br#"{"image": "a", "image": "b", "NETWORK": "n"}"#).unwrap();
    assert_eq!(c.image, "b");
    assert_eq!(c.network, "n");
    let c = decode_host_config(br#"{"IMAGE":"first","image":"second"} trailing"#).unwrap();
    assert_eq!(c.image, "second");
    let c = decode_host_config(br#"{"image":"first","IMAGE":"second"}"#).unwrap();
    assert_eq!(c.image, "second");
    let c = decode_host_config(br#"{"tailnet_management": true}"#).unwrap();
    assert!(c.tailnet_management);
    let c = decode_host_config(br#"{"image": null, "tailnet_management": null}"#).unwrap();
    assert_eq!(c.image, "");
    assert!(!c.tailnet_management);
    assert!(decode_host_config(br#"{"bogus": null}"#).is_err());
    assert!(decode_host_config(br#"{"muse_sha256": 1}"#).is_err());
    assert!(decode_host_config(br#"{"tailnet_management": "yes"}"#).is_err());
    assert!(decode_host_config(br#"{"image":"a\uD800"}"#).is_err());
    assert_eq!(
        decode_host_config(br#"{"image":"x"} {"image":"y"}"#)
            .unwrap()
            .image,
        "x"
    );
}

#[test]
fn runtime_config_validation_order() {
    // Muse errors precede identity, subnet, tailnet, and network errors.
    let c = Config {
        muse_sha256: hex_string('a', 64),
        subnet: String::from("bogus"),
        ..Default::default()
    };
    assert_eq!(
        validate_runtime_config(&c).unwrap_err(),
        "explicit muse socket, broker socket and release digest required"
    );
    // Subnet errors precede tailnet and network errors.
    let c = Config {
        subnet: String::from("bogus"),
        ..Default::default()
    };
    assert!(validate_runtime_config(&c)
        .unwrap_err()
        .starts_with("netip.ParsePrefix"));
    // Tailnet errors precede network errors.
    let c = Config {
        subnet: String::from("10.0.0.0/24"),
        tailnet_image: String::from("bare-hex"),
        ..Default::default()
    };
    assert_eq!(
        validate_runtime_config(&c).unwrap_err(),
        "invalid immutable Tailnet companion configuration"
    );
    // Network errors come last.
    let c = Config {
        subnet: String::from("10.0.0.0/24"),
        ..Default::default()
    };
    assert_eq!(
        validate_runtime_config(&c).unwrap_err(),
        "invalid native runtime configuration"
    );
    // A fully valid config passes.
    let c = Config {
        image: String::from("img"),
        network: String::from("net"),
        bridge: String::from("br"),
        subnet: String::from("10.0.0.0/24"),
        ..Default::default()
    };
    validate_runtime_config(&c).unwrap();
}
