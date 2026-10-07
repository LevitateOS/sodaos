// Host runtime callbacks, mirroring cmd/soda-identity/runtime.go over
// internal/host: the broker attests terminal sessions and supervised
// factory runs by their exact bindings; it never shells out itself.
use crate::wire::{DeliveryWire, Error, Lease};
use std::path::Path;
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
        let response = soda_unix_http::request(
            Path::new(&self.socket),
            "POST",
            path,
            "soda-host",
            &body,
            CALL_TIMEOUT,
            soda_unix_http::Limits {
                header_bytes: 65536,
                body_bytes: limit.unwrap_or(FINISH_LIMIT).saturating_add(1),
            },
        )
        .map_err(|error| match error {
            soda_unix_http::Error::Protocol
            | soda_unix_http::Error::InvalidRequest
            | soda_unix_http::Error::BodyTooLarge => Error::internal("invalid native response"),
            _ => Error::internal("host service unavailable"),
        })?;
        let status = response.status;
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
        if response.body.len() > limit || response.body.trim_ascii() == b"null" {
            return Err(Error::internal("invalid native response"));
        }
        serde_json::from_slice(&response.body)
            .map_err(|_| Error::internal("invalid native response"))
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wire::{Binding, UnixTime};
    use std::io::{Read, Write};
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
        let dir = Path::new(env!("CARGO_MANIFEST_DIR"))
            .ancestors()
            .nth(2)
            .expect("workspace root")
            .join(format!(
                ".artifacts/l08-l09/identity-{}",
                std::process::id()
            ));
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
                    if let Some((name, value)) = line
                        .iter()
                        .position(|b| *b == b':')
                        .map(|i| (&line[..i], &line[i + 1..]))
                    {
                        if name.eq_ignore_ascii_case(b"content-length") {
                            content_length = std::str::from_utf8(value.trim_ascii())
                                .unwrap()
                                .parse()
                                .unwrap();
                        }
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
    fn legal_chunked_response_is_accepted_by_host_client() {
        let client = stub(
            |_, _| {
                b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nTrailer: x-done\r\n\r\n2\r\n{}\r\n0\r\nx-done: yes\r\n\r\n".to_vec()
            },
            1,
        );
        assert!(client.validate(&lease(true)).is_ok());
    }
}
