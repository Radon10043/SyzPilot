# SyzGenPlusPlus

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

./setup.sh
source fuzz/bin/activate
pip install pexpect ipython angr==9.2.42 pycparser==2.21 "capstone<5.0.0" "setuptools<70.0.0"
python3 scripts/genConfig.py --name 6.18 -t linux --type qemu --image linux-distro/image --version 6.18
python3 main.py -s find_drivers
jq -r 'to_entries[] | select(.value.ops != null) | .key' workdir/6.18/model/services.json | xargs -I {} python3 main.py -s all --target {}
```
