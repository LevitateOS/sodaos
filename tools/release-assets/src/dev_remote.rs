//! The remote half of the SSH development overlay. Scripts run as root and only
//! own their two Quadlet drop-ins, releases, development database, and cache entry.

fn quote(value: &str) -> String {
    format!("'{}'", value.replace('\'', "'\\''"))
}

const COMMON: &str = r#"set -Eeuo pipefail
root=/var/home/soda-dev
forgejo_dropin=/etc/containers/systemd/forgejo.container.d/zz-soda-dev.conf
dashboard_dropin=/etc/containers/systemd/soda-dashboard.container.d/zz-soda-dev.conf
die() { printf '%s\n' "$*" >&2; exit 1; }
[[ $(id -u) == 0 ]] || die 'Run the remote development action as root.'
for command in flock systemctl curl install cp mv mktemp readlink; do
    command -v "$command" >/dev/null || die "Missing remote command: $command"
done
install -d -m 0755 "$root" "$root/releases"
exec 9>"$root/lock"
flock -x 9

wait_ready() {
    local attempt
    printf '%s\n' 'Waiting for Forgejo and dashboard readiness...'
    for attempt in {1..90}; do
        if systemctl --quiet is-failed forgejo.service soda-dashboard.service; then
            printf '%s\n' 'Forgejo or the dashboard entered a failed state.' >&2
            return 1
        fi
        if curl --fail --silent --max-time 2 http://127.0.0.1:3000/api/healthz >/dev/null &&
           curl --fail --silent --max-time 2 http://127.0.0.1:8080/healthz >/dev/null; then
            return 0
        fi
        sleep 2
    done
    printf '%s\n' 'Forgejo or the dashboard did not become ready.' >&2
    return 1
}

restart_services() {
    systemctl daemon-reload || return $?
    systemctl reset-failed forgejo.service soda-dashboard.service || return $?
    systemctl restart forgejo.service soda-dashboard.service || return $?
    wait_ready
}

snapshot() {
    transaction=$(mktemp -d "$root/transaction.XXXXXX")
    [[ ! -e "$root/current" || -L "$root/current" ]] || die "$root/current must be a symlink."
    [[ ! -e "$forgejo_dropin" && ! -L "$forgejo_dropin" ]] || cp -a -- "$forgejo_dropin" "$transaction/forgejo.conf"
    [[ ! -e "$dashboard_dropin" && ! -L "$dashboard_dropin" ]] || cp -a -- "$dashboard_dropin" "$transaction/dashboard.conf"
    [[ ! -L "$root/current" ]] || cp -a -- "$root/current" "$transaction/current"
    changed=0
    trap finish EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
}

finish() {
    local result=$?
    trap - EXIT INT TERM
    set +e
    if (( result != 0 && changed )); then
        printf '%s\n' 'Development activation failed; restoring the previous service overlays.' >&2
        rm -f -- "$forgejo_dropin" "$dashboard_dropin" "$root/current"
        [[ ! -e "$transaction/forgejo.conf" && ! -L "$transaction/forgejo.conf" ]] || cp -a -- "$transaction/forgejo.conf" "$forgejo_dropin"
        [[ ! -e "$transaction/dashboard.conf" && ! -L "$transaction/dashboard.conf" ]] || cp -a -- "$transaction/dashboard.conf" "$dashboard_dropin"
        [[ ! -L "$transaction/current" ]] || cp -a -- "$transaction/current" "$root/current"
        if ! restart_services; then
            printf '%s\n' 'Previous overlays were restored, but service readiness still failed. Inspect the service journals.' >&2
        fi
    fi
    rm -rf -- "$transaction"
    exit "$result"
}
"#;

/// Activate an already uploaded, uniquely named release under the fixed dev root.
pub fn activate_script(remote_dir: &str) -> String {
    let mut script = COMMON.to_owned();
    script.push_str("release=");
    script.push_str(&quote(remote_dir));
    script.push('\n');
    script.push_str(
        r#"[[ "$release" =~ ^/var/home/soda-dev/releases/[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || die 'Invalid remote development release path.'
[[ -d "$release" && ! -L "$release" ]] || die 'The uploaded development release directory is missing.'
for command in jq sha256sum stat find chown chmod restorecon podman sed cmp; do
    command -v "$command" >/dev/null || die "Missing remote command: $command"
done
[[ -f "$release/checksums.sha256" ]] || die 'The uploaded checksum inventory is missing.'
printf '%s\n' 'Verifying uploaded development files...'
(cd "$release" && sha256sum --check --strict --quiet checksums.sha256)
[[ -d "$release/forgejo" && -f "$release/bin/soda-dashboard" ]] || die 'The uploaded Forgejo presentation or dashboard is missing.'
[[ -f "$release/dev-schema.sql" ]] || die 'The development database schema additions are missing.'
[[ -L "$release/forgejo/conf" && $(readlink "$release/forgejo/conf") == /data/gitea/conf ]] || die 'Forgejo conf must link to /data/gitea/conf.'
[[ ! -e "$release/dev-catalog.json" ]] || die 'This release has already been prepared; upload a new release directory.'
read -r package_sha <"$release/package.sha256"
[[ "$package_sha" =~ ^[0-9a-f]{64}$ ]] || die 'Invalid development package SHA-256.'
actual_sha=$(sha256sum "$release/soda.tar.gz")
[[ ${actual_sha%% *} == "$package_sha" ]] || die 'The development package hash does not match.'
jq --exit-status --arg hash "$package_sha" --argjson size "$(stat -c %s "$release/soda.tar.gz")" '
    .version == 1 and (.packages | type == "array") and (.packages | length == 1) and
    .packages[0].id == "soda" and .packages[0].sha256 == $hash and .packages[0].size == $size
' "$release/catalog.json" >/dev/null || die 'The development catalog must contain exactly the uploaded Soda package.'

snapshot
# Keep the native Soda database untouched. The development copy is retained
# between deploys, including any changes made while exercising the application.
database_exists=$(podman exec --user postgres soda-postgres psql -X -U postgres -d postgres -Atc "SELECT 1 FROM pg_database WHERE datname = 'soda_dev'")
database_state=
if [[ -f "$root/dev-db-created" ]]; then
    read -r database_state <"$root/dev-db-created"
fi
if [[ "$database_exists" == 1 ]]; then
    [[ "$database_state" == initialized ]] || die 'soda_dev already exists without a completed development clone; inspect it and recover initialization explicitly.'
else
    [[ "$database_state" != initialized ]] || die 'The retained development database is missing; refusing to replace it silently.'
    printf '%s\n' 'Cloning the native Soda database for development...'
    if [[ ! -f "$root/soda-native.dump" ]]; then
        install -m 0600 /dev/null "$root/soda-native.dump.pending"
        podman exec --user postgres soda-postgres pg_dump -U postgres --format=custom -d soda >"$root/soda-native.dump.pending"
        mv -f -- "$root/soda-native.dump.pending" "$root/soda-native.dump"
    fi
    install -m 0600 /dev/null "$root/dev-db-created"
    printf '%s\n' initializing >"$root/dev-db-created"
    podman exec --user postgres soda-postgres createdb -U postgres --owner=soda soda_dev
    podman exec -i --user postgres soda-postgres pg_restore -U postgres --role=soda --no-owner --exit-on-error --single-transaction -d soda_dev <"$root/soda-native.dump"
    printf '%s\n' initialized >"$root/dev-db-created"
fi
printf '%s\n' 'Applying development schema additions to soda_dev...'
podman exec -i --user postgres soda-postgres psql -X -U postgres -v ON_ERROR_STOP=1 --quiet -d soda_dev -c 'SET ROLE soda' -f - <"$release/dev-schema.sql"

# Generate private per-release connection/config files without printing secrets.
sed -E 's@/soda(\?|$)@/soda_dev\1@' /etc/soda/postgres/soda.dsn >"$transaction/dev.dsn"
if cmp --silent /etc/soda/postgres/soda.dsn "$transaction/dev.dsn"; then
    die 'The native Soda DSN does not name the expected soda database.'
fi
jq '.database_dsn_file = "/etc/soda/postgres/dev.dsn"' /etc/soda/dashboard.json >"$transaction/dev-dashboard.json"
install -o 2000 -g 2000 -m 0400 "$transaction/dev.dsn" "$release/soda-dev.dsn"
install -o 2000 -g 2000 -m 0400 "$transaction/dev-dashboard.json" "$release/dev-dashboard.json"

# Read the active catalog inside Forgejo so enabled optional packages survive.
if ! podman exec soda-forgejo sh -c 'cat "${FORGEJO__extensions__CATALOG_PATH:-/etc/soda/extensions/catalog.json}"' >"$transaction/live-catalog.json"; then
    if [[ -f "$forgejo_dropin" && -f "$root/current/dev-catalog.json" ]]; then
        cp -- "$root/current/dev-catalog.json" "$transaction/live-catalog.json"
    else
        cp -- /usr/share/soda/extensions/catalog.json "$transaction/live-catalog.json"
    fi
fi
jq --exit-status '.version == 1 and (.packages | type == "array")' "$transaction/live-catalog.json" >/dev/null || die 'The installed extension catalog is invalid.'
jq --slurpfile dev "$release/catalog.json" '
    .packages = ($dev[0].packages + [.packages[] | select(.id != "soda")])
' "$transaction/live-catalog.json" >"$release/dev-catalog.json"

# Fountain checks this cache by digest before downloading the catalog URL.
cache=/var/lib/soda/forgejo/gitea/extensions/.cache/packages
install -d -o 1000 -g 1000 -m 0755 "$cache"
install -o 1000 -g 1000 -m 0644 "$release/soda.tar.gz" "$cache/$package_sha.tar.gz"
restorecon -F "$cache/$package_sha.tar.gz"

# Seal the uploaded public files; Podman applies the labels for each bind mount.
chown -hR root:root "$release"
find "$release" -type d -exec chmod 0755 {} +
find "$release" -type f ! -path "$release/soda-dev.dsn" ! -path "$release/dev-dashboard.json" -exec chmod 0444 {} +
chown 2000:2000 "$release/soda-dev.dsn" "$release/dev-dashboard.json"
chmod 0755 "$release/bin/soda-dashboard"
[[ ! -f "$release/bin/fountain" ]] || chmod 0755 "$release/bin/fountain"
{
    printf '%s\n' '[Container]'
    printf 'Volume=%s/forgejo:/usr/share/soda/forgejo:ro,z\n' "$release"
    printf 'Volume=%s/dev-catalog.json:/etc/soda/dev-catalog.json:ro,z\n' "$release"
    printf '%s\n' 'Environment=FORGEJO__extensions__CATALOG_PATH=/etc/soda/dev-catalog.json'
    if [[ -f "$release/presentation.json" ]]; then
        printf 'Volume=%s/presentation.json:/usr/share/soda/presentation.json:ro,z\n' "$release"
    fi
    if [[ -f "$release/bin/fountain" ]]; then
        printf 'Volume=%s/bin/fountain:/app/gitea/gitea:ro,z\n' "$release"
    fi
} >"$transaction/forgejo.new"
{
    printf '%s\n' '[Container]'
    printf 'Volume=%s/bin/soda-dashboard:/usr/local/bin/soda-dashboard:ro,Z\n' "$release"
    printf 'Volume=%s/soda-dev.dsn:/etc/soda/postgres/dev.dsn:ro,z\n' "$release"
    printf 'Volume=%s/dev-dashboard.json:/etc/soda/dev-dashboard.json:ro,z\n' "$release"
    printf '%s\n' 'Exec=--extension-socket=/ipc/soda.sock --operator-socket=/run/soda/operator/operator.sock --operator-uid=0 --config=/etc/soda/dev-dashboard.json'
} >"$transaction/dashboard.new"
install -d -m 0755 "${forgejo_dropin%/*}" "${dashboard_dropin%/*}"
changed=1
install -m 0644 "$transaction/forgejo.new" "$forgejo_dropin"
install -m 0644 "$transaction/dashboard.new" "$dashboard_dropin"
restart_services
jq --exit-status --arg hash "$package_sha" '.package_id == "soda" and .archive_sha256 == $hash' \
    /var/lib/soda/forgejo/gitea/extensions/soda/.fountain-package.json >/dev/null || die 'Forgejo did not install the uploaded Soda package.'
ln -s -- "$release" "$transaction/current.new"
mv -Tf -- "$transaction/current.new" "$root/current"
changed=0
printf 'Development overlay active: %s\n' "$release"
"#,
    );
    script
}

/// Remove only this tool's overlays and restart on the installed native assets.
pub fn disable_script() -> String {
    let mut script = COMMON.to_owned();
    script.push_str(
        r#"snapshot
changed=1
rm -f -- "$forgejo_dropin" "$dashboard_dropin"
restart_services
rm -f -- "$root/current"
changed=0
printf '%s\n' 'Development overlay disabled; installed SodaOS assets are active.'
"#,
    );
    script
}

/// Report the overlay and both services without changing the installed machine.
pub fn status_script() -> String {
    r#"set -uo pipefail
root=/var/home/soda-dev
forgejo_dropin=/etc/containers/systemd/forgejo.container.d/zz-soda-dev.conf
dashboard_dropin=/etc/containers/systemd/soda-dashboard.container.d/zz-soda-dev.conf
if [[ -f "$forgejo_dropin" && -f "$dashboard_dropin" ]]; then
    printf '%s\n' 'mode=development'
    printf '%s\n' 'database=soda_dev'
elif [[ -e "$forgejo_dropin" || -e "$dashboard_dropin" ]]; then
    printf '%s\n' 'mode=incomplete'
else
    printf '%s\n' 'mode=native'
    printf '%s\n' 'database=soda'
fi
if [[ -L "$root/current" ]]; then
    printf 'release=%s\n' "$(readlink "$root/current")"
fi
for service in forgejo soda-dashboard; do
    printf '%s=' "$service"
    systemctl is-active "$service.service" || true
done
result=0
for endpoint in 'forgejo http://127.0.0.1:3000/api/healthz' 'dashboard http://127.0.0.1:8080/healthz'; do
    name=${endpoint%% *}
    url=${endpoint#* }
    if curl --fail --silent --max-time 5 "$url" >/dev/null; then
        printf '%s_health=ready\n' "$name"
    else
        printf '%s_health=unavailable\n' "$name"
        result=1
    fi
done
exit "$result"
"#
    .to_owned()
}
