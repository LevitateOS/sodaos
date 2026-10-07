use std::fs;

use crate::origin::{valid_listen, valid_origin};

/// The embedded Python block: read only public origins and the dashboard
/// port, never credentials or the whole document.
pub(crate) fn render_config(config: &str, tunnel_host: &str) {
    let host = if tunnel_host == "HOST" {
        ""
    } else {
        tunnel_host
    };
    let raw = fs::read(config).unwrap_or_default();
    let text = String::from_utf8_lossy(&raw);
    let parsed = parse_top_object(&text);
    let rendered = parsed.as_ref().and_then(|fields| {
        // `config.get('listen', default)`: absent takes the default, but a
        // present non-string is invalid, never defaulted.
        let listen = match fields.get("listen") {
            None => "127.0.0.1:8080".to_string(),
            Some(Some(value)) => value.clone(),
            Some(None) => return None,
        };
        let (listen_host, listen_port) = valid_listen(&listen)?;
        let url = match fields.get("forgejo_url") {
            Some(Some(url)) => url,
            _ => return None,
        };
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

/// Read the consumed top-level string fields while Serde validates the entire value.
pub(crate) fn parse_top_object(
    text: &str,
) -> Option<std::collections::HashMap<String, Option<String>>> {
    use serde::de::{IgnoredAny, MapAccess, Visitor};
    use serde::Deserialize;

    struct TopObject(std::collections::HashMap<String, Option<String>>);
    impl<'de> Deserialize<'de> for TopObject {
        fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
            struct ObjectVisitor;
            impl<'de> Visitor<'de> for ObjectVisitor {
                type Value = TopObject;
                fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                    f.write_str("a JSON object")
                }
                fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> Result<Self::Value, M::Error> {
                    let mut fields = std::collections::HashMap::new();
                    while let Some(key) = map.next_key::<String>()? {
                        if key == "listen" || key == "forgejo_url" {
                            let raw = map.next_value::<Box<serde_json::value::RawValue>>()?;
                            let value = if raw.get().starts_with('"') {
                                Some(
                                    serde_json::from_str::<String>(raw.get())
                                        .map_err(serde::de::Error::custom)?,
                                )
                            } else {
                                None
                            };
                            fields.insert(key, value);
                        } else {
                            let _: IgnoredAny = map.next_value()?;
                        }
                    }
                    Ok(TopObject(fields))
                }
            }
            deserializer.deserialize_map(ObjectVisitor)
        }
    }

    let mut deserializer = serde_json::Deserializer::from_str(text);
    let TopObject(fields) = TopObject::deserialize(&mut deserializer).ok()?;
    deserializer.end().ok()?;
    Some(fields)
}
