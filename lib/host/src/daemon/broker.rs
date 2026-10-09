//! Identity-broker adapters for factory orchestration and Muse runtime hooks.

use crate::{factory, iclient::BrokerClient, muse, terminal};
use std::time::Instant;

pub(crate) fn cv_lease_to_texec(l: &factory::Lease) -> terminal::Lease {
    terminal::Lease {
        repository_id: l.repository_id,
        provider_id: l.provider_id.clone(),
        id: l.id.clone(),
        connection_id: l.connection_id.clone(),
        generation: l.generation,
        actor_id: l.actor_id,
        project_id: l.project_id.clone(),
        execution_id: l.execution_id.clone(),
        kind: l.kind.clone(),
        role: l.role.clone(),
        deadline_raw: l.deadline.clone(),
        deadline: terminal::parse_rfc3339(&l.deadline),
        grant_id: l.grant_id.clone(),
        grant_revision: l.grant_revision,
        binding: l.binding.clone(),
    }
}

fn cv_acquire_to_texec(r: &factory::AcquireRequest) -> terminal::AcquireRequest {
    let (secs, nanos) = terminal::parse_rfc3339(&r.deadline).unwrap_or((0, 0));
    terminal::AcquireRequest {
        repository_id: 0,
        provider_id: r.provider_id.clone(),
        execution_id: r.execution_id.clone(),
        actor_id: r.actor_id,
        connection_id: r.connection_id.clone(),
        project_id: r.project_id.clone(),
        kind: r.kind.clone(),
        deadline_secs: secs,
        deadline_nanos: nanos,
        role: r.role.clone(),
    }
}

fn cv_lease_to_factory(l: &terminal::Lease) -> factory::Lease {
    factory::Lease {
        repository_id: l.repository_id,
        provider_id: l.provider_id.clone(),
        id: l.id.clone(),
        connection_id: l.connection_id.clone(),
        generation: l.generation,
        actor_id: l.actor_id,
        project_id: l.project_id.clone(),
        execution_id: l.execution_id.clone(),
        kind: l.kind.clone(),
        role: l.role.clone(),
        deadline: l.deadline_raw.clone(),
        grant_id: l.grant_id.clone(),
        grant_revision: l.grant_revision,
        binding: l.binding.clone(),
    }
}

impl factory::FactoryBroker for BrokerClient {
    fn acquire(
        &self,
        req: &factory::AcquireRequest,
        deadline: Instant,
    ) -> Result<factory::Lease, factory::FactoryError> {
        let terminal_request = cv_acquire_to_texec(req);
        BrokerClient::acquire(self, &terminal_request, deadline)
            .map(|lease| cv_lease_to_factory(&lease))
            .map_err(factory::FactoryError::Msg)
    }

    fn register(
        &self,
        lease_id: &str,
        binding: &terminal::Binding,
        deadline: Instant,
    ) -> Result<Vec<u8>, factory::FactoryError> {
        BrokerClient::register(self, lease_id, binding, deadline)
            .map(|delivery| delivery.credential.unwrap_or_default())
            .map_err(factory::FactoryError::Msg)
    }

    fn return_lease(
        &self,
        lease_id: &str,
        binding: &terminal::Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        BrokerClient::return_lease(self, lease_id, binding, credential, deadline)
            .map_err(factory::FactoryError::Msg)
    }

    fn reconcile_lease(&self, _lease_id: &str, _deadline: Instant) {}

    fn execution_is_terminal(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<bool, factory::FactoryError> {
        BrokerClient::get_execution(self, kind, execution_id, deadline)
            .map(|execution| crate::iclient::execution_is_terminal(&execution))
            .map_err(factory::FactoryError::Msg)
    }

    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), factory::FactoryError> {
        BrokerClient::close_execution(self, kind, execution_id, deadline)
            .map_err(factory::FactoryError::Msg)
    }
}

impl muse::MuseHooks for BrokerClient {
    fn acquire(
        &self,
        req: &terminal::AcquireRequest,
        deadline: Instant,
    ) -> Result<terminal::Lease, String> {
        BrokerClient::acquire(self, req, deadline)
    }

    fn attach(
        &self,
        lease_id: &str,
        binding: &terminal::Binding,
        deadline: Instant,
    ) -> Result<terminal::Delivery, String> {
        BrokerClient::register(self, lease_id, binding, deadline)
    }

    fn end(&self, actor: i64, lease_id: &str, deadline: Instant) -> Result<(), String> {
        BrokerClient::end_lease(self, actor, lease_id, deadline)
    }

    fn authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String> {
        // A Muse+Ready connection must exist before launch.
        let available = BrokerClient::available(self, actor, project, deadline)?;
        muse::muse_connection_authorized(&available)
    }

    fn nested_authorize(&self, actor: i64, project: &str, deadline: Instant) -> Result<(), String> {
        muse::MuseHooks::authorize(self, actor, project, deadline)
    }

    fn select(
        &self,
        actor: i64,
        project: &str,
        selected: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let available = BrokerClient::available(self, actor, project, deadline)?;
        muse::select_muse_connection(&available, selected)
    }
}

#[cfg(test)]
mod tests {
    use super::{cv_lease_to_factory, cv_lease_to_texec};
    use crate::factory;

    #[test]
    fn lease_binding_round_trip() {
        let lease = factory::Lease {
            repository_id: 7,
            provider_id: "codex".to_string(),
            id: "l".to_string(),
            connection_id: "c".to_string(),
            generation: 3,
            actor_id: 1001,
            project_id: "p".to_string(),
            execution_id: "e".to_string(),
            kind: "factory".to_string(),
            role: "coder".to_string(),
            deadline: "2026-10-05T00:00:00Z".to_string(),
            grant_id: "g".to_string(),
            grant_revision: 2,
            binding: Some(crate::terminal::Binding {
                child_id: "child".to_string(),
                uid: 1001,
                gid: 1001,
                scope: "project".to_string(),
                credential_root: "/run/cred".to_string(),
                invocation_id: "i".to_string(),
                kind: "factory".to_string(),
                id: "b".to_string(),
                project: "p".to_string(),
                login: "coder".to_string(),
                generation: 1,
            }),
        };
        let back = cv_lease_to_factory(&cv_lease_to_texec(&lease));
        assert_eq!(back, lease);
    }
}
