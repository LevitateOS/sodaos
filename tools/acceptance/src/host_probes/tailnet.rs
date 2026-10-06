use soda_json::JsonValue;

use super::{is_truthy, parse_json, ProbeFailure};

/// Approved running Tailnet gate: the `forgejo-advertisement.sh`
/// inline `python3 -c` over stdin. Silent on success.
pub fn forgejo_tailnet(stdin: &str) -> Result<String, ProbeFailure> {
    let doc = parse_json(stdin, "parse tailscale status")?;
    let running = doc.get("BackendState").and_then(|v| v.as_str()) == Some("Running");
    let expired = match doc.get("Self") {
        None => false,
        Some(value) if !is_truthy(value) => false,
        Some(JsonValue::Object(_)) => doc
            .get("Self")
            .and_then(|v| v.get("Expired"))
            .is_some_and(is_truthy),
        Some(_) => return Err(ProbeFailure::failed("validate tailscale status")),
    };
    if !running || expired {
        return Err(ProbeFailure::exit(
            "approved running Tailnet required; no enrollment is performed",
        ));
    }
    Ok(String::new())
}
