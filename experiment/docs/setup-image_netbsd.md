# setup image/netbsd

(container.syzpilot): generate sshkey:
```bash
mkdir -p $EXPERIMENT_ROOT/image/netbsd && cd $EXPERIMENT_ROOT/image/netbsd
ssh-keygen -t rsa -f netbsd.id_rsa -N ""
```

## 2026.1-15e7fbc5

(container.syzpilot): download iso file and setup vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
wget https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2026.1-15e7fbc5.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026.1-15e7fbc5.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompted to select bootblocks.

(container.syzpilot): after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026.1-15e7fbc5.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
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

(container.syzpilot): copy the sshkey and built kernel to vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
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

# output of the uname command should look like:
#   NetBSD  10.1_STABLE NetBSD 10.1_STABLE (CLOUD) #1: Sat Mar 28 20:59:39 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/15e7fbc5...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

## 2025.11-ceec3d80

(container.syzpilot): download iso file and setup vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
# wget https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2025.11-ceec3d80.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2025.11-ceec3d80.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompted to select bootblocks.

(container.syzpilot): after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2025.11-ceec3d80.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
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

(container.syzpilot): copy the sshkey and built kernel to vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/ceec3d80/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of the uname command should look like:
#   NetBSD  10.99.12 NetBSD 10.99.12 (CLOUD) #0: Sat Mar 28 22:53:06 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/ceec3d80...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

## 2025.10-3c0f56ea

(container.syzpilot): download iso file and setup vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
# wget https://cdn.netbsd.org/pub/NetBSD/images/10.1/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2025.10-3c0f56ea.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2025.10-3c0f56ea.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompted to select bootblocks.

(container.syzpilot): after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2025.10-3c0f56ea.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
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

(container.syzpilot): copy sshkey and built kernel to vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/3c0f56ea/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of the uname command should look like:
#   NetBSD  10.99.10 NetBSD 10.99.10 (CLOUD) #0: Sat Mar 28 21:44:46 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/3c0f56ea...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```

## 2026.1-15e7fbc5-tprof

(container.syzpilot): download iso file and setup vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
# wget https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64.iso
qemu-img create -f qcow2 2026.1-15e7fbc5-tprof.qcow2 100G
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026.1-15e7fbc5-tprof.qcow2 -cdrom NetBSD-10.1-amd64.iso -boot d -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -display curses
```

(vm) during installation, select `use serial port com0` when prompted to select bootblocks.

(container.syzpilot): after installation complete, start vm.
```bash
qemu-system-x86_64 -enable-kvm -m 16G -smp 16 -cpu host -hda 2026.1-15e7fbc5-tprof.qcow2 -net nic,model=virtio -net user,hostfwd=tcp::6382-:22 -device virtio-rng-pci -nographic
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

(container.syzpilot): copy sshkey and built kernel to vm:
```bash
cd $EXPERIMENT_ROOT/image/netbsd
ssh-copy-id -i ./netbsd.id_rsa.pub -p 6382 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost
scp -P 6382 \
    -i ./netbsd.id_rsa \
    -o UserKnownHostsFile=/dev/null \
    -o StrictHostKeyChecking=no \
    $EXPERIMENT_ROOT/kernel/netbsd/15e7fbc5-tprof/src/sys/arch/amd64/compile/obj/CLOUD/netbsd root@localhost:/netbsd
```

(vm) reboot, verify kernel version, load kcov module and poweroff vm:
```sh
reboot

# output of the uname command should look like:
#   NetBSD  10.1_STABLE NetBSD 10.1_STABLE (CLOUD) #1: Sat Mar 28 20:59:39 CST 2026  root@HOSTNAME:/vol/kernel/netbsd/15e7fbc5...
uname -a

cd /dev
sh MAKEDEV kcov
poweroff
```
