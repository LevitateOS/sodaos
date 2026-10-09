use std::os::unix::net::UnixStream;
use std::sync::atomic::Ordering;
use std::sync::Arc;

use crate::daemon::backend::{BackendError, ExecBackend, TerminalSession};
use crate::{domain, factory, json, project, terminal};

use super::tailnet::{decode_tailnet_empty, map_tailnet_err};
use super::websocket;
use super::{
    decode_empty, internal, map_factory_err, native_deadline, tailnet_deadline, DaemonBackend,
};

#[cfg(test)]
mod tests;

fn map_preparation_observation_error(
    error: crate::prepare::PreparationObservationError,
) -> BackendError {
    match error {
        crate::prepare::PreparationObservationError::NotFound => BackendError::NotFound,
        crate::prepare::PreparationObservationError::Invalid(message) => internal(message),
    }
}

#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct FactoryHarnessRequest {
    harness: String,
}

// -- project routes (daemon.go dispatchProfile/Create/Targeted/Mutation) --

impl DaemonBackend {
    /// Shared `/inspect`, `/os`, `/connection` prelude: decode a creation
    /// identity but validate the ID only (Go checks `ValidID` alone here,
    /// with the "invalid project identity" message, not full `Validate`).
    fn targeted_id(&self, body: &[u8]) -> Result<String, BackendError> {
        let id = domain::Create::decode(body)
            .map(|c| c.id)
            .map_err(internal)?;
        if !domain::valid_id(&id) {
            return Err(internal("invalid project identity".to_string()));
        }
        Ok(id)
    }
}

impl ExecBackend for DaemonBackend {
    fn profile(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        decode_empty(body)?;
        let out = self
            .project
            .resolve_profile(native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn create(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // The mux holds the writer gate for /create; see module docs.
        let input = domain::Create::decode(body).map_err(internal)?;
        // Go checks identity before Validate, with its own message.
        if !domain::valid_id(&input.id) || input.owner <= 0 {
            return Err(internal("invalid project identity".to_string()));
        }
        input.validate().map_err(internal)?;
        let out = self
            .project
            .create(&input, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        // Go returns the environment and drops the generation counter.
        let (env, _) = self
            .project
            .inspect(&id, native_deadline())
            .map_err(internal)?;
        Ok(env.encode().into_bytes())
    }

    fn observe_os(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        let out = self
            .project
            .observe_os(&id, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn connection(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let id = self.targeted_id(body)?;
        let out = self
            .project
            .connection(&id, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn lifecycle(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let input = project::Lifecycle::decode(body).map_err(internal)?;
        let out = self
            .project
            .lifecycle(&input, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn access_keys(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::AccessKeysReq::decode(body).map_err(internal)?;
        self.pops
            .access_keys(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn account(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::AccountReq::decode(body).map_err(internal)?;
        self.pops
            .account(&req, native_deadline())
            .map_err(internal)?;
        // Go's `dispatchMutation` synthesizes this; there is no backend value.
        Ok(b"{\"ok\":true}".to_vec())
    }

    fn project_access(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = domain::ProjectAccessRequest::decode(body).map_err(internal)?;
        self.pops
            .project_access(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn prepare(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::PrepareReq::decode(body).map_err(internal)?;
        self.pops
            .prepare(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn prepare_candidate(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::PrepareCandidateReq::decode(body).map_err(internal)?;
        self.pops
            .prepare_candidate(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn prepare_context(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::PrepareContextReq::decode(body).map_err(internal)?;
        self.pops
            .prepare_context(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn inspect_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::InspectPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .inspect_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(map_preparation_observation_error)
    }

    fn stop_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::StopPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .stop_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn hold_preparation(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let req = crate::pops::HoldPreparationReq::decode(body).map_err(internal)?;
        self.pops
            .hold_preparation(&req, native_deadline())
            .map(String::into_bytes)
            .map_err(internal)
    }

    fn factory_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // Factory routes stay off the shared mutation gate (Go comment):
        // per-run locks plus broker serialization are the control.
        let factory = self.factory()?;
        let req = factory::FactoryLaunch::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .launch(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryInspect::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_stop(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryStop::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .stop(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_takeover(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryTakeover::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .takeover(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_output(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryOutput::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .output(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::OutputStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_harness(&self, body: &[u8], image: &str) -> Result<Vec<u8>, BackendError> {
        let req: FactoryHarnessRequest =
            json::decode_strict_as(body).map_err(|err| internal(err.to_string()))?;
        if !factory::valid_harness_family(&req.harness) {
            return Err(BackendError::Unavailable);
        }
        let factory = self.factory()?;
        // Go stamps the daemon image onto the pin before validation.
        let mut pin = factory
            .harness_pin(&req.harness)
            .map_err(|_| BackendError::Unavailable)?;
        pin.image = image.to_string();
        pin.validate().map_err(|_| BackendError::Unavailable)?;
        Ok(pin.encode().into_bytes())
    }

    fn factory_export(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryExport::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .export(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn factory_candidate_inspect(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let factory = self.factory()?;
        let req = factory::FactoryCandidateInspect::decode(body).map_err(internal)?;
        req.validate().map_err(internal)?;
        let out = factory
            .inspect_candidate(&req, native_deadline())
            .map_err(|e| map_factory_err(e, BackendError::ExportStale))?;
        Ok(out.encode().into_bytes())
    }

    fn identity_launch(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        // The mux maps every non-501 identity error to 409, like Go's
        // `identityHandler`.
        let input = terminal::TerminalStart::decode(body).map_err(internal)?;
        let lease = terminal::identity_launch(
            self,
            &self.terminal,
            &input,
            !self.codex_harness.is_empty(),
            native_deadline(),
        )
        .map_err(internal)?;
        Ok(lease.encode().into_bytes())
    }

    fn identity_action(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let delivery = terminal::Delivery::decode(body).map_err(internal)?;
        if delivery.lease.kind == terminal::KIND_FACTORY {
            // Broker validate/stop/finish callbacks for supervised runs.
            let out = self
                .terminal
                .factory_identity_operation(action, &delivery, native_deadline())
                .map_err(internal)?;
            return Ok(out.encode().into_bytes());
        }
        let muse_scoped = delivery.lease.provider_id == terminal::PROVIDER_MUSE
            && delivery
                .lease
                .binding
                .as_ref()
                .is_some_and(|b| b.scope == "muse-project");
        if muse_scoped {
            if let Some(m) = &self.muse {
                let out = m
                    .muse(action, &delivery, native_deadline())
                    .map_err(internal)?;
                return Ok(out.encode().into_bytes());
            }
            // Muse-scoped without a muse runtime falls through to the
            // terminal identity path, which denies it (Go parity).
        }
        let out = self
            .terminal
            .identity(action, &delivery, native_deadline())
            .map_err(internal)?;
        Ok(out.encode().into_bytes())
    }

    fn tailnet(&self, action: &str, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let deadline = tailnet_deadline();
        match action {
            "settings" => {
                decode_tailnet_empty(body)?;
                self.tailnet.settings(deadline).map_err(map_tailnet_err)
            }
            "options" => {
                decode_tailnet_empty(body)?;
                self.tailnet.options(deadline).map_err(map_tailnet_err)
            }
            "host" => self
                .tailnet
                .host_action(body, deadline)
                .map_err(map_tailnet_err),
            "enrollment" => self
                .tailnet
                .enrollment(body, deadline)
                .map_err(map_tailnet_err),
            "project" | "policy" => self.tailnet_project_or_policy(action, body, deadline),
            _ => Err(BackendError::Invalid),
        }
    }

    fn terminal_accept(&self, _key: &str) -> Result<TerminalSession, BackendError> {
        // Go asserts the executor provides the attach launcher and
        // registers the stream before the 101. The mux now owns stream
        // registration (TerminalSlot), and native attach is always
        // available in this backend, so admission always succeeds; the
        // session id correlates pump logging.
        Ok(TerminalSession {
            id: self.next_session.fetch_add(1, Ordering::SeqCst),
        })
    }

    fn pump_terminal(
        &self,
        stream: tungstenite::protocol::WebSocket<UnixStream>,
        _session: TerminalSession,
        shutdown: Arc<std::sync::atomic::AtomicBool>,
    ) -> Result<(), BackendError> {
        websocket::pump_terminal(&self.terminal, self, stream, shutdown).map_err(internal)
    }
}
