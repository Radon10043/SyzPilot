# setup image/freebsd

(host) generate an SSH key:
```bash
mkdir -p $EXPERIMENT_ROOT/image/freebsd && cd $EXPERIMENT_ROOT/image/freebsd
ssh-keygen -t rsa -f ./freebsd.id_rsa -N ""
```

## 15.0.0

(host) download 15.0 image:
```bash
cd $EXPERIMENT_ROOT/image/freebsd
wget https://download.freebsd.org/releases/VM-IMAGES/15.0-RELEASE/amd64/Latest/FreeBSD-15.0-RELEASE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-15.0-RELEASE-amd64-ufs.qcow2.xz
mv FreeBSD-15.0-RELEASE-amd64-ufs.qcow2 15.0.0.qcow2
qemu-img resize 15.0.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./15.0.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input `set console="comconsole"` and `boot`
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
cd $EXPERIMENT_ROOT/image/freebsd
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   FreeBSD freebsd 15.0-RELEASE FreeBSD 15.0-RELEASE 7aedc8de6446 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
poweroff
```

## 14.3.0

(host) download 14.3 image:
```bash
cd $EXPERIMENT_ROOT/image/freebsd
wget https://archive.freebsd.org/old-releases/VM-IMAGES/14.3-RELEASE/amd64/Latest/FreeBSD-14.3-RELEASE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-14.3-RELEASE-amd64-ufs.qcow2.xz
mv FreeBSD-14.3-RELEASE-amd64-ufs.qcow2 14.3.0.qcow2
qemu-img resize 14.3.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./14.3.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input `set console="comconsole"` and `boot`
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
cd $EXPERIMENT_ROOT/image/freebsd
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   FreeBSD freebsd 14.3-RELEASE FreeBSD 14.3-RELEASE 8c9ce319fef7 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
poweroff
```

## 14.2.0

(host) download 14.2 image:
```bash
cd $EXPERIMENT_ROOT/image/freebsd
wget https://archive.freebsd.org/old-releases/VM-IMAGES/14.2-RELEASE/amd64/Latest/FreeBSD-14.2-RELEASE-amd64-ufs.qcow2.xz
unxz -k FreeBSD-14.2-RELEASE-amd64-ufs.qcow2.xz
mv FreeBSD-14.2-RELEASE-amd64-ufs.qcow2 14.2.0.qcow2
qemu-img resize 14.2.0.qcow2 100G
qemu-system-x86_64 -m 16G -smp 16 -hda ./14.2.0.qcow2 -enable-kvm -net nic -net user,hostfwd=tcp::3733-:22 -nographic -cpu host
# press 3, input `set console="comconsole"` and `boot`
```

(vm) install freebsd kernel 14.2.0:
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
git clone -b release/14.2.0 --depth 1 https://github.com/freebsd/freebsd-src 14.2.0
cd 14.2.0
cp /root/cloud/configs/kernel/freebsd.config sys/amd64/conf/CLOUD
cd sys/amd64/conf && config CLOUD
cd ../compile/CLOUD
make cleandepend && make depend
make -j16 && make install
reboot
```

(host) install sshkey and verify kernel version:
```bash
cd $EXPERIMENT_ROOT/image/freebsd
ssh-copy-id -i ./freebsd.id_rsa.pub -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost

# output of the following command should look like:
#   FreeBSD freebsd 14.2-RELEASE FreeBSD 14.2-RELEASE c8918d6c7 CLOUD amd64
ssh -i ./freebsd.id_rsa -p 3733 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@localhost uname -a
```

(vm) close vm:
```sh
poweroff
```
