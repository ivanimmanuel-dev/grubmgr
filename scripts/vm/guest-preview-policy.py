#!/usr/bin/env python3
"""Enable the packaged optional preview policy in an Ubuntu test VM only."""
import json
import os
import pathlib
import subprocess

assert os.geteuid() == 0
assert pathlib.Path('/sys/class/dmi/id/product_name').read_text().strip() == 'grubmgr-disposable-v1'
assert '\nID=ubuntu\n' in '\n'+pathlib.Path('/etc/os-release').read_text()
restriction = pathlib.Path('/proc/sys/kernel/apparmor_restrict_unprivileged_userns')
assert restriction.read_text().strip() == '1'
source = pathlib.Path('/usr/share/grubmgr/apparmor/usr.bin.grubmgr')
destination = pathlib.Path('/etc/apparmor.d/usr.bin.grubmgr')
assert source.is_file() and source.stat().st_uid == 0
assert not destination.is_symlink()
if destination.exists():
    assert destination.read_bytes() == source.read_bytes(), 'Existing administrator policy differs'
else:
    subprocess.run(['install', '-o', 'root', '-g', 'root', '-m', '0644', str(source), str(destination)], check=True)
subprocess.run(['/usr/sbin/apparmor_parser', '-r', str(destination)], check=True)
assert restriction.read_text().strip() == '1'
# The permission belongs to grubmgr, not arbitrary direct Bubblewrap callers.
probe = subprocess.run(['sudo', '-u', 'tester', '/usr/bin/bwrap', '--unshare-all', '--ro-bind', '/', '/', '/usr/bin/true'], capture_output=True, text=True)
assert probe.returncode != 0 and 'Operation not permitted' in probe.stderr, probe
evidence = pathlib.Path('/home/tester/grubmgr-test/evidence/preview-policy.json')
evidence.write_text(json.dumps({'profile': str(destination), 'global_userns_restriction': 1,
                               'unrelated_bwrap_exit': probe.returncode, 'unrelated_bwrap_stderr': probe.stderr}, indent=2)+'\n')
