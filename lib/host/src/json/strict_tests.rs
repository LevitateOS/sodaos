use super::*;
use std::collections::BTreeMap;

type AnyObject = BTreeMap<String, serde_json::Value>;

#[test]
fn strict_admission_rejects_decoded_duplicates_even_in_ignored_values() {
    assert!(decode_strict_as::<AnyObject>(
        br#"{"ignored":[{"name":"first","\u006eame":"second"}]}"#
    )
    .is_err());
    assert!(decode_strict_as::<AnyObject>(br#"{"x":1,"\u0078":2}"#).is_err());
    assert!(decode_strict_as::<AnyObject>(br#"{"x":1} {}"#).is_err());
    assert!(decode_strict_as::<AnyObject>(b"[]").is_err());
}

#[test]
fn strict_depth_matches_accepted_empty_container_boundary_and_scalar_leaf() {
    for (depth, accepted) in [(101, true), (102, false)] {
        let doc = format!("{{\"v\":{}}}", "[".repeat(depth) + &"]".repeat(depth));
        assert_eq!(
            decode_strict_as::<AnyObject>(doc.as_bytes()).is_ok(),
            accepted
        );
    }
    for (depth, accepted) in [(100, true), (101, false)] {
        let doc = format!("{{\"v\":{}0{}}}", "[".repeat(depth), "]".repeat(depth));
        assert_eq!(
            decode_strict_as::<AnyObject>(doc.as_bytes()).is_ok(),
            accepted
        );
    }
}

#[test]
fn strict_typed_path_preserves_sorted_root_and_nested_source_alias_order() {
    let request = crate::muse::LaunchRequest::decode(
        br#"{"cwd":"/lower","CWD":"/upper","register":{"child_id":"first","CHILD_ID":null,"Child_Id":"last"}}"#,
    ).unwrap();
    assert_eq!(request.cwd, "/lower");
    assert_eq!(request.register.unwrap().child_id, "last");
}

#[test]
fn strict_caps_utf8_and_go_string_emission() {
    let mut invalid = b"{\"v\":\"".to_vec();
    invalid.push(0xff);
    invalid.extend_from_slice(b"\"}");
    assert!(decode_strict_as::<AnyObject>(&invalid).is_err());
    let oversized = format!("{{\"v\":{:?}}}", "x".repeat(MAXIMUM_REQUEST_BYTES));
    assert!(decode_strict_as::<AnyObject>(oversized.as_bytes()).is_err());
    assert_eq!(
        quote("a<b>&\"c\"\n\t\x01\u{2028}é"),
        "\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\t\\u0001\\u2028é\""
    );
}

#[test]
fn signed_and_unsigned_number_adapters_keep_distinct_minus_zero_rules() {
    assert_eq!(serde_json::from_str::<SignedInteger>("-0").unwrap().0, 0);
    for input in ["1.0", "1e0", "9223372036854775808", "-9223372036854775809"] {
        assert!(
            serde_json::from_str::<SignedInteger>(input).is_err(),
            "{input}"
        );
    }
    assert!(serde_json::from_str::<u32>("-0").is_err());
    assert_eq!(
        serde_json::from_str::<BytesField>("[null,0,255]")
            .unwrap()
            .0,
        [0, 0, 255]
    );
    assert!(serde_json::from_str::<BytesField>("[-0]").is_err());
}
