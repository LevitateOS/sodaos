#!/bin/bash
# Refresh appliance/locks/host-packages-x86_64.json when Fedora supersedes
# pinned builds. Fails closed: a pin whose best replacement would upgrade a
# pinned base package is refused (container rpm-ostree install cannot replace
# @System), as is any requested package that no longer resolves at all.
#
# Usage: scripts/relock-host-packages.sh [--check] [--lock FILE]
#   --check  report drift and the planned new pins without writing anything
#   --lock   lock file (default appliance/locks/host-packages-x86_64.json)
#
# Needs network plus dnf and python3-rpm; no privilege required. Repos are
# Fedora 44 release plus updates (direct master mirror, no metalink roulette
# for the lock computation) plus Tailscale stable. After a rewrite, the
# candidate build's packages.expected diff is the final arbiter: if the
# build reports missing pins right after a relock, wait for mirror sync and
# retry, since installer mirrors lag the master by hours.
set -euo pipefail
cd "$(dirname "$0")/.."

LOCK=appliance/locks/host-packages-x86_64.json
CHECK=0
for arg in "$@"; do
  case "$arg" in
    --check) CHECK=1 ;;
    --lock=*) LOCK="${arg#--lock=}" ;;
    *) printf 'usage: relock-host-packages.sh [--check] [--lock=FILE]\n' >&2; exit 2 ;;
  esac
done
[ -f "$LOCK" ] || { printf 'lock not found: %s\n' "$LOCK" >&2; exit 1; }
command -v dnf >/dev/null || { printf 'dnf required for lock resolution\n' >&2; exit 1; }

TMPD="$(mktemp -d)"
trap 'rm -rf "$TMPD"' EXIT
cat >"$TMPD/fedora.repo" <<'EOF'
[f44]
name=f44
baseurl=https://dl.fedoraproject.org/pub/fedora/linux/releases/44/Everything/x86_64/os/
enabled=1
gpgcheck=0
[f44-updates]
name=f44-updates
baseurl=https://dl.fedoraproject.org/pub/fedora/linux/updates/44/Everything/x86_64/
enabled=1
gpgcheck=0
EOF
curl -s --max-time 60 https://pkgs.tailscale.com/stable/fedora/tailscale.repo -o "$TMPD/tailscale.repo" || {
  printf 'cannot fetch tailscale repo definition; network required\n' >&2; exit 1; }
sed -i 's|$basearch|x86_64|; s|^gpgcheck=1|gpgcheck=0|; s|^repo_gpgcheck=1|repo_gpgcheck=0|' "$TMPD/tailscale.repo"

export LOCK_FILE="$LOCK" REPOS_TMPD="$TMPD" CHECK
export DNF_BASE="dnf --setopt=reposdir=$TMPD --disablerepo=* --enablerepo=f44,f44-updates,tailscale-stable"

python3 - <<'PY'
import functools, json, os, re, subprocess, sys

try:
    import rpm
except ImportError:
    sys.exit('python3-rpm required for version comparison')

lock_path = os.environ['LOCK_FILE']
check_only = os.environ['CHECK'] == '1'
dnf_base = os.environ['DNF_BASE'].split()
default_lock = lock_path == 'appliance/locks/host-packages-x86_64.json'
ARCHES = ('x86_64', 'noarch')

lock = json.load(open(lock_path))
requested, install, inventory = lock['Requested'], lock['Install'], lock['Inventory']

def split_nevra(n):
    m = re.match(r'^(.*?)-(\d+):(.*)\.([^.]+)$', n)
    if not m:
        sys.exit(f'unparseable NEVRA in lock: {n}')
    return m.group(1), m.group(2), m.group(3), m.group(4)

def nevra_name(n):
    return split_nevra(n)[0]

def evr_key(evr):
    e, vr = evr[0], evr[1]
    v, r = vr.rsplit('-', 1)
    return (e or '0', v, r)

def newest(nevras):
    return max(nevras, key=functools.cmp_to_key(
        lambda a, b: rpm.labelCompare(evr_key(split_nevra(a)[1:3]), evr_key(split_nevra(b)[1:3]))))

query_cache = {}

def dnf(*args):
    key = tuple(args)
    if key not in query_cache:
        query_cache[key] = subprocess.run([*dnf_base, *args], capture_output=True, text=True)
    return query_cache[key]

def nevra_lines(proc):
    return [l.strip() for l in proc.stdout.splitlines()
            if l.strip() and not l.startswith('Last metadata expiration')]

# Phase A: every pinned install NEVRA must still exist upstream.
avail = dnf('repoquery', '--available',
            '--queryformat=%{name}-%{epoch}:%{version}-%{release}.%{arch}\n')
if avail.returncode != 0:
    sys.exit(f'repo query failed; network required:\n{avail.stderr.strip()[:500]}')
have = set(nevra_lines(avail))
missing = [n for n in install if n not in have]
if not missing:
    print(f'lock is current: {len(install)} install pins, {len(inventory)} inventory lines')
    sys.exit(0)
print(f'drifted pins ({len(missing)}):')
for m in missing:
    print(f'  - {m}')

# Base map and world names. The base is what rpm-ostree layers onto: a bump
# whose dependency closure would replace a base package is uninstallable.
# Inventory EVRA carries the arch; base versions compare without it.
def base_evr(evr_arch):
    evr = evr_arch.rsplit('.', 1)[0]
    e, vr = evr.split(':', 1) if ':' in evr else ('0', evr)
    return (e, vr)

base = {}
for line in inventory:
    name, evr = line.split(' ', 1)
    base.setdefault(name, (base_evr(evr), evr.rsplit('.', 1)[1]))
for n in install:
    base.pop(nevra_name(n), None)

by_name_arch = {}
for n in have:
    name, _, _, arch = split_nevra(n)
    if arch in ARCHES:
        by_name_arch.setdefault((name, arch), []).append(n)

def ok_lines(proc, what):
    # A failed repo query must abort, never read as empty: empty requirements
    # would silently accept every bump, and empty providers would silently
    # skip every conditional or drop every requirer.
    if proc.returncode != 0:
        sys.exit(f'repo query failed ({what}); network required:\n{proc.stderr.strip()[:500]}')
    return nevra_lines(proc)

def providers(req):
    return [n for n in ok_lines(dnf('repoquery', '--whatprovides', req), f'whatprovides {req}')
            if split_nevra(n)[3] in ARCHES]

def concretize(req, view, names):
    # (inner if condition): the condition holds when the installed world
    # provides it. Exact-NEVRA matching alone is unsound: pruned base packages
    # vanish from repo queries while remaining installed (selinux-policy-44.6
    # provides selinux-policy-base, but only -43.3/-44.9 are downloadable).
    # So a condition also holds when any package NAME providing it upstream is
    # installed, and an unresolvable condition fires fail-closed: a spurious
    # requirement can only refuse a bump, while a skipped one ships a lock
    # rpm-ostree cannot layer (proven against the pinned CoreOS base image).
    req = req.strip()
    m = re.match(r'^\((.*) if (.*)\)$', req)
    if m:
        cond, inner = m.group(2).strip(), m.group(1).strip()
        prov = providers(cond)
        if not prov:
            return inner
        if set(prov) & set(view.values()):
            return inner
        if {nevra_name(n) for n in prov} & set(view):
            return inner
        return None
    return req

def base_nevra(name, evr):
    e, vr = evr
    for arch in ARCHES:
        cand = f'{name}-{e}:{vr}.{arch}'
        if cand in have:
            return cand
    return ''

def check_tree(nev, world, names, stack, memo):
    # World-consistent walk of nev's requirements: a requirement already met
    # by the world's version stays (the solver keeps installed packages when
    # they satisfy it); otherwise the newest provider must not replace a
    # world member. Returns (ok, additions, refusal). The world is read-only;
    # callers merge accepted additions. Memoized per world generation.
    if nev in memo:
        return memo[nev]
    if nev in stack:
        memo[nev] = (True, set(), '')
        return memo[nev]
    additions = set()
    view = dict(world)
    view_names = set(names)
    for req in ok_lines(dnf('repoquery', '--requires', nev), f'requires {nev}'):
        want = concretize(req, view, view_names)
        if want is None:
            continue
        cands = providers(want)
        if not cands:
            memo[nev] = (False, set(), f'{nev} needs {want}: nothing provides it')
            return memo[nev]
        same = [p for p in cands if nevra_name(p) in view and p == view[nevra_name(p)]]
        if same:
            pick = same[0]
        else:
            pick = newest(cands)
            pn = nevra_name(pick)
            if pn in view:
                memo[nev] = (False, set(), f'{nev} needs {pick}, replacing {view[pn]}')
                return memo[nev]
        pn = nevra_name(pick)
        if pn in base:
            continue
        if pn not in view:
            additions.add(pick)
            view[pn] = pick
            view_names.add(pn)
        ok, sub, refusal = check_tree(pick, view, view_names, stack + (nev,), memo)
        if not ok:
            memo[nev] = (False, set(), refusal)
            return memo[nev]
        additions |= sub
    memo[nev] = (True, additions, '')
    return memo[nev]

def requirers(name):
    found = set()
    for flag in ('--whatrequires', '--whatrecommends', '--whatsuggests'):
        for n in ok_lines(dnf('repoquery', flag, name), f'{flag} {name}'):
            found.add(nevra_name(n))
    return found

# Frozen snapshot of the current world: base plus currently pinned install
# versions. Per-pin checks copy and extend it; the final pass uses exact
# chosen versions and is authoritative.
world0 = {}
for name, (evr, arch) in base.items():
    nev = base_nevra(name, evr)
    if not nev and arch in ARCHES:
        # Pruned upstream but installed in the base: retain the installed
        # NEVRA so conditions and replacements still see the real world.
        e, vr = evr
        nev = f'{name}-{e}:{vr}.{arch}'
    if nev:
        world0[name] = nev
for n in install:
    world0[nevra_name(n)] = n
names0 = set(world0) | set(requested)

bumps, drops, additions, refused = {}, [], set(), []
accepted = {}
old_refusal = {}
pending = sorted(missing, key=nevra_name)
# A bump can unblock another pin (botocore .93 satisfies boto3 .93's range),
# so repeat passes until a pass accepts nothing new.
while pending:
    progressed = False
    world = dict(world0)
    world.update(accepted)
    names = names0 | set(world)
    for old in list(pending):
        name, _, _, arch = split_nevra(old)
        cands = sorted(by_name_arch.get((name, arch), []),
                       key=functools.cmp_to_key(
                           lambda a, b: rpm.labelCompare(evr_key(split_nevra(a)[1:3]), evr_key(split_nevra(b)[1:3]))),
                       reverse=True)
        last_refusal = 'no available build'
        for cand in cands:
            ok, sub, refusal = check_tree(cand, world, names, (), {})
            if ok:
                bumps[name] = (old, cand)
                accepted[name] = cand
                world[name] = cand
                names.add(name)
                additions |= {n for n in sub if nevra_name(n) not in world}
                pending.remove(old)
                progressed = True
                break
            if last_refusal == 'no available build':
                last_refusal = refusal
        else:
            old_refusal[old] = last_refusal
    if not progressed:
        break

world = dict(world0)
world.update(accepted)
names = names0 | set(world)
drifted_names = {nevra_name(p) for p in missing}
for old in pending:
    name = nevra_name(old)
    reason = old_refusal[old]
    # A 'replacing X' block on a sibling that is itself drifted is an ordering
    # artifact: the pin could move jointly. Chain the sibling's own refusal so
    # the message points at the root base conflict, not the joint move.
    m = re.search(r'replacing (\S+)', reason)
    if m and nevra_name(m.group(1)) in drifted_names - {name}:
        sib = next(p for p in missing if nevra_name(p) == nevra_name(m.group(1)))
        reason += f'; {sib} is itself blocked ({old_refusal.get(sib, "no replacement")})'
    if name in requested:
        refused.append(f'{old}: requested package no longer installable ({reason})')
        continue
    if requirers(name) & names - {name}:
        refused.append(f'{old}: no base-compatible replacement ({reason})')
        continue
    drops.append(old)

if refused:
    sys.exit('relock refused; base bump likely needed:\n  ' + '\n  '.join(refused))

print(f'plan: {len(bumps)} bumped, {len(additions)} added, {len(drops)} dropped')
for name in sorted(bumps):
    print(f'  ~ {bumps[name][0]}\n    -> {bumps[name][1]}')
for n in sorted(additions, key=nevra_name):
    print(f'  + {n} (new dependency)')
for n in drops:
    print(f'  - {n} (nothing pulls it anymore)')

# Final pass with exact chosen versions before composing anything.
chosen = {nevra_name(v[1]): v[1] for v in bumps.values()}
for n in additions:
    chosen.setdefault(nevra_name(n), n)
for old in install:
    if nevra_name(old) not in chosen and old not in drops:
        chosen[nevra_name(old)] = old
final_world = dict(world0)
final_world.update(chosen)
final_names = names0 | set(final_world)
final_memo = {}
for name, nev in sorted(chosen.items()):
    ok, _, refusal = check_tree(nev, final_world, final_names, (), final_memo)
    if not ok:
        sys.exit(f'combined transaction fails on {nev}: {refusal}')

new_install = sorted(chosen.values())
if set(nevra_name(n) for n in new_install) & set(base):
    sys.exit('internal error: install would replace base packages')


def to_inv(nevra):
    m = re.match(r'^(.*)-(\d+:.*)$', nevra)
    return m.group(1) + ' ' + m.group(2)


base_part = [l for l in inventory if l.split(' ')[0] not in set(nevra_name(n) for n in new_install)]
new_inventory = sorted(base_part + [to_inv(n) for n in new_install])
name_re = re.compile(r'^[a-z0-9][a-zA-Z0-9+._:-]*$')
inv_re = re.compile(r'^[a-zA-Z0-9][a-zA-Z0-9+._-]* [0-9]+:[a-zA-Z0-9+._~^-]+$')
if len(set(new_install)) != len(new_install) or any(not name_re.match(nevra_name(n)) for n in new_install):
    sys.exit('install validation failed')
if len(set(new_inventory)) != len(new_inventory) or any(not inv_re.match(l) for l in new_inventory):
    sys.exit('inventory validation failed (sorted unique RPM inventory required)')

if check_only:
    print('--check: no files written')
    sys.exit(0)

lock['Install'] = new_install
lock['Inventory'] = new_inventory
open(lock_path, 'w').write(json.dumps(lock, indent=2) + '\n')
print(f'wrote {lock_path}: {len(new_install)} install pins, {len(new_inventory)} inventory lines')

if default_lock:
    test = 'internal/release/image/complete_test.go'
    src = open(test).read()
    pat = re.compile(r'(require\.Equal\(t, )\d+(, len\(strings\.FieldsFunc)')
    updated, count = pat.subn(r'\g<1>%d\g<2>' % len(new_inventory), src)
    if count != 1:
        sys.exit(f'wrote lock but could not locate inventory-count assertion in {test}; update it to {len(new_inventory)} by hand')
    open(test, 'w').write(updated)
    print(f'synced {test} inventory count to {len(new_inventory)}')
PY
if [ "$CHECK" -eq 1 ] || [ "$LOCK" != appliance/locks/host-packages-x86_64.json ]; then
  exit 0
fi
go test ./internal/release/image/ -run TestLockedHostTransactionMatchesActualPackageOwner 2>&1 | tail -2
