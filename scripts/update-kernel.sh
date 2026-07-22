#!/bin/bash

# this script is used to update or clone kernel sources to align with the latest source tree
#
# usage:
#   ./scripts/update-kernel.sh -d <KERNEL_DIR>

set -euo pipefail

# required args
KERNDIR=

# optional args
KERNTYP=all

KERNEL_TYPES=(linux freebsd openbsd netbsd android gvisor)
VALID_KERNTYP=(all "${KERNEL_TYPES[@]}")

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -d, --dir     <DIR>       path to the directory to store kernel sources"
    echo "  args (optional):"
    echo "    -k, --kernel  <KERNEL>    kernel types"
    echo "                                  * options: all, linux, freebsd, openbsd, netbsd, android, gvisor"
    echo "                                  * default: all"
    echo "    -h, --help                print help message"
}

valid_kerntyp() {
    local ktype=$1
    local valid_ktype

    for valid_ktype in "${VALID_KERNTYP[@]}"; do
        if [[ "$ktype" == "$valid_ktype" ]]; then
            return 0
        fi
    done

    return 1
}

# arg parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -d | --dir)
        # check if dir exists
        if [[ $# -lt 2 || -z "$2" ]]; then
            echo "Error: Missing directory for '$1'."
            print_help
            exit 1
        fi
        if [[ ! -d "$2" ]]; then
            echo "Error: Directory '$2' does not exist."
            exit 1
        fi
        KERNDIR=$(realpath "$2")
        shift 2
        ;;
    -k | --kernel)
        if [[ $# -lt 2 || -z "$2" || "$2" == -* ]]; then
            echo "Error: Missing kernel type for '$1'."
            print_help
            exit 1
        fi
        KERNTYP=$2
        if ! valid_kerntyp "$KERNTYP"; then
            echo "Error: Invalid kernel type '$KERNTYP'."
            print_help
            exit 1
        fi
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

if [[ -z "$KERNDIR" ]]; then
    print_help
    exit 1
fi

linux() {
    echo "updating linux/mainline ..."
    if [ ! -d "mainline" ]; then
        git clone https://github.com/torvalds/linux mainline
    else
        git -C mainline pull
    fi

    echo "updating linux/stable ..."
    if [ ! -d "stable" ]; then
        git clone https://github.com/gregkh/linux stable
    else
        git -C stable pull
    fi
}

freebsd() {
    echo "updating freebsd/mainline ..."
    if [ ! -d "mainline" ]; then
        git clone https://github.com/freebsd/freebsd-src mainline
    else
        git -C mainline pull
    fi
}

openbsd() {
    echo "updating openbsd/mainline ..."
    if [ ! -d "mainline" ]; then
        git clone https://github.com/openbsd/src mainline
    else
        git -C mainline pull
    fi
}

netbsd() {
    echo "updating netbsd/mainline ..."
    if [ ! -d "mainline" ]; then
        git clone https://github.com/NetBSD/src mainline
    else
        git -C mainline pull
    fi
}

android() {
    echo "updating android/mainline ..."
    if [ ! -d "mainline" ]; then
        mkdir mainline && cd mainline
        yes | repo init -u https://android.googlesource.com/kernel/manifest \
            -b common-android-mainline \
            --partial-clone --clone-filter=blob:none
        cd ..
    fi
    cd mainline && repo sync -c -j4
}

gvisor() {
    echo "updating gvisor/mainline ..."
    if [ ! -d "mainline" ]; then
        git clone https://github.com/google/gvisor mainline
    else
        git -C mainline pull
    fi
}

update_kernel() {
    local ktype=$1
    mkdir -p "$KERNDIR/$ktype"
    cd "$KERNDIR/$ktype"
    "$ktype"
}

if [[ "$KERNTYP" == "all" ]]; then
    for ktype in "${KERNEL_TYPES[@]}"; do
        update_kernel "$ktype"
    done
else
    update_kernel "$KERNTYP"
fi
