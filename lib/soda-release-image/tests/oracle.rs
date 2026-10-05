//! Differential oracle: frozen Go-owner outputs captured 2026-10-04.
//! Each case replays the oracle battery against the Rust port and
//! requires byte-identical outputs or identical error messages.

use base64::Engine;
use soda_release_image::{
    compression, extension, ignition, jsonio, layout, media, model, packages, quadlet, recall,
    rootfs, sys,
};

fn b64(data: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(data)
}

fn check_ok(name: &str, expected_b64: &str, got: &[u8]) {
    assert_eq!(b64(got), expected_b64, "{name}");
}

fn check_err(name: &str, expected: &str, got: &soda_release_image::error::Error) {
    assert_eq!(got.0, expected, "{name}");
}

#[test]
fn oracle_media_base_url() {
    check_ok(
        "A-url-00",
        r"aHR0cHM6Ly9leGFtcGxlLmludmFsaWQvcm9vdGZz",
        r"https://example.invalid/rootfs".as_bytes(),
    );
    assert!(
        media::media_base_url(r"https://example.invalid/rootfs").is_ok(),
        "A-url-00"
    );
    check_ok(
        "A-url-01",
        r"aHR0cDovL2V4YW1wbGUuaW52YWxpZC9hL2I=",
        r"http://example.invalid/a/b".as_bytes(),
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
    check_ok(
        "A-url-09",
        r"aHR0cHM6Ly8xMC4wLjAuMS9y",
        r"https://10.0.0.1/r".as_bytes(),
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
    check_ok(
        "A-url-11",
        r"SFRUUFM6Ly9FWEFNUExFLklOVkFMSUQvUg==",
        r"HTTPS://EXAMPLE.INVALID/R".as_bytes(),
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
    check_ok("A-url-14", r"aHR0cHM6Ly9oL3A/", r"https://h/p?".as_bytes());
    assert!(media::media_base_url(r"https://h/p?").is_ok(), "A-url-14");
}

#[test]
fn oracle_local_quadlet() {
    check_ok(
        "B-quad-0",
        r"W1VuaXRdClJlcXVpcmVzPXNvZGEtaW1hZ2UtaW1wb3J0LnNlcnZpY2UKQWZ0ZXI9c29kYS1pbWFnZS1pbXBvcnQuc2VydmljZQpEZXNjcmlwdGlvbj14CltDb250YWluZXJdClB1bGw9bmV2ZXIKSW1hZ2U9c2hhMjU2OnJlZgpFeGVjPWEK",
        quadlet::local_quadlet(
            r"[Unit]
Description=x
[Container]
Image=localhost/soda-dashboard:dev
Pull=newer
Exec=a
",
            "sha256:ref",
        )
        .unwrap()
        .as_bytes(),
    );
    check_ok(
        "B-quad-1",
        r"W1VuaXRdClJlcXVpcmVzPXNvZGEtaW1hZ2UtaW1wb3J0LnNlcnZpY2UKQWZ0ZXI9c29kYS1pbWFnZS1pbXBvcnQuc2VydmljZQpbQ29udGFpbmVyXQpQdWxsPW5ldmVyCkltYWdlPXNoYTI1NjpyZWYK",
        quadlet::local_quadlet(
            r"[Unit]
[Container]
Image=a
",
            "sha256:ref",
        )
        .unwrap()
        .as_bytes(),
    );
    check_err(
        "B-quad-2",
        r"unexpected existing Quadlet storage arguments",
        &quadlet::local_quadlet(
            r"[Unit]
[Container]
GlobalArgs=--x
Image=a
",
            "sha256:ref",
        )
        .unwrap_err(),
    );
    check_err(
        "B-quad-3",
        r"one fixed Quadlet container/image required",
        &quadlet::local_quadlet(
            r"[Unit]
[Unit]
[Container]
Image=a
",
            "sha256:ref",
        )
        .unwrap_err(),
    );
    check_err(
        "B-quad-4",
        r"one fixed Quadlet container/image required",
        &quadlet::local_quadlet(
            r"[Container]
Image=a
",
            "sha256:ref",
        )
        .unwrap_err(),
    );
}

#[test]
fn oracle_live_ignition() {
    use flate2::write::GzEncoder;
    use flate2::Compression;
    use std::io::Write;
    let fragment =
        br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]},"systemd":{"units":[]}}"#;
    let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(fragment).unwrap();
    let gzipped = encoder.finish().unwrap();
    let wrapped = format!("{{\"ignition\":{{\"config\":{{\"merge\":[{{\"source\":\"data:;base64,{}\",\"compression\":\"gzip\"}}]}}}}}}", b64(&gzipped));
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), fragment).is_ok(),
        "C-live-ok"
    );
    let nulls = br#"{"ignition":{"version":"3.5.0","config":null},"storage":{"files":[]},"systemd":{"units":[]},"networkd":null}"#;
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), nulls).is_ok(),
        "C-live-nulls"
    );
    check_err(
        "C-live-diff",
        r"embedded live Ignition differs",
        &ignition::verify_live_ignition(wrapped.as_bytes(), br#"{"ignition":{"version":"3.4.0"}}"#)
            .unwrap_err(),
    );
    check_err(
        "C-live-bad",
        r"unexpected native live Ignition",
        &ignition::verify_live_ignition(b"{}", fragment).unwrap_err(),
    );
}

#[test]
fn oracle_package_inputs() {
    let (pkgs, repo) = packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo bar-1.0\n"}]}}"#.as_bytes()).unwrap();
    check_ok(
        "D-pkg-good",
        r"Zm9vLGJhci0xLjB8aHR0cHM6Ly9wa2dzLnRhaWxzY2FsZS5jb20vc3RhYmxlL2ZlZG9yYS90YWlsc2NhbGUucmVwbw==",
        format!("{}|{}", pkgs.join(","), repo).as_bytes(),
    );
    check_err("D-pkg-dup", r"invalid or duplicate host package", &packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo foo\n"}]}}"#.as_bytes()).unwrap_err());
    check_err("D-pkg-bad", r"unexpected package installation command", &packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/bin/false\n"}]}}"#.as_bytes()).unwrap_err());
    check_err("D-pkg-nore", r"missing host package inputs", &packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo\n"}]}}"#.as_bytes()).unwrap_err());
    check_err(
        "D-pkg-none",
        r"missing host package inputs",
        &packages::package_inputs(r"{}".as_bytes()).unwrap_err(),
    );
}

#[test]
fn oracle_media_compression() {
    let def = r"-zlzma,level=6 -Efragments -C1048576 --quiet";
    let mut cfg_prod = jsonio::parse(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        jsonio::to_compact(&soda_json::JsonValue::Str(def.to_string()))
    ))
    .unwrap();
    compression::set_media_compression(&mut cfg_prod, "").unwrap();
    let rendered_prod = jsonio::to_compact(cfg_prod.get("live-rootfs-fsoptions").unwrap());
    check_ok(
        "E-comp-prod",
        r"Ii16bHptYSxsZXZlbD02IC1FZnJhZ21lbnRzIC1DMTA0ODU3NiAtLXF1aWV0Ig==",
        rendered_prod.as_bytes(),
    );
    let mut cfg_fast = jsonio::parse(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        jsonio::to_compact(&soda_json::JsonValue::Str(def.to_string()))
    ))
    .unwrap();
    compression::set_media_compression(&mut cfg_fast, "fast").unwrap();
    let rendered_fast = jsonio::to_compact(cfg_fast.get("live-rootfs-fsoptions").unwrap());
    check_ok(
        "E-comp-fast",
        r"Ii16bHptYSxsZXZlbD0xIC1FZnJhZ21lbnRzIC1DMTA0ODU3NiAtLXF1aWV0Ig==",
        rendered_fast.as_bytes(),
    );
    let mut cfg_turbo = jsonio::parse(&format!(
        "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
        jsonio::to_compact(&soda_json::JsonValue::Str(def.to_string()))
    ))
    .unwrap();
    check_err(
        "E-comp-turbo",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_turbo, "turbo").unwrap_err(),
    );
    let mut cfg_xfs = jsonio::parse(&format!(
        "{{\"live-rootfs-fstype\":\"xfs\",\"live-rootfs-fsoptions\":{}}}",
        jsonio::to_compact(&soda_json::JsonValue::Str(def.to_string()))
    ))
    .unwrap();
    check_err(
        "E-comp-xfs",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_xfs, "fast").unwrap_err(),
    );
    let mut cfg_missing = jsonio::parse("{\"live-rootfs-fstype\":\"erofs\"}").unwrap();
    check_err(
        "E-comp-missing",
        r"fast media requires the reviewed upstream EROFS/LZMA defaults",
        &compression::set_media_compression(&mut cfg_missing, "fast").unwrap_err(),
    );
}

#[test]
fn oracle_rpm_inventory() {
    assert!(
        packages::valid_rpm_inventory(&[
            r"bash 0:5.1.8-1.fc38.x86_64".to_string(),
            r"rpm-ostree 0:2023.1-1.fc38.x86_64".to_string()
        ])
        .is_ok(),
        "F-rpm-0"
    );
    check_err(
        "F-rpm-1",
        r"sorted RPM inventory required",
        &packages::valid_rpm_inventory(&[r"b 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .unwrap_err(),
    );
    check_err(
        "F-rpm-2",
        r"invalid recorded RPM inventory",
        &packages::valid_rpm_inventory(&[r"nope".to_string()]).unwrap_err(),
    );
    check_err(
        "F-rpm-3",
        r"invalid recorded RPM inventory",
        &packages::valid_rpm_inventory(&[r"a 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .unwrap_err(),
    );
    assert!(
        packages::valid_rpm_inventory(&[r"a 0:1~rc1-1^git.x86_64".to_string()]).is_ok(),
        "F-rpm-4"
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

struct Stub;
impl soda_release_image::foreign::Production for Stub {
    fn source(&self) -> &str {
        ""
    }
    fn forgejo_source(&self) -> &str {
        ""
    }
    fn forgejo_revision(&self) -> &str {
        ""
    }
    fn native(&self) -> &str {
        ""
    }
    fn out(&self) -> &str {
        ""
    }
    fn arch(&self) -> &str {
        "x86_64"
    }
    fn revision(&self) -> &str {
        ""
    }
    fn live_inputs(&self) -> &str {
        ""
    }
    fn execute(
        &self,
        _: &str,
        _: &str,
        _: &[String],
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn capture(
        &self,
        _: &str,
        _: &str,
        _: &[String],
    ) -> Result<String, soda_release_image::error::Error> {
        unreachable!()
    }
    fn next(&self, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn resolve_inputs(&mut self) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn dependencies(&self) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn compile_rust(
        &self,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn stage_fork_binary(&self, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn assets(&self, _: &str, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn images(
        &self,
        _: &str,
    ) -> Result<
        std::collections::HashMap<String, model::ProducedImage>,
        soda_release_image::error::Error,
    > {
        unreachable!()
    }
    fn inspect_oci(
        &self,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<model::Image, soda_release_image::error::Error> {
        unreachable!()
    }
    fn verify_content(
        &self,
        _: &model::Payload,
        _: &str,
    ) -> Result<(std::collections::HashMap<String, String>, u64), soda_release_image::error::Error>
    {
        unreachable!()
    }
    fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, soda_release_image::error::Error> {
        unreachable!()
    }
    fn read_live_inputs(
        &self,
        _: &str,
    ) -> Result<model::LiveInputs, soda_release_image::error::Error> {
        unreachable!()
    }
    fn check_native(&self, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn sign_media(
        &self,
        _: &model::Trust,
        _: &model::Permit,
        _: &str,
        _: &str,
        _: &str,
        _: &model::SecretFiles,
        _: &str,
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn verify_copy(
        &self,
        _: &model::Trust,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn write_document(
        &self,
        _: &str,
        _: &soda_json::JsonValue,
    ) -> Result<String, soda_release_image::error::Error> {
        unreachable!()
    }
}

#[test]
fn oracle_stage_layout() {
    let stub = Stub;
    let p3 = model::Payload {
        format: 3,
        ..model::Payload::default()
    };
    let p2 = model::Payload {
        format: 2,
        ..model::Payload::default()
    };
    assert!(
        layout::valid_stage_layout("/a", "/b", &p3, Some(&stub)),
        "J-layout-ok"
    );
    assert!(
        !(layout::valid_stage_layout("/a", "/b", &p3, None)),
        "J-layout-nil"
    );
    assert!(
        !(layout::valid_stage_layout("/a", "/b", &p2, Some(&stub))),
        "J-layout-fmt"
    );
    assert!(
        !(layout::valid_stage_layout("a", "/b", &p3, Some(&stub))),
        "J-layout-rel"
    );
    assert!(
        !(layout::valid_stage_layout("/a:x", "/b", &p3, Some(&stub))),
        "J-layout-colon"
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

#[test]
fn oracle_candidate_live_config() {
    let payload_b64 = r"eyJBcmNoaXRlY3R1cmUiOiJ4ODZfNjQiLCJCYXNlIjoicXVheS5pby9mZWRvcmEvZmVkb3JhLWNvcmVvc0BzaGEyNTY6YmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYiIsIkNvcmVPUyI6IjQxLjIwMjUwMTAxLjMuMCIsIkZvcm1hdCI6MywiSG9zdFBhY2thZ2VzU0hBMjU2IjoiZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZiIsIklEIjoiNDEuMjAyNTAxMDEuMy4wLnNvZGEtYWFhYWFhYWFhYWFhIiwiSW1hZ2VzIjp7ImRhc2hib2FyZCI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtZGFzaGJvYXJkQHNoYTI1Njo5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5In0sImV4dGVuc2lvbiI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjplZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtZXh0ZW5zaW9uQHNoYTI1Njo5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5In0sImZvcmdlam8iOnsiQXJjaGl2ZVNIQTI1NiI6ImFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWEiLCJDb25maWciOiJzaGEyNTY6Y2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjYyIsIk1hbmlmZXN0Ijoic2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkiLCJSZWZlcmVuY2UiOiJnaGNyLmlvL2Uvc29kYW9zLWZvcmdlam9Ac2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkifSwicHJvamVjdC1vcyI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtcHJvamVjdC1vc0BzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSJ9LCJwcm94eSI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtcHJveHlAc2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkifSwidGFpbG5ldCI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtdGFpbG5ldEBzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSJ9fSwiUHJlc2VudGF0aW9uU0hBMjU2IjoiZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZCIsIlJlcG9zaXRvcnlQcmVmaXgiOiJnaGNyLmlvL2Uvc29kYW9zIiwiUmV2aXNpb24iOiJhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIiwiU2NoZW1hIjoyNiwiVXBncmFkZUZyb20iOltdfQ==";
    let payload = base64::engine::general_purpose::STANDARD
        .decode(payload_b64.as_bytes())
        .unwrap();
    let dest = br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}"#;
    let manifest = format!("sha256:{}", "1".repeat(64));
    let console = "2".repeat(64);
    check_ok(
        "L-live-ok",
        r"eyJpZ25pdGlvbiI6eyJ2ZXJzaW9uIjoiMy41LjAifSwic3RvcmFnZSI6eyJmaWxlcyI6W3siY29udGVudHMiOnsic291cmNlIjoiZGF0YTo7YmFzZTY0LGV5SkJjbU5vYVhSbFkzUjFjbVVpT2lKNE9EWmZOalFpTENKU1pXeGxZWE5sSWpvaU5ERXVNakF5TlRBeE1ERXVNeTR3SWl3aVNXNXpkR0ZzYkdWeVZtVnljMmx2YmlJNkltTnZjbVZ2Y3kxcGJuTjBZV3hzWlhJZ01DNHlOaTR3SWl3aVVtVjJhWE5wYjI0aU9pSmhZV0ZoWVdGaFlXRmhZV0ZoWVdGaFlXRmhZV0ZoWVdGaFlXRmhZV0ZoWVdGaFlXRmhZV0ZoSWl3aVNHOXpkRTFoYm1sbVpYTjBJam9pYzJoaE1qVTJPakV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEV4TVRFeE1URXhNVEVpTENKUVlYbHNiMkZrVTBoQk1qVTJJam9pTm1SaE9UazRNak5tTmpBd016UmxOamRpTW1Rd09HRmpOVEEyTTJNeE56UXhOV05qTlRrNE5tRmhOV1pqWVRWaVl6RTJOVEZoTVRWbFpHSTVNMkpsTVNJc0lrTnZibk52YkdWVFNFRXlOVFlpT2lJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeU1qSXlNakl5TWpJeUluMD0ifSwibW9kZSI6NDIwLCJwYXRoIjoiL3Zhci91c3Jsb2NhbC9zaGFyZS9zb2RhLWluc3RhbGxlci9tZWRpYS5qc29uIn0seyJjb250ZW50cyI6eyJzb3VyY2UiOiJkYXRhOjtiYXNlNjQsZXlKcFoyNXBkR2x2YmlJNmV5SjJaWEp6YVc5dUlqb2lNeTQxTGpBaWZTd2ljM1J2Y21GblpTSTZleUptYVd4bGN5STZXMTE5ZlE9PSJ9LCJtb2RlIjo0MjAsInBhdGgiOiIvdmFyL3VzcmxvY2FsL3NoYXJlL3NvZGEtaW5zdGFsbGVyL2Rlc3RpbmF0aW9uLmlnbiJ9XX0sInN5c3RlbWQiOnsidW5pdHMiOlt7Im1hc2siOnRydWUsIm5hbWUiOiJnZXR0eUB0dHkxLnNlcnZpY2UifSx7Im1hc2siOnRydWUsIm5hbWUiOiJmb3JnZWpvLnNlcnZpY2UifSx7Im1hc2siOnRydWUsIm5hbWUiOiJzb2RhLWRhc2hib2FyZC5zZXJ2aWNlIn0seyJtYXNrIjp0cnVlLCJuYW1lIjoic29kYS1wcm94eS5zZXJ2aWNlIn0seyJtYXNrIjp0cnVlLCJuYW1lIjoic29kYS1ob3N0LnNlcnZpY2UifSx7Im1hc2siOnRydWUsIm5hbWUiOiJzb2RhLWhvc3Quc29ja2V0In0seyJtYXNrIjp0cnVlLCJuYW1lIjoic29kYS1pbWFnZS1pbXBvcnQuc2VydmljZSJ9LHsiY29udGVudHMiOiJbVW5pdF1cbkRlc2NyaXB0aW9uPVNvZGFPUyBpbnN0YWxsYXRpb24gY29uc29sZVxuQWZ0ZXI9c3lzdGVtZC11c2VyLXNlc3Npb25zLnNlcnZpY2UgTmV0d29ya01hbmFnZXIuc2VydmljZVxuQ29uZmxpY3RzPWdldHR5QHR0eTEuc2VydmljZVxuW1NlcnZpY2VdXG5UeXBlPXNpbXBsZVxuUHJpdmF0ZU1vdW50cz15ZXNcbkV4ZWNTdGFydD0vdXNyL2xpYmV4ZWMvc29kYS9zb2RhLWluc3RhbGwgZGlza1xuU3RhbmRhcmRJbnB1dD10dHktZm9yY2VcblN0YW5kYXJkT3V0cHV0PXR0eVxuU3RhbmRhcmRFcnJvcj10dHlcblRUWVBhdGg9L2Rldi90dHkxXG5UVFlSZXNldD15ZXNcblRUWVZIYW5ndXA9eWVzXG5SZXN0YXJ0PW5vXG5bSW5zdGFsbF1cbldhbnRlZEJ5PW11bHRpLXVzZXIudGFyZ2V0XG4iLCJlbmFibGVkIjp0cnVlLCJuYW1lIjoic29kYS1pbnN0YWxsZXItY29uc29sZS5zZXJ2aWNlIn1dfX0=",
        &ignition::candidate_live_config(&payload, dest, &manifest, &console).unwrap(),
    );
    check_err(
        "L-live-baddest",
        r"public converted destination template required",
        &ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.4.0"}}"#,
            &manifest,
            &console,
        )
        .unwrap_err(),
    );
    check_err(
        "L-live-passwd",
        r"public converted destination template required",
        &ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.5.0"},"passwd":{"users":[]}}"#,
            &manifest,
            &console,
        )
        .unwrap_err(),
    );
}
