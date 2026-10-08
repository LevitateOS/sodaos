use super::*;
use serde_json::value::RawValue;
use soda_release_image::media;
use std::collections::BTreeMap;

fn image_config(text: &str) -> BTreeMap<String, Box<RawValue>> {
    serde_json::from_str(text).unwrap()
}

fn rootfs_options(config: &BTreeMap<String, Box<RawValue>>) -> String {
    serde_json::from_str(config["live-rootfs-fsoptions"].get()).unwrap()
}

#[test]
fn oracle_media_base_url() {
    assert!(
        media::media_base_url(r"https://example.invalid/rootfs").is_ok(),
        "A-url-00"
    );
    assert!(
        media::media_base_url(r"http://example.invalid/a/b").is_ok(),
        "A-url-01"
    );
    check_err(
        "A-url-02",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"https://user@example.invalid/r").unwrap_err(),
    );
    check_err(
        "A-url-03",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"https://example.invalid/r?tag=x").unwrap_err(),
    );
    check_err(
        "A-url-04",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"https://example.invalid/r#frag").unwrap_err(),
    );
    check_err(
        "A-url-05",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"ftp://example.invalid/r").unwrap_err(),
    );
    check_err(
        "A-url-06",
        r"rootfs base URL must be reachable from the installing machine, not loopback",
        &media::media_base_url(r"https://localhost/r").unwrap_err(),
    );
    check_err(
        "A-url-07",
        r"rootfs base URL must be reachable from the installing machine, not loopback",
        &media::media_base_url(r"https://127.0.0.1/r").unwrap_err(),
    );
    check_err(
        "A-url-08",
        r"rootfs base URL must be reachable from the installing machine, not loopback",
        &media::media_base_url(r"https://[::1]/r").unwrap_err(),
    );
    assert!(
        media::media_base_url(r"https://10.0.0.1/r").is_ok(),
        "A-url-09"
    );
    check_err(
        "A-url-10",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"https://example.invalid/has space").unwrap_err(),
    );
    assert!(
        media::media_base_url(r"HTTPS://EXAMPLE.INVALID/R").is_ok(),
        "A-url-11"
    );
    check_err(
        "A-url-12",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"not-a-url").unwrap_err(),
    );
    check_err(
        "A-url-13",
        r"explicit public HTTP(S) rootfs base URL required",
        &media::media_base_url(r"").unwrap_err(),
    );
    assert!(media::media_base_url(r"https://h/p?").is_ok(), "A-url-14");
}

#[test]
fn oracle_media_compression() {
    let def = r"-zlzma,level=6 -Efragments -C1048576 --quiet";
    let mut cfg_prod = image_config(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        serde_json::to_string(def).unwrap()
    ));
    compression::set_media_compression(&mut cfg_prod, "").unwrap();
    assert_eq!(rootfs_options(&cfg_prod), def);
    let mut cfg_fast = image_config(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        serde_json::to_string(def).unwrap()
    ));
    compression::set_media_compression(&mut cfg_fast, "fast").unwrap();
    assert_eq!(
        rootfs_options(&cfg_fast),
        "-zlzma,level=1 -Efragments -C1048576 --quiet"
    );
    let mut cfg_turbo = image_config(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        serde_json::to_string(def).unwrap()
    ));
    check_err(
        "E-comp-turbo",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_turbo, "turbo").unwrap_err(),
    );
    let mut cfg_xfs = image_config(&format!(
        "{{\"live-rootfs-fstype\":\"xfs\",\"live-rootfs-fsoptions\":{}}}",
        serde_json::to_string(def).unwrap()
    ));
    check_err(
        "E-comp-xfs",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_xfs, "fast").unwrap_err(),
    );
    let mut cfg_missing = image_config("{\"live-rootfs-fstype\":\"erofs\"}");
    check_err(
        "E-comp-missing",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_missing, "fast").unwrap_err(),
    );
}

#[test]
fn oracle_media_log_events() {
    check_ok(
        "G-ev-0",
        r"b3NtZXQtc3RhcnQ=",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"Generating osmet file for x",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-1",
        r"b3NtZXQtZW5k",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"Packing successful!",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-2",
        r"cm9vdGZzLXN0YXJ0",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"Creating erofs with mkfs",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-3",
        r"cm9vdGZzLWVuZA==",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"Substituting ISO kernel arguments: k",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-4",
        r"aXNvLXN0YXJ0",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"genisoimage -o x",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-5",
        r"aXNvLWVuZA==",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"1234 extents written (567 MB)",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-6",
        r"",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"Packing successful! ",
        )
        .as_bytes(),
    );
    check_ok(
        "G-ev-7",
        r"",
        soda_release_image::events::MediaEventWriter::<Vec<u8>, Vec<u8>>::media_log_event(
            r"random noise",
        )
        .as_bytes(),
    );
}

#[test]
fn oracle_rootfs_chunks() {
    let dir = std::env::temp_dir().join(format!("sri-ok-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("r.img");
    std::fs::write(&path, vec![0x5au8; 100]).unwrap();
    let hex = sys::hex_sha256(&[0x5au8; 100]);
    let good = format!("stream-hash sha256 2097152\n{hex}\n");
    assert!(
        rootfs::verify_rootfs_chunks(path.to_str().unwrap(), &good).is_ok(),
        "K-chunk-ok"
    );
    check_err(
        "K-chunk-bad",
        r"rootfs differs from native bootstrap hashes",
        &rootfs::verify_rootfs_chunks(
            path.to_str().unwrap(),
            &format!("stream-hash sha256 2097152\n{}\n", "0".repeat(64)),
        )
        .unwrap_err(),
    );
    check_err(
        "K-chunk-hdr",
        r"unexpected native rootfs hash format",
        &rootfs::verify_rootfs_chunks(path.to_str().unwrap(), "bogus").unwrap_err(),
    );
    check_err(
        "K-chunk-empty",
        r"rootfs differs from native bootstrap hashes",
        &rootfs::verify_rootfs_chunks(path.to_str().unwrap(), "stream-hash sha256 2097152\n")
            .unwrap_err(),
    );
    let _ = std::fs::remove_dir_all(&dir);
}
