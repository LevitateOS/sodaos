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
    assert!(
        quadlet::local_quadlet(
            r"[Unit]
[Container]
GlobalArgs=--x
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-2"
    );
    assert!(
        quadlet::local_quadlet(
            r"[Unit]
[Unit]
[Container]
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-3"
    );
    assert!(
        quadlet::local_quadlet(
            r"[Container]
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-4"
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
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), br#"{"ignition":{"version":"3.4.0"}}"#)
            .is_err(),
        "C-live-diff"
    );
    assert!(
        ignition::verify_live_ignition(b"{}", fragment).is_err(),
        "C-live-bad"
    );
}

#[test]
fn oracle_package_inputs() {
    // R02 thin host: no Tailscale repository or RPM. Package inputs return
    // the bare package list; any Tailscale entry fails closed.
    let pkgs = packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo bar-1.0\n"}]}}"#.as_bytes()).unwrap();
    assert_eq!(
        pkgs,
        vec!["foo".to_string(), "bar-1.0".to_string()],
        "D-pkg-good"
    );
    assert!(packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo\n"}]}}"#.as_bytes()).is_err(), "D-pkg-repo");
    assert!(packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo foo\n"}]}}"#.as_bytes()).is_err(), "D-pkg-dup");
    assert!(packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/bin/false\n"}]}}"#.as_bytes()).is_err(), "D-pkg-bad");
    assert!(packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo tailscale\n"}]}}"#.as_bytes()).is_err(), "D-pkg-tail");
    assert!(
        packages::package_inputs(r"{}".as_bytes()).is_err(),
        "D-pkg-none"
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
    assert!(
        packages::valid_rpm_inventory(&[r"b 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .is_err(),
        "F-rpm-1"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"nope".to_string()]).is_err(),
        "F-rpm-2"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"a 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .is_err(),
        "F-rpm-3"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"a 0:1~rc1-1^git.x86_64".to_string()]).is_ok(),
        "F-rpm-4"
    );
}

#[test]
fn oracle_candidate_live_config() {
    let payload_b64 = r"eyJBcmNoaXRlY3R1cmUiOiAieDg2XzY0IiwgIkJhc2UiOiAicXVheS5pby9mZWRvcmEvZmVkb3JhLWNvcmVvc0BzaGEyNTY6YmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYiIsICJDb3JlT1MiOiAiNDEuMjAyNTAxMDEuMy4wIiwgIkV4dGVuc2lvbkNhdGFsb2ciOiB7InBhY2thZ2VzIjogW3siYnJvd3Nlcl9jb250cmFjdCI6IDEsICJjYXBhYmlsaXRpZXMiOiBbIm5hdGl2ZS5hY3Rvci5yZWFkIiwgIm5hdGl2ZS5yZXBvc2l0b3J5LnJlYWQiLCAibmF0aXZlLnJlcG9zaXRvcmllcy5vd25lZC5zZWFyY2giLCAibmF0aXZlLm9yZ2FuaXphdGlvbi5vd25lcnNoaXAiLCAibmF0aXZlLnVzZXIucHVibGljX2tleXMucmVhZCIsICJuYXRpdmUuY29udHJpYnV0aW9uLmF1dGhvcml6ZSIsICJuYXRpdmUuc2VydmljZS5icmlkZ2UiLCAibmF0aXZlLmJhY2tncm91bmQub3BlcmF0aW9ucyIsICJuYXRpdmUuZXh0ZW5zaW9uLmxpZmVjeWNsZSJdLCAiZm91bnRhaW5fcHJvdG9jb2wiOiAxLCAiaWQiOiAic29kYSIsICJyZXF1aXJlZCI6IHRydWUsICJzaGEyNTYiOiAiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsICJzaXplIjogMTAyNCwgInVybCI6ICJodHRwczovL2dpdGh1Yi5jb20vbGV2aXRhdGVvcy9zb2Rhb3MvcmVsZWFzZXMvZG93bmxvYWQvdjAuMC4wLXIwMS1maXh0dXJlL3NvZGEtZXh0ZW5zaW9uLTAuMC4wLXIwMS1maXh0dXJlLWxpbnV4LXg4Nl82NC50YXIuZ3oiLCAidmVyc2lvbiI6ICIwLjAuMC1yMDEtZml4dHVyZSJ9LCB7ImJyb3dzZXJfY29udHJhY3QiOiAxLCAiY2FwYWJpbGl0aWVzIjogWyJuYXRpdmUuYWN0b3IucmVhZCIsICJuYXRpdmUuc2VydmljZS5icmlkZ2UiXSwgImZvdW50YWluX3Byb3RvY29sIjogMSwgImlkIjogInRhaWxuZXQiLCAibGlmZWN5Y2xlX2NvbnRyb2xsZXIiOiAic29kYSIsICJyZXF1aXJlZCI6IGZhbHNlLCAic2hhMjU2IjogImJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmIiLCAic2l6ZSI6IDIwNDgsICJ1cmwiOiAiaHR0cHM6Ly9naXRodWIuY29tL2xldml0YXRlb3Mvc29kYW9zL3JlbGVhc2VzL2Rvd25sb2FkL3YwLjAuMC1yMDEtZml4dHVyZS90YWlsbmV0LWV4dGVuc2lvbi0wLjAuMC1yMDEtZml4dHVyZS1saW51eC14ODZfNjQudGFyLmd6IiwgInZlcnNpb24iOiAiMC4wLjAtcjAxLWZpeHR1cmUifV0sICJ2ZXJzaW9uIjogMX0sICJGb3JtYXQiOiA0LCAiRm91bnRhaW4iOiB7IkJyb3dzZXJDb250cmFjdCI6IDEsICJFeHRlbnNpb25Qcm90b2NvbCI6IDEsICJJbWFnZSI6ICJyZWdpc3RyeS5leGFtcGxlLmludmFsaWQvZm91bnRhaW5Ac2hhMjU2OmNjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2MiLCAiSW1hZ2VDb25maWciOiAic2hhMjU2OmRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGQiLCAiTWV0YWRhdGEiOiB7IkJ5dGVzIjogNTEyLCAiU0hBMjU2IjogImRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGQiLCAiVVJMIjogImh0dHBzOi8vcmVsZWFzZXMuZXhhbXBsZS5pbnZhbGlkL3YyLjAuMC9tZXRhZGF0YS5qc29uIn0sICJQcmVwYXJlZFNvdXJjZSI6IHsiQnl0ZXMiOiA0MDk2LCAiU0hBMjU2IjogImZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmYiLCAiVVJMIjogImh0dHBzOi8vcmVsZWFzZXMuZXhhbXBsZS5pbnZhbGlkL3YyLjAuMC9wcmVwYXJlZC1zb3VyY2UudGFyLmd6In0sICJTb3VyY2UiOiB7IkJ5dGVzIjogODE5MiwgIlNIQTI1NiI6ICJlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlIiwgIlVSTCI6ICJodHRwczovL3JlbGVhc2VzLmV4YW1wbGUuaW52YWxpZC92Mi4wLjAvc291cmNlLnRhci5neiJ9LCAiU291cmNlUmV2aXNpb24iOiAiMDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWYwMTIzNDU2NyIsICJWZXJzaW9uIjogIjIuMC4wIn0sICJIb3N0UGFja2FnZXNTSEEyNTYiOiAiZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZiIsICJJRCI6ICI0MS4yMDI1MDEwMS4zLjAuc29kYS1hYWFhYWFhYWFhYWEiLCAiSW1hZ2VzIjogeyJjYW5sYW5nLXJ1bnRpbWUiOiB7IkNvbmZpZyI6ICJzaGEyNTY6NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NSIsICJNYW5pZmVzdCI6ICJzaGEyNTY6NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NCIsICJQbGF0Zm9ybSI6ICJsaW51eC9hbWQ2NCIsICJQcm92ZW5hbmNlIjogeyJCYXNlIjogInF1YXkuaW8vZmVkb3JhL2ZlZG9yYS1jb3Jlb3NAc2hhMjU2OmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmIiLCAiS2luZCI6ICJTb2RhIiwgIlJldmlzaW9uIjogImFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWEifSwgIlJlZmVyZW5jZSI6ICJnaGNyLmlvL2xldml0YXRlb3Mvc29kYW9zLWNhbmxhbmctcnVudGltZUBzaGEyNTY6NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NCJ9LCAiZGFzaGJvYXJkIjogeyJDb25maWciOiAic2hhMjU2OjExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTEiLCAiTWFuaWZlc3QiOiAic2hhMjU2OjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAiLCAiUGxhdGZvcm0iOiAibGludXgvYW1kNjQiLCAiUHJvdmVuYW5jZSI6IHsiQmFzZSI6ICJxdWF5LmlvL2ZlZG9yYS9mZWRvcmEtY29yZW9zQHNoYTI1NjpiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiIiwgIktpbmQiOiAiU29kYSIsICJSZXZpc2lvbiI6ICJhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIn0sICJSZWZlcmVuY2UiOiAiZ2hjci5pby9sZXZpdGF0ZW9zL3NvZGFvcy1kYXNoYm9hcmRAc2hhMjU2OjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAifSwgImZvcmdlam8iOiB7IkNvbmZpZyI6ICJzaGEyNTY6MzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMyIsICJNYW5pZmVzdCI6ICJzaGEyNTY6MjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMiIsICJQbGF0Zm9ybSI6ICJsaW51eC9hbWQ2NCIsICJQcm92ZW5hbmNlIjogeyJCYXNlIjogInJlZ2lzdHJ5LmV4YW1wbGUuaW52YWxpZC9mb3VudGFpbkBzaGEyNTY6Y2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjYyIsICJLaW5kIjogIlNvZGEiLCAiUmV2aXNpb24iOiAiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSJ9LCAiUmVmZXJlbmNlIjogImdoY3IuaW8vbGV2aXRhdGVvcy9zb2Rhb3MtZm9yZ2Vqb0BzaGEyNTY6MjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMiJ9LCAicG9zdGdyZXMiOiB7IkNvbmZpZyI6ICJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsICJNYW5pZmVzdCI6ICJzaGEyNTY6ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4OCIsICJQbGF0Zm9ybSI6ICJsaW51eC9hbWQ2NCIsICJQcm92ZW5hbmNlIjogeyJLaW5kIjogIlVwc3RyZWFtIiwgIlJlcXVlc3RlZCI6ICJkb2NrZXIuaW8vbGlicmFyeS9wb3N0Z3Jlc0BzaGEyNTY6ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4OCJ9LCAiUmVmZXJlbmNlIjogImRvY2tlci5pby9saWJyYXJ5L3Bvc3RncmVzQHNoYTI1Njo4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4ODg4In0sICJwcm9qZWN0LW9zIjogeyJDb25maWciOiAic2hhMjU2Ojc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3NzciLCAiTWFuaWZlc3QiOiAic2hhMjU2OjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjYiLCAiUGxhdGZvcm0iOiAibGludXgvYW1kNjQiLCAiUHJvdmVuYW5jZSI6IHsiQmFzZSI6ICJxdWF5LmlvL2ZlZG9yYS9mZWRvcmEtY29yZW9zQHNoYTI1NjpiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiIiwgIktpbmQiOiAiU29kYSIsICJSZXZpc2lvbiI6ICJhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIn0sICJSZWZlcmVuY2UiOiAiZ2hjci5pby9sZXZpdGF0ZW9zL3NvZGFvcy1wcm9qZWN0LW9zQHNoYTI1Njo2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2In0sICJwcm94eSI6IHsiQ29uZmlnIjogInNoYTI1NjpiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiIiwgIk1hbmlmZXN0IjogInNoYTI1NjphYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIiwgIlBsYXRmb3JtIjogImxpbnV4L2FtZDY0IiwgIlByb3ZlbmFuY2UiOiB7IktpbmQiOiAiVXBzdHJlYW0iLCAiUmVxdWVzdGVkIjogImRvY2tlci5pby9saWJyYXJ5L2NhZGR5QHNoYTI1NjphYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIn0sICJSZWZlcmVuY2UiOiAiZG9ja2VyLmlvL2xpYnJhcnkvY2FkZHlAc2hhMjU2OmFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWEifX0sICJPcHRpb25hbEV4dGVuc2lvbnMiOiB7InRhaWxuZXQiOiB7IkltYWdlcyI6IHsidGFpbHNjYWxlIjogeyJDb25maWciOiAic2hhMjU2OjdmNmZjMDRmMmRlYWIwZDBjMzA5ZjI5ZDdkMjc1MTc1NzllYmI3MDQ3YWQyMzkwN2Q5ZmEyNzk3ZmE3MjI3ZGYiLCAiTWFuaWZlc3QiOiAic2hhMjU2Ojk5NjkxNjYyYTJlODJmYmZkMmJmNTVkYjBmZDUyYmFjZDJmNDY2NGNlNmZkMWRlY2EzNzQ2YTAyNDE0NGEyYjciLCAiUGxhdGZvcm0iOiAibGludXgvYW1kNjQiLCAiUHJvdmVuYW5jZSI6IHsiS2luZCI6ICJVcHN0cmVhbSIsICJSZXF1ZXN0ZWQiOiAiZG9ja2VyLmlvL3RhaWxzY2FsZS90YWlsc2NhbGVAc2hhMjU2Ojk5NjkxNjYyYTJlODJmYmZkMmJmNTVkYjBmZDUyYmFjZDJmNDY2NGNlNmZkMWRlY2EzNzQ2YTAyNDE0NGEyYjcifSwgIlJlZmVyZW5jZSI6ICJkb2NrZXIuaW8vdGFpbHNjYWxlL3RhaWxzY2FsZUBzaGEyNTY6OTk2OTE2NjJhMmU4MmZiZmQyYmY1NWRiMGZkNTJiYWNkMmY0NjY0Y2U2ZmQxZGVjYTM3NDZhMDI0MTQ0YTJiNyJ9fSwgIlBhY2thZ2VJRCI6ICJ0YWlsbmV0In19LCAiUHJlc2VudGF0aW9uU0hBMjU2IjogImRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGQiLCAiUHJvamVjdFN1cHBvcnQiOiB7IkRlc3RpbmF0aW9uIjogIi91c3IvbGliZXhlYy9zb2RhL3Byb2plY3QtdGVybWluYWwiLCAiUGF5bG9hZFBhdGgiOiAiL3Vzci9zaGFyZS9zb2RhL3Byb2plY3Qtc3VwcG9ydC9wcm9qZWN0LXRlcm1pbmFsIiwgIlByb2plY3RJbWFnZSI6ICJzaGEyNTY6Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3Nzc3NyIsICJTSEEyNTYiOiAiY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjYyJ9LCAiUmVsZWFzZVRhZyI6ICJ2MC4wLjAtcjAxLWZpeHR1cmUiLCAiUmV2aXNpb24iOiAiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsICJTY2hlbWEiOiAyNiwgIlVwZ3JhZGVGcm9tIjogW119";
    let payload = base64::engine::general_purpose::STANDARD
        .decode(payload_b64.as_bytes())
        .unwrap();
    let dest = br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}"#;
    let manifest = format!("sha256:{}", "1".repeat(64));
    let console = "2".repeat(64);
    let installer_version = "coreos-installer 0.27.1";
    let candidate =
        ignition::candidate_live_config(&payload, dest, &manifest, &console, installer_version)
            .unwrap();
    assert_eq!(
        candidate,
        ignition::candidate_live_config(&payload, dest, &manifest, &console, installer_version,)
            .unwrap(),
        "candidate live config serialization is deterministic"
    );

    let document: serde_json::Value = serde_json::from_slice(&candidate).unwrap();
    assert_eq!(document["ignition"]["version"], "3.5.0");
    let files = document["storage"]["files"].as_array().unwrap();
    assert_eq!(files.len(), 2);
    let media_file = files
        .iter()
        .find(|file| file["path"] == "/var/usrlocal/share/soda-installer/media.json")
        .unwrap();
    assert_eq!(media_file["mode"], 420);
    let media_source = media_file["contents"]["source"].as_str().unwrap();
    let media_bytes = base64::engine::general_purpose::STANDARD
        .decode(media_source.strip_prefix("data:;base64,").unwrap())
        .unwrap();
    let identity: serde_json::Value = serde_json::from_slice(&media_bytes).unwrap();
    let parsed_payload = model::Payload::parse(&payload).unwrap();
    assert_eq!(identity["Architecture"], parsed_payload.architecture);
    assert_eq!(identity["Release"], parsed_payload.core_os);
    assert_eq!(identity["Revision"], parsed_payload.revision);
    assert_eq!(identity["InstallerVersion"], installer_version);
    assert_eq!(identity["HostManifest"], manifest);
    assert_eq!(identity["PayloadSHA256"], sys::hex_sha256(&payload));
    assert_eq!(identity["ConsoleSHA256"], console);

    let destination_file = files
        .iter()
        .find(|file| file["path"] == "/var/usrlocal/share/soda-installer/destination.ign")
        .unwrap();
    assert_eq!(destination_file["mode"], 420);
    assert_eq!(
        destination_file["contents"]["source"].as_str().unwrap(),
        format!("data:;base64,{}", b64(dest))
    );

    let units = document["systemd"]["units"].as_array().unwrap();
    let masked: Vec<_> = units
        .iter()
        .filter(|unit| unit["mask"] == true)
        .map(|unit| unit["name"].as_str().unwrap())
        .collect();
    assert_eq!(
        masked,
        [
            "getty@tty1.service",
            "forgejo.service",
            "soda-dashboard.service",
            "soda-proxy.service",
            "soda-host.service",
            "soda-host.socket",
            "soda-image-import.service",
        ]
    );
    let console_unit = units
        .iter()
        .find(|unit| unit["name"] == "soda-installer-console.service")
        .unwrap();
    assert_eq!(console_unit["enabled"], true);
    assert!(console_unit["contents"]
        .as_str()
        .unwrap()
        .contains("ExecStart=/usr/libexec/soda/soda-install disk"));
    assert!(
        ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.4.0"}}"#,
            &manifest,
            &console,
            installer_version,
        )
        .is_err(),
        "L-live-baddest"
    );
    assert!(
        ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.5.0"},"passwd":{"users":[]}}"#,
            &manifest,
            &console,
            installer_version,
        )
        .is_err(),
        "L-live-passwd"
    );
}
