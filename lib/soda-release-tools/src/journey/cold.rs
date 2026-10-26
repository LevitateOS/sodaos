//! Explicit, stopped Docker cold sets. Failed outputs remain retained; neither
//! capture nor restore starts a writer, pulls an image, or retries an action.
use crate::qualification::input;
use clap::{Arg, ArgMatches, Command};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeMap,
    ffi::CString,
    fs,
    os::unix::{
        ffi::OsStrExt,
        fs::{DirBuilderExt, MetadataExt},
    },
    path::{Path, PathBuf},
    process::{Command as Process, Stdio},
};

const SETTINGS: [&str; 4] = ["compose.yaml", ".env", "identity.json", "docker.json"];
const DESTINATIONS: [&str; 3] = ["/etc", "/home", "/var"];
mod retirement;

pub(super) fn command() -> Command {
    let options = |command: Command, names: &[&'static str]| {
        names.iter().fold(command, |c, name| {
            c.arg(Arg::new(*name).long(*name).required(true).num_args(1))
        })
    };
    Command::new("cold")
        .about("Capture stopped Docker storage or prepare a stopped isolated restoration")
        .subcommand_required(true)
        .subcommand(options(
            Command::new("capture"),
            &["container", "project", "image", "settings", "out"],
        ))
        .subcommand(options(Command::new("restore"), &["source", "out"]))
        .subcommand(
            options(Command::new("retire"), &["restored", "result-sha256"]).arg(
                Arg::new("execute")
                    .long("execute")
                    .action(clap::ArgAction::SetTrue),
            ),
        )
}

pub(super) fn run(flags: &ArgMatches) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err("cold recovery requires root".into());
    }
    let (action, flags) = flags.subcommand().ok_or("cold action required")?;
    let get = |name| {
        flags
            .get_one::<String>(name)
            .map(String::as_str)
            .ok_or("cold argument required".to_owned())
    };
    let result = match action {
        "capture" => capture(
            get("container")?,
            get("project")?,
            get("image")?,
            Path::new(get("settings")?),
            Path::new(get("out")?),
        ),
        "restore" => restore(Path::new(get("source")?), Path::new(get("out")?)),
        "retire" => retirement::retire(
            Path::new(get("restored")?),
            get("result-sha256")?,
            flags.get_flag("execute"),
        ),
        _ => Err("unknown cold action".into()),
    }?;
    println!(
        "{}",
        serde_json::to_string(&result).map_err(|_| "cold receipt encoding failed")?
    );
    Ok(())
}

fn need(value: bool, reason: &str) -> Result<(), String> {
    if value {
        Ok(())
    } else {
        Err(reason.into())
    }
}
fn hex(value: &str, count: usize) -> bool {
    value.len() == count
        && value
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}
fn immutable(value: &str) -> bool {
    value.strip_prefix("sha256:").is_some_and(|s| hex(s, 64))
        || value.rsplit_once("@sha256:").is_some_and(|(p, s)| {
            !p.is_empty() && !p.chars().any(char::is_whitespace) && hex(s, 64)
        })
}
fn execute(program: &str, args: &[String]) -> Result<Vec<u8>, String> {
    let output = Process::new("/usr/bin/timeout")
        .args(["--signal=TERM", "--kill-after=10", "1800", program])
        .args(args)
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LC_ALL", "C")
        .stdin(Stdio::null())
        .output()
        .map_err(|_| "cold command unavailable")?;
    need(
        output.status.success() && output.stdout.len() <= 16 << 20,
        "cold command failed; retained output requires inspection",
    )?;
    Ok(output.stdout)
}
fn docker(args: &[String]) -> Result<Vec<u8>, String> {
    let mut selected = vec!["--host".into(), "unix:///var/run/docker.sock".into()];
    selected.extend_from_slice(args);
    execute("/usr/bin/docker", &selected)
}
fn json(bytes: &[u8]) -> Result<serde_json::Value, String> {
    serde_json::from_slice(bytes).map_err(|_| "cold Docker observation invalid".into())
}
fn private(path: &Path) -> Result<(), String> {
    soda_release_deliver::publish::qualification::protected_path(path, true)
        .map_err(|_| "root-private cold directory with trusted ancestors required".into())
}
fn fresh(path: &Path) -> Result<(), String> {
    need(
        path.is_absolute()
            // CoreOS exposes /home as a symlink; callers supply its canonical
            // home-storage path so all existing ancestor checks still apply.
            && (path.starts_with("/home") || path.starts_with("/var/home"))
            && !path.as_os_str().as_bytes().contains(&b',')
            && path.components().all(|p| {
                matches!(
                    p,
                    std::path::Component::RootDir | std::path::Component::Normal(_)
                )
            }),
        "canonical cold output under /home or /var/home required",
    )?;
    let parent = path.parent().ok_or("cold output parent required")?;
    for ancestor in parent.ancestors() {
        let meta = fs::symlink_metadata(ancestor).map_err(|_| "cold parent unavailable")?;
        need(
            meta.is_dir()
                && !meta.file_type().is_symlink()
                && meta.uid() == 0
                && meta.mode() & 0o022 == 0,
            "root-controlled cold parent required",
        )?;
    }
    fs::DirBuilder::new()
        .mode(0o700)
        .create(path)
        .map_err(|_| "fresh cold output required")?;
    private(path)
}
fn write<T: Serialize>(path: &Path, value: &T) -> Result<(), String> {
    input::write_new(
        path,
        &serde_json::to_vec_pretty(value).map_err(|_| "cold metadata encoding failed")?,
        0o600,
    )
}
fn digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
fn file_digest(path: &Path) -> Result<String, String> {
    crate::digest::hash_file(path.to_str().ok_or("cold UTF-8 path required")?)
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Entry {
    kind: String,
    mode: u32,
    uid: u32,
    gid: u32,
    modified: i64,
    modified_nanos: i64,
    bytes: u64,
    content: Option<String>,
    link: Option<String>,
    hardlink: Option<String>,
    device: u64,
    xattrs: BTreeMap<String, String>,
}
type Tree = BTreeMap<String, Entry>;
fn xattrs(path: &Path) -> Result<BTreeMap<String, String>, String> {
    let path = CString::new(path.as_os_str().as_bytes()).map_err(|_| "cold path invalid")?;
    let size = unsafe { libc::llistxattr(path.as_ptr(), std::ptr::null_mut(), 0) };
    if size < 0 {
        return Err("cold extended attributes unavailable".into());
    }
    need(size <= 1 << 20, "cold extended attributes exceed bound")?;
    let mut names = vec![0u8; size as usize];
    let actual = unsafe { libc::llistxattr(path.as_ptr(), names.as_mut_ptr().cast(), names.len()) };
    need(actual == size, "cold extended attributes changed")?;
    let mut result = BTreeMap::new();
    for name in names.split(|b| *b == 0).filter(|n| !n.is_empty()) {
        let text = std::str::from_utf8(name).map_err(|_| "cold attribute name invalid")?;
        let name = CString::new(name).map_err(|_| "cold attribute name invalid")?;
        let size =
            unsafe { libc::lgetxattr(path.as_ptr(), name.as_ptr(), std::ptr::null_mut(), 0) };
        need((0..=1 << 20).contains(&size), "cold attribute unavailable")?;
        let mut value = vec![0u8; size as usize];
        need(
            unsafe {
                libc::lgetxattr(
                    path.as_ptr(),
                    name.as_ptr(),
                    value.as_mut_ptr().cast(),
                    value.len(),
                )
            } == size,
            "cold attribute changed",
        )?;
        result.insert(text.to_owned(), digest(&value));
    }
    Ok(result)
}
fn inventory(root: &Path) -> Result<Tree, String> {
    fn walk(
        root: &Path,
        relative: &Path,
        tree: &mut Tree,
        links: &mut BTreeMap<(u64, u64), String>,
    ) -> Result<(), String> {
        need(tree.len() < 1_000_000, "cold inventory exceeds bound")?;
        let path = if relative.as_os_str().is_empty() {
            root.to_path_buf()
        } else {
            root.join(relative)
        };
        let meta = fs::symlink_metadata(&path).map_err(|_| "cold tree unavailable")?;
        let name = relative
            .to_str()
            .ok_or("cold UTF-8 paths required")?
            .to_owned();
        let kind = match meta.mode() & libc::S_IFMT {
            libc::S_IFREG => "file",
            libc::S_IFDIR => "directory",
            libc::S_IFLNK => "symlink",
            libc::S_IFCHR => "character",
            libc::S_IFBLK => "block",
            libc::S_IFIFO => "fifo",
            libc::S_IFSOCK => "socket",
            _ => return Err("cold file kind unsupported".into()),
        };
        let hardlink = if meta.nlink() > 1 && !meta.is_dir() {
            let first = links
                .entry((meta.dev(), meta.ino()))
                .or_insert_with(|| name.clone());
            Some(first.clone())
        } else {
            None
        };
        tree.insert(
            name,
            Entry {
                kind: kind.into(),
                mode: meta.mode(),
                uid: meta.uid(),
                gid: meta.gid(),
                modified: meta.mtime(),
                modified_nanos: meta.mtime_nsec(),
                bytes: if meta.is_file() { meta.len() } else { 0 },
                content: if meta.is_file() {
                    Some(file_digest(&path)?)
                } else {
                    None
                },
                link: if meta.file_type().is_symlink() {
                    Some(
                        fs::read_link(&path)
                            .map_err(|_| "cold link unavailable")?
                            .to_str()
                            .ok_or("cold UTF-8 link required")?
                            .to_owned(),
                    )
                } else {
                    None
                },
                hardlink,
                device: meta.rdev(),
                xattrs: xattrs(&path)?,
            },
        );
        if meta.is_dir() {
            let mut children = fs::read_dir(&path)
                .map_err(|_| "cold directory unavailable")?
                .collect::<Result<Vec<_>, _>>()
                .map_err(|_| "cold directory unavailable")?;
            children.sort_by_key(|e| e.file_name());
            for child in children {
                walk(root, &relative.join(child.file_name()), tree, links)?;
            }
        }
        Ok(())
    }
    let mut tree = Tree::new();
    walk(root, Path::new(""), &mut tree, &mut BTreeMap::new())?;
    Ok(tree)
}
fn copy_tree(source: &Path, out: &Path, expected: &Tree) -> Result<(), String> {
    need(
        !out.try_exists()
            .map_err(|_| "cold destination unavailable")?,
        "cold destination occupied",
    )?;
    execute(
        "/usr/bin/cp",
        &[
            "--archive".into(),
            "--reflink=auto".into(),
            "--sparse=always".into(),
            "--no-clobber".into(),
            "--".into(),
            source.to_str().ok_or("cold UTF-8 path required")?.into(),
            out.to_str().ok_or("cold UTF-8 path required")?.into(),
        ],
    )?;
    need(
        inventory(out)? == *expected,
        "cold copied bytes or metadata differ",
    )?;
    execute(
        "/usr/bin/sync",
        &[
            "--file-system".into(),
            out.to_str().ok_or("cold UTF-8 path required")?.into(),
        ],
    )?;
    Ok(())
}

#[derive(Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Volume {
    name: String,
    destination: String,
    mountpoint: PathBuf,
    compose_name: String,
}
#[derive(Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Source {
    container: String,
    project: String,
    image: String,
    engine_image: String,
    entrypoint: Vec<String>,
    command: Vec<String>,
    environment: Vec<String>,
    volumes: Vec<Volume>,
    started_at: String,
    finished_at: String,
}
fn strings(value: &serde_json::Value) -> Result<Vec<String>, String> {
    if value.is_null() {
        return Ok(vec![]);
    }
    value
        .as_array()
        .ok_or("cold command array required")?
        .iter()
        .map(|v| {
            v.as_str()
                .map(String::from)
                .ok_or("cold command string required".into())
        })
        .collect()
}
fn inspect(container: &str, project: &str, image: &str) -> Result<Source, String> {
    need(
        hex(container, 64)
            && !project.is_empty()
            && project.len() <= 63
            && project
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b"_-".contains(&b))
            && immutable(image),
        "exact cold container/project/immutable image required",
    )?;
    let observed = json(&docker(&[
        "container".into(),
        "inspect".into(),
        container.into(),
    ])?)?;
    let value = observed
        .as_array()
        .filter(|a| a.len() == 1)
        .and_then(|a| a.first())
        .ok_or("one cold container required")?;
    need(
        value["Id"] == container
            && value["Config"]["Image"] == image
            && value["State"]["Status"] == "exited"
            && value["State"]["Running"] == false
            && value["State"]["Paused"] == false
            && value["State"]["OOMKilled"] == false
            && [Some(0), Some(130)].contains(&value["State"]["ExitCode"].as_i64())
            && value["Config"]["Labels"]["com.docker.compose.project"] == project
            && value["Config"]["Labels"]["com.docker.compose.service"] == "soda",
        "source must be exact cleanly stopped Compose soda container",
    )?;
    let engine_image = value["Image"]
        .as_str()
        .ok_or("cold image identity missing")?
        .to_owned();
    let local_image = String::from_utf8(docker(&[
        "image".into(),
        "inspect".into(),
        "--format".into(),
        "{{.Id}}".into(),
        image.into(),
    ])?)
    .map_err(|_| "cold image response invalid")?;
    need(
        local_image.trim() == engine_image,
        "cold image does not match retained local image",
    )?;
    let mounts = value["Mounts"].as_array().ok_or("cold mounts missing")?;
    need(
        mounts.len() == 3,
        "cold source requires exactly three volumes and no external binds",
    )?;
    let mut volumes = Vec::new();
    for destination in DESTINATIONS {
        let mount = mounts
            .iter()
            .find(|m| m["Destination"] == destination)
            .ok_or("cold retained volume missing")?;
        need(
            mount["Type"] == "volume" && mount["RW"] == true,
            "cold named writable volume required",
        )?;
        let name = mount["Name"].as_str().ok_or("cold volume name missing")?;
        let records = json(&docker(&["volume".into(), "inspect".into(), name.into()])?)?;
        let volume = records
            .as_array()
            .filter(|a| a.len() == 1)
            .and_then(|a| a.first())
            .ok_or("one cold volume required")?;
        let compose_name = volume["Labels"]["com.docker.compose.volume"]
            .as_str()
            .ok_or("cold Compose volume label missing")?;
        need(
            volume["Name"] == name
                && volume["Driver"] == "local"
                && (volume["Options"].is_null()
                    || volume["Options"].as_object().is_some_and(|o| o.is_empty()))
                && volume["Labels"]["com.docker.compose.project"] == project
                && !compose_name.is_empty(),
            "cold volume ownership changed",
        )?;
        let mountpoint = PathBuf::from(
            volume["Mountpoint"]
                .as_str()
                .ok_or("cold volume path missing")?,
        );
        need(
            mountpoint.is_absolute()
                && fs::canonicalize(&mountpoint).map_err(|_| "cold volume unavailable")?
                    == mountpoint
                && mount["Source"] == mountpoint.to_str().ok_or("cold volume path invalid")?,
            "cold volume path changed",
        )?;
        volumes.push(Volume {
            name: name.into(),
            destination: destination.into(),
            mountpoint,
            compose_name: compose_name.into(),
        });
    }
    let running = String::from_utf8(docker(&[
        "container".into(),
        "ls".into(),
        "--quiet".into(),
        "--no-trunc".into(),
    ])?)
    .map_err(|_| "cold writer inventory invalid")?;
    for id in running.lines().filter(|s| !s.is_empty()) {
        let records = json(&docker(&["container".into(), "inspect".into(), id.into()])?)?;
        for mount in records[0]["Mounts"]
            .as_array()
            .ok_or("cold writer mounts unavailable")?
        {
            let source = Path::new(
                mount["Source"]
                    .as_str()
                    .ok_or("cold writer mount unavailable")?,
            );
            need(
                !volumes
                    .iter()
                    .any(|v| source.starts_with(&v.mountpoint) || v.mountpoint.starts_with(source)),
                "another container writes the cold set",
            )?;
        }
    }
    Ok(Source {
        container: container.into(),
        project: project.into(),
        image: image.into(),
        engine_image,
        entrypoint: strings(&value["Config"]["Entrypoint"])?,
        command: strings(&value["Config"]["Cmd"])?,
        environment: strings(&value["Config"]["Env"])?,
        volumes,
        started_at: value["State"]["StartedAt"]
            .as_str()
            .ok_or("cold start identity missing")?
            .into(),
        finished_at: value["State"]["FinishedAt"]
            .as_str()
            .ok_or("cold stop identity missing")?
            .into(),
    })
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Seal {
    format: u8,
    source: Source,
    trees: BTreeMap<String, Tree>,
    settings: Tree,
}
fn settings_inventory(settings: &Path) -> Result<Tree, String> {
    private(settings)?;
    let mut tree = Tree::new();
    for name in SETTINGS {
        let path = settings.join(name);
        let meta = fs::symlink_metadata(&path).map_err(|_| "cold deployment setting missing")?;
        need(
            meta.is_file()
                && !meta.file_type().is_symlink()
                && meta.uid() == 0
                && meta.mode() & 0o077 == 0
                && meta.nlink() == 1,
            "cold deployment setting must be private root file",
        )?;
        let mut entry = inventory(&path)?;
        tree.insert(
            name.into(),
            entry.remove("").ok_or("cold setting identity missing")?,
        );
    }
    Ok(tree)
}
fn copy_settings(source: &Path, out: &Path, expected: &Tree) -> Result<(), String> {
    fs::DirBuilder::new()
        .mode(0o700)
        .create(out)
        .map_err(|_| "fresh cold settings required")?;
    for name in SETTINGS {
        let mut one = Tree::new();
        one.insert(
            "".into(),
            expected.get(name).ok_or("cold setting missing")?.clone(),
        );
        copy_tree(&source.join(name), &out.join(name), &one)?;
    }
    need(
        settings_inventory(out)? == *expected,
        "cold copied settings differ",
    )
}
fn capture(
    container: &str,
    project: &str,
    image: &str,
    settings: &Path,
    out: &Path,
) -> Result<serde_json::Value, String> {
    let source = inspect(container, project, image)?;
    let settings_tree = settings_inventory(settings)?;
    fresh(out)?;
    write(
        &out.join("request.json"),
        &serde_json::json!({"format":1,"action":"cold-capture","container":container,"project":project,"image":image,"automatic_retry":false}),
    )?;
    let mut trees = BTreeMap::new();
    for volume in &source.volumes {
        let name = volume.destination.trim_start_matches('/');
        let tree = inventory(&volume.mountpoint)?;
        copy_tree(&volume.mountpoint, &out.join(name), &tree)?;
        need(
            inventory(&volume.mountpoint)? == tree,
            "cold source changed during capture",
        )?;
        trees.insert(name.into(), tree);
    }
    copy_settings(settings, &out.join("settings"), &settings_tree)?;
    for volume in &source.volumes {
        let name = volume.destination.trim_start_matches('/');
        need(
            inventory(&volume.mountpoint)? == trees[name],
            "cold source changed before sealing",
        )?;
    }
    need(
        inspect(container, project, image)? == source
            && settings_inventory(settings)? == settings_tree,
        "cold source identity changed during capture",
    )?;
    let seal = Seal {
        format: 1,
        source,
        trees,
        settings: settings_tree,
    };
    write(&out.join("complete.json"), &seal)?;
    Ok(
        serde_json::json!({"format":1,"action":"cold-capture","complete":true,"directory":out,"container":container,"seal_sha256":file_digest(&out.join("complete.json"))?,"writers_started":false,"automatic_retry":false}),
    )
}
fn restore(source: &Path, out: &Path) -> Result<serde_json::Value, String> {
    private(source)?;
    soda_release_deliver::publish::qualification::protected_path(
        &source.join("complete.json"),
        false,
    )
    .map_err(|_| "completed protected cold set required")?;
    let seal: Seal = serde_json::from_slice(
        &fs::read(source.join("complete.json")).map_err(|_| "cold seal unavailable")?,
    )
    .map_err(|_| "cold seal invalid")?;
    need(
        seal.format == 1
            && immutable(&seal.source.image)
            && seal.source.volumes.len() == 3
            && seal.trees.len() == 3,
        "cold seal scope invalid",
    )?;
    for destination in DESTINATIONS {
        let name = destination.trim_start_matches('/');
        need(
            seal.trees.get(name).is_some_and(|tree| {
                inventory(&source.join(name)).is_ok_and(|actual| actual == *tree)
            }),
            "cold set bytes or metadata changed",
        )?;
    }
    need(
        settings_inventory(&source.join("settings"))? == seal.settings,
        "cold settings changed",
    )?;
    fresh(out)?;
    write(
        &out.join("request.json"),
        &serde_json::json!({"format":1,"action":"isolated-cold-restore","source":source,"seal_sha256":file_digest(&source.join("complete.json"))?,"automatic_retry":false}),
    )?;
    for destination in DESTINATIONS {
        let name = destination.trim_start_matches('/');
        copy_tree(&source.join(name), &out.join(name), &seal.trees[name])?;
    }
    copy_settings(
        &source.join("settings"),
        &out.join("settings"),
        &seal.settings,
    )?;
    // Keep values in a private file instead of process arguments or receipts.
    need(
        seal.source.environment.iter().all(|s| {
            !s.contains(['\n', '\r']) && s.split_once('=').is_some_and(|(k, _)| !k.is_empty())
        }),
        "cold environment invalid",
    )?;
    input::write_new(
        &out.join("container.env"),
        &(seal.source.environment.join("\n") + "\n").into_bytes(),
        0o600,
    )?;
    let mut args = vec![
        "container".into(),
        "create".into(),
        "--pull".into(),
        "never".into(),
        "--network".into(),
        "none".into(),
        "--privileged".into(),
        "--userns".into(),
        "host".into(),
        "--cgroupns".into(),
        "private".into(),
        "--tmpfs".into(),
        "/run".into(),
        "--tmpfs".into(),
        "/tmp".into(),
        "--env-file".into(),
        out.join("container.env")
            .to_str()
            .ok_or("cold environment path invalid")?
            .into(),
        "--label".into(),
        "soda.qualification.cold=true".into(),
    ];
    for destination in DESTINATIONS {
        args.extend([
            "--mount".into(),
            format!(
                "type=bind,source={},destination={destination}",
                out.join(destination.trim_start_matches('/')).display()
            ),
        ]);
    }
    if let Some(entrypoint) = seal.source.entrypoint.first() {
        args.extend(["--entrypoint".into(), entrypoint.clone()]);
    } else {
        args.extend(["--entrypoint".into(), "".into()]);
    }
    args.push(seal.source.image.clone());
    args.extend(seal.source.entrypoint.iter().skip(1).cloned());
    args.extend(seal.source.command.iter().cloned());
    write(
        &out.join("create.request.json"),
        &serde_json::json!({"format":1,"action":"docker-create-isolated-stopped","image":seal.source.image,"network":"none","automatic_retry":false}),
    )?;
    let container = String::from_utf8(docker(&args)?)
        .map_err(|_| "cold container response invalid")?
        .trim()
        .to_owned();
    need(
        hex(&container, 64),
        "cold created container identity missing",
    )?;
    let observed = json(&docker(&[
        "container".into(),
        "inspect".into(),
        container.clone(),
    ])?)?;
    let value = &observed[0];
    need(
        value["Id"] == container
            && value["Image"] == seal.source.engine_image
            && value["Config"]["Image"] == seal.source.image
            && value["State"]["Status"] == "created"
            && value["State"]["Running"] == false
            && value["HostConfig"]["NetworkMode"] == "none"
            && value["HostConfig"]["Privileged"] == true
            && value["HostConfig"]["CgroupnsMode"] == "private",
        "cold stopped isolation unconfirmed",
    )?;
    let mounts = value["Mounts"]
        .as_array()
        .ok_or("cold restored mounts missing")?;
    need(
        mounts.len() == 3
            && DESTINATIONS.iter().all(|destination| {
                mounts.iter().any(|m| {
                    m["Type"] == "bind"
                        && m["Destination"] == *destination
                        && m["Source"]
                            == out
                                .join(destination.trim_start_matches('/'))
                                .to_str()
                                .unwrap_or("")
                })
            }),
        "cold restored storage bindings differ",
    )?;
    for destination in DESTINATIONS {
        let name = destination.trim_start_matches('/');
        need(
            inventory(&source.join(name))? == seal.trees[name]
                && inventory(&out.join(name))? == seal.trees[name],
            "cold source or restored tree changed",
        )?;
    }
    let receipt = serde_json::json!({"format":1,"action":"isolated-cold-restore","complete":true,"container":container,"directory":out,"image":seal.source.image,"network":"none","state":"created","writers_started":false,"source_untouched":true,"automatic_retry":false});
    write(&out.join("complete.json"), &receipt)?;
    Ok(receipt)
}
