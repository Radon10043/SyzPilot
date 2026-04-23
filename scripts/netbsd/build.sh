#!/bin/bash

# this script is used for building netbsd toolchain, system root, and kernel on
# linux host. compile_commands.json is also generated under netbsd source path.
#
# usage:
#   ./scripts/netbsd/build.sh <NETBSD_SRC_PATH>

set -euo pipefail

# required args
SOURCEDIR=
CONFIG=
JOBS=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -s,--sourcedir    <SOURCEDIR>     path to the NetBSD source directory"
    echo "    -c,--config       <CONFIG>        path to the NetBSD kernel config file"
    echo "    -j,--jobs         <JOBS>          number of jobs to run in parallel for building"
    echo "  args (optional):"
    echo "    -h, --help                        print help message"
}

while [[ $# -gt 0 ]]; do
    case $1 in
    -s | --sourcedir)
        SOURCEDIR=$(realpath $2)
        shift
        shift
        ;;
    -c | --config)
        CONFIG=$(realpath $2)
        shift
        shift
        ;;
    -j | --jobs)
        JOBS=$2
        shift
        shift
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

cd $SOURCEDIR
cp $CONFIG sys/arch/amd64/conf/CLOUD
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools tools
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -N 4 kernel=CLOUD | tee build.log
compiledb --parse build.log
