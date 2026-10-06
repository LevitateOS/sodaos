//! Temporary password-window public-key enrollment: the fixed sshd
//! configuration, bounded key parsing, private-address admission, the
//! transient systemd units, and peer provenance.

pub mod arm;
#[cfg(test)]
mod arm_tests;
pub mod keys;
mod native_config;
mod peer;
pub mod receive;
pub mod selinux;
pub mod serve;
pub mod session;
#[cfg(test)]
pub(crate) mod test_support;
#[cfg(test)]
pub(crate) mod tests;

pub use native_config::{
    enrollment_client_command, enrollment_config, enrollment_private_address,
    enrollment_public_key, enrollment_socket_unit_config, enrollment_start_args,
    enrollment_template_unit_config, read_bounded,
};
pub use peer::{enrollment_connection_unit, enrollment_peer_unit, enrollment_receiver_unit};
pub use receive::receive_enrollment;
pub use serve::serve_enrollment;
pub use session::arm_enrollment;

pub const ENROLLMENT_DIR: &str = "/run/soda-key-enrollment";
pub const ENROLLMENT_UNIT: &str = "soda-key-enrollment.service";
pub const ENROLLMENT_SOCKET_UNIT: &str = "soda-key-enrollment.socket";
pub const ENROLLMENT_TEMPLATE_UNIT: &str = "soda-key-enrollment@.service";
pub const ENROLLMENT_UNIT_DIRECTORY: &str = "/run/systemd/system";
pub const ENROLLMENT_PORT: &str = "22222";
pub const ENROLLMENT_SOCKET: &str = "/run/soda-key-enrollment/receive.sock";
pub const ENROLLMENT_CONFIG_PATH: &str = "/run/soda-key-enrollment/sshd_config";
pub const ENROLLMENT_KEY_LIMIT: usize = 16384;
