use std::collections::HashMap;
use std::net::Ipv4Addr;

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
            &[
                "/usr/bin/curl",
                "--fail",
                "--silent",
                "--show-error",
                "--unix-socket",
                "/run/soda-tailnet/host/control/tailscaled.sock",
                "http://local-tailscaled.sock/localapi/v0/status",
            ],
            "read tailscale status",
        )?,
        "parse tailscale status",
    )?;
    let self_peer = status.get("Self");
    let dns_name = self_peer
        .and_then(|peer| peer.get("DNSName"))
        .and_then(JsonValue::as_str);
    let ips = self_peer
        .and_then(|peer| peer.get("TailscaleIPs"))
        .and_then(|value| match value {
            JsonValue::Array(items) => Some(items.as_slice()),
            _ => None,
        })
        .or_else(|| match status.get("TailscaleIPs") {
            Some(JsonValue::Array(items)) => Some(items.as_slice()),
            _ => None,
        });
    let tailnet_ips: Vec<&str> = ips
        .into_iter()
        .flatten()
        .filter_map(JsonValue::as_str)
        .collect();
    let selected_ip = selected_listener_ip(host, dns_name, &tailnet_ips).ok_or_else(|| {
        ProbeFailure::exit("advertisement does not match a current Tailnet address")
    })?;
    let listeners = podman(&["port", "soda-forgejo", "22/tcp"], "read published ports")?;
    if !published_listener(selected_ip, port, &listeners) {
        return Err(ProbeFailure::exit(
            "advertised endpoint is not a real selected private listener",
        ));
    }
    Ok(format!(
        "Listener-checked Git SSH advertisement: {host} {port}\nCore browser origins unchanged. Probe the pinned endpoint from the intended client; this is not routed Git authentication proof.\n"
    ))
}

fn selected_listener_ip<'a>(
    host: &str,
    dns_name: Option<&str>,
    ips: &'a [&'a str],
) -> Option<&'a str> {
    if let Some(ip) = ips
        .iter()
        .copied()
        .find(|ip| *ip == host && ip.parse::<Ipv4Addr>().is_ok())
    {
        return Some(ip);
    }
    let dns_name = dns_name?;
    let canonical_dns = dns_name.strip_suffix('.').unwrap_or(dns_name);
    if canonical_dns.is_empty() || !canonical_dns.eq_ignore_ascii_case(host) {
        return None;
    }
    ips.iter()
        .copied()
        .find(|ip| ip.parse::<Ipv4Addr>().is_ok())
}

fn published_listener(ip: &str, port: &str, listeners: &str) -> bool {
    listeners.lines().any(|line| line == format!("{ip}:{port}"))
}
