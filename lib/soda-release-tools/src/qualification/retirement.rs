//! Fixed collector scratch, not a general path-pruning interface. Failed work
//! remains private and outside sealed evidence. Only completed, permitted work
//! can be consumed; an interrupted deletion is never replayed automatically.
use super::{evidence, hash, input, lock, read, write_new, Request};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeSet,
    ffi::CString,
    fs,
    os::unix::fs::{MetadataExt, PermissionsExt},
    path::{Path, PathBuf},
};

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Origin {
    format: u8,
    device: u64,
    inode: u64,
    uid: u32,
    gid: u32,
}

fn need(ok: bool, reason: &str) -> Result<(), String> {
    if ok {
        Ok(())
    } else {
        Err(reason.into())
    }
}

pub(super) fn record_origin(out: &Path, attempt: &Path) -> Result<(), String> {
    let m =
        fs::symlink_metadata(attempt.join(".work")).map_err(|_| "collector work unavailable")?;
    need(
        m.is_dir() && !m.file_type().is_symlink() && m.mode() & 0o7777 == 0o700,
        "private collector work required",
    )?;
    write_new(
        &out.join("work-origin.json"),
        &Origin {
            format: 1,
            device: m.dev(),
            inode: m.ino(),
            uid: m.uid(),
            gid: m.gid(),
        },
    )
}

pub(super) fn preserve(out: &Path, attempt: &Path) -> Result<(), String> {
    // Caller has confirmed the exact worker unit terminal. Reclaim only its
    // fixed parent, then move the original scratch inode out of evidence.
    let origin: Origin = read(&out.join("work-origin.json"))?;
    let source = attempt.join(".work");
    let m = fs::symlink_metadata(&source).map_err(|_| "collector work missing")?;
    need(
        origin.format == 1
            && m.is_dir()
            && !m.file_type().is_symlink()
            && m.dev() == origin.device
            && m.ino() == origin.inode
            && m.uid() == origin.uid,
        "collector work origin changed; preserve attempt",
    )?;
    for path in [attempt, source.as_path()] {
        let c = CString::new(path.as_os_str().as_encoded_bytes())
            .map_err(|_| "collector work path invalid")?;
        need(
            unsafe { libc::chown(c.as_ptr(), 0, 0) } == 0,
            "collector work custody unavailable",
        )?;
        fs::set_permissions(path, fs::Permissions::from_mode(0o700))
            .map_err(|_| "collector work custody unavailable")?;
    }
    let destination = out.join("consumed-work");
    let from = CString::new(source.as_os_str().as_encoded_bytes())
        .map_err(|_| "collector work path invalid")?;
    let to = CString::new(destination.as_os_str().as_encoded_bytes())
        .map_err(|_| "collector work path invalid")?;
    need(
        unsafe {
            libc::renameat2(
                libc::AT_FDCWD,
                from.as_ptr(),
                libc::AT_FDCWD,
                to.as_ptr(),
                libc::RENAME_NOREPLACE,
            )
        } == 0,
        "collector work destination occupied or preservation failed",
    )?;
    fs::File::open(out)
        .and_then(|f| f.sync_all())
        .map_err(|_| "collector work preservation sync failed")?;
    write_new(
        &out.join("work-preserved.json"),
        &serde_json::json!({"format":1,"state":"retained","resource":"collector-scratch",
        "origin_sha256":hash(&out.join("work-origin.json"))?,"request_sha256":hash(&out.join("request.json"))?,"path":destination,
        "outside_sealed_evidence":true,"automatic_retry":false}),
    )
}

#[derive(Clone, Serialize, PartialEq, Eq)]
struct Item {
    relative: PathBuf,
    device: u64,
    inode: u64,
    mode: u32,
    uid: u32,
    gid: u32,
    bytes: u64,
    blocks: u64,
    modified: i64,
    nanos: i64,
}
fn inventory(root: &Path) -> Result<Vec<Item>, String> {
    let device = fs::symlink_metadata(root)
        .map_err(|_| "collector scratch unavailable")?
        .dev();
    fn walk(root: &Path, path: &Path, device: u64, values: &mut Vec<Item>) -> Result<(), String> {
        let m =
            fs::symlink_metadata(path).map_err(|_| "collector scratch inventory unavailable")?;
        need(
            m.dev() == device
                && values.len() < 250_000
                && (m.is_dir() || m.file_type().is_symlink() || m.is_file() && m.nlink() == 1),
            "scratch has mount, shared file or unknown resource; preserve",
        )?;
        values.push(Item {
            relative: path
                .strip_prefix(root)
                .map_err(|_| "scratch path outside scope")?
                .to_owned(),
            device: m.dev(),
            inode: m.ino(),
            mode: m.mode(),
            uid: m.uid(),
            gid: m.gid(),
            bytes: m.len(),
            blocks: m.blocks(),
            modified: m.mtime(),
            nanos: m.mtime_nsec(),
        });
        if m.is_dir() {
            let mut entries = fs::read_dir(path)
                .map_err(|_| "scratch directory unavailable")?
                .map(|e| e.map(|e| e.path()))
                .collect::<Result<Vec<_>, _>>()
                .map_err(|_| "scratch directory unavailable")?;
            entries.sort();
            for child in entries {
                walk(root, &child, device, values)?;
            }
        }
        Ok(())
    }
    let mut values = Vec::new();
    walk(root, root, device, &mut values)?;
    Ok(values)
}
fn free_bytes(path: &Path) -> Result<u64, String> {
    let c = CString::new(path.as_os_str().as_encoded_bytes())
        .map_err(|_| "scratch filesystem path invalid")?;
    let mut stat = std::mem::MaybeUninit::<libc::statvfs>::uninit();
    need(
        unsafe { libc::statvfs(c.as_ptr(), stat.as_mut_ptr()) } == 0,
        "scratch filesystem observation unavailable",
    )?;
    let stat = unsafe { stat.assume_init() };
    stat.f_bavail
        .checked_mul(stat.f_frsize)
        .ok_or("scratch filesystem size overflow".into())
}
fn no_users(root: &Path) -> Result<(), String> {
    // Do not unlink a mounted tree, open VM backing file or live process cwd.
    let mounts =
        fs::read_to_string("/proc/self/mountinfo").map_err(|_| "mount inventory unavailable")?;
    let decode = |s: &str| {
        s.replace("\\040", " ")
            .replace("\\011", "\t")
            .replace("\\012", "\n")
            .replace("\\134", "\\")
    };
    for line in mounts.lines() {
        if let Some(m) = line.split_whitespace().nth(4) {
            need(
                !Path::new(&decode(m)).starts_with(root),
                "scratch has active mount; preserve",
            )?;
        }
    }
    for process in fs::read_dir("/proc").map_err(|_| "process inventory unavailable")? {
        let process = process.map_err(|_| "process inventory unavailable")?;
        if !process
            .file_name()
            .to_string_lossy()
            .bytes()
            .all(|b| b.is_ascii_digit())
        {
            continue;
        }
        let p = process.path();
        let mut paths = vec![p.join("cwd"), p.join("root")];
        match fs::read_dir(p.join("fd")) {
            Ok(files) => {
                for f in files {
                    paths.push(f.map_err(|_| "process references unavailable")?.path());
                }
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return Err("process references unconfirmed; preserve".into()),
        }
        for path in paths {
            if let Ok(target) = fs::read_link(path) {
                need(
                    !target.starts_with(root),
                    "scratch has live process reference; preserve",
                )?;
            }
        }
    }
    Ok(())
}

pub(crate) fn retire(out: &Path, execute: bool) -> Result<serde_json::Value, String> {
    need(
        out.starts_with("/home") && fs::canonicalize(out).is_ok_and(|actual| actual == out),
        "canonical scratch collection under /home required",
    )?;
    let _lock = lock(out)?;
    // Finalized receipt binds the original controller/request. Scratch
    // retirement never rereads credentials or rehashes consumed media inputs.
    evidence::protected_path(&out.join("request.json"), false).map_err(|e| e.to_string())?;
    let request: Request = read(&out.join("request.json"))?;
    let unit_id = request
        .unit
        .strip_prefix("soda-qualify-")
        .and_then(|s| s.strip_suffix(".service"))
        .ok_or("scratch worker unit invalid")?;
    need(
        unit_id.len() == 32
            && unit_id
                .bytes()
                .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase())
            && request.binding.unit == request.unit
            && request.attempt == out.join("attempt"),
        "scratch worker binding invalid",
    )?;
    if out.join("scratch-retirement.json").exists() {
        evidence::protected_path(&out.join("scratch-retirement.json"), false)
            .map_err(|e| e.to_string())?;
        let result: serde_json::Value = read(&out.join("scratch-retirement.json"))?;
        need(
            result["format"] == 1
                && result["resource"] == "collector-scratch"
                && result["state"] == "removed"
                && result["request_sha256"] == hash(&out.join("request.json"))?
                && result["unit"] == request.unit
                && result["path"] == out.join("consumed-work").to_string_lossy().as_ref(),
            "retained scratch outcome binding changed",
        )?;
        return Ok(result);
    }
    if out.join("scratch-retirement-intent.json").exists() {
        return Ok(
            serde_json::json!({"format":1,"resource":"collector-scratch","state":"uncertain","reason":"retirement_intent_without_result","automatic_retry":false}),
        );
    }
    evidence::protected_path(&out.join("work-origin.json"), false).map_err(|e| e.to_string())?;
    evidence::protected_path(&out.join("work-preserved.json"), false).map_err(|e| e.to_string())?;
    let origin: Origin = read(&out.join("work-origin.json"))?;
    let work = out.join("consumed-work");
    evidence::protected_path(&work, true).map_err(|e| e.to_string())?;
    let meta = fs::symlink_metadata(&work).map_err(|_| "preserved scratch unavailable")?;
    need(
        meta.dev() == origin.device && meta.ino() == origin.inode,
        "preserved scratch origin changed",
    )?;
    let children: BTreeSet<_> = fs::read_dir(&work)
        .map_err(|_| "scratch children unavailable")?
        .map(|e| e.map(|e| e.file_name()))
        .collect::<Result<_, _>>()
        .map_err(|_| "scratch children unavailable")?;
    need(
        children.iter().all(|n| {
            ["home", "tmp", "profile", "downloads"]
                .iter()
                .any(|v| n == *v)
        }),
        "scratch contains unrecognized resource; preserve",
    )?;
    let values = inventory(&work)?;
    no_users(&work)?;
    let terminal = crate::worker::execution::systemctl_unit_terminal(&request.unit)?;
    let complete = match evidence::read_protected(&out.join("observations")) {
        Ok(actual) => {
            let mut expected = request.binding.clone();
            expected.exit_code = 0;
            expected.native_retirement = "confirmed".into();
            expected.files = actual.files.clone();
            actual == expected
        }
        Err(_) => false,
    };
    let scope = match &request.resolved_input {
        Some(reference) => {
            input::verify(reference)?;
            read::<serde_json::Value>(&reference.path)?
        }
        None => serde_json::Value::Null,
    };
    let permitted = scope.pointer("/scenario/authority/held")
        == Some(&serde_json::Value::Bool(false))
        && scope.pointer("/scenario/permitted_effects/retire_consumed_scratch")
            == Some(&serde_json::Value::Bool(true));
    let bytes = values
        .iter()
        .filter(|v| v.mode & libc::S_IFMT == libc::S_IFREG)
        .try_fold(0u64, |n, v| {
            n.checked_add(v.bytes).ok_or("scratch byte overflow")
        })?;
    let allocated = values.iter().try_fold(0u64, |n, v| {
        n.checked_add(v.blocks.saturating_mul(512))
            .ok_or("scratch allocation overflow")
    })?;
    let before = free_bytes(out)?;
    let plan = serde_json::json!({"format":1,"resource":"collector-scratch","state":"retained","request_sha256":hash(&out.join("request.json"))?,"unit":request.unit,
        "path":work,"origin_sha256":hash(&out.join("work-origin.json"))?,"entries":values.len(),"selected_bytes":bytes,"selected_allocated_bytes":allocated,
        "completed_evidence":complete,"worker_terminal":terminal,"scope_permitted":permitted,"filesystem_free_before_bytes":before,"automatic_retry":false});
    if !execute {
        return Ok(plan);
    }
    need(complete&&terminal&&permitted,"scratch retirement requires successful finalized evidence, terminal worker and explicit scope")?;
    write_new(&out.join("scratch-retirement-intent.json"), &plan)?;
    need(
        inventory(&work)? == values,
        "scratch changed after intent; preserve",
    )?;
    no_users(&work)?;
    fs::remove_dir_all(&work)
        .map_err(|_| "scratch removal incomplete; preserve remaining state without retry")?;
    fs::File::open(out)
        .and_then(|f| f.sync_all())
        .map_err(|_| "scratch retirement sync uncertain")?;
    let mut result = plan;
    result["state"] = "removed".into();
    result["filesystem_free_after_bytes"] = free_bytes(out)?.into();
    write_new(&out.join("scratch-retirement.json"), &result)?;
    Ok(result)
}

pub(super) fn automatic(out: &Path) -> Result<(), String> {
    // Older completed observations have no retained scratch origin; their
    // evidence is immutable and never upgraded into a new cleanup authority.
    if !out.join("work-origin.json").exists() {
        return Ok(());
    }
    let request: Request = read(&out.join("request.json"))?;
    let Some(reference) = request.resolved_input else {
        return Ok(());
    };
    input::verify(&reference)?;
    let value: serde_json::Value = read(&reference.path)?;
    // The common default preserves work without a filesystem/process scan.
    if value.pointer("/scenario/authority/held") == Some(&serde_json::Value::Bool(false))
        && value.pointer("/scenario/permitted_effects/retire_consumed_scratch")
            == Some(&serde_json::Value::Bool(true))
    {
        retire(out, true)?;
    }
    Ok(())
}
