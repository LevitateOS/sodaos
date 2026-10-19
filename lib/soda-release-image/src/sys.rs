//! Small exact mirrors of the foreign `release/build` + `release/deliver`
//! helpers the pipeline calls: path lexics, file hashing, exclusive writes,
//! fresh directories, bounded strict JSON reads, native admission, command
//! inventory, and the private-file check. Heavy foreign operations (OCI,
//! signing, live resolution) live behind [`crate::foreign::Production`].

use std::collections::{BTreeMap, BTreeSet, VecDeque};
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Component, Path, PathBuf};

use crate::error::Error;
use crate::jsonio;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

/// Shared inventory primitive (`build.File`).
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct File {
    #[serde(rename = "sha256", skip_serializing_if = "String::is_empty")]
    pub sha256: String,
    #[serde(rename = "mode")]
    pub mode: u32,
    #[serde(rename = "link", skip_serializing_if = "String::is_empty")]
    pub link: String,
    #[serde(rename = "directory", skip_serializing_if = "is_false")]
    pub directory: bool,
}

fn is_false(value: &bool) -> bool {
    !value
}

impl File {
    pub fn parse(text: &str) -> Result<File, Error> {
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(File, {
    sha256: String => "sha256",
    mode: u32 => "mode",
    link: String => "link",
    directory: bool => "directory",
});

/// Render Rust path components without rewriting parent-directory components.
/// Admission callers compare this representation with the original input to
/// reject non-clean paths rather than silently changing their target.
pub fn clean_path(path: &str) -> String {
    let components: PathBuf = Path::new(path).components().collect();
    if components.as_os_str().is_empty() {
        ".".to_string()
    } else {
        components.to_string_lossy().into_owned()
    }
}

/// True when a path already has its exact Rust component representation and
/// contains no current or parent-directory components.
pub fn is_clean_path(path: &str) -> bool {
    if path.is_empty() {
        return false;
    }
    let mut rebuilt = PathBuf::new();
    for component in Path::new(path).components() {
        match component {
            Component::RootDir => rebuilt.push("/"),
            Component::Normal(part) => rebuilt.push(part),
            Component::CurDir | Component::ParentDir | Component::Prefix(_) => return false,
        }
    }
    rebuilt.as_os_str() == Path::new(path).as_os_str()
}

pub fn is_clean_abs(path: &str) -> bool {
    Path::new(path).is_absolute() && is_clean_path(path)
}

pub fn is_abs(path: &str) -> bool {
    path.starts_with('/')
}

pub fn join<S: AsRef<str>>(parts: &[S]) -> String {
    let mut buf = PathBuf::new();
    for part in parts {
        buf.push(part.as_ref());
    }
    buf.to_string_lossy().into_owned()
}

pub fn dir_name(path: &str) -> String {
    let path = Path::new(path);
    path.parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| {
            if path.is_absolute() {
                Path::new("/")
            } else {
                Path::new(".")
            }
        })
        .to_string_lossy()
        .into_owned()
}

pub fn base_name(path: &str) -> String {
    let path = Path::new(path);
    path.file_name()
        .map(|name| name.to_string_lossy().into_owned())
        .unwrap_or_else(|| if path.is_absolute() { "/" } else { "." }.to_string())
}

/// Build a lexical relative path from Rust path components.
pub fn rel_path(base: &str, target: &str) -> Result<String, Error> {
    let b: Vec<_> = Path::new(base).components().collect();
    let t: Vec<_> = Path::new(target).components().collect();
    if b.first().map(|part| matches!(part, Component::RootDir))
        != t.first().map(|part| matches!(part, Component::RootDir))
    {
        return Err(Error::msg("cannot make relative path"));
    }
    let mut common = 0;
    while common < b.len() && common < t.len() && b[common] == t[common] {
        common += 1;
    }
    let mut out = PathBuf::new();
    for _ in common..b.len() {
        out.push("..");
    }
    for component in &t[common..] {
        out.push(component.as_os_str());
    }
    if out.as_os_str().is_empty() {
        return Ok(".".to_string());
    }
    Ok(out.to_string_lossy().into_owned())
}

pub fn to_slash(path: &str) -> String {
    path.to_string()
}

/// Lexical path components without `.`/`..` surprises, for walk inventories.
pub fn components(path: &Path) -> Vec<String> {
    path.components()
        .filter_map(|c| match c {
            Component::Normal(part) => Some(part.to_string_lossy().into_owned()),
            _ => None,
        })
        .collect()
}

/// `build.HashFile`: SHA-256 of a regular non-symlink file.
pub fn hash_file(path: &str) -> Result<String, Error> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.file_type().is_file() {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let data = fs::read(path)?;
    Ok(hex_sha256(&data))
}

pub fn hex_sha256(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex_bytes(&hasher.finalize())
}

pub fn hex_bytes(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

/// `build.FreshDirectory`: create a new mode-700 directory below a real parent.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !is_clean_abs(path) {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = dir_name(path);
    if !is_clean_abs(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    let resolved = fs::canonicalize(&parent).map_err(|e| Error::msg(e.to_string()))?;
    if resolved.to_string_lossy() != clean_path(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))?;
    Ok(())
}

/// `build.WriteNew`: exclusive-create write; refuses to overwrite.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::fs::OpenOptions;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|e| Error::msg(e.to_string()))?;
    use std::io::Write;
    let write_result = file.write_all(data).map_err(Error::from);
    drop(file);
    write_result?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

fn read_bounded(path: &str, maximum: usize, too_big: &str) -> Result<Vec<u8>, Error> {
    use std::os::unix::fs::OpenOptionsExt;

    let mut options = fs::OpenOptions::new();
    options
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK);
    let file = options.open(path).map_err(|error| {
        if error.raw_os_error() == Some(libc::ELOOP) {
            Error::msg("bounded regular JSON input required")
        } else {
            Error::from(error)
        }
    })?;
    let meta = file.metadata()?;
    if !meta.file_type().is_file() || meta.len() > maximum as u64 {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let data = read_limited(file, maximum)?.ok_or_else(|| Error::msg(too_big))?;
    Ok(data)
}

/// Keep the exact bounded input text available to the schema-specific DTO
/// visitor. The raw JSON bytes remain the authority for every consumer that
/// hashes or signs these files.
pub fn read_json_build_text(path: &str) -> Result<String, Error> {
    let data = read_bounded(path, 4 << 20, "JSON input exceeds limit")?;
    String::from_utf8(data).map_err(|_| Error::msg("invalid JSON"))
}

/// Deliver's bounded text counterpart for schema-specific raw-slot visitors.
pub fn read_json_deliver_text(path: &str) -> Result<String, Error> {
    use std::os::unix::fs::OpenOptionsExt;

    let mut options = fs::OpenOptions::new();
    options
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK);
    let file = options.open(path).map_err(|_| Error::msg(refused()))?;
    let meta = file.metadata().map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.len() > (1 << 20) as u64 {
        return Err(Error::msg(refused()));
    }
    let data = read_limited(file, 1 << 20)
        .map_err(|_| Error::msg(refused()))?
        .ok_or_else(|| Error::msg(refused()))?;
    String::from_utf8(data).map_err(|_| Error::msg(refused()))
}

fn read_limited(file: fs::File, maximum: usize) -> std::io::Result<Option<Vec<u8>>> {
    use std::io::Read;
    let mut data = Vec::new();
    file.take(maximum as u64 + 1).read_to_end(&mut data)?;
    Ok((data.len() <= maximum).then_some(data))
}

pub fn refused() -> String {
    "release authority or completeness refused".to_string()
}

/// `deliver.PrivateFile`: absolute, symlink-free, owned private regular file.
pub fn private_file(path: &str) -> Result<(), Error> {
    if !is_clean_abs(path) {
        return Err(Error::msg(refused()));
    }
    let resolved = fs::canonicalize(path).map_err(|_| Error::msg(refused()))?;
    if resolved.to_string_lossy() != clean_path(path) {
        return Err(Error::msg(refused()));
    }
    let meta = fs::symlink_metadata(path).map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.permissions().mode() & 0o077 != 0 {
        return Err(Error::msg(refused()));
    }
    Ok(())
}

/// `build.RequireNative`: x86_64 Linux only.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let oci =
        soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))?;
    // D01-F1: Rust target arch and OCI arch are separate namespaces; each
    // is validated against its own supported value.
    if std::env::consts::OS != "linux" || std::env::consts::ARCH != "x86_64" || oci != "amd64" {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// CORR-C-001: a cmd directory owned by Rust carries its own manifest. A
/// Go command directory never has a top-level Cargo.toml, so its presence
/// selects the Rust recipe; anything else stays on the Go recipe.
pub fn is_rust_command(cmd_dir: &str) -> bool {
    fs::metadata(join(&[cmd_dir, "Cargo.toml"]))
        .map(|meta| meta.is_file())
        .unwrap_or(false)
}

#[derive(Debug, Deserialize)]
struct CargoMetadata {
    workspace_root: String,
    workspace_members: Vec<String>,
    packages: Vec<CargoPackageMetadata>,
}

#[derive(Debug, Deserialize)]
struct CargoPackageMetadata {
    id: String,
    name: String,
    manifest_path: String,
    #[serde(default)]
    features: BTreeMap<String, Vec<String>>,
    targets: Vec<CargoTargetMetadata>,
}

#[derive(Debug, Deserialize)]
struct CargoTargetMetadata {
    name: String,
    kind: Vec<String>,
    #[serde(default, rename = "required-features")]
    required_features: Vec<String>,
}

#[derive(Debug, Clone)]
struct ShippingPackage {
    id: String,
    name: String,
    manifest_path: PathBuf,
    bins: BTreeMap<String, BTreeSet<String>>,
    default_features: BTreeSet<String>,
}

/// Immutable Cargo target inventory for one admitted workspace snapshot.
#[derive(Debug, Clone)]
pub struct ShippingInventory {
    workspace_root: PathBuf,
    packages: Vec<ShippingPackage>,
    commands: Vec<String>,
}

fn active_default_features(features: &BTreeMap<String, Vec<String>>) -> BTreeSet<String> {
    let mut active = BTreeSet::new();
    let mut pending = VecDeque::new();
    if features.contains_key("default") {
        active.insert("default".to_string());
        pending.push_back("default".to_string());
    }
    while let Some(feature) = pending.pop_front() {
        for item in features.get(&feature).into_iter().flatten() {
            if features.contains_key(item) && active.insert(item.clone()) {
                pending.push_back(item.clone());
            }
        }
    }
    active
}

/// Parse Cargo's tolerant metadata schema and bind it to the admitted
/// workspace. Package IDs remain opaque strings throughout.
pub fn parse_shipping_inventory(snapshot: &str, json: &str) -> Result<ShippingInventory, Error> {
    let root = fs::canonicalize(snapshot).map_err(|_| Error::msg("invalid Cargo workspace"))?;
    let manifest = fs::canonicalize(root.join("Cargo.toml"))
        .map_err(|_| Error::msg("invalid Cargo workspace"))?;
    let metadata: CargoMetadata =
        jsonio::parse(json).map_err(|_| Error::msg("invalid Cargo metadata"))?;
    let metadata_root = fs::canonicalize(&metadata.workspace_root)
        .map_err(|_| Error::msg("invalid Cargo metadata"))?;
    if metadata_root != root || !manifest.starts_with(&root) || !manifest.is_file() {
        return Err(Error::msg("Cargo metadata workspace mismatch"));
    }
    let workspace_ids: BTreeSet<&str> = metadata
        .workspace_members
        .iter()
        .map(String::as_str)
        .collect();
    if workspace_ids.len() != metadata.workspace_members.len() {
        return Err(Error::msg("invalid Cargo workspace membership"));
    }
    let mut found_ids = BTreeSet::new();
    let mut manifests = BTreeSet::new();
    let mut packages = Vec::new();
    for package in metadata.packages {
        if !workspace_ids.contains(package.id.as_str()) {
            continue;
        }
        if package.id.is_empty() || package.name.is_empty() || !found_ids.insert(package.id.clone())
        {
            return Err(Error::msg("invalid Cargo package identity"));
        }
        packages.push(admit_workspace_package(&root, &mut manifests, package)?);
    }
    if found_ids.len() != workspace_ids.len() {
        return Err(Error::msg("Cargo metadata omitted workspace package"));
    }
    Ok(ShippingInventory {
        workspace_root: root,
        packages,
        commands: Vec::new(),
    })
}

fn admit_workspace_package(
    root: &std::path::Path,
    manifests: &mut BTreeSet<std::path::PathBuf>,
    package: CargoPackageMetadata,
) -> Result<ShippingPackage, Error> {
    let path = fs::canonicalize(&package.manifest_path)
        .map_err(|_| Error::msg("invalid Cargo package manifest"))?;
    if !path.starts_with(root)
        || !path.is_file()
        || path.file_name().and_then(|name| name.to_str()) != Some("Cargo.toml")
        || !manifests.insert(path.clone())
    {
        return Err(Error::msg("Cargo package manifest outside workspace"));
    }
    let default_features = active_default_features(&package.features);
    let mut bins = BTreeMap::new();
    for target in package.targets {
        if !target.kind.iter().any(|kind| kind == "bin") {
            continue;
        }
        let required: BTreeSet<String> = target.required_features.into_iter().collect();
        if bins.insert(target.name, required).is_some() {
            return Err(Error::msg("duplicate Cargo binary target"));
        }
    }
    Ok(ShippingPackage {
        id: package.id,
        name: package.name,
        manifest_path: path,
        bins,
        default_features,
    })
}

impl ShippingInventory {
    fn package_at(&self, manifest: &Path) -> Result<Option<&ShippingPackage>, Error> {
        let path = fs::canonicalize(manifest).map_err(|_| Error::msg("Cargo manifest missing"))?;
        if !path.starts_with(&self.workspace_root) {
            return Err(Error::msg("Cargo manifest outside admitted workspace"));
        }
        Ok(self
            .packages
            .iter()
            .find(|package| package.manifest_path == path))
    }

    /// Confirms one actual bin target and that its required feature set is
    /// enabled by Cargo's default-feature build recipe.
    pub fn require_bin(&self, package_name: &str, bin: &str) -> Result<(), Error> {
        let mut matches = self
            .packages
            .iter()
            .filter(|package| package.name == package_name);
        let package = matches
            .next()
            .ok_or_else(|| Error::msg("Cargo package target missing"))?;
        if matches.next().is_some() {
            return Err(Error::msg("ambiguous Cargo package target"));
        }
        Self::require_package_bin(package, bin)?;
        // Read the opaque ID as part of the admitted identity without parsing
        // Cargo's version-sensitive ID representation.
        if package.id.is_empty() {
            return Err(Error::msg("invalid Cargo package identity"));
        }
        Ok(())
    }

    fn contains_bin_at(&self, manifest: &Path, bin: &str) -> Result<bool, Error> {
        let Some(package) = self.package_at(manifest)? else {
            return Err(Error::msg("Cargo metadata omitted command manifest"));
        };
        let Some(required) = package.bins.get(bin) else {
            return Ok(false);
        };
        if required
            .iter()
            .any(|feature| !package.default_features.contains(feature))
        {
            return Err(Error::msg("Cargo binary requires a non-default feature"));
        }
        Ok(true)
    }

    pub fn require_bin_at(&self, manifest: &str, bin: &str) -> Result<(), Error> {
        let package = self
            .package_at(Path::new(manifest))?
            .ok_or_else(|| Error::msg("Cargo metadata omitted command manifest"))?;
        Self::require_package_bin(package, bin)
    }

    fn require_package_bin(package: &ShippingPackage, bin: &str) -> Result<(), Error> {
        let required = package
            .bins
            .get(bin)
            .ok_or_else(|| Error::msg("Cargo binary target missing"))?;
        if required
            .iter()
            .any(|feature| !package.default_features.contains(feature))
        {
            return Err(Error::msg("Cargo binary requires a non-default feature"));
        }
        Ok(())
    }

    pub fn package_name_for_manifest(&self, manifest: &str) -> Result<&str, Error> {
        self.package_name_at(Path::new(manifest))
    }

    fn has_bins_at(&self, manifest: &Path) -> Result<bool, Error> {
        self.package_at(manifest)?
            .map(|package| !package.bins.is_empty())
            .ok_or_else(|| Error::msg("Cargo metadata omitted command manifest"))
    }

    fn package_name_at(&self, manifest: &Path) -> Result<&str, Error> {
        self.package_at(manifest)?
            .map(|package| package.name.as_str())
            .ok_or_else(|| Error::msg("Cargo metadata omitted command manifest"))
    }

    pub fn commands(&self) -> &[String] {
        &self.commands
    }

    /// Freeze the existing appliance-command policy into this snapshot's
    /// inventory. `tool_members` comes from the authoritative RUST_TOOLS table.
    pub fn select_commands(
        mut self,
        source: &str,
        tool_members: &[&str],
    ) -> Result<ShippingInventory, Error> {
        let mut names = Vec::new();
        let entries = fs::read_dir(join(&[source, "cmd"]))?;
        let mut dirs: Vec<String> = entries
            .filter_map(|entry| entry.ok())
            .filter(|entry| entry.file_type().map(|t| t.is_dir()).unwrap_or(false))
            .map(|entry| entry.file_name().to_string_lossy().into_owned())
            .collect();
        dirs.sort();
        for name in dirs {
            let cmd_dir = join(&[source, "cmd", &name]);
            if is_rust_command(&cmd_dir) {
                let manifest = PathBuf::from(&cmd_dir).join("Cargo.toml");
                let package_name = self.package_name_at(&manifest)?;
                if self.has_bins_at(&manifest)? && !self.contains_bin_at(&manifest, &name)? {
                    continue;
                }
                if tool_members.contains(&package_name) {
                    continue;
                }
            }
            if name == "soda-forgejo-tailnet" && !has_go_sources(&cmd_dir) {
                continue;
            }
            if !is_soda_command(&name) || name == "soda-artifacts" || name == "soda-acceptance" {
                return Err(Error::msg(
                    "support tools must remain outside appliance commands",
                ));
            }
            names.push(name);
        }
        if names.is_empty() {
            return Err(Error::msg("missing Soda commands"));
        }
        self.commands = names;
        Ok(self)
    }
}

/// N07-T3: true when a cmd directory still carries Go sources. Unreadable
/// directories report true so discovery keeps its previous listing; only a
/// readable directory with no `.go` files counts as retired.
fn has_go_sources(cmd_dir: &str) -> bool {
    fs::read_dir(cmd_dir)
        .map(|entries| {
            entries.filter_map(|entry| entry.ok()).any(|entry| {
                entry
                    .file_type()
                    .map(|kind| kind.is_file())
                    .unwrap_or(false)
                    && entry.file_name().to_string_lossy().ends_with(".go")
            })
        })
        .unwrap_or(true)
}

fn is_soda_command(name: &str) -> bool {
    let rest = match name.strip_prefix("soda-") {
        Some(rest) => rest,
        None => return false,
    };
    !rest.is_empty()
        && rest
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

/// Mount mode bits for a freshly created directory (`os.MkdirAll` parity).
pub fn mkdir_all(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir_all(path)?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

pub fn create_dir(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

/// Recursive directory walk yielding (path, is_dir, is_symlink).
pub fn walk<F>(root: &str, mut visit: F) -> Result<(), Error>
where
    F: FnMut(&str, bool, bool) -> Result<(), Error>,
{
    let mut stack = vec![root.to_string()];
    while let Some(path) = stack.pop() {
        let meta = fs::symlink_metadata(&path)?;
        let is_link = meta.file_type().is_symlink();
        let is_dir = meta.file_type().is_dir();
        visit(&path, is_dir, is_link)?;
        if is_dir && !is_link {
            let mut entries: Vec<String> = fs::read_dir(&path)?
                .filter_map(|entry| entry.ok())
                .map(|entry| entry.path().to_string_lossy().into_owned())
                .collect();
            entries.sort();
            entries.reverse();
            stack.extend(entries);
        }
    }
    Ok(())
}
