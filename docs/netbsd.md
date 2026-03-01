# setup cloud and run fuzzing for netbsd kernel

this doc instruct to build and run cloud for netbsd kernel on linux host. Please replace following variables via your actual situation:
- `$CLOUD`: directory for saving cloud source.
- `$KERNDIR`: directory for saveing NetBSD kernel sources(s).
- `$VMDIR`: directory for saving NetBSD image(s).

## netbsd vm setup

download iso file and setup vm.
```bash
# run following commands on host
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

after installation complete, start vm.
```bash
# run following commands on host
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

run following commands to setup environment of netbsd vm.
```bash
# run following commands on vm
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

## cloud setup

build cloud and patch syzkaller.
```sh
# run following commands on host
git clone --recursive https://github.com/Radon10043/cloud

cd $CLOUD/syzkaller
git apply ../patch/syzkaller/generic.patch ../patch/syzkaller/netbsd.patch
cd ..
make all TARGETOS=netbsd
```

download source of netbsd kernel. I use 15e7fbc53d77cd7cc1d62511982b8972c4c0c421 to run cloud for specification generation, which can be built success on linux in 2026 early.
```sh
# run following commands on host
mkdir -p $KERNDIR/15e7fbc5
cd $KERNDIR/15e7fbc5
git clone https://github.com/NetBSD/src
git -C src checkout 15e7fbc5
cp -r src extract   # for const extraction debugging
```

build kernel, generate `compile_commands.json` and make it clean.
```bash
# run following commands on host
cd $KERNDIR/15e7fbc5/src
cp $CLOUD/configs/kernel/netbsd.config sys/arch/amd64/conf/CLOUD
./build.sh -j4 -m amd64 -c clang -U -T ../tools tools
./build.sh -j4 -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j4 -m amd64 -c clang -U -T ../tools -N 4 kernel=CLOUD | tee build.log
compiledb --parse build.log

# make compile_commands.json clean
jq 'map(select((.command // (.arguments | join(" "))) | test("mkdep") | not))' compile_commands.json > compile_commands_clean.json
```

analyze `compile_commands_clean.json`.
```bash
# run following commands on host
cd $CLOUD
./bin/analyzer \
    -i $KERNDIR/15e7fbc5/src/compile_commands_clean.json \
    -I $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/CLOUD \
    -o data/database/netbsd.db \
    -j 4
```

minimize tasks and generate variable list.
```bash
# run following commands on host
cd $CLOUD
# this may take a while ...
./bin/minitask \
    -db=./data/database/netbsd.db \
    -os=netbsd \
    -outdir=./workdir/minitask \
    -model=gemini-2.5-flash > logs/minitask.log 2>&1
./script/varlist.sh ./workdir/minitask > ./workdir/minitask/varlist.txt
```

generate syzlang specs on the basis of minimized tasks.
```bash
# run following commands on host
cd $CLOUD
./bin/generator \
    -db=./data/database/netbsd.db \
    -os=netbsd \
    -model=gemini-2.5-flash \
    -kernel=$KERNDIR/15e7fbc5/src \
    -outdir=./workdir/minitask \
    -varlist=./workdir/minitask/varlist.txt \
    -jobs=4 > logs/generate.log 2>&1
```

## syzkaller setup

generate a ssh key and copy it to vm.
```bash
# run following commands on host
cd $VMDIR
ssh-keygen -t rsa -f netbsd.id_rsa -N ""
```

copy content of `netbsd.id_tsa.pub` to `/root/.ssh/authorized_keys` on vm.

make sure vm can be connected via ssh key.
```bash
# run following commands on host
ssh -p 6382 \
    -i $VMDIR/netbsd.id_rsa \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no \
    root@localhost
```

copy new-built kernel to vm.
```bash
# run following commands on host
scp -P 6382 \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $KERNDIR/15e7fbc5/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

load kcov and vhci modules.
```sh
# run following commands on vm
cd /dev
sh MAKEDEV kcov vhci0 vhci1 vhci2 vhci3
```

feel free to run `poweroff` to shutdown vm.

build syzkaller after patching.
```bash
# run following commands on host
cd $CLOUD/syzkaller
make TARGETOS=netbsd SOURCEDIR=$KERNDIR/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```

## fuzzing netbsd kernel

duplicate a vm for fuzzing.
```bash
# run following commands on host
cp $VMDIR/dev.qcow2 $VMDIR/target.qcow2
```

wirte fuzzing config file and start fuzzing.
```bash
# run following commands on host
cd $CLOUD && mkdir workdir
cat <<__EOF__ > workdir/netbsd.cfg
{
	"name": "netbsd",
	"target": "netbsd/amd64",
	"http": ":10000",
	"workdir": "$CLOUD/workdir/out",
	"syzkaller": "$CLOUD/syzkaller",
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
