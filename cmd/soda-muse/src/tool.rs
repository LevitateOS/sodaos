//! Which provider CLI this launcher runs. One binary serves both: installed
//! as `muse` it launches Muse, linked as `codex` it launches Codex with the
//! account's connected Codex credential. Selected once from argv0.

use soda_provider_tools::Provider;
use std::sync::OnceLock;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum Tool {
    Muse,
    Codex,
}

static CURRENT: OnceLock<Tool> = OnceLock::new();

impl Tool {
    pub(crate) fn current() -> Tool {
        *CURRENT.get_or_init(|| {
            let argv0 = std::env::args_os().next().unwrap_or_default();
            let name = std::path::Path::new(&argv0)
                .file_name()
                .map(|n| n.to_string_lossy().into_owned())
                .unwrap_or_default();
            if name == "codex" {
                Tool::Codex
            } else {
                Tool::Muse
            }
        })
    }

    /// Provider named in the launch request; empty keeps the Muse wire.
    pub(crate) fn wire(self) -> &'static str {
        match self {
            Tool::Muse => "",
            Tool::Codex => "codex",
        }
    }

    pub(crate) fn name(self) -> &'static str {
        match self {
            Tool::Muse => "muse",
            Tool::Codex => "codex",
        }
    }

    pub(crate) fn store(self) -> Provider {
        match self {
            Tool::Muse => Provider::Muse,
            Tool::Codex => Provider::Codex,
        }
    }

    /// Staged executable the host installs for this tool.
    pub(crate) fn staged(self) -> &'static str {
        match self {
            Tool::Muse => "/usr/local/libexec/soda/muse",
            Tool::Codex => "/usr/local/libexec/soda/codex/bin/codex",
        }
    }

    pub(crate) fn receipt(self) -> &'static str {
        match self {
            Tool::Muse => "/usr/local/libexec/soda/muse-selection.json",
            Tool::Codex => "/usr/local/libexec/soda/codex-selection.json",
        }
    }

    /// Environment setting that picks one of several connected accounts.
    pub(crate) fn connection_setting(self) -> &'static str {
        match self {
            Tool::Muse => "SODA_MUSE_CONNECTION",
            Tool::Codex => "SODA_CODEX_CONNECTION",
        }
    }
}
