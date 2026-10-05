"""Create, operate and stop disposable QEMU test guests."""
import hashlib
import json
import os
import pathlib
import shutil
import socket
import subprocess
import tarfile
import tempfile
import time
import urllib.request
from run import bootstrap_host_key

ROOT = pathlib.Path(__file__).resolve().parents[2]
TARGETS = json.loads((ROOT/'scripts/vm/targets.json').read_text())


def image_for(name, tools, env):
    target = TARGETS[name]
    cache = ROOT/'work/vm-images'; cache.mkdir(parents=True, exist_ok=True)
    archive = cache/target['url'].rsplit('/', 1)[1]
    if not archive.exists():
        part = archive.with_suffix(archive.suffix+'.part')
        with urllib.request.urlopen(target['url'], timeout=60) as src, part.open('wb') as dest:
            shutil.copyfileobj(src, dest)
        part.rename(archive)
    with archive.open('rb') as src:
        if hashlib.file_digest(src, 'sha256').hexdigest() != target['sha256']:
            raise RuntimeError('Official image digest differs: '+name)
    if target['format'] == 'qcow2':
        return archive
    image = cache/(name+'-base.qcow2'); record = image.with_suffix('.json')
    if image.exists() and record.exists():
        data = json.loads(record.read_text())
        with image.open('rb') as src:
            if data == {'source': target['sha256'], 'sha256': hashlib.file_digest(src, 'sha256').hexdigest()}:
                return image
        raise RuntimeError('Converted image changed')
    # Native Linux temporary storage keeps the 25 GiB Kali raw disk sparse.
    with tempfile.TemporaryDirectory(prefix='grubmgr-image-') as temp:
        raw = pathlib.Path(temp)/'disk.raw'
        with tarfile.open(archive) as tar:
            members = [m for m in tar if m.isfile() and m.name == 'disk.raw']
            if len(members) != 1 or members[0].size > 40<<30:
                raise RuntimeError('Unexpected cloud archive')
            with tar.extractfile(members[0]) as src, raw.open('wb') as dest:
                while chunk := src.read(1<<20):
                    if chunk.count(0) == len(chunk): dest.seek(len(chunk), 1)
                    else: dest.write(chunk)
                dest.truncate(members[0].size)
        subprocess.run([str(tools/'usr/bin/qemu-img'), 'convert', '-f', 'raw', '-O', 'qcow2', str(raw), str(image)], check=True, env=env)
    with image.open('rb') as src:
        record.write_text(json.dumps({'source': target['sha256'], 'sha256': hashlib.file_digest(src, 'sha256').hexdigest()}))
    return image


class Machine:
    def __init__(self, name, tools, run, packages):
        if os.name != 'posix' or os.geteuid() == 0:
            raise RuntimeError('Run as an ordinary Linux user')
        self.name, self.tools, self.run = name, pathlib.Path(tools).resolve(), pathlib.Path(run).resolve()
        self.env = os.environ.copy()
        if str(self.tools) != '/': self.env['LD_LIBRARY_PATH'] = str(self.tools/'usr/lib/x86_64-linux-gnu')
        self.image = image_for(name, self.tools, self.env)
        self.packages, self.proc = packages, None

    def __enter__(self):
        self.runtime = tempfile.TemporaryDirectory(prefix='grubmgr-matrix-')
        runtime = pathlib.Path(self.runtime.name)
        # Guest package installation performs many synced writes. Native Linux
        # temporary storage avoids WSL's mounted-Windows filesystem overhead.
        self.disk_dir = pathlib.Path(tempfile.mkdtemp(prefix='grubmgr-disk-'))
        self.disk = self.disk_dir/'disk.qcow2'
        self.key, self.known = runtime/'key', runtime/'known_hosts'
        subprocess.run(['ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', str(self.key)], check=True)
        public = self.key.with_suffix('.pub').read_text().strip()
        user = {'hostname': 'grubmgr-disposable', 'users': [{'name': 'tester', 'groups': TARGETS[self.name]['groups'],
                'shell': '/bin/bash', 'sudo': 'ALL=(ALL) NOPASSWD:ALL', 'ssh_authorized_keys': [public]}],
                'ssh_pwauth': False, 'disable_root': True, 'package_update': bool(self.packages)}
        if self.packages: user['packages'] = self.packages
        if self.name != 'arch': user['apt'] = {'conf': 'APT::Install-Recommends "false";'}
        else: user['package_upgrade'] = bool(self.packages)  # No partial Arch upgrades.
        (self.run/'user-data').write_text('#cloud-config\n'+json.dumps(user))
        (self.run/'meta-data').write_text('instance-id: '+self.run.name+'\n')
        self.local('genisoimage', '-quiet', '-output', self.run/'seed.iso', '-volid', 'cidata', '-joliet', '-rock', self.run/'user-data', self.run/'meta-data')
        self.local('qemu-img', 'create', '-f', 'qcow2', '-F', 'qcow2', '-b', self.image, self.disk, '30G')
        fw = self.tools/'usr/share/OVMF'; shutil.copyfile(fw/'OVMF_VARS_4M.fd', self.run/'vars.fd')
        with socket.socket() as s: s.bind(('127.0.0.1', 0)); self.port = s.getsockname()[1]
        argv = [self.tools/'usr/bin/qemu-system-x86_64', '-machine', 'q35,smm=on', '-accel', 'tcg,thread=multi', '-smp', '2', '-m', '2048',
                '-L', self.tools/'usr/share/qemu', '-smbios', 'type=1,product=grubmgr-disposable-v1',
                '-drive', f'if=pflash,format=raw,readonly=on,file={fw}/OVMF_CODE_4M.secboot.fd',
                '-drive', f'if=pflash,format=raw,file={self.run}/vars.fd',
                '-drive', f'if=virtio,format=qcow2,file={self.disk}',
                '-drive', f'if=virtio,format=raw,readonly=on,file={self.run}/seed.iso',
                '-netdev', f'user,id=net0,hostfwd=tcp:127.0.0.1:{self.port}-:22', '-device', 'virtio-net-pci,netdev=net0,romfile=',
                '-device', f'VGA,romfile={self.tools}/usr/share/seabios/vgabios-stdvga.bin',
                '-display', 'none', '-serial', f'file:{self.run}/console.log', '-monitor', 'none']
        (self.run/'launch.json').write_text(json.dumps([str(x) for x in argv], indent=2))
        with (self.run/'qemu.log').open('wb') as log:
            self.proc = subprocess.Popen([str(x) for x in argv], env=self.env, stdout=log, stderr=log)
        try:
            self.ready()
            self.ssh('sudo cloud-init status --wait')
            return self
        except BaseException:
            self.__exit__(None, None, None)
            raise

    def local(self, name, *args):
        subprocess.run([str(self.tools/'usr/bin'/name), *map(str, args)], check=True, env=self.env)

    def options(self):
        return ['-i', str(self.key), '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=30', '-o', 'ConnectionAttempts=2', '-o', 'ServerAliveInterval=10', '-o', 'ServerAliveCountMax=3', '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile='+str(self.known)]

    def ssh(self, command, check=True, timeout=2400):
        argv = ['ssh', *self.options(), '-p', str(self.port), 'tester@127.0.0.1', command]
        try:
            result = subprocess.run(argv, capture_output=True, text=True, timeout=timeout)
        except subprocess.TimeoutExpired:
            result = subprocess.CompletedProcess(argv, 124, '', 'SSH command timed out')
        with (self.run/'host-commands.jsonl').open('a') as out:
            out.write(json.dumps({'command': command, 'exit': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr})+'\n')
        if check and result.returncode: raise RuntimeError(command+': '+result.stdout+result.stderr)
        return result

    def ready(self, previous=None):
        deadline = time.monotonic()+1800
        while time.monotonic() < deadline:
            if self.proc.poll() is not None: raise RuntimeError('QEMU exited')
            if not self.known.exists(): bootstrap_host_key(self.proc.pid, self.port, self.known)
            if self.known.exists():
                result = self.ssh('cat /proc/sys/kernel/random/boot_id', False, timeout=45)
                if result.returncode == 0 and result.stdout.strip() != previous: return
            time.sleep(5)
        raise RuntimeError('VM boot timed out')

    def reboot(self):
        before = self.ssh('cat /proc/sys/kernel/random/boot_id').stdout.strip()
        self.ssh('sudo reboot', False); self.ready(before)

    def copy(self, files):
        subprocess.run(['scp', *self.options(), '-P', str(self.port), *map(str, files), 'tester@127.0.0.1:/home/tester/grubmgr-test/'], check=True)

    def __exit__(self, *_):
        if self.proc is not None:
            if self.known.exists():
                try:
                    self.ssh('if test -d /home/tester/grubmgr-test/evidence; then if test -d /home/tester/.cache/grubmgr/previews; then cp -R /home/tester/.cache/grubmgr/previews /home/tester/grubmgr-test/evidence/preview-diagnostics; fi; sudo journalctl -b -u polkit --no-pager > /home/tester/grubmgr-test/evidence/polkit.log; fi', False)
                    subprocess.run(['scp', *self.options(), '-P', str(self.port), '-r', 'tester@127.0.0.1:/home/tester/grubmgr-test/evidence', str(self.run)], timeout=120, check=False)
                    self.ssh('sudo poweroff', False)
                except (OSError, subprocess.TimeoutExpired): pass
            try: self.proc.wait(timeout=30)
            except subprocess.TimeoutExpired:
                self.proc.terminate()
                try: self.proc.wait(timeout=10)
                except subprocess.TimeoutExpired: self.proc.kill(); self.proc.wait()
        try:
            if self.disk.exists():
                shutil.copyfile(self.disk, self.run/'disk.qcow2')
                self.disk.unlink()
            self.disk_dir.rmdir()
        except OSError as error:
            (self.run/'retained-disk.txt').write_text(str(self.disk)+'\n'+str(error)+'\n')
            print('VM disk retained at:', self.disk, flush=True)
        self.runtime.cleanup()
