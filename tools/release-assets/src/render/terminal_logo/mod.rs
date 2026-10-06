//! Terminal-logo renderer (ports `scripts/render-terminal-logo.py`).
//!
//! Samples the canonical polygon emblem into a 32-column, 16-row ASCII
//! mark. Geometry, sampling, colorization and gate messages match the
//! script. The script leans on a full XML parser; the port instead runs a
//! strict gate that only accepts the canonical document shape (SVG root,
//! title/desc/path children, comments) and fails closed on anything else,
//! which is exactly what the emblem gate is for: any changed syntax must
//! trip it explicitly.

mod geometry;
mod svg;

#[cfg(test)]
mod tests;

use std::path::Path;

use geometry::{polygons, render_layers};
use svg::{attr, parse_svg};

#[cfg(test)]
use geometry::{tokenize, Token};

/// Render one SVG document to `(sodaos.txt, motd.txt)`, with the script's
/// gate order: viewBox, then layers, then fill rule, then geometry.
pub fn render_svg(text: &str) -> Result<(String, String), String> {
    let (root, paths) = parse_svg(text)?;
    if attr(&root, "viewBox") != Some("0 0 128 128") {
        return Err("Unexpected emblem viewBox".to_string());
    }
    // `findall` only sees the SVG namespace: without the default binding
    // there are no layers, and the layers gate fires first, like the owner.
    let paths = if attr(&root, "xmlns") == Some("http://www.w3.org/2000/svg") {
        paths
    } else {
        Vec::new()
    };
    let fills: Vec<Option<&str>> = paths.iter().map(|attrs| attr(attrs, "fill")).collect();
    if fills.as_slice() != [Some("#df001b"), Some("#101010")] {
        return Err("Unexpected emblem layers".to_string());
    }
    if paths.is_empty()
        || paths
            .iter()
            .any(|attrs| attr(attrs, "fill-rule") != Some("evenodd"))
    {
        return Err("Unexpected emblem fill rule".to_string());
    }
    let mut layers = Vec::new();
    for attrs in &paths {
        let data = attr(attrs, "d").ok_or("Emblem path has no geometry".to_string())?;
        layers.push(polygons(data)?);
    }
    Ok(render_layers(&layers))
}

/// Render the canonical emblem, or `--check` the committed outputs.
pub fn run(root: &Path, check: bool) -> Result<(), String> {
    let source = root.join("assets/branding/source/soda-symbol-brutalist.svg");
    let out = root.join("assets/branding/terminal");
    let svg = std::fs::read_to_string(&source)
        .map_err(|e| format!("cannot read {}: {e}", source.display()))?;
    let (colored, plain) = render_svg(&svg)?;
    for (name, text) in [("sodaos.txt", colored), ("motd.txt", plain)] {
        let path = out.join(name);
        if check {
            let current = std::fs::read_to_string(&path)
                .map_err(|e| format!("cannot read {}: {e}", path.display()))?;
            if current != text {
                return Err(format!("Stale terminal branding: {}", path.display()));
            }
        } else {
            std::fs::write(&path, text)
                .map_err(|e| format!("cannot write {}: {e}", path.display()))?;
        }
    }
    Ok(())
}
