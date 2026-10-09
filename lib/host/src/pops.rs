//! JSON adapters for project operations in the privileged host daemon.
//!
//! `Ops` provides request decoding and response adaptation around typed
//! project, account, and preparation operations. `DaemonBackend` stores
//! `Ops<Native>` and invokes it for account, access-key, privilege-status,
//! and preparation routes. Operation validation and errors remain with their
//! typed runtimes and daemon route handlers.

use std::time::Instant;

use crate::domain;
use crate::preparation;
use crate::project;

/// `/access-keys` request: the exact `domain.AccessKeys` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeysReq(pub domain::AccessKeys);

impl AccessKeysReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(domain::AccessKeys::decode(body)?))
    }
}

/// `/account` request: the exact `domain.Account` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountReq(pub domain::Account);

impl AccountReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(domain::Account::decode(body)?))
    }
}

/// `/prepare` request: the exact `domain.Prepare` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareReq(pub preparation::Prepare);

impl PrepareReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::Prepare::decode(body)?))
    }
}

/// `/prepare-candidate` request: the exact `domain.FactoryCandidate` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareCandidateReq(pub preparation::FactoryCandidate);

impl PrepareCandidateReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::FactoryCandidate::decode(body)?))
    }
}

/// `/prepare-context` request: a bounded read bound to one recorded prep and
/// exact approved, diff-base, and candidate commits.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareContextReq(pub preparation::PrepareContextRead);

impl PrepareContextReq {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::PrepareContextRead::decode(body)?))
    }
}

/// `/prepare-inspect` request: the exact `domain.PrepareInspect` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InspectPreparationReq(pub preparation::PrepareInspect);

impl InspectPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::PrepareInspect::decode(body)?))
    }
}

/// `/prepare-stop` request: the exact `domain.PrepareStop` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StopPreparationReq(pub preparation::PrepareStop);

impl StopPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::PrepareStop::decode(body)?))
    }
}

/// `/prepare-hold` request: the exact `domain.PrepareHold` JSON shape.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HoldPreparationReq(pub preparation::PrepareHold);

impl HoldPreparationReq {
    /// Strict decode of one request body (unknown fields rejected).
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        Ok(Self(preparation::PrepareHold::decode(body)?))
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

    /// `/project-access`: observe native Project administrator status for
    /// the identity after binding its running container.
    pub fn project_access(
        &self,
        req: &domain::ProjectAccessRequest,
        deadline: Instant,
    ) -> Result<String, String> {
        let status = crate::account::observe_project_access(&self.runtime(), req, deadline)?;
        serde_json::to_string(&status)
            .map_err(|_| "native project privilege observation was not confirmed".to_string())
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

    /// `/prepare-context`: read bounded exact-object prompt material. The
    /// request's absolute deadline caps every native operation.
    pub fn prepare_context(
        &self,
        req: &PrepareContextReq,
        deadline: Instant,
    ) -> Result<String, String> {
        Ok(self
            .runtime()
            .read_preparation_context(&req.0, deadline)?
            .encode())
    }

    /// `/prepare-inspect`: authoritative observed state, never mutating.
    /// Returns the `PrepareState` JSON render.
    pub fn inspect_preparation(
        &self,
        req: &InspectPreparationReq,
        deadline: Instant,
    ) -> Result<String, crate::prepare::PreparationObservationError> {
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
