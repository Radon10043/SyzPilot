# SyzPilot

Thank you for visiting the SyzPilot repository. This document explains how to use SyzPilot to synthesize specifications and run fuzzing for the Linux kernel. SyzPilot also supports [FreeBSD](docs/freebsd.md), [OpenBSD](docs/openbsd.md), and [NetBSD](docs/netbsd.md). Some commands in this document apply only to anonymous review and may differ from those in the release version.

We have uploaded all intermediate data (~30 GB, including LLM query records and fuzzing results) to [Google Drive](https://workspace.google.com/products/drive/). However, we cannot release the link currently since it may leak author information. We will release the link as soon as the paper is accepted.

**NOTE1:** The documentation is still being improved, and some content may contain typos. We are doing our best to review and fix them :)

**NOTE2:** "cloud" is an alias for SyzPilot. If "cloud" appears in the documentation, you can simply replace it with "SyzPilot".

Please replace the following variables according to your environment:
- `$SYZPILOT`: directory for saving the SyzPilot source code.
- `$KERNSRC`: directory for saving the kernel source code.

## Build the Docker image

We recommend running SyzPilot with Docker. You can build the Docker image with the following command:
```bash
wget -O Dockerfile https://anonymous.4open.science/api/repo/SyzPilot/file/docker/Dockerfile?v=ed2d2c86&download=true
docker build -t syzpilot:latest --network host -f ./Dockerfile .
```

Start a container and setup SyzPilot:
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

We recommend downloading fuzzers, kernels, images, and other artifacts to the mounted `/vol` directory for continuous storage :)

## Build and run SyzPilot

This section explains how to setup and use SyzPilot for syscall spec synthesis. All commands below are executed inside the container.

If you do not want to re-synthesize specs, you can [reuse our synthesized specs](#reuse-synthesized-specs), which are also used in our evaluation.

### Download SyzPilot

Download SyzPilot and the associated syzkaller checkout:
```bash
# export SYZPILOT=/vol/SyzPilot
mkdir /vol/SyzPilot
wget -O /vol/SyzPilot/src.zip https://anonymous.4open.science/api/repo/SyzPilot/zip
unzip src.zip && rm src.zip
git clone https://github.com/google/syzkaller
cd syzkaller && git checkout ac3c71e7063b1fc3b1ede9f76fd3c3b4ce072219
```

### Build SyzPilot

SyzPilot can be built with the following commands:
```bash
cd $SYZPILOT
make
```

### Analyze kernel

SyzPilot needs to analyze the kernel and construct the corresponding knowledge base. Here, we use Linux v6.18 as an example:
```bash
# export KERNSRC=/vol/linux/v6.18
git clone --depth 1 -b v6.18 https://github.com/torvalds/linux $KERNSRC
cp $SYZPILOT/configs/kernel/linux.config $KERNSRC/.config
cd $KERNSRC
make CC="ccache clang" olddefconfig modules_prepare all -j16
python3 scripts/clang-tools/gen_compile_commands.py
```

Analyze the kernel compile commands and construct the knowledge base:
```bash
cd $SYZPILOT
./bin/analyzer -i $KERNSRC/compile_commands.json -o data/database/linux.db -j 16 > logs/analyze.log 2>&1
```

Flags of `analyzer`:
- `-i`: path to `compile_commands.json`
- `-o`: path to the output database (default: ./data/kernel.db)
- `-j`: number of parallel jobs (default: 1)

### Synthesize specs in an agentic manner

Set up the `.env` file:
```bash
cd $SYZPILOT
echo "OPENAI_BASE_URL=[YOUR_BASE_URL]" > .env
echo "OPENAI_API_KEY=[YOUR_API_KEY]" >> .env
```

Prepare a reference file for spec synthesis. Here, we use `dvb_frontend_fops` from the Linux DVB subsystem as an example:
```bash
cd $SYZPILOT
mkdir workdir
echo "variable,dvb_frontend_fops" > workdir/ref.txt
```

Manually enumerating all syscall related elements is too tedious, you can use SyzPilot's [minitask](#minitask) tool to automatically filter related elements and list the syscalls whose specs need to be synthesized.

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

**CAUTION:** Watch the token costs during synthesis!

Flags of `generator`:
- required:
    - `-model`: model to query, e.g. gemini-3-flash-preview
    - `-db`: path to the kernel knowledge database produced in the [Analyze kernel](#analyze-kernel) section
    - `-outdir`: output path for specs synthesized by SyzPilot
    - `-kernel`: path to the kernel used for spec validation
    - `-os`: target OS type. Currently supported values are linux, freebsd, openbsd, and netbsd.
    - `-ref`: path to the file containing reference global variables or functions for spec synthesis.
- optional:
    - `-env`: path to the .env file (default: `$PWD/.env`)
    - `-extract-bin`: path to `syz-extract` (default: `$PWD/bin/syz-extract`)
    - `-check-bin`: path to `syz-check` (default: `$PWD/bin/syz-check`)
    - `-sysdir`: path to a directory such as `syzkaller/sys`. SyzPilot reuses specs under `-sysdir` to avoid duplicate synthesis of common flags, syscalls, and similar elements. (default: `$PWD/syzkaller/sys`)
    - `-resume`: whether to resume previous progress (default: true)
    - `-max-fix`: maximum number of attempts to fix a generated spec (default: 5)
    - `-max-retry`: maximum number of outline-generate-fix attempts. `-1` means unlimited retries. (default: 5)
    - `-jobs`: number of parallel jobs (default: 1)
    - `-otl-system-prompt`: path to outline prompt file(s). Use commas to separate multiple files. (default: `$PWD/data/prompts/outline/instruction.md,$PWD/data/prompts/outline/example_media.md,$PWD/data/prompts/outline/example_ppp.md`)
    - `-gen-system-prompt`: path to generation prompt file(s). Use commas to separate multiple files. (default: `$PWD/data/prompts/generate/instruction.md,$PWD/data/prompts/generate/example_media.md,$PWD/data/prompts/generate/example_ppp.md`)
    - `-fix-system-prompt`: path to fix prompt file(s). Use commas to separate multiple files. (default: `$PWD/data/prompts/fix/instruction.md,$PWD/data/prompts/fix/example_v4l2.md`)

### Refactor synthesized specs

Refactor synthesized specs by adding a unique suffix to each element to avoid conflicts:
```bash
cd $SYZPILOT
./bin/refactor -indir=./workdir/specs -outdir=./workdir/refactored
```

(Optional) Add `meta arches["amd64"]` to limit the scope of the specs:
```bash
sed -i '1i meta arches["amd64"]' workdir/refactored/*.txt
```

**TODO:** Currently, the variable or function name is added as a suffix to each spec element, e.g. `ioctl$ABC` -> `ioctl$ABC_dvb_frontend_fops`. However, this refactoring can be inconvenient for subsystem fuzzing since we have to list the full names of all synthesized syscalls to distinguish them from syzkaller's existing syscalls. We are considering a more suitable refactoring method.

### Integrate synthesized specs into syzkaller

**NOTE:** During integration, some errors in the synthesized specs may need to be fixed manually. This typically involves adjusting the order of include files and removing unused elements. SyzPilot provides [several utility tools](#utility-tools) to help fix these errors.

Patch syzkaller to support additional constant extraction:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
```

Integrate the specs with syzkaller:
```bash
cd $SYZPILOT/syzkaller
cp ../workdir/refactored/* sys/linux
make bin/syz-extract
ls sys/linux/gen*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
make generate
```

### Fuzzing with synthesized specs

Create a Debian Bullseye image for fuzzing:
```bash
mkdir -p /vol/images/Debian && cd /vol/images/Debian
cp $SYZPILOT/scripts/linux/create-image.sh .
chmod +x ./create-image.sh
./create-image.sh
```

Start fuzzing with the synthesized specifications:
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

`minitask` selects global variables associated with syscalls by using string matching, then lists all syscalls that require specifications. It helps avoid the tedious process of manually enumerating syscall related elements and prevents duplicate spec synthesis for the same syscall. **You can directly run `generator` based on the output of `minitask` for optimal efficiency.**

Build:
```bash
make minitask   # it will also be built via `make all`
```

Setup the `.env` file if you have not already done so:
```bash
echo "OPENAI_BASE_URL=[YOUR_BASE_URL]" > .env
echo "OPENAI_API_KEY=[YOUR_API_KEY]" >> .env
```

Run `minitask` to select all syscall related elements and enumerate all unique syscalls that require spec synthesis:
```bash
cd $SYZPILOT
./bin/minitask \
    -db=./data/database/linux.db \
    -os=linux \
    -outdir=./workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
```

Extract references:
```bash
./scripts/reflist.sh ./workdir/minitask > ./workdir/minitask/ref.txt
```

### rmunused

Remove unused elements in place:
```bash
$SYZPILOT/bin/rmunused -indir=$SYZPILOT/syzkaller/sys/linux
```

## Reuse synthesized specs

Specifications used in our evaluation are saved under `$SYZPILOT/patch/specs-*`. Feel free to reuse them to avoid duplicate synthesis. Remember to apply the patch that enables syzkaller to support fuzzing the OpenBSD kernel on Linux.

`specs-kern` stores specs for full kernel fuzzing:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-kern/*
```

`specs-subsys` stores specs for subsystem fuzzing. We re-synthesize specs for subsystems that already have specs:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/syzkaller/*
git apply ../patch/specs-subsystem/*
```

## Reproduce evaluation

Please follow [setup-env.md](experiment/docs/setup-env.md) to setup the evaluation environment. You can then reproduce our evaluation using the [Docker Compose files](experiment/docs/docker-compose.md).

## Re-synthesize subsystem specs

Please see [subsystem.md](docs/subsystem.md#how-to-re-synthesize-specs-for-a-subsystem) for instructions on re-synthesizing specs for a subsystem that already has specs.

## Trophies

To ensure anonymity, we will release all links after the paper is accepted.

**NOTE:** We will update the trophies as soon as new specs synthesized by SyzPilot are merged into the syzkaller repository or related issues are assigned CVEs.

### Merged specifications

- sys/linux: add descriptions for mtd and tee subsystems
- sys/netbsd: add and update descriptions for acpi, agp, hdaudio, etc.
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

Bugs related to our generated specifications and reported by syzbot:

- CVE-0000-00000, BUG: corrupted list in io_poll_remove_entries
- CVE-0000-00000, INFO: task hung in nilfs_transaction_begin
- CVE-0000-00000, KMSAN: uninit-value in vidtv_ts_null_write_into
- CVE-0000-00000, general protection fault in nilfs_mdt_save_to_shadow_map
- CVE-0000-00000, memory leak in vidtv_psi_service_desc_init
- CVE-0000-00000, WARNING in nilfs_btree_mark
- CVE-0000-00000, WARNING in nilfs_ioctl_prepare_clean_segments
- BUG: corrupted list in nilfs_lookup_dirty_data_buffers
- INFO: trying to register non-static key in as102_dvb_dmx_start_feed
- KASAN: slab-use-after-free Read in dvb_device_open
- KASAN: slab-use-after-free Read in dvb_frontend_release
- KASAN: slab-use-after-free Read in dvb_frontend_open
- KMSAN: uninit-value in dvbdmx_release_ts_feed
- KMSAN: uninit-value in dvb_demux_read
- WARNING in as102_dvb_dmx_start_feed media
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

Test cases generated by SyzPilot have been incorporated into FreeBSD's test suite:

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
