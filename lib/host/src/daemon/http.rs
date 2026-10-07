use crate::gmux_admission::{body_limit_for, RequestHead};
use hyper::{body::Incoming, Request};
use percent_encoding::percent_decode_str;
use std::time::Duration;

pub const MAX_HEADER: usize = 8192;
pub const HEADER_TIMEOUT: Duration = Duration::from_secs(5);
pub const BODY_TIMEOUT: Duration = Duration::from_secs(30);

pub fn request_head(request: &Request<Incoming>) -> Result<RequestHead, ()> {
    let uri = request.uri();
    let target = uri
        .path_and_query()
        .map(|p| p.as_str())
        .unwrap_or(uri.path());
    let raw_path = target.split('?').next().unwrap_or(target);
    let path = percent_decode(raw_path)?;
    let headers = request.headers();
    let upgrade_websocket = headers
        .get_all(hyper::header::UPGRADE)
        .iter()
        .flat_map(|v| v.as_bytes().split(|b| *b == b','))
        .any(|v| v.trim_ascii().eq_ignore_ascii_case(b"websocket"));
    let ws_key = headers
        .get(hyper::header::SEC_WEBSOCKET_KEY)
        .and_then(|v| v.to_str().ok())
        .map(str::to_owned);
    let mut websocket_request = Request::builder()
        .method(request.method().clone())
        .uri(uri.clone())
        .version(request.version())
        .body(())
        .map_err(|_| ())?;
    *websocket_request.headers_mut() = headers.clone();
    Ok(RequestHead {
        method: request.method().as_str().to_owned(),
        path,
        has_query: target.contains('?'),
        escaped: raw_path.contains('%'),
        origin_present: headers.contains_key(hyper::header::ORIGIN),
        upgrade_websocket,
        ws_key,
        websocket_request: Some(websocket_request),
    })
}

pub async fn read_body(mut body: Incoming, path: &str) -> Result<(Vec<u8>, bool), ()> {
    use http_body_util::BodyExt;
    let limit = body_limit_for(path);
    let deadline = tokio::time::Instant::now() + BODY_TIMEOUT;
    let mut bytes = Vec::with_capacity(limit.min(8192));
    loop {
        let remaining = deadline.saturating_duration_since(tokio::time::Instant::now());
        if remaining.is_zero() {
            return Err(());
        }
        let frame = tokio::time::timeout(remaining, body.frame())
            .await
            .map_err(|_| ())?
            .transpose()
            .map_err(|_| ())?;
        let Some(frame) = frame else {
            return Ok((bytes, true));
        };
        if let Ok(data) = frame.into_data() {
            let remaining = limit.saturating_add(1).saturating_sub(bytes.len());
            bytes.extend_from_slice(&data[..data.len().min(remaining)]);
            if bytes.len() > limit {
                return Ok((bytes, false));
            }
        }
    }
}

fn percent_decode(input: &str) -> Result<String, ()> {
    if !valid_percent_escapes(input) {
        return Err(());
    }
    percent_decode_str(input)
        .decode_utf8()
        .map(|decoded| decoded.into_owned())
        .map_err(|_| ())
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
