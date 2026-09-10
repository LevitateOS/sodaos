#!/usr/bin/env python3
"""Build a CoreOS ISO carrying Soda's console; never boot, publish or install it.

Requires explicitly supplied native Butane/CoreOS Installer tools and trusted
Fedora signing inputs. Outputs are fresh and retained, including failed attempts.
"""
import argparse
import base64
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import re
import stat
import struct
import subprocess

ROOT = Path(__file__).resolve().parents[1]
INSTALLER_VERSION = 'coreos-installer 0.26.0'
LOADER_PATH = '/var/usrlocal/libexec/soda/load-install-console'
DATA_PATH = '/var/usrlocal/share/soda-installer'
BOOT_CONFIGS = ('/EFI/fedora/grub.cfg', '/isolinux/isolinux.cfg')


def sha256(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def payload_files(root):
    """Return exact regular ISO payload inputs without following symlinks."""
    return {name: value[2] for name, value in payload_entries(root).items()
            if value[0] == '-'}


def payload_entries(root):
    """Return type, mode, path/link for every /soda Rock Ridge entry."""
    root_mode = root.lstat().st_mode
    if not stat.S_ISDIR(root_mode):
        raise ValueError('real media payload directory required')
    result = {'/soda': ('d', root_mode & 0o7777, root)}
    for path in sorted(root.rglob('*')):
        info = path.lstat()
        mode = info.st_mode
        name = '/soda/' + path.relative_to(root).as_posix()
        if "'" in name or '\n' in name:
            raise ValueError('unsupported media payload path')
        if mode & (stat.S_ISUID | stat.S_ISGID | stat.S_ISVTX):
            raise ValueError('special media payload permissions refused')
        if stat.S_ISREG(mode):
            if info.st_size > (1 << 32) - 1:
                raise ValueError('ISO 9660 level 1 cannot carry an individual payload file larger than 4 GiB minus 1 byte')
            result[name] = ('-', mode & 0o7777, path)
        elif stat.S_ISDIR(mode):
            result[name] = ('d', mode & 0o7777, path)
        elif stat.S_ISLNK(mode):
            target = os.readlink(path)
            if "'" in target or '\n' in target:
                raise ValueError('unsupported media payload link')
            result[name] = ('l', mode & 0o7777, target)
        else:
            raise ValueError('unsupported media payload file type')
    if len(result) == 1:
        raise ValueError('empty media payload')
    return result


def mode_bits(text):
    if not re.fullmatch('[rwx-]{9}', text):
        raise ValueError('unsupported ISO permissions')
    value = 0
    for index, bit in enumerate((0o400, 0o200, 0o100, 0o040, 0o020, 0o010, 0o004, 0o002, 0o001)):
        if text[index] != '-':
            value |= bit
    return value


def iso_payload_entries(xorriso, iso):
    """Read back Rock Ridge types, modes and numeric ownership from the ISO."""
    report = subprocess.check_output([str(xorriso), '-indev', str(iso), '-find',
        '/soda', '-exec', 'lsdl', '--'], stderr=subprocess.DEVNULL, text=True)
    result = {}
    for line in report.splitlines():
        match = re.fullmatch(r"([dl-])([rwx-]{9})\s+\d+\s+(\d+)\s+(\d+)\s+.*?\s'([^']+)'(?: -> '([^']*)')?", line)
        if not match or match[5] in result:
            raise ValueError('unsupported or ambiguous Rock Ridge payload metadata')
        kind, mode, uid, gid, name, target = match.groups()
        if uid != '0' or gid != '0':
            raise ValueError('on-media payload must be owned by root')
        result[name] = (kind, mode_bits(mode), target if kind == 'l' else None)
    if not result:
        raise ValueError('missing Rock Ridge payload metadata')
    return result


def snapshot_bundle(verifier, source, destination, arch, revision):
    source = Path(source)
    if not source.is_absolute() or source.resolve() != source or not source.is_dir():
        raise ValueError('absolute canonical sealed bundle source required')
    destination.parent.mkdir(mode=0o700)
    subprocess.run([str(verifier), 'bundle', '--source', str(source), '--out', str(destination),
                    '--arch', arch, '--revision', revision], check=True)
    digest = sha256(destination / 'SHA256SUMS')
    if not re.fullmatch('[0-9a-f]{64}', digest):
        raise ValueError('sealed bundle manifest digest required')
    return digest


def verify_bundle_readback(xorriso, iso, out, verifier, arch, revision, expected_digest):
    readback = out / 'bundle-readback'
    readback.mkdir(mode=0o700)
    destination = readback / arch
    subprocess.run([str(xorriso), '-osirrox', 'on', '-indev', str(iso), '-extract',
                    '/soda/bundle/' + arch, str(destination)], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if sha256(destination / 'SHA256SUMS') != expected_digest:
        raise ValueError('on-media bundle manifest differs from sealed snapshot')
    subprocess.run([str(verifier), 'verify', '--source', str(destination), '--arch', arch,
                    '--revision', revision], check=True)


def live_config(binary_hash, destination, media, artwork, branding):
    # The small Ignition configuration fits the embed area. The executable is an
    # ordinary ISO file, copied and verified from the native read-only live mount.
    if not re.fullmatch('[0-9a-f]{64}', binary_hash):
        raise ValueError('payload digest required')
    profile = """# Guidance only; no automatic disk action or credential collection.
case $- in
  *i*) if [ -t 1 ]; then
    printf '%s\\n' 'SodaOS installer: sudo /usr/local/libexec/soda/soda-install disk'
    printf '%s\\n' 'Fresh installation only. Disk erasure requires explicit confirmation.'
  fi ;;
esac
"""
    return {
        'variant': 'fcos', 'version': '1.6.0',
        # A supplied live Ignition disables CoreOS's default console autologin.
        # Run just the bounded UI on tty1, not an unauthenticated root shell.
        'systemd': {'units': [{'name': 'soda-installer-console.service', 'enabled': True,
            'contents': '[Unit]\nDescription=SodaOS installation console\n'
                        'After=systemd-user-sessions.service NetworkManager.service\nConflicts=getty@tty1.service\n'
                        'RequiresMountsFor=/run/media/iso\n'
                        '[Service]\nType=idle\n'
                        'PrivateMounts=yes\n'
                        f'ExecStart=/usr/local/libexec/soda/load-install-console {binary_hash}\n'
                        'StandardInput=tty-force\nStandardOutput=tty\nStandardError=tty\n'
                        'TTYPath=/dev/tty1\nTTYReset=yes\nTTYVHangup=yes\n'
                        'Restart=no\n[Install]\nWantedBy=multi-user.target\n'},
            {'name': 'getty@tty1.service', 'mask': True}]},
        'storage': {'files': branding + [
            {'path': LOADER_PATH, 'mode': 0o755,
             'contents': {'inline': (ROOT / 'appliance/installer/load-console.sh').read_text()}},
            {'path': DATA_PATH + '/destination.ign', 'mode': 0o644,
             'contents': {'inline': json.dumps(destination)}},
            {'path': DATA_PATH + '/media.json', 'mode': 0o644,
             'contents': {'inline': json.dumps(media)}},
            {'path': '/etc/profile.d/soda-installer.sh', 'mode': 0o644,
             'contents': {'inline': profile}},
            # Stock CoreOS already supplies this file. Replace only the live
            # welcome text, not arbitrary existing provisioning destinations.
            {'path': '/etc/motd', 'mode': 0o644, 'overwrite': True,
             'contents': {'inline': artwork + '\nSodaOS Installer\n'
                          'No disk is selected or erased automatically.\n'}},
        ]},
    }


def convert(butane, config, out):
    # No credentials reach Butane in this path. Still suppress converter
    # diagnostics: this is the same protection as the private provisioning caller.
    result = subprocess.run([str(butane), '--strict'],
                            input=json.dumps(config).encode(), stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, check=True, timeout=60)
    data = json.loads(result.stdout)
    if data.get('ignition', {}).get('version') != '3.5.0':
        raise ValueError('expected strict FCOS 1.6.0 -> Ignition 3.5.0 conversion')
    with out.open('xb') as f:
        f.write(result.stdout)
    return data


def output(args, cwd=ROOT, env=None):
    return subprocess.check_output(args, cwd=cwd, env=env, timeout=120).decode().strip()


def tool(path):
    path = Path(path)
    if not path.is_absolute() or not path.is_file() or not os.access(path, os.X_OK):
        raise ValueError('absolute existing native tool executable required')
    return path.resolve()


def build(args):
    if platform.system() != 'Linux' or platform.machine() != args.arch:
        raise ValueError('matching native Linux required')
    if output(['git', 'status', '--porcelain', '--untracked-files=normal']):
        raise ValueError('clean exact-revision source required')
    revision = output(['git', 'rev-parse', 'HEAD'])
    butane, installer = tool(args.butane), tool(args.coreos_installer)
    installer_version = output([str(installer), '--version'])
    if installer_version != INSTALLER_VERSION:
        raise ValueError('selected CoreOS Installer 0.26.0 required')
    butane_version = output([str(butane), '--version'])
    env = dict(os.environ, GOTOOLCHAIN='local', GOWORK='off', GOFLAGS='-mod=readonly',
               GOOS='linux', GOARCH={'x86_64': 'amd64', 'aarch64': 'arm64'}[args.arch],
               CGO_ENABLED='0')
    go_version = output(['go', 'env', 'GOVERSION'], env=env)
    if go_version != 'go1.26.7':
        raise ValueError('repository-pinned Go toolchain required')
    lock = ROOT / 'appliance/locks/coreos-iso.json'
    selected = json.loads(lock.read_text())
    filename = 'soda-install'
    xorriso = tool(args.xorriso)
    xorriso_version = output([str(xorriso), '-version'])
    out = Path(args.out)
    artifact_root = ROOT / '.artifacts'
    if (not out.is_absolute() or out.parent.resolve() != out.parent
            or not out.is_relative_to(artifact_root) or not out.parent.is_dir()):
        raise ValueError('new absolute output below real existing .artifacts parent required')
    for parent in (out.parent, *out.parent.parents):
        st = parent.stat()
        if st.st_uid != os.getuid() or st.st_mode & 0o022:
            raise ValueError('output ancestry must be owned and not writable by others')
        if parent == ROOT:
            break
    out.mkdir(mode=0o700)  # exclusive; never clear previous attempts
    payload = out / 'payload'
    payload.mkdir()
    subprocess.run(['go', 'mod', 'verify'], cwd=ROOT, env=env, check=True)
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=true', '-ldflags=-s -w',
                    '-o', str(payload / filename), './appliance/installer'],
                   cwd=ROOT, env=env, check=True)
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=true',
                    '-o', str(out / 'soda-artifacts'), './tools/soda-artifacts'],
                   cwd=ROOT, env=env, check=True)
    subprocess.run([str(out / 'soda-artifacts'), 'fetch-coreos-iso', '--arch', args.arch,
                    '--lock', str(lock), '--keyring', str(Path(args.keyring).absolute()),
                    '--signer', args.signer, '--out', str(out / 'upstream')], check=True)
    # Reuse the production public bootstrap; all per-machine fields are collected
    # privately on the live host, never embedded in general-purpose output.
    import importlib.util
    spec = importlib.util.spec_from_file_location('soda_provisioning', ROOT / 'scripts/render-provisioning.py')
    provisioning = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(provisioning)
    destination = convert(butane, provisioning.public_config(), out / 'destination.ign')
    bundle = payload / 'bundle' / args.arch
    bundle_digest = snapshot_bundle(out / 'soda-artifacts', args.bundle_source, bundle,
                                    args.arch, revision)
    media = {'Architecture': args.arch, 'Release': selected['Release'],
             'InstallerVersion': installer_version, 'Revision': revision,
             'BundleSHA256': bundle_digest}
    artwork = (ROOT / 'assets/branding/terminal/sodaos.txt').read_text()
    for marker in ('$1', '$2', '$3'):
        artwork = artwork.replace(marker, '')
    for name in ('LICENSE', 'NOTICE'):
        (payload / name).write_bytes((ROOT / name).read_bytes())
        (payload / name).chmod(0o644)
    (payload / filename).chmod(0o755)
    payload.chmod(0o755)
    config = live_config(sha256(payload / filename), destination, media, artwork, provisioning.branding_files())
    convert(butane, config, out / 'live.ign')
    remaster(xorriso, out / 'upstream/coreos.iso', payload, out / 'with-console.iso', out / 'remaster.log')
    customize = [str(installer), 'iso', 'customize', '--live-ignition', str(out / 'live.ign')]
    network = None
    if args.network_keyfile:
        network = snapshot_network(Path(args.network_keyfile), out)
        customize += ['--network-keyfile', str(network)]
    subprocess.run(customize + ['--output', str(out / 'soda.iso'),
                               str(out / 'with-console.iso')], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    verify_bundle_readback(xorriso, out / 'soda.iso', out, out / 'soda-artifacts',
                           args.arch, revision, bundle_digest)
    if network:
        extracted = out / 'network-readback'
        extracted.mkdir(mode=0o700)
        subprocess.run([str(installer), 'iso', 'network', 'extract', '--directory',
                        str(extracted), str(out / 'soda.iso')], check=True,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if list(extracted.iterdir()) != [extracted / network.name] or sha256(extracted / network.name) != sha256(network):
            raise ValueError('embedded network configuration differs from private input')
    # Read back the exact public live configuration. Optional private networking
    # is separately verified above. No destination-device/config, insecure kargs,
    # initramfs rebuild or OS source patch occurs.
    embedded = subprocess.check_output([str(installer), 'iso', 'ignition', 'show',
                                       str(out / 'soda.iso')], timeout=60,
                                      stderr=subprocess.DEVNULL)
    verify_embedded(embedded, json.loads((out / 'live.ign').read_bytes()))
    preservation = verify_remaster(xorriso, out / 'upstream/coreos.iso', out / 'soda.iso', payload)
    before = output([str(installer), 'iso', 'kargs', 'show', str(out / 'upstream/coreos.iso')])
    after = output([str(installer), 'iso', 'kargs', 'show', str(out / 'soda.iso')])
    if before != after:
        raise ValueError('upstream live kernel arguments changed')
    preservation['LiveKernelArguments'] = before
    (out / 'iso-inspection.json').write_text(json.dumps(preservation, indent=2) + '\n')
    if output(['git', 'rev-parse', 'HEAD']) != revision or output(['git', 'status', '--porcelain', '--untracked-files=normal']):
        raise ValueError('source changed during media build; output is not sealed')
    record = dict(media, ConsoleISOPath='/soda/soda-install',
                  BundleISOPath='/soda/bundle/' + args.arch,
                  ConsoleSHA256=sha256(payload / filename),
                  XorrisoVersion=xorriso_version, XorrisoSHA256=sha256(xorriso),
                  ISOSHA256=sha256(out / 'soda.iso'), LiveIgnitionSHA256=sha256(out / 'live.ign'),
                  PrivateMedia=network is not None,
                  NetworkKeyfileSHA256=sha256(network) if network else None,
                  CoreOSLockSHA256=sha256(lock), GoVersion=go_version,
                  ButaneVersion=butane_version,
                  ButaneSHA256=sha256(butane), CoreOSInstallerSHA256=sha256(installer))
    (out / 'media-build.json').write_text(json.dumps(record, indent=2) + '\n')
    (out / 'SHA256SUMS').write_text(''.join(f'{sha256(p)}  {p.relative_to(out)}\n' for p in
        (out / 'soda.iso', out / 'live.ign', out / 'destination.ign', payload / filename,
         bundle / 'SHA256SUMS', bundle / 'build-info.json',
         out / 'media-build.json', out / 'iso-inspection.json')))
    print(f'Media built, not booted or installed: {out}')
    if network:
        print('PRIVATE PER-MACHINE ISO: contains network configuration; do not publish or distribute as general media.')
    print('The console is on the ISO; no web server or payload URL is required. Soda RPM setup still needs network access.')
    print('No hosting, publication, certificate changes, disk writes or validation VM were performed.')


def branded_boot_config(name, data):
    # Keep each replacement byte-for-byte the same length so the stock kargs
    # embed offsets stay valid. Do not edit commands, paths or attribution.
    old = b'Fedora CoreOS (Live)'
    if data.count(old) != 1:
        raise ValueError('unexpected upstream boot menu label')
    branded = data.replace(old, b'SodaOS Installer'.ljust(len(old)))
    if name == '/EFI/fedora/grub.cfg':
        if branded.count(b'--class fedora') != 1:
            raise ValueError('unexpected upstream GRUB menu class')
        branded = branded.replace(b'--class fedora', b'--class sodaos')
    elif name == '/isolinux/isolinux.cfg':
        old = b'menu title Fedora CoreOS'
        if branded.count(old) != 1:
            raise ValueError('unexpected upstream BIOS menu title')
        branded = branded.replace(old, b'menu title SodaOS'.ljust(len(old)))
    else:
        raise ValueError('unsupported boot branding path')
    if len(branded) != len(data):
        raise ValueError('boot branding shifted native embed offsets')
    return branded


def read_boot_config(iso, entry):
    offset, size = entry
    if size > 65536:
        raise ValueError('boot config exceeds bound')
    with iso.open('rb') as source:
        source.seek(offset)
        data = source.read(size)
    if len(data) != size:
        raise ValueError('truncated boot config')
    return data


def brand_boot_files(xorriso, upstream, directory):
    files = iso_files(xorriso, upstream)
    if BOOT_CONFIGS[0] not in files:
        raise ValueError('selected EFI menu config missing')
    directory.mkdir()
    result = {}
    for name in BOOT_CONFIGS:
        if name not in files:
            continue  # aarch64 has no ISOLINUX BIOS menu
        target = directory / name.lstrip('/')
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(branded_boot_config(name, read_boot_config(upstream, files[name])))
        result[name] = target
    return result


def remaster(xorriso, upstream, payload, destination, log):
    if destination.exists() or destination.is_symlink():
        raise FileExistsError('refusing occupied ISO output')
    branded = brand_boot_files(xorriso, upstream, destination.parent / 'boot-branding')
    menu_args = []
    for name, path in branded.items():
        menu_args.extend(['-map', str(path), name, '-chown', '0', name, '--', '-chgrp', '0', name, '--'])
    with log.open('xb') as capture:
        # Replay the imported BIOS/EFI hybrid boot equipment, not a guessed
        # mkisofs command. Add /soda, replace bounded display labels and remove
        # stale miniso metadata; upstream OS images remain untouched.
        subprocess.run([str(xorriso), '-abort_on', 'FAILURE', '-indev', str(upstream),
                        '-outdev', str(destination), '-boot_image', 'any', 'replay',
                        '-follow', 'off',
                        # v0.26.0 reads primary ISO names such as KARGS.JSO;
                        # Rock Ridge names alone are not its lookup contract.
                        '-compliance', 'iso_9660_level=1',
                        '-map', str(payload), '/soda', '-chown_r', '0', '/soda', '--',
                        '-chgrp_r', '0', '/soda', '--', '-rm', '/coreos/miniso.dat', '--',
                        *menu_args, '-commit', '-end'], check=True, stdout=capture, stderr=capture)


def iso_files(xorriso, iso, filesystem='any'):
    report = subprocess.check_output([str(xorriso), '-read_fs', filesystem, '-indev', str(iso), '-find', '/',
        '-type', 'f', '-exec', 'report_lba', '--'], stderr=subprocess.DEVNULL, text=True)
    files = {}
    for line in report.splitlines():
        if not line.startswith('File data lba:'):
            continue
        match = re.fullmatch(r"File data lba:\s+0\s*,\s*(\d+)\s*,\s*\d+\s*,\s*(\d+)\s*,\s*'([^']+)'", line)
        if not match or match[3] in files:
            raise ValueError('unsupported multi-extent or ambiguous ISO file inventory')
        offset, size = int(match[1]) * 2048, int(match[2])
        if offset + size > iso.stat().st_size:
            raise ValueError('ISO file extent exceeds image')
        files[match[3]] = (offset, size)
    if not files:
        raise ValueError('empty ISO file inventory')
    return files


def iso_digest(iso, entry, boot_info=False):
    offset, size = entry
    digest = hashlib.sha256()
    with iso.open('rb') as source:
        source.seek(offset)
        if boot_info:
            # ISOLINUX's standard boot-info table is patched by xorriso when
            # relocating the file. Validate its real LBA/length/checksum and
            # compare all executable bytes outside those four 32-bit fields.
            data = source.read(size)
            if len(data) != size or size < 64 or size % 4:
                raise ValueError('invalid BIOS boot image size')
            pvd, lba, length, checksum = struct.unpack_from('<IIII', data, 8)
            actual = sum(struct.unpack('<' + 'I' * ((size - 64) // 4), data[64:])) & 0xffffffff
            if (lba, length, checksum) != (offset // 2048, size, actual) or pvd < 16:
                raise ValueError('BIOS boot-info table does not match relocated image')
            # xorriso's emulated session can reference a PVD at LBA 48.
            # Its volume-space size excludes the 32-block session prefix; root
            # directory and every other descriptor byte must remain identical.
            source.seek(16 * 2048)
            primary = source.read(2048)
            source.seek(pvd * 2048)
            referenced = source.read(2048)
            if (len(referenced) != 2048 or primary[:7] != b'\x01CD001\x01'
                    or primary[:80] != referenced[:80] or primary[88:] != referenced[88:]
                    or int.from_bytes(primary[80:84], 'little') != int.from_bytes(primary[84:88], 'big')
                    or int.from_bytes(referenced[80:84], 'little') != int.from_bytes(referenced[84:88], 'big')
                    or int.from_bytes(primary[80:84], 'little') - int.from_bytes(referenced[80:84], 'little') != pvd - 16):
                raise ValueError('BIOS boot-info table references the wrong volume descriptor')
            digest.update(data[:8] + bytes(16) + data[24:])
        else:
            remaining = size
            while remaining:
                data = source.read(min(remaining, 1024 * 1024))
                if not data:
                    raise ValueError('truncated ISO file')
                digest.update(data)
                remaining -= len(data)
    return digest.hexdigest()


def boot_metadata(xorriso, iso, files):
    report = subprocess.check_output([str(xorriso), '-indev', str(iso), '-pvd_info',
        '-report_el_torito', 'plain', '-report_system_area', 'plain'],
        stderr=subprocess.DEVNULL, text=True)
    result = {'volume': '', 'system_area': '', 'images': {}, 'paths': {}, 'partition_paths': {}}
    lbas = {}
    for line in report.splitlines():
        key, sep, value = line.partition(':')
        if not sep:
            continue
        key, value = key.strip(), value.strip()
        if key == 'Volume Id':
            result['volume'] = value
        elif key == 'System area summary':
            result['system_area'] = value
        elif key == 'El Torito boot img':
            fields = value.split()
            if len(fields) != 8 or fields[1] not in ('BIOS', 'UEFI') or fields[2] != 'y':
                raise ValueError('unsupported or disabled ISO boot entry')
            result['images'][fields[0]] = fields[1:-1]
            lbas[fields[0]] = int(fields[-1])
        elif key == 'El Torito img path':
            number, path = value.split(maxsplit=1)
            result['paths'][number] = path
        elif key in ('MBR partition path', 'GPT partition path'):
            result['partition_paths'][key + ' ' + value.split()[0]] = value.split(maxsplit=1)[1]
    if not result['volume'] or not result['images'] or set(result['images']) != set(result['paths']):
        raise ValueError('incomplete ISO boot metadata')
    for number, path in result['paths'].items():
        if path not in files or files[path][0] // 2048 != lbas[number]:
            raise ValueError('El Torito boot entry points outside its boot image')
    return result


def verify_remaster(xorriso, upstream, final, payload):
    original = iso_files(xorriso, upstream)
    current = iso_files(xorriso, final)
    added = payload_files(payload)
    local_entries = payload_entries(payload)
    on_media_entries = iso_payload_entries(xorriso, final)
    expected_entries = {name: (kind, mode, value if kind == 'l' else None)
                        for name, (kind, mode, value) in local_entries.items()}
    if on_media_entries != expected_entries:
        raise ValueError('on-media Rock Ridge payload type, mode or link differs from source')
    removed = {'/coreos/miniso.dat'}
    if set(current) != (set(original) - removed) | set(added):
        raise ValueError('unexpected remastered ISO file inventory')
    before = boot_metadata(xorriso, upstream, original)
    after = boot_metadata(xorriso, final, current)
    if before != after:
        raise ValueError('upstream volume identity or boot equipment changed')
    preserved = {}
    bios = {}
    branded = {}
    for name, entry in original.items():
        if name in removed or name == '/images/ignition.img':
            continue
        if name in BOOT_CONFIGS:
            expected = hashlib.sha256(branded_boot_config(name, read_boot_config(upstream, entry))).hexdigest()
            if iso_digest(final, current[name]) != expected:
                raise ValueError('boot configuration differs beyond selected branding: ' + name)
            branded[name] = expected
            continue
        boot_info = name == '/isolinux/isolinux.bin'
        expected = iso_digest(upstream, entry, boot_info)
        if expected != iso_digest(final, current[name], boot_info):
            raise ValueError('upstream ISO file changed: ' + name)
        (bios if boot_info else preserved)[name] = expected
    for name, path in added.items():
        if iso_digest(final, current[name]) != sha256(path):
            raise ValueError('on-media console payload differs from built source')
    return {'BootMetadata': after, 'PreservedFileSHA256': preserved, 'BrandedBootFileSHA256': branded,
            'BIOSBootInfoNormalizedSHA256': bios, 'RemovedMetadata': sorted(removed),
            'AddedFileSHA256': {name: sha256(path) for name, path in added.items()}}


def verify_embedded(raw, expected):
    # v0.26.0 LiveInitrd::live_config wraps each fragment in a gzip data-URL
    # merge. Admit that one exact wrapper, not arbitrary extra installer effects.
    wrapper = json.loads(raw)
    if (set(wrapper) != {'ignition'} or set(wrapper['ignition']) != {'version', 'config'}
            or wrapper['ignition']['version'] != '3.3.0'):
        raise ValueError('unexpected embedded Ignition effects')
    config = wrapper['ignition']['config']
    if set(config) != {'merge'} or len(config['merge']) != 1:
        raise ValueError('expected one embedded live fragment')
    resource = config['merge'][0]
    if set(resource) != {'source', 'compression'} or resource['compression'] != 'gzip' or not resource['source'].startswith('data:;base64,'):
        raise ValueError('unexpected live fragment reference')
    with gzip.GzipFile(fileobj=io.BytesIO(base64.b64decode(resource['source'].split(',', 1)[1], validate=True))) as source:
        decoded = source.read(4 * 1024 * 1024 + 1)
    if len(decoded) > 4 * 1024 * 1024:
        raise ValueError('embedded live fragment exceeds limit')
    actual = json.loads(decoded)
    if actual == expected:
        return
    # v0.26.0 serializes through ignition-config 0.6.1. These particular
    # optional v3.5 fields use serde(default), not skip_serializing_if, so
    # absent inputs become explicit nulls. Admit only that exact expansion;
    # do not strip arbitrary nulls or accept non-null extra effects.
    serialized = json.loads(json.dumps(expected))
    for file in serialized.get('storage', {}).get('files', []):
        file.setdefault('overwrite', None)
    for unit in serialized.get('systemd', {}).get('units', []):
        for field in ('contents', 'enabled', 'mask'):
            unit.setdefault(field, None)
    if actual != serialized:
        raise ValueError('ISO embedded Ignition differs from generated live config')


def snapshot_network(source, out):
    if not source.is_absolute() or source.parent.resolve() != source.parent:
        raise ValueError('absolute private NetworkManager keyfile required')
    with os.fdopen(os.open(source, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), 'rb') as f:
        st = os.fstat(f.fileno())
        if not stat.S_ISREG(st.st_mode) or st.st_mode & 0o077 or st.st_size > 65536:
            raise ValueError('bounded mode-0600 network keyfile required')
        data = f.read(65537)
    if len(data) > 65536:
        raise ValueError('network keyfile exceeds limit')
    dest = out / 'soda-installer.nmconnection'
    with dest.open('xb') as f:
        os.chmod(dest, 0o600)
        f.write(data)
    return dest


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    p.add_argument('--butane', required=True)
    p.add_argument('--coreos-installer', required=True)
    p.add_argument('--keyring', required=True)
    p.add_argument('--signer', required=True)
    p.add_argument('--bundle-source', required=True,
                   help='absolute canonical sealed native stage or exported bundle')
    p.add_argument('--xorriso', default='/usr/bin/xorriso')
    p.add_argument('--network-keyfile', help='optional private NetworkManager keyfile for pre-Ignition/static networking; makes the ISO private')
    p.add_argument('--out', required=True)
    args = p.parse_args()
    try:
        build(args)
    except (ValueError, OSError, subprocess.SubprocessError) as err:
        p.exit(1, f'Media build failed ({type(err).__name__}); retained outputs are not boot/install proof.\n')


if __name__ == '__main__':
    main()
