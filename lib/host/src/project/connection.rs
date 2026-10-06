use super::Executor;
use crate::{domain, ssh};
use std::time::Instant;

impl<E: Executor> super::Runtime<E> {
    /// `Connection`: environment plus the fixed public Ed25519 host key.
    pub fn connection(&self, id: &str, deadline: Instant) -> Result<domain::Connection, String> {
        let (env, _) = self.inspect(id, deadline)?;
        let mut result = domain::Connection {
            environment: env,
            host_key: String::new(),
            fingerprint: String::new(),
        };
        if !result.environment.running {
            return Ok(result);
        }
        let data = match self.podman(
            &[],
            &[
                "exec",
                &format!("soda-{id}"),
                "/usr/bin/head",
                "-c",
                "16385",
                "/etc/ssh/ssh_host_ed25519_key.pub",
            ],
            deadline,
        ) {
            Ok(data) if data.len() <= 16384 => data,
            _ => return Err("public host key unavailable".to_string()),
        };
        let (key, _) =
            ssh::parse_authorized_key(&data).map_err(|_| "invalid public host key".to_string())?;
        if key.key_type != ssh::ALGO_ED25519 {
            return Err("invalid public host key".to_string());
        }
        result.host_key = ssh::marshal_authorized_key(&key.key_type, &key.blob);
        result.fingerprint = ssh::fingerprint_sha256(&key.blob);
        Ok(result)
    }
}
