use super::*;

#[test]
fn base64_round_trip_matches_go_vectors() {
    assert_eq!(base64_encode(b""), "");
    assert_eq!(base64_encode(b"f"), "Zg==");
    assert_eq!(base64_encode(b"fo"), "Zm8=");
    assert_eq!(base64_encode(b"foo"), "Zm9v");
    assert_eq!(base64_encode(b"foob"), "Zm9vYg==");
    assert_eq!(base64_encode(b"fooba"), "Zm9vYmE=");
    assert_eq!(base64_encode(b"foobar"), "Zm9vYmFy");
    for raw in [b"".as_slice(), b"f", b"fo", b"foo", b"record-bytes-12345"] {
        assert_eq!(base64_decode(&base64_encode(raw)).unwrap(), raw);
    }
    assert!(base64_decode("Zg=").is_err());
    assert!(base64_decode("Zg*=").is_err());
    assert!(base64_decode("Zm9v\n").is_err());
    // Non-canonical trailing bits rejected like Go's decoder.
    assert!(base64_decode("Zh==").is_err());
}

#[test]
fn strict_decode_rejects_duplicates_and_unknowns() {
    let value = parse_strict(br#"{"A": 1, "B": [1, 2]}"#).unwrap();
    let mut binder = Binder::new(&value).unwrap();
    assert_eq!(binder.integer("A").unwrap(), Some(1));
    assert_eq!(binder.array("B").unwrap().unwrap().len(), 2);
    assert!(binder.finish().is_ok());
    assert!(parse_strict(br#"{"A": 1, "A": 2}"#).is_err());
    assert!(parse_strict(br#"{"A": {"B": 1, "B": 2}}"#).is_err());
    assert!(parse_strict(br#"[1, 2]"#).is_err());
    let value = parse_strict(br#"{"A": 1, "Extra": true}"#).unwrap();
    let mut binder = Binder::new(&value).unwrap();
    assert_eq!(binder.integer("A").unwrap(), Some(1));
    assert!(binder.finish().is_err());
    // Go type rules: null means absent, wrong types refuse.
    let value = parse_strict(br#"{"A": null, "B": "x"}"#).unwrap();
    let mut binder = Binder::new(&value).unwrap();
    assert_eq!(binder.integer("A").unwrap(), None);
    assert_eq!(binder.string("B").unwrap(), Some("x".to_string()));
    assert!(binder.finish().is_ok());
    let value = parse_strict(br#"{"A": "1"}"#).unwrap();
    let mut binder = Binder::new(&value).unwrap();
    assert!(binder.integer("A").is_err());
    let value = parse_strict(br#"{"A": 1.5}"#).unwrap();
    let mut binder = Binder::new(&value).unwrap();
    assert!(binder.integer("A").is_err());
}
