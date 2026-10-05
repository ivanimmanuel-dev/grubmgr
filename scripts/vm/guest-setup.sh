#!/bin/sh
set -eu
test "$(id -u)" = 0
test "$(cat /sys/class/dmi/id/product_name)" = grubmgr-disposable-v1
test -d /sys/firmware/efi
. /etc/os-release
test "$ID:$VERSION_ID" = debian:13

# Run from the test upload directory, inside the disposable VM only.
install -d -m 0755 /etc/grubmgr /usr/libexec
install -d -m 0700 /var/lib/grubmgr
printf 'grubmgr disposable VM v1\n' > /etc/grubmgr/vm-test
chmod 0644 /etc/grubmgr/vm-test
install -m 0755 grubmgr-phase2 /usr/local/bin/grubmgr
install -m 0755 grubmgr-helper-phase2 /usr/libexec/grubmgr-helper
install -m 0755 grubmgr-preview-qemu /usr/libexec/grubmgr-preview-qemu
install -m 0644 io.github.ivanimmanuel.grubmgr.policy /usr/share/polkit-1/actions/io.github.ivanimmanuel.grubmgr.policy

# Test-only authorization. This rule is never part of a normal installation.
cat > /etc/polkit-1/rules.d/49-grubmgr-vm.rules <<'RULE'
polkit.addRule(function(action, subject) {
    if (action.id === "io.github.ivanimmanuel.grubmgr.manage" && subject.user === "tester") {
        return polkit.Result.YES;
    }
});
RULE
chmod 0644 /etc/polkit-1/rules.d/49-grubmgr-vm.rules
