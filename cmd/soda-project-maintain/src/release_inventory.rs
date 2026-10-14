//! Complete read-only enrollment/native inventory for coordinated maintenance.
use std::collections::{BTreeMap, BTreeSet};
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use soda_host::domain::{valid_container_id, valid_id};
use soda_host::terminal::{terminal_target_ready, TerminalInspection, TERMINAL_INSPECT};

use crate::command;

const LIMIT: usize = 128;
const OPERATOR_SOCKET: &str = "/run/soda/operator/operator.sock";
const OPERATOR_URL: &str = "http://localhost/operator/projects";

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Entry {
    pub(crate) project_id: String,
    pub(crate) container_id: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Enrollment {
    projects: Vec<EnrolledProject>,
    factory_work: usize,
    preparation_work: usize,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct EnrolledProject {
    id: String,
    ready: bool,
}

fn enrolled_projects(body: &[u8]) -> Result<BTreeSet<String>, String> {
    if body.len() > 65536 {
        return Err("Project enrollment inventory exceeds its byte bound".into());
    }
    let observation: Enrollment = serde_json::from_slice(body)
        .map_err(|_| "Project enrollment inventory is invalid".to_string())?;
    if observation.projects.len() > LIMIT
        || observation.factory_work > LIMIT
        || observation.preparation_work > LIMIT
    {
        return Err("Project enrollment inventory exceeds its supported bound".into());
    }
    if observation.factory_work != 0 {
        return Err("Active or unsettled factory work blocks release maintenance".into());
    }
    if observation.preparation_work != 0 {
        return Err("Active or uncertain Project preparation blocks release maintenance".into());
    }
    let mut projects = BTreeSet::new();
    for row in observation.projects {
        if !valid_id(&row.id) || !projects.insert(row.id.clone()) {
            return Err("Project enrollment inventory has an invalid or duplicate identity".into());
        }
        if !row.ready {
            return Err(format!("Project provisioning is not confirmed: {}", row.id));
        }
    }
    Ok(projects)
}

fn native_ids(body: &[u8]) -> Result<BTreeSet<String>, String> {
    if body.len() > LIMIT * 65 {
        return Err("Native Project inventory exceeds its supported bound".into());
    }
    let text =
        std::str::from_utf8(body).map_err(|_| "Native Project inventory is invalid".to_string())?;
    let mut ids = BTreeSet::new();
    for id in text.split_terminator('\n') {
        if !valid_container_id(id) || !ids.insert(id.to_string()) || ids.len() > LIMIT {
            return Err("Native Project inventory has an invalid or duplicate identity".into());
        }
    }
    Ok(ids)
}

type Runner<'a> = dyn FnMut(&str, &[&str], Instant) -> Result<Vec<u8>, String> + 'a;

fn read_enrolled(run: &mut Runner<'_>, deadline: Instant) -> Result<BTreeSet<String>, String> {
    let remaining = deadline.saturating_duration_since(Instant::now());
    if remaining.is_zero() {
        return Err("Project enrollment inventory deadline expired".into());
    }
    let seconds = remaining
        .min(Duration::from_secs(5))
        .as_secs_f64()
        .to_string();
    let body = run(
        "/usr/bin/curl",
        &[
            "--silent",
            "--fail",
            "--proxy",
            "",
            "--max-time",
            &seconds,
            "--max-filesize",
            "65536",
            "--unix-socket",
            OPERATOR_SOCKET,
            OPERATOR_URL,
        ],
        deadline,
    )
    .map_err(|_| "Project enrollment inventory is unavailable".to_string())?;
    enrolled_projects(&body)
}

fn read_native_ids(run: &mut Runner<'_>, deadline: Instant) -> Result<BTreeSet<String>, String> {
    let body = run(
        "/usr/bin/podman",
        &[
            "--remote=false",
            "ps",
            "--all",
            "--no-trunc",
            "--filter",
            "label=org.soda.project",
            "--format",
            "{{.ID}}",
        ],
        deadline,
    )
    .map_err(|_| "Native Project inventory is unavailable".to_string())?;
    native_ids(&body)
}

fn inspect(
    run: &mut Runner<'_>,
    key: &str,
    deadline: Instant,
) -> Result<TerminalInspection, String> {
    let body = run(
        "/usr/bin/podman",
        &[
            "--remote=false",
            "inspect",
            "--format",
            TERMINAL_INSPECT,
            key,
        ],
        deadline,
    )
    .map_err(|_| "Native Project identity cannot be inspected".to_string())?;
    if body.len() > 4096 {
        return Err("Native Project identity inspection exceeds its byte bound".into());
    }
    let inspection = TerminalInspection::decode(&body)
        .map_err(|_| "Native Project identity inspection is invalid".to_string())?;
    if !valid_id(&inspection.project)
        || !terminal_target_ready(&inspection, &inspection.project, false)
    {
        return Err("Native Project identity or isolation is not admitted".into());
    }
    Ok(inspection)
}

fn inventory_with(deadline: Instant, run: &mut Runner<'_>) -> Result<Vec<Entry>, String> {
    let enrolled = read_enrolled(run, deadline)?;
    let ids = read_native_ids(run, deadline)?;
    let mut entries = BTreeMap::new();
    for id in &ids {
        let target = inspect(run, id, deadline)?;
        if &target.id != id || !enrolled.contains(&target.project) {
            return Err("Native Project inventory differs from enrolled Projects".into());
        }
        let by_name = inspect(run, &format!("soda-{}", target.project), deadline)?;
        if by_name != target {
            return Err(
                "Project container identity or runtime state changed during inventory".into(),
            );
        }
        if entries
            .insert(
                target.project.clone(),
                Entry {
                    project_id: target.project,
                    container_id: target.id,
                },
            )
            .is_some()
        {
            return Err("Multiple native containers claim one enrolled Project".into());
        }
    }
    if entries.keys().cloned().collect::<BTreeSet<_>>() != enrolled {
        return Err("An enrolled Project has no confirmed native container".into());
    }
    if read_native_ids(run, deadline)? != ids || read_enrolled(run, deadline)? != enrolled {
        return Err("Project enrollment or native inventory changed during observation".into());
    }
    Ok(entries.into_values().collect())
}

/// Caller holds exclusive admission while fixing the complete operation set.
pub(crate) fn inventory(deadline: Instant) -> Result<Vec<Entry>, String> {
    inventory_with(deadline, &mut command::run)
}

/// Confirm both the complete enrollment set and each retained container identity.
pub(crate) fn revalidate(entries: &[Entry], deadline: Instant) -> Result<(), String> {
    let expected = checked_entries(entries)?;
    if inventory(deadline)? != expected {
        return Err("Project inventory changed; release maintenance must remain closed".into());
    }
    Ok(())
}

fn checked_entries(entries: &[Entry]) -> Result<Vec<Entry>, String> {
    let mut projects = BTreeSet::new();
    let mut containers = BTreeSet::new();
    if entries.len() > LIMIT
        || entries.iter().any(|entry| {
            !valid_id(&entry.project_id)
                || !valid_container_id(&entry.container_id)
                || !projects.insert(entry.project_id.clone())
                || !containers.insert(entry.container_id.clone())
        })
    {
        return Err("Saved Project maintenance inventory is invalid".into());
    }
    let mut entries = entries.to_vec();
    entries.sort_by(|a, b| a.project_id.cmp(&b.project_id));
    Ok(entries)
}

#[cfg(test)]
mod tests {
    use super::*;
    const PROJECT: &str = "p0123456789abcdef01234567";
    const OTHER: &str = "p1123456789abcdef01234567";
    fn enrolled(projects: &[(&str, bool)], factory: usize, preparation: usize) -> Vec<u8> {
        serde_json::to_vec(&serde_json::json!({"projects":projects.iter().map(|(id,ready)|serde_json::json!({"id":id,"ready":ready})).collect::<Vec<_>>(),"factory_work":factory,"preparation_work":preparation})).unwrap()
    }
    fn observed(project: &str, id: &str, running: bool) -> Vec<u8> {
        format!(r#"{{"id":"{id}","project":"{project}","owner":"1","running":{running},"privileged":false,"userns":"private","mappings":{{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}}}"#).into_bytes()
    }
    #[test]
    fn enrollment_refuses_work_unready_unknown_duplicates_and_overflow() {
        for body in [
            enrolled(&[(PROJECT, true)], 1, 0),
            enrolled(&[(PROJECT, true)], 0, 1),
            enrolled(&[(PROJECT, false)], 0, 0),
            enrolled(&[(PROJECT, true), (PROJECT, true)], 0, 0),
            enrolled(&[("unknown", true)], 0, 0),
            enrolled(&[], 129, 0),
            b"{\"projects\":[],\"factory_work\":0,\"factory_work\":0,\"preparation_work\":0}"
                .to_vec(),
            b"{\"projects\":[],\"factory_work\":0,\"preparation_work\":0,\"truncated\":true}"
                .to_vec(),
            b"{\"projects\":null,\"factory_work\":0,\"preparation_work\":0}".to_vec(),
        ] {
            assert!(enrolled_projects(&body).is_err());
        }
        assert!(
            native_ids(format!("{}\n{}\n", "a".repeat(64), "a".repeat(64)).as_bytes()).is_err()
        );
        assert!(native_ids(&vec![b'a'; LIMIT * 65 + 1]).is_err());
        assert!(native_ids(b"unknown\n").is_err());
    }
    fn fixture(
        deadline: Instant,
        enrollment: Vec<u8>,
        projects: BTreeMap<String, Vec<u8>>,
    ) -> Result<Vec<Entry>, String> {
        let mut calls = Vec::new();
        let result = inventory_with(deadline, &mut |program, args, _| {
            calls.push((
                program.to_string(),
                args.iter()
                    .map(|arg| (*arg).to_string())
                    .collect::<Vec<_>>(),
            ));
            if program == "/usr/bin/curl" {
                return Ok(enrollment.clone());
            }
            if args[1] == "ps" {
                return Ok(projects
                    .keys()
                    .map(|id| format!("{id}\n"))
                    .collect::<String>()
                    .into_bytes());
            }
            let key = args.last().unwrap();
            if let Some(body) = projects.get(*key) {
                return Ok(body.clone());
            }
            for body in projects.values() {
                let target = TerminalInspection::decode(body).unwrap();
                if *key == format!("soda-{}", target.project) {
                    return Ok(body.clone());
                }
            }
            Err("missing target".into())
        });
        assert!(calls.iter().all(|(_, args)| {
            !args.iter().any(|arg| {
                ["start", "stop", "exec", "create", "mount", "rm"].contains(&arg.as_str())
            })
        }));
        result
    }
    #[test]
    fn inventory_includes_running_and_stopped_exact_roots_and_rejects_missing_or_extra() {
        let a = "a".repeat(64);
        let b = "b".repeat(64);
        let native = BTreeMap::from([
            (a.clone(), observed(PROJECT, &a, true)),
            (b.clone(), observed(OTHER, &b, false)),
        ]);
        let deadline = Instant::now() + Duration::from_secs(2);
        let entries = fixture(
            deadline,
            enrolled(&[(OTHER, true), (PROJECT, true)], 0, 0),
            native.clone(),
        )
        .unwrap();
        assert_eq!(
            entries,
            [
                Entry {
                    project_id: PROJECT.into(),
                    container_id: a.clone()
                },
                Entry {
                    project_id: OTHER.into(),
                    container_id: b.clone()
                }
            ]
        );
        assert!(fixture(deadline, enrolled(&[(PROJECT, true)], 0, 0), native.clone()).is_err());
        assert!(fixture(
            deadline,
            enrolled(&[(PROJECT, true), (OTHER, true)], 0, 0),
            BTreeMap::from([(a.clone(), observed(PROJECT, &a, true))])
        )
        .is_err());
        assert!(fixture(
            deadline,
            enrolled(&[(PROJECT, true)], 0, 0),
            BTreeMap::from([
                (a.clone(), observed(PROJECT, &a, true)),
                (b.clone(), observed(PROJECT, &b, false))
            ])
        )
        .is_err());
        assert!(checked_entries(&[entries[0].clone(), entries[0].clone()]).is_err());
    }
    #[test]
    fn inventory_refuses_changed_incarnation_or_set_during_read() {
        let a = "a".repeat(64);
        let b = "b".repeat(64);
        let mut scans = 0;
        assert!(inventory_with(
            Instant::now() + Duration::from_secs(1),
            &mut |program, args, _| {
                if program == "/usr/bin/curl" {
                    return Ok(enrolled(&[(PROJECT, true)], 0, 0));
                }
                if args[1] == "ps" {
                    scans += 1;
                    return Ok(format!("{}\n", if scans == 1 { &a } else { &b }).into_bytes());
                }
                Ok(observed(PROJECT, &a, true))
            }
        )
        .is_err());
        assert!(inventory_with(
            Instant::now() + Duration::from_secs(1),
            &mut |program, args, _| {
                if program == "/usr/bin/curl" {
                    return Ok(enrolled(&[(PROJECT, true)], 0, 0));
                }
                if args[1] == "ps" {
                    return Ok(format!("{a}\n").into_bytes());
                }
                Ok(observed(
                    PROJECT,
                    if args.last() == Some(&a.as_str()) {
                        &a
                    } else {
                        &b
                    },
                    true,
                ))
            }
        )
        .is_err());
    }
}
