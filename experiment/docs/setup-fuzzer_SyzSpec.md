# setup fuzzer/SyzSpec

(container.syzpilot): directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer/SyzSpec
git apply -3 $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzSpec/repo.patch
git submodule update --init --recursive

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
