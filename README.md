# SyzPilot

Thank you for browsing the SyzPilot repository. This document details steps about using SyzPilot to synthesize specifications and run fuzzing for Linux kernel, it also support [FreeBSD](docs/freebsd.md), [OpenBSD](docs/openbsd.md), and [NetBSD](docs/netbsd.md). Some commands in the document are only applicable during anonymous reviewing, which may differ from those in the release version.

**NOTE:** "cloud" is an alias for SyzPilot. If "cloud" appears in the documentation, you can simply replace it with "SyzPilot" :)

Please replace the following variables according to the actual situation:
- `$SYZPILOT`: directory for saveing SyzPilot source.
- `$KERNSRC`: directory for saving kernel source.

## Build docker image

We recommend running SyzPilot using docker, you can build the docker image via following command:
```bash
wget -O Dockerfile https://anonymous.4open.science/api/repo/SyzPilot/file/docker/Dockerfile?v=ed2d2c86&download=true
docker build -t syzpilot:latest --network host -f ./Dockerfile .
```

Let's start a container and setup SyzPilot:
```bash
docker run \
    -d \
    -v ./vol:/vol \
    --cpus 20 \
    --network host \
    --privileged \
    --name syzpilot-test \
    syzpilot:latest tail -f /dev/null
docker exec -it syzpilot-test bash
```

I recommend download fuzzers, kernels, images, etc. to the mounted directory `/vol` for continuous storage :)

## Build & run SyzPilot

We next detail how to setup and use SyzPilot for syscall spec synthesis. Note that all commands below are executed in the container.

If you don't want to re-synthesize specs, you can also [reuse our synthesized specs](#reuse-synthesized-specs), they are also used in our evaluation.

### Download SyzPilot

Download SyzPilot and associated syzkaller:
```bash
# export SYZPILOT=/vol/SyzPilot
mkdir /vol/SyzPilot && cd /vol/SyzPilot
wget -O /vol/SyzPilot/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
cd SyzPilot && unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
```

### Build SyzPilot

SyzPilot can be easily built via following commands:
```bash
cd $SYZPILOT
make
```

### Analyze kernel

We need analyze kernel and construct the corresponding knowledgebase, let's use linux v6.18 as an example:
```bash
# export KERNSRC=/vol/linux/v6.18
git clone --depth 1 -b v6.18 https://github.com/torvalds/linux $KERNSRC
cp $SYZPILOT/configs/kernel/linux.config $KERNSRC/.config
cd $KERNSRC
make CC="ccache clang" olddefconfig modules_prepare all -j16
python3 scripts/clang-tools/gen_compile_commands.py
```

Analyze kernel compile commands and construct knowledge base:
```bash
cd $SYZPILOT
./bin/analyzer -i $KERNSRC/compile_commands.json -o data/database/linux.db -j 16 > logs/analyze.log 2>&1
```

Flags for analyzer:
- `-i`: path to `compile_commands.json`
- `-o`: path to the output database (default: ./data/kernel.db)
- `-j`: number of parallel jobs (default: 1)

### Synthesize specs in an agentic manner

setup .env file:
```bash
cd $SYZPILOT
echo "OPENAI_BASE_URL=[YOUR_BASE_URL]" > .env
echo "OPENAI_API_KEY=[YOUR_API_KEY]" >> .env
```

We need prepare a reference file for spec synthesis, let's use `dvb_frontend_fops` from Linux DVB subsystem as an example:
```bash
cd $SYZPILOT
mkdir workdir
echo "variable,dvb_frontend_fops" > workdir/ref.txt
```

If you find it tedious to manually enumerate all syscall related elements, you can use the tool [minitask](#minitask) provided by SyzPilot to automatically filter related elements and list syscalls whose spec need to be synthesized.

Generate syscall specs:
```bash
cd $SYZPILOT
./bin/generator \
    -db=./data/database/linux.db \
    -outdir=./workdir/minitask \
    -kernel=$KERNSRC \
    -os=linux \
    -model=gemini-3-flash-preview \
    -ref=./workdir/ref.txt \
    -jobs=4 > logs/generate.log 2>&1
```

**CAUTION:** Please watch out the token costs during synthesis!

Description for flags of generator:
- required:
    - `-model`: model to be queried, e.g. gemini-3-flash-preview
    - `-db`: path to the kernel knowledge database, which is produced by following [Analyze kernel](#analyze-kernel) section
    - `-outdir`: output path for saving specs synthesized by SyzPilot
    - `-kernel`: path to kernel for spec validation
    - `-os`: target OS type, currently support linux, freebsd, openbsd, and netbsd.
    - `-ref`: path to file includes reference global variables or functions for spec synthesis.
- optional:
    - `-env`: path to the .env file (default: `$PWD/.env`)
    - `-extract-bin`: path to `syz-extract` (default: `$PWD/bin/syz-extract`)
    - `-check-bin`: path to `syz-check` (default: `$PWD/bin/syz-check`)
    - `-sysdir`: path to directory like `syzkaller/sys`, SyzPilot will reuse specs under `-sysdir` to avoid duplicate synthesis of some common flags, syscalls, etc.. (default: `$PWD/syzkaller/sys`)
    - `-resume`: whether resume previous progress (default: true)
    - `-max-fix`: max attempts for fixing generated spec (default: 5)
    - `-max-retry`: max attempts for performing outline-generate-fix, -1 means inifinity tries (default: 5)
    - `-jobs`: number of parallel jobs (default: 1)
    - `-otl-system-prompt`: path to file(s) for outline prompt, use comma to separate multiple files (default: `$PWD/data/prompts/outline/instruction.md,$PWD/data/prompts/outline/example_media.md,$PWD/data/prompts/outline/example_ppp.md`)
    - `-gen-system-prompt`: path to file(s) for generate prompt, use comma to separate multiple files (default: `$PWD/data/prompts/generate/instruction.md,$PWD/data/prompts/generate/example_media.md,$PWD/data/prompts/generate/example_ppp.md`)
    - `-fix-system-prompt`: path to file(s) for fix prompt, use comma to separate multiple files (default: `$PWD/data/prompts/fix/instruction.md,$PWD/data/prompts/fix/example_v4l2.md`)

### Refactor synthesized specs

Refactor synthesized specs, add unique suffix to each elements to avoid conflicts among syntheszied specs:
```bash
cd $SYZPILOT
./bin/refactor -indir=./workdir/specs -outdir=./workdir/refactored
```

(OptionaL) Add `meta arches["amd64"]` to limit the scope of the specs:
```bash
sed -i '1i meta arches["amd64"]' workdir/refactored/*.txt
```

**TODO:** Currently variable/function name is added as suffix to each spec elements, e.g. `iocrl$ABC` -> `ioctl$ABC_dvb_frontend_fops`, but such refactoring may inconvenient for subsystem fuzzing, we have to list the full names of all synthesized syscalls to distinguish them from syscalls of syzkaller. We are currently considering a more reasonable refactoring method.

### Integrate synthesized specs to syzkaller

**NOTE:** During integration, some errors in the synthesized specs may need to fix manually, typically involves adjusting the order of include files and removing unused elements. [Several utility tools](#utility-tools) is provided by SyzPilot to help fix errors.

Patch syzkaller to support some const value extraction:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
```

Integrate specs with syzkaller:
```bash
cd $SYZPILOT/syzkaller
cp ../workdir/refactored/* sys/linux
make bin/syz-extract
ls sys/linux/cloud*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
make generate
```

### Fuzzing with synthesized specs

Create a Debian bullseye image to support fuzzing:
```bash
mkdir -p /vol/images/Debian && cd /vol/images/Debian
cp $SYZPILOT/scripts/linux/create-image.sh .
chmod +x ./create-image.sh
./create-image.sh
```

Start fuzzing with synthesized specs:
```bash
cd $SYZPILOT
cat <<__EOF__ > workdir/fuzz.cfg
{
	"target": "linux/amd64",
	"http": "127.0.0.1:56741",
	"workdir": "$SYZPILOT/workdir",
	"kernel_obj": "$KERNSRC",
	"image": "$IMAGE/bullseye.img",
	"sshkey": "$IMAGE/bullseye.id_rsa",
	"syzkaller": "$SYZPILOT/syzkaller",
	"procs": 8,
	"type": "qemu",
	"reproduce": false,
	"vm": {
		"count": 4,
		"kernel": "$KERNSRC/arch/x86/boot/bzImage",
		"cpu": 8,
		"mem": 2048
	}
}
__EOF__

./syzkaller/bin/syz-manager -config=./workdir/fuzz.cfg
```

## Utility tools

### minitask

`minitask` selects global variables that associated with syscalls by string matching and lists all syscalls that require specification. By utilizing it, we can avoid tedious of manually enumerate syscall related elements and avoid duplicate spec synthesis for the same syscall.

Build:
```bash
make minitask   # it will also be built via `make all`
```

Setup .env file if you haven't already:
```bash
echo "OPENAI_BASE_URL=[YOUR_BASE_URL]" > .env
echo "OPENAI_API_KEY=[YOUR_API_KEY]" >> .env
```

Run minitask to select all syscall related elements and enumerate all unique syscalls requiring spec synthesis:
```bash
cd $SYZPILOT
./bin/minitask \
    -db=./data/database/linux.db \
    -os=linux \
    -outdir=./workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
```

extract references:
```bash
./scripts/reflist.sh ./workdir/minitask > ./workdir/minitask/ref.txt
```

### rmunused

Remove unused elements inplace:
```bash
$SYZPILOT/bin/rmunused -indir=$SYZPILOT/syzkaller/sys/linux
```

## Reuse synthesized specs

Specifications used in our evaluation are saved under `$SYZPILOT/patch/specs-*`, feel free to reuse them to avoid duplicate synthesis. Remember to apply the patch to enable syzkaller to support fuzzing OpenBSD kernel on Linux.

`specs-kern` saves specs for full kernel fuzzing:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
```

`specs-subsys` save specs for subsystem fuzzing, we re-synthesize specs for those subsystems that already exist spec:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-subsystem/*
```

## Reproduce evaluation

please follow [env-setup.md](experiment/docs/env-setup.md) to setup evaluation environment, and then you can reproduce our evaluation via [docker compose files](experiment/docs/docker-compose.md) easily.

## Re-synthesize subsystem specs

Please see [subsystem.md](docs/subsystem.md#how-to-re-generate-specs-for-other-subsystem) to check how to re-synthesize specs for a subsystem that already have specs.

## Trophies

To ensure anonymity, we will release all links after the paper is accepted.

### Merged specifications

- sys/freebsd: generate headers for const extraction and add syscall descriptions
- sys/openbsd: update wscons.txt and add dev_dri.txt
- sys/freebsd: add descriptions for acpi, apm, and auditpipe devices
- sys/linux: update syscall descriptions for multiple file systems
- Add descriptions for XFS subsystem
- sys/linux: add descriptions for dvb subsystem
- sys/linux: update flags in dev_video4linux.txt
- sys/linux: add v4l2_meta_format

### Linux bugs

- CVE-0000-00000, WARNING in find_free_extent
- CVE-0000-00000, general protection fault in xchk_btree
- CVE-0000-00000, general protection fault in xchk_metadata_inode_forks
- CVE-0000-00000, general protection fault in xfarray_destroy
- CVE-0000-00000, general protection fault in alloc_file_pseudo
- CVE-0000-00000, KASAN slab-use-after-free Read in xchk_btree_check_block_owner
- WARNING in iterate_dir
- KASAN: slab-use-after-free Write in dvb_device_open
- KFENCE: use-after-free read in dvb_frontend_release
- KASAN: slab-use-after-free Read in dvb_frontend_thread
- WARNING: still has locks held in _dmxdev_lock
- WARNING: bad unlock balance in _dmxdev_unlock
- possible deadlock in dvb_dvr_release
- possible deadlock in ocfs2_try_to_free_truncate_log (*syzbot report before but cannot generate reproducer*)
- WARNING in exc_debug_kernel
- general protection fault in dvb_device_open

bugs related to our generated specification and reported by syzbot:

- CVE-0000-00000, BUG: corrupted list in io_poll_remove_entries
- CVE-0000-00000, KMSAN: uninit-value in vidtv_ts_null_write_into
- CVE-0000-00000, general protection fault in nilfs_mdt_save_to_shadow_map
- CVE-0000-00000, memory leak in vidtv_psi_service_desc_init
- INFO: task hung in nilfs_transaction_begin
- INFO: trying to register non-static key in as102_dvb_dmx_start_feed
- KASAN: slab-use-after-free Read in dvb_frontend_release
- KMSAN: uninit-value in dvbdmx_release_ts_feed
- KMSAN: uninit-value in dvb_demux_read
- WARNING in as102_dvb_dmx_start_feed media
- WARNING in nilfs_btree_mark
- WARNING in nilfs_ioctl_prepare_clean_segments
- general protection fault in bio_add_page
- general protection fault in bio_alloc_bioset
- memory leak in dvb_register_device
- memory leak in vidtv_psi_short_event_desc_init

### FreeBSD bugs

- Fatal trap NUM: general protection fault while in kernel mode in cam_periph_runccb
- Fatal trap NUM: page fault while in kernel mode in cam_periph_runccb
- Fatal trap NUM: page fault while in kernel mode in passdoioctl
- Fatal trap NUM: page fault while in kernel mode in passsendccb
- panic: ata_action: ccb ADDR, func_code CODE should not be allocated from UMA zone
- panic: AUX register unsupported
- panic: cam_periph_ccbwait: proceeding with incomplete ccb
- panic: dst_m ADDR is not wired
- panic: _free(NUM): address ADDR(ADDR) has not been allocated
- panic: mutex ACPI global lock owned at ../../../kern/kern_event.c:LINE

test cases generated by SyzPilot are incorporated into FreeBSD's test suite:

- stress2: Added syzkaller reproducers. Update the exclude file

### OpenBSD bugs

- uvm_fault: dovutimens
- uvm_fault: lptpushbytes

### NetBSD bugs

- assert failed: chp->ch_drive[drive].drv_softc == NULL
- assert failed: hispgrp->pg_jobc > NUM
- assert failed: it->it_time.it_value.tv_sec >= NUM
- assert failed: kn->kn_fop == &proc_filtops
- assert failed: kq->kq_fdp == fdp
- assert failed: ks->ks_pshared_proc == NULL
- assert failed: ps->ps_endoffset != endoffset
- assert failed: sc->sc_base.me_evp != NULL
- assert failed: ts->tv_nsec >= NUM
- assert failed: uio->uio_iovcnt > NUM
- panic: ASan: Unauthorized Access In ADDR: Addr ADDR [ADDR bytes, read, KmemRedZone]
- panic: ASan: Unauthorized Access In ADDR: Addr ADDR [NUM byte, read, KmemRedZone]
- panic: LOCKDEBUG: Mutex error: rw_vector_enter,NUM: spin lock held
