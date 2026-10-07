use super::content::{resolve_oci_members, scan_archive_layer, scan_oci_archive_layers};
use super::layers::{scan_oci_layer, LayerMember};
use super::manifest::{parse_oci_manifest, read_oci_index};
use super::*;
use crate::sha256_hex;
use crate::test_support::{fixture_oci_bytes, FIXTURE_REVISION};
use std::collections::HashMap;
use std::io::Write;

/// Hand-crafted ustar entry for names `tar::Builder` refuses (`..`).
fn raw_tar_entry(name: &str, body: &[u8]) -> Vec<u8> {
    let mut header = [0u8; 512];
    header[..name.len()].copy_from_slice(name.as_bytes());
    header[100..108].copy_from_slice(b"0000644\0");
    header[108..116].copy_from_slice(b"0000000\0");
    header[116..124].copy_from_slice(b"0000000\0");
    let size = format!("{:011o}\0", body.len());
    header[124..136].copy_from_slice(size.as_bytes());
    header[136..148].copy_from_slice(b"00000000000\0");
    header[148..156].copy_from_slice(b"        ");
    header[156] = b'0';
    header[257..262].copy_from_slice(b"ustar");
    let sum: u32 = header.iter().map(|b| *b as u32).sum();
    let chksum = format!("{:06o}\0 ", sum);
    header[148..156].copy_from_slice(chksum.as_bytes());
    let mut out = Vec::new();
    out.extend_from_slice(&header);
    out.extend_from_slice(body);
    out.resize(out.len() + (512 - body.len() % 512) % 512, 0);
    out.extend_from_slice(&[0u8; 1024]);
    out
}

fn write_fixture(name: &str, bytes: &[u8]) -> std::path::PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "soda-oci-{}-{}",
        std::process::id(),
        std::sync::atomic::AtomicU64::new(0).fetch_add(1, std::sync::atomic::Ordering::SeqCst)
    ));
    let _ = std::fs::create_dir_all(&dir);
    let path = dir.join(name);
    std::fs::write(&path, bytes).unwrap();
    path
}

#[test]
fn marshal_compact_matches_go_encoder_bytes() {
    let image = Image {
        manifest: "m\"anifest".to_string(),
        config: "a<b".to_string(),
        architecture: "x&y".to_string(),
        revision: "l1\nl2".to_string(),
        source: "héllo".to_string(),
        base_name: "plain".to_string(),
        base_digest: "d".to_string(),
    };
    let doc = image.marshal_compact();
    assert_eq!(
        doc,
        "{\"Manifest\":\"m\\\"anifest\",\"Config\":\"a\\u003cb\",\"Architecture\":\"x\\u0026y\",\"Revision\":\"l1\\nl2\",\"Source\":\"héllo\",\"BaseName\":\"plain\",\"BaseDigest\":\"d\"}"
    );
    assert!(!doc.ends_with('\n'));
}

#[test]
fn oracle_identity_and_wrong_platform() {
    // Oracle: TestOCIIdentityAndWrongPlatform over fixtureOCI.
    let file = write_fixture("image.oci", &fixture_oci_bytes("amd64"));
    let image = inspect_oci(&file, "x86_64", FIXTURE_REVISION).unwrap();
    assert_eq!(image.architecture, "amd64");
    assert_eq!(image.revision, FIXTURE_REVISION);
    assert_eq!(image.base_name, "synthetic-base");
    assert_eq!(
        inspect_oci(&file, "aarch64", FIXTURE_REVISION)
            .unwrap_err()
            .message(),
        "expected x86_64"
    );
    assert_eq!(
        inspect_oci(&file, "x86_64", &"c".repeat(40))
            .unwrap_err()
            .message(),
        "OCI source revision mismatch"
    );
}

#[test]
fn oracle_content_members() {
    // Oracle: InspectOCIContent resolves fixture.txt through the overlay.
    let file = write_fixture("content.oci", &fixture_oci_bytes("amd64"));
    let (image, content) = inspect_oci_content(
        &file,
        "x86_64",
        FIXTURE_REVISION,
        &[String::from("/fixture.txt")],
    )
    .unwrap();
    assert_eq!(image.revision, FIXTURE_REVISION);
    assert_eq!(
        content.get("/fixture.txt").unwrap(),
        &sha256_hex(b"synthetic layer fixture; never executed")
    );
    assert!(inspect_oci_content(
        &file,
        "x86_64",
        FIXTURE_REVISION,
        &[String::from("/missing")]
    )
    .is_err());
}

#[test]
fn oracle_layer_scanner_vectors() {
    // Oracle: TestOCIMemberScanner* whiteout/removal/blocked outcomes.
    let layer_bytes = |names: &[&str]| -> Vec<u8> {
        let mut out = Vec::new();
        let mut writer = tar::Builder::new(&mut out);
        for name in names {
            let mut header = tar::Header::new_ustar();
            header.set_size(name.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            writer
                .append_data(&mut header, *name, name.as_bytes())
                .unwrap();
        }
        writer.into_inner().unwrap();
        out
    };
    let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
        .into_iter()
        .collect();
    let dup = layer_bytes(&["wanted", "wanted"]);
    assert_eq!(
        scan_oci_layer(&mut &dup[..], &wanted)
            .unwrap_err()
            .message(),
        "duplicate OCI layer entry"
    );
    let evil = raw_tar_entry("../wanted", b"evil");
    assert_eq!(
        scan_oci_layer(&mut &evil[..], &wanted)
            .unwrap_err()
            .message(),
        "unsafe OCI layer path"
    );
    let benign = layer_bytes(&[".", "wanted"]);
    let members = scan_oci_layer(&mut &benign[..], &wanted).unwrap();
    assert!(members.get("wanted").unwrap().present);
    // Whiteout in the upper layer removes the lower member.
    let lower: HashMap<String, LayerMember> = [(
        "wanted".to_string(),
        LayerMember {
            hash: "a".repeat(64),
            present: true,
            blocked: false,
        },
    )]
    .into_iter()
    .collect();
    let upper_bytes = layer_bytes(&[".wh.wanted"]);
    let upper = scan_oci_layer(&mut &upper_bytes[..], &wanted).unwrap();
    assert_eq!(
        resolve_oci_members(&[lower, upper], &[false, false], &wanted)
            .unwrap_err()
            .message(),
        "requested OCI member was removed"
    );
}

#[test]
fn oracle_gzip_and_zstd_layers() {
    // Oracle: TestOCIArchiveLayerCompressionAndDigest outcomes.
    use flate2::write::GzEncoder;
    let content = b"verified member";
    let mut layer = Vec::new();
    {
        let mut writer = tar::Builder::new(&mut layer);
        let mut header = tar::Header::new_ustar();
        header.set_size(content.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        writer
            .append_data(&mut header, "wanted", &content[..])
            .unwrap();
        writer.into_inner().unwrap();
    }
    let mut compressed = Vec::new();
    {
        let mut gz = GzEncoder::new(&mut compressed, flate2::Compression::default());
        gz.write_all(&layer).unwrap();
        gz.finish().unwrap();
    }
    let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
        .into_iter()
        .collect();
    for (media, data, blocked) in [
        (LAYER_TAR, layer.clone(), false),
        (LAYER_TAR_GZIP, compressed.clone(), false),
        (LAYER_TAR_ZSTD, compressed.clone(), true),
    ] {
        let sum = sha256_hex(&data);
        let desc = Descriptor {
            digest: format!("sha256:{sum}"),
            size: data.len() as i64,
            media_type: media.to_string(),
            ..Descriptor::default()
        };
        let mut archive = Vec::new();
        {
            let mut writer = tar::Builder::new(&mut archive);
            let name = format!("blobs/sha256/{sum}");
            let mut header = tar::Header::new_ustar();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            writer
                .append_data(&mut header, name.as_str(), &data[..])
                .unwrap();
            writer.into_inner().unwrap();
        }
        let (found, unsupported) =
            scan_oci_archive_layers(&mut &archive[..], &[desc], &wanted).unwrap();
        assert_eq!(unsupported[0], blocked);
        if !blocked {
            assert_eq!(found[0].get("wanted").unwrap().hash, sha256_hex(content));
        }
    }
    // Corrupt gzip trailer with a matching blob digest must still fail.
    let mut corrupt = compressed.clone();
    let last = corrupt.len() - 1;
    corrupt[last] ^= 0xff;
    let sum = sha256_hex(&corrupt);
    let desc = Descriptor {
        digest: format!("sha256:{sum}"),
        size: corrupt.len() as i64,
        media_type: LAYER_TAR_GZIP.to_string(),
        ..Descriptor::default()
    };
    assert!(scan_archive_layer(&mut &corrupt[..], &desc, &wanted).is_err());
}

const MEDIA_TYPE_DIGEST: &str =
    "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

fn index_entries(media_fragment: &str) -> HashMap<String, Blob> {
    let layout = Blob {
        hash: String::new(),
        size: 0,
        data: Some(br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec()),
    };
    let index = format!(
        r#"{{"schemaVersion":2,"mediaType":{media_fragment},"manifests":[{{"digest":"{MEDIA_TYPE_DIGEST}","size":1,"mediaType":"{MANIFEST_MEDIA_TYPE}"}}]}}"#
    );
    HashMap::from([
        ("oci-layout".to_string(), layout),
        (
            "index.json".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(index.into_bytes()),
            },
        ),
    ])
}

fn manifest_doc(media_fragment: &str) -> Vec<u8> {
    format!(
        r#"{{"schemaVersion":2,"mediaType":{media_fragment},"config":{{"digest":"{MEDIA_TYPE_DIGEST}","size":1,"mediaType":"{CONFIG_MEDIA_TYPE}"}},"layers":[]}}"#
    )
    .into_bytes()
}

#[test]
fn index_gate_refuses_wrong_type_outer_media_type() {
    for fragment in ["7", "{}", "[]", "true"] {
        assert_eq!(
            read_oci_index(&index_entries(fragment))
                .unwrap_err()
                .message(),
            "valid OCI index required",
            "outer mediaType {fragment} must refuse"
        );
    }
}

#[test]
fn manifest_gate_refuses_wrong_type_outer_media_type() {
    for fragment in ["7", "{}", "[]", "true"] {
        assert_eq!(
            parse_oci_manifest(&manifest_doc(fragment))
                .err()
                .expect("must refuse")
                .message(),
            "invalid OCI image manifest",
            "outer mediaType {fragment} must refuse"
        );
    }
}

#[test]
fn outer_media_type_absence_keeps_lenient_behavior() {
    let entries = HashMap::from([
        (
            "oci-layout".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec()),
            },
        ),
        (
            "index.json".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(
                    format!(
                        r#"{{"schemaVersion":2,"manifests":[{{"digest":"{MEDIA_TYPE_DIGEST}","size":1,"mediaType":"{MANIFEST_MEDIA_TYPE}"}}]}}"#
                    )
                    .into_bytes(),
                ),
            },
        ),
    ]);
    assert!(read_oci_index(&entries).is_ok());
    let doc = format!(
        r#"{{"schemaVersion":2,"config":{{"digest":"{MEDIA_TYPE_DIGEST}","size":1,"mediaType":"{CONFIG_MEDIA_TYPE}"}},"layers":[]}}"#
    );
    assert!(parse_oci_manifest(doc.as_bytes()).is_ok());
    for fragment in [
        "null",
        r#""""#,
        "\"application/vnd.oci.image.index.v1+json\"",
    ] {
        assert!(
            read_oci_index(&index_entries(fragment)).is_ok(),
            "index mediaType {fragment} must stay accepted"
        );
    }
    for fragment in [
        "null",
        r#""""#,
        "\"application/vnd.oci.image.manifest.v1+json\"",
    ] {
        assert!(
            parse_oci_manifest(&manifest_doc(fragment)).is_ok(),
            "manifest mediaType {fragment} must stay accepted"
        );
    }
}

#[test]
fn index_and_descriptor_aliases_decode_only_the_final_match() {
    let index = format!(
        r#"{{"schemaVersion":"bad","SCHEMAVERSION":2,"manifests":[{{"digest":"{MEDIA_TYPE_DIGEST}","size":"bad","SIZE":1,"mediaType":"{MANIFEST_MEDIA_TYPE}"}}]}}"#
    );
    let entries = HashMap::from([
        (
            "oci-layout".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec()),
            },
        ),
        (
            "index.json".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(index.into_bytes()),
            },
        ),
    ]);
    let descriptors = read_oci_index(&entries).unwrap();
    assert_eq!(descriptors[0].size, 1);

    let index = format!(
        r#"{{"schemaVersion":2,"schemaVersion":null,"manifests":[{{"digest":"{MEDIA_TYPE_DIGEST}","size":1,"SIZE":"bad","mediaType":"{MANIFEST_MEDIA_TYPE}"}}]}}"#
    );
    let entries = HashMap::from([
        (
            "oci-layout".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec()),
            },
        ),
        (
            "index.json".to_string(),
            Blob {
                hash: String::new(),
                size: 0,
                data: Some(index.into_bytes()),
            },
        ),
    ]);
    assert_eq!(
        read_oci_index(&entries).unwrap_err().message(),
        "valid OCI index required"
    );
}

#[test]
fn descriptor_size_keeps_go_integer_tokens() {
    let descriptor: Descriptor = serde_json::from_str(r#"{"size":-0}"#).unwrap();
    assert_eq!(descriptor.size, 0);
    for token in ["1.0", "1e0", "9223372036854775808", "-9223372036854775809"] {
        let body = format!(r#"{{"size":{token}}}"#);
        assert!(
            serde_json::from_str::<Descriptor>(&body).is_err(),
            "{token} must fail"
        );
    }
}
