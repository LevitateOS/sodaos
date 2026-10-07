use super::payload::{LockedAsset, LockedItem, PayloadEntries};
use super::*;

#[test]
fn stage_json_visitors_preserve_payload_pairs_and_last_known_slots() {
    let payload: PayloadEntries = serde_json::from_str(r#"{"asset":"first","asset":"second"}"#)
        .expect("string payload entries");
    assert_eq!(
        payload.0,
        [
            ("asset".to_string(), "first".to_string()),
            ("asset".to_string(), "second".to_string())
        ]
    );

    let item: LockedItem = serde_json::from_str(r#"{"files":[{"file":1e400}]}"#)
        .expect("unknown raw fields retain large number tokens");
    let files: Vec<Box<serde_json::value::RawValue>> =
        serde_json::from_str(item.files.unwrap().get()).unwrap();
    let asset: LockedAsset = serde_json::from_str(files[0].get()).unwrap();
    assert!(serde_json::from_str::<String>(asset.file.unwrap().get()).is_err());

    let asset: LockedAsset = serde_json::from_str(
        r#"{"file":false,"file":"last","sha256":"old","sha256":"digest","ignored":1e400}"#,
    )
    .expect("raw known slots defer conversion until after duplicate selection");
    assert_eq!(
        serde_json::from_str::<String>(asset.file.unwrap().get()).unwrap(),
        "last"
    );
    assert_eq!(
        serde_json::from_str::<String>(asset.sha256.unwrap().get()).unwrap(),
        "digest"
    );
}

#[test]
fn favicon_container_matches_struct_layout() {
    let frames = vec![(16u8, vec![0x89u8, 0x50]), (32u8, vec![0x89u8])];
    let ico = favicon_bytes(&frames);
    assert_eq!(&ico[0..6], &[0, 0, 1, 0, 2, 0]);
    assert_eq!(
        u32::from_le_bytes(ico[6 + 8..6 + 12].try_into().unwrap()),
        2
    );
    assert_eq!(
        u32::from_le_bytes(ico[6 + 12..6 + 16].try_into().unwrap()),
        6 + 32
    );
    assert_eq!(
        u32::from_le_bytes(ico[22 + 12..22 + 16].try_into().unwrap()),
        6 + 32 + 2
    );
    assert_eq!(&ico[ico.len() - 3..], &[0x89, 0x50, 0x89]);
}

#[test]
fn payload_source_roots_build_tokens() {
    let source = Path::new("/src");
    let build = Path::new("/src/.artifacts/native/x86_64");
    assert_eq!(
        payload_source(source, build, "@build/forgejo-js/app.js"),
        Path::new("/src/.artifacts/native/x86_64/forgejo-js/app.js")
    );
    assert_eq!(
        payload_source(source, build, "assets/x.svg"),
        Path::new("/src/assets/x.svg")
    );
}
