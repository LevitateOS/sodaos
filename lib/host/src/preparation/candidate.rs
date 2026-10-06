// Fresh review candidate preparation.
use super::{
    valid_preparation_id, Preparation, MAX_SOURCE_BUNDLE, PREPARATION_SPECS, ROLE_REVIEWER,
};
use crate::json::{self, BoundMap, Kind, Spec, Value};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FactoryCandidate {
    pub preparation: Preparation,
    pub source_preparation: String,
    pub bundle: Vec<u8>,
}

const FACTORY_CANDIDATE_SPECS: &[Spec] = &[
    Spec {
        name: "preparation",
        kind: Kind::Object {
            go_type: "project.Preparation",
            struct_name: "Preparation",
            specs: PREPARATION_SPECS,
        },
    },
    Spec {
        name: "source_preparation",
        kind: Kind::Str,
    },
    Spec {
        name: "bundle",
        kind: Kind::Bytes,
    },
];

impl FactoryCandidate {
    pub fn validate(&self) -> Result<(), String> {
        self.preparation.validate()?;
        if self.preparation.role != ROLE_REVIEWER {
            return Err("only review receives a fresh candidate preparation".to_string());
        }
        if !valid_preparation_id(&self.source_preparation)
            || self.source_preparation == self.preparation.id
        {
            return Err("candidate preparation requires a fresh identity".to_string());
        }
        if self.bundle.is_empty() || self.bundle.len() > MAX_SOURCE_BUNDLE {
            return Err("candidate source bundle exceeds preparation bounds".to_string());
        }
        Ok(())
    }

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "FactoryCandidate", FACTORY_CANDIDATE_SPECS, false)
            .map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        FactoryCandidate {
            preparation: Preparation::from_map(&m.take_map("preparation")),
            source_preparation: m.take_string("source_preparation"),
            bundle: m.take_bytes("bundle"),
        }
    }
}
