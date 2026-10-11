use crate::json::GoFormatter;
use serde::Serialize;

#[derive(Serialize)]
struct DashboardConfig<'a> {
    listen: &'a str,
    forgejo_url: &'a str,
    forgejo_internal_url: &'a str,
    database_dsn_file: &'a str,
    host_socket: &'a str,
    identity_socket: &'a str,
    operator_id: i64,
    forgejo_background_socket: &'a str,
    forgejo_background_host_uid: Option<u32>,
    forgejo_background_credential_file: &'a str,
    forgejo_review_credential_file: &'a str,
    forgejo_merge_credential_file: &'a str,
    factory_intake_secret_file: &'a str,
    factory_publication_root: &'a str,
}

// Field declaration order, explicit empty/null optionals, Go HTML-safe string
// escaping and one trailing newline are part of the generated file contract.
#[allow(clippy::too_many_arguments)]
pub(crate) fn encode_dashboard_config(
    listen: &str,
    forgejo_url: &str,
    forgejo_internal_url: &str,
    database_dsn_file: &str,
    host_socket: &str,
    identity_socket: &str,
    operator_id: i64,
) -> Vec<u8> {
    let config = DashboardConfig {
        listen,
        forgejo_url,
        forgejo_internal_url,
        database_dsn_file,
        host_socket,
        identity_socket,
        operator_id,
        forgejo_background_socket: "",
        forgejo_background_host_uid: None,
        forgejo_background_credential_file: "",
        forgejo_review_credential_file: "",
        forgejo_merge_credential_file: "",
        factory_intake_secret_file: "",
        factory_publication_root: "",
    };
    let mut serializer = serde_json::Serializer::with_formatter(Vec::new(), GoFormatter);
    config
        .serialize(&mut serializer)
        .expect("serializing a config to a Vec cannot fail");
    let mut output = serializer.into_inner();
    output.push(b'\n');
    output
}

#[allow(clippy::too_many_arguments)]
pub(crate) fn encode_enrolled_dashboard_config(
    listen: &str,
    forgejo_url: &str,
    forgejo_internal_url: &str,
    database_dsn_file: &str,
    host_socket: &str,
    identity_socket: &str,
    operator_id: i64,
    forgejo_background_socket: &str,
    forgejo_background_host_uid: u32,
    forgejo_background_credential_file: &str,
    forgejo_review_credential_file: &str,
    forgejo_merge_credential_file: &str,
    factory_intake_secret_file: &str,
    factory_publication_root: &str,
) -> Vec<u8> {
    let config = DashboardConfig {
        listen,
        forgejo_url,
        forgejo_internal_url,
        database_dsn_file,
        host_socket,
        identity_socket,
        operator_id,
        forgejo_background_socket,
        forgejo_background_host_uid: Some(forgejo_background_host_uid),
        forgejo_background_credential_file,
        forgejo_review_credential_file,
        forgejo_merge_credential_file,
        factory_intake_secret_file,
        factory_publication_root,
    };
    let mut serializer = serde_json::Serializer::with_formatter(Vec::new(), GoFormatter);
    config
        .serialize(&mut serializer)
        .expect("serializing a config to a Vec cannot fail");
    let mut output = serializer.into_inner();
    output.push(b'\n');
    output
}
