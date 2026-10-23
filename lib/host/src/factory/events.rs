//! Typed native task-event journal (D05 / FW13).
//!
//! One bounded per-run journal beside the existing private host receipts
//! (`Factory.state_dir`), under the existing run lock. Events bind parent
//! run, native invocation, plan digest, task key, sequence and the previous
//! event digest; ordinary prose can never advance task state. The read path
//! (`FactoryTaskEvents`) pages whole events with a next-sequence/terminal
//! cursor and is free of checkpoint mutations. Journal exhaustion stops
//! admission with a terminal reason; it never drops events.

use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

use crate::domain;
use crate::json;
use crate::preparation;

use super::run::valid_factory_run_id;

/// Typed task observations. Only these advance task state.
pub const TASK_EVENT_CHILD_START_INTENT: &str = "child_start_intent";
pub const TASK_EVENT_CHILD_STARTED: &str = "child_started";
pub const TASK_EVENT_CHILD_REPORTED: &str = "child_reported";
pub const TASK_EVENT_PARENT_STARTED: &str = "parent_task_started";
pub const TASK_EVENT_PARENT_OUTPUT: &str = "parent_output_observed";
pub const TASK_EVENT_CHECKPOINT_PREPARED: &str = "checkpoint_prepared";
pub const TASK_EVENT_TASK_COMMITTED: &str = "task_committed";
pub const TASK_EVENT_TASK_FAILED: &str = "task_failed";
pub const TASK_EVENT_RUN_STOPPED: &str = "run_stopped";

/// `ValidTaskEventType`: the seven typed task observations.
pub fn valid_task_event_type(event_type: &str) -> bool {
    matches!(
        event_type,
        "parent_task_started"
            | "parent_output_observed"
            | "child_start_intent"
            | "child_started"
            | "child_reported"
            | "checkpoint_prepared"
            | "task_committed"
            | "task_failed"
            | "run_stopped"
    )
}

/// `TaskEventTerminalType`: `run_stopped` alone ends the stream. A failed
/// task ends its task, not the run; new task admission stops through the
/// run record instead.
pub fn task_event_terminal_type(event_type: &str) -> bool {
    event_type == TASK_EVENT_RUN_STOPPED
}

/// `MaxTaskEventBytes`: one encoded event.
pub const MAX_TASK_EVENT_BYTES: usize = 16 * 1024;
/// `MaxTaskEventsPerRead`: one paged response.
pub const MAX_TASK_EVENTS_PER_READ: usize = 64;
/// `MinTaskEventsBytes` / `MaxTaskEventsBytes`: one read request.
pub const MIN_TASK_EVENTS_BYTES: i64 = 1;
pub const MAX_TASK_EVENTS_BYTES: i64 = 24 * 1024 - 256;
/// `MaxTaskJournalEvents` / `MaxTaskJournalBytes`: the per-run journal,
/// reserving two terminal events. Exhaustion stops admission; events are
/// never dropped.
pub const MAX_TASK_JOURNAL_EVENTS: usize = 512;
pub const MAX_TASK_JOURNAL_BYTES: usize = 8 << 20;
pub const TASK_JOURNAL_RESERVED_TERMINAL: usize = 2;

/// Plan-local task key: `[a-z][a-z0-9_-]{0,31}`.
pub fn valid_task_key(key: &str) -> bool {
    let bytes = key.as_bytes();
    !bytes.is_empty()
        && bytes.len() <= 32
        && bytes[0].is_ascii_lowercase()
        && bytes[1..]
            .iter()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'-' || *c == b'_')
}

/// `project.TaskEvent`: one typed native observation inside an admitted
/// parent run. Sequences start at 1 and advance contiguously; `prev_digest`
/// chains each event to its predecessor (empty for sequence 1).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TaskEvent {
    pub sequence: i64,
    pub prev_digest: String,
    pub event_type: String,
    pub task_key: String,
    pub child_id: String,
    pub invocation: String,
    pub plan_digest: String,
    pub detail: String,
}

macro_rules! typed_event_fields {
    ($ty:ty, $expect:literal, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where A: MapAccess<'de> {
                        let mut out = <$ty>::default();
                        while let Some(key) = map.next_key::<String>()? {
                            $(if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<$value>>()? { out.$field = value.into(); }
                                continue;
                            })+
                            return Err(de::Error::unknown_field(&key, &[$($name),+]));
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_event_fields!(TaskEvent, "a task event object", {
    sequence => "sequence": json::SignedInteger,
    prev_digest => "prev_digest": String,
    event_type => "type": String,
    task_key => "task_key": String,
    child_id => "child_id": String,
    invocation => "invocation": String,
    plan_digest => "plan_digest": String,
    detail => "detail": String,
});

impl TaskEvent {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    /// Canonical encoding, field-ordered like Go's struct marshal, for
    /// chain digests and journal bytes.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"sequence\":");
        out.push_str(&self.sequence.to_string());
        out.push_str(",\"prev_digest\":");
        out.push_str(&json::quote(&self.prev_digest));
        out.push_str(",\"type\":");
        out.push_str(&json::quote(&self.event_type));
        out.push_str(",\"task_key\":");
        out.push_str(&json::quote(&self.task_key));
        if !self.child_id.is_empty() {
            out.push_str(",\"child_id\":");
            out.push_str(&json::quote(&self.child_id));
        }
        out.push_str(",\"invocation\":");
        out.push_str(&json::quote(&self.invocation));
        out.push_str(",\"plan_digest\":");
        out.push_str(&json::quote(&self.plan_digest));
        if !self.detail.is_empty() {
            out.push_str(",\"detail\":");
            out.push_str(&json::quote(&self.detail));
        }
        out.push('}');
        out
    }

    /// Chain digest over the canonical encoding. Readers recompute it to
    /// verify the journal instead of trusting stored hashes.
    pub fn digest(&self) -> String {
        crate::sha256::hex_lower(&crate::sha256::digest(self.encode().as_bytes()))
    }

    /// Shape validation. Chain continuity against the journal head is
    /// enforced at append, not here.
    pub fn validate(&self) -> Result<(), String> {
        if self.sequence < 1 {
            return Err("task event sequence is invalid".to_string());
        }
        if !self.prev_digest.is_empty() && !domain::valid_container_id(&self.prev_digest) {
            return Err("task event chain digest is invalid".to_string());
        }
        if (self.sequence == 1) != self.prev_digest.is_empty() {
            return Err("task event chain origin is invalid".to_string());
        }
        if !valid_task_event_type(&self.event_type) {
            return Err("unknown task event type".to_string());
        }
        if !valid_task_key(&self.task_key) {
            return Err("task event task key is invalid".to_string());
        }
        if !self.child_id.is_empty() && !valid_native_task_child(&self.child_id) {
            return Err("task event child identity is invalid".to_string());
        }
        if self.invocation.is_empty()
            || self.invocation.len() > 128
            || self.invocation.bytes().any(|b| b < 0x20 || b == 0x7f)
        {
            return Err("task event invocation is invalid".to_string());
        }
        if !preparation::valid_digest(&self.plan_digest) {
            return Err("task event plan digest is invalid".to_string());
        }
        if self.detail.len() > MAX_TASK_EVENT_BYTES {
            return Err("task event detail exceeds its bound".to_string());
        }
        Ok(())
    }
}

fn valid_native_task_child(id: &str) -> bool {
    !id.is_empty()
        && id.len() <= 128
        && id
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'-' | b'_' | b'.' | b':'))
}

/// `project.FactoryTaskEvents`: page one admitted run's typed journal.
/// Read-only: polling never mutates checkpoints.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryTaskEvents {
    pub project: String,
    pub id: String,
    pub after_sequence: i64,
    pub byte_limit: i64,
}

typed_event_fields!(FactoryTaskEvents, "a task events request", {
    project => "project": String,
    id => "id": String,
    after_sequence => "after_sequence": json::SignedInteger,
    byte_limit => "byte_limit": json::SignedInteger,
});

impl FactoryTaskEvents {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        if body.is_empty() || body.len() > 4096 {
            return Err("invalid task events request".to_string());
        }
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        if self.after_sequence < 0 {
            return Err("invalid task events cursor".to_string());
        }
        if self.byte_limit < MIN_TASK_EVENTS_BYTES || self.byte_limit > MAX_TASK_EVENTS_BYTES {
            return Err("invalid task events read bound".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryTaskEventsState`: one paged journal response of whole
/// events only, with the next-sequence/terminal cursor.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryTaskEventsState {
    pub id: String,
    pub project: String,
    pub next_sequence: i64,
    pub terminal: bool,
    pub events: Vec<TaskEvent>,
}

impl FactoryTaskEventsState {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        if self.next_sequence < 1 {
            return Err("invalid task events cursor".to_string());
        }
        if self.events.len() > MAX_TASK_EVENTS_PER_READ {
            return Err("task events response exceeds its count bound".to_string());
        }
        for event in &self.events {
            event.validate()?;
        }
        Ok(())
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"next_sequence\":");
        out.push_str(&self.next_sequence.to_string());
        out.push_str(",\"terminal\":");
        out.push_str(if self.terminal { "true" } else { "false" });
        out.push_str(",\"events\":[");
        for (index, event) in self.events.iter().enumerate() {
            if index > 0 {
                out.push(',');
            }
            out.push_str(&event.encode());
        }
        out.push_str("]}");
        out
    }
}

/// Durable journal file: the strict wire decoder admits only top-level
/// objects, so the event array travels under one `events` key.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(in crate::factory) struct JournalFile {
    pub(in crate::factory) events: Vec<TaskEvent>,
}

impl<'de> Deserialize<'de> for JournalFile {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct FileVisitor;
        impl<'de> Visitor<'de> for FileVisitor {
            type Value = JournalFile;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a task journal file")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut events = None;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("events") {
                        events = Some(map.next_value()?);
                        continue;
                    }
                    return Err(de::Error::unknown_field(&key, &["events"]));
                }
                Ok(JournalFile {
                    events: events.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_map(FileVisitor)
    }
}

impl JournalFile {
    pub(in crate::factory) fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    pub(in crate::factory) fn encode(&self) -> String {
        let mut out = String::from("{\"events\":[");
        for (index, event) in self.events.iter().enumerate() {
            if index > 0 {
                out.push(',');
            }
            out.push_str(&event.encode());
        }
        out.push_str("]}");
        out
    }
}

/// Bounded per-run task journal. The single writer (the daemon under the
/// run lock) appends; readers page whole events with a cursor.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EventJournal {
    events: Vec<TaskEvent>,
    bytes: usize,
    terminal: bool,
}

impl EventJournal {
    /// Append one typed observation. The event must continue the chain
    /// exactly; an exact duplicate of the head is a no-op so a lost reply
    /// can be looked up instead of replayed. Anything else out of order
    /// refuses without advancing.
    pub fn append(&mut self, event: &TaskEvent) -> Result<(), String> {
        event.validate()?;
        let encoded = event.encode();
        if encoded.len() > MAX_TASK_EVENT_BYTES {
            return Err("task event exceeds its bound".to_string());
        }
        if let Some(head) = self.events.last() {
            if event.sequence == head.sequence {
                if event.digest() == head.digest() && event.encode() == head.encode() {
                    return Ok(());
                }
                return Err("task event replay conflicts".to_string());
            }
            if event.sequence != head.sequence + 1 {
                return Err("task event sequence gap refuses".to_string());
            }
            if event.prev_digest != head.digest() {
                return Err("task event chain refuses".to_string());
            }
        } else {
            if event.sequence != 1 || !event.prev_digest.is_empty() {
                return Err("task event chain origin is invalid".to_string());
            }
        }
        if self.terminal {
            return Err("task journal is terminal".to_string());
        }
        if !task_event_terminal_type(&event.event_type)
            && (self.events.len() + TASK_JOURNAL_RESERVED_TERMINAL >= MAX_TASK_JOURNAL_EVENTS
                || self.bytes + encoded.len() > MAX_TASK_JOURNAL_BYTES)
        {
            return Err("task journal exhausted".to_string());
        }
        if self.events.len() >= MAX_TASK_JOURNAL_EVENTS
            || self.bytes + encoded.len() > MAX_TASK_JOURNAL_BYTES
        {
            return Err("task journal exhausted".to_string());
        }
        if task_event_terminal_type(&event.event_type) {
            self.terminal = true;
        }
        self.bytes += encoded.len();
        self.events.push(event.clone());
        Ok(())
    }

    /// Page whole events after `after_sequence` within `byte_limit`. A
    /// response too small for its next event is a bounded size error
    /// without advancing. `terminal` reports a recorded `run_stopped`.
    pub fn query(
        &self,
        after_sequence: i64,
        byte_limit: usize,
    ) -> Result<(Vec<TaskEvent>, i64, bool), String> {
        if after_sequence < 0 {
            return Err("invalid task events cursor".to_string());
        }
        let start = (after_sequence as usize).min(self.events.len());
        let mut out = Vec::new();
        let mut used = 0usize;
        for event in self.events.iter().skip(start) {
            let encoded = event.encode();
            if out.len() >= MAX_TASK_EVENTS_PER_READ {
                break;
            }
            if used + encoded.len() > byte_limit {
                if out.is_empty() {
                    return Err("task events response too small".to_string());
                }
                break;
            }
            used += encoded.len();
            out.push(event.clone());
        }
        let next_sequence = after_sequence + out.len() as i64 + 1;
        Ok((out, next_sequence, self.terminal))
    }

    pub(in crate::factory) fn events(&self) -> &[TaskEvent] {
        &self.events
    }

    pub fn len(&self) -> usize {
        self.events.len()
    }

    pub fn is_empty(&self) -> bool {
        self.events.is_empty()
    }

    pub fn is_terminal(&self) -> bool {
        self.terminal
    }
}
