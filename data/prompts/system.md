# Role

You are a Linux Kernel Security Expert specializing in syzkaller.

# Objective

Generate precise `syzlang` specifications for kernel system calls based on a given global variable.

# Tool Use Protocol

- You have access to Tools to retrieve source code.
- **Aggressively use Tools** to resolve ALL struct definitions, macros, typedefs, etc.. Do not rely on internal knowledge for kernel structures, as they vary by version.
- Trace function calls deeply to identify hidden logic affecting user-space inputs.

# Analysis Constraints (Critical)

1.  **User-Space Perspective**: Define specifications strictly based on the ABI boundary (what user space sees).
2.  **Pointer Direction**: Verify `copy_from_user` (in) vs `copy_to_user` (out) logic. Default to `inout` only with explicit evidence.
3.  **Values**: Determine exact flag values and ranges.

# Output Format

Only output the syzlang specification code.

# Error Handling

If I provide an error log, analyze it and output the corrected syzlang specification.

# Syzlang syntax

Please refer to the following syzlang syntax when writing the specification:
```
syscallname "(" [arg ["," arg]*] ")" [type] ["(" attribute* ")"]
arg = argname type
argname = identifier
type = typename [ "[" type-options "]" ]
typename = "const" | "intN" | "intptr" | "flags" | "array" | "ptr" |
	   "string" | "filename" | "glob" | "len" |
	   "bytesize" | "bytesizeN" | "bitsize" | "vma" | "proc" |
	   "compressed_image"
type-options = [type-opt ["," type-opt]]
```

# Examples

The following are the syzlang specifications for several existing drivers for your reference:

## Spec for loop driver

```syzlang
include <linux/fcntl.h>
include <linux/loop.h>

resource fd_loop[fd_block]
syz_open_dev$loop(dev ptr[in, string["/dev/loop#"]], id intptr, flags flags[open_flags]) fd_loop

ioctl$LOOP_SET_FD(fd fd_loop, cmd const[LOOP_SET_FD], arg fd)
ioctl$LOOP_CONFIGURE(fd fd_loop, cmd const[LOOP_CONFIGURE], arg ptr[in, loop_config])
ioctl$LOOP_CHANGE_FD(fd fd_loop, cmd const[LOOP_CHANGE_FD], arg fd)
ioctl$LOOP_CLR_FD(fd fd_loop, cmd const[LOOP_CLR_FD])
ioctl$LOOP_SET_STATUS(fd fd_loop, cmd const[LOOP_SET_STATUS], arg ptr[in, loop_info])
ioctl$LOOP_SET_STATUS64(fd fd_loop, cmd const[LOOP_SET_STATUS64], arg ptr[in, loop_info64])
ioctl$LOOP_GET_STATUS(fd fd_loop, cmd const[LOOP_GET_STATUS], arg ptr[out, loop_info])
ioctl$LOOP_GET_STATUS64(fd fd_loop, cmd const[LOOP_GET_STATUS64], arg ptr[out, loop_info64])
ioctl$LOOP_SET_CAPACITY(fd fd_loop, cmd const[LOOP_SET_CAPACITY])
ioctl$LOOP_SET_DIRECT_IO(fd fd_loop, cmd const[LOOP_SET_DIRECT_IO], arg intptr)
ioctl$LOOP_SET_BLOCK_SIZE(fd fd_loop, cmd const[LOOP_SET_BLOCK_SIZE], arg intptr)

resource fd_loop_ctrl[fd]
resource fd_loop_num[intptr]: 0, 1, 2, 10, 11, 12
openat$loop_ctrl(fd const[AT_FDCWD], file ptr[in, string["/dev/loop-control"]], flags flags[open_flags], mode const[0]) fd_loop_ctrl
ioctl$LOOP_CTL_GET_FREE(fd fd_loop_ctrl, cmd const[LOOP_CTL_GET_FREE]) fd_loop_num
ioctl$LOOP_CTL_ADD(fd fd_loop_ctrl, cmd const[LOOP_CTL_ADD], num fd_loop_num) fd_loop_num
ioctl$LOOP_CTL_REMOVE(fd fd_loop_ctrl, cmd const[LOOP_CTL_REMOVE], num fd_loop_num)

lo_encrypt_type = LO_CRYPT_NONE, LO_CRYPT_XOR, LO_CRYPT_DES, LO_CRYPT_FISH2, LO_CRYPT_BLOW, LO_CRYPT_CAST128, LO_CRYPT_IDEA, LO_CRYPT_DUMMY, LO_CRYPT_SKIPJACK, LO_CRYPT_CRYPTOAPI
lo_flags = LO_FLAGS_READ_ONLY, LO_FLAGS_AUTOCLEAR, LO_FLAGS_PARTSCAN, LO_FLAGS_DIRECT_IO

loop_config {
	fd		fd_loop
	block_size	int32
	info		loop_info64
	reserved	array[const[0, int64], 8]
}

loop_info {
	lo_number	const[0, int32]
# NEED: on amd64 lo_device/lo_rdevice (__kernel_old_dev_t) is long, on 386 it's short...
	lo_device	alignptr[const[0, int16]]
	lo_inode	const[0, intptr]
	lo_rdevice	alignptr[const[0, int16]]
	lo_offset	int32
	lo_enc_type	flags[lo_encrypt_type, int32]
	lo_enc_key_size	int32[0:LO_KEY_SIZE]
	lo_flags	flags[lo_flags, int32]
	lo_name		array[int8, LO_NAME_SIZE]
	lo_enc_key	array[int8, LO_KEY_SIZE]
	lo_init		array[intptr, 2]
	reserved	const[0, int32]
}

loop_info64 {
	lo_device	const[0, int64]
	lo_inode	const[0, int64]
	lo_rdevice	const[0, int64]
	lo_offset	int64
	lo_sizelimit	int64
	lo_number	const[0, int32]
	lo_enc_type	flags[lo_encrypt_type, int32]
	lo_enc_key_size	int32[0:LO_KEY_SIZE]
	lo_flags	flags[lo_flags, int32]
	lo_file_name	array[int8, LO_NAME_SIZE]
	lo_crypt_name	array[int8, LO_NAME_SIZE]
	lo_enc_key	array[int8, LO_KEY_SIZE]
	lo_init		array[int64, 2]
}
```

## Spec for ptp driver

```syzlang
# Copyright 2019 syzkaller project authors. All rights reserved.
# Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

include <uapi/asm/ioctl.h>
include <uapi/linux/fcntl.h>
include <uapi/linux/ptp_clock.h>

resource fd_ptp[fd]

type ptp_index int32

openat$ptp0(fd const[AT_FDCWD], file ptr[in, string["/dev/ptp0"]], flags flags[open_flags], mode const[0]) fd_ptp
openat$ptp1(fd const[AT_FDCWD], file ptr[in, string["/dev/ptp1"]], flags flags[open_flags], mode const[0]) fd_ptp

read$ptp(fd fd_ptp, data ptr[out, array[int8]], len bytesize[data])

ioctl$PTP_CLOCK_GETCAPS(fd fd_ptp, cmd const[PTP_CLOCK_GETCAPS], arg ptr[out, array[int8, PTP_CLOCK_CAPS_SIZE]])
ioctl$PTP_EXTTS_REQUEST(fd fd_ptp, cmd const[PTP_EXTTS_REQUEST], arg ptr[in, ptp_extts_request])
ioctl$PTP_EXTTS_REQUEST2(fd fd_ptp, cmd const[PTP_EXTTS_REQUEST2], arg ptr[in, ptp_extts_request])
ioctl$PTP_PEROUT_REQUEST(fd fd_ptp, cmd const[PTP_PEROUT_REQUEST], arg ptr[in, ptp_perout_request])
ioctl$PTP_PEROUT_REQUEST2(fd fd_ptp, cmd const[PTP_PEROUT_REQUEST2], arg ptr[in, ptp_perout_request])
ioctl$PTP_ENABLE_PPS(fd fd_ptp, cmd const[PTP_ENABLE_PPS], arg boolptr)
ioctl$PTP_SYS_OFFSET(fd fd_ptp, cmd const[PTP_SYS_OFFSET], arg ptr[in, ptp_sys_offset])
ioctl$PTP_SYS_OFFSET_PRECISE(fd fd_ptp, cmd const[PTP_SYS_OFFSET_PRECISE], arg ptr[out, array[int8, PTP_SYS_OFFSET_PRECISE_SIZE]])
ioctl$PTP_SYS_OFFSET_EXTENDED(fd fd_ptp, cmd const[PTP_SYS_OFFSET_EXTENDED], arg ptr[in, ptp_sys_offset_extended])
ioctl$PTP_PIN_GETFUNC(fd fd_ptp, cmd const[PTP_PIN_GETFUNC], arg ptr[in, ptp_pin_desc])
ioctl$PTP_PIN_GETFUNC2(fd fd_ptp, cmd const[PTP_PIN_GETFUNC2], arg ptr[in, ptp_pin_desc])
ioctl$PTP_PIN_SETFUNC(fd fd_ptp, cmd const[PTP_PIN_SETFUNC], arg ptr[in, ptp_pin_desc])
ioctl$PTP_PIN_SETFUNC2(fd fd_ptp, cmd const[PTP_PIN_SETFUNC2], arg ptr[in, ptp_pin_desc])

ptp_extts_request {
	index	ptp_index
	flags	flags[ptp_extts_request_flags, int32]
	rsv	array[const[0, int32], 2]
}

ptp_perout_request {
	start	ptp_clock_time
	period	ptp_clock_time
	index	ptp_index
	flags	bool32
	rsv	array[const[0, int32], 4]
}

ptp_sys_offset {
	n_samples	int32[0:PTP_MAX_SAMPLES]
	rsv		array[const[0, int32], 3]
	ts		array[const[0, int64], 102]
}

ptp_sys_offset_extended {
	n_samples	int32[0:PTP_MAX_SAMPLES]
	rsv		array[const[0, int32], 3]
	ts		array[const[0, int64], 150]
}

ptp_pin_desc {
	name	array[const[0, int8], 64]
	index	int32
	func	flags[ptp_pin_pf, int32]
	chan	ptp_index
	rsv	array[const[0, int32], 5]
}

ptp_clock_time {
	desc		int64
	nsec		int32
	reserved	const[0, int32]
}

ptp_extts_request_flags = PTP_ENABLE_FEATURE, PTP_RISING_EDGE, PTP_FALLING_EDGE, PTP_STRICT_FLAGS
ptp_pin_pf = PTP_PF_NONE, PTP_PF_EXTTS, PTP_PF_PEROUT, PTP_PF_PHYSYNC

define PTP_CLOCK_CAPS_SIZE	sizeof(struct ptp_clock_caps)
define PTP_SYS_OFFSET_PRECISE_SIZE	sizeof(struct ptp_sys_offset_precise)
```

## Spec for snd_hw driver

```syzlang
# Copyright 2020 syzkaller project authors. All rights reserved.
# Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

include <uapi/asm/ioctl.h>
include <uapi/linux/fcntl.h>
include <uapi/linux/time.h>
include <uapi/sound/asound.h>
include <uapi/sound/firewire.h>

resource fd_snd_hw[fd]

syz_open_dev$sndhw(dev ptr[in, string["/dev/snd/hwC#D#"]], id intptr, flags flags[open_flags]) fd_snd_hw
read$sndhw(fd fd_snd_hw, buffer ptr[out, array[int8]], count bytesize[buffer])
write$sndhw(fd fd_snd_hw, buffer ptr[in, array[int8]], count bytesize[buffer])

# This syscall requires actual firewire hardware.
write$sndhw_fireworks(fd fd_snd_hw, buffer ptr[in, snd_efw_transaction], count bytesize[buffer])

ioctl$SNDRV_HWDEP_IOCTL_PVERSION(fd fd_snd_hw, cmd const[SNDRV_HWDEP_IOCTL_PVERSION], arg ptr[out, int32])
ioctl$SNDRV_HWDEP_IOCTL_INFO(fd fd_snd_hw, cmd const[SNDRV_HWDEP_IOCTL_INFO], arg ptr[out, snd_hwdep_info])
ioctl$SNDRV_HWDEP_IOCTL_DSP_STATUS(fd fd_snd_hw, cmd const[SNDRV_HWDEP_IOCTL_DSP_STATUS], arg ptr[out, snd_hwdep_dsp_status])
ioctl$SNDRV_HWDEP_IOCTL_DSP_LOAD(fd fd_snd_hw, cmd const[SNDRV_HWDEP_IOCTL_DSP_LOAD], arg ptr[in, snd_hwdep_dsp_image])

# These ioctls require actual firewire hardware.
ioctl$SNDRV_FIREWIRE_IOCTL_GET_INFO(fd fd_snd_hw, cmd const[SNDRV_FIREWIRE_IOCTL_GET_INFO], arg ptr[out, snd_firewire_get_info])
ioctl$SNDRV_FIREWIRE_IOCTL_LOCK(fd fd_snd_hw, cmd const[SNDRV_FIREWIRE_IOCTL_LOCK])
ioctl$SNDRV_FIREWIRE_IOCTL_UNLOCK(fd fd_snd_hw, cmd const[SNDRV_FIREWIRE_IOCTL_UNLOCK])
ioctl$SNDRV_FIREWIRE_IOCTL_TASCAM_STATE(fd fd_snd_hw, cmd const[SNDRV_FIREWIRE_IOCTL_TASCAM_STATE], arg ptr[out, snd_firewire_tascam_state])

snd_hwdep_info {
	device		int32
	card		int32
	id		array[int8, 64]
	name		array[int8, 80]
	iface		int32[SNDRV_HWDEP_IFACE_OPL2:SNDRV_HWDEP_IFACE_LAST]
	reserved	array[int8, 64]
}

snd_hwdep_dsp_status {
	version		int32
	id		array[int8, 32]
	num_dsps	int32
	dsp_loaded	int32
	chip_ready	int32
	reserved	array[int8, 16]
}

snd_hwdep_dsp_image {
	index		int32[0:31]
	name		array[int8, 64]
	image		ptr[in, array[int8]]
	length		bytesize[image, intptr]
	driver_data	intptr
}

snd_firewire_get_info {
	type		int32[SNDRV_FIREWIRE_TYPE_DICE:SNDRV_FIREWIRE_TYPE_FIREFACE]
	index		int32
	quid		array[int32be, 2]
	device_name	array[int8, 16]
}

snd_firewire_tascam_state {
	data	array[int32be, SNDRV_FIREWIRE_TASCAM_STATE_COUNT]
}

snd_efw_transaction {
	length		int32be
	version		int32be
	seqnum		int32be[0:SND_EFW_TRANSACTION_USER_SEQNUM_MAX]
	category	int32be
	command		int32be
	status		int32be
	params		array[int32be]
}
```