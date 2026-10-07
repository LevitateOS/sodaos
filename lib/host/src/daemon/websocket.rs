use super::response::error_response;
use bytes::Bytes;
use http_body_util::Full;
use hyper::{Request, Response};
use tungstenite::handshake::server::create_response;

/// Build the RFC6455 handshake through tungstenite's validated HTTP API.
/// Hyper owns the actual request parsing and response serialization.
pub fn websocket_upgrade_response(request: &Request<()>) -> Response<Full<Bytes>> {
    match create_response(&request) {
        Ok(handshake) => Response::from_parts(handshake.into_parts().0, Full::new(Bytes::new())),
        Err(_) => error_response(400, "invalid private terminal request"),
    }
}
