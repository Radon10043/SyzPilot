# setup fuzzer/syzkaller

## linux binaries

(container.syzpilot):
```bash
cd $EXPERIMENT_ROOT/fuzzer/syzkaller
git apply $EXPERIMENT_ROOT/fuzzer/cloud/patch/syzkaller/openbsd.patch $EXPERIMENT_ROOT/fuzzer/cloud/patch/syzkaller/netbsd.patch
make all
```

## freebsd binaries

(container.syzpilot): start a freebsd vm, let's add `-snapshot` so that we can do whatever we want on vm:
```bash
qemu-system-x86_64 -m 16G -smp 16 -hda $EXPERIMENT_ROOT/image/freebsd/15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host -snapshot
```

(vm) build needed binaries for SyzPilot on freebsd vm:
```sh
cd /root
git clone https://github.com/google/syzkaller && cd syzkaller
git checkout ac3c71e7
git apply /root/cloud/patch/syzkaller/freebsd.patch /root/cloud/patch/syzkaller/openbsd.patch
gmake target
```

(container.syzpilot): copy needed binaries to the host:
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

## openbsd binaries

(container.syzpilot): start up vm:
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -drive file=$EXPERIMENT_ROOT/image/openbsd/dev.qcow2,format=qcow2 -nic user,model=virtio,hostfwd=tcp::6736-:22 -nographic
```

(vm) build the needed binaries on openbsd vm:
```sh
git clone https://github.com/google/syzkaller && cd syzkaller
git checkout ac3c71e7
git apply /root/cloud/patch/syzkaller/freebsd.patch /root/cloud/patch/syzkaller/openbsd.patch
gmake target
```

(container.syzpilot): copy needed binaries to the host:
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

## netbsd binaries

(container.syzpilot)::
```bash
cd $EXPERIMENT_ROOT/fuzzer/syzkaller
make target TARGETOS=netbsd SOURCEDIR=$EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5 CCFLAGS="-static-libstdc++" CXXFLAGS="-static-libstdc++"
```
