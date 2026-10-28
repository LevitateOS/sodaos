//! Explicit, bounded configuration of the native factory callback and broker.
//! Native actor bindings are reconciled through Forgejo's extensions CLI.

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{FileTypeExt, MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::process::Command;

pub(crate) const FORMAT: &str = "soda-factory-enrollment/v1";
const MAX_MANIFEST: usize = 64 * 1024;
const DASHBOARD_PATH: &str = "/etc/soda/dashboard.json";
const IDENTITY_PATH: &str = "/etc/soda/identity.json";
const ADMIN_SOCKET: &str = "/run/soda/identity/admin.sock";
const RUNTIME_SOCKET: &str = "/run/soda/identity/runtime.sock";
const HOST_SOCKET: &str = "/run/soda/host/host.sock";
const CALLBACK_SOCKET: &str = "/ipc/host.sock";
const CALLBACK_SOURCE: &str = "/run/soda/extension/host.sock";
const DASHBOARD_FACTORY_DIR: &str = "/etc/soda/factory";
const PUBLISHER_PAT: &str = "/etc/soda/factory/publisher.pat";
const REVIEWER_PAT: &str = "/etc/soda/factory/reviewer.pat";
const RESOLVER_PAT: &str = "/etc/soda/factory/resolver.pat";
const MERGER_PAT: &str = "/etc/soda/factory/merger.pat";
const INTAKE_SECRET: &str = "/etc/soda/factory/intake.secret";
const PUBLICATION_ROOT: &str = "/var/lib/soda/dashboard/publication";
const DASHBOARD_UID: u32 = 2000;
const DASHBOARD_GID: u32 = 2000;
const IDENTITY_UID: u32 = 2001;
const SODA_GID: u32 = 2000;
const FORGEJO_CONTAINER: &str = "soda-forgejo";
const FORGEJO_EXECUTABLE: &str = "/usr/local/bin/gitea";
const FORGEJO_CONFIG: &str = "/data/gitea/conf/app.ini";
const SODA_EXTENSION_ID: &str = "soda";
const NATIVE_HOOK_TOKEN: &str = "/etc/soda/enrollment/native-hooks.pat";
const NATIVE_HOOK_RECEIPT: &str = "/etc/soda/enrollment/hooks.json";
const NATIVE_INTAKE_URL: &str = "http://host.containers.internal:8080/api/factory/intake";

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct EnrollmentManifest {
    pub(crate) format: String,
    pub(crate) target: Target,
    pub(crate) dashboard: DashboardInputs,
    pub(crate) identity: IdentitySettings,
    pub(crate) bindings: NativeBindings,
    pub(crate) hooks: NativeHooks,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct NativeHooks {
    pub(crate) token_file: String,
    pub(crate) repositories: Vec<NativeHookRepository>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct NativeHookRepository {
    pub(crate) repository_id: i64,
    pub(crate) owner: String,
    pub(crate) name: String,
}

#[derive(Debug, Deserialize)]
struct ForgejoHookView {
    id: i64,
    #[serde(rename = "type")]
    hook_type: String,
    url: String,
    content_type: String,
    events: Vec<String>,
    active: bool,
}

#[derive(Deserialize)]
struct ForgejoRepositoryView {
    id: i64,
}

#[derive(Deserialize)]
struct ForgejoActorView {
    id: i64,
}

#[derive(Debug, Clone, Deserialize, Serialize, Default)]
#[serde(deny_unknown_fields)]
struct NativeHookReceipt {
    format: String,
    operator_id: i64,
    forgejo_url: String,
    forgejo_internal_url: String,
    hooks: Vec<NativeHookReceiptEntry>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct NativeHookReceiptEntry {
    repository_id: i64,
    owner: String,
    name: String,
    hook_id: i64,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct NativeBindings {
    pub(crate) installation_id: String,
    pub(crate) operations: Vec<NativeOperationBinding>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct NativeOperationBinding {
    pub(crate) token_id: i64,
    pub(crate) actor_id: i64,
    pub(crate) repository_id: i64,
    pub(crate) kind: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Target {
    pub(crate) dashboard_config: String,
    pub(crate) identity_config: String,
    pub(crate) operator_id: i64,
    pub(crate) forgejo_url: String,
    pub(crate) forgejo_internal_url: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct DashboardInputs {
    pub(crate) forgejo_background_socket: String,
    pub(crate) forgejo_background_host_uid: u32,
    pub(crate) forgejo_background_credential_file: String,
    pub(crate) forgejo_review_credential_file: String,
    pub(crate) forgejo_resolve_credential_file: String,
    pub(crate) forgejo_merge_credential_file: String,
    pub(crate) factory_intake_secret_file: String,
    pub(crate) factory_publication_root: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct IdentitySettings {
    pub(crate) database_dsn_file: String,
    pub(crate) key_file: String,
    pub(crate) admin_socket: String,
    pub(crate) runtime_socket: String,
    pub(crate) host_socket: String,
    #[serde(default)]
    pub(crate) codex: ProviderSettings,
    #[serde(default)]
    pub(crate) muse: ProviderSettings,
}

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ProviderSettings {
    #[serde(default)]
    pub(crate) binary: String,
    #[serde(default)]
    pub(crate) version: String,
    #[serde(default)]
    pub(crate) sha256: String,
    #[serde(default)]
    pub(crate) root: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
#[allow(dead_code)]
struct ExistingDashboard {
    #[serde(default)]
    factory_native_bindings: Vec<NativeOperationBinding>,
    #[serde(default = "default_listen")]
    listen: String,
    forgejo_url: String,
    forgejo_internal_url: String,
    database_dsn_file: String,
    host_socket: String,
    identity_socket: String,
    operator_id: i64,
    #[serde(default)]
    forgejo_background_socket: String,
    #[serde(default)]
    forgejo_background_host_uid: Option<u32>,
    #[serde(default)]
    forgejo_background_credential_file: String,
    #[serde(default)]
    forgejo_review_credential_file: String,
    #[serde(default)]
    forgejo_resolve_credential_file: String,
    #[serde(default)]
    forgejo_merge_credential_file: String,
    #[serde(default)]
    factory_intake_secret_file: String,
    #[serde(default)]
    factory_publication_root: String,
    #[serde(default)]
    typesafe_api_key_file: String,
    #[serde(default)]
    typesafe_model: String,
    #[serde(default)]
    forgejo_graph_credential_file: String,
    #[serde(default)]
    forgejo_graph_actor_id: i64,
}

fn default_listen() -> String {
    "127.0.0.1:8080".to_string()
}

pub(crate) fn read_manifest(path: &Path) -> Result<EnrollmentManifest, String> {
    let info =
        fs::symlink_metadata(path).map_err(|_| "enrollment manifest is unavailable".to_string())?;
    if !info.file_type().is_file() || info.permissions().mode() & 0o077 != 0 || info.uid() != 0 {
        return Err("enrollment manifest must be a private regular file".to_string());
    }
    let mut raw = Vec::new();
    fs::File::open(path)
        .and_then(|file| file.take(MAX_MANIFEST as u64 + 1).read_to_end(&mut raw))
        .map_err(|_| "enrollment manifest is unavailable".to_string())?;
    if raw.len() > MAX_MANIFEST {
        return Err("enrollment manifest exceeds 64 KiB".to_string());
    }
    let manifest: EnrollmentManifest =
        serde_json::from_slice(&raw).map_err(|_| "enrollment manifest is invalid".to_string())?;
    validate(&manifest)?;
    Ok(manifest)
}

fn absolute(value: &str) -> bool {
    Path::new(value).is_absolute() && !value.as_bytes().contains(&0)
}

fn check_provider_config(name: &str, provider: &ProviderSettings) -> Result<(), String> {
    if [&provider.binary, &provider.sha256, &provider.root]
        .iter()
        .any(|value| value.is_empty())
        || !absolute(&provider.binary)
        || !absolute(&provider.root)
        || provider.sha256.len() != 64
        || !provider.sha256.bytes().all(|byte| byte.is_ascii_hexdigit())
    {
        return Err(format!("{name} provider configuration is incomplete"));
    }
    Ok(())
}

fn digest_provider_binary(name: &str, binary: &str) -> Result<String, String> {
    let mut file =
        fs::File::open(binary).map_err(|_| format!("{name} provider executable is unavailable"))?;
    let mut digest = Sha256::new();
    let mut buffer = [0u8; 64 * 1024];
    loop {
        let count = file
            .read(&mut buffer)
            .map_err(|_| format!("{name} provider executable is unavailable"))?;
        if count == 0 {
            break;
        }
        digest.update(&buffer[..count]);
    }
    Ok(digest
        .finalize()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect::<String>())
}

fn validate_provider_binary(
    name: &str,
    provider: &ProviderSettings,
    owner: u32,
    group: u32,
) -> Result<(), String> {
    let binary = fs::symlink_metadata(&provider.binary)
        .map_err(|_| format!("{name} provider executable is unavailable"))?;
    if !binary.file_type().is_file()
        || binary.uid() != owner
        || binary.gid() != group
        || binary.permissions().mode() & 0o077 != 0
        || binary.permissions().mode() & 0o100 == 0
    {
        return Err(format!(
            "{name} provider executable is not a regular executable"
        ));
    }
    let actual = digest_provider_binary(name, &provider.binary)?;
    if actual != provider.sha256.to_ascii_lowercase() {
        return Err(format!(
            "{name} provider executable does not match its selected digest"
        ));
    }
    Ok(())
}

fn validate_provider_root(
    name: &str,
    provider: &ProviderSettings,
    owner: u32,
    group: u32,
) -> Result<(), String> {
    let root = fs::symlink_metadata(&provider.root)
        .map_err(|_| format!("{name} provider root is unavailable"))?;
    if !root.file_type().is_dir()
        || root.uid() != owner
        || root.gid() != group
        || root.permissions().mode() & 0o077 != 0
    {
        return Err(format!("{name} provider root must be a private directory"));
    }
    Ok(())
}

fn validate_provider(
    name: &str,
    provider: &ProviderSettings,
    owner: u32,
    group: u32,
) -> Result<(), String> {
    let values = [&provider.binary, &provider.sha256, &provider.root];
    if values.iter().all(|value| value.is_empty()) {
        return Ok(());
    }
    check_provider_config(name, provider)?;
    validate_provider_binary(name, provider, owner, group)?;
    validate_provider_root(name, provider, owner, group)
}

fn restricted_regular(path: &str, name: &str, owner: u32, group: u32) -> Result<(), String> {
    if !absolute(path) {
        return Err(format!("{name} must be an absolute path"));
    }
    let info = fs::symlink_metadata(path).map_err(|_| format!("{name} is unavailable"))?;
    if !info.file_type().is_file()
        || info.uid() != owner
        || info.gid() != group
        || info.permissions().mode() & 0o077 != 0
        || info.permissions().mode() & 0o400 == 0
    {
        return Err(format!("{name} must be a restricted regular file"));
    }
    Ok(())
}

fn validate(manifest: &EnrollmentManifest) -> Result<(), String> {
    validate_bindings(&manifest.bindings)?;
    validate_with(
        manifest,
        ValidationPolicy {
            dashboard_config: DASHBOARD_PATH,
            identity_config: IDENTITY_PATH,
            identity_uid: IDENTITY_UID,
            identity_gid: SODA_GID,
            dashboard_uid: DASHBOARD_UID,
            dashboard_gid: DASHBOARD_GID,
            callback_uid: 1000,
            callback_source: CALLBACK_SOURCE,
            fixed_factory_paths: true,
        },
    )
}

#[derive(Clone, Copy)]
struct ValidationPolicy<'a> {
    dashboard_config: &'a str,
    identity_config: &'a str,
    identity_uid: u32,
    identity_gid: u32,
    dashboard_uid: u32,
    dashboard_gid: u32,
    callback_uid: u32,
    callback_source: &'a str,
    fixed_factory_paths: bool,
}

fn validate_native_paths(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    if manifest.format != FORMAT {
        return Err("unsupported enrollment manifest format".to_string());
    }
    validate_hook_selection(&manifest.hooks)?;
    if policy.fixed_factory_paths {
        if manifest.hooks.token_file != NATIVE_HOOK_TOKEN {
            return Err(
                "native hook credential path differs from the installed enrollment input"
                    .to_string(),
            );
        }
        let parent = fs::symlink_metadata("/etc/soda/enrollment")
            .map_err(|_| "native enrollment directory is unavailable".to_string())?;
        if !parent.file_type().is_dir()
            || parent.uid() != 0
            || parent.gid() != 0
            || parent.permissions().mode() & 0o077 != 0
        {
            return Err("native enrollment directory must be private and root-owned".to_string());
        }
        restricted_regular(NATIVE_HOOK_TOKEN, "native hook credential", 0, 0)?;
    }
    Ok(())
}

fn validate_target_and_identity(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    let target = &manifest.target;
    if target.dashboard_config != policy.dashboard_config
        || target.identity_config != policy.identity_config
        || target.operator_id <= 0
        || target.forgejo_url.is_empty()
        || target.forgejo_internal_url.is_empty()
    {
        return Err("enrollment target does not match the installed dashboard".to_string());
    }
    if manifest.identity.admin_socket != ADMIN_SOCKET
        || manifest.identity.runtime_socket != RUNTIME_SOCKET
        || manifest.identity.host_socket != HOST_SOCKET
    {
        return Err("identity socket paths differ from installed service units".to_string());
    }
    restricted_regular(
        &manifest.identity.database_dsn_file,
        "identity database DSN",
        policy.identity_uid,
        policy.identity_gid,
    )?;
    restricted_regular(
        &manifest.identity.key_file,
        "identity encryption key",
        policy.identity_uid,
        policy.identity_gid,
    )?;
    let key = fs::read(&manifest.identity.key_file)
        .map_err(|_| "identity encryption key is unavailable".to_string())?;
    let decoded = crate::secrets::decode_base64(&key)?;
    if decoded.len() != 32 {
        return Err("identity encryption key must encode 32 bytes".to_string());
    }
    validate_provider(
        "Codex",
        &manifest.identity.codex,
        policy.identity_uid,
        policy.identity_gid,
    )?;
    validate_provider(
        "Muse",
        &manifest.identity.muse,
        policy.identity_uid,
        policy.identity_gid,
    )
}

fn validate_factory_mounts(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    let dashboard = &manifest.dashboard;
    if policy.fixed_factory_paths
        && (dashboard.forgejo_background_socket != CALLBACK_SOCKET
            || dashboard.forgejo_background_credential_file != PUBLISHER_PAT
            || dashboard.forgejo_review_credential_file != REVIEWER_PAT
            || dashboard.forgejo_resolve_credential_file != RESOLVER_PAT
            || dashboard.forgejo_merge_credential_file != MERGER_PAT
            || dashboard.factory_intake_secret_file != INTAKE_SECRET
            || dashboard.factory_publication_root != PUBLICATION_ROOT)
    {
        return Err("factory input paths differ from installed dashboard mounts".to_string());
    }
    if policy.fixed_factory_paths {
        let factory_dir = fs::symlink_metadata(DASHBOARD_FACTORY_DIR)
            .map_err(|_| "factory credential directory is unavailable".to_string())?;
        if !factory_dir.file_type().is_dir()
            || factory_dir.uid() != policy.dashboard_uid
            || factory_dir.gid() != policy.dashboard_gid
            || factory_dir.permissions().mode() & 0o077 != 0
        {
            return Err(
                "factory credential directory must be private and dashboard-owned".to_string(),
            );
        }
    }
    for (name, path) in [
        (
            "background credential",
            dashboard.forgejo_background_credential_file.as_str(),
        ),
        (
            "review credential",
            dashboard.forgejo_review_credential_file.as_str(),
        ),
        (
            "resolve credential",
            dashboard.forgejo_resolve_credential_file.as_str(),
        ),
        (
            "merge credential",
            dashboard.forgejo_merge_credential_file.as_str(),
        ),
        (
            "factory intake secret",
            dashboard.factory_intake_secret_file.as_str(),
        ),
    ] {
        restricted_regular(path, name, policy.dashboard_uid, policy.dashboard_gid)?;
    }
    Ok(())
}

fn validate_publication_and_callback(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    let dashboard = &manifest.dashboard;
    let publication = fs::symlink_metadata(&dashboard.factory_publication_root)
        .map_err(|_| "publication root is unavailable".to_string())?;
    if !publication.file_type().is_dir()
        || publication.uid() != policy.dashboard_uid
        || publication.permissions().mode() & 0o077 != 0
    {
        return Err("publication root must be a private directory".to_string());
    }
    let socket = fs::symlink_metadata(policy.callback_source)
        .map_err(|_| "background callback socket is unavailable".to_string())?;
    if !absolute(&dashboard.forgejo_background_socket)
        || dashboard.forgejo_background_host_uid != policy.callback_uid
        || !socket.file_type().is_socket()
        || socket.uid() != dashboard.forgejo_background_host_uid
    {
        return Err("background callback socket does not match its selected host UID".to_string());
    }
    Ok(())
}

fn validate_dashboard_paths(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    validate_factory_mounts(manifest, policy)?;
    validate_publication_and_callback(manifest, policy)
}

fn validate_with(
    manifest: &EnrollmentManifest,
    policy: ValidationPolicy<'_>,
) -> Result<(), String> {
    validate_native_paths(manifest, policy)?;
    validate_target_and_identity(manifest, policy)?;
    validate_dashboard_paths(manifest, policy)
}

fn validate_hook_selection(hooks: &NativeHooks) -> Result<(), String> {
    if !absolute(&hooks.token_file)
        || hooks.repositories.is_empty()
        || hooks.repositories.len() > 64
    {
        return Err(
            "native intake hook selection must name a private credential and repositories"
                .to_string(),
        );
    }
    let mut seen = std::collections::HashSet::new();
    for repository in &hooks.repositories {
        if repository.repository_id <= 0
            || !forgejo_slug(&repository.owner)
            || !forgejo_slug(&repository.name)
            || !seen.insert((
                repository.repository_id,
                repository.owner.as_str(),
                repository.name.as_str(),
            ))
        {
            return Err(
                "native intake hook repository selection is invalid or repeated".to_string(),
            );
        }
    }
    Ok(())
}

fn forgejo_slug(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 100
        && value
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b'.'))
        && value != "."
        && value != ".."
}

fn private_hook_token(path: &str, root_owned: bool) -> Result<String, String> {
    let info = fs::symlink_metadata(path)
        .map_err(|_| "native hook credential is unavailable".to_string())?;
    let owner = if root_owned { 0 } else { info.uid() };
    let group = if root_owned { 0 } else { info.gid() };
    if !info.file_type().is_file()
        || info.uid() != owner
        || info.gid() != group
        || info.permissions().mode() & 0o077 != 0
    {
        return Err("native hook credential must be a restricted regular file".to_string());
    }
    let bytes = fs::read(path).map_err(|_| "native hook credential is unavailable".to_string())?;
    let token = std::str::from_utf8(&bytes)
        .map_err(|_| "native hook credential has invalid encoding".to_string())?
        .trim_end_matches(['\n', '\r']);
    if token.is_empty()
        || token.len() > 512
        || token
            .bytes()
            .any(|byte| byte.is_ascii_control() || byte == b'"' || byte == b'\\')
    {
        return Err("native hook credential has invalid contents".to_string());
    }
    Ok(token.to_string())
}

fn curl_config_string(value: &str) -> String {
    // JSON string escaping is accepted by curl's double-quoted config syntax.
    serde_json::to_string(value).unwrap_or_else(|_| "\"\"".to_string())
}

fn native_hook_api(
    native: &dyn NativeBindingCommand,
    token: &str,
    method: &str,
    url: &str,
    body: Option<&[u8]>,
) -> Result<(u16, String), String> {
    let mut config = format!(
        "url = {}\nrequest = {}\nheader = {}\nheader = {}\nmax-time = 20\nwrite-out = {}\n",
        curl_config_string(url),
        curl_config_string(method),
        curl_config_string(&format!("Authorization: token {token}")),
        curl_config_string("Accept: application/json"),
        curl_config_string("\\nSODA_STATUS:%{http_code}\\n"),
    );
    if let Some(body) = body {
        let body = std::str::from_utf8(body)
            .map_err(|_| "native hook request encoding is invalid".to_string())?;
        config.push_str(&format!(
            "header = {}\ndata = {}\n",
            curl_config_string("Content-Type: application/json"),
            curl_config_string(body),
        ));
    }
    let response = native.request(&[], config.as_bytes())?;
    let (body, status) = response
        .rsplit_once("\nSODA_STATUS:")
        .ok_or_else(|| "native hook API response did not include an HTTP status".to_string())?;
    let status = status
        .trim()
        .parse::<u16>()
        .map_err(|_| "native hook API response status was invalid".to_string())?;
    Ok((status, body.trim().to_string()))
}

fn preflight_factory_actors(
    native: &dyn NativeBindingCommand,
    manifest: &EnrollmentManifest,
) -> Result<(), String> {
    let url = format!(
        "{}/api/v1/user",
        manifest.target.forgejo_internal_url.trim_end_matches('/'),
    );
    for (role, path, kinds) in [
        (
            "publisher",
            &manifest.dashboard.forgejo_background_credential_file,
            &["git.ref.publish", "pull_request.create"][..],
        ),
        (
            "reviewer",
            &manifest.dashboard.forgejo_review_credential_file,
            &["pull_request.review.submit"][..],
        ),
        (
            "resolver",
            &manifest.dashboard.forgejo_resolve_credential_file,
            &["pull_request.review.thread.resolve"][..],
        ),
        (
            "merger",
            &manifest.dashboard.forgejo_merge_credential_file,
            &["pull_request.merge"][..],
        ),
    ] {
        let mut bindings = manifest
            .bindings
            .operations
            .iter()
            .filter(|binding| kinds.contains(&binding.kind.as_str()));
        let actor = bindings
            .next()
            .ok_or_else(|| format!("{role} credential has no selected native operation binding"))?
            .actor_id;
        if bindings.any(|binding| binding.actor_id != actor) {
            return Err(format!(
                "{role} credential operation bindings select different native actors"
            ));
        }
        let token = private_hook_token(path, false)
            .map_err(|_| format!("{role} credential must contain a valid restricted PAT"))?;
        let (status, body) = native_hook_api(native, &token, "GET", &url, None)
            .map_err(|_| format!("{role} native credential identity could not be confirmed"))?;
        if status != 200 {
            return Err(format!(
                "{role} native credential identity was refused; read:user is required"
            ));
        }
        let actual: ForgejoActorView = serde_json::from_str(&body)
            .map_err(|_| format!("{role} native credential identity response is invalid"))?;
        if actual.id != actor {
            return Err(format!(
                "{role} credential owner differs from its selected native operation actor"
            ));
        }
    }
    Ok(())
}

fn hook_list_url(origin: &str, repository: &NativeHookRepository, page: usize) -> String {
    format!(
        "{}/api/v1/repos/{}/{}/hooks?page={page}&limit=50",
        origin.trim_end_matches('/'),
        repository.owner,
        repository.name
    )
}

fn verify_hook_repository(
    native: &dyn NativeBindingCommand,
    token: &str,
    origin: &str,
    repository: &NativeHookRepository,
) -> Result<(), String> {
    let url = format!(
        "{}/api/v1/repos/{}/{}",
        origin.trim_end_matches('/'),
        repository.owner,
        repository.name
    );
    let (status, body) = native_hook_api(native, token, "GET", &url, None)?;
    let actual: ForgejoRepositoryView = serde_json::from_str(&body)
        .map_err(|_| "selected native repository identity could not be verified".to_string())?;
    if status != 200 || actual.id != repository.repository_id {
        return Err(format!(
            "selected repository {}/{} did not match the admitted native ID; no hook was changed",
            repository.owner, repository.name
        ));
    }
    Ok(())
}

fn list_native_hooks(
    native: &dyn NativeBindingCommand,
    token: &str,
    origin: &str,
    repository: &NativeHookRepository,
) -> Result<Vec<ForgejoHookView>, String> {
    let mut all = Vec::new();
    for page in 1..=8 {
        let (status, body) = native_hook_api(
            native,
            token,
            "GET",
            &hook_list_url(origin, repository, page),
            None,
        )?;
        if status != 200 {
            return Err(format!(
                "native hook inventory was refused with HTTP {status}; no hook was changed"
            ));
        }
        let hooks: Vec<ForgejoHookView> = serde_json::from_str(&body)
            .map_err(|_| "native hook inventory response was invalid".to_string())?;
        if hooks.len() < 50 {
            all.extend(hooks);
            return Ok(all);
        }
        all.extend(hooks);
    }
    Err("native hook inventory exceeds the bounded enrollment page limit".to_string())
}

fn exact_intake_hook(hook: &ForgejoHookView) -> bool {
    // Fountain expands the requested "issues" group into these native events.
    let events = [
        "issues",
        "issue_assign",
        "issue_label",
        "issue_milestone",
        "issue_comment",
    ];
    hook.hook_type == "forgejo"
        && hook.url == NATIVE_INTAKE_URL
        && hook.content_type == "json"
        && hook.events.len() == events.len()
        && events
            .iter()
            .all(|expected| hook.events.iter().any(|event| event == expected))
        && hook.active
}

fn receipt_path(manifest: &EnrollmentManifest, policy: ValidationPolicy<'_>) -> PathBuf {
    if policy.fixed_factory_paths {
        PathBuf::from(NATIVE_HOOK_RECEIPT)
    } else {
        Path::new(&manifest.target.dashboard_config).with_file_name("hooks.json")
    }
}

/// Read the bounded receipt bytes, or `None` when no receipt exists yet.
/// A missing receipt is a fresh enrollment, not an error.
fn read_receipt_bytes(path: &Path, root_owned: bool) -> Result<Option<Vec<u8>>, String> {
    let info = match fs::symlink_metadata(path) {
        Ok(info) => info,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(_) => return Err("native hook ownership receipt is unavailable".to_string()),
    };
    let expected_owner = if root_owned {
        0
    } else {
        unsafe { libc::geteuid() }
    };
    let expected_group = if root_owned {
        0
    } else {
        unsafe { libc::getegid() }
    };
    if !info.file_type().is_file()
        || info.uid() != expected_owner
        || info.gid() != expected_group
        || info.permissions().mode() & 0o777 != 0o600
    {
        return Err("native hook ownership receipt must be a private regular file".to_string());
    }
    let mut bytes = Vec::new();
    fs::File::open(path)
        .and_then(|file| file.take(16 * 1024 + 1).read_to_end(&mut bytes))
        .map_err(|_| "native hook ownership receipt is unavailable".to_string())?;
    if bytes.len() > 16 * 1024 {
        return Err("native hook ownership receipt is invalid".to_string());
    }
    Ok(Some(bytes))
}

fn check_receipt_consistency(
    receipt: &NativeHookReceipt,
    manifest: &EnrollmentManifest,
) -> Result<(), String> {
    if receipt.format != "soda-native-hooks/v1"
        || receipt.hooks.len() > 64
        || receipt.operator_id != manifest.target.operator_id
        || receipt.forgejo_url != manifest.target.forgejo_url
        || receipt.forgejo_internal_url != manifest.target.forgejo_internal_url
        || receipt.hooks.iter().any(|entry| {
            entry.repository_id <= 0
                || entry.hook_id <= 0
                || entry.owner.is_empty()
                || entry.name.is_empty()
        })
        || receipt.hooks.iter().enumerate().any(|(index, entry)| {
            receipt.hooks[..index].iter().any(|previous| {
                previous.repository_id == entry.repository_id
                    || previous.hook_id == entry.hook_id
                    || (previous.owner == entry.owner && previous.name == entry.name)
            })
        })
        || receipt.hooks.iter().any(|entry| {
            !manifest.hooks.repositories.iter().any(|repository| {
                repository.repository_id == entry.repository_id
                    && repository.owner == entry.owner
                    && repository.name == entry.name
            })
        })
    {
        return Err("native hook ownership receipt is invalid".to_string());
    }
    Ok(())
}

fn read_hook_receipt(
    path: &Path,
    root_owned: bool,
    manifest: &EnrollmentManifest,
) -> Result<NativeHookReceipt, String> {
    let Some(bytes) = read_receipt_bytes(path, root_owned)? else {
        return Ok(NativeHookReceipt {
            format: "soda-native-hooks/v1".to_string(),
            operator_id: manifest.target.operator_id,
            forgejo_url: manifest.target.forgejo_url.clone(),
            forgejo_internal_url: manifest.target.forgejo_internal_url.clone(),
            hooks: Vec::new(),
        });
    };
    let receipt: NativeHookReceipt = serde_json::from_slice(&bytes)
        .map_err(|_| "native hook ownership receipt is invalid".to_string())?;
    check_receipt_consistency(&receipt, manifest)?;
    Ok(receipt)
}

fn write_hook_receipt(
    path: &Path,
    receipt: &NativeHookReceipt,
    root_owned: bool,
) -> Result<(), String> {
    let mut bytes = serde_json::to_vec_pretty(receipt)
        .map_err(|_| "native hook ownership receipt could not be encoded".to_string())?;
    bytes.push(b'\n');
    let (uid, gid) = if root_owned {
        (0, 0)
    } else {
        (unsafe { libc::geteuid() }, unsafe { libc::getegid() })
    };
    let temp = stage(path, &bytes, 0o600, uid, gid)?;
    if fs::rename(&temp, path).is_err() {
        let _ = fs::remove_file(temp);
        return Err("native hook ownership receipt could not be committed".to_string());
    }
    Ok(())
}

fn matching_receipt<'a>(
    receipt: &'a NativeHookReceipt,
    repository: &NativeHookRepository,
) -> Option<&'a NativeHookReceiptEntry> {
    receipt.hooks.iter().find(|entry| {
        entry.repository_id == repository.repository_id
            && entry.owner == repository.owner
            && entry.name == repository.name
    })
}

fn preflight_native_hooks(
    native: &dyn NativeBindingCommand,
    token: &str,
    origin: &str,
    hooks: &NativeHooks,
    maintenance: bool,
    receipt: &NativeHookReceipt,
) -> Result<(), String> {
    for repository in &hooks.repositories {
        verify_hook_repository(native, token, origin, repository)?;
        let inventory = list_native_hooks(native, token, origin, repository)?;
        let destinations = inventory
            .iter()
            .filter(|hook| hook.url == NATIVE_INTAKE_URL)
            .collect::<Vec<_>>();
        if let Some(hook) = destinations.first() {
            if destinations.len() != 1 || !exact_intake_hook(hook) {
                return Err(format!(
                    "repository {}/{} has a conflicting factory intake destination; existing hooks were preserved",
                    repository.owner, repository.name
                ));
            }
            let owned =
                matching_receipt(receipt, repository).is_some_and(|entry| entry.hook_id == hook.id);
            if !maintenance || !owned {
                return Err(format!(
                    "repository {}/{} already has an intake hook without an exact maintenance receipt; existing hook was preserved",
                    repository.owner, repository.name
                ));
            }
        } else if matching_receipt(receipt, repository).is_some() {
            return Err(format!(
                "repository {}/{} is recorded as enrolled but its hook is missing; no replacement was created",
                repository.owner, repository.name
            ));
        }
    }
    Ok(())
}

fn check_intake_secret(intake_secret: &[u8]) -> Result<&str, String> {
    let secret = std::str::from_utf8(intake_secret)
        .map_err(|_| "factory intake secret has invalid encoding".to_string())?
        .trim_end_matches(['\n', '\r']);
    if secret.is_empty() || secret.len() > 512 || secret.bytes().any(|b| b.is_ascii_control()) {
        return Err("factory intake secret has invalid contents".to_string());
    }
    Ok(secret)
}

fn select_hook_method(
    existing: &[&ForgejoHookView],
    repository: &NativeHookRepository,
    maintenance: bool,
    receipt: &NativeHookReceipt,
) -> Result<(&'static str, Option<i64>), String> {
    if let Some(hook) = existing.first() {
        if existing.len() != 1
            || !exact_intake_hook(hook)
            || !maintenance
            || matching_receipt(receipt, repository).map_or(true, |entry| entry.hook_id != hook.id)
        {
            return Err(format!(
                "repository {}/{} intake hook changed after preflight; no hook was changed",
                repository.owner, repository.name
            ));
        }
        return Ok(("PATCH", Some(hook.id)));
    }
    if matching_receipt(receipt, repository).is_some() {
        return Err(format!("repository {}/{} enrolled hook disappeared after preflight; no replacement was created", repository.owner, repository.name));
    }
    Ok(("POST", None))
}

fn send_repository_hook(
    native: &dyn NativeBindingCommand,
    token: &str,
    origin: &str,
    secret: &str,
    repository: &NativeHookRepository,
    method: &str,
    hook_id: Option<i64>,
) -> Result<u16, String> {
    let url = if let Some(id) = hook_id {
        format!(
            "{}/api/v1/repos/{}/{}/hooks/{id}",
            origin.trim_end_matches('/'),
            repository.owner,
            repository.name
        )
    } else {
        format!(
            "{}/api/v1/repos/{}/{}/hooks",
            origin.trim_end_matches('/'),
            repository.owner,
            repository.name
        )
    };
    let create_body = serde_json::to_vec(&serde_json::json!({
        "type": "forgejo",
        "config": {
            "url": NATIVE_INTAKE_URL,
            "content_type": "json",
            "secret": secret,
        },
        "events": ["issues"],
        "active": true,
    }))
    .map_err(|_| "native hook request could not be encoded".to_string())?;
    let (status, _) = native_hook_api(native, token, method, &url, Some(&create_body))?;
    if (method == "POST" && status != 201) || (method == "PATCH" && status != 200 && status != 204)
    {
        return Err(format!(
            "native intake hook {method} failed with HTTP {status} for {}/{}; any earlier changes were retained",
            repository.owner, repository.name
        ));
    }
    Ok(status)
}

fn confirm_repository_hook(
    native: &dyn NativeBindingCommand,
    token: &str,
    origin: &str,
    repository: &NativeHookRepository,
    hook_id: Option<i64>,
) -> Result<i64, String> {
    let inventory = list_native_hooks(native, token, origin, repository).map_err(|_| {
        format!(
            "native intake hook was created for {}/{}, but its readback failed; inspect the retained hook",
            repository.owner, repository.name
        )
    })?;
    let matching = inventory
        .iter()
        .filter(|hook| hook.id > 0 && hook.url == NATIVE_INTAKE_URL && exact_intake_hook(hook))
        .collect::<Vec<_>>();
    if matching.len() != 1 || hook_id.is_some_and(|id| matching[0].id != id) {
        return Err(format!(
            "native intake hook was created for {}/{}, but exact readback was not confirmed; inspect the retained hook",
            repository.owner, repository.name
        ));
    }
    Ok(matching[0].id)
}

fn record_repository_hook(
    receipt: &mut NativeHookReceipt,
    repository: &NativeHookRepository,
    hook_id: i64,
) -> Result<(), String> {
    let entry = NativeHookReceiptEntry {
        repository_id: repository.repository_id,
        owner: repository.owner.clone(),
        name: repository.name.clone(),
        hook_id,
    };
    if let Some(prior) = matching_receipt(receipt, repository) {
        if prior.hook_id != entry.hook_id {
            return Err("native hook ownership changed during reconciliation".to_string());
        }
    } else {
        receipt.hooks.push(entry);
    }
    Ok(())
}

fn create_native_hooks(
    native: &dyn NativeBindingCommand,
    token: &str,
    intake_secret: &[u8],
    origin: &str,
    hooks: &NativeHooks,
    maintenance: bool,
    receipt_path: &Path,
    receipt: &mut NativeHookReceipt,
    root_owned_receipt: bool,
) -> Result<(), String> {
    let secret = check_intake_secret(intake_secret)?;
    for repository in &hooks.repositories {
        verify_hook_repository(native, token, origin, repository)?;
        let inventory = list_native_hooks(native, token, origin, repository)?;
        let existing = inventory
            .iter()
            .filter(|hook| hook.url == NATIVE_INTAKE_URL)
            .collect::<Vec<_>>();
        let (method, hook_id) = select_hook_method(&existing, repository, maintenance, receipt)?;
        send_repository_hook(native, token, origin, secret, repository, method, hook_id)?;
        let confirmed = confirm_repository_hook(native, token, origin, repository, hook_id)?;
        record_repository_hook(receipt, repository, confirmed)?;
        write_hook_receipt(receipt_path, receipt, root_owned_receipt)?;
    }
    Ok(())
}

pub(crate) trait IdentityServiceAction {
    fn run(&self, args: &[&str]) -> Result<(), String>;
}

pub(crate) trait NativeBindingCommand {
    fn run(&self, args: &[&str]) -> Result<String, String>;
    fn request(&self, _args: &[&str], _input: &[u8]) -> Result<String, String> {
        Err("native hook API command is unavailable".to_string())
    }
}

struct PodmanForgejo;

impl NativeBindingCommand for PodmanForgejo {
    fn run(&self, args: &[&str]) -> Result<String, String> {
        let mut command = Command::new("/usr/bin/podman");
        command.args([
            "exec",
            "--user",
            "1000:1000",
            FORGEJO_CONTAINER,
            FORGEJO_EXECUTABLE,
            "--config",
            FORGEJO_CONFIG,
        ]);
        command.args(args);
        let output = command
            .output()
            .map_err(|_| "native Forgejo binding command could not run".to_string())?;
        if !output.status.success() {
            return Err(
                "native Forgejo binding command failed; inspect the selected installation and credentials"
                    .to_string(),
            );
        }
        String::from_utf8(output.stdout)
            .map_err(|_| "native Forgejo binding response was invalid".to_string())
    }

    fn request(&self, args: &[&str], input: &[u8]) -> Result<String, String> {
        let mut command = Command::new("/usr/bin/podman");
        command.args([
            "exec",
            "--interactive",
            "--user",
            "1000:1000",
            FORGEJO_CONTAINER,
            "/usr/bin/curl",
            "--silent",
            "--show-error",
            "--config",
            "-",
        ]);
        command.args(args);
        command.stdin(std::process::Stdio::piped());
        command.stdout(std::process::Stdio::piped());
        command.stderr(std::process::Stdio::null());
        let mut child = command
            .spawn()
            .map_err(|_| "native Forgejo hook API command could not run".to_string())?;
        if child
            .stdin
            .take()
            .ok_or_else(|| "native Forgejo hook API input is unavailable".to_string())?
            .write_all(input)
            .is_err()
        {
            let _ = child.kill();
            let _ = child.wait();
            return Err("native Forgejo hook API input could not be supplied".to_string());
        }
        let output = child
            .wait_with_output()
            .map_err(|_| "native Forgejo hook API response is unavailable".to_string())?;
        if !output.status.success() {
            return Err("native Forgejo hook API command failed".to_string());
        }
        String::from_utf8(output.stdout)
            .map_err(|_| "native Forgejo hook API response was invalid".to_string())
    }
}

struct Systemctl;

impl IdentityServiceAction for Systemctl {
    fn run(&self, args: &[&str]) -> Result<(), String> {
        let status = Command::new("/usr/bin/systemctl")
            .args(args)
            .status()
            .map_err(|_| "systemd identity-service action could not run".to_string())?;
        if status.success() {
            Ok(())
        } else {
            Err(
                "systemd identity-service action failed; configuration files were retained"
                    .to_string(),
            )
        }
    }
}

pub(crate) fn apply(
    manifest: EnrollmentManifest,
    maintenance: bool,
) -> Result<BindingEnrollmentReport, String> {
    apply_with_commands(manifest, maintenance, &Systemctl, &PodmanForgejo)
}

pub(crate) fn apply_with_commands(
    manifest: EnrollmentManifest,
    maintenance: bool,
    service: &dyn IdentityServiceAction,
    native: &dyn NativeBindingCommand,
) -> Result<BindingEnrollmentReport, String> {
    apply_at(
        manifest,
        maintenance,
        service,
        native,
        ValidationPolicy {
            dashboard_config: DASHBOARD_PATH,
            identity_config: IDENTITY_PATH,
            identity_uid: IDENTITY_UID,
            identity_gid: SODA_GID,
            dashboard_uid: DASHBOARD_UID,
            dashboard_gid: DASHBOARD_GID,
            callback_uid: 1000,
            callback_source: CALLBACK_SOURCE,
            fixed_factory_paths: true,
        },
    )
}

/// Admitted enrollment target: the validated manifest plus the existing
/// dashboard record, hook custody and maintenance-mode gates.
struct AdmittedEnrollmentTarget {
    existing: ExistingDashboard,
    identity_exists: bool,
    hook_token: String,
    hook_receipt_path: PathBuf,
    hook_receipt: NativeHookReceipt,
}

fn read_existing_dashboard(manifest: &EnrollmentManifest) -> Result<ExistingDashboard, String> {
    let dashboard_path = Path::new(&manifest.target.dashboard_config);
    let dashboard_info = fs::symlink_metadata(dashboard_path)
        .map_err(|_| "existing dashboard configuration is unavailable".to_string())?;
    if !dashboard_info.file_type().is_file() {
        return Err("dashboard configuration must be a regular file".to_string());
    }
    let dashboard_raw = fs::read(dashboard_path)
        .map_err(|_| "dashboard configuration is unavailable".to_string())?;
    let existing: ExistingDashboard = serde_json::from_slice(&dashboard_raw)
        .map_err(|_| "existing dashboard configuration is invalid".to_string())?;
    if existing.operator_id != manifest.target.operator_id
        || existing.forgejo_url != manifest.target.forgejo_url
        || existing.forgejo_internal_url != manifest.target.forgejo_internal_url
        || existing.host_socket != HOST_SOCKET
        || !absolute(&existing.database_dsn_file)
        || !matches!(existing.identity_socket.as_str(), "" | ADMIN_SOCKET)
    {
        return Err(
            "dashboard operator, origins or native sockets changed; inspect before enrollment"
                .to_string(),
        );
    }
    Ok(existing)
}

fn check_maintenance_gates(
    identity_exists: bool,
    maintenance: bool,
    hook_receipt: &NativeHookReceipt,
) -> Result<(), String> {
    if identity_exists && !maintenance {
        return Err("identity configuration exists; pass --maintenance after reviewing the selected replacement".to_string());
    }
    if !identity_exists && maintenance && hook_receipt.hooks.is_empty() {
        return Err(
            "--maintenance without an identity configuration requires a prior native hook receipt"
                .to_string(),
        );
    }
    if !identity_exists && !maintenance && !hook_receipt.hooks.is_empty() {
        return Err("an incomplete native enrollment has a hook receipt; pass --maintenance to resume the exact selected target".to_string());
    }
    Ok(())
}

fn admit_enrollment_target(
    manifest: &EnrollmentManifest,
    maintenance: bool,
    policy: ValidationPolicy<'_>,
) -> Result<AdmittedEnrollmentTarget, String> {
    validate_with(manifest, policy)?;
    validate_bindings(&manifest.bindings)?;
    let existing = read_existing_dashboard(manifest)?;
    let identity_path = Path::new(&manifest.target.identity_config);
    let identity_exists = match fs::symlink_metadata(identity_path) {
        Ok(info) if info.file_type().is_file() => true,
        Ok(_) => return Err("identity configuration must be a regular file".to_string()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => false,
        Err(_) => return Err("identity configuration cannot be inspected".to_string()),
    };
    let hook_token = private_hook_token(&manifest.hooks.token_file, policy.fixed_factory_paths)?;
    let hook_receipt_path = receipt_path(manifest, policy);
    let hook_receipt = read_hook_receipt(&hook_receipt_path, policy.fixed_factory_paths, manifest)?;
    check_maintenance_gates(identity_exists, maintenance, &hook_receipt)?;
    Ok(AdmittedEnrollmentTarget {
        existing,
        identity_exists,
        hook_token,
        hook_receipt_path,
        hook_receipt,
    })
}

/// Captured pre-write state for one enrollment run.
struct EnrollmentPreflight {
    dashboard_before: Snapshot,
    identity_before: Snapshot,
    existing_bindings: Vec<ExistingNativeBinding>,
}

fn preflight_enrollment_writes(
    native: &dyn NativeBindingCommand,
    manifest: &EnrollmentManifest,
    admitted: &AdmittedEnrollmentTarget,
    maintenance: bool,
) -> Result<EnrollmentPreflight, String> {
    preflight_factory_actors(native, manifest)?;
    let dashboard_path = Path::new(&manifest.target.dashboard_config);
    let identity_path = Path::new(&manifest.target.identity_config);
    let dashboard_before = Snapshot::capture(dashboard_path)?;
    let identity_before = Snapshot::capture(identity_path)?;
    let existing_bindings = inspect_native_bindings(native, &manifest.bindings.installation_id)?;
    preflight_native_hooks(
        native,
        &admitted.hook_token,
        &manifest.target.forgejo_internal_url,
        &manifest.hooks,
        maintenance,
        &admitted.hook_receipt,
    )?;
    Ok(EnrollmentPreflight {
        dashboard_before,
        identity_before,
        existing_bindings,
    })
}

/// Roll back written configs after a failed enrollment step and restore
/// services. The config-rollback error propagates when the rollback itself
/// fails; otherwise the unrecovered message when service recovery is
/// unconfirmed, else the step's own error.
fn recover_enrolled_configs(
    service: &dyn IdentityServiceAction,
    preflight: &EnrollmentPreflight,
    identity_exists: bool,
    primary: String,
    unrecovered: &str,
) -> Result<String, String> {
    rollback(&preflight.dashboard_before, &preflight.identity_before)?;
    if restore_services_after_rollback(service, identity_exists).is_err() {
        return Ok(unrecovered.to_string());
    }
    Ok(primary)
}

fn apply_at(
    manifest: EnrollmentManifest,
    maintenance: bool,
    service: &dyn IdentityServiceAction,
    native: &dyn NativeBindingCommand,
    policy: ValidationPolicy<'_>,
) -> Result<BindingEnrollmentReport, String> {
    let admitted = admit_enrollment_target(&manifest, maintenance, policy)?;
    let preflight = preflight_enrollment_writes(native, &manifest, &admitted, maintenance)?;
    let AdmittedEnrollmentTarget {
        existing,
        identity_exists,
        hook_token,
        hook_receipt_path,
        mut hook_receipt,
    } = admitted;
    let dashboard_path = Path::new(&manifest.target.dashboard_config);
    let identity_path = Path::new(&manifest.target.identity_config);
    let dsn = existing.database_dsn_file;
    let policy_bindings: Vec<_> = manifest
        .bindings
        .operations
        .iter()
        .filter(|binding| binding.kind != "issue.graph.mutate")
        .cloned()
        .collect();
    let dashboard = crate::config::encode_enrolled_dashboard_config(
        &existing.listen,
        &manifest.target.forgejo_url,
        &manifest.target.forgejo_internal_url,
        &dsn,
        HOST_SOCKET,
        ADMIN_SOCKET,
        manifest.target.operator_id,
        &manifest.dashboard.forgejo_background_socket,
        manifest.dashboard.forgejo_background_host_uid,
        &manifest.dashboard.forgejo_background_credential_file,
        &manifest.dashboard.forgejo_review_credential_file,
        &manifest.dashboard.forgejo_resolve_credential_file,
        &manifest.dashboard.forgejo_merge_credential_file,
        &manifest.dashboard.factory_intake_secret_file,
        &manifest.dashboard.factory_publication_root,
        &existing.typesafe_api_key_file,
        &existing.typesafe_model,
        &existing.forgejo_graph_credential_file,
        existing.forgejo_graph_actor_id,
        &policy_bindings,
    );
    let identity = serde_json::to_vec(&manifest.identity)
        .map_err(|_| "could not encode identity configuration".to_string())?;
    let mut identity = identity;
    identity.push(b'\n');
    replace_pair_with_identity_owner(
        dashboard_path,
        &dashboard,
        identity_path,
        &identity,
        policy.identity_uid,
        policy.identity_gid,
    )?;
    if let Err(err) = activate_services(service, maintenance) {
        return Err(recover_enrolled_configs(
            service,
            &preflight,
            identity_exists,
            err,
            "service activation failed and previous configuration was restored, but service recovery was not confirmed",
        )?);
    }
    match reconcile_native_bindings(native, &manifest.bindings, &preflight.existing_bindings) {
        Ok(report) => {
            let intake_secret = match fs::read(&manifest.dashboard.factory_intake_secret_file) {
                Ok(secret) => secret,
                Err(_) => {
                    return Err(recover_enrolled_configs(
                        service,
                        &preflight,
                        identity_exists,
                        "factory intake secret became unavailable; previous configuration was restored".to_string(),
                        "factory intake secret became unavailable; previous configuration was restored, but service recovery was not confirmed",
                    )?);
                }
            };
            if let Err(err) = create_native_hooks(
                native,
                &hook_token,
                &intake_secret,
                &manifest.target.forgejo_internal_url,
                &manifest.hooks,
                maintenance,
                &hook_receipt_path,
                &mut hook_receipt,
                policy.fixed_factory_paths,
            ) {
                return Err(recover_enrolled_configs(
                    service,
                    &preflight,
                    identity_exists,
                    err,
                    "native hook enrollment was incomplete; previous configuration was restored, but service recovery was not confirmed",
                )?);
            }
            Ok(report)
        }
        Err(err) => Err(recover_enrolled_configs(
            service,
            &preflight,
            identity_exists,
            err,
            "native binding enrollment was incomplete; previous configuration was restored, but service recovery was not confirmed",
        )?),
    }
}

fn restore_services_after_rollback(
    service: &dyn IdentityServiceAction,
    maintenance: bool,
) -> Result<(), String> {
    if maintenance {
        service.run(&["restart", "soda-identity.service"])?;
    } else {
        service.run(&[
            "stop",
            "soda-identity.service",
            "soda-identity.socket",
            "soda-identity-runtime.socket",
        ])?;
        service.run(&[
            "disable",
            "--now",
            "soda-identity.socket",
            "soda-identity-runtime.socket",
        ])?;
    }
    service.run(&["restart", "soda-dashboard.service"])
}

fn activate_services(service: &dyn IdentityServiceAction, maintenance: bool) -> Result<(), String> {
    service.run(&[
        "enable",
        "--now",
        "soda-identity.socket",
        "soda-identity-runtime.socket",
    ])?;
    service.run(&[
        if maintenance { "restart" } else { "start" },
        "soda-identity.service",
    ])?;
    service.run(&["restart", "soda-dashboard.service"])
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct ExistingNativeBinding {
    token_id: i64,
    actor_id: i64,
    repository_id: i64,
    kind: String,
}

fn validate_bindings(bindings: &NativeBindings) -> Result<(), String> {
    let bytes = bindings.installation_id.as_bytes();
    let valid_uuid = bytes.len() == 36
        && [8, 13, 18, 23].iter().all(|index| bytes[*index] == b'-')
        && bytes.iter().enumerate().all(|(index, byte)| {
            [8, 13, 18, 23].contains(&index)
                || byte.is_ascii_hexdigit() && !byte.is_ascii_uppercase()
        });
    if !valid_uuid || bindings.operations.is_empty() || bindings.operations.len() > 64 {
        return Err(
            "native binding selection must name the existing installation and selected operations"
                .to_string(),
        );
    }
    let mut unique = std::collections::HashSet::new();
    for binding in &bindings.operations {
        if binding.token_id <= 0
            || binding.actor_id <= 0
            || binding.repository_id <= 0
            || !matches!(
                binding.kind.as_str(),
                "git.ref.publish"
                    | "pull_request.create"
                    | "pull_request.review.submit"
                    | "pull_request.review.thread.resolve"
                    | "issue.graph.mutate"
                    | "pull_request.merge"
            )
        {
            return Err("native binding selection contains an invalid token, actor, repository or operation".to_string());
        }
        if !unique.insert((
            binding.token_id,
            binding.actor_id,
            binding.repository_id,
            binding.kind.as_str(),
        )) {
            return Err("native binding selection repeats an operation tuple".to_string());
        }
    }
    Ok(())
}

fn inspect_native_bindings(
    native: &dyn NativeBindingCommand,
    installation: &str,
) -> Result<Vec<ExistingNativeBinding>, String> {
    let identity = native.run(&["extensions", "installation", SODA_EXTENSION_ID])?;
    let expected = format!("{SODA_EXTENSION_ID}\t{installation}\n");
    if identity != expected {
        return Err(
            "native Soda extension installation identity differs from the reviewed manifest"
                .to_string(),
        );
    }
    let output = native.run(&["extensions", "bindings", "--installation", installation])?;
    parse_native_bindings(&output)
}

fn parse_native_bindings(output: &str) -> Result<Vec<ExistingNativeBinding>, String> {
    let mut parsed = Vec::new();
    for line in output.lines() {
        if line.is_empty() {
            continue;
        }
        let fields = line.split('\t').collect::<Vec<_>>();
        if fields.len() != 5 || fields[0].parse::<i64>().ok().filter(|id| *id > 0).is_none() {
            return Err("native binding inventory is malformed".to_string());
        }
        let value = |field: &str, key: &str| {
            field
                .strip_prefix(key)
                .and_then(|value| value.parse::<i64>().ok())
                .filter(|value| *value > 0)
        };
        let binding = ExistingNativeBinding {
            token_id: value(fields[1], "token=")
                .ok_or_else(|| "native binding inventory is malformed".to_string())?,
            actor_id: value(fields[2], "actor=")
                .ok_or_else(|| "native binding inventory is malformed".to_string())?,
            repository_id: value(fields[3], "repository=")
                .ok_or_else(|| "native binding inventory is malformed".to_string())?,
            kind: fields[4]
                .strip_prefix("kind=")
                .filter(|kind| {
                    matches!(
                        *kind,
                        "git.ref.publish"
                            | "pull_request.create"
                            | "pull_request.review.submit"
                            | "pull_request.review.thread.resolve"
                            | "issue.graph.mutate"
                            | "pull_request.merge"
                    )
                })
                .ok_or_else(|| "native binding inventory is malformed".to_string())?
                .to_string(),
        };
        parsed.push(binding);
    }
    Ok(parsed)
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct BindingEnrollmentReport {
    pub(crate) already_present: usize,
    pub(crate) added: usize,
}

fn reconcile_native_bindings(
    native: &dyn NativeBindingCommand,
    manifest: &NativeBindings,
    before: &[ExistingNativeBinding],
) -> Result<BindingEnrollmentReport, String> {
    let mut already_present = 0;
    let mut added = 0;
    for binding in &manifest.operations {
        // `bind` is idempotent for the exact native tuple and validates that
        // its token still exists and belongs to the selected actor.
        let token = binding.token_id.to_string();
        let actor = binding.actor_id.to_string();
        let repository = binding.repository_id.to_string();
        native
            .run(&[
                "extensions",
                "bind",
                "--installation",
                &manifest.installation_id,
                "--token-id",
                &token,
                "--actor-id",
                &actor,
                "--repository-id",
                &repository,
                "--kind",
                &binding.kind,
            ])
            .map_err(|_| {
                format!(
                    "native token or actor for {} in repository {} is missing, revoked or mismatched; any earlier exact bindings were retained",
                    binding.kind, binding.repository_id
                )
            })?;
    }
    let after = inspect_native_bindings(native, &manifest.installation_id).map_err(|_| {
        "native bindings could not be confirmed after enrollment; inspect the installation, any exact bindings added were retained".to_string()
    })?;
    for binding in &manifest.operations {
        let expected = ExistingNativeBinding {
            token_id: binding.token_id,
            actor_id: binding.actor_id,
            repository_id: binding.repository_id,
            kind: binding.kind.clone(),
        };
        if !after.contains(&expected) {
            return Err(format!(
                "native binding for {} in repository {} was not confirmed; any earlier exact bindings were retained",
                binding.kind, binding.repository_id
            ));
        }
        if before.contains(&expected) {
            already_present += 1;
        } else {
            added += 1;
        }
    }
    Ok(BindingEnrollmentReport {
        already_present,
        added,
    })
}

struct Snapshot {
    path: PathBuf,
    bytes: Option<Vec<u8>>,
    mode: u32,
    uid: u32,
    gid: u32,
}

impl Snapshot {
    fn capture(path: &Path) -> Result<Self, String> {
        match fs::symlink_metadata(path) {
            Ok(info) if info.file_type().is_file() => Ok(Self {
                path: path.to_path_buf(),
                bytes: Some(
                    fs::read(path).map_err(|_| "configuration is unavailable".to_string())?,
                ),
                mode: info.permissions().mode() & 0o777,
                uid: info.uid(),
                gid: info.gid(),
            }),
            Ok(_) => Err("configuration must be a regular file".to_string()),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(Self {
                path: path.to_path_buf(),
                bytes: None,
                mode: 0o600,
                uid: IDENTITY_UID,
                gid: SODA_GID,
            }),
            Err(_) => Err("configuration cannot be inspected".to_string()),
        }
    }
}

fn rollback(dashboard: &Snapshot, identity: &Snapshot) -> Result<(), String> {
    restore_snapshot(identity)?;
    restore_snapshot(dashboard)
}

fn restore_snapshot(snapshot: &Snapshot) -> Result<(), String> {
    if let Some(bytes) = &snapshot.bytes {
        restore(
            &snapshot.path,
            bytes,
            snapshot.mode,
            snapshot.uid,
            snapshot.gid,
        )
    } else {
        match fs::remove_file(&snapshot.path) {
            Ok(()) => Ok(()),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(_) => Err("configuration rollback could not be confirmed".to_string()),
        }
    }
}

fn temp_for(path: &Path) -> Result<(PathBuf, fs::File), String> {
    let parent = path
        .parent()
        .ok_or_else(|| "configuration target has no parent".to_string())?;
    let name = path
        .file_name()
        .ok_or_else(|| "configuration target has no name".to_string())?
        .to_string_lossy();
    for attempt in 0..16 {
        let temp = parent.join(format!(".{name}.enroll-{}-{attempt}", std::process::id()));
        match OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&temp)
        {
            Ok(file) => return Ok((temp, file)),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(_) => return Err("cannot stage configuration file".to_string()),
        }
    }
    Err("cannot reserve private configuration staging file".to_string())
}

fn stage(path: &Path, bytes: &[u8], mode: u32, uid: u32, gid: u32) -> Result<PathBuf, String> {
    let (temp, mut file) = temp_for(path)?;
    if file.write_all(bytes).and_then(|_| file.sync_all()).is_err() {
        let _ = fs::remove_file(&temp);
        return Err("cannot write staged configuration".to_string());
    }
    drop(file);
    if std::os::unix::fs::chown(&temp, Some(uid), Some(gid)).is_err()
        || fs::set_permissions(&temp, fs::Permissions::from_mode(mode)).is_err()
    {
        let _ = fs::remove_file(&temp);
        return Err("cannot set staged configuration ownership or mode".to_string());
    }
    Ok(temp)
}

fn replace_pair_with_identity_owner(
    dashboard_path: &Path,
    dashboard: &[u8],
    identity_path: &Path,
    identity: &[u8],
    new_identity_uid: u32,
    new_identity_gid: u32,
) -> Result<(), String> {
    let dashboard_info = fs::metadata(dashboard_path)
        .map_err(|_| "dashboard configuration is unavailable".to_string())?;
    let dashboard_stage = stage(
        dashboard_path,
        dashboard,
        dashboard_info.permissions().mode() & 0o777,
        dashboard_info.uid(),
        dashboard_info.gid(),
    )?;
    let identity_info = fs::symlink_metadata(identity_path).ok();
    let (identity_mode, identity_uid, identity_gid) = identity_info
        .map(|info| (info.permissions().mode() & 0o777, info.uid(), info.gid()))
        .unwrap_or((0o600, new_identity_uid, new_identity_gid));
    let identity_stage = match stage(
        identity_path,
        identity,
        identity_mode,
        identity_uid,
        identity_gid,
    ) {
        Ok(stage) => stage,
        Err(err) => {
            let _ = fs::remove_file(dashboard_stage);
            return Err(err);
        }
    };
    let old_dashboard = fs::read(dashboard_path)
        .map_err(|_| "dashboard configuration is unavailable".to_string())?;
    let old_dashboard_mode = fs::metadata(dashboard_path)
        .map_err(|_| "dashboard configuration is unavailable".to_string())?
        .permissions()
        .mode()
        & 0o777;
    if fs::rename(&dashboard_stage, dashboard_path).is_err() {
        let _ = fs::remove_file(&dashboard_stage);
        let _ = fs::remove_file(&identity_stage);
        return Err("cannot install dashboard configuration".to_string());
    }
    if fs::rename(&identity_stage, identity_path).is_err() {
        let _ = fs::remove_file(&identity_stage);
        restore(
            dashboard_path,
            &old_dashboard,
            old_dashboard_mode,
            dashboard_info.uid(),
            dashboard_info.gid(),
        )?;
        return Err(
            "cannot install identity configuration; dashboard bytes were restored".to_string(),
        );
    }
    Ok(())
}

fn restore(path: &Path, bytes: &[u8], mode: u32, uid: u32, gid: u32) -> Result<(), String> {
    let temp = stage(path, bytes, mode, uid, gid)?;
    if fs::rename(&temp, path).is_err() {
        let _ = fs::remove_file(temp);
        return Err("configuration rollback could not be confirmed".to_string());
    }
    Ok(())
}
