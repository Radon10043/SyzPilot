#!/bin/bash

# qemu command used by syzkaller, for debugging
#
# usage:
#   IMAGE=/path/to/image-under-test.qcow2 $SYZPILOT/scripts/freebsd/qemu.sh

qemu-system-x86_64 \
    -m 16384 \
    -smp 4 \
    -chardev socket,id=SOCKSYZ,server=on,wait=off,host=localhost,port=46821 \
    -mon chardev=SOCKSYZ,mode=control \
    -display none \
    -serial stdio \
    -no-reboot \
    -device virtio-rng-pci \
    -enable-kvm \
    -device e1000,netdev=net0 \
    -netdev user,id=net0,restrict=on,hostfwd=tcp:127.0.0.1:37333-:22 \
    -hda $IMAGE \
    -snapshot