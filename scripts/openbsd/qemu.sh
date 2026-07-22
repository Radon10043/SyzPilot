#!/bin/bash

# qemu command used for openbsd startup, using the same options as syzkaller
#
# usage:
#   $CLOUD/scripts/openbsd/qemu.sh -i /path/to/openbsd.qcow2

set -euo pipefail

# required args
IMAGE=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -i, --image           <IMAGE>         path to the openbsd image"
    echo "  args (optional):"
    echo "    -h, --help                            print help message"
}

# Argument parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -i | --image)
        IMAGE=$(realpath "$2")
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
if [[ -z "$IMAGE" ]]; then
    echo "error: missing required arguments."
    print_help
    exit 1
fi
if [[ ! -f "$IMAGE" ]]; then
    echo "error: '$IMAGE' does not exist."
    exit 1
fi

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