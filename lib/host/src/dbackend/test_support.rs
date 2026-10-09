use crate::dbackend::{BackendConfig, DaemonBackend, MuseConfig};
use crate::project;

pub(in crate::dbackend) fn test_config() -> BackendConfig {
    BackendConfig {
        project: project::Config {
            muse_socket: "/run/soda-test-muse.sock".to_string(),
            image: "sha256:0000000000000000000000000000000000000000000000000000000000000000"
                .to_string(),
            network: "soda-test".to_string(),
            subnet: "10.99.0.0/24".to_string(),
            bridge: "soda-test0".to_string(),
        },
        codex_harness: "/run/soda-test-harness".to_string(),
        codex_harness_sha256: "0000000000000000000000000000000000000000000000000000000000000000"
            .to_string(),
        codex_harness_version: "v0".to_string(),
        muse_harness: String::new(),
        muse_harness_sha256: String::new(),
        muse_harness_version: String::new(),
        broker_socket: "/run/soda-test-broker.sock".to_string(),
        tailnet_image: "ghcr.io/test/tailnet:latest".to_string(),
        muse: None,
    }
}

/// Backend whose factory root can never open: `create_dir_all` on an
/// existing file always fails, so factory routes deterministically
/// report `Unavailable` without touching the filesystem. (This source
/// file is guaranteed to exist wherever the test builds.)
pub(in crate::dbackend) fn backend() -> DaemonBackend {
    DaemonBackend::open(test_config(), file!())
}

pub(in crate::dbackend) fn backend_with_muse() -> DaemonBackend {
    let mut cfg = test_config();
    cfg.muse = Some(MuseConfig {
        version: "v0".to_string(),
        sha256: "0000000000000000000000000000000000000000000000000000000000000000".to_string(),
    });
    DaemonBackend::open(cfg, file!())
}
