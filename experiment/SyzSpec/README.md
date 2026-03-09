## SyzSpec

this document shows how to setup SyzSpec and use it for generating syscall specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$SYZSPEC`: directory for saving SyzSpec source
- `$KERNSRC`: directory for saving linux kernel source

create a docker image via `$CLOUD/experiment/SyzDescribe/Dockerfile` and enter the container.
```bash
docker build -t syzspec:latest --network host -f $CLOUD/experiment/SyzDescribe/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name syzspec-exp \
    syzspec:latest tail -f /dev/null
docker exec -it syzspec-exp bash
```

in container, setup SyzSpec:
```bash
git clone https://github.com/seclab-ucr/SyzSpec
cd SyzSpec
git checkout 1edbcffd6f56786d914b0c04458bee86abf215ac
git apply $CLOUD/experiment/SyzSpec/repo.patch

mkdir build && cd build
cmake .. \
	-DCMAKE_C_COMPILER=/tmp/llvm-150-install_O_D_A/bin/clang \
    -DCMAKE_CXX_COMPILER=/tmp/llvm-150-install_O_D_A/bin/clang++ \
	-DLLVM_DIR=/tmp/llvm-150-install_O_D_A/lib/cmake/llvm \
	-DENABLE_SOLVER_Z3=ON \
	-DENABLE_TCMALLOC=OFF \
	-DCMAKE_PREFIX_PATH="/tmp/z3-4.8.15-install;/tmp/sqlite-amalgamation-3400100"
make -j8
```

build linux kernel and generate .bc files.
```bash
cd $KERNSRC # v6.18
cp $CLOUD/configs/kernel/syzbot.config .config
make LLVM=1 \
	 PATH=/llvm-15/bin:$PATH \
	 KCFLAGS="-Xclang -no-opaque-pointers -mllvm -opaque-pointers=0" \
	 KBUILD_LDFLAGS="-mllvm -opaque-pointers=0" \
	 olddefconfig all -j16

go run $SYZSPEC/kernelbc/gen.go -isSaveTemp -toolchain=/llvm-15/bin
bash build.sh
```

generate specification for driver(s), e.g. for ppp, run:
```bash
$SYZSPEC/build/bin/klee \
	--entry-point=ppp_ioctl \
	--spec-arguments-index=1 \
	--spec-arguments-num=2 \
	--spec-interface-name=ioctl \
	--spec-prefix="fd fd_spec" \
	--spec-suffix="" \
	--spec-output="ioctl" \
	$KERNSRC/drivers/net/ppp/built-in.bc
```