# setup fuzzer/SyzSpec

(container.syzpilot): directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone https://github.com/seclab-ucr/SyzSpec
cd SyzSpec && git checkout 1edbcffd6f56786d914b0c04458bee86abf215ac
git apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzSpec/repo.patch
git clone https://github.com/google/syzkaller
git -C syzkaller checkout ac3c71e7

# bin-kern
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzSpec/specs/linux-v6.18-kernel.patch
make -C syzkaller all
mv syzkaller/bin bin-kern

# bin-subsys
git -C syzkaller checkout .
git -C syzkaller clean -fdx
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzSpec/specs/linux-v6.18-subsys.patch
make -C syzkaller all
mv syzkaller/bin bin-subsys
```

[docs for generating specs via SyzSpec](../SyzSpec/README.md)
