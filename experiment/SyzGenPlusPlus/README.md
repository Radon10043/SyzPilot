# setup SyzGenPlusPlus for spec generation

this document shows how to setup SyzGenPlusPlus and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing SyzPilot source
- `$SYZGENPP`: directory for saving SyzGenPlusPlus source
- `$LINUX`: directory for saving linux kernel source
- `$ANDROID`: directory for saving android kernel source

## preparation

create a docker image via `$CLOUD/experiment/SyzGenPlusPlus/Dockerfile` and enter the container.
```bash
docker build -t syzgenpp:latest --network host -f $CLOUD/experiment/SyzGenPlusPlus/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name syzgenpp-exp \
    syzgenpp:latest tail -f /dev/null
docker exec -it syzgenpp-exp bash
```

## setup SyzGenPlusPlus

in the container, run following commands to setup SyzGenPlusPlus and generate specifications.
```bash
git clone https://github.com/seclab-ucr/SyzGenPlusPlus
cd SyzGenPlusPlus
git checkout 7c0838106554796dfdab1c3285858f53d6fd76bb
git apply $CLOUD/experiment/SyzGenPlusPlus/repo.patch

mkdir linux-distro
python3 scripts/download.py -c "https://raw.githubusercontent.com/google/syzkaller/ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219/dashboard/config/linux/upstream-apparmor-kasan.config" --build -v 6.18

mkdir linux-distro/image && cd linux-distro/image
cp $CLOUD/scripts/linux/create-image.sh .
chmod +x ./create-image.sh
./create-image.sh

cd $SYZGENPP
./setup.sh
source fuzz/bin/activate
pip install pexpect ipython angr==9.2.42 pycparser==2.21 "capstone<5.0.0" "setuptools<70.0.0"

git clone https://github.com/angr/angr-targets && cd angr-targets
git checkout b3c4ce5dfced0437f4810057aea4f78ecf56f4f7
pip install -e .
```

## generate specs for linux

```bash
cd $SYZGENPP
source fuzz/bin/activate
python3 scripts/genConfig.py --name 6.18 -t linux --type qemu --image linux-distro/image --version 6.18
python3 main.py -s FIND_DRIVER

jq -r 'to_entries[] | select(.value.ops != null) | .key' workdir/6.18/model/services.json | xargs -I {} python3 main.py -s ALL --target {}
```

integrate them into syzkaller and feel free to perform fuzzing.
```bash
cd $SYZGENPP
cp gopath/src/github.com/google/syzkaller/sys/linux/*_gen.txt syzkaller/sys/linux
```

If you want to re-extract const for specs of SyzGen++:
```bash
cd $SYZGENPP/syzkaller
make bin/syz-extract
ls sys/linux/*_gen.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$LINUX -os=linux -arch=amd64
```

generate specs for subsystems:
```bash
cd $SYZGENPP
./run_subsystems.sh
```

## generate specs for android

re-create an android-specific debian image:
```bash
cd $SYZGENPP
rm -rf linux-distro/image/*
cp $CLOUD/scripts/android/create-image.sh linux-distro/image/
cd linux-distro/image
./create-image.sh
```

build android kernel under `linux-distro`:
```bash
cd $SYZGENPP
source fuzz/bin/activate

mkdir linux-distro
cp -a "$ANDROID/common" linux-distro/linux-android17_6.18-fuzz
ln -sfn linux-android17_6.18-fuzz linux-distro/linux-android17_6.18-raw
cd linux-distro/linux-android17_6.18-fuzz
cp "$ANDROID/dist/kernel_x86_64_dot_config" .config

export PATH="$ANDROID/prebuilts/clang/host/linux-x86/clang-r547379/bin":$PATH
scripts/kconfig/merge_config.sh -m .config "$CLOUD/experiment/SyzGenPlusPlus/configs/android.config"
make LLVM=1 LLVM_IAS=1 olddefconfig
make LLVM=1 LLVM_IAS=1 -j$(nproc) all
```

generate specs:
```bash
cd $SYZGENPP

python3 scripts/genConfig.py \
    --name common-android17-6.18 \
    -t linux \
    --type qemu \
    --image linux-distro/image \
    --version android17_6.18 \
    --config android.config
jq --arg pwd "$PWD" --arg android "$ANDROID" '. + {
    kernel_source: "\($pwd)/linux-distro/linux-android17_6.18-fuzz",
    kernel_toolchain: "\($android)/prebuilts/clang/host/linux-x86/clang-r547379/bin",
    LLVM: 1,
    LLVM_IAS: 1,
    kbuild_args: "CONFIG_DEBUG_INFO_BTF_MODULES="
}' android.config > /tmp/a.config
mv /tmp/a.config android.config

python3 main.py --config android.config -s FIND_DRIVER
jq -r 'to_entries[] | select(.value.ops != null) | .key' workdir/common-android17-6.18/model/services.json | xargs -I {} python3 main.py --config android.config -s ALL --target {}
```

integrate them into syzkaller and feel free to perform fuzzing.
```bash
cd $SYZGENPP
cp gopath/src/github.com/google/syzkaller/sys/linux/*_gen.txt syzkaller/sys/linux
```

If you want to re-extract const for specs of SyzGen++:
```bash
cd $SYZGENPP/syzkaller
make bin/syz-extract
ls sys/linux/*_gen.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$ANDROID/common -os=linux -arch=amd64
```
