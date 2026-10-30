//! Build the working tree and activate a development overlay over SSH.
use std::fs;
use std::io::{Read, Write};
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

use flate2::{write::GzEncoder, Compression};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use soda_release_assets::{
    locales::merge,
    render::{source_root, stage},
};

#[path = "../dev_remote.rs"]
mod remote;
#[path = "../dev_ui.rs"]
mod ui;

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

const HELP: &str = "usage: soda-dev [ui|watch-ui|deploy|build|status|disable] [options]

Build local Soda Forge changes and activate them on a retained SodaOS host.
  --host NAME          SSH host (default: soda; root login required)
  --ssh-config FILE    SSH config (default: ~/.ssh/config when present)
  --work-dir DIR       Local builds/cache (default: ../.soda-dev)
  --fountain-bin FILE  Compatible host executable (default: remembered, then cached)
  --sdk-dir DIR        Matching SDK (default: remembered, then cached source SDK)

ui updates only extension browser assets in the active development deployment.
watch-ui repeats that update when browser sources change. Refresh to see edits.
Neither command builds Go, changes the database, or restarts services.
deploy builds, uploads, activates and checks health; build only stages locally.
status inspects the host; disable removes the owned development overlay.
Successful full deployments remember their host executable and SDK per SSH host.
The working tree is deployed, including uncommitted source changes.";

struct Args {
    action: String,
    host: String,
    ssh_config: Option<PathBuf>,
    work: PathBuf,
    fountain: Option<PathBuf>,
    sdk: Option<PathBuf>,
}

fn args(source: &Path) -> Result<Args> {
    let mut result = Args {
        action: "deploy".into(),
        host: "soda".into(),
        ssh_config: std::env::var_os("HOME")
            .map(PathBuf::from)
            .map(|p| p.join(".ssh/config"))
            .filter(|p| p.is_file()),
        work: source
            .parent()
            .ok_or("source parent missing")?
            .join(".soda-dev"),
        fountain: None,
        sdk: None,
    };
    let mut tokens = std::env::args().skip(1).peekable();
    if tokens.peek().is_some_and(|s| !s.starts_with('-')) {
        result.action = tokens.next().unwrap();
    }
    while let Some(token) = tokens.next() {
        if token == "--help" || token == "-h" {
            println!("{HELP}");
            std::process::exit(0);
        }
        let (name, inline) = token
            .split_once('=')
            .map_or((token.as_str(), None), |(k, v)| (k, Some(v.to_string())));
        let value = inline
            .or_else(|| tokens.next())
            .ok_or_else(|| format!("{name} needs a value"))?;
        match name {
            "--host" => result.host = value,
            "--ssh-config" => result.ssh_config = Some(PathBuf::from(value)),
            "--work-dir" => result.work = PathBuf::from(value),
            "--fountain-bin" => result.fountain = Some(PathBuf::from(value)),
            "--sdk-dir" => result.sdk = Some(PathBuf::from(value)),
            _ => return Err(format!("unknown option {name}\n{HELP}").into()),
        }
    }
    if !["ui", "watch-ui", "deploy", "build", "status", "disable"].contains(&result.action.as_str())
    {
        return Err(HELP.into());
    }
    if result.host.starts_with('-')
        || result.host.is_empty()
        || !result
            .host
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b"@._-".contains(&b))
    {
        return Err("invalid SSH host".into());
    }
    fs::create_dir_all(&result.work)?;
    result.work = result.work.canonicalize()?;
    if !result.work.starts_with("/home") {
        return Err("development builds must be under /home".into());
    }
    let saved = result
        .work
        .join(format!("deploy-inputs-{}.json", result.host));
    if saved.is_file() {
        let saved: Value = serde_json::from_slice(&fs::read(saved)?)?;
        if matches!(result.action.as_str(), "deploy" | "build") {
            if result.fountain.is_none() {
                result.fountain = saved["fountain"].as_str().map(PathBuf::from);
            }
            if result.sdk.is_none() {
                result.sdk = saved["sdk"].as_str().map(PathBuf::from);
            }
        }
    }
    if matches!(result.action.as_str(), "deploy" | "build") {
        if let Some(path) = &result.fountain {
            if !path.is_file() {
                return Err(format!(
                    "host executable missing at {}; supply --fountain-bin",
                    path.display()
                )
                .into());
            }
        }
        if let Some(path) = &result.sdk {
            if !path.join("go.mod").is_file() {
                return Err(
                    format!("host SDK missing at {}; supply --sdk-dir", path.display()).into(),
                );
            }
        }
    }
    Ok(result)
}

fn run(command: &mut Command) -> Result<()> {
    let status = command.status()?;
    if !status.success() {
        return Err(format!(
            "{} failed: {status}",
            command.get_program().to_string_lossy()
        )
        .into());
    }
    Ok(())
}

fn ssh(args: &Args) -> Command {
    let mut command = Command::new("ssh");
    if let Some(config) = &args.ssh_config {
        command.arg("-F").arg(config);
    }
    command.args(["-o", "BatchMode=yes", "-o", "ConnectTimeout=10", &args.host]);
    command
}

fn remote_script(args: &Args, script: &str) -> Result<()> {
    let mut child = ssh(args).arg("bash -se").stdin(Stdio::piped()).spawn()?;
    child
        .stdin
        .take()
        .ok_or("SSH stdin missing")?
        .write_all(script.as_bytes())?;
    if !child.wait()?.success() {
        return Err("remote operation failed; inspect the health/status output".into());
    }
    Ok(())
}

fn digest(path: &Path) -> Result<String> {
    let mut hash = Sha256::new();
    let mut file = fs::File::open(path)?;
    let mut buffer = [0u8; 65536];
    loop {
        let n = file.read(&mut buffer)?;
        if n == 0 {
            break;
        }
        hash.update(&buffer[..n]);
    }
    Ok(format!("{:x}", hash.finalize()))
}

fn files(root: &Path, directory: &Path, output: &mut Vec<PathBuf>) -> Result<()> {
    for entry in fs::read_dir(directory)? {
        let entry = entry?;
        if entry.file_type()?.is_dir() {
            files(root, &entry.path(), output)?;
        } else if entry.file_type()?.is_file() {
            output.push(entry.path().strip_prefix(root)?.to_path_buf());
        }
    }
    output.sort();
    Ok(())
}

// Native assets otherwise keep the source's presentation epoch across edits.
// Change only staged URLs and rebuild the production-format hash inventory.
fn version_presentation(context: &Path, id: &str) -> Result<()> {
    let inventory_path = context.join("presentation.json");
    let mut inventory: std::collections::BTreeMap<String, String> =
        serde_json::from_slice(&fs::read(&inventory_path)?)?;
    for (name, hash) in &mut inventory {
        let path = context.join("forgejo").join(name);
        if ["tmpl", "js", "css"].contains(&path.extension().and_then(|s| s.to_str()).unwrap_or(""))
        {
            let text = fs::read_to_string(&path)?;
            let mut output = String::new();
            let mut rest = text.as_str();
            while let Some(start) = rest.find("?v=") {
                output.push_str(&rest[..start + 3]);
                rest = &rest[start + 3..];
                let end = rest
                    .find(|c: char| !c.is_ascii_alphanumeric() && c != '.' && c != '-')
                    .unwrap_or(rest.len());
                if end == 0 {
                    continue;
                }
                output.push_str(id);
                rest = &rest[end..];
            }
            output.push_str(rest);
            if name == "templates/custom/header.tmpl" {
                let prefix = "name=\"soda-presentation-revision\" content=\"";
                if let Some(start) = output.find(prefix) {
                    let start = start + prefix.len();
                    if let Some(end) = output[start..].find('"') {
                        output.replace_range(start..start + end, id);
                    }
                }
            }
            fs::write(&path, output)?;
        }
        *hash = digest(&path)?;
    }
    fs::write(inventory_path, serde_json::to_vec_pretty(&inventory)?)?;
    Ok(())
}

fn package(source: &Path, build: &Path, context: &Path, id: &str) -> Result<()> {
    let manifest_path = "system/containers/extension/extension.json";
    let mut manifest: Value = serde_json::from_slice(&fs::read(source.join(manifest_path))?)?;
    let version = format!("0.0.0-dev.{id}");
    manifest["version"] = json!(version);
    let package_root = build.join("package");
    fs::create_dir(&package_root)?;
    fs::write(
        package_root.join("extension.json"),
        serde_json::to_vec_pretty(&manifest)?,
    )?;
    fs::copy(
        build.join("bin/soda-extension"),
        package_root.join("backend"),
    )?;
    fs::copy(
        source.join("system/containers/extension/run"),
        package_root.join("run"),
    )?;
    for name in ["LICENSE", "NOTICE"] {
        fs::copy(source.join(name), package_root.join(name))?;
    }
    run(Command::new("cp")
        .args(["-R"])
        .arg(build.join("extension/assets"))
        .arg(package_root.join("assets")))?;
    let archive_path = context.join("soda.tar.gz");
    let encoder = GzEncoder::new(fs::File::create(&archive_path)?, Compression::fast());
    let mut archive = tar::Builder::new(encoder);
    let mut entries = Vec::new();
    files(&package_root, &package_root, &mut entries)?;
    for name in entries {
        let path = package_root.join(&name);
        let mut header = tar::Header::new_ustar();
        header.set_size(fs::metadata(&path)?.len());
        header.set_mode(
            if name == Path::new("backend") || name == Path::new("run") {
                0o755
            } else {
                0o644
            },
        );
        header.set_uid(0);
        header.set_gid(0);
        header.set_mtime(0);
        header.set_entry_type(tar::EntryType::Regular);
        header.set_cksum();
        archive.append_data(&mut header, name, fs::File::open(path)?)?;
    }
    archive.into_inner()?.finish()?;
    let sha = digest(&archive_path)?;
    fs::write(context.join("package.sha256"), format!("{sha}\n"))?;
    let catalog = json!({"version":1,"packages":[{"id":"soda","version":version,
        "url":"https://soda-dev.invalid/soda.tar.gz","size":fs::metadata(&archive_path)?.len(),
        "sha256":sha,"required":true,"fountain_protocol":manifest["protocol"],
        "browser_contract":1,"capabilities":manifest["capabilities"]}]});
    fs::write(
        context.join("catalog.json"),
        serde_json::to_vec_pretty(&catalog)?,
    )?;
    Ok(())
}

fn build(source: &Path, args: &Args) -> Result<(PathBuf, String)> {
    let id = format!(
        "dev-{}",
        SystemTime::now().duration_since(UNIX_EPOCH)?.as_millis()
    );
    let release = args.work.join("releases").join(&id);
    let build = release.join("build");
    let context = release.join("payload");
    fs::create_dir_all(build.join("bin"))?;
    fs::create_dir_all(args.work.join("tmp"))?;
    let pin: Value = serde_json::from_slice(&fs::read(source.join("fountain-source.json"))?)?;
    let revision = pin["revision"]
        .as_str()
        .ok_or("Fountain source revision missing")?;
    let fountain_source = args
        .work
        .join("inputs")
        .join(format!("fountain-source-{}", &revision[..7]));
    let sdk = args
        .sdk
        .clone()
        .or_else(|| Some(fountain_source.join("sdk")).filter(|p| p.is_dir()));
    let mut go = Command::new("go");
    go.current_dir(source)
        .env("CGO_ENABLED", "0")
        .env("GOOS", "linux")
        .env("GOARCH", "amd64")
        .env("TMPDIR", args.work.join("tmp"))
        .args([
            "build",
            "-mod=readonly",
            "-trimpath",
            "-buildvcs=false",
            "-ldflags=-s -w",
        ]);
    if let Some(sdk) = sdk {
        let sdk = sdk.canonicalize()?;
        let original = fs::read_to_string(source.join("go.mod"))?;
        let modfile = build.join("go.mod");
        fs::write(
            &modfile,
            original.replace(
                "replace forgejo.org/extension-sdk => ../forgejo-ext/sdk",
                &format!("replace forgejo.org/extension-sdk => {}", sdk.display()),
            ),
        )?;
        fs::copy(source.join("go.sum"), build.join("go.sum"))?;
        go.arg(format!("-modfile={}", modfile.display()));
    }
    println!("Building dashboard and extension backend");
    run(go.arg("-o").arg(build.join("bin")).args([
        "./cmd/soda-dashboard",
        "./cmd/soda-extension",
        "./tools/dev-db",
    ]))?;
    println!("Building native presentation and extension browser assets");
    let terminal = source.join(".artifacts/browser-terminal/vendor");
    soda_release_assets::fetch::terminal::fetch(
        &source.join("tools/release-assets/terminal-assets.lock.json"),
        &terminal,
    )?;
    run(Command::new("bun")
        .current_dir(source)
        .arg("scripts/build-forgejo.ts")
        .arg("--out")
        .arg(build.join("forgejo-js")))?;
    run(Command::new("bun")
        .current_dir(source)
        .args(["scripts/build-soda-extension.ts", "--package=soda", "--out"])
        .arg(build.join("extension"))
        .arg("--terminal-assets")
        .arg(terminal))?;
    let lock: Value =
        serde_json::from_slice(&fs::read(source.join("frontend/forgejo/locale.lock.json"))?)?;
    let locale = fountain_source.join("options/locale/locale_en-US.ini");
    let native = if locale.is_file() && digest(&locale)? == lock["sha256"].as_str().unwrap_or("") {
        merge::Native::File(locale)
    } else {
        let sha = lock["sha256"].as_str().ok_or("native locale SHA missing")?;
        let cached = args.work.join("inputs").join(format!("locale-{sha}.ini"));
        if !cached.is_file() || digest(&cached)? != sha {
            let url = lock["url"].as_str().ok_or("native locale URL missing")?;
            let bytes = merge::fetch_locked(url, sha).map_err(|e| e.message().to_string())?;
            fs::create_dir_all(cached.parent().unwrap())?;
            fs::write(&cached, bytes)?;
        }
        merge::Native::File(cached)
    };
    merge::run(
        &native,
        &source.join("frontend/forgejo/locales/en-US.ini"),
        &build.join("forgejo-locales/locale_en-US.ini"),
    )
    .map_err(|e| e.message().to_string())?;
    stage::stage_dev_forgejo(source, &build, &context).map_err(|e| match e {
        stage::StageError::Refusal(s) | stage::StageError::Failure(s) => s,
    })?;
    version_presentation(&context, &id)?;
    fs::create_dir(context.join("bin"))?;
    let schema = Command::new(build.join("bin/dev-db")).output()?;
    if !schema.status.success() {
        return Err(String::from_utf8_lossy(&schema.stderr).into_owned().into());
    }
    fs::write(context.join("dev-schema.sql"), schema.stdout)?;
    fs::copy(
        build.join("bin/soda-dashboard"),
        context.join("bin/soda-dashboard"),
    )?;
    let fountain = args
        .fountain
        .clone()
        .or_else(|| Some(args.work.join("inputs/fountain")).filter(|p| p.is_file()));
    if let Some(fountain) = fountain {
        fs::copy(fountain, context.join("bin/fountain"))?;
    }
    for entry in fs::read_dir(context.join("bin"))? {
        fs::set_permissions(entry?.path(), fs::Permissions::from_mode(0o755))?;
    }
    package(source, &build, &context, &id)?;
    let mut entries = Vec::new();
    files(&context, &context, &mut entries)?;
    let mut checksums = String::new();
    for entry in entries {
        checksums.push_str(&format!(
            "{}  {}\n",
            digest(&context.join(&entry))?,
            entry.display()
        ));
    }
    fs::write(context.join("checksums.sha256"), checksums)?;
    println!("Built {}", context.display());
    Ok((context, id))
}

fn main() {
    if let Err(error) = execute() {
        eprintln!("soda-dev: {error}");
        std::process::exit(1);
    }
}

fn execute() -> Result<()> {
    let source = source_root()?;
    let args = args(&source)?;
    match args.action.as_str() {
        "ui" => ui::deploy(&source, &args),
        "watch-ui" => ui::watch(&source, &args),
        "status" => remote_script(&args, &remote::status_script()),
        "disable" => remote_script(&args, &remote::disable_script()),
        _ => {
            let (context, id) = build(&source, &args)?;
            if args.action == "build" {
                return Ok(());
            }
            let destination = format!("/var/home/soda-dev/releases/{id}");
            // Send the archive on stdin; paths never pass through local shell interpolation.
            let archive = context.parent().unwrap().join("upload.tar");
            run(Command::new("tar")
                .arg("-cf")
                .arg(&archive)
                .arg("-C")
                .arg(&context)
                .arg("."))?;
            println!("Uploading to {}:{destination}", args.host);
            run(ssh(&args)
                .arg(format!(
                    "umask 022\nmkdir -p '{destination}'\ntar -xf - -C '{destination}'"
                ))
                .stdin(Stdio::from(fs::File::open(archive)?)))?;
            remote_script(&args, &remote::activate_script(&destination))?;
            // Retain the actual working inputs, rather than silently selecting
            // the cached release SDK/host again on the next small change.
            let sdk = args.sdk.clone().or_else(|| {
                let text = fs::read_to_string(context.parent()?.join("build/go.mod")).ok()?;
                text.lines().find_map(|line| {
                    line.strip_prefix("replace forgejo.org/extension-sdk => ")
                        .map(PathBuf::from)
                })
            });
            let fountain = context.join("bin/fountain");
            fs::write(
                args.work.join(format!("deploy-inputs-{}.json", args.host)),
                serde_json::to_vec_pretty(&json!({
                    "fountain": fountain.is_file().then(|| fountain.to_string_lossy().into_owned()),
                    "sdk": sdk
                }))?,
            )?;
            println!(
                "Development Forge active on {}. Refresh the browser to test changes.",
                args.host
            );
            Ok(())
        }
    }
}
