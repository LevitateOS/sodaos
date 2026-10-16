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

/// Per-run instruction route for the exact harness/version, or `None`
/// when the exact harness/version/profile stays unavailable. Mirrors
/// [`stdio_config`] admission: only pinned harnesses carry a route.
pub fn instruction_route(harness: &str, harness_vers: &str) -> Option<PerRunInstructionRoute> {
    match (harness, harness_vers) {
        (h, v)
            if h == FACTORY_HARNESS_MUSE && (v == PINNED_MUSE_HARNESS || v == "1.4.9-R6297.1") =>
        {
            Some(PerRunInstructionRoute::MuseDeveloperPrompt)
        }
        (h, v) if h == FACTORY_HARNESS_CODEX && v == PINNED_CODEX_HARNESS => {
            Some(PerRunInstructionRoute::CodexDeveloperInstructions)
        }
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
        // D14: prompt-file/stdin user-prompt inputs are never relabeled
        // as trusted instructions. Per-run developer instructions ride
        // the separate C11-proven `-c` routes, never this constant.
        assert_eq!(TRUSTED_INSTRUCTION_CHANNEL, None);
    }

    #[test]
    fn instruction_routes_match_stdio_admission() {
        // Config shape: only the exact pinned harness/versions carry a
        // per-run instruction route, mirroring stdio_config.
        use PerRunInstructionRoute::*;
        assert_eq!(
            instruction_route("muse", "1.4.9"),
            Some(MuseDeveloperPrompt)
        );
        assert_eq!(
            instruction_route("muse", "1.4.9-R6297.1"),
            Some(MuseDeveloperPrompt)
        );
        assert_eq!(
            instruction_route("codex", "0.161.0"),
            Some(CodexDeveloperInstructions)
        );
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
                instruction_route(harness, vers).is_none(),
                "{harness} {vers}: unpinned harness carries a route"
            );
        }
    }

    #[test]
    fn instruction_keys_are_exact() {
        // Config shape: the exact C11-proven `-c` keys, byte for byte.
        // Bare `developer_prompt`, `base_instructions`,
        // `run.base_instructions` and the base-instructions-slot
        // `run.system_prompt` are NOT routable: wrong keys are
        // unrepresentable through `PerRunInstructionRoute`.
        assert_eq!(
            instruction_config_key(PerRunInstructionRoute::CodexDeveloperInstructions),
            "developer_instructions"
        );
        assert_eq!(
            instruction_config_key(PerRunInstructionRoute::MuseDeveloperPrompt),
            "run.developer_prompt"
        );
    }

    #[test]
    fn instruction_argv_carries_exact_c11_shape() {
        // Config shape: a single `-c` pair with the exact key and the
        // value encoding its CLI parses, matching the C11 evidence
        // commands (`codex ... -c 'developer_instructions="..."'`,
        // `muse ... -c 'run.developer_prompt="..."'`).
        let codex = instruction_argv(
            PerRunInstructionRoute::CodexDeveloperInstructions,
            "follow R1",
        )
        .expect("valid instruction builds");
        assert_eq!(codex, ["-c", "developer_instructions=\"follow R1\""]);
        let muse = instruction_argv(PerRunInstructionRoute::MuseDeveloperPrompt, "follow R1")
            .expect("valid instruction builds");
        assert_eq!(muse, ["-c", "run.developer_prompt=\"follow R1\""]);
    }

    #[test]
    fn instruction_argv_escapes_for_its_cli() {
        // The value encoding must survive its CLI's parser: quotes,
        // backslashes and newlines round-trip through TOML (Codex) and
        // JSON (Muse) string escapes instead of breaking the `-c` value.
        let codex = instruction_argv(
            PerRunInstructionRoute::CodexDeveloperInstructions,
            "say \"hi\"\npath C:\\tmp",
        )
        .expect("escapable instruction builds");
        assert_eq!(
            codex[1],
            "developer_instructions=\"say \\\"hi\\\"\\npath C:\\\\tmp\""
        );
        let muse = instruction_argv(
            PerRunInstructionRoute::MuseDeveloperPrompt,
            "say \"hi\"\npath C:\\tmp",
        )
        .expect("escapable instruction builds");
        assert_eq!(
            muse[1],
            "run.developer_prompt=\"say \\\"hi\\\"\\npath C:\\\\tmp\""
        );
    }

    #[test]
    fn instruction_argv_is_one_per_run_pair() {
        // Precedence shape (honest scope): the builder emits exactly one
        // `-c` pair with the exact key and no config-file side channel,
        // which is the shape standard CLI override acts on — per-run
        // text wins over a same-key file value (C11 native evidence)
        // while base instructions outside that key are preserved. The
        // unit test pins the shape, not the CLI's override itself.
        for route in [
            PerRunInstructionRoute::CodexDeveloperInstructions,
            PerRunInstructionRoute::MuseDeveloperPrompt,
        ] {
            let pair = instruction_argv(route, "per-run wins").expect("valid instruction builds");
            assert_eq!(pair[0], "-c");
            assert!(pair[1].starts_with(instruction_config_key(route)));
            assert_eq!(pair[1].matches('=').count(), 1);
        }
    }

    #[test]
    fn instruction_argv_overrides_nothing_global() {
        // No-override: the pair names no config file, flag or path —
        // no `--config`, no `config.toml`, no `settings.json`, no home
        // directory — so it can neither load nor rewrite global user
        // config. It also refuses texts that cannot ride argv safely.
        for route in [
            PerRunInstructionRoute::CodexDeveloperInstructions,
            PerRunInstructionRoute::MuseDeveloperPrompt,
        ] {
            let pair =
                instruction_argv(route, "bounded per-run text").expect("valid instruction builds");
            let joined = pair.join(" ");
            for banned in [
                "--config",
                "config.toml",
                "settings.json",
                ".codex",
                ".config",
                "--strict-config",
            ] {
                assert!(
                    !joined.contains(banned),
                    "{banned}: global config surface in per-run pair"
                );
            }
        }
        assert!(instruction_argv(PerRunInstructionRoute::MuseDeveloperPrompt, "").is_err());
        assert!(instruction_argv(PerRunInstructionRoute::CodexDeveloperInstructions, "").is_err());
        assert!(instruction_argv(
            PerRunInstructionRoute::CodexDeveloperInstructions,
            &"x".repeat(MAX_INSTRUCTION + 1)
        )
        .is_err());
        assert!(instruction_argv(PerRunInstructionRoute::MuseDeveloperPrompt, "a\0b").is_err());
        // Boundary text still rides: exactly MAX_INSTRUCTION bytes pass.
        assert!(instruction_argv(
            PerRunInstructionRoute::CodexDeveloperInstructions,
            &"y".repeat(MAX_INSTRUCTION)
        )
        .is_ok());
    }

    #[test]
    fn staged_prompts_stay_user_prompts() {
        // D14 honesty: adding the per-run instruction routes changes
        // nothing about staged prompt delivery — prompt-file/stdin
        // content is still delivered as the user prompt, and the relay
        // still claims no trusted channel for those shapes.
        let muse = stdio_config("muse", "1.4.9").expect("pinned muse admitted");
        assert_eq!(muse.prompt, UserPromptDelivery::PromptFile);
        let codex = stdio_config("codex", "0.161.0").expect("pinned codex admitted");
        assert_eq!(codex.prompt, UserPromptDelivery::StdinPrompt);
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
