use super::*;

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
