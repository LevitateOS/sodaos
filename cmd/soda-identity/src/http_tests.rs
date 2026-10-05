use super::*;

#[test]
fn percent_paths_decode() {
    assert_eq!(percent_decode("/acquire").unwrap(), "/acquire");
    assert_eq!(percent_decode("/%61cquire").unwrap(), "/acquire");
    assert!(percent_decode("/%zz").is_err());
    assert!(percent_decode("/%4").is_err());
}

#[test]
fn error_bodies_match_go() {
    let response = error_response(403, "denied");
    let text = String::from_utf8(response).unwrap();
    assert!(text.starts_with("HTTP/1.1 403 Forbidden\r\n"));
    assert!(text.ends_with("\r\n\r\ndenied\n"));
    assert!(text.contains("Content-Type: text/plain; charset=utf-8"));
}

#[test]
fn success_envelope_matches_go() {
    let response = success_response(&None);
    let text = String::from_utf8(response).unwrap();
    assert!(text.contains("Content-Type: application/json"));
    assert!(text.contains("Cache-Control: no-store"));
    assert!(text.ends_with("\r\n\r\n{}\n"));
}
