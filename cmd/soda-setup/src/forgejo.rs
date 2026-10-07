use std::io::Read;
use std::time::Duration;

use crate::system::{FORGEJO_RESPONSE_LIMIT, FORGEJO_TIMEOUT};
use serde::Deserialize;

#[derive(Debug)]
pub(crate) struct ForgejoUser {
    pub(crate) id: i64,
    pub(crate) admin: bool,
}

#[derive(Debug, Default)]
struct ForgejoUserResponse {
    id: i64,
    admin: bool,
}

impl<'de> Deserialize<'de> for ForgejoUserResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        use serde::de::{IgnoredAny, MapAccess, Visitor};
        use std::fmt;

        struct UserVisitor;

        impl<'de> Visitor<'de> for UserVisitor {
            type Value = ForgejoUserResponse;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("a Forgejo user object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut user = ForgejoUserResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "id" => user.id = map.next_value()?,
                        "is_admin" => user.admin = map.next_value()?,
                        _ => {
                            map.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(user)
            }
        }

        deserializer.deserialize_map(UserVisitor)
    }
}

pub(crate) fn forgejo_get_user(base: &str, token: &str) -> Result<ForgejoUser, String> {
    let (status, body) = forgejo_request(base, "GET", "/api/v1/user", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    decode_user(&body)
}

pub(crate) fn forgejo_revoke_token(base: &str, token: &str) -> Result<(), String> {
    let (status, _) = forgejo_request(base, "DELETE", "/api/v1/user/token", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    Ok(())
}

fn forgejo_request(
    base: &str,
    method: &str,
    path: &str,
    token: &str,
) -> Result<(u16, Vec<u8>), String> {
    forgejo_request_with_timeout(base, method, path, token, FORGEJO_TIMEOUT)
}

fn forgejo_request_with_timeout(
    base: &str,
    method: &str,
    path: &str,
    token: &str,
    timeout: Duration,
) -> Result<(u16, Vec<u8>), String> {
    let base = base.trim_end_matches('/');
    let url = format!("{base}{path}");
    let agent = ureq::AgentBuilder::new()
        .timeout(timeout)
        .redirects(0)
        .try_proxy_from_env(false)
        .build();
    let response = match agent
        .request(method, &url)
        .set("Accept", "application/json")
        .set("Content-Type", "application/json")
        .set("Authorization", &format!("token {token}"))
        .call()
    {
        Ok(response) => response,
        Err(ureq::Error::Status(_, response)) => response,
        Err(ureq::Error::Transport(_)) => {
            return Err("forgejo could not be reached".to_string());
        }
    };
    let status = response.status();
    let mut body = Vec::new();
    response
        .into_reader()
        .take((FORGEJO_RESPONSE_LIMIT + 1) as u64)
        .read_to_end(&mut body)
        .map_err(|_| "forgejo could not be reached".to_string())?;
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    Ok((status, body))
}

// decode_user mirrors the forgejo client's decodeResponse into User: the body
// must be one JSON value within the size bound, "null" is invalid, and id /
// is_admin keep Go's strict scalar types (unknown fields allowed).
pub(crate) fn decode_user(body: &[u8]) -> Result<ForgejoUser, String> {
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "invalid Forgejo response".to_string())?;
    if text.trim() == "null" {
        return Err("invalid Forgejo response".to_string());
    }
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let user = ForgejoUserResponse::deserialize(&mut deserializer)
        .map_err(|_| "invalid Forgejo response".to_string())?;
    deserializer
        .end()
        .map_err(|_| "invalid Forgejo response".to_string())?;
    Ok(ForgejoUser {
        id: user.id,
        admin: user.admin,
    })
}

#[cfg(test)]
mod http_tests {
    use super::*;
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::thread::{self, JoinHandle};
    use std::time::Duration;

    fn read_request(stream: &mut std::net::TcpStream) -> Vec<u8> {
        let mut request = Vec::new();
        let mut byte = [0u8; 1];
        while !request.ends_with(b"\r\n\r\n") {
            stream.read_exact(&mut byte).expect("read request");
            request.push(byte[0]);
            assert!(request.len() <= 65536, "request headers bounded");
        }
        request
    }

    fn serve(response: &'static [u8]) -> (String, JoinHandle<()>) {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture");
        let address = listener.local_addr().expect("fixture address");
        let handle = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept fixture");
            stream
                .set_read_timeout(Some(Duration::from_secs(2)))
                .expect("request timeout");
            stream
                .set_write_timeout(Some(Duration::from_secs(2)))
                .expect("response timeout");
            read_request(&mut stream);
            stream.write_all(response).expect("write response");
        });
        (format!("http://{address}"), handle)
    }

    #[test]
    fn request_reads_chunked_and_eof_bodies() {
        let (chunked, server) = serve(
            b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n2\r\nok\r\n0\r\n\r\n",
        );
        assert_eq!(
            forgejo_request(&chunked, "GET", "/api/v1/user", "secret").unwrap(),
            (200, b"ok".to_vec())
        );
        server.join().expect("chunked fixture joined");

        let (eof, server) = serve(b"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\nend");
        assert_eq!(
            forgejo_request(&eof, "GET", "/api/v1/user", "secret").unwrap(),
            (200, b"end".to_vec())
        );
        server.join().expect("EOF fixture joined");
    }

    #[test]
    fn request_returns_error_status_body_and_does_not_follow_redirects() {
        let (denied, server) = serve(
            b"HTTP/1.1 403 Forbidden\r\nContent-Length: 6\r\nConnection: close\r\n\r\ndenied",
        );
        assert_eq!(
            forgejo_request(&denied, "GET", "/api/v1/user", "secret").unwrap(),
            (403, b"denied".to_vec())
        );
        server.join().expect("denied fixture joined");

        let (redirect, server) = serve(b"HTTP/1.1 302 Found\r\nLocation: http://127.0.0.1:1/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n");
        assert_eq!(
            forgejo_request(&redirect, "GET", "/api/v1/user", "secret").unwrap(),
            (302, Vec::new())
        );
        server.join().expect("redirect fixture joined");
    }

    #[test]
    fn request_sends_the_literal_authorization_scheme() {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture");
        let address = listener.local_addr().expect("fixture address");
        let handle = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept request");
            stream
                .set_read_timeout(Some(Duration::from_secs(2)))
                .expect("request timeout");
            stream
                .set_write_timeout(Some(Duration::from_secs(2)))
                .expect("response timeout");
            let request = read_request(&mut stream);
            stream
                .write_all(
                    b"HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
                )
                .expect("write response");
            String::from_utf8(request).expect("request headers")
        });
        let url = format!("http://{address}");
        assert_eq!(
            forgejo_request(&url, "DELETE", "/api/v1/user/token", "fixture-token").unwrap(),
            (204, Vec::new())
        );
        let request = handle.join().expect("authorization fixture joined");
        assert!(request.contains("Authorization: token fixture-token\r\n"));
        assert!(!url.contains("fixture-token"));
    }

    #[test]
    fn request_caps_success_and_error_status_bodies() {
        for status in [200, 403] {
            let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture");
            let address = listener.local_addr().expect("fixture address");
            let handle = thread::spawn(move || {
                let (mut stream, _) = listener.accept().expect("accept request");
                stream
                    .set_read_timeout(Some(Duration::from_secs(2)))
                    .expect("request timeout");
                stream
                    .set_write_timeout(Some(Duration::from_secs(2)))
                    .expect("response timeout");
                read_request(&mut stream);
                write!(
                    stream,
                    "HTTP/1.1 {status} Test\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
                    FORGEJO_RESPONSE_LIMIT + 1
                )
                .expect("write headers");
                stream
                    .write_all(&vec![b'x'; FORGEJO_RESPONSE_LIMIT + 1])
                    .expect("write body");
            });
            let url = format!("http://{address}");
            assert_eq!(
                forgejo_request(&url, "GET", "/api/v1/user", "secret").unwrap_err(),
                "forgejo response exceeds the supported size"
            );
            handle.join().expect("oversized fixture joined");
        }
    }

    #[test]
    fn request_rejects_malformed_and_truncated_responses_neutrally() {
        let (malformed, server) = serve(b"not an HTTP response\r\n\r\n");
        assert_eq!(
            forgejo_request(&malformed, "GET", "/api/v1/user", "private-token").unwrap_err(),
            "forgejo could not be reached"
        );
        server.join().expect("malformed fixture joined");

        let (truncated, server) =
            serve(b"HTTP/1.1 200 OK\r\nContent-Length: 9\r\nConnection: close\r\n\r\nshort");
        let error =
            forgejo_request(&truncated, "GET", "/api/v1/user", "private-token").unwrap_err();
        assert_eq!(error, "forgejo could not be reached");
        assert!(!error.contains("private-token"));
        server.join().expect("truncated fixture joined");

        let (malformed_chunk, server) = serve(
            b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\nnope\r\n",
        );
        let error =
            forgejo_request(&malformed_chunk, "GET", "/api/v1/user", "private-token").unwrap_err();
        assert_eq!(error, "forgejo could not be reached");
        assert!(!error.contains("private-token"));
        server.join().expect("malformed chunk fixture joined");

        let error = forgejo_request(
            "http://127.0.0.1:1",
            "GET",
            "/api/v1/user",
            "fixture-token\r\nInjected: private-token",
        )
        .unwrap_err();
        assert_eq!(error, "forgejo could not be reached");
        assert!(!error.contains("private-token"));
    }

    #[test]
    fn request_timeout_covers_headers_and_body_together() {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture");
        let address = listener.local_addr().expect("fixture address");
        let handle = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept request");
            stream
                .set_read_timeout(Some(Duration::from_secs(2)))
                .expect("request timeout");
            stream
                .set_write_timeout(Some(Duration::from_secs(2)))
                .expect("response timeout");
            read_request(&mut stream);
            thread::sleep(Duration::from_millis(70));
            let _ = stream
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\n");
            thread::sleep(Duration::from_millis(70));
            let _ = stream.write_all(b"ok");
        });
        let url = format!("http://{address}");
        let error = forgejo_request_with_timeout(
            &url,
            "GET",
            "/api/v1/user",
            "secret",
            Duration::from_millis(110),
        )
        .unwrap_err();
        assert_eq!(error, "forgejo could not be reached");
        handle.join().expect("delayed response fixture joined");
    }

    #[test]
    fn request_rejects_plaintext_from_https_peer_neutrally() {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind fixture");
        let address = listener.local_addr().expect("fixture address");
        let handle = thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept TLS request");
            stream
                .set_read_timeout(Some(Duration::from_secs(2)))
                .expect("TLS read timeout");
            stream
                .set_write_timeout(Some(Duration::from_secs(2)))
                .expect("TLS write timeout");
            let mut hello = [0u8; 5];
            let _ = stream.read(&mut hello);
            let _ = stream.write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok");
        });
        let url = format!("https://{address}");
        let error = forgejo_request_with_timeout(
            &url,
            "GET",
            "/api/v1/user",
            "secret",
            Duration::from_millis(300),
        )
        .unwrap_err();
        assert_eq!(error, "forgejo could not be reached");
        handle.join().expect("plaintext TLS fixture joined");
    }
}
