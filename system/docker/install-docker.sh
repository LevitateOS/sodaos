#!/usr/bin/env bash
# Rendered from the exact release source by soda-release prepare. Keep the final
# compound command intact: a truncated stdin download must not start installation.
{
set -euo pipefail
set +x
umask 077

fail() { printf 'Soda Docker installer: %s\n' "$*" >&2; exit 1; }
say() { printf '%s\n' "$*"; }
usage() {
  cat <<'USAGE'
Usage: install-docker.sh [--bind-ip PRIVATE_IPV4] [--project-subnet CIDR]
                         [--install-dir /home/sodaos] [--yes] [--qualification]
Installs the script's exact release on a dedicated Linux x86-64 Docker host.
--yes accepts privileged host access; without a terminal, browser setup remains
an explicit operator step. --qualification admits this immutable prerelease for
release testing only. Neither option skips download or host validation.
USAGE
}
fetch() {
  curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    --connect-timeout 20 --max-time 180 --max-filesize "$3" "$1" -o "$2" ||
    fail "Download failed: $1. No deployment changes were made."
}
check_file() {
  [[ $(stat -c %s -- "$1") == "$3" ]] || fail "Incorrect byte size: ${1##*/}"
  [[ $(sha256sum -- "$1" | cut -d ' ' -f 1) == "$2" ]] || fail "Checksum mismatch: ${1##*/}"
}
ipv4() {
  local a b c d part
  [[ $1 =~ ^([0-9]+\.){3}[0-9]+$ ]] || return 1
  IFS=. read -r a b c d <<< "$1"
  for part in "$a" "$b" "$c" "$d"; do
    [[ $part == 0 || $part =~ ^[1-9][0-9]{0,2}$ ]] && (( 10#$part <= 255 )) || return 1
  done
  printf '%s\n' "$(( (10#$a << 24) + (10#$b << 16) + (10#$c << 8) + 10#$d ))"
}
private_ip() {
  local n
  n=$(ipv4 "$1") || return 1
  (( (n >> 24) == 10 || (n >> 20) == 2753 || (n >> 16) == 49320 || (n >> 22) == 401 ))
}
cidr_range() {
  local address bits n mask
  IFS=/ read -r address bits <<< "$1"
  [[ $bits =~ ^([0-9]|[12][0-9]|3[0-2])$ ]] || return 1
  n=$(ipv4 "$address") || return 1
  mask=$(( (0xffffffff << (32-bits)) & 0xffffffff ))
  printf '%s %s\n' "$((n & mask))" "$(((n & mask) | (0xffffffff ^ mask)))"
}
ask() {
  [[ -r /dev/tty && -w /dev/tty ]] || fail "A terminal is required; supply --bind-ip, --project-subnet and --yes for unattended runtime installation."
  printf '%s' "$1" > /dev/tty
  IFS= read -r reply < /dev/tty || fail 'Terminal input ended.'
}
admit_sudo() {
  command -v sudo >/dev/null || fail 'Administrative access requires sudo or root.'
  sudo -n -v 2>/dev/null && return
  if ! { : < /dev/tty; } 2>/dev/null; then
    fail 'Sudo authentication requires a terminal; authenticate first or run as root.'
  fi
  sudo -v < /dev/tty || fail 'Administrative access denied.'
}
docker_cmd() {
  local -a client=()
  [[ -z ${scratch:-} ]] || client=(--config "$scratch/docker-client")
  "${docker_prefix[@]}" docker "${client[@]}" --host unix:///var/run/docker.sock "$@"
}
compose() {
  SODA_IMAGE="$image" SODA_PROJECT_SUBNET="$subnet" docker_cmd compose \
    --project-name sodaos --project-directory "$install_dir" \
    --env-file "$install_dir/.env" -f "$install_dir/compose.yaml" "$@"
}
protected_file() {
  [[ -f $1 && ! -L $1 && $(stat -c %u -- "$1") == "$(id -u)" ]] || fail "Owned regular file required: $1"
  local mode
  mode=$(stat -c %a -- "$1")
  (( (8#$mode & 077) == 0 )) || fail "Private file permissions required: $1"
}
check_existing() {
  local file
  for file in identity.json docker.json compose.yaml .env; do protected_file "$install_dir/$file"; done
  jq -e --arg tag "$release_tag" --arg revision "$source_revision" --arg image "$image" \
    --arg meta "$metadata_sha" --arg compose "$compose_sha" '
      .format == 1 and .release == $tag and .source_revision == $revision and
      .image == $image and .metadata_sha256 == $meta and .compose_sha256 == $compose and
      (.bind_ip | type == "string") and (.project_subnet | type == "string")
    ' "$install_dir/identity.json" >/dev/null ||
    fail 'Existing installation differs. Cross-version upgrades and adoption are unsupported; existing state was preserved.'
  check_file "$install_dir/docker.json" "$metadata_sha" "$metadata_bytes"
  check_file "$install_dir/compose.yaml" "$compose_sha" "$compose_bytes"
  local saved_bind saved_subnet expected_env
  saved_bind=$(jq -r .bind_ip "$install_dir/identity.json")
  saved_subnet=$(jq -r .project_subnet "$install_dir/identity.json")
  [[ -z $bind_ip || $bind_ip == "$saved_bind" ]] || fail 'Changing the retained browser address is unsupported.'
  [[ -z $subnet || $subnet == "$saved_subnet" ]] || fail 'Changing the retained Project subnet is unsupported.'
  bind_ip=$saved_bind; subnet=$saved_subnet
  expected_env=$(printf 'SODA_IMAGE=%s\nSODA_PROJECT_SUBNET=%s\n' "$image" "$subnet")
  [[ $(cat "$install_dir/.env") == "$expected_env" ]] || fail 'Existing Compose configuration differs; inspect it before retrying.'
}
check_runtime_identity() {
  local id volume volume_count=0
  docker_cmd ps -aq --filter label=com.docker.compose.project=sodaos > "$scratch/containers"
  [[ $(wc -l < "$scratch/containers") -le 1 ]] || fail 'Unexpected containers in the retained Soda Compose project.'
  while IFS= read -r id; do
    [[ -n $id ]] || continue
    docker_cmd container inspect "$id" > "$scratch/container.json"
    jq -e --arg image "$image" --arg config "$config" --arg manifest "${image##*@}" '
      length == 1 and .[0].Config.Image == $image and
      (.[0].Image == $config or
        (.[0].Image == $manifest and .[0].ImageManifestDescriptor.digest == $manifest)) and
      .[0].Config.Labels["com.docker.compose.project"] == "sodaos" and
      .[0].Config.Labels["com.docker.compose.service"] == "soda" and
      .[0].HostConfig.Privileged == true and .[0].HostConfig.NetworkMode == "host" and
      ([.[0].Mounts[] | select(.Type == "bind")] | length) == 0 and
      ([.[0].Mounts[] | select(.Type == "volume") | {Name,Destination}] | sort_by(.Destination)) ==
        [{Name:"sodaos_config",Destination:"/etc"},{Name:"sodaos_home",Destination:"/home"},{Name:"sodaos_state",Destination:"/var"}]
    ' "$scratch/container.json" >/dev/null || fail 'Retained container identity or mounts differ; no replacement was attempted.'
  done < "$scratch/containers"
  for volume in config state home; do
    if docker_cmd volume inspect "sodaos_$volume" > "$scratch/volume.json" 2>/dev/null; then
      volume_count=$((volume_count + 1))
      jq -e --arg volume "$volume" 'length == 1 and
        .[0].Labels["com.docker.compose.project"] == "sodaos" and
        .[0].Labels["com.docker.compose.volume"] == $volume' "$scratch/volume.json" >/dev/null ||
        fail "Retained sodaos_$volume volume is not owned by this Compose project."
    fi
  done
  (( volume_count == 0 || volume_count == 3 )) || fail 'A persistent volume is missing. Restore the complete matching volume set before restarting.'
  if [[ -e $install_dir/runtime-initialized ]]; then
    protected_file "$install_dir/runtime-initialized"
    (( volume_count == 3 )) || fail 'Initialized runtime volumes are missing. Restore the complete matching volume set; initialization was not replayed.'
  fi
}
check_network() {
  private_ip "$bind_ip" || fail 'Choose a private IPv4 browser address assigned to this host.'
  ip -j -4 address show > "$scratch/addresses.json"
  jq -e --arg ip "$bind_ip" 'any(.[].addr_info[]; .local == $ip)' "$scratch/addresses.json" >/dev/null ||
    fail 'The browser address is not assigned to this Docker host.'
  local start end base bits route rstart rend addresses
  read -r start end < <(cidr_range "$subnet") || fail 'A canonical private IPv4 Project subnet is required.'
  IFS=/ read -r base bits <<< "$subnet"
  private_ip "$base" && [[ $(ipv4 "$base") == "$start" ]] && (( bits >= 8 && bits <= 30 )) ||
    fail 'Use a private IPv4 network address with prefix length 8 through 30.'
  # Both endpoints must remain in the same RFC1918 range; shared-address space
  # may be used for browser access but is not a default Project allocation.
  (( (start >> 24 == 10 && end >> 24 == 10) ||
     (start >> 20 == 2753 && end >> 20 == 2753) ||
     (start >> 16 == 49320 && end >> 16 == 49320) )) || fail 'Project subnet must fit within RFC1918 private space.'
  for route in 10.88.0.0/16 10.90.0.0/24; do
    read -r rstart rend < <(cidr_range "$route")
    (( start > rend || rstart > end )) || fail "Project subnet overlaps Soda's reserved network $route."
  done
  ip -j -4 route show table all > "$scratch/routes.json"
  jq -r '.[].dst // empty | select(. != "default")' "$scratch/routes.json" > "$scratch/routes"
  while IFS= read -r route; do
    [[ $route == */* ]] || route="$route/32"
    read -r rstart rend < <(cidr_range "$route") || fail 'Cannot validate a host IPv4 route.'
    # Retained Projects legitimately own this exact route on a rerun.
    if (( start <= rend && rstart <= end )); then
      [[ $existing == true ]] && (( rstart >= start && rend <= end )) || fail "Project subnet overlaps host route $route; select another subnet."
    fi
  done < "$scratch/routes"
  docker_cmd network ls --quiet > "$scratch/network-ids"
  while IFS= read -r network; do
    [[ -n $network ]] || continue
    docker_cmd network inspect "$network" > "$scratch/network.json"
    jq -r '.[].IPAM.Config[]?.Subnet // empty | select(contains(":") | not)' "$scratch/network.json" > "$scratch/network-subnets"
    while IFS= read -r route; do
      read -r rstart rend < <(cidr_range "$route") || fail 'Cannot validate a Docker subnet.'
      (( start > rend || rstart > end )) || fail "Project subnet overlaps Docker network $route."
    done < "$scratch/network-subnets"
  done < "$scratch/network-ids"
  [[ $(sysctl -n net.ipv4.ip_forward) == 1 ]] || fail 'Enable persistent IPv4 forwarding in the host network configuration, then rerun.'
  if [[ $existing == false ]]; then
    ss -H -ltn > "$scratch/listeners"
    if awk '{print $4}' "$scratch/listeners" | grep -Eq ':(443|2222|3000|9090)$'; then
      fail 'Required TCP port 443, 2222, 3000 or 9090 is already in use. Use a dedicated host.'
    fi
  fi
}
setup_browser() {
  local setup_state bootstrap_token
  setup_state=$(compose exec -T soda bash -c 'if test -f /etc/soda/activated; then echo activated; elif test -f /etc/soda/dashboard.json; then echo configured; else echo new; fi')
  case $setup_state in
    activated) say "Private browser configuration retained: https://$bind_ip" ;;
    configured) fail 'Setup configuration exists but activation is incomplete. Inspect the operator guide and service state; setup was not replayed.' ;;
    new)
      say 'Runtime ready. Complete the native Forgejo browser installation and create its site administrator.'
      say 'From your client, forward port 3000 over your existing trusted SSH connection to this host, then open http://localhost:33000.'
      say 'SSH forwarding syntax: ssh -N -L 33000:127.0.0.1:3000 YOUR_EXISTING_HOST_SSH_DESTINATION'
      say 'Create an administrator read:user bootstrap token. It will be read without echo and revoked by soda-setup.'
      if [[ $yes == true ]] || ! { : < /dev/tty; } 2>/dev/null; then
        say 'Browser setup is pending. Rerun this same installer interactively to enter the bootstrap token and activate private HTTPS.'
        return
      fi
      printf 'Bootstrap token (empty to leave setup pending): ' > /dev/tty
      IFS= read -r -s bootstrap_token < /dev/tty || fail 'Terminal input ended.'
      printf '\n' > /dev/tty
      if [[ -z $bootstrap_token ]]; then say 'Browser setup remains pending; rerun this installer to continue.'; return; fi
      printf '%s\n' "$bootstrap_token" | compose exec -T soda bash -c '
        set -euo pipefail; set +x; umask 077
        token=$(mktemp /run/soda-bootstrap.XXXXXX)
        trap '\''rm -f -- "$token"'\'' EXIT
        cat > "$token"
        soda-setup --forgejo-url "$1" --token-file "$token"
      ' bash "https://$bind_ip" || fail 'Native setup did not confirm completion. Inspect private runtime state before retrying.'
      unset bootstrap_token
      compose exec -T soda soda-activate --bind-ip "$bind_ip" --local-tls
      ;;
    *) fail 'Cannot determine native browser setup state.' ;;
  esac
  # Only the public certificate is exported; keys and credentials remain inside.
  compose cp soda:/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt "$scratch/soda-root.crt"
  openssl x509 -in "$scratch/soda-root.crt" -noout -fingerprint -sha256
  compose exec -T soda systemctl is-active --quiet soda-dashboard.service
  compose exec -T soda systemctl is-active --quiet soda-proxy.service
  curl --fail --silent --show-error --proto '=https' --connect-timeout 10 --max-time 30 \
    --cacert "$scratch/soda-root.crt" "https://$bind_ip/api/healthz" > /dev/null ||
    fail 'Private HTTPS is not ready with the exported CA. Inspect activation and services; setup was not replayed.'
  install -m 0644 "$scratch/soda-root.crt" "$install_dir/soda-root.crt"
  say "Transfer $install_dir/soda-root.crt through trusted host SSH, compare its SHA-256 fingerprint, and add it to each client's trust store."
  say "Then open https://$bind_ip. Configure client routes to $subnet through this host for direct Project access."
  say 'Factory/provider enrollment and any Cockpit password remain separate operator actions. No client trust or firewall rules were changed.'
}
main() {
  local release_tag='@RELEASE_TAG@' source_revision='@SOURCE_REVISION@'
  local metadata_sha='@METADATA_SHA256@' metadata_bytes='@METADATA_BYTES@'
  local install_dir=/home/sodaos bind_ip= subnet= yes=false qualification=false existing=false reply=
  local scratch= staging= temp_parent command info compose_version release_url image manifest config compose_sha compose_bytes marker
  local -a docker_prefix=() admin_prefix=()
  while (( $# )); do
    case $1 in
      --help|-h) usage; return ;;
      --yes) yes=true; shift ;;
      --qualification) qualification=true; shift ;;
      --bind-ip|--project-subnet|--install-dir)
        (( $# >= 2 )) || fail "Missing value for $1"
        case $1 in --bind-ip) bind_ip=$2;; --project-subnet) subnet=$2;; --install-dir) install_dir=$2;; esac
        shift 2 ;;
      *) fail "Unknown option: $1" ;;
    esac
  done
  [[ $release_tag =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ && $source_revision =~ ^[0-9a-f]{40}$ &&
     $metadata_sha =~ ^[0-9a-f]{64}$ && $metadata_bytes =~ ^[1-9][0-9]*$ ]] || fail 'This is an unrendered installer template; use a published release asset.'
  [[ $(uname -s) == Linux && $(uname -m) == x86_64 ]] || fail 'Only native x86-64 Linux Docker Engine is supported.'
  [[ $(uname -r) != *[Mm]icrosoft* && $(uname -r) != *WSL* ]] || fail 'WSL2 is unsupported; use a dedicated Linux host or VM.'
  local missing=()
  for command in curl jq docker ip ss sysctl sha256sum stat mktemp cut awk grep install realpath id flock openssl dirname mkdir chown wc mv cat rm; do
    command -v "$command" >/dev/null || missing+=("$command")
  done
  (( ${#missing[@]} == 0 )) || fail "Install required host tools first: ${missing[*]}. Docker Engine and its Compose plugin must already be installed."
  [[ $install_dir == /home/* && $(realpath -m -- "$install_dir") == "$install_dir" && ! -L $install_dir ]] ||
    fail 'Use an absolute installation directory below /home without symlinks or traversal.'
  if ! docker_cmd info --format '{{json .}}' >/dev/null 2>&1; then
    say 'Administrative access is needed to use the local rootful Docker daemon.'
    admit_sudo
    docker_prefix=(sudo --preserve-env=SODA_IMAGE,SODA_PROJECT_SUBNET)
  fi
  info=$(docker_cmd info --format '{{json .}}') || fail 'The local Docker Engine is unavailable.'
  jq -e '.OSType == "linux" and (.Architecture == "x86_64" or .Architecture == "amd64") and
    .CgroupVersion == "2" and (.OperatingSystem | test("Docker Desktop"; "i") | not) and
    ([.SecurityOptions[]? | contains("rootless")] | any | not)' <<< "$info" >/dev/null ||
    fail 'A local rootful Linux x86-64 Docker Engine with cgroup v2 is required; Desktop and rootless are unsupported.'
  compose_version=$(docker_cmd compose version --short)
  compose_version=${compose_version#v}
  [[ $compose_version =~ ^([0-9]+)\.([0-9]+)\. ]] &&
    (( BASH_REMATCH[1] > 2 || (BASH_REMATCH[1] == 2 && BASH_REMATCH[2] >= 17) )) ||
    fail 'Docker Compose 2.17 or newer is required.'
  temp_parent=${TMPDIR:-${HOME:-/home}}
  [[ $EUID != 0 || -n ${TMPDIR:-} ]] || temp_parent=/home
  [[ $temp_parent == /home || $temp_parent == /home/* ]] || fail 'Set TMPDIR to an existing writable directory on /home.'
  scratch=$(mktemp -d "$temp_parent/soda-install.XXXXXXXX")
  # The exact fresh private scratch is the only automatic cleanup target.
  local cleanup
  printf -v cleanup 'rm -rf -- %q' "$scratch"
  trap "$cleanup" EXIT
  mkdir -m 0700 "$scratch/docker-client"
  release_url="https://github.com/LevitateOS/sodaos/releases/download/$release_tag"
  fetch "https://api.github.com/repos/LevitateOS/sodaos/releases/tags/$release_tag" "$scratch/release.json" 1048576
  jq -e --arg tag "$release_tag" '.tag_name == $tag and .draft == false and .immutable == true and
    ([.assets[] | select(.name == "docker.json" or .name == "sodaos-docker-compose.yaml" or .name == "install-docker.sh") | .name] | sort) ==
    ["docker.json", "install-docker.sh", "sodaos-docker-compose.yaml"]' "$scratch/release.json" >/dev/null || fail 'The selected immutable release lacks a complete Docker installation asset set.'
  fetch "$release_url/docker.json" "$scratch/docker.json" "$metadata_bytes"
  check_file "$scratch/docker.json" "$metadata_sha" "$metadata_bytes"
  jq -e --arg tag "$release_tag" --arg revision "$source_revision" '
    .Format == 1 and .Purpose == "production" and (.Distribution == "docker" or .Distribution == "all") and
    .ReleaseTag == $tag and .SodaRevision == $revision and .Architecture == "x86_64" and .Platform == "linux/amd64" and
    .Image.Architecture == "amd64" and .Image.Revision == $revision and
    .Image.Source == "https://github.com/LevitateOS/sodaos" and
    (.Image.Manifest | test("^sha256:[0-9a-f]{64}$")) and (.Image.Config | test("^sha256:[0-9a-f]{64}$")) and
    (.Compose.SHA256 | test("^[0-9a-f]{64}$")) and
    (.Compose.Bytes | type == "number" and . > 0 and . <= 1048576 and . == floor)
  ' "$scratch/docker.json" >/dev/null || fail 'Docker metadata does not identify this release and native platform.'
  manifest=$(jq -r .Image.Manifest "$scratch/docker.json")
  config=$(jq -r .Image.Config "$scratch/docker.json")
  image="ghcr.io/levitateos/sodaos@$manifest"
  marker="<!-- soda-docker-qualified: $release_tag $source_revision $manifest -->"
  if [[ $qualification == true ]]; then
    jq -e '.prerelease == true' "$scratch/release.json" >/dev/null || fail '--qualification requires an immutable prerelease.'
    say 'Release qualification mode: these bytes have not yet been recommended for production.'
  else
    jq -e --arg marker "$marker" '.prerelease == false and (.body | split("\n") | index($marker) != null)' "$scratch/release.json" >/dev/null ||
      fail 'This exact Docker release is not qualified. No installation changes were made.'
  fi
  compose_sha=$(jq -r .Compose.SHA256 "$scratch/docker.json")
  compose_bytes=$(jq -r .Compose.Bytes "$scratch/docker.json")
  fetch "$release_url/sodaos-docker-compose.yaml" "$scratch/compose.yaml" "$compose_bytes"
  check_file "$scratch/compose.yaml" "$compose_sha" "$compose_bytes"
  [[ ! -e $install_dir ]] || existing=true
  if [[ $existing == true ]]; then
    [[ -d $install_dir && $(stat -c %u -- "$install_dir") == "$(id -u)" && $(stat -c %a -- "$install_dir") == 700 ]] || fail 'Existing installation directory must be private and owned by this operator.'
    check_existing
    check_runtime_identity
  else
    docker_cmd ps -aq --filter label=com.docker.compose.project=sodaos > "$scratch/containers"
    [[ ! -s $scratch/containers ]] || fail 'An existing Soda Compose project requires its original installation directory.'
    local volume
    for volume in sodaos_config sodaos_state sodaos_home; do
      if docker_cmd volume inspect "$volume" >/dev/null 2>&1; then fail "Retained volume $volume already exists; use its original installation directory."; fi
    done
  fi
  if [[ -z $bind_ip ]]; then
    ip -brief -4 address show
    ask 'Private browser IPv4 address assigned to this host: '; bind_ip=$reply
  fi
  if [[ -z $subnet ]]; then
    if [[ $yes == true ]]; then subnet=10.89.0.0/24
    else ask 'Project subnet [10.89.0.0/24]: '; subnet=${reply:-10.89.0.0/24}; fi
  fi
  check_network
  # Parse the verified Compose before any installation directory/image/volume mutation.
  SODA_IMAGE="$image" SODA_PROJECT_SUBNET="$subnet" docker_cmd compose --env-file /dev/null \
    --project-name sodaos -f "$scratch/compose.yaml" config --quiet
  say "Release: $release_tag ($source_revision)"
  say "Image: $image"
  say 'Soda runs privileged with host networking and administrative authority over this host. Use a dedicated machine.'
  say "Persistent volumes: sodaos_config (/etc), sodaos_state (/var), sodaos_home (/home). Installation: $install_dir"
  if [[ $yes == false ]]; then
    ask 'Install/start this exact release with privileged host access? [yes/no]: '
    [[ $reply == yes ]] || fail 'Installation cancelled without deployment changes.'
  fi
  # Anonymous digest pull validates the image before committing deployment files
  # or creating any runtime volumes. It may populate Docker's image cache.
  docker_cmd pull --platform linux/amd64 "$image"
  docker_cmd image inspect "$image" > "$scratch/image.json"
  # Docker storage backends expose either the configuration or OCI manifest as
  # the image ID. A manifest ID must carry its matching content descriptor;
  # both forms still require the exact anonymously pulled digest reference.
  jq -e --arg config "$config" --arg image "$image" --arg manifest "${image##*@}" --arg revision "$source_revision" '
    length == 1 and
    (.[0].Id == $config or (.[0].Id == $manifest and .[0].Descriptor.digest == $manifest)) and
    .[0].Os == "linux" and .[0].Architecture == "amd64" and
    (.[0].RepoDigests | index($image) != null) and .[0].Config.Labels["org.opencontainers.image.revision"] == $revision
  ' "$scratch/image.json" >/dev/null || fail 'Pulled image identity differs from release metadata; runtime was not started.'
  if [[ $existing == false ]]; then
    if [[ ! -w $(dirname "$install_dir") ]]; then
      admit_sudo
      admin_prefix=(sudo)
    fi
    staging=$("${admin_prefix[@]}" mktemp -d "${install_dir}.install.XXXXXXXX")
    if (( ${#admin_prefix[@]} )); then
      printf -v cleanup 'rm -rf -- %q; sudo -n rm -rf -- %q' "$scratch" "$staging"
      trap "$cleanup" EXIT
      sudo chown "$(id -u):$(id -g)" "$staging"
    else
      printf -v cleanup 'rm -rf -- %q %q' "$scratch" "$staging"
      trap "$cleanup" EXIT
    fi
    install -m 0600 "$scratch/docker.json" "$staging/docker.json"
    install -m 0600 "$scratch/compose.yaml" "$staging/compose.yaml"
    printf 'SODA_IMAGE=%s\nSODA_PROJECT_SUBNET=%s\n' "$image" "$subnet" > "$staging/.env"
    jq -n --arg release "$release_tag" --arg revision "$source_revision" --arg image "$image" \
      --arg meta "$metadata_sha" --arg compose "$compose_sha" --arg bind "$bind_ip" --arg subnet "$subnet" \
      '{format:1, release:$release, source_revision:$revision, image:$image, metadata_sha256:$meta,
        compose_sha256:$compose, bind_ip:$bind, project_subnet:$subnet}' > "$staging/identity.json"
    # Rename a complete private configuration once; never replace another run.
    "${admin_prefix[@]}" mv -T -n -- "$staging" "$install_dir"
    [[ ! -e $staging ]] || fail 'Installation directory appeared concurrently; no existing state was overwritten.'
    printf -v cleanup 'rm -rf -- %q' "$scratch"
    trap "$cleanup" EXIT
  fi
  local lock_fd
  exec {lock_fd}< "$install_dir"
  flock -n "$lock_fd" || fail 'Another installer is using this installation directory.'
  check_existing
  check_runtime_identity
  compose up -d --no-recreate --wait --wait-timeout 1800
  if [[ ! -e $install_dir/runtime-initialized ]]; then
    printf '%s\n' "$image" > "$install_dir/runtime-initialized"
  fi
  compose ps
  say 'Runtime services are ready; persistent configuration and Project volumes are retained.'
  setup_browser
  say "Lifecycle commands: cd $install_dir && docker compose --project-name sodaos stop"
  say 'Never use down --volumes or prune retained Project storage. Rerunning this release preserves state; cross-version replacement is refused.'
}
main "$@"
}
