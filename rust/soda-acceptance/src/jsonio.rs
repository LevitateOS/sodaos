//! Strict JSON input and Go-exact JSON output.
//!
//! [`read_json_file`] mirrors `ReadJSON`: the parent directory stays open,
//! the file must be regular and bounded, and its bytes must not change
//! between stat and read. Decoding rejects unknown fields and trailing
//! data, like Go's `DisallowUnknownFields` plus the second-decode check.
//!
//! Emission mirrors Go's `encoding/json` byte for byte: HTML escaping
//! (`<`, `>`, `&`), `U+2028/2029` escaping, raw number passthrough (the
//! `UseNumber` identity the evidence tests pin), compact separators for
//! reports and two-space indented separators for structured evidence.

use std::fs::File;
use std::io::Read;
use std::path::Path;

use soda_json::JsonValue;

use crate::error::Error;
use crate::files::{self, FileAttr, OwnedDir};
use crate::sha256;

/// Read one JSON value from a bounded regular file, returning the value and
/// the SHA-256 hex of the exact bytes decoded. Mirrors `ReadJSON`.
pub fn read_json_file(path: &str) -> Result<(JsonValue, String), Error> {
    let file = Path::new(path);
    let parent = file.parent().unwrap_or(Path::new("/"));
    let base = file.file_name().ok_or_else(|| Error::msg("bounded regular JSON input required"))?;
    let root = OwnedDir::open(&parent.to_string_lossy())?;
    let name = base.to_string_lossy();
    let before = root.lstat_at(&name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let mut handle = root.open_file_at(&name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

/// Read one JSON value through an already-open directory. Mirrors
/// `ReadJSONAt`: same bounds and change checks, digest of decoded bytes.
pub fn read_json_at(root: &OwnedDir, name: &str) -> Result<(JsonValue, String), Error> {
    let before = root.lstat_at(name)?;
    if !before.is_regular || before.size > files::JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let mut handle = root.open_file_at(name)?;
    let after = fstat_attr(&handle)?;
    if !after.is_regular || !files::same_file(before, after) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    handle.take(files::JSON_LIMIT + 1).read_to_end(&mut data)?;
    if data.len() as u64 > files::JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid JSON input"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid JSON input"))?;
    Ok((value, sha256::hex_lower(&sha256::digest(&data))))
}

fn fstat_attr(file: &File) -> Result<FileAttr, Error> {
    use std::os::fd::AsRawFd;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let mode = st.st_mode;
    Ok(FileAttr {
        is_regular: mode & libc::S_IFMT == libc::S_IFREG,
        is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
        is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
        perm: mode & 0o7777,
        size: st.st_size.max(0) as u64,
        uid: st.st_uid,
        gid: st.st_gid,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    })
}

/// Reject any object key outside `known`, like `DisallowUnknownFields`.
/// Duplicate keys keep last-wins lookup (matching Go), but every spelling
/// present must be known.
pub fn check_no_unknown(value: &JsonValue, known: &[&str]) -> Result<(), Error> {
    if let JsonValue::Object(entries) = value {
        for (key, _) in entries {
            if !known.contains(&key.as_str()) {
                return Err(Error::msg(format!("json: unknown field {key:?}")));
            }
        }
    }
    Ok(())
}

/// Required string field, like decoding into a Go `string`.
pub fn require_string<'a>(value: &'a JsonValue, field: &str) -> Result<&'a str, Error> {
    value
        .get(field)
        .and_then(|v| v.as_str())
        .ok_or_else(|| Error::msg(format!("invalid {field}: string required")))
}

/// Required strict integer field.
pub fn require_integer(value: &JsonValue, field: &str) -> Result<i128, Error> {
    value
        .get(field)
        .and_then(|v| v.as_integer())
        .ok_or_else(|| Error::msg(format!("invalid {field}: integer required")))
}

/// Required boolean field.
pub fn require_bool(value: &JsonValue, field: &str) -> Result<bool, Error> {
    value
        .get(field)
        .and_then(|v| v.as_bool())
        .ok_or_else(|| Error::msg(format!("invalid {field}: boolean required")))
}

/// Escape a string exactly like Go's `encoding/json`: short escapes, HTML
/// characters, `U+2028/2029`, and `\u00xx` for other controls.
pub fn escape_go(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

/// Compact emission, like `json.Marshal`. Object entries keep caller order;
/// callers sort maps first, matching Go's sorted map keys.
pub fn write_compact(out: &mut String, value: &JsonValue) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(s) => escape_go(out, s),
        JsonValue::Array(items) => {
            out.push('[');
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_compact(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            out.push('{');
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                escape_go(out, key);
                out.push(':');
                write_compact(out, item);
            }
            out.push('}');
        }
    }
}

/// Two-space indented emission, like `json.MarshalIndent(v, "", "  ")`.
/// No trailing newline; callers append it like the Go owner does.
pub fn write_indent(out: &mut String, value: &JsonValue) {
    write_indent_at(out, value, 0);
}

fn write_indent_at(out: &mut String, value: &JsonValue, depth: usize) {
    match value {
        JsonValue::Array(items) if !items.is_empty() => {
            out.push_str("[\n");
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push(']');
        }
        JsonValue::Object(entries) if !entries.is_empty() => {
            out.push_str("{\n");
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 {
                    out.push_str(",\n");
                }
                write_padding(out, depth + 1);
                escape_go(out, key);
                out.push_str(": ");
                write_indent_at(out, item, depth + 1);
            }
            out.push('\n');
            write_padding(out, depth);
            out.push('}');
        }
        _ => write_compact(out, value),
    }
}

fn write_padding(out: &mut String, depth: usize) {
    for _ in 0..depth {
        out.push_str("  ");
    }
}

/// Current UTC time in Go's `time.RFC3339Nano` shape: trailing zero
/// fractional digits are trimmed, and a whole second has no fraction.
pub fn now_rfc3339_nano() -> String {
    let now = std::time::SystemTime::now();
    let elapsed = now.duration_since(std::time::UNIX_EPOCH).unwrap_or_default();
    format_unix_nano(elapsed.as_secs() as i64, elapsed.subsec_nanos())
}

fn format_unix_nano(secs: i64, nanos: u32) -> String {
    let days = secs.div_euclid(86_400);
    let time = secs.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    let (hour, minute, second) = (time / 3_600, (time % 3_600) / 60, time % 60);
    let mut out = format!("{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}");
    if nanos != 0 {
        let mut frac = format!("{nanos:09}");
        while frac.ends_with('0') {
            frac.pop();
        }
        out.push('.');
        out.push_str(&frac);
    }
    out.push('Z');
    out
}

/// Days since the Unix epoch to civil date (Hinnant's algorithm).
fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z.rem_euclid(146_097);
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let year = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let month = (if mp < 10 { mp + 3 } else { mp - 9 }) as u32;
    (if month <= 2 { year + 1 } else { year }, month, day)
}

/// Validate strict RFC3339 timestamps, like Go's `time.Time` JSON decode:
/// real calendar dates, `T` separator, second precision with an optional
/// 1-9 digit fraction, and `Z` or a numeric offset.
pub fn validate_rfc3339(text: &str) -> Result<(), Error> {
    let err = || Error::msg(format!("invalid timestamp: {text:?}"));
    let (date, rest) = text.split_once('T').ok_or_else(err)?;
    let year: i64 = date.get(0..4).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    let month: u32 = date.get(5..7).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    let day: u32 = date.get(8..10).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    if date.len() != 10 || date.as_bytes()[4] != b'-' || date.as_bytes()[7] != b'-' {
        return Err(err());
    }
    if !(1..=12).contains(&month) || day == 0 || day > days_in_month(year, month) {
        return Err(err());
    }
    let (clock, zone) = split_zone(rest).ok_or_else(err)?;
    let hour: u32 = clock.get(0..2).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    let minute: u32 = clock.get(3..5).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    let second: u32 = clock.get(6..8).and_then(|s| s.parse().ok()).ok_or_else(err)?;
    if clock.len() < 8 || clock.as_bytes()[2] != b':' || clock.as_bytes()[5] != b':' {
        return Err(err());
    }
    if hour > 23 || minute > 59 || second > 59 {
        return Err(err());
    }
    if clock.len() > 8 {
        let frac = clock.get(8..).ok_or_else(err)?;
        if !frac.starts_with('.') || frac.len() < 2 || frac.len() > 10 || !frac[1..].bytes().all(|b| b.is_ascii_digit()) {
            return Err(err());
        }
    }
    match zone {
        "Z" => Ok(()),
        _ => {
            let sign = zone.as_bytes().first().ok_or_else(err)?;
            if *sign != b'+' && *sign != b'-' {
                return Err(err());
            }
            let hour: u32 = zone.get(1..3).and_then(|s| s.parse().ok()).ok_or_else(err)?;
            let minute: u32 = zone.get(4..6).and_then(|s| s.parse().ok()).ok_or_else(err)?;
            if zone.len() != 6 || zone.as_bytes()[3] != b':' || hour > 23 || minute > 59 {
                return Err(err());
            }
            Ok(())
        }
    }
}

fn split_zone(rest: &str) -> Option<(&str, &str)> {
    if let Some(clock) = rest.strip_suffix('Z') {
        return Some((clock, "Z"));
    }
    let bytes = rest.as_bytes();
    for i in (0..bytes.len()).rev() {
        if bytes[i] == b'+' || (bytes[i] == b'-' && i > 7) {
            return Some((&rest[..i], &rest[i..]));
        }
    }
    None
}

fn days_in_month(year: i64, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        _ => {
            if year % 4 == 0 && (year % 100 != 0 || year % 400 == 0) {
                29
            } else {
                28
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse(text: &str) -> JsonValue {
        JsonValue::parse(text).unwrap()
    }

    #[test]
    fn emitter_matches_go_byte_for_byte() {
        // Fixtures generated with Go's encoding/json during implementation;
        // see the oracle note in the commit message.
        let cases = [
            (
                "{\"b\":[1,true,null,\"x\"],\"a\":{\"n\":-12.5e2,\"s\":\"a<b>&\\\"c\\\"\\n\\u0001\\u2028\"}}",
                "{\"a\":{\"n\":-12.5e2,\"s\":\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\u0001\\u2028\"},\"b\":[1,true,null,\"x\"]}",
            ),
            ("{}", "{}"),
            ("[]", "[]"),
            ("{\"e\":{},\"l\":[]}", "{\"e\":{},\"l\":[]}"),
            ("{\"big\":9223372036854775807}", "{\"big\":9223372036854775807}"),
        ];
        for (input, want) in cases {
            let value = parse(input);
            // Go sorts map keys; emulate by sorting top-level object entries.
            let mut sorted = String::new();
            if let JsonValue::Object(entries) = &value {
                let mut ordered = entries.clone();
                ordered.sort_by(|a, b| a.0.cmp(&b.0));
                write_compact(&mut sorted, &JsonValue::Object(ordered));
            } else {
                write_compact(&mut sorted, &value);
            }
            assert_eq!(sorted, want, "compact {input}");
        }
        let mut indent = String::new();
        write_indent(&mut indent, &parse("{\"a\":null,\"b\":[1,{}]}"));
        assert_eq!(indent, "{\n  \"a\": null,\n  \"b\": [\n    1,\n    {}\n  ]\n}");
    }

    #[test]
    fn civil_dates_cover_edges() {
        assert_eq!(format_unix_nano(0, 0), "1970-01-01T00:00:00Z");
        assert_eq!(format_unix_nano(0, 120_000_000), "1970-01-01T00:00:00.12Z");
        assert_eq!(format_unix_nano(1_704_067_200, 0), "2024-01-01T00:00:00Z");
        assert_eq!(format_unix_nano(1_893_456_000, 1), "2030-01-01T00:00:00.000000001Z");
        assert_eq!(format_unix_nano(-1, 0), "1969-12-31T23:59:59Z");
        assert!(validate_rfc3339(&now_rfc3339_nano()).is_ok());
    }

    #[test]
    fn timestamps_validate_strictly() {
        for good in [
            "2026-10-04T18:05:00Z",
            "2026-10-04T18:05:00.123456789Z",
            "2024-02-29T00:00:00+00:00",
            "0001-01-01T00:00:00Z",
        ] {
            assert!(validate_rfc3339(good).is_ok(), "{good}");
        }
        for bad in [
            "2026-10-04 18:05:00Z",
            "2026-13-01T00:00:00Z",
            "2023-02-29T00:00:00Z",
            "2026-10-04T24:00:00Z",
            "2026-10-04T00:00:00",
            "2026-10-04T00:00:00.",
            "2026-10-04T00:00:00.1234567890Z",
            "2026-10-04T00:00:61Z",
            "not-a-time",
        ] {
            assert!(validate_rfc3339(bad).is_err(), "{bad}");
        }
    }

    #[test]
    fn strict_read_rejects_unknown_fields_and_drift() {
        let dir = std::env::temp_dir().join(format!("soda-jsonio-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("input.json");
        std::fs::write(&path, b"{\"Revision\":\"x\"} trailing").unwrap();
        assert!(read_json_file(path.to_str().unwrap()).is_err());
        std::fs::write(&path, b"{\"Revision\":\"x\",\"Extra\":1}").unwrap();
        let (value, digest) = read_json_file(path.to_str().unwrap()).unwrap();
        assert_eq!(digest.len(), 64);
        assert_eq!(require_string(&value, "Revision").unwrap(), "x");
        assert!(check_no_unknown(&value, &["Revision"]).is_err());
        assert!(check_no_unknown(&value, &["Revision", "Extra"]).is_ok());
        assert!(require_integer(&value, "Extra").is_ok());
        assert!(require_bool(&value, "Extra").is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
