#!/bin/bash

# this script is used to build freebsd kernel on linux/freebsd for fuzzing

set -e

# required args
SOURCEDIR=  # path to freebsd-src

# optional args
JOBS=8  # number of parallel jobs
GEN_COMPILE_CMD=0  # whether to generate compile_commands.json

HOSTOS=$(go env GOHOSTOS)
HOSTARCH=$(go env GOHOSTARCH)

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -s, --sourcedir   <SOURCEDIR> path to freebsd source directory"
    echo "  args (optional):"
    echo "    -j, --jobs        <JOBS>      number of parallel build jobs (default: 8)"
    echo "    -c, --compile_commands        whether to generate compile_commands.json of buildkernel (default: false)"
    echo "                                  if set, the compile_commands.json will be generated at the root of <SOURCEDIR>"
    echo "    -h, --help                    print help message"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        -s | --sourcedir)
            SOURCEDIR="$2"
            shift
            shift
            ;;
        -j | --jobs)
            JOBS="$2"
            shift
            shift
            ;;
        -c | --compile_commands)
            GEN_COMPILE_CMD=1
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

if [ -z "$SOURCEDIR" ]; then
    echo "error: --sourcedir is required"
    exit 1
fi

build_freebsd_on_linux() {
    mkdir -p $SOURCEDIR/build
    MAKEOBJDIRPREFIX=$SOURCEDIR/build $SOURCEDIR/tools/build/make.py \
        -j $JOBS --cross-bindir=$LLVM_HOME/bin \
        TARGET=$HOSTARCH TARGET_ARCH=$HOSTARCH \
        buildworld
    MAKEOBJDIRPREFIX=$SOURCEDIR/build $BEAR $SOURCEDIR/tools/build/make.py \
        -j $JOBS --cross-bindir=$LLVM_HOME/bin \
        TARGET=$HOSTARCH TARGET_ARCH=$HOSTARCH \
        buildkernel KERNCONF=CLOUD
    MAKEOBJDIRPREFIX=$SOURCEDIR/build $SOURCEDIR/tools/build/make.py \
        -j $JOBS --cross-bindir=$LLVM_HOME/bin \
        TARGET=$HOSTARCH TARGET_ARCH=$HOSTARCH \
        installkernel KERNCONF=CLOUD DESTDIR=$SOURCEDIR/build/dist

    # TODO: how to generate compile_commands.json on linux for freebsd?
    if [ $GEN_COMPILE_CMD -eq 1 ]; then
        echo "generate compile_commands.json for freebsd on linux host is not supported yet."
    fi
}

build_freebsd_on_freebsd() {
    cd $SOURCEDIR/sys/$HOSTARCH/conf
    config CLOUD && cd ../compile/CLOUD
    make cleandepend && make depend
    if [ $GEN_COMPILE_CMD -eq 1 ]; then
        compiledb make -n
    fi
    make -j$JOBS
    if [ $GEN_COMPILE_CMD -eq 1 ]; then
        echo "$SOURCEDIR/sys/$HOSTARCH/conf/compile/CLOUD/compile_commands.json is generated"
    fi
    echo "you can run 'make installkernel' under $SOURCEDIR to install the kernel on the machine"
}

echo "building freebsd kernel on $HOSTOS ..."
KERNEL_CONFIG_PATH=$(dirname $(realpath $0))/../../configs/kernel/freebsd.config
cd $SOURCEDIR
cp $KERNEL_CONFIG_PATH $SOURCEDIR/sys/$HOSTARCH/conf/CLOUD
build_freebsd_on_$HOSTOS
echo "done!"