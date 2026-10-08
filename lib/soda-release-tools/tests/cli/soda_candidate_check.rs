use serde_json::Value;
use std::fs;

use super::{candidate_check_bin, run, TempDir};

const DELIVERY_GOLDEN: &str =
    include_str!("../../../soda-release-deliver/tests/goldens/deliver.json");

#[test]
fn candidate_check_binary_binds_identity_before_archive_read() {
    let golden: Value = serde_json::from_str(DELIVERY_GOLDEN).expect("delivery golden");
    let payload = golden["payload"].as_str().expect("payload bytes");
    let candidate = golden["candidate"].as_str().expect("candidate bytes");
    let payload_value: Value = serde_json::from_str(payload).expect("payload record");
    let candidate_value: Value = serde_json::from_str(candidate).expect("candidate record");
    let arch = payload_value["Architecture"]
        .as_str()
        .expect("payload architecture");
    let soda_revision = payload_value["Revision"]
        .as_str()
        .expect("payload revision");
    let forgejo_revision = candidate_value["ForgejoRevision"]
        .as_str()
        .expect("candidate Forgejo revision");

    let dir = TempDir::new("candidate-check");
    let payload_path = dir.path.join("payload.json");
    let candidate_path = dir.path.join("candidate.json");
    fs::write(&payload_path, payload).expect("write payload");
    fs::write(&candidate_path, candidate).expect("write candidate");
    let payload_before = fs::read(&payload_path).expect("read payload");
    let candidate_before = fs::read(&candidate_path).expect("read candidate");
    assert!(!dir.path.join("host.oci").exists());

    let cases = [
        ("architecture", "aarch64", soda_revision, forgejo_revision),
        (
            "soda revision",
            arch,
            "0000000000000000000000000000000000000000",
            forgejo_revision,
        ),
        (
            "candidate forgejo revision",
            arch,
            soda_revision,
            "0000000000000000000000000000000000000000",
        ),
        ("host archive", arch, soda_revision, forgejo_revision),
    ];

    for (boundary, requested_arch, requested_soda, requested_forgejo) in cases {
        let args = [
            "--candidate".to_owned(),
            dir.path.to_string_lossy().into_owned(),
            "--arch".to_owned(),
            requested_arch.to_owned(),
            "--soda-revision".to_owned(),
            requested_soda.to_owned(),
            "--forgejo-revision".to_owned(),
            requested_forgejo.to_owned(),
        ];
        let (status, stdout, stderr) = run(&candidate_check_bin(), &dir.path, &args);
        assert_eq!(status, 1, "{boundary}: {stderr}");
        assert!(
            stdout.is_empty(),
            "{boundary}: unexpected stdout {stdout:?}"
        );
        assert_eq!(stderr.lines().count(), 1, "{boundary}: {stderr:?}");
        assert!(stderr.contains(boundary), "{boundary}: {stderr:?}");
    }

    assert_eq!(
        fs::read(&payload_path).expect("payload after check"),
        payload_before
    );
    assert_eq!(
        fs::read(&candidate_path).expect("candidate after check"),
        candidate_before
    );
    assert!(!dir.path.join("host.oci").exists());
}
