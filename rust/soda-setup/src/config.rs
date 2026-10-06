use crate::json::push_json_string;

// encode_dashboard_config mirrors Go's json.Encoder over config.Config for a
// fresh setup record: all keys in struct order (including "" and null for
// unset optionals), Go string escaping, and a trailing newline.
#[allow(clippy::too_many_arguments)]
pub(crate) fn encode_dashboard_config(
    listen: &str,
    forgejo_url: &str,
    forgejo_internal_url: &str,
    database_dsn_file: &str,
    host_socket: &str,
    identity_socket: &str,
    grant_key_file: &str,
    operator_id: i64,
) -> Vec<u8> {
    let mut body = String::from("{");
    let mut first = true;
    let field = |body: &mut String, first: &mut bool, key: &str, value: &str| {
        if !*first {
            body.push(',');
        }
        *first = false;
        push_json_string(body, key);
        body.push(':');
        push_json_string(body, value);
    };
    field(&mut body, &mut first, "listen", listen);
    field(&mut body, &mut first, "forgejo_url", forgejo_url);
    field(
        &mut body,
        &mut first,
        "forgejo_internal_url",
        forgejo_internal_url,
    );
    field(
        &mut body,
        &mut first,
        "database_dsn_file",
        database_dsn_file,
    );
    field(&mut body, &mut first, "host_socket", host_socket);
    field(&mut body, &mut first, "identity_socket", identity_socket);
    field(&mut body, &mut first, "grant_key_file", grant_key_file);
    body.push(',');
    push_json_string(&mut body, "operator_id");
    body.push(':');
    body.push_str(&operator_id.to_string());
    body.push(',');
    push_json_string(&mut body, "forgejo_background_socket");
    body.push_str(":\"\",");
    push_json_string(&mut body, "forgejo_background_host_uid");
    body.push_str(":null");
    for key in [
        "forgejo_background_credential_file",
        "forgejo_review_credential_file",
        "forgejo_merge_credential_file",
        "factory_intake_secret_file",
        "factory_publication_root",
    ] {
        body.push(',');
        push_json_string(&mut body, key);
        body.push_str(":\"\"");
    }
    body.push_str("}\n");
    body.into_bytes()
}
