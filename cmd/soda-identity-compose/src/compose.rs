use super::json_string;
use super::options::Options;
use super::MUSE_LAUNCH_SOCKET;
use std::fs;
use std::os::unix::fs::OpenOptionsExt;
use std::path::Path;
use std::process::{Command, Stdio};

pub(crate) fn launch_compose(o: &Options, root: &str) -> Result<String, String> {
    let override_path = format!("{root}/compose.json");
    write_override(&override_path, &o.service, root)?;
    let args = ["-f", o.file.as_str(), "-f", override_path.as_str()];
    let mut up_cmd = Vec::from(args);
    up_cmd.extend(["up", "-d", o.service.as_str()]);
    let status = Command::new("/usr/local/bin/podman-compose")
        .args(&up_cmd)
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status()
        .map_err(|_| String::from("compose did not confirm opted-in service creation"))?;
    if !status.success() {
        return Err(String::from(
            "compose did not confirm opted-in service creation",
        ));
    }
    let mut ps_cmd = Vec::from(args);
    ps_cmd.extend(["ps", "-q"]);
    let out = Command::new("/usr/local/bin/podman-compose")
        .args(&ps_cmd)
        .output()
        .map_err(|_| String::from("compose container identification failed"))?;
    if !out.status.success() {
        return Err(String::from("compose container identification failed"));
    }
    let text = String::from_utf8_lossy(&out.stdout).into_owned();
    select_compose_child(&text, &o.service, &|id| {
        let body = Command::new("/usr/bin/podman")
            .args([
                "--remote=false",
                "inspect",
                "--format",
                "{{.ID}} {{index .Config.Labels \"io.podman.compose.service\"}}",
                id,
            ])
            .output()
            .map_err(|e| e.to_string())?;
        Ok(String::from_utf8_lossy(&body.stdout).into_owned())
    })
}

pub(crate) fn write_override(path: &str, service: &str, root: &str) -> Result<(), String> {
    let sock_dir = Path::new(MUSE_LAUNCH_SOCKET)
        .parent()
        .and_then(|p| p.to_str())
        .unwrap_or("/run/soda-muse-interface");
    let mounts = [
        format!("{root}:/run/soda-muse/credentials:ro"),
        String::from("/usr/local/bin/muse:/usr/local/bin/muse:ro"),
        String::from("/usr/local/libexec/soda/muse:/usr/local/libexec/soda/muse:ro"),
        format!("{sock_dir}:{sock_dir}:ro"),
    ];
    let mut data = String::from("{\"services\":{");
    data.push_str(&json_string(service));
    data.push_str(":{\"volumes\":[");
    for (i, m) in mounts.iter().enumerate() {
        if i > 0 {
            data.push(',');
        }
        data.push_str(&json_string(m));
    }
    data.push_str("]}}}");
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true);
    opts.mode(0o600);
    use std::io::Write;
    let mut f = opts.open(path).map_err(|e| e.to_string())?;
    f.write_all(data.as_bytes()).map_err(|e| e.to_string())?;
    Ok(())
}

// Resolve only this Compose project's handles to native immutable IDs.
pub(crate) fn select_compose_child(
    output: &str,
    service: &str,
    inspect: &dyn Fn(&str) -> Result<String, String>,
) -> Result<String, String> {
    let mut child = String::new();
    for id in output.split_whitespace() {
        if !compose_container_handle(id) {
            return Err(String::from("invalid Compose container identity"));
        }
        let label = inspect(id).map_err(|_| String::from("compose service attribution failed"))?;
        let full = compose_observed_child(id, service, &label)?;
        if full.is_empty() {
            continue;
        }
        if !child.is_empty() {
            return Err(String::from(
                "exactly one immutable Compose container required",
            ));
        }
        child = full;
    }
    if child.is_empty() {
        return Err(String::from("requested Compose service container missing"));
    }
    Ok(child)
}

fn compose_container_handle(id: &str) -> bool {
    (id.len() == 12 || id.len() == 64) && is_hex(id)
}

fn compose_observed_child(handle: &str, service: &str, output: &str) -> Result<String, String> {
    let parts: Vec<&str> = output.split_whitespace().collect();
    if parts.len() != 2 || !immutable_compose_id(parts[0]) || !parts[0].starts_with(handle) {
        return Err(String::from(
            "native Compose container identity unconfirmed",
        ));
    }
    if parts[1] != service {
        return Ok(String::new());
    }
    Ok(parts[0].to_string())
}

fn immutable_compose_id(id: &str) -> bool {
    id.len() == 64 && is_hex(id)
}

fn is_hex(s: &str) -> bool {
    !s.is_empty() && s.bytes().all(|b| b.is_ascii_hexdigit())
}
