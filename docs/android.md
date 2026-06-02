# setup cloud and run fuzzing for Android kernel

There are two ways for Android kernel fuzzing: [Debian image + Android GKI](#debian-image-android-gki) and [Android GSI + Android GKI](#android-gsi-android-gki). I recommend the former :)

Please replace the following variables according to the actual situation:
- `$DEBIAN`: Directory for saving Debian image.
- `$GKI`: Directory for saving Android GKI source.
- `$CLOUD`: Directory for saveing cloud source.

## Debian image, Android GKI

Let's hack Android kernel and run Debian image upon it.

### Download Android GKI source

Let's use common-android17-6.18 as example:
```bash
mkdir $GKI && cd $GKI
repo init -u https://android.googlesource.com/kernel/manifest -b common-android17-6.18 --depth=1
repo sync -c
```

### Build Android GKI

See [https://source.android.com/docs/setup/reference/bazel-support](https://source.android.com/docs/setup/reference/bazel-support) for build system support. You can use [kleaf](#build-via-kleaf) or [build.sh](#build-via-buildsh) to build the kernel.

#### Build via kleaf

Patch and build GKI:
```bash
cd $GKI
# please choose the correct patch according GKI branch
git -C common apply $CLOUD/patch/android/android17-6.18.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist
```

Generate corresponding `compile_commands.json`:
```bash
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_compile_commands
```

If you want to specific save path of `compile_commands.json`, run:
```bash
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_compile_commands -- $PWD/dist/compile_commands.json
```

#### Build via build.sh

We use common-android13-5.15 here to describe how to build and generate compile_commands.json for Android kernel. Patch and build GKI:
```bash
cd $GKI
git -C common apply $CLOUD/patch/android/android13-5.15.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh
```

Generate corresponding `compile_commands.json`:
```bash
cd $GKI
python3 common/scripts/clang-tools/gen_compile_commands.py -d out/android13-5.15/common -o compile_commands.json out/android13-5.15/common out/android13-5.15/common-modules/virtual-device
```

### Construct kernel database

Build analyzer with Android patch:
```bash
cd $CLOUD
make TARGETOS=android anayzler
```

Analyze compile_commands.json to constrcut a kernel database:
```bash
cd $CLOUD
./bin/analyzer -i $GKI/compile_commands.json -o data/database/android.db -j 4
```

**Note:** You may encounter `unknown options` errors during analysis, which may lead to incomplete database. To avoid such errors, please add options to be skipped in [analyze.cpp](/src/analyzer/analyze.cpp) and rebuild analyzer (perhaps it can be handled in a parameterized manner?).

### Generate specs

Run generator to generate specs for android kernel
```bash
$CLOUD/bin/generator \
	-db=$CLOUD/data/database/android.db \
	-os=android \
	-model=gemini-3-flash-preview \
	-kernel=$GKI \
	-outdir=$WORKDIR \
	-ref=$CLOUD/data/refs/android/test.txt
```

### Start fuzzing

Create Debian image:
```bash
cd $DEBIAN
cp $CLOUD/scripts/android/create-image.sh .
chmod +x ./create-image.sh && ./create-image.sh
```

Patch and build syzkaller:
```bash
cd $CLOUD/syzkaller
git apply ../patch/syzkaller/android.patch
make all
```

Start fuzzing:
```bash
cd $CLOUD && mkdir workdir
cat <<__EOF__ > workdir/android.cfg
{
	"name": "android",
	"target": "linux/amd64",
	"http": "127.0.0.1:56741",
	"workdir": "$CLOUD/workdir",
	"kernel_obj": "$GKI/dist",
	"image": "$IMAGE/bullseye.img",
	"sshkey": "$IMAGE/bullseye.id_rsa",
	"syzkaller": "$CLOUD/syzkaller",
	"procs": 8,
	"type": "qemu",
	"reproduce": false,
	"vm": {
		"count": 1,
		"kernel": "$GKI/dist/bzImage",
		"cpu": 2,
		"mem": 2048,
		"image_device": "drive format=raw,if=none,id=rootdisk,file="
	}
}
__EOF__
```

## Android GSI, Android GKI

please follow [syzkaller's doc about running Android virtual device](https://github.com/google/syzkaller/blob/master/docs/linux/setup_linux-host_android-virtual-device_x86-64-kernel.md), there are some differences:

update deps:
```bash
apt update
apt install -y elfutils libelf-dev libdw-dev lz4 liblz4-dev
```

you don't need to install cuttlefish from scratch, install it from apt is ok:
```bash
curl -fsSL https://us-apt.pkg.dev/doc/repo-signing-key.gpg \
    -o /etc/apt/trusted.gpg.d/artifact-registry.asc
chmod a+r /etc/apt/trusted.gpg.d/artifact-registry.asc
echo "deb https://us-apt.pkg.dev/projects/android-cuttlefish-artifacts android-cuttlefish main" \
    | tee -a /etc/apt/sources.list.d/artifact-registry.list
apt update
apt-get install -y cuttlefish-base cuttlefish-user cuttlefish-orchestration
```

download kernel source:
```bash
mkdir common-android13-5.15 && cd common-android13-5.15
repo init -u https://android.googlesource.com/kernel/manifest -b common-android13-5.15 --depth=1
repo sync -c
```

note: ubuntu 24 don't support libncurses5, it use libncurses6. We need libncurses5 to compile android13-gsi, let's use libncurses6 to hack it:
```bash
apt-get update
apt-get install libncurses6 libtinfo6
ln -s /usr/lib/x86_64-linux-gnu/libncurses.so.6 /usr/lib/x86_64-linux-gnu/libncurses.so.5
ln -s /usr/lib/x86_64-linux-gnu/libtinfo.so.6 /usr/lib/x86_64-linux-gnu/libtinfo.so.5
```

```bash
repo init -u https://android.googlesource.com/platform/manifest -b android13-gsi --depth=1
```

build android kernel via `build.sh`:
```bash
# for common-android13-5.15, 13-5.10, 12-5.4
BUILD_CONFIG=common/build.config.gki_kasan.x86_64 build/build.sh
BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh
```

build android kernel via `bazel`:
```bash
# for common-android17-6.18, 16-6.12, 15-6.6, 14-6.1
tools/bazel run --kasan //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist
```

generate compile_commands.json:
```bash
# for build.sh
python3 common/scripts/clang-tools/gen_compile_commands.py -d out/android13-5.15/common

# for bazel
tools/bazel run --kasan //common-modules/virtual-device:virtual_device_x86_64_compile_commands
```

generate kernel .config file:
```bash
tools/bazel build --kasan //common-modules/virtual-device:virtual_device_x86_64_config
```

startup android kernel:
```bash
cd $GSI
source build/envsetup.sh
lunch aosp_cf_x86_64_phone-userdebug
launch_cvd \
     -kernel_path=$GKI/out/android13-5.15/dist/bzImage \
     -initramfs_path=$GKI/out/android13-5.15/dist/initramfs.img \
     -daemon
```

feel free to stop:
```bash
stop_cvd
```
