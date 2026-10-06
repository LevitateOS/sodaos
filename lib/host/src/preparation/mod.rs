//! Factory preparation domain: roles, approvals, bounded setup inputs and
//! observed states. Port of `internal/project/preparation.go` plus the
//! candidate half of `internal/project/factory_candidate.go` (pure
//! validation and wire shapes only; the store-side grant decisions and
//! durable records stay in Go).

use crate::domain;
use crate::json::{self, BoundMap, Kind, Spec, Value};

mod candidate;
mod decisions;
mod setup;
mod state;

pub use self::candidate::FactoryCandidate;
pub use self::decisions::{AdminApproval, RequirementAcceptance};
pub use self::setup::{setup_digest_of, ApprovedSetup};
pub use self::state::{
    HoldState, PrepareHold, PrepareInspect, PrepareState, PrepareStop, ResolvedTool,
};

pub(crate) use self::state::{HOLD_STATE_SPECS, RESOLVED_TOOL_SPECS};

use self::decisions::{ADMIN_APPROVAL_SPECS, REQUIREMENT_ACCEPTANCE_SPECS};
use self::setup::APPROVED_SETUP_SPECS;

pub const ROLE_CODER: &str = "soda-coder";
pub const ROLE_REVIEWER: &str = "soda-reviewer";

pub const FACTORY_DIR: &str = "/var/lib/soda/factory";
pub const FACTORY_PREPARATIONS_DIR: &str = "/var/lib/soda/factory/preparations";
pub const FACTORY_CREDENTIALS_DIR: &str = "/var/lib/soda/factory/credentials";
pub const FACTORY_HOLD_FILE: &str = "/var/lib/soda/factory/maintenance-hold";
pub const FACTORY_HELPER: &str = "/usr/libexec/soda/project-factory-roles";
pub const FACTORY_SETUP_ENTRY: &str = "setup.sh";
pub const FACTORY_CHECK_ENTRY: &str = "check.sh";

pub const MAX_APPROVED_FILES: usize = 8;
pub const MAX_APPROVED_FILE_SIZE: usize = 32 * 1024;
pub const MAX_APPROVED_TOTAL: usize = 128 * 1024;
pub const MAX_SOURCE_BUNDLE: usize = 512 * 1024;
pub const MAX_PREPARE_TOOLS: usize = 8;

pub const PREPARE_APPROVED: &str = "approved";
pub const PREPARE_WAITING: &str = "waiting";
pub const PREPARE_RUNNING: &str = "running";
pub const PREPARE_READY: &str = "ready";
pub const PREPARE_FAILED: &str = "failed";
pub const PREPARE_STOPPED: &str = "stopped";
pub const PREPARE_INTERRUPTED: &str = "interrupted";

pub fn valid_factory_role(role: &str) -> bool {
    role == ROLE_CODER || role == ROLE_REVIEWER
}

pub fn valid_prepare_phase(phase: &str) -> bool {
    matches!(
        phase,
        "approved" | "waiting" | "running" | "ready" | "failed" | "stopped" | "interrupted"
    )
}

fn is_hex_lower(s: &str) -> bool {
    !s.is_empty()
        && s.bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

/// `^f[0-9a-f]{24}$`
pub fn valid_preparation_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'f' && is_hex_lower(&id[1..])
}

/// `^d[0-9a-f]{24}$`
pub fn valid_decision_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'd' && is_hex_lower(&id[1..])
}

/// `^[0-9a-f]{64}$`
pub fn valid_digest(d: &str) -> bool {
    d.len() == 64 && is_hex_lower(d)
}

/// `^[0-9a-f]{40}$`
pub fn valid_commit(c: &str) -> bool {
    c.len() == 40 && is_hex_lower(c)
}

/// `^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$` without `..`.
pub fn valid_approved_name(name: &str) -> bool {
    let b = name.as_bytes();
    if b.is_empty() || b.len() > 64 || !b[0].is_ascii_alphanumeric() {
        return false;
    }
    if !b[1..]
        .iter()
        .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
    {
        return false;
    }
    !name.contains("..")
}

/// `^[a-z0-9][a-z0-9+_.-]{0,31}$`
pub fn valid_tool_name(name: &str) -> bool {
    let b = name.as_bytes();
    if b.is_empty() || b.len() > 32 || !(b[0].is_ascii_lowercase() || b[0].is_ascii_digit()) {
        return false;
    }
    b[1..].iter().all(|c| {
        c.is_ascii_lowercase() || c.is_ascii_digit() || matches!(*c, b'+' | b'_' | b'.' | b'-')
    })
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Preparation {
    pub id: String,
    pub project: String,
    pub role: String,
    pub revision: i64,
    pub requirements: RequirementAcceptance,
    pub approval: AdminApproval,
    pub source_commit: String,
    pub setup_digest: String,
    pub tools: Vec<String>,
    pub credential: String,
}

pub(crate) const PREPARATION_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
    Spec {
        name: "requirements",
        kind: Kind::Object {
            go_type: "project.RequirementAcceptance",
            struct_name: "RequirementAcceptance",
            specs: REQUIREMENT_ACCEPTANCE_SPECS,
        },
    },
    Spec {
        name: "approval",
        kind: Kind::Object {
            go_type: "project.AdminApproval",
            struct_name: "AdminApproval",
            specs: ADMIN_APPROVAL_SPECS,
        },
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "setup_digest",
        kind: Kind::Str,
    },
    Spec {
        name: "tools",
        kind: Kind::StrList,
    },
    Spec {
        name: "credential",
        kind: Kind::Str,
    },
];

impl Preparation {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_preparation_id(&self.id)
            || !domain::valid_id(&self.project)
            || !valid_factory_role(&self.role)
            || self.revision < 0
        {
            return Err("invalid preparation identity".to_string());
        }
        self.requirements.validate()?;
        self.approval.validate()?;
        if !valid_commit(&self.source_commit) || !valid_digest(&self.setup_digest) {
            return Err("invalid preparation source identity".to_string());
        }
        if self.tools.len() > MAX_PREPARE_TOOLS {
            return Err("too many required tools".to_string());
        }
        for name in &self.tools {
            if !valid_tool_name(name) {
                return Err("invalid required tool name".to_string());
            }
        }
        if !self.credential.is_empty() && !valid_approved_name(&self.credential) {
            return Err("invalid credential reference".to_string());
        }
        Ok(())
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Preparation {
            id: m.take_string("id"),
            project: m.take_string("project"),
            role: m.take_string("role"),
            revision: m.take_i64("revision"),
            requirements: RequirementAcceptance::from_map(&m.take_map("requirements")),
            approval: AdminApproval::from_map(&m.take_map("approval")),
            source_commit: m.take_string("source_commit"),
            setup_digest: m.take_string("setup_digest"),
            tools: m.take_str_list("tools"),
            credential: m.take_string("credential"),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Prepare {
    pub preparation: Preparation,
    pub setup: ApprovedSetup,
}

const PREPARE_SPECS: &[Spec] = &[
    Spec {
        name: "preparation",
        kind: Kind::Object {
            go_type: "project.Preparation",
            struct_name: "Preparation",
            specs: PREPARATION_SPECS,
        },
    },
    Spec {
        name: "setup",
        kind: Kind::Object {
            go_type: "project.ApprovedSetup",
            struct_name: "ApprovedSetup",
            specs: APPROVED_SETUP_SPECS,
        },
    },
];

impl Prepare {
    pub fn validate(&self) -> Result<(), String> {
        self.preparation.validate()?;
        self.setup.validate()?;
        if self.preparation.setup_digest != setup_digest_of(&self.setup.files) {
            return Err("approved inputs do not match their digest".to_string());
        }
        Ok(())
    }

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Prepare", PREPARE_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Prepare {
            preparation: Preparation::from_map(&m.take_map("preparation")),
            setup: ApprovedSetup::from_map(&m.take_map("setup")),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    const DIGEST: &str = "32c794ef2201b76b757bfba2c23bba06dcc5a8c6121f6fabc99115ef12043ced";
    const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
    const EFFECTS: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

    fn pid() -> String {
        format!("p{}", "a".repeat(24))
    }

    fn fid() -> String {
        format!("f{}", "b".repeat(24))
    }

    fn did(n: u8) -> String {
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
        let v = crate::json::decode_strict(prepare_json().as_bytes()).unwrap();
        let p = Prepare::from_value(&v).unwrap();
        assert!(p.validate().is_ok());
        assert_eq!(p.setup.files["setup.sh"], b"#!/bin/sh\n");
        assert_eq!(p.setup.bundle, b"BUNDLE");
    }

    #[test]
    fn decode_messages_match_go_exactly() {
        // Pinned against strictjson + encoding/json from the pinned toolchain.
        let cases = [
            (
                "{\"preparation\":{\"revision\":\"x\"},\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}"
                    .to_string(),
                "decode request: json: cannot unmarshal string into Go struct field Preparation.preparation.revision of type int64",
            ),
            (
                format!(
                    "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":\"!!!\"}}}}",
                    prep_json()
                ),
                "decode request: illegal base64 data at input byte 0",
            ),
            (
                format!(
                    "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":5}},\"bundle\":\"QUJD\"}}}}",
                    prep_json()
                ),
                "decode request: json: cannot unmarshal number into Go struct field ApprovedSetup.setup.files of type []uint8",
            ),
            (
                format!(
                    "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":[300]}}}}",
                    prep_json()
                ),
                "decode request: json: cannot unmarshal number 300 into Go struct field ApprovedSetup.setup.bundle of type uint8",
            ),
            (
                format!(
                    "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":{{}}}}}}",
                    prep_json()
                ),
                "decode request: json: cannot unmarshal object into Go struct field ApprovedSetup.setup.bundle of type []uint8",
            ),
            (
                "{\"preparation\":[],\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}"
                    .to_string(),
                "decode request: json: cannot unmarshal array into Go struct field Prepare.preparation of type project.Preparation",
            ),
            (
                format!(
                    "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":\"QUJD\"}},\"bundle\":\"QUJD\"}},\"zzz\":1}}",
                    prep_json()
                ),
                "decode request: json: unknown field \"zzz\"",
            ),
            (
                "{\"requirements\":{\"revision\":\"x\"}}".to_string(),
                // Decoded as Prepare: unknown top-level field (sorted first).
                "decode request: json: unknown field \"requirements\"",
            ),
        ];
        for (body, want) in cases {
            let v = crate::json::decode_strict(body.as_bytes()).unwrap();
            assert_eq!(Prepare::from_value(&v).unwrap_err(), want, "{body}");
        }
        // Numeric byte arrays decode; null struct binds zero.
        let body = format!(
            "{{\"preparation\":{},\"setup\":{{\"files\":{{\"a\":[1,2]}}}}}}",
            prep_json()
        );
        // Missing bundle binds empty (validity is Validate's job).
        let v = crate::json::decode_strict(body.as_bytes()).unwrap();
        let p = Prepare::from_value(&v).unwrap();
        assert_eq!(p.setup.files["a"], vec![1u8, 2]);
        assert!(p.setup.bundle.is_empty());
        let v = crate::json::decode_strict(
            b"{\"preparation\":null,\"setup\":{\"files\":{\"a\":\"QUJD\"},\"bundle\":\"QUJD\"}}",
        )
        .unwrap();
        let p = Prepare::from_value(&v).unwrap();
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
}
