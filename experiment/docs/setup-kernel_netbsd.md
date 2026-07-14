# setup kernel/netbsd

```bash
mkdir -p $EXPERIMENT_ROOT/kernel/netbsd && cd $EXPERIMENT_ROOT/kernel/netbsd
mkdir -p 15e7fbc5/src

cd 15e7fbc5/src
git init .
git remote add origin https://github.com/NetBSD/src
git fetch --depth 1 origin 15e7fbc53d77cd7cc1d62511982b8972c4c0c421
git checkout 15e7fbc5

cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/netbsd.config sys/arch/amd64/conf/CLOUD
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools tools
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools kernel=CLOUD
```

```bash
cd $EXPERIMENT_ROOT/kernel/netbsd
mkdir -p ceec3d80/src

cd ceec3d80/src
git init .
git remote add origin https://github.com/NetBSD/src
git fetch --depth 1 origin ceec3d80eed1a1082cacf866e3f09e31657b8525
git checkout ceec3d80

cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/netbsd.config sys/arch/amd64/conf/CLOUD
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools tools
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools kernel=CLOUD
```

```bash
cd $EXPERIMENT_ROOT/kernel/netbsd
mkdir -p 3c0f56ea/src

cd 3c0f56ea/src
git init .
git remote add origin https://github.com/NetBSD/src
git fetch --depth 1 origin 3c0f56ea164d7e5bea72b1c64425fb24f9f3be6f
git checkout 3c0f56ea

cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/netbsd.config sys/arch/amd64/conf/CLOUD
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools tools
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools kernel=CLOUD
```

tprof is not enabled in syzbot's config, so prepare a configuration for targeted fuzzing on it:
```bash
cd $EXPERIMENT_ROOT/kernel/netbsd
mkdir -p 15e7fbc5-tprof/src

cd 15e7fbc5-tprof/src
git init .
git remote add origin https://github.com/NetBSD/src
git fetch --depth 1 origin 15e7fbc53d77cd7cc1d62511982b8972c4c0c421
git checkout 15e7fbc5

cp $EXPERIMENT_ROOT/fuzzer/cloud/configs/kernel/netbsd.extend.config sys/arch/amd64/conf/CLOUD
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools tools
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools -D ../dest distribution
./build.sh -j$JOBS -m amd64 -c clang -U -T ../tools kernel=CLOUD
```
