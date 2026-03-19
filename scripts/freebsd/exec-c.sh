#!/bin/bash

# this script is used to execute a c reproducer on the VM, for checking whether the reproducer
# can trigger the specific bug.
#
# usage:
#   REPRO=/path/to/repro.c SSHKEY=/path/to/freebsd.sshkey $CLOUD/scripts/freebsd/exec-c.sh

set -e

scp -P 3733 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    -v \
    $REPRO root@localhost:/tmp/repro.c

ssh -p 3733 \
    -F /dev/null \
    -o UserKnownHostsFile=/dev/null \
    -o IdentitiesOnly=yes \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -i $SSHKEY \
    root@localhost "cd /tmp && gcc repro.c -lpthread -static -o repro.out && chmod +x ./repro.out && ./repro.out"
