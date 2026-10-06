use super::*;

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
