//! P6 fixed MCP relay surface (FW33, instruction claim corrected
//! lane-b19 per CHECKUP-012 R1).
//!
//! Fixed stdio relay, private IPC directory/socket and untrusted
//! user-prompt delivery shapes for the supported
//! harnesses. The relay translates transport and adds run
//! authorization; workflow policy stays in Go. No TCP/public listener,
//! no caller-chosen socket, URL, repository, executable or listener.
//! There is no trusted high-priority instruction channel on these
//! harnesses (see below): staged prompts are user instructions.

use super::run::{FACTORY_HARNESS_CODEX, FACTORY_HARNESS_MUSE};

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
/// Both harnesses accept the staged role prompt only as USER
/// instructions: `muse exec --prompt-file <PATH>` reads the prompt
/// from a file, and `codex exec - <prompt` reads initial instructions
/// from stdin (piped stdin is appended as a `<stdin>` block). D14
/// forbids treating these shapes as trusted instruction channels:
/// staged bytes are untrusted user input to the harness, never
/// relabeled as trusted instructions. Prefixing a user message and
/// calling it a system prompt never passes.
///
/// Per-run developer instructions ride a SEPARATE channel below
/// ([`PerRunInstructionRoute`]): the C11-proven `-c` configuration
/// keys. The staged prompt bytes never move there on their own.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UserPromptDelivery {
    /// Muse reads the staged user prompt through `--prompt-file`.
    PromptFile,
    /// Codex reads the staged user prompt through stdin (`- <prompt`).
    StdinPrompt,
}

/// No trusted high-priority instruction channel exists for the staged
/// prompt delivery shapes above: D14 forbids relabeling the
/// prompt-file/stdin user-prompt inputs as trusted instructions. This
/// stays `None`; the lane-b19 CLI probes record that boundary.
/// Per-run developer instructions use the separate C11-proven `-c`
/// routes below ([`PerRunInstructionRoute`]), never this constant.
pub const TRUSTED_INSTRUCTION_CHANNEL: Option<&'static str> = None;

/// C11-proven per-run developer-instruction route per admitted harness
/// family (evidence: `evidence/instruction-channel/`, pinned
/// `codex-cli 0.161.0` and `Muse Code 1.4.9`).
///
/// Codex `exec -c developer_instructions="<text>"` parses the value as
/// TOML with the highest precedence over `config.toml`: the marker
/// lands first in the role=developer input item while skills,
/// multi-agent, project-doc, env-context and user items are preserved;
/// a base-file value is REPLACED and the file itself stays byte-intact.
/// Muse `exec -c run.developer_prompt="<text>"` takes a JSON value and
/// records it with `developer_prompt_source=settings` (reviewed for
/// approval authorization) while sibling run config stays intact; bare
/// `developer_instructions`/`developer_prompt`, `run.base_instructions`
/// and the base-instructions-slot `run.system_prompt` are NOT this
/// route. Model-template composition (Codex) and real-provider wire
/// placement (Muse) need authed custody and stay unproven: P6/P7 gates
/// stay OPEN and this plumbing claims nothing about them.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PerRunInstructionRoute {
    /// Codex reads per-run developer instructions from
    /// `-c developer_instructions`.
    CodexDeveloperInstructions,
    /// Muse reads per-run developer instructions from
    /// `-c run.developer_prompt`.
    MuseDeveloperPrompt,
}

/// Per-run instruction route selected by the harness interface.
pub fn instruction_route(harness: &str) -> Option<PerRunInstructionRoute> {
    match harness {
        FACTORY_HARNESS_MUSE => Some(PerRunInstructionRoute::MuseDeveloperPrompt),
        FACTORY_HARNESS_CODEX => Some(PerRunInstructionRoute::CodexDeveloperInstructions),
        _ => None,
    }
}

/// Exact `-c` configuration key for the route: `developer_instructions`
/// (Codex, TOML value) or `run.developer_prompt` (Muse, JSON value).
/// Wrong keys are unrepresentable: callers cannot route instruction
/// text through a bare, base-instructions or system-prompt key.
pub fn instruction_config_key(route: PerRunInstructionRoute) -> &'static str {
    match route {
        PerRunInstructionRoute::CodexDeveloperInstructions => "developer_instructions",
        PerRunInstructionRoute::MuseDeveloperPrompt => "run.developer_prompt",
    }
}

/// TOML basic-string body for a Codex `-c developer_instructions`
/// value: the CLI parses `-c` as TOML, so the text rides quoted with
/// TOML escapes. Control characters take short escapes where TOML
/// defines them and `\u{...}` otherwise; all other Unicode passes
/// through verbatim.
fn toml_string_body(text: &str) -> String {
    let mut out = String::with_capacity(text.len() + 2);
    for ch in text.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04X}", c as u32));
            }
            c => out.push(c),
        }
    }
    out
}

/// JSON string body for a Muse `-c run.developer_prompt` value: the CLI
/// takes JSON values, so the text rides quoted with JSON escapes.
/// Characters below U+0020 take short escapes where JSON defines them
/// and `\u00XX` otherwise; all other Unicode passes through verbatim.
fn json_string_body(text: &str) -> String {
    let mut out = String::with_capacity(text.len() + 2);
    for ch in text.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out
}

/// Exact per-run `-c` argv pair carrying developer-instruction text:
/// `["-c", "<key>=\"<escaped>\""]` with the route's exact key and the
/// value encoding its CLI parses (TOML for Codex, JSON for Muse).
///
/// Per-run only: the pair rides the harness argv. It writes no config
/// file, loads no global user config and overrides none — the relay
/// never points a harness at a rewritten `config.toml`/`settings.json`
/// — and base instructions are preserved (C11: the pair replaces only
/// the same-key file value while non-config items and sibling run
/// config stay intact). Standard CLI override gives the pair
/// precedence over a same-key file value; that precedence is C11
/// native evidence, not a unit-test claim.
///
/// The text must be non-empty, within [`MAX_INSTRUCTION`] bytes and
/// NUL-free (argv cannot carry NUL); anything else refuses.
pub fn instruction_argv(route: PerRunInstructionRoute, text: &str) -> Result<[String; 2], String> {
    if text.is_empty() {
        return Err("instruction text is empty".to_string());
    }
    if text.len() > MAX_INSTRUCTION {
        return Err("instruction text exceeds the bounded size".to_string());
    }
    if text.contains('\0') {
        return Err("instruction text carries NUL".to_string());
    }
    let body = match route {
        PerRunInstructionRoute::CodexDeveloperInstructions => toml_string_body(text),
        PerRunInstructionRoute::MuseDeveloperPrompt => json_string_body(text),
    };
    Ok([
        "-c".to_string(),
        format!("{}=\"{}\"", instruction_config_key(route), body),
    ])
}

/// The supported fixed stdio configuration for one admitted harness:
/// fixed relay argv plus its (untrusted) user-prompt delivery shape.
/// The configuration claims no trusted instruction channel.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StdioConfig {
    pub harness: &'static str,
    pub relay_argv: [&'static str; 2],
    pub prompt: UserPromptDelivery,
}

/// Supported fixed stdio configuration for each harness interface.
pub fn stdio_config(harness: &str) -> Option<StdioConfig> {
    let (harness, prompt) = match harness {
        FACTORY_HARNESS_MUSE => (FACTORY_HARNESS_MUSE, UserPromptDelivery::PromptFile),
        FACTORY_HARNESS_CODEX => (FACTORY_HARNESS_CODEX, UserPromptDelivery::StdinPrompt),
        _ => return None,
    };
    Some(StdioConfig {
        harness,
        relay_argv: ["soda-project-terminal", FACTORY_MCP_RELAY],
        prompt,
    })
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
