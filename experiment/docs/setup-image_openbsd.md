# setup image/openbsd

(host) generate sshkey for openbsd images:
```bash
mkdir -p $EXPERIMENT_ROOT/image/openbsd && cd $EXPERIMENT_ROOT/image/openbsd
ssh-keygen -t rsa -f openbsd.id_rsa -N ''
```

## 23290a22 (2025.12)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
wget https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/install78.iso
qemu-img create -f qcow2 2025-23290a22.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025-23290a22.qcow2,format=qcow2 -cdrom ./install78.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during installation, use the following non-default settings:
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

(vm) after installation is complete, reboot and press `CTRL-A`, `X` to shutdown vm.

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

(host) install sshkey and verify the kernel version:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   OpenBSD openbsd.my.domain 7.8 CLOUD#0 amd64 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```

## 6bf0f93a (2025.11)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
# wget https://artfiles.org/openbsd/7.8/amd64/install78.iso
qemu-img create -f qcow2 2025.11-6bf0f93a.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025.11-6bf0f93a.qcow2,format=qcow2 -cdrom ./install78.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during installation, use the following non-default settings:
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

(vm) after installation is complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025.11-6bf0f93a.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) install openbsd kernel (version 6bf0f93a, last version in 2025.11):
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

mkdir openbsd-6bf0f93a && cd openbsd-6bf0f93a
git init .
git remote add origin https://github.com/openbsd/src
git fetch --depth 1 origin 6bf0f93af4a8aa5d28d638525b1eb0c5b2f57941
git checkout 6bf0f93a

cp /root/cloud/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j16 && make install
reboot
```

(host) install sshkey and verify the kernel version:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   OpenBSD openbsd.my.domain 7.8 CLOUD#0 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```

## 6dac8606 (2025.10)

(host) download .iso file, init a qcow2 file:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
# wget https://artfiles.org/openbsd/7.4/amd64/install78.iso
qemu-img create -f qcow2 2025.10-6dac8606.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025.10-6dac8606.qcow2,format=qcow2 -cdrom ./install78.iso -boot d -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) input following commands (be quick!):
```
set tty com0
boot
```

(vm) during installation, use the following non-default settings:
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

(vm) after installation is complete, reboot and press `CTRL-A`, `X` to shutdown vm.

(host) start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./2025.10-6dac8606.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) install openbsd kernel (version 6dac8606, last version in 2025.10):
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

mkdir openbsd-6dac8606 && cd openbsd-6dac8606
git init .
git remote add origin https://github.com/openbsd/src
git fetch --depth 1 origin 6dac8606615b68ce13d259f805724b9d640096fa
git checkout 6dac8606

cp /root/cloud/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j16 && make install
reboot
```

(host) install sshkey and verify the kernel version:
```bash
cd $EXPERIMENT_ROOT/image/openbsd
ssh-copy-id -i ./openbsd.id_rsa.pub -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   OpenBSD openbsd.my.domain 7.4 CLOUD#0 amd64
ssh -i ./openbsd.id_rsa -p 6736 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
shutdown -p now
```
