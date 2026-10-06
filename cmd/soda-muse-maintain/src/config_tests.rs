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
    assert_eq!(
        load_config(&bad).unwrap_err(),
        "invalid character 'o' looking for beginning of object key string"
    );
}

#[test]
fn host_config_decode_vectors() {
    assert_eq!(decode_host_config(b"").unwrap_err(), "EOF");
    assert_eq!(decode_host_config(b"   ").unwrap_err(), "EOF");
    let c = decode_host_config(b"null").unwrap();
    assert_eq!(c.image, "");
    assert_eq!(
        decode_host_config(b"[1,2]").unwrap_err(),
        "json: cannot unmarshal array into Go value of type host.Config"
    );
    assert_eq!(
        decode_host_config(b"\"str\"").unwrap_err(),
        "json: cannot unmarshal string into Go value of type host.Config"
    );
    assert_eq!(decode_host_config(b"[1,2").unwrap_err(), "unexpected EOF");
    // Last duplicate wins; keys match case-insensitively.
    let c = decode_host_config(br#"{"image": "a", "image": "b", "NETWORK": "n"}"#).unwrap();
    assert_eq!(c.image, "b");
    assert_eq!(c.network, "n");
    let c = decode_host_config(br#"{"tailnet_management": true}"#).unwrap();
    assert!(c.tailnet_management);
    let c = decode_host_config(br#"{"image": null, "tailnet_management": null}"#).unwrap();
    assert_eq!(c.image, "");
    assert!(!c.tailnet_management);
    // Type errors name the key as written.
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": 1}"#).unwrap_err(),
        "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
    );
    assert_eq!(
        decode_host_config(br#"{"MUSE_SHA256": 1}"#).unwrap_err(),
        "json: cannot unmarshal number into Go struct field Config.MUSE_SHA256 of type string"
    );
    assert_eq!(
        decode_host_config(br#"{"tailnet_management": "yes"}"#).unwrap_err(),
        "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": [1,2]}"#).unwrap_err(),
        "json: cannot unmarshal array into Go struct field Config.muse_sha256 of type string"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": {"a":1}}"#).unwrap_err(),
        "json: cannot unmarshal object into Go struct field Config.muse_sha256 of type string"
    );
    // Unknown fields, including null-valued ones.
    assert_eq!(
        decode_host_config(br#"{"bogus": null}"#).unwrap_err(),
        "json: unknown field \"bogus\""
    );
    assert_eq!(
        decode_host_config(br#"{"quo\"te": 1}"#).unwrap_err(),
        "json: unknown field \"quo\\\"te\""
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": "x", "bogus": 1}"#).unwrap_err(),
        "json: unknown field \"bogus\""
    );
    // Broken values beat unknown fields; a later syntax error beats a
    // saved error; the first saved error beats later saved errors.
    assert_eq!(
        decode_host_config(br#"{"bogus": truX}"#).unwrap_err(),
        "invalid character 'X' in literal true (expecting 'e')"
    );
    assert_eq!(
        decode_host_config(br#"{"bogus": [1,2}"#).unwrap_err(),
        "invalid character '}' after array element"
    );
    assert_eq!(
        decode_host_config(br#"{"bogus": 1, "tailnet_management": truX}"#).unwrap_err(),
        "invalid character 'X' in literal true (expecting 'e')"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": 1, "tailnet_management": truX}"#).unwrap_err(),
        "invalid character 'X' in literal true (expecting 'e')"
    );
    assert_eq!(
        decode_host_config(br#"{"tailnet_management": "x", "muse_sha256": 1}"#).unwrap_err(),
        "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": 1, "bogus": 2}"#).unwrap_err(),
        "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
    );
    assert_eq!(
        decode_host_config(br#"{"bogus": 2, "muse_sha256": 1}"#).unwrap_err(),
        "json: unknown field \"bogus\""
    );
    // Escapes and structural errors, byte-identical to encoding/json.
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": "a\qb"}"#).unwrap_err(),
        "invalid escape sequence `\\q` in string"
    );
    assert_eq!(
        decode_host_config(b"{\"muse_sha256\": \"a\x01b\"}").unwrap_err(),
        "invalid character '\\x01' in string"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": "\u00G1"}"#).unwrap_err(),
        "invalid escape sequence `\\u00G1` in string"
    );
    assert_eq!(
        decode_host_config(br#"{"muse_sha256": "\uD800\u00G1"}"#).unwrap_err(),
        "invalid escape sequence `\\u00G1` in string"
    );
    let c = decode_host_config("{\"image\": \"a�b\"}".as_bytes()).unwrap();
    assert_eq!(c.image, "a�b");
    let c = decode_host_config(br#"{"image": "a\uD800b"}"#).unwrap();
    assert_eq!(c.image, "a\u{FFFD}b");
    let c = decode_host_config(br#"{"image": "a\uD800\u0041"}"#).unwrap();
    assert_eq!(c.image, "a\u{FFFD}A");
    assert_eq!(
        decode_host_config(b"{': 1}").unwrap_err(),
        "invalid character '\\'' looking for beginning of object key string"
    );
    assert_eq!(
        decode_host_config(br#"{"a": "b",}"#).unwrap_err(),
        "invalid character '}' looking for beginning of object key string"
    );
    assert_eq!(
        decode_host_config(b"{\"a\": \"b\"").unwrap_err(),
        "unexpected EOF"
    );
    assert_eq!(decode_host_config(b"nul").unwrap_err(), "unexpected EOF");
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
