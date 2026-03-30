# some notes for experiment

Please replace the following variables according to the actual situation:
- `$EXPERIMENT_ROOT`: directory for saving experiment artifacts.
- `$CLOUD`: directory for saving source of cloud.
- `$JOBS`: number of parallel jobs.

## setup experiment environment

prepare ~1T free space, start up a container based on cloud image. In container, run:
```bash
cd $EXPERIMENT_ROOT
mkdir kernel fuzzer images
```

### setup kernel/linux

```bash
mkdir $EXPERIMENT_ROOT/kernel/linux && cd $EXPERIMENT_ROOT/kernel/linux
git clone -b v6.18 --depth 1 https://github.com/gregkh v6.18
git clone -b v6.17.13 --depth 1 https://github.com/gregkh v6.17.13
git clone -b v6.12.63 --depth 1 https://github.com/gregkh v6.12.63
git clone -b v6.6.119 --depth 1 https://github.com/gregkh v6.6.119
git clone -b v6.1.159 --depth 1 https://github.com/gregkh v6.1.159
git clone -b v5.15.197 --depth 1 https://github.com/gregkh v5.15.197

cd $EXPERIMENT_ROOT/kernel/linux/v6.18 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.17.13 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.12.63 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.6.119 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.1.159 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v5.15.197 && cp $CLOUD/configs/kernel/linux.cfg && make CC=clang olddefconfig all -j$JOBS
```

### setup kernel/netbsd

```bash
cd $EXPERIMENT_ROOT
mkdir -p 15e7fbc5/src 98d2ae54/src 43eae48d/src

cd 15e7fbc5/src
git init .
git remote add origin https://github.com/NetBSD/src
git fetch --depth 1 origin 15e7fbc53d77cd7cc1d62511982b8972c4c0c421
git checkout 15e7fbc5
```

### setup image/linux

```bash
mkdir -p $EXPERIMENT_ROOT/images/Debian/bullseye && cd $EXPERIMENT_ROOT/images/Debian/bullseye
cp $CLOUD/scripts/linux/create-image.sh && chmod +x ./create-image.sh
./create-image.sh
```

### setup image/freebsd

(host) generate a sshkey:
```bash
cd $EXPERIMENT_ROOT
ssh-keygen -t rsa -f ./freebsd.id_rsa -N ""
```

#### 15.0.0

(host) download 15.0 image:
```bash
cd $EXPERIMENT_ROOT/images/freebsd
wget https://download.freebsd.org/snapshots/VM-IMAGES/15.0-STABLE/amd64/Latest/FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-15.0-STABLE-amd64-ufs.qcow2.xz
mv FreeBSD-15.0-STABLE-amd64-ufs.qcow2 15.0.0.qcow2
qemu-img resize 15.0.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input press 3 and input `set console="comconsole"` and `boot`
```

(vm) install freebsd kernel 15.0.0:
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

# you may need proxies
# echo "export http_proxy=http://10.0.2.2:7890" >> ~/.shrc
# echo "export https_proxy=http://10.0.2.2:7890" >> ~/.shrc
# exec sh
ASSUME_ALWAYS_YES=true pkg update -f
ASSUME_ALWAYS_YES=true pkg install bash curl gcc git gmake go golangci-lint llvm cmake
ASSUME_ALWAYS_YES=true pkg install vim dnsmasq wget tmux ccache pkgconf sqlite3 python3

cd /root
git clone https://github.com/Radon10043/cloud
git clone -b release/15.0.0 --depth 1 https://github.com/freebsd/freebsd-src 15.0.0
cd 15.0.0
cp /root/cloud/configs/kernel/freebsd.config sys/amd64/conf/CLOUD
cd sys/amd64/conf && config CLOUD
cd ../compile/CLOUD
make cleandepend && make depend
make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/images/freebsd
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   FreeBSD freebsd 15.0-RELEASE FreeBSD 15.0-RELEASE 7aedc8de6446 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
poweroff
```

#### 14.3.0

(host) download 14.X image:
```bash
cd $EXPERIMENT_ROOT/images/freebsd
wget https://download.freebsd.org/snapshots/VM-IMAGES/14.4-STABLE/amd64/Latest/FreeBSD-14.4-STABLE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-14.4-STABLE-amd64-ufs.qcow2.xz
mv FreeBSD-14.4-STABLE-amd64-ufs.qcow2 14.3.0.qcow2
qemu-img resize 14.3.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./14.3.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input press 3 and input `set console="comconsole"` and `boot`
```

(vm) install freebsd kernel 14.3.0:
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

# you may need proxies
# echo "export http_proxy=http://10.0.2.2:7890" >> ~/.shrc
# echo "export https_proxy=http://10.0.2.2:7890" >> ~/.shrc
# exec sh
ASSUME_ALWAYS_YES=true pkg update -f
ASSUME_ALWAYS_YES=true pkg install bash curl gcc git gmake go golangci-lint llvm cmake
ASSUME_ALWAYS_YES=true pkg install vim dnsmasq wget tmux ccache pkgconf sqlite3 python3

cd /root
git clone https://github.com/Radon10043/cloud
git clone -b release/14.3.0 --depth 1 https://github.com/freebsd/freebsd-src 14.3.0
cd 14.3.0
cp /root/cloud/configs/kernel/freebsd.config sys/amd64/conf/CLOUD
cd sys/amd64/conf && config CLOUD
cd ../compile/CLOUD
make cleandepend && make depend
make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/images/freebsd
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   FreeBSD freebsd 14.3-RELEASE FreeBSD 14.3-RELEASE 8c9ce319fef7 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
poweroff
```

#### 13.5.0

(host) start up vm to install freebsd kernel 13.5.0:
```bash
cd $EXPERIMENT_ROOT/images/freebsd
wget https://download.freebsd.org/snapshots/VM-IMAGES/13.5-STABLE/amd64/Latest/FreeBSD-13.5-STABLE-amd64.qcow2.xz
unxz -k FreeBSD-13.5-STABLE-amd64.qcow2.xz
mv FreeBSD-13.5-STABLE-amd64.qcow2 13.5.0.qcow2
qemu-img resize 13.5.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./13.5.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -bios /usr/share/ovmf/OVMF.fd
# press 3, input press 3 and input `set console="comconsole"` and `boot`
```

(vm):
```sh
chsh -s /bin/sh
exec sh
echo "-h" > /boot.config
echo "autoboot_delay=\"0\"" >> /boot/loader.conf
echo "console=\"comconsole\"" >> /boot/loader.conf
poweroff
```

(host) restart vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda ./13.5.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input press 3 and input `set console="comconsole"` and `boot`
```

(vm) install freebsd kernel 13.5.0:
```sh
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

cd /root
git clone https://github.com/Radon10043/cloud
git clone -b release/13.5.0 --depth 1 https://github.com/freebsd/freebsd-src 13.5.0
cd 13.5.0
cp /root/cloud/configs/kernel/freebsd.config sys/amd64/conf/CLOUD
cd sys/amd64/conf && config CLOUD
cd ../compile/CLOUD
make cleandepend && make depend
make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   FreeBSD freebsd 13.5-RELEASE FreeBSD 13.5-RELEASE 882b9f3f2 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```


#### build needed binaries

qemu, git clone, gmake, scp

### setup image/openbsd

(host) generate sshkey for openbsd images:
```bash
mkdir -p $EXPERIMENT/images/openbsd && cd $EXPERIMENT/images/openbsd
ssh-keygen -t rsa -f openbsd.id_rsa -N ''
```

#### 23290a22 (2025)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT/images/openbsd
wget https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/install78.iso
qemu-img create -f qcow2 2025-23290a22.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025-23290a22.qcow2,format=qcow2 -cdrom ./install78.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during openbsd, following contents are different with default:
```
Allow root ssh login? <yes>

Use (A)uto layout, (E)dit auto layout, or create (C)ustom layout? <c>
d a
d b
d d
d e
d f
d g
d h
d i
d j
d k

a b
<enter>
16G
<enter>

a a
<enter>
<enter>
<enter>
/

w
q

Location of set? <cd0>
Directory does not contain SHA256.sig. Continue without verification? <yes>
```

(vm) After installation complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025-23290a22.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) install openbsd kernel (version 23290a22, last version in 2025):
```sh
# if you need proxy, run following commands
# echo "export http_proxy=http://10.0.2.2:7890" >> /root/.profile
# echo "export https_proxy=http://10.0.2.2:7890" >> /root/.profile
# . ~/.profile

echo "https://mirrors.aliyun.com/openbsd/" > /etc/installurl
# vim: vim-9.1.1706-no_x11
# llvm: llvm-19.1.7p9
pkg_add wget bash curl git vim fastfetch llvm go gmake
# python: python-3.x
pkg_add ccache sqlite3 bear python py3-pip gdb cmake
pip3 install compiledb --break-system-packages
echo "export PATH=/root/go/bin:\$PATH" >> /root/.profile

git clone https://github.com/Radon10043/cloud

mkdir openbsd-23290a22 && cd openbsd-23290a22
git init .
git remote add origin https://github.com/openbsd/src
git fetch --depth 1 origin 23290a22d1dee9d1d0b277c2896d441128a32f42
git checkout 23290a22

cp /root/cloud/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/images/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   OpenBSD openbsd.my.domain 7.8 CLOUD#0 amd64 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```

#### 507b5b4 (2024)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT/images/openbsd
wget https://artfiles.org/openbsd/7.6/amd64/install76.iso
qemu-img create -f qcow2 2024-507b5b4.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2024-507b5b4.qcow2,format=qcow2 -cdrom ./install76.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during openbsd, following contents are different with default:
```
Allow root ssh login? <yes>

Use (A)uto layout, (E)dit auto layout, or create (C)ustom layout? <c>
d a
d b
d d
d e
d f
d g
d h
d i
d j
d k

a b
<enter>
16G
<enter>

a a
<enter>
<enter>
<enter>
/

w
q

Location of set? <cd0>
Directory does not contain SHA256.sig. Continue without verification? <yes>
```

(vm) After installation complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2024-507b5b4.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) install openbsd kernel (version 507b5b4, last version in 2024):
```sh
# if you need proxy, run following commands
# echo "export http_proxy=http://10.0.2.2:7890" >> /root/.profile
# echo "export https_proxy=http://10.0.2.2:7890" >> /root/.profile
# . ~/.profile

echo "https://mirrors.aliyun.com/openbsd/" > /etc/installurl
# vim: vim-9.1.1006-no_x11
# llvm: llvm-17.0.6p12
pkg_add wget bash curl git vim fastfetch llvm go gmake
# python: python-3.x
pkg_add ccache sqlite3 bear python py3-pip gdb cmake
pip3 install compiledb --break-system-packages
echo "export PATH=/root/go/bin:\$PATH" >> /root/.profile

git clone https://github.com/Radon10043/cloud

mkdir openbsd-507b5b4 && cd openbsd-507b5b4
git init .
git remote add origin https://github.com/openbsd/src
git fetch --depth 1 origin 507b5b4162b0d25e34b75b8e43070e2d6faf72e5
git checkout 507b5b4

cp /root/cloud/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/images/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   OpenBSD openbsd.my.domain 7.6 CLOUD#0 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```

#### 4dba83b8 (2023)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT/images/openbsd
wget https://artfiles.org/openbsd/7.4/amd64/install74.iso
qemu-img create -f qcow2 2023-4dba83b8.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2023-4dba83b8.qcow2,format=qcow2 -cdrom ./install74.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during openbsd, following contents are different with default:
```
Allow root ssh login? <yes>

Use (A)uto layout, (E)dit auto layout, or create (C)ustom layout? <c>
d a
d b
d d
d e
d f
d g
d h
d i
d j
d k

a b
<enter>
16G
<enter>

a a
<enter>
<enter>
<enter>
/

w
q

Location of set? <cd0>
Directory does not contain SHA256.sig. Continue without verification? <yes>
```

(vm) After installation complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2023-4dba83b8.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) install openbsd kernel (version 4dba83b8, last version in 2023):
```sh
# if you need proxy, run following commands
# echo "export http_proxy=http://10.0.2.2:7890" >> /root/.profile
# echo "export https_proxy=http://10.0.2.2:7890" >> /root/.profile
# . /root/.profile

echo "https://mirrors.aliyun.com/openbsd/" > /etc/installurl
# vim: vim-9.0.2035-no_x11
# llvm: llvm-16.0.6p8
pkg_add wget bash curl git vim fastfetch llvm go gmake
# python: python-3.x
pkg_add ccache sqlite3 bear python py3-pip gdb cmake
pip3 install compiledb --break-system-packages
echo "export PATH=/root/go/bin:\$PATH" >> /root/.profile

git clone https://github.com/Radon10043/cloud

mkdir openbsd-4dba83b8 && cd openbsd-4dba83b8
git init .
git remote add origin https://github.com/openbsd/src
git fetch --depth 1 origin 4dba83b83de21fd6727491f01e7db6c48cac59fc
git checkout 4dba83b8

cp /root/cloud/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/images/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of following command should like:
#   OpenBSD openbsd.my.domain 7.4 CLOUD#0 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```

### setup image/netbsd

(hsot) generate sshkey:
```bash
cd $EXPERIMENT_ROOT/images/netbsd
ssh-keygen -t rsa -f netbsd.id_rsa -N ""
```

#### 2026-15e7fbc5

(host) download iso file and setup vm:
```bash
cd $EXPERIMENT/images/netbsd
wget https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2026-15e7fbc5.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026-15e7fbc5.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompt to select bootblocks.

(host) after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026-15e7fbc5.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
```

(vm) setup environment of netbsd:
```bash
sed -i 's/timeout=5/timeout=1/' /boot.cfg

cat <<__EOF__ >> /etc/rc.conf

sshd=YES
dhcpcd=YES
__EOF__

cat <<__EOF__ >> /etc/ssh/sshd_config

Port 22
ListenAddress 0.0.0.0
PermitRootLogin yes
PermitRootLogin without-password
__EOF__

reboot
```

(host) copy sshkey and built kernel to vm:
```bash
cd $EXPERIMENT/images/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of uname command should like:
#   NetBSD  10.1_STABLE NetBSD 10.1_STABLE (CLOUD) #1: Sat Mar 28 20:59:39 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/15e7fbc5...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

#### 2024-98d2ae54

(host) download iso file and setup vm:
```bash
cd $EXPERIMENT/images/netbsd
wget https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2024-98d2ae54.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2024-98d2ae54.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompt to select bootblocks.

(host) after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2024-98d2ae54.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
```

(vm) setup environment of netbsd:
```bash
sed -i 's/timeout=5/timeout=1/' /boot.cfg

cat <<__EOF__ >> /etc/rc.conf

sshd=YES
dhcpcd=YES
__EOF__

cat <<__EOF__ >> /etc/ssh/sshd_config

Port 22
ListenAddress 0.0.0.0
PermitRootLogin yes
PermitRootLogin without-password
__EOF__

reboot
```

(host) copy sshkey and built kernel to vm:
```bash
cd $EXPERIMENT/images/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/98d2ae54/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of uname command should like:
#   NetBSD  10.99.12 NetBSD 10.99.12 (CLOUD) #0: Sat Mar 28 22:53:06 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/98d2ae54...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

#### 2023-43eae48d

(host) download iso file and setup vm:
```bash
cd $EXPERIMENT/images/netbsd
wget https://cdn.netbsd.org/pub/NetBSD/images/9.3/NetBSD-9.3-amd64.iso
qemu-img create -f qcow2 2023-43eae48d.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2023-43eae48d.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompt to select bootblocks.

(host) after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2023-43eae48d.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
```

(vm) setup environment of netbsd:
```bash
sed -i 's/timeout=5/timeout=1/' /boot.cfg

cat <<__EOF__ >> /etc/rc.conf

sshd=YES
dhcpcd=YES
__EOF__

cat <<__EOF__ >> /etc/ssh/sshd_config

Port 22
ListenAddress 0.0.0.0
PermitRootLogin yes
PermitRootLogin without-password
__EOF__

reboot
```

(host) copy sshkey and built kernel to vm:
```bash
cd $EXPERIMENT/images/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/43eae48d/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of uname command should like:
#   NetBSD  10.99.10 NetBSD 10.99.10 (CLOUD) #0: Sat Mar 28 21:44:46 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/43eae48d...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

### setup fuzzer/cloud

#### linux binaries

(host) build cloud/syzkaller for linux:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone --recurse-submodules https://github.com/Radon10043/cloud
# or git clone https://github.com/Radon10043/cloud && cd cloud && git submodule update --init --recursive

cd cloud/syzkaller
git apply \
    ../patch/syzkaller/linux.patch \
    ../patch/syzkaller/freebsd.patch \
    ../patch/syzkaller/openbsd.patch \
    ../patch/syzkaller/netbsd.patch
git apply \
    ../patch/specs#syzkaller-ac3c71e7#freebsd-15.0.0#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#linux-v6.18#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#netbsd-15e7fbc5#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#openbsd-23290a22#gemini-3-flash-preview.patch
make all -j16
```

#### freebsd binaries

(host) start a freebsd vm, let's add `-snapshot` so that we can do whatever we want on vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda $EXPERIMENT_ROOT/images/freebsd/15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -snapshot
```

(vm) build needed binaries for cloud on freebsd vm:
```sh
cd /root/cloud
git submodule update --init --recursive
cd syzkaller
git apply \
    ../patch/syzkaller/linux.patch \
    ../patch/syzkaller/freebsd.patch \
    ../patch/syzkaller/openbsd.patch \
    ../patch/syzkaller/netbsd.patch
git apply \
    ../patch/specs#syzkaller-ac3c71e7#freebsd-15.0.0#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#linux-v6.18#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#netbsd-15e7fbc5#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#openbsd-23290a22#gemini-3-flash-preview.patch
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/cloud/syzkaller/bin/freebsd_amd64 $EXPERIMENT_ROOT/fuzzer/cloud/syzkaller/bin
```

#### openbsd binaries

(host) setup a new vm for compiling:
```bash
qemu-img create -f qcow2 dev.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./dev.qcow2,format=qcow2 -cdrom ./install78.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during openbsd, following contents are different with default:
```
Allow root ssh login? <yes>

Use (A)uto layout, (E)dit auto layout, or create (C)ustom layout? <c>
d a
d b
d d
d e
d f
d g
d h
d i
d j
d k

a b
<enter>
16G
<enter>

a a
<enter>
<enter>
<enter>
/

w
q

Location of set? <cd0>
Directory does not contain SHA256.sig. Continue without verification? <yes>
```

(vm) After installation complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./dev.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) build needed binaries for cloud on freebsd vm:
```sh
# if you need proxy, run following commands
# echo "export http_proxy=http://10.0.2.2:7890" >> /root/.profile
# echo "export https_proxy=http://10.0.2.2:7890" >> /root/.profile
# . ~/.profile

echo "https://mirrors.aliyun.com/openbsd/" > /etc/installurl
# vim: vim-9.1.1706-no_x11
# llvm: llvm-19.1.7p9
pkg_add wget bash curl git vim fastfetch llvm go gmake
# python: python-3.x
pkg_add ccache sqlite3 bear python py3-pip gdb cmake
pip3 install compiledb --break-system-packages
echo "export PATH=/root/go/bin:\$PATH" >> /root/.profile

git clone https://github.com/radon10043/cloud && cd cloud
git submodule update --init --recursive

cd syzkaller
git apply \
    ../patch/syzkaller/linux.patch \
    ../patch/syzkaller/freebsd.patch \
    ../patch/syzkaller/openbsd.patch \
    ../patch/syzkaller/netbsd.patch
git apply \
    ../patch/specs#syzkaller-ac3c71e7#freebsd-15.0.0#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#linux-v6.18#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#netbsd-15e7fbc5#gemini-3-flash-preview.patch \
    ../patch/specs#syzkaller-ac3c71e7#openbsd-23290a22#gemini-3-flash-preview.patch
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/cloud/syzkaller/bin/openbsd_amd64 $EXPERIMENT_ROOT/fuzzer/cloud/syzkaller/bin
```

(vm) close vm:
```sh
shutdown -p now
```

#### netbsd binaries

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/cloud/syzkaller
make target TARGETOS=netbsd SOURCEDIR=$EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

### setup fuzzer/syzkaller

#### linux binaries

(host)
```bash
cd $EXPERIMENT/fuzzer/syzkaller
git apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/syzkaller/openbsd.patch $EXPERIMENT_ROOT/fuzzer/cloud/patch/syzkaller/netbsd.patch
make all
```

#### freebsd binaries

(host) start a freebsd vm, let's add `-snapshot` so that we can do whatever we want on vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda $EXPERIMENT_ROOT/images/freebsd/15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -snapshot
```

(vm) build needed binaries for cloud on freebsd vm:
```sh
cd /root
git clone https://github.com/google/syzkaller && cd syzkaller
git checkout ac3c71e7
git apply /root/cloud/patch/syzkaller/freebsd.patch /root/cloud/patch/syzkaller/openbsd.patch
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/syzkaller/bin/freebsd_amd64 $EXPERIMENT_ROOT/fuzzer/syzkaller/bin
```

(vm) poweroff:
```sh
poweroff
```

#### openbsd binaries

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=$EXPERIMENT_ROOT/images/openbsd/dev.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) build needed binaries for cloud on freebsd vm:
```sh
git clone https://github.com/google/syzkaller && cd syzkaller
git checkout ac3c71e7
git apply /root/cloud/patch/syzkaller/freebsd.patch /root/cloud/patch/syzkaller/openbsd.patch
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/syzkaller/bin/openbsd_amd64 $EXPERIMENT_ROOT/fuzzer/syzkaller/bin
```

(vm) close vm:
```sh
shutdown -p now
```

#### netbsd binaries

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/syzkaller
make target TARGETOS=netbsd SOURCEDIR=$EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

### setup fuzzer/KernelGPT

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/KernelGPT
git apply -3 $EXPERIMENT_ROOT/fuzzer/cloud/experiment/KernelGPT/repo.patch
git submodule update --init syzkaller-KernelGPT syzkaller-KernelGEM
git -C syzkaller-KernelGPT apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/KernelGPT/specs#linux-v6.7#gpt-4.patch
git -C syzkaller-KernelGEM apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/KernelGPT/specs#linux-v6.18#gemini-3-flash-preview.patch
make -C syzkaller-KernelGPT all && make -C syzkaller-KernelGEM all
```

[docs for generating specs via KernelGPT](./KernelGPT/README.md)

### setup fuzzer/KernelGEM

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/KernelGPT
git apply -3 $EXPERIMENT_ROOT/fuzzer/cloud/experiment/KernelGPT/repo.patch
git submodule update --init syzkaller-KernelGEM
git -C syzkaller-KernelGEM apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/KernelGPT/specs#linux-v6.18#gemini-3-flash-preview.patch
make -C syzkaller-KernelGEM all
```

[docs for generating specs via KernelGEM](./KernelGPT/README.md)

### setup fuzzer/SyzDescribe

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzDescribe
git apply -3 $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzDescribe/repo.patch
git submodule update --init --recursive
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzDescribe/specs#linux-v6.18.patch
make -C syzkaller all
```

[docs for generating specs via SyzDescribe](./SyzDescribe/README.md)

### setup fuzzer/SyzGenPlusPlus

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzGenPlusPlus
git apply -3 $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzGenPlusPlus/repo.patch
git submodule update --init --recursive
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzGenPlusPlus/specs#linux-v6.18.patch
make -C syzkaller all
```

[docs for generating specs via SyzGenPlusPlus](./SyzGenPlusPlus/README.md)

### setup fuzzer/SyzSpec

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzSpec
git apply -3 $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzSpec/repo.patch
git submodule update --init --recursive
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/cloud/experiment/SyzSpec/specs#linux-v6.18.patch
make -C syzkaller all
```

[docs for generating specs via SyzSpec](./SyzSpec/README.md)

### update docker-compose files

Finally, you can update docker-compose files :)
