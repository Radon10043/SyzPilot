# setup cloud and fuzzing gvisor

According to [syzkaller's description of gvisor](https://github.com/google/syzkaller/tree/master/docs/gvisor), we can reuse the specs generated for linux for fuzzing.

The prefix of each description in the doc means the environment that the instructions are executed. This document covers the following envs:
- (host): host machine
- (container.build): gvisor build container
- (container.fuzz): kernel fuzzing container

Please replace the following variables according to the actual situation:
- `$GVISOR`: directory for saving gvisor and its artifacts.
- `CLOUD`: directory for saveing cloud source.

## setup gvisor build image

(host) build gvisor image via official Dockerfile, let's use `release-20260511.0` as example:
```bash
mkdir $GVISOR
git clone -b release-20260511.0 --depth 1 https://github.com/google/gvisor $GVISOR/src
cd $GVISOR/src
docker build -t gvisor-build:20260511 --network host -f images/default/Dockerfile .
```

## build gvisor

you can build gvisor via startup container with build command, or startup container, enter it, and running build command manually. The latter usually for debugging.

(host) startup a container and build gvisor, the artifacts will be saved under `$GVISOR/output`:
```bash
docker run \
    -v $GVISOR/src:/src \
    -v $GVISOR/output:/output \
    -w /src \
    --cpus 16 \
    --network host \
    --privileged \
    --rm gvisor-build:latest \
        --output_user_root=/output \
		build \
		--verbose_failures \
		//runsc:runsc_coverage \
		--action_env=HTTP_PROXY=http://127.0.0.1:7890 \
		--action_env=HTTPS_PROXY=http://127.0.0.1:7890 \
		--action_env=http_proxy=http://127.0.0.1:7890 \
		--action_env=https_proxy=http://127.0.0.1:7890
```

(host, optional) startup a container and build gvisor manually:
```bash
docker run \
    -v $GVISOR/src:/src \
	-v $GVISOR/output:/output \
    --cpus 16 \
    --network host \
    --privileged \
	--entrypoint /bin/bash \
    --rm -it gvisor-build:latest
```

(container.build) In container, run following command, the corresponding artifacts will also be saved under `$GVISOR/output`:
```bash
bazel --output_user_root=/output build \
	--verbose_failures \
	//runsc:runsc_coverage \
	--action_env=HTTP_PROXY=http://127.0.0.1:7890 \
	--action_env=HTTPS_PROXY=http://127.0.0.1:7890 \
	--action_env=http_proxy=http://127.0.0.1:7890 \
	--action_env=https_proxy=http://127.0.0.1:7890
```

## startup fuzzing

(host) startup a fuzzing container and enter it:
```bash
docker run \
    -v $GVISOR/src:/src \
    -v $GVISOR/output:/output \
	-v $CLOUD:/cloud \
    -w /cloud \
    --cpus 16 \
    --network host \
	--name gvisor-fuzz \
    --privileged \
    --rm -d cloud:latest tail -f /dev/null
docker exec -it
```

(container.fuzz) in the fuzzing container, install binary:
```bash
find /output -name runsc_cov -type f | xargs -I {} cp {} /usr/local/bin/
```

(container.fuzz) setup fuzzing configuration:
```bash
cat <<__EOF__ /cloud/workdir/gvisor.cfg
{
	"name": "gvisor",
	"target": "linux/amd64",
	"http": ":12345",
	"workdir": "/cloud/workdir",
	"image": "/usr/local/bin/runsc_cov",
	"syzkaller": "/cloud/syzkaller",
	"procs": 8,
	"type": "gvisor",
	"vm": {
		"count": 5,
		"runsc_args": "-platform=kvm --network=none --ignore-cgroups"
	}
}
__EOF__
```

(container.fuzz) build fuzzer and start fuzzing:
```bash
cd /cloud/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
make all -j16
/cloud/syzkaller/bin/syz-manager -config=./workdir/gvisor.cfg
```
