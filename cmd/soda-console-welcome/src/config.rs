use std::fs;
use std::io::Read;

use crate::origin::{valid_listen, valid_origin};

const MAX_CONFIG_BYTES: usize = 64 * 1024;

/// Display only the configured public origin and dashboard port; never emit
/// unrelated configuration fields or credentials.
pub(crate) fn render_config(config: &str, tunnel_host: &str) {
    let host = if tunnel_host == "HOST" {
        ""
    } else {
        tunnel_host
    };
    let parsed = fs::File::open(config).ok().and_then(read_config);
    let rendered = parsed.as_ref().and_then(|value| {
        let fields = value.as_object()?;
        // `config.get('listen', default)`: absent takes the default, but a
        // present non-string is invalid, never defaulted.
        let listen = match fields.get("listen") {
            None => "127.0.0.1:8080".to_string(),
            Some(value) => value.as_str()?.to_string(),
        };
        let (listen_host, listen_port) = valid_listen(&listen)?;
        let url = fields.get("forgejo_url")?.as_str()?;
        let display = valid_origin(url)?;
        Some((listen_host, listen_port, display))
    });
    match rendered {
        None => {
            println!(
                "\nBrowser origins are not configured or cannot be read; complete operator setup."
            );
            println!("\nForgejo installer is loopback-first: http://127.0.0.1:3000");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L 33000:127.0.0.1:3000 root@{host}");
                println!("Then open http://localhost:33000 in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
        }
        Some((listen_host, listen_port, forgejo)) => {
            println!("\nDashboard is loopback-first: http://{listen_host}:{listen_port}");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L {listen_port}:{listen_host}:{listen_port} root@{host}");
                println!("Then open http://{listen_host}:{listen_port} in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
            println!("\nConfigured browser origins (not a listener or reachability check):");
            println!("  Forgejo / Sodaspaces: {forgejo}");
        }
    }
}

/// Cap the opened input before decoding the public guidance configuration.
pub(crate) fn read_config(reader: impl Read) -> Option<serde_json::Value> {
    let mut raw = Vec::with_capacity(MAX_CONFIG_BYTES + 1);
    reader
        .take((MAX_CONFIG_BYTES + 1) as u64)
        .read_to_end(&mut raw)
        .ok()?;
    if raw.len() > MAX_CONFIG_BYTES {
        return None;
    }
    let text = String::from_utf8_lossy(&raw);
    serde_json::from_str(&text).ok()
}
