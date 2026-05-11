# Synthesize specs for subsystems

## How to re-synthesize specs for a subsystem

1. Copy `syzkaller/sys/$OS/*.txt` to a temporary directory.
2. Delete the related specs and `.const` files.
3. Extract constants, run `syz-check`, manually fix errors, and repeat until no errors are reported.
4. Setup reference file, then run SyzPilot to synthesize specs.
5. Manually fix errors in specs if needed, run `extract` and `generate` command to extract consts and format specs respectively.

An example of synthesize specs for can subsystem in Linux kernel:
```bash
cd $SYZPILOT
nohup ./bin/generator \
    -db=./data/database/linux.db \
    -outdir=./workdir/specs/linux-v6.18-subsystems-gemini-3-flash-preview \
    -kernel=/vol/linux/v6.18/build \
    -model=gemini-3-flash-preview \
    -ref=./data/refs/linux/can.txt \
    -sysdir=./data/trimsys \
    -os=linux \
    -jobs=2 > logs/can.log 2>&1 &
```

## Subsystem fuzzing

Merge the subsystem configuration file with `kernel.cfg` to test a subsystem in a targeted manner. For example, to test `linux/ocfs2`:
```bash
cd $SYZPILOT
jq -s 'add' configs/fuzz/linux/kernel.cfg configs/fuzz/linux/ocfs2.cfg > $WORKDIR/test.cfg
./syzkaller/bin/syz-manager -config=$WORKDIR/test.cfg
```
