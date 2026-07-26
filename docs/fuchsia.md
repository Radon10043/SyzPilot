# setup cloud and fuzzing fuchsia kernel

## setup fuzzer

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
source scripts/fx-env.sh && fx-update-path
```

checkout third-party/syzkaller to ac3c71e7, patch fuchsia and syzkaller to enable coverage-guided fuzzing:
```bash
cd $KERNSRC
git -C third_party/syzkaller fetch
git -C third_party/syzkaller checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
git apply $CLOUD/patch/fuchsia/f30.patch
git -C third_party/syzkaller apply $CLOUD/patch/syzkaller/*
```

build syzkaller first:
```bash
cd $KERNSRC/third_party/syzkaller
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR=$KERNSRC
```

build fuchsia for x64 and arm64:
```bash
cd $KERNSRC

fx --dir "out/x64" set core.x64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j16

fx --dir "out/arm64" set core.arm64 \
    --with "//bundles/tools" \
    --with "//src/testing/fuzzing/syzkaller" \
    --include-clippy=false \
    --variant=kasan-sancov
fx build -j16
```

refresh artifacts for fuzzing, you need refresh after each `fx build`:
```bash
$CLOUD/scripts/fuchsia/refresh.sh -f $KERNSRC
```

patch syzkaller and startup fuzzing:
```bash
cd $CLOUD/syzkaller
git apply ../patch/syzkaller/*
make TARGETOS=fuchsia TARGETARCH=amd64 SOURCEDIR=$KERNSRC
```

startup fuzzing:
```bash
export PATH=$KERNSRC/prebuilt/third_party/qemu/linux-x64/bin:$PATH
cat <<__EOF__ > $CLOUD/workdir/fuchsia.cfg
{
	"name": "fuchsia",
	"target": "fuchsia/amd64",
	"http": ":12345",
	"workdir": "$CLOUD/workdir",
	"kernel_obj": "$KERNSRC/out/x64/syzkaller/obj",
	"syzkaller": "$CLOUD/syzkaller",
	"image": "$KERNSRC/out/x64/syzkaller/fxfs.blk",
	"sshkey": "$KERNSRC/out/x64/syzkaller/fuchsia_ed25519",
	"reproduce": false,
	"cover": true,
	"procs": 8,
	"type": "qemu",
	"vm": {
		"count": 4,
		"cpu": 2,
		"mem": 4096,
		"kernel": "$KERNSRC/out/x64/syzkaller/kernel",
		"initrd": "$KERNSRC/out/x64/syzkaller/fuchsia-ssh.zbi",
		"image_device": "drive if=none,format=raw,id=vdisk,cache=unsafe,file=",
		"network_device": "virtio-net-pci",
		"qemu_args": "-enable-kvm -machine q35,smbios-entry-point-type=32 -cpu host,migratable=off -object iothread,id=iothread0 -device virtio-blk-pci,drive=vdisk,iothread=iothread0"
	}
}
__EOF__
$CLOUD/syzkaller/bin/syz-manager -config=$CLOUD/workdir/fuchsia.cfg
```

## spec generation

syzkaller has provided a complete framework to generate syscall specifications for fuchsia, cloud just fix some errors that exists in generated specs is okay.

Let's generate specs for bluetooth subsystem:
```bash
cd $KERNSRC
./out/x64/host_x64/fidlgen_syzkaller -json ./out/x64/fidling/gen/sdk/fidl/fuchsia.bluetooth/fuchsia.bluetooth.fidl.json -output-syz $WORKDIR/fidlsyz/fuchsia_bluetooth.syz.txt
```

Or, generate specs for subsystems under `$KERNSRC/out/x64/fidling/gen/sdk/fidl`:
```bash
$CLOUD/scripts/fuchsia/fidlgen.sh -f $KERNSRC -o $WORKDIR/fidlsyz
```

Refine necessary artifacts from fuchsia directory for spec fixing:
```bash
$CLOUD/scripts/fuchsia/refine.sh -f $KERNSRC -o $KERNSRC/mini-fuchsia
```

Run cloud to fix and improve them:
```bash
cd $CLOUD
./bin/generator-fuchsia \
	-os=fuchsia \
	-model=gemini-3-flash-preview \
	-kernel=$KERNSRC/mini-fuchsia \
	-outdir=$WORKDIR/generator-fuchsia \
	-fidlsyz-dir=$WORKDIR/fidlsyz \
	-jobs=4
```

## useful links

- https://fuchsia.dev/fuchsia-src/
- https://fuchsia.googlesource.com/fuchsia/
