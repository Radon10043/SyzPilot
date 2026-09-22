# setup SyzSpec for spec generation

this document shows how to setup SyzSpec and use it for generating syscall specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing SyzPilot source
- `$SYZSPEC`: directory for saving SyzSpec source
- `$KERNSRC`: directory for saving linux kernel source

## preparation

create a docker image via `$CLOUD/experiment/SyzDescribe/Dockerfile` and enter the container.
```bash
docker build -t syzspec:latest --network host -f $CLOUD/experiment/SyzDescribe/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name syzspec-exp \
    syzspec:latest tail -f /dev/null
docker exec -it syzspec-exp bash
```

## setup SyzSpec

in container, setup SyzSpec:
```bash
git clone https://github.com/seclab-ucr/SyzSpec
cd SyzSpec
git checkout 1edbcffd6f56786d914b0c04458bee86abf215ac
git apply $CLOUD/experiment/SyzSpec/repo.patch

mkdir build && cd build
cmake .. \
	-DCMAKE_C_COMPILER=/tmp/llvm-150-install_O_D_A/bin/clang \
    -DCMAKE_CXX_COMPILER=/tmp/llvm-150-install_O_D_A/bin/clang++ \
	-DLLVM_DIR=/tmp/llvm-150-install_O_D_A/lib/cmake/llvm \
	-DENABLE_SOLVER_Z3=ON \
	-DENABLE_TCMALLOC=OFF \
	-DCMAKE_PREFIX_PATH="/tmp/z3-4.8.15-install;/tmp/sqlite-amalgamation-3400100"
make -j8
```

## generate specs for linux

build linux kernel and generate .bc files.
```bash
cd $KERNSRC # v6.18
cp $CLOUD/configs/kernel/syzbot.config .config
make LLVM=1 \
	 PATH=/llvm-15/bin:$PATH \
	 KCFLAGS="-Xclang -no-opaque-pointers -mllvm -opaque-pointers=0" \
	 KBUILD_LDFLAGS="-mllvm -opaque-pointers=0" \
	 olddefconfig all -j16

go run "$SYZSPEC/kernelbc/gen.go" -isSaveTemp -toolchain=/llvm-15/bin
bash build.sh
```

generate specification for driver(s), e.g. for ppp, run:
```bash
"$SYZSPEC/build/bin/klee" \
	--entry-point=ppp_ioctl \
	--spec-arguments-index=1 \
	--spec-arguments-num=2 \
	--spec-interface-name=ioctl \
	--spec-prefix="fd fd_syzspec_ppp" \
	--spec-suffix="" \
	--spec-output="ioctl" \
    --max-time=24h \
	"$KERNSRC/drivers/net/ppp/built-in.bc"
```

generate specification for subsystems used in SyzSpec paper (24h timeout):
```bash
cd $SYZPSEC
SYZSPEC=$SYZSPEC KERNSRC=$KERNSRC OUTDIR=./workdir ./scripts/genspec.sh
```

copy generated specs to syzkaller:
```bash
cd $SYZSPEC
find workdir/ -name "syz*.txt" | xargs cp -I {} syzkaller/sys/linux/
```

manually fix generated spec, then extract consts and format generated specs:
```bash
cd $SYZSPEC/syzkaller
make bin/syz-extract
ls sys/linux/syz*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```

## generate specs for android

build android via kleaf, e.g. common-android17-6.18:
```bash
cd "$ANDROID"
git -C common apply "$CLOUD/patch/android/android17-6.18.common.patch"
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist
```

```bash
export ANDROID_OUT=$(find "$ANDROID/out/bazel/output_user_root" -type d -path '*/sandbox_stash/KernelBuild/*/execroot/_main/out/android17-6.18/common' | sort -V | tail -1)

cd "$ANDROID_OUT"
go run "$SYZSPEC/kernelbc/gen.go" -path "$ANDROID_OUT" -toolchain /llvm-15/bin -isSaveTemp

export SANDBOX_ROOT=$(rg --no-filename -o "$ANDROID/out/bazel/output_user_root/[^ ]+/sandbox/linux-sandbox/[0-9]+/execroot/_main" build.sh | head -1)

perl -0pi -e "s|\Q$SANDBOX_ROOT\E|$ANDROID|g;
    s/ -gz=zstd//g;
    s/ -fsanitize=kcfi//g;
    s/ -fsanitize-cfi-icall-experimental-normalize-integers//g;
    s/ -mindirect-branch-cs-prefix//g" build.sh

awk '
    /^#!/ {next}
    /^# path:/ {next}
    /^[[:space:]]*(\/llvm-15\/bin\/clang|echo "" > )/ {
        print > "build.compile.sh"
        next
    }
    NF {
        print > "build.link.sh"
    }
' build.sh

JOBS=16
xargs -r -P "$JOBS" -I{} bash -lc '{}' < build.compile.sh

bash build.link.sh
```

generate spec for a subsystem, e.g. binderfs:
```bash
"$SYZSPEC/build/bin/klee" \
	--entry-point=binder_ctl_ioctl \
    --spec-arguments-index=1 \
	--spec-arguments-num=2 \
    --spec-interface-name=ioctl \
    --spec-prefix="fd fd_binderfs_ctrl" \
    --spec-suffix="" \
	--max-time=24h \
    --spec-output="binderfs" \
    "$ANDROID_OUT/drivers/android/binderfs.bc"
```
