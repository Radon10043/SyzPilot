#!/bin/bash

# qemu command used by syzkaller, just for debugging

qemu-system-x86_64 \
    -m 2048 \
    -smp 2 \
    -chardev socket,id=SOCKSYZ,server=on,wait=off,host=localhost,port=11045 \
    -mon chardev=SOCKSYZ,mode=control \
    -display none \
    -serial stdio \
    -no-reboot \
    -name VM-0 \
    -device virtio-rng-pci \
    -enable-kvm \
    -cpu host,migratable=off \
    -device virtio-net-pci,netdev=net0 \
    -netdev user,id=net0,restrict=on,hostfwd=tcp:127.0.0.1:56736-:22 \
    -hda $IMAGE \
    -snapshot