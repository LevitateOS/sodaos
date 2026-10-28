use crate::json::GoFormatter;
use serde::Serialize;

#[derive(Serialize)]
struct DashboardConfig<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    factory_native_bindings: Option<&'a [crate::factory_enrollment::NativeOperationBinding]>,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    forgejo_resolve_credential_file: Option<&'a str>,
    forgejo_merge_credential_file: &'a str,
    factory_intake_secret_file: &'a str,
    factory_publication_root: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    typesafe_api_key_file: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    typesafe_model: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    forgejo_graph_credential_file: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    forgejo_graph_actor_id: Option<i64>,
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
        factory_native_bindings: None,
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
        forgejo_resolve_credential_file: None,
        forgejo_merge_credential_file: "",
        factory_intake_secret_file: "",
        factory_publication_root: "",
        typesafe_api_key_file: None,
        typesafe_model: None,
        forgejo_graph_credential_file: None,
        forgejo_graph_actor_id: None,
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
    forgejo_resolve_credential_file: &str,
    forgejo_merge_credential_file: &str,
    factory_intake_secret_file: &str,
    factory_publication_root: &str,
    typesafe_api_key_file: &str,
    typesafe_model: &str,
    forgejo_graph_credential_file: &str,
    forgejo_graph_actor_id: i64,
    factory_native_bindings: &[crate::factory_enrollment::NativeOperationBinding],
) -> Vec<u8> {
    let config = DashboardConfig {
        factory_native_bindings: Some(factory_native_bindings),
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
        forgejo_resolve_credential_file: Some(forgejo_resolve_credential_file),
        forgejo_merge_credential_file,
        factory_intake_secret_file,
        factory_publication_root,
        typesafe_api_key_file: (!typesafe_api_key_file.is_empty()).then_some(typesafe_api_key_file),
        typesafe_model: (!typesafe_model.is_empty()).then_some(typesafe_model),
        forgejo_graph_credential_file: (!forgejo_graph_credential_file.is_empty())
            .then_some(forgejo_graph_credential_file),
        forgejo_graph_actor_id: (forgejo_graph_actor_id != 0).then_some(forgejo_graph_actor_id),
    };
    let mut serializer = serde_json::Serializer::with_formatter(Vec::new(), GoFormatter);
    config
        .serialize(&mut serializer)
        .expect("serializing a config to a Vec cannot fail");
    let mut output = serializer.into_inner();
    output.push(b'\n');
    output
}
