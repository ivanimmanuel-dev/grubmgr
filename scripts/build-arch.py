#!/usr/bin/env python3
"""Build a pacman archive from the verified Debian package payload."""
import argparse
import hashlib
import io
import json
import os
import pathlib
import subprocess
import tarfile


def build(deb, output):
    if os.name != 'posix' or os.geteuid() == 0:
        raise SystemExit('Build as an ordinary Linux user.')
    deb = pathlib.Path(deb).resolve()
    subprocess.run(['python3', str(pathlib.Path(__file__).with_name('check-deb.py')), str(deb)], check=True)
    source = json.loads(deb.with_suffix('.manifest.json').read_text())
    version = source['upstream_version'].replace('-rc.', 'rc')+'-1'
    output = pathlib.Path(output).resolve(); output.mkdir(parents=True, exist_ok=True)
    archive = output/f'grubmgr-{version}-x86_64.pkg.tar.xz'
    payload = subprocess.check_output(['dpkg-deb', '--fsys-tarfile', str(deb)])
    with tarfile.open(fileobj=io.BytesIO(payload)) as src:
        members = list(src); epoch = max(m.mtime for m in members)
        license_member = next(m for m in members if m.name.removeprefix('./') == 'usr/share/doc/grubmgr/copyright')
        license_data = src.extractfile(license_member).read()
        license_path = 'usr/share/licenses/grubmgr/LICENSE'
        files = dict(source['files']); files[license_path] = hashlib.sha256(license_data).hexdigest()
        size = sum(m.size for m in members if m.isfile()) + len(license_data)
        info = f'''pkgname = grubmgr
pkgbase = grubmgr
pkgver = {version}
pkgdesc = GRUB theme package manager
url = https://github.com/ivanimmanuel-dev/grubmgr
builddate = {int(epoch)}
packager = Ivan Immanuel
size = {size}
arch = x86_64
license = MIT
depend = polkit
optdepend = grub: GRUB configuration and theme tools
optdepend = bubblewrap: isolated preview
optdepend = qemu-system-x86: preview virtual machine
optdepend = edk2-ovmf: preview firmware
optdepend = libisoburn: preview images
optdepend = mtools: preview images
'''.encode()
        with tarfile.open(archive, 'w:xz', format=tarfile.GNU_FORMAT, preset=6) as out:
            meta = tarfile.TarInfo('.PKGINFO'); meta.mode = 0o644; meta.mtime = epoch; meta.size = len(info)
            out.addfile(meta, io.BytesIO(info))
            for member in members:
                member.name = member.name.removeprefix('./')
                if member.name in ('', '.'):
                    continue
                out.addfile(member, src.extractfile(member) if member.isfile() else None)
            for name in ['usr/share/licenses', 'usr/share/licenses/grubmgr']:
                directory = tarfile.TarInfo(name); directory.type = tarfile.DIRTYPE; directory.mode = 0o755; directory.mtime = epoch
                out.addfile(directory)
            license_info = tarfile.TarInfo(license_path); license_info.size = len(license_data); license_info.mode = 0o644; license_info.mtime = epoch
            out.addfile(license_info, io.BytesIO(license_data))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    archive.with_suffix(archive.suffix+'.sha256').write_text(digest+'  '+archive.name+'\n')
    archive.with_suffix('.manifest.json').write_text(json.dumps({
        'package': archive.name, 'version': version, 'upstream_version': source['upstream_version'],
        'sha256': digest, 'files': files, 'source_deb_sha256': source['sha256']}, indent=2)+'\n')
    return archive


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--deb', required=True, type=pathlib.Path)
    parser.add_argument('--output', default=pathlib.Path('outputs/arch'), type=pathlib.Path)
    args = parser.parse_args()
    print(build(args.deb, args.output))
