use super::*;
use http_body_util::BodyExt;

#[test]
fn percent_paths_decode() {
    assert_eq!(percent_decode("/acquire").unwrap(), "/acquire");
    assert_eq!(percent_decode("/%61cquire").unwrap(), "/acquire");
    assert!(percent_decode("/%zz").is_err());
    assert!(percent_decode("/%4").is_err());
}

#[test]
fn error_envelope_preserves_status_headers_and_lf() {
    let response = error_response(403, "denied");
    assert_eq!(response.status(), hyper::StatusCode::FORBIDDEN);
    assert_eq!(
        response.headers()[hyper::header::CONTENT_TYPE],
        "text/plain; charset=utf-8"
    );
    assert_eq!(
        response.headers()[hyper::header::X_CONTENT_TYPE_OPTIONS],
        "nosniff"
    );
    let body = tokio::runtime::Builder::new_current_thread()
        .build()
        .unwrap()
        .block_on(response.into_body().collect())
        .unwrap()
        .to_bytes();
    assert_eq!(body, b"denied\n"[..]);
}

#[test]
fn success_envelope_preserves_cache_policy_and_lf() {
    let response = success_response(&None);
    assert_eq!(response.status(), hyper::StatusCode::OK);
    assert_eq!(
        response.headers()[hyper::header::CONTENT_TYPE],
        "application/json"
    );
    assert_eq!(response.headers()[hyper::header::CACHE_CONTROL], "no-store");
    let body = tokio::runtime::Builder::new_current_thread()
        .build()
        .unwrap()
        .block_on(response.into_body().collect())
        .unwrap()
        .to_bytes();
    assert_eq!(body, b"{}\n"[..]);
}
