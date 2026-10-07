// HTTP admission policy and typed response envelopes. Hyper owns wire framing.
use bytes::Bytes;
use http_body_util::Full;
use hyper::header::{HeaderValue, CACHE_CONTROL, CONTENT_TYPE, X_CONTENT_TYPE_OPTIONS};
use hyper::{Response, StatusCode};
use percent_encoding::percent_decode_str;

pub(crate) const MAX_BODY: usize = 512 << 10;
pub(crate) const MAX_HEADER: usize = 8192;
const MAX_SUCCESS_BODY: usize = 512 << 10;

pub(crate) struct HttpRequest {
    pub(crate) method: String,
    pub(crate) path: String,
    pub(crate) query: String,
    pub(crate) origin: String,
}

pub(crate) fn percent_decode(input: &str) -> Result<String, String> {
    if !valid_percent_escapes(input) {
        return Err("invalid percent encoding".to_string());
    }
    percent_decode_str(input)
        .decode_utf8()
        .map(|decoded| decoded.into_owned())
        .map_err(|_| "invalid percent encoding".to_string())
}

fn valid_percent_escapes(input: &str) -> bool {
    let bytes = input.as_bytes();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at] == b'%' {
            if at + 2 >= bytes.len()
                || !bytes[at + 1].is_ascii_hexdigit()
                || !bytes[at + 2].is_ascii_hexdigit()
            {
                return false;
            }
            at += 3;
        } else {
            at += 1;
        }
    }
    true
}

pub(crate) fn success_response(output: Option<Vec<u8>>) -> Response<Full<Bytes>> {
    let mut body = output.unwrap_or_else(|| b"{}".to_vec());
    if body.len() >= MAX_SUCCESS_BODY {
        return error_response(500, "unavailable");
    }
    body.push(b'\n');
    let mut response = Response::new(Full::new(Bytes::from(body)));
    *response.status_mut() = StatusCode::OK;
    let headers = response.headers_mut();
    headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
    headers.insert(CACHE_CONTROL, HeaderValue::from_static("no-store"));
    response
}

pub(crate) fn error_response(status: u16, code: &str) -> Response<Full<Bytes>> {
    let status = StatusCode::from_u16(status).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
    let mut response = Response::new(Full::new(Bytes::from(format!("{code}\n"))));
    *response.status_mut() = status;
    let headers = response.headers_mut();
    headers.insert(
        CONTENT_TYPE,
        HeaderValue::from_static("text/plain; charset=utf-8"),
    );
    headers.insert(X_CONTENT_TYPE_OPTIONS, HeaderValue::from_static("nosniff"));
    response
}

#[cfg(test)]
#[path = "http_tests.rs"]
mod tests;
