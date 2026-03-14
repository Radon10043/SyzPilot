#!/bin/bash

ARCH=amd64
SYZ_EXTRACT_BIN=$(realpath $(dirname $0)/../../bin/syz-extract)

$SYZ_EXTRACT_BIN \
    -build \
    -sourcedir=$KERNEL \
    -os=netbsd \
    -arch=$ARCH \
    $@