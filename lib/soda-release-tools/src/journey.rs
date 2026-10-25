//! Read-only installed-journey operator. The driver owns scenario validation
//! and observed business state; Rust retains only exact invocation inputs.
use crate::qualification::input::{self, FileReference};
use clap::{Arg, ArgMatches, Command};
use serde::{Deserialize, Serialize};
use std::{
    fs,
    os::unix::fs::{DirBuilderExt, MetadataExt},
    path::{Path, PathBuf},
    process::Stdio,
};

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Request {
    format: u8,
    driver: FileReference,
    scenario: FileReference,
}

pub(crate) fn command() -> Command {
    Command::new("journey")
        .about("Inspect installed qualification readiness without target mutation")
        .subcommand_required(true)
        .subcommand(["scenario", "out"].into_iter().fold(
            Command::new("check").about("Snapshot scenario inputs and inspect readiness"),
            |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
        ))
        .subcommand(
            Command::new("status")
                .about("Report the last read-only check using its retained scenario and driver")
                .arg(Arg::new("run").long("run").required(true).num_args(1)),
        )
}

pub(crate) fn run(flags: &ArgMatches) -> Result<(), String> {
    let (action, flags) = flags.subcommand().ok_or("journey action required")?;
    let path = |name| {
        flags
            .get_one::<String>(name)
            .map(Path::new)
            .ok_or_else(|| format!("{name} required"))
    };
    match action {
        "check" => check(path("scenario")?, path("out")?),
        "status" => status(path("run")?),
        _ => Err("unknown journey action".into()),
    }
}

fn canonical(path: &Path) -> Result<PathBuf, String> {
    fs::canonicalize(path).map_err(|_| "journey path unavailable".into())
}

fn hash(path: &Path) -> Result<String, String> {
    crate::digest::hash_file(path.to_str().ok_or("UTF-8 journey path required")?)
}

fn read(path: &Path) -> Result<serde_json::Value, String> {
    serde_json::from_slice(&fs::read(path).map_err(|_| "journey input unavailable")?)
        .map_err(|_| "journey JSON document required".into())
}

fn driver(scenario: &Path) -> Result<FileReference, String> {
    let value = read(scenario)?;
    let reference: FileReference = serde_json::from_value(
        value
            .pointer("/driver/executable")
            .ok_or("scenario driver executable required")?
            .clone(),
    )
    .map_err(|_| "scenario driver executable path and sha256 required")?;
    if !reference.path.is_absolute() {
        return Err("absolute journey driver path required".into());
    }
    verify_driver(&reference)?;
    Ok(reference)
}

fn verify_driver(driver: &FileReference) -> Result<(), String> {
    let meta = fs::symlink_metadata(&driver.path).map_err(|_| "journey driver unavailable")?;
    if !meta.is_file()
        || meta.file_type().is_symlink()
        || meta.nlink() != 1
        || meta.mode() & 0o111 == 0
        || !crate::digest::is_digest(&driver.sha256)
        || hash(&driver.path)? != driver.sha256
    {
        return Err("journey executable identity changed or executable file required".into());
    }
    Ok(())
}

fn private_run(path: &Path) -> Result<PathBuf, String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "journey run unavailable")?;
    let path = canonical(path)?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.mode() & 0o7777 != 0o700
        || meta.uid() != unsafe { libc::geteuid() }
        || !path.starts_with("/home")
    {
        return Err("owned private journey run directory under /home required".into());
    }
    Ok(path)
}

fn check(scenario: &Path, out: &Path) -> Result<(), String> {
    let scenario = canonical(scenario)?;
    let source = FileReference {
        sha256: hash(&scenario)?,
        path: scenario,
    };
    // Validate the one invocation field before allocating a run. The driver
    // validates all remaining scope and discovers current runtime facts.
    driver(&source.path)?;
    let parent = out.parent().ok_or("journey output parent required")?;
    let parent = canonical(if parent.as_os_str().is_empty() {
        Path::new(".")
    } else {
        parent
    })?;
    if !parent.starts_with("/home") {
        return Err("journey output under /home required".into());
    }
    let out = parent.join(out.file_name().ok_or("journey output name required")?);
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&out)
        .map_err(|_| "fresh journey output directory required")?;
    let out = private_run(&out)?;
    let scenario = input::snapshot(&source, &out.join("scenario.json"))?;
    let request = Request {
        format: 1,
        driver: driver(&scenario.path)?,
        scenario,
    };
    input::write_new(
        &out.join("request.json"),
        &serde_json::to_vec_pretty(&request).map_err(|_| "journey request encoding failed")?,
        0o444,
    )?;
    invoke("check", &request, &out)
}

fn status(out: &Path) -> Result<(), String> {
    let out = private_run(out)?;
    let request_path = out.join("request.json");
    let meta = fs::symlink_metadata(&request_path).map_err(|_| "journey request unavailable")?;
    if !meta.is_file()
        || meta.file_type().is_symlink()
        || meta.nlink() != 1
        || meta.mode() & 0o7777 != 0o444
    {
        return Err("read-only unshared journey request required".into());
    }
    let request: Request =
        serde_json::from_value(read(&request_path)?).map_err(|_| "journey request invalid")?;
    if request.format != 1 || request.scenario.path != out.join("scenario.json") {
        return Err("journey request binding changed".into());
    }
    input::verify(&request.scenario)?;
    let executable = driver(&request.scenario.path)?;
    if executable.path != request.driver.path || executable.sha256 != request.driver.sha256 {
        return Err("journey driver binding changed".into());
    }
    invoke("status", &request, &out)
}

fn invoke(action: &str, request: &Request, out: &Path) -> Result<(), String> {
    verify_driver(&request.driver)?;
    let mut command = std::process::Command::new(&request.driver.path);
    command.arg(action);
    if action == "check" {
        command.arg("--scenario").arg(&request.scenario.path);
        command.arg("--out").arg(out);
    } else {
        command.arg("--run").arg(out);
    }
    let status = command
        .stdin(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .map_err(|_| "journey driver could not start")?;
    if !status.success() {
        return Err(format!("journey {action} driver failed ({status})"));
    }
    Ok(())
}
