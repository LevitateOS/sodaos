use base64::Engine;
use serde_json::Value as JsonValue;
use soda_release_deliver::admission::admit_qualification;
use soda_release_deliver::check::check_candidate;
use soda_release_deliver::document::{read_document, write_document};
use soda_release_deliver::finalize::Config;
use soda_release_deliver::model::{Channel, Release};
use soda_release_deliver::oci::{inspect_oci, inspect_oci_content};
use soda_release_deliver::payload::load;

use super::{
    decode_candidate, decode_channel, decode_payload, decode_release, golden_result, golden_str,
    goldens, hex_sha256, json_escape, temp_dir,
};

#[test]
fn write_document_digests_match_oracle() {
    let g = goldens();
    let channel = decode_channel(&g);
    let release = decode_release(&g);
    let dir = temp_dir("srd-doc");
    let channel_out = format!("{dir}/ch");
    let digest = write_document(&channel_out, &channel).expect("write channel");
    assert_eq!(digest, golden_str(&g, "document.channel_digest"));
    let back: Channel = read_document(&copy_as_dir(&channel_out), &digest).expect("read channel");
    assert_eq!(back, channel);

    let release_out = format!("{dir}/rel");
    let digest = write_document(&release_out, &release).expect("write release");
    assert_eq!(digest, golden_str(&g, "document.release_digest"));
    let back: Release = read_document(&copy_as_dir(&release_out), &digest).expect("read release");
    assert_eq!(back, release);
}

/// `ReadDocument` reads the skopeo `dir:` copy shape (`manifest.json` plus
/// bare blobs); adapt a `WriteDocument` OCI layout into that shape.
fn copy_as_dir(layout: &str) -> String {
    let out = format!("{layout}-copy");
    std::fs::create_dir_all(&out).unwrap();
    // manifest.json in dir: copies is the manifest blob bytes.
    let index = std::fs::read(format!("{layout}/index.json")).unwrap();
    let index: JsonValue = serde_json::from_slice(&index).unwrap();
    let digest = index["manifests"][0]["digest"].as_str().unwrap();
    let hex = digest.strip_prefix("sha256:").unwrap();
    let manifest = std::fs::read(format!("{layout}/blobs/sha256/{hex}")).unwrap();
    std::fs::write(format!("{out}/manifest.json"), &manifest).unwrap();
    for entry in std::fs::read_dir(format!("{layout}/blobs/sha256")).unwrap() {
        let entry = entry.unwrap();
        let name = entry.file_name();
        std::fs::copy(
            format!("{layout}/blobs/sha256/{}", name.to_string_lossy()),
            format!("{out}/{}", name.to_string_lossy()),
        )
        .unwrap();
    }
    out
}

#[test]
fn oci_inspection_matches_oracle() {
    let g = goldens();
    let dir = temp_dir("srd-oci");
    let revision = "d".repeat(40);

    let plain = base64::engine::general_purpose::STANDARD
        .decode(&golden_str(&g, "oci.plain"))
        .unwrap();
    let plain_path = format!("{dir}/plain.oci");
    std::fs::write(&plain_path, &plain).unwrap();
    let image = inspect_oci(&plain_path, "x86_64", &revision).expect("oci.inspect_plain");
    let want_image: JsonValue = serde_json::from_str(&golden_str(&g, "oci.plain_image")).unwrap();
    assert_eq!(
        image.manifest,
        want_image.get("Manifest").unwrap().as_str().unwrap()
    );
    assert_eq!(
        image.config,
        want_image.get("Config").unwrap().as_str().unwrap()
    );
    assert_eq!(
        image.architecture,
        want_image.get("Architecture").unwrap().as_str().unwrap()
    );
    assert_eq!(image.revision, revision);

    let (_, observed) = inspect_oci_content(
        &plain_path,
        "x86_64",
        &revision,
        &["/usr/bin/app".to_string(), "/etc/config".to_string()],
    )
    .expect("oci.content_plain");
    let want_content: JsonValue =
        serde_json::from_str(&golden_str(&g, "oci.plain_content")).unwrap();
    for (path, hash) in &observed {
        assert_eq!(
            hash,
            want_content.get(path).unwrap().as_str().unwrap(),
            "{path}"
        );
    }
    assert_eq!(observed.len(), 2);

    let gzip = base64::engine::general_purpose::STANDARD
        .decode(&golden_str(&g, "oci.gzip"))
        .unwrap();
    let gzip_path = format!("{dir}/gzip.oci");
    std::fs::write(&gzip_path, &gzip).unwrap();
    let (_, observed) = inspect_oci_content(
        &gzip_path,
        "x86_64",
        &revision,
        &["/usr/bin/app".to_string()],
    )
    .expect("oci.content_gzip");
    let want_gzip: JsonValue = serde_json::from_str(&golden_str(&g, "oci.gzip_content")).unwrap();
    assert_eq!(
        observed["/usr/bin/app"],
        want_gzip.get("/usr/bin/app").unwrap().as_str().unwrap()
    );

    let (ok, _) = golden_result(&g, "oci.inspect_empty_revision");
    assert_eq!(inspect_oci(&plain_path, "x86_64", "").is_ok(), ok);
    let (ok, err) = golden_result(&g, "oci.content_missing");
    let result = inspect_oci_content(&plain_path, "x86_64", &revision, &["/missing".to_string()]);
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
}

#[test]
fn strict_decode_edges_match_oracle() {
    let g = goldens();
    let (ok, _) = golden_result(&g, "strict.duplicate");
    let duplicate_path = format!("{}/duplicate.json", temp_dir("srd-json"));
    std::fs::write(&duplicate_path, br#"{"Format":1,"Format":2}"#).unwrap();
    assert_eq!(
        soda_release_deliver::document::read_json::<Channel>(&duplicate_path).is_ok(),
        ok
    );
    let (ok, _) = golden_result(&g, "strict.unknown");
    let unknown_path = format!("{}/unknown.json", temp_dir("srd-json"));
    std::fs::write(&unknown_path, br#"{"Format":1,"Bogus":true}"#).unwrap();
    assert_eq!(
        soda_release_deliver::document::read_json::<Channel>(&unknown_path).is_ok(),
        ok
    );
    let (ok, _) = golden_result(&g, "strict.trailing");
    let trailing_path = format!("{}/trailing.json", temp_dir("srd-json"));
    std::fs::write(&trailing_path, b"{\"Format\":1} ").unwrap();
    assert_eq!(
        soda_release_deliver::document::read_json::<Channel>(&trailing_path).is_ok(),
        ok
    );
    let (ok, _) = golden_result(&g, "strict.trailing_garbage");
    let trailing_garbage_path = format!("{}/trailing-garbage.json", temp_dir("srd-json"));
    std::fs::write(&trailing_garbage_path, b"{\"Format\":1}x").unwrap();
    assert_eq!(
        soda_release_deliver::document::read_json::<Channel>(&trailing_garbage_path).is_ok(),
        ok
    );
    let (ok, _) = golden_result(&g, "strict.nonobject");
    let nonobject_path = format!("{}/nonobject.json", temp_dir("srd-json"));
    std::fs::write(&nonobject_path, b"[1]").unwrap();
    assert_eq!(
        soda_release_deliver::document::read_json::<Channel>(&nonobject_path).is_ok(),
        ok
    );
}

#[test]
fn payload_load_matches_owner() {
    let g = goldens();
    let dir = temp_dir("srd-load");
    let path = format!("{dir}/payload.json");
    std::fs::write(&path, golden_str(&g, "payload")).unwrap();
    let payload = load(&path).expect("load");
    let (want, _) = decode_payload(&g);
    assert_eq!(payload, want);
    // Build-JSON sites use Go's unknown-field error text.
    let bad_path = format!("{dir}/bad.json");
    std::fs::write(&bad_path, r#"{"Format":3,"Bogus":1}"#).unwrap();
    let err = load(&bad_path).unwrap_err().0;
    assert!(err.contains(r#"json: unknown field "Bogus""#), "{err}");
}

#[test]
fn check_candidate_binds_before_archives() {
    let g = goldens();
    let dir = temp_dir("srd-check");
    std::fs::write(format!("{dir}/payload.json"), golden_str(&g, "payload")).unwrap();
    std::fs::write(format!("{dir}/candidate.json"), golden_str(&g, "candidate")).unwrap();
    let (payload, _) = decode_payload(&g);
    let (candidate, _) = decode_candidate(&g);
    let err = check_candidate(
        &dir,
        "aarch64",
        &payload.revision,
        &candidate.forgejo_revision,
    )
    .unwrap_err()
    .0;
    assert_eq!(
        err,
        "candidate architecture: release authority or completeness refused"
    );
    let err = check_candidate(&dir, "x86_64", &"0".repeat(40), &candidate.forgejo_revision)
        .unwrap_err()
        .0;
    assert_eq!(
        err,
        "candidate soda revision: release authority or completeness refused"
    );
    // Correct bindings reach the archive check, which fails without .oci files.
    let err = check_candidate(
        &dir,
        "x86_64",
        &payload.revision,
        &candidate.forgejo_revision,
    )
    .unwrap_err()
    .0;
    assert!(err.contains("host archive"), "{err}");
}

#[test]
fn admit_qualification_accepts_bound_evidence() {
    let g = goldens();
    let dir = temp_dir("srd-admit");
    let payload_bytes = golden_str(&g, "payload").into_bytes();
    let (payload, _) = decode_payload(&g);
    let (candidate, _) = decode_candidate(&g);
    std::fs::write(format!("{dir}/payload.json"), &payload_bytes).unwrap();
    std::fs::write(format!("{dir}/candidate.json"), golden_str(&g, "candidate")).unwrap();
    let media_path = format!("{dir}-media.json");
    std::fs::write(&media_path, golden_str(&g, "media")).unwrap();
    let evidence_path = format!("{dir}-evidence.json");
    let media: JsonValue = serde_json::from_str(&golden_str(&g, "media")).unwrap();
    let evidence = format!(
        r#"{{"Format":1,"Outcome":"passed","Scope":"native-install-upgrade-recovery","Revision":{},"Architecture":{},"PayloadSHA256":{},"HostManifest":{},"ISOSHA256":{},"RootfsSHA256":{},"Checks":[{{"Name":"install","Outcome":"passed","Detail":""}},{{"Name":"upgrade","Outcome":"passed","Detail":""}},{{"Name":"recovery","Outcome":"passed","Detail":""}},{{"Name":"preservation","Outcome":"passed","Detail":""}}],"Fixture":false}}"#,
        json_escape(&payload.revision),
        json_escape(&payload.architecture),
        json_escape(&hex_sha256(&payload_bytes)),
        json_escape(&candidate.host.manifest),
        json_escape(
            media
                .get("ISO")
                .unwrap()
                .get("SHA256")
                .unwrap()
                .as_str()
                .unwrap()
        ),
        json_escape(
            media
                .get("Rootfs")
                .unwrap()
                .get("SHA256")
                .unwrap()
                .as_str()
                .unwrap()
        ),
    );
    std::fs::write(&evidence_path, &evidence).unwrap();
    let config = Config {
        serial: 7,
        class: "normal".to_string(),
        notes: "admit me".to_string(),
        ..Config::default()
    };
    let qualification =
        admit_qualification(&config, &dir, &media_path, &evidence_path).expect("admit");
    assert_eq!(qualification.serial, 7);
    assert_eq!(qualification.scope, "native-install-upgrade-recovery");
    assert!(qualification.evidence.contains_key("qualification.json"));

    // Fixture evidence is explicitly non-qualifying.
    let bad_path = format!("{dir}-evidence-bad.json");
    std::fs::write(
        &bad_path,
        evidence.replace(r#""Fixture":false"#, r#""Fixture":true"#),
    )
    .unwrap();
    let err = admit_qualification(&config, &dir, &media_path, &bad_path)
        .unwrap_err()
        .0;
    assert_eq!(err, "fixture evidence is explicitly non-qualifying");
}
