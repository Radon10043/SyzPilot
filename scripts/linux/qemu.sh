#!/bin/bash
qemu-system-x86_64 \
    -m 2048 \
    -smp 2 \
    -chardev socket,id=SOCKSYZ,server=on,wait=off,host=localhost,port=27617 \
    -mon chardev=SOCKSYZ,mode=control \
    -display none \
    -serial stdio \
    -no-reboot \
    -device virtio-rng-pci \
    -enable-kvm \
    -cpu host,migratable=off \
    -device e1000,netdev=net0 \
    -netdev user,id=net0,restrict=on,hostfwd=tcp:127.0.0.1:2324-:22 \
    -hda $IMAGE/bullseye.img \
    -snapshot \
    -kernel $KERNEL/arch/x86/boot/bzImage \
    -append "root=/dev/sda console=ttyS0" 2>&1 | tee vm.log