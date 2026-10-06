use super::*;

#[test]
fn flags_match_argparse_long_forms() {
    assert_eq!(
        split_flag("--arch=x86_64"),
        Some(("arch".to_string(), Some("x86_64".to_string())))
    );
    assert_eq!(split_flag("--out"), Some(("out".to_string(), None)));
    assert_eq!(split_flag("--"), None);
    assert_eq!(split_flag("-arch"), None);
    assert_eq!(split_flag("positional"), None);
    assert_eq!(split_flag("--=x"), None);
    assert_eq!(split_flag("--"), None);
}

#[test]
fn sha256_matches_hashlib() {
    assert_eq!(
        sha256_hex(b"abc"),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
}
