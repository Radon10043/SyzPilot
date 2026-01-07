#!/bin/bash

# this scripts is written for spec debugging
#
# usage:
#   KERNEL=/path/to/kernel-for-extract $CLOUD/scripts/extract.sh $ARCH
#
# if $ARCH is not set, extract constants for all arches.
#
# example:
#   KERNEL=/vol/linux/v6.12-extract $CLOUD/scripts/extract.sh amd64

ARCH=386,amd64,arm,arm64,mips64le,ppc64le,riscv64,s390x
if [ ! -z $1 ]; then
    ARCH=$1
fi

$(dirname $0)/../bin/syz-extract -build -sourcedir=$KERNEL -os=linux -includedirs=$KERNEL/fs/xfs/libxfs -arch=$ARCH