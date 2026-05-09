# setup SyzPilot and run fuzzing for netbsd kernel

this doc instruct to build and run SyzPilot for netbsd kernel on linux host. Please replace following variables via your actual situation:
- `$SYZPILOT`: directory for saving SyzPilot source.
- `$KERNDIR`: directory for saveing NetBSD kernel sources(s).
- `$VMDIR`: directory for saving NetBSD image(s).

## netbsd vm setup

(host) download iso file and setup vm.
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

during installation, select `use serial port com0` when prompt to select bootblocks.

(host) after installation complete, start vm.
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

(vm) run following commands to setup environment of netbsd vm.
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

(host) build SyzPilot and patch syzkaller.
```sh
mkdir $SYZPILOT
wget -O $SYZPILOT/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
cd $SYZPILOT && unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git apply ../patch/syzkaller/*
cd ..
make all TARGETOS=netbsd SOURCEDIR=$KERNSRC
```

(host) download source of netbsd kernel. I use 15e7fbc53d77cd7cc1d62511982b8972c4c0c421 to run SyzPilot for specification generation, which can be built success on linux in 2026 early.
```sh
mkdir -p $KERNDIR/15e7fbc5
cd $KERNDIR/15e7fbc5
git clone https://github.com/NetBSD/src
git -C src checkout 15e7fbc5
cp -r src extract   # for const extraction debugging
```

(host) build kernel, generate `compile_commands.json` and make it clean.
```bash
cd $KERNDIR/15e7fbc5/src
cp $SYZPILOT/configs/kernel/netbsd.config sys/arch/amd64/conf/SYZPILOT
./build.sh -j4 -m amd64 -c clang -U -T ../tools tools
./build.sh -j4 -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j4 -m amd64 -c clang -U -T ../tools -N 4 kernel=SYZPILOT | tee build.log
compiledb --parse build.log

# make compile_commands.json clean
jq 'map(select((.command // (.arguments | join(" "))) | test("mkdep") | not))' compile_commands.json > compile_commands_clean.json
```

(host) analyze `compile_commands_clean.json`.
```bash
cd $SYZPILOT
./bin/analyzer \
    -i $KERNDIR/15e7fbc5/src/compile_commands_clean.json \
    -I $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/SYZPILOT \
    -o data/database/netbsd.db \
    -j 4
```

(host) minimize tasks and generate references.
```bash
cd $SYZPILOT
# this may take a while ...
./bin/minitask \
    -db=./data/database/netbsd.db \
    -os=netbsd \
    -outdir=./workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
./script/reflist.sh ./workdir/minitask/specs > ./workdir/minitask/ref.txt
```

(host) generate syzlang specs on the basis of minimized tasks.
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

(host) generate a ssh key and copy it to vm.
```bash
cd $VMDIR
ssh-keygen -t rsa -f netbsd.id_rsa -N ""
```

copy content of `netbsd.id_tsa.pub` to `/root/.ssh/authorized_keys` on vm.

(host) make sure vm can be connected via ssh key.
```bash
ssh -p 6382 \
    -i $VMDIR/netbsd.id_rsa \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no \
    root@localhost
```

(host) copy new-built kernel to vm.
```bash
scp -P 6382 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/SYZPILOT/netbsd root@localhost:/netbsd
```

(vm) load kcov and vhci modules.
```sh
cd /dev
sh MAKEDEV kcov
```

feel free to run `poweroff` to shutdown vm.

(host) build syzkaller after patching.
```bash
cd $SYZPILOT/syzkaller
make TARGETOS=netbsd SOURCEDIR=$KERNDIR/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

## fuzzing netbsd kernel

(host) duplicate a vm for fuzzing.
```bash
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

(host) write fuzzing config file and start fuzzing.
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

## frequently used commands

start netbsd vm silently.
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

start vm.
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

copy file(s) to vm:
```bash
scp -P 6382 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $HOST_PATH root@localhost:$VM_PATH
```
