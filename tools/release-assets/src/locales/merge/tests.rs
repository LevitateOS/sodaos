use super::*;

#[test]
fn locale_lock_raw_slots_take_the_last_string_and_ignore_large_unknown_numbers() {
    let lock: LocaleLock = serde_json::from_str(
        r#"{"url":1e400,"url":"https://codeberg.org/forgejo/forgejo/raw/tag/v1/locale","sha256":"old","sha256":"final","ignored":{"number":1e400}}"#,
    )
    .expect("raw slots retain grammar-valid ignored values");
    assert_eq!(
        serde_json::from_str::<String>(lock.url.as_ref().unwrap().get()).unwrap(),
        "https://codeberg.org/forgejo/forgejo/raw/tag/v1/locale"
    );
    assert_eq!(
        serde_json::from_str::<String>(lock.sha256.as_ref().unwrap().get()).unwrap(),
        "final"
    );
}

const NATIVE: &str = "[common]\nhome = Home %s\n[settings]\nprofile = Profile\n";
const EXTRA: &str = "[soda]\nnav_personal = Personal\n";

fn sections_of(text: &str) -> Vec<String> {
    parse_ini(text)
        .unwrap()
        .sections()
        .iter()
        .map(ToString::to_string)
        .collect()
}

#[test]
fn strip_set_matches_python_over_all_chars() {
    // Probed from the interpreter: exactly these codepoints.
    let probed: Vec<char> = [
        0x0009u32, 0x000a, 0x000b, 0x000c, 0x000d, 0x001c, 0x001d, 0x001e, 0x001f, 0x0020, 0x0085,
        0x00a0, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008,
        0x2009, 0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000,
    ]
    .into_iter()
    .map(|n| char::from_u32(n).unwrap())
    .collect();
    assert_eq!(probed.len(), 29);
    for n in 0..0x110000u32 {
        let Some(c) = char::from_u32(n) else { continue };
        assert_eq!(
            python_space(c),
            probed.contains(&c),
            "strip mismatch at U+{n:04X}"
        );
    }
}

#[test]
fn merge_is_byte_exact() {
    assert_eq!(merge(NATIVE, EXTRA).unwrap(), format!("{NATIVE}\n{EXTRA}"));
}

#[test]
fn merge_rstrips_python_whitespace() {
    let native = "[common]\nhome = x\n[settings]\ny = z\x1c\x1f  \n";
    let extra = "[soda]\nnav = x\x1e\n";
    assert_eq!(
        merge(native, extra).unwrap(),
        "[common]\nhome = x\n[settings]\ny = z\n\n[soda]\nnav = x\n"
    );
}

#[test]
fn merge_rejects_incomplete_native() {
    let err = merge("[soda]\nx = y\n", EXTRA).unwrap_err();
    assert_eq!(err, "Expected a complete native Forgejo English catalog");
}

#[test]
fn merge_rejects_non_soda_additions() {
    let err = merge(NATIVE, "[settings]\nprofile = Wrong\n").unwrap_err();
    assert_eq!(err, "Additions must use only the Soda namespace");
}

#[test]
fn merge_rejects_native_soda_collision() {
    let native = "[common]\na = b\n[settings]\nc = d\n[soda]\nx = y\n";
    let err = merge(native, EXTRA).unwrap_err();
    assert_eq!(err, "Native catalog already owns the Soda namespace");
}

#[test]
fn merge_rejects_duplicate_addition_keys() {
    let err = merge(NATIVE, "[soda]\nx = one\nx = two\n").unwrap_err();
    assert!(err.starts_with("invalid additions catalog: "), "{err}");
}

#[test]
fn trailing_text_after_section_is_ignored() {
    assert_eq!(sections_of("[soda] whatever\nx = 1\n"), ["soda"]);
}

#[test]
fn section_header_matches_greedily_to_last_bracket() {
    assert_eq!(sections_of("[a]b[c]\nx = 1\n"), ["a]b[c"]);
}

#[test]
fn indented_section_after_value_is_a_continuation() {
    let ini = parse_ini("[a]\nk = v\n  [b]\ny = 2\n").unwrap();
    assert_eq!(ini.sections(), ["a"]);
}

#[test]
fn colon_is_not_a_delimiter() {
    assert!(parse_ini("[a]\nkey : value\n").is_err());
}

#[test]
fn bare_words_are_rejected() {
    assert!(parse_ini("[a]\njustword\n").is_err());
}

#[test]
fn strict_duplicates_are_rejected() {
    assert!(parse_ini("[a]\n[a]\n").is_err());
    assert!(parse_ini("[a]\nx = 1\nx = 2\n").is_err());
    assert!(parse_ini("[a]\nx = 1\n\nx = 2\n").is_err());
}

#[test]
fn option_names_are_case_sensitive() {
    let ini = parse_ini("[a]\nKey = 1\nkey = 2\n").unwrap();
    assert_eq!(ini.options["a"], ["Key", "key"]);
}

#[test]
fn options_before_any_section_are_rejected() {
    assert!(parse_ini("x = 1\n[a]\n").is_err());
    assert!(parse_ini("[unclosed\n").is_err());
}

#[test]
fn comments_and_blank_lines_end_continuations() {
    let ini = parse_ini("[a]\n   # hi\n; there\n\nx = 1\n").unwrap();
    assert_eq!(ini.options["a"], ["x"]);
}

#[test]
fn option_name_stops_at_first_equals_and_rstrips() {
    let ini = parse_ini("[a]\nkey   =   v\nj = a=b\nk =\nmy key = v\n").unwrap();
    assert_eq!(ini.options["a"], ["key", "j", "k", "my key"]);
}

#[test]
fn empty_option_name_is_rejected() {
    assert!(parse_ini("[a]\n= v\n").is_err());
    assert!(parse_ini("[a]\n   = v\n").is_err());
}

#[test]
fn colons_are_allowed_inside_option_names() {
    let ini = parse_ini("[a]\na:b = v\n: = w\n").unwrap();
    assert_eq!(ini.options["a"], ["a:b", ":"]);
}

#[test]
fn default_section_stays_out_of_sections() {
    let ini = parse_ini("[DEFAULT]\nx = 1\n[a]\ny = 2\n").unwrap();
    assert_eq!(ini.sections(), ["a"]);
    assert!(!ini.has_section("DEFAULT"));
}

#[test]
fn continuation_ignores_indent_changes_until_blank() {
    let ini = parse_ini("[a]\nk = v\n    deep\n  y = 2\n").unwrap();
    assert_eq!(ini.options["a"], ["k"]);
    let ini = parse_ini("[a]\nk = v\n  cont\n\n  more = x\n").unwrap();
    assert_eq!(ini.options["a"], ["k", "more"]);
}

#[test]
fn values_may_hold_percent_comment_and_crlf_bytes() {
    let ini = parse_ini("[a]\nk = 100%s\r\nj = a;b\r\nl = a#b\r\n").unwrap();
    assert_eq!(ini.options["a"], ["k", "j", "l"]);
}

#[test]
fn lone_carriage_returns_do_not_split_lines() {
    let ini = parse_ini("[a]\rk = v\r[m]\rj = w\r").unwrap();
    assert_eq!(ini.sections(), ["a]\rk = v\r[m"]);
}

#[test]
fn same_option_in_two_sections_is_allowed() {
    let ini = parse_ini("[a]\nx = 1\n[b]\nx = 2\n").unwrap();
    assert_eq!(ini.sections(), ["a", "b"]);
}

/// Serve one canned response on loopback, like the script tests'
/// mocked `urlopen`: the fetch rule is pinned without the network.
fn serve_once(status: u16, body: Vec<u8>) -> (String, std::thread::JoinHandle<()>) {
    use std::io::{Read, Write};
    let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
    let url = format!("http://{}/locale.ini", listener.local_addr().unwrap());
    let handle = std::thread::spawn(move || {
        let (mut stream, _) = listener.accept().unwrap();
        let mut request = [0u8; 4096];
        let _ = stream.read(&mut request);
        let reason = if status == 200 { "OK" } else { "Error" };
        let head = format!(
            "HTTP/1.1 {status} {reason}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
            body.len()
        );
        stream.write_all(head.as_bytes()).unwrap();
        stream.write_all(&body).unwrap();
    });
    (url, handle)
}

fn fetch_case(status: u16, body: Vec<u8>, sha256: &str) -> Result<Vec<u8>, Error> {
    let (url, handle) = serve_once(status, body);
    let result = fetch_locked(&url, sha256);
    handle.join().unwrap();
    result
}

#[test]
fn locked_fetch_accepts_exact_bytes() {
    let body = b"[common]\nname = Native\n".to_vec();
    let sha = crate::locales::sha256_hex(&body);
    assert_eq!(fetch_case(200, body.clone(), &sha).unwrap(), body);
}

#[test]
fn locked_fetch_refuses_changed_bytes() {
    let err = fetch_case(200, b"incorrect".to_vec(), &"0".repeat(64)).unwrap_err();
    assert!(matches!(err, Error::Usage(_)));
    assert_eq!(err.message(), "native catalog differs from locked bytes");
}

#[test]
fn locked_fetch_refuses_oversize_bodies() {
    let mut body = vec![b'x'; 1024 * 1024 + 1];
    body[..8].copy_from_slice(b"[common]");
    let sha = crate::locales::sha256_hex(&body);
    let err = fetch_case(200, body, &sha).unwrap_err();
    assert!(matches!(err, Error::Usage(_)));
    assert_eq!(err.message(), "native catalog differs from locked bytes");
}

#[test]
fn locked_fetch_accepts_exactly_one_mib() {
    let mut body = vec![b'x'; 1024 * 1024];
    body[..8].copy_from_slice(b"[common]");
    let sha = crate::locales::sha256_hex(&body);
    assert_eq!(fetch_case(200, body.clone(), &sha).unwrap(), body);
}

#[test]
fn locked_fetch_refuses_error_status() {
    let body = b"[common]\nname = Native\n".to_vec();
    let sha = crate::locales::sha256_hex(&body);
    let err = fetch_case(404, body, &sha).unwrap_err();
    assert!(matches!(err, Error::Runtime(_)));
    assert!(err.message().contains("404"), "{}", err.message());
}
