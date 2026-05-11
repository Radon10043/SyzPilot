#!/bin/bash

# this script is used to execute a syz reproducer on the VM, for checking whether the reproducer
# can trigger the specific bug.
#
# usage:
#   REPRO=/path/to/repro.syz SSHKEY=/path/to/freebsd.sshkey $SYZPILOT/scripts/freebsd/exec-syz.sh

set -e

SYZPILOT=$(realpath $(dirname $0)/../..)

scp -P 63822 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    -v \
    $SYZPILOT/syzkaller/bin/netbsd_amd64/* root@localhost:/tmp/

scp -P 63822 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    -v \
    $REPRO root@localhost:/tmp/repro.syz

ssh -p 63822 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    root@localhost "cd /tmp && ./syz-execprog -enable=all -repeat=0 -procs=8 ./repro.syz"
