# setup fuzzer/SyzDescribe

(container.syzpilot): directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone https://github.com/seclab-ucr/SyzDescribe
cd SyzDescribe && git checkout a1c0e55bb111c076980ddf64c108cb7cb08dafb9
git apply -3 $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzDescribe/repo.patch
git submodule update --init --recursive

# bin-kern
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzDescribe/specs/linux-v6.18-kernel.patch
make -C syzkaller all
mv syzkaller/bin bin-kern

# bin-subsys
git -C syzkaller checkout .
git -C syzkaller clean -fdx
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/SyzDescribe/specs/linux-v6.18-subsystems.patch
make -C syzkaller all
mv syzkaller/bin bin-subsys
```

[docs for generating specs via SyzDescribe](../SyzDescribe/README.md)
