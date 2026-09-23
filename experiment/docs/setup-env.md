# experiment environment setup

please replace the following variables according to the actual situation:
- `$EXPERIMENT_ROOT`: directory for saving experiment artifacts.
- `$JOBS`: number of parallel jobs.
- `$CPUS`: number of CPUs assigned to container.

the prefix of a description means the context that the subsequent operation is performed:
- `(host)`: the host machine.
- `(container.syzpilot)`: a container derived from `cloud/docker/Dockerfile` image.
- `(container.syzkaller)`: a container derived from `cloud/experiment/syzkaller/Dockerfile` image.
- `(container.kernelgem)`: a container derived from `cloud/experiment/KernelGEM/Dockerfile` image.
- `(container.kernelgpt)`: a container derived from `cloud/experiment/KernelGPT/Dockerfile` image.
- `(container.syzdescribe)`: a container derived from `cloud/experiment/SyzDescribe/Dockerfile` image.
- `(container.syzgenplusplus)`: a container derived from `cloud/experiment/SyzGenPlusPlus/Dockerfile` image.
- `(container.syzspec)`: a container derived from `cloud/experiment/SyzSpec/Dockerfile` image.
- `(vm)`: the virtual machine.

## preparation

(host) prepare ~1T free space, build docker images:
```bash
cd /tmp
git clone https://github.com/Radon10043/cloud && cd cloud
docker build -t github.com/radon10043/syzpilot:latest --network host -f ./docker/Dockerfile .
docker build -t github.com/radon10043/kernelgem:latest --network host -f ./experiment/KernelGEM/Dockerfile .
docker build -t github.com/seclab-ucr/syzdescribe:latest --network host -f ./experiment/SyzDescribe/Dockerfile .
docker build -t github.com/seclab-ucr/syzgenplusplus:latest --network host -f ./experiment/SyzGenPlusPlus/Dockerfile .
docker build -t github.com/seclab-ucr/syzspec:latest --network host -f ./experiment/SyzSpec/Dockerfile .
docker tag github.com/radon10043/syzpilot:latest github.com/google/syzkaller:latest
docker tag github.com/radon10043/kernelgem:latest github.com/ise-uiuc/kernelgpt:latest
cd .. && rm -rf cloud
```

(host) startup containers:
```bash
docker run \
    -v $EXPERIMENT_ROOT:$EXPERIMENT_ROOT \
    --cpus $CPUS --network host \
    --privileged -d --rm \
    --name syzpilot-build \
    github.com/radon10043/syzpilot:latest tail -f /dev/null
```

(host) you can run `docker exec -it syzpilot-build bash` to enter the container.

> [!NOTE]
> If you want to re-synthesize specs using specific tools, derive a container from the corresponding image and re-synthesize it within the container; if you only want to build the kernel/image/fuzzer and re-use existing specs for evaluation, derive a container from the SyzPilot image is enough.

(container.syzpilot) in container, run:
```bash
cd $EXPERIMENT_ROOT
mkdir kernel fuzzer image
```

(container.syzpilot) download SyzPilot first, many important artifacts are in this repository:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone https://github.com/Radon10043/cloud && cd cloud
git submodule update --init --recursive
```

## table of contents

- [setup kernel/linux](setup-kernel_linux.md)
- [setup kernel/netbsd](setup-kernel_netbsd.md)
- [setup kernel/android](setup-kernel_android.md)
- [setup kernel/gvisor](setup-kernel_gvisor.md)
- [setup kernel/fuchsia](setup-kernel_fuchsia.md)
- [setup image/debian](setup-image_debian.md)
- [setup image/freebsd](setup-image_freebsd.md)
- [setup image/openbsd](setup-image_openbsd.md)
- [setup image/netbsd](setup-image_netbsd.md)
- [setup image/debdroid](setup-image_debdroid.md)
- [setup fuzzer/SyzPilot](setup-fuzzer_SyzPilot.md)
- [setup fuzzer/syzkaller](setup-fuzzer_syzkaller.md)
- [setup fuzzer/KernelGPT](setup-fuzzer_KernelGPT.md)
- [setup fuzzer/KernelGEM](setup-fuzzer_KernelGEM.md)
- [setup fuzzer/SyzDescribe](setup-fuzzer_SyzDescribe.md)
- [setup fuzzer/SyzGenPlusPlus](setup-fuzzer_SyzGenPlusPlus.md)
- [setup fuzzer/SyzSpec](setup-fuzzer_SyzSpec.md)
- [setup environment file](setup-env_file.md)

## run evaluation

(host) finally, we can perform fuzzing, for example:
```bash
docker compose --env-file ./compose.env -f $EXPERIMENT_ROOT/fuzzer/cloud/experiment/docker-compose/compose.SyzPilot.yaml up linux-v6.18-kernel --scale linux-v6.18-kernel=5 -d
```
