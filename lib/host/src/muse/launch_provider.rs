//! Which provider CLI one launch runtime serves. Muse and Codex share the
//! whole launch path (kernel-attested caller, broker lease, staged bytes,
//! supervised unit); only these names and the credential custody differ.
//! Muse credentials are immutable and read-only; a Codex credential rotates,
//! so its private `CODEX_HOME` is writable and the broker captures it back
//! through `finish` when the run ends.

use crate::terminal;

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub enum LaunchProvider {
    #[default]
    Muse,
    Codex,
}

impl LaunchProvider {
    /// Broker provider id.
    pub fn id(self) -> &'static str {
        match self {
            LaunchProvider::Muse => terminal::PROVIDER_MUSE,
            LaunchProvider::Codex => terminal::PROVIDER_CODEX,
        }
    }

    /// Binding scope that routes broker callbacks back to this runtime.
    pub fn scope(self) -> &'static str {
        match self {
            LaunchProvider::Muse => terminal::SCOPE_MUSE_PROJECT,
            LaunchProvider::Codex => terminal::SCOPE_CODEX_PROJECT,
        }
    }

    /// Provider of a recorded launch binding; `None` for other scopes.
    pub fn of_binding(binding: &terminal::Binding) -> Option<LaunchProvider> {
        LaunchProvider::from_scope(&binding.scope)
    }

    pub fn from_scope(scope: &str) -> Option<LaunchProvider> {
        match scope {
            terminal::SCOPE_MUSE_PROJECT => Some(LaunchProvider::Muse),
            terminal::SCOPE_CODEX_PROJECT => Some(LaunchProvider::Codex),
            _ => None,
        }
    }

    /// Provider-store family the executable comes from.
    pub fn store(self) -> soda_provider_tools::Provider {
        match self {
            LaunchProvider::Muse => soda_provider_tools::Provider::Muse,
            LaunchProvider::Codex => soda_provider_tools::Provider::Codex,
        }
    }

    /// Staged provider install inside the Project: Muse is one binary,
    /// Codex is its whole release bundle directory.
    pub fn guest_native(self) -> &'static str {
        match self {
            LaunchProvider::Muse => "/usr/local/libexec/soda/muse",
            LaunchProvider::Codex => "/usr/local/libexec/soda/codex",
        }
    }

    /// The staged executable the launcher execs.
    pub fn guest_executable(self) -> &'static str {
        match self {
            LaunchProvider::Muse => "/usr/local/libexec/soda/muse",
            LaunchProvider::Codex => "/usr/local/libexec/soda/codex/bin/codex",
        }
    }

    /// Selection receipt beside the staged executable.
    pub fn guest_receipt(self) -> &'static str {
        match self {
            LaunchProvider::Muse => "/usr/local/libexec/soda/muse-selection.json",
            LaunchProvider::Codex => "/usr/local/libexec/soda/codex-selection.json",
        }
    }

    /// The launcher command a person types (one binary, chosen by name).
    pub fn launcher(self) -> &'static str {
        match self {
            LaunchProvider::Muse => "/usr/local/bin/muse",
            LaunchProvider::Codex => "/usr/local/bin/codex",
        }
    }

    /// Environment setting that picks one of several ready connections.
    pub fn connection_setting(self) -> &'static str {
        match self {
            LaunchProvider::Muse => super::MUSE_CONNECTION_SETTING,
            LaunchProvider::Codex => "SODA_CODEX_CONNECTION",
        }
    }

    /// Private credential file inside one execution root. Codex reads and
    /// rotates it in its own writable `CODEX_HOME`.
    pub fn credential_file(self, root: &str) -> String {
        match self {
            LaunchProvider::Muse => format!("{root}/auth.json"),
            LaunchProvider::Codex => format!("{root}/codex/auth.json"),
        }
    }

    /// Size of the per-execution tmpfs root.
    pub fn root_tmpfs(self) -> &'static str {
        match self {
            LaunchProvider::Muse => "--options=mode=0700,size=4M",
            LaunchProvider::Codex => "--options=mode=0700,size=64M",
        }
    }

    /// Supervised guest unit for one execution.
    pub fn unit(self, id: &str) -> String {
        format!("soda-{}-{id}.service", self.id())
    }
}
