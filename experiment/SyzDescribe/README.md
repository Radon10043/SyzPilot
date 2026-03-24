# SyzDescribe

this document shows how to setup SyzDescribe and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$SYZDESCRIBE`: directory for saving SyzDescribe source
- `$KERNSRC`: directory for saving linux kernel source

create a docker image via `$CLOUD/experiment/SyzDescribe/Dockerfile` and enter the container.
```bash
docker build -t syzdescribe:latest --network host -f $CLOUD/experiment/SyzDescribe/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name syzdescribe-exp \
    syzdescribe:latest tail -f /dev/null
docker exec -it syzdescribe-exp bash
```

in the container, run following commands to setup SyzDescribe and generate specifications.
```bash
git clone https://github.com/seclab-ucr/SyzDescribe
cd SyzDescribe
git checkout a1c0e55bb111c076980ddf64c108cb7cb08dafb9
git apply $CLOUD/experiment/SyzDescribe/repo.patch
mkdir build && cd build
cmake -G "Unix Makefiles" -DLLVM_CONFIG_BINARY=/llvm-15/bin/llvm-config ..
make -j16
cd ..

cd $KERNSRC
cp $CLOUD/configs/kernel/syzbot.config .config
PATH=/llvm-15/bin:$PATH make LLVM=1 olddefconfig all -j16

go run $SYZDESCRIBE/kernelbc/gen.go -isSaveTemp -toolchain=/llvm-15/bin
bash build.sh

cd $SYZDESCRIBE
./scripts/genspec.sh -k $KERNSRC -o workdir
```

integrate them into syzkaller and feel free to perform fuzzing.
```bash
find $SYZDESCRIBE/workdir -name "syz*" | xargs -I {} cp {} $SYZKALLER/sys/linux
```