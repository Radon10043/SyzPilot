# experiment environment setup

> [!NOTE]
> the document currently contains many redundant steps and verbose statements and needs to be refactored.

Please replace the following variables according to the actual situation:
- `$EXPERIMENT_ROOT`: directory for saving experiment artifacts.
- `$JOBS`: number of parallel jobs.

## preparation

prepare ~1T free space, start up a container based on SyzPilot image. In container, run:
```bash
cd $EXPERIMENT_ROOT
mkdir kernel fuzzer image
```

download SyzPilot first, many important artifacts are in this repository:
```bash
mkdir $EXPERIMENT_ROOT/fuzzer/SyzPilot
wget -O $EXPERIMENT_ROOT/fuzzer/SyzPilot/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
cd $EXPERIMENT_ROOT/fuzzer/SyzPilot && unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
```

## table of contents

- [setup kernel/linux](setup-kernel_linux.md)
- [setup kernel/netbsd](setup-kernel_netbsd.md)
- [setup image/debian](setup-image_debian.md)
- [setup image/freebsd](setup-image_freebsd.md)
- [setup image/openbsd](setup-image_openbsd.md)
- [setup image/netbsd](setup-image_netbsd.md)
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
docker compose --env-file ./compose.env -f $EXPERIMENT_ROOT/fuzzer/SyzPilot/experiment/docker-compose/compose.SyzPilot.yaml up linux-v6.18-kernel --scale linux-v6.18-kernel=5 -d
```
