# setup SyzDescribe for spec generation

this document shows how to setup SyzDescribe and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$SYZDESCRIBE`: directory for saving SyzDescribe source
- `$LINUX`: directory for saving linux kernel source
- `$ANDROID`: directory for saving android kernel source

## preparation

create a docker image via `$CLOUD/experiment/SyzDescribe/Dockerfile` and enter the container.
```bash
docker build -t syzdescribe:latest --network host -f "$CLOUD/experiment/SyzDescribe/Dockerfile" .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name syzdescribe-exp \
    syzdescribe:latest tail -f /dev/null
docker exec -it syzdescribe-exp bash
```

## setup SyzDescribe

in the container, run following commands to setup SyzDescribe and generate specifications.
```bash
git clone https://github.com/seclab-ucr/SyzDescribe
cd SyzDescribe
git checkout a1c0e55bb111c076980ddf64c108cb7cb08dafb9
git apply "$CLOUD/experiment/SyzDescribe/repo.patch"
mkdir build && cd build
cmake -G "Unix Makefiles" -DLLVM_CONFIG_BINARY=/llvm-15/bin/llvm-config ..
make -j16
cd ..
```

## generate specs for linux

```bash
cd "$LINUX"
cp "$CLOUD/configs/kernel/syzbot.config" .config
PATH=/llvm-15/bin:$PATH make LLVM=1 olddefconfig all -j16

go run "$SYZDESCRIBE/kernelbc/gen.go" -isSaveTemp -toolchain=/llvm-15/bin
bash build.sh

cd "$SYZDESCRIBE"
./scripts/genspec.sh -k "$LINUX" -o workdir/specs-linux-v6.18
```

integrate them into syzkaller and feel free to perform fuzzing.
```bash
find "$SYZDESCRIBE/workdir/specs-linux-v6.18" -name "syz*" | xargs -I {} cp {} "$SYZKALLER/sys/linux"
```

generate specs for subsystems:
```bash
./script/genspec_subsystems.sh -o workdir/output_subsystems
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
go run "$SYZDESCRIBE/kernelbc/gen.go" -path "$ANDROID_OUT" -toolchain /llvm-15/bin

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

generate specs:
```bash
"$SYZDESCRIBE/script/genspec.sh" \
    -k "$ANDROID_OUT" \
    -o "$SYZDESCRIBE/workdir/specs-android17-6.18"
```
