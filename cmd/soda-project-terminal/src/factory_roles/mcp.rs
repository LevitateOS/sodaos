//! P6 fixed relay staging guard (FW33).
//!
//! The trusted host stages the fixed relay, capability and MCP
//! configuration before the start marker; clients choose no socket, URL,
//! repository, executable, account or listener. This guard pins the exact
//! IPC paths and relay argv the helper stages; production delivery and
//! start-marker wiring ride FW39.

use std::path::{Path, PathBuf};

/// Fixed relay argv: the helper launches nothing else.
pub const RELAY_ARGV: [&str; 2] = ["soda-project-terminal", "factory-mcp"];

/// Fixed IPC directory name under the service private data root.
pub const IPC_DIR_NAME: &str = "factory-ipc";
/// Fixed socket name inside the IPC directory.
pub const SOCKET_NAME: &str = "mcp.sock";

/// Admitted pinned harness versions proven by P6 native evidence.
pub fn is_pinned_harness(harness: &str, harness_vers: &str) -> bool {
    matches!(
        (harness, harness_vers),
        ("muse", "1.4.9" | "1.4.9-R6297.1") | ("codex", "0.161.0")
    )
}

/// Exact staged socket path under the service root. Any caller-supplied
/// destination, absolute socket path or alternative listener refuses:
/// reconnects observe the recreated inode through the read-only
/// directory mount, never a caller-chosen bind.
pub fn staged_socket(root: &Path) -> PathBuf {
    root.join(IPC_DIR_NAME).join(SOCKET_NAME)
}

/// Accept only the exact staged socket path. Extension, operator, broker
/// and host-operation sockets refuse, as does any caller destination.
pub fn check_socket(candidate: &Path, root: &Path) -> Result<(), String> {
    if candidate == staged_socket(root) {
        Ok(())
    } else {
        Err("unsupported factory socket".to_string())
    }
}

/// Accept only the fixed relay argv. Secrets, URLs, repository paths and
/// caller executables never ride argv: anything else refuses.
pub fn check_relay_argv(argv: &[&str]) -> Result<(), String> {
    if argv == RELAY_ARGV {
        Ok(())
    } else {
        Err("unsupported factory relay".to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stages_exact_ipc_paths() {
        let root = Path::new("/var/lib/soda/factory-ipc-root");
        let socket = staged_socket(root);
        assert_eq!(
            socket,
            Path::new("/var/lib/soda/factory-ipc-root/factory-ipc/mcp.sock")
        );
        assert!(check_socket(&socket, root).is_ok());
    }

    #[test]
    fn refuses_foreign_sockets() {
        let root = Path::new("/var/lib/soda/factory-ipc-root");
        for candidate in [
            "/var/lib/soda/factory-ipc-root/extension.sock",
            "/var/lib/soda/factory-ipc-root/operator.sock",
            "/var/lib/soda/factory-ipc-root/broker.sock",
            "/var/lib/soda/factory-ipc-root/mcp.sock",
            "/tmp/evil.sock",
            "/var/lib/soda/factory-ipc-root/factory-ipc/other.sock",
        ] {
            assert!(
                check_socket(Path::new(candidate), root).is_err(),
                "{candidate}: foreign socket admitted"
            );
        }
    }

    #[test]
    fn pins_relay_and_harness() {
        assert!(check_relay_argv(&["soda-project-terminal", "factory-mcp"]).is_ok());
        assert!(check_relay_argv(&["soda-project-terminal"]).is_err());
        assert!(check_relay_argv(&["/tmp/relay", "factory-mcp"]).is_err());
        assert!(is_pinned_harness("muse", "1.4.9"));
        assert!(is_pinned_harness("codex", "0.161.0"));
        assert!(!is_pinned_harness("muse", "1.4.2"));
        assert!(!is_pinned_harness("codex", "0.162.0"));
        assert!(!is_pinned_harness("shell", "1.4.9"));
    }
}
