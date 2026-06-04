#!/bin/bash

# this script refines fuchsia compilation artifacts and duplicate a minimization one
# for supporting spec validation.
#
# usage:
#   bash scripts/fuchsia/refine.sh -f $FUCHSIA -o $MINI_FUCHSIA
#
# $MINI_FUCHSIA can be used for -kernel of bin/generator

set -euo pipefail

# required args
FUCHSIA=
OUTDIR=

# optional args
ARCH=$(go env GOARCH)

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -f, --fuchsia <FUCHSIA>   path to the fuchsia directory"
    echo "    -o, --outdir  <OUTDIR>    path to the output directory"
    echo "  args (optional):"
    echo "    -a, --arch    <ARCH>      target arch (default: $(go env GOARCH))"
    echo "    -h, --help                print help message"
}

# args parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -f | --fuchsia)
        FUCHSIA=$(realpath "$2")
        shift 2
        ;;
    -o | --outdir)
        OUTDIR=$(realpath "$2")
        shift 2
        ;;
    -a | --arch)
        ARCH=$2
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
if [[ -z "$FUCHSIA" ]]; then
    echo "Error: Missing required arguments."
    print_help
    exit 1
fi
if [[ ! -d "$FUCHSIA" ]]; then
    echo "Error: Directory '$FUCHSIA' does not exist."
    exit 1
fi
if [[ -z "$OUTDIR" ]]; then
    echo "Error: Missing required arguments."
    print_help
    exit 1
fi

if [[ $ARCH == "amd64" ]]; then
    ARCH="x64"
fi

# duplication
duplicate() {
    src=$1
    dst=$OUTDIR/"${src#$FUCHSIA}"
    mkdir -p $(dirname $dst) && cp -r $src $(dirname $dst)/
}
mkdir -p $OUTDIR
duplicate $FUCHSIA/zircon
duplicate $FUCHSIA/out/$ARCH/gen/zircon
duplicate $FUCHSIA/out/$ARCH/fidling
duplicate $FUCHSIA/out/$ARCH/kernel_$ARCH.lk_debug_level_0/vmzircon
duplicate $FUCHSIA/prebuilt/third_party/clang
duplicate $FUCHSIA/third_party/llvm-libc
echo "done, size of $OUTDIR: $(du -sh $OUTDIR | cut -f1)"
