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

fn parse_json_string(text: &str, start: usize) -> Result<(String, usize), ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() || bytes[start] != b'"' {
        return Err(());
    }
    let mut out = String::new();
    let mut i = start + 1;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Ok((out, i + 1)),
            b'\\' => {
                i += 1;
                if i >= bytes.len() {
                    return Err(());
                }
                match bytes[i] {
                    b'"' => out.push('"'),
                    b'\\' => out.push('\\'),
                    b'/' => out.push('/'),
                    b'b' => out.push('\u{0008}'),
                    b'f' => out.push('\u{000c}'),
                    b'n' => out.push('\n'),
                    b'r' => out.push('\r'),
                    b't' => out.push('\t'),
                    b'u' => {
                        if i + 4 >= bytes.len() {
                            return Err(());
                        }
                        let hex = &text[i + 1..i + 5];
                        let cp = u32::from_str_radix(hex, 16).map_err(|_| ())?;
                        let c = char::from_u32(cp).ok_or(())?;
                        if (0xd800..0xe000).contains(&cp) {
                            return Err(());
                        }
                        out.push(c);
                        i += 4;
                    }
                    _ => return Err(()),
                }
            }
            0x00..=0x1f => return Err(()),
            _ => {
                let c = text[i..].chars().next().ok_or(())?;
                out.push(c);
                i += c.len_utf8() - 1;
            }
        }
        i += 1;
    }
    Err(())
}

fn parse_json_integer(text: &str, start: usize) -> Result<(i64, usize), ()> {
    let bytes = text.as_bytes();
    let mut i = start;
    if i < bytes.len() && bytes[i] == b'-' {
        i += 1;
    }
    let digits = i;
    while i < bytes.len() && bytes[i].is_ascii_digit() {
        i += 1;
    }
    if i == digits || i == start {
        return Err(());
    }
    // Go rejects fractions and exponents for int fields.
    if i < bytes.len() && (bytes[i] == b'.' || bytes[i] == b'e' || bytes[i] == b'E') {
        return Err(());
    }
    text[start..i]
        .parse::<i64>()
        .map_err(|_| ())
        .map(|v| (v, i))
}

fn skip_json_value(text: &str, start: usize) -> Result<usize, ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() {
        return Err(());
    }
    match bytes[start] {
        b'"' => parse_json_string(text, start).map(|(_, next)| next),
        b'{' | b'[' => {
            let open = bytes[start];
            let close = if open == b'{' { b'}' } else { b']' };
            let mut i = start + 1;
            let mut depth = 1;
            let mut in_string = false;
            let mut escaped = false;
            while i < bytes.len() {
                let b = bytes[i];
                if in_string {
                    if escaped {
                        escaped = false;
                    } else if b == b'\\' {
                        escaped = true;
                    } else if b == b'"' {
                        in_string = false;
                    }
                } else if b == b'"' {
                    in_string = true;
                } else if b == open {
                    depth += 1;
                } else if b == close {
                    depth -= 1;
                    if depth == 0 {
                        return Ok(i + 1);
                    }
                }
                i += 1;
            }
            Err(())
        }
        b't' => {
            if text[start..].starts_with("true") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'f' => {
            if text[start..].starts_with("false") {
                Ok(start + 5)
            } else {
                Err(())
            }
        }
        b'n' => {
            if text[start..].starts_with("null") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'-' | b'0'..=b'9' => {
            let mut i = start;
            if bytes[i] == b'-' {
                i += 1;
            }
            while i < bytes.len()
                && (bytes[i].is_ascii_digit()
                    || matches!(bytes[i], b'.' | b'e' | b'E' | b'+' | b'-'))
            {
                i += 1;
            }
            if i == start {
                return Err(());
            }
            Ok(i)
        }
        _ => Err(()),
    }
}
