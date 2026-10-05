#!/usr/bin/env python3
"""Verify Debian archive contents against the build manifest."""
import hashlib
import importlib.util
import io
import json
import pathlib
import subprocess
import sys
import tarfile
import xml.etree.ElementTree as ET

spec = importlib.util.spec_from_file_location('doc_links', pathlib.Path(__file__).with_name('check-doc-links.py'))
doc_links = importlib.util.module_from_spec(spec)
spec.loader.exec_module(doc_links)


def check(archive):
    archive = pathlib.Path(archive)
    manifest = json.loads(archive.with_suffix('.manifest.json').read_text())
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == manifest['sha256']
    assert archive.name == manifest['package']
    assert archive.with_suffix('.deb.sha256').read_text() == manifest['sha256']+'  '+archive.name+'\n'
    payload = subprocess.check_output(['dpkg-deb', '--ctrl-tarfile', str(archive)])
    with tarfile.open(fileobj=io.BytesIO(payload)) as control:
        names = {p.name.removeprefix('./') for p in control if p.isfile()}
        assert names == {'control', 'md5sums'}, names
    payload = subprocess.check_output(['dpkg-deb', '--fsys-tarfile', str(archive)])
    with tarfile.open(fileobj=io.BytesIO(payload)) as data:
        actual = {}
        documents = {}
        for member in data:
            name = member.name.removeprefix('./').rstrip('/')
            assert member.isfile() or member.isdir(), name
            assert member.uid == member.gid == 0, name
            assert member.mode & 0o6022 == 0, name
            assert name in ('', '.', 'usr', 'var', 'var/lib', 'var/lib/grubmgr') or name.startswith((
                'usr/bin', 'usr/libexec', 'usr/share')), name
            if member.isfile():
                content = data.extractfile(member).read()
                actual[name] = hashlib.sha256(content).hexdigest()
                if name.startswith('usr/share/doc/grubmgr/'):
                    documents[name.removeprefix('usr/share/doc/grubmgr/')] = content
                if name in ('usr/bin/grubmgr', 'usr/libexec/grubmgr-helper', 'usr/libexec/grubmgr-preview-qemu'):
                    assert member.mode == 0o755 and content[:5] == b'\x7fELF\x02'
                    assert content[18:20] == b'\x3e\x00'  # x86-64 ELF
                if name.endswith('.policy'):
                    policy = ET.fromstring(content)
                    action = policy.find('action')
                    assert action.attrib['id'] == 'io.github.ivanimmanuel.grubmgr.manage'
                    assert action.findtext('defaults/allow_active') == 'auth_admin'
                    assert action.findtext('defaults/allow_any') == action.findtext('defaults/allow_inactive') == 'no'
            if name == 'var/lib/grubmgr':
                assert member.mode == 0o700
        assert actual == manifest['files']
        doc_links.check(documents)
    print(json.dumps({'package': archive.name, 'sha256': manifest['sha256'],
                      'files': len(actual), 'maintainer_scripts': False, 'checked': True}))


if __name__ == '__main__':
    check(sys.argv[1])
