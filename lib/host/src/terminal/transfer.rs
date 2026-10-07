use super::clean_path;

const HARNESS_PIPELINE_SCRIPT: &str = r#"
"$1" --create --file=- --directory "$3" . |
  "$2" --remote=false exec --interactive "$4" /usr/bin/tar --extract --file=- --directory "$5" --no-same-owner --same-permissions
"#;

/// Bash argv for a single process-group-owned tar-to-Podman transfer. All
/// varying values remain positional arguments; none are interpolated into
/// the command string.
pub(crate) fn harness_pipeline_argv(
    tar: &str,
    podman: &str,
    harness: &str,
    container: &str,
    path: &str,
) -> Vec<String> {
    vec![
        "-o".to_string(),
        "pipefail".to_string(),
        "-c".to_string(),
        HARNESS_PIPELINE_SCRIPT.to_string(),
        "soda-harness-transfer".to_string(),
        tar.to_string(),
        podman.to_string(),
        clean_path(harness),
        container.to_string(),
        path.to_string(),
    ]
}
