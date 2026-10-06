// -- response envelopes --

fn reason(status: u16) -> &'static str {
    match status {
        101 => "Switching Protocols",
        400 => "Bad Request",
        404 => "Not Found",
        405 => "Method Not Allowed",
        409 => "Conflict",
        413 => "Content Too Large",
        422 => "Unprocessable Entity",
        431 => "Request Header Fields Too Large",
        500 => "Internal Server Error",
        501 => "Not Implemented",
        502 => "Bad Gateway",
        503 => "Service Unavailable",
        _ => "Internal Server Error",
    }
}

/// Go `http.Error` envelope: text/plain + nosniff + `{message}\n`.
/// Exported for the server's parser-level rejections (same envelope).
pub fn error_response(status: u16, message: &str) -> Vec<u8> {
    let mut body = message.as_bytes().to_vec();
    body.push(b'\n');
    let mut head = format!(
        "HTTP/1.1 {status} {}\r\nContent-Type: text/plain; charset=utf-8\r\nX-Content-Type-Options: nosniff\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        reason(status),
        body.len(),
    )
    .into_bytes();
    head.extend_from_slice(&body);
    head
}

pub fn not_found_response() -> Vec<u8> {
    error_response(404, "404 page not found")
}

/// Native/identity JSON success: application/json + trailing `\n`.
pub fn json_response(json: &[u8]) -> Vec<u8> {
    let mut head = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        json.len() + 1,
    )
    .into_bytes();
    head.extend_from_slice(json);
    head.push(b'\n');
    head
}

/// Tailnet JSON success: no trailing newline (writeTailnetResponse);
/// `Cache-Control: no-store` is injected by `tailnet_response`.
pub fn tailnet_json_response(json: &[u8]) -> Vec<u8> {
    let mut head = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        json.len(),
    )
    .into_bytes();
    head.extend_from_slice(json);
    head
}

/// Inject `Cache-Control: no-store` into a rendered tailnet error response
/// (Go sets the header before any tailnet error write).
pub fn tailnet_response(mut response: Vec<u8>) -> Vec<u8> {
    const NO_STORE: &[u8] = b"Cache-Control: no-store\r\n";
    if let Some(end) = find_header_end(&response) {
        let mut out = Vec::with_capacity(response.len() + NO_STORE.len());
        out.extend_from_slice(&response[..end]);
        out.extend_from_slice(NO_STORE);
        out.extend_from_slice(&response[end..]);
        std::mem::swap(&mut response, &mut out);
    }
    response
}

fn find_header_end(response: &[u8]) -> Option<usize> {
    response
        .windows(2)
        .position(|w| w == b"\r\n")
        .map(|first_end| first_end + 2)
}
