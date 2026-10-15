//! Bounded lifecycle composition. No durable preference is stored here.
use std::collections::BTreeSet;
use std::path::Path;
use std::time::Instant;

use sha2::{Digest, Sha256};

use crate::project::Executor;
use crate::tailnet_admission::{manager_state, GrantStore, OPERATOR_SOCKET};
use crate::tailnet_companion::{Companion, TailnetControl};
use crate::tailnet_host::HostRuntime;
use crate::tcontrol_wire::{self as wire, LifecycleCall, LifecycleObservation};

pub struct Runtime<'a, E, T> {
    pub host: &'a HostRuntime<E>,
    pub companion: &'a Companion<E, T>,
    pub payload: &'a str,
}

impl<E: Executor, T: TailnetControl> Runtime<'_, E, T> {
    pub fn observe(&self, call: &LifecycleCall, deadline: Instant) -> LifecycleObservation {
        let mut observation = super::observation_for(
            call,
            wire::LIFECYCLE_STOPPED,
            "Owned Tailnet resources are stopped.",
        );
        let host = self.host.inspect(Some(&call.installation_id), deadline);
        let mut unknown = host.is_err();
        let host = host.ok().flatten();
        let inventory =
            self.companion
                .native_children(&call.installation_id, host.as_ref(), deadline);
        unknown |= inventory.unknown;
        if unknown {
            observation.state = wire::LIFECYCLE_UNKNOWN.into();
            observation.detail =
                "Owned-resource inspection is incomplete; admission stays closed.".into();
            return observation;
        }
        let active_host = host.as_ref().filter(|p| p.running);
        let active_child = inventory.children.iter().find(|p| p.record.running);
        if active_host.is_some() || active_child.is_some() {
            observation.state = wire::LIFECYCLE_RUNNING.into();
            observation.effect_generation = active_host
                .map(|p| p.record.generation.clone())
                .or_else(|| active_child.map(|p| p.generation.clone()));
            observation.detail = "Exact-owned Tailnet resources are running.".into();
            observation.admission_ready = call.operation != wire::LIFECYCLE_DEACTIVATE
                && active_host.is_some_and(|host| host.record.image == self.companion.image)
                && inventory
                    .children
                    .iter()
                    .filter(|p| p.record.running)
                    .all(|p| p.image == self.companion.image)
                && GrantStore::default()
                    .read()
                    .is_ok_and(|grant| grant.matches_call(call, &self.companion.image));
        }
        observation
    }

    fn activation(&self, call: &LifecycleCall, deadline: Instant) -> Result<(), String> {
        // The current payload owns catalog projection and both package/image pins.
        // A changed catalog cannot be blessed merely because two package fields agree.
        let payload = soda_release_deliver::payload::load(self.payload)
            .map_err(|_| "trusted release unavailable".to_string())?;
        let package = payload
            .extension_catalog
            .packages
            .iter()
            .find(|p| p.id == "tailnet")
            .ok_or_else(|| "trusted package unavailable".to_string())?;
        let image = payload
            .optional_extensions
            .tailnet
            .images
            .tailscale
            .config
            .as_str();
        if package.version != call.package_version
            || package.sha256 != call.package_sha256
            || image != self.companion.image
        {
            return Err("trusted release identity changed".into());
        }
        let catalog = payload
            .projected_catalog_bytes()
            .map_err(|_| "trusted catalog unavailable".to_string())?;
        let catalog = format!("{:x}", Sha256::digest(catalog));
        let state = manager_state(Path::new(OPERATOR_SOCKET), deadline)?;
        let grant = state.activation_grant(call, image, &catalog)?;
        let store = GrantStore::default();
        let starts = store.start_lock(deadline)?;
        self.host
            .ensure(&grant, &starts, deadline)
            .map_err(|e| format!("host activation: {e}"))?;
        let mut units = vec!["start".into(), "soda-tailnet-host.service".into()];
        for id in self
            .running_projects(deadline)
            .map_err(|e| format!("project inventory: {e}"))?
        {
            let cid = self
                .companion
                .start_tailnet_admitted(&id, &grant, &starts, deadline)?;
            if !cid.is_empty() {
                units.push(format!("soda-tailnet@{id}.service"));
            }
        }
        // Recheck authority after all bounded native/provider work and before
        // publishing the applied grant. Ordinary admission still belongs to Fountain.
        let state = manager_state(Path::new(OPERATOR_SOCKET), deadline)?;
        let grant = state.activation_grant(call, image, &catalog)?;
        if Instant::now() >= deadline {
            return Err("activation deadline exceeded".into());
        }
        store.publish(&grant)?;
        drop(starts);
        // Supervision adopts the already-running exact CID. It never receives
        // a lifecycle token or a bypass option, and waits outside the starts lock.
        crate::tailnet_host::command(&self.host.exec, "/usr/bin/systemctl", &units, deadline)
            .map_err(|e| format!("unit supervision: {e}"))?;
        Ok(())
    }

    fn running_projects(&self, deadline: Instant) -> Result<BTreeSet<String>, String> {
        let data = crate::tailnet_host::podman(
            &self.host.exec,
            &[
                "ps".into(),
                "--filter".into(),
                "label=org.soda.project".into(),
                r#"--format={"project":{{json (.Label "org.soda.project")}},"owner":{{json (.Label "org.soda.owner")}}}"#.into(),
            ],
            deadline,
        )?;
        let text =
            std::str::from_utf8(&data).map_err(|_| "project inventory unavailable".to_string())?;
        managed_projects(text)
    }

    fn deactivate(&self, call: &LifecycleCall, deadline: Instant) -> Result<(), String> {
        let mut complete = true;
        let before = self.host.inspect(Some(&call.installation_id), deadline);
        complete &= before.is_ok();
        let host = before.ok().flatten();
        let inventory =
            self.companion
                .native_children(&call.installation_id, host.as_ref(), deadline);
        complete &= !inventory.unknown;
        let mut projects = match self.companion.known_projects() {
            Ok(ids) => ids,
            Err(_) => {
                complete = false;
                BTreeSet::new()
            }
        };
        projects.extend(
            inventory
                .children
                .iter()
                .map(|child| child.target.project.clone()),
        );
        if projects.len() > 128 {
            complete = false;
        }
        let mut units = vec![
            "stop".into(),
            "--no-block".into(),
            "soda-tailnet-host.service".into(),
        ];
        units.extend(
            projects
                .into_iter()
                .take(128)
                .map(|id| format!("soda-tailnet@{id}.service")),
        );
        complete &=
            crate::tailnet_host::command(&self.host.exec, "/usr/bin/systemctl", &units, deadline)
                .is_ok();
        let store = GrantStore::default();
        let _drain = store.drain_lock(deadline)?;
        // A start admitted before the close has now ended. Enumerate again
        // under exclusion; preserve every unknown candidate and stop the known subset.
        let host = self.host.inspect(Some(&call.installation_id), deadline);
        complete &= host.is_ok();
        let host = host.ok().flatten();
        let inventory =
            self.companion
                .native_children(&call.installation_id, host.as_ref(), deadline);
        complete &= !inventory.unknown;
        for child in inventory.children {
            complete &= self.companion.stop_native_child(&child, deadline).is_ok();
        }
        complete &= self
            .host
            .stop(Some(&call.installation_id), deadline)
            .is_ok();
        let after = self.observe(call, deadline);
        if !complete || after.state != wire::LIFECYCLE_STOPPED || after.admission_ready {
            return Err("Tailnet cleanup is unconfirmed".into());
        }
        Ok(())
    }

    pub fn apply(&self, call: &LifecycleCall, deadline: Instant) -> LifecycleObservation {
        if call.operation == wire::LIFECYCLE_OBSERVE {
            return self.observe(call, deadline);
        }
        let store = GrantStore::default();
        let result = (|| {
            let _maintenance = if call.operation == wire::LIFECYCLE_ACTIVATE {
                crate::project::admission::admit(None)?
            } else {
                crate::project::admission::cleanup_guard()?
            };
            let _lifecycle = store.lifecycle_lock(deadline)?;
            store.close()?; // Irreversible for this apply, before waiting on active starts.
            if call.operation == wire::LIFECYCLE_ACTIVATE {
                self.activation(call, deadline)
            } else {
                self.deactivate(call, deadline)
            }
        })();
        if let Err(error) = result {
            eprintln!("Tailnet lifecycle incomplete: {error}");
            let _ = store.close();
            return super::failed_observation(call);
        }
        self.observe(call, deadline)
    }
}

fn managed_projects(text: &str) -> Result<BTreeSet<String>, String> {
    #[derive(serde::Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Row {
        project: String,
        owner: String,
    }
    let mut projects = BTreeSet::new();
    for (index, line) in text.lines().enumerate() {
        if index >= 128 {
            return Err("project inventory exceeds its supported bound".into());
        }
        let row: Row =
            serde_json::from_str(line).map_err(|_| "project inventory unavailable".to_string())?;
        // This is the same ownership rule as native Project inspection.
        // A labelled development/foreign container is not a managed Project.
        if !crate::json::parse_go_int64(&row.owner).is_some_and(|owner| owner > 0) {
            continue;
        }
        if !crate::tailnet_domain::valid_project_id(&row.project) {
            return Err("managed Project identity is invalid".into());
        }
        projects.insert(row.project);
    }
    Ok(projects)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn inventory_uses_native_project_ownership_and_keeps_bounds() {
        let id = "p0123456789abcdef01234567";
        let rows = format!(
            r#"{{"project":"{id}","owner":"42"}}
{{"project":"p000000000000000000000004","owner":"tailnet-gate-task"}}"#
        );
        assert_eq!(
            managed_projects(&rows).unwrap(),
            BTreeSet::from([id.into()])
        );
        assert!(managed_projects(r#"{"project":"bad","owner":"42"}"#).is_err());
        assert!(managed_projects("invalid").is_err());
        assert!(managed_projects(&format!("{rows}\n").repeat(65)).is_err());
    }
}
