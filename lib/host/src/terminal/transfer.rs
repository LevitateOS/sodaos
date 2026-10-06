use super::clean_path;

// ---------- identity harness transfer (identity_transfer.go) ----------

/// Host tar producer argv: `tar --create --file=- --directory <harness> .`.
/// The harness path is cleaned exactly like Go's `filepath.Clean`.
pub fn tar_producer_argv(harness: &str) -> Vec<String> {
    vec![
        "--create".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        clean_path(harness),
        ".".to_string(),
    ]
}

/// Guest tar consumer argv: `podman ... exec --interactive <container> tar
/// --extract --file=- --directory <path> --no-same-owner --same-permissions`.
pub fn tar_consumer_argv(container: &str, path: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        container.to_string(),
        "/usr/bin/tar".to_string(),
        "--extract".to_string(),
        "--file=-".to_string(),
        "--directory".to_string(),
        path.to_string(),
        "--no-same-owner".to_string(),
        "--same-permissions".to_string(),
    ]
}
