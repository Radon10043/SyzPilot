# setup kernel/starnix

## releases/f30

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/starnix/releases.f30"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/starnix/releases.f30"
export PATH="$EXPERIMENT_ROOT/kernel/starnix/releases.f30/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/starnix/releases.f30"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=4c117a94d1cafc3f48d0f3deee07676f6501a0af \
	-remote-branch=releases/f30 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f30.patch"

mkdir local
cat <<__EOF__ > local/BUILD.gn
import("//build/assembly/developer_overrides.gni")

assembly_developer_overrides("syzkaller_starnix") {
  testonly = true
  base_packages = [
    "//src/testing/fuzzing/syzkaller/starnix:syzkaller_starnix",
  ]
}
__EOF__

fx --dir out/x64 set workbench_eng.x64 \
	--args=starnix_sancov=true \
	--assembly-override //local:syzkaller_starnix \
	--with //bundles/tools
fx build -j$JOBS
```

## releases/f29

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/starnix/releases.f29"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/starnix/releases.f29"
export PATH="$EXPERIMENT_ROOT/kernel/starnix/releases.f29/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/starnix/releases.f29"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=be673b9c46cf8b6dc4273cff07d5b5382eefd035 \
	-remote-branch=releases/f29 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f29.patch"

mkdir local
cat <<__EOF__ > local/BUILD.gn
import("//build/assembly/developer_overrides.gni")

assembly_developer_overrides("syzkaller_starnix") {
  testonly = true
  base_packages = [
    "//src/testing/fuzzing/syzkaller/starnix:syzkaller_starnix",
  ]
}
__EOF__

fx --dir out/x64 set workbench_eng.x64 \
	--args=starnix_sancov=true \
	--assembly-override //local:syzkaller_starnix \
	--with //bundles/tools
fx build -j$JOBS
```

## releases/f28

(container.cloud) **NOTE**: please disable `perf_event_open*` when fuzzing f28 starnix:
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/starnix/releases.f28"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/starnix/releases.f28"
export PATH="$EXPERIMENT_ROOT/kernel/starnix/releases.f28/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/starnix/releases.f28"
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=080bdf332f9d6bf778baf0e3c4285df421bf1c79 \
	-remote-branch=releases/f28 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update

source scripts/fx-env.sh && fx-update-path
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f28.patch"

mkdir local
cat <<__EOF__ > local/BUILD.gn
import("//build/assembly/developer_overrides.gni")

assembly_developer_overrides("syzkaller_starnix") {
  testonly = true
  base_packages = [
    "//src/testing/fuzzing/syzkaller/starnix:syzkaller_starnix",
  ]
}
__EOF__

fx --dir out/x64 set workbench_eng.x64 \
	--args=starnix_sancov=true \
	--assembly-override //local:syzkaller_starnix \
	--with //bundles/tools
fx build -j$JOBS
```

## releases/f27

(container.cloud):
```bash
mkdir -p "$EXPERIMENT_ROOT/kernel/starnix/releases.f27"
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$EXPERIMENT_ROOT/kernel/starnix/releases.f27"
export PATH="$EXPERIMENT_ROOT/kernel/starnix/releases.f27/.jiri_root/bin:$PATH"

cd "$EXPERIMENT_ROOT/kernel/starnix/releases.f27"
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
git apply "$EXPERIMENT_ROOT/fuzzer/cloud/patch/fuchsia/f27.patch"

mkdir local
cat <<__EOF__ > local/BUILD.gn
import("//build/assembly/developer_overrides.gni")

assembly_developer_overrides("syzkaller_starnix") {
  testonly = true
  base_packages = [
    "//src/testing/fuzzing/syzkaller/starnix:syzkaller_starnix",
  ]
}
__EOF__

fx --dir out/x64 set workbench_eng.x64 \
	--args=starnix_sancov=true \
	--assembly-override //local:syzkaller_starnix \
	--with //bundles/tools
fx build -j$JOBS
```
