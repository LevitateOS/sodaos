#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "render_support/mod.rs"]
mod render_support;

use std::fs;
use std::path::PathBuf;

use render_support::{logo_bin, repo_root, run, TempDir};

// ---- soda-render-terminal-logo ----

fn logo_root(scratch: &TempDir) -> PathBuf {
    let root = scratch.path.join("logoroot");
    let source = root.join("assets/branding/source");
    let terminal = root.join("assets/branding/terminal");
    fs::create_dir_all(&source).unwrap();
    fs::create_dir_all(&terminal).unwrap();
    let marker = root.join("assets/branding/forgejo");
    fs::create_dir_all(&marker).unwrap();
    fs::write(marker.join("forgejo-payload.json"), "{}").unwrap();
    let repo = repo_root();
    fs::copy(
        repo.join("assets/branding/source/soda-symbol-brutalist.svg"),
        source.join("soda-symbol-brutalist.svg"),
    )
    .unwrap();
    root
}

#[test]
fn logo_help_and_usage() {
    let scratch = TempDir::new("logo-cli");
    let (code, out, _) = run(&logo_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert_eq!(out, "usage: soda-render-terminal-logo [--check]\n\nSample the canonical polygon emblem into a 32-column, 16-row ASCII mark.\n\noptions:\n  --check  verify the committed outputs instead of rewriting them\n");
    for args in [vec!["stray"], vec!["--bogus"], vec!["--check=x"]] {
        let (code, _, err) = run(&logo_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(
            err.starts_with("usage: soda-render-terminal-logo "),
            "{args:?}: {err}"
        );
    }
}

#[test]
fn logo_renders_the_canonical_emblem_byte_for_byte() {
    let scratch = TempDir::new("logo-real");
    let root = logo_root(&scratch);
    let (code, _, err) = run(&logo_bin(), &root, &[]);
    assert_eq!(code, 0, "{err}");
    let repo = repo_root();
    for name in ["sodaos.txt", "motd.txt"] {
        assert_eq!(
            fs::read(root.join("assets/branding/terminal").join(name)).unwrap(),
            fs::read(repo.join("assets/branding/terminal").join(name)).unwrap(),
            "{name} drifted from the committed branding"
        );
    }
    let (code, _, err) = run(&logo_bin(), &root, &["--check"]);
    assert_eq!(code, 0, "{err}");
    fs::write(root.join("assets/branding/terminal/sodaos.txt"), "drift").unwrap();
    let (code, _, err) = run(&logo_bin(), &root, &["--check"]);
    assert_eq!(code, 1);
    assert!(err.starts_with("Stale terminal branding: "), "{err}");
    assert!(err.ends_with("sodaos.txt\n"), "{err}");
}

#[test]
fn logo_gates_match_the_script() {
    let scratch = TempDir::new("logo-gate");
    let root = logo_root(&scratch);
    let svg_path = root.join("assets/branding/source/soda-symbol-brutalist.svg");
    let svg = fs::read_to_string(&svg_path).unwrap();
    for (name, bad, want) in [
        (
            "viewbox",
            svg.replace("0 0 128 128", "0 0 64 64"),
            "Unexpected emblem viewBox",
        ),
        (
            "layers",
            svg.replacen("#df001b", "#000000", 1),
            "Unexpected emblem layers",
        ),
        (
            "fill-rule",
            svg.replacen("evenodd", "nonzero", 1),
            "Unexpected emblem fill rule",
        ),
        (
            "geometry",
            svg.replacen(
                "M40 0H128V88L88 128H0V40Z M24 72L48 56L64 72V96H24Z",
                "M40 Q",
                1,
            ),
            "could not convert string to float: 'Q'",
        ),
    ] {
        fs::write(&svg_path, &bad).unwrap();
        let (code, _, err) = run(&logo_bin(), &root, &[]);
        assert_eq!(code, 1, "{name}");
        assert_eq!(err, format!("{want}\n"), "{name}");
    }
}
