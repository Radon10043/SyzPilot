#!/bin/bash
qemu-system-x86_64 \
    -m 2048 \
    -smp 2 \
    -chardev socket,id=SOCKSYZ,server=on,wait=off,host=localhost,port=26376 \
    -mon chardev=SOCKSYZ,mode=control \
    -display none \
    -serial stdio \
    -no-reboot \
    -device virtio-rng-pci \
    -enable-kvm \
    -cpu host,migratable=off \
    -drive file=$IMAGE,format=raw,if=none,id=rootdisk \
    -device nvme,drive=rootdisk,serial=debian-root \
    -kernel $GKI/dist/bzImage \
    -device e1000,netdev=net0 \
    -netdev user,id=net0,restrict=on,hostfwd=tcp:127.0.0.1:2637-:22 \
    -append "root=/dev/nvme0n1 rootwait console=ttyS0 net.ifnames=0" \
    -snapshot
