#!/bin/bash

# qemu command used by syzkaller, for debugging
#
# usage:
#   IMAGE=/path/to/image-under-test.qcow2 $SYZPILOT/scripts/netbsd/qemu.sh

qemu-system-x86_64 \
    -m 2048 \
    -smp 2 \
    -chardev socket,id=SOCKSYZ,server=on,wait=off,host=localhost,port=37643 \
    -mon chardev=SOCKSYZ,mode=control \
    -display none \
    -serial stdio \
    -no-reboot \
    -name VM-0 \
    -device virtio-rng-pci \
    -enable-kvm \
    -device e1000,netdev=net0 \
    -netdev user,id=net0,restrict=on,hostfwd=tcp:127.0.0.1:63822-:22 \
    -hda $IMAGE \
    -snapshot