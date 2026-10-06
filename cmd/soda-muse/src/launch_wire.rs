use super::launch_json::{parse_json_integer, parse_json_string, skip_json_value};
use super::ShellRequest;

// shell_request_json emits the exact Go LaunchRequest field order for a
// shell: empty home/config_home/term omitted, argv always an array.
pub(crate) fn shell_request_json(r: &ShellRequest) -> String {
    let mut out = String::from("{");
    let mut first = true;
    let field = |out: &mut String, first: &mut bool, name: &str, value: &str| {
        if !*first {
            out.push(',');
        }
        *first = false;
        out.push_str(&json_string(name));
        out.push(':');
        out.push_str(value);
    };
    if !r.home.is_empty() {
        field(&mut out, &mut first, "home", &json_string(&r.home));
    }
    if !r.config_home.is_empty() {
        field(
            &mut out,
            &mut first,
            "config_home",
            &json_string(&r.config_home),
        );
    }
    if !r.term.is_empty() {
        field(&mut out, &mut first, "term", &json_string(&r.term));
    }
    field(
        &mut out,
        &mut first,
        "connection_id",
        &json_string(&r.connection_id),
    );
    field(&mut out, &mut first, "cwd", &json_string(&r.cwd));
    let mut args = String::from("[");
    for (i, a) in r.args.iter().enumerate() {
        if i > 0 {
            args.push(',');
        }
        args.push_str(&json_string(a));
    }
    args.push(']');
    field(&mut out, &mut first, "args", &args);
    field(
        &mut out,
        &mut first,
        "tty",
        if r.tty { "true" } else { "false" },
    );
    field(&mut out, &mut first, "cols", &r.cols.to_string());
    field(&mut out, &mut first, "rows", &r.rows.to_string());
    out.push('}');
    out
}

pub(crate) fn base64_encode(data: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let mut n: u32 = 0;
        for (i, b) in chunk.iter().enumerate() {
            n |= (*b as u32) << (16 - 8 * i);
        }
        out.push(ALPHABET[(n >> 18) as usize & 63] as char);
        out.push(ALPHABET[(n >> 12) as usize & 63] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[(n >> 6) as usize & 63] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[n as usize & 63] as char);
        } else {
            out.push('=');
        }
    }
    out
}

// json_string matches Go encoding/json string escaping, including its
// HTML-safe <, >, & forms, so wire bytes are identical for any input.
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
