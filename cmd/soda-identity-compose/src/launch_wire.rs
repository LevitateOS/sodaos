use super::launch_json;

pub(crate) struct NestedRegistration {
    pub(crate) child_id: String,
    pub(crate) actor_id: String,
    pub(crate) registration_id: String,
    pub(crate) muse: bool,
}

pub(crate) fn json_string(value: &str) -> String {
    launch_json::json_string(value)
}

pub(crate) fn launch_request_json(request: &NestedRegistration) -> String {
    launch_json::launch_request_json(
        &request.child_id,
        &request.actor_id,
        &request.registration_id,
        request.muse,
    )
}

pub(crate) fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    launch_json::parse_launch_exit(body)
}
