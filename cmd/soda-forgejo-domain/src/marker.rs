use std::fs;
use std::io::Write;
use std::os::fd::OwnedFd;
use std::path::{Component, Path, PathBuf};

use rustix::fs::{
    openat, openat2, statat, unlinkat, AtFlags, FileType, Mode, OFlags, ResolveFlags, CWD,
};
use rustix::io::Errno;

use crate::system::{Paths, MARKER_NAME};

pub(crate) const HOST_UNIT: &str = include_str!("../../../system/host/services/forgejo.container");
const DATA_PATH_DECLARATION: &str = "Environment=FORGEJO__server__APP_DATA_PATH=";

pub(crate) fn declared_app_data_path(unit: &str) -> Result<&str, String> {
    let mut section = "";
    let mut declaration = None;
    for line in unit.lines().map(str::trim) {
        if line.starts_with('[') && line.ends_with(']') {
            section = &line[1..line.len() - 1];
        } else if let Some(path) = line.strip_prefix(DATA_PATH_DECLARATION) {
            if section != "Container" {
                return Err("Forgejo data path declaration must be in [Container]".to_string());
            }
            if declaration.replace(path).is_some() {
                return Err("Forgejo data path is declared more than once".to_string());
            }
        }
    }
    declaration.ok_or_else(|| "Forgejo data path is not declared in forgejo.container".to_string())
}

/// A marker parent opened without traversing a symlink. Operations stay
/// relative to this retained directory even if its pathname is later replaced.
#[derive(Debug)]
pub(crate) struct MarkerLocation {
    path: PathBuf,
    parent: Option<OwnedFd>,
}

impl MarkerLocation {
    pub(crate) fn path(&self) -> &Path {
        &self.path
    }

    /// Only ENOENT means the marker is absent. In particular, symlinks,
    /// non-regular marker entries, and other metadata failures are errors.
    pub(crate) fn is_present(&self) -> Result<bool, String> {
        let Some(parent) = &self.parent else {
            return Ok(false);
        };
        match statat(parent, MARKER_NAME, AtFlags::SYMLINK_NOFOLLOW) {
            Ok(stat) if FileType::from_raw_mode(stat.st_mode).is_file() => Ok(true),
            Ok(_) => Err(format!(
                "marker {} is not a regular file",
                self.path.display()
            )),
            Err(Errno::NOENT) => Ok(false),
            Err(err) => Err(format!(
                "cannot inspect marker {}: {err}",
                self.path.display()
            )),
        }
    }

    /// Create a new marker without truncating an existing inode. An existing
    /// regular marker already satisfies inhibit and is left byte-for-byte
    /// untouched.
    pub(crate) fn create(&self) -> Result<(), String> {
        let Some(parent) = &self.parent else {
            return Err(format!(
                "marker parent {} is absent; refusing to create marker",
                self.path.parent().unwrap_or(&self.path).display()
            ));
        };
        let fd = match openat(
            parent,
            MARKER_NAME,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        ) {
            Ok(fd) => fd,
            Err(Errno::EXIST) if self.is_present()? => return Ok(()),
            Err(err) => {
                return Err(format!(
                    "cannot create marker {}: {err}",
                    self.path.display()
                ))
            }
        };
        rustix::fs::fchmod(&fd, Mode::from_raw_mode(0o600))
            .map_err(|err| format!("cannot set marker mode {}: {err}", self.path.display()))?;
        let mut file = fs::File::from(fd);
        file.write_all(b"offline recovery\n")
            .map_err(|err| format!("cannot write marker {}: {err}", self.path.display()))
    }

    /// Remove only a regular marker entry from the retained parent directory.
    pub(crate) fn remove(&self) -> Result<bool, String> {
        let Some(parent) = &self.parent else {
            return Ok(false);
        };
        if !self.is_present()? {
            return Ok(false);
        }
        unlinkat(parent, MARKER_NAME, AtFlags::empty())
            .map_err(|err| format!("cannot remove marker {}: {err}", self.path.display()))?;
        Ok(true)
    }
}

fn relative_components(path: &Path) -> Result<(), String> {
    if path
        .components()
        .any(|component| !matches!(component, Component::Normal(_)))
    {
        return Err(format!(
            "marker path component in {} is not a normal relative component",
            path.display()
        ));
    }
    Ok(())
}

fn open_directory_beneath(
    root: &OwnedFd,
    relative: &Path,
    display: &Path,
) -> Result<Option<OwnedFd>, String> {
    relative_components(relative)?;
    let relative = if relative.as_os_str().is_empty() {
        Path::new(".")
    } else {
        relative
    };
    match openat2(
        root,
        relative,
        OFlags::RDONLY | OFlags::DIRECTORY | OFlags::CLOEXEC,
        Mode::empty(),
        ResolveFlags::BENEATH | ResolveFlags::NO_SYMLINKS,
    ) {
        Ok(fd) => Ok(Some(fd)),
        Err(Errno::NOENT) => Ok(None),
        Err(err) => Err(format!(
            "cannot safely open directory {}: {err}",
            display.display()
        )),
    }
}

/// Resolve the declared container data path to a marker location. The marker
/// parent is held by descriptor; missing parents are known absence for
/// readers but cannot be used to create a marker.
pub(crate) fn marker_path(paths: &Paths) -> Result<MarkerLocation, String> {
    let app_data = declared_app_data_path(HOST_UNIT)?;
    let app_data = app_data.trim_end_matches('/');
    if app_data != "/data" && !app_data.starts_with("/data/") {
        return Err(format!(
            "declared AppDataPath {app_data:?} is outside the /data volume; cannot map to the host"
        ));
    }
    let relative = if app_data == "/data" {
        ""
    } else {
        &app_data["/data/".len()..]
    };
    let relative_path = Path::new(relative);
    relative_components(relative_path)?;

    let root = openat(
        CWD,
        "/",
        OFlags::RDONLY | OFlags::DIRECTORY | OFlags::CLOEXEC | OFlags::NOFOLLOW,
        Mode::empty(),
    )
    .map_err(|err| format!("cannot open filesystem root for marker resolution: {err}"))?;
    let parent_path = paths.data_root.join(relative_path);
    let parent_relative = parent_path.strip_prefix("/").map_err(|_| {
        format!(
            "marker parent {} is not absolute; refusing to resolve marker",
            parent_path.display()
        )
    })?;
    let parent = open_directory_beneath(&root, parent_relative, &parent_path)?;
    let path = parent_path.join(MARKER_NAME);
    Ok(MarkerLocation { path, parent })
}
