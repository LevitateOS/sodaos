use super::*;

#[test]
fn oracle_extension_asset_names() {
    assert!(extension::safe_extension_asset_name(r"app.js"), "H-asset-0");
    assert!(
        extension::safe_extension_asset_name(r"css/main.css"),
        "H-asset-1"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"../x.js")),
        "H-asset-2"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"/abs.js")),
        "H-asset-3"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"a/../b.js")),
        "H-asset-4"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"b\c.js")),
        "H-asset-5"
    );
    assert!(!(extension::safe_extension_asset_name(r"")), "H-asset-6");
}

#[test]
fn oracle_qualify_reason() {
    check_ok(
        "I-reason-0",
        r"Ym9vbQ==",
        recall::qualify_reason(r"  boom  ").as_bytes(),
    );
    check_ok(
        "I-reason-1",
        r"",
        recall::qualify_reason(r"COMMAND podman").as_bytes(),
    );
    check_ok(
        "I-reason-2",
        r"",
        recall::qualify_reason(r"$ echo x").as_bytes(),
    );
    check_ok("I-reason-3", r"", recall::qualify_reason(r"").as_bytes());
    check_ok(
        "I-reason-4",
        r"",
        recall::qualify_reason(r"  $ x").as_bytes(),
    );
}

#[test]
fn oracle_native_package_catalog_contract() {
    // Producer fixture: two sealed packages bind the exact catalog URLs,
    // versions, sizes, and digests the importer and Fountain validate.
    let files = vec![
        ("LICENSE".to_string(), 0o644, b"license".to_vec()),
        ("NOTICE".to_string(), 0o644, b"notice".to_vec()),
        ("backend".to_string(), 0o755, b"backend".to_vec()),
        ("extension.json".to_string(), 0o644, b"{}".to_vec()),
        ("run".to_string(), 0o755, b"#!/bin/sh\n".to_vec()),
    ];
    let package = |id: &str, seed: u8| {
        let mut archive = native_package::seal_package(&files, 1_700_000_000).unwrap();
        archive[20] = archive[20].wrapping_add(seed);
        native_package::StagedPackage {
            id: id.to_string(),
            version: "0.1.0".to_string(),
            sha256: sys::hex_sha256(&archive),
            size: archive.len() as u64,
            filename: format!("{id}-extension-0.1.0-linux-x86_64.tar.gz"),
            archive,
        }
    };
    let mut payload = model::Payload::default();
    let bytes = native_package::record_extension_catalog(
        &mut payload,
        "https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/",
        &package("soda", 0),
        &package("tailnet", 1),
    )
    .unwrap();
    assert!(bytes.ends_with(b"\n"), "J-cat-newline");
    assert_eq!(payload.extension_catalog.packages.len(), 2, "J-cat-two");
    let soda = &payload.extension_catalog.packages[0];
    assert!(
        soda.required && soda.lifecycle_controller.is_none(),
        "J-cat-soda"
    );
    assert_eq!(
        soda.url,
        "https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/soda-extension-0.1.0-linux-x86_64.tar.gz",
        "J-cat-soda-url"
    );
    let tailnet = &payload.extension_catalog.packages[1];
    assert!(
        !tailnet.required && tailnet.lifecycle_controller.as_deref() == Some("soda"),
        "J-cat-tail"
    );
}

#[test]
fn oracle_thin_host_carries_no_shared_layout() {
    // The dead shared-OCI-layout production is removed: no application
    // archive enters the host, and no v3 layout admission remains. The six
    // required images bind by identity in Payload v4; acquisition happens
    // at boot through the importer.
    for name in [
        "dashboard",
        "forgejo",
        "proxy",
        "postgres",
        "project-os",
        "canlang-runtime",
    ] {
        assert!(
            model::Role::from_name(name).is_some(),
            "J-layout-role-{name}"
        );
    }
    assert!(
        model::Role::from_name("extension").is_none(),
        "J-layout-noext"
    );
    assert!(
        model::Role::from_name("tailnet").is_none(),
        "J-layout-notail"
    );
}
