//! Go `fmt` subset used by the installer: `%s %d %q %v %t %.1f %%`,
//! plus Go string helpers with exact bytes (`TrimSpace`, `Fields`,
//! `ToLower`, ASCII-anchored `EqualFold`) and Go `%q` quoting.

use std::fmt::Write as _;

/// Format argument, mirroring the Go call sites (`%s`/`%v` take [`Arg::Str`]).
pub enum Arg<'a> {
    Str(&'a str),
    Bytes(&'a [u8]),
    Int(i64),
    Uint(u64),
    Float(f64),
    Bool(bool),
}

/// Minimal `fmt.Sprintf`: supports `%s %d %q %v %t %.1f %%` only.
/// `%q` quotes [`Arg::Bytes`] (or [`Arg::Str`] bytes) with Go `%q` rules.
pub fn sprintf(format: &str, args: &[Arg<'_>]) -> String {
    let mut out = String::new();
    let mut rest = format;
    let mut index = 0;
    while let Some(pos) = rest.find('%') {
        out.push_str(&rest[..pos]);
        rest = &rest[pos + 1..];
        let (verb, tail) = match rest.chars().next() {
            Some('%') => {
                out.push('%');
                rest = &rest[1..];
                continue;
            }
            Some('.') if rest.starts_with(".1f") => ("f1", &rest[3..]),
            Some(c) => {
                let len = c.len_utf8();
                (&rest[..len], &rest[len..])
            }
            None => panic!("dangling format verb"),
        };
        let arg = args
            .get(index)
            .unwrap_or_else(|| panic!("missing format arg"));
        index += 1;
        match (verb, arg) {
            ("s", Arg::Str(s)) => out.push_str(s),
            ("s", Arg::Bytes(b)) => out.push_str(&String::from_utf8_lossy(b)),
            ("v", Arg::Str(s)) => out.push_str(s),
            ("v", Arg::Bytes(b)) => out.push_str(&String::from_utf8_lossy(b)),
            ("v", Arg::Int(n)) => write!(out, "{n}").unwrap(),
            ("v", Arg::Uint(n)) => write!(out, "{n}").unwrap(),
            ("v", Arg::Bool(b)) => out.push_str(if *b { "true" } else { "false" }),
            ("d", Arg::Int(n)) => write!(out, "{n}").unwrap(),
            ("d", Arg::Uint(n)) => write!(out, "{n}").unwrap(),
            ("t", Arg::Bool(b)) => out.push_str(if *b { "true" } else { "false" }),
            ("q", Arg::Str(s)) => go_quote_into(&mut out, s.as_bytes()),
            ("q", Arg::Bytes(b)) => go_quote_into(&mut out, b),
            ("f1", Arg::Float(f)) => write!(out, "{f:.1}").unwrap(),
            _ => panic!("unsupported format verb use"),
        }
        rest = tail;
    }
    out.push_str(rest);
    out
}

/// Go `%q` over bytes: printable runes raw, the rest escaped; each invalid
/// byte becomes `\xNN`. Non-ASCII printable classification follows Go's
/// `strconv.Quote` except exotic non-control runes (format/separator marks,
/// unassigned planes) are emitted raw where Go would escape them; installer
/// inputs (interface names, `ip`/`lsblk` lines, sysfs strings) never carry
/// those, and every realistic vector below matches byte for byte.
pub fn go_quote_into(out: &mut String, bytes: &[u8]) {
    out.push('"');
    let mut i = 0;
    while i < bytes.len() {
        match std::str::from_utf8(&bytes[i..]) {
            Ok(tail) => {
                for ch in tail.chars() {
                    quote_char(out, ch);
                }
                break;
            }
            Err(err) => {
                // Go emits one `\xNN` per invalid byte; consume one bad byte
                // per step so a multi-byte run quotes each byte separately.
                let valid = err.valid_up_to();
                for ch in std::str::from_utf8(&bytes[i..i + valid])
                    .unwrap_or("")
                    .chars()
                {
                    quote_char(out, ch);
                }
                write!(out, "\\x{:02x}", bytes[i + valid]).unwrap();
                i += valid + 1;
            }
        }
    }
    out.push('"');
}

fn quote_char(out: &mut String, ch: char) {
    match ch {
        '"' => out.push_str("\\\""),
        '\\' => out.push_str("\\\\"),
        '\n' => out.push_str("\\n"),
        '\r' => out.push_str("\\r"),
        '\t' => out.push_str("\\t"),
        c if (c as u32) < 0x20 => match c {
            '\u{7}' => out.push_str("\\a"),
            '\u{8}' => out.push_str("\\b"),
            '\u{c}' => out.push_str("\\f"),
            '\u{b}' => out.push_str("\\v"),
            _ => write!(out, "\\x{:02x}", c as u32).unwrap(),
        },
        '\u{7f}' => out.push_str("\\x7f"),
        c if c.is_control() => {
            let n = c as u32;
            if n < 0x10000 {
                write!(out, "\\u{n:04x}").unwrap();
            } else {
                write!(out, "\\U{n:08x}").unwrap();
            }
        }
        c => out.push(c),
    }
}

/// Go `strings.TrimSpace`: Go's `unicode.IsSpace` is exactly Unicode
/// White_Space, which is Rust's `char::is_whitespace` (including U+1680,
/// U+2000..U+200A, U+2028/2029, U+202F, U+205F, U+3000).
pub fn go_trim_space(value: &str) -> &str {
    value.trim_matches(char::is_whitespace)
}

/// Go `strings.Fields` with Go's space set.
pub fn go_fields(value: &str) -> Vec<&str> {
    value.split_whitespace().collect()
}

/// Go `strings.ToLower`: per-rune simple mapping (`İ` folds to `i`, unlike
/// Rust's full mapping which yields `i` plus a combining dot).
pub fn go_lower(value: &str) -> String {
    let mut out = String::with_capacity(value.len());
    for ch in value.chars() {
        if ch == 'İ' {
            out.push('i');
        } else {
            for lower in ch.to_lowercase() {
                out.push(lower);
            }
        }
    }
    out
}

/// Go `strings.EqualFold` against an ASCII constant: ASCII case-insensitive
/// plus the SimpleFold-to-ASCII mappings Go applies (`İ`/`ı` fold to `i`,
/// `ſ` folds to `s`; Kelvin `K` already lowercases to `k` in Rust).
/// Non-ASCII input that folds to nothing ASCII never matches, as in Go.
pub fn fold_eq_ascii(value: &str, constant: &str) -> bool {
    let mut vc = value.chars();
    let mut cc = constant.chars();
    loop {
        match (vc.next(), cc.next()) {
            (None, None) => return true,
            (Some(v), Some(c)) => {
                debug_assert!(c.is_ascii());
                let folded = match v {
                    'İ' | 'ı' => 'i',
                    'ſ' => 's',
                    _ => v,
                };
                let lower = if folded == 'İ' {
                    'i'
                } else {
                    let mut it = folded.to_lowercase();
                    match (it.next(), it.next()) {
                        (Some(single), None) => single,
                        _ => return false,
                    }
                };
                if lower != c.to_ascii_lowercase() {
                    return false;
                }
            }
            _ => return false,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn q(input: &[u8]) -> String {
        let mut out = String::new();
        go_quote_into(&mut out, input);
        out
    }

    #[test]
    fn quote_matches_go_vectors() {
        // Oracle: TestZZOracleFmt `Q` lines (Go `%q` of the same bytes).
        assert_eq!(q(b"eth0"), "\"eth0\"");
        assert_eq!(q(b"a\"b\\c"), "\"a\\\"b\\\\c\"");
        assert_eq!(q(b"line\nfeed\ttab"), "\"line\\nfeed\\ttab\"");
        assert_eq!(q(b"\x01\x02\x7f"), "\"\\x01\\x02\\x7f\"");
        assert_eq!(q("héllo".as_bytes()), "\"héllo\"");
        assert_eq!(q("日本語".as_bytes()), "\"日本語\"");
        assert_eq!(q(b"a\xffb"), "\"a\\xffb\"");
        assert_eq!(q(b"\xff\xfe"), "\"\\xff\\xfe\"");
        assert_eq!(q(b"trailing space "), "\"trailing space \"");
        assert_eq!(q(b"UPPER"), "\"UPPER\"");
        assert_eq!(q(b"sod\x00a"), "\"sod\\x00a\"");
        assert_eq!(q(b"\x00\x1f ~"), "\"\\x00\\x1f ~\"");
        assert_eq!(q("𝄞musical".as_bytes()), "\"𝄞musical\"");
        assert_eq!(q("écombining".as_bytes()), "\"écombining\"");
        assert_eq!(q(b"\x07\x08\x0c\x0b"), "\"\\a\\b\\f\\v\"");
    }

    #[test]
    fn floats_match_go_vectors() {
        // Oracle: TestZZOracleFmt `F1` lines.
        let cases: &[(u64, &str, &str)] = &[
            (0, "0.0", "0.0"),
            (1, "0.0", "0.0"),
            (1073741824, "1.0", "1024.0"),
            (1610612736, "1.5", "1536.0"),
            (68719476736, "64.0", "65536.0"),
            (1000000000, "0.9", "953.7"),
            (1503238553, "1.4", "1433.6"),
            (44040192, "0.0", "42.0"),
            (123456789, "0.1", "117.7"),
            (999999999999, "931.3", "953674.3"),
            (5905580032, "5.5", "5632.0"),
            (5905580033, "5.5", "5632.0"),
            (3221225472, "3.0", "3072.0"),
            (3758096384, "3.5", "3584.0"),
        ];
        for (size, gib, mib) in cases {
            let g = sprintf("%.1f", &[Arg::Float(*size as f64 / (1u64 << 30) as f64)]);
            let m = sprintf("%.1f", &[Arg::Float(*size as f64 / (1u64 << 20) as f64)]);
            assert_eq!((g.as_str(), m.as_str()), (*gib, *mib), "size {size}");
        }
        let max = u64::MAX as f64;
        assert_eq!(
            sprintf("%.1f", &[Arg::Float(max / (1u64 << 30) as f64)]),
            "17179869184.0"
        );
    }

    #[test]
    fn space_and_case_match_go() {
        // Oracle: TRIM/FIELDS/LOWER/FOLD lines.
        assert_eq!(
            go_trim_space("  \t\n\u{b}\u{c}\r \u{85}\u{a0}x\u{85}\u{a0}  "),
            "x"
        );
        assert_eq!(
            go_fields("a\tb\nc\u{b}d\u{c}e\rf\u{85}g\u{a0}h  i"),
            vec!["a", "b", "c", "d", "e", "f", "g", "h", "i"]
        );
        // Go's White_Space set includes the exotic Unicode spaces.
        assert_eq!(go_trim_space("\u{2000}x\u{2000}"), "x");
        assert_eq!(go_trim_space("\u{1680}\u{2028}x\u{2029}\u{3000}"), "x");
        assert_eq!(go_fields("a\u{2000}b\u{3000}c"), vec!["a", "b", "c"]);
        assert_eq!(go_lower("BACK"), "back");
        assert_eq!(go_lower("İ"), "i");
        assert_eq!(go_lower("Σς"), "σς");
        assert!(fold_eq_ascii("USB", "usb"));
        assert!(fold_eq_ascii("Back", "back"));
        assert!(fold_eq_ascii("k", "K"));
        assert!(fold_eq_ascii("UſB", "usb"));
        assert!(fold_eq_ascii("BACİ", "baci"));
        assert!(!fold_eq_ascii("uxb", "usb"));
        assert!(!fold_eq_ascii("us", "usb"));
        assert!(!fold_eq_ascii("usbb", "usb"));
    }

    #[test]
    fn verbs_cover_call_sites() {
        assert_eq!(sprintf("%d. %s", &[Arg::Uint(2), Arg::Str("x")]), "2. x");
        assert_eq!(
            sprintf(
                "%d. %q: %s",
                &[Arg::Int(1), Arg::Str("eth0"), Arg::Str("ip")]
            ),
            "1. \"eth0\": ip"
        );
        assert_eq!(
            sprintf(
                "%s failed (exit %d, interrupted %t); raw diagnostics suppressed",
                &[Arg::Str("ip"), Arg::Int(1), Arg::Bool(false)]
            ),
            "ip failed (exit 1, interrupted false); raw diagnostics suppressed"
        );
        assert_eq!(sprintf("100%% %v", &[Arg::Str("ok")]), "100% ok");
    }
}
