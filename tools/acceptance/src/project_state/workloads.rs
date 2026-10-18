use std::path::{Path, PathBuf};

use crate::structured::Value as JsonValue;

use super::{
    arg_list, command, glob_prefix, obj, output_lines, s, set, SnapshotFailure, SnapshotKind,
};

/// Project IP gate, like `assert ip_address(ip) in ip_network('10.89.0.0/24')`:
/// unparseable input is a `ValueError`, parsed-but-outside (v4 or v6) fails
/// the bare membership assert.
pub(super) fn check_project_ip(ip: &str) -> Result<(), SnapshotFailure> {
    match ip.parse::<std::net::IpAddr>() {
        Ok(std::net::IpAddr::V4(v4)) => {
            let octets = v4.octets();
            if octets[0] == 10 && octets[1] == 89 && octets[2] == 0 {
                Ok(())
            } else {
                Err(SnapshotFailure::assertion(""))
            }
        }
        Ok(_) => Err(SnapshotFailure::assertion("")),
        Err(_) => Err(SnapshotFailure::bare(SnapshotKind::ValueError)),
    }
}

/// List workload container inspections, requiring at least two containers.
fn collect_workloads(podman: &[&str; 3]) -> Result<Vec<JsonValue>, SnapshotFailure> {
    let ids = output_lines(&command(
        &arg_list(&[podman[0], podman[1], podman[2], "ps", "-aq", "--no-trunc"]),
        &[],
    )?);
    if ids.len() < 2 {
        return Err(SnapshotFailure::assertion(
            "Missing required workload containers",
        ));
    }
    let mut sorted_ids = ids;
    sorted_ids.sort();
    let mut workloads = Vec::new();
    for identifier in &sorted_ids {
        workloads.push(s(command(
            &arg_list(&[
                podman[0],
                podman[1],
                podman[2],
                "container",
                "inspect",
                "--format",
                "{{json .ID}} {{json .Name}} {{json .Image}} {{json .HostConfig.NetworkMode}} {{json .Mounts}}",
                identifier,
            ]),
            &[],
        )?));
    }
    Ok(workloads)
}

/// List sorted podman volume names.
fn collect_volumes(podman: &[&str; 3]) -> Result<Vec<JsonValue>, SnapshotFailure> {
    let mut volumes = output_lines(&command(
        &arg_list(&[
            podman[0],
            podman[1],
            podman[2],
            "volume",
            "ls",
            "--format",
            "{{.Name}}",
        ]),
        &[],
    )?);
    volumes.sort();
    Ok(volumes.into_iter().map(s).collect())
}

/// Find the running workload database containers.
fn find_databases(podman: &[&str; 3]) -> Result<Vec<String>, SnapshotFailure> {
    let names = output_lines(&command(
        &arg_list(&[
            podman[0],
            podman[1],
            podman[2],
            "ps",
            "--format",
            "{{.Names}}",
        ]),
        &[],
    )?);
    let databases: Vec<String> = names
        .into_iter()
        .filter(|name| name == "u08-projectnet-database" || name == "workload_database_1")
        .collect();
    if databases.is_empty() {
        return Err(SnapshotFailure::assertion(
            "No running database; restore existing workload before snapshot",
        ));
    }
    Ok(databases)
}

/// Read the live project IP from the environment and gate it.
fn live_project_ip() -> Result<String, SnapshotFailure> {
    let ip = std::env::var("SODA_PROJECT_IP")
        .map_err(|_| SnapshotFailure::bare(SnapshotKind::KeyError))?;
    check_project_ip(&ip)?;
    Ok(ip)
}

/// Find the newest native client credential file.
fn find_passfile() -> Result<PathBuf, SnapshotFailure> {
    let mut passfiles = glob_prefix(Path::new("/home/u08-alice-8417/.config"), "u08-db-client-")?
        .into_iter()
        .map(|dir| dir.join("pgpass"))
        .filter(|path| path.exists())
        .collect::<Vec<PathBuf>>();
    passfiles.sort();
    if passfiles.is_empty() {
        return Err(SnapshotFailure::assertion(
            "Missing native client credential input",
        ));
    }
    Ok(passfiles.pop().unwrap())
}

/// Check the credential file is a private regular file for the live target.
fn check_passfile(passfile: &Path, ip: &str) -> Result<(), SnapshotFailure> {
    let link_type = std::fs::symlink_metadata(passfile)
        .map_err(SnapshotFailure::io)?
        .file_type();
    if link_type.is_symlink() {
        return Err(SnapshotFailure::assertion(""));
    }
    let pass_meta = std::fs::metadata(passfile).map_err(SnapshotFailure::io)?;
    use std::os::unix::fs::MetadataExt;
    if pass_meta.mode() & 0o077 != 0 {
        return Err(SnapshotFailure::assertion(""));
    }
    let cached = std::fs::read_to_string(passfile).map_err(SnapshotFailure::io)?;
    let endpoint = cached.split(':').next().unwrap_or("");
    if endpoint != ip {
        return Err(SnapshotFailure::assertion(
            "Cached credential endpoint differs from live native target",
        ));
    }
    Ok(())
}

/// Snapshot the probe rows from each workload database over native TCP.
/// No dependency on exec into a different-UID workload and no credential
/// export, like the inline query it replaces.
fn collect_database(
    ip: &str,
    passfile: &Path,
    databases: &[String],
) -> Result<JsonValue, SnapshotFailure> {
    let mut database = obj();
    for name in databases {
        let rows = command(
            &arg_list(&[
                "psql",
                "-X",
                "-w",
                "-h",
                ip,
                "-p",
                "5432",
                "-U",
                "developer",
                "-d",
                "soda_example",
                "-At",
                "-c",
                "SELECT run_id,value FROM soda_u08_probe ORDER BY run_id",
            ]),
            &[
                ("PGPASSFILE", &passfile.to_string_lossy()),
                ("PGCONNECT_TIMEOUT", "5"),
            ],
        )?;
        if rows.is_empty() {
            return Err(SnapshotFailure::assertion(
                "Empty required database snapshot",
            ));
        }
        set(&mut database, name, s(rows));
    }
    Ok(database)
}

pub(super) fn snapshot_workloads(
    mut data: &mut JsonValue,
    expect: &str,
) -> Result<(), SnapshotFailure> {
    if expect != "0" && expect != "1" {
        return Err(SnapshotFailure::assertion(
            "Caller must declare required workload observations",
        ));
    }
    if expect == "1" {
        let podman = ["podman", "--url", "unix:///run/soda-podman/podman.sock"];
        let workloads = collect_workloads(&podman)?;
        set(&mut data, "workloads", JsonValue::Array(workloads));
        let volumes = collect_volumes(&podman)?;
        set(&mut data, "volumes", JsonValue::Array(volumes));
        let databases = find_databases(&podman)?;
        let ip = live_project_ip()?;
        let passfile = find_passfile()?;
        check_passfile(&passfile, &ip)?;
        let database = collect_database(&ip, &passfile, &databases)?;
        set(&mut data, "database", database);
    }
    Ok(())
}
