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
