# KernelGPT

this document shows how to setup KernelGPT and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$KERNELGPT`: directory for saving KernelGPT source
- `$KERNSRC`: directory for saving linux kernel source

## TL;DR

Hope the following commands are self-evident.
```bash
cd $KERNELGPT
git apply -3 $CLOUD/experiment/KernelGPT/repo.patch
git submodule update --init --recursive
git -C syzkaller-KernelGPT apply $CLOUD/experiment/KernelGPT/specs#linux-v6.7#gpt-4.patch
git -C syzkaller-KernelGEM apply $CLOUD/experiment/KernelGPT/specs#linux-v6.18#gemini-3-flash-preview.patch
```

## setup KernelGPT

create a docker image via `$CLOUD/experiment/KernelGPT/Dockerfile` and enter the container.
```bash
docker build -t kernelgpt:latest --network host -f $CLOUD/experiment/KernelGPT/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name kernelgpt-exp \
    kernelgpt:latest tail -f /dev/null
docker exec -it kernelgpt-exp bash
```

setup KernelGPT.
```bash
git clone https://github.com/KernelGPT/KernelGPT.git
cd KernelGPT
git checkout e3464d23b8d59ffffb1bd5b2f7100c102c48bb3d
git apply $CLOUD/experiment/KernelGPT/repo.patch
git submodule update --init --recursive
pip install -r requirements.txt
```

build linux v6.18.
```bash
cd $KERNELGPT
cp $CLOUD/configs/kernel/syzbot.config linux/.config
bear -- make CC=clang HOSTCC=clang olddefconfig all -j16
```

build and run analysis tool.
```bash
cd $KERNELGPT/spec-gen/analyzer
make all

./analyze -p ./analyze -p $KERNELGPT/linux/compile_commands.json
python3 process_output.py --linux-path $KERNELGPT/linux
./usage -p $KERNELGPT/linux/compile_commands.json
python3 process_output.py --linux-path $KERNELGPT/linux --usage
```

setup .env file under `$KERNELGPT/spec-gen` and run syscall description generation.
```bash
cd $KERNELGPT/spec-gen
echo "OPENAI_BASE_URL=YOUR_BASE_URL" > .env
echo "OPENAI_API_KEY=YOUR_API_KEY" > .env
echo "OPENAI_MODEL=YOUR_FAVORITE_LLM" > .env

# for debugging
python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1
# for generating complete specifications, watch out your wallet
# python3 gen_spec.py -d analyzer/processed_handlers.json -o spec-output -n 1000
```

integrate generated specifications into syzkaller and feel free to perform fuzzing.

If you want to re-extract const for KernelGPT's specs:
```bash
cd $KERNLGPT/syzkaller
make bin/syz-extract
ls sys/linux/gpt4*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```