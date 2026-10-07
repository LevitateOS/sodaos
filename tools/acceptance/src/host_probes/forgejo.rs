use std::collections::HashMap;

use crate::structured::Value as JsonValue;

use super::{dumps, get_str, parse_json, podman, read_text, run_output, ProbeFailure};

/// Core browser origins: the `forgejo-advertisement.sh` `origins()`
/// helper.
pub fn forgejo_origins() -> Result<String, ProbeFailure> {
    let cfg = parse_json(
        &read_text("/etc/soda/dashboard.json", "read dashboard.json")?,
        "parse dashboard.json",
    )?;
    Ok(dumps(&JsonValue::Object(vec![
        (
            "forgejo_internal_url".to_string(),
            JsonValue::Str(
                get_str(&cfg, "forgejo_internal_url", "parse dashboard.json")?.to_string(),
            ),
        ),
        (
            "forgejo_url".to_string(),
            JsonValue::Str(get_str(&cfg, "forgejo_url", "parse dashboard.json")?.to_string()),
        ),
    ])) + "\n")
}

/// Listener-checked Git SSH advertisement: the final
/// `forgejo-advertisement.sh` Python block.
pub fn forgejo_advertisement() -> Result<String, ProbeFailure> {
    let mut values = HashMap::new();
    for line in read_text("/etc/soda/forgejo.env", "read forgejo.env")?.lines() {
        if line.contains('=') && !line.starts_with('#') {
            let (key, value) = line.split_once('=').unwrap();
            values.insert(key.to_string(), value.to_string());
        }
    }
    let host = values
        .get("FORGEJO__server__SSH_DOMAIN")
        .ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let port = values
        .get("FORGEJO__server__SSH_PORT")
        .ok_or_else(|| ProbeFailure::failed("parse forgejo.env"))?;
    let status = parse_json(
        &run_output(
            &["/usr/bin/tailscale", "status", "--json"],
            "read tailscale status",
        )?,
        "parse tailscale status",
    )?;
    let on_tailnet = match status.get("TailscaleIPs") {
        Some(JsonValue::Array(ips)) => ips.iter().any(|ip| ip.as_str() == Some(host.as_str())),
        _ => false,
    };
    if !on_tailnet {
        return Err(ProbeFailure::exit(
            "advertisement does not match a current Tailnet address",
        ));
    }
    let listeners = podman(&["port", "soda-forgejo", "22/tcp"], "read published ports")?;
    if !listeners
        .lines()
        .any(|line| line == format!("{host}:{port}"))
    {
        return Err(ProbeFailure::exit(
            "advertised endpoint is not a real selected private listener",
        ));
    }
    Ok(format!(
        "Listener-checked Git SSH advertisement: {host} {port}\nCore browser origins unchanged. Probe the pinned endpoint from the intended client; this is not routed Git authentication proof.\n"
    ))
}
