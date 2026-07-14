# setup fuzzer/SyzPilot

## setup binaries for full kernel fuzzing

### linux binaries

(host) build SyzPilot/syzkaller for linux:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
make all -j16
```

### freebsd binaries

(host) start a freebsd vm, let's add `-snapshot` so that we can do whatever we want on vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda $EXPERIMENT_ROOT/images/freebsd/15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -snapshot
```

(vm) build needed binaries for SyzPilot on freebsd vm:
```sh
cd /root/SyzPilot/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/SyzPilot/syzkaller/bin/freebsd_amd64 $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller/bin
```

### openbsd binaries

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

(vm) during openbsd installation, use the following non-default settings:
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
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./dev.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) build needed binaries on the OpenBSD vm:
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

mkdir /root/SyzPilot
wget -O /root/SyzPilot/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
cd /root/SyzPilot && unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219

git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/SyzPilot/syzkaller/bin/openbsd_amd64 $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller/bin
```

(vm) close vm:
```sh
shutdown -p now
```

### netbsd binaries

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
make target TARGETOS=netbsd SOURCEDIR=$EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

### fuzzer/SyzPilot/bin-kern

(host) move full kernel fuzzing binaries to SyzPilot/:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot
mv syzkaller/bin bin-kern
```

## setup binaries for subsystem fuzzing

### linux binaries

(host) build SyzPilot/syzkaller for linux subsystems:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
git checkout . && git clean -fdx
git apply ../patch/syzkaller/*
git apply ../patch/specs-subsys/*
make all -j16
```

### freebsd binaries

(host) start a freebsd vm, let's add `-snapshot` so that we can do whatever we want on vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda $EXPERIMENT_ROOT/images/freebsd/15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -snapshot
```

(vm) build needed binaries for SyzPilot on freebsd vm:
```sh
cd /root/SyzPilot/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-subsys/*
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 3733 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/SyzPilot/syzkaller/bin/freebsd_amd64 $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller/bin
```

### openbsd binaries

(host) start up vm with `-snapshot` to avoid modifying its disk image:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=./dev.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic -snapshot
```

(vm) build the needed binaries on the OpenBSD vm:
```sh
cd /root/SyzPilot/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-subsys/*
gmake target
```

(host) copy needed binaries to the host:
```bash
scp -P 6736 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    -r \
    root@localhost:/root/SyzPilot/syzkaller/bin/openbsd_amd64 $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller/bin
```

(vm) close vm:
```sh
shutdown -p now
```

### netbsd binaries

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
make target TARGETOS=netbsd SOURCEDIR=$EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5-tprof CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

### fuzzer/SyzPilot/bin-subsys

(host) move subsystem fuzzing binaries to SyzPilot/:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot
mv syzkaller/bin bin-subsys
```

## setup binaries for ablation fuzzing

### nodb variant

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
git checkout . && git clean -fdx
git apply -3 ../patch/syzkaller/*
git apply ../patch/specs-ablation/specs-syzkaller-ac3c71e7-linux-v6.18-subsystem-nodb-gemini-3-flash-preview.patch
make all -j$JOBS
mv bin ../bin-nodb
```

### noiter variant

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
git checkout . && git clean -fdx
git apply -3 ../patch/syzkaller/*
git apply ../patch/specs-ablation/specs-syzkaller-ac3c71e7-linux-v6.18-subsystem-noiter-gemini-3-flash-preview.patch
make all -j$JOBS
mv bin ../bin-noiter
```

### openllm variant

(host):
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot/syzkaller
git checkout . && git clean -fdx
git apply -3 ../patch/syzkaller/*
git apply ../patch/specs-ablation/specs-syzkaller-ac3c71e7-linux-v6.18-subsystem-openllm-qwen3-235b-a22b-instruct-2507.patch
make all -j$JOBS
mv bin ../bin-openllm
```
