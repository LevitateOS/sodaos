//! Exact-source native dispatch, mirroring `remote.go` with a compiled
//! payload.
//!
//! The Go owner pipes embedded Python source to `python3 -c` with the
//! request on stdin. Targets have no Rust toolchain, so the port ships a
//! prebuilt payload binary instead: one SSH command writes stdin to a fresh
//! tempfile, executes the requested phase with the request as an argument,
//! and always removes the tempfile. Nothing is installed on the target.

use serde_json::value::RawValue;
use std::collections::BTreeMap;

use crate::command::{quote, CommandSpec, Remote, StdinSpec};
use crate::error::Error;
use crate::files;
use crate::jsonio;

/// Exact-source native request, mirroring Go's `RemoteRequest`.
pub struct RemoteRequest {
    /// Full source revision.
    pub revision: String,
    /// Target architecture.
    pub architecture: String,
    /// Actual target hostname.
    pub target: String,
    /// Fresh work path on the target.
    pub work: String,
    /// Requested phase.
    pub phase: String,
}

/// Decode a restricted native request with exactly the Go owner's fields.
pub fn decode_remote_request(value: &RawValue) -> Result<RemoteRequest, Error> {
    let fields = match serde_json::from_str::<BTreeMap<String, Box<RawValue>>>(value.get()) {
        Ok(fields) => fields,
        Err(_) if !value.get().trim_start().starts_with('{') => BTreeMap::new(),
        Err(_) => return Err(Error::msg("invalid native request")),
    };
    for key in fields.keys() {
        if !["Revision", "Architecture", "Target", "Work", "Phase"].contains(&key.as_str()) {
            return Err(Error::msg("unknown native request field"));
        }
    }
    let string_field = |name: &str| -> Result<String, Error> {
        match fields.get(name) {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                .map(|v| v.unwrap_or_default())
                .map_err(|_| Error::msg("native request fields must be strings")),
        }
    };
    Ok(RemoteRequest {
        revision: string_field("Revision")?,
        architecture: string_field("Architecture")?,
        target: string_field("Target")?,
        work: string_field("Work")?,
        phase: string_field("Phase")?,
    })
}

/// Build the one SSH command running a native phase. `payload` holds the
/// prebuilt `soda-acceptance-remote` bytes piped over stdin; the temporary
/// copy is always removed after the phase, preserving the no-install
/// property. Validation order mirrors the Go owner: request binding before
/// any SSH validation.
pub fn native_phase(
    remote: &Remote,
    request_file: &str,
    revision: &str,
    arch: &str,
    target: &str,
    payload: &[u8],
) -> Result<CommandSpec, Error> {
    let raw = files::private_file(request_file)?;
    let (request, _) = jsonio::read_json_file(request_file)?;
    let request = decode_remote_request(&request)?;
    if request.revision != revision || request.architecture != arch || request.target != target {
        return Err(Error::msg(
            "remote request does not match selected revision/platform/target",
        ));
    }
    if !["prepare", "build", "check"].contains(&request.phase.as_str()) {
        return Err(Error::msg("unknown native phase"));
    }
    if !request.work.starts_with('/') || request.work.contains(['\r', '\n', '\0']) {
        return Err(Error::msg("absolute remote work path required"));
    }
    let request_text = String::from_utf8(raw).map_err(|_| Error::msg("invalid native request"))?;
    let quoted = quote(std::slice::from_ref(&request_text));
    let script = format!(
        "tmp=$(mktemp) && cat > \"$tmp\" && chmod 700 \"$tmp\" && \"$tmp\" native-phase {quoted}; rc=$?; rm -f \"$tmp\"; exit $rc"
    );
    remote.command(
        &["sh".to_string(), "-c".to_string(), script],
        StdinSpec::Bytes(payload.to_vec()),
    )
}

#[cfg(test)]
mod json_tests {
    use super::decode_remote_request;
    use serde_json::value::RawValue;

    #[test]
    fn native_request_uses_last_exact_string_and_ignores_overwritten_raw_number() {
        let raw = RawValue::from_string(
            r#"{"Phase":1e400,"Phase":"check","Target":null,"Target":"builder"}"#.to_string(),
        )
        .unwrap();
        let decoded = decode_remote_request(&raw).unwrap();
        assert_eq!(decoded.phase, "check");
        assert_eq!(decoded.target, "builder");
        let bad = RawValue::from_string(r#"{"Phase":false}"#.to_string()).unwrap();
        assert!(decode_remote_request(&bad).is_err());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    fn fixture_dir(prefix: &str) -> std::path::PathBuf {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-remote-{prefix}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn write_request(dir: &std::path::Path, revision: &str, target: &str, phase: &str) -> String {
        let mut text = String::new();
        #[derive(serde::Serialize)]
        struct Request<'a> {
            #[serde(rename = "Revision")]
            revision: &'a str,
            #[serde(rename = "Architecture")]
            architecture: &'a str,
            #[serde(rename = "Target")]
            target: &'a str,
            #[serde(rename = "Work")]
            work: &'a str,
            #[serde(rename = "Phase")]
            phase: &'a str,
        }
        jsonio::write_compact(
            &mut text,
            &Request {
                revision,
                architecture: "x86_64",
                target,
                work: "/private/fresh",
                phase,
            },
        );
        let path = dir.join("request.json");
        std::fs::write(&path, &text).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        path.to_string_lossy().into_owned()
    }

    #[test]
    fn native_request_binding_rejects_mismatch() {
        let dir = fixture_dir("binding");
        let revision = "a".repeat(40);
        let request = write_request(&dir, &revision, "builder", "prepare");
        let remote = Remote {
            user: String::new(),
            host: String::new(),
            key: String::new(),
            known_hosts: String::new(),
            port: 0,
            timeout: Duration::ZERO,
        };
        assert!(native_phase(
            &remote,
            &request,
            &"b".repeat(40),
            "x86_64",
            "builder",
            b"payload"
        )
        .is_err());
        assert!(native_phase(&remote, &request, &revision, "x86_64", "other", b"payload").is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn remote_unknown_phase_fails_before_ssh() {
        let dir = fixture_dir("unknown");
        let revision = "a".repeat(40);
        let request = write_request(&dir, &revision, "fixture", "publish");
        // No identity/pins are supplied: local request validation must win first.
        let remote = Remote {
            user: String::new(),
            host: String::new(),
            key: String::new(),
            known_hosts: String::new(),
            port: 0,
            timeout: Duration::ZERO,
        };
        let err = native_phase(
            &remote, &request, &revision, "x86_64", "fixture", b"payload",
        )
        .unwrap_err();
        assert!(err.to_string().contains("unknown native phase"), "{err}");
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn native_delivery_runs_payload_from_tempfile() {
        let dir = fixture_dir("delivery");
        let key = dir.join("key");
        let hosts = dir.join("known_hosts");
        std::fs::write(&key, b"synthetic identity, not used for authentication").unwrap();
        std::fs::write(&hosts, b"synthetic pin, not used for authentication").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&key, std::fs::Permissions::from_mode(0o600)).unwrap();
        let revision = "a".repeat(40);
        let request = write_request(&dir, &revision, "builder", "prepare");
        let remote = Remote {
            user: "root".to_string(),
            host: "127.0.0.1".to_string(),
            key: key.to_string_lossy().into_owned(),
            known_hosts: hosts.to_string_lossy().into_owned(),
            port: 22222,
            timeout: Duration::ZERO,
        };
        let command = native_phase(
            &remote,
            &request,
            &revision,
            "x86_64",
            "builder",
            b"payload-bytes",
        )
        .unwrap();
        assert_eq!(command.name, "ssh");
        let last = command.args.last().unwrap();
        for part in ["mktemp", "chmod 700", "native-phase", "rm -f", &revision] {
            assert!(last.contains(part), "{last}");
        }
        match &command.stdin {
            StdinSpec::Bytes(bytes) => assert_eq!(bytes, b"payload-bytes"),
            _ => panic!("payload must travel over stdin"),
        }
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
