#!/usr/bin/env python3
"""Exercise the shipped Polkit policy from a local login in the disposable VM."""
import json
import os
import pathlib
import pty
import select
import signal
import subprocess
import termios
import time

assert pathlib.Path('/sys/class/dmi/id/product_name').read_text().strip() == 'grubmgr-disposable-v1'
assert os.getuid() != 0
base = pathlib.Path('/home/tester/grubmgr-test')
# The root setup checks rule absence; Debian restricts reading that directory.
secret = (base/'auth-secret').read_bytes().strip()
session = subprocess.check_output(['loginctl', 'show-session', os.environ['XDG_SESSION_ID'],
                                  '-p', 'Active', '-p', 'Remote', '-p', 'Seat'], text=True)
(base/'evidence/auth-session.txt').write_text(session)
assert 'Active=yes' in session and 'Remote=no' in session and 'Seat=seat0' in session, session


def fingerprint():
    # Sudo reads the protected configuration; the CLI uses the shipped Polkit policy.
    data = subprocess.check_output(['sudo', '-n', 'sha256sum', '/etc/default/grub', '/boot/grub/grub.cfg'], text=True)
    return [line.split()[0] for line in data.splitlines()]


def attempt(case):
    before = fingerprint()
    pid, fd = pty.fork()
    if pid == 0:
        os.execv('/usr/bin/grubmgr', ['grubmgr', '--json', 'doctor'])
    output = bytearray()
    prompts = 0
    scanned = 0
    deadline = time.monotonic()+90
    status = None
    try:
        while time.monotonic() < deadline:
            ready, _, _ = select.select([fd], [], [], 0.2)
            if ready:
                try:
                    chunk = os.read(fd, 8192)
                except OSError:
                    chunk = b''
                output.extend(chunk)
                if not chunk:
                    if status is None:
                        _, status = os.waitpid(pid, 0)
                    break
                if b'Password:' in output[scanned:]:
                    prompts += 1
                    scanned = len(output)
                    if case == 'cancel' or prompts > 1:
                        os.write(fd, b'\x03')
                    else:
                        echo_deadline = time.monotonic()+2
                        while termios.tcgetattr(fd)[3] & termios.ECHO and time.monotonic() < echo_deadline:
                            time.sleep(0.02)
                        assert not termios.tcgetattr(fd)[3] & termios.ECHO, 'Password input still echoes'
                        password = b'grubmgr-intentionally-wrong' if case == 'wrong-password' else secret
                        os.write(fd, password+b'\n')
            if status is None:
                done, status_value = os.waitpid(pid, os.WNOHANG)
                if done:
                    status = status_value
        if status is None:
            done, status_value = os.waitpid(pid, os.WNOHANG)
            if done:
                status = status_value
            else:
                os.kill(pid, signal.SIGKILL)
                _, status = os.waitpid(pid, 0)
                transcript = bytes(output).replace(secret, b'[REDACTED]').decode(errors='replace')
                (base/'evidence'/('auth-'+case+'.log')).write_text(transcript)
                raise RuntimeError('authentication test timed out: '+case)
    finally:
        os.close(fd)
    transcript = bytes(output).replace(secret, b'[REDACTED]').decode(errors='replace')
    code = os.waitstatus_to_exitcode(status)
    (base/'evidence'/('auth-'+case+'.log')).write_text(transcript)
    assert prompts >= 1, transcript
    assert 'io.github.ivanimmanuel.grubmgr.manage' in transcript, transcript
    if case == 'accepted':
        assert code == 0 and 'SUPPORTED' in transcript, transcript
    else:
        assert code != 0 and 'SUPPORTED' not in transcript, transcript
    assert fingerprint() == before
    return {'case': case, 'exit': code, 'password_prompts': prompts, 'boot_files_unchanged': True}


results = [attempt(case) for case in ['cancel', 'wrong-password', 'accepted']]
(base/'evidence/authentication.json').write_text(json.dumps(results, indent=2)+'\n')
print('Production Polkit password checks passed', flush=True)
