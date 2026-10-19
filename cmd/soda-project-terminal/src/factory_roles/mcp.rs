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

/// Per-run developer-instruction text bound (16 KiB, D14): the same
/// bound as staged user-prompt bytes. Anything larger refuses rather
/// than truncates.
pub const MAX_INSTRUCTION_TEXT: usize = 16 * 1024;

/// C11-proven per-run developer-instruction route per admitted harness
/// family (evidence: `evidence/instruction-channel/`, pinned
/// `codex-cli 0.161.0` and `Muse Code 1.4.9`). This is NOT the staged
/// prompt channel above: prompt-file/stdin bytes stay untrusted user
/// prompts per D14 and are never relabeled through this route.
///
/// Codex `exec -c developer_instructions="<text>"` parses the value as
/// TOML with the highest precedence over `config.toml` (per-run text
/// replaces only the same-key file value; the file stays byte-intact;
/// skills, multi-agent, project-doc, env-context and user items are
/// preserved). Muse `exec -c run.developer_prompt="<text>"` takes a
/// JSON value recorded with `developer_prompt_source=settings` while
/// sibling run config stays intact. Model-template composition (Codex)
/// and real-provider wire placement (Muse) need authed custody and
/// stay unproven: P6/P7 gates stay OPEN and this plumbing claims
/// nothing about them.
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
        "muse" => Some(PerRunInstructionRoute::MuseDeveloperPrompt),
        "codex" => Some(PerRunInstructionRoute::CodexDeveloperInstructions),
        _ => None,
    }
}

/// Exact `-c` configuration key for the route: `developer_instructions`
/// (Codex, TOML value) or `run.developer_prompt` (Muse, JSON value).
/// Wrong keys are unrepresentable: bare `developer_prompt`,
/// `base_instructions`, `run.base_instructions` and the
/// base-instructions-slot `run.system_prompt` cannot route here.
pub fn instruction_config_key(route: PerRunInstructionRoute) -> &'static str {
    match route {
        PerRunInstructionRoute::CodexDeveloperInstructions => "developer_instructions",
        PerRunInstructionRoute::MuseDeveloperPrompt => "run.developer_prompt",
    }
}

/// TOML basic-string body for a Codex `-c developer_instructions`
/// value. Control characters take short escapes where TOML defines
/// them and `\u{...}` otherwise; all other Unicode passes through.
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

/// JSON string body for a Muse `-c run.developer_prompt` value.
/// Characters below U+0020 take short escapes where JSON defines them
/// and `\u00XX` otherwise; all other Unicode passes through.
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
/// file, loads no global user config and overrides none, and base
/// instructions are preserved (C11: the pair replaces only the
/// same-key file value while non-config items and sibling run config
/// stay intact). Standard CLI override gives the pair precedence over
/// a same-key file value; that precedence is C11 native evidence, not
/// a unit-test claim.
///
/// The text must be non-empty, within [`MAX_INSTRUCTION_TEXT`] bytes
/// and NUL-free (argv cannot carry NUL); anything else refuses.
pub fn instruction_argv(route: PerRunInstructionRoute, text: &str) -> Result<[String; 2], String> {
    if text.is_empty() {
        return Err("instruction text is empty".to_string());
    }
    if text.len() > MAX_INSTRUCTION_TEXT {
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
