#!/bin/sh
set -eu
test "$(id -u)" = 0
test "$(cat /sys/class/dmi/id/product_name)" = grubmgr-disposable-v1
test -d /sys/firmware/efi
. /etc/os-release
case "$ID:${VERSION_ID:-rolling}" in
    debian:13|ubuntu:24.04|arch:rolling|kali:*) ;;
    *) echo 'Unsupported disposable guest' >&2; exit 1 ;;
esac

# Run after installing the frozen package in the guest.
test -x /usr/bin/grubmgr
test -x /usr/libexec/grubmgr-helper
if test "${1:-}" = prepare; then
test ! -e /etc/polkit-1/rules.d/49-grubmgr-vm.rules
python3 - <<'PY'
import os, pathlib, pwd, secrets, subprocess
secret = secrets.token_urlsafe(24)
subprocess.run(['chpasswd'], input='tester:'+secret+'\n', text=True, check=True)
file = pathlib.Path('/home/tester/grubmgr-test/auth-secret')
file.write_text(secret+'\n'); file.chmod(0o600)
user = pwd.getpwnam('tester'); os.chown(file, user.pw_uid, user.pw_gid)
PY
cat > /home/tester/.bash_profile <<'PROFILE'
if test "$(tty)" = /dev/tty1; then
    python3 /home/tester/grubmgr-test/auth-check.py > /home/tester/grubmgr-test/evidence/auth-run.log 2>&1
    printf '%s\n' "$?" > /home/tester/grubmgr-test/evidence/auth-exit
fi
PROFILE
chown tester:tester /home/tester/.bash_profile
install -d -m 0755 /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/grubmgr-test.conf <<'UNIT'
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin tester --noclear %I 38400 linux
UNIT
systemctl daemon-reload
systemctl restart getty@tty1.service
elif test "${1:-}" = automation; then
rm -f /home/tester/grubmgr-test/auth-secret
usermod --lock tester
rm -f /home/tester/.bash_profile /etc/systemd/system/getty@tty1.service.d/grubmgr-test.conf
systemctl daemon-reload
systemctl restart getty@tty1.service

# Authorize the tester after completing the shipped password-policy checks.
cat > /etc/polkit-1/rules.d/49-grubmgr-vm.rules <<'RULE'
polkit.addRule(function(action, subject) {
    if (action.id === "io.github.ivanimmanuel.grubmgr.manage" && subject.user === "tester") {
        return polkit.Result.YES;
    }
});
RULE
chmod 0644 /etc/polkit-1/rules.d/49-grubmgr-vm.rules
else
    echo 'Expected prepare or automation' >&2
    exit 2
fi
