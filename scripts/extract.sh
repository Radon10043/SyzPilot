#!/bin/bash

KERNEL=  # Path to kernel source tree, e.g. /vol/linux/v6.12-extract

ARCH=386,amd64,arm,arm64,mips64le,ppc64le,riscv64,s390x
if [ ! -z $1 ]; then
    ARCH=$1
fi

$(dirname $0)/../bin/syz-extract -build -sourcedir=$KERNEL -os=linux -includedirs=$KERNEL/fs/xfs/libxfs -arch=$ARCH