# Setup SyzPilot and run fuzzing for the FreeBSD kernel

Please replace the following variables according to your environment:
- `$VMDIR`: directory for saving FreeBSD image(s).
- `$KERNSRC`: directory for saving the FreeBSD kernel source code.
- `$SYZPILOT`: directory for saving the SyzPilot source code.
- `$LLVM_HOME`: directory for LLVM.

## Ubuntu host, qemu vm

### Environment setup

(host) Download the FreeBSD image from [https://download.freebsd.org/snapshots/VM-IMAGES](https://download.freebsd.org/snapshots/VM-IMAGES). This guide uses [15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz](https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz).
```bash
cd $VMDIR
wget https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
mv FreeBSD-15.0-STABLE-amd64-ufs.qcow2 dev.qcow2
qemu-img resize dev.qcow2 200G
qemu-system-x86_64 -m 16G -smp 16 -hda ./dev.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
```

Press 3, then enter `set console="comconsole"` and `boot`.

(vm) setup the vm environment:
```sh
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

# you may need proxies.
# echo "export http_proxy=http://10.0.2.2:7890" >> ~/.shrc
# echo "export https_proxy=http://10.0.2.2:7890" >> ~/.shrc
# exec sh
ASSUME_ALWAYS_YES=true pkg update -f
ASSUME_ALWAYS_YES=true pkg install bash curl gcc git gmake go golangci-lint llvm cmake
ASSUME_ALWAYS_YES=true pkg install vim dnsmasq wget tmux ccache pkgconf sqlite3 python3

python3 -m ensurepip
pip3 install compiledb
```

(vm) Install flatbuffers v23.5.26:
```sh
wget https://github.com/google/flatbuffers/archive/refs/tags/v23.5.26.tar.gz
tar -xzvf v23.5.26.tar.gz
cd flatbuffers-23.5.26
cmake -G "Unix Makefiles"
make -j$(sysctl -n hw.ncpu) && make install
cd ..
rm -rf flatbuffers-23.5.26 v23.5.26.tar.gz
```

### SyzPilot setup

(vm) Download SyzPilot and its submodules:
```sh
cd /root
git clone --recurse-submodules https://github.com/Radon10043/SyzPilot
# if you forgot to clone with --recurse-submodules, run `git submodule update --init --recursive` under the SyzPilot directory
```

(vm) Build SyzPilot:
```sh
cd $SYZPILOT
gmake
```

(vm) Download the FreeBSD source code, generate `compile_commands.json`, and build the kernel:
```sh
cd /root
mkdir -p freebsd/15.0.0
cd freebsd/15.0.0
git clone -b release/15.0.0 --depth 1 https://github.com/freebsd/freebsd-src build
cp -r build extract # the former is for kernel building, the latter for constant extraction

cd build/sys/amd64/conf
cp $SYZPILOT/configs/kernel/freebsd.config SYZPILOT
config SYZPILOT && cd ../compile/SYZPILOT
make cleandepend && make depend
compiledb make -n
make -j$(sysctl -n hw.ncpu)
```

(vm) Analyze FreeBSD's `compile_commands.json` and create the kernel source database:
```sh
cd $SYZPILOT
./bin/analyzer \
    -i $KERNSRC/build/15.0.0/sys/amd64/compile/SYZPILOT/compile_commands.json \
    -j 8 \
    -o $SYZPILOT/data/database/freebsd.db
```

(vm) Run the spec generator:
```sh
$SYZPILOT/bin/generator \
    -db=$SYZPILOT/data/database/freebsd.db \
    -os=freebsd \
    -outdir=$SYZPILOT/workdir/gen-specs \
    -kernel=$KERNSRC/build/15.0.0 \
    -model=gemini-2.5-flash \
    -ref=$REFFILE \
    -jobs=4 > logs/generator.log 2>&1
```

After the generator finishes, check `$SYZPILOT/workdir/gen-specs` for details.

(vm) Shutdown the vm when needed:
```sh
shutdown -p now
```

### Start fuzzing

(host) Start the vm with the `dev.qcow2` image and ssh to it:
```bash
# start the vm as a daemon.
qemu-system-x86_64 -m 16G -smp 16 -hda $VMDIR/dev.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

(host) Generate and install the ssh key:
```bash
cd $VMDIR
ssh-keygen -t rsa -f ./freebsd.id_rsa -N ""
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

(vm) Build syzkaller:
```sh
cd $SYZPILOT/syzkaller && gmake target
```

(host) Copy the executor programs to the host:
```bash
scp -i ./freebsd.id_rsa -P 3733 -r -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost:$SYZPILOT/syzkaller/freebsd_amd64 $SYZPILOT/syzkaller/bin
```

(vm) Shut down the vm:
```sh
shutdown -p now
```

(host) Duplicate the image to separate the development vm from the fuzzing target vm.
```bash
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

(host) Start the vm with the `target.qcow2` image and ssh to it:
```bash
# start the vm as a daemon.
qemu-system-x86_64 -m 16G -smp 16 -hda $VMDIR/target.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
```

(host) Replace the kernel in `target.qcow2` with the newly built kernel:
```bash
cd $KERNSRC/sys/amd64/compile/SYZPILOT
make install
reboot
uname -i # expected output is SYZPILOT
shutdown -p now
```

(host) Run syzkaller:
```bash
cd $SYZPILOT
mkdir workdir
cat <<__EOF__ > workdir/freebsd.cfg
{
    "name": "freebsd",
    "target": "freebsd/amd64",
    "http": ":10000",
    "workdir": "$SYZPILOT/workdir",
    "syzkaller": "$SYZPILOT/syzkaller",
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

## Ubuntu host, FreeBSD vm, nested bhyve vm

[https://secfault-security.com/blog/fuzzing_freebsd.html](https://secfault-security.com/blog/fuzzing_freebsd.html)

```sh
wget -P vm-images https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-zfs.qcow2.xz
unxz -k vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2.xz
qemu-img resize vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2 +50G
qemu-system-x86_64 -m 16G -smp 16 -hda vm-images/FreeBSD-15.0-STABLE-amd64-zfs.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::10022-:22 -nographic -cpu host
```

Start the vm, press 3, and enter:
```
set console="comconsole"
boot
```

Inside the vm:
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

# set a proxy.
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

Configure nested vm:
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

Inside the nested bhyve vm:
```sh
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config

sysrc sshd_enable=YES
sysrc ifconfig_DEFAULT=DHCP
/etc/rc.d/sshd start

passwd # root
```

Do not stop the nested vm. In the host vm:
```sh
ssh-keygen -f nest.id_rsa -t rsa -N ''
ssh-copy-id -i ./nest.id_rsa.pub root@169.254.0.46 # you can connect to nest vm via nest.id_rsa now
```

Shutdown the nested vm now:
```sh
shutdown -p now
```

In the host vm, build the kernel:
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

Run the following commands on the host vm to install the kernel into the nested vm:
```sh
mdconfig -a -f /syzkaller/FreeBSD-15.0-STABLE-amd64.raw
mount /dev/md0p4 /mnt
cd /usr/src
make KERNCONF=SYZKALLER installkernel DESTDIR=/mnt
umount /mnt
mdconfig -d -u 0
```

Configure syzkaller:
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

Run syzkaller:
```sh
cd /root
syzkaller/bin/syz-manager -config=./freebsd.cfg
```

## Development environment setup (optional)

### neovim

(vm) Run the following commands to setup the neovim development environment:
```sh
ASSUME_ALWAYS_YES=true pkg install neovim rust ripgrep fd-find lazygit
cargo install tree-sitter-cli
echo "export PATH=$PATH:$HOME/.cargo/bin" >> $HOME/.shrc
exec sh
git clone https://github.com/Radon10043/nvimcfg /root/.config/nvim # my neovim config
```

### vscode + sshfs

(host) Use sshfs to mount a directory on the host, then develop with vscode or another tool you prefer:
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
    root@localhost:$SYZPILOT ./mnt/SyzPilot
```

## Build and replace the FreeBSD kernel on a Linux host

Building and replacing the FreeBSD kernel on a Linux host is possible, but it is less efficient than doing it directly on FreeBSD. This method is documented here in case it is needed later.

(host) Build the FreeBSD kernel:
```bash
cd $KERNSRC
cp $SYZPILOT/configs/kernel/freebsd.config sys/amd64/conf/SYZPILOT

mkdir build dist
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 buildworld
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 buildkernel KERNCONF=SYZPILOT
MAKEOBJDIRPREFIX=$PWD/build ./tools/build/make.py --cross-bindir=$LLVM_HOME/bin TARGET=amd64 TARGET_ARCH=amd64 installkernel KERNCONF=SYZPILOT DESTDIR=$PWD/dist
```

(host) Start the FreeBSD vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda target.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -daemonize -cpu host -display none   # start vm in daemon ...
```

(host) Install the built kernel into the vm:
```bash
cd $VMDIR
scp -P 3733 -r -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no $KERNSRC/build/15.0.0/dist/* root@localhost:/
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost reboot
ssh -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -i # output should be SYZPILOT
```

## Frequently used commands

Start the FreeBSD vm silently.
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

Start the vm.
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

ssh to the vm:
```bash
ssh -p 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    root@localhost
```

Copy file(s) to the vm:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```

Use sshfs to mount a directory:
```bash
mkdir -p mnt/SyzPilot
sshfs -p 3733 \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -o sftp_server=/usr/libexec/sftp-server \
    -o cache=yes \
    -o kernel_cache \
    -o compression=no \
    -o idmap=user \
    -o follow_symlinks \
    root@localhost:/root/SyzPilot ./mnt/SyzPilot
```

Unmount a directory mounted by sshfs:
```bash
fusermount -u mnt/SyzPilot
```
