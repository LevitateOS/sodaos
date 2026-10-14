//! Guided production handoff; delegates builds and preparation to their owners.
use clap::ArgMatches;
use soda_release_deliver::{model::Distribution, publish};
use std::{
    collections::BTreeMap,
    io::{BufRead, Write},
    path::Path,
};

pub fn distribution(value: &str) -> Result<Distribution, String> {
    match value {
        "appliance" => Ok(Distribution::Appliance),
        "docker" => Ok(Distribution::Docker),
        "all" => Ok(Distribution::All),
        _ => Err("select appliance, docker or all".into()),
    }
}
const FIELDS: [&str; 6] = [
    "distribution",
    "tag",
    "worker-config",
    "notes",
    "out",
    "arch",
];
fn choices(
    flags: &ArgMatches,
    input: &mut dyn BufRead,
    output: &mut dyn Write,
) -> Result<BTreeMap<String, String>, String> {
    let mut values = FIELDS
        .into_iter()
        .map(|name| {
            (
                name.to_owned(),
                flags.get_one::<String>(name).cloned().unwrap_or_default(),
            )
        })
        .collect::<BTreeMap<_, _>>();
    if !flags.get_flag("non-interactive") {
        loop {
            writeln!(
                output,
                "Production release inputs (build and prepare only):"
            )
            .map_err(|e| e.to_string())?;
            for (i, name) in FIELDS.iter().enumerate() {
                writeln!(output, "  {}. {}: {}", i + 1, name, values[*name])
                    .map_err(|e| e.to_string())?;
            }
            write!(
                output,
                "Enter a field number to edit, go to build, or quit: "
            )
            .map_err(|e| e.to_string())?;
            output.flush().map_err(|e| e.to_string())?;
            let mut line = String::new();
            if input.read_line(&mut line).map_err(|e| e.to_string())? == 0 {
                return Err("input ended; use complete flags with --non-interactive".into());
            }
            match line.trim() {
                "quit" => return Err("release start cancelled before execution".into()),
                "go" => {
                    if let Err(error) = validate(&values) {
                        writeln!(output, "{error}").map_err(|e| e.to_string())?;
                    } else {
                        break;
                    }
                }
                value => {
                    let Some(field) = value
                        .parse::<usize>()
                        .ok()
                        .and_then(|index| FIELDS.get(index.wrapping_sub(1)))
                    else {
                        writeln!(output, "Choose a field number 1-6, go or quit.")
                            .map_err(|error| error.to_string())?;
                        continue;
                    };
                    write!(output, "{field}: ").map_err(|e| e.to_string())?;
                    output.flush().map_err(|e| e.to_string())?;
                    line.clear();
                    if input.read_line(&mut line).map_err(|e| e.to_string())? == 0 {
                        return Err("input ended".into());
                    }
                    values.insert((*field).into(), line.trim().into());
                }
            }
        }
    }
    validate(&values)?;
    Ok(values)
}
fn validate(values: &BTreeMap<String, String>) -> Result<(), String> {
    for name in FIELDS {
        if values[name].is_empty() {
            return Err(format!("{name} required"));
        }
    }
    distribution(&values["distribution"])?;
    crate::digest::oci_architecture(&values["arch"])?;
    if !soda_build_tools::reader::settings::release_tag(&values["tag"])
        || ["latest", "latest-arm64"]
            .iter()
            .any(|tag| values["tag"].eq_ignore_ascii_case(tag))
    {
        return Err("exact version tag required".into());
    }
    for name in ["worker-config", "notes", "out"] {
        if !Path::new(&values[name]).is_absolute() {
            return Err(format!("absolute {name} required"));
        }
    }
    Ok(())
}
fn workflow_output(source: &Path, out: &Path) -> Result<(), String> {
    let path = out.to_str().ok_or("safe workflow output required")?;
    if !out.is_absolute()
        || crate::worker::go_clean(path) != path
        || !out.starts_with(source.join(".artifacts"))
    {
        return Err(
            "workflow output must stay below the selected worktree's .artifacts without traversal"
                .into(),
        );
    }
    let parent = out
        .parent()
        .and_then(Path::to_str)
        .ok_or("existing workflow output parent required")?;
    crate::worker::valid_worker_path(parent)?;
    crate::candidate::check_fresh_out(path)
}
fn packaging_args(selected: Distribution, tag: &str) -> Vec<String> {
    let mut args = Vec::new();
    if selected != Distribution::Appliance {
        args.extend(["--release-tag".into(), tag.into()]);
    }
    if selected != Distribution::Docker {
        args.extend([
            "--rootfs-base-url".into(),
            format!("https://github.com/LevitateOS/sodaos/releases/download/{tag}/"),
        ]);
    }
    args
}
pub fn run(flags: &ArgMatches) -> Result<(), String> {
    let values = choices(
        flags,
        &mut std::io::stdin().lock(),
        &mut std::io::stderr().lock(),
    )?;
    if crate::worker::euid() != 0 {
        return Err("run soda-release start with sudo: its admitted controller and restricted worker configuration require root; preserve GH_CONFIG_DIR and TMPDIR on the workspace disk".into());
    }
    let notes = soda_release_deliver::document::read_file(&values["notes"], 64 << 10)
        .map_err(|error| error.to_string())?;
    if std::str::from_utf8(&notes)
        .map_err(|_| "UTF-8 release notes required")?
        .trim()
        .is_empty()
    {
        return Err("nonempty release notes required before building".into());
    }
    let selected = distribution(&values["distribution"])?;
    crate::worker::private_file(&values["worker-config"]).map_err(|_| {
        "run start as the admitted build operator with its root-owned restricted worker config"
            .to_owned()
    })?;
    let config = crate::worker::read_worker_config(&values["worker-config"])?;
    crate::worker::trusted_executable(&config.executable)?;
    crate::worker::admit_storage_root(&config)?;
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    if cwd.to_str() != Some(config.source.as_str()) {
        return Err("run start from the worker's selected clean checkout or worktree".into());
    }
    crate::digest::require_native(&values["arch"])?;
    let revision = crate::build_cli::controller_revision()?;
    crate::build_cli::check_source_revision(&config.source, &revision)?;
    let forgejo = crate::build_cli::forgejo_checkout_revision(&config.forgejo_source)?;
    #[derive(serde::Deserialize)]
    struct Pin {
        revision: String,
    }
    let pin: Pin = serde_json::from_slice(
        &std::fs::read(cwd.join("fountain-source.json")).map_err(|e| e.to_string())?,
    )
    .map_err(|e| e.to_string())?;
    if pin.revision != forgejo {
        return Err("Fountain checkout differs from selected source pin".into());
    }
    let out = Path::new(&values["out"]);
    workflow_output(&cwd, out)?;
    if !Path::new(&config.build_home).starts_with(cwd.join(".artifacts"))
        || !Path::new(&config.runtime).starts_with(cwd.join(".artifacts"))
    {
        return Err("workflow output, build home and runtime must stay below the selected worktree's .artifacts".into());
    }
    let leaf = out
        .file_name()
        .and_then(|v| v.to_str())
        .ok_or("workflow output name required")?;
    let build = Path::new(&config.output_parent).join(format!("{leaf}-build"));
    crate::candidate::check_fresh_out(build.to_str().ok_or("build path required")?)?;
    let request = crate::build_spec::Request {
        source: config.source.clone(),
        forgejo_source: config.forgejo_source.clone(),
        out: build.to_string_lossy().into_owned(),
        distribution: values["distribution"].clone(),
        arch: values["arch"].clone(),
        ..Default::default()
    };
    if !crate::worker::valid_worker_task_paths(&config, &request) {
        return Err("build output does not match admitted worker paths".into());
    }
    for path in crate::worker::worker_config_paths(&config, &request) {
        crate::worker::valid_worker_path(&path)?;
    }
    #[derive(serde::Deserialize)]
    #[serde(rename_all = "camelCase")]
    struct Canlang {
        archive_filename: String,
        sha256: String,
    }
    let lock: Canlang = serde_json::from_slice(
        &std::fs::read(cwd.join(soda_build_tools::reader::canlang::release_path(&values["arch"]).map_err(|error| error.0)?))
            .map_err(|_| format!("verified {} Canlang release lock is missing; produce and pin its native tools archive first", values["arch"]))?,
    )
    .map_err(|e| e.to_string())?;
    let filename = soda_build_tools::reader::canlang::archive_filename(&values["arch"])
        .map_err(|error| error.0)?;
    let archive = cwd.join(".artifacts/canlang").join(filename);
    if !std::fs::symlink_metadata(&archive)
        .map_err(|_| "locked offline Canlang archive is missing")?
        .is_file()
    {
        return Err("locked offline Canlang archive must be a regular file".into());
    }
    if lock.archive_filename != filename
        || crate::digest::hash_file(
            cwd.join(".artifacts/canlang")
                .join(&lock.archive_filename)
                .to_str()
                .ok_or("Canlang path required")?,
        )? != lock.sha256
    {
        return Err("locked offline Canlang archive is missing or differs".into());
    }
    for tool in if selected == Distribution::Appliance {
        &["podman"][..]
    } else {
        &["podman", "skopeo"][..]
    } {
        if !std::process::Command::new(tool)
            .arg("--version")
            .status()
            .map_err(|_| format!("required release tool {tool} unavailable"))?
            .success()
        {
            return Err(format!("required release tool {tool} unavailable"));
        }
    }
    publish::preflight(&publish::Gh, &revision, &forgejo, &values["tag"], selected)
        .map_err(|e| e.to_string())?;
    let metadata = std::fs::metadata(&config.output_parent).map_err(|e| e.to_string())?;
    if !metadata.is_dir() {
        return Err("existing output parent required".into());
    }
    let mut args = vec![
        "--worker-config".into(),
        values["worker-config"].clone(),
        "--arch".into(),
        values["arch"].clone(),
        "--distribution".into(),
        values["distribution"].clone(),
        "--out".into(),
        build.to_string_lossy().into_owned(),
        "--forgejo-source".into(),
        config.forgejo_source.clone(),
    ];
    args.extend(packaging_args(selected, &values["tag"]));
    let mut preflight_args = args.clone();
    preflight_args.push("--preflight".into());
    if !std::process::Command::new(&config.executable)
        .args(&preflight_args)
        .current_dir(&config.source)
        .status()
        .map_err(|error| error.to_string())?
        .success()
    {
        return Err("admitted controller preflight failed; no build started".into());
    }
    let storage = std::ffi::CString::new(config.output_parent.as_str())
        .map_err(|_| "safe output path required")?;
    let mut space = std::mem::MaybeUninit::<libc::statvfs>::uninit();
    if unsafe { libc::statvfs(storage.as_ptr(), space.as_mut_ptr()) } != 0 {
        return Err("cannot inspect build workspace capacity".into());
    }
    let space = unsafe { space.assume_init() };
    if space.f_bavail == 0 {
        return Err("build workspace has no free capacity".into());
    }
    eprintln!("Build workspace free capacity: {} GiB; keep diagnostics and failed attempts on this workspace.", (space.f_bavail as u64).saturating_mul(space.f_frsize as u64) / (1 << 30));
    if flags.get_flag("preflight") {
        println!("Release prerequisites passed; no build started or attempt created.");
        return Ok(());
    }
    use std::os::unix::fs::DirBuilderExt;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(out)
        .map_err(|e| e.to_string())?;
    let mut request = serde_json::to_value(&values).map_err(|e| e.to_string())?;
    request["source_revision"] = revision.into();
    request["forgejo_revision"] = forgejo.into();
    request["build_output"] = build.to_string_lossy().into_owned().into();
    request["prepared"] = out.join("release").to_string_lossy().into_owned().into();
    std::fs::write(
        out.join("request.json"),
        serde_json::to_vec_pretty(&request).map_err(|e| e.to_string())?,
    )
    .map_err(|e| e.to_string())?;
    let code = crate::candidate_controller::start_release_controller(
        &config.executable,
        &args,
        &request["build_output"].as_str().unwrap_or_default(),
        flags.get_flag("non-interactive"),
    )?
    .wait()?;
    if code != 0 {
        return Err(format!(
            "production build failed ({code}); diagnostics retained at {}",
            build.display()
        ));
    }
    publish::prepare(
        &build.join("artifacts"),
        &values["tag"],
        selected,
        Path::new(&values["notes"]),
        &out.join("release"),
    )
    .map_err(|e| e.to_string())?;
    println!(
        "Prepared unqualified release at {}. Next: soda-release status --input {}",
        out.join("release").display(),
        out.join("release").display()
    );
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn guided_distribution_arguments_pass_the_real_producer_admission() {
        for name in ["appliance", "docker", "all"] {
            let selected = distribution(name).unwrap();
            let mut args = vec!["--distribution".into(), name.into()];
            args.extend(packaging_args(selected, "v1.2.3"));
            let flags = crate::build_cli::parse_build_flags(&args).unwrap();
            let request = crate::pipeline::image_request(&flags.request);
            request.validate_target().unwrap();
            assert_eq!(request.wants_docker(), selected != Distribution::Appliance);
            assert_eq!(request.wants_media(), selected != Distribution::Docker);
        }
    }
    #[test]
    fn missing_noninteractive_inputs_refuse_before_execution() {
        let flags = crate::release_cli::command()
            .try_get_matches_from(["soda-release", "start", "--non-interactive"])
            .unwrap();
        let (_, flags) = flags.subcommand().unwrap();
        assert!(
            choices(flags, &mut std::io::Cursor::new(b""), &mut Vec::new())
                .unwrap_err()
                .contains("distribution required")
        );
    }
    #[test]
    fn guided_arm_inputs_keep_explicit_architecture() {
        let flags = crate::release_cli::command()
            .try_get_matches_from([
                "soda-release",
                "start",
                "--distribution",
                "all",
                "--arch",
                "aarch64",
                "--tag",
                "v1.0.0-aarch64",
                "--worker-config",
                "/home/build/worker.json",
                "--notes",
                "/home/build/notes.md",
                "--out",
                "/home/build/workflow",
                "--non-interactive",
            ])
            .unwrap();
        let (_, flags) = flags.subcommand().unwrap();
        let values = choices(flags, &mut std::io::Cursor::new(b""), &mut Vec::new()).unwrap();
        assert_eq!(values["arch"], "aarch64");
        assert_eq!(values["distribution"], "all");
        let mut unsupported = values;
        unsupported.insert("arch".into(), "armv7".into());
        assert!(validate(&unsupported).is_err());
    }
    #[test]
    fn guided_overview_edits_and_preserves_explicit_inputs() {
        let flags = crate::release_cli::command()
            .try_get_matches_from([
                "soda-release",
                "start",
                "--distribution",
                "appliance",
                "--tag",
                "v1.0.0",
                "--worker-config",
                "/home/build/worker.json",
                "--notes",
                "/home/build/notes.md",
                "--out",
                "/home/build/workflow",
            ])
            .unwrap();
        let (_, flags) = flags.subcommand().unwrap();
        let mut output = Vec::new();
        let values = choices(
            flags,
            &mut std::io::Cursor::new(b"1\ndocker\ngo\n"),
            &mut output,
        )
        .unwrap();
        assert_eq!(values["distribution"], "docker");
        assert_eq!(values["tag"], "v1.0.0");
        assert!(String::from_utf8(output)
            .unwrap()
            .contains("build and prepare only"));
    }

    #[test]
    fn workflow_output_rejects_traversal_symlinks_and_existing_attempts() {
        use std::os::unix::fs::symlink;
        let source =
            std::env::temp_dir().join(format!("soda-release-paths-{}", std::process::id()));
        std::fs::create_dir(&source).unwrap();
        std::fs::create_dir(source.join(".artifacts")).unwrap();
        std::fs::create_dir(source.join("outside")).unwrap();
        symlink(source.join("outside"), source.join(".artifacts/link")).unwrap();
        assert!(workflow_output(&source, &source.join(".artifacts/fresh")).is_ok());
        for path in [
            ".artifacts/../outside/run",
            ".artifacts/link/run",
            "outside/run",
            ".artifacts/missing/run",
        ] {
            assert!(
                workflow_output(&source, &source.join(path)).is_err(),
                "{path}"
            );
        }
        std::fs::create_dir(source.join(".artifacts/fresh")).unwrap();
        assert!(workflow_output(&source, &source.join(".artifacts/fresh")).is_err());
        std::fs::remove_dir_all(source).unwrap();
    }
}
