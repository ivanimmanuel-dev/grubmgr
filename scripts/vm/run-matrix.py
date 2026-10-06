#!/usr/bin/env python3
"""Verify a frozen package in one fresh Arch, Kali or Ubuntu QEMU guest."""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import time
from machine import Machine, ROOT, TARGETS

PACKAGES = {
    'ubuntu': ['pkexec', 'polkitd', 'grub-efi-amd64-bin', 'grub2-common', 'xorriso', 'mtools', 'qemu-system-x86', 'ovmf', 'bubblewrap', 'python3-venv'],
    'kali': ['pkexec', 'polkitd', 'grub-efi-amd64-bin', 'grub2-common', 'xorriso', 'mtools', 'qemu-system-x86', 'ovmf', 'bubblewrap', 'python3-venv'],
    'arch': ['polkit', 'python', 'python-pip', 'qemu-system-x86', 'edk2-ovmf', 'libisoburn', 'mtools', 'bubblewrap'],
}
KERNEL = {
    'ubuntu': 'sudo apt-get install -y --no-install-recommends linux-image-virtual',
    'kali': 'sudo apt-get install -y --no-install-recommends linux-image-amd64',
    'arch': 'sudo pacman -S --needed --noconfirm linux-lts',
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('target', choices=TARGETS)
    parser.add_argument('--package', required=True, type=pathlib.Path)
    parser.add_argument('--tools-root', default='/', type=pathlib.Path)
    parser.add_argument('--go', default='go')
    parser.add_argument('--accel', choices=['tcg', 'kvm'], default='tcg')
    args = parser.parse_args()
    if os.name != 'posix' or os.geteuid() == 0: raise SystemExit('Run as an ordinary Linux user')
    runs = ROOT/'work/vm-runs'; runs.mkdir(parents=True, exist_ok=True)
    run = pathlib.Path(tempfile.mkdtemp(prefix=args.target+'-matrix-', dir=runs))
    print('Evidence directory:', run, flush=True)
    package_name = 'grubmgr.pkg.tar.xz' if args.target == 'arch' else 'grubmgr.deb'
    shutil.copyfile(args.package, run/package_name)
    shutil.copyfile(args.package.with_suffix('.manifest.json'), run/'grubmgr.manifest.json')
    manifest = json.loads((run/'grubmgr.manifest.json').read_text())
    assert hashlib.sha256((run/package_name).read_bytes()).hexdigest() == manifest['sha256']
    env = os.environ.copy(); env['CGO_ENABLED'] = '0'
    subprocess.run([args.go, 'test', '-c', '-tags', 'grubmgr_vmtest', '-o', str(run/'grubmgr-vm-tests'), './internal/debian'], check=True, cwd=ROOT, env=env)
    scripts = ['guest-setup.sh', 'guest-flow.py', 'guest-package.py', 'auth-check.py', 'guest-matrix-prepare.py', 'guest-preview-policy.py', 'machine.py', 'run-matrix.py', 'targets.json']
    for name in scripts: shutil.copyfile(ROOT/'scripts/vm'/name, run/name)
    frozen = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in run.iterdir() if p.is_file()}
    (run/'inputs.json').write_text(json.dumps(frozen, indent=2)+'\n')
    with Machine(args.target, args.tools_root, run, PACKAGES[args.target], args.accel) as vm:
        if args.target == 'arch':
            # The full update replaces the running kernel and installs Polkit's
            # statically enabled socket. Start from a normal boot of that state.
            print('VM stage: boot updated Arch system', flush=True)
            vm.reboot()
            vm.ssh('systemctl is-active polkit-agent-helper.socket')
        vm.ssh('test "$(cat /sys/class/dmi/id/product_name)" = grubmgr-disposable-v1 && mkdir -p /home/tester/grubmgr-test/evidence')
        vm.copy([run/p for p in [package_name, 'grubmgr.manifest.json', 'grubmgr-vm-tests', *scripts]])
        print('VM stage: graphical fixture preparation', flush=True)
        vm.ssh('sudo python3 /home/tester/grubmgr-test/guest-matrix-prepare.py')
        vm.ssh('sudo python3 /home/tester/grubmgr-test/guest-package.py install')
        vm.ssh('cd /home/tester/grubmgr-test && sudo sh guest-setup.sh prepare')
        print('VM stage: production-policy authentication', flush=True)
        deadline = time.monotonic()+360
        while time.monotonic() < deadline:
            result = vm.ssh('cat /home/tester/grubmgr-test/evidence/auth-exit', False)
            if result.returncode == 0:
                if result.stdout.strip() != '0': raise RuntimeError('Production authentication failed; see auth-run.log')
                break
            time.sleep(3)
        else: raise RuntimeError('Authentication console timed out')
        vm.ssh('cd /home/tester/grubmgr-test && sudo sh guest-setup.sh automation')
        def flow(stage):
            print('VM stage:', stage, flush=True)
            vm.ssh('python3 /home/tester/grubmgr-test/guest-flow.py '+stage)
        flow('activate'); vm.reboot(); flow('after-activation-boot')
        print('VM stage: second kernel installation', flush=True)
        vm.ssh(KERNEL[args.target]+' > /home/tester/grubmgr-test/evidence/kernel-install.log 2>&1')
        flow('rollback'); vm.reboot(); flow('after-rollback-boot')
        flow('community-activate'); vm.reboot(); flow('after-community-boot')
        flow('community-rollback'); vm.reboot(); flow('after-community-rollback-boot')
        print('VM stage: transaction failures and locks', flush=True)
        vm.ssh('sudo /home/tester/grubmgr-test/grubmgr-vm-tests -test.v -test.timeout=45m > /home/tester/grubmgr-test/evidence/failures.log 2>&1')
        vm.ssh('sudo python3 -m venv /usr/local/lib/grub2-theme-preview && sudo /usr/local/lib/grub2-theme-preview/bin/pip install grub2-theme-preview==2.10.0 > /home/tester/grubmgr-test/evidence/preview-install.log 2>&1 && sudo ln -s /usr/local/lib/grub2-theme-preview/bin/grub2-theme-preview /usr/local/bin/grub2-theme-preview')
        if args.target == 'ubuntu':
            vm.ssh('sudo python3 /home/tester/grubmgr-test/guest-preview-policy.py')
        flow('matrix-preview')
        print('VM stage: package lifecycle', flush=True)
        vm.ssh('sudo python3 /home/tester/grubmgr-test/guest-package.py lifecycle')
        flow('matrix-finish')
        for name, digest in frozen.items():
            if hashlib.sha256((run/name).read_bytes()).hexdigest() != digest: raise RuntimeError('Frozen input changed: '+name)
        (run/'PASS').write_text('Frozen package: authentication, activation/reboot, later rollback/reboot, failures, preview, package lifecycle passed.\n')
    print('VM tests passed:', run, flush=True)


if __name__ == '__main__': main()
