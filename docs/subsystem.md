# generate specs for subsystems

linux:
```bash
cd $SYZPILOT
nohup ./bin/generator \
    -db=./data/database/linux.db \
    -outdir=./.workdir/specs/linux-v6.18-subsystems-gemini-3-flash-preview \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -ref=./data/refs/linux/can.txt \
    -sysdir=./data/trimsys \
    -os=linux \
    -jobs=2 > logs/can.log 2>&1 &
```

freebsd:
```bash
cd $SYZPILOT
nohup ./bin/generator \
    -db=./data/database/freebsd.db \
    -outdir=./.workdir/specs/freebsd-15.0.0-inet_icmp-gemini-3-flash-preview \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -ref=./data/refs/freebsd/inet_icmp.txt \
    -sysdir=./data/trimsys \
    -os=freebsd \
    -jobs=1 > logs/inet_icmp.log 2>&1 &
```

openbsd:
```bash
cd $SYZPILOT
nohup ./bin/generator \
    -db=./data/database/openbsd.db \
    -outdir=./.workdir/specs/openbsd-23290a22-vnd-gemini-3-flash-preview \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -ref=./data/refs/openbsd/vnd.txt \
    -sysdir=./data/trimsys \
    -os=openbsd \
    -jobs=1 > logs/vnd.log 2>&1 &
```

netbsd:
```bash
cd $SYZPILOT
nohup ./bin/generator \
    -db=./data/database/netbsd.db \
    -outdir=./.workdir/specs/netbsd-15e7fbc5-tprof-gemini-3-flash-preview \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -ref=./data/refs/netbsd/tprof.txt \
    -sysdir=./data/trimsys \
    -os=netbsd \
    -jobs=2 > logs/tprof.log 2>&1 &
```

# subsystem fuzzing

please merge subsystem's config file to kernel.cfg to test subsystem in target manner, e.g. test linux/ocfs2:
```bash
cd $SYZPILOT
jq -s 'add' configs/fuzz/linux/kernel.cfg configs/fuzz/linux/ocfs2.cfg > $WORKDIR/test.cfg
./syzkaller/bin/syz-manager -config=$WORKDIR/test.cfg
```

extract const:
```bash
ls sys/linux/gen#*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```

# how to re-generate specs for other subsystem?

1. copy syzkaller/sys/$OS/*.txt to a tmp dir
2. delete related specs
3. extract consts, run syz-check, manual fix errors, repeat until no error report
4. run SyzPilot to generate specs