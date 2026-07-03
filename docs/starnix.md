# setup cloud for starnix fuzzing

```bash
curl -sO https://storage.googleapis.com/fuchsia-ffx/ffx-linux-x64
chmod +x ffx-linux-x64
./ffx-linux-x64 platform preflight
```

```bash
cd $KERNSRC
curl -s "https://fuchsia.googlesource.com/fuchsia/+/HEAD/scripts/bootstrap?format=TEXT" | base64 --decode | bash
```

change version (e.g. 55e4d9ecc4b7 in releases/f31 branch):
```bash
cd $KERNSRC/fuchsia
git -C integration checkout -B tmp --no-track 55e4d9ecc4b7
jiri update -local-manifest -gc -fetch-packages=false
jiri fetch-packages -local-manifest-project=fuchsia
```

refresh environment:
```bash
export PATH=$KERNSRC/fuchsia/.jiri_root/bin:$PATH
source $KERNSRC/fuchsia/scripts/fx-env.sh
```

build artifacts for starnix:
```bash
cd $KERNSRC/fuchsia && mkdir local
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
	--assembly-override //local:syzkaller_starnix \
	--with //bundles/tools
fx build
```

build syzkaller:
```bash
cd $CLOUD/syzlaller
SYZ_STARNIX_HACK=1 make TARGETOS=linux TARGETARCH=amd64
```

start fuzzing:
```bash
cd $CLOUD
cat <<__EOF__ > $WORKDIR/starnix.cfg
{
    "target": "linux/amd64",
    "http": "127.0.0.1:56741",
    "workdir": "$WORKDIR",
    "kernel_obj": "$KERNSRC/fuchsia/out/x64/exe.unstripped/starnix_kernel",
    "kernel_src": "$KERNSRC/fuchsia",
    "syzkaller": "$CLOUD/syzkaller",
    "procs": 1,
    "type": "starnix",
    "cover": true
    "vm": {
        "count": 1,
        "coverage_pcs": "/tmp/starnix-sancov/starnix_kernel.sancov",
        "coverage_counts": "/tmp/starnix-sancov/starnix_kernel.sancov-counts"
    }
}

__EOF__

SYZ_STARNIX_HACK=1 $CLOUD/bin/syz-manager -config=$WORKDIR/starnix.cfg
```
