pub use super::muse::commands::{
    factory_muse_supervisor, muse_exec_argv, muse_setup_script, muse_start_gate_script,
};
// `muse_serve_oracle` compiles this module through a private `#[path]` copy
// that never touches `FactoryMusePaths`; it serves the real library.
#[allow(unused_imports)]
pub use super::muse::paths::{
    factory_muse_binding, factory_muse_guest, factory_muse_paths, factory_muse_run_paths,
    FactoryMusePaths,
};
