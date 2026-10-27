//! Retirement owns only one verified stopped restoration and its three copies.
//! A durable intent makes uncertain destructive outcomes inspection-only.
use super::*;
use std::collections::BTreeSet;

#[derive(Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Plan {
    format: u8,
    action: String,
    container: String,
    restored: PathBuf,
    source: PathBuf,
    source_seal_sha256: String,
    result_sha256: String,
    trees: Vec<PathBuf>,
    selected_allocated_bytes: u64,
    filesystem_available_bytes: u64,
    automatic_retry: bool,
    source_preserved: bool,
    private_settings_preserved: bool,
}

fn document(path: &Path) -> Result<serde_json::Value, String> {
    soda_release_deliver::publish::qualification::protected_path(path, false)
        .map_err(|_| "protected retirement metadata required")?;
    let meta = fs::metadata(path).map_err(|_| "retirement metadata unavailable")?;
    need(meta.len() <= 16 << 20, "retirement metadata exceeds bound")?;
    serde_json::from_slice(&fs::read(path).map_err(|_| "retirement metadata unavailable")?)
        .map_err(|_| "retirement metadata invalid".into())
}

fn canonical_directory(path: &Path) -> Result<(), String> {
    need(
        path.is_absolute()
            && (path.starts_with("/home") || path.starts_with("/var/home"))
            && fs::canonicalize(path).is_ok_and(|actual| actual == path),
        "canonical retirement directory under /home or /var/home required",
    )?;
    private(path)
}

fn overlap(a: &Path, b: &Path) -> bool {
    a.starts_with(b) || b.starts_with(a)
}

fn available(path: &Path) -> Result<u64, String> {
    let path = CString::new(path.as_os_str().as_bytes()).map_err(|_| "retirement path invalid")?;
    let mut state = std::mem::MaybeUninit::<libc::statvfs>::uninit();
    need(
        unsafe { libc::statvfs(path.as_ptr(), state.as_mut_ptr()) } == 0,
        "retirement free space unavailable",
    )?;
    let state = unsafe { state.assume_init() };
    state
        .f_bavail
        .checked_mul(state.f_frsize)
        .ok_or("retirement free space overflow".into())
}

fn tree_allocation(
    path: &Path,
    device: u64,
    seen: &mut BTreeSet<(u64, u64)>,
) -> Result<u64, String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "retirement tree unavailable")?;
    need(meta.dev() == device, "retirement tree crosses filesystem")?;
    let mut bytes = if seen.insert((meta.dev(), meta.ino())) {
        meta.blocks()
            .checked_mul(512)
            .ok_or("retirement allocation overflow")?
    } else {
        0
    };
    if meta.is_dir() {
        for child in fs::read_dir(path).map_err(|_| "retirement tree unavailable")? {
            let child = child.map_err(|_| "retirement tree unavailable")?;
            bytes = bytes
                .checked_add(tree_allocation(&child.path(), device, seen)?)
                .ok_or("retirement allocation overflow")?;
        }
    }
    Ok(bytes)
}

fn mount_path(value: &str) -> Result<PathBuf, String> {
    // mountinfo escapes space, tab, newline and backslash with octal digits.
    let mut bytes = Vec::new();
    let value = value.as_bytes();
    let mut i = 0;
    while i < value.len() {
        if value[i] == b'\\' {
            need(i + 3 < value.len(), "retirement mount escape invalid")?;
            let code = &value[i + 1..i + 4];
            need(
                code.iter().all(|b| (b'0'..=b'7').contains(b)),
                "retirement mount escape invalid",
            )?;
            bytes.push((code[0] - b'0') * 64 + (code[1] - b'0') * 8 + code[2] - b'0');
            i += 4;
        } else {
            bytes.push(value[i]);
            i += 1;
        }
    }
    Ok(PathBuf::from(std::ffi::OsStr::from_bytes(&bytes)))
}

fn no_active_mounts(trees: &[PathBuf]) -> Result<(), String> {
    // Inspect each live mount namespace: a bind retained by another process can
    // survive Docker's stopped-container inventory and must prevent deletion.
    let host = fs::read_to_string("/proc/self/mountinfo")
        .map_err(|_| "retirement filesystem inventory unavailable")?;
    let mut backing = Vec::new();
    for tree in trees {
        let mut selected: Option<(usize, String, PathBuf, PathBuf)> = None;
        for line in host.lines() {
            let fields: Vec<_> = line.split(' ').collect();
            need(fields.len() > 6, "retirement mount inventory invalid")?;
            let mount = mount_path(fields[4])?;
            if let Ok(relative) = tree.strip_prefix(&mount) {
                let depth = mount.components().count();
                if selected.as_ref().is_none_or(|(old, _, _, _)| depth > *old) {
                    let root = mount_path(fields[3])?;
                    selected = Some((depth, fields[2].into(), root.join(relative), root));
                }
            }
        }
        let (_, device, path, root) =
            selected.ok_or("retirement backing filesystem unavailable")?;
        backing.push((device, path, root));
    }
    let mut namespaces = BTreeSet::new();
    for process in fs::read_dir("/proc").map_err(|_| "retirement process inventory unavailable")? {
        let process = process.map_err(|_| "retirement process inventory unavailable")?;
        if !process
            .file_name()
            .as_bytes()
            .iter()
            .all(u8::is_ascii_digit)
        {
            continue;
        }
        let namespace = match fs::metadata(process.path().join("ns/mnt")) {
            Ok(meta) => (meta.dev(), meta.ino()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return Err("retirement mount namespace unavailable".into()),
        };
        if !namespaces.insert(namespace) {
            continue;
        }
        let info = match fs::read_to_string(process.path().join("mountinfo")) {
            Ok(info) => info,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                namespaces.remove(&namespace);
                continue;
            }
            Err(_) => return Err("retirement active mounts unavailable".into()),
        };
        for line in info.lines() {
            let fields: Vec<_> = line.split(' ').collect();
            need(fields.len() > 6, "retirement mount inventory invalid")?;
            let root = mount_path(fields[3])?;
            let mount = mount_path(fields[4])?;
            // Filesystem root "/" is an ancestor of every ordinary path; it
            // isn't an active mount of that selected tree. Specific bind roots
            // and mountpoints within the selected copies are meaningful.
            need(
                !trees.iter().any(|tree| mount.starts_with(tree))
                    && !backing.iter().any(|(device, path, baseline)| {
                        fields[2] == device && root != *baseline && overlap(path, &root)
                    }),
                "active mount retains retirement storage",
            )?;
        }
    }
    Ok(())
}

fn inspect_clone(
    container: &str,
    restored: &Path,
    source: &Source,
    trees: &[PathBuf],
) -> Result<(), String> {
    need(
        hex(container, 64) && container != source.container,
        "distinct exact retirement clone required",
    )?;
    let values = json(&docker(&[
        "container".into(),
        "inspect".into(),
        container.into(),
    ])?)?;
    let value = values
        .as_array()
        .filter(|v| v.len() == 1)
        .and_then(|v| v.first())
        .ok_or("one retirement clone required")?;
    let state = &value["State"];
    let clean = state["Status"] == "created"
        || (state["Status"] == "exited"
            && [Some(0), Some(130)].contains(&state["ExitCode"].as_i64()));
    need(
        value["Id"] == container
            && value["Config"]["Labels"]["soda.qualification.cold"] == "true"
            && value["Image"] == source.engine_image
            && value["Config"]["Image"] == source.image
            && clean
            && state["Running"] == false
            && state["Paused"] == false
            && state["Restarting"] == false
            && state["Dead"] == false
            && state["OOMKilled"] == false
            && value["HostConfig"]["NetworkMode"] == "none"
            && value["HostConfig"]["CgroupnsMode"] == "private"
            && value["HostConfig"]["UsernsMode"] == "host"
            && value["HostConfig"]["PidMode"] == ""
            && value["HostConfig"]["IpcMode"] == "private"
            && value["HostConfig"]["Privileged"] == true,
        "exact cleanly stopped isolated retirement clone required",
    )?;
    let mounts = value["Mounts"]
        .as_array()
        .ok_or("retirement mounts unavailable")?;
    need(
        mounts.len() == 3
            && DESTINATIONS.iter().all(|destination| {
                mounts.iter().any(|mount| {
                    mount["Type"] == "bind"
                        && mount["RW"] == true
                        && mount["Destination"] == *destination
                        && mount["Source"]
                            == restored
                                .join(destination.trim_start_matches('/'))
                                .to_str()
                                .unwrap_or("")
                })
            }),
        "retirement clone bindings differ",
    )?;
    no_other_containers(container, trees)?;
    no_active_mounts(trees)
}

fn no_other_containers(container: &str, trees: &[PathBuf]) -> Result<(), String> {
    let ids = String::from_utf8(docker(&[
        "container".into(),
        "ls".into(),
        "--all".into(),
        "--quiet".into(),
        "--no-trunc".into(),
    ])?)
    .map_err(|_| "retirement container inventory invalid")?;
    for id in ids.lines().filter(|id| !id.is_empty() && *id != container) {
        need(hex(id, 64), "retirement container identity invalid")?;
        let values = json(&docker(&["container".into(), "inspect".into(), id.into()])?)?;
        let mounts = values[0]["Mounts"]
            .as_array()
            .ok_or("retirement dependency inventory unavailable")?;
        for mount in mounts {
            let path = Path::new(
                mount["Source"]
                    .as_str()
                    .ok_or("retirement dependency path unavailable")?,
            );
            let resolved = fs::canonicalize(path).unwrap_or_else(|_| path.into());
            need(
                !trees
                    .iter()
                    .any(|tree| overlap(path, tree) || overlap(&resolved, tree)),
                "another container retains retirement storage",
            )?;
        }
        need(
            !values[0]["HostConfig"]["VolumesFrom"]
                .as_array()
                .is_some_and(|values| {
                    values.iter().any(|value| {
                        value
                            .as_str()
                            .is_some_and(|id| id.split(':').next() == Some(container))
                    })
                }),
            "another container retains retirement clone",
        )?;
    }
    Ok(())
}

fn verify(restored: &Path, result: &str) -> Result<Plan, String> {
    need(
        hex(result, 64),
        "exact successful journey result SHA256 required",
    )?;
    canonical_directory(restored)?;
    let request = document(&restored.join("request.json"))?;
    let create = document(&restored.join("create.request.json"))?;
    let complete = document(&restored.join("complete.json"))?;
    let source = PathBuf::from(
        request["source"]
            .as_str()
            .ok_or("retirement cold source missing")?,
    );
    canonical_directory(&source)?;
    need(
        !overlap(restored, &source),
        "retirement copies must be distinct from captured source",
    )?;
    let seal_path = source.join("complete.json");
    soda_release_deliver::publish::qualification::protected_path(&seal_path, false)
        .map_err(|_| "protected retirement cold seal required")?;
    need(
        fs::metadata(&seal_path)
            .map_err(|_| "retirement cold seal unavailable")?
            .len()
            <= 512 << 20,
        "retirement cold seal exceeds existing inventory envelope",
    )?;
    // The existing capture seal contains full private tree inventories, unlike
    // the bounded small journals; stream the same canonical Seal directly.
    let seal: Seal = serde_json::from_reader(std::io::BufReader::new(
        fs::File::open(&seal_path).map_err(|_| "retirement cold seal unavailable")?,
    ))
    .map_err(|_| "retirement cold seal invalid")?;
    let seal_hash = file_digest(&seal_path)?;
    let capture = document(&source.join("request.json"))?;
    need(
        seal.format == 1
            && seal.trees.len() == 3
            && seal.source.volumes.len() == 3
            && immutable(&seal.source.image)
            && capture["format"] == 1
            && capture["action"] == "cold-capture"
            && capture["container"] == seal.source.container
            && capture["project"] == seal.source.project
            && capture["image"] == seal.source.image
            && capture["automatic_retry"] == false
            && request["format"] == 1
            && request["action"] == "isolated-cold-restore"
            && request["seal_sha256"] == seal_hash
            && request["automatic_retry"] == false
            && create["format"] == 1
            && create["action"] == "docker-create-isolated-stopped"
            && create["image"] == seal.source.image
            && create["network"] == "none"
            && create["automatic_retry"] == false
            && complete["format"] == 1
            && complete["action"] == "isolated-cold-restore"
            && complete["complete"] == true
            && complete["directory"] == restored.to_str().unwrap_or("")
            && complete["image"] == seal.source.image
            && complete["network"] == "none"
            && complete["state"] == "created"
            && complete["writers_started"] == false
            && complete["source_untouched"] == true
            && complete["automatic_retry"] == false,
        "retirement restore provenance differs",
    )?;
    need(
        settings_inventory(&source.join("settings"))? == seal.settings,
        "retirement captured settings changed",
    )?;
    let mut trees = Vec::new();
    let mut identities = BTreeSet::new();
    let device = fs::metadata(restored)
        .map_err(|_| "retirement filesystem unavailable")?
        .dev();
    let mut selected = 0u64;
    let mut seen = BTreeSet::new();
    for destination in DESTINATIONS {
        let name = destination.trim_start_matches('/');
        let captured = source.join(name);
        need(
            seal.trees.get(name).is_some_and(|expected| {
                inventory(&captured).is_ok_and(|actual| actual == *expected)
            }),
            "retirement captured bytes or metadata changed",
        )?;
        let tree = restored.join(name);
        need(
            fs::canonicalize(&tree).is_ok_and(|actual| actual == tree),
            "retirement copy path changed",
        )?;
        let meta = fs::symlink_metadata(&tree).map_err(|_| "retirement copy missing")?;
        need(
            meta.is_dir()
                && !meta.file_type().is_symlink()
                && meta.dev() == device
                && identities.insert((meta.dev(), meta.ino())),
            "retirement copy identity invalid",
        )?;
        for protected in [captured]
            .into_iter()
            .chain(seal.source.volumes.iter().map(|v| v.mountpoint.clone()))
        {
            need(
                !overlap(&tree, &protected),
                "retirement copy overlaps retained storage",
            )?;
            let original =
                fs::metadata(protected).map_err(|_| "retirement retained storage unavailable")?;
            need(
                (meta.dev(), meta.ino()) != (original.dev(), original.ino()),
                "retirement copy aliases retained storage",
            )?;
        }
        selected = selected
            .checked_add(tree_allocation(&tree, device, &mut seen)?)
            .ok_or("retirement allocation overflow")?;
        trees.push(tree);
    }
    let container = complete["container"]
        .as_str()
        .ok_or("retirement clone identity missing")?;
    inspect_clone(container, restored, &seal.source, &trees)?;
    Ok(Plan {
        format: 1,
        action: "isolated-cold-retirement-plan".into(),
        container: container.into(),
        restored: restored.into(),
        source,
        source_seal_sha256: seal_hash,
        result_sha256: result.into(),
        trees,
        selected_allocated_bytes: selected,
        filesystem_available_bytes: available(restored)?,
        automatic_retry: false,
        source_preserved: true,
        private_settings_preserved: true,
    })
}

pub(super) fn retire(
    restored: &Path,
    result: &str,
    execute_requested: bool,
) -> Result<serde_json::Value, String> {
    need(
        hex(result, 64),
        "exact successful journey result SHA256 required",
    )?;
    canonical_directory(restored)?;
    let result_path = restored.join("retirement.json");
    if result_path
        .try_exists()
        .map_err(|_| "retirement result unavailable")?
    {
        // Successful retirement removed its clone/trees. Recover the protected
        // outcome by the original restoration provenance, never repeat work.
        let receipt = document(&result_path)?;
        let complete = document(&restored.join("complete.json"))?;
        need(
            receipt["format"] == 1
                && receipt["action"] == "isolated-cold-retirement"
                && receipt["complete"] == true
                && receipt["result_sha256"] == result
                && receipt["restored"] == restored.to_str().unwrap_or("")
                && receipt["container"] == complete["container"]
                && complete["format"] == 1
                && complete["action"] == "isolated-cold-restore"
                && complete["complete"] == true
                && complete["directory"] == restored.to_str().unwrap_or("")
                && receipt["container"].as_str().is_some_and(|id| hex(id, 64))
                && receipt["source_preserved"] == true
                && receipt["private_settings_preserved"] == true
                && receipt["provenance_preserved"] == true
                && receipt["automatic_retry"] == false,
            "retained retirement outcome provenance differs",
        )?;
        return Ok(receipt);
    }
    need(
        !restored
            .join("retirement.intent.json")
            .try_exists()
            .map_err(|_| "retirement intent unavailable")?,
        "retirement was already attempted; inspect retained intent and result, never retry",
    )?;
    let mut plan = verify(restored, result)?;
    let plan_path = restored.join("retirement.plan.json");
    if plan_path
        .try_exists()
        .map_err(|_| "retirement plan unavailable")?
    {
        let old: Plan =
            serde_json::from_value(document(&plan_path)?).map_err(|_| "retirement plan invalid")?;
        // Free space is an observation, not a promised reclamation amount.
        plan.filesystem_available_bytes = old.filesystem_available_bytes;
        need(plan == old, "retirement selection changed since plan")?;
    } else {
        write(&plan_path, &plan)?;
    }
    need(
        !restored
            .join("retirement.intent.json")
            .try_exists()
            .map_err(|_| "retirement intent unavailable")?,
        "retirement was already attempted; inspect retained intent and result, never retry",
    )?;
    if !execute_requested {
        return serde_json::to_value(plan).map_err(|_| "retirement plan encoding failed".into());
    }
    let before = available(restored)?;
    write(
        &restored.join("retirement.intent.json"),
        &serde_json::json!({"format":1,
        "action":"isolated-cold-retirement", "container":plan.container, "result_sha256":result,
        "plan_sha256":file_digest(&plan_path)?, "automatic_retry":false}),
    )?;
    // Revalidate after the durable boundary. Any failure leaves the intent and
    // all remaining storage available for inspection, with no automatic replay.
    let mut now = verify(restored, result)?;
    now.filesystem_available_bytes = plan.filesystem_available_bytes;
    need(now == plan, "retirement state changed after intent")?;
    docker(&["container".into(), "rm".into(), plan.container.clone()])?;
    for tree in &plan.trees {
        no_other_containers(&plan.container, &plan.trees)?;
        no_active_mounts(&plan.trees)?;
        fs::remove_dir_all(tree)
            .map_err(|_| "retirement incomplete; preserve remaining trees and inspect intent")?;
    }
    fs::File::open(restored)
        .and_then(|file| file.sync_all())
        .map_err(|_| "retirement directory sync failed")?;
    let after = available(restored)?;
    let receipt = serde_json::json!({"format":1,"action":"isolated-cold-retirement", "complete":true,
        "container":plan.container,"restored":restored,"result_sha256":result,
        "selected_allocated_bytes":plan.selected_allocated_bytes,
        "filesystem_available_before_bytes":before,"filesystem_available_after_bytes":after,
        "observed_available_delta_bytes":i128::from(after)-i128::from(before),
        "source_preserved":true,"private_settings_preserved":true,"provenance_preserved":true,
        "automatic_retry":false});
    write(&result_path, &receipt)?;
    Ok(receipt)
}
