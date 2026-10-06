use super::super::commands::{factory_muse_supervisor, muse_setup_script, muse_start_gate_script};
use super::super::paths::factory_muse_paths;
use super::common::muse_run;

#[test]
fn supervisor_shape() {
    let p = factory_muse_paths(&muse_run()).unwrap();
    let guest = "/usr/local/bin/muse-factory-1.4.2";
    let script = factory_muse_supervisor(&p, guest, "muse-spark-1.3");
    // Fixed headless entrypoint: meta provider pinned, low effort,
    // owned checkout trusted, no prompts, no sandbox, no session log.
    for marker in [
        "'/usr/local/bin/muse-factory-1.4.2' exec --provider meta",
        "--reasoning-effort low",
        "--trust-workspace",
        "--no-session-log",
        "--disable-approval",
        "--disable-sandbox",
        "--model 'muse-spark-1.3'",
        &format!("--workspace '{}'", p.checkout),
        &format!("--prompt-file '{}'", p.prompt),
        // Stream contract: answer to last-message, diagnostics log.
        &format!(">'{}' 2>>'{}'", p.output, p.stdout),
        // File backend: no inherited key may smuggle past the file.
        "unset META_API_KEY",
        &format!("[ -s '{}' ] || fail 45", p.auth),
        // Marker gate keeps the codex meanings.
        "fail 44",
        "fail 42",
        "fail 43",
        "supervisor.pid",
    ] {
        assert!(script.contains(marker), "supervisor lost {marker:?}");
    }
    assert!(!script.contains("CODEX_HOME"));
    assert!(!script.contains("META_API_KEY=\""));
    assert!(!script.contains("--output-last-message"));
    // Empty model keeps the CLI default.
    let bare = factory_muse_supervisor(&p, guest, "");
    assert!(!bare.contains("--model"));
}

#[test]
fn setup_and_gate_scripts() {
    let p = factory_muse_paths(&muse_run()).unwrap();
    let setup = muse_setup_script(&p, 1001, 1001);
    assert!(setup.contains("mkdir -p -m 700 "));
    assert!(setup.contains(&p.home));
    assert!(setup.contains(&p.muse_config));
    assert!(setup.contains("chown 1001:1001 "));
    let gate = muse_start_gate_script(&p);
    assert!(gate.contains(&p.auth));
    assert!(gate.contains(&p.prompt));
    assert!(gate.contains(&p.marker));
    assert!(gate.contains(&p.started));
}
