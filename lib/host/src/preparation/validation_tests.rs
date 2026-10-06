use super::*;
use std::collections::HashMap;

pub(super) const DIGEST: &str = "32c794ef2201b76b757bfba2c23bba06dcc5a8c6121f6fabc99115ef12043ced";
pub(super) const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
pub(super) const EFFECTS: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

pub(super) fn pid() -> String {
    format!("p{}", "a".repeat(24))
}

pub(super) fn fid() -> String {
    format!("f{}", "b".repeat(24))
}

pub(super) fn did(n: u8) -> String {
    format!("d{:024x}", n)
}

fn requirements() -> RequirementAcceptance {
    RequirementAcceptance {
        id: did(1),
        revision: 0,
        approver: 1,
        source_commit: COMMIT.to_string(),
        digest: EFFECTS.to_string(),
    }
}

fn approval() -> AdminApproval {
    AdminApproval {
        id: did(2),
        revision: 0,
        approver: 1,
        effects_digest: EFFECTS.to_string(),
    }
}

fn preparation() -> Preparation {
    Preparation {
        id: fid(),
        project: pid(),
        role: ROLE_CODER.to_string(),
        revision: 0,
        requirements: requirements(),
        approval: approval(),
        source_commit: COMMIT.to_string(),
        setup_digest: DIGEST.to_string(),
        tools: vec![],
        credential: String::new(),
    }
}

fn setup() -> ApprovedSetup {
    ApprovedSetup {
        files: [
            ("setup.sh".to_string(), b"#!/bin/sh\n".to_vec()),
            ("check.sh".to_string(), b"#!/bin/sh\n".to_vec()),
        ]
        .into_iter()
        .collect(),
        bundle: b"BUNDLE".to_vec(),
    }
}

#[test]
fn validators_match_go_shapes() {
    assert!(valid_preparation_id(&fid()));
    assert!(!valid_preparation_id("pbbbbbbbbbbbbbbbbbbbbbbbb"));
    assert!(!valid_preparation_id("fBBBBBBBBBBBBBBBBBBBBBBBB"));
    assert!(!valid_preparation_id("fbbbb"));
    assert!(valid_decision_id(&did(1)));
    assert!(!valid_decision_id(&fid()));
    assert!(valid_digest(DIGEST));
    assert!(!valid_digest(&DIGEST[..63]));
    assert!(valid_commit(COMMIT));
    assert!(!valid_commit(""));
    assert!(valid_factory_role("soda-coder"));
    assert!(valid_factory_role("soda-reviewer"));
    assert!(!valid_factory_role("coder"));
    for phase in [
        "approved",
        "waiting",
        "running",
        "ready",
        "failed",
        "stopped",
        "interrupted",
    ] {
        assert!(valid_prepare_phase(phase));
    }
    assert!(!valid_prepare_phase("done"));
    assert!(valid_approved_name("setup.sh"));
    assert!(valid_approved_name("a"));
    assert!(valid_approved_name(&"a".repeat(64)));
    assert!(!valid_approved_name(""));
    assert!(!valid_approved_name(".sh"));
    assert!(!valid_approved_name("a..b"));
    assert!(!valid_approved_name(".."));
    assert!(!valid_approved_name("a/b"));
    assert!(!valid_approved_name(&"a".repeat(65)));
    assert!(valid_tool_name("git"));
    assert!(valid_tool_name("c++"));
    assert!(!valid_tool_name("Git"));
    assert!(!valid_tool_name(""));
    assert!(!valid_tool_name(&"a".repeat(33)));
}

#[test]
fn setup_digest_matches_reference() {
    assert_eq!(setup_digest_of(&setup().files), DIGEST);
}

#[test]
fn preparation_validation_branches() {
    assert!(preparation().validate().is_ok());
    let mut p = preparation();
    p.id = "x".to_string();
    assert_eq!(p.validate().unwrap_err(), "invalid preparation identity");
    let mut p = preparation();
    p.requirements.approver = 0;
    assert_eq!(
        p.validate().unwrap_err(),
        "invalid requirement acceptance reference"
    );
    let mut p = preparation();
    p.approval.id = "x".to_string();
    assert_eq!(
        p.validate().unwrap_err(),
        "invalid privileged-effect approval reference"
    );
    let mut p = preparation();
    p.source_commit = "zz".to_string();
    assert_eq!(
        p.validate().unwrap_err(),
        "invalid preparation source identity"
    );
    let mut p = preparation();
    p.tools = vec!["t".to_string(); 9];
    assert_eq!(p.validate().unwrap_err(), "too many required tools");
    let mut p = preparation();
    p.tools = vec!["Bad!".to_string()];
    assert_eq!(p.validate().unwrap_err(), "invalid required tool name");
    let mut p = preparation();
    p.credential = "../x".to_string();
    assert_eq!(p.validate().unwrap_err(), "invalid credential reference");
}

#[test]
fn approved_setup_validation_branches() {
    assert!(setup().validate().is_ok());
    let mut s = setup();
    s.files.remove("setup.sh");
    assert_eq!(
        s.validate().unwrap_err(),
        "approved setup entrypoint is required"
    );
    let mut s = setup();
    s.files.remove("check.sh");
    assert_eq!(
        s.validate().unwrap_err(),
        "approved check entrypoint is required"
    );
    let mut s = setup();
    s.files.insert("bad/name".to_string(), b"x".to_vec());
    assert_eq!(s.validate().unwrap_err(), "invalid approved file name");
    let mut s = setup();
    s.files.insert("empty.sh".to_string(), Vec::new());
    assert_eq!(s.validate().unwrap_err(), "invalid approved file size");
    let mut s = setup();
    s.files
        .insert("big.sh".to_string(), vec![0u8; 32 * 1024 + 1]);
    assert_eq!(s.validate().unwrap_err(), "invalid approved file size");
    let mut s = setup();
    for i in 0..7 {
        s.files.insert(format!("f{i}.sh"), vec![0u8; 32 * 1024]);
    }
    // 2 + 7 = 9 files exceeds the count first.
    assert_eq!(s.validate().unwrap_err(), "invalid approved file set");
    let mut files = HashMap::new();
    files.insert("setup.sh".to_string(), vec![0u8; 32 * 1024]);
    files.insert("check.sh".to_string(), vec![0u8; 32 * 1024]);
    for i in 0..6 {
        files.insert(format!("f{i}.sh"), vec![0u8; 32 * 1024]);
    }
    // Exactly 8 files at 32KiB each = 256KiB total.
    let s = ApprovedSetup {
        files,
        bundle: b"x".to_vec(),
    };
    assert_eq!(
        s.validate().unwrap_err(),
        "approved inputs exceed the bounded size"
    );
    let mut s = setup();
    s.bundle.clear();
    assert_eq!(s.validate().unwrap_err(), "invalid source bundle size");
}

#[test]
fn prepare_validates_digest_binding() {
    let p = Prepare {
        preparation: preparation(),
        setup: setup(),
    };
    assert!(p.validate().is_ok());
    let mut bad = preparation();
    bad.setup_digest = EFFECTS.to_string();
    let p = Prepare {
        preparation: bad,
        setup: setup(),
    };
    assert_eq!(
        p.validate().unwrap_err(),
        "approved inputs do not match their digest"
    );
}

#[test]
fn address_and_hold_validation() {
    let a = PrepareInspect {
        project: pid(),
        id: fid(),
    };
    assert!(a.validate().is_ok());
    let a = PrepareInspect {
        project: "x".to_string(),
        id: fid(),
    };
    assert_eq!(a.validate().unwrap_err(), "invalid preparation address");
    let h = PrepareHold {
        project: pid(),
        hold: true,
        revision: 0,
    };
    assert!(h.validate().is_ok());
    let h = PrepareHold {
        project: pid(),
        hold: false,
        revision: -1,
    };
    assert_eq!(h.validate().unwrap_err(), "invalid maintenance hold");
}

#[test]
fn candidate_validation_branches() {
    let mut prep = preparation();
    prep.role = ROLE_REVIEWER.to_string();
    let c = FactoryCandidate {
        preparation: prep,
        source_preparation: format!("f{}", "c".repeat(24)),
        bundle: b"BUNDLE".to_vec(),
    };
    assert!(c.validate().is_ok());
    let c = FactoryCandidate {
        preparation: preparation(),
        source_preparation: format!("f{}", "c".repeat(24)),
        bundle: b"BUNDLE".to_vec(),
    };
    assert_eq!(
        c.validate().unwrap_err(),
        "only review receives a fresh candidate preparation"
    );
    let mut prep = preparation();
    prep.role = ROLE_REVIEWER.to_string();
    let id = prep.id.clone();
    let c = FactoryCandidate {
        preparation: prep,
        source_preparation: id,
        bundle: b"BUNDLE".to_vec(),
    };
    assert_eq!(
        c.validate().unwrap_err(),
        "candidate preparation requires a fresh identity"
    );
    let mut prep = preparation();
    prep.role = ROLE_REVIEWER.to_string();
    let c = FactoryCandidate {
        preparation: prep,
        source_preparation: format!("f{}", "c".repeat(24)),
        bundle: Vec::new(),
    };
    assert_eq!(
        c.validate().unwrap_err(),
        "candidate source bundle exceeds preparation bounds"
    );
}
