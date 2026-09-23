#!/bin/bash

# qemu command used for linux startup, using the same options as syzkaller
#
# usage:
#   $SYZPILOT/scripts/linux/qemu.sh -i /path/to/bullseye.img -k /path/to/linux

set -euo pipefail

# required args
IMAGE=
KERNEL=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -i, --image   <IMAGE>     path to the bulleye image"
    echo "    -k, --kernel  <KERNEL>    path to the linux kernel directory"
    echo "  args (optional):"
    echo "    -h, --help                print help message"
}

# Argument parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -i | --image)
        IMAGE=$(realpath "$2")
        shift 2
        ;;
    -k | --kernel)
        KERNEL=$(realpath "$2")
        shift 2
        ;;
    -h | --help)
        print_help
        exit 0
        ;;
    *)
        echo "unknown arg: $1"
        exit 1
        ;;
    esac
done

# validation
if [[ -z "$IMAGE" || -z "$KERNEL" ]]; then
    echo "error: missing required arguments."
    print_help
    exit 1
fi
if [[ ! -f "$IMAGE" ]]; then
    echo "error: '$IMAGE' does not exist."
    exit 1
fi
if [[ ! -d "$KERNEL" ]]; then
    echo "error: '$KERNEL' does not exist."
    exit 1
fi

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