# setup KernelGPT

this document shows how to setup KernelGPT and use it for generating specifications and fuzzing.

please replace the following variables according to your actual situation:
- `$CLOUD`: directory for saveing cloud source
- `$KERNELGPT`: directory for saving KernelGPT source
- `$KERNSRC`: directory for saving linux kernel source

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
git apply -3 $CLOUD/experiment/KernelGPT/repo.patch
git submodule update --init --recursive --pdeth 1 --progress
pip install -r requirements.txt
```

setup .env file under `$KernelGEM/spec-gen`:
```bash
cd $KernelGEM/spec-gen
touch .env
echo "OPENAI_BASE_URL=YOUR_BASE_URL" >> .env
echo "OPENAI_API_KEY=YOUR_API_KEY" >> .env
echo "OPENAI_MODEL=YOUR_FAVORITE_LLM" >> .env
```

If you want to re-extract const for KernelGPT's specs:
```bash
cd $KERNELGPT/spec-gen
python3 eval_spec.py -u -s ../generated-specs/specs-6.7/correct-driver-spec --output-name debug -o eval-output --merge
python3 eval_spec.py -u -s ../generated-specs/specs-6.7/correct-socket-spec --output-name debug -o eval-output --merge
cp ../spec-eval/debug/*.txt syzkaller/sys/linux

cd ../syzkaller
make bin/syz-extract
ls sys/linux/gpt*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
```
