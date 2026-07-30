# Example of writing spec for a request carrying a handle and a string

## Input

```c
[
  {"kind": "twoway", "ordinal": 3476297537454156075, "name": "ConnectToService", "has_request": true,
   "maybe_request_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.adb/ProviderConnectToServiceRequest",
     "type_shape_v2": {"inline_size": 24, "alignment": 8, "max_handles": 1, "max_out_of_line": 1024}},
   "has_response": true,
   "maybe_response_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.adb/Provider_ConnectToService_Result",
     "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 0}}}
]
```

Please write specification for syscall `zx_channel_call$fuchsia_hardware_adb_ProviderConnectToService`.

## Output

### Thought

**Task**: `fuchsia.hardware.adb/Provider.ConnectToService`, `twoway`, ordinal
`3476297537454156075` -> a `zx_channel_call` variant on `zx_chan_fuchsia_hardware_adb_Provider_client`.

**Request body**:

(Calling `get_struct_members_by_name("fuchsia.hardware.adb/ProviderConnectToServiceRequest")`)

```json
[{"name": "socket",
  "type": {"kind_v2": "handle", "subtype": "socket",
           "type_shape_v2": {"inline_size": 4, "alignment": 4, "max_handles": 1, "max_out_of_line": 0}},
  "field_shape_v2": {"offset": 0, "padding": 4}},
 {"name": "args",
  "type": {"kind_v2": "string",
           "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 1024}},
  "field_shape_v2": {"offset": 8, "padding": 0}}]
```

(Calling `get_struct_type_shape_by_name("fuchsia.hardware.adb/ProviderConnectToServiceRequest")`)

```json
{"inline_size": 24, "alignment": 8, "max_handles": 1, "max_out_of_line": 1024, "has_padding": true}
```

**Layout**:

- `socket` is a `handle<socket>`: inline it is only the 4-byte presence word
  `flags[fidl_handle_presence, int32]`, and the real handle travels in the handle table as
  `zx_socket`. It reports `padding: 4`, so a 4-byte filler follows to align the string at offset 8.
- `args` is a `string`: inline it is the 16-byte `fidl_string` header (`{size, presence}`), and its
  characters live out of line as `fidl_aligned[stringnoz]`. Because it has an out-of-line part, the
  field is split into `argsInLine` and `argsOutOfLine`, and the out-of-line entry goes after every
  inline field.

Check: 4 (presence) + 4 (padding) + 16 (`fidl_string`) = 24 = `inline_size`.

**Response handles**: the result union reports `max_handles: 0`, so `...ResponseHandles` is
`void void`.

**Dependencies**: none — `fidl_string`, `fidl_aligned`, `stringnoz`, `fidl_handle_presence` and
`zx_socket` all come from the existing `sys/fuchsia` files, and no named FIDL type needs its own
node. Only `ZX_CHANNEL_MAX_MSG_BYTES` needs an include.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "zircon/syscalls.h", "code": "include <zircon/syscalls.h>"},
		{"type": "struct", "name": "fuchsia_hardware_adb_ProviderConnectToServiceRequest", "code": "fuchsia_hardware_adb_ProviderConnectToServiceRequest {\n\thdr\t\tfidl_message_header[3476297537454156075]\n\tsocket\t\tflags[fidl_handle_presence, int32]\n\tpadding0\tarray[const[0, int8], 4]\n\targsInLine\tfidl_string\n\targsOutOfLine\tfidl_aligned[stringnoz]\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_adb_ProviderConnectToServiceRequestHandles", "code": "fuchsia_hardware_adb_ProviderConnectToServiceRequestHandles {\n\tsocket\tzx_socket\n} [packed]"},
		{"type": "struct", "name": "fuchsia_hardware_adb_ProviderConnectToServiceResponseHandles", "code": "fuchsia_hardware_adb_ProviderConnectToServiceResponseHandles {\n\tvoid\tvoid\n} [packed]"},
		{"type": "syscall", "name": "zx_channel_call$fuchsia_hardware_adb_ProviderConnectToService", "code": "zx_channel_call$fuchsia_hardware_adb_ProviderConnectToService(handle zx_chan_fuchsia_hardware_adb_Provider_client, options const[0], deadline zx_time, args ptr[in, fidl_call_args[fuchsia_hardware_adb_ProviderConnectToServiceRequest, fuchsia_hardware_adb_ProviderConnectToServiceRequestHandles, array[int8, ZX_CHANNEL_MAX_MSG_BYTES], fuchsia_hardware_adb_ProviderConnectToServiceResponseHandles]], actual_bytes ptr[out, int32], actual_handles ptr[out, int32])"}
	],
	"required": []
}
```
