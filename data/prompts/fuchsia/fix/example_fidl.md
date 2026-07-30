# Example of fixing a fuchsia spec (1): missing companion nodes

## Input

```syzlang
include <zircon/syscalls.h>

resource zx_chan_fuchsia_hardware_rtc_Device_client[zx_chan]
resource zx_chan_fuchsia_hardware_rtc_Device_server[zx_chan]

zx_channel_create$fuchsia_hardware_rtc_Device(options const[0], out0 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_client], out1 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_server])
fdio_service_connect$fuchsia_hardware_rtc_Device(path ptr[in, string["/svc/fuchsia.hardware.rtc.Device"]], handle zx_chan_fuchsia_hardware_rtc_Device_server)

zx_channel_call$fuchsia_hardware_rtc_DeviceSet2(handle zx_chan_fuchsia_hardware_rtc_Device_client, options const[0], deadline zx_time, args ptr[in, fidl_call_args[fuchsia_hardware_rtc_DeviceSet2Request, fuchsia_hardware_rtc_DeviceSet2RequestHandles, array[int8, ZX_CHANNEL_MAX_MSG_BYTES], fuchsia_hardware_rtc_DeviceSet2ResponseHandles]], actual_bytes ptr[out, int32], actual_handles ptr[out, int32])

fuchsia_hardware_rtc_DeviceSet2Request {
	hdr		fidl_message_header[1614978035535395557]
	rtcInLine	fuchsia_hardware_rtc_TimeInLine
	rtcOutOfLine	fuchsia_hardware_rtc_TimeOutOfLine
} [packed]

fuchsia_hardware_rtc_DeviceSet2RequestHandles {
	void	void
} [packed]

fuchsia_hardware_rtc_DeviceSet2ResponseHandles {
	void	void
} [packed]

fuchsia_hardware_rtc_TimeInLine {
	seconds	int8
	minutes	int8
	hours	int8
	day	int8
	month	int8
	year	int16
} [packed]

fuchsia_hardware_rtc_TimeHandles {
	void	void
} [packed]
```

```
sys/fuchsia/spec-1183048421.txt:14:15: unknown type fuchsia_hardware_rtc_TimeOutOfLine
```

## Output

### Thought

The `Time` triple is incomplete: the request body references `fuchsia_hardware_rtc_TimeOutOfLine`,
which was never declared. And `fuchsia_hardware_rtc_TimeHandles` *is* declared but nothing points at
it — the `...RequestHandles` node says `void void` instead of carrying the nested struct's handle
entry. That second defect is hidden behind the first (the compiler stops at the unknown type), and
declaring the missing node without wiring the handles entry in would just turn it into
`unused struct fuchsia_hardware_rtc_TimeHandles` on the next round, so I fix both now.

Confirming the declaration behind the nodes:

(Calling `get_struct_type_shape_by_name("fuchsia.hardware.rtc/Time")`)

```json
{"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0, "has_padding": true}
```

`max_out_of_line` and `max_handles` are 0, so both companion nodes are empty (`void void`) — but
they must still exist and still be referenced, because a struct-typed field contributes to all three
nodes. So: add `fuchsia_hardware_rtc_TimeOutOfLine`, and replace the `void void` in
`...DeviceSet2RequestHandles` with `rtc fuchsia_hardware_rtc_TimeHandles`.

Since I am touching this triple, I check the inline layout as well:

(Calling `get_struct_members_by_name("fuchsia.hardware.rtc/Time")`)

```json
[{"name": "seconds", "type": {"subtype": "uint8"}, "field_shape_v2": {"offset": 0, "padding": 0}},
 {"name": "minutes", "type": {"subtype": "uint8"}, "field_shape_v2": {"offset": 1, "padding": 0}},
 {"name": "hours",   "type": {"subtype": "uint8"}, "field_shape_v2": {"offset": 2, "padding": 0}},
 {"name": "day",     "type": {"subtype": "uint8"}, "field_shape_v2": {"offset": 3, "padding": 0}},
 {"name": "month",   "type": {"subtype": "uint8"}, "field_shape_v2": {"offset": 4, "padding": 1}},
 {"name": "year",    "type": {"subtype": "uint16"}, "field_shape_v2": {"offset": 6, "padding": 0}}]
```

`month` has `padding: 1` and `year` sits at offset 6, but the spec has no filler: the node is 7
bytes where the wire format is 8, so every `Set2` message the fuzzer builds would be malformed. Add
`padding0 array[const[0, int8], 1]`. Nothing else changes.

### Final output

```syzlang
include <zircon/syscalls.h>

resource zx_chan_fuchsia_hardware_rtc_Device_client[zx_chan]
resource zx_chan_fuchsia_hardware_rtc_Device_server[zx_chan]

zx_channel_create$fuchsia_hardware_rtc_Device(options const[0], out0 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_client], out1 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_server])
fdio_service_connect$fuchsia_hardware_rtc_Device(path ptr[in, string["/svc/fuchsia.hardware.rtc.Device"]], handle zx_chan_fuchsia_hardware_rtc_Device_server)

zx_channel_call$fuchsia_hardware_rtc_DeviceSet2(handle zx_chan_fuchsia_hardware_rtc_Device_client, options const[0], deadline zx_time, args ptr[in, fidl_call_args[fuchsia_hardware_rtc_DeviceSet2Request, fuchsia_hardware_rtc_DeviceSet2RequestHandles, array[int8, ZX_CHANNEL_MAX_MSG_BYTES], fuchsia_hardware_rtc_DeviceSet2ResponseHandles]], actual_bytes ptr[out, int32], actual_handles ptr[out, int32])

fuchsia_hardware_rtc_DeviceSet2Request {
	hdr		fidl_message_header[1614978035535395557]
	rtcInLine	fuchsia_hardware_rtc_TimeInLine
	rtcOutOfLine	fuchsia_hardware_rtc_TimeOutOfLine
} [packed]

fuchsia_hardware_rtc_DeviceSet2RequestHandles {
	rtc	fuchsia_hardware_rtc_TimeHandles
} [packed]

fuchsia_hardware_rtc_DeviceSet2ResponseHandles {
	void	void
} [packed]

fuchsia_hardware_rtc_TimeInLine {
	seconds		int8
	minutes		int8
	hours		int8
	day		int8
	month		int8
	padding0	array[const[0, int8], 1]
	year		int16
} [packed]

fuchsia_hardware_rtc_TimeOutOfLine {
	void	void
} [packed]

fuchsia_hardware_rtc_TimeHandles {
	void	void
} [packed]
```

# Example of fixing a fuchsia spec (2): bad flags and a union written as a struct

## Input

```syzlang
resource zx_chan_fuchsia_media_UsageReporter_client[zx_chan]
resource zx_chan_fuchsia_media_UsageReporter_server[zx_chan]
resource zx_chan_fuchsia_media_UsageWatcher_client[zx_chan]
resource zx_chan_fuchsia_media_UsageWatcher_server[zx_chan]

zx_channel_create$fuchsia_media_UsageReporter(options const[0], out0 ptr[out, zx_chan_fuchsia_media_UsageReporter_client], out1 ptr[out, zx_chan_fuchsia_media_UsageReporter_server])
zx_channel_create$fuchsia_media_UsageWatcher(options const[0], out0 ptr[out, zx_chan_fuchsia_media_UsageWatcher_client], out1 ptr[out, zx_chan_fuchsia_media_UsageWatcher_server])
fdio_service_connect$fuchsia_media_UsageWatcher(path ptr[in, string["/svc/"]], handle zx_chan_fuchsia_media_UsageWatcher_server)
fdio_service_connect$fuchsia_media_UsageReporter(path ptr[in, string["/svc/fuchsia.media.UsageReporter"]], handle zx_chan_fuchsia_media_UsageReporter_server)
zx_channel_write$fuchsia_media_UsageReporterWatch(handle zx_chan_fuchsia_media_UsageReporter_client, options const[0], bytes ptr[in, fuchsia_media_UsageReporterWatchRequest], num_bytes bytesize[bytes], handles ptr[in, fuchsia_media_UsageReporterWatchRequestHandles], num_handles bytesize4[handles])

fuchsia_media_AudioRenderUsage = fuchsia_media_AudioRenderUsage_BACKGROUND, fuchsia_media_AudioRenderUsage_MEDIA, fuchsia_media_AudioRenderUsage_INTERRUPTION, fuchsia_media_AudioRenderUsage_SYSTEM_AGENT, fuchsia_media_AudioRenderUsage_COMMUNICATION
fuchsia_media_AudioCaptureUsage = fuchsia_media_AudioCaptureUsage_BACKGROUND, fuchsia_media_AudioCaptureUsage_MEDIA, fuchsia_media_AudioCaptureUsage_SYSTEM_AGENT, fuchsia_media_AudioCaptureUsage_COMMUNICATION

fuchsia_media_UsageReporterWatchRequest {
	hdr		fidl_message_header[8547391950682442073]
	usageInLine	fuchsia_media_UsageInLine
	usage_watcher	flags[fidl_handle_presence, int32]
	padding0	array[const[0, int8], 4]
	usageOutOfLine	fuchsia_media_UsageOutOfLine
} [packed]

fuchsia_media_UsageReporterWatchRequestHandles {
	usage		fuchsia_media_UsageHandles
	usage_watcher	zx_chan_fuchsia_media_UsageWatcher_client
} [packed]

fuchsia_media_UsageInLine {
	render_usage	fidl_union_member[1, flags[fuchsia_media_AudioRenderUsages, int32]]
	capture_usage	fidl_union_member[2, flags[fuchsia_media_AudioCaptureUsage, int32]]
} [packed]

fuchsia_media_UsageOutOfLine {
	void	void
} [packed]

fuchsia_media_UsageHandles {
	void	void
} [packed]
```

```
sys/fuchsia/spec-2032118884.txt:29:42: unknown flags fuchsia_media_AudioRenderUsages
```

## Output

### Thought

**The reported error** is a typo: line 29 references `fuchsia_media_AudioRenderUsages` (trailing
`s`) while the declaration on line 12 is `fuchsia_media_AudioRenderUsage`. Fixing the reference also
prevents the `unused flags fuchsia_media_AudioRenderUsage` that would surface next round, since the
declaration currently has no user at all.

**While in the node**, I verify it against the IR instead of waiting for the next round.

(Calling `get_union_members_by_name("fuchsia.media/Usage")`) confirms two options, `render_usage`
with ordinal `1` and `capture_usage` with ordinal `2`, so the tags are right.

(Calling `get_enum_members_by_name("fuchsia.media/AudioCaptureUsage")`)

```json
[{"name": "BACKGROUND", "value": {"value": "0"}}, {"name": "FOREGROUND", "value": {"value": "1"}},
 {"name": "SYSTEM_AGENT", "value": {"value": "2"}}, {"name": "COMMUNICATION", "value": {"value": "3"}}]
```

The capture enum has no `MEDIA` member — that one belongs to `AudioRenderUsage`; the second slot is
`FOREGROUND`. `fuchsia_media_AudioCaptureUsage_MEDIA` would fail in `syz-extract` as
`... is unsupported on all arches (typo?)`, so I replace it now and keep the rest of the list.

`fuchsia_media_UsageInLine` also describes a FIDL union but is written with `{ }`, which makes it a
syzlang *struct* — every message would then carry both options one after another, while a FIDL union
carries exactly one tagged option. Rewrite it with `[ ]` and drop `[packed]` (a union takes
`[varlen]`, and only when its size varies; here both options are 4-byte enums, so no attribute is
needed). Nothing else changes: the `...OutOfLine` and `...Handles` nodes are correctly empty and
correctly referenced, and both `UsageWatcher` channel ends have a consumer.

### Final output

```syzlang
resource zx_chan_fuchsia_media_UsageReporter_client[zx_chan]
resource zx_chan_fuchsia_media_UsageReporter_server[zx_chan]
resource zx_chan_fuchsia_media_UsageWatcher_client[zx_chan]
resource zx_chan_fuchsia_media_UsageWatcher_server[zx_chan]

zx_channel_create$fuchsia_media_UsageReporter(options const[0], out0 ptr[out, zx_chan_fuchsia_media_UsageReporter_client], out1 ptr[out, zx_chan_fuchsia_media_UsageReporter_server])
zx_channel_create$fuchsia_media_UsageWatcher(options const[0], out0 ptr[out, zx_chan_fuchsia_media_UsageWatcher_client], out1 ptr[out, zx_chan_fuchsia_media_UsageWatcher_server])
fdio_service_connect$fuchsia_media_UsageWatcher(path ptr[in, string["/svc/"]], handle zx_chan_fuchsia_media_UsageWatcher_server)
fdio_service_connect$fuchsia_media_UsageReporter(path ptr[in, string["/svc/fuchsia.media.UsageReporter"]], handle zx_chan_fuchsia_media_UsageReporter_server)
zx_channel_write$fuchsia_media_UsageReporterWatch(handle zx_chan_fuchsia_media_UsageReporter_client, options const[0], bytes ptr[in, fuchsia_media_UsageReporterWatchRequest], num_bytes bytesize[bytes], handles ptr[in, fuchsia_media_UsageReporterWatchRequestHandles], num_handles bytesize4[handles])

fuchsia_media_AudioRenderUsage = fuchsia_media_AudioRenderUsage_BACKGROUND, fuchsia_media_AudioRenderUsage_MEDIA, fuchsia_media_AudioRenderUsage_INTERRUPTION, fuchsia_media_AudioRenderUsage_SYSTEM_AGENT, fuchsia_media_AudioRenderUsage_COMMUNICATION
fuchsia_media_AudioCaptureUsage = fuchsia_media_AudioCaptureUsage_BACKGROUND, fuchsia_media_AudioCaptureUsage_FOREGROUND, fuchsia_media_AudioCaptureUsage_SYSTEM_AGENT, fuchsia_media_AudioCaptureUsage_COMMUNICATION

fuchsia_media_UsageReporterWatchRequest {
	hdr		fidl_message_header[8547391950682442073]
	usageInLine	fuchsia_media_UsageInLine
	usage_watcher	flags[fidl_handle_presence, int32]
	padding0	array[const[0, int8], 4]
	usageOutOfLine	fuchsia_media_UsageOutOfLine
} [packed]

fuchsia_media_UsageReporterWatchRequestHandles {
	usage		fuchsia_media_UsageHandles
	usage_watcher	zx_chan_fuchsia_media_UsageWatcher_client
} [packed]

fuchsia_media_UsageInLine [
	render_usage	fidl_union_member[1, flags[fuchsia_media_AudioRenderUsage, int32]]
	capture_usage	fidl_union_member[2, flags[fuchsia_media_AudioCaptureUsage, int32]]
]

fuchsia_media_UsageOutOfLine {
	void	void
} [packed]

fuchsia_media_UsageHandles {
	void	void
} [packed]
```
