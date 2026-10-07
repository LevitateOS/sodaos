//! Identity broker client: Rust port of the `internal/identity/client`
//! calls the PR26 runtime adapter needs, over the private Unix HTTP socket.
//!
//! Request paths, JSON body shapes and response limits match the Go client
//! byte for byte (request bodies were captured from a live Go probe and are
//! pinned in `tests/iclient_oracle.rs`). Only the HTTP header framing is
//! minimal — `Host`/`Content-Type`/`Content-Length`/`Connection: close` —
//! instead of Go's `net/http` defaults; the broker accepts normal HTTP/1
//! framing, and sending Go's `Accept-Encoding: gzip` would obligate a gzip
//! decoder.
//!
//! Deadlines arrive as `Instant`s and cover Unix connect through the complete
//! response body and Hyper driver cleanup.
//! Error mapping mirrors `decodeError`: the broker's `denied`/`busy`/
//! `stale`/`reauth`/`missing` codes map to the Go typed-error strings,
//! except `stale` and `missing` carry an explicit `(stale)`/`(not found)`
//! marker — the Go messages (`identity generation changed`, `identity
//! execution missing`) lack the substrings the adapter matches on.
//!
//! The `/identity/finish` 384 KiB cap lives on the host-daemon callback
//! client (`internal/host/identity.go`), not on any broker-client call:
//! every method here uses the Go client's 512 KiB response limit.

use std::path::Path;
use std::time::Instant;

use crate::json::{self, SignedInteger};
use crate::muse::MuseConnection;
use crate::terminal::{parse_string_i64, AcquireRequest, Binding, Delivery, Lease};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

/// `decodeResponse` limit: `512<<10`, probed with one byte of slack.
pub const RESPONSE_LIMIT: usize = 512 * 1024;
/// `decodeError` reads the first 64 body bytes.
const ERROR_PREFIX_LIMIT: usize = 64;
/// Response head cap, matching the sibling broker callback client.
const HEAD_LIMIT: usize = 65536;

/// `identity.ErrBusy`: no `texec` constant exists (nothing else maps it).
const ERR_BUSY: &str = "subscription is in use";

/// Private Unix HTTP broker client (`identity/client.Client`).
pub struct BrokerClient {
    socket_path: String,
}

/// Broker execution identity (`identity.Execution`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Execution {
    pub binding: Option<Binding>,
    pub kind: String,
    pub execution_id: String,
    pub digest: String,
    pub state: String,
    pub lease_id: String,
}

/// `identity.ExecutionTerminal`: the factory treats a terminal execution as
/// terminally settled — custody converged without further broker calls
/// (`internal/host/project/factory.go`: a `Return` failure with a terminal
/// execution still reports the stop as returned).
pub fn execution_is_terminal(e: &Execution) -> bool {
    e.state == "terminal"
}

fn unavailable() -> String {
    "identity broker unavailable".to_string()
}

/// `identity.AcquireRequest` JSON, field order included.
fn encode_acquire(req: &AcquireRequest) -> Result<String, String> {
    let mut out = String::from("{");
    let mut first = true;
    let mut field = |out: &mut String, name: &str, value: &str| {
        if !first {
            out.push(',');
        }
        first = false;
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(value);
    };
    if req.repository_id != 0 {
        field(
            &mut out,
            "repository_id",
            &json::quote(&req.repository_id.to_string()),
        );
    }
    field(&mut out, "provider_id", &json::quote(&req.provider_id));
    field(&mut out, "execution_id", &json::quote(&req.execution_id));
    field(
        &mut out,
        "actor_id",
        &json::quote(&req.actor_id.to_string()),
    );
    field(&mut out, "connection_id", &json::quote(&req.connection_id));
    field(&mut out, "project_id", &json::quote(&req.project_id));
    field(&mut out, "kind", &json::quote(&req.kind));
    let deadline = soda_wire_time::format(req.deadline_secs, req.deadline_nanos)
        .ok_or_else(|| "invalid deadline timestamp".to_string())?;
    field(&mut out, "deadline", &json::quote(&deadline));
    if !req.role.is_empty() {
        field(&mut out, "role", &json::quote(&req.role));
    }
    out.push('}');
    Ok(out)
}

/// `identity.Request` envelope: the five always-present fields in struct
/// order, then at most one `omitempty` payload. `kind`/`execution_id`
/// carry the execution calls; `nested` carries `acquire`/`binding`.
fn encode_request(
    owner_id: i64,
    id: &str,
    project_id: &str,
    kind: &str,
    execution_id: &str,
    nested: Option<(&str, &str)>,
    credential: Option<&str>,
) -> String {
    let mut out = String::from("{\"provider_id\":\"\",\"owner_id\":\"");
    out.push_str(&owner_id.to_string());
    out.push_str("\",\"id\":");
    out.push_str(&json::quote(id));
    out.push_str(",\"label\":\"\",\"project_id\":");
    out.push_str(&json::quote(project_id));
    if !kind.is_empty() {
        out.push_str(",\"kind\":");
        out.push_str(&json::quote(kind));
    }
    if !execution_id.is_empty() {
        out.push_str(",\"execution_id\":");
        out.push_str(&json::quote(execution_id));
    }
    if let Some((name, value)) = nested {
        out.push(',');
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(value);
    }
    if let Some(data) = credential {
        out.push_str(",\"credential\":");
        out.push_str(&json::quote(data));
    }
    out.push_str("}\n");
    out
}

fn map_error(body: &[u8]) -> String {
    let prefix = &body[..body.len().min(ERROR_PREFIX_LIMIT)];
    let code = std::str::from_utf8(prefix)
        .unwrap_or("")
        .trim_matches(|c: char| c.is_ascii_whitespace());
    match code {
        "denied" => crate::terminal::ERR_DENIED.to_string(),
        "busy" => ERR_BUSY.to_string(),
        "stale" => format!("{} (stale)", crate::terminal::ERR_STALE),
        "reauth" => crate::terminal::ERR_UNCERTAIN.to_string(),
        "missing" => format!("{} (not found)", crate::terminal::ERR_NOT_FOUND),
        _ => "identity operation failed".to_string(),
    }
}

/// `decodeResponse` framing: the 512 KiB limit, then one JSON value.
fn check_response_limit(body: &[u8]) -> Result<(), String> {
    if body.len() > RESPONSE_LIMIT {
        return Err("identity response exceeds limit".to_string());
    }
    Ok(())
}

#[derive(Default)]
struct ConnectionWire {
    provider_id: String,
    id: String,
    owner_id: Option<String>,
    label: String,
    email: String,
    plan: String,
    generation: i64,
    state: String,
}

impl<'de> Deserialize<'de> for ConnectionWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ConnectionVisitor;
        impl<'de> Visitor<'de> for ConnectionVisitor {
            type Value = ConnectionWire;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an identity connection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ConnectionWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("provider_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.provider_id = v;
                        }
                    } else if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("owner_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.owner_id = Some(v);
                        }
                    } else if key.eq_ignore_ascii_case("label") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.label = v;
                        }
                    } else if key.eq_ignore_ascii_case("email") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.email = v;
                        }
                    } else if key.eq_ignore_ascii_case("plan") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.plan = v;
                        }
                    } else if key.eq_ignore_ascii_case("generation") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.generation = v.0;
                        }
                    } else if key.eq_ignore_ascii_case("state") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.state = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &[
                                "provider_id",
                                "id",
                                "owner_id",
                                "label",
                                "email",
                                "plan",
                                "generation",
                                "state",
                            ],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ConnectionVisitor)
    }
}

/// Strict `identity.Connection` decode, projected to the minimal
/// `MuseConnection` the daemon directory needs. Unknown fields and
/// malformed `,string`/integer fields fail like Go's `DisallowUnknownFields`
/// binding into the full struct.
fn decode_connections(body: &[u8]) -> Result<Vec<MuseConnection>, String> {
    check_response_limit(body)?;
    let items: Vec<ConnectionWire> = json::decode_tolerant_as(body).map_err(|e| e.0)?;
    let mut out = Vec::with_capacity(items.len());
    for item in items {
        if let Some(owner) = &item.owner_id {
            parse_string_i64(owner).ok_or_else(|| "invalid owner_id".to_string())?;
        }
        out.push(MuseConnection {
            id: item.id,
            provider_id: item.provider_id,
            state: item.state,
        });
    }
    Ok(out)
}

#[derive(Default)]
struct BindingWire {
    child_id: String,
    uid: i64,
    gid: i64,
    scope: String,
    credential_root: String,
    invocation_id: String,
    kind: String,
    id: String,
    project: String,
    login: String,
    generation: i64,
}

impl<'de> Deserialize<'de> for BindingWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct BindingVisitor;
        impl<'de> Visitor<'de> for BindingVisitor {
            type Value = BindingWire;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an identity binding object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = BindingWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("child_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.child_id = v;
                        }
                    } else if key.eq_ignore_ascii_case("uid") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.uid = v.0;
                        }
                    } else if key.eq_ignore_ascii_case("gid") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.gid = v.0;
                        }
                    } else if key.eq_ignore_ascii_case("scope") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.scope = v;
                        }
                    } else if key.eq_ignore_ascii_case("credential_root") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.credential_root = v;
                        }
                    } else if key.eq_ignore_ascii_case("invocation_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.invocation_id = v;
                        }
                    } else if key.eq_ignore_ascii_case("kind") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.kind = v;
                        }
                    } else if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                    } else if key.eq_ignore_ascii_case("login") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.login = v;
                        }
                    } else if key.eq_ignore_ascii_case("generation") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.generation = v.0;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &[
                                "child_id",
                                "uid",
                                "gid",
                                "scope",
                                "credential_root",
                                "invocation_id",
                                "kind",
                                "id",
                                "project",
                                "login",
                                "generation",
                            ],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(BindingVisitor)
    }
}

#[derive(Default)]
struct ExecutionWire {
    binding: Option<BindingWire>,
    kind: String,
    execution_id: String,
    digest: String,
    state: String,
    lease_id: String,
}

impl<'de> Deserialize<'de> for ExecutionWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ExecutionVisitor;
        impl<'de> Visitor<'de> for ExecutionVisitor {
            type Value = ExecutionWire;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an identity execution object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ExecutionWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("binding") {
                        if let Some(v) = map.next_value::<Option<BindingWire>>()? {
                            out.binding = Some(v);
                        }
                    } else if key.eq_ignore_ascii_case("kind") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.kind = v;
                        }
                    } else if key.eq_ignore_ascii_case("execution_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.execution_id = v;
                        }
                    } else if key.eq_ignore_ascii_case("digest") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.digest = v;
                        }
                    } else if key.eq_ignore_ascii_case("state") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.state = v;
                        }
                    } else if key.eq_ignore_ascii_case("lease_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.lease_id = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &[
                                "binding",
                                "kind",
                                "execution_id",
                                "digest",
                                "state",
                                "lease_id",
                            ],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ExecutionVisitor)
    }
}

fn decode_execution(body: &[u8]) -> Result<Execution, String> {
    check_response_limit(body)?;
    let wire: ExecutionWire = json::decode_tolerant_as(body).map_err(|e| e.0)?;
    let binding = wire.binding.map(|b| Binding {
        child_id: b.child_id,
        uid: b.uid,
        gid: b.gid,
        scope: b.scope,
        credential_root: b.credential_root,
        invocation_id: b.invocation_id,
        kind: b.kind,
        id: b.id,
        project: b.project,
        login: b.login,
        generation: b.generation,
    });
    Ok(Execution {
        binding,
        kind: wire.kind,
        execution_id: wire.execution_id,
        digest: wire.digest,
        state: wire.state,
        lease_id: wire.lease_id,
    })
}

impl BrokerClient {
    pub fn new(socket_path: &str) -> Self {
        BrokerClient {
            socket_path: socket_path.to_string(),
        }
    }

    fn call(&self, path: &str, body: &str, deadline: Instant) -> Result<Vec<u8>, String> {
        let timeout = deadline
            .checked_duration_since(Instant::now())
            .filter(|d| !d.is_zero())
            .ok_or_else(unavailable)?;
        let response = soda_unix_http::request(
            Path::new(&self.socket_path),
            "POST",
            path,
            "soda-identity",
            body.as_bytes(),
            timeout,
            soda_unix_http::Limits {
                header_bytes: HEAD_LIMIT,
                body_bytes: RESPONSE_LIMIT,
            },
        )
        .map_err(|error| match error {
            soda_unix_http::Error::BodyTooLarge => "identity response exceeds limit".to_string(),
            _ => unavailable(),
        })?;
        if response.status != 200 {
            let response = response.body;
            return Err(map_error(&response));
        }
        Ok(response.body)
    }

    /// `Client.Acquire`: `POST /acquire`.
    pub fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String> {
        let nested = encode_acquire(req)?;
        let body = encode_request(0, "", "", "", "", Some(("acquire", &nested)), None);
        let raw = self.call("/acquire", &body, deadline)?;
        check_response_limit(&raw)?;
        Lease::decode(&raw)
    }

    /// `Client.Register`: `POST /register`.
    pub fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let nested = binding.encode();
        let body = encode_request(0, lease_id, "", "", "", Some(("binding", &nested)), None);
        let raw = self.call("/register", &body, deadline)?;
        check_response_limit(&raw)?;
        Delivery::decode(&raw)
    }

    /// `Client.ReconcileLease`: `POST /reconcile-lease`. The body is
    /// unread like Go's nil `out`, beyond the bounded transport read.
    pub fn reconcile_lease(&self, lease_id: &str, deadline: Instant) -> Result<(), String> {
        let body = encode_request(0, lease_id, "", "", "", None, None);
        self.call("/reconcile-lease", &body, deadline)?;
        Ok(())
    }

    /// `Client.EndLease`: `POST /lease/end`.
    pub fn end_lease(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String> {
        let body = encode_request(actor, lease_id, "", "", "", None, None);
        self.call("/lease/end", &body, deadline)?;
        Ok(())
    }

    /// `Client.Available`: `POST /available`.
    pub fn available(
        &self,
        actor: i64,
        project: &str,
        deadline: Instant,
    ) -> Result<Vec<MuseConnection>, String> {
        let body = encode_request(actor, "", project, "", "", None, None);
        let raw = self.call("/available", &body, deadline)?;
        decode_connections(&raw)
    }

    /// `Client.Return`: `POST /return`. An empty credential is omitted,
    /// like Go's `omitempty` on the byte slice.
    pub fn return_lease(
        &self,
        lease_id: &str,
        binding: &Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let nested = binding.encode();
        let encoded = (!credential.is_empty()).then(|| crate::ssh::b64_encode(credential));
        let body = encode_request(
            0,
            lease_id,
            "",
            "",
            "",
            Some(("binding", &nested)),
            encoded.as_deref(),
        );
        self.call("/return", &body, deadline)?;
        Ok(())
    }

    /// `Client.GetExecution`: `POST /execution/get`.
    pub fn get_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<Execution, String> {
        let body = encode_request(0, "", "", kind, execution_id, None, None);
        let raw = self.call("/execution/get", &body, deadline)?;
        decode_execution(&raw)
    }

    /// `Client.CloseExecution`: `POST /execution/close`.
    pub fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = encode_request(0, "", "", kind, execution_id, None, None);
        self.call("/execution/close", &body, deadline)?;
        Ok(())
    }
}
