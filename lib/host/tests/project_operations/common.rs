use soda_host::preparation;

use crate::{project, Duration, Instant, Ops, RefCell, VecDeque};

// ---------- Go-captured goldens ----------

pub(super) const PID: &str = "p0123456789abcdef01234567";
pub(super) const FID: &str = "f0123456789abcdef01234567";
pub(super) const REV: &str = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789";
pub(super) const COMMIT: &str = "abcdef0123456789abcdef0123456789abcdef01";
pub(super) const ED: &str =
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre";

pub(super) const GO_ACCESS_KEY_STATE: &str = "{\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\"]}";
pub(super) const GO_ACCESS_KEY_STATE_EMPTY: &str = "{\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"keys\":[]}";
pub(super) const GO_HOLD_TRUE: &str = "{\"active\":true,\"revision\":2}";
pub(super) const GO_HOLD_FALSE: &str = "{\"active\":false,\"revision\":0}";
pub(super) const GO_PREPARE_FULL: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"running\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"tools\":[{\"name\":\"python3\",\"path\":\"/usr/bin/python3\",\"version\":\"9.9-test\"}],\"setup_exit\":0,\"check_exit\":1,\"output\":\"--- setup ---\\nok\\n--- check ---\\nfail\",\"ready\":false,\"stopped\":false}";
pub(super) const GO_PREPARE_READY: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"ready\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"ready\":true,\"stopped\":false}";
pub(super) const GO_PREPARE_STOPPED: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"stopped\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"ready\":false,\"stopped\":true,\"retirement\":\"confirmed\"}";
pub(super) const GO_PREPARE_MISSING: &str = "{\"id\":\"f0123456789abcdef01234567\",\"project\":\"p0123456789abcdef01234567\",\"role\":\"soda-coder\",\"phase\":\"waiting\",\"container\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"source_commit\":\"abcdef0123456789abcdef0123456789abcdef01\",\"setup_digest\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\",\"missing\":\"node\",\"ready\":false,\"stopped\":false}";
// Stdin bodies the ops post to podman (Go map marshal => sorted keys).
pub(super) const GO_KEYS_BODY_APPLY: &str = "{\"apply\":true,\"identity\":7,\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\"],\"login\":\"alice\",\"revision\":\"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\"}";
pub(super) const GO_ACCOUNT_BODY: &str = "{\"admin\":true,\"identity\":7,\"keys\":[\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBMdFel81xfpdgYZN9cIZY6DmqOAt/QOzaSujzaIrcre\\n\"],\"login\":\"alice\"}";
pub(super) const GO_HELPER_INSPECT: &str =
    "{\"id\":\"f0123456789abcdef01234567\",\"op\":\"inspect\"}";
// ---------- fake executor ----------
type MockCall = (Vec<u8>, String, Vec<String>);

pub(super) struct Mock {
    pub(super) calls: RefCell<Vec<MockCall>>,
    pub(super) script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
}

impl Mock {
    pub(super) fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
        Mock {
            calls: RefCell::new(Vec::new()),
            script: RefCell::new(responses.into()),
        }
    }
}

impl project::Executor for Mock {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.borrow_mut().push((
            stdin.to_vec(),
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        self.script
            .borrow_mut()
            .pop_front()
            .unwrap_or(Err("no scripted response".to_string()))
    }
}

pub(super) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

fn test_config() -> project::Config {
    project::Config {
        muse_socket: String::new(),
        image: "img".to_string(),
        network: "sodanet".to_string(),
        subnet: "10.0.0.0/24".to_string(),
        bridge: "sodabr".to_string(),
    }
}

pub(super) fn ops(mock: &Mock) -> Ops<&Mock> {
    Ops {
        exec: mock,
        config: test_config(),
    }
}

// ---------- fixtures ----------
/// `--format` inspection for the container-binding paths (container = REV so
/// state renders match the goldens byte for byte).
pub(super) fn format_inspect(running: bool) -> Vec<u8> {
    format!(
        "{{\"id\":{REV:?},\"running\":{running},\"project\":{PID:?},\"owner\":\"3\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:524288:262144\"],\"GidMap\":[\"0:524288:262144\"]}}}}"
    )
    .into_bytes()
}

/// Full `podman inspect` for the `inspect` path (account op).
pub(super) fn container_inspect(owner: &str, running: bool, ip: &str) -> Vec<u8> {
    format!(
        "[{{\"Image\":\"\",\"Config\":{{\"Labels\":{{\"org.soda.project\":{PID:?},\"org.soda.owner\":{owner:?}}}}},\"State\":{{\"Running\":{running}}},\"NetworkSettings\":{{\"Networks\":{{\"sodanet\":{{\"IPAddress\":{ip:?}}}}}}}}}]"
    )
    .into_bytes()
}

/// Factory-helper `inspect` observation, mirroring Go's `execHelperState`.
#[allow(clippy::too_many_arguments)]
pub(super) fn helper_state(
    role: &str,
    phase: &str,
    digest: &str,
    tools_json: &str,
    missing: &str,
    stopped: bool,
    ready: bool,
    exits: Option<(i64, i64)>,
    setup_log: &str,
    check_log: &str,
) -> Vec<u8> {
    let mut out = format!(
        "{{\"hold\":{{\"active\":false,\"revision\":-1}},\"known\":true,\"phase\":{phase:?},\"role\":{role:?},\"setup_digest\":{digest:?},\"source_commit\":{COMMIT:?},\"tools\":[{tools_json}],\"verified\":{{\"uid\":\"1001\",\"login\":{role:?},\"groups\":{role:?}}},\"missing\":{missing:?},\"stopped\":{stopped},\"ready\":{ready}"
    );
    if let Some((setup_exit, check_exit)) = exits {
        out.push_str(&format!(
            ",\"setup_exit\":{setup_exit},\"check_exit\":{check_exit}"
        ));
    }
    out.push_str(&format!(
        ",\"setup_log\":{setup_log:?},\"check_log\":{check_log:?}}}"
    ));
    out.into_bytes()
}

pub(super) fn approve_response(fid: &str, role: &str) -> Vec<u8> {
    format!(
        "{{\"approved\":{fid:?},\"repeated\":false,\"checkout\":\"/home/{role}/checkouts/{fid}\",\"credential_file\":\"\"}}"
    )
    .into_bytes()
}

/// Files all prepare fixtures share: setup.sh + check.sh holding `true\n`.
fn fixture_files() -> std::collections::HashMap<String, Vec<u8>> {
    std::collections::HashMap::from([
        ("setup.sh".to_string(), b"true\n".to_vec()),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ])
}

pub(super) fn fixture_digest() -> String {
    preparation::setup_digest_of(&fixture_files())
}

/// Valid `/prepare` request body (base64 `dHJ1ZQo=` = `true\n`,
/// `YnVuZGxlLWJ5dGVz` = `bundle-bytes`).
pub(super) fn prepare_body(fid: &str, role: &str, digest: &str) -> Vec<u8> {
    format!(
        "{{\"preparation\":{{\"id\":{fid:?},\"project\":{PID:?},\"role\":{role:?},\"revision\":1,\
        \"requirements\":{{\"id\":\"d0123456789abcdef01234567\",\"revision\":1,\"approver\":7,\"source_commit\":{COMMIT:?},\"digest\":{REV:?}}},\
        \"approval\":{{\"id\":\"d123456789abcdef012345678\",\"revision\":1,\"approver\":9,\"effects_digest\":{REV:?}}},\
        \"source_commit\":{COMMIT:?},\"setup_digest\":{digest:?},\"tools\":[\"python3\"],\"credential\":\"\"}},\
        \"setup\":{{\"files\":{{\"setup.sh\":\"dHJ1ZQo=\",\"check.sh\":\"dHJ1ZQo=\"}},\"bundle\":\"YnVuZGxlLWJ5dGVz\"}}}}"
    )
    .into_bytes()
}

// ---------- strict request decode ----------

// ---------- op unit tests (scripted executor) ----------
pub(super) fn binding_argv() -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        project::PROJECT_INSPECT_FORMAT.to_string(),
        format!("soda-{PID}"),
    ]
}

pub(super) fn helper_argv() -> Vec<String> {
    vec![
        "exec".to_string(),
        "--interactive".to_string(),
        format!("soda-{PID}"),
        preparation::FACTORY_HELPER.to_string(),
    ]
}

/// `{"op":...}` values posted to the factory helper, in call order.
pub(super) fn helper_ops(mock: &Mock) -> Vec<String> {
    let helper = helper_argv();
    mock.calls
        .borrow()
        .iter()
        .filter(|(_, cmd, args)| cmd == "/usr/bin/podman" && *args == helper)
        .map(|(stdin, _, _)| {
            let text = String::from_utf8_lossy(stdin);
            let start = text.find("\"op\":\"").unwrap() + "\"op\":\"".len();
            text[start..].split('"').next().unwrap().to_string()
        })
        .collect()
}
