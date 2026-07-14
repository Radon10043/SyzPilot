# setup kernel/android

(host) download and build android kernels via manifests under cloud/configs/kernel/android:
```bash
mkdir $EXPERIMENT_ROOT/kernel/android

# common-android17-6.18
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android17-6.18 && cd common-android17-6.18
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android17-6.18.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android17-6.18.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android17-6.18.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist

# common-android16-6.12
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android16-6.12 && cd common-android16-6.12
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android16-6.12.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android16-6.12.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android16-6.12.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist

# common-android15-6.6
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android15-6.6 && cd common-android15-6.6
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android15-6.6.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android15-6.6.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android15-6.6.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist

# common-android14-6.1
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android14-6.1 && cd common-android14-6.1
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android14-6.1.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android14-6.1.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android14-6.1.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=dist

# common-android14-5.15
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android14-5.15 && cd common-android14-5.15
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android14-5.15.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android14-5.15.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android14-5.15.common.patch
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_dist -- --destdir=

# common-android13-5.15
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android13-5.15 && cd common-android13-5.15
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android13-5.15.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android13-5.15.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android13-5.15.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh

# common-android13-5.10
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android13-5.10 && cd common-android13-5.10
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android13-5.10.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android13-5.10.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android13-5.10.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh

# common-android12-5.10
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android12-5.10 && cd common-android12-5.10
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android12-5.10.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android12-5.10.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android12-5.10.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh

# common-android12-5.4
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android12-5.4 && cd common-android12-5.4
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android12-5.4.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android12-5.4.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android12-5.4.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.virtual_device_kasan.x86_64 build/build.sh

# common-android11-5.4
cd $EXPERIMENT_ROOT/kernel/android
mkdir common-android11-5.4 && cd common-android11-5.4
mkdir -p .repo/manifests && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/android/common-android11-5.4.xml .repo/manifests/
repo init -u https://android.googlesource.com/kernel/manifest
repo init -m common-android11-5.4.xml --depth=1
repo sync -c
git -C common apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/android/android11-5.4.common.patch
DIST_DIR=dist BUILD_CONFIG=common/build.config.gki_kasan_debian.x86_64 build/build.sh
DIST_DIR=dist BUILD_CONFIG=common-modules/virtual-device/build.config.cuttlefish_kasan.x86_64 build/build.sh
```
