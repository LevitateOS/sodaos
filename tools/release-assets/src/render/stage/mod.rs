//! Staging renderer (ports `scripts/stage.py`).
//!
//! Stages existing native build outputs plus configuration into a prepared
//! host context and a fresh Forgejo context; it never builds or installs.
//! Staged bytes, modes, refusals and the printed stage path match the
//! script; only the argparse envelope carries the new binary name.

mod branding;
mod files;
mod payload;

#[cfg(test)]
mod tests;

use std::path::{Path, PathBuf};

use branding::favicon_bytes;
use files::{copy, copy_tree, is_real_dir, mkdir_fresh, mkdir_leaf, normalize_tree, read_text};
use payload::{locked_terminal_assets, payload_entries, payload_source};

use crate::render::{chmod, sha256_hex};

pub use files::check_platform;

/// A script refusal (`parser.error`): the bin prints the message with the
/// usage preface and exits 2. A failure (traceback class: IO, corrupt
/// inputs, undiscoverable tree) prints the bare message and exits 1.
pub enum StageError {
    Refusal(String),
    Failure(String),
}

impl StageError {
    fn refusal(message: &str) -> StageError {
        StageError::Refusal(message.to_string())
    }

    fn failure(message: String) -> StageError {
        StageError::Failure(message)
    }
}

/// Validate the two context directories without touching the source
/// tree, like the script's upfront gates. Returns the stage directory
/// (for the trailing `print(stage)`) and the Forgejo presentation root.
pub fn check_contexts(
    host_context: &Path,
    forgejo_context: &Path,
) -> Result<(PathBuf, PathBuf), StageError> {
    let stage = host_context.join("rootfs");
    let forgejo = forgejo_context.join("forgejo");
    for directory in [&stage, forgejo_context] {
        if !is_real_dir(directory) {
            return Err(StageError::refusal(
                "real prepared host and fresh Forgejo context directories required",
            ));
        }
    }
    if std::fs::symlink_metadata(&forgejo).is_ok() {
        return Err(StageError::refusal("occupied Forgejo presentation refused"));
    }
    Ok((stage, forgejo))
}

/// Stage the tree into already-validated contexts.
pub fn run(source: &Path, arch: &str, stage: &Path, forgejo: &Path) -> Result<(), StageError> {
    let build = source.join(".artifacts/native").join(arch);

    // Programs/units/configuration are already emitted directly by Go.
    // Public, pinned tools only; runtime credentials never enter a build context.
    for name in ["muse", "muse-native", "soda-identity-compose"] {
        copy(
            stage,
            &build.join("project-tools/bin").join(name),
            &format!("/usr/share/soda/muse-tools/{name}"),
            Some(0o755),
        )?;
    }
    // Config-root branding avoids immutable /usr/share and native package conflicts.
    let brand = stage.join("etc/cockpit/branding");
    mkdir_fresh(&brand)?;
    for name in ["branding.css", "theme.css"] {
        let target = brand.join(name);
        std::fs::copy(source.join("assets/branding/cockpit").join(name), &target)
            .map_err(|e| StageError::failure(format!("cannot copy cockpit {name}: {e}")))?;
    }
    let css = brand.join("branding.css");
    let text = read_text(&css)?;
    std::fs::write(&css, text.replace("../theme/palette.css", "palette.css"))
        .map_err(|e| StageError::failure(e.to_string()))?;
    std::fs::copy(
        source.join("assets/branding/theme/palette.css"),
        brand.join("palette.css"),
    )
    .map_err(|e| StageError::failure(e.to_string()))?;
    for name in [
        "soda-symbol-brutalist.svg",
        "soda-symbol-brutalist-dark.svg",
    ] {
        std::fs::copy(
            source.join("assets/branding/source").join(name),
            brand.join(name),
        )
        .map_err(|e| StageError::failure(e.to_string()))?;
    }
    std::fs::copy(
        source.join("assets/branding/forgejo/apple-touch-icon.png"),
        brand.join("apple-touch-icon.png"),
    )
    .map_err(|e| StageError::failure(e.to_string()))?;
    // ICO is just a container: reuse the canonical rendered PNGs without redrawing.
    let mut frames = Vec::new();
    for (size, name) in [(16u8, "favicon-16.png"), (32u8, "favicon.png")] {
        let data = std::fs::read(source.join("assets/branding/forgejo").join(name))
            .map_err(|e| StageError::failure(e.to_string()))?;
        frames.push((size, data));
    }
    std::fs::write(brand.join("favicon.ico"), favicon_bytes(&frames))
        .map_err(|e| StageError::failure(e.to_string()))?;
    chmod(&brand, 0o755).map_err(StageError::failure)?;
    normalize_tree(&brand)?;
    // Adapt directly to the final Forgejo build context for vendor images.
    mkdir_fresh(forgejo)?;
    chmod(forgejo, 0o755).map_err(StageError::failure)?;
    let custom = forgejo.join("public/assets");
    copy_tree(
        &source.join("assets/branding/forgejo/css"),
        &custom.join("css"),
    )?;
    copy_tree(&source.join("assets/branding/theme"), &custom.join("theme"))?;
    let styles: Vec<PathBuf> = std::fs::read_dir(custom.join("css"))
        .map_err(|e| StageError::failure(e.to_string()))?
        .filter_map(|entry| entry.ok().map(|entry| entry.path()))
        .filter(|path| path.extension().is_some_and(|ext| ext == "css"))
        .collect();
    for stylesheet in styles {
        // The staged copy lives beside Forgejo's native css/ instead of the Soda
        // payload dir, so upstream theme imports become same-directory too.
        let text = read_text(&stylesheet)?;
        let rewritten = text
            .replace("../../theme/palette.css", "../theme/palette.css")
            .replace("../../../css/theme-forgejo-", "theme-forgejo-");
        std::fs::write(&stylesheet, rewritten).map_err(|e| StageError::failure(e.to_string()))?;
    }
    let images = custom.join("img");
    mkdir_leaf(&images)?;
    for name in ["logo.svg", "favicon.svg"] {
        std::fs::copy(
            source.join("assets/branding/source/soda-symbol-brutalist.svg"),
            images.join(name),
        )
        .map_err(|e| StageError::failure(e.to_string()))?;
    }
    for name in ["logo.png", "favicon.png", "apple-touch-icon.png"] {
        std::fs::copy(
            source.join("assets/branding/forgejo").join(name),
            images.join(name),
        )
        .map_err(|e| StageError::failure(e.to_string()))?;
    }
    // copytree/copy2 preserve checkout modes (a private worktree may be 0700/0600).
    // Normalize only this run-owned public adaptation, never canonical source assets.
    normalize_tree(&custom)?;
    // Exact reviewed presentation payload: templates, local assets, fonts/notices and
    // generated full native locale. Never copy a mutable Forgejo tree or partial hooks.
    let payload = payload_entries(source)?;
    // Branding uses the same reviewed font files/notices, not a Cockpit extension.
    for (dest, origin) in &payload {
        if let Some(relative) = dest.strip_prefix("public/assets/soda/fonts/") {
            copy(
                stage,
                &source.join(origin),
                &format!("/etc/cockpit/branding/fonts/{relative}"),
                Some(0o644),
            )?;
        }
    }
    let locked = locked_terminal_assets(source)?;
    for (name, origin) in &payload {
        let dest = Path::new(name);
        if dest.is_absolute()
            || dest
                .components()
                .any(|c| c == std::path::Component::ParentDir)
        {
            return Err(StageError::refusal("invalid Forgejo payload destination"));
        }
        let src = payload_source(source, &build, origin);
        let meta = std::fs::symlink_metadata(&src);
        if meta.is_err() || meta.is_ok_and(|m| m.file_type().is_symlink()) || !src.is_file() {
            return Err(StageError::refusal(
                "missing or unsafe Forgejo payload input",
            ));
        }
        if origin.starts_with("@build/terminal-assets/") {
            let data = std::fs::read(&src).map_err(|e| StageError::failure(e.to_string()))?;
            let file = src
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or_default();
            match locked.get(file) {
                Some(pinned) if sha256_hex(&data) == *pinned => {}
                Some(_) => {
                    return Err(StageError::refusal(
                        "terminal asset differs from locked upstream bytes",
                    ));
                }
                None => {
                    return Err(StageError::failure(format!(
                        "terminal asset {file} has no locked bytes"
                    )));
                }
            }
        }
        let target = copy(forgejo, &src, name, Some(0o644))?;
        for parent in target.ancestors().skip(1) {
            chmod(parent, 0o755).map_err(StageError::failure)?;
            if parent == forgejo {
                break;
            }
        }
    }
    // MOTD is plain text; fastfetch alone interprets the logo's color placeholders.
    copy(
        stage,
        &source.join("assets/branding/terminal/motd.txt"),
        "/etc/motd",
        Some(0o644),
    )?;
    copy(
        stage,
        &source.join("assets/branding/terminal/sodaos.txt"),
        "/usr/share/soda/fastfetch/sodaos.txt",
        Some(0o644),
    )?;
    let fastfetch = copy(
        stage,
        &source.join("assets/branding/terminal/fastfetch.jsonc"),
        "/etc/fastfetch/config.jsonc",
        Some(0o644),
    )?;
    let text = read_text(&fastfetch)?;
    std::fs::write(
        &fastfetch,
        text.replace("/usr/local/share/soda/", "/usr/share/soda/"),
    )
    .map_err(|e| StageError::failure(e.to_string()))?;
    copy(
        stage,
        &source.join("appliance/config/forgejo.env"),
        "/etc/soda/forgejo.env",
        Some(0o600),
    )?;
    copy(
        stage,
        &source.join("appliance/config/proxy.Caddyfile"),
        "/etc/soda/proxy.Caddyfile",
        Some(0o644),
    )?;
    Ok(())
}
