# setup kernel/fuchsia

## releases/f30

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"
export PATH="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=4c117a94d1cafc3f48d0f3deee07676f6501a0af \
	-remote-branch=releases/f30 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git -C third_party/syzkaller fetch
git -C third_party/syzkaller checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git -C third_party/syzkaller apply -3 "$EXPERIMENT_ROOT"/fuzzer/cloud/patch/syzkaller/*
git -C third_party/syzkaller restore --staged .
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f30.patch"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30/third_party/syzkaller"
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"
fx --dir "out/x64" set core.x64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j$JOBS

"$EXPERIMENT_ROOT/fuzzer/cloud/scripts/fuchsia/refresh.sh" -f "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f30"
```

## releases/f29

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"
export PATH="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=be673b9c46cf8b6dc4273cff07d5b5382eefd035 \
	-remote-branch=releases/f29 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git -C third_party/syzkaller fetch
git -C third_party/syzkaller checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git -C third_party/syzkaller apply -3 "$EXPERIMENT_ROOT"/fuzzer/cloud/patch/syzkaller/*
git -C third_party/syzkaller restore --staged .
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f29.patch"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29/third_party/syzkaller"
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"
fx --dir "out/x64" set core.x64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j$JOBS

"$EXPERIMENT_ROOT/fuzzer/cloud/scripts/fuchsia/refresh.sh" -f "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f29"
```

## releases/f28

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"
export PATH="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=080bdf332f9d6bf778baf0e3c4285df421bf1c79 \
	-remote-branch=releases/f28 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git -C third_party/syzkaller fetch
git -C third_party/syzkaller checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git -C third_party/syzkaller apply -3 "$EXPERIMENT_ROOT"/fuzzer/cloud/patch/syzkaller/*
git -C third_party/syzkaller restore --staged .
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f28.patch"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28/third_party/syzkaller"
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"
fx --dir "out/x64" set core.x64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j$JOBS

"$EXPERIMENT_ROOT/fuzzer/cloud/scripts/fuchsia/refresh.sh" -f "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f28"
```

## releases/f27

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"
export PATH="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=25ac6a95975054e125524e02c73363308857d24d \
	-remote-branch=releases/f27 \
	flower \
	https://fuchsia.googlesource.com/integration
git config --global \
	url.https://android.googlesource.com/platform/external/perfetto.insteadOf \
    https://fuchsia.googlesource.com/third_party/android.googlesource.com/platform/external/perfetto/
git config --global --add \
	url.https://android.googlesource.com/platform/external/perfetto.insteadOf \
	https://fuchsia.googlesource.com/third_party/android.googlesource.com/platform/external/perfetto
jiri update

source scripts/fx-env.sh && fx-update-path
git -C third_party/syzkaller fetch
git -C third_party/syzkaller checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git -C third_party/syzkaller apply -3 "$EXPERIMENT_ROOT"/fuzzer/cloud/patch/syzkaller/*
git -C third_party/syzkaller restore --staged .
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f27.patch"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27/third_party/syzkaller"
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR="$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"

cd "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"
fx --dir "out/x64" set core.x64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j$JOBS

"$EXPERIMENT_ROOT/fuzzer/cloud/scripts/fuchsia/refresh.sh" -f "$EXPERIMENT_ROOT/kernel/fuchsia/releases.f27"
```
