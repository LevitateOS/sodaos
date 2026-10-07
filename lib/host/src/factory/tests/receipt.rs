use super::common::{
    dummy_factory, fixed_binding, fixed_binding_json, fixed_lease, fixed_lease_json, fixed_run,
    fixed_run_json, test_state_dir,
};
use crate::factory::receipt::FactoryReceipt;
use crate::factory::*;

#[test]
fn signed_factory_numbers_preserve_negative_zero_and_nullable_exit_code() {
    let null = crate::json::decode_strict_as::<FactoryReceipt>(br#"{"exit_code":null}"#).unwrap();
    assert_eq!(null.exit_code, None);
    let zero =
        crate::json::decode_strict_as::<FactoryReceipt>(br#"{"exit_code":-0,"generation":-0}"#)
            .unwrap();
    assert_eq!(zero.exit_code, Some(0));
    assert_eq!(zero.generation, 0);

    for token in ["1.0", "1e0", "9223372036854775808"] {
        let body = format!("{{\"generation\":{token}}}");
        assert!(
            crate::json::decode_strict_as::<FactoryReceipt>(body.as_bytes()).is_err(),
            "{token}"
        );
    }
    // Byte arrays are unsigned in the original host binder.
    assert!(FactoryLaunch::decode(br#"{"prompt":[-0]}"#).is_err());
}

#[test]
fn receipt_bytes_match_go_marshal() {
    let dir = test_state_dir("receipt-bytes");
    let factory = dummy_factory(&dir);
    // Approved receipt: everything optional omitted.
    let approved = FactoryReceipt {
        run: fixed_run(),
        phase: "approved".to_string(),
        ..FactoryReceipt::default()
    };
    factory.store_receipt(&approved).unwrap();
    let path = dir.join(format!("p{}-{}.json", "b".repeat(24), "a".repeat(32)));
    let data = std::fs::read(&path).unwrap();
    let expected = format!(
        "{{\"run\":{},\"phase\":\"approved\",\"started\":false,\"delivered\":false,\"credential_returned\":false}}",
        fixed_run_json()
    );
    assert_eq!(data, expected.as_bytes());
    // Mode 0600 on the stored receipt.
    use std::os::unix::fs::PermissionsExt;
    assert_eq!(
        std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
        0o600
    );
    // No temporary siblings left behind.
    let leftovers: Vec<_> = std::fs::read_dir(&dir)
        .unwrap()
        .filter_map(|e| e.ok())
        .map(|e| e.file_name().into_string().unwrap())
        .filter(|n| n.starts_with(".receipt-"))
        .collect();
    assert!(leftovers.is_empty());

    // Completed receipt with lease, binding, exit and output.
    let completed = FactoryReceipt {
        run: fixed_run(),
        lease: Some(fixed_lease()),
        binding: Some(fixed_binding()),
        exit_code: Some(0),
        generation: 3,
        phase: "completed".to_string(),
        started: true,
        delivered: true,
        credential_returned: true,
        output: "done\n".to_string(),
        retirement: "confirmed".to_string(),
        ..FactoryReceipt::default()
    };
    factory.store_receipt(&completed).unwrap();
    let data = std::fs::read(&path).unwrap();
    let expected = format!(
        "{{\"run\":{},\"lease\":{},\"binding\":{},\"exit_code\":0,\"generation\":3,\"phase\":\"completed\",\"started\":true,\"delivered\":true,\"credential_returned\":true,\"output\":\"done\\n\",\"retirement\":\"confirmed\"}}",
        fixed_run_json(),
        fixed_lease_json(),
        fixed_binding_json()
    );
    assert_eq!(data, expected.as_bytes());

    // Tombstones skip validation but keep the exact shape.
    let tombstone = FactoryReceipt {
        run: FactoryRun {
            id: "a".repeat(32),
            project: format!("p{}", "b".repeat(24)),
            deadline: "0001-01-01T00:00:00Z".to_string(),
            ..FactoryRun::default()
        },
        phase: "stopped".to_string(),
        retirement: "confirmed".to_string(),
        reason: "stop-before-start".to_string(),
        ..FactoryReceipt::default()
    };
    factory.store_tombstone(&tombstone).unwrap();
    let data = std::fs::read(&path).unwrap();
    assert!(data.starts_with(
        format!(
            "{{\"run\":{{\"deadline\":\"0001-01-01T00:00:00Z\",\"actor\":0,\"id\":\"{}\"",
            "a".repeat(32)
        )
        .as_bytes()
    ));
    assert!(data.ends_with(
        b"\"phase\":\"stopped\",\"started\":false,\"delivered\":false,\"credential_returned\":false,\"retirement\":\"confirmed\",\"reason\":\"stop-before-start\"}"
    ));
}

#[test]
fn receipt_decode_matrix() {
    let dir = test_state_dir("receipt-decode");
    let factory = dummy_factory(&dir);
    let project = format!("p{}", "b".repeat(24));
    let run = "a".repeat(32);
    let path = dir.join(format!("{project}-{run}.json"));

    // Round trip of a full receipt, including a nested lease binding.
    let mut lease = fixed_lease();
    lease.binding = Some(fixed_binding());
    let full = FactoryReceipt {
        run: fixed_run(),
        lease: Some(lease),
        binding: Some(fixed_binding()),
        exit_code: Some(3),
        generation: 9,
        phase: "failed".to_string(),
        started: true,
        delivered: true,
        credential_returned: true,
        output: "x".to_string(),
        retirement: "confirmed".to_string(),
        reason: "execution-failed".to_string(),
    };
    factory.store_receipt(&full).unwrap();
    let (loaded, exists) = factory.load_receipt(&project, &run).unwrap();
    assert!(exists);
    assert_eq!(loaded.run, full.run);
    assert_eq!(loaded.exit_code, Some(3));
    assert_eq!(loaded.generation, 9);
    assert_eq!(loaded.reason, "execution-failed");
    assert_eq!(loaded.lease.as_ref().unwrap().actor_id, 7);
    assert!(loaded.lease.as_ref().unwrap().binding.is_some());

    // Missing receipt reports absence.
    let (_, exists) = factory.load_receipt(&project, &"b".repeat(32)).unwrap();
    assert!(!exists);

    // Oversized.
    std::fs::write(&path, vec![b'x'; (128 << 10) + 1]).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "run receipt exceeds bounds"
    );
    // Truncated JSON.
    std::fs::write(&path, b"{\"run\":").unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run receipt"
    );
    // Unknown field (DisallowUnknownFields).
    let mut raw = full.encode();
    raw.pop();
    raw.push_str(",\"extra\":1}");
    std::fs::write(&path, raw.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run receipt"
    );
    // Identity mismatch.
    let mismatch = full.encode().replacen(&run, &"b".repeat(32), 1);
    std::fs::write(&path, mismatch.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "run receipt identity mismatch"
    );
    // Tombstone with a lease is corrupt.
    let bad_tomb = format!(
        "{{\"run\":{{\"deadline\":\"0001-01-01T00:00:00Z\",\"actor\":0,\"id\":\"{run}\",\"project\":\"{project}\",\"role\":\"\",\"preparation\":\"\",\"harness\":\"\",\"harness_version\":\"\",\"assignment\":\"\",\"source_commit\":\"\",\"connection\":\"\"}},\"lease\":{},\"phase\":\"stopped\",\"started\":false,\"delivered\":false,\"credential_returned\":false}}",
        fixed_lease_json()
    );
    std::fs::write(&path, bad_tomb.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run tombstone"
    );
    // Bad phase.
    let bad_phase = full
        .encode()
        .replace("\"phase\":\"failed\"", "\"phase\":\"waiting\"");
    std::fs::write(&path, bad_phase.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run phase"
    );
    // Lease identity mismatch.
    let bad_lease = full.encode().replace(
        &format!("\"execution_id\":\"{run}\""),
        &format!("\"execution_id\":\"{}\"", "b".repeat(32)),
    );
    std::fs::write(&path, bad_lease.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "run lease identity mismatch"
    );
    // Record exceeds bounds.
    let bad_bounds = full.encode().replace(
        "\"reason\":\"execution-failed\"",
        &format!("\"reason\":\"{}\"", "r".repeat(257)),
    );
    std::fs::write(&path, bad_bounds.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "run record exceeds bounds"
    );
    // Malformed `,string` lease integer.
    let bad_actor = full
        .encode()
        .replace("\"actor_id\":\"7\"", "\"actor_id\":\"zz\"");
    std::fs::write(&path, bad_actor.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run receipt"
    );
    // Bare (unquoted) `,string` integer is rejected like Go.
    let bare_actor = full
        .encode()
        .replace("\"actor_id\":\"7\"", "\"actor_id\":7");
    std::fs::write(&path, bare_actor.as_bytes()).unwrap();
    assert_eq!(
        factory.load_receipt(&project, &run).unwrap_err().message(),
        "invalid run receipt"
    );
}
