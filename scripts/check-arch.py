#!/usr/bin/env python3
"""Inspect a pacman archive without installing it or running hooks."""
import hashlib
import json
import pathlib
import sys
import tarfile
import xml.etree.ElementTree as ET


def check(archive):
    archive = pathlib.Path(archive)
    manifest = json.loads(archive.with_suffix('.manifest.json').read_text())
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == manifest['sha256']
    actual = {}; seen = set(); info = None
    with tarfile.open(archive) as tar:
        for member in tar:
            name = member.name.rstrip('/')
            assert name not in seen and not name.startswith('/') and '..' not in pathlib.PurePosixPath(name).parts
            seen.add(name)
            assert member.isfile() or member.isdir(), name
            assert member.uid == member.gid == 0 and member.mode & 0o6022 == 0, name
            if member.isdir():
                if name == 'var/lib/grubmgr': assert member.mode == 0o700
                continue
            data = tar.extractfile(member).read()
            if name == '.PKGINFO':
                info = data.decode(); continue
            assert name.startswith(('usr/bin/', 'usr/libexec/', 'usr/share/')), name
            actual[name] = hashlib.sha256(data).hexdigest()
            if name in ('usr/bin/grubmgr', 'usr/libexec/grubmgr-helper', 'usr/libexec/grubmgr-preview-qemu'):
                assert member.mode == 0o755 and data[:5] == b'\x7fELF\x02' and data[18:20] == b'\x3e\x00'
            if name.endswith('.policy'):
                action = ET.fromstring(data).find('action')
                assert action.findtext('defaults/allow_active') == 'auth_admin'
                assert action.findtext('defaults/allow_any') == action.findtext('defaults/allow_inactive') == 'no'
    assert actual == manifest['files']
    assert info and 'pkgname = grubmgr\n' in info and 'arch = x86_64\n' in info and 'depend = polkit\n' in info
    assert 'pkgver = '+manifest['version']+'\n' in info
    print(json.dumps({'package': archive.name, 'sha256': manifest['sha256'], 'files': len(actual), 'install_scripts': False, 'checked': True}))


if __name__ == '__main__': check(sys.argv[1])
