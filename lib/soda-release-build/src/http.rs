//! Bounded HTTPS fetching (`coreos.go` client, `coreos_stream.go`
//! `cappedClient`): strict redirect policy over an injectable transport.
//!
//! The redirect loop mirrors Go's `CheckRedirect`: at most five follows,
//! every hop a strict HTTPS URL. Transports perform single requests with
//! redirects disabled; tests substitute stubs, exactly like the Go owner's
//! transport swap.

use crate::coreos::https_url;
use crate::Error;
use std::io::Read;
use std::time::Duration;

/// One completed single request: status plus an owned body reader.
pub struct HttpResponse {
    pub status: u16,
    pub status_text: String,
    pub location: Option<String>,
    pub body: Box<dyn Read>,
}

impl HttpResponse {
    /// Go `resp.Status` shape: `<code> <reason>`.
    pub fn status_line(&self) -> String {
        format!("{} {}", self.status, self.status_text)
    }
}

impl std::fmt::Debug for HttpResponse {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("HttpResponse")
            .field("status", &self.status)
            .field("status_text", &self.status_text)
            .field("location", &self.location)
            .finish_non_exhaustive()
    }
}

/// Single-request transport with redirects disabled.
pub trait HttpTransport {
    fn get(
        &self,
        url: &str,
        accept: Option<&str>,
        timeout: Duration,
    ) -> Result<HttpResponse, Error>;
}

/// Production transport over ureq with rustls roots.
pub struct UreqTransport;

impl HttpTransport for UreqTransport {
    fn get(
        &self,
        url: &str,
        accept: Option<&str>,
        timeout: Duration,
    ) -> Result<HttpResponse, Error> {
        let agent = ureq::AgentBuilder::new()
            .redirects(0)
            .timeout(timeout)
            .build();
        let mut request = agent.get(url);
        if let Some(accept) = accept {
            request = request.set("Accept", accept);
        }
        match request.call() {
            Ok(response) => Ok(ureq_response(response)),
            Err(ureq::Error::Status(status, response)) => Ok(HttpResponse {
                status,
                status_text: response.status_text().to_string(),
                location: response.header("Location").map(str::to_string),
                body: Box::new(std::io::empty()),
            }),
            Err(ureq::Error::Transport(transport)) => Err(Error::msg(transport.to_string())),
        }
    }
}

fn ureq_response(response: ureq::Response) -> HttpResponse {
    HttpResponse {
        status: response.status(),
        status_text: response.status_text().to_string(),
        location: response.header("Location").map(str::to_string),
        body: Box::new(response.into_reader()),
    }
}

fn is_redirect(status: u16) -> bool {
    matches!(status, 301 | 302 | 303 | 307 | 308)
}

/// Splits `scheme://authority/path...` for redirect resolution.
fn split_url(url: &str) -> Option<(&str, &str, &str)> {
    let (scheme, rest) = url.split_once("://")?;
    let end = rest.find('/').unwrap_or(rest.len());
    Some((scheme, &rest[..end], &rest[end..]))
}

/// Resolves a `Location` value against the request URL, mirroring Go's
/// `ResolveReference` for the shapes this client follows. The strict HTTPS
/// gate re-validates the result; resolution itself never fails closed-open.
pub fn resolve_location(base: &str, location: &str) -> Option<String> {
    if location.contains([' ', '\t', '\r', '\n']) {
        return None;
    }
    if let Some((scheme, _, _)) = split_url(location) {
        if scheme.eq_ignore_ascii_case("http") || scheme.eq_ignore_ascii_case("https") {
            return Some(location.to_string());
        }
        return None;
    }
    // Reject anything with a scheme (mailto:, //-relative, etc.).
    if location.contains("://") || location.starts_with("//") {
        return None;
    }
    if let Some(prefix) = location.split_once(':').map(|(head, _)| head) {
        if !prefix.contains('/') && !prefix.is_empty() {
            return None;
        }
    }
    let (scheme, authority, path) = split_url(base)?;
    if let Some(absolute) = location.strip_prefix('/') {
        return Some(format!("{scheme}://{authority}/{absolute}"));
    }
    let dir = match path.rfind('/') {
        Some(i) => &path[..i + 1],
        None => "/",
    };
    Some(format!("{scheme}://{authority}{dir}{location}"))
}

/// GET with the Go owner's redirect policy: at most five follows, every
/// hop gated by strict HTTPS. `redirect_err` is the caller's message
/// (`unsafe download redirect` / `unsafe metadata redirect`).
pub fn get_follow<T: HttpTransport>(
    transport: &T,
    url: &str,
    accept: Option<&str>,
    timeout: Duration,
    redirect_err: &str,
) -> Result<HttpResponse, Error> {
    let mut current = url.to_string();
    for _ in 0..6 {
        let response = transport.get(&current, accept, timeout)?;
        if !is_redirect(response.status) {
            return Ok(response);
        }
        let location = response
            .location
            .clone()
            .filter(|h| !h.is_empty())
            .ok_or_else(|| Error::msg("http: no Location header in response"))?;
        drop(response);
        let next =
            resolve_location(&current, location.trim()).ok_or_else(|| Error::msg(redirect_err))?;
        if !soda_build_tools::reader::url::https_url(&next) {
            return Err(Error::msg(redirect_err));
        }
        current = next;
    }
    Err(Error::msg(redirect_err))
}

pub(crate) fn fetch_capped_json<T: HttpTransport>(
    transport: &T,
    url: &str,
    max_bytes: i64,
) -> Result<Vec<u8>, Error> {
    if !https_url(url) || max_bytes <= 0 {
        return Err(Error::msg("bounded HTTPS fetch required"));
    }
    let response = get_follow(
        transport,
        url,
        None,
        Duration::from_secs(60),
        "unsafe metadata redirect",
    )
    .map_err(|e| Error::msg(format!("live input fetch failed: {}", e.message())))?;
    if response.status != 200 {
        return Err(Error::msg(format!(
            "live input HTTP failure: {}",
            response.status_line()
        )));
    }
    let mut data = Vec::new();
    response
        .body
        .take((max_bytes + 1) as u64)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(e.to_string()))?;
    if data.len() as i64 > max_bytes {
        return Err(Error::msg("live input exceeds size limit"));
    }
    Ok(data)
}

pub(crate) fn fetch_capped_text<T: HttpTransport>(
    transport: &T,
    url: &str,
    max_bytes: i64,
) -> Result<String, Error> {
    Ok(String::from_utf8_lossy(&fetch_capped_json(transport, url, max_bytes)?).into_owned())
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use std::collections::HashMap;
    use std::sync::{Arc, Mutex};

    /// Stub transport: canned status + headers + body per path.
    pub struct Stub {
        pub routes: HashMap<String, (u16, Option<String>, Vec<u8>)>,
        pub seen_accept: Arc<Mutex<Vec<Option<String>>>>,
    }

    impl Stub {
        pub fn new(routes: &[(&str, u16, Option<&str>, &[u8])]) -> Stub {
            Stub {
                routes: routes
                    .iter()
                    .map(|(path, status, location, body)| {
                        (
                            (*path).to_string(),
                            (*status, location.map(str::to_string), body.to_vec()),
                        )
                    })
                    .collect(),
                seen_accept: Arc::new(Mutex::new(Vec::new())),
            }
        }
    }

    impl HttpTransport for Stub {
        fn get(
            &self,
            url: &str,
            accept: Option<&str>,
            _timeout: Duration,
        ) -> Result<HttpResponse, Error> {
            self.seen_accept
                .lock()
                .unwrap()
                .push(accept.map(str::to_string));
            let path = url.split_once("://").map(|(_, rest)| rest).unwrap_or(url);
            let path = path
                .split_once('/')
                .map(|(_, p)| format!("/{p}"))
                .unwrap_or_else(|| "/".to_string());
            match self.routes.get(&path) {
                Some((status, location, body)) => Ok(HttpResponse {
                    status: *status,
                    status_text: reason(*status).to_string(),
                    location: location.clone(),
                    body: Box::new(std::io::Cursor::new(body.clone())),
                }),
                None => Err(Error::msg(format!("stub has no route for {url}"))),
            }
        }
    }

    fn reason(status: u16) -> &'static str {
        match status {
            200 => "OK",
            301 => "Moved Permanently",
            302 => "Found",
            401 => "Unauthorized",
            404 => "Not Found",
            _ => "Status",
        }
    }

    #[test]
    fn oracle_redirect_policy_vectors() {
        // Oracle: Go CheckRedirect outcomes from TestCoreOSDownloadTLSRedirectBoundsAndNoOverwrite.
        let stub = Stub::new(&[
            ("/ok", 200, None, b"fixture"),
            ("/redirect", 302, Some("http://example.invalid/base"), b""),
            ("/query", 302, Some("/ok?token=must-not-follow"), b""),
            ("/loop", 302, Some("/loop"), b""),
            ("/missing", 404, None, b"nope"),
            ("/rel", 302, Some("/ok"), b""),
        ]);
        let base = "https://fixtures.test";
        for bad in ["/redirect", "/query", "/loop"] {
            let err = get_follow(
                &stub,
                &format!("{base}{bad}"),
                None,
                Duration::from_secs(5),
                "unsafe download redirect",
            )
            .unwrap_err();
            assert_eq!(err.message(), "unsafe download redirect", "{bad}");
        }
        // Relative same-origin redirect resolves and succeeds.
        let ok = get_follow(
            &stub,
            &format!("{base}/rel"),
            None,
            Duration::from_secs(5),
            "unsafe download redirect",
        )
        .unwrap();
        assert_eq!(ok.status, 200);
        // Non-redirect statuses pass through for the caller to judge.
        let missing = get_follow(
            &stub,
            &format!("{base}/missing"),
            None,
            Duration::from_secs(5),
            "unsafe download redirect",
        )
        .unwrap();
        assert_eq!(missing.status, 404);
    }

    #[test]
    fn oracle_location_resolution_vectors() {
        // Oracle: Go url.ResolveReference outcomes for redirect shapes.
        assert_eq!(
            resolve_location("https://h.test/a/b", "/ok").as_deref(),
            Some("https://h.test/ok")
        );
        assert_eq!(
            resolve_location("https://h.test/a/b", "c").as_deref(),
            Some("https://h.test/a/c")
        );
        assert_eq!(
            resolve_location("https://h.test/a/b", "https://o.test/x").as_deref(),
            Some("https://o.test/x")
        );
        assert_eq!(resolve_location("https://h.test/a/b", "//o.test/x"), None);
        assert_eq!(resolve_location("https://h.test/a/b", "mailto:x@y"), None);
        assert_eq!(resolve_location("https://h.test/a/b", "a b"), None);
    }
}
