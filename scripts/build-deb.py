#!/usr/bin/env python3
"""Build the Linux amd64 Debian package as an ordinary user."""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]


def source_epoch(env):
    if 'SOURCE_DATE_EPOCH' in env:
        value = env['SOURCE_DATE_EPOCH']
        if not re.fullmatch(r'[0-9]+', value):
            raise SystemExit('SOURCE_DATE_EPOCH must be a nonnegative integer.')
        return int(value)
    if (ROOT/'.git').exists():
        return int(subprocess.check_output(
            ['git', 'show', '-s', '--format=%ct', 'HEAD'], cwd=ROOT, text=True).strip())
    return max(0, int((ROOT/'internal/cli/cli.go').stat().st_mtime))


def build(go, output):
    if os.name != 'posix' or os.geteuid() == 0:
        raise SystemExit('Build as an ordinary Linux user.')
    version = re.search(r'const Version = "([0-9.]+)(-rc\.\d+)?"',
                        (ROOT/'internal/cli/cli.go').read_text())
    if not version:
        raise SystemExit('Unrecognized CLI version')
    upstream = version.group(0).split('"')[1]
    deb_version = upstream.replace('-rc.', '~rc.') + '-1'
    output = pathlib.Path(output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    # GitHub rewrites '~' in asset names; the control record retains Debian ordering.
    filename_version = deb_version.replace('~rc.', 'rc')
    archive = output/f'grubmgr_{filename_version}_amd64.deb'
    env = os.environ.copy()
    env.update(CGO_ENABLED='0', GOOS='linux', GOARCH='amd64')
    env['SOURCE_DATE_EPOCH'] = str(source_epoch(env))
    with tempfile.TemporaryDirectory(prefix='grubmgr-deb-') as tmp:
        stage = pathlib.Path(tmp)/'package'
        stage.mkdir(mode=0o755)

        def copy(source, destination, mode=0o644):
            dest = stage/destination
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, dest)
            dest.chmod(mode)

        for command, destination in [('grubmgr', 'usr/bin/grubmgr'),
                                     ('grubmgr-helper', 'usr/libexec/grubmgr-helper'),
                                     ('grubmgr-preview-qemu', 'usr/libexec/grubmgr-preview-qemu')]:
            dest = stage/destination
            dest.parent.mkdir(parents=True, exist_ok=True)
            subprocess.run([go, 'build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w',
                            '-o', str(dest), './cmd/'+command], cwd=ROOT, env=env, check=True)
            dest.chmod(0o755)
        copy(ROOT/'packaging/io.github.ivanimmanuel.grubmgr.policy',
             'usr/share/polkit-1/actions/io.github.ivanimmanuel.grubmgr.policy')
        # Install the optional profile as data for administrator setup.
        copy(ROOT/'packaging/apparmor/usr.bin.grubmgr',
             'usr/share/grubmgr/apparmor/usr.bin.grubmgr')
        for name in ['LICENSE', 'README.md', 'THIRD_PARTY_NOTICES.md']:
            copy(ROOT/name, 'usr/share/doc/grubmgr/'+('copyright' if name == 'LICENSE' else name))
        for name in ['installation.md', 'usage.md', 'package-format.md']:
            copy(ROOT/'docs'/name, 'usr/share/doc/grubmgr/docs/'+name)
        copy(ROOT/'examples/synthetic-recipe.json',
             'usr/share/doc/grubmgr/examples/synthetic-recipe.json')
        for source in (stage/'usr/share/doc/grubmgr').rglob('*.md'):
            source.write_text(source.read_text(encoding='utf-8').replace('](LICENSE)', '](copyright)'),
                              encoding='utf-8')
        for source in sorted((ROOT/'third_party').rglob('*')):
            if source.is_file():
                copy(source, 'usr/share/doc/grubmgr/'+source.relative_to(ROOT).as_posix())
        # Dpkg retains this directory on removal/purge if it contains receipts.
        (stage/'var/lib/grubmgr').mkdir(parents=True)
        for directory in stage.rglob('*'):
            if directory.is_dir():
                directory.chmod(0o700 if directory == stage/'var/lib/grubmgr' else 0o755)
        files = sorted(p for p in stage.rglob('*') if p.is_file())
        control = stage/'DEBIAN'
        control.mkdir(mode=0o755)
        size = sum((p.stat().st_size+1023)//1024 for p in files)
        (control/'control').write_text(f'''Package: grubmgr
Version: {deb_version}
Section: admin
Priority: optional
Architecture: amd64
Maintainer: Ivan Immanuel <255008001+ivanimmanuel-dev@users.noreply.github.com>
Installed-Size: {size}
Depends: pkexec, polkitd
Suggests: grub2-common, grub-theme-starfield, bubblewrap, qemu-system-x86, ovmf, xorriso, mtools
Homepage: https://github.com/ivanimmanuel-dev/grubmgr
Description: GRUB theme package manager
 Import, validate, preview, switch and roll back reviewed GRUB themes.
 Activation supports specific disposable Debian, Ubuntu, Kali and Arch VMs.
''')
        (control/'md5sums').write_text(''.join(
            hashlib.md5(p.read_bytes(), usedforsecurity=False).hexdigest()+'  '+p.relative_to(stage).as_posix()+'\n'
            for p in files))
        manifest = {p.relative_to(stage).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
        for file in stage.rglob('*'):
            os.utime(file, (int(env['SOURCE_DATE_EPOCH']),)*2)
        subprocess.run(['dpkg-deb', '--root-owner-group', '-Zxz', '--threads-max=2',
                        '--build', str(stage), str(archive)], env=env, check=True)
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    archive.with_suffix('.deb.sha256').write_text(digest+'  '+archive.name+'\n')
    archive.with_suffix('.manifest.json').write_text(json.dumps({
        'package': archive.name, 'version': deb_version, 'upstream_version': upstream,
        'sha256': digest, 'files': manifest}, indent=2)+'\n')
    return archive


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--go', default='go')
    parser.add_argument('--output', type=pathlib.Path, default=ROOT/'outputs/debian')
    args = parser.parse_args()
    print(build(args.go, args.output))
