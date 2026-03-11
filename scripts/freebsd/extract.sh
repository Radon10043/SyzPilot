#!/bin/sh

ARCH=386,amd64,arm64,riscv64
SYZ_EXTRACT_BIN=$(realpath $(dirname $0)/../../bin/syz-extract)

$SYZ_EXTRACT_BIN \
    -build \
    -sourcedir=$KERNEL \
    -os=freebsd \
    -includedirs=$KERNEL/tools/tools/vhba \
    -includedirs=$KERNEL/sys/amd64/compile/GENERIC \
    -arch=$ARCH \
    $@