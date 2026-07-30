# Example of writing spec for a oneway method with a union and a protocol endpoint (1)

## Input

```c
[
  {"kind": "oneway", "ordinal": 8547391950682442073, "name": "Watch", "has_request": true,
   "maybe_request_payload": {"kind_v2": "identifier", "identifier": "fuchsia.media/UsageReporterWatchRequest",
     "type_shape_v2": {"inline_size": 24, "alignment": 8, "max_handles": 1, "max_out_of_line": 0}},
   "has_response": false, "maybe_response_payload": null}
]
```

Please write specification for syscall `zx_channel_write$fuchsia_media_UsageReporterWatch`.

## Output

### Thought

**Task**: `fuchsia.media/UsageReporter.Watch` is `oneway`, so the fuzzer only writes a request and
there is no response to wait for — this is a `zx_channel_write` variant, not `zx_channel_call`.

**Request body**:

(Calling `get_struct_members_by_name("fuchsia.media/UsageReporterWatchRequest")`)

```json
[{"name": "usage",
  "type": {"kind_v2": "identifier", "identifier": "fuchsia.media/Usage",
           "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 0}},
  "field_shape_v2": {"offset": 0, "padding": 0}},
 {"name": "usage_watcher",
  "type": {"kind_v2": "endpoint", "role": "client", "protocol": "fuchsia.media/UsageWatcher",
           "type_shape_v2": {"inline_size": 4, "alignment": 4, "max_handles": 1, "max_out_of_line": 0}},
  "field_shape_v2": {"offset": 16, "padding": 4}}]
```

(Calling `get_decl_by_name("fuchsia.media/Usage")`) -> `union`

**Layout**:

- `usage` is a union, so it contributes to all three nodes: `usageInLine` and `usageOutOfLine` in
  the body, `usage` in the handle table.
- `usage_watcher` is an endpoint with `role: client` for protocol `fuchsia.media/UsageWatcher`.
  Inline it is the 4-byte presence word; in the handle table it is the typed channel end
  `zx_chan_fuchsia_media_UsageWatcher_client`. `role: server` would map to `..._server` instead.
- `usage_watcher` reports `padding: 4`, which pads the body to its 8-byte alignment.

Check against the IR: 16 (`Usage` inline) + 4 (presence) + 4 (padding) = 24 = `inline_size`. The
emitted node is shorter than that because `fidl_union_member` models a smaller envelope than
wire-format v2 uses — that is the known helper limitation, so I keep the IR padding as it is and
add nothing to compensate.

**Cross-protocol dependency**: `zx_chan_fuchsia_media_UsageWatcher_client` is only produced by
`zx_channel_create$fuchsia_media_UsageWatcher`. Without that call the fuzzer can never obtain a
handle of this type, so I add it to `required` — together with
`fdio_service_connect$fuchsia_media_UsageWatcher`, which consumes the server end that the create
call also produces (a resource nothing consumes is a compile error). I do **not** add
`UsageWatcher`'s own methods — that protocol gets its own run.

**No include**: this call uses no symbolic constants (`ZX_CHANNEL_MAX_MSG_BYTES` only appears in
`zx_channel_call` variants), so no `include` is emitted.

### Final output

```json
{
	"spec": [
		{"type": "struct", "name": "fuchsia_media_UsageReporterWatchRequest", "code": "fuchsia_media_UsageReporterWatchRequest {\n\thdr\t\tfidl_message_header[8547391950682442073]\n\tusageInLine\tfuchsia_media_UsageInLine\n\tusage_watcher\tflags[fidl_handle_presence, int32]\n\tpadding0\tarray[const[0, int8], 4]\n\tusageOutOfLine\tfuchsia_media_UsageOutOfLine\n} [packed]"},
		{"type": "struct", "name": "fuchsia_media_UsageReporterWatchRequestHandles", "code": "fuchsia_media_UsageReporterWatchRequestHandles {\n\tusage\t\tfuchsia_media_UsageHandles\n\tusage_watcher\tzx_chan_fuchsia_media_UsageWatcher_client\n} [packed]"},
		{"type": "syscall", "name": "zx_channel_write$fuchsia_media_UsageReporterWatch", "code": "zx_channel_write$fuchsia_media_UsageReporterWatch(handle zx_chan_fuchsia_media_UsageReporter_client, options const[0], bytes ptr[in, fuchsia_media_UsageReporterWatchRequest], num_bytes bytesize[bytes], handles ptr[in, fuchsia_media_UsageReporterWatchRequestHandles], num_handles bytesize4[handles])"}
	],
	"required": [
		{"type": "union", "name": "fuchsia_media_UsageInLine"},
		{"type": "union", "name": "fuchsia_media_UsageOutOfLine"},
		{"type": "union", "name": "fuchsia_media_UsageHandles"},
		{"type": "init_syscall", "name": "zx_channel_create$fuchsia_media_UsageWatcher"},
		{"type": "init_syscall", "name": "fdio_service_connect$fuchsia_media_UsageWatcher"}
	]
}
```

# Example of writing spec for a oneway method with a union and a protocol endpoint (2)

## Input

(same methods array as above)

Please write specification for union `fuchsia_media_UsageInLine`.

## Output

### Thought

**Task**: the inline node of the FIDL union `fuchsia.media/Usage` (strip `InLine`, split the mangled
name at the first `_` before an uppercase letter -> library `fuchsia.media`, declaration `Usage`).

(Calling `get_union_members_by_name("fuchsia.media/Usage")`)

```json
[{"name": "render_usage", "ordinal": 1,
  "type": {"kind_v2": "identifier", "identifier": "fuchsia.media/AudioRenderUsage",
           "type_shape_v2": {"inline_size": 4, "max_handles": 0, "max_out_of_line": 0}}},
 {"name": "capture_usage", "ordinal": 2,
  "type": {"kind_v2": "identifier", "identifier": "fuchsia.media/AudioCaptureUsage",
           "type_shape_v2": {"inline_size": 4, "max_handles": 0, "max_out_of_line": 0}}}]
```

(Calling `get_union_type_shape_by_name("fuchsia.media/Usage")`)

```json
{"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 0, "has_padding": false}
```

Both options are enums. Their ordinals are `1` and `2` — taken from the `ordinal` field, never from
the position in the list.

(Calling `get_enum_type_by_name("fuchsia.media/AudioRenderUsage")`) -> `uint32`

(Calling `get_enum_members_by_name("fuchsia.media/AudioRenderUsage")`)

```json
[{"name": "BACKGROUND", "value": {"value": "0"}}, {"name": "MEDIA", "value": {"value": "1"}},
 {"name": "INTERRUPTION", "value": {"value": "2"}}, {"name": "SYSTEM_AGENT", "value": {"value": "3"}},
 {"name": "COMMUNICATION", "value": {"value": "4"}}]
```

(Calling `get_enum_type_by_name("fuchsia.media/AudioCaptureUsage")`) -> `uint32`

(Calling `get_enum_members_by_name("fuchsia.media/AudioCaptureUsage")`) -> `BACKGROUND 0`,
`FOREGROUND 1`, `SYSTEM_AGENT 2`, `COMMUNICATION 3`

Both enums are `uint32`, so they map to `flags[..., int32]`. The member constants are the mangled
FIDL names (`fuchsia.media/AudioRenderUsage.MEDIA` -> `fuchsia_media_AudioRenderUsage_MEDIA`), which
`syz-extract` resolves from the FIDL API summaries; all values are non-negative, so no literal
fallback is needed.

**Empty companions**: `max_out_of_line` and `max_handles` are both 0 and no option carries
out-of-line data or handles, so `...OutOfLine` and `...Handles` are emitted as empty structs
(`void void`) rather than as unions. They must exist because the request body and its handle table
reference them.

**Option names**: both options are scalars after mapping, so they keep their plain names; an option
whose type had an out-of-line part would be named `<opt>InLine`.

### Final output

```json
{
	"spec": [
		{"type": "flag", "name": "fuchsia_media_AudioRenderUsage", "code": "fuchsia_media_AudioRenderUsage = fuchsia_media_AudioRenderUsage_BACKGROUND, fuchsia_media_AudioRenderUsage_MEDIA, fuchsia_media_AudioRenderUsage_INTERRUPTION, fuchsia_media_AudioRenderUsage_SYSTEM_AGENT, fuchsia_media_AudioRenderUsage_COMMUNICATION"},
		{"type": "flag", "name": "fuchsia_media_AudioCaptureUsage", "code": "fuchsia_media_AudioCaptureUsage = fuchsia_media_AudioCaptureUsage_BACKGROUND, fuchsia_media_AudioCaptureUsage_FOREGROUND, fuchsia_media_AudioCaptureUsage_SYSTEM_AGENT, fuchsia_media_AudioCaptureUsage_COMMUNICATION"},
		{"type": "union", "name": "fuchsia_media_UsageInLine", "code": "fuchsia_media_UsageInLine [\n\trender_usage\tfidl_union_member[1, flags[fuchsia_media_AudioRenderUsage, int32]]\n\tcapture_usage\tfidl_union_member[2, flags[fuchsia_media_AudioCaptureUsage, int32]]\n]"},
		{"type": "struct", "name": "fuchsia_media_UsageOutOfLine", "code": "fuchsia_media_UsageOutOfLine {\n\tvoid\tvoid\n} [packed]"},
		{"type": "struct", "name": "fuchsia_media_UsageHandles", "code": "fuchsia_media_UsageHandles {\n\tvoid\tvoid\n} [packed]"}
	],
	"required": []
}
```
