#!/usr/bin/env python3
"""Boot tests in a newly created QEMU VM. Requires Linux, Go, QEMU and OVMF."""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[2]
IMAGE = 'debian-13-genericcloud-amd64.qcow2'
SHA512 = 'f46f0671a6e5bdec5291ab8972bae2f10e5408c2f64a74078f11efc2f06a436a9d0313ed50e0472542eeabf780e9f7c792ac0a314c6c20507fcd9fd81b468c3d'
URL = 'https://cloud.debian.org/images/cloud/trixie/latest/' + IMAGE


def bootstrap_host_key(pid, port, known):
    """Record the guest's first SSH key after checking QEMU listener ownership."""
    address = '0100007F:%04X' % port
    inodes = {row.split()[9] for row in pathlib.Path('/proc/net/tcp').read_text().splitlines()[1:]
              if row.split()[1] == address and row.split()[3] == '0A'}
    owned = set()
    try:
        for fd in pathlib.Path('/proc', str(pid), 'fd').iterdir():
            try: owned.add(os.readlink(fd))
            except FileNotFoundError: pass
    except FileNotFoundError:
        return
    if not any('socket:['+inode+']' in owned for inode in inodes):
        return
    result = subprocess.run(['ssh-keyscan','-T','5','-t','ed25519','-p',str(port),'127.0.0.1'],capture_output=True,text=True,timeout=10)
    for line in result.stdout.splitlines():
        if re.fullmatch(r'\[127\.0\.0\.1\]:'+str(port)+r' ssh-ed25519 [A-Za-z0-9+/=]+',line):
            known.write_text(line+'\n')
            known.chmod(0o600)
            return


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tools-root', type=pathlib.Path, default=pathlib.Path('/'))
    parser.add_argument('--go', default='go')
    parser.add_argument('--accel', choices=['tcg', 'kvm'], default='tcg')
    parser.add_argument('--deb', type=pathlib.Path, help='Use this frozen .deb and its adjacent .manifest.json')
    args = parser.parse_args()
    if os.name != 'posix' or os.geteuid() == 0:
        raise SystemExit('Run as an ordinary Linux user.')
    tools = args.tools_root.resolve()
    env = os.environ.copy()
    if tools != pathlib.Path('/'):
        env['LD_LIBRARY_PATH'] = str(tools / 'usr/lib/x86_64-linux-gnu')
    bins = {name: tools / 'usr/bin' / name for name in ['qemu-system-x86_64', 'qemu-img', 'genisoimage']}
    firmware = tools / 'usr/share/OVMF'
    for file in [*bins.values(), firmware/'OVMF_CODE_4M.secboot.fd', firmware/'OVMF_VARS_4M.fd']:
        if not file.is_file():
            raise SystemExit('Missing dependency: ' + str(file))
    for command in [args.go, 'ssh', 'scp', 'ssh-keygen', 'ssh-keyscan']:
        if not shutil.which(command):
            raise SystemExit('Missing executable: ' + command)
    cache = ROOT/'work/vm-images'
    cache.mkdir(parents=True, exist_ok=True)
    image = cache/IMAGE
    if not image.exists():
        partial = cache/(IMAGE+'.part')
        with urllib.request.urlopen(URL, timeout=60) as source, partial.open('wb') as dest:
            shutil.copyfileobj(source, dest)
        partial.rename(image)
    with image.open('rb') as stream:
        if hashlib.file_digest(stream, 'sha512').hexdigest() != SHA512:
            raise SystemExit('Debian image digest differs from the pin. Use the matching image or review a pin update.')
    runs = ROOT/'work/vm-runs'
    runs.mkdir(exist_ok=True)
    run = pathlib.Path(tempfile.mkdtemp(prefix='debian13-', dir=runs))
    print('Evidence directory:', run, flush=True)
    if args.deb:
        package = args.deb.resolve()
    else:
        subprocess.run(['python3', str(ROOT/'scripts/build-deb.py'), '--go', args.go,
                        '--output', str(run/'package')], cwd=ROOT, check=True)
        package, = (run/'package').glob('*.deb')
    subprocess.run(['python3', str(ROOT/'scripts/check-deb.py'), str(package)], check=True)
    shutil.copyfile(package, run/'grubmgr.deb')
    shutil.copyfile(package.with_suffix('.manifest.json'), run/'grubmgr.manifest.json')
    buildenv = os.environ.copy()
    buildenv['CGO_ENABLED'] = '0'
    subprocess.run([args.go, 'test', '-c', '-tags', 'grubmgr_vmtest', '-o',
                    str(run/'grubmgr-vm-tests'), './internal/debian'], check=True, cwd=ROOT, env=buildenv)
    for name in ['guest-setup.sh', 'guest-flow.py', 'guest-package.py', 'auth-check.py', 'run.py']:
        shutil.copyfile(ROOT/'scripts/vm'/name, run/name)
    (run/'inputs.json').write_text(json.dumps({p.name: hashlib.sha256(p.read_bytes()).hexdigest()
        for p in run.iterdir() if p.is_file()}, indent=2)+'\n')
    # Runtime secrets/sockets stay on a native Linux filesystem, including in WSL.
    with tempfile.TemporaryDirectory(prefix='grubmgr-vm-') as runtime_dir:
        runtime = pathlib.Path(runtime_dir)
        disk = runtime/'disk.qcow2'
        key = runtime/'key'
        subprocess.run(['ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', str(key)], check=True)
        public = key.with_suffix('.pub').read_text().strip()
        (run/'user-data').write_text('''#cloud-config
hostname: grubmgr-disposable
users:
  - name: tester
    groups: [sudo]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - '''+public+'''
ssh_pwauth: false
disable_root: true
package_update: true
apt:
  conf: |
    APT::Install-Recommends "false";
packages: [pkexec, polkitd, desktop-base, fonts-unifont, grub-efi-amd64-bin, grub2-common, grub-theme-starfield, xorriso, mtools, qemu-system-x86, ovmf, bubblewrap, python3-venv]
''')
        (run/'meta-data').write_text('instance-id: '+run.name+'\nlocal-hostname: grubmgr-disposable\n')
        def local(argv):
            return subprocess.run([str(x) for x in argv], check=True, env=env, cwd=ROOT)
        local([bins['genisoimage'], '-quiet', '-output', run/'seed.iso', '-volid', 'cidata', '-joliet', '-rock', run/'user-data', run/'meta-data'])
        local([bins['qemu-img'], 'create', '-f', 'qcow2', '-F', 'qcow2', '-b', image, disk, '16G'])
        shutil.copyfile(firmware/'OVMF_VARS_4M.fd', run/'vars.fd')
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            port = s.getsockname()[1]
        accelerator = 'kvm' if args.accel == 'kvm' else 'tcg,thread=multi'
        qemu = [bins['qemu-system-x86_64'], '-machine', 'q35,smm=on', '-accel', accelerator, '-smp', '2', '-m', '2048',
                '-L', tools/'usr/share/qemu', '-smbios', 'type=1,product=grubmgr-disposable-v1',
                '-drive', f'if=pflash,format=raw,readonly=on,file={firmware}/OVMF_CODE_4M.secboot.fd',
                '-drive', f'if=pflash,format=raw,file={run}/vars.fd',
                '-drive', f'if=virtio,format=qcow2,file={disk}',
                '-drive', f'if=virtio,format=raw,readonly=on,file={run}/seed.iso',
                '-netdev', f'user,id=net0,hostfwd=tcp:127.0.0.1:{port}-:22', '-device', 'virtio-net-pci,netdev=net0,romfile=',
                '-device', f'VGA,romfile={tools}/usr/share/seabios/vgabios-stdvga.bin',
                '-display', 'none', '-serial', f'file:{run}/console.log', '-monitor', 'none']
        (run/'launch.json').write_text(json.dumps([str(x) for x in qemu], indent=2))
        with (run/'qemu.log').open('wb') as log:
            proc = subprocess.Popen([str(x) for x in qemu], stdout=log, stderr=log, env=env)
        known = runtime/'known_hosts'
        opts = ['-i', str(key), '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=5', '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile='+str(known)]
        def ssh(command, check=True):
            result = subprocess.run(['ssh', *opts, '-p', str(port), 'tester@127.0.0.1', command], capture_output=True, text=True, timeout=2400)
            with (run/'host-commands.jsonl').open('a') as out:
                out.write(json.dumps({'command':command, 'exit':result.returncode, 'stdout':result.stdout, 'stderr':result.stderr})+'\n')
            if check and result.returncode:
                raise RuntimeError(command+': '+result.stderr)
            return result
        def ready(previous=None):
            until = time.monotonic()+900
            while time.monotonic()<until:
                if proc.poll() is not None:
                    raise RuntimeError('QEMU exited; see qemu.log')
                if not known.exists():
                    console = (run/'console.log').read_text(errors='replace') if (run/'console.log').exists() else ''
                    keys = re.findall(r'ssh-ed25519 [A-Za-z0-9+/=]+', console)
                    if keys:
                        known.write_text(f'[127.0.0.1]:{port} '+keys[-1]+'\n')
                    else:
                        bootstrap_host_key(proc.pid, port, known)
                if known.exists():
                    result = ssh('cat /proc/sys/kernel/random/boot_id', False)
                    if result.returncode == 0 and result.stdout.strip() != previous:
                        return result.stdout.strip()
                time.sleep(5)
            raise RuntimeError('VM boot timed out')
        def reboot():
            before = ssh('cat /proc/sys/kernel/random/boot_id').stdout.strip()
            ssh('sudo reboot', False)
            ready(before)
        def flow(stage):
            print('VM stage:', stage, flush=True)
            ssh('python3 /home/tester/grubmgr-test/guest-flow.py '+stage)
        def authentication():
            print('VM stage: production-policy authentication', flush=True)
            deadline = time.monotonic()+300
            while time.monotonic() < deadline:
                result = ssh('cat /home/tester/grubmgr-test/evidence/auth-exit', False)
                if result.returncode == 0:
                    if result.stdout.strip() != '0':
                        raise RuntimeError('Production-policy authentication failed; see evidence/auth-run.log')
                    return
                time.sleep(3)
            raise RuntimeError('Local authentication console timed out')
        try:
            ready()
            ssh('cloud-init status --wait')
            ssh('test "$(cat /sys/class/dmi/id/product_name)" = grubmgr-disposable-v1 && mkdir -p /home/tester/grubmgr-test/evidence')
            files = [run/x for x in ['grubmgr.deb', 'grubmgr.manifest.json', 'grubmgr-vm-tests',
                                    'guest-setup.sh', 'guest-flow.py', 'guest-package.py', 'auth-check.py']]
            subprocess.run(['scp',*opts,'-P',str(port),*[str(x) for x in files],'tester@127.0.0.1:/home/tester/grubmgr-test/'],check=True)
            ssh('sudo python3 /home/tester/grubmgr-test/guest-package.py install')
            ssh('cd /home/tester/grubmgr-test && sudo sh guest-setup.sh prepare')
            authentication()
            ssh('cd /home/tester/grubmgr-test && sudo sh guest-setup.sh automation')
            flow('activate');reboot();flow('after-activation-boot')
            ssh('sudo apt-get install -y --no-install-recommends linux-image-amd64 > /home/tester/grubmgr-test/evidence/kernel-install.log 2>&1')
            flow('rollback');reboot();flow('after-rollback-boot')
            flow('community-activate');reboot();flow('after-community-boot')
            flow('community-rollback');reboot();flow('after-community-rollback-boot')
            ssh('sudo /home/tester/grubmgr-test/grubmgr-vm-tests -test.v -test.timeout=30m > /home/tester/grubmgr-test/evidence/failures.log 2>&1')
            ssh('sudo python3 -m venv /usr/local/lib/grub2-theme-preview && sudo /usr/local/lib/grub2-theme-preview/bin/pip install grub2-theme-preview==2.10.0 > /home/tester/grubmgr-test/evidence/preview-install.log 2>&1 && sudo ln -s /usr/local/lib/grub2-theme-preview/bin/grub2-theme-preview /usr/local/bin/grub2-theme-preview')
            flow('starfield');reboot();flow('after-starfield-boot')
            print('VM stage: package reinstall/remove/purge', flush=True)
            ssh('sudo python3 /home/tester/grubmgr-test/guest-package.py lifecycle')
            flow('after-package-reinstall')
            for name, digest in json.loads((run/'inputs.json').read_text()).items():
                if hashlib.sha256((run/name).read_bytes()).hexdigest() != digest:
                    raise RuntimeError('Frozen input changed during run: '+name)
            (run/'PASS').write_text('All VM stages passed. See evidence and console.log.\n')
        finally:
            if known.exists():
                subprocess.run(['scp',*opts,'-P',str(port),'-r','tester@127.0.0.1:/home/tester/grubmgr-test/evidence',str(run)],check=False)
                ssh('sudo poweroff',False)
            try:
                proc.wait(timeout=30)
            except subprocess.TimeoutExpired:
                proc.terminate()
                try: proc.wait(timeout=10)
                except subprocess.TimeoutExpired: proc.kill();proc.wait()
        print('VM tests passed:',run,flush=True)


if __name__ == '__main__':
    main()
