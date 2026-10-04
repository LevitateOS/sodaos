//! Control-plane protocol: dimensions, control frames, names, argv.
//! Vectored against CPython behavior.

use soda_json::JsonValue;

use crate::b64;
use crate::pyemit::{as_int, shape, unique_entries};

pub const FRAME_LIMIT: usize = 32768;
pub const QUEUE_LIMIT: usize = 262144;
pub const HEARTBEAT_SECONDS: u64 = 60;

pub fn dimensions(cols: i64, rows: i64) -> bool {
    (2..=500).contains(&cols) && (2..=300).contains(&rows)
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ControlFrame {
    Input(Vec<u8>),
    Resize { cols: i64, rows: i64 },
    Heartbeat,
    Close,
}

/// `decode_frame`: strict shapes over unique-key objects.
pub fn decode_frame(raw: &[u8]) -> Option<ControlFrame> {
    let text = std::str::from_utf8(raw).ok()?;
    let value = JsonValue::parse(text).ok()?;
    let entries = unique_entries(&value)?;
    let kind = entries.iter().find(|(k, _)| k == "type")?.1.as_str()?;
    match kind {
        "input" => {
            let fields = shape(entries, &["type", "data"])?;
            let data = fields.iter().find(|(k, _)| k == "data")?.1.as_str()?;
            let decoded = b64::decode(data)?;
            if decoded.is_empty() || decoded.len() > 16384 {
                return None;
            }
            Some(ControlFrame::Input(decoded))
        }
        "resize" => {
            let fields = shape(entries, &["type", "cols", "rows"])?;
            let cols = as_int(fields.iter().find(|(k, _)| k == "cols")?.1)?;
            let rows = as_int(fields.iter().find(|(k, _)| k == "rows")?.1)?;
            if !dimensions(cols, rows) {
                return None;
            }
            Some(ControlFrame::Resize { cols, rows })
        }
        "heartbeat" | "close" if shape(entries, &["type"]).is_some() => {
            if kind == "heartbeat" {
                Some(ControlFrame::Heartbeat)
            } else {
                Some(ControlFrame::Close)
            }
        }
        _ => None,
    }
}

/// `unicodedata.category(c) in ('Cc', 'Cf')`, Unicode 15.0 table.
pub fn is_cc_or_cf(ch: char) -> bool {
    let cp = ch as u32;
    matches!(cp, 0x00..=0x1f | 0x7f..=0x9f)
        || matches!(
            cp,
            0x00ad
                | 0x0600..=0x0605
                | 0x061c
                | 0x06dd
                | 0x070f
                | 0x0890..=0x0891
                | 0x08e2
                | 0x180e
                | 0x200b..=0x200f
                | 0x202a..=0x202e
                | 0x2060..=0x2064
                | 0x2066..=0x206f
                | 0xfeff
                | 0xfff9..=0xfffb
                | 0x110bd
                | 0x110cd
                | 0x13430..=0x1343f
                | 0x1bca0..=0x1bca3
                | 0x1d173..=0x1d17a
                | 0xe0001
                | 0xe0020..=0xe007f
        )
}

/// `valid_name`: at most 80 chars, no Cc/Cf.
pub fn valid_name(value: &str) -> bool {
    value.chars().count() <= 80 && !value.chars().any(is_cc_or_cf)
}

/// 10 control arguments after the action.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ControlArgs {
    pub action: String,
    pub identifier: String,
    pub login: String,
    pub identity: i64,
    pub cols: i64,
    pub rows: i64,
    pub seconds: i64,
    pub source_hash: String,
    pub name: String,
    pub scope: String,
}

pub fn parse_control_argv(argv: &[String]) -> Option<ControlArgs> {
    if argv.len() != 10 {
        return None;
    }
    let identity: i64 = argv[3].parse().ok()?;
    let cols: i64 = argv[4].parse().ok()?;
    let rows: i64 = argv[5].parse().ok()?;
    let seconds: i64 = argv[6].parse().ok()?;
    // `int()` accepts surrounding whitespace and +/-; Rust's parse is
    // stricter, so re-check the canonical shapes the callers emit.
    for (raw, parsed) in [
        (&argv[3], identity),
        (&argv[4], cols),
        (&argv[5], rows),
        (&argv[6], seconds),
    ] {
        if raw.is_empty()
            || (raw != &parsed.to_string()
                && !(raw.starts_with('+') && &raw[1..] == &parsed.to_string()))
        {
            // Python int() also strips whitespace; emulate exactly.
            let trimmed = raw.trim();
            if trimmed.parse::<i64>().ok() != Some(parsed) {
                return None;
            }
        }
    }
    if !matches!(
        argv[0].as_str(),
        "reserve" | "create" | "attach" | "inspect" | "list" | "end" | "rename"
    ) || !(1..=43200).contains(&seconds)
    {
        return None;
    }
    require_terminal_target(&argv[0], &argv[1])?;
    if (matches!(argv[0].as_str(), "reserve" | "create" | "attach") && !dimensions(cols, rows))
        || !valid_name(&argv[8])
    {
        return None;
    }
    require_creation_scope(&argv[0], &argv[9])?;
    Some(ControlArgs {
        action: argv[0].clone(),
        identifier: argv[1].clone(),
        login: argv[2].clone(),
        identity,
        cols,
        rows,
        seconds,
        source_hash: argv[7].clone(),
        name: argv[8].clone(),
        scope: argv[9].clone(),
    })
}

pub fn valid_identifier(identifier: &str) -> bool {
    identifier.len() == 32
        && identifier
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

fn require_terminal_target(action: &str, identifier: &str) -> Option<()> {
    if action != "list" {
        if !valid_identifier(identifier) {
            return None;
        }
    } else if !identifier.is_empty() {
        return None;
    }
    Some(())
}

pub fn valid_scope(scope: &str) -> bool {
    scope.len() == 64
        && scope
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

fn require_creation_scope(action: &str, scope: &str) -> Option<()> {
    if matches!(action, "reserve" | "create") {
        if !valid_scope(scope) {
            return None;
        }
    } else if !scope.is_empty() {
        return None;
    }
    Some(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn argv(action: &str) -> Vec<String> {
        vec![
            action.to_string(),
            "a".repeat(32),
            "op".to_string(),
            "42".to_string(),
            "80".to_string(),
            "24".to_string(),
            "3600".to_string(),
            "b".repeat(64),
            "term".to_string(),
            if matches!(action, "reserve" | "create") {
                "c".repeat(64)
            } else {
                String::new()
            },
        ]
    }

    #[test]
    fn frames() {
        assert_eq!(
            decode_frame(br#"{"type":"input","data":"YWI="}"#),
            Some(ControlFrame::Input(b"ab".to_vec()))
        );
        assert_eq!(
            decode_frame(br#"{"type":"resize","cols":80,"rows":24}"#),
            Some(ControlFrame::Resize { cols: 80, rows: 24 })
        );
        assert_eq!(
            decode_frame(br#"{"type":"heartbeat"}"#),
            Some(ControlFrame::Heartbeat)
        );
        assert_eq!(
            decode_frame(br#"{"type":"close"}"#),
            Some(ControlFrame::Close)
        );
        // Shape violations.
        assert_eq!(decode_frame(br#"{"type":"close","x":1}"#), None);
        assert_eq!(decode_frame(br#"{"type":"input","data":""}"#), None);
        assert_eq!(decode_frame(br#"{"type":"input","data":"!!!}"#), None);
        assert_eq!(
            decode_frame(br#"{"type":"resize","cols":1,"rows":24}"#),
            None
        );
        assert_eq!(
            decode_frame(br#"{"type":"resize","cols":80,"rows":24.0}"#),
            None
        );
        assert_eq!(decode_frame(br#"{"type":"nope"}"#), None);
        assert_eq!(decode_frame(br#"{"type":"close","type":"close"}"#), None);
        assert_eq!(decode_frame(b"[1]"), None);
        assert_eq!(decode_frame(b"\xff\xfe"), None);
        // Oversized input rejected (16385 decoded bytes).
        let big = "Y".repeat(21848);
        assert_eq!(
            decode_frame(format!("{{\"type\":\"input\",\"data\":\"{big}\"}}").as_bytes()),
            None
        );
    }

    #[test]
    fn names() {
        assert!(valid_name("term"));
        assert!(valid_name(""));
        assert!(valid_name(&"é".repeat(80)));
        assert!(!valid_name(&"a".repeat(81)));
        assert!(!valid_name("a\nb"));
        assert!(!valid_name("a\x7fb"));
        assert!(!valid_name("a\u{200b}b"));
        assert!(!valid_name("a\u{feff}b"));
        assert!(valid_name("a\u{2000}b")); // Zs, not Cf
    }

    #[test]
    fn argv_matrix() {
        for action in ["reserve", "create", "attach", "inspect", "end", "rename"] {
            assert!(parse_control_argv(&argv(action)).is_some(), "{action}");
        }
        let mut list = argv("list");
        list[1] = String::new();
        assert!(parse_control_argv(&list).is_some());
        // Violations.
        let mut bad = argv("attach");
        bad[1] = "ZZZ".to_string();
        assert!(parse_control_argv(&bad).is_none());
        let mut bad = argv("list");
        assert!(parse_control_argv(&bad).is_none()); // list with target
        let mut bad = argv("reserve");
        bad[9] = String::new();
        assert!(parse_control_argv(&bad).is_none()); // missing scope
        let mut bad = argv("attach");
        bad[9] = "c".repeat(64);
        assert!(parse_control_argv(&bad).is_none()); // unexpected scope
        let mut bad = argv("attach");
        bad[6] = "0".to_string();
        assert!(parse_control_argv(&bad).is_none()); // seconds bound
        let mut bad = argv("create");
        bad[4] = "1".to_string();
        assert!(parse_control_argv(&bad).is_none()); // dimensions
        let mut bad = argv("bogus");
        assert!(parse_control_argv(&bad).is_none());
        assert!(parse_control_argv(&argv("attach")[..9]).is_none());
        // Uppercase hex rejected (fullmatch [0-9a-f]).
        let mut bad = argv("attach");
        bad[1] = "A".repeat(32);
        assert!(parse_control_argv(&bad).is_none());
    }
}
