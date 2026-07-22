# setup kernel/gvisor

(host) download gvisor:
```bash
mkdir -p $EXPERIMENT_ROOT/kernel/gvisor/release-20251215.0
mkdir -p $EXPERIMENT_ROOT/kernel/gvisor/release-20251208.0
mkdir -p $EXPERIMENT_ROOT/kernel/gvisor/release-20251201.0
git clone -b release-20251215.0 --depth 1 https://github.com/google/gvisor $EXPERIMENT_ROOT/kernel/gvisor/release-20251215.0/src
git clone -b release-20251208.0 --depth 1 https://github.com/google/gvisor $EXPERIMENT_ROOT/kernel/gvisor/release-20251208.0/src
git clone -b release-20251201.0 --depth 1 https://github.com/google/gvisor $EXPERIMENT_ROOT/kernel/gvisor/release-20251201.0/src
```

(host) create an image for building gvisor:
```bash
cd $EXPERIMENT_ROOT/kernel/gvisor/release-20251215.0/src
docker build -t gvisor-build:20251215 --network host -f images/default/Dockerfile images/default
```

(host) build gvisor releases via docker container:
```bash
docker run \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251215.0/src:/src \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251215.0/output:/output \
    -w /src \
    --cpus 16 \
    --network host \
    --privileged \
    --rm gvisor-build:20251215 \
        --output_user_root=/output \
		build \
		--verbose_failures \
		//runsc:runsc_coverage

docker run \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251208.0/src:/src \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251208.0/output:/output \
    -w /src \
    --cpus 16 \
    --network host \
    --privileged \
    --rm gvisor-build:20251215 \
        --output_user_root=/output \
		build \
		--verbose_failures \
		//runsc:runsc_coverage

docker run \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251201.0/src:/src \
    -v $EXPERIMENT_ROOT/kernel/gvisor/release-20251201.0/output:/output \
    -w /src \
    --cpus 16 \
    --network host \
    --privileged \
    --rm gvisor-build:20251215 \
        --output_user_root=/output \
		build \
		--verbose_failures \
		//runsc:runsc_coverage

# if you need proxy, add following options to the end of above commands:
# --action_env=HTTP_PROXY=http://127.0.0.1:7890 \
# --action_env=HTTPS_PROXY=http://127.0.0.1:7890 \
# --action_env=http_proxy=http://127.0.0.1:7890 \
# --action_env=https_proxy=http://127.0.0.1:7890
```
