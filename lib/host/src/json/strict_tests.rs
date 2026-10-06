use super::*;

#[test]
fn strict_accepts_one_known_object() {
    let v = decode_strict(br#"{"id":"one"}"#).unwrap();
    assert_eq!(
        v.as_object().unwrap(),
        &vec![("id".to_string(), Value::Str("one".to_string()))]
    );
}

#[test]
fn strict_rejects_bad_shapes() {
    for input in [
        "{\"id\":\"one\",\"id\":\"two\"}",
        "[]",
        "{\"id\":\"one\"}{\"id\":\"two\"}",
        "{\"id\":\"one\"",
        "null",
        "\"str\"",
        "42",
        "{\"id\":}",
        "{,}",
    ] {
        assert!(decode_strict(input.as_bytes()).is_err(), "{input}");
    }
    // Trailing whitespace is fine.
    assert!(decode_strict(b"{\"id\":\"one\"}\n").is_ok());
}

#[test]
fn strict_rejects_nested_duplicates() {
    for input in [
        "{\"id\":\"one\",\"meta\":{\"key\":\"1\",\"key\":\"2\"}}",
        "{\"id\":\"one\",\"meta\":{\"outer\":{\"key\":\"1\",\"key\":\"2\"}}}",
        "{\"id\":\"one\",\"items\":[{\"key\":\"1\"},{\"key\":\"1\",\"key\":\"2\"}]}",
    ] {
        let e = decode_strict(input.as_bytes()).unwrap_err();
        assert!(e.0.contains("duplicate"), "{input}: {e}");
    }
    assert!(decode_strict(br#"{"id":"one","meta":{"first":"1","second":"2"}}"#).is_ok());
}

#[test]
fn strict_rejects_float64_overflowing_numbers() {
    // Go converts every scanned number to float64: overflow fails the
    // gate with the top-level field name, at any nesting depth.
    for (input, field, lit) in [
        (r#"{"z":1e999}"#, "z", "1e999"),
        (r#"{"a":{"b":-2e999}}"#, "a", "-2e999"),
        (r#"{"a":[1.8e308]}"#, "a", "1.8e308"),
        (r#"{"m":1,"z":1e309,"a":1}"#, "z", "1e309"),
        (r#"{"z":1e999,"y":2e999}"#, "z", "1e999"),
        (
            r#"{"z":1.7976931348623159e308}"#,
            "z",
            "1.7976931348623159e308",
        ),
    ] {
        let e = decode_strict(input.as_bytes()).unwrap_err();
        assert_eq!(
            e.0,
            format!(
                "decode request field \"{field}\": json: cannot unmarshal number {lit} into Go value of type float64"
            ),
            "{input}"
        );
    }
    // Underflow to zero and the largest finite double stay accepted.
    for input in [
        r#"{"z":1e-999}"#,
        r#"{"z":-1e-999}"#,
        r#"{"z":4e-324}"#,
        r#"{"z":0e999}"#,
        r#"{"z":1.7976931348623157e308}"#,
        r#"{"z":1e308}"#,
        r#"{"z":42}"#,
    ] {
        assert!(decode_strict(input.as_bytes()).is_ok(), "{input}");
    }
    // Walk order: a duplicate earlier in the same value beats the
    // float, but a float in an earlier field beats a later duplicate.
    let e = decode_strict(br#"{"a":{"x":1,"x":2,"y":1e999}}"#).unwrap_err();
    assert!(e.0.contains("duplicate"), "{e}");
    let e = decode_strict(br#"{"a":{"x":1,"x":2},"b":1e999}"#).unwrap_err();
    assert!(e.0.contains("duplicate"), "{e}");
    let e = decode_strict(br#"{"b":1e999,"a":{"x":1,"x":2}}"#).unwrap_err();
    assert!(e.0.contains("cannot unmarshal number 1e999"), "{e}");
    // Root and trailing `Token`s convert too: bare overflow errors.
    for (input, want) in [
        (
            "1e999",
            "decode request: json: cannot unmarshal number 1e999 into Go value of type float64",
        ),
        (
            "{} 1e999",
            "decode request: json: cannot unmarshal number 1e999 into Go value of type float64",
        ),
        (
            r#"{"a":1} -2e999"#,
            "decode request: json: cannot unmarshal number -2e999 into Go value of type float64",
        ),
        ("42", "request must be one JSON object"),
        ("{} 42", "request must contain exactly one JSON object"),
    ] {
        assert_eq!(
            decode_strict(input.as_bytes()).unwrap_err().0,
            want,
            "{input}"
        );
    }
    // Tolerant decode keeps accepting: machine parsers skip values.
    assert!(decode_tolerant(br#"{"z":1e999}"#.as_ref()).is_ok());
}

#[test]
fn strict_rejects_invalid_utf8_and_oversized() {
    let mut bad = b"{\"id\":\"".to_vec();
    bad.push(0xff);
    bad.extend_from_slice(b"\"}");
    assert!(decode_strict(&bad).unwrap_err().0.contains("UTF-8"));
    let big = format!("{{\"id\":{:?}}}", "a".repeat(MAXIMUM_REQUEST_BYTES));
    assert!(decode_strict(big.as_bytes())
        .unwrap_err()
        .0
        .contains("1 MiB"));
}

#[test]
fn strict_scanner_messages_match_go_toolchain() {
    // Exact `strictjson.Decode` message shapes, pinned against the
    // pinned Go toolchain (see the corpus4 differential).
    for (input, want) in [
        ("", "decode request: EOF"),
        ("  ", "decode request: EOF"),
        ("[", "request must be one JSON object"),
        ("[1]", "request must be one JSON object"),
        ("1", "request must be one JSON object"),
        ("\"a", "decode request: unexpected EOF"),
        ("\"a\"", "request must be one JSON object"),
        ("tru", "decode request: unexpected EOF"),
        ("true", "request must be one JSON object"),
        ("-", "decode request: unexpected EOF"),
        ("-x", "decode request: invalid character 'x' in numeric literal"),
        ("x", "decode request: invalid character 'x' looking for beginning of value"),
        (",", "decode request: invalid character ',' looking for beginning of value"),
        ("]", "decode request: invalid character ']' looking for beginning of value"),
        ("{", "decode request: EOF"),
        ("{]", "decode request: invalid character ']'"),
        ("{1", "decode request: invalid character '1'"),
        ("{,", "decode request: invalid character ','"),
        ("{\"a\":1,}", "decode request: invalid character '}' looking for beginning of object key string"),
        ("{\"a\":1,2", "decode request: invalid character '2' looking for beginning of object key string"),
        ("{\"a\":1,", "decode request: EOF"),
        ("{\"a\"", "decode request field \"a\": EOF"),
        ("{\"a\" ", "decode request field \"a\": EOF"),
        ("{\"a\" 1}", "decode request field \"a\": expected colon after object key"),
        ("{\"a\":", "decode request field \"a\": EOF"),
        ("{\"a\":x}", "decode request field \"a\": invalid character 'x' looking for beginning of value"),
        ("{\"a\":tru}", "decode request field \"a\": invalid character '}' in literal true (expecting 'e')"),
        ("{\"a\":1", "decode request: EOF"),
        ("{\"a\":1 ", "decode request: EOF"),
        ("{\"a\":1]", "decode request: invalid character ']' after object key:value pair"),
        ("{\"a\":1x}", "decode request: invalid character 'x' after object key:value pair"),
        ("{\"a\":01}", "decode request: invalid character '1' after object key:value pair"),
        ("{\"a\":1.5e+}", "decode request field \"a\": invalid character '}' in exponent of numeric literal"),
        ("{\"a\":1.5e+", "decode request field \"a\": unexpected EOF"),
        ("{\"a\":1.5ex}", "decode request field \"a\": invalid character 'x' in exponent of numeric literal"),
        ("{\"a\":1.x}", "decode request field \"a\": invalid character 'x' after decimal point in numeric literal"),
        ("{\"a\":\"b\\x}", "decode request field \"a\": invalid character 'x' in string escape code"),
        ("{\"a\":\"b\\u12}", "decode request field \"a\": invalid character '}' in \\u hexadecimal character escape"),
        ("{\"a\":1}{", "request must contain exactly one JSON object"),
        ("{\"a\":1}[", "request must contain exactly one JSON object"),
        ("{\"a\":1}1", "request must contain exactly one JSON object"),
        ("{\"a\":1}1x", "request must contain exactly one JSON object"),
        ("{\"a\":1}}", "decode request: invalid character '}' looking for beginning of value"),
        ("{\"a\":1},", "decode request: invalid character ',' looking for beginning of value"),
        ("{\"a\":1}\"a", "decode request: unexpected EOF"),
        ("{\"a\":1}tru", "decode request: unexpected EOF"),
        ("{\"a\":1}-x", "decode request: invalid character 'x' in numeric literal"),
        ("{\"f\":{\"g\":1", "decode request field \"f\": unexpected EOF"),
        ("{\"f\":{\"g\":1,", "decode request field \"f\": unexpected EOF"),
        ("{\"f\":{\"g\"", "decode request field \"f\": unexpected EOF"),
        ("{\"f\":{\"g\" 1}", "decode request field \"f\": invalid character '1' after object key"),
        ("{\"f\":[1 2]}", "decode request field \"f\": invalid character '2' after array element"),
        ("{\"f\":[01]}", "decode request field \"f\": invalid character '1' after array element"),
        ("{\"f\":[x]}", "decode request field \"f\": invalid character 'x' looking for beginning of value"),
        ("{\"f\":{1}}", "decode request field \"f\": invalid character '1' looking for beginning of object key string"),
        ("{\"a\":1,\"a\":2}", "duplicate request field \"a\""),
        ("{\"a\":{\"x\":1,\"x\":2}}", "duplicate request field \"x\""),
        // A scan error anywhere in the value beats a deferred nested dup.
        ("{\"a\":{\"x\":1,\"x\":truX}}", "decode request field \"a\": invalid character 'X' in literal true (expecting 'e')"),
        // A top duplicate reports before the later value is scanned.
        ("{\"a\":1,\"a\":{\"x\":truX}}", "duplicate request field \"a\""),
    ] {
        let got = decode_strict(input.as_bytes()).unwrap_err().0;
        assert_eq!(got, want, "input={input:?}");
    }
}

#[test]
fn strict_depth_bounds_match_reject_duplicate_keys() {
    // strictjson rejects values nested past depth 100 (102 opens) once
    // the top-level value completes; the scanner itself fails past 10000.
    for (n, ok) in [(101, true), (102, false), (103, false)] {
        let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
        assert_eq!(
            decode_strict(doc.as_bytes()).is_ok(),
            ok,
            "empty arrays n={n}"
        );
    }
    // A scalar counts one deeper than its containers.
    for (n, ok) in [(100, true), (101, false), (102, false)] {
        let doc = format!("{{\"n\":{}1{}}}", "[".repeat(n), "]".repeat(n));
        assert_eq!(
            decode_strict(doc.as_bytes()).is_ok(),
            ok,
            "scalar arrays n={n}"
        );
        let doc = format!("{{\"n\":{}}}", "{\"a\":".repeat(n) + "1" + &"}".repeat(n));
        assert_eq!(decode_strict(doc.as_bytes()).is_ok(), ok, "objects n={n}");
    }
    let doc = format!("{{\"n\":{}}}", "[".repeat(102) + &"]".repeat(102));
    assert_eq!(
        decode_strict(doc.as_bytes()).unwrap_err().0,
        "decode request field \"n\": request is nested too deeply"
    );
    for (n, want) in [
        (
            10000,
            "decode request field \"n\": request is nested too deeply",
        ),
        (
            10001,
            "decode request field \"n\": invalid character '[' exceeded max depth",
        ),
    ] {
        let doc = format!("{{\"n\":{}}}", "[".repeat(n) + &"]".repeat(n));
        assert_eq!(decode_strict(doc.as_bytes()).unwrap_err().0, want, "n={n}");
    }
}
