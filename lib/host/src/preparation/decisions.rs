// Preparation decision records: requirement acceptance and admin approval.
use super::{valid_commit, valid_decision_id, valid_digest};
use crate::json::{BoundMap, Kind, Spec};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RequirementAcceptance {
    pub id: String,
    pub revision: i64,
    pub approver: i64,
    pub source_commit: String,
    pub digest: String,
}

pub(crate) const REQUIREMENT_ACCEPTANCE_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
    Spec {
        name: "approver",
        kind: Kind::I64,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "digest",
        kind: Kind::Str,
    },
];

impl RequirementAcceptance {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_decision_id(&self.id)
            || self.revision < 0
            || self.approver <= 0
            || !valid_commit(&self.source_commit)
            || !valid_digest(&self.digest)
        {
            return Err("invalid requirement acceptance reference".to_string());
        }
        Ok(())
    }

    pub fn from_map(m: &BoundMap) -> Self {
        RequirementAcceptance {
            id: m.take_string("id"),
            revision: m.take_i64("revision"),
            approver: m.take_i64("approver"),
            source_commit: m.take_string("source_commit"),
            digest: m.take_string("digest"),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AdminApproval {
    pub id: String,
    pub revision: i64,
    pub approver: i64,
    pub effects_digest: String,
}

pub(crate) const ADMIN_APPROVAL_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
    Spec {
        name: "approver",
        kind: Kind::I64,
    },
    Spec {
        name: "effects_digest",
        kind: Kind::Str,
    },
];

impl AdminApproval {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_decision_id(&self.id)
            || self.revision < 0
            || self.approver <= 0
            || !valid_digest(&self.effects_digest)
        {
            return Err("invalid privileged-effect approval reference".to_string());
        }
        Ok(())
    }

    pub fn from_map(m: &BoundMap) -> Self {
        AdminApproval {
            id: m.take_string("id"),
            revision: m.take_i64("revision"),
            approver: m.take_i64("approver"),
            effects_digest: m.take_string("effects_digest"),
        }
    }
}
