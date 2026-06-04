#!/bin/bash

# this script scans *.fidl.json files under out/x64/fidling/gen/sdk/fidl (max depth: 2) and
# calls fidlgen_syzkaller under $FUCHSIA/out/x64/host_x64 to initially generate syscall specs
# for fuchsia.
#
# usage:
#   bash scripts/fuchsia/fidlgen.sh -f $FUCHSIA -o $OUTDIR

set -euo pipefail

# required args
FUCHSIA=
OUTDIR=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -f, --fuchsia <FUCHSIA>   path to the fuchsia directory"
    echo "    -o, --outdir  <OUTDIR>    path to the output directory"
    echo "  args (optional):"
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

# generate specs
mkdir -p $OUTDIR
FIDLGEN_SYZKALLER=$FUCHSIA/out/x64/host_x64/fidlgen_syzkaller
FIDL_JSONIR_ROOT=$FUCHSIA/out/x64/fidling/gen/sdk/fidl
FILES=$(find $FIDL_JSONIR_ROOT -maxdepth 2 -name "*.fidl.json")
NUM_SUCCESS=0
NUM_FAILED=0
for file in $FILES; do
    echo -n "generating spec for $(basename $file) ... "
    fname=$(basename $file .fidl.json).syz.txt
    if $FIDLGEN_SYZKALLER -json=$file -output-syz=$OUTDIR/$fname 2> /dev/null; then
        sed -E -i 's|include <[^>]+c/fidl\.h>|include <zircon/syscalls.h>\ninclude <zircon/types.h>|g' $OUTDIR/$fname
        ((++NUM_SUCCESS))
        echo "success"
    else
        ((++NUM_FAILED))
        echo "failed, skip"
    fi
done
echo "done, $NUM_SUCCESS success, $NUM_FAILED failed"
