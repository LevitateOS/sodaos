use super::*;

const WIDGET_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "count",
        kind: Kind::I64,
    },
    Spec {
        name: "flag",
        kind: Kind::Bool,
    },
];

fn bind_widget(raw: &[u8]) -> Result<BoundMap, Error> {
    let v = decode_strict(raw)?;
    bind_root(&v, "Widget", WIDGET_SPECS, false)
}

#[test]
fn binder_matches_go_field_rules() {
    let m = bind_widget(br#"{"ID":"x","Count":3,"Flag":true}"#).unwrap();
    assert_eq!(m.take_string("id"), "x"); // case-insensitive fallback
    assert_eq!(m.take_i64("count"), 3);
    assert!(m.take_bool("flag"));

    // Unknown field, Go-quoted.
    let err = bind_widget(br#"{"id":"x","bogus":1}"#).unwrap_err();
    assert_eq!(err.0, "decode request: json: unknown field \"bogus\"");

    // Null leaves the zero value.
    let m = bind_widget(br#"{"id":null,"count":null}"#).unwrap();
    assert_eq!(m.take_string("id"), "");
    assert_eq!(m.take_i64("count"), 0);

    // Exact Go mismatch messages, literal echoed for int targets only.
    for (bad, want) in [
        (
            br#"{"count":1.5}"#.as_slice(),
            "decode request: json: cannot unmarshal number 1.5 into Go struct field Widget.count of type int64",
        ),
        (
            br#"{"count":"3"}"#,
            "decode request: json: cannot unmarshal string into Go struct field Widget.count of type int64",
        ),
        (
            br#"{"count":true}"#,
            "decode request: json: cannot unmarshal bool into Go struct field Widget.count of type int64",
        ),
        (
            br#"{"count":1e3}"#,
            "decode request: json: cannot unmarshal number 1e3 into Go struct field Widget.count of type int64",
        ),
        (
            br#"{"id":7}"#,
            "decode request: json: cannot unmarshal number into Go struct field Widget.id of type string",
        ),
    ] {
        let err = bind_widget(bad).unwrap_err();
        assert_eq!(err.0, want);
    }

    // Fold-matching keys all bind, last in sorted order wins; no error.
    let m = bind_widget(br#"{"ID":"a","id":"b"}"#).unwrap();
    assert_eq!(m.take_string("id"), "b");

    // Sorted top-level order decides between two faults.
    let err = bind_widget(br#"{"id":1,"count":"x"}"#).unwrap_err();
    assert!(err.0.contains("Widget.count"), "{err:?}");
}

#[test]
fn surrogate_pairs_match_go_substitution() {
    // Lone surrogates become U+FFFD, like encoding/json.
    let v = decode_strict(b"{\"a\":\"\\ud800\"}").unwrap();
    assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "\u{FFFD}");
    let v = decode_strict(b"{\"a\":\"\\udc00\"}").unwrap();
    assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "\u{FFFD}");
    // Valid pair decodes to the astral character.
    let v = decode_strict(b"{\"a\":\"\\ud800\\udc00\"}").unwrap();
    assert_eq!(
        tolerant_get(&v, "a").unwrap().as_str().unwrap(),
        "\u{10000}"
    );
    // Bad hex in an escape is a hard error.
    assert!(decode_strict(b"{\"a\":\"\\ud80x\"}").is_err());
    // Raw C0 controls inside strings are rejected.
    assert!(decode_strict(b"{\"a\":\"a\x01b\"}").is_err());
}

#[test]
fn go_string_escaping_matches() {
    assert_eq!(
        quote("a<b>&\"c\"\n\t\x01\u{2028}é"),
        "\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\t\\u0001\\u2028é\""
    );
}

#[test]
fn tolerant_decode_keeps_last_duplicate() {
    let v = decode_tolerant(br#"{"a":"1","a":"2"}"#).unwrap();
    assert_eq!(tolerant_get(&v, "a").unwrap().as_str().unwrap(), "2");
    let v = decode_tolerant(br#"[{"a":1}]"#).unwrap();
    assert!(v.as_array().is_some());
}

#[test]
fn tolerant_decode_matches_go_nesting_limit() {
    // encoding/json fails opening the 10001st container; scalars at
    // depth are fine. Pinned against the pinned Go toolchain.
    for (n, ok) in [(9999, true), (10000, false)] {
        let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
        assert_eq!(decode_tolerant(doc.as_bytes()).is_ok(), ok, "n={n}");
        let doc = format!("{{\"n\":{}1{}}}", "[".repeat(n), "]".repeat(n));
        assert_eq!(decode_tolerant(doc.as_bytes()).is_ok(), ok, "scalar n={n}");
    }
    // Strict requests keep strictjson's much smaller depth-100 scan.
    let doc = format!("{{\"n\":{}}}", "[".repeat(102) + &"]".repeat(102));
    assert!(decode_strict(doc.as_bytes()).is_err());
}
