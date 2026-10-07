use bytes::Bytes;
use http_body_util::Full;
use hyper::{Response, StatusCode};

pub type HttpResponse = Response<Full<Bytes>>;

fn response(status: u16, content_type: &'static str, body: Vec<u8>) -> HttpResponse {
    let mut out = Response::new(Full::new(Bytes::from(body)));
    *out.status_mut() = StatusCode::from_u16(status).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
    out.headers_mut().insert(
        hyper::header::CONTENT_TYPE,
        content_type.parse().expect("static content type"),
    );
    out
}

pub fn error_response(status: u16, message: &str) -> HttpResponse {
    let mut body = message.as_bytes().to_vec();
    body.push(b'\n');
    let mut out = response(status, "text/plain; charset=utf-8", body);
    out.headers_mut().insert(
        "x-content-type-options",
        "nosniff".parse().expect("static header"),
    );
    out
}

pub fn not_found_response() -> HttpResponse {
    error_response(404, "404 page not found")
}

pub fn json_response(json: &[u8]) -> HttpResponse {
    let mut body = json.to_vec();
    body.push(b'\n');
    response(200, "application/json", body)
}

pub fn tailnet_json_response(json: &[u8]) -> HttpResponse {
    let mut out = response(200, "application/json", json.to_vec());
    out.headers_mut()
        .insert("cache-control", "no-store".parse().expect("static header"));
    out
}

pub fn tailnet_response(mut response: HttpResponse) -> HttpResponse {
    response
        .headers_mut()
        .insert("cache-control", "no-store".parse().expect("static header"));
    response
}
