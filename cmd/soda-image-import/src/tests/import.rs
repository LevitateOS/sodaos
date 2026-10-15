use super::super::payload::load as load_payload;
use super::super::*;
use super::fixtures::*;

fn required_configs(payload: &Payload) -> Vec<String> {
    import::selected(payload, ImageSet::Required)
        .into_iter()
        .map(|(_, binding)| binding.config.clone())
        .collect()
}

#[test]
fn required_set_verifies_present_store_without_pulls() {
    let root = test_root("required-present");
    let payload = load_payload(&write_release(&root)).expect("load");
    let configs = required_configs(&payload);
    let present: Vec<&str> = configs.iter().map(String::as_str).collect();
    let mut store = FakeStore::new(&payload, &present);
    let cancelled = AtomicBool::new(false);
    let ctx = test_ctx(&cancelled);
    ensure_images(
        &payload,
        ImageSet::Required,
        "/usr/bin/podman",
        &ctx,
        &mut |cmd, args| store.run(cmd, args),
    )
    .expect("ensure");
    assert!(store.pulls.is_empty());
    assert_eq!(store.queries, configs.len());
}

#[test]
fn required_set_pulls_only_missing_references_then_verifies() {
    let root = test_root("required-missing");
    let payload = load_payload(&write_release(&root)).expect("load");
    let configs = required_configs(&payload);
    let have: Vec<&str> = configs[..2].iter().map(String::as_str).collect();
    let mut store = FakeStore::new(&payload, &have);
    let cancelled = AtomicBool::new(false);
    let ctx = test_ctx(&cancelled);
    ensure_images(
        &payload,
        ImageSet::Required,
        "/usr/bin/podman",
        &ctx,
        &mut |cmd, args| store.run(cmd, args),
    )
    .expect("ensure");
    // Pulled references are the exact sealed references of the missing four.
    let missing: Vec<String> = import::selected(&payload, ImageSet::Required)
        .into_iter()
        .skip(2)
        .map(|(_, binding)| binding.reference.clone())
        .collect();
    assert_eq!(store.pulls, missing);
}

#[test]
fn tailnet_set_selects_only_the_official_binding() {
    let root = test_root("tailnet-set");
    let payload = load_payload(&write_release(&root)).expect("load");
    let mut store = FakeStore::new(&payload, &[]);
    let cancelled = AtomicBool::new(false);
    let ctx = test_ctx(&cancelled);
    ensure_images(
        &payload,
        ImageSet::Tailnet,
        "/usr/bin/podman",
        &ctx,
        &mut |cmd, args| store.run(cmd, args),
    )
    .expect("ensure");
    assert_eq!(store.pulls.len(), 1);
    assert!(store.pulls[0].contains("docker.io/tailscale/tailscale@sha256:"));
    let selected = import::selected(&payload, ImageSet::Tailnet);
    assert_eq!(selected.len(), 1);
    assert_eq!(selected[0].0, "tailscale");
}

#[test]
fn inspect_mismatch_and_failed_calls_are_distinct() {
    let root = test_root("inspect-mismatch");
    let payload = load_payload(&write_release(&root)).expect("load");
    let configs = required_configs(&payload);
    let present: Vec<&str> = configs.iter().map(String::as_str).collect();
    let mut store = FakeStore::new(&payload, &present);
    store.corrupt_inspect(&configs[0]);
    let cancelled = AtomicBool::new(false);
    let ctx = test_ctx(&cancelled);
    let err = ensure_images(
        &payload,
        ImageSet::Required,
        "/usr/bin/podman",
        &ctx,
        &mut |cmd, args| store.run(cmd, args),
    )
    .unwrap_err();
    assert!(err.contains("OCI identity mismatch"), "unexpected: {err}");

    let mut failing = |_: &str, _: &[String]| (PodmanOutcome::Failed, Vec::new());
    let err = ensure_images(
        &payload,
        ImageSet::Required,
        "/usr/bin/podman",
        &ctx,
        &mut failing,
    )
    .unwrap_err();
    assert!(err.contains("observation failed"), "unexpected: {err}");
}

#[test]
fn inspect_capture_drains_beyond_pipe_capacity_within_limit() {
    use std::os::unix::fs::PermissionsExt;

    // Fake Podman emits its whole stdout before exiting: once the stream
    // exceeds OS pipe capacity, a wait-before-drain parent deadlocks with
    // the writer blocked. The 30s bound below fails such a parent instead
    // of waiting out the 600s importer deadline.
    let root = test_root("inspect-pipe");
    let fake = root.join("podman");
    fs::write(&fake, "#!/bin/sh\nexec head -c \"$1\" /dev/zero\n").expect("write fake");
    let mut perms = fs::metadata(&fake).expect("stat fake").permissions();
    perms.set_mode(0o755);
    fs::set_permissions(&fake, perms).expect("chmod fake");
    let podman = fake.to_str().expect("fake path").to_string();

    let cancelled = AtomicBool::new(false);
    let ctx = ImportCtx {
        deadline: Instant::now() + Duration::from_secs(30),
        cancelled: &cancelled,
    };

    // 128KiB exceeds pipe capacity but stays within the 256KiB cap: passes.
    let (outcome, output) = import::run_podman_output(&podman, &["131072".to_string()], &ctx);
    assert_eq!(outcome, PodmanOutcome::Code(0));
    assert_eq!(output.len(), 131072);

    // One byte over the cap is rejected with no captured bytes; the blocked
    // writer is killed and reaped, not waited out.
    let (outcome, output) = import::run_podman_output(&podman, &["262145".to_string()], &ctx);
    assert_eq!(outcome, PodmanOutcome::Failed);
    assert!(output.is_empty());

    // A cancelled context kills and reaps a slow child with no bytes.
    fs::write(&fake, "#!/bin/sh\nexec sleep 60\n").expect("write slow fake");
    let held = AtomicBool::new(true);
    let cancelled_ctx = ImportCtx {
        deadline: Instant::now() + Duration::from_secs(30),
        cancelled: &held,
    };
    let (outcome, output) = import::run_podman_output(&podman, &[], &cancelled_ctx);
    assert_eq!(outcome, PodmanOutcome::Failed);
    assert!(output.is_empty());
}

#[test]
fn cancel_and_deadline_stay_distinct_from_identity_failure() {
    let root = test_root("cancel-deadline");
    let payload = load_payload(&write_release(&root)).expect("load");
    let cancelled = AtomicBool::new(true);
    let ctx = test_ctx(&cancelled);
    let mut store = FakeStore::new(&payload, &[]);
    let err = ensure_images(
        &payload,
        ImageSet::Required,
        "/usr/bin/podman",
        &ctx,
        &mut |cmd, args| store.run(cmd, args),
    )
    .unwrap_err();
    assert_eq!(err, "context canceled");
}
