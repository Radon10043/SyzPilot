# setup SyzPilot and run fuzzing for openbsd kernel

Please replace the following variables according to the actual situation:
- `$VMDIR`: directory for saving OpenBSD image(s).
- `$KERNSRC`: directory for saving OpenBSD kernel source.
- `$CLOUD`: directory for saving SyzPilot source.
- `$WORKDIR`: directory for working.

## openbsd vm setup

Download OpenBSD image from [https://www.openbsd.org/faq/faq4.html#Download](https://www.openbsd.org/faq/faq4.html#Download), I use `install78.iso`.
```bash
# run following commands on host
cd $VMDIR
wget https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/install78.iso
qemu-img create -f qcow2 dev.qcow2 200G
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -drive file=vm/dev.qcow2,format=qcow2 \
    -cdrom vm/install78.iso \
    -boot d \
    -nic user,model=virtio,hostfwd=tcp::6736-:22 \
    -nographic
```

in vm, input following contents (be quick!):
```
set tty com0
boot
```

during openbsd installation, following contents are different with default:
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

After installation complete, shutdown vm and boot it via:
```bash
# run following commands on the host
# both username and password are root
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -drive file=vm/dev.qcow2,format=qcow2 \
    -nic user,model=virtio,hostfwd=tcp::6736-:22 \
    -nographic
```

run following commands for basic package installation:
```sh
# run following commands on vm

# if you need proxy, uncomment following commands
# echo "export http_proxy=http://10.0.2.2:7890" >> /root/.profile
# echo "export https_proxy=http://10.0.2.2:7890" >> /root/.profile
# . ~/.profile

echo "https://mirrors.aliyun.com/openbsd/" > /etc/installurl
# vim: vim-9.1.1706-no_x11
# llvm: llvm-19.1.7p9
pkg_add wget bash curl git vim fastfetch llvm go gmake gcc
pkg_add ccache sqlite3 bear python py3-pip gdb cmake
pip3 install compiledb --break-system-packages

echo "export PATH=/root/go/bin:\$PATH" >> /root/.profile

wget https://github.com/google/flatbuffers/archive/refs/tags/v23.5.26.zip
tar -xzvf v23.5.26.tar.gz
cd flatbuffers-23.5.26
cmake . && make -j4 && make install
cd ..
rm -rf v23.5.26.tar.gz flatbuffers-23.5.26

ln -s /usr/local/bin/clang-format-19 /usr/local/bin/clang-format

rcctl -f start vmd
rcctl enable vmd
```

## SyzPilot setup

SyzPilot must be setup under openbsd environment, let's download it first:
```sh
# run following commands on vm
cd /root
git clone --recurse-submodules https://github.com/Radon10043/cloud
# if you forgot to clone with --recurse-submodules, run `git submodule update --init --recursive` under cloud directory to update submodules
```

build SyzPilot:
```sh
# run following commands on vm
cd $CLOUD
LLVM_CONFIG=llvm-config-19 gmake
```

download source of OpenBSD and checkout to `23290a22`, which is the latest version in 2025:
```sh
# run following commands on vm
mkdir -p $KERNSRC
git clone https://github.com/openbsd/src $KERNSRC
cd $KERNSRC
# the latest version in 2025
git checkout 23290a22d1dee9d1d0b277c2896d441128a32f42
```

build kernel and generate `compile_commands.json`:
```sh
# run following commands on vm
cp $CLOUD/configs/kernel/openbsd.config $KERNSRC/sys/arch/amd64/conf/CLOUD
cd $KERNSRC/sys/arch/amd64/conf
config CLOUD
cd ../compile/CLOUD
make depend
make -j4 | tee make.log

compiledb --parse make.log
```

construct database:
```sh
# run following commands on vm
cd $CLOUD
LD_LIBRARY_PATH=/usr/local/llvm19/lib:$LD_LIBRARY_PATH ./bin/analyzer -i $KERNSRC/sys/arch/amd64/compile/CLOUD/compile_commands.json -j 8 -o data/database/openbsd.db
```

feel free to run minitask or generator:
```sh
# run following commands on vm
# minitask
$CLOUD/bin/minitask \
    -db=$CLOUD/data/database/openbsd.db \
    -os=openbsd \
    -outdir=$CLOUD/workdir/minitask \
    -model=gemini-2.5-flash > logs/minitask.log 2>&1

# generator
$CLOUD/bin/generator \
    -db=$CLOUD/data/database/openbsd.db \
    -os=openbsd \
    -outdir=$CLOUD/workdir/out \
    -kernel=$KERNSRC \
    -model=gemini-2.5-flash \
    -varlist=$CLOUD/workdir/out/varlist.txt \
    -jobs=4 > logs/generate.log 2>&1
```

## syzkaller setup

### ubuntu host, qemu vm

**NOTE: you can't symbolize openbsd crash reports on linux, but you can copy report to openbsd to symbolize it.**

duplicate a vm as the fuzz target.
```bash
# run following commands on host
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

start and setup fuzz target vm.
```bash
# run following commands on host
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -drive file=vm/dev.qcow2,format=qcow2 \
    -cdrom vm/install78.iso \
    -boot d \
    -nic user,model=virtio,hostfwd=tcp::6736-:22 \
    -nographic
```

generate and copy ssh key to target vm.
```bash
# run following commands on host
cd $VMDIR
ssh-keygen -t rsa -f openbsd.id_rsa -N ''
ssh-copy-id \
    -p 6736 \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no \
    -i ./openbsd.id_rsa \
    root@localhost
```

install target version of kernel in fuzz target vm.
```sh
# run following commands on fuzz target vm
cd $KERNSRC/sys/arch/amd64/compile/CLOUD
make install
shutdown -p now
```

run syzkaller on ubuntu host and fuzz openbsd kernel with qemu vm.
```sh
# run following comands on host
cd $WORKDIR
cat <<__EOF__ > test.cfg
{
    "name": "openbsd",
    "target": "openbsd/amd64",
    "http": ":10000",
    "workdir": "$WORKDIR/out",
    "syzkaller": "$CLOUD/syzkaller",
    "image": "$VM/target.qcow2",
    "sshkey": "$VM/openbsd.id_rsa",
    "sandbox": "none",
    "procs": 8,
    "type": "qemu",
    "vm": {
        "count": 1,
        "cpu": 2,
        "mem": 2048
    }
}
__EOF__

git clone https://github.com/openbsd/src openbsd
$CLOUD/syzkaller/bin/syz-manager -config=$WORKDIR/test.cfg
```

### openbsd host, openbsd vm

setup a nest vm for fuzzing.
```sh
# run following commands on vm
cat <<__EOF__ > /etc/vm.conf
vm "syzkaller" {
  disable
  disk "/dev/null"
  local interface
  owner root
  allow instance { boot, disk, memory }
}
__EOF__

cat <<EOF > /sys/arch/amd64/conf/SYZKALLER
include "arch/amd64/conf/GENERIC"
pseudo-device kcov 1
EOF

cd /sys/arch/amd64/conf/SYZKALLER
config SYZKALLER
cd ../compile/SYZKALLER
make

wget https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/install78.iso
vmctl create -s 10G /root/vm.qcow2
vmctl start -c -d /root/vm.qcow2 -r install78.iso install_vm
# install vm ...
# Some settings diffs from default:
#   Allow root ssh login? (yes, no, prohibit-password): yes
# After setup finish, run following commands in nest vm:
#   ifconfig vio0 autoconf
#   echo "inet autoconf" > /etc/hostname.vio0
#   sh /etc/netstart vio0
#   shutdown -p now

# run vm sliently
vmctl start -t syzkaller -d "/root/vm.qcow2" syzkaller-1
ssh-keygen -f vm.sshkey
ssh "root@100.64.2.3" 'cat >~/.ssh/authorized_keys' <vm.sshkey.pub
ssh "root@100.64.2.3" 'echo library_aslr=NO >>/etc/rc.conf.local'   # disable ASLR to improve boot time
vmctl stop -w syzkaller-1
```

Now we can start fuzzing:
```bash
cd $CLOUD && mkdir workdir
cat <<__EOF__ > workdir/test.cfg
{
  "name": "openbsd",
  "target": "openbsd/amd64",
  "http": ":10000",
  "workdir": "$CLOUD/workdir/out",
  "kernel_obj": "/sys/arch/amd64/compile/SYZKALLER/obj",
  "kernel_src": "/",
  "syzkaller": "$CLOUD/syzkaller",
  "image": "/root/vm.qcow2",
  "sshkey": "/root/vm.sshkey",
  "sandbox": "none",
  "procs": 2,
  "type": "vmm",
  "vm": {
    "count": 1,
    "mem": 512,
    "kernel": "/sys/arch/amd64/compile/SYZKALLER/obj/bsd",
    "template": "syzkaller"
  }
}
__EOF__
./syzkaller/bin/syz-manager -config=./workdir/test.cfg
```

## development environment setup (optional)

you can use neovim on vm or vscode+sshfs on host, I use the latter.

### neovim

```sh
# run following commands on vm
pkg_add neovim rust ripgrep

env LIBCLANG_PATH=/usr/local/llvm19/lib/libclang.so.0.0 cargo install tree-sitter-cli
echo "export PATH=/root/.cargo/bin:\$PATH" >> /root/.profile

# lazygit
git clone https://github.com/jesseduffield/lazygit.git
cd lazygit
go install
cd ..

# my neovim config
mkdir -p /root/.config/nvim
git clone https://github.com/Radon10043/nvimcfg /root/.config/nvim
```

### vscode+sshfs

use following commands to mount file or directory in the vm:
```bash
# run following commands on host
sshfs -p 6736 \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -o sftp_server=/usr/libexec/sftp-server \
    -o cache=yes \
    -o kernel_cache \
    -o compression=no \
    -o idmap=user \
    -o follow_symlinks \
    root@localhost:$CLOUD ./syzpilot
```

feel free to unmount it:
```bash
# run following commands on host
fusermount -u ./syzpilot
```

## fuzzing latest kernel

> [!CAUTION]
> If a new kernel is installed with an old user-space, the image may broken! please upgrade use-space and kernel-space first then install the customized kernel for fuzzing.

upgrade user-space and kernel-space to the latest snapshot:
```bash
sysupgrade -s
```

compile and install customized latest OpenBSD kernel.
```bash
cd $KERNSRC && git pull
cp $CLOUD/configs/kernel/openbsd.config sys/arch/amd64/conf/CLOUD
cd sys/arch/amd64/conf && config CLOUD
cd ../compile/CLOUD
make depend && make -j4 && make install
```

then we can run fuzzing :)

## skills

Press `~`+`~`+`.` to exit nest VM.

run vm silently:
```bash
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -drive file=vm/dev.qcow2,format=qcow2 \
    -nic user,model=virtio,hostfwd=tcp::6736-:22 \
    -display none \
    -daemonize
```

start vm.
```bash
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -drive file=vm/dev.qcow2,format=qcow2 \
    -nic user,model=virtio,hostfwd=tcp::6736-:22 \
    -display none \
    -nographic
```

ssh to openbsd vm:
```bash
ssh -p 6736 \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    root@localhost
```

copy file(s) to vm:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```

use sshfs to mount directory:
```bash
sshfs -p 6736 \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -o sftp_server=/usr/libexec/sftp-server \
    -o cache=yes \
    -o kernel_cache \
    -o compression=no \
    -o idmap=user \
    -o follow_symlinks \
    root@localhost:/root/cloud ./mnt/syzpilot
```