use std::time::{Duration, Instant};

use super::command::podman;
use super::release_validation::is_hex_string;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::fmt;

const INSPECT_FORMAT: &str = r#"{"id":{{json .ID}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"pid":{{json .State.Pid}},"running":{{json .State.Running}}}"#;

#[derive(PartialEq, Debug)]
pub(crate) struct Observation {
    pub(crate) id: String,
    pub(crate) project: String,
    pub(crate) owner: String,
    pub(crate) pid: i64,
    pub(crate) running: bool,
}

fn inspect_project(key: &str, project: &str, deadline: Instant) -> Result<Observation, String> {
    let body = podman(&["inspect", "--format", INSPECT_FORMAT, key], deadline)?;
    if body.len() > 8192 {
        return Err(String::from("invalid project observation"));
    }
    let o = decode_observation(&body)?;
    validate_observation(&o, project)?;
    Ok(o)
}

pub(crate) fn validate_observation(o: &Observation, project: &str) -> Result<(), String> {
    if !is_hex_string(&o.id, 64) || o.project != project {
        return Err(String::from("project container identity differs"));
    }
    match o.owner.parse::<i64>() {
        Ok(n) if n > 0 => {}
        _ => return Err(String::from("project owner label is invalid")),
    }
    Ok(())
}

// decode_observation mirrors strictjson.Decode into project inspection:
// one object, no duplicate or unknown fields, Go value semantics.
// Pairs apply in sorted-key order like the normalized re-decode, so exact
// lowercase keys win over case-variant duplicates exactly as in Go.
pub(crate) fn decode_observation(body: &[u8]) -> Result<Observation, String> {
    let invalid = String::from("invalid project observation");
    if body.len() > 1 << 20 {
        return Err(invalid.clone());
    }
    if std::str::from_utf8(body).is_err() {
        return Err(invalid.clone());
    }
    #[derive(Deserialize)]
    #[serde(untagged)]
    enum Field {
        Str(String),
        Int(i64),
        Bool(bool),
        Null(()),
    }
    struct Pairs(Vec<(String, Field)>);
    impl<'de> Deserialize<'de> for Pairs {
        fn deserialize<D>(d: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            struct PairsVisitor;
            impl<'de> Visitor<'de> for PairsVisitor {
                type Value = Pairs;
                fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                    f.write_str("an observation object")
                }
                fn visit_map<A>(self, mut map: A) -> Result<Pairs, A::Error>
                where
                    A: MapAccess<'de>,
                {
                    let mut pairs = Vec::new();
                    while let Some(key) = map.next_key::<String>()? {
                        if pairs.iter().any(|(seen, _)| seen == &key) {
                            return Err(de::Error::custom("duplicate key"));
                        }
                        pairs.push((key, map.next_value::<Field>()?));
                    }
                    Ok(Pairs(pairs))
                }
            }
            d.deserialize_map(PairsVisitor)
        }
    }
    let mut deserializer = serde_json::Deserializer::from_slice(body);
    let Pairs(mut pairs) = Pairs::deserialize(&mut deserializer).map_err(|_| invalid.clone())?;
    deserializer.end().map_err(|_| invalid.clone())?;
    pairs.sort_by(|a, b| a.0.as_bytes().cmp(b.0.as_bytes()));
    let mut o = Observation {
        id: String::new(),
        project: String::new(),
        owner: String::new(),
        pid: 0,
        running: false,
    };
    for (key, field) in &pairs {
        let slot = match observation_slot(key) {
            Some(s) => s,
            None => return Err(invalid.clone()),
        };
        match (slot, field) {
            (0, Field::Str(v)) => o.id = v.clone(),
            (1, Field::Str(v)) => o.project = v.clone(),
            (2, Field::Str(v)) => o.owner = v.clone(),
            (3, Field::Int(v)) => o.pid = *v,
            (4, Field::Bool(v)) => o.running = *v,
            (_, Field::Null(_)) => {}
            _ => return Err(invalid.clone()),
        }
    }
    Ok(o)
}

fn observation_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 5] = ["id", "project", "owner", "pid", "running"];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

pub(crate) fn wait_project(project: &str, deadline: Instant) -> Result<Observation, String> {
    let end = Instant::now() + Duration::from_secs(10);
    let end = end.min(deadline);
    let name = format!("soda-{project}");
    loop {
        let target = inspect_project(&name, project, end)?;
        if target.running && target.pid > 0 {
            return Ok(target);
        }
        if Instant::now() >= end {
            return Err(String::from(
                "project did not become running within maintenance deadline",
            ));
        }
        std::thread::sleep(Duration::from_millis(100));
    }
}

pub(crate) fn confirm_project(target: &Observation, deadline: Instant) -> Result<(), String> {
    let live = inspect_project(&target.id, &target.project, deadline)?;
    if live != *target || !live.running {
        return Err(String::from(
            "project incarnation changed during maintenance",
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "project_tests.rs"]
mod project_tests;
