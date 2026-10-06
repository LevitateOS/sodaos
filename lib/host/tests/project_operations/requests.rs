use crate::candidate::candidate_body;
use crate::common::{
    fixture_digest, prepare_body, ED, FID, GO_BAD_BASE64, GO_NESTED_TYPE_ERROR, GO_UNKNOWN_FIELD,
    PID, REV,
};
use crate::{
    json, AccessKeysReq, AccountReq, HoldPreparationReq, InspectPreparationReq,
    PrepareCandidateReq, PrepareReq, StopPreparationReq,
};

#[test]
fn access_keys_req_strict_shape() {
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(req.0.project, PID);
    assert_eq!(req.0.login, "alice");
    assert_eq!(req.0.identity, 7);
    assert_eq!(req.0.revision, REV);
    assert_eq!(req.0.keys, vec![ED.to_string()]);
    assert!(req.0.apply);
    // Missing revision/keys/apply decode as empty (observe shape).
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert!(req.0.revision.is_empty() && req.0.keys.is_empty() && !req.0.apply);
    // Explicit nulls behave like missing fields, as in encoding/json.
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":null}}")
            .as_bytes(),
    )
    .unwrap();
    assert!(req.0.keys.is_empty());
    // Unknown fields rejected with the exact Go strictjson text.
    let err = AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1,"bogus":true}"#)
        .unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD);
    // Non-integer identity rejected.
    assert!(AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1.5}"#).is_err());
    // Duplicates rejected at the strict layer.
    assert!(AccessKeysReq::decode(br#"{"project":"p","project":"q"}"#).is_err());
}

#[test]
fn account_req_strict_shape() {
    let raw =
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[{ED:?}]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!((&req.0.login, req.0.identity), (&"alice".to_string(), 7));
    assert_eq!(req.0.keys, vec![ED.to_string()]);
    // Missing keys decodes as empty.
    let req = AccountReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert!(req.0.keys.is_empty());
    let err = AccountReq::decode(br#"{"project":"p","login":"a","identity":1,"admin":true}"#)
        .unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD.replace("bogus", "admin"));
}

#[test]
fn prepare_req_strict_shape() {
    let digest = fixture_digest();
    let req = PrepareReq::decode(&prepare_body(FID, "soda-coder", &digest)).unwrap();
    assert_eq!(req.0.preparation.id, FID);
    assert_eq!(req.0.preparation.role, "soda-coder");
    assert_eq!(req.0.setup.files.len(), 2);
    assert_eq!(req.0.setup.files["setup.sh"], b"true\n");
    assert_eq!(req.0.setup.bundle, b"bundle-bytes");
    req.0.validate().unwrap();
    // Unknown nested field rejected.
    let bad = prepare_body(FID, "soda-coder", &digest);
    let bad = String::from_utf8(bad).unwrap();
    let bad = bad.replacen(
        "\"role\":\"soda-coder\"",
        "\"role\":\"soda-coder\",\"bogus\":1",
        1,
    );
    let err = PrepareReq::decode(bad.as_bytes()).unwrap_err();
    assert_eq!(err, GO_UNKNOWN_FIELD);
    // Nested type error names the innermost struct, exactly like Go.
    let err = PrepareReq::decode(br#"{"preparation":{"id":1}}"#).unwrap_err();
    assert_eq!(err, GO_NESTED_TYPE_ERROR);
    // Bad base64 in a []byte field reports the Go offset text.
    let err =
        PrepareReq::decode(br#"{"preparation":{},"setup":{"files":{},"bundle":"!!!not-base64"}}"#)
            .unwrap_err();
    assert_eq!(err, GO_BAD_BASE64);
}

#[test]
fn prepare_candidate_req_strict_shape() {
    let digest = fixture_digest();
    let raw = String::from_utf8(candidate_body(FID, "f123456789abcdef012345678", &digest)).unwrap();
    let req = PrepareCandidateReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(req.0.source_preparation, "f123456789abcdef012345678");
    assert_eq!(req.0.bundle, b"candidate-bundle");
    assert_eq!(req.0.preparation.role, "soda-reviewer");
    req.0.validate().unwrap();
    // Unknown fields rejected, including inside the nested preparation.
    let bad = raw.replacen("\"bundle\":", "\"bogus\":1,\"bundle\":", 1);
    assert_eq!(
        PrepareCandidateReq::decode(bad.as_bytes()).unwrap_err(),
        GO_UNKNOWN_FIELD
    );
}

#[test]
fn inspect_stop_hold_req_shapes() {
    let addr = format!("{{\"project\":{PID:?},\"id\":{FID:?}}}");
    let ins = InspectPreparationReq::decode(addr.as_bytes()).unwrap();
    assert_eq!(
        (&ins.0.project, &ins.0.id),
        (&PID.to_string(), &FID.to_string())
    );
    ins.0.validate().unwrap();
    let stop = StopPreparationReq::decode(addr.as_bytes()).unwrap();
    stop.0.validate().unwrap();
    let hold = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    assert!(hold.0.hold && hold.0.revision == 2);
    hold.0.validate().unwrap();
    assert_eq!(
        InspectPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"id\":{FID:?},\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        StopPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"id\":{FID:?},\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        HoldPreparationReq::decode(
            format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2,\"bogus\":1}}").as_bytes()
        )
        .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
}

#[test]
fn oversize_body_rejected_before_shape() {
    let big = vec![b'x'; (1 << 20) + 1];
    assert_eq!(
        json::decode_strict(&big).unwrap_err().0,
        "request exceeds 1 MiB"
    );
    assert_eq!(
        AccessKeysReq::decode(&big).unwrap_err(),
        "request exceeds 1 MiB"
    );
}

#[test]
fn oracle_decode_errors_match_go_verbatim() {
    assert_eq!(
        AccessKeysReq::decode(br#"{"project":"p","login":"a","identity":1,"bogus":true}"#)
            .unwrap_err(),
        GO_UNKNOWN_FIELD
    );
    assert_eq!(
        PrepareReq::decode(br#"{"preparation":{"id":1}}"#).unwrap_err(),
        GO_NESTED_TYPE_ERROR
    );
    assert_eq!(
        PrepareCandidateReq::decode(
            br#"{"preparation":{},"source_preparation":"f0123456789abcdef01234567","bundle":"!!!not-base64"}"#
        )
        .unwrap_err(),
        GO_BAD_BASE64
    );
}
