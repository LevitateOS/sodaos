use crate::common::{
    approve_response, deadline, fixture_digest, format_inspect, helper_ops, helper_state, ops,
    prepare_body, Mock, COMMIT, PID, REV,
};
use crate::PrepareCandidateReq;

/// Valid `/prepare-candidate` body reusing the shared fixture files.
pub(super) fn candidate_body(new_fid: &str, src_fid: &str, digest: &str) -> Vec<u8> {
    let raw = String::from_utf8(prepare_body(new_fid, "soda-reviewer", digest)).unwrap();
    let prep_start = raw.find("\"preparation\":").unwrap() + "\"preparation\":".len();
    let prep_end = raw.find(",\"setup\":").unwrap();
    let prep = &raw[prep_start..prep_end];
    format!("{{\"preparation\":{prep},\"source_preparation\":{src_fid:?},\"bundle\":\"Y2FuZGlkYXRlLWJ1bmRsZQ==\"}}")
        .into_bytes()
}

#[test]
fn prepare_candidate_reuses_protected_snapshot() {
    const SRC: &str = "f123456789abcdef012345678";
    const NEW: &str = "f223456789abcdef012345678";
    let digest = fixture_digest();
    let source = helper_state(
        "soda-reviewer",
        "ready",
        &digest,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let request = format!(
        "{{\"id\":{SRC:?},\"role\":\"soda-reviewer\",\"setup_digest\":{digest:?},\"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
    );
    let meta_ok = b"0:0:644:1:regular file\n".to_vec();
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(source),
        Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(request.into_bytes()),
        Ok(b"check.sh\nsetup.sh\nsource.bundle\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(b"true\n".to_vec()),
        Ok(meta_ok.clone()),
        Ok(b"true\n".to_vec()),
        Ok(format_inspect(true)),
        Ok(b"{}".to_vec()),
        Ok(approve_response(NEW, "soda-reviewer")),
        Ok(helper_state(
            "soda-reviewer",
            "ready",
            &digest,
            "",
            "",
            false,
            true,
            None,
            "",
            "",
        )),
    ]);
    let req = PrepareCandidateReq::decode(&candidate_body(NEW, SRC, &digest)).unwrap();
    let out = ops(&mock).prepare_candidate(&req, deadline()).unwrap();
    // Go `PrepareState` field order with the fresh identity.
    assert_eq!(
        out,
        format!(
            "{{\"id\":{NEW:?},\"project\":{PID:?},\"role\":\"soda-reviewer\",\"phase\":\"ready\",\"container\":{REV:?},\"source_commit\":{COMMIT:?},\"setup_digest\":{digest:?},\"ready\":true,\"stopped\":false}}"
        )
    );
    assert_eq!(
        helper_ops(&mock),
        vec!["inspect", "ensure", "approve", "inspect"]
    );
    assert_eq!(mock.calls.borrow().len(), 14);
}

#[test]
fn prepare_candidate_rejects_unready_source() {
    const SRC: &str = "f123456789abcdef012345678";
    const NEW: &str = "f223456789abcdef012345678";
    let digest = fixture_digest();
    let source = helper_state(
        "soda-reviewer",
        "running",
        &digest,
        "",
        "",
        false,
        false,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(source)]);
    let req = PrepareCandidateReq::decode(&candidate_body(NEW, SRC, &digest)).unwrap();
    assert_eq!(
        ops(&mock).prepare_candidate(&req, deadline()).unwrap_err(),
        "candidate source preparation is not ready for this role and setup"
    );
    assert_eq!(mock.calls.borrow().len(), 2);
}
