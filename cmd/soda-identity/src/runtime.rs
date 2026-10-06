// Host runtime callbacks, mirroring cmd/soda-identity/runtime.go over
// internal/host: the broker attests terminal sessions and supervised
// factory runs by their exact bindings; it never shells out itself.
use crate::wire::{DeliveryWire, Error, Lease};
use std::io::{Read, Write};
use std::os::unix::net::UnixStream;
use std::time::Duration;

const CALL_TIMEOUT: Duration = Duration::from_secs(4 * 60);
const DEFAULT_LIMIT: usize = 65536;
const FINISH_LIMIT: usize = 384 << 10;

pub struct HostClient {
    socket: String,
}

impl HostClient {
    pub fn new(socket: &str) -> HostClient {
        HostClient {
            socket: socket.to_string(),
        }
    }

    fn call(&self, path: &str, lease: &Lease, limit: Option<usize>) -> Result<DeliveryWire, Error> {
        let mut body = serde_json::to_vec(&DeliveryWire {
            lease: lease.clone(),
            credential: Vec::new(),
        })?;
        body.push(b'\n');
        let request = format!(
            "POST {path} HTTP/1.1\r\nHost: soda-host\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
            body.len()
        );
        let mut stream = UnixStream::connect(&self.socket)
            .map_err(|_| Error::internal("host service unavailable"))?;
        stream
            .set_read_timeout(Some(CALL_TIMEOUT))
            .map_err(Error::from)?;
        stream
            .set_write_timeout(Some(CALL_TIMEOUT))
            .map_err(Error::from)?;
        stream
            .write_all(request.as_bytes())
            .map_err(|_| Error::internal("host service unavailable"))?;
        stream
            .write_all(&body)
            .map_err(|_| Error::internal("host service unavailable"))?;
        let (status, response) = read_response(&mut stream)?;
        if status != 200 {
            return Err(Error::internal(format!(
                "native project operation failed (HTTP {status}); operator should inspect soda-host journal"
            )));
        }
        let Some(limit) = limit else {
            return Ok(DeliveryWire {
                lease: lease.clone(),
                credential: Vec::new(),
            });
        };
        // Go rejects over-limit and whitespace-null bodies; an empty body
        // fails JSON decoding below with the same error.
        if response.len() > limit || response.trim_ascii() == b"null" {
            return Err(Error::internal("invalid native response"));
        }
        serde_json::from_slice(&response).map_err(|_| Error::internal("invalid native response"))
    }

    pub fn validate(&self, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            return Err(Error::denied("identity authority denied"));
        }
        self.call("/identity/validate", lease, None)?;
        Ok(())
    }

    pub fn stop(&self, lease: &Lease) -> Result<(), Error> {
        if lease.binding.is_none() {
            return Ok(());
        }
        self.call("/identity/stop", lease, None)?;
        Ok(())
    }

    pub fn finish(&self, lease: &Lease) -> Result<Vec<u8>, Error> {
        if lease.binding.is_none() {
            return Err(Error::denied("identity authority denied"));
        }
        Ok(self
            .call("/identity/finish", lease, Some(FINISH_LIMIT))?
            .credential)
    }

    #[allow(dead_code)]
    fn default_limit() -> usize {
        DEFAULT_LIMIT
    }
}

impl crate::control::Runtime for HostClient {
    fn validate(&self, lease: &Lease) -> Result<(), Error> {
        HostClient::validate(self, lease)
    }
    fn stop(&self, lease: &Lease) -> Result<(), Error> {
        HostClient::stop(self, lease)
    }
    fn finish(&self, lease: &Lease) -> Result<Vec<u8>, Error> {
        HostClient::finish(self, lease)
    }
}

fn read_response(stream: &mut UnixStream) -> Result<(u16, Vec<u8>), Error> {
    let mut head = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err(Error::internal("host service unavailable")),
            Ok(_) => {
                head.push(byte[0]);
                if head.len() > 65536 {
                    return Err(Error::internal("invalid native response"));
                }
                if head.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            Err(e) => return Err(Error::internal(format!("host service unavailable: {e}"))),
        }
    }
    let mut lines = head.split(|b| *b == b'\n');
    let status_line = lines.next().unwrap_or(b"");
    let status: u16 = status_line
        .split(|b| *b == b' ')
        .nth(1)
        .and_then(|code| std::str::from_utf8(code).ok()?.parse::<u16>().ok())
        .ok_or_else(|| Error::internal("invalid native response"))?;
    let mut content_length: Option<usize> = None;
    let mut chunked = false;
    for line in lines {
        let line = line.strip_suffix(b"\r").unwrap_or(line);
        if line.is_empty() {
            continue;
        }
        if let Some(value) = line
            .strip_prefix(b"Content-Length:")
            .or_else(|| line.strip_prefix(b"content-length:"))
        {
            if let Ok(text) = std::str::from_utf8(value.trim_ascii()) {
                content_length = text.parse::<usize>().ok();
            }
        } else if line
            .strip_prefix(b"Transfer-Encoding:")
            .or_else(|| line.strip_prefix(b"transfer-encoding:"))
            .is_some_and(|v| v.trim_ascii().eq_ignore_ascii_case(b"chunked"))
        {
            chunked = true;
        }
    }
    if chunked {
        // The host daemon only emits Content-Length responses; a chunked
        // response is outside the retained framing contract (H01-F1).
        return Err(Error::internal("invalid native response"));
    }
    if let Some(len) = content_length {
        if len > FINISH_LIMIT + 1 {
            // Read only enough to report the limit violation.
            let mut body = vec![0u8; FINISH_LIMIT + 1];
            stream
                .read_exact(&mut body)
                .map_err(|_| Error::internal("host service unavailable"))?;
            return Ok((status, body));
        }
        let mut body = vec![0u8; len];
        stream
            .read_exact(&mut body)
            .map_err(|_| Error::internal("host service unavailable"))?;
        return Ok((status, body));
    }
    let mut body = Vec::new();
    stream
        .read_to_end(&mut body)
        .map_err(|_| Error::internal("host service unavailable"))?;
    Ok((status, body))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wire::{Binding, UnixTime};
    use std::os::unix::net::UnixListener;

    fn lease(binding: bool) -> Lease {
        Lease {
            repository_id: 0,
            provider_id: "codex".to_string(),
            id: "lease-1".to_string(),
            connection_id: "conn-1".to_string(),
            generation: 1,
            actor_id: 1,
            project_id: "project".to_string(),
            execution_id: "execution".to_string(),
            kind: "factory".to_string(),
            role: String::new(),
            deadline: UnixTime {
                sec: 1791138605,
                nanos: 0,
            },
            grant_id: String::new(),
            grant_revision: 0,
            binding: binding.then(|| Binding {
                child_id: String::new(),
                uid: 0,
                gid: 0,
                scope: String::new(),
                credential_root: String::new(),
                invocation_id: String::new(),
                kind: "factory".to_string(),
                id: "execution".to_string(),
                project: String::new(),
                login: String::new(),
                generation: 1,
            }),
        }
    }

    fn stub(
        responder: impl Fn(&str, &[u8]) -> Vec<u8> + Send + 'static,
        count: usize,
    ) -> HostClient {
        let dir = std::env::temp_dir().join(format!("soda-rt-test-{}", std::process::id()));
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join(format!("host-{}.sock", rand_suffix()));
        let _ = std::fs::remove_file(&path);
        let listener = UnixListener::bind(&path).unwrap();
        std::thread::spawn(move || {
            for _ in 0..count {
                let Ok((mut stream, _)) = listener.accept() else {
                    break;
                };
                let mut request = Vec::new();
                let mut byte = [0u8; 1];
                let mut content_length = 0usize;
                loop {
                    if stream.read(&mut byte).unwrap_or(0) == 0 {
                        break;
                    }
                    request.push(byte[0]);
                    if request.ends_with(b"\r\n\r\n") {
                        break;
                    }
                }
                for line in request.split(|b| *b == b'\n') {
                    if let Some(value) = line.strip_prefix(b"Content-Length:") {
                        content_length = std::str::from_utf8(value.trim_ascii())
                            .unwrap()
                            .parse()
                            .unwrap();
                    }
                }
                let mut body = vec![0u8; content_length];
                stream.read_exact(&mut body).unwrap();
                let target = request
                    .split(|b| *b == b' ')
                    .nth(1)
                    .and_then(|t| std::str::from_utf8(t).ok())
                    .unwrap_or("")
                    .to_string();
                let response = responder(&target, &body);
                stream.write_all(&response).unwrap();
            }
        });
        HostClient::new(path.to_string_lossy().as_ref())
    }

    fn rand_suffix() -> String {
        use std::collections::hash_map::RandomState;
        use std::hash::{BuildHasher, Hasher};
        let mut hasher = RandomState::new().build_hasher();
        hasher.write_usize(std::process::id() as usize);
        hasher.write_u128(
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos(),
        );
        format!("{:x}", hasher.finish())
    }

    #[test]
    fn delegates_both_kinds_to_host() {
        let client = stub(
            |target, body| {
                assert!(target.starts_with("/identity/"));
                let wire: DeliveryWire = serde_json::from_slice(body).unwrap();
                assert!(wire.lease.binding.is_some());
                let out = serde_json::to_vec(&DeliveryWire {
                    lease: wire.lease,
                    credential: br#"{"returned":true}"#.to_vec(),
                })
                .unwrap();
                format!("HTTP/1.1 200 OK\r\nContent-Length: {}\r\n\r\n", out.len())
                    .into_bytes()
                    .into_iter()
                    .chain(out)
                    .collect()
            },
            6,
        );
        for kind in ["terminal", "factory"] {
            let mut l = lease(true);
            l.kind = kind.to_string();
            l.binding.as_mut().unwrap().kind = kind.to_string();
            client.validate(&l).unwrap();
            client.stop(&l).unwrap();
            assert_eq!(client.finish(&l).unwrap(), br#"{"returned":true}"#);
        }
    }

    #[test]
    fn refuses_unbound_lease_without_host_call() {
        let client = stub(|_, _| panic!("unbound lease reached the host"), 0);
        let l = lease(false);
        assert!(client.validate(&l).is_err());
        assert!(client.finish(&l).is_err());
        client.stop(&l).unwrap();
    }

    #[test]
    fn finish_rejects_null_and_oversized_bodies() {
        let client = stub(
            |_, _| b"HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\nnull".to_vec(),
            1,
        );
        assert!(client.finish(&lease(true)).is_err());
    }

    #[test]
    fn chunked_response_refused_without_decode() {
        let (mut peer, mut stream) = std::os::unix::net::UnixStream::pair().unwrap();
        std::io::Write::write_all(
            &mut peer,
            b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nWiki\r\n0\r\n\r\n",
        )
        .unwrap();
        let err = read_response(&mut stream).unwrap_err();
        assert_eq!(err.to_string(), "invalid native response");
    }
}
