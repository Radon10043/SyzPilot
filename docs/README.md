# SyzPilot

SyzPilot is a state-guided agentic syscall specification synthesizer, designed for [syzlang](https://github.com/google/syzkaller/blob/master/docs/syscall_descriptions.md).

This document explains how to use SyzPilot to synthesize syscall specifications and run fuzzing for the Linux kernel. SyzPilot also supports [FreeBSD](freebsd.md), [OpenBSD](openbsd.md), [NetBSD](netbsd.md), and [Android](android.md). The synthesized specs can also be used to fuzz [gVisor](gvisor.md) and [Starnix](starnix.md).

> [!NOTE]
> The documentation is still being improved, and some content may contain typos. We are doing our best to review and fix them :)

Please replace the following variables according to your environment:
- `$SYZPILOT`: directory for saving the SyzPilot source code.
- `$KERNSRC`: directory for saving the kernel source code.
- `$IMAGE`: directory for saving the vm image used for fuzzing.

## Build the Docker image

We recommend running SyzPilot with Docker. You can build the Docker image with the following commands:
```bash
wget -O Dockerfile https://raw.githubusercontent.com/Radon10043/SyzPilot/main/docker/Dockerfile
docker build -t syzpilot:latest --network host -f ./Dockerfile .
```

Start a container and enter it:
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

We recommend downloading fuzzers, kernels, images, and other artifacts to the mounted `/vol` directory for persistent storage :)

## Build and run SyzPilot

This section explains how to set up and use SyzPilot for syscall spec synthesis. All commands below are executed inside the container.

If you do not want to re-synthesize specs, you can [reuse our synthesized specs](#reuse-synthesized-specs), which are also used in our evaluation.

### Download SyzPilot

Download SyzPilot together with its syzkaller submodule:
```bash
export SYZPILOT=/vol/SyzPilot
git clone --recurse-submodules https://github.com/Radon10043/SyzPilot $SYZPILOT
# if you forgot to clone with --recurse-submodules, run `git submodule update --init --recursive` under $SYZPILOT
```

Patch syzkaller to support additional constant extraction. SyzPilot validates synthesized specs with the `syz-extract` built from this syzkaller checkout, so apply the patches before building SyzPilot:
```bash
cd $SYZPILOT/syzkaller
git apply -3 ../patch/syzkaller/*
git reset .
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
export KERNSRC=/vol/linux/v6.18
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

Setup the `.env` file:
```bash
cd $SYZPILOT
echo "OPENAI_BASE_URL=[YOUR_BASE_URL]" > .env
echo "OPENAI_API_KEY=[YOUR_API_KEY]" >> .env
```

Prepare a reference file for spec synthesis. Here, we use `dvb_frontend_fops` from the Linux DVB subsystem as an example:
```bash
cd $SYZPILOT
mkdir .workdir
echo "variable,dvb_frontend_fops" > .workdir/ref.txt
```

Manually enumerating all syscall related elements is tedious. You can use SyzPilot's [minitask](#minitask) tool to automatically filter related elements and list the syscalls whose specs need to be synthesized.

Synthesize syscall specs:
```bash
cd $SYZPILOT
./bin/generator \
    -db=./data/database/linux.db \
    -outdir=./.workdir \
    -kernel=$KERNSRC \
    -os=linux \
    -model=gemini-3-flash-preview \
    -ref=./.workdir/ref.txt \
    -jobs=4 > logs/generate.log 2>&1
```

The synthesized specs are saved under `./.workdir/specs`.

> [!CAUTION]
> Watch the costs during synthesis!

Flags of `generator`:
- required:
    - `-model`: model to query, e.g. gemini-3-flash-preview
    - `-db`: path to the kernel knowledge database produced in the [Analyze kernel](#analyze-kernel) section
    - `-outdir`: output path for specs synthesized by SyzPilot
    - `-kernel`: path to the kernel used for spec validation
    - `-os`: target OS type. Currently supported values are linux, freebsd, openbsd, netbsd, and android.
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
./bin/refactor -indir=./.workdir/specs -outdir=./.workdir/refactored
```

(Optional) Add `meta arches["amd64"]` to limit the scope of the specs:
```bash
sed -i '1i meta arches["amd64"]' .workdir/refactored/*.txt
```

**TODO:** Currently, the variable or function name is added as a suffix to each spec element, e.g. `ioctl$ABC` -> `ioctl$ABC_dvb_frontend_fops`. However, this refactoring can be inconvenient for subsystem fuzzing since we have to list the full names of all synthesized syscalls to distinguish them from syzkaller's existing syscalls. We are considering a more suitable refactoring method.

### Integrate synthesized specs into syzkaller

> [!NOTE]
> During integration, some errors in the synthesized specs may need to be fixed manually. This typically involves adjusting the order of include files and removing unused elements. SyzPilot provides [several utility tools](#utility-tools) to help fix these errors.

Integrate the specs with syzkaller, extract constants, and build syzkaller:
```bash
cd $SYZPILOT/syzkaller
cp ../.workdir/refactored/* sys/linux
make bin/syz-extract
ls sys/linux/gen#*.txt | xargs -n 1 basename | xargs ./bin/syz-extract -build -sourcedir=$KERNSRC -os=linux -arch=amd64
make generate
make all -j16
```

### Fuzzing with synthesized specs

Create a Debian Bullseye image for fuzzing:
```bash
export IMAGE=/vol/images/Debian
mkdir -p $IMAGE && cd $IMAGE
cp $SYZPILOT/scripts/linux/create-image.sh .
chmod +x ./create-image.sh
./create-image.sh
```

Start fuzzing with the synthesized specifications:
```bash
cd $SYZPILOT
cat <<__EOF__ > .workdir/fuzz.cfg
{
	"target": "linux/amd64",
	"http": "127.0.0.1:56741",
	".workdir": "$SYZPILOT/.workdir",
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

./syzkaller/bin/syz-manager -config=./.workdir/fuzz.cfg
```

## Utility tools

### minitask

`minitask` selects global variables associated with syscalls by using string matching, then lists all syscalls that require specifications. It helps avoid the tedious process of manually enumerating syscall related elements and prevents duplicate spec synthesis for the same syscall. **You can directly run `generator` based on the output of `minitask` for optimal efficiency.**

Build:
```bash
make minitask   # it will also be built via `make all`
```

Set up the `.env` file if you have not already done so:
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
    -outdir=./.workdir/minitask \
    -model=gemini-3-flash-preview > logs/minitask.log 2>&1
```

Extract references:
```bash
./scripts/reflist.sh ./.workdir/minitask > ./.workdir/minitask/ref.txt
```

Then run `generator` with `-outdir=./.workdir/minitask -ref=./.workdir/minitask/ref.txt` to reuse the results of `minitask`.

### rmunused

Remove unused elements in place:
```bash
$SYZPILOT/bin/rmunused -indir=$SYZPILOT/syzkaller/sys/linux
```

## Reuse synthesized specs

Specifications used in our evaluation are saved under `$SYZPILOT/patch/specs-*`. Feel free to reuse them to avoid duplicate synthesis. Remember to apply the syzkaller patches first (see [Download SyzPilot](#download-syzpilot)); they also enable syzkaller to fuzz the OpenBSD kernel on Linux.

`specs-kern` stores specs for full kernel fuzzing:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/specs-kern/*
```

`specs-subsys` stores specs for subsystem fuzzing. We re-synthesize specs for subsystems that already have specs:
```bash
cd $SYZPILOT/syzkaller
git apply ../patch/specs-subsys/*
```

`specs-ablation` stores specs synthesized by the variants used in our [ablation study](ablation.md).

## Reproduce evaluation

Please follow [setup-env.md](../experiment/docs/setup-env.md) to set up the evaluation environment. You can then reproduce our evaluation using the [Docker Compose files](../experiment/docs/docker-compose.md).

## Re-synthesize subsystem specs

Please see [subsystem.md](subsystem.md#how-to-re-synthesize-specs-for-a-subsystem) for instructions on re-synthesizing specs for a subsystem that already has specs.

## Trophies

> [!NOTE]
> We will update the trophies as soon as new specs synthesized by SyzPilot are merged into the syzkaller repository or related issues are assigned CVEs.

### Merged specifications

- [sys/linux: add descriptions for mtd and tee subsystems](https://github.com/google/syzkaller/pull/7545)
- [sys/netbsd: add and update descriptions for acpi, agp, hdaudio, etc.](https://github.com/google/syzkaller/pull/7277)
- [sys/freebsd: generate headers for const extraction and add syscall descriptions](https://github.com/google/syzkaller/pull/6961)
- [sys/openbsd: update wscons.txt and add dev_dri.txt](https://github.com/google/syzkaller/pull/6943)
- [sys/freebsd: add descriptions for acpi, apm, and auditpipe devices](https://github.com/google/syzkaller/pull/6922)
- [sys/linux: update syscall descriptions for multiple file systems](https://github.com/google/syzkaller/pull/6875)
- [Add descriptions for XFS subsystem](https://github.com/google/syzkaller/pull/6790)
- [sys/linux: add descriptions for dvb subsystem](https://github.com/google/syzkaller/pull/6746)
- [sys/linux: update flags in dev_video4linux.txt](https://github.com/google/syzkaller/pull/6555)
- [sys/linux: add v4l2_meta_format](https://github.com/google/syzkaller/pull/6525)

### Linux bugs

- [CVE-2026-23214](https://lore.kernel.org/linux-cve-announce/2026021800-CVE-2026-23214-c822@gregkh/), [WARNING in find_free_extent](https://groups.google.com/g/syzkaller/c/4_Wizixja3I)
- [CVE-2026-23249](https://lore.kernel.org/linux-cve-announce/2026031843-CVE-2026-23249-c309@gregkh/), [general protection fault in xchk_btree](https://groups.google.com/g/syzkaller/c/PdqG_MarO5Y)
- [CVE-2026-23250](https://lore.kernel.org/linux-cve-announce/2026031845-CVE-2026-23250-271e@gregkh/), [general protection fault in xchk_metadata_inode_forks](https://groups.google.com/g/syzkaller/c/RsEEzP_yxlc/m/rE0a7FDoAQAJ)
- [CVE-2026-23251](https://lore.kernel.org/linux-cve-announce/2026031845-CVE-2026-23251-259a@gregkh/), [general protection fault in xfarray_destroy](https://groups.google.com/g/syzkaller/c/CIKKUTDIRq4)
- [CVE-2026-23252](https://lore.kernel.org/linux-cve-announce/2026031846-CVE-2026-23252-6bef@gregkh/), [general protection fault in alloc_file_pseudo](https://groups.google.com/g/syzkaller/c/CIKKUTDIRq4)
- [CVE-2026-23223](https://lore.kernel.org/linux-cve-announce/2026021806-CVE-2026-23223-e3d3@gregkh/), [KASAN slab-use-after-free Read in xchk_btree_check_block_owner](https://groups.google.com/g/syzkaller/c/PdqG_MarO5Y/m/RR-4ffJuAgAJ)
- [KASAN: slab-use-after-free Write in dvb_device_open](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [KASAN: slab-use-after-free Read in dvb_frontend_thread](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [KFENCE: use-after-free read in dvb_frontend_release](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [WARNING: still has locks held in _dmxdev_lock](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [WARNING: bad unlock balance in _dmxdev_unlock](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [WARNING in iterate_dir](https://groups.google.com/g/syzkaller/c/psHLMX5Drko/m/CRsGM-3fCQAJ)
- [WARNING in exc_debug_kernel](https://groups.google.com/g/syzkaller/c/PoXFwx7Qw3w/m/ywb0H7Y_BAAJ)
- [general protection fault in dvb_device_open](https://groups.google.com/g/syzkaller/c/h3rqEuibYtk/m/5g2nKlWLAAAJ)
- [possible deadlock in dvb_dvr_release](https://groups.google.com/g/syzkaller/c/gopfrRcxp4w/m/yBhPduWTAwAJ)
- [possible deadlock in ocfs2_try_to_free_truncate_log](https://groups.google.com/g/syzkaller/c/Pjrgfrlrk68/m/5IfA3Y_SAQAJ) (*syzbot report before but cannot generate reproducer*)

Bugs related to our generated specifications and reported by syzbot:

- [CVE-2026-23253](https://lore.kernel.org/linux-cve-announce/2026031846-CVE-2026-23253-b1c6@gregkh/), [BUG: corrupted list in io_poll_remove_entries](https://syzkaller.appspot.com/bug?extid=ab12f0c08dd7ab8d057c)
- [CVE-2026-64359](https://nvd.nist.gov/vuln/detail/CVE-2026-64359), [INFO: task hung in nilfs_transaction_begin](https://syzkaller.appspot.com/bug?extid=62f0f99d2f2bb8e3bbd7)
- [CVE-2026-31577](https://lore.kernel.org/linux-cve-announce/2026042410-CVE-2026-31577-5e81@gregkh/), [general protection fault in nilfs_mdt_save_to_shadow_map](https://syzkaller.appspot.com/bug?extid=4b4093b1f24ad789bf37)
- [CVE-2026-31585](https://lore.kernel.org/linux-cve-announce/2026042413-CVE-2026-31585-b423@gregkh/), [memory leak in vidtv_psi_service_desc_init](https://syzkaller.appspot.com/bug?extid=639ebc6ec75e96674741)
- [CVE-2026-43058](https://lore.kernel.org/linux-cve-announce/2026050254-CVE-2026-43058-4a86@gregkh/), [KMSAN: uninit-value in vidtv_ts_null_write_into](https://syzkaller.appspot.com/bug?extid=96f901260a0b2d29cd1a)
- [CVE-2026-53320](https://lore.kernel.org/linux-cve-announce/2026062622-CVE-2026-53320-a56a@gregkh/), [WARNING in nilfs_btree_mark](https://syzkaller.appspot.com/bug?extid=98a040252119df0506f8)
- [CVE-2026-53320](https://lore.kernel.org/linux-cve-announce/2026062622-CVE-2026-53320-a56a@gregkh/), [WARNING in nilfs_ioctl_prepare_clean_segments](https://syzkaller.appspot.com/bug?extid=466a45fcfb0562f5b9a0)
- [BUG: corrupted list in nilfs_lookup_dirty_data_buffers](https://syzkaller.appspot.com/bug?extid=c37bed40868932d790e9)
- [INFO: trying to register non-static key in as102_dvb_dmx_start_feed](https://syzkaller.appspot.com/bug?extid=3f395d8da879a58fb019)
- [KASAN: slab-use-after-free Read in dvb_device_open](https://syzkaller.appspot.com/bug?extid=1eb177ecc3943b883f0a)
- [KASAN: slab-use-after-free Read in dvb_frontend_release](https://syzkaller.appspot.com/bug?extid=ae466a728017ec940b41)
- [KASAN: slab-use-after-free Read in dvb_frontend_open](https://syzkaller.appspot.com/bug?extid=40339ea82afa8184ad5d)
- [KMSAN: uninit-value in dvbdmx_release_ts_feed](https://syzkaller.appspot.com/bug?extid=01d4620886bee3db0e74)
- [KMSAN: uninit-value in dvb_demux_read](https://syzkaller.appspot.com/bug?extid=bd7c90de4c9f1f8ab660)
- [WARNING in as102_dvb_dmx_start_feed media](https://syzkaller.appspot.com/bug?extid=3825a6102073c418fe41)
- [general protection fault in bio_add_page](https://syzkaller.appspot.com/bug?extid=ed8bc247f231c1a48e21)
- [general protection fault in bio_alloc_bioset](https://syzkaller.appspot.com/bug?extid=09ddb593eea76a158f42)
- [memory leak in dvb_register_device](https://syzkaller.appspot.com/bug?extid=d37184d9d8cc34602616)
- [memory leak in vidtv_psi_short_event_desc_init](https://syzkaller.appspot.com/bug?extid=afc686a471d70896c5d9)

### FreeBSD bugs

- [Fatal trap NUM: general protection fault while in kernel mode in cam_periph_runccb](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293888)
- [Fatal trap NUM: page fault while in kernel mode in cam_periph_runccb](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293890)
- [Fatal trap NUM: page fault while in kernel mode in passdoioctl](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293891)
- [Fatal trap NUM: page fault while in kernel mode in passsendccb](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293892)
- [panic: _free(NUM): address ADDR(ADDR) has not been allocated](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293893)
- [panic: ata_action: ccb ADDR, func_code CODE should not be allocated from UMA zone](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293895)
- [panic: AUX register unsupported](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293898)
- [panic: cam_periph_ccbwait: proceeding with incomplete ccb](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293899)
- [panic: dst_m ADDR is not wired](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293900)
- [panic: mutex ACPI global lock owned at ../../../kern/kern_event.c:LINE](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=293901)

Test cases generated by SyzPilot have been incorporated into FreeBSD's test suite:

- [stress2: Added syzkaller reproducers. Update the exclude file](https://cgit.freebsd.org/src/commit/?id=4f8a1b4dffa8a6fa5fbe7fce05278792afd83a82)

### OpenBSD bugs

- [uvm_fault: dovutimens](https://marc.info/?l=openbsd-bugs&m=177398214322506&w=2)
- [uvm_fault: lptpushbytes](https://marc.info/?l=openbsd-bugs&m=177398271022704&w=2)

### NetBSD bugs

- [assert failed: chp->ch_drive[drive].drv_softc == NULL](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092421.html)
- [assert failed: hispgrp->pg_jobc > NUM](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092422.html)
- [assert failed: it->it_time.it_value.tv_sec >= NUM](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092423.html)
- [assert failed: kn->kn_fop == &proc_filtops](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092424.html)
- [assert failed: kq->kq_fdp == fdp](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092425.html)
- [assert failed: ks->ks_pshared_proc == NULL](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092426.html)
- [assert failed: ps->ps_endoffset != endoffset](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092427.html)
- [assert failed: sc->sc_base.me_evp != NULL](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092428.html)
- [assert failed: ts->tv_nsec >= NUM](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092429.html)
- [assert failed: uio->uio_iovcnt > NUM](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092430.html)
- [panic: ASan: Unauthorized Access In ADDR: Addr ADDR [ADDR bytes, read, KmemRedZone]](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092431.html)
- [panic: ASan: Unauthorized Access In ADDR: Addr ADDR [NUM byte, read, KmemRedZone]](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092432.html)
- [panic: LOCKDEBUG: Mutex error: rw_vector_enter,NUM: spin lock held](https://mail-index.netbsd.org/netbsd-bugs/2026/04/02/msg092433.html)
