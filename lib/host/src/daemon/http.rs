use std::io::Read;
use std::os::unix::net::UnixStream;
use std::time::Duration;

use crate::gmux_admission::{body_limit_for, RequestHead};

/// Max request head in bytes (Go: MaxHeaderBytes 8192).
const MAX_HEADER: usize = 8192;
/// Head read deadline (Go: ReadHeaderTimeout 5s).
const HEADER_TIMEOUT: Duration = Duration::from_secs(5);
/// Body read deadline (Rust http.rs precedent: 30s).
const BODY_TIMEOUT: Duration = Duration::from_secs(30);

pub enum ParseFailure {
    Invalid,
    HeadersTooLarge,
}

/// Parse the head, then read the body under the route's own limit.
/// Returns the head, the body bytes kept, and whether the body arrived
/// complete (a `false` is a decode failure for every subsystem, exactly
/// like Go's `MaxBytesReader` tripping before strict decode).
pub fn read_request(stream: &mut UnixStream) -> Result<(RequestHead, Vec<u8>, bool), ParseFailure> {
    stream
        .set_read_timeout(Some(HEADER_TIMEOUT))
        .map_err(|_| ParseFailure::Invalid)?;
    let mut head = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(ParseFailure::Invalid),
            Ok(_) => {
                head.push(byte[0]);
                if head.len() > MAX_HEADER {
                    return Err(ParseFailure::HeadersTooLarge);
                }
                if head.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            Err(_) => return Err(ParseFailure::Invalid),
        }
    }
    let mut lines = head.split(|b| *b == b'\n');
    let request_line = lines.next().ok_or(ParseFailure::Invalid)?;
    let request_line = request_line.strip_suffix(b"\r").unwrap_or(request_line);
    let mut parts = request_line.split(|b| *b == b' ');
    let method = parts.next().ok_or(ParseFailure::Invalid)?;
    let target = parts.next().ok_or(ParseFailure::Invalid)?;
    if parts.next().is_none() {
        return Err(ParseFailure::Invalid);
    }
    let method = std::str::from_utf8(method)
        .map_err(|_| ParseFailure::Invalid)?
        .to_string();
    let target = std::str::from_utf8(target).map_err(|_| ParseFailure::Invalid)?;
    let (raw_path, has_query) = match target.find('?') {
        Some(i) => (&target[..i], true),
        None => (target, false),
    };
    let escaped = raw_path.contains('%');
    let path = percent_decode(raw_path).map_err(|_| ParseFailure::Invalid)?;
    let mut origin_present = false;
    let mut upgrade_websocket = false;
    let mut ws_key = None;
    let mut content_lengths = Vec::new();
    let mut chunked = false;
    for line in lines {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if line.is_empty() {
            continue;
        }
        let Some(colon) = line.iter().position(|b| *b == b':') else {
            return Err(ParseFailure::Invalid);
        };
        let (name, value) = (&line[..colon], &line[colon + 1..]);
        if name.eq_ignore_ascii_case(b"origin") {
            // Any Origin line counts, even empty (Go: Header.Values != 0).
            origin_present = true;
        } else if name.eq_ignore_ascii_case(b"upgrade") {
            if value
                .split(|b| *b == b',')
                .any(|token| token.trim_ascii().eq_ignore_ascii_case(b"websocket"))
            {
                upgrade_websocket = true;
            }
        } else if name.eq_ignore_ascii_case(b"sec-websocket-key") {
            if ws_key.is_none() {
                ws_key = std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| ParseFailure::Invalid)?
                    .to_string()
                    .into();
            }
        } else if name.eq_ignore_ascii_case(b"content-length") {
            content_lengths.push(
                std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| ParseFailure::Invalid)?
                    .to_string(),
            );
        } else if name.eq_ignore_ascii_case(b"transfer-encoding")
            && value.trim_ascii().eq_ignore_ascii_case(b"chunked")
        {
            chunked = true;
        }
    }
    if content_lengths.len() > 1 || chunked {
        return Err(ParseFailure::Invalid);
    }
    let content_length = content_lengths
        .first()
        .map(|v| v.parse::<usize>().map_err(|_| ParseFailure::Invalid))
        .transpose()?
        .unwrap_or(0);
    let request = RequestHead {
        method,
        path: path.clone(),
        has_query,
        escaped,
        origin_present,
        upgrade_websocket,
        ws_key,
    };
    // Terminal upgrades hijack after the head like Go: no body is consumed.
    if request.method == "GET" && request.path == "/terminal" {
        return Ok((request, Vec::new(), true));
    }
    let limit = body_limit_for(&path);
    stream
        .set_read_timeout(Some(BODY_TIMEOUT))
        .map_err(|_| ParseFailure::Invalid)?;
    let want = content_length.min(limit + 1);
    let mut body = vec![0u8; want];
    stream
        .read_exact(&mut body)
        .map_err(|_| ParseFailure::Invalid)?;
    Ok((request, body, content_length <= limit))
}

/// Discard the remainder of an over-limit head: until the blank line,
/// a 1 MiB cap, or any read failure. Bounded so a hostile client
/// cannot park the connection thread past the read deadline.
pub fn drain_head(stream: &mut UnixStream) {
    const DRAIN_CAP: usize = 1 << 20;
    stream.set_read_timeout(Some(HEADER_TIMEOUT)).ok();
    let mut tail = [0u8; 4];
    let mut discarded = 0usize;
    let mut chunk = [0u8; 1024];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => return,
            Ok(n) => {
                discarded += n;
                for &b in &chunk[..n] {
                    tail.copy_within(1.., 0);
                    tail[3] = b;
                    if tail == *b"\r\n\r\n" {
                        return;
                    }
                }
                if discarded >= DRAIN_CAP {
                    return;
                }
            }
            Err(_) => return,
        }
    }
}

fn percent_decode(input: &str) -> Result<String, ()> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(());
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3]).map_err(|_| ())?;
            out.push(u8::from_str_radix(hex, 16).map_err(|_| ())?);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| ())
}
