#!/bin/bash

# this script is used for building netbsd toolchain, system root, and kernel on
# linux host. compile_commands.json is also generated under netbsd source path.
#
# usage:
#   ./scripts/netbsd/build.sh <NETBSD_SRC_PATH>

set -euo pipefail

CLOUD=$(realpath $(dirname $0)/../..)
NETBSD_SRC=$(realpath $1)

cd $NETBSD_SRC
cp $CLOUD/configs/kernel/netbsd.config sys/arch/amd64/conf/CLOUD
./build.sh -j16 -m amd64 -c clang -U -T ../tools tools
./build.sh -j16 -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j16 -m amd64 -c clang -U -T ../tools -N 4 kernel=CLOUD | tee build.log
compiledb --parse build.log
