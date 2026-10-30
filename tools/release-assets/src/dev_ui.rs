//! Browser-only iteration against an already active development deployment.
//! The normal package and its backend stay installed; this explicit development
//! override replaces browser files only. A full deployment replaces the package.
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use sha2::{Digest, Sha256};

use super::{digest, files, remote_script, run, ssh, Args, Result};

pub fn deploy(source: &Path, args: &Args) -> Result<()> {
    let started = Instant::now();
    let id = format!(
        "ui-{}",
        SystemTime::now().duration_since(UNIX_EPOCH)?.as_millis()
    );
    let release = args.work.join("ui").join(&id);
    let output = release.join("browser");
    fs::create_dir_all(&release)?;
    println!("Building browser assets only");
    run(Command::new("bun")
        .current_dir(source)
        .args(["scripts/build-soda-extension.ts", "--package=soda", "--out"])
        .arg(&output)
        .arg("--terminal-assets")
        .arg(source.join(".artifacts/browser-terminal/vendor")))?;
    fs::copy(
        source.join("system/containers/extension/extension.json"),
        output.join("extension.json"),
    )?;
    let mut paths = Vec::new();
    files(&output, &output, &mut paths)?;
    let mut hashes = String::new();
    for path in &paths {
        hashes.push_str(&format!(
            "{}  {}\n",
            digest(&output.join(path))?,
            path.display()
        ));
    }
    fs::write(output.join("checksums.sha256"), hashes)?;
    let archive = release.join("upload.tar");
    run(Command::new("tar")
        .arg("-cf")
        .arg(&archive)
        .arg("-C")
        .arg(&output)
        .arg("."))?;
    let destination = format!("/var/home/soda-dev/ui/{id}");
    println!("Uploading browser assets to {}", args.host);
    run(ssh(args)
        .arg(format!(
            "umask 022\nmkdir -p '{destination}'\ntar -xf - -C '{destination}'"
        ))
        .stdin(Stdio::from(fs::File::open(archive)?)))?;
    remote_script(args, &activation(&destination))?;
    println!(
        "Browser update ready in {:.1}s. Refresh the app. No services restarted.",
        started.elapsed().as_secs_f64()
    );
    Ok(())
}

fn activation(directory: &str) -> String {
    // Directory is generated locally from a fixed root and a decimal timestamp.
    format!(
        r#"set -Eeuo pipefail
root=/var/home/soda-dev
release='{directory}'
package=/var/lib/soda/forgejo/gitea/extensions/soda
die() {{ printf '%s\n' "$*" >&2; exit 1; }}
[[ $(id -u) == 0 ]] || die 'Run the remote development action as root.'
[[ "$release" =~ ^/var/home/soda-dev/ui/ui-[0-9]+$ ]] || die 'Invalid browser release.'
[[ -L "$root/current" && -f /etc/containers/systemd/forgejo.container.d/zz-soda-dev.conf && -f /etc/containers/systemd/soda-dashboard.container.d/zz-soda-dev.conf ]] || die 'No active development deployment. Run soda-dev deploy once first.'
exec 9>"$root/lock"
flock -x 9
[[ -d "$package/assets" && ! -L "$package/assets" ]] || die 'The active Soda assets directory is unavailable.'
(cd "$release" && sha256sum --check --strict --quiet checksums.sha256)
diff -u <(jq --sort-keys 'del(.version)' "$package/extension.json") <(jq --sort-keys 'del(.version)' "$release/extension.json") || die 'The extension manifest changed. Use a full deployment for API, permission or route changes.'
before=$(sha256sum "$package/backend" "$package/extension.json" "$package/.fountain-package.json")
services=$(systemctl show forgejo.service soda-dashboard.service -p MainPID -p ActiveEnterTimestampMonotonic)
cp -a -- "$package/assets" "$release/previous-assets"
# Keep old content-addressed chunks available to pages already open. Publish
# dependency files before fixed-name entry points so new imports are available.
publish() {{
    local source=$1 target="$package/assets/$2" pending
    install -d -o 1000 -g 1000 -m 0755 "${{target%/*}}"
    chcon --reference="$package/assets" "${{target%/*}}"
    pending=$(mktemp "${{target%/*}}/.soda-ui.XXXXXX")
    install -o 1000 -g 1000 -m 0644 "$source" "$pending"
    chcon --reference="$package/assets" "$pending"
    mv -fT -- "$pending" "$target"
}}
restore() {{
    local result=$?
    trap - EXIT
    if (( result != 0 )); then
        printf '%s\n' 'Browser update failed; restoring previous browser files.' >&2
        cp -a -- "$release/previous-assets/." "$package/assets/"
    fi
    exit "$result"
}}
trap restore EXIT
while IFS= read -r -d '' file; do
    name=${{file#"$release/assets/"}}
    [[ "$name" == soda-*-entry.js ]] || publish "$file" "$name"
done < <(find "$release/assets" -type f -print0)
for file in "$release"/assets/soda-*-entry.js; do publish "$file" "${{file##*/}}"; done
# Opening as the running container user catches both DAC and SELinux failures;
# root-side checks and a healthy HTTP process alone cannot prove assets load.
podman exec --user 1000 soda-forgejo sh -ec 'for file in /data/gitea/extensions/soda/assets/soda-*-entry.js; do cat "$file" >/dev/null; done'
[[ "$before" == "$(sha256sum "$package/backend" "$package/extension.json" "$package/.fountain-package.json")" ]] || die 'Backend or package identity changed during the browser update.'
[[ "$services" == "$(systemctl show forgejo.service soda-dashboard.service -p MainPID -p ActiveEnterTimestampMonotonic)" ]] || die 'A service restarted during the browser update.'
curl --fail --silent --max-time 5 http://127.0.0.1:3000/api/healthz >/dev/null
curl --fail --silent --max-time 5 http://127.0.0.1:8080/healthz >/dev/null
jq --null-input --arg release "$release" --slurpfile base "$package/.fountain-package.json" '{{release:$release,base_archive:$base[0].archive_sha256}}' >"$root/browser-override.json"
trap - EXIT
printf '%s\n' 'Browser files updated; backend, package identity, database and running services preserved.'
"#
    )
}

fn fingerprint(source: &Path) -> Result<String> {
    let mut paths = Vec::new();
    for directory in ["frontend/spaces", "assets/branding"] {
        files(source, &source.join(directory), &mut paths)?;
    }
    paths.push(PathBuf::from("scripts/build-soda-extension.ts"));
    paths.push(PathBuf::from("system/containers/extension/extension.json"));
    let mut hash = Sha256::new();
    paths.sort();
    for path in paths {
        hash.update(path.as_os_str().as_encoded_bytes());
        let metadata = fs::metadata(source.join(path))?;
        hash.update(metadata.len().to_le_bytes());
        hash.update(
            metadata
                .modified()?
                .duration_since(UNIX_EPOCH)?
                .as_nanos()
                .to_le_bytes(),
        );
    }
    Ok(format!("{:x}", hash.finalize()))
}

pub fn watch(source: &Path, args: &Args) -> Result<()> {
    let mut previous = fingerprint(source)?;
    deploy(source, args)?;
    println!("Watching extension browser sources. Save an edit, then refresh after the update. Ctrl-C stops watching.");
    loop {
        std::thread::sleep(Duration::from_millis(500));
        let current = fingerprint(source)?;
        if current == previous {
            continue;
        }
        // Wait for an editor's sequence of writes to settle before building.
        std::thread::sleep(Duration::from_millis(200));
        previous = fingerprint(source)?;
        if let Err(error) = deploy(source, args) {
            eprintln!("Browser update failed: {error}. Watching for the next edit.");
        }
        std::io::stdout().flush()?;
    }
}
