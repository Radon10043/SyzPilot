# Setup SyzPilot and run fuzzing for the OpenBSD kernel

Please replace the following variables according to your environment:
- `$VMDIR`: directory for saving OpenBSD image(s).
- `$KERNSRC`: directory for saving the OpenBSD kernel source code.
- `$SYZPILOT`: directory for saving the SyzPilot source code.
- `$WORKDIR`: working directory.

## Setup OpenBSD VM

(host) Download the OpenBSD image from [https://www.openbsd.org/faq/faq4.html#Download](https://www.openbsd.org/faq/faq4.html#Download). This guide uses `install78.iso`.
```bash
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

(VM) Enter the following commands quickly:
```
set tty com0
boot
```

(VM) During OpenBSD installation, use the following values where they differ from the defaults:
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

(host) After installation completes, shut down the VM and boot it:
```bash
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

(VM) Run the following commands to install the basic packages:
```sh
# if you need a proxy, uncomment the following commands.
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

(VM) SyzPilot must be set up in an OpenBSD environment. Download it first:
```sh
mkdir /root/SyzPilot
wget -O /root/SyzPilot/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
cd /root/SyzPilot && unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
```

(VM) Build SyzPilot:
```sh
cd $SYZPILOT
LLVM_CONFIG=llvm-config-19 gmake
```

(VM) Download the OpenBSD source code and check out `23290a22`, which is the latest version used in 2025:
```sh
mkdir -p $KERNSRC
git clone https://github.com/openbsd/src $KERNSRC
cd $KERNSRC
# the latest version in 2025
git checkout 23290a22d1dee9d1d0b277c2896d441128a32f42
```

(VM) Build the kernel and generate `compile_commands.json`:
```sh
cp $SYZPILOT/configs/kernel/openbsd.config $KERNSRC/sys/arch/amd64/conf/SYZPILOT
cd $KERNSRC/sys/arch/amd64/conf
config SYZPILOT
cd ../compile/SYZPILOT
make depend
make -j4 | tee make.log

compiledb --parse make.log
```

(VM) Construct the database:
```sh
cd $SYZPILOT
LD_LIBRARY_PATH=/usr/local/llvm19/lib:$LD_LIBRARY_PATH ./bin/analyzer -i $KERNSRC/sys/arch/amd64/compile/SYZPILOT/compile_commands.json -j 8 -o data/database/openbsd.db
```

(VM) You can now run `minitask` or `generator`:
```sh
# minitask
$SYZPILOT/bin/minitask \
    -db=$SYZPILOT/data/database/openbsd.db \
    -os=openbsd \
    -outdir=$SYZPILOT/workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
$SYZPILOT/scripts/reflist.sh $SYZPILOT/workdir/minitask/specs > workdir/minitask/ref.txt

# generator
$SYZPILOT/bin/generator \
    -db=$SYZPILOT/data/database/openbsd.db \
    -os=openbsd \
    -outdir=$SYZPILOT/workdir/minitask \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -ref=$SYZPILOT/workdir/minitask/ref.txt \
    -jobs=4 > logs/generate.log 2>&1
```

## syzkaller setup

### Ubuntu host, QEMU VM

**NOTE:** You cannot symbolize OpenBSD crash reports on Linux, but you can copy the reports to OpenBSD and symbolize them there.

(host) Duplicate the VM to use as the fuzzing target.
```bash
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

(host) Start and set up the fuzzing target VM.
```bash
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

(host) Generate an SSH key and copy it to the target VM.
```bash
cd $VMDIR
ssh-keygen -t rsa -f openbsd.id_rsa -N ''
ssh-copy-id \
    -p 6736 \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no \
    -i ./openbsd.id_rsa \
    root@localhost
```

(VM) Install the target kernel version in the fuzzing target VM.
```sh
cd $KERNSRC/sys/arch/amd64/compile/SYZPILOT
make install
shutdown -p now
```

(host) Run syzkaller on the Ubuntu host and fuzz the OpenBSD kernel with the QEMU VM.
```sh
cd $WORKDIR
cat <<__EOF__ > test.cfg
{
    "name": "openbsd",
    "target": "openbsd/amd64",
    "http": ":10000",
    "workdir": "$WORKDIR/out",
    "syzkaller": "$SYZPILOT/syzkaller",
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

$SYZPILOT/syzkaller/bin/syz-manager -config=$WORKDIR/test.cfg
```

### OpenBSD host, OpenBSD VM

Setup a nested VM for fuzzing.
```sh
# run the following commands on the VM.
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
# install the VM ...
# some settings differ from the defaults:
#   Allow root ssh login? (yes, no, prohibit-password): yes
# after setup finishes, run the following commands in the nested VM:
#   ifconfig vio0 autoconf
#   echo "inet autoconf" > /etc/hostname.vio0
#   sh /etc/netstart vio0
#   shutdown -p now

# Run the VM silently.
vmctl start -t syzkaller -d "/root/vm.qcow2" syzkaller-1
ssh-keygen -f vm.sshkey
ssh "root@100.64.2.3" 'cat >~/.ssh/authorized_keys' <vm.sshkey.pub
ssh "root@100.64.2.3" 'echo library_aslr=NO >>/etc/rc.conf.local'   # disable ASLR to improve boot time
vmctl stop -w syzkaller-1
```

Now we can start fuzzing:
```bash
cd $SYZPILOT && mkdir workdir
cat <<__EOF__ > workdir/test.cfg
{
  "name": "openbsd",
  "target": "openbsd/amd64",
  "http": ":10000",
  "workdir": "$SYZPILOT/workdir/out",
  "kernel_obj": "/sys/arch/amd64/compile/SYZKALLER/obj",
  "kernel_src": "/",
  "syzkaller": "$SYZPILOT/syzkaller",
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

## Setup development environment (optional)

You can use Neovim on the VM or VS Code with SSHFS on the host. This guide uses the latter.

### Neovim

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
```

### VSCode + SSHFS

Use the following commands to mount a file or directory from the VM:
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
    root@localhost:$SYZPILOT ./SyzPilot
```

Unmount it when needed:
```bash
# run the following commands on the host.
fusermount -u ./SyzPilot
```

## Fuzzing the latest kernel

**CAUTION:** If a new kernel is installed with an old user space, the image may break. Please upgrade both user space and kernel space before installing the customized kernel for fuzzing.

Upgrade user space and kernel space to the latest snapshot:
```bash
sysupgrade -s
```

Compile and install the customized latest OpenBSD kernel:
```bash
cd $KERNSRC && git pull
cp $SYZPILOT/configs/kernel/openbsd.config sys/arch/amd64/conf/SYZPILOT
cd sys/arch/amd64/conf && config SYZPILOT
cd ../compile/SYZPILOT
make depend && make -j4 && make install
```

Then you can run fuzzing :)

## Frequently used commands

Press `~`+`~`+`.` to exit nest VM.

Run the VM silently:
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

Start the VM:
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

SSH to the OpenBSD VM:
```bash
ssh -p 6736 \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    root@localhost
```

Copy file(s) to the VM:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```

Use SSHFS to mount a directory:
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
    root@localhost:/root/SyzPilot ./mnt/SyzPilot
```
