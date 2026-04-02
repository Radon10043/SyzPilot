#!/bin/bash

# this script is used to execute a c reproducer on the VM, for checking whether the reproducer
# can trigger the specific bug.
#
# usage:
#   NETBSD=/path/to/netbsd REPRO=/path/to/repro.c SSHKEY=/path/to/freebsd.sshkey $CLOUD/scripts/freebsd/exec-c.sh

set -e

$NETBSD/tools/bin/x86_64--netbsd-clang++ $REPRO -o /tmp/repro.out -static --sysroot $NETBSD/dest/ -pthread

scp -P 63822 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    -v \
    /tmp/repro.out root@localhost:/tmp/repro.out

ssh -p 63822 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    root@localhost "cd /tmp && chmod +x ./repro.out && ./repro.out"
