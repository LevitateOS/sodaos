use super::*;

use super::validation_tests::{did, fid, pid, COMMIT, DIGEST, EFFECTS};

fn prep_json() -> String {
    format!(
        "{{\"id\":{:?},\"project\":{:?},\"role\":\"soda-coder\",\"revision\":0,\
         \"requirements\":{{\"id\":{:?},\"revision\":0,\"approver\":1,\
         \"source_commit\":{:?},\"digest\":{:?}}},\
         \"approval\":{{\"id\":{:?},\"revision\":0,\"approver\":1,\
         \"effects_digest\":{:?}}},\"source_commit\":{:?},\
         \"setup_digest\":{:?}}}",
        fid(),
        pid(),
        did(1),
        COMMIT,
        EFFECTS,
        did(2),
        EFFECTS,
        COMMIT,
        DIGEST
    )
}

fn prepare_json() -> String {
    format!(
        "{{\"preparation\":{},\"setup\":{{\"files\":{{\
         \"setup.sh\":\"IyEvYmluL3NoCg==\",\
         \"check.sh\":\"IyEvYmluL3NoCg==\"}},\
         \"bundle\":\"QlVORExF\"}}}}",
        prep_json()
    )
}

#[test]
fn prepare_decode_round_trip() {
    let p = Prepare::decode(prepare_json().as_bytes()).unwrap();
    assert!(p.validate().is_ok());
    assert_eq!(p.setup.files["setup.sh"], b"#!/bin/sh\n");
    assert_eq!(p.setup.bundle, b"BUNDLE");
}

#[test]
fn preparation_tool_list_null_elements_decode_as_empty_strings() {
    let body = prepare_json().replace(
        "\"preparation\":{",
        "\"preparation\":{\"tools\":[\"go\",null,\"git\"],",
    );
    let p = Prepare::decode(body.as_bytes()).unwrap();
    assert_eq!(
        p.preparation.tools,
        vec!["go".to_string(), String::new(), "git".to_string()]
    );
}

#[test]
fn preparation_signed_revision_fields_accept_negative_zero() {
    let body = prepare_json()
        .replace("\"revision\":0", "\"revision\":-0")
        .replace("\"approver\":1", "\"approver\":-0");
    let p = Prepare::decode(body.as_bytes()).unwrap();
    assert_eq!(p.preparation.revision, 0);
    assert_eq!(p.preparation.requirements.revision, 0);
    assert_eq!(p.preparation.requirements.approver, 0);
    assert_eq!(p.preparation.approval.revision, 0);
}

#[test]
fn prepare_rejects_malformed_and_unknown_fields() {
    let cases = [
        "{\"preparation\":{\"revision\":\"x\"},\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}".to_string(),
        format!("{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":\"!!!\"}}}}", prep_json()),
        format!("{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":5}},\"bundle\":\"QUJD\"}}}}", prep_json()),
        format!("{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":[300]}}}}", prep_json()),
        format!("{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":{{}}}}}}", prep_json()),
        "{\"preparation\":[],\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}".to_string(),
        format!("{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":\"QUJD\"}},\"zzz\":1}}", prep_json()),
        "{\"requirements\":{\"revision\":\"x\"}}".to_string(),
    ];
    for body in cases {
        assert!(Prepare::decode(body.as_bytes()).is_err(), "{body}");
    }
    // Numeric byte arrays decode; null struct binds zero.
    let body = format!(
        "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":[1,2]}}}}}}",
        prep_json()
    );
    // Missing bundle binds empty (validity is Validate's job).
    let p = Prepare::decode(body.as_bytes()).unwrap();
    assert_eq!(p.setup.files["a"], vec![1u8, 2]);
    assert!(p.setup.bundle.is_empty());
    let p = Prepare::decode(
        b"{\"preparation\":null,\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}",
    )
    .unwrap();
    assert!(p.preparation.id.is_empty());
}

#[test]
fn state_encodes_honor_omitempty() {
    let mut s = PrepareState {
        id: "f".to_string(),
        project: "p".to_string(),
        role: "soda-coder".to_string(),
        phase: "ready".to_string(),
        container: "c".to_string(),
        source_commit: COMMIT.to_string(),
        setup_digest: DIGEST.to_string(),
        ready: true,
        ..Default::default()
    };
    assert_eq!(
        s.encode(),
        format!(
            "{{\"id\":\"f\",\"project\":\"p\",\"role\":\"soda-coder\",\"phase\":\"ready\",\
             \"container\":\"c\",\"source_commit\":{COMMIT:?},\"setup_digest\":{DIGEST:?},\
             \"ready\":true,\"stopped\":false}}"
        )
    );
    s.tools = vec![ResolvedTool {
        name: "git".to_string(),
        path: "/usr/bin/git".to_string(),
        version: "git 2".to_string(),
    }];
    s.missing = "go".to_string();
    s.setup_exit = Some(0);
    s.check_exit = Some(1);
    s.output = "out".to_string();
    s.stopped = true;
    s.retirement = "confirmed".to_string();
    let encoded = s.encode();
    assert!(encoded.contains(
        "\"tools\":[{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"git 2\"}]"
    ));
    assert!(encoded.contains("\"missing\":\"go\""));
    assert!(encoded.contains("\"setup_exit\":0"));
    assert!(encoded.contains("\"check_exit\":1"));
    assert!(encoded.contains("\"output\":\"out\""));
    assert!(encoded.contains("\"stopped\":true"));
    assert!(encoded.contains("\"retirement\":\"confirmed\""));
    let h = HoldState {
        active: true,
        revision: 3,
    };
    assert_eq!(h.encode(), "{\"active\":true,\"revision\":3}");
}
