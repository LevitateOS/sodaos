// HTTP codec and envelopes, extracted from http.rs (A05.M).
use crate::http_routes::{BODY_TIMEOUT, HEADER_TIMEOUT, MAX_BODY, MAX_HEADER};
use std::io::Read;
use std::os::unix::net::UnixStream;
pub(crate) enum Admission {
    Invalid,
    HeadersTooLarge,
}

pub(crate) struct HttpRequest {
    pub(crate) method: String,
    pub(crate) path: String,
    pub(crate) query: String,
    pub(crate) origin: String,
}

pub(crate) fn read_request(stream: &mut UnixStream) -> Result<(HttpRequest, Vec<u8>), Admission> {
    stream
        .set_read_timeout(Some(HEADER_TIMEOUT))
        .map_err(|_| Admission::Invalid)?;
    let mut head = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(Admission::Invalid),
            Ok(_) => {
                head.push(byte[0]);
                if head.len() > MAX_HEADER {
                    return Err(Admission::HeadersTooLarge);
                }
                if head.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            Err(_) => return Err(Admission::Invalid),
        }
    }
    let mut lines = head.split(|b| *b == b'\n');
    let request_line = lines.next().ok_or(Admission::Invalid)?;
    let mut parts = request_line.split(|b| *b == b' ');
    let method = parts.next().ok_or(Admission::Invalid)?;
    let target = parts.next().ok_or(Admission::Invalid)?;
    if parts.next().is_none() {
        return Err(Admission::Invalid);
    }
    let method = std::str::from_utf8(method)
        .map_err(|_| Admission::Invalid)?
        .to_string();
    let target = std::str::from_utf8(target).map_err(|_| Admission::Invalid)?;
    let (path, query) = match target.find('?') {
        Some(i) => (&target[..i], target[i + 1..].to_string()),
        None => (target, String::new()),
    };
    let path = percent_decode(path).map_err(|_| Admission::Invalid)?;
    let mut origin = String::new();
    let mut content_lengths = Vec::new();
    let mut chunked = false;
    for line in lines {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if line.is_empty() {
            continue;
        }
        let Some(colon) = line.iter().position(|b| *b == b':') else {
            return Err(Admission::Invalid);
        };
        let (name, value) = (&line[..colon], &line[colon + 1..]);
        if name.eq_ignore_ascii_case(b"origin") {
            if let Ok(text) = std::str::from_utf8(value.trim_ascii()) {
                if !text.is_empty() {
                    origin = text.to_string();
                }
            }
        } else if name.eq_ignore_ascii_case(b"content-length") {
            content_lengths.push(
                std::str::from_utf8(value.trim_ascii())
                    .map_err(|_| Admission::Invalid)?
                    .to_string(),
            );
        } else if name.eq_ignore_ascii_case(b"transfer-encoding")
            && value.trim_ascii().eq_ignore_ascii_case(b"chunked")
        {
            chunked = true;
        }
    }
    if content_lengths.len() > 1 || chunked {
        return Err(Admission::Invalid);
    }
    let content_length = content_lengths
        .first()
        .map(|v| v.parse::<usize>().map_err(|_| Admission::Invalid))
        .transpose()?
        .unwrap_or(0);
    stream
        .set_read_timeout(Some(BODY_TIMEOUT))
        .map_err(|_| Admission::Invalid)?;
    let mut body = vec![0u8; content_length.min(MAX_BODY + 1)];
    stream
        .read_exact(&mut body)
        .map_err(|_| Admission::Invalid)?;
    if content_length > MAX_BODY {
        return Err(Admission::Invalid);
    }
    Ok((
        HttpRequest {
            method,
            path,
            query,
            origin,
        },
        body,
    ))
}

pub(crate) fn percent_decode(input: &str) -> Result<String, String> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err("invalid percent encoding".to_string());
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| "invalid percent encoding".to_string())?;
            out.push(
                u8::from_str_radix(hex, 16).map_err(|_| "invalid percent encoding".to_string())?,
            );
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| "invalid percent encoding".to_string())
}

pub(crate) fn success_response(output: &Option<Vec<u8>>) -> Vec<u8> {
    let mut body = match output {
        Some(bytes) => bytes.clone(),
        None => b"{}".to_vec(),
    };
    body.push(b'\n');
    let mut response = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    )
    .into_bytes();
    response.extend_from_slice(&body);
    response
}

pub(crate) fn error_response(status: u16, code: &str) -> Vec<u8> {
    let reason = match status {
        400 => "Bad Request",
        403 => "Forbidden",
        404 => "Not Found",
        409 => "Conflict",
        431 => "Request Header Fields Too Large",
        _ => "Internal Server Error",
    };
    let body = format!("{code}\n");
    let mut response = format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: text/plain; charset=utf-8\r\nX-Content-Type-Options: nosniff\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    )
    .into_bytes();
    response.extend_from_slice(body.as_bytes());
    response
}

#[cfg(test)]
#[path = "http_tests.rs"]
mod tests;
