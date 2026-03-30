# setup SyzGenPlusPlus

this document shows how to setup SyzGenPlusPlus and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$SYZGENPP`: directory for saving SyzGenPlusPlus source

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
python3 scripts/genConfig.py --name 6.18 -t linux --type qemu --image linux-distro/image --version 6.18
python3 main.py -s find_drivers

jq -r 'to_entries[] | select(.value.ops != null) | .key' workdir/6.18/model/services.json | xargs -I {} python3 main.py -s all --target {}
```

integrate them into syzkaller and feel free to perform fuzzing.
```bash
cd $SYZGENPP
cp gopath/src/github.com/google/syzkaller/sys/linux/*_gen.txt syzkaller/sys/linux
```

If you want to re-extract const for KernelGPT's specs:
```bash
cd $KERNLGPT/syzkaller
make bin/syz-extract
ls sys/linux/gpt4*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```