//! Per-run developer-instruction argv for the supervised harnesses
//! (FW33, instruction claim corrected lane-b19 per CHECKUP-012 R1).
//!
//! Staged role prompts reach the harness only as untrusted USER
//! instructions; there is no trusted high-priority instruction channel.
//! Per-run developer instructions ride the separate C11-proven `-c`
//! configuration route below. The staged prompt bytes never move there
//! on their own.

/// Staged user-prompt bytes bound (16 KiB, D14).
pub const MAX_INSTRUCTION: usize = 16 * 1024;

/// C11-proven per-run developer-instruction route for the admitted
/// harness family (evidence: `evidence/instruction-channel/`, pinned
/// `Muse Code 1.4.9`).
///
/// Muse `exec -c run.developer_prompt="<text>"` takes a JSON value and
/// records it with `developer_prompt_source=settings` (reviewed for
/// approval authorization) while sibling run config stays intact; bare
/// `developer_instructions`/`developer_prompt`, `run.base_instructions`
/// and the base-instructions-slot `run.system_prompt` are NOT this
/// route. Real-provider wire placement needs authed custody and stays
/// unproven: P6/P7 gates stay OPEN and this plumbing claims nothing
/// about them.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PerRunInstructionRoute {
    /// Muse reads per-run developer instructions from
    /// `-c run.developer_prompt`.
    MuseDeveloperPrompt,
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
/// value encoding its CLI parses (JSON for Muse).
///
/// Per-run only: the pair rides the harness argv. It writes no config
/// file, loads no global user config and overrides none — the relay
/// never points a harness at a rewritten `settings.json`
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
    let (key, body) = match route {
        PerRunInstructionRoute::MuseDeveloperPrompt => {
            ("run.developer_prompt", json_string_body(text))
        }
    };
    Ok(["-c".to_string(), format!("{key}=\"{body}\"")])
}
