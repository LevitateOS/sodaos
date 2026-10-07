// HTTP admission policy and typed response envelopes. Hyper owns wire framing.
use bytes::Bytes;
use http_body_util::Full;
use hyper::header::{HeaderValue, CACHE_CONTROL, CONTENT_TYPE, X_CONTENT_TYPE_OPTIONS};
use hyper::{Response, StatusCode};

pub(crate) const MAX_BODY: usize = 512 << 10;
pub(crate) const MAX_HEADER: usize = 8192;

pub(crate) struct HttpRequest {
    pub(crate) method: String,
    pub(crate) path: String,
    pub(crate) query: String,
    pub(crate) origin: String,
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

pub(crate) fn success_response(output: &Option<Vec<u8>>) -> Response<Full<Bytes>> {
    let mut body = output.clone().unwrap_or_else(|| b"{}".to_vec());
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
