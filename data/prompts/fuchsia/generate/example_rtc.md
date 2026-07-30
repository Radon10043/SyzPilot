# Example of writing spec for the RTC driver protocol (1)

## Input

```c
[
  {"kind": "twoway", "ordinal": 2881649623829671924, "name": "Get", "has_request": true,
   "maybe_request_payload": null, "has_response": true,
   "maybe_response_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.rtc/Device_Get_Result",
     "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 8}}},
  {"kind": "twoway", "ordinal": 1614978035535395557, "name": "Set2", "has_request": true,
   "maybe_request_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.rtc/DeviceSet2Request",
     "type_shape_v2": {"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0}},
   "has_response": true,
   "maybe_response_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.rtc/Device_Set2_Result",
     "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 0}}}
]
```

Please write specification for init_syscall `zx_channel_create$fuchsia_hardware_rtc_Device`.

## Output

### Thought

**Task**: create the typed channel pair for `fuchsia.hardware.rtc/Device` (mangled
`fuchsia_hardware_rtc_Device`, derived from the payload identifiers).

**Resources**: the client end is what every method call consumes, the server end is what
`fdio_service_connect` gives away. Both derive from `zx_chan`, which `channel.txt` already declares.
Nothing else is needed — `zx_channel_create` takes no constants.

### Final output

```json
{
	"spec": [
		{"type": "resource", "name": "zx_chan_fuchsia_hardware_rtc_Device_client", "code": "resource zx_chan_fuchsia_hardware_rtc_Device_client[zx_chan]"},
		{"type": "resource", "name": "zx_chan_fuchsia_hardware_rtc_Device_server", "code": "resource zx_chan_fuchsia_hardware_rtc_Device_server[zx_chan]"},
		{"type": "init_syscall", "name": "zx_channel_create$fuchsia_hardware_rtc_Device", "code": "zx_channel_create$fuchsia_hardware_rtc_Device(options const[0], out0 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_client], out1 ptr[out, zx_chan_fuchsia_hardware_rtc_Device_server])"}
	],
	"required": []
}
```

# Example of writing spec for the RTC driver protocol (2)

## Input

(same methods array as above)

Please write specification for init_syscall `fdio_service_connect$fuchsia_hardware_rtc_Device`.

## Output

### Thought

**Task**: hand the server end to whoever serves the protocol. The path depends on whether the
protocol is discoverable.

(Calling `get_protocol_attrs_by_name("fuchsia.hardware.rtc/Device")`)

```json
[{"name": "discoverable", "arguments": []}]
```

The attribute is present and carries no `name` argument, so the default service path applies:
`/svc/` + the FIDL FQN with `/` replaced by `.` -> `/svc/fuchsia.hardware.rtc.Device`.

The server-end resource was already declared by `zx_channel_create$fuchsia_hardware_rtc_Device`, so
I reference it instead of redeclaring it.

### Final output

```json
{
	"spec": [
		{"type": "init_syscall", "name": "fdio_service_connect$fuchsia_hardware_rtc_Device", "code": "fdio_service_connect$fuchsia_hardware_rtc_Device(path ptr[in, string[\"/svc/fuchsia.hardware.rtc.Device\"]], handle zx_chan_fuchsia_hardware_rtc_Device_server)"}
	],
	"required": []
}
```

# Example of writing spec for the RTC driver protocol (3)

## Input

(same methods array as above)

Please write specification for syscall `zx_channel_call$fuchsia_hardware_rtc_DeviceSet2`.

## Output

### Thought

**Task**: the `Set2` method, `kind: twoway` with ordinal `1614978035535395557`, so this is a
`zx_channel_call` variant.

**Request body**: the request payload is `fuchsia.hardware.rtc/DeviceSet2Request`.

(Calling `get_struct_members_by_name("fuchsia.hardware.rtc/DeviceSet2Request")`)

```json
[{"name": "rtc",
  "type": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.rtc/Time",
           "type_shape_v2": {"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0}},
  "field_shape_v2": {"offset": 0, "padding": 0}}]
```

(Calling `get_struct_type_shape_by_name("fuchsia.hardware.rtc/DeviceSet2Request")`)

```json
{"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0, "has_padding": true}
```

One field, a nested struct at offset 0 with no trailing padding, and the payload's total inline size
is 8 — the single `Time` field accounts for all of it. A struct-typed field contributes to all three
nodes, so the body gets `rtcInLine` + `rtcOutOfLine` and the handle table gets `rtc`, even though
`Time` has neither out-of-line data nor handles (those nodes will be `void void`).

**Response handles**: the response payload is the result union `fuchsia.hardware.rtc/Device_Set2_Result`
whose type shape reports `max_handles: 0`, so `...ResponseHandles` is `void void`. The response
*body* stays opaque (`array[int8, ZX_CHANNEL_MAX_MSG_BYTES]`), which is why the result union itself
never needs a syzlang node.

**Dependencies**: the three `fuchsia_hardware_rtc_Time*` nodes are named FIDL declarations shared by
other methods, so they go to `required`. `ZX_CHANNEL_MAX_MSG_BYTES` comes from
`<zircon/syscalls.h>`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "zircon/syscalls.h", "code": "include <zircon/syscalls.h>"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_DeviceSet2Request", "code": "fuchsia_hardware_rtc_DeviceSet2Request {\n\thdr\t\tfidl_message_header[1614978035535395557]\n\trtcInLine\tfuchsia_hardware_rtc_TimeInLine\n\trtcOutOfLine\tfuchsia_hardware_rtc_TimeOutOfLine\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_DeviceSet2RequestHandles", "code": "fuchsia_hardware_rtc_DeviceSet2RequestHandles {\n\trtc\tfuchsia_hardware_rtc_TimeHandles\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_DeviceSet2ResponseHandles", "code": "fuchsia_hardware_rtc_DeviceSet2ResponseHandles {\n\tvoid\tvoid\n} [packed]"},
		{"type": "syscall", "name": "zx_channel_call$fuchsia_hardware_rtc_DeviceSet2", "code": "zx_channel_call$fuchsia_hardware_rtc_DeviceSet2(handle zx_chan_fuchsia_hardware_rtc_Device_client, options const[0], deadline zx_time, args ptr[in, fidl_call_args[fuchsia_hardware_rtc_DeviceSet2Request, fuchsia_hardware_rtc_DeviceSet2RequestHandles, array[int8, ZX_CHANNEL_MAX_MSG_BYTES], fuchsia_hardware_rtc_DeviceSet2ResponseHandles]], actual_bytes ptr[out, int32], actual_handles ptr[out, int32])"}
	],
	"required": [
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeInLine"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeOutOfLine"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeHandles"}
	]
}
```

# Example of writing spec for the RTC driver protocol (4)

## Input

(same methods array as above)

Please write specification for struct `fuchsia_hardware_rtc_TimeInLine`.

## Output

### Thought

**Task**: the inline node of a named FIDL struct. Strip the `InLine` suffix and split the mangled
name at the first `_` followed by an uppercase letter: library `fuchsia.hardware.rtc`, declaration
`Time`, i.e. `fuchsia.hardware.rtc/Time`.

(Calling `get_struct_members_by_name("fuchsia.hardware.rtc/Time")`)

```json
[{"name": "seconds", "type": {"kind_v2": "primitive", "subtype": "uint8"}, "field_shape_v2": {"offset": 0, "padding": 0}},
 {"name": "minutes", "type": {"kind_v2": "primitive", "subtype": "uint8"}, "field_shape_v2": {"offset": 1, "padding": 0}},
 {"name": "hours",   "type": {"kind_v2": "primitive", "subtype": "uint8"}, "field_shape_v2": {"offset": 2, "padding": 0}},
 {"name": "day",     "type": {"kind_v2": "primitive", "subtype": "uint8"}, "field_shape_v2": {"offset": 3, "padding": 0}},
 {"name": "month",   "type": {"kind_v2": "primitive", "subtype": "uint8"}, "field_shape_v2": {"offset": 4, "padding": 1}},
 {"name": "year",    "type": {"kind_v2": "primitive", "subtype": "uint16"}, "field_shape_v2": {"offset": 6, "padding": 0}}]
```

(Calling `get_struct_type_shape_by_name("fuchsia.hardware.rtc/Time")`)

```json
{"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0, "has_padding": true}
```

**Layout**: five `uint8` fields become `int8` (syzkaller ignores signedness). `month` reports
`padding: 1`, so a one-byte filler follows it, which is what lifts `year` to its offset 6. Check:
5 + 1 + 2 = 8 = `inline_size`.

**Empty nodes**: `max_out_of_line` and `max_handles` are both 0, so `...OutOfLine` and `...Handles`
are `void void`. They are still emitted here (rather than left to later tasks) because the request
struct already references them by name.

### Final output

```json
{
	"spec": [
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeInLine", "code": "fuchsia_hardware_rtc_TimeInLine {\n\tseconds\t\tint8\n\tminutes\t\tint8\n\thours\t\tint8\n\tday\t\tint8\n\tmonth\t\tint8\n\tpadding0\tarray[const[0, int8], 1]\n\tyear\t\tint16\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeOutOfLine", "code": "fuchsia_hardware_rtc_TimeOutOfLine {\n\tvoid\tvoid\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_rtc_TimeHandles", "code": "fuchsia_hardware_rtc_TimeHandles {\n\tvoid\tvoid\n} [packed]"}
	],
	"required": []
}
```
