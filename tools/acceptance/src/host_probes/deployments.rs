use soda_json::JsonValue;

use super::{dumps, is_truthy, parse_json, ProbeFailure};

/// `rpm-ostree status --json` deployments summary: the `host.sh`
/// inline `python3 -c` over stdin.
pub fn host_deployments(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse deployments")?;
    let deployments = doc.get("deployments").and_then(|v| match v {
        JsonValue::Array(items) => Some(items),
        _ => None,
    });
    let Some(deployments) = deployments else {
        return Err(ProbeFailure::failed("parse deployments"));
    };
    let mut out = Vec::new();
    for deployment in deployments {
        let JsonValue::Object(_) = deployment else {
            return Err(ProbeFailure::failed("parse deployments"));
        };
        let mut summary = Vec::new();
        for key in ["booted", "version", "checksum", "requested-packages"] {
            summary.push((
                key.to_string(),
                deployment.get(key).cloned().unwrap_or(JsonValue::Null),
            ));
        }
        out.push(JsonValue::Object(summary));
    }
    Ok(dumps(&JsonValue::Array(out)) + "\n")
}

/// Retained Tailnet state summary: the `operator.sh` inline
/// `python3 -c` over stdin.
pub fn operator_tailscale(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse tailscale status")?;
    let Some(JsonValue::Str(state)) = doc.get("BackendState") else {
        return Err(ProbeFailure::failed("validate tailscale status"));
    };
    // `(d.get("Self") or {})`: falsy shapes yield no expiry, truthy
    // non-objects fail like the owner's attribute error.
    let expired = match doc.get("Self") {
        None => JsonValue::Null,
        Some(value) if !is_truthy(value) => JsonValue::Null,
        Some(JsonValue::Object(_)) => doc
            .get("Self")
            .and_then(|v| v.get("Expired"))
            .cloned()
            .unwrap_or(JsonValue::Null),
        Some(_) => return Err(ProbeFailure::failed("validate tailscale status")),
    };
    Ok(dumps(&JsonValue::Object(vec![
        ("BackendState".to_string(), JsonValue::Str(state.clone())),
        ("Expired".to_string(), expired),
    ])) + "\n")
}
