# setup kernel/linux

```bash
mkdir $EXPERIMENT_ROOT/kernel/linux && cd $EXPERIMENT_ROOT/kernel/linux
git clone -b v6.18 --depth 1 https://github.com/gregkh/linux v6.18
git clone -b v6.17.13 --depth 1 https://github.com/gregkh/linux v6.17.13
git clone -b v6.12.63 --depth 1 https://github.com/gregkh/linux v6.12.63
git clone -b v6.6.119 --depth 1 https://github.com/gregkh/linux v6.6.119
git clone -b v6.1.159 --depth 1 https://github.com/gregkh/linux v6.1.159
git clone -b v5.15.197 --depth 1 https://github.com/gregkh/linux v5.15.197

cd $EXPERIMENT_ROOT/kernel/linux/v6.18 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.17.13 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.12.63 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.6.119 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v6.1.159 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
cd $EXPERIMENT_ROOT/kernel/linux/v5.15.197 && cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/linux.config .config && make CC=clang olddefconfig all -j$JOBS
```
