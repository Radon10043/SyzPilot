# Examples of fixing spec for v4l2

`(Tool call: <tool> <target>)` marks a tool call and the code block after it is the tool result. `/* ... */` marks lines omitted in this example only; real tool results are complete.

## Example 1

### Input

```syzlang
meta arches["amd64"]

include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

ioctl$VIDIOC_G_SLICED_VBI_CAP(fd fd_video, cmd const[VIDIOC_G_SLICED_VBI_CAP], arg ptr[inout, v4l2_sliced_vbi_cap])

v4l2_buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE, V4L2_BUF_TYPE_VIDEO_OUTPUT, V4L2_BUF_TYPE_VBI_CAPTURE, V4L2_BUF_TYPE_SLICED_VBI_CAPTURE, V4L2_BUF_TYPE_SLICED_VBI_OUTPUT
v4l2_sliced_vbi_service = V4L2_SLICED_VBI_SERVICE_TELETEXT_B, V4L2_SLICED_VBI_SERVICE_VPS, V4L2_SLICED_VBI_SERVICE_CAPTION_525, V4L2_SLICED_VBI_SERVICE_WSS_625

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

### Thought

(Tool call: get_struct_code_by_name `v4l2_sliced_vbi_cap`)

```c
struct v4l2_sliced_vbi_cap {
	__u16   service_set;
	/* ... */
	__u16   service_lines[2][24];
	__u32	type;		/* enum v4l2_buf_type */
	__u32   reserved[3];    /* must be 0 */
};
```

Syzlang has no `[a][b]` suffix: the 2-D array `service_lines[2][24]` becomes `array[array[int16, 24], 2]`.

### Final output

```syzlang
meta arches["amd64"]

include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

ioctl$VIDIOC_G_SLICED_VBI_CAP(fd fd_video, cmd const[VIDIOC_G_SLICED_VBI_CAP], arg ptr[inout, v4l2_sliced_vbi_cap])

v4l2_buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE, V4L2_BUF_TYPE_VIDEO_OUTPUT, V4L2_BUF_TYPE_VBI_CAPTURE, V4L2_BUF_TYPE_SLICED_VBI_CAPTURE, V4L2_BUF_TYPE_SLICED_VBI_OUTPUT
v4l2_sliced_vbi_service = V4L2_SLICED_VBI_SERVICE_TELETEXT_B, V4L2_SLICED_VBI_SERVICE_VPS, V4L2_SLICED_VBI_SERVICE_CAPTION_525, V4L2_SLICED_VBI_SERVICE_WSS_625

v4l2_sliced_vbi_cap {
	service_set	flags[v4l2_sliced_vbi_service, int16]
	service_lines	array[array[int16, 24], 2]
	type		flags[v4l2_buf_type, int32]
	reserved	array[const[0, int32], 3]
}
```

## Example 2

### Input

```syzlang
include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

syz_open_dev$video(dev ptr[in, string["/dev/video#"]], id intptr, flags flags[open_flags]) fd_video
ioctl$VIDIOC_G_FBUF(fd fd_video, cmd const[VIDIOC_G_FBUF], arg ptr[out, v4l2_framebuffer])
ioctl$VIDIOC_S_FBUF(fd fd_video, cmd const[VIDIOC_S_FBUF], arg ptr[in, v4l2_framebuffer])

v4l2_fbuf_flags = V4L2_FBUF_FLAG_OVERLAY, V4L2_FBUF_FLAG_CHROMAKEY, V4L2_FBUF_FLAG_LOCAL_ALPHA
v4l2_field = V4L2_FIELD_ANY, V4L2_FIELD_NONE, V4L2_FIELD_INTERLACED
v4l2_colorspace = V4L2_COLORSPACE_DEFAULT, V4L2_COLORSPACE_REC709, V4L2_COLORSPACE_SRGB
v4l2_pix_fmt_flags = V4L2_PIX_FMT_FLAG_PREMUL_ALPHA, V4L2_PIX_FMT_FLAG_SET_CSC
v4l2_ycbcr_encoding = V4L2_YCBCR_ENC_DEFAULT, V4L2_YCBCR_ENC_601, V4L2_YCBCR_ENC_709
v4l2_hsv_encoding = V4L2_HSV_ENC_180, V4L2_HSV_ENC_256
v4l2_quantization = V4L2_QUANTIZATION_DEFAULT, V4L2_QUANTIZATION_FULL_RANGE, V4L2_QUANTIZATION_LIM_RANGE
v4l2_xfer_func = V4L2_XFER_FUNC_DEFAULT, V4L2_XFER_FUNC_709, V4L2_XFER_FUNC_SRGB

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

### Thought

Syzlang unions are declared as `unionname [ ... ]` with no `union` keyword (see Unions in the syntax reference) → remove `union` before `v4l2_pix_format_encoding_union`.

### Final output

```syzlang
include <uapi/linux/videodev2.h>
include <uapi/linux/time.h>

resource fd_video[fd]

syz_open_dev$video(dev ptr[in, string["/dev/video#"]], id intptr, flags flags[open_flags]) fd_video
ioctl$VIDIOC_G_FBUF(fd fd_video, cmd const[VIDIOC_G_FBUF], arg ptr[out, v4l2_framebuffer])
ioctl$VIDIOC_S_FBUF(fd fd_video, cmd const[VIDIOC_S_FBUF], arg ptr[in, v4l2_framebuffer])

v4l2_fbuf_flags = V4L2_FBUF_FLAG_OVERLAY, V4L2_FBUF_FLAG_CHROMAKEY, V4L2_FBUF_FLAG_LOCAL_ALPHA
v4l2_field = V4L2_FIELD_ANY, V4L2_FIELD_NONE, V4L2_FIELD_INTERLACED
v4l2_colorspace = V4L2_COLORSPACE_DEFAULT, V4L2_COLORSPACE_REC709, V4L2_COLORSPACE_SRGB
v4l2_pix_fmt_flags = V4L2_PIX_FMT_FLAG_PREMUL_ALPHA, V4L2_PIX_FMT_FLAG_SET_CSC
v4l2_ycbcr_encoding = V4L2_YCBCR_ENC_DEFAULT, V4L2_YCBCR_ENC_601, V4L2_YCBCR_ENC_709
v4l2_hsv_encoding = V4L2_HSV_ENC_180, V4L2_HSV_ENC_256
v4l2_quantization = V4L2_QUANTIZATION_DEFAULT, V4L2_QUANTIZATION_FULL_RANGE, V4L2_QUANTIZATION_LIM_RANGE
v4l2_xfer_func = V4L2_XFER_FUNC_DEFAULT, V4L2_XFER_FUNC_709, V4L2_XFER_FUNC_SRGB

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
