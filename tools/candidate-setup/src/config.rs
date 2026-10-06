/// `json_escape` mirrors Python `json.dumps` with `ensure_ascii`: the
/// script generated every JSON file through it, so the bytes stay identical.
pub(super) fn json_escape(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len() + 2);
    for ch in value.chars() {
        match ch {
            '"' => escaped.push_str("\\\""),
            '\\' => escaped.push_str("\\\\"),
            '\n' => escaped.push_str("\\n"),
            '\r' => escaped.push_str("\\r"),
            '\t' => escaped.push_str("\\t"),
            '\u{08}' => escaped.push_str("\\b"),
            '\u{0c}' => escaped.push_str("\\f"),
            _ if (ch < '\u{20}' || ch == '\u{7f}') => {
                escaped.push_str(&format!("\\u{:04x}", ch as u32));
            }
            _ if ch > '\u{7e}' => {
                let mut encoded = [0u16; 2];
                for unit in ch.encode_utf16(&mut encoded) {
                    escaped.push_str(&format!("\\u{unit:04x}"));
                }
            }
            _ => escaped.push(ch),
        }
    }
    escaped
}

fn json_field(key: &str, value: &str, indent: usize) -> String {
    format!(
        "{:indent$}\"{}\": \"{}\"",
        "",
        key,
        json_escape(value),
        indent = indent
    )
}

fn json_number_field(key: &str, value: u64, indent: usize) -> String {
    format!("{:indent$}\"{}\": {}", "", key, value, indent = indent)
}

/// `worker_json` mirrors the `json.dump(..., indent=2)` for worker.json,
/// including the absent trailing newline.
#[allow(clippy::too_many_arguments)]
pub(super) fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
    authority: &str,
) -> String {
    [
        "{".to_string(),
        json_field("Executable", executable, 2) + ",",
        json_field("Source", source, 2) + ",",
        json_field("ForgejoSource", forgejo_source, 2) + ",",
        json_field("OutputParent", output_parent, 2) + ",",
        json_field("StorageRoot", storage_root, 2) + ",",
        json_field("BuildHome", build_home, 2) + ",",
        json_field("Runtime", runtime, 2) + ",",
        json_field("Tools", tools, 2) + ",",
        json_field("MediaAuthorityDirectory", authority, 2),
        "}".to_string(),
    ]
    .join("\n")
}

/// `trust_json` mirrors the `json.dumps(..., indent=2) + "\n"` trust file.
pub(super) fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let roles = ["artifact", "candidate", "preview", "stable"];
    let mut lines = vec![
        "{".to_string(),
        json_number_field("Format", 1, 2) + ",",
        json_field("Prefix", prefix, 2) + ",",
        json_number_field("Epoch", 1, 2) + ",",
        "  \"Keys\": {".to_string(),
    ];
    for (index, (role, key)) in roles.iter().zip(pubs.iter()).enumerate() {
        let comma = if index + 1 < roles.len() { "," } else { "" };
        lines.push(format!("    \"{role}\": ["));
        lines.push(format!("      \"{}\"", json_escape(key)));
        lines.push(format!("    ]{comma}"));
    }
    lines.push("  },".to_string());
    lines.push(json_number_field("NotBefore", now - 600, 2) + ",");
    lines.push(json_number_field("MaxAgeSeconds", 3600, 2) + ",");
    lines.push(json_number_field("ClockSkewSeconds", 10, 2) + ",");
    lines.push("  \"MinimumSequence\": {".to_string());
    lines.push(json_number_field("candidate", 1, 4) + ",");
    lines.push(json_number_field("preview", 1, 4) + ",");
    lines.push(json_number_field("stable", 1, 4));
    lines.push("  }".to_string());
    lines.push("}".to_string());
    lines.join("\n") + "\n"
}

/// `config_json` mirrors the `json.dumps(..., indent=2) + "\n"` config file.
pub(super) fn config_json() -> String {
    [
        "{".to_string(),
        json_field("Trust", "/run/soda-media-authority/trust.json", 2) + ",",
        "  \"Keys\": {".to_string(),
        json_field("Key", "/run/soda-media-authority/artifact.private", 4) + ",",
        json_field("Passphrase", "/run/soda-media-authority/passphrase", 4),
        "  }".to_string(),
        "}".to_string(),
    ]
    .join("\n")
        + "\n"
}
