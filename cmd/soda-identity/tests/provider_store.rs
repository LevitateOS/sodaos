// V2-A: every enabled broker section resolves new sessions from the shared
// provider store, including sections that still carry legacy binary/sha256
// pins. Store mode accepts (and ignores) those pins for new sessions;
// explicit-pin validation is unchanged byte-for-byte. The single test below
// exercises the real filesystem behavior (admit, bind, tamper refusal,
// stale refusal, retained binding) on an isolated store; constructor-only
// checks are not kept (unit-test ban).
use soda_identity::providers::{codex, muse, resolution};
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

/// Stale-on-purpose legacy pin: well-formed but names no admitted bundle.
const LEGACY_BINARY: &str = "/opt/stale-install/provider-cli";
const LEGACY_SHA: &str = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef";

/// One tmpfs scratch dir holding both the isolated store and the private
/// enrollment root. tmpfs is a production requirement for enrollment roots
/// (the broker requires a private root); all scratch stays off the root disk.
fn scratch() -> Option<(tempfile::TempDir, PathBuf, PathBuf)> {
    let dir = match tempfile::TempDir::new_in("/dev/shm") {
        Ok(dir) => dir,
        Err(e) => {
            eprintln!("SKIP detail: new_in /dev/shm failed: {e}");
            return None;
        }
    };
    let store_root = dir.path().join("store");
    let enroll_root = dir.path().join("enroll");
    std::fs::create_dir_all(&store_root).expect("store dir");
    std::fs::create_dir_all(&enroll_root).expect("enroll dir");
    // Production enrollment roots must be private; enforce it explicitly.
    std::fs::set_permissions(&enroll_root, std::fs::Permissions::from_mode(0o700))
        .expect("chmod enroll root");
    Some((dir, store_root, enroll_root))
}

fn sha256_hex(bytes: &[u8]) -> String {
    use sha2::Digest as _;
    let mut hasher = sha2::Sha256::new();
    hasher.update(bytes);
    hasher
        .finalize()
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect()
}

fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// Admit a synthetic bundle through the public store APIs only (same shape
/// as the maintained review driver): real files, real admission record,
/// real digest inventory. The executable is never spawned here; resolution
/// and verification are what bind it.
fn admit_bundle(
    root: &Path,
    provider: soda_provider_tools::Provider,
    version: &str,
) -> soda_provider_tools::ReleaseIdentity {
    let content = format!(
        "#!/bin/sh\necho lane-v2-a-{}-{version}\n",
        provider.as_str()
    )
    .into_bytes();
    let digest = sha256_hex(&content);
    let bundle = root.join("bundles").join(provider.as_str()).join(&digest);
    let bin = bundle.join("bin");
    std::fs::create_dir_all(&bin).expect("bin dir");
    let exe_name = provider.executable_relpath().rsplit('/').next().unwrap();
    let exe = bin.join(exe_name);
    std::fs::write(&exe, &content).expect("exe bytes");
    let mut perms = std::fs::metadata(&exe).expect("stat").permissions();
    perms.set_mode(0o755);
    std::fs::set_permissions(&exe, perms).expect("chmod");
    if provider == soda_provider_tools::Provider::Codex {
        let companion = bin.join("codex-code-mode-host");
        std::fs::write(&companion, b"#!/bin/sh\necho companion\n").expect("companion bytes");
        let mut perms = std::fs::metadata(&companion).expect("stat").permissions();
        perms.set_mode(0o755);
        std::fs::set_permissions(&companion, perms).expect("chmod");
    }
    let release = soda_provider_tools::ReleaseIdentity {
        provider,
        version: version.to_string(),
        digest,
        size: content.len() as u64,
        artifact_url: "https://example.invalid/lane-v2-a".to_string(),
    };
    soda_provider_tools::store::write_release_record(&bundle, &release).expect("admission record");
    soda_provider_tools::store::record_inventory(&bundle, &release).expect("inventory");
    let now = now_secs();
    soda_provider_tools::store::write_selection(
        root,
        &soda_provider_tools::Selection {
            provider,
            release: release.clone(),
            checked_at_secs: now,
            admitted_at_secs: now,
            freshness: soda_provider_tools::Freshness::Current,
        },
    )
    .expect("selection");
    release
}

/// Read one selection through the public store API (never piecemeal).
fn read_selection(
    root: &Path,
    provider: soda_provider_tools::Provider,
) -> soda_provider_tools::Selection {
    soda_provider_tools::store::read_selection(root, provider)
        .expect("selection reads")
        .expect("selection admitted")
}

/// New sessions from legacy-pinned sections resolve ONE coherent store
/// snapshot; tampered bytes and known-stale selections refuse new work while
/// the retained bundle still binds exactly for existing work.
#[test]
fn legacy_pins_cannot_shadow_store_selection() {
    let Some((_scratch, root, enroll_dir)) = scratch() else {
        eprintln!("SKIP: no tmpfs scratch available");
        return;
    };
    // Isolated store, no network: the freshly written snapshot is within its
    // check TTL, so resolution never attempts an upstream refresh and no
    // credentials or model calls are involved at any point.
    std::env::set_var("SODA_PROVIDERS_ROOT", &root);
    let codex_release = admit_bundle(
        &root,
        soda_provider_tools::Provider::Codex,
        "rust-v9.9.9-lane-v2-a",
    );
    let muse_release = admit_bundle(
        &root,
        soda_provider_tools::Provider::Muse,
        "9.9.9-lane-v2-a",
    );

    let enroll_root = enroll_dir.to_string_lossy().into_owned();
    // Exactly how open_broker now constructs enabled legacy sections.
    let codex = codex::Provider::new(codex::Config {
        binary: LEGACY_BINARY.to_string(),
        version: "rust-v0.1.0".to_string(),
        sha256: LEGACY_SHA.to_string(),
        root: enroll_root.clone(),
        store: true,
    })
    .expect("legacy codex section constructs in store mode");
    let muse = muse::Provider::new(muse::Config {
        binary: LEGACY_BINARY.to_string(),
        version: "0.0.0-stale".to_string(),
        sha256: LEGACY_SHA.to_string(),
        root: enroll_root,
        store: true,
    })
    .expect("legacy muse section constructs in store mode");

    // The actual configured binary path verifies against the isolated
    // admitted store: live bytes match the admission-time inventory entry.
    for (provider, release) in [
        (soda_provider_tools::Provider::Codex, &codex_release),
        (soda_provider_tools::Provider::Muse, &muse_release),
    ] {
        let selection = read_selection(&root, provider);
        assert_eq!(selection.release.digest, release.digest);
        let admitted = selection.executable(&root);
        let live = std::fs::read(&admitted).expect("admitted binary readable");
        let inventory: soda_provider_tools::store::Inventory = serde_json::from_slice(
            &std::fs::read(
                root.join("bundles")
                    .join(provider.as_str())
                    .join(&release.digest)
                    .join("inventory.json"),
            )
            .expect("inventory readable"),
        )
        .expect("inventory parses");
        let entry = inventory
            .entries
            .iter()
            .find(|e| Path::new(&e.path) == Path::new(provider.executable_relpath()))
            .expect("executable inventoried");
        assert_eq!(sha256_hex(&live), entry.sha256);
        assert_eq!(live.len() as u64, entry.size);
    }

    // One coherent snapshot per provider: binary + version + digest + ref.
    let Some(mut codex_bound) = codex.bind_new().expect("codex binds") else {
        panic!("store mode must bind the shared codex selection");
    };
    assert!(codex_bound.via_store);
    assert!(codex_bound.token.is_some(), "binding must pin the bundle");
    assert_eq!(
        codex_bound.binary,
        read_selection(&root, soda_provider_tools::Provider::Codex).executable(&root),
        "new codex session must spawn the admitted bundle, not the legacy pin"
    );
    assert_eq!(codex_bound.version, codex_release.version);
    assert_eq!(codex_bound.digest, codex_release.digest);
    let Some(mut muse_bound) = muse.bind_new().expect("muse binds") else {
        panic!("store mode must bind the shared muse selection");
    };
    assert_eq!(muse_bound.version, muse_release.version);
    assert_eq!(muse_bound.digest, muse_release.digest);
    resolution::release_bound(&mut codex_bound);
    resolution::release_bound(&mut muse_bound);

    // Tampered bytes fail closed: the legacy pin cannot substitute.
    let exe = read_selection(&root, soda_provider_tools::Provider::Codex).executable(&root);
    let original = std::fs::read(&exe).expect("admitted binary readable");
    let mut tampered = original.clone();
    tampered[0] ^= 0xff;
    std::fs::write(&exe, &tampered).expect("tamper writes");
    assert!(
        codex.bind_new().is_err(),
        "tampered bundle must refuse new work"
    );
    std::fs::write(&exe, &original).expect("restore writes");

    // A known-stale selection cannot be bypassed with the legacy pin: new
    // work refuses while the retained bundle stays exactly bindable.
    let mut record: serde_json::Value = serde_json::from_slice(
        &std::fs::read(root.join("current-codex.json")).expect("selection record"),
    )
    .expect("selection parses");
    record["freshness"] = serde_json::json!({"state": "stale", "latest": "review-known-newer"});
    std::fs::write(
        root.join("current-codex.json"),
        serde_json::to_vec_pretty(&record).expect("selection serializes"),
    )
    .expect("stale selection writes");
    let blocked = codex
        .bind_new()
        .expect_err("known-stale must refuse new work");
    assert!(
        blocked.to_string().contains("blocks new"),
        "refusal must name the block, got: {blocked}"
    );
    let mut retained =
        resolution::resolve_existing(resolution::ProviderKind::Codex, &codex_release.digest)
            .expect("retained bundle still binds exactly");
    assert_eq!(
        retained.binary,
        read_selection(&root, soda_provider_tools::Provider::Codex).executable(&root)
    );
    assert_eq!(retained.digest, codex_release.digest);
    resolution::release_bound(&mut retained);
}
