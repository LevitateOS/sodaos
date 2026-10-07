// Fresh review candidate preparation.
use super::{valid_preparation_id, Preparation, MAX_SOURCE_BUNDLE, ROLE_REVIEWER};
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct FactoryCandidate {
    pub preparation: Preparation,
    pub source_preparation: String,
    pub bundle: Vec<u8>,
}

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

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl<'de> Deserialize<'de> for FactoryCandidate {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct CandidateVisitor;
        impl<'de> Visitor<'de> for CandidateVisitor {
            type Value = FactoryCandidate;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a factory candidate object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut preparation = None;
                let mut source_preparation = None;
                let mut bundle = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("preparation") {
                        if let Some(v) = map.next_value::<Option<Preparation>>()? {
                            preparation = Some(v);
                        }
                    } else if key.eq_ignore_ascii_case("source_preparation") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            source_preparation = Some(v);
                        }
                    } else if key.eq_ignore_ascii_case("bundle") {
                        if let Some(v) = map.next_value::<Option<super::setup::GoBytes>>()? {
                            bundle = Some(v.0);
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["preparation", "source_preparation", "bundle"],
                        ));
                    }
                }
                Ok(FactoryCandidate {
                    preparation: preparation.unwrap_or_default(),
                    source_preparation: source_preparation.unwrap_or_default(),
                    bundle: bundle.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_map(CandidateVisitor)
    }
}
