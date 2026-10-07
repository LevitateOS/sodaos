use super::write_json;
use crate::json_serde::strict;
use crate::model::Trust;
use crate::Error;
use base64::Engine;
use serde::de::{MapAccess, SeqAccess, Visitor};
use serde::ser::{SerializeMap, SerializeSeq};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::collections::BTreeMap;
use std::fmt;

#[derive(Debug, Clone, Default, Serialize)]
struct Requirement {
    #[serde(rename = "type")]
    type_name: String,
    #[serde(rename = "keyDatas", skip_serializing_if = "Vec::is_empty")]
    key_datas: Vec<String>,
    #[serde(rename = "signedIdentity", skip_serializing_if = "BTreeMap::is_empty")]
    signed_identity: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize)]
pub(crate) struct PolicyDocument {
    #[serde(rename = "default")]
    default_policy: Vec<Requirement>,
    transports: BTreeMap<String, BTreeMap<String, Vec<Requirement>>>,
}

#[derive(Serialize)]
#[serde(untagged)]
enum ScopeValue {
    Existing(PolicyValue),
    Generated(Vec<Requirement>),
}

// Arbitrary policy objects retain pair order, while numeric leaves retain
// their original valid token instead of being rounded through f64.
enum PolicyValue {
    Null,
    Bool(bool),
    Number(Box<serde_json::value::RawValue>),
    String(String),
    Array(Vec<PolicyValue>),
    Object(Vec<(String, PolicyValue)>),
}

struct PolicyPairs(Vec<(String, Box<serde_json::value::RawValue>)>);
impl<'de> Deserialize<'de> for PolicyPairs {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PolicyPairs;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a policy object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> Result<PolicyPairs, M::Error> {
                let mut entries = Vec::new();
                while let Some(key) = map.next_key::<String>()? {
                    entries.push((key, map.next_value::<Box<serde_json::value::RawValue>>()?));
                }
                Ok(PolicyPairs(entries))
            }
        }
        d.deserialize_map(V)
    }
}

struct PolicyItems(Vec<Box<serde_json::value::RawValue>>);
impl<'de> Deserialize<'de> for PolicyItems {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PolicyItems;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a policy array")
            }
            fn visit_seq<S: SeqAccess<'de>>(self, mut seq: S) -> Result<PolicyItems, S::Error> {
                let mut values = Vec::new();
                while let Some(value) = seq.next_element::<Box<serde_json::value::RawValue>>()? {
                    values.push(value);
                }
                Ok(PolicyItems(values))
            }
        }
        d.deserialize_seq(V)
    }
}

impl PolicyValue {
    fn from_raw<E: serde::de::Error>(raw: Box<serde_json::value::RawValue>) -> Result<Self, E> {
        let Some(raw) = serde_json::from_str::<Option<Box<serde_json::value::RawValue>>>(raw.get())
            .map_err(E::custom)?
        else {
            return Ok(Self::Null);
        };
        match raw.get().as_bytes().first().copied() {
            Some(b'{') => {
                let PolicyPairs(entries) = serde_json::from_str(raw.get()).map_err(E::custom)?;
                entries
                    .into_iter()
                    .map(|(key, value)| Self::from_raw::<E>(value).map(|value| (key, value)))
                    .collect::<Result<Vec<_>, E>>()
                    .map(Self::Object)
            }
            Some(b'[') => {
                let PolicyItems(values) = serde_json::from_str(raw.get()).map_err(E::custom)?;
                values
                    .into_iter()
                    .map(Self::from_raw::<E>)
                    .collect::<Result<Vec<_>, E>>()
                    .map(Self::Array)
            }
            Some(b'"') => serde_json::from_str::<String>(raw.get())
                .map(Self::String)
                .map_err(E::custom),
            Some(b't' | b'f') => serde_json::from_str::<bool>(raw.get())
                .map(Self::Bool)
                .map_err(E::custom),
            _ => Ok(Self::Number(raw)),
        }
    }

    fn is_null(&self) -> bool {
        matches!(self, Self::Null)
    }
}

impl<'de> Deserialize<'de> for PolicyValue {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        Self::from_raw::<D::Error>(Box::<serde_json::value::RawValue>::deserialize(d)?)
    }
}

impl Serialize for PolicyValue {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        match self {
            Self::Null => serializer.serialize_unit(),
            Self::Bool(value) => serializer.serialize_bool(*value),
            Self::Number(raw) => raw.serialize(serializer),
            Self::String(value) => serializer.serialize_str(value),
            Self::Array(values) => {
                let mut seq = serializer.serialize_seq(Some(values.len()))?;
                for value in values {
                    seq.serialize_element(value)?;
                }
                seq.end()
            }
            Self::Object(entries) => {
                let mut map = serializer.serialize_map(Some(entries.len()))?;
                for (key, value) in entries {
                    map.serialize_entry(key, value)?;
                }
                map.end()
            }
        }
    }
}

fn simple_rule(kind: &str) -> Requirement {
    Requirement {
        type_name: kind.to_string(),
        ..Requirement::default()
    }
}

fn requirement(t: &Trust, repo: &str) -> Result<Requirement, Error> {
    let role = t.role(repo)?;
    let key_datas = t
        .keys
        .get(&role)
        .into_iter()
        .flatten()
        .map(|key| base64::engine::general_purpose::STANDARD.encode(key.as_bytes()))
        .collect();
    let signed_identity = BTreeMap::from([
        ("type".to_string(), "exactRepository".to_string()),
        ("dockerRepository".to_string(), repo.to_string()),
    ]);
    Ok(Requirement {
        type_name: "sigstoreSigned".to_string(),
        key_datas,
        signed_identity,
    })
}

pub(crate) fn policy_for_publish(
    t: &Trust,
    repo: &str,
    transport: &str,
    scope: &str,
) -> Result<PolicyDocument, Error> {
    policy_for(t, repo, transport, scope)
}

pub(crate) fn registry_config_for_publish(out: &str, t: &Trust) -> Result<String, Error> {
    registry_config(out, t)
}

pub(super) fn policy_for(
    t: &Trust,
    repo: &str,
    transport: &str,
    scope: &str,
) -> Result<PolicyDocument, Error> {
    let req = requirement(t, repo)?;
    Ok(PolicyDocument {
        default_policy: vec![simple_rule("reject")],
        transports: BTreeMap::from([(
            transport.to_string(),
            BTreeMap::from([(scope.to_string(), vec![req])]),
        )]),
    })
}

pub(crate) fn local_policy(transport: &str, path: &str) -> PolicyDocument {
    PolicyDocument {
        default_policy: vec![simple_rule("reject")],
        transports: BTreeMap::from([(
            transport.to_string(),
            BTreeMap::from([(
                path.to_string(),
                vec![simple_rule("insecureAcceptAnything")],
            )]),
        )]),
    }
}

fn soda_trust_repos(t: &Trust) -> Vec<String> {
    let mut repos = vec![
        format!("{}-host", t.prefix),
        format!("{}-release", t.prefix),
    ];
    for name in crate::payload::NAMES {
        repos.push(format!("{}-{name}", t.prefix));
    }
    for channel in ["candidate", "preview", "stable"] {
        repos.push(format!("{}-channel-{channel}", t.prefix));
    }
    repos
}

fn soda_override_exists(existing: &str, repo: &str) -> bool {
    existing == repo
        || existing.starts_with(&format!("{repo}:"))
        || existing.starts_with(&format!("{repo}@"))
        || existing.starts_with(&format!("{repo}/"))
}

fn apply_soda_trust(t: &Trust, docker: &mut BTreeMap<String, ScopeValue>) -> Result<(), Error> {
    for repo in soda_trust_repos(t) {
        if docker
            .keys()
            .any(|existing| soda_override_exists(existing, &repo))
        {
            return Err(Error::msg(
                "existing Soda trust override requires explicit review",
            ));
        }
        docker.insert(
            repo.clone(),
            ScopeValue::Generated(vec![requirement(t, &repo)?]),
        );
    }
    Ok(())
}

#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct PolicyInput {
    #[serde(rename = "default")]
    default_policy: PolicyValue,
    #[serde(default, deserialize_with = "crate::json_serde::null_default")]
    transports: BTreeMap<String, BTreeMap<String, PolicyValue>>,
}

/// `MergePolicy`: emit a proposed policy; never installs it.
pub fn merge_policy(t: &Trust, original: &[u8]) -> Result<Vec<u8>, Error> {
    t.validate()?;
    let input: PolicyInput = strict(original).map_err(|_| Error::refused())?;
    if input.default_policy.is_null() {
        return Err(Error::refused());
    }
    let mut transports: BTreeMap<String, BTreeMap<String, ScopeValue>> = input
        .transports
        .into_iter()
        .map(|(name, scopes)| {
            (
                name,
                scopes
                    .into_iter()
                    .map(|(scope, value)| (scope, ScopeValue::Existing(value)))
                    .collect(),
            )
        })
        .collect();
    let docker = transports.entry("docker".to_string()).or_default();
    // Preserve source members verbatim as values while ordering maps deterministically.
    apply_soda_trust(t, docker)?;
    #[derive(Serialize)]
    struct MergedPolicy {
        #[serde(rename = "default")]
        default_policy: PolicyValue,
        transports: BTreeMap<String, BTreeMap<String, ScopeValue>>,
    }
    let merged = MergedPolicy {
        default_policy: input.default_policy,
        transports,
    };
    crate::document::marshal_go_pretty(&merged)
}

/// `WriteRegistryConfig`: registries.d snippet for Sigstore attachments.
pub fn write_registry_config(out: &str, t: &Trust) -> Result<(), Error> {
    t.validate()?;
    registry_config(out, t)?;
    Ok(())
}

pub(super) fn registry_config(out: &str, t: &Trust) -> Result<String, Error> {
    let dir = format!("{out}/registries.d");
    {
        use std::os::unix::fs::DirBuilderExt;
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&dir)
            .map_err(|e| Error::msg(format!("mkdir {dir}: {e}")))?;
    }
    let mut repos = BTreeMap::new();
    let mut names = vec![
        "host",
        "release",
        "channel-candidate",
        "channel-preview",
        "channel-stable",
    ];
    names.extend(crate::payload::NAMES.iter().copied());
    for name in names {
        repos.insert(
            format!("{}-{name}", t.prefix),
            serde_json::json!({"use-sigstore-attachments": true}),
        );
    }
    write_json(
        &format!("{dir}/soda.yaml"),
        &serde_json::json!({"docker": repos}),
    )?;
    Ok(dir)
}
