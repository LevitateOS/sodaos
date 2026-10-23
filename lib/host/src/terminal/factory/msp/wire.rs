use serde::Deserialize;
use serde_json::Value;

use super::Error;

const MAX_TEXT: usize = super::super::run::MAX_REASSESS_PROMPT;

/// Native provenance is opaque. Preserve it for exact replay, never infer
/// record contents from its spelling or treat a model-supplied token as one.
#[derive(Clone, Debug)]
pub struct Provenance {
    pub cursor: String,
    pub source: Value,
}

/// A full native item revision. `args`, `objective`, `result` and `text` are
/// untrusted content carried by the native record, not successful task evidence.
#[derive(Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Item {
    pub item_id: String,
    pub kind: String,
    pub revision: u64,
    pub status: String,
    pub turn_id: Option<String>,
    pub subagent_id: Option<String>,
    pub child_session_id: Option<String>,
    pub objective: Option<String>,
    pub role: Option<String>,
    pub control_status: Option<String>,
    pub tool: Option<String>,
    pub args: Option<String>,
    pub result: Option<Value>,
    pub text: Option<String>,
}

impl Item {
    fn validate(&self) -> Result<(), Error> {
        if !short(&self.item_id) || !short(&self.kind) || !short(&self.status) || self.revision == 0
        {
            return Err(Error::Invalid("native item identity"));
        }
        for text in [&self.args, &self.objective, &self.text]
            .into_iter()
            .flatten()
        {
            if text.len() > MAX_TEXT {
                return Err(Error::Bounds);
            }
        }
        if self.kind == "subagent"
            && (!self.subagent_id.as_deref().is_some_and(short)
                || !self.child_session_id.as_deref().is_some_and(short))
        {
            return Err(Error::Invalid("native child identity"));
        }
        Ok(())
    }
}

/// Safe failure metadata. Raw provider diagnostic text is deliberately omitted
/// from Debug and transport errors; it may contain account-specific context.
#[derive(Clone, Debug)]
pub struct TurnFailure {
    pub kind: String,
    pub retryable: bool,
    pub quota_exceeded: bool,
}

#[derive(Clone)]
pub enum Notice {
    /// A frame from the separate native Codex app-server schema. Its item and
    /// child vocabulary must be interpreted by the Codex owner, never Muse.
    Codex {
        method: String,
        params: Value,
    },
    /// A native server request. Its arguments remain untrusted until the
    /// admitted run owner validates the exact tool and parent/turn binding.
    CodexRequest {
        id: Value,
        method: String,
        params: Value,
    },
    Item {
        session: String,
        item: Item,
        provenance: Option<Provenance>,
    },
    TurnStarted {
        session: String,
        turn: String,
        provenance: Provenance,
    },
    TurnCompleted {
        session: String,
        turn: String,
        terminal: String,
        failure: Option<TurnFailure>,
        provenance: Provenance,
    },
    Other {
        method: String,
        session: Option<String>,
    },
    End,
}

pub(super) fn short(value: &str) -> bool {
    !value.is_empty() && value.len() <= 1024 && !value.contains('\0')
}

fn text(value: &Value, key: &str) -> Result<String, Error> {
    value
        .get(key)
        .and_then(Value::as_str)
        .filter(|s| short(s))
        .map(str::to_owned)
        .ok_or(Error::Invalid("native event identity"))
}

fn provenance(params: &Value, required: bool) -> Result<Option<Provenance>, Error> {
    let Some(source) = params.get("sourceRange") else {
        return if required {
            Err(Error::Invalid("native durable provenance"))
        } else {
            Ok(None)
        };
    };
    if !source.is_object()
        || !["first", "last", "stream"]
            .iter()
            .all(|key| source.get(key).is_some())
    {
        return Err(Error::Invalid("native durable provenance"));
    }
    Ok(Some(Provenance {
        cursor: text(params, "viewCursor")?,
        source: source.clone(),
    }))
}

fn failure(params: &Value) -> Result<Option<TurnFailure>, Error> {
    let Some(value) = params.get("error") else {
        return Ok(None);
    };
    let kind = text(value, "kind")?;
    let retryable = value
        .get("retryable")
        .and_then(Value::as_bool)
        .ok_or(Error::Invalid("native terminal failure"))?;
    let message = value
        .get("message")
        .and_then(Value::as_str)
        .unwrap_or_default();
    if message.len() > MAX_TEXT {
        return Err(Error::Bounds);
    }
    let lower = message.to_ascii_lowercase();
    Ok(Some(TurnFailure {
        kind,
        retryable,
        quota_exceeded: lower.contains("429") && lower.contains("quota"),
    }))
}

pub(super) fn notice(method: &str, params: Value) -> Result<Notice, Error> {
    if method == "view/gap" {
        return Err(Error::Gap);
    }
    match method {
        "item/started" | "item/updated" | "item/completed" => {
            let item: Item = serde_json::from_value(
                params
                    .get("item")
                    .cloned()
                    .ok_or(Error::Invalid("native item"))?,
            )
            .map_err(|_| Error::Invalid("native item"))?;
            item.validate()?;
            Ok(Notice::Item {
                session: text(&params, "sessionId")?,
                item,
                provenance: provenance(&params, method != "item/started")?,
            })
        }
        "turn/started" => Ok(Notice::TurnStarted {
            session: text(&params, "sessionId")?,
            turn: text(&params, "turnId")?,
            provenance: provenance(&params, true)?
                .ok_or(Error::Invalid("native turn provenance"))?,
        }),
        "turn/completed" => Ok(Notice::TurnCompleted {
            session: text(&params, "sessionId")?,
            turn: text(&params, "turnId")?,
            terminal: text(&params, "terminal")?,
            failure: failure(&params)?,
            provenance: provenance(&params, true)?
                .ok_or(Error::Invalid("native turn provenance"))?,
        }),
        _ if short(method) => Ok(Notice::Other {
            method: method.to_owned(),
            session: params
                .get("sessionId")
                .and_then(Value::as_str)
                .filter(|s| short(s))
                .map(str::to_owned),
        }),
        _ => Err(Error::Invalid("native notification")),
    }
}
