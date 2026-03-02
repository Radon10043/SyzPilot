#!/bin/bash

# this script is used to update or clone kernel sources to align with the latest source tree
#
# usage:
#   ./scripts/update-kernel.sh <KERNEL_DIR>

set -euo pipefail

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

if [ $# -ne 1 ]; then
    echo "Usage: $0 <KERNEL_DIR>"
    exit 1
fi

KERNDIR=$1

mkdir -p $KERNDIR/linux && cd $KERNDIR/linux
linux

mkdir -p $KERNDIR/freebsd && cd $KERNDIR/freebsd
freebsd

mkdir -p $KERNDIR/openbsd && cd $KERNDIR/openbsd
openbsd

mkdir -p $KERNDIR/netbsd && cd $KERNDIR/netbsd
netbsd
