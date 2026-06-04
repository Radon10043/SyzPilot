#!/bin/bash

# this script is used to refresh the fuchsia artifacts for syzkaller fuzzing.
# It creates symbolic links to the latest artifacts and generate a new zbi with ssh support.
#
# usage:
#   bash scripts/fuchsia/refresh.sh -f $FUCHSIA

set -euo pipefail

# required args
FUCHSIA=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -f, --fuchsia <FUCHSIA>  path to the fuchsia directory"
    echo "  args (optional):"
    echo "    -h, --help             print help message"
}

# args parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -f | --fuchsia)
        FUCHSIA=$(realpath "$2")
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

OUT_DIR=$FUCHSIA/out/x64
SYZ_DIR=$OUT_DIR/syzkaller
PB_SYSTEM_A=$OUT_DIR/obj/products/core/product_bundle.x64/product_bundle/system_a
BOOTFS_DIR=$SYZ_DIR/obj/bootfs

mkdir -p $SYZ_DIR/obj

# find the kernel object file, prefer the one with more sanitizers enabled
KERNEL_CANDS=(
    $OUT_DIR/kernel_x64.lk_debug_level_2-sancov/vmzircon.with-tests
    $OUT_DIR/kernel_x64.lk_debug_level_2-kasan-sancov/vmzircon.with-tests
    $OUT_DIR/kernel_x64.lk_debug_level_2-kasan/vmzircon.with-tests
    $OUT_DIR/kernel_x64.lk_debug_level_2/vmzircon.with-tests
)
for cand in "${KERNEL_CANDS[@]}"; do
    if [[ -e $cand ]]; then
        KERNEL_OBJ=$cand
        break
    fi
done
if [[ -z "${KERNEL_OBJ:-}" ]]; then
    echo "failed to find a vmzircon.with-tests artifact" >&2
    exit 1
fi

ln -sfn $KERNEL_OBJ $SYZ_DIR/obj/zircon.elf
ln -sfn $OUT_DIR/obj/products/core/product_bundle.x64/product_bundle/system_a/linux-x86-boot-shim.bin $SYZ_DIR/kernel

# find the zbi file, prefer the one with more debug info
ZBI_CANDS=(
    $OUT_DIR/obj/bundles/assembly/zircon_eng/kernel/kernel.eng.zbi
    $PB_SYSTEM_A/fuchsia.zbi
)
for cand in "${ZBI_CANDS[@]}"; do
    if [[ -e $cand ]]; then
        ZIRCON_ZBI=$cand
        break
    fi
done
if [[ -z "${ZIRCON_ZBI:-}" ]]; then
    echo "failed to find a zircon ZBI artifact" >&2
    exit 1
fi

rm -rf "${BOOTFS_DIR}"
mkdir -p "${BOOTFS_DIR}"
"$OUT_DIR/host_x64/zbi" -x -D $BOOTFS_DIR $PB_SYSTEM_A/fuchsia.zbi

"$FUCHSIA/prebuilt/third_party/libsparse/bin/simg2img" $PB_SYSTEM_A/fxfs.sparse.blk $SYZ_DIR/fxfs.blk

"$OUT_DIR/host_x64/zbi" \
    -o $SYZ_DIR/fuchsia-ssh.zbi \
    $ZIRCON_ZBI \
    $BOOTFS_DIR \
    --replace \
    --entry data/ssh/authorized_keys=/root/.ssh/fuchsia_authorized_keys \
    --entry bin/syz-executor="$OUT_DIR/syz-executor" \
    --entry lib/libfdio.so="$OUT_DIR/x64-novariant-shared/libfdio.so" \
    --entry lib/libdriver.so="$OUT_DIR/x64-novariant-shared/libdriver.so" \
    --entry lib/libtrace-engine.so="$OUT_DIR/x64-novariant-shared/libtrace-engine.so" \
    --entry lib/libzircon.so="$OUT_DIR/x64-novariant/gen/zircon/public/sysroot/lib/libzircon.so" \
    --entry lib/libc.so="$OUT_DIR/x64-novariant/gen/zircon/public/sysroot/lib/libc.so"

file -L $SYZ_DIR/obj/zircon.elf $SYZ_DIR/kernel $SYZ_DIR/fxfs.blk
