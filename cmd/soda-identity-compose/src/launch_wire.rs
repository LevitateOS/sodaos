use super::parse_json_integer;
use super::parse_json_string;
use super::skip_json_value;

// json_string matches Go encoding/json string escaping, including its
// HTML-safe <, >, & forms, so override bytes are identical for any input.
pub(crate) fn json_string(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
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
    out
}

pub(crate) struct NestedRegistration {
    pub(crate) child_id: String,
    pub(crate) actor_id: String,
    pub(crate) registration_id: String,
    pub(crate) muse: bool,
}

// launch_request_json emits the exact Go LaunchRequest field order for a
// registration: empty home/config_home/term omitted, nil args as null.
pub(crate) fn launch_request_json(request: &NestedRegistration) -> String {
    let mut out = String::from("{\"register\":{\"child_id\":");
    out.push_str(&json_string(&request.child_id));
    out.push_str(",\"actor_id\":");
    out.push_str(&json_string(&request.actor_id));
    out.push_str(",\"registration_id\":");
    out.push_str(&json_string(&request.registration_id));
    out.push_str(",\"muse\":");
    out.push_str(if request.muse { "true" } else { "false" });
    out.push_str(
        "},\"connection_id\":\"\",\"cwd\":\"\",\"args\":null,\"tty\":false,\"cols\":0,\"rows\":0}",
    );
    out
}

// parse_launch_exit mirrors Go json.Unmarshal into LaunchExit: missing
// fields stay zero, unknown fields are ignored, malformed JSON rejects.
pub(crate) fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    let text = std::str::from_utf8(body).map_err(|_| ())?;
    let mut code: i64 = 0;
    let mut error_text = String::new();
    let mut i = 0;
    let bytes = text.as_bytes();
    let skip_ws = |i: &mut usize| {
        while *i < bytes.len() && matches!(bytes[*i], b' ' | b'\t' | b'\n' | b'\r') {
            *i += 1;
        }
    };
    skip_ws(&mut i);
    if i >= bytes.len() || bytes[i] != b'{' {
        return Err(());
    }
    i += 1;
    // Go rejects a trailing comma, so `}` is only valid here for `{}` or
    // right after a value; after a comma a key is required.
    let mut after_comma = false;
    loop {
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b'}' {
            if after_comma {
                return Err(());
            }
            i += 1;
            break;
        }
        if i >= bytes.len() || bytes[i] != b'"' {
            return Err(());
        }
        let (key, next) = parse_json_string(text, i)?;
        i = next;
        skip_ws(&mut i);
        if i >= bytes.len() || bytes[i] != b':' {
            return Err(());
        }
        i += 1;
        skip_ws(&mut i);
        if key == "code" {
            let (value, next) = parse_json_integer(text, i)?;
            code = value;
            i = next;
        } else if key == "error" {
            if i >= bytes.len() || bytes[i] != b'"' {
                return Err(());
            }
            let (value, next) = parse_json_string(text, i)?;
            error_text = value;
            i = next;
        } else {
            i = skip_json_value(text, i)?;
        }
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b',' {
            i += 1;
            after_comma = true;
            continue;
        }
        if i < bytes.len() && bytes[i] == b'}' {
            i += 1;
            break;
        }
        return Err(());
    }
    skip_ws(&mut i);
    if i != bytes.len() {
        return Err(());
    }
    Ok((code, error_text))
}
