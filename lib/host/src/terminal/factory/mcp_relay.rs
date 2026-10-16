//! P6 fixed MCP relay surface (FW33, instruction claim corrected
//! lane-b19 per CHECKUP-012 R1).
//!
//! Fixed stdio relay, private IPC directory/socket and untrusted
//! user-prompt delivery shapes for the exact admitted pinned
//! harnesses. The relay translates transport and adds run
//! authorization; workflow policy stays in Go. No TCP/public listener,
//! no caller-chosen socket, URL, repository, executable or listener.
//! There is no trusted high-priority instruction channel on the pinned
//! harnesses (see below): staged prompts are user instructions.

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
/// Staged user-prompt bytes bound (16 KiB, D14).
pub const MAX_INSTRUCTION: usize = 16 * 1024;

/// Untrusted user-prompt delivery shape per admitted harness family.
///
/// Both pinned harnesses accept the staged role prompt only as USER
/// instructions: `muse exec --prompt-file <PATH>` reads the prompt
/// from a file, and `codex exec - <prompt` reads initial instructions
/// from stdin (piped stdin is appended as a `<stdin>` block). Neither
/// pinned CLI documents a high-priority (system/developer) instruction
/// channel, so D14 forbids treating these shapes as trusted
/// instruction channels: staged bytes are untrusted user input to the
/// harness. Prefixing a user message and calling it a system prompt
/// never passes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UserPromptDelivery {
    /// Muse reads the staged user prompt through `--prompt-file`.
    PromptFile,
    /// Codex reads the staged user prompt through stdin (`- <prompt`).
    StdinPrompt,
}

/// No trusted high-priority instruction channel exists on the pinned
/// harnesses: neither pinned `muse exec --help` nor `codex exec --help`
/// documents a system/developer instruction flag, and D14 forbids
/// relabeling their user-prompt inputs as trusted instructions. This
/// stays `None` until a pinned CLI documents such a mechanism; the
/// lane-b19 CLI probes record the exact boundary.
pub const TRUSTED_INSTRUCTION_CHANNEL: Option<&'static str> = None;

/// The supported fixed stdio configuration for one admitted harness:
/// fixed relay argv plus its (untrusted) user-prompt delivery shape.
/// The configuration claims no trusted instruction channel.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StdioConfig {
    pub harness: &'static str,
    pub harness_vers: &'static str,
    pub relay_argv: [&'static str; 2],
    pub prompt: UserPromptDelivery,
}

/// Supported fixed stdio configuration, or `None` when the exact
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
                prompt: UserPromptDelivery::PromptFile,
            })
        }
        (h, v) if h == FACTORY_HARNESS_CODEX && v == PINNED_CODEX_HARNESS => Some(StdioConfig {
            harness: FACTORY_HARNESS_CODEX,
            harness_vers: PINNED_CODEX_HARNESS,
            relay_argv: ["soda-project-terminal", FACTORY_MCP_RELAY],
            prompt: UserPromptDelivery::StdinPrompt,
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
        assert_eq!(muse.prompt, UserPromptDelivery::PromptFile);
        let muse_build = stdio_config("muse", "1.4.9-R6297.1").expect("muse build admitted");
        assert_eq!(muse_build.prompt, UserPromptDelivery::PromptFile);
        let codex = stdio_config("codex", "0.161.0").expect("pinned codex admitted");
        assert_eq!(codex.relay_argv, ["soda-project-terminal", "factory-mcp"]);
        assert_eq!(codex.prompt, UserPromptDelivery::StdinPrompt);
    }

    #[test]
    fn trusted_instruction_channel_stays_unavailable() {
        // D14: user-prompt inputs are never relabeled as trusted
        // instructions. Neither pinned CLI documents a
        // system/developer instruction flag.
        assert_eq!(TRUSTED_INSTRUCTION_CHANNEL, None);
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
