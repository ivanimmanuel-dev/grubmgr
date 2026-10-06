#!/usr/bin/env python3
"""Exercise theme activation and rollback inside the harness guest."""
import hashlib,json,pathlib,subprocess,sys,time

assert pathlib.Path('/sys/class/dmi/id/product_name').read_text().strip() == 'grubmgr-disposable-v1'
base=pathlib.Path('/home/tester/grubmgr-test/evidence');base.mkdir(exist_ok=True)

def config_bytes():
    # Some distros protect grub.cfg with mode 0600. Preserve that protection.
    return subprocess.check_output(['sudo','-n','cat','/boot/grub/grub.cfg'])
def run(*args):
    process=subprocess.run(['grubmgr','--json',*args],capture_output=True,text=True)
    with (base/'commands.jsonl').open('a') as out:
        out.write(json.dumps({'time':time.time(),'argv':process.args,'exit':process.returncode,'stdout':process.stdout,'stderr':process.stderr})+'\n')
    if process.returncode: raise RuntimeError(process.stderr)
    return json.loads(process.stdout)
def apply(action,target,variant=None):
    args=['plan',action,target]
    if variant:args+=['--variant',variant]
    plan=run(*args);(base/(action+'-plan.json')).write_text(json.dumps(plan,indent=2))
    result=run('apply',plan['plan_id']);assert result['phase']=='committed';return result
def record(name):
    data={'boot_id':pathlib.Path('/proc/sys/kernel/random/boot_id').read_text().strip(),
          'grubmgr':run('version'),
          'kernel':subprocess.check_output(['uname','-r'],text=True).strip(),
          'status':run('status'),
          'defaults':pathlib.Path('/etc/default/grub').read_text(),
          'config_sha256':hashlib.sha256(config_bytes()).hexdigest()}
    (base/(name+'.json')).write_text(json.dumps(data,indent=2));print(json.dumps(data,indent=2),flush=True)

stage=sys.argv[1]
if stage=='activate':
    run('doctor');run('fetch','cyberpunk-demo');run('validate','cyberpunk-demo')
    before=pathlib.Path('/etc/default/grub').read_bytes();cfg=config_bytes()
    apply('install','grubmgr/cyberpunk-demo')
    assert pathlib.Path('/etc/default/grub').read_bytes()==before
    assert config_bytes()==cfg
    apply('switch','grubmgr/cyberpunk-demo')
    after=pathlib.Path('/etc/default/grub').read_bytes()
    without_theme=lambda b:b''.join(line for line in b.splitlines(keepends=True) if not line.startswith(b'GRUB_THEME='))
    assert without_theme(after)==without_theme(before)
    record('activated')
elif stage=='after-activation-boot':
    record('activation-reboot');assert json.loads((base/'activated.json').read_text())['boot_id']!=json.loads((base/'activation-reboot.json').read_text())['boot_id']
    switched=apply('switch','grubmgr/cyberpunk-demo','hd')
    (base/'rollback-target.txt').write_text(switched['transaction'])
    record('hd-activated')
elif stage=='rollback':
    kernel_files=sorted(p.name for p in pathlib.Path('/boot').glob('vmlinuz-*'))
    assert len(kernel_files)>=2, kernel_files
    apply('rollback',(base/'rollback-target.txt').read_text())
    cfg=config_bytes().decode()
    for kernel in kernel_files:assert kernel in cfg,kernel
    assert '/hd/theme.txt' not in pathlib.Path('/etc/default/grub').read_text()
    (base/'rollback-kernels.json').write_text(json.dumps(kernel_files));record('rolled-back')
elif stage=='after-rollback-boot':
    record('rollback-reboot');assert json.loads((base/'rolled-back.json').read_text())['boot_id']!=json.loads((base/'rollback-reboot.json').read_text())['boot_id']
elif stage=='community-activate':
    assert not pathlib.Path('/etc/grubmgr/vm-test').exists()
    demo=next(p for p in run('list') if p['manifest']['id']=='grubmgr/cyberpunk-demo')
    source=pathlib.Path.home()/'.local/share/grubmgr/packages'/demo['manifest']['revision']/'content'
    directory=pathlib.Path('/home/tester/grubmgr-test/community-theme');directory.mkdir()
    for name in ['theme.txt','LICENSE']:(directory/name).write_bytes((source/name).read_bytes())
    run('--yes','install',str(directory))
    imported=next(p for p in run('list') if p['manifest']['source']['url']==str(directory))
    assert not imported['manifest']['reviewed']
    result=run('--yes','switch',imported['manifest']['id'])
    (base/'community-rollback-target.txt').write_text(result['transaction'])
    (base/'community-revision.txt').write_text(imported['manifest']['revision'])
    record('community-activated')
elif stage=='after-community-boot':
    record('community-reboot')
    assert json.loads((base/'community-activated.json').read_text())['boot_id']!=json.loads((base/'community-reboot.json').read_text())['boot_id']
    assert (base/'community-revision.txt').read_text() in pathlib.Path('/etc/default/grub').read_text()
elif stage=='community-rollback':
    run('--yes','rollback',(base/'community-rollback-target.txt').read_text())
    record('community-rolled-back')
elif stage=='after-community-rollback-boot':
    record('community-rollback-reboot')
    assert json.loads((base/'community-rolled-back.json').read_text())['boot_id']!=json.loads((base/'community-rollback-reboot.json').read_text())['boot_id']
    assert '/grubmgr/cyberpunk-demo/' in pathlib.Path('/etc/default/grub').read_text()
elif stage=='starfield':
    run('info','starfield');run('fetch','starfield');run('validate','starfield')
    preview=run('preview','starfield')
    assert preview['kind']=='grub-qemu'
    (base/'starfield-preview.png').write_bytes(pathlib.Path(preview['image']).read_bytes())
    (base/'preview.log').write_bytes(pathlib.Path(preview['image']).with_name('preview.log').read_bytes())
    apply('install','debian/starfield');apply('switch','debian/starfield')
    record('starfield-activated')
elif stage=='after-starfield-boot':
    record('starfield-reboot')
    assert json.loads((base/'starfield-activated.json').read_text())['boot_id']!=json.loads((base/'starfield-reboot.json').read_text())['boot_id']
    assert '/debian/starfield/' in pathlib.Path('/etc/default/grub').read_text()
    versions=subprocess.check_output(['dpkg-query','-W','grub-common','grub2-common','grub-theme-starfield','qemu-system-x86','ovmf','bubblewrap','xorriso','mtools','pkexec','polkitd'],text=True)
    (base/'package-versions.txt').write_text(versions)
elif stage=='after-package-reinstall':
    record('package-reinstalled')
    packages=run('status')
    assert any(p['manifest']['id']=='debian/starfield' and p['active'] for p in packages)
elif stage=='matrix-finish':
    record('package-reinstalled')
    (base/'os-release.txt').write_bytes(pathlib.Path('/etc/os-release').read_bytes())
    packages=run('status')
    assert any(p['manifest']['id']=='grubmgr/cyberpunk-demo' and p['active'] for p in packages)
    try: versions=subprocess.check_output(['dpkg-query','-W'],text=True)
    except FileNotFoundError: versions=subprocess.check_output(['pacman','-Q'],text=True)
    (base/'package-versions.txt').write_text(versions)
elif stage=='matrix-preview':
    result=run('preview','cyberpunk-demo')
    assert result['kind']=='grub-qemu'
    (base/'theme-preview.png').write_bytes(pathlib.Path(result['image']).read_bytes())
else:raise SystemExit('unknown test stage')
