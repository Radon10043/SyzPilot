# experiment environment setup

> [!NOTE]
> the document currently contains many redundant steps and verbose statements and needs to be refactored.

Please replace the following variables according to the actual situation:
- `$EXPERIMENT_ROOT`: directory for saving experiment artifacts.
- `$JOBS`: number of parallel jobs.

## preparation

prepare ~1T free space, start up a container based on cloud image. In container, run:
```bash
cd $EXPERIMENT_ROOT
mkdir kernel fuzzer image
```

download cloud first, many important artifacts are in this repository:
```bash
cd $EXPERIMENT_ROOT/fuzzer
git clone https://github.com/Radon10043/cloud && cd cloud
git submodule update --init --recursive
```

## table of contents

- [setup kernel/linux](setup-kernel_linux.md)
- [setup kernel/netbsd](setup-kernel_netbsd.md)
- [setup image/debian](setup-image_debian.md)
- [setup image/freebsd](setup-image_freebsd.md)
- [setup image/openbsd](setup-image_openbsd.md)
- [setup image/netbsd](setup-image_netbsd.md)
- [setup fuzzer/cloud](setup-fuzzer_cloud.md)
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
docker compose --env-file ./compose.env -f $EXPERIMENT_ROOT/fuzzer/cloud/experiment/docker-compose/compose.cloud.yaml up linux-v6.18-kernel --scale linux-v6.18-kernel=5 -d
```
