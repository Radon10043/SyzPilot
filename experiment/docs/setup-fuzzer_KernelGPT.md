# setup fuzzer/KernelGPT

(host) directly use generated specs:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone https://github.com/ise-uiuc/KernelGPT
cd KernelGPT && git checkout e3464d23b8d59ffffb1bd5b2f7100c102c48bb3d
git apply -3 $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/KernelGPT/repo.patch
git submodule update --init --depth 1 --progress syzkaller

# bin-kern
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/KernelGPT/specs/linux-v6.18-kernel-gpt-5.2.patch
make -C syzkaller all
mv syzkaller/bin bin-kern

# bin-subsys
git -C syzkaller checkout .
git -C syzkaller clean -fdx
git -C syzkaller apply $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/KernelGPT/specs/linux-v6.18-subsystem-gpt-5.2.patch
make -C syzkaller all
mv syzkaller/bin bin-subsys
```

[docs for generating specs via KernelGPT](../KernelGPT/README.md)
