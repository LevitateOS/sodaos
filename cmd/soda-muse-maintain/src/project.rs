use std::time::{Duration, Instant};

use super::command::podman;
use super::json::JsonParser;
use super::release_validation::is_hex_string;

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
    enum Field {
        Str(String),
        Int(i64),
        Bool(bool),
        Null,
    }
    let mut p = JsonParser::new(body);
    p.skip_ws();
    if p.peek() != Some(b'{') {
        return Err(invalid.clone());
    }
    p.bump();
    let mut pairs: Vec<(String, Field)> = Vec::new();
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(invalid.clone());
        }
        let key = p.parse_string().map_err(|_| invalid.clone())?;
        if pairs.iter().any(|(k, _)| k == &key) {
            return Err(invalid.clone());
        }
        p.skip_ws();
        if p.peek() != Some(b':') {
            return Err(invalid.clone());
        }
        p.bump();
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        let field = match p.peek() {
            Some(b'"') => Field::Str(p.parse_string().map_err(|_| invalid.clone())?),
            Some(b't') | Some(b'f') | Some(b'n') => {
                // Literals must be exact; prefixes fail like encoding/json.
                if p.bytes[p.pos..].starts_with(b"true") {
                    p.pos += 4;
                    Field::Bool(true)
                } else if p.bytes[p.pos..].starts_with(b"false") {
                    p.pos += 5;
                    Field::Bool(false)
                } else if p.bytes[p.pos..].starts_with(b"null") {
                    p.pos += 4;
                    Field::Null
                } else {
                    return Err(invalid.clone());
                }
            }
            Some(b'-') | Some(b'0'..=b'9') => {
                Field::Int(p.parse_payload_int().map_err(|_| invalid.clone())?)
            }
            _ => return Err(invalid.clone()),
        };
        pairs.push((key, field));
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(invalid.clone()),
        }
        first = false;
    }
    p.skip_ws();
    if !p.eof() {
        return Err(invalid.clone());
    }
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
            (_, Field::Null) => {}
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
