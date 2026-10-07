// Preparation decision records: requirement acceptance and admin approval.
use super::{valid_commit, valid_decision_id, valid_digest};
use crate::json::SignedInteger;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct RequirementAcceptance {
    pub id: String,
    pub revision: i64,
    pub approver: i64,
    pub source_commit: String,
    pub digest: String,
}

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
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct AdminApproval {
    pub id: String,
    pub revision: i64,
    pub approver: i64,
    pub effects_digest: String,
}

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
}

macro_rules! typed_decision {
    ($ty:ident, $expect:literal, {$($field:ident : $kind:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where A: MapAccess<'de> {
                        let mut out = $ty { $($field: Default::default()),+ };
                        while let Some(key) = map.next_key::<String>()? {
                            let mut recognized = false;
                            $(if key.eq_ignore_ascii_case(stringify!($field)) {
                                if let Some(value) = map.next_value::<Option<$kind>>()? { out.$field = value.into(); }
                                recognized = true;
                            })+
                            if !recognized { return Err(de::Error::unknown_field(&key, &[$(stringify!($field)),+])); }
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_decision!(RequirementAcceptance, "a requirement acceptance object", {
    id: String, revision: SignedInteger, approver: SignedInteger, source_commit: String, digest: String
});
typed_decision!(AdminApproval, "an admin approval object", {
    id: String, revision: SignedInteger, approver: SignedInteger, effects_digest: String
});
