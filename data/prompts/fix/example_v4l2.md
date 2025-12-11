# Example of fix spec for v4l2 (1)

## Input

```syzlang
meta arches["amd64"]

include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

ioctl$VIDIOC_G_SLICED_VBI_CAP(fd fd_video, cmd const[VIDIOC_G_SLICED_VBI_CAP], arg ptr[inout, v4l2_sliced_vbi_cap])

v4l2_buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE, V4L2_BUF_TYPE_VIDEO_OUTPUT, V4L2_BUF_TYPE_VIDEO_OVERLAY, V4L2_BUF_TYPE_VBI_CAPTURE, V4L2_BUF_TYPE_VBI_OUTPUT, V4L2_BUF_TYPE_SLICED_VBI_CAPTURE, V4L2_BUF_TYPE_SLICED_VBI_OUTPUT, V4L2_BUF_TYPE_VIDEO_OUTPUT_OVERLAY, V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE, V4L2_BUF_TYPE_VIDEO_OUTPUT_MPLANE, V4L2_BUF_TYPE_SDR_CAPTURE, V4L2_BUF_TYPE_SDR_OUTPUT, V4L2_BUF_TYPE_META_CAPTURE, V4L2_BUF_TYPE_META_OUTPUT, V4L2_BUF_TYPE_PRIVATE
v4l2_sliced_vbi_service = V4L2_SLICED_VBI_SERVICE_TELETEXT_B, V4L2_SLICED_VBI_SERVICE_VPS, V4L2_SLICED_VBI_SERVICE_CAPTION_525, V4L2_SLICED_VBI_SERVICE_WSS_625, V4L2_SLICED_VBI_SERVICE_SDI_90_RPD, V4L2_SLICED_VBI_SERVICE_100_ACD, V4L2_SLICED_VBI_SERVICE_100_AFD, V4L2_SLICED_VBI_SERVICE_625_WEW, V4L2_SLICED_VBI_SERVICE_750_WEW, V4L2_SLICED_VBI_SERVICE_1250_WEW

v4l2_sliced_vbi_cap {
	service_set	flags[v4l2_sliced_vbi_service, int16]
	service_lines	array[int16, 2][24]
	type		flags[v4l2_buf_type, int32]
	reserved	array[const[0, int32], 3]
}
```

```
sys/linux/spec-3325912462.txt:17:31: unexpected '[', expecting '\n'
sys/linux/spec-3325912462.txt:18:35: unexpected '\n', expecting int, identifier, string
sys/linux/spec-3325912462.txt:19:11: unexpected identifier, expecting '(', '{', '[', '='
sys/linux/spec-3325912462.txt:20:1: unexpected '}', expecting comment, define, include, resource, identifier
```

## Output

### Thought

(Calling tool to get source of `struct v4l2_sliced_vbi_cap`)

```c
struct v4l2_sliced_vbi_cap {
	__u16   service_set;
	/* service_lines[0][...] specifies lines 0-23 (1-23 used) of the first field
	   service_lines[1][...] specifies lines 0-23 (1-23 used) of the second field
				 (equals frame lines 313-336 for 625 line video
				  standards, 263-286 for 525 line standards) */
	__u16   service_lines[2][24];
	__u32	type;		/* enum v4l2_buf_type */
	__u32   reserved[3];    /* must be 0 */
};
```

### Final output

```syzlang
v4l2_sliced_vbi_cap {
	service_set	int16
	service_lines	array[array[int16, 24], 2]
	type		flags[v4l2_buf_type, int32]
	reserved	array[const[0, int32], 3]
}
```

# Example of fix spec for v4l2 (2)

## Input

```syzlang
include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

syz_open_dev$video(dev ptr[in, string["/dev/video#"]], id intptr, flags flags[open_flags]) fd_video
ioctl$VIDIOC_G_FBUF(fd fd_video, cmd const[VIDIOC_G_FBUF], arg ptr[out, v4l2_framebuffer])
ioctl$VIDIOC_S_FBUF(fd fd_video, cmd const[VIDIOC_S_FBUF], arg ptr[in, v4l2_framebuffer])

v4l2_fbuf_flags = V4L2_FBUF_FLAG_OVERLAY, V4L2_FBUF_FLAG_CHROMAKEY, V4L2_FBUF_FLAG_LOCAL_ALPHA, V4L2_FBUF_FLAG_LOCAL_Opaque, V4L2_FBUF_FLAG_SRC_CHROMAKEY, V4L2_FBUF_FLAG_DEFL_COLORKEY
v4l2_field = V4L2_FIELD_ANY, V4L2_FIELD_NONE, V4L2_FIELD_TOP, V4L2_FIELD_BOTTOM, V4L2_FIELD_INTERLACED, V4L2_FIELD_SEQ_TB, V4L2_FIELD_SEQ_BT, V4L2_FIELD_ALTERNATE, V4L2_FIELD_INTERLACED_TB, V4L2_FIELD_INTERLACED_BT
v4l2_colorspace = V4L2_COLORSPACE_DEFAULT, V4L2_COLORSPACE_SMPTE170M, V4L2_COLORSPACE_SMPTE240M, V4L2_COLORSPACE_REC709, V4L2_COLORSPACE_BT878, V4L2_COLORSPACE_470_SYSTEM_M, V4L2_COLORSPACE_470_SYSTEM_BG, V4L2_COLORSPACE_JPEG, V4L2_COLORSPACE_SRGB, V4L2_COLORSPACE_OPRGB, V4L2_COLORSPACE_BT2020, V4L2_COLORSPACE_RAW, V4L2_COLORSPACE_DCI_P3
v4l2_pix_fmt_flags = V4L2_PIX_FMT_FLAG_PREMUL_ALPHA, V4L2_PIX_FMT_FLAG_SET_CSC
v4l2_ycbcr_encoding = V4L2_YCBCR_ENC_DEFAULT, V4L2_YCBCR_ENC_601, V4L2_YCBCR_ENC_709, V4L2_YCBCR_ENC_XV601, V4L2_YCBCR_ENC_XV709, V4L2_YCBCR_ENC_SYCC, V4L2_YCBCR_ENC_BT2020, V4L2_YCBCR_ENC_BT2020_CONST_LUM, V4L2_YCBCR_ENC_SMPTE240M
v4l2_hsv_encoding = V4L2_HSV_ENC_180, V4L2_HSV_ENC_256
v4l2_quantization = V4L2_QUANTIZATION_DEFAULT, V4L2_QUANTIZATION_FULL_RANGE, V4L2_QUANTIZATION_LIM_RANGE
v4l2_xfer_func = V4L2_XFER_FUNC_DEFAULT, V4L2_XFER_FUNC_709, V4L2_XFER_FUNC_SRGB, V4L2_XFER_FUNC_OPRGB, V4L2_XFER_FUNC_SMPTE240M, V4L2_XFER_FUNC_NONE, V4L2_XFER_FUNC_DCI_P3, V4L2_XFER_FUNC_SMPTE2084

v4l2_framebuffer {
	capability	flags[v4l2_fbuf_flags, int32]
	flags		flags[v4l2_fbuf_flags, int32]
	fmt		v4l2_pix_format
}
v4l2_pix_format {
	width		int32
	height		int32
	pixelformat	int32
	field		flags[v4l2_field, int32]
	bytesperline	int32
	sizeimage	int32
	colorspace	flags[v4l2_colorspace, int32]
	priv		int32
	flags		flags[v4l2_pix_fmt_flags, int32]
	encoding_union	v4l2_pix_format_encoding_union
	quantization	flags[v4l2_quantization, int32]
	xfer_func	flags[v4l2_xfer_func, int32]
}

union v4l2_pix_format_encoding_union [
	ycbcr_enc	flags[v4l2_ycbcr_encoding, int32]
	hsv_enc		flags[v4l2_hsv_encoding, int32]
]
```

```
sys/linux/0.txt:39:7: unexpected identifier, expecting '(', '{', '[', '='
sys/linux/0.txt:40:12: unexpected identifier, expecting '(', '{', '[', '='
sys/linux/0.txt:41:11: unexpected identifier, expecting '(', '{', '[', '='
sys/linux/0.txt:42:1: unexpected ']', expecting comment, define, include, resource, identifier
```

## Output

### Thought

In syzlang, union is described as:

```
unionname "[" "\n"
	(fieldname type (if[expression])? "\n")+
"]" ("[" attribute* "]")?
```

Remove `union` before `v4l2_pix_format_encoding_union`.

### Final output

```syzlang
include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

syz_open_dev$video(dev ptr[in, string["/dev/video#"]], id intptr, flags flags[open_flags]) fd_video
ioctl$VIDIOC_G_FBUF(fd fd_video, cmd const[VIDIOC_G_FBUF], arg ptr[out, v4l2_framebuffer])
ioctl$VIDIOC_S_FBUF(fd fd_video, cmd const[VIDIOC_S_FBUF], arg ptr[in, v4l2_framebuffer])

v4l2_fbuf_flags = V4L2_FBUF_FLAG_OVERLAY, V4L2_FBUF_FLAG_CHROMAKEY, V4L2_FBUF_FLAG_LOCAL_ALPHA, V4L2_FBUF_FLAG_LOCAL_Opaque, V4L2_FBUF_FLAG_SRC_CHROMAKEY, V4L2_FBUF_FLAG_DEFL_COLORKEY
v4l2_field = V4L2_FIELD_ANY, V4L2_FIELD_NONE, V4L2_FIELD_TOP, V4L2_FIELD_BOTTOM, V4L2_FIELD_INTERLACED, V4L2_FIELD_SEQ_TB, V4L2_FIELD_SEQ_BT, V4L2_FIELD_ALTERNATE, V4L2_FIELD_INTERLACED_TB, V4L2_FIELD_INTERLACED_BT
v4l2_colorspace = V4L2_COLORSPACE_DEFAULT, V4L2_COLORSPACE_SMPTE170M, V4L2_COLORSPACE_SMPTE240M, V4L2_COLORSPACE_REC709, V4L2_COLORSPACE_BT878, V4L2_COLORSPACE_470_SYSTEM_M, V4L2_COLORSPACE_470_SYSTEM_BG, V4L2_COLORSPACE_JPEG, V4L2_COLORSPACE_SRGB, V4L2_COLORSPACE_OPRGB, V4L2_COLORSPACE_BT2020, V4L2_COLORSPACE_RAW, V4L2_COLORSPACE_DCI_P3
v4l2_pix_fmt_flags = V4L2_PIX_FMT_FLAG_PREMUL_ALPHA, V4L2_PIX_FMT_FLAG_SET_CSC
v4l2_ycbcr_encoding = V4L2_YCBCR_ENC_DEFAULT, V4L2_YCBCR_ENC_601, V4L2_YCBCR_ENC_709, V4L2_YCBCR_ENC_XV601, V4L2_YCBCR_ENC_XV709, V4L2_YCBCR_ENC_SYCC, V4L2_YCBCR_ENC_BT2020, V4L2_YCBCR_ENC_BT2020_CONST_LUM, V4L2_YCBCR_ENC_SMPTE240M
v4l2_hsv_encoding = V4L2_HSV_ENC_180, V4L2_HSV_ENC_256
v4l2_quantization = V4L2_QUANTIZATION_DEFAULT, V4L2_QUANTIZATION_FULL_RANGE, V4L2_QUANTIZATION_LIM_RANGE
v4l2_xfer_func = V4L2_XFER_FUNC_DEFAULT, V4L2_XFER_FUNC_709, V4L2_XFER_FUNC_SRGB, V4L2_XFER_FUNC_OPRGB, V4L2_XFER_FUNC_SMPTE240M, V4L2_XFER_FUNC_NONE, V4L2_XFER_FUNC_DCI_P3, V4L2_XFER_FUNC_SMPTE2084

v4l2_framebuffer {
	capability	flags[v4l2_fbuf_flags, int32]
	flags		flags[v4l2_fbuf_flags, int32]
	fmt		v4l2_pix_format
}
v4l2_pix_format {
	width		int32
	height		int32
	pixelformat	int32
	field		flags[v4l2_field, int32]
	bytesperline	int32
	sizeimage	int32
	colorspace	flags[v4l2_colorspace, int32]
	priv		int32
	flags		flags[v4l2_pix_fmt_flags, int32]
	encoding_union	v4l2_pix_format_encoding_union
	quantization	flags[v4l2_quantization, int32]
	xfer_func	flags[v4l2_xfer_func, int32]
}

v4l2_pix_format_encoding_union [
	ycbcr_enc	flags[v4l2_ycbcr_encoding, int32]
	hsv_enc		flags[v4l2_hsv_encoding, int32]
]
```