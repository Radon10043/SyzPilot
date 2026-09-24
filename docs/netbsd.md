# Setup SyzPilot and run fuzzing for the NetBSD kernel

This document explains how to build and run SyzPilot for the NetBSD kernel on a Linux host. Please replace the following variables according to your environment:
- `$SYZPILOT`: directory for saving the SyzPilot source code.
- `$KERNDIR`: directory for saving the NetBSD kernel source code.
- `$VMDIR`: directory for saving NetBSD image(s).

## Setup the NetBSD vm

(host) Download the ISO file and setup the vm.
```bash
wget -P $VMDIR https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 $VMDIR/dev.qcow2 200G
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -hda $VMDIR/dev.qcow2 \
    -cdrom $VMDIR/NetBSD-10.1-amd64.iso \
    -boot d \
    -net nic,model=virtio \
    -net user,hostfwd=tcp::6382-:22 \
    -display curses
```

During installation, select `use serial port com0` when prompted to select bootblocks.

(host) After installation completes, start the vm.
```bash
qemu-system-x86_64 \
    -enable-kvm \
    -m 16G \
    -smp 16 \
    -cpu host \
    -hda $VMDIR/dev.qcow2 \
    -net nic,model=virtio \
    -net user,hostfwd=tcp::6382-:22 \
    -device virtio-rng-pci \
    -nographic
```

(vm) Run the following commands to setup the NetBSD vm environment.
```bash
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

## SyzPilot setup

(host) Build SyzPilot and patch syzkaller.
```sh
git clone --recursive https://github.com/Radon10043/SyzPilot $SYZPILOT
cd $SYZPILOT/syzkaller
git apply -3 ../patch/syzkaller/*
git reset .
cd ..
make all TARGETOS=netbsd SOURCEDIR=$KERNSRC
```

(host) Download the NetBSD kernel source code. This guide uses `15e7fbc53d77cd7cc1d62511982b8972c4c0c421` to run SyzPilot for specification generation, because it can be built successfully on Linux in early 2026.
```sh
mkdir -p $KERNDIR/15e7fbc5
cd $KERNDIR/15e7fbc5
git clone https://github.com/NetBSD/src
git -C src checkout 15e7fbc5
cp -r src extract   # for const extraction debugging
```

(host) Build the kernel, generate `compile_commands.json`, and clean it up.
```bash
cd $KERNDIR/15e7fbc5/src
cp $SYZPILOT/configs/kernel/netbsd.config sys/arch/amd64/conf/SYZPILOT
./build.sh -j4 -m amd64 -c clang -U -T ../tools tools
./build.sh -j4 -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j4 -m amd64 -c clang -U -T ../tools -N 4 kernel=SYZPILOT | tee build.log
compiledb --parse build.log

# Clean up compile_commands.json.
jq 'map(select((.command // (.arguments | join(" "))) | test("mkdep") | not))' compile_commands.json > compile_commands_clean.json
```

(host) Analyze `compile_commands_clean.json`.
```bash
cd $SYZPILOT
./bin/analyzer \
    -i $KERNDIR/15e7fbc5/src/compile_commands_clean.json \
    -I $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/SYZPILOT \
    -o data/database/netbsd.db \
    -j 4
```

(host) Minimize tasks and generate references.
```bash
cd $SYZPILOT
# this may take a while ...
./bin/minitask \
    -db=./data/database/netbsd.db \
    -os=netbsd \
    -outdir=./workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
./scripts/reflist.sh ./workdir/minitask > ./workdir/minitask/ref.txt
```

(host) Generate syzlang specs based on the minimized tasks.
```bash
cd $SYZPILOT
./bin/generator \
    -db=./data/database/netbsd.db \
    -os=netbsd \
    -model=gemini-3-flash-preview \
    -kernel=$KERNDIR/15e7fbc5/src \
    -outdir=./workdir/minitask \
    -ref=./workdir/minitask/ref.txt \
    -jobs=4 > logs/generate.log 2>&1
```

## syzkaller setup

(host) Generate an ssh key and copy it to the vm.
```bash
cd $VMDIR
ssh-keygen -t rsa -f netbsd.id_rsa -N ""
```

Copy the contents of `netbsd.id_rsa.pub` to `/root/.ssh/authorized_keys` on the vm.

(host) Make sure the vm can be accessed with the ssh key.
```bash
ssh -p 6382 \
    -i $VMDIR/netbsd.id_rsa \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no \
    root@localhost
```

(host) Copy the newly built kernel to the vm.
```bash
scp -P 6382 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/SYZPILOT/netbsd root@localhost:/netbsd
```

(vm) Load the `kcov` and `vhci` modules.
```sh
cd /dev
sh MAKEDEV kcov
```

Run `poweroff` when you want to shut down the vm.

(host) Build syzkaller after patching.
```bash
cd $SYZPILOT/syzkaller
make TARGETOS=netbsd SOURCEDIR=$KERNDIR/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

## Fuzzing the NetBSD kernel

(host) Duplicate the vm for fuzzing.
```bash
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

(host) Write the fuzzing configuration file and start fuzzing.
```bash
cd $SYZPILOT && mkdir workdir
cat <<__EOF__ > workdir/netbsd.cfg
{
	"name": "netbsd",
	"target": "netbsd/amd64",
	"http": ":10000",
	"workdir": "$SYZPILOT/workdir/out",
	"syzkaller": "$SYZPILOT/syzkaller",
	"image": "$VMDIR/target.qcow2",
	"sshkey": "$VMDIR/netbsd.id_rsa",
	"sandbox": "none",
	"procs": 2,
	"type": "qemu",
	"vm": {
		"count": 1,
		"cpu": 2,
		"mem": 2048
	}
}
__EOF__
./syzkaller/bin/syz-manager -config=./workdir/netbsd.cfg
```

## Frequently used commands

Start the NetBSD vm silently.
```bash
qemu-system-x86_64 \
    -enable-kvm \
    -m 8G \
    -smp 4 \
    -cpu host \
    -hda $VMDIR/dev.qcow2 \
    -net nic,model=virtio \
    -net user,hostfwd=tcp::6382-:22 \
    -display none \
    -daemonize
```

Start the vm.
```bash
qemu-system-x86_64 \
    -enable-kvm \
    -m 8G \
    -smp 4 \
    -cpu host \
    -hda $VMDIR/dev.qcow2 \
    -net nic,model=virtio \
    -net user,hostfwd=tcp::6382-:22 \
    -nographic
```

Copy file(s) to the vm:
```bash
scp -P 6382 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```
