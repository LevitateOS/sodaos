// Admission gates for the Rust daemon mux (PR26).
//
// Mirrors the admission logic in internal/host/daemon.go plus the subsystem
// validators in identity.go, tailnet.go and terminal/service.go:
// request-shape validation (method, clean path, no query, no Origin),
// the shared mutation gate (`acquireAdmission`), the terminal stream cap,
// and kernel-attested peer credentials (`SO_PEERCRED`, and `SO_PEERPIDFD`
// for the muse-launch path) following terminal/muse_socket_linux.go.
//
// The main HTTP socket itself relies on filesystem authorization (the root
// systemd socket): peer credentials are attested and available to the
// backend, but the mux denies nothing on UID/GID there, exactly like Go.
#[path = "daemon/admission.rs"]
mod admission;

#[path = "daemon/peer.rs"]
mod peer;

pub use self::admission::{
    body_limit_for, is_admitted_mutation_path, valid_identity_request, valid_terminal_request,
    validate_native_request, validate_tailnet_request, AdmissionGate, AdmissionGuard,
    NativeRejection, RequestHead, TerminalGate, TerminalSlot, ADMITTED_MUTATION_PATHS,
    BODY_LIMIT_DEFAULT, BODY_LIMIT_IDENTITY, BODY_LIMIT_LARGE, IDENTITY_ACTIONS,
    NATIVE_CLEAN_PATHS, TAILNET_ACTIONS, TERMINAL_FRAME_LIMIT, TERMINAL_REQUEST_LIMIT,
    TERMINAL_STREAM_CAP,
};
pub use self::peer::{close_pidfd, muse_peer, peer_cred, MusePeer, PeerCred};
