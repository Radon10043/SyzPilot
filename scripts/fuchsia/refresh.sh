#!/bin/bash

# this script is used to refresh the fuchsia artifacts for syzkaller fuzzing.
# It creates symbolic links to the latest artifacts and generate a new zbi with ssh support.
#
# usage:
#   bash scripts/fuchsia/refresh.sh -f $FUCHSIA

set -e

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

cd $FUCHSIA
source scripts/fx-env.sh && fx-update-path

PRODUCT_BUNDLE="$(ffx config get product.path | tr -d '"')"
PB_SYSTEM_A=$PRODUCT_BUNDLE/system_a
FUCHSIA_ZBI="$(ffx product get-image-path "$PRODUCT_BUNDLE" --slot a --image-type zbi)"
FXFS_SPARSE_BLK="$(ffx product get-image-path "$PRODUCT_BUNDLE" --slot a --image-type fxfs.fastboot)"

OUT_DIR=$FUCHSIA/out/x64
SYZ_DIR=$OUT_DIR/syzkaller
BOOTFS_DIR=$SYZ_DIR/obj/bootfs

mkdir -p $SYZ_DIR/obj

# find the kernel object file, prefer the one with more sanitizers enabled
KERNEL_CANDS=(
    $OUT_DIR/kernel_x64-sancov/vmzircon
    $OUT_DIR/kernel_x64-kasan-sancov/vmzircon
    $OUT_DIR/kernel_x64-kasan/vmzircon
    $OUT_DIR/kernel_x64/vmzircon
)
for cand in "${KERNEL_CANDS[@]}"; do
    if [[ -e $cand ]]; then
        KERNEL_OBJ=$cand
        break
    fi
done
if [[ -z "${KERNEL_OBJ:-}" ]]; then
    echo "failed to find a vmzircon artifact" >&2
    exit 1
fi

ln -sfn $KERNEL_OBJ $SYZ_DIR/obj/zircon.elf
ln -sfn $PB_SYSTEM_A/linux-x86-boot-shim.bin $SYZ_DIR/kernel

rm -rf "${BOOTFS_DIR}"
mkdir -p "${BOOTFS_DIR}"
"$OUT_DIR/host_x64/zbi" -x -D $BOOTFS_DIR $FUCHSIA_ZBI

# generate sshkey
ffx config check-ssh-keys
AUTH_SSHKEY="$(ffx config get ssh.pub | tr -d '"')"
PRIV_SSHKEY="$(ffx config get ssh.priv | tr -d '"')"
cp $PRIV_SSHKEY $SYZ_DIR/

"$FUCHSIA/prebuilt/third_party/libsparse/bin/simg2img" $FXFS_SPARSE_BLK $SYZ_DIR/fxfs.blk

"$OUT_DIR/host_x64/zbi" \
    -o $SYZ_DIR/fuchsia-ssh.zbi \
    --replace \
    $FUCHSIA_ZBI \
    $BOOTFS_DIR \
    --entry data/ssh/authorized_keys=/root/.ssh/fuchsia_authorized_keys \
    --entry bin/syz-executor="$OUT_DIR/syz-executor" \
    --entry lib/libfdio.so="$OUT_DIR/x64-novariant-shared/libfdio.so" \
    --entry lib/libdriver.so="$OUT_DIR/x64-novariant-shared/libdriver.so" \
    --entry lib/libtrace-engine.so="$OUT_DIR/x64-novariant-shared/libtrace-engine.so" \
    --entry lib/libzircon.so="$OUT_DIR/x64-novariant/gen/zircon/public/sysroot/lib/libzircon.so" \
    --entry lib/libc.so="$OUT_DIR/x64-novariant/gen/zircon/public/sysroot/lib/libc.so"

file -L $SYZ_DIR/obj/zircon.elf $SYZ_DIR/kernel $SYZ_DIR/fxfs.blk
