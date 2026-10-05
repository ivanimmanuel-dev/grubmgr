#!/usr/bin/env python3
"""Set up a graphical GRUB test menu inside the disposable guest only."""
import hashlib
import json
import os
import pathlib
import re
import subprocess

assert os.geteuid() == 0
assert pathlib.Path('/sys/class/dmi/id/product_name').read_text().strip() == 'grubmgr-disposable-v1'
assert pathlib.Path('/sys/firmware/efi').is_dir()
base = pathlib.Path('/home/tester/grubmgr-test/evidence')
files = [pathlib.Path('/etc/default/grub')]
files += sorted(pathlib.Path('/etc/default/grub.d').glob('*.cfg'))
before = {str(p): p.read_text() for p in files}
for p in files:
    text = p.read_text()
    # Cloud images use serial/text-only menus and zero timeout. This test
    # preparation is separate from grubmgr, which edits only GRUB_THEME.
    text = re.sub(r'(?m)^(GRUB_TERMINAL|GRUB_TERMINAL_OUTPUT|GRUB_TIMEOUT|GRUB_TIMEOUT_STYLE)=.*$', r'# grubmgr VM fixture: \g<0>', text)
    p.write_text(text)
with files[0].open('a') as out:
    out.write('\nGRUB_TERMINAL_OUTPUT="gfxterm serial"\nGRUB_TIMEOUT=2\nGRUB_TIMEOUT_STYLE=menu\n')
generator = '/usr/bin/grub-mkconfig' if pathlib.Path('/etc/arch-release').exists() else '/usr/sbin/grub-mkconfig'
subprocess.run([generator, '-o', '/boot/grub/grub.cfg'], check=True)
subprocess.run(['/usr/bin/grub-script-check', '/boot/grub/grub.cfg'], check=True)
(base/'fixture-preparation.json').write_text(json.dumps({'before': before, 'after': {str(p): p.read_text() for p in files}}, indent=2)+'\n')
os.chown(base/'fixture-preparation.json', 1000, 1000)
