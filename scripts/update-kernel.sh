#!/bin/bash

# this script is used to update or clone kernel sources to align with the latest source tree
#
# usage:
#   ./scripts/update-kernel.sh -d <KERNEL_DIR>

set -euo pipefail

# required args
KERNDIR=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -d, --dir <DIR>        path to the directory to store kernel sources"
    echo "  args (optional):"
    echo "    -h, --help             print help message"
}

# arg parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -d | --dir)
        # check if dir exists
        if [[ -z "$2" ]] || [[ ! -d "$2" ]]; then
            echo "Error: Directory '$2' does not exist."
            exit 1
        fi
        KERNDIR=$(realpath "$2")
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

mkdir -p $KERNDIR/linux && cd $KERNDIR/linux
linux

mkdir -p $KERNDIR/freebsd && cd $KERNDIR/freebsd
freebsd

mkdir -p $KERNDIR/openbsd && cd $KERNDIR/openbsd
openbsd

mkdir -p $KERNDIR/netbsd && cd $KERNDIR/netbsd
netbsd
