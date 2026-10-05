#!/usr/bin/env python3
"""Install/reinstall/remove our frozen package in the marked disposable VM only."""
import hashlib
import json
import os
import pathlib
import subprocess
import sys

assert os.geteuid() == 0
assert pathlib.Path('/sys/class/dmi/id/product_name').read_text().strip() == 'grubmgr-disposable-v1'
base = pathlib.Path('/home/tester/grubmgr-test')
manifest = json.loads((base/'grubmgr.manifest.json').read_text())
assert hashlib.sha256((base/'grubmgr.deb').read_bytes()).hexdigest() == manifest['sha256']


def boot_files():
    names = ['/etc/default/grub', '/boot/grub/grub.cfg']
    for folder in ['/boot/grub/themes/grubmgr', '/var/lib/grubmgr']:
        names += [str(p) for p in pathlib.Path(folder).rglob('*') if p.is_file()]
    return {p: hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest() for p in sorted(names)}


def command(*args):
    result = subprocess.run(args, capture_output=True, text=True)
    with (base/'evidence/package-commands.jsonl').open('a') as log:
        log.write(json.dumps({'argv': args, 'exit': result.returncode,
                              'stdout': result.stdout, 'stderr': result.stderr})+'\n')
    assert result.returncode == 0, result.stdout+result.stderr
    return result.stdout


def installed():
    for name, digest in manifest['files'].items():
        p = pathlib.Path('/')/name
        assert p.stat().st_uid == 0 and p.stat().st_mode & 0o022 == 0, name
        assert hashlib.sha256(p.read_bytes()).hexdigest() == digest, name
    assert pathlib.Path('/var/lib/grubmgr').stat().st_mode & 0o777 == 0o700
    assert command('dpkg', '--verify', 'grubmgr') == ''
    version = command('dpkg-query', '-W', '-f=${Version}', 'grubmgr')
    assert version == manifest['version']


stage = sys.argv[1]
before = boot_files()
if stage == 'install':
    assert not pathlib.Path('/etc/grubmgr/vm-test').exists()
    assert not pathlib.Path('/usr/local/bin/grubmgr').exists()
    command('dpkg', '--install', str(base/'grubmgr.deb'))
    installed()
    assert boot_files() == before
    assert not pathlib.Path('/etc/grubmgr/vm-test').exists()
    report = command('runuser', '-u', 'tester', '--', '/usr/bin/grubmgr', '--json', 'doctor')
    assert 'UNSUPPORTED' in report, report
elif stage == 'lifecycle':
    command('dpkg', '--install', str(base/'grubmgr.deb'))
    installed()
    assert boot_files() == before
    command('dpkg', '--remove', 'grubmgr')
    assert boot_files() == before
    assert not pathlib.Path('/usr/bin/grubmgr').exists()
    assert not pathlib.Path('/usr/libexec/grubmgr-helper').exists()
    command('dpkg', '--purge', 'grubmgr')
    assert boot_files() == before
    command('dpkg', '--install', str(base/'grubmgr.deb'))
    installed()
    assert boot_files() == before
else:
    raise SystemExit('unknown stage')
(base/'evidence'/('package-'+stage+'.json')).write_text(json.dumps({
    'package_sha256': manifest['sha256'], 'version': manifest['version'],
    'stage': stage, 'boot_and_retained_files_unchanged': True,
    'installed_files_verified': len(manifest['files'])}, indent=2)+'\n')
