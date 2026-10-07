// Request admission for the Rust daemon mux.
//
// Retains request-shape validation, the shared mutation gate and terminal caps.
// The root-owned systemd socket authorizes HTTP operations. Muse packet callers
// have separate kernel peer/pidfd attestation in muse::socket.
#[path = "daemon/admission.rs"]
mod admission;

pub use self::admission::{
    body_limit_for, is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, AdmissionGuard,
    NativeRejection, RequestHead, TerminalGate, TerminalSlot, ADMITTED_MUTATION_PATHS,
    BODY_LIMIT_DEFAULT, BODY_LIMIT_IDENTITY, BODY_LIMIT_LARGE, IDENTITY_ACTIONS,
    NATIVE_CLEAN_PATHS, TAILNET_ACTIONS, TERMINAL_FRAME_LIMIT, TERMINAL_REQUEST_LIMIT,
    TERMINAL_STREAM_CAP,
};
