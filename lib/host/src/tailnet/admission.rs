//! Ephemeral applied authority. Fountain remains the sole desired-state owner.

use std::fs;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use serde::{Deserialize, Deserializer, Serialize};

use crate::tcontrol_wire::LifecycleCall;

pub const RUNTIME_ROOT: &str = "/run/soda-tailnet";
pub const OPERATOR_SOCKET: &str = "/run/soda/operator/operator.sock";

fn unavailable() -> String {
    "unavailable: Tailnet admission is closed".to_string()
}

fn nullable<'de, D: Deserializer<'de>, T: Deserialize<'de>>(d: D) -> Result<Option<T>, D::Error> {
    Option::deserialize(d)
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ManagerState {
    pub catalog_sha256: String,
    pub controller_id: String,
    #[serde(deserialize_with = "nullable")]
    pub controller_instance_id: Option<String>,
    pub controller_ready: bool,
    pub admission_open: bool,
    pub entry: Entry,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Entry {
    pub package_id: String,
    #[serde(deserialize_with = "nullable")]
    pub installation_id: Option<String>,
    pub generation: String,
    pub desired_enabled: bool,
    #[serde(deserialize_with = "nullable")]
    pub pending_operation: Option<String>,
    #[serde(deserialize_with = "nullable")]
    pub installed_version: Option<String>,
    #[serde(deserialize_with = "nullable")]
    pub package_sha256: Option<String>,
    pub required: bool,
}

pub fn manager_state(socket: &Path, deadline: Instant) -> Result<ManagerState, String> {
    let timeout = deadline
        .checked_duration_since(Instant::now())
        .ok_or_else(unavailable)?;
    let response = soda_unix_http::request(
        socket,
        "POST",
        "/operator/extensions/state",
        "soda-operator",
        br#"{"package_id":"tailnet"}"#,
        timeout.min(Duration::from_secs(5)),
        soda_unix_http::Limits {
            header_bytes: 8192,
            body_bytes: 128 << 10,
        },
    )
    .map_err(|_| unavailable())?;
    if response.status != 200 {
        return Err(unavailable());
    }
    crate::json::decode_strict_as(&response.body).map_err(|_| unavailable())
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ServiceEpoch {
    pub pid: u32,
    pub start: String,
    pub boot: String,
}

impl ServiceEpoch {
    pub fn capture(pid: u32) -> Result<Self, String> {
        let read = || {
            Box::new(|path: &str| fs::read(path).map_err(|_| unavailable()))
                as crate::tailnet_domain::ReadFile
        };
        Ok(Self {
            pid,
            start: crate::tailnet_runtime::parse_process_start_time(i64::from(pid), read())?,
            boot: crate::tailnet_runtime::read_system_boot_id(read())?,
        })
    }

    pub fn live(&self) -> bool {
        Self::capture(self.pid).is_ok_and(|current| current == *self)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Grant {
    pub installation_id: String,
    pub generation: String,
    pub controller_instance_id: String,
    pub package_version: String,
    pub package_sha256: String,
    pub catalog_sha256: String,
    pub image: String,
    pub service_epoch: ServiceEpoch,
}

impl ManagerState {
    fn matches(&self, grant: &Grant) -> bool {
        self.controller_id == "soda"
            && self.controller_ready
            && self.controller_instance_id.as_ref() == Some(&grant.controller_instance_id)
            && self.catalog_sha256 == grant.catalog_sha256
            && self.entry.package_id == "tailnet"
            && !self.entry.required
            && self.entry.installation_id.as_ref() == Some(&grant.installation_id)
            && self.entry.generation == grant.generation
            && self.entry.desired_enabled
            && self.entry.installed_version.as_ref() == Some(&grant.package_version)
            && self.entry.package_sha256.as_ref() == Some(&grant.package_sha256)
    }

    // This is used only inside the verified lifecycle-owned activation, never
    // as an ordinary helper option or token file. Pending activation is required.
    pub fn activation_grant(&self, call: &LifecycleCall, image: &str) -> Result<Grant, String> {
        let grant = Grant {
            installation_id: call.installation_id.clone(),
            generation: call.generation.clone(),
            controller_instance_id: call.controller_instance_id.clone(),
            package_version: call.package_version.clone(),
            package_sha256: call.package_sha256.clone(),
            catalog_sha256: self.catalog_sha256.clone(),
            image: image.to_string(),
            service_epoch: ServiceEpoch::capture(std::process::id())?,
        };
        if call.controller_id != "soda"
            || call.package_id != "tailnet"
            || call.operation != "activate"
            || !crate::tailnet_domain::valid_image_id(image)
            || self.catalog_sha256.len() != 64
            || !self
                .catalog_sha256
                .bytes()
                .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
            || !self.matches(&grant)
            || self.entry.pending_operation.as_deref() != Some("activate")
        {
            return Err(unavailable());
        }
        Ok(grant)
    }

    pub fn admits(&self, grant: &Grant, image: &str) -> bool {
        self.admission_open
            && self.entry.pending_operation.is_none()
            && self.matches(grant)
            && grant.image == image
            && grant.service_epoch.live()
    }
}

pub struct GrantStore {
    pub(crate) files: crate::tailnet_files::OwnedState,
}

impl Default for GrantStore {
    fn default() -> Self {
        Self {
            files: crate::tailnet_files::OwnedState {
                root: PathBuf::from(RUNTIME_ROOT),
                owner: 0,
            },
        }
    }
}

pub use crate::tailnet_files::OwnedLock as StartGuard;

impl GrantStore {
    pub fn read(&self) -> Result<Grant, String> {
        self.files.read("admission.json")
    }
    pub fn close(&self) -> Result<(), String> {
        self.files.remove("admission.json")
    }
    pub fn publish(&self, grant: &Grant) -> Result<(), String> {
        self.files.write("admission.json", grant)
    }
    pub fn admit(&self, image: &str, deadline: Instant) -> Result<Grant, String> {
        let grant = self.read()?;
        let state = manager_state(Path::new(OPERATOR_SOCKET), deadline)?;
        if !state.admits(&grant, image) {
            return Err(unavailable());
        }
        Ok(grant)
    }

    pub fn start_lock(&self, deadline: Instant) -> Result<StartGuard, String> {
        self.files.lock(".starts.lock", false, deadline)
    }
    pub fn drain_lock(&self, deadline: Instant) -> Result<StartGuard, String> {
        self.files.lock(".starts.lock", true, deadline)
    }
    pub fn lifecycle_lock(&self, deadline: Instant) -> Result<StartGuard, String> {
        self.files.lock(".lifecycle.lock", true, deadline)
    }
}

#[cfg(test)]
#[path = "admission/tests.rs"]
mod tests;
