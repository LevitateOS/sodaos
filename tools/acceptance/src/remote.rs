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
