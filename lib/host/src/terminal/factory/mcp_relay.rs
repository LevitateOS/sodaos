//! P6 fixed MCP relay surface (FW33).
//!
//! Fixed stdio relay, private IPC directory/socket and trusted
//! instruction-channel shapes for the exact admitted pinned harnesses.
//! The relay translates transport and adds run authorization; workflow
//! policy stays in Go. No TCP/public listener, no caller-chosen socket,
//! URL, repository, executable or listener.

use super::run::{FACTORY_HARNESS_CODEX, FACTORY_HARNESS_MUSE};

/// Admitted pinned harness versions proven by P6 native evidence.
pub const PINNED_MUSE_HARNESS: &str = "1.4.9";
/// Admitted pinned harness versions proven by P6 native evidence.
pub const PINNED_CODEX_HARNESS: &str = "0.161.0";

/// Fixed relay subcommand: `soda-project-terminal factory-mcp`.
/// Callers supply no executable, socket, URL or extra arguments.
pub const FACTORY_MCP_RELAY: &str = "factory-mcp";

/// Fixed IPC directory relative to the service private data root.
pub const FACTORY_IPC_DIR: &str = "factory-ipc";
/// Fixed Factory-only socket relative to the service private data root.
/// The run-local read-only mount exposes this directory, never a single
/// socket bind, so reconnects observe a recreated inode after restart.
pub const FACTORY_MCP_SOCKET: &str = "factory-ipc/mcp.sock";

/// Managed MCP configuration bound (16 KiB, D14).
pub const MAX_MCP_CONFIG: usize = 16 * 1024;
/// Static instruction bytes bound (16 KiB, D14).
pub const MAX_INSTRUCTION: usize = 16 * 1024;

/// Trusted instruction channel per admitted harness family.
///
/// `None` means the harness/profile stays unavailable: prefixing a user
/// message and calling it a system prompt never passes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum InstructionChannel {
    /// Muse reads the staged prompt through `--prompt-file`.
    PromptFile,
    /// Codex reads the staged prompt through stdin (`- <prompt`).
    StdinPrompt,
}

/// The supported trusted stdio configuration for one admitted harness:
/// fixed relay argv plus its instruction channel.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StdioConfig {
    pub harness: &'static str,
    pub harness_vers: &'static str,
    pub relay_argv: [&'static str; 2],
    pub instruction: InstructionChannel,
}

/// Supported trusted stdio configuration, or `None` when the exact
/// harness/version/profile stays unavailable.
pub fn stdio_config(harness: &str, harness_vers: &str) -> Option<StdioConfig> {
    match (harness, harness_vers) {
        (h, v)
            if h == FACTORY_HARNESS_MUSE && (v == PINNED_MUSE_HARNESS || v == "1.4.9-R6297.1") =>
        {
            Some(StdioConfig {
                harness: FACTORY_HARNESS_MUSE,
                harness_vers: PINNED_MUSE_HARNESS,
                relay_argv: ["soda-project-terminal", FACTORY_MCP_RELAY],
                instruction: InstructionChannel::PromptFile,
            })
        }
        (h, v) if h == FACTORY_HARNESS_CODEX && v == PINNED_CODEX_HARNESS => Some(StdioConfig {
            harness: FACTORY_HARNESS_CODEX,
            harness_vers: PINNED_CODEX_HARNESS,
            relay_argv: ["soda-project-terminal", FACTORY_MCP_RELAY],
            instruction: InstructionChannel::StdinPrompt,
        }),
        _ => None,
    }
}

/// Fixed relay stdout contract: stdout carries MCP messages only;
/// bounded diagnostics go to stderr.
pub fn relay_stream_is_mcp(stream: &str) -> bool {
    stream == "stdout"
}

/// Fixed serving route of the Factory-only listener.
pub fn mcp_route() -> &'static str {
    "/mcp"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn admits_exact_pinned_harnesses() {
        let muse = stdio_config("muse", "1.4.9").expect("pinned muse admitted");
        assert_eq!(muse.relay_argv, ["soda-project-terminal", "factory-mcp"]);
        assert_eq!(muse.instruction, InstructionChannel::PromptFile);
        let muse_build = stdio_config("muse", "1.4.9-R6297.1").expect("muse build admitted");
        assert_eq!(muse_build.instruction, InstructionChannel::PromptFile);
        let codex = stdio_config("codex", "0.161.0").expect("pinned codex admitted");
        assert_eq!(codex.relay_argv, ["soda-project-terminal", "factory-mcp"]);
        assert_eq!(codex.instruction, InstructionChannel::StdinPrompt);
    }

    #[test]
    fn unsupported_harness_stays_unavailable() {
        for (harness, vers) in [
            ("muse", "1.4.2"),
            ("muse", "9.9.9"),
            ("codex", "0.160.0"),
            ("codex", "0.162.0"),
            ("shell", "1.4.9"),
            ("muse", ""),
            ("", "0.161.0"),
        ] {
            assert!(
                stdio_config(harness, vers).is_none(),
                "{harness} {vers}: unsupported harness admitted"
            );
        }
    }

    #[test]
    fn fixed_paths_and_route() {
        assert_eq!(FACTORY_IPC_DIR, "factory-ipc");
        assert_eq!(FACTORY_MCP_SOCKET, "factory-ipc/mcp.sock");
        assert_eq!(mcp_route(), "/mcp");
        assert!(!FACTORY_MCP_SOCKET.contains("extension"));
        assert!(!FACTORY_MCP_SOCKET.contains("operator"));
        assert!(relay_stream_is_mcp("stdout"));
        assert!(!relay_stream_is_mcp("stderr"));
    }
}
