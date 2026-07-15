# setup KernelGEM

this document shows how to setup KernelGEM and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$SYZPILOT`: directory for saveing SyzPilot source
- `$KERNELGEM`: directory for saving KernelGEM source
- `$KERNSRC`: directory for saving linux kernel source

create a docker image via `$SYZPILOT/experiment/KernelGEM/Dockerfile` and enter the container.
```bash
docker build -t kernelgpt:latest --network host -f $SYZPILOT/experiment/KernelGEM/Dockerfile .
docker run \
    -d \
    --cpus 16 \
    --network host \
    --privileged \
    --name kernelgem-exp \
    kernelgem:latest tail -f /dev/null
docker exec -it kernelgem-exp bash
```

setup KernelGEM.
```bash
git clone https://github.com/ise-uiuc/KernelGPT.git KernelGEM
cd KernelGEM
git checkout e3464d23b8d59ffffb1bd5b2f7100c102c48bb3d
git apply -3 $SYZPILOT/experiment/KernelGEM/repo.patch
git submodule update --init --recursive --depth 1 --progress
pip install -r requirements.txt
```

build linux v6.18.
```bash
cd $KernelGEM
cp $SYZPILOT/configs/kernel/linux.config linux/.config
bear -- make CC=clang HOSTCC=clang olddefconfig all -j16
```

build and run analysis tool.
```bash
cd $KernelGEM/spec-gen/analyzer
make all

./analyze -p $KernelGEM/linux/compile_commands.json
python3 process_output.py --linux-path $KernelGEM/linux
./usage -p $KernelGEM/linux/compile_commands.json
python3 process_output.py --linux-path $KernelGEM/linux --usage
```

setup .env file under `$KernelGEM/spec-gen` and run syscall description generation.
```bash
cd $KernelGEM/spec-gen
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
ls sys/linux/gpt*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```
