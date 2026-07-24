# setup cloud for starnix fuzzing

check environment:
```bash
curl -sO https://storage.googleapis.com/fuchsia-ffx/ffx-linux-x64
chmod +x ffx-linux-x64
./ffx-linux-x64 platform preflight
```

download jiri binary:
```bash
curl -s "https://fuchsia.googlesource.com/jiri/+/HEAD/scripts/bootstrap_jiri?format=TEXT" | base64 --decode | bash -s "$KERNSRC"
export PATH="$KERNSRC/.jiri_root/bin:$PATH"
```

(optional) get the specific commit of fuchsia, e.g. last commit in releases/f30 branch in 2025.12:
```bash
cd $KERNSRC
git clone -b releases/f30 https://fuchsia.googlesource.com/integration
git -C integration log --first-parent --before="2026-01-01T00:00:00" -1 releases/f30
rm -rf integration
```

change version (e.g. 4c117a94d1cafc3f48d0f3deee07676f6501a0af in releases/f30 branch, also the last commit before 2026.1):
```bash
cd $KERNSRC
jiri init -partial=true .
jiri import \
	-name=integration \
	-revision=4c117a94d1cafc3f48d0f3deee07676f6501a0af \
	-remote-branch=releases/f30 \
	flower \
	https://fuchsia.googlesource.com/integration
jiri update
```

refresh environment:
```bash
cd $KERNSRC
source scripts/fx-env.sh && fx-update-path
```

patch fuchsia:
```bash
cd $KERNSRC
git apply $CLOUD/patch/fuchsia/f30.patch
```

build artifacts for starnix:
```bash
cd $KERNSRC && mkdir local
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
fx build
```

build syzkaller:
```bash
cd $CLOUD/syzkaller
make TARGETOS=linux TARGETARCH=amd64
```

start fuzzing:
```bash
cd $CLOUD
cat <<__EOF__ > $WORKDIR/starnix.cfg
{
    "target": "linux/amd64",
    "http": "127.0.0.1:56741",
    "workdir": "$WORKDIR",
    "kernel_obj": "$KERNSRC/out/x64/exe.unstripped/starnix_kernel",
    "kernel_src": "$KERNSRC",
    "syzkaller": "$CLOUD/syzkaller",
    "procs": 1,
    "type": "starnix",
    "cover": true,
    "vm": {
        "count": 1
    }
}

__EOF__

$CLOUD/syzkaller/bin/syz-manager -config=$WORKDIR/starnix.cfg
```
