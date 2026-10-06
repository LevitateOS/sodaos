//! Project executor operations: the JSON adapter over the typed `Runtime` ops.
//!
//! This module serves the seven project-executor routes the Go daemon exposes
//! in `dispatchMutation`/`dispatchPrepare` (`/access-keys`, `/account`,
//! `/prepare`, `/prepare-candidate`, `/prepare-inspect`, `/prepare-stop`,
//! `/prepare-hold`). The typed logic already lives in [`project`],
//! [`account`](crate::account), [`prepare`](crate::prepare) and the
//! [`domain`]/[`preparation`] DTOs; this layer only binds the wire contract
//! the daemon adapter programs against: strict-decode request types plus
//! `Result<String, String>` JSON responses with the exact Go field
//! shapes. It ports no executor logic itself, so behavior can never drift
//! from the typed ops.
//!
//! Wiring: the integrator adds `pub mod pops;` to `lib.rs`. Until then this
//! file is compiled through `tests/pops_oracle.rs` via `#[path]` (the same
//! arrangement as the gmux modules), so `crate::` imports must keep
//! resolving in both contexts: the test root re-exports the used crate
//! modules with `use soda_host::{...};`.
//!
//! # Constructor
//!
//! `Runtime` has no `new`: callers build the struct literal
//! `Runtime { exec, config }` (see `project.rs`). `Ops` mirrors that
//! exactly:
//!
//! ```ignore
//! let ops = pops::Ops { exec, config };
//! ```
//!
//! where `exec: E` is any [`project::Executor`] and `config` is the shared
//! [`project::Config`]. Each method borrows the executor through a
//! short-lived `Runtime<&E>`; `&E` implements `Executor` via the blanket
//! impl in `project.rs`.
//!
//! # Responses
//!
//! Success bodies are the bare `encoding/json` struct renders (no trailing
//! newline; the mux appends `\n` like `json.Encoder`). `account` returns
//! `()` on success and the adapter synthesizes `{"ok":true}`, exactly like
//! the Go `/account` route.
//!
//! # Errors
//!
//! Every op error is an exact Go message string, wire-opaque: the adapter
//! maps by substring. `Req::decode` failures carry the strictjson shape
//! (`request exceeds 1 MiB`, `request must contain valid UTF-8`,
//! `decode request: ...`, including `json: unknown field %q`,
//! `json: cannot unmarshal ...` and `illegal base64 data at input byte N`).
//! Executor (podman/systemctl/helper) failures pass through raw unless the
//! op maps them, exactly like Go. Per-op sets:
//!
//! * `access_keys`: `invalid own-account key operation`; key-set faults
//!   (`too many development keys`, `invalid development key`,
//!   `noncanonical or duplicate development key`,
//!   `development key set too large`); container binding (`invalid project`,
//!   `terminal inspection unavailable`, `invalid terminal inspection`,
//!   `terminal target not ready or isolated`);
//!   `native keys changed or are not managed canonical keys`;
//!   `native key operation not confirmed`;
//!   `invalid native key observation`;
//!   `native key result differs from requested set`.
//! * `account`: `invalid project account`; inspection faults
//!   (`invalid project id`, `invalid native inspection`,
//!   `container is not owned by this project`,
//!   `invalid native project owner`, `native creation profile mismatch`,
//!   `project IP outside configured network`, `invalid IP address ...`);
//!   `project is stopped`; `invalid public key`; raw helper-exec failure;
//!   `native account identity was not confirmed`.
//! * `prepare`: `Prepare` validation (`invalid preparation identity`,
//!   `invalid requirement acceptance reference`,
//!   `invalid privileged-effect approval reference`,
//!   `invalid preparation source identity`, `too many required tools`,
//!   `invalid required tool name`, `invalid credential reference`,
//!   `invalid approved file set`, `approved setup entrypoint is required`,
//!   `approved check entrypoint is required`, `invalid approved file name`,
//!   `invalid approved file size`, `approved inputs exceed the bounded size`,
//!   `invalid source bundle size`,
//!   `approved inputs do not match their digest`); container binding
//!   (`invalid project`, `preparation target unavailable`,
//!   `invalid preparation target`,
//!   `preparation target not ready or isolated`); raw helper failure or
//!   `factory helper response exceeds the bounded size`; approval faults
//!   (`preparation approval unconfirmed`,
//!   `preparation approval resolved unexpected paths`); observation faults
//!   (`invalid preparation observation`,
//!   `preparation observation exceeds the bounded size`,
//!   `preparation observation is inconsistent`); clone faults
//!   (`preparation checkout is not inspectable`,
//!   `preparation checkout holds unknown partial effects; use a new identity`,
//!   `preparation source clone unconfirmed`, `private role home unconfirmed`,
//!   `preparation source identity unconfirmed`,
//!   `preparation checkout is not the approved commit`).
//! * `prepare_candidate`: `FactoryCandidate` validation (the `Preparation`
//!   set above plus `only review receives a fresh candidate preparation`,
//!   `candidate preparation requires a fresh identity`,
//!   `candidate source bundle exceeds preparation bounds`); container
//!   binding, helper and observation faults as in `prepare`;
//!   `candidate source preparation is not ready for this role and setup`;
//!   snapshot faults (`candidate approved snapshot is not protected`,
//!   `candidate approved snapshot differs from its recorded inputs`,
//!   `candidate approved snapshot is unavailable`,
//!   `candidate approved snapshot has an invalid file set`,
//!   `candidate approved snapshot differs from its digest`,
//!   `candidate approved file is not protected`,
//!   `candidate approved file is unavailable or exceeds bounds`); then the
//!   full `prepare` set for the delegated fresh preparation.
//! * `inspect_preparation`: `invalid preparation address`, then container
//!   binding, helper and observation faults as in `prepare`.
//! * `stop_preparation`: `invalid preparation address`,
//!   `preparation stop unconfirmed`, then container binding, helper and
//!   observation faults as in `prepare`.
//! * `hold_preparation`: `invalid maintenance hold`,
//!   `maintenance hold unconfirmed`,
//!   `maintenance hold outcome not confirmed`, then container binding and
//!   helper faults as in `prepare`.

use std::time::Instant;

use crate::domain;
use crate::json;
use crate::preparation;
use crate::project;

/// `/access-keys` request: the exact `domain.AccessKeys` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeysReq(pub domain::AccessKeys);

impl AccessKeysReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(domain::AccessKeys::from_value(&v)?))
    }
}

/// `/account` request: the exact `domain.Account` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountReq(pub domain::Account);

impl AccountReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(domain::Account::from_value(&v)?))
    }
}

/// `/prepare` request: the exact `domain.Prepare` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareReq(pub preparation::Prepare);

impl PrepareReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(preparation::Prepare::from_value(&v)?))
    }
}

/// `/prepare-candidate` request: the exact `domain.FactoryCandidate` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareCandidateReq(pub preparation::FactoryCandidate);

impl PrepareCandidateReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(preparation::FactoryCandidate::from_value(&v)?))
    }
}

/// `/prepare-inspect` request: the exact `domain.PrepareInspect` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InspectPreparationReq(pub preparation::PrepareInspect);

impl InspectPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(preparation::PrepareInspect::from_value(&v)?))
    }
}

/// `/prepare-stop` request: the exact `domain.PrepareStop` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StopPreparationReq(pub preparation::PrepareStop);

impl StopPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(preparation::PrepareStop::from_value(&v)?))
    }
}

/// `/prepare-hold` request: the exact `domain.PrepareHold` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HoldPreparationReq(pub preparation::PrepareHold);

impl HoldPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Ok(Self(preparation::PrepareHold::from_value(&v)?))
    }
}

/// Project executor operations over any [`project::Executor`].
///
/// Constructed as a struct literal mirroring `Runtime` (see the module
/// docs): `Ops { exec, config }`.
pub struct Ops<E> {
    pub exec: E,
    pub config: project::Config,
}

impl<E: project::Executor> Ops<E> {
    fn runtime(&self) -> project::Runtime<&E> {
        project::Runtime {
            exec: &self.exec,
            config: self.config.clone(),
        }
    }

    /// `/access-keys`: observe or revision-checked replace of a login's
    /// authorized keys. Returns the `AccessKeyState` JSON render.
    pub fn access_keys(&self, req: &AccessKeysReq, deadline: Instant) -> Result<String, String> {
        Ok(self.runtime().access_keys(&req.0, deadline)?.encode())
    }

    /// `/account`: provision a project login via the in-container helper.
    /// The adapter synthesizes `{"ok":true}` on success.
    pub fn account(&self, req: &AccountReq, deadline: Instant) -> Result<(), String> {
        self.runtime().account(&req.0, deadline)
    }

    /// `/prepare`: approve, clone, verify, resolve tools, record and start
    /// one preparation. Returns the `PrepareState` JSON render.
    pub fn prepare(&self, req: &PrepareReq, deadline: Instant) -> Result<String, String> {
        Ok(self.runtime().prepare(&req.0, deadline)?.encode())
    }

    /// `/prepare-candidate`: fresh reviewer preparation reusing a ready
    /// preparation's protected approved setup. Returns the `PrepareState`
    /// JSON render.
    pub fn prepare_candidate(
        &self,
        req: &PrepareCandidateReq,
        deadline: Instant,
    ) -> Result<String, String> {
        Ok(self.runtime().prepare_candidate(&req.0, deadline)?.encode())
    }

    /// `/prepare-inspect`: authoritative observed state, never mutating.
    /// Returns the `PrepareState` JSON render.
    pub fn inspect_preparation(
        &self,
        req: &InspectPreparationReq,
        deadline: Instant,
    ) -> Result<String, String> {
        Ok(self
            .runtime()
            .inspect_preparation(&req.0, deadline)?
            .encode())
    }

    /// `/prepare-stop`: persist the stop tombstone and report the state.
    /// Returns the `PrepareState` JSON render.
    pub fn stop_preparation(
        &self,
        req: &StopPreparationReq,
        deadline: Instant,
    ) -> Result<String, String> {
        Ok(self.runtime().stop_preparation(&req.0, deadline)?.encode())
    }

    /// `/prepare-hold`: enforce the maintenance hold marker natively.
    /// Returns the `HoldState` JSON render.
    pub fn hold_preparation(
        &self,
        req: &HoldPreparationReq,
        deadline: Instant,
    ) -> Result<String, String> {
        Ok(self.runtime().hold_preparation(&req.0, deadline)?.encode())
    }
}
