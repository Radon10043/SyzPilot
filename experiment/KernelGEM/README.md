# setup KernelGEM for spec generation

this document shows how to setup KernelGEM and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing SyzPilot source
- `$KERNELGEM`: directory for saving KernelGEM source
- `$LINUX`: directory for saving linux kernel source
- `$ANDROID`: directory for saving android kernel source

## preparation

create a docker image via `$CLOUD/experiment/KernelGEM/Dockerfile` and enter the container.
```bash
docker build -t kernelgpt:latest --network host -f $CLOUD/experiment/KernelGEM/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name kernelgem-exp \
    kernelgem:latest tail -f /dev/null
docker exec -it kernelgem-exp bash
```

## setup KernelGEM

```bash
git clone https://github.com/ise-uiuc/KernelGPT.git KernelGEM
cd KernelGEM
git checkout e3464d23b8d59ffffb1bd5b2f7100c102c48bb3d
git apply -3 $CLOUD/experiment/KernelGEM/repo.patch
git submodule update --init --recursive --depth 1 --progress
pip install -r requirements.txt
```

## generate specs for linux

build linux v6.18.
```bash
cd $KERNELGEM
cp $CLOUD/configs/kernel/linux.config linux/.config
bear -- make CC=clang HOSTCC=clang olddefconfig all -j16
```

build and run analysis tool.
```bash
cd $KERNELGEM/spec-gen/analyzer
make all

./analyze -p $KERNELGEM/linux/compile_commands.json
python3 process_output.py --linux-path $KERNELGEM/linux
./usage -p $KERNELGEM/linux/compile_commands.json
python3 process_output.py --linux-path $KERNELGEM/linux --usage
```

setup .env file under `$KERNELGEM/spec-gen` and run syscall description generation.
```bash
cd $KERNELGEM/spec-gen
touch .env
echo "OPENAI_BASE_URL=YOUR_BASE_URL" >> .env
echo "OPENAI_API_KEY=YOUR_API_KEY" >> .env
echo "OPENAI_MODEL=YOUR_FAVORITE_LLM" >> .env

# for debugging
python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1
# for generating complete specifications, watch out your wallet
# python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1000
```

integrate generated specifications into syzkaller and feel free to perform fuzzing.

(re-)generate specs for subsystems, before running, please delete content of `cmds.txt`, `sockopt.txt`, and `types.txt` under `$KERNELGEM/spec-gen/exsitings` first, then run:
```bash
python3 gen_spec.py \
  -d analyzer/processed_handlers.json \
  -n 50 \
  --subsystems can autofs phonet comedi dri fuse i2c input kvm ax25 ptp ppp rdma_cm sequencer sg x25
```

extract consts for synthesized specifications:
```bash
cd $KERNELGPT/syzkaller
make bin/syz-extract
ls sys/linux/gpt*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$LINUX -os=linux -arch=amd64
```

## generate specs for android

generate `compile_commands.json` of android kernel:
```bash
cd $ANDROID
tools/bazel run --kasan --defconfig_fragment=//common:debian_image_x86_64_defconfig //common-modules/virtual-device:virtual_device_x86_64_compile_commands -- $PWD/dist/compile_commands.json
```

build and run analysis tool:
```bash
cd $KERNELGEM/spec-gen/analyzer
make TARGETOS=android all

./analyze -p $ANDROID/dist/compile_commands.json
python3 process_output.py --linux-path $PWD
./usage -p $ANDROID/dist/compile_commands.json
python3 process_output.py --linux-path $PWD --usage
```

setup .env file under `$KERNELGEM/spec-gen` and run syscall description generation.
```bash
cd $KERNELGEM/spec-gen
touch .env
echo "OPENAI_BASE_URL=YOUR_BASE_URL" >> .env
echo "OPENAI_API_KEY=YOUR_API_KEY" >> .env
echo "OPENAI_MODEL=YOUR_FAVORITE_LLM" >> .env

# for debugging
python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1
# for generating complete specifications, watch out your wallet
# python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1000
```
