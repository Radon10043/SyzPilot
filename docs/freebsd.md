# setup cloud and run fuzzing for freebsd kernel

Please replace the following variables according to the actual situation:
- `$VMDIR`: directory for saving FreeBSD image(s).
- `$KERNSRC_HOST`: directory for saving FreeBSD kernel source (host).
- `$KERNSRC_VM`: directory for saving FreeBSD kernel source (vm).
- `CLOUD_HOST`: directory for saveing cloud source (host).
- `CLOUD_VM`: directory for saveing cloud source (vm).
- `$LLVM_HOME`: directory for llvm

## ubuntu host, qemu vm

### environment setup

Download FreeBSD image from [https://download.freebsd.org/snapshots/VM-IMAGES](https://download.freebsd.org/snapshots/VM-IMAGES). I use [15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz](https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz).
```bash
# run following commands on host
cd $VMDIR
wget https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
mv FreeBSD-15.0-STABLE-amd64-ufs.qcow2 dev.qcow2
qemu-img resize dev.qcow2 200G
qemu-system-x86_64 -m 16G -smp 16 -hda ./dev.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
```

press 3 and input `set console="comconsole"` and `boot`.

setup environment of vm:
```sh
# run following commands on vm
echo "autoboot_delay=\"-1\"" >> /boot/loader.conf
echo "console=\"comconsole\"" >> /boot/loader.conf
/etc/rc.d/growfs onestart
echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config
echo 'PermitEmptyPasswords yes' >> /etc/ssh/sshd_config
echo 'Subsystem sftp /usr/libexec/sftp-server' >> /etc/ssh/sshd_config
service sshd enable
service sshd start
passwd # root

sysrc sshd_enable=YES
sysrc ifconfig_DEFAULT=DHCP
echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
echo "UseDNS no" >> /etc/ssh/sshd_config
echo "GSSAPIAuthentication no" >> /etc/ssh/sshd_config

# you may need proxies
# echo "export http_proxy=http://10.0.2.2:7890" >> ~/.shrc
# echo "export https_proxy=http://10.0.2.2:7890" >> ~/.shrc
# exec sh
ASSUME_ALWAYS_YES=true pkg update -f
ASSUME_ALWAYS_YES=true pkg install bash curl gcc git gmake go golangci-lint llvm cmake
ASSUME_ALWAYS_YES=true pkg install vim dnsmasq wget tmux ccache pkgconf sqlite3 python3

python3 -m ensurepip
pip3 install compiledb
```

install flatbuffers v23.5.26:
```sh
# run following commands on vm
wget https://github.com/google/flatbuffers/archive/refs/tags/v23.5.26.tar.gz
tar -xzvf v23.5.26.tar.gz
cd flatbuffers-23.5.26
cmake -G "Unix Makefiles"
make -j$(sysctl -n hw.ncpu) && make install
cd ..
rm -rf flatbuffers-23.5.26 v23.5.26.tar.gz
```

### cloud setup

download cloud and submodules:
```sh
# run following commands on vm
cd /root
git clone --recurse-submodules https://github.com/Radon10043/cloud
# if you forgot to clone with --recurse-submodules, run `git submodule update --init --recursive` under cloud directory to update submodules
```

build cloud:
```sh
# run following commands on vm
cd $CLOUD_VM
gmake
```

download source of FreeBSD, generate `compile_commands.json` and build kernel:
```sh
# run following commands on vm
cd /root
mkdir -p freebsd/15.0.0
cd freebsd/15.0.0
git clone -b release/15.0.0 --depth 1 https://github.com/freebsd/freebsd-src build
cp -r build extract # former for kernel building, latter for const extraction

cd build/sys/amd64/conf
cp $CLOUD_VM/configs/kernel/freebsd.config CLOUD
config CLOUD && cd ../compile/CLOUD
make cleandepend && make depend
compiledb make -n
make -j$(sysctl -n hw.ncpu)
```

analyze `compile_commands.json` of FreeBSD and create kernel source database:
```sh
# run following commands on vm
cd $CLOUD_VM
./bin/analyzer \
    -i $KERNSRC_VM/build/15.0.0/sys/amd64/compile/CLOUD/compile_commands.json \
    -j 8 \
    -o $CLOUD/data/database/freebsd.db
```

run spec generator:
```sh
# run following commands on vm
$CLOUD_VM/bin/generator \
    -db=$CLOUD_VM/data/dadabase/freebsd.db \
    -os=freebsd \
    -outdir=$CLOUD_VM/workdir/gen-specs \
    -kernel=$KERNSRC_VM/build/15.0.0 \
    -model=gemini-2.5-flash \
    -varlist=$CLOUD_VM/workdir/gen-specs/varlist.txt \
    -jobs=4 > logs/generator.log 2>&1
```

After generator finishes executing, check `$CLOUD_VM/workdir/gen-specs` for details.

feel free to shutdown vm:
```sh
# run following commands on vm
shutdown -p now
```

### start fuzzing

start vm with image `dev.qcow2` and ssh to it:
```bash
# run following commands on host
# start vm in daemon
qemu-system-x86_64 -m 16G -smp 16 -hda $VMDIR/dev.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

generate and install ssh key:
```bash
# run following commands on host
cd $VMDIR
ssh-keygen -t rsa -f ./freebsd.id_rsa -N ""
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

in vm, build syzkaller and copy executor programs to host:
```sh
# run following commands on vm
cd $CLOUD/syzkaller && gmake target

# run following commands on host
scp -i ./freebsd.id_rsa -P 3733 -r -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost:$CLOUD/syzkaller/freebsd_amd64 $CLOUD/syzkaller/bin
```

shutdown vm:
```sh
# run following commands on vm
shutdown -p now
```

duplicate an image to differentiate between the development machine and the fuzzing target machine.
```bash
# run following commands on host
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

start vm with image `target.qcow2` and ssh to it:
```bash
# run following commands on host
# start vm in daemon
qemu-system-x86_64 -m 16G -smp 16 -hda $VMDIR/target.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

replace kernel in `target.qcow2` with new built kernel:
```bash
# run following commands on host
cd $KERNSRC/sys/amd64/compile/CLOUD
make install
reboot
uname -i # expect is CLOUD
shutdown -p now
```

run syzkaller:
```bash
# run following commands on host
cd $CLOUD
mkdir workdir
cat <<__EOF__ > workdir/freebsd.cfg
{
    "name": "freebsd",
    "target": "freebsd/amd64",
    "http": ":10000",
    "workdir": "$CLOUD/workdir",
    "syzkaller": "$CLOUD/syzkaller",
    "sshkey": "$VMDIR/freebsd.id_rsa",
    "sandbox": "none",
    "procs": 8,
    "image": "$VMDIR/target.qcow2",
    "type": "qemu",
    "vm": {
        "count": 1,
        "cpu": 4,
        "mem": 2048
    }
}
__EOF__

./syzkaller/bin/syz-manager -config=./workdir/freebsd.cfg
```

## ubuntu host, freebsd vm, bhyve nest vm

[https://secfault-security.com/blog/fuzzing_freebsd.html](https://secfault-security.com/blog/fuzzing_freebsd.html)

```sh
wget -P vm-images https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-zfs.qcow2.xz
unxz -k vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2.xz
qemu-img resize vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2 +50G
qemu-system-x86_64 -m 16G -smp 16 -hda vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::10022-:22 -nographic -cpu host
```

start vm, press 3 and input:
```
set console="comconsole"
boot
```

under vm:
```sh
echo "autoboot_delay=\"-1\"" >> /boot/loader.conf
echo "console=\"comconsole\"" >> /boot/loader.conf
/etc/rc.d/growfs onestart

echo "hw.vmm.vmx.use_apic_vid=\"0\"" >> /boot/loader.conf
echo "hw.vmm.vmx.use_tpr_shadow=\"0\"" >> /boot/loader.conf
# echo "hw.vmm.vmx.use_msr_bitmap=\"0\"" >> /boot/loader.conf

echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
sysrc sshd_enable=YES
sysrc ifconfig_DEFAULT=DHCP
/etc/rc.d/sshd start

passwd # root

reboot

# set proxy
# export http_proxy=http://10.0.2.2:7890
# export https_proxy=http://10.0.2.2:7890

ASSUME_ALWAYS_YES=true pkg update -f
ASSUME_ALWAYS_YES=true pkg install bash curl gcc git gmake go golangci-lint llvm vim dnsmasq wget tmux

dd if=/dev/zero of=/usr/swap0 bs=1m count=16384
chmod 0600 /usr/swap0
echo -e "md99\tnone\tswap\tsw,file=/usr/swap0,late\t0\t0" >> /etc/fstab
swapon -aL

# build syzkaller
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout 4b25d554
gmake -j4
cd -

wget https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64.raw.xz
unxz FreeBSD-15.0-STABLE-amd64.raw.xz
```

Configure nested VM:
```sh
zfs create -o mountpoint=/syzkaller zroot/syzkaller
mv FreeBSD-15.0-STABLE-amd64.raw /syzkaller/

ifconfig bridge create bridge0
ifconfig bridge0 inet 169.254.0.1
echo 'dhcp-range=169.254.0.2,169.254.0.254,255.255.255.0' > /usr/local/etc/dnsmasq.conf
echo 'interface=bridge0' >> /usr/local/etc/dnsmasq.conf
sysrc dnsmasq_enable=YES
service dnsmasq start
echo 'net.link.tap.up_on_open=1' >> /etc/sysctl.conf
sysctl net.link.tap.up_on_open=1

echo "cloned_interfaces=\"bridge0 tap0\"" >> /etc/rc.conf
echo "ifconfig_bridge0=\"inet 169.254.0.1 addm tap0 up\"" >> /etc/rc.conf
echo "ifconfig_tap0=\"up\"" >> /etc/rc.conf

ifconfig tap create
ifconfig bridge0 addm tap0
bhyveload -c stdio -m 512M -d /syzkaller/FreeBSD-15.0-STABLE-amd64.raw -e autoboot_delay=0 testvm0
bhyve -H -A -P -c 1 -m 512M -s 0:0,hostbridge -s 1:0,lpc -s 2:0,virtio-net,tap0 -s 3:0,virtio-blk,/syzkaller/FreeBSD-15.0-STABLE-amd64.raw -l com1,stdio testvm0
```

in the nested bhyve vm:
```sh
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config

sysrc sshd_enable=YES
sysrc ifconfig_DEFAULT=DHCP
/etc/rc.d/sshd start

passwd # root
```

dont stop the nested vm, in the host vm:
```sh
ssh-keygen -f nest.id_rsa -t rsa -N ''
ssh-copy-id -i ./nest.id_rsa.pub root@169.254.0.46 # You can connect to nest vm via nest.id_rsa now
```

shutdown nested vm now:
```sh
shutdown -p now
```

in the host vm, build kenrel:
```sh
git clone https://github.com/freebsd/freebsd-src /usr/src
cd /usr/src
git checkout release/15.0.0

cd /usr/src/sys/amd64/conf
cat <<__EOF__ > SYZKALLER
include "./GENERIC"

ident	SYZKALLER

options 	COVERAGE
options 	KCOV
__EOF__

cd /usr/src
make -j $(sysctl -n hw.ncpu) KERNCONF=SYZKALLER buildkernel
```

Run following commands on host vm to install kernel to nested vm:
```sh
mdconfig -a -f /syzkaller/FreeBSD-15.0-STABLE-amd64.raw
mount /dev/md0p4 /mnt
cd /usr/src
make KERNCONF=SYZKALLER installkernel DESTDIR=/mnt
umount /mnt
mdconfig -d -u 0
```

configure syzkaller:
```json
{
    "name": "freebsd",
    "target": "freebsd/amd64",
    "http": "127.0.0.1:56741",
    "workdir": "/root/workdir",
    "syzkaller": "/root/syzkaller/",
    "kernel_obj": "/usr/obj/usr/src/amd64.amd64/sys/SYZKALLER",
    "kernel_src": "/",
    "sshkey": "/root/nest.id_rsa",
    "sandbox": "none",
    "procs": 8,
    "image": "/syzkaller/FreeBSD-15.0-STABLE-amd64.raw",
    "type": "bhyve",
    "vm": {
        "count": 1,
        "cpu": 2,
        "mem": "2048M",
        "bridge": "bridge0",
        "hostip": "169.254.0.1",
        "dataset": "zroot/syzkaller"
    }
}
```

run syzkaller:
```sh
cd /root
syzkaller/bin/syz-manager -config=./freebsd.cfg
```

## development environment setup (optional)

### neovim

run following commands to setup development environment for neovim:
```sh
# run following commands on vm:
ASSUME_ALWAYS_YES=true pkg install neovim rust ripgrep fd-find lazygit
cargo install tree-sitter-cli
echo "export PATH=$PATH:$HOME/.cargo/bin" >> $HOME/.shrc
exec sh
git clone https://github.com/Radon10043/nvimcfg /root/.config/nvim # my neovim config
```

### sshfs+vscode

use sshfs to mount directory on host and develop via vscode or other tools you prefer:
```bash
sshfs -p 3733 \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -o sftp_server=/usr/libexec/sftp-server \
    -o cache=yes \
    -o kernel_cache \
    -o compression=no \
    -o idmap=user \
    -o follow_symlinks \
    root@localhost:$CLOUD_VM ./mnt/cloud
```

## build and replace freebsd kernel on linux host

build and replace FreeBSD kernel on Linux host is an option, but it's less efficient than build and replace it directly on FreeBSD. I'm noting this method down here, as I might need it in the future.

build freebsd kernel:
```bash
# run following command on host
cd $KERNSRC
cp $CLOUD/configs/kernel/freebsd.config sys/amd64/conf/CLOUD

mkdir build dist
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 buildworld
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 buildkernel KERNCONF=CLOUD
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 installkernel KERNCONF=CLOUD DESTDIR=$PWD/dist
```

start freebsd vm:
```bash
# run following commands on host
qemu-system-x86_64 -m 16G -smp 16 -hda target.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none   # start vm in daemon ...
```

install built kernel to vm:
```bash
# run following commands on host
cd $VMDIR
scp -P 3733 -r -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no $KERNSRC_HOST/build/15.0.0/dist/* root@localhost:/
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost reboot
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -i # output should be CLOUD
```

## frequently used commands

start freebsd vm silently.
```bash
qemu-system-x86_64 \
    -m 16G \
    -smp 16 \
    -hda $VMDIR/dev.qcow2 \
    -enable-kvm \
    -net nic \
    -net user,hostfwd=tcp::3733-:22 \
    -cpu host \
    -display none \
    -daemonize
```

start vm.
```bash
qemu-system-x86_64 \
    -m 16G \
    -smp 16 \
    -hda $VMDIR/dev.qcow2 \
    -enable-kvm \
    -net nic \
    -net user,hostfwd=tcp::3733-:22 \
    -nographic \
    -cpu host
```

ssh to vm:
```bash
ssh -p 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    root@localhost
```

copy file(s) to vm:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```

use sshfs to mount directory:
```bash
mkdir -p mnt/cloud
sshfs -p 3733 \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -o sftp_server=/usr/libexec/sftp-server \
    -o cache=yes \
    -o kernel_cache \
    -o compression=no \
    -o idmap=user \
    -o follow_symlinks \
    root@localhost:/root/cloud ./mnt/cloud
```

unmount directory mounted by sshfs:
```bash
fusermount -u mnt/cloud
```
