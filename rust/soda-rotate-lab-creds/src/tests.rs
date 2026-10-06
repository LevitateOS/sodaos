use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;

use crate::fixture_authority::random_hex_passphrase;
use crate::inventory::{first_live_inputs, note, stat_line};
use crate::process::{config_json, json_escape, trust_json};

#[test]
fn json_escape_matches_python_dumps() {
    let input = "a\"b\\c\nd\re\tf\x08g\x0ch\x01i\x7fj\u{80}ké😀l\x1fm";
    assert_eq!(
        json_escape(input),
        "a\\\"b\\\\c\\nd\\re\\tf\\bg\\fh\\u0001i\\u007fj\\u0080k\\u00e9\\ud83d\\ude00l\\u001fm"
    );
}

#[test]
fn trust_json_matches_python_dump_with_trailing_newline() {
    let got = trust_json(
        "ghcr.io/levitateos/sodaos",
        424242,
        [
            "artifact-pub\n",
            "candidate-pub\n",
            "preview-pub\n",
            "stable-pub\n",
        ],
    );
    let want = "{\n  \"Format\": 1,\n  \"Prefix\": \"ghcr.io/levitateos/sodaos\",\n  \"Epoch\": 1,\n  \"Keys\": {\n    \"artifact\": [\n      \"artifact-pub\\n\"\n    ],\n    \"candidate\": [\n      \"candidate-pub\\n\"\n    ],\n    \"preview\": [\n      \"preview-pub\\n\"\n    ],\n    \"stable\": [\n      \"stable-pub\\n\"\n    ]\n  },\n  \"NotBefore\": 423642,\n  \"MaxAgeSeconds\": 3600,\n  \"ClockSkewSeconds\": 10,\n  \"MinimumSequence\": {\n    \"candidate\": 1,\n    \"preview\": 1,\n    \"stable\": 1\n  }\n}\n";
    assert_eq!(got, want);
}

#[test]
fn config_json_matches_python_dump_with_trailing_newline() {
    let got = config_json();
    let want = "{\n  \"Trust\": \"/run/soda-media-authority/trust.json\",\n  \"Keys\": {\n    \"Key\": \"/run/soda-media-authority/artifact.private\",\n    \"Passphrase\": \"/run/soda-media-authority/passphrase\"\n  }\n}\n";
    assert_eq!(got, want);
}

#[test]
fn note_pads_tag_to_seven() {
    assert_eq!(note("PASS", "/x [s]"), "PASS    /x [s]");
    assert_eq!(note("SKIP", "m"), "SKIP    m");
    assert_eq!(note("INFO", "m"), "INFO    m");
}

#[test]
fn stat_line_reports_modes_without_content() {
    let dir = env::temp_dir().join(format!("soda-rotate-stat-{}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).unwrap();
    let secret = dir.join("secret.token");
    fs::write(&secret, b"SYNTHETIC_SECRET_MARKER").unwrap();
    fs::set_permissions(&secret, fs::Permissions::from_mode(0o600)).unwrap();
    let path = secret.to_string_lossy().into_owned();
    let pass = stat_line(&path, "600").unwrap();
    assert!(pass.starts_with("PASS   "), "{pass}");
    assert!(pass.contains(&path), "{pass}");
    assert!(!pass.contains("SYNTHETIC_SECRET_MARKER"), "{pass}");
    let warn = stat_line(&path, "640").unwrap();
    assert!(warn.starts_with("WARN   "), "{warn}");
    assert!(warn.contains("mode is 600, want 640"), "{warn}");
    assert!(!warn.contains("SYNTHETIC_SECRET_MARKER"), "{warn}");
    fs::set_permissions(&secret, fs::Permissions::from_mode(0o640)).unwrap();
    let pass640 = stat_line(&path, "640").unwrap();
    assert!(pass640.starts_with("PASS   "), "{pass640}");
    let missing = dir.join("absent.token").to_string_lossy().into_owned();
    let skip = stat_line(&missing, "600").unwrap();
    assert!(skip.starts_with("SKIP   "), "{skip}");
    assert!(skip.contains("absent or not visible to"), "{skip}");
    let _ = fs::remove_dir_all(&dir);
}

#[test]
fn first_live_inputs_takes_sorted_first_match() {
    let root = env::temp_dir().join(format!("soda-rotate-glob-{}", std::process::id()));
    let _ = fs::remove_dir_all(&root);
    let dir = root.join(".artifacts/releases/isolated");
    fs::create_dir_all(&dir).unwrap();
    let base = root.to_string_lossy().into_owned();
    assert_eq!(first_live_inputs(&base), None);
    fs::write(dir.join("soda-live-inputs-b.json"), b"b").unwrap();
    fs::write(dir.join("soda-live-inputs-a.json"), b"a").unwrap();
    fs::write(dir.join("notes.txt"), b"n").unwrap();
    assert_eq!(
        first_live_inputs(&base),
        Some(format!(
            "{}/.artifacts/releases/isolated/soda-live-inputs-a.json",
            base
        ))
    );
    let _ = fs::remove_dir_all(&root);
}

#[test]
fn first_live_inputs_skips_dangling_symlinks() {
    use std::os::unix::fs::symlink;
    let root = env::temp_dir().join(format!("soda-rotate-dangle-{}", std::process::id()));
    let _ = fs::remove_dir_all(&root);
    let dir = root.join(".artifacts/releases/isolated");
    fs::create_dir_all(&dir).unwrap();
    let base = root.to_string_lossy().into_owned();
    symlink("nowhere-target.json", dir.join("soda-live-inputs-zz.json")).unwrap();
    assert_eq!(first_live_inputs(&base), None);
    fs::write(dir.join("soda-live-inputs-aa.json"), b"a").unwrap();
    assert_eq!(
        first_live_inputs(&base),
        Some(format!(
            "{}/.artifacts/releases/isolated/soda-live-inputs-aa.json",
            base
        ))
    );
    let _ = fs::remove_dir_all(&root);
}

#[test]
fn passphrase_is_64_lowercase_hex_without_newline() {
    let passphrase = random_hex_passphrase().unwrap();
    assert_eq!(passphrase.len(), 64);
    assert!(passphrase
        .bytes()
        .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
}
